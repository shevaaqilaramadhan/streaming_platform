# WatchParty — Feature Documentation Source

**Last updated:** 17 July 2026  
**Purpose:** Source material for the in-app Documentation page (`/docs`), FAQ, and external README.  
**Stack:** Vue 3 + Vite (frontend) · Go + WebSocket (backend) · HLS.js · Chromium/chromedp (scrapers)

---

## 1. Product Overview

WatchParty is a **synchronized watch-together** platform. Users create a room, share a link, and play video in real time with friends — no account required.

| Capability | Description |
|------------|-------------|
| Real-time sync | Host play / pause / seek broadcasts to all guests over WebSocket |
| Multi-source video | Direct MP4/HLS, YouTube, and scraped anime/movie pages |
| Live chat | Ephemeral room chat (not persisted) |
| Play queue | Playlist of upcoming videos with auto-advance |
| Public lobby | Optional public rooms discoverable at `/lobby` |
| Host controls | Server-authoritative host; auto-reassign on host leave |
| Documentation site | In-app `/docs`, `/faq`, and `/status` pages |

---

## 2. User-Facing Features (Frontend)

### 2.1 Home Page (`/`)

- **Create Watch Room** — `POST /api/rooms` → navigates to `/room/:roomId?host=true`
- **Join by code** — enter short room ID, go to `/room/:roomId`
- Marketing hero, feature cards, footer links to Docs / FAQ / Status
- Design system: dark glass UI, cyan brand gradient
- Sticky top nav with `router-link` navigation to `/docs`, `/faq`, `/status`

### 2.2 Public Lobby (`/lobby`)

- Lists rooms marked **Public** via `GET /api/public-rooms`
- Cards show: title / metadata thumbnail, host name, viewer count, queue size
- Auto-refresh every 15 seconds
- Empty state with CTA to create a room

### 2.3 Watch Room (`/room/:roomId`)

| Area | Behavior |
|------|----------|
| Nickname modal | Guests pick a nickname (2–24 chars) before joining |
| Host entry | `?host=true` skips modal (nickname defaults to "Host"); **server still decides** who is host |
| Video player | Custom controls; host-only play/pause/seek; guests follow sync |
| Now Watching | Title, episode, thumbnail from scrape metadata |
| Chat tab | Live messages, system join/leave, own-message styling |
| Queue tab | Add URL, remove/skip/clear (host), auto-advance on end |
| Public toggle | Host can make room appear in lobby |
| Share link | Copies invite URL **without** `?host=true` |
| Connection status | Live / Connecting / Reconnecting / Offline |
| Disconnect banner | Shown when WebSocket drops; auto-reconnect with backoff |

### 2.4 Video Sources Supported in Player

| Mode | Detection | Playback |
|------|-----------|----------|
| **YouTube** | youtube.com / youtu.be / 11-char ID | YouTube IFrame API (custom chrome, no native YT controls) |
| **HLS** | URL ends with `.m3u8` | hls.js via `/api/proxy` (CORS + referer fix) |
| **Native** | `.mp4`, `.webm`, other direct URLs | HTML5 `<video>` (also proxied when needed) |
| **Scraped page** | Anime/movie page URL (not a direct file) | Backend resolves → stream URL → same as HLS/native |

### 2.5 Documentation Site Pages

All pages share the dark glassmorphism design system, animated constellation background, and a sticky top navigation bar with active-link indicator and `router-link` SPA navigation.

| Route | Page | Status |
|-------|------|--------|
| `/docs` | DocumentationPage | Done |
| `/faq` | FAQPage | Done |
| `/status` | ServerStatusPage | Done |

#### DocumentationPage (`/docs`)

Full developer & user guide with sidebar navigation:

| Section | Content |
|---------|---------|
| **Sidebar TOC** | Sticky table-of-contents with scroll-spy (highlights active section on scroll) |
| **Quick Start** | 3-step cards: Create Room → Share Link → Watch Together, with code examples and format badges |
| **Room Controls** | Permission matrix table — Host vs Guest for: Play/Pause, Seek, Volume, Load URL, Manage Queue, Chat, Kick User |
| **Supported Video Sources** | 3 source cards (MP4, HLS, YouTube) with icons, descriptions, and example URLs |
| **Chat & Interaction** | Feature checklist: instant messaging, nickname system, participant count |
| **Queue System** | Visual queue demo with "Now Playing" indicator, numbered items, durations, and auto-advance note |
| **Keyboard Shortcuts** | 2-column grid: Space (play/pause), F (fullscreen), M (mute), ←/→ (seek), Enter (send chat) |
| **WebSocket API** | Endpoint `WS /ws/room/{roomId}` with event list: play, pause, seek, chat, queue_update |
| **Footer nav** | CTA links to Home and FAQ page |

#### FAQPage (`/faq`)

