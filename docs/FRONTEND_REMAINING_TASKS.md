# Frontend Remaining Tasks — WatchParty Vue Frontend

> **Untuk frontend agent**: Dokumen ini berisi semua task frontend yang belum selesai, diurutkan berdasarkan prioritas. Baca `FRONTEND_LOBBY_TASK.md`, `FRONTEND_METADATA_TASK.md`, dan `FRONTEND_QUEUE_TASK.md` untuk konteks fitur yang backend-nya sudah siap tapi frontend-nya belum/belum sempurna.
>
> Setiap task mandiri — bisa dikerjakan satu per satu. Verifikasi setiap task sebelum lanjut ke task berikutnya.

---

## Task 1 — Wire Lobby: Route + RoomHeader Integration [KRITIS]

### Masalah
Backend **sudah 100% siap** untuk fitur Lobby/Public Room:
- `GET /api/public-rooms` ✅ (endpoint aktif, CORS sudah di-setup)
- `TOGGLE_PUBLIC` WebSocket handler ✅
- `useRoom.js` sudah punya `togglePublic()` dan `isPublic` ref ✅
- `RoomHeader.vue` sudah punya tombol toggle Public/Private dengan styling lengkap ✅

**Tapi ada 2 gap yang bikin fitur ini tidak jalan:**

#### Gap A: Tidak ada route lobby di router
`src/main.js` cuma punya 2 route:
```js
routes: [
    { path: '/',        name: 'home',  component: HomePage },  // ← ini homepage CTA, bukan lobby
    { path: '/room/:roomId', name: 'room', component: RoomPage },
]
```
Tidak ada route `/lobby` atau perubahan pada `/` untuk menampilkan lobby.

#### Gap B: RoomHeader tidak menerima props/events dari RoomPage
`RoomPage.vue` line 13-18:
```vue
<RoomHeader
    :room-id="roomId"
    :nickname="nickname"
    :is-host="isHost"
    :ws-status="wsStatus"
    :participant-count="participants.length || 1"
    <!-- ❌ Tidak ada :is-public -->
    <!-- ❌ Tidak ada @toggle-public -->
/>
```

Padahal `RoomHeader.vue` sudah mendefinisikan:
```js
// Props
isPublic: { type: Boolean, default: false }
// Emits
defineEmits(['toggle-public'])
```

### File yang Diubah

| File | Perubahan |
|------|-----------|
| `src/main.js` | Tambah route `/lobby` |
| `src/pages/LobbyPage.vue` | **BARU** — halaman lobby |
| `src/pages/RoomPage.vue` | Wire `:is-public` dan `@toggle-public` ke RoomHeader |
| `src/pages/HomePage.vue` | Tambah link "Browse Rooms" / "Lobby" di hero section (opsional) |

### Implementasi

#### 1a. Buat `src/pages/LobbyPage.vue`

