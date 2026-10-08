package handlers

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
	"watchparty-backend/services"
	"watchparty-backend/utils"
)

const (
	maxM3U8BodyBytes    = 2 * 1024 * 1024
	maxProxyStreamBytes = 256 * 1024 * 1024
)

var (
	nanoProxyHTTPURL = os.Getenv("NANOPROXY_HTTP_URL")
	proxyPublicBase  = parsePublicBase(os.Getenv("PROXY_PUBLIC_BASE"))

	proxyDialer = &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	proxyTransport = &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           safeDialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		// Header wait only — body stream for progressive MP4 can take minutes
		ResponseHeaderTimeout: 20 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	// Timeout MUST be 0 for progressive MP4 (Sokuja ~80–100MB). A 30s overall
	// timeout aborts mid-stream and leaves the video stuck on "Loading…".
	// Client disconnect cancels via r.Context(); headers still bounded above.
	proxyClient = &http.Client{
		Timeout:   0,
		Transport: proxyTransport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return http.ErrUseLastResponse
			}
			if err := validateProxyURL(req.URL.String()); err != nil {
				return fmt.Errorf("redirect blocked: %w", err)
			}
			return nil
		},
	}

	// Exact host or domain-suffix tokens only (no ultra-generic substrings).
	proxyHostAllowlist = []string{
		"vidhide.com", "vidhidepro.com", "filedon.com", "filemoon.sx", "filemoon.to",
		"mp4upload.com", "streamtape.com", "dood.watch", "doodstream.com",
		"sbplay.org", "streamsb.net", "mixdrop.co", "upstream.to",
		"cloudfront.net", "akamaized.net", "akamaihd.net",
		"googleusercontent.com", "googlevideo.com", "blogger.com", "blogspot.com",
		"b-cdn.net", "bunnycdn.com",
		// Otakudesu / Anoboy HLS CDN (rotating subdomains: *.acek-cdn.com)
		"acek-cdn.com",
		"otakudesu.blog", "otakudesu.moe", "otakudesu.cloud",
		"anoboy.si", "anoboy.vip", "anoboy.live", "anoboy.be",
		"samehadaku.care", "samehadaku.win", "samehadaku.day", "samehadaku.how",
		"sokuja.uk",
		"animasu.me", "kuronime.vip", "nanime.tv",
		"idlix.com", "idlix.asia", "idlixku.com", "nontonanimeid.com",
		"majorplay.net", "ruangskill.space", "pancal.space",
		"streamwish.to", "filelions.com", "vidguard.to", "luluvdo.com",
		"krakenfiles.com", "pixeldrain.com", "fembed.com", "vanfem.com",
		"cybervynx.com", "rabbitstream.net", "megacloud.tv", "rapid-cloud.com",
		"shadowlandschronicles.com", "dokicloud.one", "cloudnestra.com",
		"wibufile.com", "mega.nz", "xtwap.top", "play.xtwap.top", "hls.xtwap.top",
		"youtube.com", "youtu.be", "ytimg.com", "vimeo.com",
		"tmdb.org", "themoviedb.org", "image.tmdb.org",
	}
)

func parsePublicBase(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	for _, sep := range []string{"\n", "\r", ","} {
		if i := strings.Index(raw, sep); i >= 0 {
			raw = strings.TrimSpace(raw[:i])
		}
	}
	return strings.TrimRight(raw, "/")
}

func init() {
	if nanoProxyHTTPURL != "" {
		proxyURL, err := url.Parse(nanoProxyHTTPURL)
		if err == nil {
			proxyTransport.Proxy = http.ProxyURL(proxyURL)
			log.Printf("[Proxy] Routing through NanoProxy: %s", nanoProxyHTTPURL)
		}
	}

	if extra := os.Getenv("PROXY_ALLOWLIST_EXTRA"); extra != "" {
		for _, part := range strings.Split(extra, ",") {
			part = strings.TrimSpace(strings.ToLower(part))
			if part != "" {
				proxyHostAllowlist = append(proxyHostAllowlist, part)
			}
		}
	}
}

// ~30 req/min sustained; burst absorbs HLS playlist + segment flurries.
var proxyRateLimit = utils.NewTokenBucket(30.0/60.0, 40)

