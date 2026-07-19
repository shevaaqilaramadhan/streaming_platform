# 🔍 Streaming Platform — Analisis Bug & Masalah

**Tanggal:** 14 Juli 2026  
**Scope:** Full-stack (Frontend Vue.js + Backend Go)  
**Masalah Utama:** Scraper hanya bisa handle URL otakudesu, belum universal

---

## 📋 RINGKASAN EKSEKUTIF

| Kategori | Total Issue | Kritis | Sedang | Rendah |
|----------|:-----------:|:------:|:------:|:------:|
| **Backend** | 10 | 4 | 4 | 2 |
| **Frontend** | 7 | 2 | 3 | 2 |
| **Total** | **17** | **6** | **7** | **4** |

---

## 🔴 BACKEND ISSUES

### BUG-01 [KRITIS] — Scraper Hanya Untuk Otakudesu (Masalah Utama)

**File:** `backend/services/scraper.go:38-43`

```go
func ScrapeStreamURL(pageURL string) (*models.VideoMetadata, error) {
    if strings.Contains(pageURL, "otakudesu") {
        return scrapeOtakudesu(pageURL)
    }
    return scrapeGeneric(pageURL)
}
```

**Masalah:**
- `scrapeGeneric()` sangat terbatas — hanya mencari `<iframe>` dan regex `.m3u8/.mp4` langsung di HTML
- Kebanyakan situs anime streaming menggunakan **obfuscated JavaScript** (Dean Edwards Packer, base64, AES), bukan embed langsung
- Tidak support **nested iframe chains** (page → iframe1 → iframe2 → player)
- Tidak ada handling untuk **referer/cookie requirements** dari CDN

**Solusi:** Implementasi multi-layer universal scraper:

```go
// Solusi: Architecture baru untuk universal scraper

// 1. Site Detector — identifikasi platform berdasarkan domain/pattern
func detectSitePlatform(pageURL string) string {
    domain := extractDomain(pageURL)
    switch {
    case strings.Contains(domain, "otakudesu"):
        return "otakudesu"
    case strings.Contains(domain, "samehadaku"):
        return "samehadaku"  // WordPress-based, mirip otakudesu
    case strings.Contains(domain, "animasu"):
        return "animasu"
    case strings.Contains(domain, "kuronime"):
        return "kuronime"
    case strings.Contains(domain, "nanime"):
        return "nanime"
    default:
        return "generic"
    }
}

// 2. Enhanced Generic Scraper — multi-depth iframe traversal + JS unpacking
func scrapeGeneric(pageURL string) (*models.VideoMetadata, error) {
    doc, err := fetchPageDocument(pageURL, "")
    if err != nil {
        return nil, err
    }

    html, _ := doc.Html()

    // Layer 1: Direct stream URL in page HTML (including packed JS)
    if streamURL := findStreamURLInHTML(html); streamURL != "" {
        metadata := extractGenericMetadata(doc, pageURL)
        metadata.VideoURL = streamURL
        return metadata, nil
    }

    // Layer 2: Collect ALL iframes from page
    var iframeSrcs []string
    doc.Find("iframe").Each(func(_ int, s *goquery.Selection) {
        if src, exists := s.Attr("src"); exists && src != "" {
            src = normalizeURL(src, pageURL)
            iframeSrcs = append(iframeSrcs, src)
        }
    })

    // Layer 3: For each iframe, fetch and search deeper (up to 3 levels)
    for _, iframeSrc := range iframeSrcs {
        streamURL, err := deepSearchStream(iframeSrc, pageURL, 3)
        if err == nil && streamURL != "" {
            metadata := extractGenericMetadata(doc, pageURL)
            metadata.VideoURL = streamURL
            return metadata, nil
        }
    }

    return nil, fmt.Errorf("could not find stream URL from %s", pageURL)
}

// 3. Deep recursive iframe/stream searcher
func deepSearchStream(url, referer string, maxDepth int) (string, error) {
    if maxDepth <= 0 {
        return "", fmt.Errorf("max iframe depth reached")
    }

    html, err := fetchPageHTML(url, referer)
    if err != nil {
        return "", err
    }

    // Check for stream URL in HTML (including packed JS)
    if streamURL := findStreamURLInHTML(html); streamURL != "" {
        return streamURL, nil
    }

    // Check for stream URL in script tags content
    if streamURL := findStreamURLInScripts(html); streamURL != "" {
        return streamURL, nil
    }

    // Follow nested iframes
    iframes := extractIframeSrcs(html)
    for _, iframeSrc := range iframes {
        iframeSrc = normalizeURL(iframeSrc, url)
        if result, err := deepSearchStream(iframeSrc, url, maxDepth-1); err == nil {
            return result, nil
        }
    }

    return "", fmt.Errorf("no stream found at depth %d", maxDepth)
}

// 4. Extract stream URLs from <script> tags (JSON configs, JS variables)
func findStreamURLInScripts(html string) string {
    // Pattern: "file":"https://...m3u8"
    fileRegex := regexp.MustCompile(`"file"\s*:\s*"(https?://[^"]+?\.m3u8[^"]*)"`)
    if match := fileRegex.FindStringSubmatch(html); len(match) > 1 {
        return match[1]
    }

    // Pattern: "sources":[{"file":"..."}]
    sourcesRegex := regexp.MustCompile(`"sources"\s*:\s*\[.*?"file"\s*:\s*"(https?://[^"]+)"`)
    if match := sourcesRegex.FindStringSubmatch(html); len(match) > 1 {
        return match[1]
    }

    // Pattern: source: "https://...m3u8"
    sourceRegex := regexp.MustCompile(`source\s*[:=]\s*["'](https?://[^"']+?\.m3u8[^"']*)["']`)
    if match := sourceRegex.FindStringSubmatch(html); len(match) > 1 {
        return match[1]
    }

    // Pattern: var videoUrl = "https://..."
    varRegex := regexp.MustCompile(`var\s+\w*[Vv]ideo\w*\s*=\s*["'](https?://[^"']+)["']`)
    if match := varRegex.FindStringSubmatch(html); len(match) > 1 {
        return match[1]
    }

    return ""
}