```vue
<template>
  <div class="lobby-page">
    <!-- Nav (reuse dari HomePage style) -->
    <nav class="home-nav glass">
      <div class="nav-logo">
        <svg width="32" height="32" viewBox="0 0 28 28" fill="none">
          <circle cx="14" cy="14" r="14" fill="url(#lobby-logo-grad)"/>
          <polygon points="11,9 21,14 11,19" fill="white"/>
          <defs>
            <linearGradient id="lobby-logo-grad" x1="0" y1="0" x2="28" y2="28">
              <stop offset="0%" stop-color="hsl(195,100%,45%)"/>
              <stop offset="100%" stop-color="hsl(215,90%,50%)"/>
            </linearGradient>
          </defs>
        </svg>
        <span class="gradient-text nav-brand">WatchParty</span>
      </div>
      <div class="nav-right">
        <router-link to="/" class="nav-link">Home</router-link>
      </div>
    </nav>

    <!-- Header -->
    <header class="lobby-header">
      <h1 class="gradient-text lobby-title">Public Rooms</h1>
      <p class="lobby-subtitle">Join an active watch party or create your own</p>
    </header>

    <!-- Loading -->
    <div v-if="loading" class="lobby-loading">
      <div class="spinner"></div>
      <span>Loading active rooms…</span>
    </div>

    <!-- Empty State -->
    <div v-else-if="rooms.length === 0" class="lobby-empty glass">
      <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1">
        <circle cx="12" cy="12" r="10"/>
        <path d="M8 12h8"/>
      </svg>
      <p>No active public rooms right now</p>
      <small>Be the first to create a watch party!</small>
      <router-link to="/" class="btn btn-primary" style="margin-top: 1rem;">
        Create Room
      </router-link>
    </div>

    <!-- Room Grid -->
    <div v-else class="room-grid">
      <div
        v-for="room in rooms"
        :key="room.roomId"
        class="room-card glass"
        @click="joinRoom(room.roomId)"
      >
        <div class="room-card-thumbnail">
          <img
            v-if="room.currentMetadata?.thumbnail"
            :src="room.currentMetadata.thumbnail"
            :alt="room.currentMetadata?.title || 'Thumbnail'"
            referrerpolicy="no-referrer"
          />
          <div v-else class="room-card-thumb-fallback">
            <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1">
              <polygon points="5 3 19 12 5 21 5 3"/>
            </svg>
          </div>
          <div class="room-card-overlay">
            <span class="room-card-viewers">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/>
                <circle cx="12" cy="12" r="3"/>
              </svg>
              {{ room.participantCount }} watching
            </span>
          </div>
        </div>

        <div class="room-card-info">
          <h3 class="room-card-title">
            {{ getRoomTitle(room) }}
          </h3>
          <p v-if="room.currentMetadata?.episode" class="room-card-episode">
            {{ room.currentMetadata.episode }}
          </p>
          <div class="room-card-meta">
            <span class="room-card-host">{{ room.hostUsername }}</span>
            <span v-if="room.queueSize > 0" class="room-card-queue">
              {{ room.queueSize }} queued
            </span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()
const rooms = ref([])
const loading = ref(true)
let pollInterval = null

async function fetchPublicRooms() {
  try {
    const res = await fetch('/api/public-rooms')
    if (res.ok) {
      rooms.value = await res.json()
    }
  } catch (err) {
    console.error('[Lobby] Failed to fetch rooms:', err)
  } finally {
    loading.value = false
  }
}

function getRoomTitle(room) {
  if (room.roomName) return room.roomName
  if (room.currentMetadata?.title) return room.currentMetadata.title
  return `${room.hostUsername}'s Room`
}

function joinRoom(roomId) {
  router.push({ name: 'room', params: { roomId } })
}

onMounted(() => {
  fetchPublicRooms()
  pollInterval = setInterval(fetchPublicRooms, 15000)
})

onUnmounted(() => {
  if (pollInterval) clearInterval(pollInterval)
})
</script>

<style scoped>
.lobby-page {
  min-height: 100vh;
  padding-bottom: var(--space-16);
}

.lobby-header {
  text-align: center;
  padding: var(--space-10) var(--space-4) var(--space-6);
}

.lobby-title {
  font-size: clamp(1.8rem, 4vw, 2.8rem);
  font-weight: 800;
  letter-spacing: -0.04em;
  margin: 0 0 var(--space-2);
}

.lobby-subtitle {
  color: var(--text-secondary);
  font-size: 1rem;
  margin: 0;
}

.lobby-loading,
.lobby-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-12) var(--space-4);
  text-align: center;
  color: var(--text-muted);
  max-width: 400px;
  margin: 0 auto;
  border-radius: var(--radius-lg);
}

.spinner {
  width: 32px; height: 32px;
  border: 3px solid rgba(255,255,255,0.1);
  border-top-color: var(--color-primary);
  border-radius: 50%;
  animation: spin-slow 0.8s linear infinite;
}

.room-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: var(--space-5);
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 var(--space-6);
}