func HandleStreamProxy(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Range")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Range, Content-Length, Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	ip := utils.ClientIP(r.RemoteAddr, r.Header.Get("X-Forwarded-For"), r.Header.Get("X-Real-IP"))
	if !proxyRateLimit.Allow(ip) {
		log.Printf("[Proxy] Rate limited IP %s", ip)
		http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
		return
	}

	targetURL := r.URL.Query().Get("url")
	if targetURL == "" {
		http.Error(w, "Missing 'url' query parameter", http.StatusBadRequest)
		return
	}

	parsed, err := url.Parse(targetURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	if err := validateProxyURL(targetURL); err != nil {
		log.Printf("[Proxy] Blocked SSRF attempt: %s — %v", targetURL, err)
		http.Error(w, "Forbidden: "+err.Error(), http.StatusForbidden)
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), "GET", targetURL, nil)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")

	referer := r.URL.Query().Get("referer")
	if referer == "" {
		// CDN hosts often require the anime-site origin, not the storage host
		referer = defaultProxyReferer(parsed)
	}
	req.Header.Set("Referer", referer)
	if origin := originFromReferer(referer, parsed); origin != "" {
		req.Header.Set("Origin", origin)
	}

	if rangeHeader := r.Header.Get("Range"); rangeHeader != "" {
		req.Header.Set("Range", rangeHeader)
	}

	resp, err := proxyClient.Do(req)
	if err != nil {
		log.Printf("Proxy error fetching %s: %v", targetURL, err)
		http.Error(w, "Failed to fetch upstream", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for key, values := range resp.Header {
		lk := strings.ToLower(key)
		if lk == "transfer-encoding" || lk == "connection" || lk == "keep-alive" {
			continue
		}
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Range")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Range, Content-Length, Content-Type, Accept-Ranges")
	// Help browsers seek progressive MP4 even if upstream omitted the header
	if w.Header().Get("Accept-Ranges") == "" {
		w.Header().Set("Accept-Ranges", "bytes")
	}

	if isM3U8Response(resp, targetURL) {
		rewriteAndServeM3U8(w, resp, targetURL)
		return
	}

	w.WriteHeader(resp.StatusCode)

	// Stream with immediate flushing to prevent player buffering stalls
	flusher, _ := w.(http.Flusher)
	buf := make([]byte, 32*1024) // 32KB chunks for low latency
	limitReader := io.LimitReader(resp.Body, maxProxyStreamBytes)
	for {
		n, err := limitReader.Read(buf)
		if n > 0 {
			if _, wErr := w.Write(buf[:n]); wErr != nil {
				break
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
		if err != nil {
			if err != io.EOF && r.Context().Err() == nil {
				log.Printf("[Proxy] stream copy error for %s: %v", targetURL, err)
			}
			break
		}
	}
}

// defaultProxyReferer picks a referer that CDN hosts (Sokuja storages, AceK, etc.) accept.
func defaultProxyReferer(u *url.URL) string {
	if u == nil {
		return ""
	}
	host := strings.ToLower(u.Hostname())
	// storages.sokuja.uk → https://sokuja.uk/
	if strings.Contains(host, "sokuja") {
		return "https://sokuja.uk/"
	}
	// *.acek-cdn.com is used by otakudesu mirrors
	if strings.Contains(host, "acek-cdn") {
		return "https://otakudesu.blog/"
	}
	// Blogger / Anoboy progressive streams live on googlevideo.com
	if strings.Contains(host, "googlevideo") || strings.Contains(host, "googleusercontent") {
		return "https://www.blogger.com/"
	}
	// xtwap HLS streaming requires play.xtwap.top referer
	if strings.Contains(host, "xtwap") {
		return "https://play.xtwap.top/"
	}
	return u.Scheme + "://" + u.Host + "/"
}

func originFromReferer(referer string, fallback *url.URL) string {
	if referer != "" {
		if p, err := url.Parse(referer); err == nil && p.Scheme != "" && p.Host != "" {
			return p.Scheme + "://" + p.Host
		}
	}
	if fallback != nil {
		return fallback.Scheme + "://" + fallback.Host
	}
	return ""
}

func isM3U8Response(resp *http.Response, targetURL string) bool {
	ct := resp.Header.Get("Content-Type")
	if strings.Contains(ct, "mpegurl") || strings.Contains(ct, "m3u8") {
		return true
	}
	lower := strings.ToLower(targetURL)
	if strings.HasSuffix(lower, ".m3u8") || strings.Contains(lower, ".m3u8?") {
		return true
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 32))
	if err != nil {
		return false
	}
	resp.Body = io.NopCloser(io.MultiReader(strings.NewReader(string(body)), resp.Body))
	return strings.Contains(string(body), "#EXTM3U")
}

func rewriteAndServeM3U8(w http.ResponseWriter, resp *http.Response, sourceURL string) {
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxM3U8BodyBytes))
	if err != nil {
		http.Error(w, "Failed to read m3u8 body", http.StatusBadGateway)
		return
	}

	sourceParsed, _ := url.Parse(sourceURL)
	registerHostsFromM3U8(string(body), sourceParsed)
	if sourceParsed != nil {
		services.RegisterStreamDomain(sourceParsed.Hostname())
	}

	rewritten := rewriteM3U8Content(string(body), sourceParsed)

	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.Header().Del("Content-Length")
	w.WriteHeader(resp.StatusCode)
	if _, err := w.Write([]byte(rewritten)); err != nil {
		log.Printf("[Proxy] m3u8 write error: %v", err)
	}
}

func registerHostsFromM3U8(content string, source *url.URL) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			if strings.Contains(line, "URI=\"") {
				const marker = `URI="`
				if i := strings.Index(line, marker); i >= 0 {
					start := i + len(marker)
					if end := strings.Index(line[start:], `"`); end >= 0 {
						raw := line[start : start+end]
						if resolved := resolveM3U8URL(raw, source); resolved != "" {
							services.RegisterStreamURLs(resolved)
						}
					}
				}
			}
			continue
		}
		if resolved := resolveM3U8URL(line, source); resolved != "" {
			services.RegisterStreamURLs(resolved)
		}
	}
}

