package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/sync/singleflight"
	"watchparty-backend/models"
)

var (
	scrapeDialer = &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	httpClient = &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			DialContext:           scrapeSafeDialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          50,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			if err := guardScrapeURL(req.URL.String()); err != nil {
				return err
			}
			return nil
		},
	}
	m3u8Regex   = regexp.MustCompile(`https?://[^\s"'<>\)\]\}]+\.m3u8(?:\?[^\s"'<>\)\]\}]*)?`)
	mp4Regex    = regexp.MustCompile(`https?://[^\s"'<>\)\]\}]+\.mp4(?:\?[^\s"'<>\)\]\}]*)?`)
	actionRegex = regexp.MustCompile(`action:\s*["']([a-f0-9]{32})["']`)
	nonceRegex  = regexp.MustCompile(`data:\s*\{\s*action:\s*["']([a-f0-9]{32})["']\s*\}`)
	iframeRegex = regexp.MustCompile(`<iframe[^>]+src="([^"]+)"`)
	packerRegex = regexp.MustCompile(`(?s)eval\(function\(p,a,c,k,e,d\).*?\}\('(.*?)',\s*(\d+),\s*(\d+),\s*'(.*?)'\.split\('\|'\)`)

	scriptFileRegex     = regexp.MustCompile(`"file"\s*:\s*"(https?://[^"]+?\.m3u8[^"]*)"`)
	scriptSourcesRegex  = regexp.MustCompile(`"sources"\s*:\s*\[.*?"file"\s*:\s*"(https?://[^"]+)"`)
	scriptSourceRegex   = regexp.MustCompile(`source\s*[:=]\s*["'](https?://[^"']+?\.m3u8[^"']*)["']`)
	scriptVarVideoRegex = regexp.MustCompile(`var\s+\w*[Vv]ideo\w*\s*=\s*["'](https?://[^"']+)["']`)
)

// Scrape cache — avoids re-scraping the same URL within the TTL window.
var (
	scrapeCache   = make(map[string]scrapeCacheEntry)
	scrapeCacheMu sync.RWMutex
	scrapeGroup   singleflight.Group
)

type scrapeCacheEntry struct {
	metadata  *models.VideoMetadata
	expiresAt time.Time
}

const scrapeCacheTTL = 30 * time.Minute

func init() {
	// Route scraper HTTP client through NanoProxy if configured
	nanoHTTP := os.Getenv("NANOPROXY_HTTP_URL")
	if nanoHTTP != "" {
		proxyURL, err := url.Parse(nanoHTTP)
		if err == nil {
			if tr, ok := httpClient.Transport.(*http.Transport); ok {
				tr.Proxy = http.ProxyURL(proxyURL)
			} else {
				httpClient.Transport = &http.Transport{
					Proxy:       http.ProxyURL(proxyURL),
					DialContext: scrapeSafeDialContext,
				}
			}
			log.Printf("[Scraper] HTTP client routing through NanoProxy: %s", nanoHTTP)
		}
	}

	// Periodic scrape-cache cleanup
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			cleanupScrapeCache()
		}
	}()
}

func copyVideoMetadata(m *models.VideoMetadata) *models.VideoMetadata {
	if m == nil {
		return nil
	}
	c := *m
	return &c
}

// isPlayableStreamURL reports whether u looks like a real media stream
// (not an anime episode HTML page). Used to reject bad scrape results / cache.
func isPlayableStreamURL(u string) bool {
	u = strings.TrimSpace(u)
	if u == "" {
		return false
	}
	lower := strings.ToLower(u)
	if strings.Contains(lower, "googlevideo.com") || strings.Contains(lower, "videoplayback") {
		return true
	}
	parsed, err := url.Parse(u)
	if err != nil || parsed.Host == "" {
		return false
	}
	path := strings.ToLower(parsed.Path)
	if strings.HasSuffix(path, ".mp4") || strings.HasSuffix(path, ".m3u8") ||
		strings.HasSuffix(path, ".webm") || strings.HasSuffix(path, ".mkv") {
		return true
	}
	// Common stream path markers
	if strings.Contains(path, "/hls/") || strings.Contains(path, "/stream/") ||
		strings.Contains(lower, ".m3u8") || strings.Contains(lower, ".mp4?") {
		return true
	}
	return false
}

// InvalidateScrapeCache removes a page URL from the scrape cache (e.g. bad entry).
func InvalidateScrapeCache(pageURL string) {
	scrapeCacheMu.Lock()
	delete(scrapeCache, pageURL)
	scrapeCacheMu.Unlock()
}

// ScrapeStreamURLCached wraps ScrapeStreamURL with an in-memory TTL cache,
// singleflight stampede protection, and defensive metadata copies.
func ScrapeStreamURLCached(pageURL string) (*models.VideoMetadata, error) {
	scrapeCacheMu.RLock()
	if entry, ok := scrapeCache[pageURL]; ok && time.Now().Before(entry.expiresAt) {
		// Drop poisoned cache entries (page URL stored as VideoURL)
		if entry.metadata != nil && isPlayableStreamURL(entry.metadata.VideoURL) {
			scrapeCacheMu.RUnlock()
			log.Printf("Cache hit for %s", pageURL)
			return copyVideoMetadata(entry.metadata), nil
		}
		scrapeCacheMu.RUnlock()
		InvalidateScrapeCache(pageURL)
		log.Printf("Cache invalidated (non-playable VideoURL) for %s", pageURL)
	} else {
		scrapeCacheMu.RUnlock()
	}

	v, err, _ := scrapeGroup.Do(pageURL, func() (interface{}, error) {
		scrapeCacheMu.RLock()
		if entry, ok := scrapeCache[pageURL]; ok && time.Now().Before(entry.expiresAt) {
			if entry.metadata != nil && isPlayableStreamURL(entry.metadata.VideoURL) {
				scrapeCacheMu.RUnlock()
				return copyVideoMetadata(entry.metadata), nil
			}
		}
		scrapeCacheMu.RUnlock()

		metadata, err := ScrapeStreamURL(pageURL)
		if err != nil {
			return nil, err
		}
		if metadata == nil || !isPlayableStreamURL(metadata.VideoURL) {
			vu := ""
			if metadata != nil {
				vu = metadata.VideoURL
			}
			return nil, fmt.Errorf("scrape returned non-playable URL for %s: %s", pageURL, truncateURL(vu, 80))
		}

		RegisterStreamMetadata(metadata.VideoURL, metadata.ThumbnailURL)

		stored := copyVideoMetadata(metadata)
		scrapeCacheMu.Lock()
		scrapeCache[pageURL] = scrapeCacheEntry{
			metadata:  stored,
			expiresAt: time.Now().Add(scrapeCacheTTL),
		}
		if len(scrapeCache) > 100 {
			now := time.Now()
			for key, entry := range scrapeCache {
				if now.After(entry.expiresAt) {
					delete(scrapeCache, key)
				}
			}
		}
		scrapeCacheMu.Unlock()

		return copyVideoMetadata(stored), nil
	})
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, nil
	}
	return v.(*models.VideoMetadata), nil
}

func isBlockedScrapeIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return true
	}
	if ip.Equal(net.ParseIP("169.254.169.254")) {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil && ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127 {
		return true
	}
	return false
}

func guardScrapeURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid scrape URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("unsupported scrape scheme")
	}
	host := parsed.Hostname()
	if host == "" {
		return fmt.Errorf("missing scrape host")
	}
	blocked := []string{"localhost", "127.0.0.1", "0.0.0.0", "::1", "host.docker.internal", "metadata.google.internal"}
	for _, b := range blocked {
		if strings.EqualFold(host, b) {
			return fmt.Errorf("scrape blocked host %s", host)
		}
	}
	if ip := net.ParseIP(host); ip != nil && isBlockedScrapeIP(ip) {
		return fmt.Errorf("scrape blocked private IP %s", host)
	}
	return nil
}

func scrapeSafeDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	if ip := net.ParseIP(host); ip != nil {
		if isBlockedScrapeIP(ip) {
			return nil, fmt.Errorf("blocked private IP: %s", host)
		}
		return scrapeDialer.DialContext(ctx, network, addr)
	}
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	var lastErr error
	for _, ipAddr := range ips {
		if isBlockedScrapeIP(ipAddr.IP) {
			lastErr = fmt.Errorf("blocked private IP for %s: %s", host, ipAddr.IP)
			continue
		}
		conn, dialErr := scrapeDialer.DialContext(ctx, network, net.JoinHostPort(ipAddr.IP.String(), port))
		if dialErr == nil {
			return conn, nil
		}
		lastErr = dialErr
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no public IP for host %s", host)
	}
	return nil, lastErr
}

func cleanupScrapeCache() {
	scrapeCacheMu.Lock()
	defer scrapeCacheMu.Unlock()

	now := time.Now()
	before := len(scrapeCache)
	for key, entry := range scrapeCache {
		if now.After(entry.expiresAt) {
			delete(scrapeCache, key)
		}
	}
	after := len(scrapeCache)
	if before != after {
		log.Printf("[Scraper] Cache cleanup: %d → %d entries", before, after)
	}
}

// User-Agent rotation — avoids fingerprint-based blocking from anime CDNs.
var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Safari/605.1.15",
}

var (
	uaIndex int
	uaMu    sync.Mutex
)

func getRandomUserAgent() string {
	uaMu.Lock()
	defer uaMu.Unlock()
	ua := userAgents[uaIndex%len(userAgents)]
	uaIndex++
	return ua
}