// 5. Normalize relative URLs
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
```

---

### BUG-02 [KRITIS] — Hardcoded AJAX URL untuk Otakudesu

**File:** `backend/services/scraper.go:133`

```go
ajaxURL := "https://otakudesu.blog/wp-admin/admin-ajax.php"
```

**Masalah:** Jika otakudesu pindah domain (e.g., `.cloud`, `.me`, `.xyz`), scraper akan gagal total.

**Solusi:**
```go
func getOtakudesuAjaxURL(pageURL string) string {
    parsed, err := url.Parse(pageURL)
    if err != nil {
        return "https://otakudesu.blog/wp-admin/admin-ajax.php"
    }
    return parsed.Scheme + "://" + parsed.Host + "/wp-admin/admin-ajax.php"
}
```

---

### BUG-03 [KRITIS] — Regex M3U8 Bisa Match URL Invalid

**File:** `backend/services/scraper.go:30`

```go
m3u8Regex = regexp.MustCompile(`https?://[^\s"'<>]+?\.m3u8[^\s"'<>]*`)
```

**Masalah:** Regex ini bisa match URL yang mengandung karakter trailing seperti `)`, `}`, `]`, `;` yang bukan bagian dari URL.

**Contoh:** `(function(){var url="https://cdn.example.com/stream.m3u8";})` → match: `https://cdn.example.com/stream.m3u8";})`

