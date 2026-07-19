# Streaming Platform — Comprehensive Progress & Bug Analysis Report

**Date:** 17 Juli 2026  
**Scope:** Full-stack (Frontend Vue 3 + Backend Go)  
**Commits:** 7 total commits on `main` branch  
**Server Log:** Clean startup, no runtime errors captured  

---

## Executive Summary

| Category | Total Issues | Critical | Medium | Low |
|----------|:---:|:---:|:---:|:---:|
| **Backend** | 44 | 9 | 21 | 14 |
| **Frontend** | 10 | 0 | 4 | 6 |
| **Total** | **54** | **9** | **25** | **20** |

### Project Completion Status

| Area | Status | Notes |
|------|--------|-------|
| Room creation & joining | Done | Host/guest flow works |
| Video sync (play/pause/seek) | Done | YouTube, HLS, native MP4 |
| Live chat | Done | Real-time via WebSocket |
| Video queue system | Done | Add/remove/skip/clear |
| Public rooms / Lobby | Done | Route, page, toggle wired |
| Video metadata display | Done | Title, episode, thumbnail |
| Stream proxy | Done | `/api/proxy` with M3U8 rewrite |
| Scraper (Otakudesu) | Done | Dean Edwards Packer unpacking |
| Scraper (anoboy) | Not wired | Function exists but never called |
| Scraper (SPA/headless) | Pending | chromedp integration documented |
| Graceful shutdown | Missing | No signal handling |
| Production hardening | Missing | CORS, SSRF, rate limits |

---

## Backend Issues (Go)

### CRITICAL (9)

#### BE-C01: WebSocket `writePump` Goroutine Leak
- **File:** `handlers/websocket.go:66-104`
- **Problem:** `readPump` closes the connection and removes the user, but never closes `user.Send` channel. `writePump` blocks forever on `for message := range user.Send`.
- **Impact:** One leaked goroutine + channel buffer per disconnect. Memory grows under normal usage.
- **Fix:** Close `user.Send` once in `readPump` defer using `sync.Once`.

#### BE-C02: Deadlock — Lock Order Inversion
- **File:** `services/room_manager.go:83-103` vs `129-154`
- **Problem:** Cleanup path takes `mu.Lock()` then `room.Mutex.RLock()` (global→room). Remove path takes `room.Mutex.Lock()` then `mu.Lock()` (room→global).
- **Impact:** Concurrent empty-room leave + cleanup ticker can deadlock the entire server.
- **Fix:** Enforce consistent lock order (always `mu` then `room.Mutex`).

#### BE-C03: Host Privilege is Client-Claimed; Host Never Reassigned
- **Files:** `services/message_handler.go:70-78`, `services/room_manager.go:129-154`
- **Problem:** (1) `user.IsHost = payload.IsHost` trusts the client — any user can claim host. (2) `RemoveUserFromRoom` never clears `HostID` when host leaves. (3) New host only assigned when `room.HostID == ""`, so after real host leaves, no one can ever become host.
- **Impact:** Privilege escalation; permanent loss of host controls.
- **Fix:** Server decides host (first joiner = host). On host leave, promote next client and broadcast `HOST_CHANGED`. Never trust client `isHost`.

#### BE-C04: Open SSRF Proxy
- **File:** `handlers/proxy.go:47-90`
- **Problem:** `/api/proxy?url=` fetches any HTTP/HTTPS URL with no allowlist, no private-IP block.
- **Impact:** Attackers can hit `127.0.0.1:*`, Docker services, cloud metadata (`169.254.169.254`).
- **Fix:** Block private/link-local/loopback ranges. Optional CDN allowlist.

#### BE-C05: Unbounded Memory via `io.ReadAll`
- **Files:** `handlers/proxy.go:135,147`, `services/scraper.go:1039`
- **Problem:** M3U8 path and HTML fetch load entire response bodies into memory with no limit.
- **Impact:** Attacker-controlled URL → multi-GB response → OOM kill.
- **Fix:** Use `io.LimitReader(resp.Body, maxBytes)` (e.g., 5-10MB).

#### BE-C06: Scrape Cache Grows Without Bound
- **File:** `services/scraper.go:62-98`
- **Problem:** Expired entries are never deleted, only skipped on hit. Map grows forever.
- **Impact:** Long-running process memory leak.
- **Fix:** Periodic cleanup, max entries (LRU), or TTL cache library.