// fetchWithRetry retries the request up to maxRetries times with exponential
// backoff and rotating User-Agent headers. Retries on 429 (rate-limit) and
// 503 (server overloaded) status codes.
func fetchWithRetry(fetchURL, referer string, maxRetries int) (*http.Response, error) {
	if err := guardScrapeURL(fetchURL); err != nil {
		return nil, err
	}
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		if i > 0 {
			time.Sleep(time.Duration(i*2) * time.Second)
		}

		req, err := http.NewRequest("GET", fetchURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", getRandomUserAgent())
		if referer != "" {
			req.Header.Set("Referer", referer)
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode == 429 || resp.StatusCode == 503 {
			resp.Body.Close()
			lastErr = fmt.Errorf("HTTP %d from %s (attempt %d/%d)", resp.StatusCode, fetchURL, i+1, maxRetries)
			continue
		}

		return resp, nil
	}
	return nil, lastErr
}

// detectSitePlatform identifies the known scraping adapter for a page URL based on its domain.
func detectSitePlatform(pageURL string) string {
	domain := extractDomain(pageURL)
	switch {
	case strings.Contains(domain, "otakudesu"):
		return "otakudesu"
	case strings.Contains(domain, "anoboy"):
		return "anoboy"
	case strings.Contains(domain, "samehadaku"):
		return "samehadaku"
	case strings.Contains(domain, "animasu"):
		return "animasu"
	case strings.Contains(domain, "kuronime"):
		return "kuronime"
	case strings.Contains(domain, "nanime"):
		return "nanime"
	case strings.Contains(domain, "idlix"):
		return "idlix"
	case strings.Contains(domain, "sokuja"):
		return "sokuja"
	default:
		return "generic"
	}
}

// isBloggerVideoURL returns true if the URL points to a Blogger/Google video player.
func isBloggerVideoURL(rawURL string) bool {
	return strings.Contains(rawURL, "blogger.com/video") ||
		(strings.Contains(rawURL, "blogspot.com") && strings.Contains(rawURL, "/video."))
}

var (
	bloggerTokenRe = regexp.MustCompile(`(?i)[?&]token=([^&]+)`)
	// batchexecute embeds progressive MP4s; body may use \u003d / \u0026 escapes
	bloggerGooglevideoRe = regexp.MustCompile(`https://[^"'\s]+googlevideo\.com/videoplayback[^"'\s]*`)
	bloggerLegacyPlayRe  = []*regexp.Regexp{
		regexp.MustCompile(`"play_url"\s*:\s*"(https?://[^"]+)"`),
		regexp.MustCompile(`"stream_url"\s*:\s*"(https?://[^"]+)"`),
		regexp.MustCompile(`"download_url"\s*:\s*"(https?://[^"]+)"`),
		regexp.MustCompile(`"downloadUrl"\s*:\s*"(https?://[^"]+)"`),
		regexp.MustCompile(`var\s+_play_url\s*=\s*['"](https?://[^'"]+)['"]`),
	}
)

func unescapeBloggerURL(raw string) string {
	s := raw
	// Nested batchexecute JSON often has \\u003d which becomes \= after one pass.
	// Loop until stable so we fully decode.
	for i := 0; i < 5; i++ {
		prev := s
		s = strings.ReplaceAll(s, `\u003d`, "=")
		s = strings.ReplaceAll(s, `\u0026`, "&")
		s = strings.ReplaceAll(s, `\u002f`, "/")
		s = strings.ReplaceAll(s, `\u002F`, "/")
		s = strings.ReplaceAll(s, `\/`, "/")
		// Residual after partial decode of \\u003d → \=
		s = strings.ReplaceAll(s, `\=`, "=")
		s = strings.ReplaceAll(s, `\&`, "&")
		s = strings.ReplaceAll(s, `\/`, "/")
		if s == prev {
			break
		}
	}
	// Strip trailing JSON debris
	s = strings.TrimRight(s, `",]\ `)
	s = strings.TrimSpace(s)
	return s
}

func extractBloggerToken(bloggerPageURL string) string {
	if m := bloggerTokenRe.FindStringSubmatch(bloggerPageURL); len(m) > 1 {
		tok, err := url.QueryUnescape(m[1])
		if err == nil && tok != "" {
			return tok
		}
		return m[1]
	}
	return ""
}

func bloggerItagScore(videoURL string) int {
	// Prefer higher progressive quality: itag 22 (720p) > 18 (360p)
	u, err := url.Parse(videoURL)
	if err != nil {
		return 0
	}
	itag := u.Query().Get("itag")
	switch itag {
	case "22":
		return 300
	case "18":
		return 100
	case "37", "38":
		return 400
	default:
		if n, e := strconv.Atoi(itag); e == nil {
			return n
		}
		return 1
	}
}

func pickBestBloggerStream(urls []string) string {
	best := ""
	bestScore := -1
	seen := make(map[string]bool)
	for _, raw := range urls {
		u := unescapeBloggerURL(raw)
		if !strings.Contains(u, "googlevideo.com/videoplayback") &&
			!strings.Contains(u, ".mp4") &&
			!strings.Contains(u, ".m3u8") {
			continue
		}
		if !strings.HasPrefix(u, "http") {
			continue
		}
		if seen[u] {
			continue
		}
		seen[u] = true
		score := bloggerItagScore(u)
		if score > bestScore {
			bestScore = score
			best = u
		}
	}
	return best
}

// fetchBloggerViaBatchexecute resolves modern Blogger video.g players.
// As of 2025+, play_url is no longer in HTML — the SPA POSTs to
// /_/BloggerVideoPlayerUi/data/batchexecute with rpcid WcwnYd and receives
// googlevideo.com/videoplayback progressive MP4 URLs.
func fetchBloggerViaBatchexecute(token, referer string) (string, error) {
	if token == "" {
		return "", fmt.Errorf("missing blogger token")
	}

	// f.req shape (matches browser): [[["WcwnYd","[\"TOKEN\",null,0]",null,"generic"]]]
	innerArgs, err := json.Marshal([]interface{}{token, nil, 0})
	if err != nil {
		return "", err
	}
	outer, err := json.Marshal([][]interface{}{
		{"WcwnYd", string(innerArgs), nil, "generic"},
	})
	if err != nil {
		return "", err
	}
	// Wrap one more array level: [[...]]
	freq := "[" + string(outer) + "]"

	endpoint := "https://www.blogger.com/_/BloggerVideoPlayerUi/data/batchexecute?rpcids=WcwnYd&source-path=%2Fvideo.g&bl=boq_bloggeruiserver_20260715.01_p0&hl=en-US&rt=c"
	form := url.Values{}
	form.Set("f.req", freq)

	req, err := http.NewRequest("POST", endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8")
	req.Header.Set("User-Agent", getRandomUserAgent())
	req.Header.Set("X-Same-Domain", "1")
	if referer != "" {
		req.Header.Set("Referer", referer)
	} else {
		req.Header.Set("Referer", "https://www.blogger.com/")
	}
	req.Header.Set("Origin", "https://www.blogger.com")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("blogger batchexecute HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return "", err
	}
	text := string(body)

	// Fully decode nested JSON escapes (\u003d, \\u003d → =, residual \=)
	normalized := unescapeBloggerURL(text)

	rawMatches := bloggerGooglevideoRe.FindAllString(normalized, -1)
	if best := pickBestBloggerStream(rawMatches); best != "" {
		// Final sanitize — never return backslash-escaped query strings
		best = unescapeBloggerURL(best)
		if strings.Contains(best, `\`) {
			best = strings.ReplaceAll(best, `\`, "")
		}
		return best, nil
	}
	return "", fmt.Errorf("no googlevideo URL in batchexecute response")
}

// extractBloggerVideoURL resolves a Blogger video.g?token=… page to a direct
// progressive MP4 (googlevideo). Prefers the modern batchexecute API; falls
// back to legacy play_url JSON embedded in HTML for older players.
func extractBloggerVideoURL(bloggerPageURL, referer string) (string, error) {
	token := extractBloggerToken(bloggerPageURL)
	if token != "" {
		if videoURL, err := fetchBloggerViaBatchexecute(token, referer); err == nil && videoURL != "" {
			return videoURL, nil
		}
	}

	html, err := fetchPageHTML(bloggerPageURL, referer)
	if err != nil {
		return "", fmt.Errorf("failed to fetch Blogger page: %w", err)
	}

	for _, re := range bloggerLegacyPlayRe {
		if match := re.FindStringSubmatch(html); len(match) > 1 {
			return unescapeBloggerURL(match[1]), nil
		}
	}

	if streamURL := findStreamURLInHTML(html); streamURL != "" {
		return streamURL, nil
	}
	if streamURL := findStreamURLInScripts(html); streamURL != "" {
		return streamURL, nil
	}

	// Last resort: token may still work even if first batchexecute attempt failed
	if token != "" {
		if videoURL, err := fetchBloggerViaBatchexecute(token, "https://www.blogger.com/"); err == nil && videoURL != "" {
			return videoURL, nil
		}
	}

	return "", fmt.Errorf("could not extract video URL from Blogger page")
}

func ScrapeStreamURL(pageURL string) (*models.VideoMetadata, error) {
	// Fail closed before any fetch/headless work (SSRF / private hosts).
	if err := guardScrapeURL(pageURL); err != nil {
		return nil, err
	}

	switch detectSitePlatform(pageURL) {
	case "otakudesu":
		return scrapeOtakudesu(pageURL)
	case "anoboy":
		metadata, err := scrapeAnoboy(pageURL)
		if err == nil {
			return metadata, nil
		}
		// Static Blogger resolve failed — try headless network capture
		log.Printf("Anoboy static scrape failed for %s: %v — trying headless...", pageURL, err)
		return scrapeWithNanoOrHeadless(pageURL)
	case "samehadaku":
		return scrapeSamehadaku(pageURL)
	case "idlix":
		// API chain (UUID → gate → claim → majorplay), not DOM/SVG scraping
		return scrapeIdlix(pageURL)
	case "sokuja":
		// Next.js API: /api/video-mirrors?e={episodeId} — no headless
		metadata, err := scrapeSokuja(pageURL)
		if err == nil {
			return metadata, nil
		}
		log.Printf("Sokuja static scrape failed for %s: %v — trying headless...", pageURL, err)
		return scrapeWithNanoOrHeadless(pageURL)
	default:
		// Layer 1: static scrape (fast)
		metadata, err := scrapeGeneric(pageURL)
		if err == nil {
			return metadata, nil
		}
		// Layer 2: headless + network interception (SPA / encrypted players)
		log.Printf("Static scrape failed for %s, trying headless...", pageURL)
		return scrapeWithNanoOrHeadless(pageURL)
	}
}

// scrapeWithNanoOrHeadless tries NanoScraper first (network interception via NanoProxy),
// then falls back to headless HTML-only scraping.
func scrapeWithNanoOrHeadless(pageURL string) (*models.VideoMetadata, error) {
	// Quick check: if Chrome is not installed, skip headless entirely
	if findChromePath() == "" {
		return nil, fmt.Errorf("could not find stream URL from %s (Chrome/Chromium not installed — install with: sudo pacman -S chromium)", pageURL)
	}

	log.Printf("Static scrape failed for %s, trying NanoScraper...", pageURL)

	result, err := NanoScrapeWithPool(pageURL)
	if err == nil && result.StreamURL != "" {
		metadata := &models.VideoMetadata{
			VideoURL:     result.StreamURL,
			Title:        result.PageTitle,
			ThumbnailURL: result.Thumbnail,
			Source:       result.Source,
		}
		return metadata, nil
	}

	log.Printf("NanoScraper failed for %s, trying headless HTML scrape: %v", pageURL, err)
	return scrapeWithHeadless(pageURL)
}

// scrapeSamehadaku is implemented in samehadaku_scraper.go (WP player_ajax).
// Do not use headless-first for Samehadaku — ads in iframes cause deadline exceeded.

type mirror struct {
	host    string
	payload string
	score   int
}

// getOtakudesuAjaxURL derives the WordPress AJAX endpoint from the page's own
// scheme+host so scraping keeps working if otakudesu moves to a new domain.
func getOtakudesuAjaxURL(pageURL string) string {
	parsed, err := url.Parse(pageURL)
	if err != nil {
		return "https://otakudesu.blog/wp-admin/admin-ajax.php"
	}
	return parsed.Scheme + "://" + parsed.Host + "/wp-admin/admin-ajax.php"
}

func scrapeOtakudesu(pageURL string) (*models.VideoMetadata, error) {
	doc, err := fetchPageDocument(pageURL, "")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Otakudesu page: %w", err)
	}

	html, _ := doc.Html()

	allActions := actionRegex.FindAllStringSubmatch(html, -1)
	if len(allActions) < 2 {
		return nil, fmt.Errorf("could not find WordPress AJAX action IDs in page source")
	}

	var nonceAction string
	nonceMatch := nonceRegex.FindStringSubmatch(html)
	if len(nonceMatch) > 1 {
		nonceAction = nonceMatch[1]
	} else {
		nonceAction = allActions[1][1]
	}

	var iframeAction string
	for _, match := range allActions {
		if match[1] != nonceAction {
			iframeAction = match[1]
			break
		}
	}
	if iframeAction == "" {
		iframeAction = allActions[0][1]
	}

	var mirrors []mirror
	doc.Find(".mirrorstream a").Each(func(_ int, s *goquery.Selection) {
		b64Payload, exists := s.Attr("data-content")
		if !exists || b64Payload == "" {
			return
		}
		hostName := strings.TrimSpace(strings.ToLower(s.Text()))

		decoded, err := base64.StdEncoding.DecodeString(b64Payload)
		if err != nil {
			return
		}
		var pl struct {
			Q string `json:"q"`
		}
		if err := json.Unmarshal(decoded, &pl); err != nil {
			return
		}
		quality := pl.Q

		score := 0
		if strings.Contains(quality, "720p") {
			score += 100
		} else if strings.Contains(quality, "480p") {
			score += 50
		} else if strings.Contains(quality, "360p") {
			score += 10
		}

		if strings.Contains(hostName, "vidhide") {
			score += 5
		} else if strings.Contains(hostName, "filedon") {
			score += 4
		} else if strings.Contains(hostName, "mega") {
			score += 1
		}

		mirrors = append(mirrors, mirror{
			host:    hostName,
			payload: b64Payload,
			score:   score,
		})
	})

	sort.Slice(mirrors, func(i, j int) bool {
		return mirrors[i].score > mirrors[j].score
	})

	if len(mirrors) == 0 {
		return nil, fmt.Errorf("no mirrors found on episode page")
	}

	ajaxURL := getOtakudesuAjaxURL(pageURL)
	nonce, err := fetchNonce(ajaxURL, nonceAction, pageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch nonce: %w", err)
	}

	var lastErr error
	for _, m := range mirrors {
		iframeSrc, err := fetchMirrorIframeSrc(ajaxURL, m.payload, nonce, iframeAction, pageURL)
		if err != nil {
			lastErr = err
			continue
		}

		embedHTML, err := fetchPageHTML(iframeSrc, pageURL)
		if err != nil {
			lastErr = err
			continue
		}

		if streamURL := findStreamURLInHTML(embedHTML); streamURL != "" {
			metadata := extractOtakudesuMetadata(doc, pageURL)
			metadata.VideoURL = streamURL
			return metadata, nil
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("failed to resolve stream from mirrors: %w", lastErr)
	}
	return nil, fmt.Errorf("failed to resolve stream from any mirror")
}

// scrapeAnoboy handles anoboy.si and similar WordPress-based anime sites that
// use Blogger video embeds via base64-encoded mirror options.
//
// Anoboy page structure:
//   - Direct <iframe> with Blogger video URL in the main content
//   - A <select class="mirror"> dropdown with base64-encoded iframe HTML as values
//   - Each decoded value is an <iframe src="https://blogger.com/video.g?token=...">
//
// The scraper tries each mirror (base64-decoded) and extracts the actual video
// URL from the Blogger video page using extractBloggerVideoURL.
func scrapeAnoboy(pageURL string) (*models.VideoMetadata, error) {
	doc, err := fetchPageDocument(pageURL, "")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Anoboy page: %w", err)
	}

	html, _ := doc.Html()

	// Layer 1: Try direct iframe in page (Blogger video embed).
	directIframeSrc := ""
	doc.Find(".player-embed iframe, #pembed iframe, .video-content iframe").Each(func(_ int, s *goquery.Selection) {
		if src, exists := s.Attr("src"); exists && src != "" && directIframeSrc == "" {
			directIframeSrc = normalizeURL(src, pageURL)
		}
	})

	if directIframeSrc == "" {
		// Fallback: any iframe in the page
		doc.Find("iframe").Each(func(_ int, s *goquery.Selection) {
			if src, exists := s.Attr("src"); exists && src != "" && directIframeSrc == "" {
				directIframeSrc = normalizeURL(src, pageURL)
			}
		})
	}

	if directIframeSrc != "" {
		videoURL, err := resolveEmbedURL(directIframeSrc, pageURL)
		if err == nil && videoURL != "" {
			metadata := extractAnoboyMetadata(doc, pageURL)
			metadata.VideoURL = videoURL
			return metadata, nil
		}
	}

	// Layer 2: Parse mirror dropdown options (base64-encoded iframe HTML).
	// Pattern: <select class="mirror">...<option value="BASE64_IFRAME_HTML" data-index="1">...</option></select>
	mirrorOptionRegex := regexp.MustCompile(`<option\s+value="([A-Za-z0-9+/=]{20,})"`)
	mirrorMatches := mirrorOptionRegex.FindAllStringSubmatch(html, -1)

	for _, m := range mirrorMatches {
		b64Value := m[1]
		decoded, err := base64.StdEncoding.DecodeString(b64Value)
		if err != nil {
			continue
		}

		// The decoded value is an <iframe src="..."> tag.
		decodedStr := string(decoded)
		iframeSrcMatch := iframeRegex.FindStringSubmatch(decodedStr)
		if len(iframeSrcMatch) < 2 {
			continue
		}

		iframeSrc := normalizeURL(iframeSrcMatch[1], pageURL)

		videoURL, err := resolveEmbedURL(iframeSrc, pageURL)
		if err == nil && videoURL != "" {
			metadata := extractAnoboyMetadata(doc, pageURL)
			metadata.VideoURL = videoURL
			return metadata, nil
		}
	}

	// Layer 3: Deep search for any iframes in the page (catch-all).
	iframeSrcs := extractIframeSrcs(html)
	for _, iframeSrc := range iframeSrcs {
		iframeSrc = normalizeURL(iframeSrc, pageURL)
		videoURL, err := resolveEmbedURL(iframeSrc, pageURL)
		if err == nil && videoURL != "" {
			metadata := extractAnoboyMetadata(doc, pageURL)
			metadata.VideoURL = videoURL
			return metadata, nil
		}
	}

	return nil, fmt.Errorf("could not find stream URL from %s", pageURL)
}

// resolveEmbedURL takes an iframe/embed URL and tries to resolve it to a
// direct video stream URL. It handles Blogger video pages specially.
func resolveEmbedURL(embedURL, referer string) (string, error) {
	// If it's a Blogger video page, use the dedicated extractor.
	if isBloggerVideoURL(embedURL) {
		return extractBloggerVideoURL(embedURL, referer)
	}

	// For other embed URLs, fetch and search for stream URLs.
	html, err := fetchPageHTMLWithRefererChain(embedURL, []string{referer, ""})
	if err != nil {
		return "", err
	}

	// Direct stream URL in HTML
	if streamURL := findStreamURLInHTML(html); streamURL != "" {
		return streamURL, nil
	}

	// Stream URL in script tags
	if streamURL := findStreamURLInScripts(html); streamURL != "" {
		return streamURL, nil
	}

	// Follow nested iframes (up to 2 levels for embed resolution)
	for _, nestedSrc := range extractIframeSrcs(html) {
		nestedSrc = normalizeURL(nestedSrc, embedURL)
		if isBloggerVideoURL(nestedSrc) {
			if videoURL, err := extractBloggerVideoURL(nestedSrc, embedURL); err == nil {
				return videoURL, nil
			}
		}
		// Try fetching nested iframe content
		nestedHTML, err := fetchPageHTML(nestedSrc, embedURL)
		if err != nil {
			continue
		}
		if streamURL := findStreamURLInHTML(nestedHTML); streamURL != "" {
			return streamURL, nil
		}
		if streamURL := findStreamURLInScripts(nestedHTML); streamURL != "" {
			return streamURL, nil
		}
	}

	return "", fmt.Errorf("could not resolve embed URL to stream: %s", embedURL)
}

// extractAnoboyMetadata extracts title, episode, and thumbnail from an Anoboy page.
func extractAnoboyMetadata(doc *goquery.Document, pageURL string) *models.VideoMetadata {
	metadata := &models.VideoMetadata{Source: extractDomain(pageURL)}

	// Anoboy uses <h1 class="entry-title"> for episode title
	title := doc.Find("h1.entry-title").First().Text()
	if title == "" {
		title = doc.Find("h1").First().Text()
	}
	metadata.Title = strings.TrimSpace(title)

	// Episode number is usually in the title
	metadata.Episode = cleanEpisodeString(metadata.Title)

	// Thumbnail from the post image
	thumb, exists := doc.Find(".item.meta img, .postbody img").First().Attr("src")
	if exists && thumb != "" {
		if strings.HasPrefix(thumb, "//") {
			thumb = "https:" + thumb
		}
		metadata.ThumbnailURL = thumb
	}

	// Fallback to OG tags
	if metadata.Title == "" {
		ogTitle := doc.Find("meta[property='og:title']").AttrOr("content", "")
		if ogTitle != "" {
			metadata.Title = ogTitle
		}
	}
	if metadata.ThumbnailURL == "" {
		ogImage := doc.Find("meta[property='og:image']").AttrOr("content", "")
		if ogImage != "" {
			if strings.HasPrefix(ogImage, "//") {
				ogImage = "https:" + ogImage
			}
			metadata.ThumbnailURL = ogImage
		}
	}

	metadata.NextEpisodeURL = extractNextEpisodeURL(doc, pageURL)
	return metadata
}

func fetchNonce(ajaxURL, action, referer string) (string, error) {
	data := url.Values{}
	data.Set("action", action)

	req, err := http.NewRequest("POST", ajaxURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Referer", referer)

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("nonce request HTTP %d", resp.StatusCode)
	}

	var result struct {
		Data string `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.Data, nil
}

func fetchMirrorIframeSrc(ajaxURL, b64Payload, nonce, action, referer string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(b64Payload)
	if err != nil {
		return "", err
	}
	var payloadMap map[string]interface{}
	if err := json.Unmarshal(decoded, &payloadMap); err != nil {
		return "", err
	}

	data := url.Values{}
	for k, v := range payloadMap {
		data.Set(k, fmt.Sprintf("%v", v))
	}
	data.Set("nonce", nonce)
	data.Set("action", action)

	req, err := http.NewRequest("POST", ajaxURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Referer", referer)

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("iframe request HTTP %d", resp.StatusCode)
	}

	var result struct {
		Data string `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	decodedHTML, err := base64.StdEncoding.DecodeString(result.Data)
	if err != nil {
		return "", err
	}

	matches := iframeRegex.FindStringSubmatch(string(decodedHTML))
	if len(matches) < 2 {
		return "", fmt.Errorf("no iframe tag found in response HTML")
	}

	src := matches[1]
	if strings.HasPrefix(src, "//") {
		src = "https:" + src
	}
	return src, nil
}

func getWord(number int, base int) string {
	const alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	if number < base {
		return string(alphabet[number])
	}
	return getWord(number/base, base) + string(alphabet[number%base])
}

func unpack(p string, a int, c int, k []string) string {
	for i := c - 1; i >= 0; i-- {
		if i < len(k) && k[i] != "" {
			word := k[i]
			baseARep := getWord(i, a)
			re := regexp.MustCompile(`\b` + regexp.QuoteMeta(baseARep) + `\b`)
			p = re.ReplaceAllString(p, word)
		}
	}
	return p
}

func unpackPackerInHTML(html string) []string {
	matches := packerRegex.FindAllStringSubmatch(html, -1)
	var unpacked []string
	for _, match := range matches {
		if len(match) < 5 {
			continue
		}
		p := match[1]
		a, err1 := strconv.Atoi(match[2])
		c, err2 := strconv.Atoi(match[3])
		k := strings.Split(match[4], "|")
		if err1 != nil || err2 != nil {
			continue
		}

		p = strings.ReplaceAll(p, `\'`, `'`)
		p = strings.ReplaceAll(p, `\"`, `"`)

		unpacked = append(unpacked, unpack(p, a, c, k))
	}
	return unpacked
}

// scrapeGeneric is the universal fallback scraper for sites without a dedicated
// adapter. It searches the page HTML, then <script> tag contents, then follows
// iframes (including nested ones) up to a few levels deep. It also handles:
//   - Base64-encoded mirror selectors (common in WordPress anime sites)
//   - Blogger video embeds (blogger.com/video.g)
func scrapeGeneric(pageURL string) (*models.VideoMetadata, error) {
	doc, err := fetchPageDocument(pageURL, "")
	if err != nil {
		return nil, err
	}

	html, _ := doc.Html()

	// Layer 1: direct stream URL in page HTML (including packed JS).
	if streamURL := findStreamURLInHTML(html); streamURL != "" {
		metadata := extractGenericMetadata(doc, pageURL)
		metadata.VideoURL = streamURL
		return metadata, nil
	}

	// Layer 2: stream URL hidden inside <script> tag content (JSON configs, JS vars).
	if streamURL := findStreamURLInScripts(html); streamURL != "" {
		metadata := extractGenericMetadata(doc, pageURL)
		metadata.VideoURL = streamURL
		return metadata, nil
	}

	// Layer 2.5: parse base64-encoded mirror selectors (common in WordPress anime sites).
	// Many Indonesian anime sites use <select> or <div> with base64-encoded <iframe> values.
	mirrorOptionRegex := regexp.MustCompile(`(?:value|data-content)="([A-Za-z0-9+/=]{20,})"`)
	mirrorMatches := mirrorOptionRegex.FindAllStringSubmatch(html, -1)
	for _, m := range mirrorMatches {
		decoded, err := base64.StdEncoding.DecodeString(m[1])
		if err != nil {
			continue
		}
		decodedStr := string(decoded)
		iframeSrcMatch := iframeRegex.FindStringSubmatch(decodedStr)
		if len(iframeSrcMatch) < 2 {
			continue
		}
		iframeSrc := normalizeURL(iframeSrcMatch[1], pageURL)
		videoURL, err := resolveEmbedURL(iframeSrc, pageURL)
		if err == nil && videoURL != "" {
			metadata := extractGenericMetadata(doc, pageURL)
			metadata.VideoURL = videoURL
			return metadata, nil
		}
	}

	// Layer 3: collect all iframes and search each one (recursively, up to 3 levels).
	var iframeSrcs []string
	doc.Find("iframe").Each(func(_ int, s *goquery.Selection) {
		if src, exists := s.Attr("src"); exists && src != "" {
			iframeSrcs = append(iframeSrcs, normalizeURL(src, pageURL))
		}
	})

	for _, iframeSrc := range iframeSrcs {
		// Try resolving via the embed URL resolver (handles Blogger, etc.)
		videoURL, err := resolveEmbedURL(iframeSrc, pageURL)
		if err == nil && videoURL != "" {
			metadata := extractGenericMetadata(doc, pageURL)
			metadata.VideoURL = videoURL
			return metadata, nil
		}

		// Fallback: deep search for stream URLs in nested iframes
		streamURL, err := deepSearchStream(iframeSrc, pageURL, 3)
		if err == nil && streamURL != "" {
			metadata := extractGenericMetadata(doc, pageURL)
			metadata.VideoURL = streamURL
			return metadata, nil
		}
	}

	return nil, fmt.Errorf("could not find stream URL from %s", pageURL)
}

// scrapeWithHeadless uses stealth full-headless Chrome (via chromedp) with
// network interception as the primary stream discovery path. Static DOM
// parsing is only a fallback.
func scrapeWithHeadless(pageURL string) (*models.VideoMetadata, error) {
	cap, renderedHTML, err := stealthNavigate(pageURL, defaultSessionTimeout)
	if err != nil && cap == nil {
		return nil, fmt.Errorf("headless render failed for %s: %w", pageURL, err)
	}
	if err != nil {
		// Abort-on-match: context.Canceled with stream URL is success (Samehadaku ads killed session)
		if isAbortSuccess(err, cap) {
			log.Printf("[Headless] Abort-on-match SUCCESS for %s: %v", pageURL, err)
			err = nil
		} else if cap.bestStream() != "" || cap.hasUsefulCapture() {
			log.Printf("[Headless] Soft-success for %s after error (network capture hit): %v", pageURL, err)
			err = nil
		} else {
			log.Printf("[Headless] Partial navigation error for %s: %v", pageURL, err)
		}
	}

	metadata := &models.VideoMetadata{Source: extractDomain(pageURL)}
	if renderedHTML != "" {
		if doc, e := goquery.NewDocumentFromReader(strings.NewReader(renderedHTML)); e == nil {
			metadata = extractGenericMetadata(doc, pageURL)
		}
	}

	// Priority 1: network-intercepted stream URLs (primary)
	if streamURL := cap.bestStream(); streamURL != "" {
		metadata.VideoURL = streamURL
		return metadata, nil
	}

	// Priority 2: rendered HTML / scripts (fallback only)
	if streamURL := findStreamURLInHTML(renderedHTML); streamURL != "" {
		metadata.VideoURL = streamURL
		return metadata, nil
	}
	if streamURL := findStreamURLInScripts(renderedHTML); streamURL != "" {
		metadata.VideoURL = streamURL
		return metadata, nil
	}

	// Priority 3: follow iframes rendered by JS
	for _, iframeSrc := range extractIframeSrcs(renderedHTML) {
		iframeSrc = normalizeURL(iframeSrc, pageURL)
		videoURL, err := resolveEmbedURL(iframeSrc, pageURL)
		if err == nil && videoURL != "" {
			metadata.VideoURL = videoURL
			return metadata, nil
		}
		streamURL, err := deepSearchStream(iframeSrc, pageURL, 3)
		if err == nil && streamURL != "" {
			metadata.VideoURL = streamURL
			return metadata, nil
		}
	}

	return nil, fmt.Errorf("could not find stream URL from %s after headless render", pageURL)
}

// deepSearchStream recursively fetches iframe/page content (with referer set to
// the parent URL) and searches HTML, then scripts, then nested iframes, up to
// maxDepth levels. It also handles Blogger video pages specially.
func deepSearchStream(streamURL, referer string, maxDepth int) (string, error) {
	if maxDepth <= 0 {
		return "", fmt.Errorf("max iframe depth reached")
	}

	// If this is a Blogger video URL, use the dedicated extractor.
	if isBloggerVideoURL(streamURL) {
		videoURL, err := extractBloggerVideoURL(streamURL, referer)
		if err == nil && videoURL != "" {
			return videoURL, nil
		}
		// If Blogger extraction fails, continue with generic search below.
	}

	html, err := fetchPageHTMLWithRefererChain(streamURL, []string{referer, ""})
	if err != nil {
		return "", err
	}

	if found := findStreamURLInHTML(html); found != "" {
		return found, nil
	}

	if found := findStreamURLInScripts(html); found != "" {
		return found, nil
	}

	for _, iframeSrc := range extractIframeSrcs(html) {
		iframeSrc = normalizeURL(iframeSrc, streamURL)
		if result, err := deepSearchStream(iframeSrc, streamURL, maxDepth-1); err == nil && result != "" {
			return result, nil
		}
	}

	return "", fmt.Errorf("no stream found at depth %d", maxDepth)
}

// findStreamURLInScripts looks for stream URLs commonly embedded inside <script>
// tag content as JSON configs or JS variable assignments (jwplayer-style setups).
func findStreamURLInScripts(html string) string {
	// Pattern: "file":"https://...m3u8"
	if match := scriptFileRegex.FindStringSubmatch(html); len(match) > 1 {
		return match[1]
	}

	// Pattern: "sources":[{"file":"..."}]
	if match := scriptSourcesRegex.FindStringSubmatch(html); len(match) > 1 {
		return match[1]
	}

	// Pattern: source: "https://...m3u8"
	if match := scriptSourceRegex.FindStringSubmatch(html); len(match) > 1 {
		return match[1]
	}

	// Pattern: var videoUrl = "https://..."
	if match := scriptVarVideoRegex.FindStringSubmatch(html); len(match) > 1 {
		return match[1]
	}

	// Pattern: "src":"https://...m3u8"
	srcRegex := regexp.MustCompile(`"src"\s*:\s*"(https?://[^"]+?\.m3u8[^"]*)"`)
	if match := srcRegex.FindStringSubmatch(html); len(match) > 1 {
		return match[1]
	}

	// Pattern: "url":"https://...m3u8"
	urlRegex := regexp.MustCompile(`"url"\s*:\s*"(https?://[^"]+?\.m3u8[^"]*)"`)
	if match := urlRegex.FindStringSubmatch(html); len(match) > 1 {
		return match[1]
	}

	// Pattern: src: "https://...m3u8" (unquoted key)
	srcUnquotedRegex := regexp.MustCompile(`src\s*[:=]\s*["'](https?://[^"']+?\.m3u8[^"']*)["']`)
	if match := srcUnquotedRegex.FindStringSubmatch(html); len(match) > 1 {
		return match[1]
	}

	// Pattern: window.__PLAYURL__ = "https://..." (SPA globals)
	globalVarRegex := regexp.MustCompile(`window\.__\w+__\s*=\s*["'](https?://[^"']+)["']`)
	if match := globalVarRegex.FindStringSubmatch(html); len(match) > 1 {
		return match[1]
	}

	return ""
}

// extractIframeSrcs returns all iframe src attributes found in raw HTML.
func extractIframeSrcs(html string) []string {
	matches := iframeRegex.FindAllStringSubmatch(html, -1)
	srcs := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) > 1 {
			srcs = append(srcs, m[1])
		}
	}
	return srcs
}

// normalizeURL resolves a possibly-relative URL (protocol-relative, absolute
// path, or bare path) against baseURL.
func normalizeURL(rawURL, baseURL string) string {
	if strings.HasPrefix(rawURL, "//") {
		return "https:" + rawURL
	}
	if strings.HasPrefix(rawURL, "/") {
		parsed, err := url.Parse(baseURL)
		if err != nil {
			return rawURL
		}
		return parsed.Scheme + "://" + parsed.Host + rawURL
	}
	if !strings.HasPrefix(rawURL, "http") {
		parsed, err := url.Parse(baseURL)
		if err != nil {
			return rawURL
		}
		return parsed.Scheme + "://" + parsed.Host + "/" + rawURL
	}
	return rawURL
}

// maxScrapeBodyBytes caps HTML/body reads to prevent OOM from huge responses.
const maxScrapeBodyBytes = 5 * 1024 * 1024 // 5 MB

func fetchPageDocument(pageURL, referer string) (*goquery.Document, error) {
	resp, err := fetchWithRetry(pageURL, referer, 3)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, pageURL)
	}

	return goquery.NewDocumentFromReader(io.LimitReader(resp.Body, maxScrapeBodyBytes))
}

func fetchPageHTML(pageURL, referer string) (string, error) {
	resp, err := fetchWithRetry(pageURL, referer, 3)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("HTTP %d from %s", resp.StatusCode, pageURL)
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxScrapeBodyBytes))
	if err != nil {
		return "", err
	}
	return string(bodyBytes), nil
}

