# Laporan Kerja Esok Hari — WatchParty Platform

**Tanggal Laporan:** 17 Juli 2026 (malam)  
**Untuk:** Esok hari (18 Juli 2026)  
**Status Build:** `go build` OK · `go test` 24 passed  
**Total Bug Ditemukan:** 36 backend + 26 frontend = **62 issue**

---

## 📊 Ringkasan Eksekutif

| Sektor | Kritis | Sedang | Rendah | Total |
|--------|:------:|:------:|:------:|:-----:|
| **Backend** | 5 | 14 | 8 | 36 |
| **Frontend** | 5 | 11 | 5 | 26 |
| **Total** | **10** | **25** | **13** | **62** |

### Yang Sudah Selesai Hari Ini (17 Juli)
- SSRF proxy protection + dynamic CDN allowlist
- Host privilege escalation fix + auto-reassignment
- EPISODE_ENDED auth + debounce
- WebSocket goroutine leak fix (sync.Once)
- Scrape cache memory leak fix
- Lock order deadlock fix
- GetRoomState data race fix
- IDLIX API stream chain (6-step)
- Samehadaku AJAX player scraper (no headless needed)
- Headless abort-on-match + expanded ad blocking
- Stealth chromedp configuration
- Dynamic stream domain cache
- All 24 backend tests passing

---

## 🔴 KRITIS — Backend (5 issue)

### B-P0-1: Panic — send on closed channel
- **File:** `handlers/websocket.go:66-70` + `services/message_handler.go:429-442`
- **Masalah:** `cleanup()` menutup `user.Send`. Secara konkruen, `BroadcastToRoom` masih mengirim ke channel tersebut → **panic** (crash server)
- **Fix:** Hapus user dari `room.Clients` SEBELUM `close(user.Send)`. Tambah `defer recover()` di send helper.
- **Urutan:** remove from map → close(Send) → Conn.Close()

### B-P0-2: SET_VIDEO tanpa cek host
- **File:** `services/message_handler.go:158-203`
- **Masalah:** Semua guest bisa mengganti video room (dan trigger scrape mahal). Padahal `REMOVE_FROM_QUEUE`, `SKIP_TO_NEXT`, dll. sudah cek host.
- **Fix:** Tambah `if !user.IsHost { return }` di awal `handleSetVideo`

### B-P0-3: Chat spoofing + tanpa validasi
- **File:** `services/message_handler.go:143-156`
- **Masalah:** Server meneruskan `payloadRaw` mentah dari client. Client bisa set `userId`/`username`/`text` sembarangan (bisa sangat panjang = memory abuse)
- **Fix:** Timpa `userId` = `user.ID`, `username` = `user.Username`. Batasi text 1-500 karakter. Marshal ulang server-side.

### B-P0-4: DNS rebinding SSRF pada proxy
- **File:** `handlers/proxy.go:344-418`
- **Masalah:** `isPrivateIP` resolve hostname sekali, tapi `proxyClient.Do` resolve lagi. DNS attacker bisa return public IP saat check, private IP saat connect (rebinding)
- **Fix:** Custom `DialContext` yang resolve + re-check IP sebelum dial. Atau IP pinning.

### B-P0-5: Allowlist substring terlalu luas
- **File:** `handlers/proxy.go:46-71`
- **Masalah:** Token seperti `"cdn."`, `"anime"`, `"m3u8"`, `"plyr"` match banyak host tidak terkait (contoh: `not-anime-evil.com`)
- **Fix:** Gunakan exact match atau suffix label: `host == x || strings.HasSuffix(host, "."+x)`. Hapus token ultra-generic.

---

## 🟠 TINGGI — Backend (9 issue)

| # | File | Masalah | Fix |
|---|------|---------|-----|
| B-P1-1 | `handlers/public_rooms.go:37-53` | Data race: `CurrentMetadata` pointer hidup dibaca setelah unlock | Deep-copy under lock (sama seperti `GetRoomState`) |
| B-P1-2 | `queue_manager.go:120-132` | `GetQueue` shallow-copy pointer | Deep-copy setiap item |
| B-P1-3 | `message_handler.go:335-343` | EPISODE_ENDED debounce tidak atomic (check-then-act) | Per-room mutex atau fold ke `SkipToNext` |
| B-P1-4 | `websocket.go:103` | Scrape blocking readPump (IDLIX 30-90s) | Jalankan scrape di goroutine terpisah |
| B-P1-5 | `scraper.go:83-118` | Cache stampede + shared mutable pointer | Gunakan `singleflight.Group` + return copy |
| B-P1-6 | `message_handler.go:122-141` | SYNC_EVENT tanpa host check | Host-only atau validate range |
| B-P1-7 | `websocket.go:77-116` | WS tanpa read limit / ping-pong / write deadline | Tambah `SetReadLimit(64KB)`, `SetReadDeadline`, ping ticker |
| B-P1-8 | `room_manager.go:191-201` | Room dihapus langsung saat kosong | Tambah grace period 2-5 menit sebelum delete |
| B-P1-9 | `message_handler.go:14` | `lastEpisodeEnded` map tidak pernah di-prune | Hapus key saat room dihapus |

