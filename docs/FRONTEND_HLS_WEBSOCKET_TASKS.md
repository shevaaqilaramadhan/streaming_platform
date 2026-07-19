# Frontend — HLS.js + WebSocket Integration Debugging & Fixes

> **Untuk frontend agent**: Dokumen ini mendefinisikan debugging dan perbaikan untuk integrasi HLS.js + WebSocket yang sudah ada. Backend sudah menyiapkan M3U8 Content Rewriter dan NanoScraper — frontend perlu memastikan semua URL melewati proxy dan sync berjalan lancar.
>
> Baca seluruh file yang disebutkan sebelum mengedit. Verifikasi build setelah setiap task.

---

## Debugging Report: Celah yang Ditemukan

### Celah #1 — `crossorigin="anonymous"` memaksa CORS mode [KRITIS]

**File:** `src/components/VideoPlayer.vue` line 36

```html
<video
    crossorigin="anonymous"
    referrerpolicy="no-referrer"
    ...
></video>
```

**Masalah:** Atribut `crossorigin="anonymous"` memaksa browser mengirim CORS request untuk SEMUA resource yang dimuat oleh `<video>` elemen, termasuk proxied streams. Ketika backend proxy meneruskan stream tanpa CORS header yang tepat (atau CDN tidak support CORS), browser tetap memblokir.

**Ironi:** Proxy backend sudah menambahkan `Access-Control-Allow-Origin: *`, jadi ini seharusnya tidak masalah. TAPI — `crossorigin="anonymous"` juga menghilangkan cookie/credentials yang mungkin diperlukan beberapa CDN.

**Solusi:** Hapus `crossorigin="anonymous"` secara default. Tambahkan hanya jika diperlukan (misalnya untuk YouTube). Atau gunakan `crossorigin="use-credentials"` untuk CDN yang perlu credentials.

---

### Celah #2 — Safari tidak menggunakan proxy [KRITIS]

**File:** `src/components/VideoPlayer.vue` line 447-450

```js
} else if (vid.canPlayType('application/vnd.apple.mpegurl')) {
    // Safari native HLS support
    vid.src = url  // ← LANGSUNG ke URL asli, TIDAK melalui proxy!
    isLoading.value = false
}
```

**Masalah:** Safari menggunakan native HLS player (bukan hls.js). Kode ini langsung set `vid.src = url` tanpa `getProxiedUrl()`. Akibatnya, Safari akan fetch m3u8 dan .ts langsung ke CDN → CORS block.

**Solusi:** Gunakan `getProxiedUrl(url)` juga untuk Safari.

---

### Celah #3 — Native MP4 tidak menggunakan proxy [SEDANG]

**File:** `src/components/VideoPlayer.vue` line 467-473

```js
function initNative(url) {
    const vid = videoRef.value
    if (!vid) return
    isLoading.value = true
    vid.src = url  // ← LANGSUNG ke URL asli
    vid.load()
}
```

**Masalah:** Untuk direct MP4 URLs, `initNative()` langsung set `vid.src` tanpa proxy. Beberapa CDN MP4 juga memblokir tanpa Referer yang benar.

**Solusi:** Gunakan `getProxiedUrl(url)` untuk native MP4 juga.

---

### Celah #4 — `getProxiedUrl()` double-encode URLs [SEDANG]

**File:** `src/components/VideoPlayer.vue` line 393-398

```js
function getProxiedUrl(url) {
    if (url && !url.startsWith('/api/proxy')) {
        return `/api/proxy?url=${encodeURIComponent(url)}`
    }
    return url
}
```

**Masalah:** Jika backend sudah meng-rewrite m3u8 URLs di dalam playlist menjadi `/api/proxy?url=...`, maka setiap segment fetch dari hls.js akan berupa `/api/proxy?url=...`. Fungsi `getProxiedUrl()` akan memeriksa `!url.startsWith('/api/proxy')` — ini benar, TAPI hls.js akan memanggil `xhrSetup` untuk setiap segment, dan URL-nya sudah berupa proxied URL. Jadi ini sebenarnya OK.

**Tapi ada edge case:** Jika CDN meng-redirect ke URL lain, hls.js akan mengikuti redirect dan URL baru tersebut TIDAK akan di-proxy. Perlu pastikan proxy backend mengikuti redirect dan meng-rewrite URL redirect juga.

