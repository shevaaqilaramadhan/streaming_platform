# Backend Remaining Tasks — WatchParty Go Backend

> **Untuk backend agent**: Dokumen ini berisi semua task backend yang belum selesai, diurutkan berdasarkan prioritas. Baca `SCRAPER_FIX.md` dan `ANALYSIS_REPORT.md` untuk konteks masalah yang sudah di-diagnosa.
>
> Setiap task mandiri — bisa dikerjakan satu per satu. Verifikasi setiap task sebelum lanjut ke task berikutnya.

---

## Task 1 — Headless Browser (chromedp) untuk SPA Sites [KRITIS]

### Masalah
Situs SPA seperti **idlixku.com** mengembalikan HTML kosong (skeleton/loader). Video URL dimuat via JavaScript client-side. Scraper saat ini hanya bisa baca HTML mentah — selalu gagal dengan error:

```
could not find stream URL from https://z2.idlixku.com/movie/backrooms-2026
```

### Solusi
Gunakan `github.com/chromedp/chromedp` sebagai fallback terakhir ketika semua metode scraping lain gagal.

### Instalasi Dependency
```bash
cd backend
go get github.com/chromedp/chromedp
```

**Catatan:** Server/host harus punya Chrome atau Chromium terinstall. Untuk Docker/CI, gunakan base image `chromedp/headless-shell`.

### File yang Diubah
- `backend/services/scraper.go` — tambah fungsi `scrapeWithHeadless()`
- `backend/go.mod` — dependency baru (otomatis dari `go get`)

### Implementasi

Tambahkan di `backend/services/scraper.go`:

```go
import (
    "context"
    "github.com/chromedp/chromedp"
)

// scrapeWithHeadless uses a headless Chrome browser to render the page
// (executing JavaScript), then searches the rendered HTML for stream URLs.
// This is the last-resort fallback for SPA sites (Next.js, React, etc.)
// where the video URL is loaded dynamically via client-side JS.
//
// Requires Chrome/Chromium installed on the host system.
func scrapeWithHeadless(pageURL string) (*models.VideoMetadata, error) {
    // Create chromedp context with timeout
    ctx, cancel := chromedp.NewContext(context.Background(), chromedp.WithLogf(log.Printf))
    defer cancel()

    ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
    defer cancel()

    var renderedHTML string

    // Navigate to page, wait for video player elements to appear,
    // then capture the fully rendered HTML.
    err := chromedp.Run(ctx,
        chromedp.Navigate(pageURL),
        // Wait for any video-related element to appear in the DOM
        chromedp.WaitReady(`body`, chromedp.ByQuery),
        // Give JS time to execute and load video player
        chromedp.Sleep(5*time.Second),
        // Capture the rendered HTML
        chromedp.OuterHTML(`html`, &renderedHTML, chromedp.ByQuery),
    )
    if err != nil {
        return nil, fmt.Errorf("headless browser error: %w", err)
    }

    // Search rendered HTML for stream URLs (same logic as scrapeGeneric)
    if streamURL := findStreamURLInHTML(renderedHTML); streamURL != "" {
        metadata := &models.VideoMetadata{
            VideoURL: streamURL,
            Source:   extractDomain(pageURL),
        }
        // Try to extract title from rendered HTML
        doc, err := goquery.NewDocumentFromReader(strings.NewReader(renderedHTML))
        if err == nil {
            ogTitle := doc.Find("meta[property='og:title']").AttrOr("content", "")
            if ogTitle != "" {
                metadata.Title = ogTitle
            } else {
                metadata.Title = doc.Find("title").First().Text()
            }
            ogImage := doc.Find("meta[property='og:image']").AttrOr("content", "")
            if ogImage != "" {
                metadata.ThumbnailURL = ogImage
            }
        }
        return metadata, nil
    }

    if streamURL := findStreamURLInScripts(renderedHTML); streamURL != "" {
        metadata := &models.VideoMetadata{
            VideoURL: streamURL,
            Source:   extractDomain(pageURL),
        }
        return metadata, nil
    }

    // Search iframes in rendered HTML
    for _, iframeSrc := range extractIframeSrcs(renderedHTML) {
        iframeSrc = normalizeURL(iframeSrc, pageURL)
        videoURL, err := resolveEmbedURL(iframeSrc, pageURL)
        if err == nil && videoURL != "" {
            metadata := &models.VideoMetadata{
                VideoURL: videoURL,
                Source:   extractDomain(pageURL),
            }
            return metadata, nil
        }
    }

    return nil, fmt.Errorf("headless browser could not find stream URL from %s", pageURL)
}
```

