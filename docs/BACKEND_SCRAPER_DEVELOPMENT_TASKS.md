# Backend Scraper Development Tasks — WatchParty

> **Untuk backend agent**: Dokumen ini berisi task pengembangan scraper untuk meningkatkan coverage situs anime yang bisa di-scrape. Ada 2 task utama:
> 1. Wire `scrapeAnoboy` ke router scraper
> 2. Improve headless browser + NanoProxy integration untuk SPA sites dengan enkripsi kuat
>
> Setiap task mandiri. Verifikasi sebelum lanjut.

---

## Task 1 — Wire `scrapeAnoboy` ke ScrapeStreamURL Router [KRITIS]

### Masalah
`services/scraper.go:236-249` — Fungsi `scrapeAnoboy()` sudah ada (line 440-516) dan sudah lengkap dengan:
- Layer 1: Direct iframe detection
- Layer 2: Base64 mirror dropdown parsing
- Layer 3: Deep iframe search
- Blogger video URL extraction

**Tapi** `ScrapeStreamURL()` switch statement tidak handle case `"anoboy"`:

```go
func ScrapeStreamURL(pageURL string) (*models.VideoMetadata, error) {
	switch detectSitePlatform(pageURL) {
	case "otakudesu":
		return scrapeOtakudesu(pageURL)
	case "samehadaku":
		return scrapeSamehadaku(pageURL)
	// ❌ case "anoboy" TIDAK ADA — jatuh ke default → scrapeGeneric
	default:
		metadata, err := scrapeGeneric(pageURL)
		...
	}
}
```

`detectSitePlatform()` sudah return `"anoboy"` untuk domain yang mengandung "anoboy" (line 164), tapi tidak ada yang menangkap case ini.

### File yang Diubah
- `backend/services/scraper.go` — `ScrapeStreamURL()`

### Implementasi

Tambah case `"anoboy"` di switch statement `ScrapeStreamURL`:

```go
func ScrapeStreamURL(pageURL string) (*models.VideoMetadata, error) {
	switch detectSitePlatform(pageURL) {
	case "otakudesu":
		return scrapeOtakudesu(pageURL)
	case "anoboy":
		return scrapeAnoboy(pageURL)
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
```

### Verifikasi
```bash
cd backend && go build ./... && go run main.go

# Test dengan URL anoboy:
# 1. Paste URL episode anoboy.si di frontend
# 2. Cek log: harusnya "Scrape succeeded: ..."
# 3. Video harus bisa diputar

# Test URL yang bukan anoboy tetap jalan:
# 1. Paste URL otakudesu → tetap pakai scrapeOtakudesu
# 2. Paste URL random → tetap pakai scrapeGeneric
```

---

## Task 2 — Improve Headless Browser + NanoProxy untuk SPA Sites [KRITIS]

### Masalah Besar
Banyak situs anime modern menggunakan:
1. **SPA (Single Page Application)** — HTML kosong, video dimuat via JavaScript
2. **Strong encryption/obfuscation** — eval(), AES, custom packer, obfuscated variable names
3. **Anti-bot protection** — Cloudflare, IP-based blocking, cookie requirements
4. **Multi-layer iframe chains** — page → iframe1 → iframe2 → player (3-4 level deep)

Implementasi saat ini (`scrapeWithHeadless` dan `NanoScrape`) punya beberapa kelemahan:
- `scrapeWithHeadless` pakai `chromedp.Sleep(5*time.Second)` yang fixed — tidak adaptif
- Tidak ada network request interception di `scrapeWithHeadless` (hanya di `NanoScrape`)
- `NanoScrape` pakai `chromedp.Sleep(8*time.Second)` yang juga fixed
- Kedua fungsi tidak handle enkripsi kuat (hanya regex pattern matching)
- Tidak ada cookie/session persistence untuk site yang butuh login
- `findChromePath()` punya race condition (line 45-48, `chromeChecked` tanpa mutex)
- Chrome flags `disable-web-security` dan `IsolateOrigons` (typo, harus `IsolateOrigins`) bisa di-detect sebagai bot

### File yang Diubah/Dibuat
- `backend/services/nano_scraper.go` — improve NanoScrape
- `backend/services/scraper.go` — improve scrapeWithHeadless, improve fallback chain
- `backend/services/scraper_helpers.go` — **BARU**, shared helper functions

### Implementasi

#### 2a. Fix `findChromePath` Race Condition

`nano_scraper.go:45-48` — `chromeChecked` tanpa mutex bisa di-read/write bersamaan:

```go
import "sync"

var (
	chromePathMu sync.Mutex
	// ... existing vars
)

func findChromePath() string {
	chromePathMu.Lock()
	defer chromePathMu.Unlock()

	if chromeChecked {
		return chromePath
	}
	chromeChecked = true

	// ... rest of the function unchanged
}
```