#### BE-C07: `EPISODE_ENDED` Skips Queue With No Auth + Race
- **File:** `services/message_handler.go:316-351`
- **Problem:** Any client can send `EPISODE_ENDED` and advance the queue. Concurrent messages can skip multiple items.
- **Impact:** Guests force-skip content; race corrupts queue playback order.
- **Fix:** Only host can skip. Add debounce/idempotency key per room.

#### BE-C08: Data Race — `GetRoomState` Returns Live Pointers
- **File:** `services/room_manager.go:188-206`
- **Problem:** Lock is released, but callers use shared `Queue` slice and metadata pointers while other goroutines mutate them. Also `handleJoinEvent` reads `room.IsPublic` without lock.
- **Impact:** Race detector failures; possible panics/corrupt JSON.
- **Fix:** Copy queue slice and metadata under the lock before return.

#### BE-C09: HTTP Write-After-Write on Encode Error
- **Files:** `main.go:48-56`, `handlers/public_rooms.go:18-66`
- **Problem:** `Content-Type` set, body partially written by `Encode`, then `http.Error` tries another status/body.
- **Impact:** Corrupt HTTP responses.
- **Fix:** Encode to buffer first, then write status + body once.

---

### MEDIUM (21)

| ID | Issue | File | Description |
|----|-------|------|-------------|
| BE-M01 | No WS read/write deadlines or ping/pong | `handlers/websocket.go` | Dead TCP connections leave goroutines until OS timeout |
| BE-M02 | Dual `Conn.Close` without coordinated shutdown | `handlers/websocket.go:68,96` | Both pumps close the connection; `Send` lifecycle undefined |
| BE-M03 | Full send buffer → silent message drop | `services/message_handler.go:401-413` | Clients miss sync/chat/queue events with no recovery |
| BE-M04 | M3U8 rewrite misses URI attributes in tags | `handlers/proxy.go:162-188` | `#EXT-X-KEY:...URI="..."` stays pointing at CDN → 403 |
| BE-M05 | Proxy copies hop-by-hop headers | `handlers/proxy.go:93-98` | `Transfer-Encoding`, double `Content-Length` issues |
| BE-M06 | CORS is fully open (any Origin) | `main.go:24-31` | Any site can call API/WS from a browser |
| BE-M07 | Chat identity spoofing + no validation | `services/message_handler.go:131-144` | Rebroadcasts client payload as-is; no length limits |
| BE-M08 | No input validation on room ID / URLs / usernames | Various | Arbitrary room auto-creation (DoS); no URL scheme/length checks |
| BE-M09 | `scrapeAnoboy` never wired | `services/scraper.go:237-248,440-516` | `detectSitePlatform` returns `"anoboy"` but switch doesn't handle it |
| BE-M10 | Room deleted immediately when empty | `services/room_manager.go:140-152` | Brief network blip loses queue/state; no reconnect grace |
| BE-M11 | Cleanup uses `CreatedAt`, not last activity | `services/room_manager.go:80-98` | Comment says "no clients for 30 min"; code uses age since creation |
| BE-M12 | TOCTOU: room may vanish mid scrape-then-mutate | `queue_manager.go:20-58` | `GetRoom` → long scrape → `room.Mutex.Lock()` on deleted room |
| BE-M13 | Concurrent `SkipToNext` not atomic | `message_handler.go:261-272` | Pop + set current video in separate lock sections |
| BE-M14 | NanoProxy exposed with `NO_AUTH_MODE: true` | `docker-compose.yml` | Ports 1080/8180 open, no auth |
| BE-M15 | Headless Chrome missing security flags | `services/nano_scraper.go:116-122` | `disable-web-security`; 8s fixed sleep is brittle |
| BE-M16 | `json.Unmarshal` error ignored | `services/scraper.go:362` | Otakudesu mirror quality — error silently ignored |
| BE-M17 | `fetchWithRetry` defined but barely used | `services/scraper.go:125-156` | Main fetch paths don't use retry/UA rotation consistently |
| BE-M18 | Public rooms JSON encodes `null` instead of `[]` | `handlers/public_rooms.go:21,63` | `var publicRooms []*...` → `null` when empty |
| BE-M19 | No graceful shutdown | `main.go:12-21` | `log.Fatal(http.ListenAndServe(...))` — no signal handling |
| BE-M20 | Hardcoded bind address / intervals | `main.go:18-21` | `:8080`, cleanup `10 * time.Minute` hardcoded |
| BE-M21 | `go.mod` marks heavy deps as `// indirect` | `go.mod` | `goquery`, `chromedp` are direct imports but listed indirect |

