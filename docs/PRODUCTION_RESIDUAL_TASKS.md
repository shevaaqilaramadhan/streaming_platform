# Production Residual Tasks — WatchParty Platform

**Tanggal:** 19 Juli 2026  
**Sumber:** Production readiness review pasca fix `NEXT_DAY_WORK_REPORT.md`  
**Verdict review:** READY WITH CAVEATS  
**Tests:** `go test` 25 passed · `npm run build` OK  

---

## Ringkasan

| Sektor | Must-fix | Nice-to-fix | Total |
|--------|:--------:|:------------:|:-----:|
| **Backend** | 1 | 2 | 3 |
| **Frontend** | 2 | 1 | 3 |
| **Ops** | 1 | 2 | 3 |

Hampir semua P0 dari laporan sebelumnya sudah fixed. File ini hanya berisi **residual bugs / regressions** yang harus dikerjakan sebelum hard production.

---

## Backend

### B-RES-1: Reconnect tanpa re-JOIN (HIGH — must-fix)

- **File:** `backend/services/message_handler.go` + `backend/handlers/websocket.go` (koordinasi dengan frontend F-RES-1)
- **Masalah:** Setelah WS disconnect, client reconnect membuka koneksi baru. Frontend (saat ini) tidak mengirim `JOIN_EVENT` lagi → user tidak masuk `room.Clients`, chat/sync/broadcast mati.
- **Opsi fix (pilih salah satu atau kombinasikan):**
  1. **Frontend-led (disarankan cepat):** Frontend kirim `JOIN_EVENT` setiap kali status `CONNECTED` (lihat F-RES-1). Backend tetap validasi username/room seperti biasa.
  2. **Backend session resume:** Saat upgrade WS, terima query `?userId=&token=` atau re-JOIN otomatis dari cookie/session — lebih kompleks.
- **Backend checklist:**
  - [ ] Pastikan `handleJoinEvent` idempotent / aman jika dipanggil ulang dengan user baru di room yang sama
  - [ ] Pastikan `RemoveUserFromRoom` + host reassignment tetap benar saat reconnect race
  - [ ] Jangan panic jika `JOIN_EVENT` datang dua kali dari client yang sama (user ID baru vs lama)
- **Verifikasi:** Disconnect network 5s → reconnect → user muncul di participants, chat & sync jalan.

### B-RES-2: WebSocket CheckOrigin terbuka (LOW — nice-to-fix)

- **File:** `backend/handlers/websocket.go:39-41`
- **Masalah:** `CheckOrigin` selalu `return true`. HTTP CORS sudah dikunci via `CORS_ORIGINS`, tapi WS origin masih bebas (CSWSH risk di browser).
- **Fix:**
  ```go
  // Reuse allowed origins dari main / shared helper
  CheckOrigin: func(r *http.Request) bool {
      origin := r.Header.Get("Origin")
      if origin == "" {
          return true // non-browser clients
      }
      return isOriginAllowed(origin) // sama logic CORS_ORIGINS
  }
  ```
- [ ] Extract `isOriginAllowed` shared (hindari duplikasi dengan `corsMiddleware`)
- [ ] Dev: localhost tetap diizinkan saat `CORS_ORIGINS` kosong

### B-RES-3: Rate limiting penuh (LOW — nice-to-fix)

- **File:** global (`main.go`, middleware baru, atau per-handler)
- **Status:** Hard caps sudah ada (`MaxUsersPerRoom=20`, queue cap, chat text 500). Belum ada per-IP rate limit.
- **Fix (minimal):**
  - [ ] Rate limit `/api/proxy` (e.g. 30 req/min/IP)
  - [ ] Rate limit chat / SET_VIDEO / scrape triggers per user
  - [ ] Optional: `golang.org/x/time/rate` atau token bucket sederhana in-memory

---

## Frontend

### F-RES-1: Re-JOIN on reconnect (HIGH — must-fix)

- **File:** `src/composables/useRoom.js:65-79`
- **Masalah:**
  ```js
  let hasJoinedOnce = false
  watch(status, (newStatus) => {
    if (newStatus === WS_STATUS.CONNECTED && !hasJoinedOnce) {
      hasJoinedOnce = true
      send({ action: JOIN_EVENT, ... })
    }
  })
  ```
  Setelah reconnect, `hasJoinedOnce === true` → tidak kirim `JOIN_EVENT` → user ghost (tidak di room server-side).
- **Fix:**
  ```js
  watch(status, (newStatus) => {
    if (newStatus === WS_STATUS.CONNECTED) {
      send({
        action: MSG_TYPES.JOIN_EVENT,
        payload: { roomId, username: nickname, isHost: isHostHint },
      })
    }
  })
  ```
  - Atau: reset `hasJoinedOnce = false` saat status jadi `DISCONNECTED` / `RECONNECTING`.
- **Catatan:** Backend membuat user baru per koneksi; re-JOIN wajib. Host hint hanya hint — server authoritative via `ROOM_INIT.isHost`.
- **Verifikasi:** Buka room → DevTools offline → online → pastikan `JOIN_EVENT` terkirim, `ROOM_INIT` diterima, participants update.