---

## 🟡 SEDANG — Backend (14 issue)

| # | File | Masalah | Fix |
|---|------|---------|-----|
| B-P2-1 | `main.go:52-56` | `Encode` lalu `http.Error` (partial write corrupt) | Encode ke buffer dulu |
| B-P2-2 | `proxy.go:185,233` | `io.Copy` / `w.Write` error diabaikan | Log error |
| B-P2-3 | `proxy.go:147-175` | OPTIONS handler setelah upstream fetch | Handle OPTIONS di awal |
| B-P2-4 | `scraper.go:399` | `json.Unmarshal` error diabaikan (otakudesu mirror) | Skip mirror jika gagal |
| B-P2-5 | `scraper.go:160-194` | `fetchWithRetry` ada tapi tidak dipakai | Wire ke `fetchPageHTML`/`fetchPageDocument` |
| B-P2-6 | `main.go:25-39` | CORS terbuka lebar (refleksi Origin) | Env-based allowlist |
| B-P2-7 | `main.go:12-21` | Tanpa graceful shutdown | `http.Server` + `signal.Notify` + `Shutdown` |
| B-P2-8 | `websocket.go:35-39` | Tanpa validasi input (username, roomID, URL) | Charset + length validation |
| B-P2-9 | `room_manager.go:124-146` | `AddUserToRoom` masih punya param `isHost` (dead code) | Hapus atau align |
| B-P2-10 | `proxy.go:291,318` | M3U8 rewrite pakai relative URL `/api/proxy` | Gunakan absolute URL dari config |
| B-P2-11 | `scraper.go:24-32` | Scraper HTTP client tanpa SSRF guard | Block private IP sebelum fetch |
| B-P2-12 | `utils/crypto.go:19-22` | `GenerateShortID` panic saat RNG gagal | Return error / fallback |
| B-P2-13 | `queue_manager.go` + `message_handler.go` | `SkipToNext` + state update terpisah (race) | Satu fungsi `AdvanceQueue` under single lock |
| B-P2-14 | Global | Tanpa rate limit (queue, proxy, chat, scrape) | Caps: queue 50, room 20 users, rate limit |

---

## 🟢 RENDAH — Backend (8 issue)

| # | File | Masalah |
|---|------|---------|
| B-P3-1 | `scraper.go` | `isWordPressSite` tidak dipakai |
| B-P3-2 | `room_manager.go:124` | `AddUserToRoom` dead code |
| B-P3-3 | `proxy.go` | `IsStreamURL` tidak diekspor/dipakai |
| B-P3-4 | `models/types.go` | `JoinPayload.IsHost` masih ada (server ignore) |
| B-P3-5 | `stream_domain_cache.go` | Tanpa periodic TTL purge (hanya saat >500 entry) |
| B-P3-6 | `idlix_scraper.go:228` | `time.Sleep(1500ms)` ignore context |
| B-P3-7 | `chrome_stealth.go` | Block images bisa break poster-only flow |
| B-P3-8 | `public_rooms.go:63-66` | Same encode-then-Error pattern |

---

## 🔴 KRITIS — Frontend (5 issue)

### F-P0-1: `roomId` non-reactive — stale closure
- **File:** `RoomPage.vue:159-161`
- **Masalah:** `const roomId = route.params.roomId` bukan reactive. Jika user navigasi antar room, roomId tidak update.
- **Fix:** `const roomId = computed(() => route.params.roomId)`

### F-P0-2: Memory leak — message handler tidak pernah di-unregister
- **File:** `RoomPage.vue:171` + `useRoom.js:81`
- **Masalah:** `unregister` dari `onMessage` tidak pernah dipanggil. WS tidak pernah di-close saat unmount.
- **Fix:** Tambah `onUnmounted(() => { room?.leave?.() })` di RoomPage

### F-P0-3: `createRoom()` menelan error tanpa feedback
- **File:** `HomePage.vue:369-385`
- **Masalah:** Backend down → user dialihkan ke local room tanpa tahu kenapa room tidak jalan.
- **Fix:** Set `errorMsg.value` sebelum fallback

### F-P0-4: Missing `onUnmounted` — watchers + WS bocor
- **File:** `RoomPage.vue:150`
- **Masalah:** Tidak ada cleanup saat navigasi away. Watchers dan WS connection leak.
- **Fix:** Import `onUnmounted`, tambah cleanup

### F-P0-5: Invalid CSS `rgba()` dengan HSL
- **File:** `DocumentationPage.vue:754`
- **Masalah:** `rgba(195, 100%, 45%, 0.15)` — browser abaikan. Icon box tidak punya background.
- **Fix:** Ganti ke `hsla(195, 100%, 45%, 0.15)`

---

## 🟠 TINGGI — Frontend (6 issue)