---

### LOW (14)

| ID | Issue | File | Description |
|----|-------|------|-------------|
| BE-L01 | Dead/unused code: `AddUserToRoom`, `IsStreamURL` | Various | Never called; `AddUserToRoom` creates double `Send` channel |
| BE-L02 | `GenerateShortID` panics on RNG failure | `utils/crypto.go:21` | `panic(err)` can crash process |
| BE-L03 | Room ID entropy only 6 chars (~56 bits) | `utils/crypto.go` | Guessable room IDs for private rooms |
| BE-L04 | No structured logging / request IDs | Various | Harder to debug production issues |
| BE-L05 | No rate limiting | Various | Create room, public rooms, proxy, scrape — all DoS-friendly |
| BE-L06 | No metrics / health endpoint | `main.go` | No `/healthz` for orchestrators |
| BE-L07 | Typo in Chrome flag: `IsolateOrigons` | `nano_scraper.go:120` | Should be `IsolateOrigins` |
| BE-L08 | Emoji in production logs | `main.go:20` | Fine for dev; noisy for log aggregators |
| BE-L09 | Duplicate `mustMarshal` helpers | `handlers/websocket.go`, `services/message_handler.go` | Same function in two files |
| BE-L10 | `server.log` only shows clean start | `server.log` | No runtime errors captured — process not under load |
| BE-L11 | Scraper regex ReDoS surface | `services/scraper.go:35-36` | Broad regex on attacker-controlled HTML |
| BE-L12 | `findChromePath` race on first use | `nano_scraper.go:45-48` | `chromeChecked` without mutex |
| BE-L13 | No HTTPS / reverse-proxy guidance | `main.go` | Dev-only `ListenAndServe` without TLS |
| BE-L14 | SET_VIDEO / queue host checks inconsistent | Various | Anyone can SET_VIDEO/ADD_TO_QUEUE; only some ops require host |

---

## Frontend Issues (Vue 3)

### Previously Documented Bugs — Status

The following bugs from the previous `ANALYSIS_REPORT.md` and `FRONTEND_REMAINING_TASKS.md` have been **FIXED** in the current codebase:

| Bug | Status | Evidence |
|-----|--------|----------|
| BUG-11: `isOwn` chat detection | **FIXED** | `useRoom.js:92-96` uses `localUserId` from `ROOM_INIT` participants |
| Lobby route missing | **FIXED** | `main.js:18-21` has `/lobby` route; `LobbyPage.vue` exists |
| RoomHeader `isPublic`/`toggle-public` not wired | **FIXED** | `RoomPage.vue:19-20` passes `:is-public` and `@toggle-public` |
| CSS `margin-top: -var()` invalid | **FIXED** | `HomePage.vue:673` uses `calc(-1 * var(--space-2))` |
| URL validation missing | **FIXED** | `isValidVideoInput()` exists in both `VideoPlayer.vue:724` and `QueuePanel.vue:154` |
| `initRoom()` race condition | **FIXED** | `RoomPage.vue:186-207` — watchers setup BEFORE connect |
| HLS xhrSetup no-op | **FIXED** | `VideoPlayer.vue:410-415` sets Referer header; proxy URL rewriting at line 393-399 |

### MEDIUM (4)

#### FE-M01: No Error Boundary / Global Error Handler
- **Files:** `App.vue`, `main.js`
- **Problem:** Unhandled promise rejections or component errors will crash silently. No Vue `errorHandler` configured.
- **Fix:** Add `app.config.errorHandler` in `main.js` and an error boundary component.

#### FE-M02: Nav Links Point to Non-Existent Anchors
- **File:** `HomePage.vue:28-30`
- **Problem:** `#docs`, `#faq`, `#status` anchors don't exist on the page. Links do nothing when clicked.
- **Fix:** Either create these sections or remove the links.

#### FE-M03: No Reconnect State Reset on Room Change
- **Files:** `composables/useWebSocket.js`, `pages/RoomPage.vue`
- **Problem:** If user navigates from one room to another without full page reload, the old WebSocket connection's `onMessage` handlers may still be registered. The `onUnmounted` in `useWebSocket` handles cleanup, but `useRoom`'s `unregister` function (line 76) is never called.
- **Fix:** Call `unregister()` in cleanup, or ensure `useRoom` lifecycle matches component lifecycle.