### F-RES-2: Room switch tidak re-init (MED — must-fix)

- **File:** `src/pages/RoomPage.vue` + opsional `src/App.vue`
- **Masalah:** `roomId` sudah `computed`, tapi `initRoom()` hanya dipanggil di `onMounted` / nickname modal. Navigasi `/room/A` → `/room/B` **me-reuse** komponen (Vue Router default) → WS tetap di room A, state stale.
- **Fix opsi A (RoomPage):**
  ```js
  watch(roomId, (newId, oldId) => {
    if (!newId || newId === oldId) return
    // teardown
    roomWatchers.forEach(stop => stop())
    roomWatchers = []
    room?.leave()
    room = null
    // reset local state jika perlu
    if (hasJoined.value || hostHint.value) {
      initRoom()
    }
  })
  ```
- **Fix opsi B (App.vue — lebih sederhana):**
  ```vue
  <RouterView :key="$route.fullPath" />
  ```
  Memaksa remount tiap navigasi (termasuk ganti roomId).
- **Rekomendasi:** Opsi B cepat & aman; Opsi A jika ingin avoid full remount.
- **Verifikasi:** Dari room A klik join room B (atau ganti URL) → WS URL `/ws/B`, chat/queue room B.

### F-RES-3: `unregister` message handler tidak dipanggil (LOW — nice-to-fix)

- **File:** `src/composables/useRoom.js:82` + `leave()`
- **Masalah:** `const unregister = onMessage(...)` tidak pernah dipanggil. Saat `leave()` hanya `disconnect()`. Jika `initRoom()` dipanggil ulang tanpa unmount (F-RES-2 opsi A), handler menumpuk.
- **Fix:**
  ```js
  function leave() {
    unregister?.()
    disconnect()
  }
  ```
- [ ] Pastikan `onMessage` di `useWebSocket.js` memang return fungsi unregister

---

## Ops / Deploy (bukan code, tapi wajib)

### O-RES-1: Env production (must-set)

| Env | Wajib | Contoh |
|-----|:-----:|--------|
| `CORS_ORIGINS` | Ya | `https://watchparty.example.com` |
| `PROXY_PUBLIC_BASE` | Ya (jika M3U8 rewrite absolute) | `https://api.example.com` |
| `LISTEN_ADDR` | Opsional | `:8080` |
| `PROXY_ALLOWLIST_EXTRA` | Opsional | `cdn.partner.com,stream.foo.net` |
| `NANOPROXY_HTTP_URL` | Opsional | `http://nanoproxy:8080` |
| `VITE_WS_URL` | Opsional (frontend build) | `wss://api.example.com` |

- [ ] Document di README / deploy notes
- [ ] Jangan deploy dengan `CORS_ORIGINS` kosong di production (localhost fallback)

### O-RES-2: Monitoring (nice-to-fix)

- [ ] Log/metric: scrape latency, scrape error rate
- [ ] Proxy 403 rate (allowlist miss)
- [ ] WS connect/disconnect rate, room count
- [ ] Panic recover di `safeSend` (harusnya jarang — alert jika spike)

### O-RES-3: Frontend bundle (nice-to-fix)

- Chunk ~724KB — pertimbangkan dynamic import untuk pages (docs/faq/status)

---

## Urutan pengerjaan

### Sesi 1 — Must-fix (30–60 menit)

| # | ID | Owner | Task |
|---|-----|-------|------|
| 1 | **F-RES-1** | Frontend | Re-JOIN setiap CONNECTED |
| 2 | **F-RES-2** | Frontend | Room switch re-init / RouterView key |
| 3 | **B-RES-1** | Backend | Verifikasi handleJoin idempotent + race reconnect |

### Sesi 2 — Hardening (30 menit)

| # | ID | Owner | Task |
|---|-----|-------|------|
| 4 | **F-RES-3** | Frontend | Call `unregister()` di `leave()` |
| 5 | **B-RES-2** | Backend | WS CheckOrigin = CORS allowlist |
| 6 | **O-RES-1** | Ops | Set env prod + doc |

### Sesi 3 — Optional

| # | ID | Owner | Task |
|---|-----|-------|------|
| 7 | **B-RES-3** | Backend | Per-IP rate limit |
| 8 | **O-RES-2/3** | Ops/FE | Monitoring + code-split |

---

## Acceptance criteria (production gate)

- [ ] Reconnect 5s: user kembali ke room, chat & sync jalan
- [ ] Navigasi room A → room B: state & WS di room B
- [ ] `go test ./...` pass
- [ ] `npm run build` pass
- [ ] `CORS_ORIGINS` di-set di staging/prod
- [ ] Tidak ada panic di log saat multi-user join/leave spam

---

## File terkait

| File | Status |
|------|--------|
| `NEXT_DAY_WORK_REPORT.md` | 62 issue awal — mayoritas sudah fixed |
| **`PRODUCTION_RESIDUAL_TASKS.md`** | ← Ini (residual sebelum hard prod) |

---

*Berdasarkan deep review kode + `go test` 25 passed + Vite build OK, 19 Juli 2026.*
