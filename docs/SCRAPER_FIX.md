# Scraper Fix — Universal Stream Extraction

**Tanggal:** 14 Juli 2026  
**File:** `backend/services/scraper.go`  
**Status:** Implemented (Fix A + B)

---

## Masalah

Scraper hanya bisa handle URL dari **otakudesu**. Situs lain seperti **anoboy.si** dan **idlixku.com** gagal dengan error:

```
Scrape failed for https://anoboy.si/...: could not find stream URL from https://anoboy.si/...
Scrape failed for https://z2.idlixku.com/...: could not find stream URL from https://z2.idlixku.com/...
```

---

## Root Cause Analysis

### Anoboy.si — Blogger Video Embed

Anoboy menggunakan **iframe Blogger** sebagai video player:

```html
<iframe src="https://www.blogger.com/video.g?token=AD6v5dyNWDvROkUN5..." ...></iframe>
```

Selain itu, Anoboy punya **mirror dropdown** dengan base64-encoded iframes:

```html
<select class="mirror">
  <option value="PGlmcmFtZSBzcmM9Imh0dHBzOi8vd3d3LmJsb2dnZXIu..."> <!-- base64 → Blogger iframe -->
</select>
```

**Kenapa gagal:** Scraper menemukan iframe, fetch isinya, tapi Blogger video pages menggunakan **JavaScript** untuk memuat stream URL — bukan di HTML mentah. Regex `.m3u8`/`.mp4` tidak menemukan apa-apa.

### Idlixku.com — Next.js SPA

Idlixku adalah **Next.js SPA**. HTML yang dikembalikan hanya **skeleton/loader**:

```html
<div class="skeleton h-6 w-20 !rounded mb-2"></div>
<!-- Tidak ada <iframe>, tidak ada video URL -->
```

**Kenapa gagal:** Semua konten dimuat via **JavaScript client-side**. Scraper Go hanya bisa baca HTML mentah.

---

## Solusi yang Diimplementasikan

### Fix A — Blogger Video Extractor

Fungsi baru `extractBloggerVideoURL()` yang:
1. Fetch halaman Blogger video
2. Cari video URL dari JSON data di `<script>` tags (pattern: `"play_url"`, `"stream_url"`, `"downloadUrl"`)
3. Handle escaped characters (`\u0026` → `&`)

```go
func extractBloggerVideoURL(bloggerPageURL, referer string) (string, error) {
    html, err := fetchPageHTML(bloggerPageURL, referer)
    // ... search for "play_url", "stream_url", "downloadUrl" patterns
    // ... unescape \u0026, \u003d
    return videoURL, nil
}
```

### Fix B — Anoboy Site Adapter

Fungsi baru `scrapeAnoboy()` yang:
1. Cari direct iframe Blogger di page (`.player-embed iframe`)
2. Parse mirror dropdown options (base64 → iframe → Blogger → video URL)
3. Extract metadata (title, episode, thumbnail) dari Anoboy HTML structure

```go
func scrapeAnoboy(pageURL string) (*models.VideoMetadata, error) {
    // Layer 1: Direct iframe
    // Layer 2: Base64 mirror options
    // Layer 3: Deep iframe search (catch-all)
}
```

### Fix C — Enhanced Generic Scraper

`scrapeGeneric()` sekarang juga menangani:
1. **Base64 mirror selectors** — pattern `(value|data-content)="BASE64..."` di HTML
2. **Blogger iframes** — via `resolveEmbedURL()` yang mendeteksi Blogger URLs
3. **Deep iframe traversal** — tetap 3 level deep, tapi sekarang dengan Blogger awareness

### Fix D — `resolveEmbedURL()` Helper

Fungsi baru yang menyelesaikan embed URL ke direct video stream:

```go
func resolveEmbedURL(embedURL, referer string) (string, error) {
    if isBloggerVideoURL(embedURL) {
        return extractBloggerVideoURL(embedURL, referer)
    }
    // Generic: fetch, search HTML/scripts, follow nested iframes
}
```

---

## Arsitektur Scraper Setelah Fix