### Integrasi ke ScrapeStreamURL

Ubah fungsi `ScrapeStreamURL()` agar headless browser dipanggil sebagai fallback terakhir di `scrapeGeneric()`:

```go
func scrapeGeneric(pageURL string) (*models.VideoMetadata, error) {
    // ... [existing Layer 1-3 code] ...

    // Layer 4 (LAST RESORT): try headless browser for SPA sites.
    log.Printf("All static scraping methods failed for %s, trying headless browser...", pageURL)
    return scrapeWithHeadless(pageURL)
}
```

Atau, buat terpisah agar tidak selalu pakai headless (lebih lambat):

```go
func ScrapeStreamURL(pageURL string) (*models.VideoMetadata, error) {
    switch detectSitePlatform(pageURL) {
    case "otakudesu":
        return scrapeOtakudesu(pageURL)
    case "anoboy":
        return scrapeAnoboy(pageURL)
    default:
        result, err := scrapeGeneric(pageURL)
        if err != nil {
            // Fallback: try headless browser for SPA sites
            log.Printf("Static scrape failed for %s, trying headless...", pageURL)
            return scrapeWithHeadless(pageURL)
        }
        return result, nil
    }
}
```

### Verifikasi
```bash
cd backend
go build ./...
go run main.go
# Test: paste https://z2.idlixku.com/movie/backrooms-2026 di frontend
```

### Known Limitations
- Butuh Chrome/Chromium terinstall di server
- Lebih lambat (~5-30 detik vs ~1-3 detik untuk static scraping)
- Menggunakan lebih banyak memory/CPU
- Untuk production, pertimbangkan pool chromedp instances atau external service

---

## Task 2 — Stream Proxy Endpoint [KRITIS]

### Masalah
Banyak CDN anime (Vidhide, Filedon, Mp4upload, dll) memblokir request dari browser yang tidak punya referer yang benar, atau tidak support CORS. Akibatnya, hls.js di frontend gagal fetch segment `.m3u8`/`.ts` dengan error 403 atau CORS.

Contoh error di browser console:
```
Access to XMLHttpRequest at 'https://cdn.example.com/segment.ts' 
from origin 'http://localhost:5173' has been blocked by CORS policy
```

### Solusi
Buat endpoint proxy di backend yang meneruskan request stream ke CDN dengan header yang benar (Referer, Origin, User-Agent).

### File yang Diubah/Dibuat
- `backend/main.go` — tambah route `/api/proxy`
- `backend/handlers/proxy.go` — **BARU**, handler proxy

### Implementasi

**File: `backend/handlers/proxy.go`** (BARU)

```go
package handlers

import (
    "io"
    "log"
    "net/http"
    "net/url"
    "strings"
)

// HandleStreamProxy proxies video stream requests through the backend,
// injecting correct Referer and User-Agent headers that CDNs expect.
// This solves CORS and 403 errors that occur when hls.js in the browser
// tries to fetch .m3u8/.ts segments directly.
//
// Usage: GET /api/proxy?url=<encoded-stream-url>&referer=<encoded-referer>
func HandleStreamProxy(w http.ResponseWriter, r *http.Request) {
    targetURL := r.URL.Query().Get("url")
    if targetURL == "" {
        http.Error(w, "Missing 'url' query parameter", http.StatusBadRequest)
        return
    }

    // Validate URL
    parsed, err := url.Parse(targetURL)
    if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
        http.Error(w, "Invalid URL", http.StatusBadRequest)
        return
    }

    // Build request to upstream CDN
    req, err := http.NewRequest("GET", targetURL, nil)
    if err != nil {
        http.Error(w, "Failed to create request", http.StatusInternalServerError)
        return
    }

    // Set headers that CDNs typically check
    req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

    // Use referer from query param, or derive from the stream URL's origin
    referer := r.URL.Query().Get("referer")
    if referer == "" {
        referer = parsed.Scheme + "://" + parsed.Host + "/"
    }
    req.Header.Set("Referer", referer)
    req.Header.Set("Origin", parsed.Scheme+"://"+parsed.Host)

    // Forward range requests (needed for seeking)
    if rangeHeader := r.Header.Get("Range"); rangeHeader != "" {
        req.Header.Set("Range", rangeHeader)
    }

    client := &http.Client{
        CheckRedirect: func(req *http.Request, via []*http.Request) error {
            if len(via) >= 10 {
                return http.ErrUseLastResponse
            }
            return nil
        },
    }

    resp, err := client.Do(req)
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

    // Add CORS headers so browser can use the proxied stream
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
    w.Header().Set("Access-Control-Allow-Headers", "Range")
    w.Header().Set("Access-Control-Expose-Headers", "Content-Range, Content-Length, Content-Type")

    if r.Method == http.MethodOptions {
        w.WriteHeader(http.StatusOK)
        return
    }

    w.WriteHeader(resp.StatusCode)
    io.Copy(w, resp.Body)
}

// IsStreamURL checks if a URL points to a video stream that may need proxying.
func IsStreamURL(rawURL string) bool {
    lower := strings.ToLower(rawURL)
    return strings.Contains(lower, ".m3u8") ||
        strings.Contains(lower, ".ts") ||
        strings.Contains(lower, ".mp4") ||
        strings.Contains(lower, ".webm")
}
```

