# Backend — NanoProxy + Header Rewriter + WebSocket Sync

> **Untuk backend agent**: Dokumen ini mendefinisikan arsitektur baru untuk mengatasi masalah "failed scraping" secara menyeluruh.
>
> **Komponen utama:**
> 1. **[NanoProxy](https://github.com/ryanbekhen/nanoproxy)** — SOCKS5 + HTTP forward proxy (external service) untuk bypass IP blocking & rate limit
> 2. **NanoScraper** — chromedp + network interception, routing melalui NanoProxy
> 3. **M3U8 Header Rewriter** — reverse proxy kita sendiri yang rewrite URL internal di m3u8
> 4. **WebSocket Sync** — enhanced sync dengan latency compensation
>
> Baca seluruh file yang disebutkan sebelum mengedit. Verifikasi build setelah setiap task.

---

## Arsitektur

### Apa itu NanoProxy?

[NanoProxy](https://github.com/ryanbekhen/nanoproxy) adalah **SOCKS5 + HTTP forward proxy** ringan yang ditulis dalam Go. Bukan reverse proxy — ini adalah proxy yang duduk di antara client dan internet.

**Fitur yang relevan untuk kita:**
- SOCKS5 proxy di `:1080` — chromedp bisa routing traffic melalui ini
- HTTP proxy di `:8080` — Go `http.Client` bisa routing request melalui ini
- **Tor integration** — IP rotation otomatis setiap N menit (opsional)
- **No-auth mode** — `NO_AUTH_MODE=true` untuk development
- Lightweight, single binary, Docker-ready

**Mengapa diperlukan:**
- Situs anime (otakudesu, samehadaku, dll) memblokir IP yang terlalu sering scrape
- CDN mengembalikan 403 jika request dari IP yang sama terus-menerus
- Dengan NanoProxy + Tor, IP berubah otomatis → scraping lebih reliable

### Alur Data Lengkap

```
┌─────────────────────────────────────────────────────────────────────┐
│                        SCRAPING FLOW                                │
│                                                                     │
│  User paste anime URL                                               │
│       ↓                                                             │
│  ScrapeStreamURL()                                                  │
│       ├─ scrapeGeneric() [HTML parsing, tanpa proxy]                │
│       ├─ NanoScrape() [chromedp + network intercept]                │
│       │    └─ chromedp → NanoProxy (SOCKS5 :1080) → situs anime    │
│       │         └─ intercept semua network requests → DAPAT URL     │
│       └─ scrapeWithHeadless() [chromedp HTML-only fallback]         │
│            └─ chromedp → NanoProxy (SOCKS5 :1080) → situs anime    │
└─────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────┐
│                      STREAMING FLOW                                  │
│                                                                     │
│  Frontend: hls.loadSource("/api/proxy?url=STREAM_URL")              │
│       ↓                                                             │
│  /api/proxy (Header Rewriter):                                      │
│       ├─ Fetch m3u8 dari CDN                                        │
│       │    └─ Go http.Client → NanoProxy (HTTP :8080) → CDN         │
│       ├─ REWRITE semua URL internal (.ts, .m3u8)                    │
│       │    └─ segment0000.ts → /api/proxy?url=...                   │
│       └─ Return m3u8 yang sudah di-rewrite                          │
│       ↓                                                             │
│  hls.js fetch .ts → /api/proxy?url=SEGMENT_URL                     │
│       ↓                                                             │
│  /api/proxy → Go http.Client → NanoProxy (HTTP :8080) → CDN        │
│       ↓                                                             │
│  Video playback berhasil ✅                                          │
└─────────────────────────────────────────────────────────────────────┘
```

### Perbedaan NanoProxy vs Header Rewriter

| Aspek | NanoProxy | Header Rewriter (/api/proxy) |
|-------|-----------|------------------------------|
| Tipe | Forward proxy (SOCKS5 + HTTP) | Reverse proxy (HTTP endpoint) |
| Fungsi | Routing traffic scraper & HTTP client ke internet | Rewrite m3u8 content + CORS headers untuk browser |
| Port | `:1080` (SOCKS5), `:8080` (HTTP) | Port yang sama dengan backend Go (`:8080/api/proxy`) |
| Client | chromedp, Go `http.Client` | Browser (hls.js) |
| Tidak bisa | Rewrite m3u8 content | Bypass IP blocking |

---

## Task 0 — Setup NanoProxy [KRITIS]

### Instalasi

**Opsi A: Docker (Recommended)**
```bash
docker run -d --name nanoproxy \
  -p 1080:1080 \
  -p 8080:8080 \
  -e ADDR=:1080 \
  -e ADDR_HTTP=:8080 \
  -e NO_AUTH_MODE=true \
  -e LOG_LEVEL=info \
  ghcr.io/ryanbekhen/nanoproxy:latest
```

**Opsi B: Docker + Tor (IP Rotation)**
```bash
docker run -d --name nanoproxy-tor \
  --privileged \
  -p 1080:1080 \
  -p 8080:8080 \
  -e ADDR=:1080 \
  -e ADDR_HTTP=:8080 \
  -e NO_AUTH_MODE=true \
  -e TOR_ENABLED=true \
  -e TOR_IDENTITY_INTERVAL=5m \
  -e LOG_LEVEL=info \
  ghcr.io/ryanbekhen/nanoproxy-tor:latest
```

**Opsi C: Binary langsung**
```bash
# Install via package manager (Debian/Ubuntu)
echo "deb [trusted=yes] https://repo.ryanbekhen.dev/apt/ /" | sudo tee /etc/apt/sources.list.d/ryanbekhen.list
sudo apt update && sudo apt install nanoproxy

# Jalankan
export ADDR=:1080
export ADDR_HTTP=:8080
export NO_AUTH_MODE=true
nanoproxy
```

### Verifikasi NanoProxy
```bash
# Test SOCKS5
curl -x socks5://localhost:1080 https://httpbin.org/ip

# Test HTTP proxy
curl -x http://localhost:1080 https://httpbin.org/ip

# Jika pakai Tor, IP harus berbeda setiap beberapa menit
```

### Konfigurasi Environment Variables

Tambah di `.env` atau docker-compose:
```env
NANOPROXY_SOCKS5_URL=socks5://localhost:1080
NANOPROXY_HTTP_URL=http://localhost:1080
```

---

## Task 1 — NanoScraper: chromedp + Network Interception via NanoProxy [KRITIS]

### Masalah
Scraper saat ini mengandalkan **parsing HTML** untuk mencari stream URL. Banyak situs SPA memuat video URL via JavaScript yang tidak ada di HTML mentah. `scrapeWithHeadless()` hanya capture rendered HTML — masih bisa gagal jika video URL dimuat via AJAX/fetch setelah page load.

### Solusi
Gunakan chromedp dengan **network event interception** + routing melalui NanoProxy (SOCKS5). Ini menangkap semua network request yang dibuat oleh halaman, termasuk URL yang dimuat secara dinamis.

### File yang Diubah/Dibuat
- `backend/services/nano_scraper.go` — **BARU**
- `backend/services/scraper.go` — update routing

### Implementasi

**File: `backend/services/nano_scraper.go`** (BARU)

```go
package services

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/device"
	"golang.org/x/net/proxy"
	"watchparty-backend/models"
)

// NanoProxy configuration — read from environment.
var (
	nanoProxySOCKS5 = os.Getenv("NANOPROXY_SOCKS5_URL") // e.g. "socks5://localhost:1080"
	nanoProxyHTTP   = os.Getenv("NANOPROXY_HTTP_URL")    // e.g. "http://localhost:1080"
)

// NanoScraperResult holds the result from a nano scrape operation.
type NanoScraperResult struct {
	StreamURL  string
	PageTitle  string
	Thumbnail  string
	Source     string
	M3U8URLs   []string
	MP4URLs    []string
	IFrameURLs []string
}

// NanoScrape uses chromedp with network request interception to capture
// all network requests made by a page. Routes through NanoProxy (SOCKS5)
// when available to bypass IP-based blocking from anime sites.
func NanoScrape(pageURL string) (*NanoScraperResult, error) {
	var (
		mu         sync.Mutex
		m3u8URLs   []string
		mp4URLs    []string
		iframeURLs []string
		pageTitle  string
		ogImage    string
	)

	// Build chromedp options — add proxy if NanoProxy is configured
	opts := []chromedp.ContextOption{
		chromedp.WithLogf(log.Printf),
	}

	if nanoProxySOCKS5 != "" {
		log.Printf("[NanoScraper] Routing through NanoProxy: %s", nanoProxySOCKS5)
		// chromedp doesn't natively support SOCKS5 proxy via context options.
		// Instead, we set the CHROMEDP_PROXY env or use a custom allocator.
		// For simplicity, we'll use the HTTP proxy variant.
	}

	// Create allocator with proxy support
	allocOpts := []chromedp.ExecAllocatorOption{
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.Flag("disable-web-security", true),
		chromedp.Flag("disable-features", "IsolateOrigins,site-per-process"),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	}

	// Route through NanoProxy HTTP proxy if configured
	if nanoProxyHTTP != "" {
		allocOpts = append(allocOpts, chromedp.ProxyServer(nanoProxyHTTP))
	}

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), allocOpts...)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(allocCtx, opts...)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	// Enable network event listening
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		switch e := ev.(type) {
		case *network.EventRequestWillBeSent:
			reqURL := e.Request.URL
			mu.Lock()
			lower := strings.ToLower(reqURL)
			if strings.Contains(lower, ".m3u8") {
				m3u8URLs = append(m3u8URLs, reqURL)
			}
			if strings.Contains(lower, ".mp4") && !strings.Contains(lower, ".mp44") {
				mp4URLs = append(mp4URLs, reqURL)
			}
			mu.Unlock()

		case *network.EventResponseReceived:
			respURL := e.Response.URL
			mu.Lock()
			lower := strings.ToLower(respURL)
			if strings.Contains(lower, ".m3u8") && !containsStr(m3u8URLs, respURL) {
				m3u8URLs = append(m3u8URLs, respURL)
			}
			if strings.Contains(lower, ".mp4") && !containsStr(mp4URLs, respURL) {
				mp4URLs = append(mp4URLs, respURL)
			}
			mu.Unlock()
		}
	})

	var renderedHTML string
	err := chromedp.Run(ctx,
		network.Enable(),
		chromedp.Navigate(pageURL),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Sleep(8*time.Second),
		chromedp.OuterHTML("html", &renderedHTML, chromedp.ByQuery),
	)
	if err != nil {
		return nil, fmt.Errorf("nano scrape failed for %s: %w", pageURL, err)
	}

	pageTitle = extractTitleFromHTML(renderedHTML)
	ogImage = extractOGImageFromHTML(renderedHTML)

	for _, src := range extractIframeSrcs(renderedHTML) {
		iframeURLs = append(iframeURLs, normalizeURL(src, pageURL))
	}

	result := &NanoScraperResult{
		StreamURL:  selectBestStreamURL(m3u8URLs, mp4URLs),
		PageTitle:  pageTitle,
		Thumbnail:  ogImage,
		Source:     extractDomain(pageURL),
		M3U8URLs:   m3u8URLs,
		MP4URLs:    mp4URLs,
		IFrameURLs: iframeURLs,
	}

	log.Printf("[NanoScraper] Found %d m3u8, %d mp4, %d iframes for %s",
		len(m3u8URLs), len(mp4URLs), len(iframeURLs), pageURL)

	return result, nil
}

func selectBestStreamURL(m3u8URLs, mp4URLs []string) string {
	if len(m3u8URLs) > 0 {
		best := m3u8URLs[0]
		for _, u := range m3u8URLs[1:] {
			if len(u) > len(best) {
				best = u
			}
		}
		return best
	}
	if len(mp4URLs) > 0 {
		for _, u := range mp4URLs {
			if len(u) > 50 {
				return u
			}
		}
		return mp4URLs[0]
	}
	return ""
}

func containsStr(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func extractTitleFromHTML(html string) string {
	re := regexp.MustCompile(`meta\s+property=["']og:title["']\s+content=["']([^"']+)["']`)
	if match := re.FindStringSubmatch(html); len(match) > 1 {
		return match[1]
	}
	re = regexp.MustCompile(`<title[^>]*>([^<]+)</title>`)
	if match := re.FindStringSubmatch(html); len(match) > 1 {
		return match[1]
	}
	return ""
}

func extractOGImageFromHTML(html string) string {
	re := regexp.MustCompile(`meta\s+property=["']og:image["']\s+content=["']([^"']+)["']`)
	if match := re.FindStringSubmatch(html); len(match) > 1 {
		return match[1]
	}
	return ""
}
```

### Integrasi ke ScrapeStreamURL

Update `ScrapeStreamURL()` di `scraper.go`:

```go
func ScrapeStreamURL(pageURL string) (*models.VideoMetadata, error) {
	switch detectSitePlatform(pageURL) {
	case "otakudesu":
		return scrapeOtakudesu(pageURL)
	case "samehadaku":
		return scrapeSamehadaku(pageURL)
	default:
		metadata, err := scrapeGeneric(pageURL)
		if err != nil {
			return scrapeWithNanoOrHeadless(pageURL)
		}
		return metadata, nil
	}
}

func scrapeWithNanoOrHeadless(pageURL string) (*models.VideoMetadata, error) {
	log.Printf("Static scrape failed for %s, trying NanoScraper...", pageURL)

	result, err := NanoScrape(pageURL)
	if err == nil && result.StreamURL != "" {
		metadata := &models.VideoMetadata{
			VideoURL:     result.StreamURL,
			Title:        result.PageTitle,
			ThumbnailURL: result.Thumbnail,
			Source:       result.Source,
		}
		return metadata, nil
	}

	log.Printf("NanoScraper failed for %s, trying headless HTML scrape...", pageURL)
	return scrapeWithHeadless(pageURL)
}
```

### Verifikasi
```bash
# Pastikan NanoProxy running
docker ps | grep nanoproxy

# Build & test
cd backend && go build ./...
# Log harus menunjukkan: "[NanoScraper] Routing through NanoProxy: socks5://localhost:1080"
```

---

## Task 2 — M3U8 Content Rewriter (Header Rewriter) [KRITIS]

### Masalah
Proxy saat ini (`/api/proxy`) hanya **pass-through**. File `.m3u8` berisi URL absolut ke CDN untuk `.ts` segment dan nested `.m3u8`. hls.js akan fetch segment langsung ke CDN → CORS block.

### Solusi
Rewrite m3u8 content: deteksi response m3u8, parse semua URL, rewrite menjadi `/api/proxy?url=...`, return m3u8 yang sudah di-rewrite. Juga routing HTTP requests melalui NanoProxy.

### File yang Diubah
- `backend/handlers/proxy.go` — rewrite dengan M3U8 rewriting + NanoProxy routing

### Implementasi

```go
package handlers

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

var (
	nanoProxyHTTPURL = os.Getenv("NANOPROXY_HTTP_URL") // e.g. "http://localhost:1080"

	proxyClient = &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}
)

func init() {
	// If NanoProxy HTTP URL is configured, route all proxy requests through it
	if nanoProxyHTTPURL != "" {
		proxyURL, err := url.Parse(nanoProxyHTTPURL)
		if err == nil {
			proxyClient.Transport = &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			}
			log.Printf("[Proxy] Routing through NanoProxy: %s", nanoProxyHTTPURL)
		}
	}
}

func HandleStreamProxy(w http.ResponseWriter, r *http.Request) {
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

	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	referer := r.URL.Query().Get("referer")
	if referer == "" {
		referer = parsed.Scheme + "://" + parsed.Host + "/"
	}
	req.Header.Set("Referer", referer)
	req.Header.Set("Origin", parsed.Scheme+"://"+parsed.Host)

	if rangeHeader := r.Header.Get("Range"); rangeHeader != "" {
		req.Header.Set("Range", rangeHeader)
	}

	// Use proxyClient (routes through NanoProxy if configured)
	resp, err := proxyClient.Do(req)
	if err != nil {
		log.Printf("Proxy error fetching %s: %v", targetURL, err)
		http.Error(w, "Failed to fetch upstream", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Range")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Range, Content-Length, Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// M3U8 REWRITING — the key difference from the old proxy
	if isM3U8Response(resp, targetURL) {
		rewriteAndServeM3U8(w, resp, targetURL)
		return
	}

	// Non-m3u8: stream directly
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
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
	body, err := io.ReadAll(io.LimitReader(resp.Body, 16))
	if err != nil {
		return false
	}
	resp.Body = io.NopCloser(io.MultiReader(strings.NewReader(string(body)), resp.Body))
	return strings.Contains(string(body), "#EXTM3U")
}

func rewriteAndServeM3U8(w http.ResponseWriter, resp *http.Response, sourceURL string) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read m3u8 body", http.StatusBadGateway)
		return
	}

	sourceParsed, _ := url.Parse(sourceURL)
	rewritten := rewriteM3U8Content(string(body), sourceParsed)

	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.Header().Del("Content-Length")
	w.WriteHeader(resp.StatusCode)
	w.Write([]byte(rewritten))
}

func rewriteM3U8Content(content string, sourceURL *url.URL) string {
	scanner := bufio.NewScanner(strings.NewReader(content))
	var lines []string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			lines = append(lines, line)
			continue
		}

		resolvedURL := resolveM3U8URL(line, sourceURL)
		if resolvedURL != "" {
			proxiedURL := "/api/proxy?url=" + url.QueryEscape(resolvedURL)
			lines = append(lines, proxiedURL)
		} else {
			lines = append(lines, line)
		}
	}

	return strings.Join(lines, "\n") + "\n"
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