**Solusi:** Pastikan `getProxiedUrl()` juga handle kasus di mana URL sudah berupa relative path (misalnya `segment0000.ts` yang dihasilkan oleh m3u8 parser hls.js). Tambahkan logic untuk resolve relative URLs.

---

### Celah #5 — Sync drift karena host sync interval terlalu besar [SEDANG]

**File:** `src/components/VideoPlayer.vue` line 484-487

```js
if (props.isHost && Date.now() - lastSyncTime > 2000) {
    lastSyncTime = Date.now()
    if (!vid.paused) emit('sync', { isPlaying: true, currentTime: vid.currentTime })
}
```

**Masalah:** Host hanya sync setiap 2 detik. Guest bisa drift 2+ detik dari host. Untuk watch party, ini cukup mengganggu.

**Solusi:** Kurangi interval ke 1 detik, dan tambahkan duration ke sync payload.

---

### Celah #6 — Guest tidak handle buffering state [RENDAH]

**File:** `src/components/VideoPlayer.vue` — tidak ada handling untuk buffering

**Masalah:** Ketika host buffering (video buffering), guest tidak tahu dan terus play. Ini menyebabkan guest lebih ahead dari host.

**Solusi:** Tambahkan buffering detection dan pause guest saat host buffering.

---

### Celah #7 — Deep watch pada playerState masih ada [RENDAH]

**File:** `src/components/VideoPlayer.vue` line 567-600

```js
watch(
    () => props.playerState,
    (state) => { ... },
    { deep: true }
)
```

**Masalah:** Deep watch pada object yang berubah setiap 1 detik (currentTime) akan trigger watcher berlebihan. Meskipun sudah ada dari sebelumnya, ini masih menyebabkan unnecessary re-renders.

**Solusi:** Ganti dengan selective watch pada `videoUrl` dan `isPlaying` saja. Untuk `currentTime`, gunakan watch terpisah yang lebih jarang (misalnya debounced).

---

## Task 1 — Fix Proxy URL Rewriting untuk Semua Player Modes [KRITIS]

### File yang Diubah
- `src/components/VideoPlayer.vue`

### Implementasi

#### 1a. Hapus `crossorigin="anonymous"` dari `<video>`

```html
<video
    v-show="videoUrl && videoMode !== 'youtube'"
    ref="videoRef"
    id="watch-party-video"
    class="video-el"
    preload="metadata"
    playsinline
    <!-- HAPUS: crossorigin="anonymous" -->
    referrerpolicy="no-referrer"
    ...
></video>
```

#### 1b. Fix Safari HLS — gunakan proxy

```js
} else if (vid.canPlayType('application/vnd.apple.mpegurl')) {
    // Safari native HLS support — also route through proxy
    vid.src = getProxiedUrl(url)
    isLoading.value = false
}
```

#### 1c. Fix Native MP4 — gunakan proxy

```js
function initNative(url) {
    const vid = videoRef.value
    if (!vid) return
    isLoading.value = true
    vid.src = getProxiedUrl(url)
    vid.load()
}
```

#### 1d. Update `getProxiedUrl()` untuk handle relative URLs

```js
function getProxiedUrl(url) {
    if (!url) return url
    // Already proxied
    if (url.startsWith('/api/proxy')) return url
    // Relative URL (from m3u8 segment) — resolve against current page
    if (!url.startsWith('http')) {
        // This is a relative segment URL from hls.js
        // It should already be handled by the m3u8 rewriter on backend
        return url
    }
    return `/api/proxy?url=${encodeURIComponent(url)}`
}
```

### Verifikasi
```
1. Paste .m3u8 URL → video harus play di Chrome DAN Safari
2. Paste .mp4 URL → video harus play
3. DevTools Network: semua request harus ke /api/proxy (bukan langsung ke CDN)
```

---

## Task 2 — Fix HLS.js Error Recovery [KRITIS]

### Masalah
Error recovery saat ini (`hlsInstance.startLoad()` dan `hlsInstance.recoverMediaError()`) tidak cukup. Jika proxy gagal (502/504), hls.js akan retry ke URL yang sama yang juga gagal.

### File yang Diubah
- `src/components/VideoPlayer.vue` — `initHls()` error handler

### Implementasi