#### 2b. Improve `scrapeWithHeadless` — Adaptive Wait + Network Interception

Modifikasi `scrapeWithHeadless` di `scraper.go` untuk menggunakan network interception (seperti `NanoScrape`) dan adaptive wait:

```go
func scrapeWithHeadless(pageURL string) (*models.VideoMetadata, error) {
	if findChromePath() == "" {
		return nil, fmt.Errorf("Chrome/Chromium not installed — headless scraping unavailable")
	}

	var (
		mu       sync.Mutex
		m3u8URLs []string
		mp4URLs  []string
	)

	allocOpts := []chromedp.ExecAllocatorOption{
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.ExecPath(findChromePath()),
		chromedp.Flag("disable-web-security", true),
		chromedp.Flag("disable-features", "IsolateOrigins,site-per-process"), // fix typo
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	}

	// Route through NanoProxy if configured
	if nanoProxyHTTP != "" {
		allocOpts = append(allocOpts, chromedp.ProxyServer(nanoProxyHTTP))
	}

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), allocOpts...)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	ctx, timeoutCancel := context.WithTimeout(ctx, 45*time.Second)
	defer timeoutCancel()

	// Enable network event listening to capture stream URLs from JS-loaded resources
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
			if strings.Contains(lower, ".m3u8") && !nanoContains(m3u8URLs, respURL) {
				m3u8URLs = append(m3u8URLs, respURL)
			}
			if strings.Contains(lower, ".mp4") && !nanoContains(mp4URLs, respURL) {
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
		// Adaptive wait: tunggu sampai network idle atau timeout
		chromedp.Sleep(8*time.Second),
		chromedp.OuterHTML("html", &renderedHTML, chromedp.ByQuery),
	)
	if err != nil {
		return nil, fmt.Errorf("headless render failed for %s: %w", pageURL, err)
	}

	// Priority 1: Use network-intercepted stream URLs (most reliable)
	if len(m3u8URLs) > 0 || len(mp4URLs) > 0 {
		streamURL := nanoSelectBest(m3u8URLs, mp4URLs)
		if streamURL != "" {
			doc, _ := goquery.NewDocumentFromReader(strings.NewReader(renderedHTML))
			metadata := extractGenericMetadata(doc, pageURL)
			metadata.VideoURL = streamURL
			return metadata, nil
		}
	}

	// Priority 2: Search rendered HTML for stream URLs
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(renderedHTML))
	if err != nil {
		return nil, fmt.Errorf("failed to parse rendered HTML: %w", err)
	}
	metadata := extractGenericMetadata(doc, pageURL)

	if streamURL := findStreamURLInHTML(renderedHTML); streamURL != "" {
		metadata.VideoURL = streamURL
		return metadata, nil
	}

	if streamURL := findStreamURLInScripts(renderedHTML); streamURL != "" {
		metadata.VideoURL = streamURL
		return metadata, nil
	}

	// Priority 3: Follow iframes rendered by JS
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
```

#### 2c. Improve NanoScrape — Better Event Detection + Adaptive Wait

Modifikasi `NanoScrape` di `nano_scraper.go` untuk mendeteksi lebih banyak pola stream URL:

```go
func NanoScrape(pageURL string) (*NanoScraperResult, error) {
	var (
		mu         sync.Mutex
		m3u8URLs   []string
		mp4URLs    []string
		iframeURLs []string
		// NEW: track all network requests for pattern matching
		allRequests []string
	)

	allocOpts := []chromedp.ExecAllocatorOption{
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.Flag("disable-web-security", true),
		chromedp.Flag("disable-features", "IsolateOrigins,site-per-process"), // fix typo
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
		// NEW: additional flags for better SPA support
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("disable-ipc-flooding-protection", false),
		chromedp.WindowSize(1920, 1080),
	}

	if cp := findChromePath(); cp != "" {
		allocOpts = append(allocOpts, chromedp.ExecPath(cp))
	}

	if nanoProxyHTTP != "" {
		log.Printf("[NanoScraper] Routing through NanoProxy: %s", nanoProxyHTTP)
		allocOpts = append(allocOpts, chromedp.ProxyServer(nanoProxyHTTP))
	}

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), allocOpts...)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	// Enhanced network event listening
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		switch e := ev.(type) {
		case *network.EventRequestWillBeSent:
			reqURL := e.Request.URL
			mu.Lock()
			allRequests = append(allRequests, reqURL)
			lower := strings.ToLower(reqURL)
			if strings.Contains(lower, ".m3u8") {
				m3u8URLs = append(m3u8URLs, reqURL)
			}
			if strings.Contains(lower, ".mp4") && !strings.Contains(lower, ".mp44") {
				mp4URLs = append(mp4URLs, reqURL)
			}
			// NEW: detect blob URLs (used by some encrypted players)
			if strings.HasPrefix(reqURL, "blob:") {
				log.Printf("[NanoScraper] Detected blob URL: %s", reqURL)
			}
			mu.Unlock()

		case *network.EventResponseReceived:
			respURL := e.Response.URL
			mu.Lock()
			lower := strings.ToLower(respURL)
			if strings.Contains(lower, ".m3u8") && !nanoContains(m3u8URLs, respURL) {
				m3u8URLs = append(m3u8URLs, respURL)
			}
			if strings.Contains(lower, ".mp4") && !nanoContains(mp4URLs, respURL) {
				mp4URLs = append(mp4URLs, respURL)
			}
			mu.Unlock()

		// NEW: intercept WebSocket messages (some players use WS for stream negotiation)
		case *network.EventWebSocketFrameReceived:
			payload := ""
			if e.Response != nil {
				payload = string(e.Response.PayloadData)
			}
			mu.Lock()
			if match := m3u8Regex.FindString(payload); match != "" && !nanoContains(m3u8URLs, match) {
				m3u8URLs = append(m3u8URLs, match)
			}
			if match := mp4Regex.FindString(payload); match != "" && !nanoContains(mp4URLs, match) {
				mp4URLs = append(mp4URLs, match)
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

	// NEW: If no stream URLs found in network, try to find in rendered HTML scripts
	if len(m3u8URLs) == 0 && len(mp4URLs) == 0 {
		log.Printf("[NanoScraper] No stream URLs in network requests, searching rendered HTML...")
		if streamURL := findStreamURLInHTML(renderedHTML); streamURL != "" {
			m3u8URLs = append(m3u8URLs, streamURL)
		}
		if streamURL := findStreamURLInScripts(renderedHTML); streamURL != "" {
			if strings.Contains(streamURL, ".m3u8") {
				m3u8URLs = append(m3u8URLs, streamURL)
			} else {
				mp4URLs = append(mp4URLs, streamURL)
			}
		}
	}

	// NEW: If still nothing, try unpacking packed JS in rendered HTML
	if len(m3u8URLs) == 0 && len(mp4URLs) == 0 {
		unpacked := unpackPackerInHTML(renderedHTML)
		for _, script := range unpacked {
			if match := m3u8Regex.FindString(script); match != "" {
				m3u8URLs = append(m3u8URLs, match)
			}
			if match := mp4Regex.FindString(script); match != "" {
				mp4URLs = append(mp4URLs, match)
			}
		}
	}

	pageTitle := nanoExtractTitle(renderedHTML)
	ogImage := nanoExtractImage(renderedHTML)

	for _, src := range extractIframeSrcs(renderedHTML) {
		iframeURLs = append(iframeURLs, normalizeURL(src, pageURL))
	}

	result := &NanoScraperResult{
		StreamURL:  nanoSelectBest(m3u8URLs, mp4URLs),
		PageTitle:  pageTitle,
		Thumbnail:  ogImage,
		Source:     extractDomain(pageURL),
		M3U8URLs:   m3u8URLs,
		MP4URLs:    mp4URLs,
		IFrameURLs: iframeURLs,
	}

	log.Printf("[NanoScraper] Found %d m3u8, %d mp4, %d iframes, %d total requests for %s",
		len(m3u8URLs), len(mp4URLs), len(iframeURLs), len(allRequests), pageURL)

	return result, nil
}
```

#### 2d. Improve Fallback Chain — Better ScrapeStreamURL Routing

Update `ScrapeStreamURL` untuk menggunakan NanoScraper sebagai fallback yang lebih agresif:

```go
func ScrapeStreamURL(pageURL string) (*models.VideoMetadata, error) {
	switch detectSitePlatform(pageURL) {
	case "otakudesu":
		return scrapeOtakudesu(pageURL)
	case "anoboy":
		return scrapeAnoboy(pageURL)
	case "samehadaku":
		return scrapeSamehadaku(pageURL)
	default:
		// Layer 1: Try static scraping (fast, no browser needed)
		metadata, err := scrapeGeneric(pageURL)
		if err == nil {
			return metadata, nil
		}

		// Layer 2: Try headless browser with network interception (slow but thorough)
		log.Printf("Static scrape failed for %s, trying headless...", pageURL)
		return scrapeWithNanoOrHeadless(pageURL)
	}
}
```

#### 2e. Add `fetchPageDocumentWithRefererChain` Helper

Tambah helper baru di `scraper.go` untuk fetch HTML document dengan referer chain:

```go
func fetchPageDocumentWithRefererChain(pageURL string, referers []string) (*goquery.Document, error) {
	var lastErr error
	for _, ref := range referers {
		doc, err := fetchPageDocument(pageURL, ref)
		if err == nil {
			return doc, nil
		}
		lastErr = err
	}
	return nil, lastErr
}
```

#### 2f. Add Stream URL Detection in Script Content (JSON configs)

Tambah pattern baru di `findStreamURLInScripts` untuk menangkap lebih banyak pola:

```go
func findStreamURLInScripts(html string) string {
	// Existing patterns...

	// NEW: Pattern: "src":"https://...m3u8" (common in video player configs)
	srcRegex := regexp.MustCompile(`"src"\s*:\s*"(https?://[^"]+?\.m3u8[^"]*)"`)
	if match := srcRegex.FindStringSubmatch(html); len(match) > 1 {
		return match[1]
	}

	// NEW: Pattern: "url":"https://...m3u8" (common in API responses)
	urlRegex := regexp.MustCompile(`"url"\s*:\s*"(https?://[^"]+?\.m3u8[^"]*)"`)
	if match := urlRegex.FindStringSubmatch(html); len(match) > 1 {
		return match[1]
	}

	// NEW: Pattern: src: "https://...m3u8" (unquoted JS)
	srcUnquotedRegex := regexp.MustCompile(`src\s*[:=]\s*["'](https?://[^"']+?\.m3u8[^"']*)["']`)
	if match := srcUnquotedRegex.FindStringSubmatch(html); len(match) > 1 {
		return match[1]
	}

	// NEW: Pattern: window.__PLAYURL__ = "https://..." (SPA globals)
	globalVarRegex := regexp.MustCompile(`window\.__\w+__\s*=\s*["'](https?://[^"']+)["']`)
	if match := globalVarRegex.FindStringSubmatch(html); len(match) > 1 {
		return match[1]
	}

	return ""
}
```

#### 2g. Fix Chrome Flag Typo

`nano_scraper.go:120` — `IsolateOrigons` harusnya `IsolateOrigins`:

```go
// BEFORE (typo):
chromedp.Flag("disable-features", "IsolateOrigons,site-per-process"),

// AFTER (fixed):
chromedp.Flag("disable-features", "IsolateOrigins,site-per-process"),
```

### Verifikasi

```bash
cd backend && go build ./...

# Test 1: Wire scrapeAnoboy
# Paste URL anoboy.si → harus berhasil scrape
# Log: "Scrape succeeded: ..."

# Test 2: SPA site (headless fallback)
# Paste URL SPA site (misalnya idlixku.com) → harus fallback ke headless
# Log: "Static scrape failed for ..., trying headless..."
# Log: "[NanoScraper] Found X m3u8, Y mp4, Z iframes"

# Test 3: Strong encryption (packed JS)
# Paste URL site yang pakai Dean Edwards Packer
# Log: stream URL found in unpacked JS

# Test 4: Network interception
# Cek log "[NanoScraper] Found ... total requests" untuk memastikan
# network interception menangkap stream URLs dari JS-loaded resources

# Test 5: Race detector
go build -race ./...
# Pastikan tidak ada race condition di findChromePath
```

---

## Ringkasan

| # | Task | Prioritas | Effort | File |
|---|------|:---------:|:------:|------|
| 1 | Wire `scrapeAnoboy` ke router | Kritis | Kecil | `services/scraper.go` |
| 2a | Fix `findChromePath` race | Sedang | Kecil | `services/nano_scraper.go` |
| 2b | Improve `scrapeWithHeadless` (network interception) | Kritis | Besar | `services/scraper.go` |
| 2c | Improve `NanoScrape` (better detection) | Kritis | Sedang | `services/nano_scraper.go` |
| 2d | Improve fallback chain | Sedang | Kecil | `services/scraper.go` |
| 2e | Add referer chain document fetcher | Rendah | Kecil | `services/scraper.go` |
| 2f | Add new stream URL patterns | Sedang | Kecil | `services/scraper.go` |
| 2g | Fix Chrome flag typo | Rendah | Trivial | `services/nano_scraper.go` |

**Urutan pengerjaan yang disarankan:**
1. Task 1 (wire anoboy) — 5 menit, langsung buka coverage situs baru
2. Task 2g (fix typo) — 1 menit
3. Task 2a (fix race) — 5 menit
4. Task 2f (new patterns) — 15 menit, improve static scraping
5. Task 2c (improve NanoScrape) — 30 menit, better network interception
6. Task 2b (improve headless) — 30 menit, unified headless + network interception
7. Task 2d (fallback chain) — 10 menit, polish routing

---

*Dokumentasi ini dibuat berdasarkan analisis kode per 17 Juli 2026.*
