# Local Development & Feature Tasks — WatchParty Platform

**Status:** P1 ✅ (1.5 ⏭) · P2 ✅ · P3 ✅ (3.3/3.5 ⏭) · P4 pending  
**Tanggal:** 19 Juli 2026 · **Update:** double URL fix + kick flash banner  

**Tujuan:** Sempurnakan fitur di local sebelum deploy  
**Deploy target (nanti):** Frontend Vercel · Backend lokal (Cloudflare Tunnel) · Domain `justwatchit.my.id`

---

## Arsitektur Deploy (Rencana)

```
justwatchit.my.id (Vercel, always on)
  └── Vue SPA (dist/)
  └── VITE_API_URL = https://api.justwatchit.my.id
  └── VITE_WS_URL  = wss://api.justwatchit.my.id

api.justwatchit.my.id (Cloudflare Tunnel → laptop lokal)
  └── Go backend (:8080)
  └── Headless Chrome (Docker)
  └── NanoProxy (Docker)
```

### DNS Setup

| Record | Type | Target |
|--------|------|--------|
| `justwatchit.my.id` | CNAME | Vercel |
| `api.justwatchit.my.id` | CNAME | Cloudflare Tunnel |

### Cloudflare Tunnel Setup (backend lokal)

```bash
# Install
curl -L https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64 -o cloudflared
chmod +x cloudflared && sudo mv cloudflared /usr/local/bin/

# Login
cloudflared tunnel login

# Buat tunnel
cloudflared tunnel create justwatchit-api

# Route DNS
cloudflared tunnel route dns justwatchit-api api.justwatchit.my.id

# Jalankan (saat laptop nyala + backend running)
cloudflared tunnel run --url http://localhost:8080 justwatchit-api
```

**Catatan:** Backend hanya online saat laptop nyala. FE di Vercel tetap bisa dibuka, tapi fitur room/WS mati kalau BE off.

---

## Headless Chrome — Ringkasan

| Site | Tanpa Chrome | Dengan Chrome |
|------|--------------|---------------|
| otakudesu, anoboy, samehadaku | ✅ OK (HTTP) | ✅ |
| YouTube / direct .mp4/.m3u8 | ✅ Bypass scraper | ✅ |
| idlix | ❌ Mati | ✅ |
| animasu / kuronime / nanime / unknown | ⚠️ Partial | ✅ Full |

**Keputusan sementara:** Keep Chrome (Docker) untuk production path.

---

## Prioritas 1 — Core UX (wajib)

### 1.1 Reconnect Indicator — ✅ DONE
- **File:** `src/components/ConnectionStatus.vue`, `src/pages/RoomPage.vue`
- **Masalah:** WS putus → UI freeze tanpa info
- **Fix:** Banner "Connection lost, reconnecting..." + spinner saat status `reconnecting` / `disconnected` / `connecting`; style offline terpisah
- **Acceptance:** Disconnect WiFi → banner muncul → reconnect → banner hilang
- **Implementasi:** `showReconnectBanner` computed; banner fixed bottom + spinner

### 1.2 Chat Auto-Scroll — ✅ DONE
- **File:** `src/components/ChatPanel.vue`
- **Masalah:** Chat baru muncul tapi viewport tidak scroll ke bawah
- **Fix:** Cek `isNearBottom` **sebelum** DOM grow, lalu `nextTick` scroll. Own send selalu scroll bottom
- **Acceptance:** Kirim chat → auto scroll. Scroll up → tidak dipaksa ke bawah

### 1.3 Video Player Error Handling — ✅ DONE
- **File:** `src/components/VideoPlayer.vue`
- **Masalah:** URL invalid → blank tanpa feedback
- **Fix:** Overlay error untuk playback + invalid URL input; CTA "Try another URL" (host); HLS/native error messages
- **Acceptance:** Paste URL invalid → pesan error jelas

### 1.4 Mobile Responsive Check — ✅ DONE
- **File:** `RoomPage.vue`, `RoomHeader.vue`, `VideoPlayer.vue`, `style.css`, pages (hamburger)
- **Masalah:** Belum verifikasi menyeluruh di mobile
- **Fix:** Banner/toast full-width di ≤480px; header ellipsis; `max-width: 100vw`; sidebar height 375px; player controls stack
- **Acceptance:** Usable di mobile, tidak horizontal scroll (smoke CSS; visual QA manual di browser)