**Solusi:**
```go
// Lebih presisi — stop di karakter yang umumnya bukan URL
m3u8Regex = regexp.MustCompile(`https?://[^\s"'<>\)\]\}]+\.m3u8(?:\?[^\s"'<>\)\]\}]*)?`)
mp4Regex  = regexp.MustCompile(`https?://[^\s"'<>\)\]\}]+\.mp4(?:\?[^\s"'<>\)\]\}]*)?`)
```

---

### BUG-04 [KRITIS] — Queue Scrape Direct URL Secara Tidak Perlu

**File:** `backend/services/queue_manager.go:27`

```go
func AddToQueue(roomID, url string) (*models.QueueItem, error) {
    // ...
    metadata, err := ScrapeStreamURL(url)
    if err != nil {
        return nil, fmt.Errorf("failed to scrape metadata: %w", err)
    }
```

**Masalah:** Jika user paste direct `.mp4` atau `.m3u8` URL ke queue, fungsi tetap mencoba scrape (yang akan gagal karena tidak ada HTML). Harusnya langsung treat sebagai direct stream.

**Solusi:**
```go
func AddToQueue(roomID, url string) (*models.QueueItem, error) {
    room, exists := GetRoom(roomID)
    if !exists {
        return nil, fmt.Errorf("room not found")
    }

    var queueItem *models.QueueItem

    // Check if it's a direct stream URL first
    if isDirectStreamURL(url) || isYouTubeURL(url) {
        queueItem = &models.QueueItem{
            ID:        generateQueueItemID(),
            URL:       url,
            Title:     extractDomainFromURL(url),
            Episode:   "",
            Thumbnail: "",
        }
    } else {
        // Scrape the page for stream URL
        metadata, err := ScrapeStreamURL(url)
        if err != nil {
            return nil, fmt.Errorf("failed to scrape metadata: %w", err)
        }
        queueItem = &models.QueueItem{
            ID:        generateQueueItemID(),
            URL:       metadata.VideoURL,
            Title:     metadata.Title,
            Episode:   metadata.Episode,
            Thumbnail: metadata.ThumbnailURL,
        }
    }

    room.Mutex.Lock()
    room.Queue = append(room.Queue, queueItem)
    room.Mutex.Unlock()

    return queueItem, nil
}
```

---

### BUG-05 [SEDANG] — Tidak Ada Retry/Fallback untuk CDN Stream

**File:** `backend/services/scraper.go:299-336`

**Masalah:** `scrapeGeneric()` hanya coba sekali per iframe. Banyak CDN mengembalikan 403 jika referer salah, tapi scraper tidak retry dengan referer berbeda.

**Solusi:**
```go
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
```

---

### BUG-06 [SEDANG] — Room Memory Leak (Tidak Ada Cleanup)

**File:** `backend/services/room_manager.go`

**Masalah:** Room hanya dihapus saat semua client disconnect. Jika WebSocket connection hang/timeout, room akan exist selamanya di memory.

**Solusi:** Tambahkan TTL-based cleanup:
```go
func StartRoomCleanup(interval time.Duration) {
    ticker := time.NewTicker(interval)
    go func() {
        for range ticker.C {
            mu.Lock()
            now := time.Now()
            for id, room := range rooms {
                room.Mutex.RLock()
                isEmpty := len(room.Clients) == 0
                age := now.Sub(room.CreatedAt)
                room.Mutex.RUnlock()

                if isEmpty && age > 30*time.Minute {
                    delete(rooms, id)
                    log.Printf("Cleaned up stale room: %s", id)
                }
            }
            mu.Unlock()
        }
    }()
}
```

---

### BUG-07 [SEDANG] — CORS Hanya Di /api/rooms, Tidak Di /ws/ dan /api/public-rooms

**File:** `backend/main.go:12-14`

```go
http.HandleFunc("/api/rooms", corsMiddleware(handleCreateRoom))
http.HandleFunc("/api/public-rooms", handlers.GetPublicRooms) // ← no CORS middleware!
http.HandleFunc("/ws/", handlers.HandleWebSocket)             // ← no CORS middleware!
```

**Masalah:** `/api/public-rooms` punya CORS header hardcoded (`*`), tapi `/ws/` tidak konsisten. Di production, ini bisa menyebabkan masalah.

**Solusi:** Bungkus semua handler dengan CORS middleware yang konsisten, atau gunakan middleware di level router.

---

### BUG-08 [SEDANG] — `updateRoomState` Race Condition Potensial

**File:** `backend/services/room_manager.go:126-139`

```go
func UpdateRoomState(roomID string, currentTime float64, isPlaying bool) {
    mu.RLock()          // ← global read lock
    defer mu.RUnlock()

    room, exists := rooms[roomID]
    // ...
    room.Mutex.Lock()   // ← room-level write lock
    room.CurrentTime = currentTime
    room.IsPlaying = isPlaying
    room.Mutex.Unlock()
}
```

**Masalah:** Mengambil `mu.RLock()` global lalu `room.Mutex.Lock()` di dalam. Ini aman selama urutan konsisten, tapi pattern ini berisiko deadlock jika ada fungsi lain yang mengambil lock berbeda urutan.

**Solusi:** Konsisten gunakan pattern: ambil room reference di bawah global lock, lepas global lock, baru lock room.

---

### BUG-09 [RENDAH] — `generateQueueItemID` Fallback ke `len(bytes)` yang Selalu 8

**File:** `backend/services/queue_manager.go:12-16`

```go
func generateQueueItemID() string {
    bytes := make([]byte, 8)
    if _, err := rand.Read(bytes); err != nil {
        return fmt.Sprintf("q_%d", len(bytes)) // ← always "q_8"
    }
    return "q_" + hex.EncodeToString(bytes)
}
```

**Solusi:** Gunakan timestamp + random sebagai fallback.

---

### BUG-10 [RENDAH] — WebSocket Upgrader CheckOrigin Selalu True

**File:** `backend/handlers/websocket.go:18-19`

```go
CheckOrigin: func(r *http.Request) bool {
    return true  // ← insecure for production
},
```

**Solusi:** Untuk production, validasi origin terhadap whitelist domain.

---

## 🟡 FRONTEND ISSUES

### BUG-11 [KRITIS] — `isOwn` Message Detection Salah

**File:** `src/composables/useRoom.js:105`

```js
isOwn: data.payload.userId === nickname, // ← WRONG!
```

**Masalah:** Membandingkan `userId` (backend-generated short ID seperti "aB3xYz") dengan `nickname` (user-chosen name seperti "Alice"). Ini akan selalu `false`.

**Solusi:**
```js
// Simpan userId lokal saat join
const localUserId = ref('')

// Di ROOM_INIT handler:
case MSG_TYPES.ROOM_INIT:
    // Simpan userId yang diberikan server (jika ada)
    // Atau generate client-side dan kirim saat JOIN
    break

// Di CHAT_EVENT:
isOwn: data.payload.userId === localUserId.value
```

Atau lebih sederhananya, simpan userId dari ROOM_INIT response dan gunakan itu.

---

### BUG-12 [KRITIS] — HLS CORS Issue dengan CDN Streams

**File:** `src/components/VideoPlayer.vue:36`

```html
<video
    crossorigin="anonymous"
    referrerpolicy="no-referrer"
    ...
></video>
```

Dan di hls.js config (line 401-405):
```js
hlsInstance = new Hls({
    xhrSetup: (xhr) => {
        xhr.setRequestHeader && void 0 // ← NO-OP! Does nothing!
    }
})
```

**Masalah:**
1. `crossorigin="anonymous"` memaksa browser mengirim CORS request, tapi banyak anime CDN **tidak support CORS** dan akan return 403
2. `xhrSetup` adalah no-op — tidak melakukan apa-apa
3. `referrerpolicy="no-referrer"` bagus untuk privacy, tapi beberapa CDN **memerlukan referer** untuk mengizinkan akses

**Solusi:**
```js
hlsInstance = new Hls({
    xhrSetup: (xhr, url) => {
        // Set referer untuk CDN yang memerlukannya
        const parsedUrl = new URL(url)
        const refererMap = {
            'vidhide.com': 'https://vidhide.com/',
            'filedon.com': 'https://filedon.com/',
        }
        for (const [domain, ref] of Object.entries(refererMap)) {
            if (parsedUrl.hostname.includes(domain)) {
                xhr.setRequestHeader('Referer', ref)
                break
            }
        }
    }
})
```

Dan pertimbangkan **proxy stream** melalui backend untuk mengatasi CORS:
```go
// Backend: Stream proxy endpoint
func HandleStreamProxy(w http.ResponseWriter, r *http.Request) {
    targetURL := r.URL.Query().Get("url")
    // ... fetch and pipe through
}
```

---

### BUG-13 [SEDANG] — Tidak Ada URL Validasi di Frontend

**File:** `src/components/VideoPlayer.vue:675-680` dan `src/components/QueuePanel.vue:146-151`

```js
function submitVideoUrl() {
    const url = newVideoUrl.value.trim()
    if (!url) return  // ← hanya cek empty, tidak validasi format
    emit('set-video', url)
}
```

**Masalah:** User bisa paste text biasa, email, atau URL malformed yang akan menyebabkan error di backend.

**Solusi:**
```js
function isValidInput(value) {
    // Accept: URLs, YouTube video IDs (11 chars), or domain-like strings
    try {
        new URL(value)
        return true
    } catch {
        // Check if it's a YouTube ID
        if (/^[a-zA-Z0-9_-]{11}$/.test(value)) return true
        // Check if it looks like a domain/path
        if (/^[\w.-]+\.\w+/.test(value)) return true
        return false
    }
}
```

---

### BUG-14 [SEDANG] — Deep Watch pada playerState Bisa Trigger Berlebihan

**File:** `src/composables/useRoom.js` dan `src/pages/RoomPage.vue:192`

```js
watch(room.playerState, v => { playerState.value = v }, { deep: true })
```

Dan di VideoPlayer:
```js
watch(() => props.playerState, (state) => { ... }, { deep: true })
```

**Masalah:** Deep watch pada object yang berubah frequently (currentTime update setiap 500ms) akan trigger watcher berlebihan dan bisa menyebabkan performance issue.

**Solusi:** Gunakan `shallowRef` atau selective watch:
```js
// Watch hanya property yang relevan
watch(
    () => [props.playerState.videoUrl, props.playerState.isPlaying],
    ([newUrl, newPlaying], [oldUrl, oldPlaying]) => {
        if (newUrl !== oldUrl) loadVideo(newUrl)
        // Handle play state changes...
    }
)
```

---

### BUG-15 [SEDANG] — Race Condition saat initRoom()

**File:** `src/pages/RoomPage.vue:179-198`

```js
function initRoom() {
    room = useRoom(roomId, nickname.value, isHost)
    wsStatus.value = room.status.value           // ← copy initial value
    // ...
    watch(room.status, v => { wsStatus.value = v }) // ← set watcher AFTER
    room.joinRoom()                                // ← connect AFTER
}
```

**Masalah:** Ada window antara `useRoom()` dan `room.joinRoom()` dimana state bisa berubah tanpa terdeteksi oleh watcher. Juga, `useRoom` langsung watch `status` untuk send JOIN_EVENT, tapi `initRoom` belum setup watchers lokal.

**Solusi:** Setup semua watchers sebelum connect, atau gunakan langsung reactive refs dari composable tanpa copy.

---

### BUG-16 [RENDAH] — Error Message CSS Variable Salah

**File:** `src/pages/HomePage.vue:674`

```css
.error-msg {
    margin-top: -var(--space-2);  /* ← Invalid CSS! */
}
```

**Solusi:** `margin-top: calc(-1 * var(--space-2));`

---

### BUG-17 [RENDAH] — Memory Leak pada WebSocket Event Listeners

**File:** `src/composables/useWebSocket.js:126-129`

```js
if (typeof window !== 'undefined') {
    window.addEventListener('online', handleOnline)
    window.addEventListener('offline', handleOffline)
}
```

**Masalah:** Jika `useWebSocket` dipanggil berulang kali (e.g., re-mount component), event listeners akan bertumpuk.

**Solusi:** Sudah ditangani di `onUnmounted`, tapi pastikan tidak ada multiple instances.

---

## 📐 ARSITEKTUR YANG DIBUTUHKAN: UNIVERSAL SCRAPER

### Current Flow (Hanya Otakudesu):
```
User pastes URL
    ↓
ScrapeStreamURL()
    ↓
Contains "otakudesu"? → YES → scrapeOtakudesu() ✅
    ↓ NO
scrapeGeneric() → ❌ (very limited, usually fails)
```

### Proposed Flow (Universal):
```
User pastes URL
    ↓
ScrapeStreamURL()
    ↓
[1] Check if direct stream URL (.mp4/.m3u8/.webm) → Return directly ✅
    ↓
[2] Detect site platform (domain matching)
    ↓
[3a] Known site? → Use site-specific adapter
[3b] Unknown site? → Use enhanced generic scraper
    ↓
[4] Generic Scraper Pipeline:
    ├── Fetch page HTML
    ├── Search for direct stream URLs in HTML
    ├── Unpack obfuscated JS (Dean Edwards Packer, base64, etc.)
    ├── Search in script tags (JSON configs, JS variables)
    ├── Collect all iframes
    ├── For each iframe (up to 3 levels deep):
    │   ├── Fetch iframe content
    │   ├── Search for stream URLs
    │   ├── Unpack JS
    │   └── Follow nested iframes
    └── Return best match (prefer .m3u8 > .mp4)
```

### Site Adapter Pattern:
```go
type SiteAdapter interface {
    CanHandle(url string) bool
    Scrape(url string) (*models.VideoMetadata, error)
}

var adapters = []SiteAdapter{
    &OtakudesuAdapter{},
    &SamehadakuAdapter{},
    &GenericWordPressAdapter{},
    &GenericAdapter{},  // fallback
}

func ScrapeStreamURL(pageURL string) (*models.VideoMetadata, error) {
    for _, adapter := range adapters {
        if adapter.CanHandle(pageURL) {
            return adapter.Scrape(pageURL)
        }
    }
    return nil, fmt.Errorf("no adapter found for %s", pageURL)
}
```

---

## 🎯 PRIORITAS PERBAIKAN

| # | Issue | Prioritas | Effort |
|---|-------|:---------:|:------:|
| BUG-01 | Universal scraper architecture | 🔴 Tinggi | Besar |
| BUG-02 | Hardcoded AJAX URL | 🔴 Tinggi | Kecil |
| BUG-04 | Queue direct URL handling | 🔴 Tinggi | Kecil |
| BUG-11 | isOwn message detection | 🔴 Tinggi | Kecil |
| BUG-12 | HLS CORS/referrer issue | 🔴 Tinggi | Sedang |
| BUG-03 | M3U8 regex precision | 🟡 Sedang | Kecil |
| BUG-05 | CDN retry/fallback | 🟡 Sedang | Sedang |
| BUG-06 | Room memory cleanup | 🟡 Sedang | Kecil |
| BUG-07 | CORS consistency | 🟡 Sedang | Kecil |
| BUG-13 | Frontend URL validation | 🟡 Sedang | Kecil |
| BUG-14 | Deep watch performance | 🟡 Sedang | Kecil |
| BUG-15 | initRoom race condition | 🟡 Sedang | Sedang |
| BUG-08 | Lock ordering | 🟢 Rendah | Kecil |
| BUG-09 | Queue ID fallback | 🟢 Rendah | Kecil |
| BUG-10 | WebSocket origin check | 🟢 Rendah | Kecil |
| BUG-16 | CSS error message | 🟢 Rendah | Kecil |
| BUG-17 | WS event listener cleanup | 🟢 Rendah | Kecil |

---

## 📝 CATATAN TAMBAHAN

### Mengapa Generic Scraper Gagal untuk Kebanyakan Situs Anime

1. **Content Protection:** Kebanyakan situs anime menggunakan JavaScript obfuscation (eval, packer, encrypt) untuk menyembunyikan stream URL
2. **Multi-layer Embed:** Video biasanya di-embed melalui 2-3 lapis iframe ke provider berbeda (Vidhide, Filemoon, Mp4upload, dll)
3. **Dynamic Loading:** Beberapa situs memuat video URL via AJAX/fetch setelah halaman load, bukan di HTML awal
4. **Anti-Scraping:** Cloudflare protection, rate limiting, CAPTCHA

### Rekomendasi Stack untuk Universal Support

1. **Backend proxy** untuk mengatasi CORS (stream proxy endpoint)
2. **Headless browser** (chromedp/rod untuk Go) sebagai fallback untuk JS-heavy sites
3. **Referer spoofing** di request ke CDN
4. **Cache** hasil scrape untuk menghindari repeated scraping
5. **User-Agent rotation** untuk menghindari blocking

---

*Laporan ini dibuat oleh automated code analysis. Semua line references merujuk ke commit terbaru.*