.room-card {
  border-radius: var(--radius-lg);
  overflow: hidden;
  cursor: pointer;
  transition: all var(--transition-base);
}

.room-card:hover {
  transform: translateY(-4px);
  border-color: rgba(255, 255, 255, 0.12);
  box-shadow: var(--shadow-card), var(--shadow-glow-sm);
}

.room-card-thumbnail {
  position: relative;
  height: 180px;
  overflow: hidden;
  background: #000;
}

.room-card-thumbnail img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.room-card-thumb-fallback {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, rgba(14,16,26,0.9) 0%, rgba(20,24,40,0.95) 100%);
  color: rgba(255,255,255,0.15);
}

.room-card-overlay {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  padding: var(--space-3);
  background: linear-gradient(to top, rgba(0,0,0,0.8) 0%, transparent 100%);
  display: flex;
  justify-content: flex-end;
}

.room-card-viewers {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  border-radius: var(--radius-full);
  font-size: 0.75rem;
  font-weight: 600;
  background: rgba(0, 0, 0, 0.6);
  color: white;
}

.room-card-info {
  padding: var(--space-4);
}

.room-card-title {
  margin: 0 0 var(--space-1);
  font-size: 1rem;
  font-weight: 700;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.room-card-episode {
  margin: 0 0 var(--space-2);
  font-size: 0.8125rem;
  color: var(--text-secondary);
}

.room-card-meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 0.75rem;
  color: var(--text-muted);
}

.room-card-host {
  color: var(--color-primary);
  font-weight: 600;
}
</style>
```

#### 1b. Tambah route di `src/main.js`

```js
import LobbyPage from './pages/LobbyPage.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'home',
      component: HomePage,
    },
    {
      path: '/lobby',
      name: 'lobby',
      component: LobbyPage,
    },
    {
      path: '/room/:roomId',
      name: 'room',
      component: RoomPage,
    },
    {
      path: '/:pathMatch(.*)*',
      redirect: '/',
    },
  ],
})
```

#### 1c. Wire RoomHeader di `src/pages/RoomPage.vue`

Ubah template RoomHeader (line 13-18):

```vue
<RoomHeader
    :room-id="roomId"
    :nickname="nickname"
    :is-host="isHost"
    :ws-status="wsStatus"
    :participant-count="participants.length || 1"
    :is-public="isPublic"
    @toggle-public="onTogglePublic"
/>
```

Tambah di script (cari fungsi `onClearQueue` dan tambah setelahnya):

```js
const isPublic = ref(false)

// Sync dari composable
if (room) {
    watch(room.isPublic, v => { isPublic.value = v })
    isPublic.value = room.isPublic.value
}

function onTogglePublic(val) {
    room?.togglePublic(val)
}
```

Atau, lebih sederhananya — `isPublic` sudah ada di `useRoom.js` return value. Tinggal pastikan di-sync ke template:

```js
// Di initRoom(), tambah:
isPublic.value = room.isPublic.value
watch(room.isPublic, v => { isPublic.value = v })

// Di template RoomHeader, tambah:
// :is-public="isPublic"
// @toggle-public="onTogglePublic"
```

### Verifikasi
```
1. Buka http://localhost:5173/lobby → harusnya tampil lobby page (kosong jika belum ada public room)
2. Buat room → toggle Public di header → room muncul di /lobby
3. Buka /lobby di tab lain → room card tampil dengan thumbnail, title, viewer count
4. Klik room card → masuk ke room tersebut
```

---

## Task 2 — Fix `isOwn` Chat Message Detection [KRITIS]

### Masalah
`src/composables/useRoom.js` line 105:
```js
isOwn: data.payload.userId === nickname, // simplistic, backend will confirm
```

Ini **selalu salah** karena:
- `data.payload.userId` = ID dari backend (misal `"aB3xYz"`)
- `nickname` = nama yang user pilih (misal `"Alice"`)

Akibat: chat messages yang kamu kirim sendiri tidak pernah punya flag `isOwn: true`, jadi styling "own message" (bubble kanan, warna berbeda) tidak pernah muncul.

### Solusi
Simpan `userId` lokal yang dikembalikan server saat `ROOM_INIT`, lalu bandingkan dengan itu.

### File yang Diubah
- `src/composables/useRoom.js`

### Implementasi

```js
// Tambah state baru (setelah participants ref)
const localUserId = ref('')