### 1.5 Nickname Persistence — ⏭ SKIPPED (tunggu keputusan DB)
- **File:** `src/pages/RoomPage.vue`, `src/components/NicknameModal.vue`
- **Masalah:** Nickname diminta ulang tiap join room
- **Alasan skip:** User ingin store via DB (belum dipilih). Jangan `localStorage` dulu.
- **Nanti:** Prefill dari backend/session setelah DB dipilih

---

## Prioritas 2 — Missing Features — ✅ DONE

### 2.1 User List Panel — ✅
- **File:** `src/components/UserListPanel.vue` (baru), `RoomHeader.vue`, `RoomPage.vue`
- Klik participant count → list + badge host + aksi host

### 2.2 Host Transfer — ✅
- **BE:** `TRANSFER_HOST` → `HOST_CHANGED`
- **FE:** tombol "Make host" di user list

### 2.3 Kick User — ✅
- **BE:** `KICK_USER` → `KICKED` + `USER_LEFT`
- **FE:** On `KICKED` → `sessionStorage wp_flash` + leave room → home banner  
  *"You have been removed from the room by the host."*

### 2.4 Room Name Editing — ✅
- **BE:** `SET_ROOM_NAME` → `ROOM_NAME_CHANGED`; `ROOM_INIT.roomName`
- **FE:** inline edit di RoomHeader (host)

### 2.5 Empty State Improvements — ✅
- **File:** `VideoPlayer.vue`, `QueuePanel.vue` — onboarding copy lebih jelas

### Scrape loading (IDLIX) — ✅ ekstra
- **BE:** `SCRAPE_STARTED` / `SCRAPE_FINISHED` di processSetVideo & processAddToQueue
- **FE:** banner "Extracting stream… this may take up to a minute…"

---

## Prioritas 3 — Nice to Have

### 3.1 Emoji Reactions — ✅
- **BE:** `REACTION` broadcast + rate limit
- **FE:** `EmojiReactions.vue` picker + float animation

### 3.2 Typing Indicator — ✅
- **BE:** `TYPING` (exclude sender, ~1/2s)
- **FE:** ChatPanel "X is typing…" + debounce

### 3.3 Open Graph Meta Tags — ⏭ SKIPPED
- Dynamic meta butuh SSR/prerender — lewati dulu

### 3.4 Dark/Light Mode Toggle — ✅
- **FE:** `ThemeToggle.vue`, `localStorage wp_theme`, light palette di `style.css`

### 3.5 Volume Sync — ⏭ SKIPPED
- Kurang menarik / preferensi lokal

---

## Prioritas 4 — Developer Experience

### 4.1 `.env.example`
```env
# Backend
LISTEN_ADDR=:8080
CORS_ORIGINS=https://justwatchit.my.id
PROXY_PUBLIC_BASE=https://api.justwatchit.my.id
PROXY_ALLOWLIST_EXTRA=
NANOPROXY_HTTP_URL=
CHROME_PATH=

# Frontend (Vite)
VITE_API_URL=https://api.justwatchit.my.id
VITE_WS_URL=wss://api.justwatchit.my.id
```

### 4.2 README Update
- Deskripsi project
- Local setup (frontend + backend + optional Docker chrome/nanoproxy)
- Env vars reference
- Arsitektur singkat
- Deploy notes (Vercel + Cloudflare Tunnel)

### 4.3 ESLint + Prettier
- `eslint.config.js` (Vue)
- `.prettierrc`
- Scripts: `lint`, `format` di `package.json`

### 4.4 Health Check Endpoint
- `GET /health` → `{ "status": "ok", "uptime": "...", "rooms": N }`
- File: `backend/main.go`

### 4.5 Structured Logging
- Ganti `log.Println` → `slog` (JSON / levelled)
- Panic recover di `safeSend` tetap + log terstruktur

---

## Deploy Prep (setelah fitur local stabil)

| # | File | Apa |
|---|------|-----|
| D1 | `vercel.json` | SPA rewrites |
| D2 | `src/config.js` | Centralized `API_BASE` + `WS_BASE` dari env |
| D3 | Update `fetch()` + `useWebSocket.js` | Pakai config (bukan relative-only) |
| D4 | `backend/Dockerfile` | Multi-stage Go (+ Chrome path docs) |
| D5 | `docker-compose.prod.yml` | backend + chrome + nanoproxy |
| D6 | `Caddyfile` (opsional jika VPS) | Reverse proxy TLS |
| D7 | `.github/workflows/deploy.yml` | **Manual trigger** (`workflow_dispatch`) |