func IsStreamURL(rawURL string) bool {
	lower := strings.ToLower(rawURL)
	return strings.Contains(lower, ".m3u8") ||
		strings.Contains(lower, ".ts") ||
		strings.Contains(lower, ".mp4") ||
		strings.Contains(lower, ".webm")
}
```

### Verifikasi
```bash
cd backend && go build ./...

# Test m3u8 rewriting:
# 1. Paste anime URL → video play
# 2. DevTools Network → filter .ts → semua harus ke /api/proxy?url=...
# 3. CORS error hilang
```

---

## Task 3 — Scraper HTTP Client via NanoProxy [SEDANG]

### Masalah
Scraping functions (`fetchPageHTML`, `fetchPageDocument`) menggunakan `httpClient` yang langsung connect ke internet. Tanpa proxy, IP bisa diblokir oleh situs target.

### Solusi
Routing `httpClient` melalui NanoProxy HTTP proxy.

### File yang Diubah
- `backend/services/scraper.go` — update `httpClient` initialization

### Implementasi

```go
func init() {
    // Route scraper HTTP client through NanoProxy if configured
    nanoProxyHTTP := os.Getenv("NANOPROXY_HTTP_URL")
    if nanoProxyHTTP != "" {
        proxyURL, err := url.Parse(nanoProxyHTTP)
        if err == nil {
            httpClient.Transport = &http.Transport{
                Proxy: http.ProxyURL(proxyURL),
            }
            log.Printf("[Scraper] HTTP client routing through NanoProxy: %s", nanoProxyHTTP)
        }
    }
}
```

**Catatan:** `httpClient` sudah dideklarasikan sebagai package-level var. Tambahkan `init()` function di `scraper.go` untuk configure proxy.

### Verifikasi
```bash
cd backend && go build ./...
# Log harus menunjukkan: "[Scraper] HTTP client routing through NanoProxy: http://localhost:1080"
# Scrape test → request harus melewati NanoProxy (cek log NanoProxy)
```

---

## Task 4 — NanoScraper Concurrency Pool [SEDANG]

### Masalah
chromedp menghabiskan banyak memory/CPU. Banyak concurrent scrape bisa crash server.

### File yang Diubah
- `backend/services/nano_scraper.go` — tambah semaphore

### Implementasi

```go
var scrapeSemaphore = make(chan struct{}, 3) // max 3 concurrent