Searchable, categorized FAQ with accordion UI:

| Feature | Detail |
|---------|--------|
| **Search bar** | Real-time text filter across all questions and answers |
| **Category tabs** | General, Rooms, Playback, Technical, Troubleshooting — plus "All" view |
| **Accordion items** | Click to expand/collapse with smooth CSS animation |
| **Q&A count** | 20+ entries across 5 categories |
| **Topics covered** | Account-free usage, supported formats, sync mechanics, host controls, queue, self-hosting, API integration, privacy, troubleshooting playback/chat/desync/room-not-found |
| **Empty state** | "No results" message with clear-search button |
| **CTA** | "Still have questions?" card linking to GitHub Issues |

#### ServerStatusPage (`/status`)

Real-time service health dashboard:

| Feature | Detail |
|---------|--------|
| **Overall status badge** | Animated pulse dot — green (Operational), amber (Degraded), red (Down) |
| **Summary cards** (4) | Services Online (count), Active Rooms, Avg Response Time, Overall Uptime % |
| **Service health cards** (5) | Web Application, WebSocket Server, API Server, Video Proxy, CDN / Static Assets |
| **Per-service metrics** | Status badge, uptime %, response time (ms), last incident |
| **90-day uptime bar** | Color-coded segment visualization per day (green = up, amber = degraded, red = down) |
| **Recent incidents** | Severity-tagged cards (minor/major) with date, description, and duration |
| **Response time chart** | 24-hour CSS-only bar chart with hover tooltips showing ms values |
| **Subscribe button** | Toggle notification opt-in for downtime alerts |

---

## 3. Backend Features (API & Realtime)

### 3.1 REST Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| `POST` | `/api/rooms` | Create room → `{ "roomId": "abc123" }` |
| `GET` | `/api/public-rooms` | List public rooms (metadata, viewers, queue size) |
| `GET` | `/api/proxy?url=` | Stream proxy for HLS/MP4 (CORS, referer, M3U8 rewrite) |

### 3.2 WebSocket (`/ws/:roomId`)

- Auto-creates room if missing (join by link without prior POST)
- Typed actions (see WebSocket contract below)
- Auto-reconnect on client (exponential backoff, max 5 attempts)
- Online/offline browser events trigger reconnect

### 3.3 WebSocket Message Types

**Client → Server**

| Action | Who | Purpose |
|--------|-----|---------|
| `JOIN_EVENT` | All | Register nickname; server assigns host |
| `SYNC_EVENT` | Host | Play/pause/seek state |
| `CHAT_EVENT` | All | Chat text |
| `SET_VIDEO` | Host* | Load video URL (may scrape) |
| `ADD_TO_QUEUE` | All* | Queue a URL |
| `REMOVE_FROM_QUEUE` | Host | Remove queue item |
| `SKIP_TO_NEXT` | Host | Play next in queue |
| `CLEAR_QUEUE` | Host | Empty queue |
| `EPISODE_ENDED` | Host only | Auto-advance (debounced 3s) |
| `TOGGLE_PUBLIC` | Host | Lobby visibility |

\*Some host checks are product-configurable; queue skip/remove/clear require host.

**Server → Client**

| Action | Purpose |
|--------|---------|
| `ROOM_INIT` | Full room state + `isHost`, `hostId`, queue, metadata |
| `JOIN_EVENT` / `USER_LEFT` | Presence |
| `HOST_CHANGED` | New host after previous host disconnects |
| `SYNC_EVENT` | Guest playback follow (+ `serverAt` for latency) |
| `SET_VIDEO` / `CURRENT_VIDEO_CHANGED` | Video + metadata update |
| `QUEUE_UPDATE` | Full queue snapshot |
| `SCRAPE_ERROR` | Scrape failed (toast on frontend) |
| `CHAT_EVENT` | Chat broadcast |

### 3.4 Server-Authoritative Host

- First joiner becomes host (client `isHost` claim is **ignored**)
- On host disconnect: next client promoted; `HOST_CHANGED` broadcast
- Frontend updates host UI reactively (controls unlock for new host)

---

## 4. Scraping & Stream Resolution (Major New Capability)

Users paste a **page URL** (not only a direct video link). Backend resolves a playable stream.

### 4.1 Supported Platforms (Router)

The backend scraper router (`detectSitePlatform`) recognizes these domains:

**Dedicated scrapers (optimized, fast):**

| Platform | Detection | Method | Typical Latency |
|----------|-----------|--------|-----------------|
| **Samehadaku** | domain contains `samehadaku` | WordPress AJAX player API (`player_ajax`) | ~1–3s |
| **Otakudesu** | domain contains `otakudesu` | WordPress AJAX mirrors + Packer unpack | ~2–4s |
| **Anoboy** | domain contains `anoboy` | Iframe + base64 mirrors + Blogger extract | ~2–5s |
| **IDLIX** | domain contains `idlix` | Official API gate/claim chain (see below) | ~15–25s (first load) |