// fetchPageHTMLWithRefererChain tries fetching url with each referer in order,
// returning the first successful response. This helps work around CDNs that
// return 403 for the wrong (or missing) referer.
func fetchPageHTMLWithRefererChain(url string, referers []string) (string, error) {
	var lastErr error
	for _, ref := range referers {
		html, err := fetchPageHTML(url, ref)
		if err == nil {
			return html, nil
		}
		lastErr = err
	}
	return "", lastErr
}

func findStreamURLInHTML(html string) string {
	if match := m3u8Regex.FindString(html); match != "" {
		return match
	}

	unpacked := unpackPackerInHTML(html)
	for _, script := range unpacked {
		if match := m3u8Regex.FindString(script); match != "" {
			return match
		}
		if match := mp4Regex.FindString(script); match != "" {
			return match
		}
	}

	if match := mp4Regex.FindString(html); match != "" {
		return match
	}

	return ""
}

func extractDomain(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return parsed.Host
}

func cleanEpisodeString(raw string) string {
	raw = strings.TrimSpace(raw)
	lower := strings.ToLower(raw)

	if idx := strings.Index(lower, "subtitle"); idx != -1 {
		raw = strings.TrimSpace(raw[:idx])
	}
	if idx := strings.Index(lower, "sub indo"); idx != -1 {
		raw = strings.TrimSpace(raw[:idx])
	}

	if !strings.Contains(strings.ToLower(raw), "episode") {
		re := regexp.MustCompile(`(?i)(?:ep\.?\s*|episode\s*)(\d+)`)
		if match := re.FindStringSubmatch(raw); len(match) > 1 {
			return "Episode " + match[1]
		}
	}

	return raw
}