```js
hlsInstance.on(Hls.Events.ERROR, (_, data) => {
    console.error('[HLS] Error:', data.type, data.details, data.fatal)

    if (data.fatal) {
        switch (data.type) {
            case Hls.ErrorTypes.NETWORK_ERROR:
                console.warn('[HLS] Fatal network error — details:', data.details)
                // If it's a manifest load error, the URL might be wrong
                if (data.details === Hls.ErrorDetails.MANIFEST_LOAD_ERROR ||
                    data.details === Hls.ErrorDetails.MANIFEST_LOAD_TIMEOUT) {
                    console.error('[HLS] Cannot load manifest. Check if proxy is running.')
                    // Don't retry infinitely — show error to user
                    isLoading.value = false
                } else {
                    // For segment errors, try to recover
                    console.warn('[HLS] Attempting network recovery...')
                    hlsInstance.startLoad()
                }
                break
            case Hls.ErrorTypes.MEDIA_ERROR:
                console.warn('[HLS] Fatal media error — attempting recovery...')
                hlsInstance.recoverMediaError()
                break
            default:
                console.error('[HLS] Unrecoverable error, destroying instance')
                destroyHls()
                isLoading.value = false
                break
        }
    }
})
```

### Verifikasi
```
1. Paste URL yang salah/non-existent → harusnya show error, bukan infinite retry
2. Paste URL valid → video play normal
3. Matikan backend proxy → video harus show error message, bukan hang
```

---

## Task 3 — Sync Enhancement: Duration + Latency Compensation [SEDANG]

### File yang Diubah
- `src/components/VideoPlayer.vue` — sync emission
- `src/composables/useRoom.js` — sync handling

### Implementasi

#### 3a. Tambah duration ke sync payload (VideoPlayer.vue)

```js
function onNativeTimeUpdate() {
    const vid = videoRef.value
    if (!vid) return
    currentTime.value = vid.currentTime
    duration.value = vid.duration || 0

    if (props.isHost && Date.now() - lastSyncTime > 1000) {  // ← 1 detik, bukan 2
        lastSyncTime = Date.now()
        if (!vid.paused) emit('sync', {
            isPlaying: true,
            currentTime: vid.currentTime,
            duration: vid.duration || 0
        })
    }
}
```

#### 3b. Terima duration di guest (useRoom.js)

```js
case MSG_TYPES.SYNC_EVENT:
    if (!isHost) {
        playerState.value.isPlaying   = data.payload.playerState === 'PLAYING'
        playerState.value.currentTime = data.payload.currentTime
        playerState.value.duration    = data.payload.duration || 0  // NEW
    }
    break
```

#### 3c. Latency compensation di guest

```js
case MSG_TYPES.SYNC_EVENT:
    if (!isHost) {
        const latency = data.payload.serverAt
            ? (Date.now() - data.payload.serverAt) / 1000
            : 0
        playerState.value.isPlaying   = data.payload.playerState === 'PLAYING'
        playerState.value.currentTime = data.payload.currentTime + latency
    }
    break
```

### Verifikasi
```
1. Buka room, play video di host
2. Buka guest tab → harusnya langsung sync ke posisi terkini (dengan latency compensation)
3. Guest harus dalam ±0.5 detik dari host (bukan ±2 detik seperti sebelumnya)
```

---

## Task 4 — Selective Watch untuk PlayerState [RENDAH]

### Masalah
Deep watch pada `playerState` trigger berlebihan karena `currentTime` berubah setiap 1 detik.

### File yang Diubah
- `src/components/VideoPlayer.vue` — watcher di line 567

### Implementasi

```js
// REPLACE deep watch dengan selective watches:

// Watch 1: Video URL changes → load new video
watch(() => props.videoUrl, (newUrl) => { loadVideo(newUrl) })

// Watch 2: Play state changes → play/pause
watch(
    () => props.playerState.isPlaying,
    (newPlaying) => {
        if (props.isHost) return
        const vid = videoRef.value
        if (!vid) return
        if (newPlaying && vid.paused) vid.play().catch(() => {})
        else if (!newPlaying && !vid.paused) vid.pause()
    }
)

// Watch 3: Current time changes → seek (debounced)
let seekTimeout = null
watch(
    () => props.playerState.currentTime,
    (newTime) => {
        if (props.isHost) return
        const vid = videoRef.value
        if (!vid) return
        // Debounce seek to avoid excessive seeks
        clearTimeout(seekTimeout)
        seekTimeout = setTimeout(() => {
            if (Math.abs(vid.currentTime - newTime) > 1.5) {
                if (vid.readyState >= 1) {
                    vid.currentTime = newTime
                } else {
                    pendingSeekTime.value = newTime
                }
            }
        }, 300)
    }
)
```