**Domain-detected (routed to generic multi-layer scraper):**

| Platform | Detection | Method |
|----------|-----------|--------|
| **Animasu** | domain contains `animasu` | Generic: HTML → scripts → base64 mirrors → iframes → headless |
| **Kuronime** | domain contains `kuronime` | Generic: HTML → scripts → base64 mirrors → iframes → headless |
| **Nanime** | domain contains `nanime` | Generic: HTML → scripts → base64 mirrors → iframes → headless |

**Generic fallback (any URL not matching above):**

| Type | Method |
|------|--------|
| Any anime/movie page | 4-layer static scrape → NanoScraper → Headless Chromium |
| Direct `.mp4` / `.m3u8` | No scrape — played directly via proxy |
| YouTube URL | No scrape — YouTube IFrame API |

### 4.2 IDLIX Stream Extraction (API Chain)

IDLIX does **not** use play-button DOM scraping as the primary path. Flow matches [IDLIX-API](https://github.com/annurdien/IDLIX-API):

1. Resolve content UUID (`/api/movies/{slug}` or series season API)
2. Track view (optional)
3. `play-info` → `gateToken` + unlock timer (~15s)
4. Wait for gate
5. `session/claim` → `claim` + `redeemUrl` (majorplay)
6. Redeem on majorplay (`Content-Type: text/plain`) → `.m3u8` + subtitles metadata

CF-protected steps use **in-page Chromium fetch** (TLS fingerprint). Redeem uses normal HTTP.

**User impact:** Paste e.g. `https://z2.idlixku.com/movie/some-title-2026` → wait ~15–25s first time → video loads via proxy.

### 4.3 Stealth Headless Browser (SPA / JS-heavy sites)

When static scrape fails:

- Full headless Chromium (`chromedp`)
- VPS flags: `--no-sandbox`, `--disable-gpu`, `--disable-dev-shm-usage`
- Spoof `navigator.webdriver`
- Block ads/images/fonts/analytics (doubleclick, adsterra, GTM, popads, …) for speed
- Network interception for `.m3u8` / `.mp4` / embed URLs
- **Abort-on-match:** as soon as a high-confidence stream is seen (`.m3u8`, Krakenfiles, Acefile, Hxfile, …), cancel the Chromium context and treat the scrape as **success** — even if the error is `context.Canceled` / deadline. Prevents Samehadaku-style ad iframes from holding Chrome until timeout
- Early success if streams captured before timeout
- IDLIX SVG play overlay parent-click as secondary trigger for some players

Optional: route browser/HTTP through **NanoProxy** (`NANOPROXY_HTTP_URL`).

See also: `HEADLESS_ABORT_ON_MATCH_REPORT.md`.

### 4.4 Scrape Cache

- In-memory TTL cache (~30 minutes) per page URL
- Periodic + lazy cleanup to avoid unbounded growth

---

## 5. Stream Proxy (`/api/proxy`)

### Why it exists

Anime/movie CDNs often block browser CORS or require a correct `Referer`. Frontend never loads CDN URLs directly for HLS — it uses:

```
/api/proxy?url=<encoded-stream-url>
```

### Behaviors

| Feature | Detail |
|---------|--------|
| Referer / User-Agent injection | Mimics real browser to CDN |
| M3U8 rewrite | Segment + `URI=` tag URLs rewritten through proxy |
| Range support | Seeking works for progressive MP4 |
| Body size limits | Caps m3u8 (2MB) and scrape HTML (5MB) against OOM |
| SSRF protection | Blocks localhost, private IPs, cloud metadata |

### Dynamic Stream Domain Allowlist (New)

IDLIX rotates media hosts (`g5.ruangskill.space`, `g5.akademivo.website`, etc.). Static lists alone fail.

**How it works:**

1. On successful scrape, backend registers hosts from: redeem URL, stream URL, poster, subtitles  
2. When proxy rewrites an M3U8, hosts in the playlist are also registered  
3. Proxy allows: **static allowlist** OR **dynamic cache** (host + parent root domain, TTL ~2h)  
4. Private IPs remain **always blocked** even if registered  

**Ops escape hatch:**

```bash
PROXY_ALLOWLIST_EXTRA=some-cdn.com,other.space
```

---

## 6. Queue System

- Add page or direct URLs to queue  
- Host: remove item, skip to next, clear all  
- On video `ended` (host): `EPISODE_ENDED` → next item (host-only + 3s debounce)  
- Queue items store resolved stream URL + title/episode/thumbnail when scrape succeeds  
- Direct `.mp4` / `.m3u8` / YouTube skip scrape  

---

## 7. Security & Reliability Features (Recent)

Document these under "How we keep rooms safe / stable":

| Feature | User-visible effect |
|---------|---------------------|
| Server-side host | Guests cannot steal host by faking `isHost` |
| Host reassignment | Party continues if host closes the tab |
| EPISODE_ENDED auth | Guests cannot force-skip the playlist |
| SSRF blocks | Proxy cannot be abused to hit internal servers |
| Dynamic CDN allow | Rotating stream domains work after scrape |
| WebSocket Send close | Fewer memory leaks under long uptime |
| Soft scrape success | Partial loads still play if m3u8 already captured |

---

## 8. Environment Variables (Ops / Docs)

| Variable | Default / example | Purpose |
|----------|-------------------|---------|
| `NANOPROXY_HTTP_URL` | `http://localhost:8180` | Route scraper/proxy HTTP through NanoProxy |
| `CHROME_PATH` | auto-detect | Chromium binary for headless |
| `PROXY_ALLOWLIST_EXTRA` | empty | Comma-separated extra proxy hosts |
| `IDLIX_BASE_URL` | (in scraper) `https://z2.idlixku.com` | If exposed later for config |
| `VITE_WS_URL` | same origin | Override WebSocket base (optional) |

Dev: Vite proxies `/api` and `/ws` to Go `:8080`.

---

## 9. Implemented Documentation Page Structure

The `/docs` page (DocumentationPage) is implemented with the following sections:

1. **Quick Start** — Create room, share link, watch together (3-step cards)  
2. **Room Controls** — Host vs Guest permission table  
3. **Supported Video Sources** — MP4, HLS, YouTube source cards  
4. **Chat & Interaction** — Messaging, nicknames, participant count  
5. **Queue System** — Visual queue demo with auto-advance  
6. **Keyboard Shortcuts** — Playback shortcut grid  
7. **WebSocket API** — Endpoint + event reference  

The `/faq` page covers 5 categories: General, Rooms, Playback, Technical, Troubleshooting (20+ Q&A).

The `/status` page shows: service health cards, uptime bars, incident history, response time chart, subscribe button.

---

## 10. Feature Matrix (Quick Reference)

| Feature | Status |
|---------|--------|
| Room create / join | Done |
| WebSocket sync | Done |
| Live chat | Done |
| YouTube player | Done |
| HLS + stream proxy | Done |
| Play queue + auto-advance | Done |
| Video metadata UI | Done |
| Public lobby | Done |
| Host reassignment | Done |
| Otakudesu scraper | Done |
| Anoboy scraper | Done |
| Samehadaku scraper (AJAX) | Done |
| Animasu scraper (generic) | Done |
| Kuronime scraper (generic) | Done |
| Nanime scraper (generic) | Done |
| IDLIX API stream chain | Done |
| Stealth headless SPA fallback | Done |
| Dynamic CDN allowlist | Done |
| SSRF-hardened proxy | Done |
| Documentation page (`/docs`) | Done |
| FAQ page (`/faq`) | Done |
| Server Status page (`/status`) | Done |
| Accounts / auth | Not planned (prototype) |
| Persistent chat history | Not planned (prototype) |
| Subtitle track UI | Partial (data may exist; UI TBD) |

---

## 11. Copy Snippets for Docs UI

**Hero line**  
> Watch video together, in sync — no account needed.

**Create room**  
> Create a room, share the link, and everyone stays on the same second.

**Paste any link**  
> Drop a YouTube link, a direct `.m3u8`/`.mp4`, or an anime/movie page URL. We'll resolve the stream for the whole room.

**IDLIX note**  
> IDLIX titles use a secure unlock step. The first load can take about 15–25 seconds; after that, playback uses our stream proxy for smooth HLS.

**Host tip**  
> Only the host controls play, pause, and seek. If the host leaves, the next person in the room becomes host automatically.

**Public room**  
> Toggle Public in the room header to list your party on the Lobby so others can join without a code.

**Docs page**  
> Full documentation with quick start guide, room controls, video sources, keyboard shortcuts, and WebSocket API reference.

**FAQ page**  
> Searchable FAQ with 20+ answers across General, Rooms, Playback, Technical, and Troubleshooting categories.

**Status page**  
> Real-time health monitoring of all WatchParty services with uptime history, response time charts, and incident tracking.

---

## 12. Related Internal Docs

| File | Content |
|------|---------|
| `WEBSOCKET_CONTRACT.md` | Message schemas |
| `PROGRESS_ANALYSIS_REPORT.md` | Full bug/progress audit |
| `BACKEND_BUGFIX_TASKS.md` | Critical backend fixes |
| `BACKEND_SCRAPER_DEVELOPMENT_TASKS.md` | Scraper roadmap |
| `Proj Streaming Platform.md` | Original prototype spec |

---

*This document is the canonical feature inventory for writing user-facing documentation. Prefer updating this file when shipping user-visible capabilities, then sync `/docs` and `/faq` pages.*