**File: `backend/main.go`** — tambah route:

```go
http.HandleFunc("/api/proxy", corsMiddleware(handlers.HandleStreamProxy))
```

### Verifikasi
```bash
cd backend && go build ./... && go run main.go

# Test proxy directly:
curl "http://localhost:8080/api/proxy?url=https%3A%2F%2Fexample.com%2Fvideo.m3u8" -I

# Test dari frontend: paste URL anime yang sebelumnya 403, 
# seharusnya sekarang stream bisa diputar
```

---

## Task 3 — Site-Specific Adapters [SEDANG]

### Masalah
`detectSitePlatform()` sudah mendeteksi domain `samehadaku`, `animasu`, `kuronime`, `nanime` — tapi semuanya jatuh ke `scrapeGeneric()`. Beberapa situs ini mungkin punya struktur HTML khusus yang bisa di-scrape lebih efisien dengan adapter sendiri.

### File yang Diubah
- `backend/services/scraper.go` — tambah fungsi adapter per situs, update `ScrapeStreamURL()` routing

### Implementasi

**Approach: Prioritaskan situs berdasarkan traffic/popularity.** Cek satu per satu, bikin adapter kalau `scrapeGeneric()` gagal.

#### 3a. Samehadaku
Samehadaku biasanya WordPress-based, mirip Otakudesu. Cek dulu apakah `scrapeGeneric()` sudah bisa handle (karena ada base64 mirror selector parsing). Kalau gagal, tambah adapter:

```go
// Tambah di detectSitePlatform() switch:
case "samehadaku":
    return "samehadaku"

// Tambah fungsi:
func scrapeSamehadaku(pageURL string) (*models.VideoMetadata, error) {
    // Coba generic dulu (termasuk Blogger/mirror parsing)
    result, err := scrapeGeneric(pageURL)
    if err == nil {
        result.Source = "samehadaku"
        return result, nil
    }
    // Kalau gagal, coba headless browser
    return scrapeWithHeadless(pageURL)
}
```

#### 3b. Anoboy Variants
Anoboy kadang pindah domain (anoboy.me, anoboy.xyz, dll). Pastikan `detectSitePlatform()` match semua variant:

```go
case strings.Contains(domain, "anoboy"):
    return "anoboy"
```

Sudah ada di kode sekarang ✅.

#### 3c. Generic WordPress Detector
Tambahan untuk mendeteksi WordPress-based anime sites secara umum (punya `wp-admin`, `wp-content`, `.mirrorstream a`, dll):

```go
func isWordPressSite(html string) bool {
    return strings.Contains(html, "wp-content") ||
        strings.Contains(html, "wp-admin") ||
        strings.Contains(html, "wp-json")
}
```

### Verifikasi
Test dengan URL dari masing-masing situs:
```bash
# Samehadaku
# Paste URL episode samehadaku di frontend

# Anoboy variants  
# Paste URL anoboy.si, anoboy.me (kalau ada)
```

---

## Task 4 — Scrape Result Cache [RENDAH]

### Masalah
Setiap request `SET_VIDEO` atau `ADD_TO_QUEUE` dengan URL yang sama akan scrape ulang dari nol. Ini boros bandwidth, lambat, dan berisiko kena rate-limit/anti-bot dari situs target.

### Solusi
Tambah in-memory cache dengan TTL (Time-To-Live). Key = URL asli, Value = `VideoMetadata`.

### File yang Diubah
- `backend/services/scraper.go` — tambah cache layer