func extractOtakudesuMetadata(doc *goquery.Document, pageURL string) *models.VideoMetadata {
	metadata := &models.VideoMetadata{Source: "otakudesu.blog"}

	title := doc.Find(".venutama .post-title h1").First().Text()
	if title == "" {
		title = doc.Find(".fotoanime").Next().Find("h1").First().Text()
	}
	if title == "" {
		title = doc.Find(".entry-content h1").First().Text()
	}
	metadata.Title = strings.TrimSpace(title)

	episodeText := doc.Find(".venutama .epztitle").First().Text()
	if episodeText == "" {
		episodeText = doc.Find("h1.entry-title").First().Text()
	}
	if episodeText != "" {
		metadata.Episode = cleanEpisodeString(episodeText)
	}

	thumbnail, exists := doc.Find(".fotoanime img").First().Attr("src")
	if exists && thumbnail != "" {
		if strings.HasPrefix(thumbnail, "//") {
			thumbnail = "https:" + thumbnail
		}
		metadata.ThumbnailURL = thumbnail
	}

	if metadata.Title == "" {
		ogTitle := doc.Find("meta[property='og:title']").AttrOr("content", "")
		if ogTitle != "" {
			metadata.Title = ogTitle
		} else {
			metadata.Title = doc.Find("title").First().Text()
		}
	}

	metadata.NextEpisodeURL = extractNextEpisodeURL(doc, pageURL)
	return metadata
}

