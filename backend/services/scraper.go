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

// ScrapeStreamURLCached wraps ScrapeStreamURL with an in-memory TTL cache,
// singleflight stampede protection, and defensive metadata copies.
func ScrapeStreamURLCached(pageURL string) (*models.VideoMetadata, error) {
	scrapeCacheMu.RLock()
	if entry, ok := scrapeCache[pageURL]; ok && time.Now().Before(entry.expiresAt) {
		scrapeCacheMu.RUnlock()
		log.Printf("Cache hit for %s", pageURL)
		return copyVideoMetadata(entry.metadata), nil
	}
	scrapeCacheMu.RUnlock()

	v, err, _ := scrapeGroup.Do(pageURL, func() (interface{}, error) {
		scrapeCacheMu.RLock()
		if entry, ok := scrapeCache[pageURL]; ok && time.Now().Before(entry.expiresAt) {
			scrapeCacheMu.RUnlock()
			return copyVideoMetadata(entry.metadata), nil
		}
		scrapeCacheMu.RUnlock()

		metadata, err := ScrapeStreamURL(pageURL)
		if err != nil {
			return nil, err
		}

		if metadata != nil {
			RegisterStreamMetadata(metadata.VideoURL, metadata.ThumbnailURL)
		}

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
	default:
		return "generic"
	}
}

// isBloggerVideoURL returns true if the URL points to a Blogger/Google video player.
func isBloggerVideoURL(rawURL string) bool {
	return strings.Contains(rawURL, "blogger.com/video") ||
		(strings.Contains(rawURL, "blogspot.com") && strings.Contains(rawURL, "/video."))
}

// extractBloggerVideoURL fetches a Blogger video page and extracts the actual
// stream URL (MP4/M3U8) from the embedded JSON data in the page's script tags.
// Blogger stores video metadata in a JSON blob that contains "play_url" or
// direct download URLs.
func extractBloggerVideoURL(bloggerPageURL, referer string) (string, error) {
	html, err := fetchPageHTML(bloggerPageURL, referer)
	if err != nil {
		return "", fmt.Errorf("failed to fetch Blogger page: %w", err)
	}

	// Blogger video pages embed the video URL in JSON inside <script> tags.
	// Common patterns seen in Blogger video pages:
	//   "play_url":"https://...mp4"
	//   "stream_url":"https://...m3u8"
	//   "downloadUrl":"https://..."
	// The URLs may contain \u0026 for & in query params.
	bloggerPatterns := []*regexp.Regexp{
		regexp.MustCompile(`"play_url"\s*:\s*"(https?://[^"]+)"`),
		regexp.MustCompile(`"stream_url"\s*:\s*"(https?://[^"]+)"`),
		regexp.MustCompile(`"download_url"\s*:\s*"(https?://[^"]+)"`),
		regexp.MustCompile(`"downloadUrl"\s*:\s*"(https?://[^"]+)"`),
		// Blogger sometimes uses escaped JSON with \u0026
		regexp.MustCompile(`"play_url"\s*:\s*"(https?://[^"]+?\.mp4[^"]*)"`),
		regexp.MustCompile(`"play_url"\s*:\s*"(https?://[^"]+?\.m3u8[^"]*)"`),
		// Generic video source pattern in Blogger pages
		regexp.MustCompile(`var\s+_play_url\s*=\s*['"](https?://[^'"]+)['"]`),
	}

	for _, re := range bloggerPatterns {
		if match := re.FindStringSubmatch(html); len(match) > 1 {
			videoURL := match[1]
			// Unescape \u0026 → & (Go uses \u0026 in JSON for &)
			videoURL = strings.ReplaceAll(videoURL, `\u0026`, "&")
			videoURL = strings.ReplaceAll(videoURL, `\u003d`, "=")
			return videoURL, nil
		}
	}

	// Fallback: check for direct stream URLs in the Blogger page HTML
	if streamURL := findStreamURLInHTML(html); streamURL != "" {
		return streamURL, nil
	}

	// Fallback: check script tag contents
	if streamURL := findStreamURLInScripts(html); streamURL != "" {
		return streamURL, nil
	}

	return "", fmt.Errorf("could not extract video URL from Blogger page")
}

func ScrapeStreamURL(pageURL string) (*models.VideoMetadata, error) {
	switch detectSitePlatform(pageURL) {
	case "otakudesu":
		return scrapeOtakudesu(pageURL)
	case "anoboy":
		return scrapeAnoboy(pageURL)
	case "samehadaku":
		return scrapeSamehadaku(pageURL)
	case "idlix":
		// API chain (UUID → gate → claim → majorplay), not DOM/SVG scraping
		return scrapeIdlix(pageURL)
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

	return metadata
}