```
User pastes URL
    ↓
ScrapeStreamURL()
    ↓
detectSitePlatform()
    ├── "otakudesu" → scrapeOtakudesu()     [existing, unchanged]
    ├── "anoboy"    → scrapeAnoboy()         [NEW]
    └── "generic"   → scrapeGeneric()        [ENHANCED]
                        ↓
                    Layer 1: Direct stream in HTML
                    Layer 2: Stream in <script> tags
                    Layer 2.5: Base64 mirror selectors [NEW]
                    Layer 3: Iframes → resolveEmbedURL() [ENHANCED]
                                ↓
                            Blogger? → extractBloggerVideoURL() [NEW]
                            Other?   → fetch + search HTML/scripts
                            Nested?  → recursive (3 levels)
```

---

## Sites yang Sekarang Bisa Di-scrape

| Situs | Status | Adapter |
|-------|--------|---------|
| otakudesu.blog | ✅ Working | `scrapeOtakudesu` |
| anoboy.si | ✅ Fixed | `scrapeAnoboy` (Blogger embeds) |
| samehadaku.* | ✅ Should work | `scrapeGeneric` (WordPress + Blogger) |
| Situs dengan Blogger embed | ✅ Should work | `scrapeGeneric` → `resolveEmbedURL` |
| Situs SPA (idlixku, dll) | ❌ Still fails | Needs headless browser |

---

## Sites yang Masih Gagal (dan Kenapa)

### Next.js / React SPA Sites (idlixku.com, dll)

**Masalah:** HTML yang dikembalikan hanya skeleton/loader. Video URL dimuat via JavaScript client-side.

**Solusi yang dibutuhkan:** Headless browser (chromedp/rod) untuk render JavaScript.

```go
// go get github.com/chromedp/chromedp
func scrapeWithHeadless(pageURL string) (string, error) {
    ctx, cancel := chromedp.NewContext(context.Background())
    defer cancel()
    
    var html string
    chromedp.Run(ctx,
        chromedp.Navigate(pageURL),
        chromedp.WaitVisible(`video, iframe`),
        chromedp.Sleep(5*time.Second),
        chromedp.OuterHTML(`html`, &html),
    )
    return findStreamURLInHTML(html), nil
}
```

**Caveat:** Butuh Chrome/Chromium terinstall di server. Lebih berat resource.

### Sites dengan Cloudflare / Anti-Scraping

**Masalah:** Return 403 atau CAPTCHA page.

**Solusi:** Proxy headers, cookie rotation, atau gunakan external scraping service.

---

## Testing

Test dengan URL berikut setelah deploy:

```bash
# Anoboy (should work now)
curl -X POST http://localhost:8080/api/rooms
# Then via WebSocket: SET_VIDEO with anoboy URL

# Direct stream (should work, no scraping needed)
# SET_VIDEO with: https://example.com/video.mp4

# YouTube (should work, no scraping needed)
# SET_VIDEO with: https://youtube.com/watch?v=...
```

---

## Fungsi Baru yang Ditambahkan

| Fungsi | Lokasi | Deskripsi |
|--------|--------|-----------|
| `isBloggerVideoURL()` | scraper.go:65 | Deteksi URL Blogger video |
| `extractBloggerVideoURL()` | scraper.go:74 | Extract video URL dari halaman Blogger |
| `scrapeAnoboy()` | scraper.go:273 | Adapter khusus anoboy.si |
| `resolveEmbedURL()` | scraper.go:353 | Resolve embed URL ke direct stream |
| `extractAnoboyMetadata()` | scraper.go:400 | Extract metadata dari halaman Anoboy |

## Fungsi yang Diubah

| Fungsi | Perubahan |
|--------|-----------|
| `detectSitePlatform()` | Tambah `"anoboy"` case |
| `ScrapeStreamURL()` | Tambah routing ke `scrapeAnoboy` |
| `scrapeGeneric()` | Tambah Layer 2.5 (base64 mirrors) + `resolveEmbedURL` |
| `deepSearchStream()` | Tambah Blogger detection di awal |

---

## Next Steps (Belum Diimplementasikan)

1. **Headless browser support** — untuk SPA sites seperti idlixku.com
2. **Stream proxy endpoint** — untuk mengatasi CORS issues dengan CDN
3. **Site adapter pattern** — refactor ke interface-based adapter system
4. **Cache** — cache hasil scrape untuk menghindari repeated requests
5. **More site adapters** — samehadaku, animasu, kuronime, dll

---

*Dokumentasi ini dibuat bersamaan dengan implementasi fix.*