func proxyRewritePrefix() string {
	if proxyPublicBase != "" {
		return proxyPublicBase + "/api/proxy?url="
	}
	return "/api/proxy?url="
}

func rewriteM3U8Content(content string, sourceURL *url.URL) string {
	scanner := bufio.NewScanner(strings.NewReader(content))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var lines []string
	prefix := proxyRewritePrefix()

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			if strings.Contains(line, "URI=\"") {
				line = rewriteM3U8TagURIs(line, sourceURL)
			}
			lines = append(lines, line)
			continue
		}

		resolvedURL := resolveM3U8URL(line, sourceURL)
		if resolvedURL != "" {
			proxiedURL := prefix + url.QueryEscape(resolvedURL)
			lines = append(lines, proxiedURL)
		} else {
			lines = append(lines, line)
		}
	}

	return strings.Join(lines, "\n") + "\n"
}

func rewriteM3U8TagURIs(line string, sourceURL *url.URL) string {
	const marker = `URI="`
	idx := strings.Index(line, marker)
	if idx < 0 {
		return line
	}
	start := idx + len(marker)
	end := strings.Index(line[start:], `"`)
	if end < 0 {
		return line
	}
	raw := line[start : start+end]
	resolved := resolveM3U8URL(raw, sourceURL)
	if resolved == "" {
		return line
	}
	proxied := proxyRewritePrefix() + url.QueryEscape(resolved)
	return line[:start] + proxied + line[start+end:]
}

func resolveM3U8URL(line string, source *url.URL) string {
	if strings.HasPrefix(line, "http://") || strings.HasPrefix(line, "https://") {
		return line
	}
	if strings.HasPrefix(line, "//") {
		return "https:" + line
	}
	if source == nil {
		return ""
	}
	resolved, err := source.Parse(line)
	if err != nil {
		return ""
	}
	return resolved.String()
}

func isBlockedIP(ip net.IP) bool {
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

func isPrivateIP(host string) bool {
	if ip := net.ParseIP(host); ip != nil {
		return isBlockedIP(ip)
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return false
	}
	for _, ip := range ips {
		if isBlockedIP(ip) {
			return true
		}
	}
	return false
}

// safeDialContext resolves the host, re-checks every IP against private ranges,
// then dials a public address only — mitigates DNS rebinding SSRF.
func safeDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}

	if ip := net.ParseIP(host); ip != nil {
		if isBlockedIP(ip) {
			return nil, fmt.Errorf("blocked private IP: %s", host)
		}
		return proxyDialer.DialContext(ctx, network, addr)
	}

	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("no IPs for host %s", host)
	}

	var lastErr error
	for _, ipAddr := range ips {
		if isBlockedIP(ipAddr.IP) {
			lastErr = fmt.Errorf("blocked private IP for %s: %s", host, ipAddr.IP)
			continue
		}
		target := net.JoinHostPort(ipAddr.IP.String(), port)
		conn, dialErr := proxyDialer.DialContext(ctx, network, target)
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

func hostMatchesAllowlistEntry(host, allowed string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	allowed = strings.ToLower(strings.TrimSpace(allowed))
	if host == "" || allowed == "" {
		return false
	}
	if host == allowed {
		return true
	}
	return strings.HasSuffix(host, "."+allowed)
}

func hostOnAllowlist(host string) bool {
	h := strings.ToLower(host)
	for _, allowed := range proxyHostAllowlist {
		if hostMatchesAllowlistEntry(h, allowed) {
			return true
		}
	}
	if services.IsDynamicallyAllowedHost(h) {
		return true
	}
	return false
}

func validateProxyURL(targetURL string) error {
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return fmt.Errorf("invalid URL")
	}

	host := parsed.Hostname()
	if host == "" {
		return fmt.Errorf("missing hostname")
	}

	blockedHosts := []string{"localhost", "127.0.0.1", "0.0.0.0", "::1", "host.docker.internal", "metadata.google.internal"}
	for _, blocked := range blockedHosts {
		if strings.EqualFold(host, blocked) {
			return fmt.Errorf("access to %s is blocked", host)
		}
	}

	if isPrivateIP(host) {
		return fmt.Errorf("access to private network (%s) is blocked", host)
	}

	if !hostOnAllowlist(host) {
		return fmt.Errorf("host %s is not on the proxy allowlist", host)
	}

	return nil
}