#### FE-M04: Lobby Polling Continues After Navigation
- **File:** `LobbyPage.vue:130-137`
- **Problem:** `setInterval(fetchPublicRooms, 15000)` is cleaned up in `onUnmounted`, but if the component is kept alive by a parent `<KeepAlive>`, polling continues indefinitely.
- **Fix:** Use `onDeactivated` as additional cleanup hook, or use `watchEffect` with stop.

---

### LOW (6)

| ID | Issue | File | Description |
|----|-------|------|-------------|
| FE-L01 | `HelloWorld.vue` is unused dead code | `components/HelloWorld.vue` | Default Vite scaffold component, never imported |
| FE-L02 | Feature card text outdated | `HomePage.vue:400` | Says "Any MP4 link" but platform supports HLS, YouTube, scrapers |
| FE-L03 | No `<meta>` tags / Open Graph | `index.html` | Missing description, og:image, viewport meta for SEO |
| FE-L04 | YouTube player not explicitly destroyed on route change | `VideoPlayer.vue` | `onUnmounted` calls `destroyAllPlayers()` but YT iframe may linger |
| FE-L05 | Chat message count badge never resets | `RoomPage.vue:74` | `messages.length` grows forever; no way to clear history |
| FE-L06 | `ConstellationBackground.vue` performance | `components/ConstellationBackground.vue` | Canvas animation runs on every page; should pause when not visible |

---

## Infrastructure Status

| Component | Status | Notes |
|-----------|--------|-------|
| Docker Compose (main) | Configured | Go backend + Vue frontend |
| Docker Compose (nanoproxy) | Configured | SOCKS5 + HTTP proxy, ports exposed |
| Vite dev proxy | Working | `/api` → localhost:8080, `/ws` → ws://localhost:8080 |
| Server log | Clean | Only startup message, no runtime errors |

### Infrastructure Issues
- NanoProxy exposed with `NO_AUTH_MODE: true` and ports published (security risk)
- No production reverse proxy / TLS termination documented
- No health check endpoints for container orchestration

---

## Improvement Recommendations (by Sector)

### Backend Priority Fixes

| Priority | Fix | Effort | Impact |
|----------|-----|--------|--------|
| 1 | Close `user.Send` on disconnect (BE-C01) | Small | Stops goroutine leak |
| 2 | Fix lock ordering (BE-C02) | Small | Prevents deadlock |
| 3 | Server-side host ownership (BE-C03) | Medium | Prevents privilege escalation |
| 4 | SSRF protections on proxy (BE-C04) | Small | Security critical |
| 5 | Limit response body sizes (BE-C05) | Small | Prevents OOM |
| 6 | Copy room state under lock (BE-C08) | Small | Prevents data races |
| 7 | WS ping/pong + write deadlines (BE-M01) | Medium | Connection reliability |
| 8 | Wire `scrapeAnoboy` (BE-M09) | Small | Unblocks anoboy scraping |
| 9 | Graceful shutdown (BE-M19) | Medium | Production readiness |
| 10 | Origin allowlist + input validation (BE-M06,M08) | Medium | Security hardening |

### Frontend Priority Fixes

| Priority | Fix | Effort | Impact |
|----------|-----|--------|--------|
| 1 | Add global error handler (FE-M01) | Small | Better error visibility |
| 2 | Fix nav links (FE-M02) | Small | UX polish |
| 3 | Clean up `useRoom` unregister (FE-M03) | Small | Prevents memory leaks |
| 4 | Remove dead `HelloWorld.vue` (FE-L01) | Trivial | Code cleanliness |
| 5 | Update feature descriptions (FE-L02) | Trivial | Accurate marketing |
| 6 | Add meta tags (FE-L03) | Small | SEO / social sharing |

---

## Git History (7 commits)

```
549a15b feat: implement video queue system with backend manager and frontend interface
83a1a59 fix: adjust layout dimensions and overflow behavior in RoomPage
b38d69e refactor: update navigation links to include documentation, FAQ, and server status
b2f2b51 refactor: update design system theme from pink to cyan
78392d3 feat: implement video metadata display and sync
fb80f9d feat: implement Otakudesu mirror resolver and Dean Edwards Packer unpacking
87d9e8f feat: implement initial project structure and core real-time streaming
```

**Development velocity:** 7 commits spanning core features. Most recent work focused on queue system and layout fixes.

---

*Report generated on 17 Juli 2026. Based on full codebase scan of all source files, documentation, logs, and git history.*