### Implementasi

```go
var (
    scrapeCache   = make(map[string]scrapeCacheEntry)
    scrapeCacheMu sync.RWMutex
)

type scrapeCacheEntry struct {
    metadata  *models.VideoMetadata
    expiresAt time.Time
}

const scrapeCacheTTL = 30 * time.Minute

// ScrapeStreamURLCached wraps ScrapeStreamURL with an in-memory cache.
func ScrapeStreamURLCached(pageURL string) (*models.VideoMetadata, error) {
    // Check cache
    scrapeCacheMu.RLock()
    if entry, ok := scrapeCache[pageURL]; ok && time.Now().Before(entry.expiresAt) {
        scrapeCacheMu.RUnlock()
        log.Printf("Cache hit for %s", pageURL)
        return entry.metadata, nil
    }
    scrapeCacheMu.RUnlock()

    // Cache miss — scrape
    metadata, err := ScrapeStreamURL(pageURL)
    if err != nil {
        return nil, err
    }

    // Store in cache
    scrapeCacheMu.Lock()
    scrapeCache[pageURL] = scrapeCacheEntry{
        metadata:  metadata,
        expiresAt: time.Now().Add(scrapeCacheTTL),
    }
    scrapeCacheMu.Unlock()

    return metadata, nil
}
```

Lalu ubah semua pemanggilan `ScrapeStreamURL()` di `message_handler.go` dan `queue_manager.go` menjadi `ScrapeStreamURLCached()`.

### Verifikasi
```bash
# Set video dengan URL yang sama dua kali
# Kedua kalinya harus instant (cache hit), lihat log:
# "Cache hit for https://anoboy.si/..."
```

---

## Task 5 — Retry & User-Agent Rotation [RENDAH]

### Masalah
Beberapa situs memblokir request dengan User-Agent yang sama terus-menerus, atau mengembalikan 503/429 (rate limit) yang bisa di-retry.

### File yang Diubah
- `backend/services/scraper.go` — update `fetchPageHTML()`, `fetchPageDocument()`

### Implementasi

```go
var userAgents = []string{
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
    "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Safari/605.1.15",
}

var uaIndex int
var uaMu sync.Mutex

func getRandomUserAgent() string {
    uaMu.Lock()
    defer uaMu.Unlock()
    ua := userAgents[uaIndex%len(userAgents)]
    uaIndex++
    return ua
}

// fetchWithRetry retries the request up to maxRetries times with exponential
// backoff and rotating User-Agent headers.
func fetchWithRetry(url, referer string, maxRetries int) (*http.Response, error) {
    var lastErr error
    for i := 0; i < maxRetries; i++ {
        if i > 0 {
            time.Sleep(time.Duration(i*2) * time.Second) // exponential backoff
        }

        req, err := http.NewRequest("GET", url, nil)
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

        // Retry on rate-limit or server errors
        if resp.StatusCode == 429 || resp.StatusCode == 503 {
            resp.Body.Close()
            lastErr = fmt.Errorf("HTTP %d from %s (attempt %d/%d)", resp.StatusCode, url, i+1, maxRetries)
            continue
        }

        return resp, nil
    }
    return nil, lastErr
}
```

### Verifikasi
```bash
# Scrape situs yang sebelumnya kadang gagal (intermittent)
# Perhatikan log: harusnya ada retry attempts
```

---

## Ringkasan Prioritas

| # | Task | Prioritas | Effort | Dependency |
|---|------|:---------:|:------:|:----------:|
| 1 | Headless browser (chromedp) | 🔴 Kritis | Besar | — |
| 2 | Stream proxy endpoint | 🔴 Kritis | Sedang | — |
| 3 | Site-specific adapters | 🟡 Sedang | Sedang | Task 1 |
| 4 | Scrape result cache | 🟢 Rendah | Kecil | — |
| 5 | Retry & UA rotation | 🟢 Rendah | Kecil | — |

**Rekomendasi urutan pengerjaan:**
1. Task 2 (proxy) — langsung solve masalah 403/CORS di banyak CDN
2. Task 1 (chromedp) — solve SPA sites, tapi butuh Chrome di server
3. Task 4 (cache) — quick win, langsung kurangi beban scraping
4. Task 3 (adapters) — test satu per satu, bikin adapter kalau generic gagal
5. Task 5 (retry/UA) — polish, handle edge cases

---

*Dokumentasi ini dibuat berdasarkan analisis aktual kode per 14 Juli 2026.*