// Di ROOM_INIT handler (sekitar line 75-88):
case MSG_TYPES.ROOM_INIT:
    if (data.payload) {
        playerState.value.videoUrl    = data.payload.currentVideo || ''
        playerState.value.currentTime = data.payload.currentTime  || 0
        playerState.value.isPlaying   = data.payload.isPlaying    || false
    }
    if (data.payload?.participants) {
        participants.value = data.payload.participants
    }
    currentMetadata.value = data.payload.metadata || null
    queue.value = data.payload.queue || []
    isPublic.value = data.payload.isPublic || false

    // NEW: find our userId from the participants list
    // The server assigns userId on WebSocket connect, and sends it back
    // in ROOM_INIT as part of the participants array.
    // Since we know our nickname, find the matching participant.
    if (data.payload?.participants) {
        const self = data.payload.participants.find(p => p.username === nickname)
        if (self) {
            localUserId.value = self.userId
        }
    }
    break

// Fix CHAT_EVENT handler (line 105):
case MSG_TYPES.CHAT_EVENT:
    messages.value.push({
        id:       Date.now() + Math.random(),
        userId:   data.payload.userId,
        username: data.payload.username,
        text:     data.payload.text,
        sentAt:   data.payload.sentAt,
        isOwn:    data.payload.userId === localUserId.value, // ← FIXED
    })
    break