### Verifikasi
```
1. Play video sebagai host, buka guest tab
2. Guest harus play/pause sinkron dengan host
3. Guest harus seek ke posisi host saat host seek
4. Performance: tidak ada excessive re-renders (cek React DevTools atau Vue DevTools)
```

---

## Task 5 — Error Feedback ke User [RENDAH]

### Masalah
Saat stream gagal (scrape error, proxy error, HLS error), user hanya melihat console error. Tidak ada UI feedback.

### File yang Diubah
- `src/components/VideoPlayer.vue` — tambah error state
- `src/pages/RoomPage.vue` — tampilkan error

### Implementasi

```js
// Di VideoPlayer.vue, tambah emit untuk errors:
const streamError = ref(null)

// Di HLS error handler:
if (data.fatal && data.type === Hls.ErrorTypes.NETWORK_ERROR) {
    streamError.value = 'Failed to load stream. The video source may be unavailable.'
    emit('error', streamError.value)
}

// Di RoomPage.vue, handle error:
<VideoPlayer
    ...
    @error="onStreamError"
/>

function onStreamError(msg) {
    // Show toast or inline error
    console.error('[Stream Error]', msg)
}
```

---

## Ringkasan Prioritas

| # | Task | Prioritas | Effort | Dependency |
|---|------|:---------:|:------:|:----------:|
| 1 | Fix proxy URL rewriting (all modes) | 🔴 Kritis | Sedang | Backend proxy |
| 2 | Fix HLS error recovery | 🔴 Kritis | Kecil | — |
| 3 | Sync enhancement (duration + latency) | 🟡 Sedang | Kecil | Backend sync |
| 4 | Selective watch (performance) | 🟢 Rendah | Kecil | — |
| 5 | Error feedback UI | 🟢 Rendah | Kecil | — |

**Rekomendasi urutan:**
1. Task 1 (proxy URL) — langsung solve CORS untuk semua player modes
2. Task 2 (error recovery) — improve reliability
3. Task 3 (sync) — improve watch experience
4. Task 4 (selective watch) — performance polish
5. Task 5 (error UI) — UX polish

---

## Integrasi dengan Backend

Frontend tasks ini **bergantung pada backend** untuk:
1. **M3U8 Content Rewriter** (BACKEND_NANOPROXY_TASKS.md Task 2) — agar m3u8 URLs di-rewrite
2. **WebSocket Sync Enhancement** (BACKEND_NANOPROXY_TASKS.md Task 4) — agar duration + serverAt tersedia

Koordinasi dengan backend agent:
- Backend Task 2 harus selesai dulu sebelum frontend Task 1 bisa full-test
- Backend Task 4 harus selesai dulu sebelum frontend Task 3 bisa full-test
- Frontend Task 2, 4, 5 bisa dikerjakan tanpa dependency backend

---

## Diagram Alur Akhir

```
User paste URL
    │
    ▼
Frontend: room.setVideo(url)
    │
    ▼
Backend: ScrapeStreamURL()
    ├─ scrapeGeneric() [HTML parsing]
    ├─ NanoScrape() [chromedp + network intercept] ← NEW
    └─ scrapeWithHeadless() [chromedp HTML fallback]
    │
    ▼
Backend: broadcast SET_VIDEO { videoUrl: "https://cdn.example.com/stream.m3u8" }
    │
    ▼
Frontend: loadVideo("https://cdn.example.com/stream.m3u8")
    │
    ├─ detectMode() → 'hls'
    │
    ▼
initHls("https://cdn.example.com/stream.m3u8")
    │
    ├─ getProxiedUrl() → "/api/proxy?url=https%3A%2F%2Fcdn.example.com%2Fstream.m3u8"
    │
    ▼
hls.loadSource("/api/proxy?url=...")
    │
    ▼
Backend /api/proxy:
    ├─ Fetch m3u8 dari CDN (dengan Referer/UA)
    ├─ REWRITE: segment0000.ts → /api/proxy?url=https%3A%2F%2Fcdn.example.com%2Fsegment0000.ts
    └─ Return rewritten m3u8
    │
    ▼
hls.js parse rewritten m3u8 → fetch segments melalui /api/proxy
    │
    ▼
Video playback berhasil ✅
```

---

*Dokumentasi ini dibuat berdasarkan debugging aktual kode frontend per 15 Juli 2026.*