func extractGenericMetadata(doc *goquery.Document, pageURL string) *models.VideoMetadata {
	metadata := &models.VideoMetadata{Source: extractDomain(pageURL)}

	ogTitle := doc.Find("meta[property='og:title']").AttrOr("content", "")
	if ogTitle != "" {
		metadata.Title = ogTitle
	} else {
		metadata.Title = doc.Find("title").First().Text()
	}

	ogImage := doc.Find("meta[property='og:image']").AttrOr("content", "")
	if ogImage != "" {
		if strings.HasPrefix(ogImage, "//") {
			ogImage = "https:" + ogImage
		}
		metadata.ThumbnailURL = ogImage
	}

	metadata.NextEpisodeURL = extractNextEpisodeURL(doc, pageURL)
	return metadata
}

func extractNextEpisodeURL(doc *goquery.Document, pageURL string) string {
	if doc == nil {
		return ""
	}
	var nextLink string
	// 1. rel="next"
	doc.Find("a[rel='next']").Each(func(_ int, s *goquery.Selection) {
		if nextLink == "" {
			if href, ok := s.Attr("href"); ok {
				href = strings.TrimSpace(href)
				if href != "" && href != "#" && !strings.HasPrefix(href, "javascript:") {
					nextLink = href
				}
			}
		}
	})
	// 2. Class containing next / naveps / next-episode / flir
	if nextLink == "" {
		doc.Find(".nvs.rght a, .naveps .next a, a.next, a.next-episode, .flir a").Each(func(_ int, s *goquery.Selection) {
			if nextLink == "" {
				text := strings.ToLower(s.Text())
				if strings.Contains(text, "next") || strings.Contains(text, "berikut") || strings.Contains(text, "selanjutnya") {
					if href, ok := s.Attr("href"); ok {
						href = strings.TrimSpace(href)
						if href != "" && href != "#" && !strings.HasPrefix(href, "javascript:") {
							nextLink = href
						}
					}
				}
			}
		})
	}
	// 3. Any <a> tag with text matching Next or Episode Berikutnya
	if nextLink == "" {
		doc.Find("a").Each(func(_ int, s *goquery.Selection) {
			if nextLink == "" {
				text := strings.TrimSpace(strings.ToLower(s.Text()))
				if text == "next" || text == "next eps" || text == "next episode" || strings.Contains(text, "eps berikutnya") || strings.Contains(text, "episode berikutnya") {
					if href, ok := s.Attr("href"); ok {
						href = strings.TrimSpace(href)
						if href != "" && href != "#" && !strings.HasPrefix(href, "javascript:") {
							nextLink = href
						}
					}
				}
			}
		})
	}
	if nextLink != "" {
		if u, err := url.Parse(nextLink); err == nil {
			if base, err := url.Parse(pageURL); err == nil {
				return base.ResolveReference(u).String()
			}
		}
		return nextLink
	}
	return ""
}
