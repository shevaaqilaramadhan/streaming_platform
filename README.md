# WatchParty

Synchronized watch-together platform for anime streams and direct video. Create a room, share a link, and play in real time with friends — no accounts required.

**Stack:** Vue 3 + Vite (frontend) · Go + WebSocket (backend) · HLS.js · optional Chromium for headless scrapers

---

## What it does

- **Real-time sync** — host play / pause / seek broadcasts to all guests over WebSocket
- **Multi-source video** — direct MP4/HLS, YouTube, and scraped anime/movie page URLs
- **Live chat** — ephemeral room chat with anti-spoof server overrides
- **Play queue** — playlist with auto-advance on episode end
- **Public lobby** — optional public rooms at `/lobby`
- **Host controls** — server-authoritative host; auto-reassign when host leaves; kick / transfer
- **Stream proxy** — CORS/referer fix + M3U8 rewrite for scrapers

In-app docs: `/docs`, `/faq`, `/status`. More detail under [`docs/`](./docs/).

---

## Prerequisites

| Tool | Notes |
|------|--------|
| **Go** 1.21+ | Backend (`backend/`) |
| **Node.js** 18+ / npm | Frontend |
| **Chrome / Chromium** (optional) | Headless scrapers (Anoboy fallback, etc.). Set `CHROME_PATH` if not on `PATH` |

Optional: [NanoProxy](./docker-compose.yml) for outbound HTTP when scraping behind a proxy.

---

## Quick start

### 1. Backend

```bash
cd backend
go run .
# listens on :8080 by default
```

### 2. Frontend

```bash
# from repo root
npm install
npm run dev
# Vite on http://localhost:5173 — proxies /api and /ws → :8080
```

Open the app, create a room, share the invite link (without `?host=true`).

### Production build (frontend)

```bash
npm run build
npm run preview   # optional local check of dist/
```

Serve `dist/` behind your reverse proxy; point API/WS to the Go process (or same-origin proxy).

---

## Environment variables (backend)

| Variable | Default / notes |
|----------|------------------|
| `LISTEN_ADDR` | `:8080` |
| `CORS_ORIGINS` | Comma-separated origins. **Unset** = allow localhost / 127.0.0.1 any port (dev). **Set in production** (e.g. `https://watchparty.example.com`) |
| `PROXY_PUBLIC_BASE` | Public base URL of the API for absolute M3U8 rewrite (e.g. `https://api.example.com`) |
| `PROXY_ALLOWLIST_EXTRA` | Extra host suffixes for stream proxy allowlist |
| `NANOPROXY_HTTP_URL` | Optional HTTP proxy for scrapers/proxy (e.g. `http://localhost:8180`) |
| `CHROME_PATH` | Path to Chrome/Chromium for headless scrapers |

Example production:

```bash
export CORS_ORIGINS=https://watchparty.example.com
export PROXY_PUBLIC_BASE=https://api.watchparty.example.com
export LISTEN_ADDR=:8080
cd backend && go run .
```

### Optional NanoProxy

```bash
docker compose up -d   # or docker-compose.nanoproxy.yml
export NANOPROXY_HTTP_URL=http://localhost:8180
```

---

## Project layout

```
├── backend/           # Go HTTP + WebSocket server, scrapers, stream proxy
├── src/               # Vue 3 SPA (pages, components, composables)
├── docs/              # Feature notes, contracts, residual tasks
├── vite.config.js     # Dev proxy: /api, /ws → localhost:8080
└── docker-compose.yml # Optional NanoProxy
```

---

## Key routes (frontend)

| Path | Page |
|------|------|
| `/` | Home — create / join room |
| `/lobby` | Public rooms |
| `/room/:roomId` | Watch room |
| `/docs` | Documentation |
| `/faq` | FAQ |
| `/status` | Server status |

---

## API surface (backend)

| Endpoint | Role |
|----------|------|
| `POST /api/rooms` | Create room (+ host token) |
| `GET /api/public-rooms` | List public rooms |
| `GET /api/proxy?url=…` | Stream proxy (SSRF-hardened) |
| `WS /ws/{roomId}` | Room sync, chat, queue, scrape events |

WebSocket message contract: [`docs/WEBSOCKET_CONTRACT.md`](./docs/WEBSOCKET_CONTRACT.md).  
Feature overview: [`docs/FEATURES_DOCUMENTATION.md`](./docs/FEATURES_DOCUMENTATION.md).

---

## Scripts

| Command | Description |
|---------|-------------|
| `npm run dev` | Vite dev server |
| `npm run build` | Production frontend build |
| `npm run preview` | Preview production build |
| `cd backend && go test ./...` | Backend unit tests |
| `cd backend && go run .` | Run API + WebSocket server |

---

## License / notes

Personal / project use. Scrapers depend on third-party sites; respect their ToS and local law. Do not deploy with empty `CORS_ORIGINS` in production.