```

### Verifikasi
```
1. Buka room di 2 tab (host + guest)
2. Kirim chat dari host → bubble muncul di kanan dengan label "(you)"
3. Kirim chat dari guest → bubble muncul di guest tab dengan "(you)"
4. Cek: tidak ada chat yang salah posisi
```

---

## Task 3 — Fix HLS `xhrSetup` Referer [SEDANG]

### Masalah
`src/components/VideoPlayer.vue` line 401-405:
```js
hlsInstance = new Hls({
    xhrSetup: (xhr) => {
        xhr.setRequestHeader && void 0 // no-op; does literally nothing
    }
})
```

Banyak CDN anime (Vidhide, Filedon, dll) memblokir request yang tidak punya Referer header yang benar. Karena `xhrSetup` ini no-op, hls.js fetch `.m3u8` dan `.ts` segments tanpa Referer → CDN return 403.

**Catatan:** Ini hanya bisa full-fix kalau backend **Task 2 (stream proxy)** sudah dibuat. Tanpa proxy, browser CORS policy tetap blokir cross-origin requests. Dengan proxy, frontend tinggal rewrite URL ke `/api/proxy?url=...`.

### Solusi Dua Tahap

#### Tahap A: Rewrite URL ke proxy (butuh backend proxy endpoint)
```js
function getProxiedUrl(url) {
    // If backend proxy is available, route through it
    // to avoid CORS/referrer issues with anime CDNs
    if (url && !url.startsWith('/api/proxy')) {
        return `/api/proxy?url=${encodeURIComponent(url)}`
    }
    return url
}
```

Gunakan di `initHls()`:
```js
function initHls(url) {
    destroyHls()
    const vid = videoRef.value
    if (!vid) return
    isLoading.value = true

    if (Hls.isSupported()) {
        const proxiedUrl = getProxiedUrl(url)
        hlsInstance = new Hls()
        hlsInstance.loadSource(proxiedUrl)
        hlsInstance.attachMedia(vid)
        // ... rest unchanged
    }
}
```

#### Tahap B: Set Referer di xhrSetup (tanpa proxy, partial fix)
```js
hlsInstance = new Hls({
    xhrSetup: (xhr, url) => {
        try {
            const parsed = new URL(url)
            // Set referer to the stream's own origin
            xhr.setRequestHeader('Referer', parsed.origin + '/')
        } catch (e) {
            // ignore
        }
    }
})
```

**Penting:** Tahap B saja tidak cukup karena browser CORS policy tetap blokir cross-origin requests. Tahap A (proxy) adalah solusi utama.

### File yang Diubah
- `src/components/VideoPlayer.vue` — `initHls()` function

### Verifikasi
```
1. Paste URL anime dari vidhide/filedon/mp4upload
2. Sebelum fix: video gagal load, console error 403/CORS
3. Sesudah fix: video bisa diputar (dengan proxy backend aktif)
```

---

## Task 4 — Fix Invalid CSS `margin-top: -var(...)` [SEDANG]

### Masalah
3 file menggunakan CSS syntax yang invalid:
```css
margin-top: -var(--space-2);  /* ← INVALID! */
```

CSS tidak mendukung negasi langsung pada `var()`. Ini diabaikan browser, jadi `margin-top` tidak terpakai.

### File yang Diubah
- `src/pages/HomePage.vue` — line 673
- `src/components/NicknameModal.vue` — line 157, 188

### Implementasi

Ganti semua:
```css
margin-top: -var(--space-2);
```
Menjadi:
```css
margin-top: calc(-1 * var(--space-2));
```

### Verifikasi
```
1. Buka halaman home → error message di bawah join input harus lebih rapat
2. Buka nickname modal (guest join) → error message lebih rapat
3. Cek console: tidak ada CSS parsing warning
```

---

## Task 5 — Validasi URL Input [RENDAH]

### Masalah
`VideoPlayer.vue` dan `QueuePanel.vue` tidak validasi URL sebelum emit. User bisa paste teks biasa, email, atau string random yang akan dikirim ke backend dan gagal di-scrape.

### File yang Diubah
- `src/components/VideoPlayer.vue` — `submitVideoUrl()`
- `src/components/QueuePanel.vue` — `handleAddToQueue()`

### Implementasi

Tambah helper function (bisa di file terpisah atau inline):

```js
function isValidVideoInput(value) {
    if (!value || typeof value !== 'string') return false
    const v = value.trim()
    if (!v) return false

    // Valid URL
    try {
        new URL(v)
        return true
    } catch {}

    // YouTube video ID (11 chars alphanumeric)
    if (/^[a-zA-Z0-9_-]{11}$/.test(v)) return true

    // Looks like a domain/path (e.g., "anoboy.si/episode/...")
    if (/^[\w.-]+\.\w+/.test(v)) return true

    return false
}
```

Gunakan di `submitVideoUrl()`:
```js
function submitVideoUrl() {
    const url = newVideoUrl.value.trim()
    if (!url) return
    if (!isValidVideoInput(url)) {
        // Optional: show error feedback
        return
    }
    emit('set-video', url)
    newVideoUrl.value = ''
}
```

Dan di `handleAddToQueue()`:
```js
function handleAddToQueue() {
    const url = newVideoUrl.value.trim()
    if (!url) return
    if (!isValidVideoInput(url)) return
    emit('add', url)
    newVideoUrl.value = ''
}
```

### Verifikasi
```
1. Paste "hello world" di input → tidak terkirim (tidak ada error di backend)
2. Paste "test@email.com" → tidak terkirim
3. Paste "https://example.com/video.mp4" → terkirim normal
4. Paste "anoboy.si/episode/xxx" → terkirim normal
```

---

## Task 6 — Fix Race Condition `initRoom()` [RENDAH]

### Masalah
`src/pages/RoomPage.vue` line 179-198:
```js
function initRoom() {
    room = useRoom(roomId, nickname.value, isHost)
    // Copy values BEFORE watchers are set up
    wsStatus.value        = room.status.value       // ← snapshot
    messages.value        = room.messages.value     // ← snapshot
    // ...
    // Set up watchers AFTER
    watch(room.status, v => { wsStatus.value = v })
    // ...
    // Connect AFTER everything
    room.joinRoom()
}
```

Masalah: ada window antara `useRoom()` dan `room.joinRoom()` dimana state bisa berubah tanpa terdeteksi. Juga, `useRoom` internal sudah watch `status` untuk send `JOIN_EVENT` — ini bisa trigger sebelum watcher lokal di-setup.

### Solusi
Jangan copy values secara manual. Gunakan langsung reactive refs dari composable, atau setup watchers sebelum connect.

### File yang Diubah
- `src/pages/RoomPage.vue`

### Implementasi

```js
function initRoom() {
    room = useRoom(roomId, nickname.value, isHost)

    // Setup watchers FIRST (before any state changes)
    watch(room.status,          v => { wsStatus.value    = v })
    watch(room.messages,        v => { messages.value    = v }, { deep: true })
    watch(room.playerState,     v => { playerState.value = v }, { deep: true })
    watch(room.participants,    v => { participants.value = v }, { deep: true })
    watch(room.scrapeError,     v => { scrapeError.value  = v })
    watch(room.currentMetadata, v => { currentMetadata.value = v }, { deep: true })
    watch(room.queue,           v => { queue.value = v }, { deep: true })
    watch(room.isPublic,        v => { isPublic.value = v })

    // THEN sync initial values (now any changes will be caught by watchers)
    wsStatus.value        = room.status.value
    messages.value        = room.messages.value
    playerState.value     = room.playerState.value
    participants.value    = room.participants.value
    currentMetadata.value = room.currentMetadata.value
    queue.value           = room.queue.value
    isPublic.value        = room.isPublic.value

    // LAST: connect (triggers JOIN_EVENT internally)
    room.joinRoom()
}
```

### Verifikasi
```
1. Buka room sebagai host → tidak ada flicker/stuck
2. Buka room sebagai guest → langsung masuk, tidak ada delay aneh
3. Buka room dengan video yang sudah playing → langsung sync ke currentTime
```

---

## Ringkasan Prioritas

| # | Task | Prioritas | Effort | Dependency |
|---|------|:---------:|:------:|:----------:|
| 1 | Wire Lobby (route + RoomHeader) | 🔴 Kritis | Sedang | — |
| 2 | Fix `isOwn` chat detection | 🔴 Kritis | Kecil | — |
| 3 | Fix HLS xhrSetup referer | 🟡 Sedang | Kecil | Backend proxy |
| 4 | Fix CSS `margin-top: -var()` | 🟡 Sedang | Kecil | — |
| 5 | Validasi URL input | 🟢 Rendah | Kecil | — |
| 6 | Fix `initRoom()` race condition | 🟢 Rendah | Kecil | — |

**Rekomendasi urutan pengerjaan:**
1. Task 2 (isOwn) — 5 menit fix, langsung improve UX chat
2. Task 4 (CSS) — 5 menit fix, hapus console warnings
3. Task 6 (race condition) — 10 menit fix, improve reliability
4. Task 1 (lobby) — 30-60 menit, biggest user-facing feature
5. Task 5 (URL validation) — 15 menit, polish
6. Task 3 (HLS referer) — 15 menit, tapi butuh backend proxy dulu

---

## Catatan untuk Task 3 (HLS Referer)

Task 3 ini **bergantung pada backend**. Frontend agent bisa:
- **Opsi A:** Tunggu backend selesai bikin `/api/proxy` endpoint, lalu implement `getProxiedUrl()`
- **Opsi B:** Implement tahap B dulu (set Referer di xhrSetup), walau ini partial fix
- **Opsi C:** Skip dulu, fokus ke task lain yang tidak punya dependency

Koordinasi dengan backend agent untuk timeline Task 2 di `BACKEND_REMAINING_TASKS.md`.

---

*Dokumentasi ini dibuat berdasarkan analisis aktual kode per 14 Juli 2026.*