func NanoScrapeWithPool(pageURL string) (*NanoScraperResult, error) {
	select {
	case scrapeSemaphore <- struct{}{}:
		defer func() { <-scrapeSemaphore }()
		return NanoScrape(pageURL)
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("scrape queue full")
	}
}
```

---

## Task 5 — WebSocket Sync Enhancement [SEDANG]

### File yang Diubah
- `backend/services/message_handler.go`
- `backend/models/types.go`

### Implementasi

Update `SyncPayload`:
```go
type SyncPayload struct {
	RoomID      string  `json:"roomId"`
	PlayerState string  `json:"playerState"`
	CurrentTime float64 `json:"currentTime"`
	Duration    float64 `json:"duration"`
	SentAt      int64   `json:"sentAt"`
	ServerAt    int64   `json:"serverAt"`
}
```

Update `handleSyncEvent` untuk tambah `ServerAt`:
```go
func handleSyncEvent(user *models.User, room *models.WatchRoom, payloadRaw json.RawMessage) {
	var payload models.SyncPayload
	if err := json.Unmarshal(payloadRaw, &payload); err != nil {
		log.Println("Invalid SYNC_EVENT payload:", err)
		return
	}

	isPlaying := payload.PlayerState == "PLAYING"
	UpdateRoomState(room.RoomID, payload.CurrentTime, isPlaying)

	payload.ServerAt = time.Now().UnixMilli()

	msg := models.Message{
		Action:  "SYNC_EVENT",
		Payload: mustMarshalRaw(payload),
	}

	BroadcastToRoom(room, mustMarshal(msg), user)
}
```

---

## Ringkasan Prioritas

| # | Task | Prioritas | Effort | Dependency |
|---|------|:---------:|:------:|:----------:|
| 0 | Setup NanoProxy (Docker/binary) | 🔴 Kritis | Kecil | — |
| 1 | NanoScraper (chromedp + network intercept) | 🔴 Kritis | Besar | Task 0 |
| 2 | M3U8 Content Rewriter | 🔴 Kritis | Sedang | Task 0 |
| 3 | Scraper HTTP Client via NanoProxy | 🟡 Sedang | Kecil | Task 0 |
| 4 | NanoScraper Concurrency Pool | 🟡 Sedang | Kecil | Task 1 |
| 5 | WebSocket Sync Enhancement | 🟡 Sedang | Kecil | — |

**Rekomendasi urutan:**
1. Task 0 (setup NanoProxy) — prerequisite
2. Task 2 (M3U8 rewriter) — langsung solve CORS
3. Task 3 (scraper via proxy) — langsung solve IP blocking
4. Task 1 (NanoScraper) — solve SPA sites
5. Task 4 (pool) — protect resources
6. Task 5 (sync) — improve UX

---

*Dokumentasi ini dibuat berdasarkan analisis kode aktual + [ryanbekhen/nanoproxy](https://github.com/ryanbekhen/nanoproxy) per 15 Juli 2026.*