| # | File | Masalah | Fix |
|---|------|---------|-----|
| F-P1-1 | Semua page (640px) | Nav links hilang di mobile — tidak ada hamburger | Tambah hamburger menu |
| F-P1-2 | `HomePage.vue:347-349` | Footer `#privacy` / `#terms` link mati | Hapus atau buat halaman |
| F-P1-3 | `LobbyPage.vue:107-117` | API error tampilkan "No rooms" (misleading) | Tambah error state |
| F-P1-4 | `RoomHeader.vue:134` | `clipboard.writeText()` tanpa `.catch()` | Tambah error handler |
| F-P1-5 | `VideoPlayer.vue:605` | `seekTimeout` tidak dibersihkan saat unmount | Tambah `clearTimeout(seekTimeout)` |
| F-P1-6 | `useRoom.js:66-78` | `JOIN_EVENT` dikirim setiap reconnect | Hanya kirim pada connect pertama |

---

## 🟡 SEDANG — Frontend (5 issue)

| # | File | Masalah | Fix |
|---|------|---------|-----|
| F-P2-1 | `useRoom.js` + `ChatPanel` | Chat messages tanpa batas (unbounded growth) | Batasi 200 pesan |
| F-P2-2 | `VideoPlayer.vue:704-713` | Controls hilang saat user drag volume slider | Track `isInteracting` state |
| F-P2-3 | `FAQPage.vue:164` | Tidak ada tombol "All" tab | Tambah button "All" |
| F-P2-4 | `DocumentationPage.vue` | WS API docs hanya 5/11 event | Tambah event lengkap |
| F-P2-5 | `ServerStatusPage.vue` | Semua data mock — tidak ada API real | Tambah disclaimer atau fetch real data |

---

## 🟢 RENDAH — Frontend (5 issue)

| # | File | Masalah |
|---|------|---------|
| F-P3-1 | `HelloWorld.vue` | Dead component — tidak dipakai |
| F-P3-2 | `src/router/` | Empty directory — sisa scaffold |
| F-P3-3 | `ConstellationBackground.vue` | `ctx` tidak di-null saat unmount |
| F-P3-4 | `VideoPlayer.vue` + `QueuePanel.vue` | `isValidVideoInput` duplikat (extract ke utility) |
| F-P3-5 | `DocumentationPage.vue:484-493` | Scroll handler tanpa throttle |

---

## 📋 Urutan Pengerjaan yang Direkomendasikan

### Sesi 1: Backend Kritis (1-2 jam)
1. **B-P0-1** — Safe send protocol (fix panic crash) ← **PALING PENTING**
2. **B-P0-2** — Host check SET_VIDEO
3. **B-P0-3** — Chat server-side identity
4. **B-P0-5** — Tighten allowlist matching

### Sesi 2: Backend Reliability (1-2 jam)
5. **B-P1-4** — Async scrape off readPump
6. **B-P1-5** — singleflight scrape cache
7. **B-P1-7** — WS read limit + ping/pong
8. **B-P1-8** — Room empty grace period
9. **B-P2-7** — Graceful shutdown

### Sesi 3: Frontend Kritis (30-60 menit)
10. **F-P0-1** — Reactive roomId
11. **F-P0-2 + F-P0-4** — onUnmounted cleanup
12. **F-P0-3** — createRoom error feedback
13. **F-P0-5** — CSS rgba fix

### Sesi 4: Frontend UX (1-2 jam)
14. **F-P1-1** — Mobile hamburger menu
15. **F-P1-3** — Lobby error state
16. **F-P1-4** — Clipboard error handler
17. **F-P2-1** — Chat message limit
18. **F-P2-4** — Update docs WS API

### Sesi 5: Cleanup (30 menit)
19. **F-P3-1** — Delete HelloWorld.vue
20. **F-P3-2** — Delete empty router/
21. **B-P3-1-4** — Remove dead code
22. **F-P3-4** — Extract shared validation

---

## 📄 File Laporan Terkait

| File | Status |
|------|--------|
| `PROGRESS_ANALYSIS_REPORT.md` | Final audit (54 issues) — sebagian sudah di-fix |
| `BACKEND_BUGFIX_TASKS.md` | 7/7 task selesai |
| `BACKEND_SCRAPER_DEVELOPMENT_TASKS.md` | Anoboy wired + headless improved |
| `FEATURES_DOCUMENTATION.md` | Source material untuk /docs page |
| `HEADLESS_ABORT_ON_MATCH_REPORT.md` | Abort-on-match selesai |
| `SAMEHADAKU_SCRAPER_REPORT.md` | AJAX scraper selesai |
| **`NEXT_DAY_WORK_REPORT.md`** | ← Ini (62 issue untuk esok hari) |

---

## 🎯 Target Esok Hari

| Target | Metrik |
|--------|--------|
| Fix 5 backend kritis | 0 panic, 0 auth bypass, 0 chat spoof |
| Fix 5 frontend kritis | Room navigasi lancar, tidak ada memory leak |
| Backend reliability | WS heartbeat, async scrape, graceful shutdown |
| Mobile UX | Hamburger menu, error states, responsive |

---

*Laporan ini berdasarkan deep scan kode per 17 Juli 2026 malam. Semua line reference akurat pada commit terbaru.*