### Env frontend production (Vercel)
```
VITE_API_URL=https://api.justwatchit.my.id
VITE_WS_URL=wss://api.justwatchit.my.id
```

### Env backend (local / tunnel)
```
CORS_ORIGINS=https://justwatchit.my.id
PROXY_PUBLIC_BASE=https://api.justwatchit.my.id
```

---

## File Changes Summary (estimasi)

### New (kemungkinan)
```
src/config.js
src/components/UserListPanel.vue
backend/Dockerfile
backend/.dockerignore
.env.example
.golangci.yml
eslint.config.js
.prettierrc
vercel.json
docker-compose.prod.yml
.github/workflows/deploy.yml
```

### Modified (kemungkinan)
```
src/composables/useWebSocket.js
src/composables/useRoom.js
src/pages/RoomPage.vue
src/pages/HomePage.vue
src/pages/LobbyPage.vue
src/components/ChatPanel.vue
src/components/VideoPlayer.vue
src/components/RoomHeader.vue
src/components/NicknameModal.vue
src/components/ConnectionStatus.vue
backend/main.go
backend/services/message_handler.go
backend/models/types.go
README.md
```

---

## Urutan Pengerjaan

### Sesi 1 — Core UX (2–3 jam) — ✅ 1.1–1.4 done · 1.5 skipped
1. ~~1.1 Reconnect indicator~~ ✅  
2. ~~1.2 Chat auto-scroll~~ ✅  
3. ~~1.3 Video error handling~~ ✅  
4. ~~1.4 Mobile check~~ ✅  
5. 1.5 Nickname persistence — ⏭ skip (tunggu DB)

### Sesi 2 — Missing Features (3–4 jam) — ✅ DONE
6. ~~2.1 User list panel~~ ✅  
7. ~~2.2 Host transfer~~ ✅  
8. ~~2.3 Kick user~~ ✅  
9. ~~2.4 Room name editing~~ ✅  
10. ~~2.5 Empty state~~ ✅  
10b. ~~Scrape loading banner (IDLIX)~~ ✅

### Sesi 3 — Dev Experience (1–2 jam)
11. 4.1 `.env.example`  
12. 4.2 README  
13. 4.3 ESLint + Prettier  
14. 4.4 Health check  
15. 4.5 Structured logging  

### Sesi 4 — Deploy Prep (1–2 jam)
16. D1–D3 Frontend Vercel + config URL  
17. D4–D5 Docker backend  
18. Cloudflare Tunnel  
19. D7 GitHub Actions manual deploy  

---

## Acceptance Gate (siap “soft public”)

- [x] Reconnect UX jelas  
- [x] Chat & player error tidak blank  
- [ ] Nickname tersimpan *(skip — tunggu DB)*  
- [x] Host: list users, transfer, kick, room name  
- [x] Scrape loading notif (IDLIX)  
- [x] Emoji reactions + typing + theme toggle  
- [x] `go test ./...` pass — **46 passed**  
- [x] `npm run build` pass  
- [ ] `/health` OK  
- [ ] FE Vercel + BE tunnel (optional) smoke test  

### Changelog P1–P3 (19 Juli 2026)

| Area | Status | Notes |
|------|--------|-------|
| P1 1.1–1.4 | ✅ | Reconnect, chat scroll, video error, mobile |
| P1 1.5 | ⏭ | Nickname — tunggu DB |
| P2 2.1–2.5 | ✅ | User list, transfer, kick, room name, empty state |
| Scrape UX | ✅ | `SCRAPE_STARTED`/`FINISHED` + FE banner |
| P3 3.1, 3.2, 3.4 | ✅ | Reactions, typing, theme |
| P3 3.3, 3.5 | ⏭ | OG tags, volume sync |
| Next | P4 | `.env.example`, README, health, slog, lint |
| UX fix | ✅ | Single host URL input (RoomPage only); kick → home flash banner |
| Docs page | ✅ | WS events: transfer/kick/room name/scrape/reaction/typing |

---

## File Terkait

| File | Isi |
|------|-----|
| `NEXT_DAY_WORK_REPORT.md` | Bugfix batch sebelumnya (62 issue) |
| `PRODUCTION_RESIDUAL_TASKS.md` | Residual reconnect / room switch / rate limit |
| **`LOCAL_FEATURE_TASKS.md`** | ← Ini (fitur local + rencana deploy) |

---

*Berdasarkan audit kode + gap analysis, Juli 2026. Istirahat dulu — kerjakan kapan siap.*
