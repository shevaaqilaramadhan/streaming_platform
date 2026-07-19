# Samehadaku Scraper Fix — Frontend Interface Report

**Date:** 17 July 2026  
**Status:** Backend implemented & verified live  
**Audience:** Frontend agents updating UI, docs, toasts, and loading UX  
**Related:** `backend/services/samehadaku_scraper.go`, `FEATURES_DOCUMENTATION.md`

---

## 1. Summary (for UI copy)

Samehadaku episode URLs now resolve via a **fast WordPress AJAX path** instead of a slow headless browser. Users should get a playable stream in **~1–3 seconds** (not 40+ seconds), with a real quality label such as **Wibufile 1080p**.

---

## 2. What was broken (user-visible)

| Symptom | Cause |
|---------|-------|
| Long wait, then "Failed to extract stream" | Headless Chrome timed out on ad iframes |
| Log: `Early DOM ready via selector iframe` then `context deadline exceeded` | Ads held Chromium; no stream URL captured |
| Room felt "stuck loading" | Scrape ran ~40s before error toast |

**Example URL that failed before:**  
`https://v2.samehadaku.how/tensei-shitara-slime-datta-ken-season-4-episode-14/`

---

## 3. What works now (backend)

### 3.1 Resolution path

```
User pastes Samehadaku episode URL
        ↓
Backend detects domain "samehadaku"
        ↓
Parse player mirrors (.east_player_option)
  data-post, data-nume, labels (Blogspot, Wibufile 720p/1080p, Mega…)
        ↓
POST /wp-admin/admin-ajax.php
  action=player_ajax&post=…&nume=…&type=schtml
        ↓
<iframe src="https://s0.wibufile.com/…mp4">  (or Blogspot, etc.)
        ↓
Return VideoMetadata { videoUrl, title, episode/quality, thumbnail, source }
        ↓
Frontend SET_VIDEO / queue → play via /api/proxy if needed
```

### 3.2 Preference order (mirrors)

1. Wibufile **1080p** (preferred — direct MP4)  
2. Wibufile **720p** / **480p**  
3. Blogspot / VIP  
4. Mega (lower priority)  
5. Only if AJAX fails: generic HTML scrape → headless last resort  

### 3.3 Live verification

```
[Samehadaku] SUCCESS via "Wibufile 1080p"
video=https://s0.wibufile.com/video01/TSS-S4-14-FULLHD-SAMEHADAKU.CARE.mp4
title=Tensei shitara… Episode 14 Sub Indo
source=samehadaku
```

Typical latency: **~1 second** after page fetch.

---

## 4. What the frontend already receives

No new WebSocket action types are required. Existing flow still applies.

### 4.1 Success — `SET_VIDEO` / `CURRENT_VIDEO_CHANGED`

Payload (conceptual):

```json
{
  "roomId": "…",
  "videoUrl": "https://s0.wibufile.com/video01/….mp4",
  "metadata": {
    "videoUrl": "https://s0.wibufile.com/…",
    "title": "Tensei shitara Slime Datta Ken Season 4 Episode 14 Sub Indo",
    "episode": "Episode 14",
    "thumbnailUrl": "https://…",
    "source": "samehadaku"
  }
}
```

**Frontend should:**

- Set player `videoUrl` (may already be direct `.mp4` — VideoPlayer native/HLS path)  
- Show **Now Watching** with `title` + `episode`  
- Show `source` if UI has a source badge (optional)  
- Clear previous `SCRAPE_ERROR` toast  

**Note:** Direct `.mp4` from Wibufile should go through existing `getProxiedUrl()` / `/api/proxy` so CDN referer/CORS stay reliable. Proxy allowlist includes `wibufile` / `samehadaku`.

### 4.2 Failure — `SCRAPE_ERROR` (unchanged)

```json
{
  "originalUrl": "https://v2.samehadaku.how/…",
  "error": "samehadaku scrape failed (ajax+generic+headless): …"
}
```

**Frontend should:** keep existing scrape-error toast; optional friendlier message for Samehadaku (see §6).

### 4.3 Queue items

Same as other scrapes: queue entry gets resolved URL + title/episode/thumbnail when add succeeds.

---

## 5. Frontend UI updates recommended

### 5.1 Loading / scrape UX (high value)

While waiting for `SET_VIDEO` after host pastes a page URL:

| State | Suggested UI |
|-------|----------------|
| Scraping | Keep/extend loading overlay: "Resolving stream…" |
| Samehadaku / anime page | Optional: "Fetching mirror (usually 1–3s)…" |
| IDLIX | Keep longer copy: "Unlocking stream (~15–25s)…" if you special-case idlix domains |
| Success | Hide spinner; show Now Watching metadata |
| Error | Existing `SCRAPE_ERROR` toast |

**Detection hint (client-side only, optional):**

```js
function scrapeHint(url) {
  const u = url.toLowerCase()
  if (u.includes('idlix')) return 'Unlocking IDLIX stream (~15–25s)…'
  if (u.includes('samehadaku')) return 'Resolving Samehadaku mirror…'
  if (u.includes('otakudesu') || u.includes('anoboy')) return 'Resolving anime stream…'
  if (u.includes('animasu') || u.includes('kuronime') || u.includes('nanime')) return 'Resolving anime stream…'
  return 'Resolving stream…'
}
```

### 5.2 Supported sources list (docs / home / VideoPlayer placeholder)

Update copy where sources are listed:

**Before (incomplete):** "Any MP4 link"  

**After (recommended):**

> Paste a YouTube URL, direct `.m3u8`/`.mp4`, or an anime/movie page  
> (Samehadaku, Otakudesu, Anoboy, Animasu, Kuronime, Nanime, IDLIX, and more).

**VideoPlayer empty state (host):**

```text
Paste a YouTube URL, stream link, or anime page URL
(Samehadaku, Otakudesu, Anoboy, Animasu, Kuronime, Nanime, IDLIX…)
```

### 5.3 Documentation page (`/docs`) & FAQ

Add/update bullets:

**Samehadaku**

- Paste the episode page URL (not only the download link).  
- Backend picks the best mirror (prefers HD Wibufile when available).  
- First resolve is usually **1–3 seconds**.  
- Playback goes through the secure stream proxy when needed.

**FAQ**

| Q | A |
|---|---|
| Why did Samehadaku fail before? | Ads blocked headless scraping; fixed via official player API. |
| Which quality do I get? | Backend prefers 1080p → 720p → 480p when mirrors exist. |
| Do I need a special format? | Full episode page URL on `*.samehadaku.*` is enough. |

### 5.4 Quality / mirror label (optional enhancement)

Today `metadata.episode` may be `"Episode 14"` (from page). Mirror label (`Wibufile 1080p`) may not always be in metadata.

**Optional future backend field** (not required now): `metadata.quality` or `metadata.mirror`.

**Frontend optional polish without backend change:**

- If `videoUrl` contains `wibufile` → badge "Direct MP4"  
- If `.m3u8` → badge "HLS" (already have stream mode badge)

### 5.5 Source badge (optional)

If `currentMetadata.source === 'samehadaku'`, show a small pill:

```
SAMEHADAKU
```

Same pattern can apply for `otakudesu`, `anoboy`, `animasu`, `kuronime`, `nanime`, `idlix`.

---

## 6. Suggested user-facing strings

### Success (toast — optional, only if you add success toasts)

> Stream ready — playing from Samehadaku mirror.

### Error (friendlier mapping)

If `SCRAPE_ERROR.error` contains `samehadaku`:

> Couldn't extract this Samehadaku episode. Try another mirror page or a different host (Wibufile/Blogspot may be down).

Generic:

> Failed to extract stream. Check the URL or try again.

### Placeholder / docs

> Samehadaku: paste the episode page link. We resolve the best available stream automatically.

---

## 7. Domains / proxy (frontend awareness)

| Host pattern | Role |
|--------------|------|
| `*.samehadaku.*` | Episode page (scrape input) |
| `*.wibufile.com` | Direct MP4 (common success path) |
| `blogger.com` / Blogspot video | Alternate mirror |
| `/api/proxy?url=` | Always use for cross-origin media when already using proxy helper |

Frontend **should not** open raw CDN URLs outside the player proxy path if CORS issues appear.

---

## 8. All Supported Scrapable Websites

The backend scraper router (`detectSitePlatform` in `backend/services/scraper.go`) recognizes these domains:

### 8.1 Dedicated Scrapers (optimized, fast)

| Platform | Domain Detection | Method | Typical Latency |
|----------|-----------------|--------|-----------------|
| **Samehadaku** | `*.samehadaku.*` | WordPress AJAX player API (`player_ajax`) | ~1–3s |
| **Otakudesu** | `*.otakudesu.*` | WordPress AJAX mirrors + Packer unpack | ~2–4s |
| **Anoboy** | `*.anoboy.*` | Iframe + base64 mirrors + Blogger extract | ~2–5s |
| **IDLIX** | `*.idlix.*` | API gate/claim chain (UUID → gate → claim → majorplay) | ~15–25s (first load) |

### 8.2 Domain-Detected (routed to generic scraper with multi-layer fallback)

| Platform | Domain Detection | Method |
|----------|-----------------|--------|
| **Animasu** | `*.animasu.*` | Generic multi-layer: HTML → scripts → base64 mirrors → iframes → headless |
| **Kuronime** | `*.kuronime.*` | Generic multi-layer: HTML → scripts → base64 mirrors → iframes → headless |
| **Nanime** | `*.nanime.*` | Generic multi-layer: HTML → scripts → base64 mirrors → iframes → headless |

### 8.3 Generic Fallback (any URL not matching above)

| Type | Method |
|------|--------|
| **Any anime/movie page** | 4-layer static scrape → NanoScraper network interception → Headless Chromium with stealth |
| **Direct `.mp4` URL** | No scrape needed — played directly (proxied if cross-origin) |
| **Direct `.m3u8` URL** | No scrape needed — played via hls.js (proxied if cross-origin) |
| **YouTube URL** | No scrape — YouTube IFrame API |

### 8.4 Generic Scraper Layers (for unknown sites)

1. **Layer 1 — Static HTML**: Search for `.m3u8` / `.mp4` URLs in page source
2. **Layer 2 — Script parsing**: Extract from `<script>` tags (jwplayer configs, JSON, JS vars)
3. **Layer 2.5 — Base64 mirrors**: Decode `<select>` / `<div>` with base64-encoded iframe values
4. **Layer 3 — Iframe recursion**: Follow iframes up to 3 levels deep, resolve Blogger video embeds
5. **Layer 4 — NanoScraper**: Network interception via NanoProxy for JS-heavy SPAs
6. **Layer 5 — Headless Chromium**: Full stealth browser with ad/image/font blocking, `navigator.webdriver` spoofing

### 8.5 Example URLs

```
# Samehadaku (fast — ~1s)
https://v2.samehadaku.how/tensei-shitara-slime-datta-ken-season-4-episode-14/

# Otakudesu (fast — ~2s)
https://otakudesu.blog/episode/something-episode-1/

# Anoboy (fast — ~3s)
https://anoboy.si/episode/something-episode-1/

# IDLIX (slower — ~15-25s first load due to anti-bot gate)
https://z2.idlixku.com/movie/some-title-2026/

# Animasu / Kuronime / Nanime (generic fallback — ~3-10s)
https://animasu.vip/episode/something-episode-1/
https://kuronime.com/episode/something-episode-1/
https://nanime.in/episode/something-episode-1/

# Any other anime/movie site (generic + headless fallback — ~5-15s)
https://some-anime-site.com/episode/something/

# Direct links (no scrape — instant)
https://example.com/video.mp4
https://example.com/stream/index.m3u8
https://youtube.com/watch?v=xxxxxxxxxxx
```

---

## 9. What frontend does **not** need to change

| Area | Status |
|------|--------|
| WebSocket contract | Unchanged |
| `SET_VIDEO` / `SCRAPE_ERROR` handlers | Already sufficient |
| Queue add flow | Works as-is |
| Host-only set video | Unchanged |
| Room layout | No required layout change |

Minimum frontend work: **copy + loading hints + docs/FAQ**. Optional: source/quality badges.

---

## 10. Manual QA checklist (frontend + backend running)

1. Restart Go backend (`cd backend && go run .`).  
2. `npm run dev` frontend.  
3. Create room as host.  
4. Paste:  
   `https://v2.samehadaku.how/tensei-shitara-slime-datta-ken-season-4-episode-14/`  
5. Expect within a few seconds:  
   - Video loads (MP4)  
   - Title in Now Watching  
   - No long "deadline" hang  
6. Guest join → same video + metadata.  
7. Add same URL to queue → queue card title/thumbnail if present.  
8. Failure case: garbage URL → existing error toast.

---

## 11. Files reference

| File | Role |
|------|------|
| `backend/services/scraper.go` | Main router: `detectSitePlatform` + generic scraper + headless fallback |
| `backend/services/samehadaku_scraper.go` | Samehadaku AJAX player resolution |
| `backend/services/idlix_scraper.go` | IDLIX API gate/claim chain |
| `backend/services/nano_scraper.go` | NanoProxy network interception scraper |
| `backend/handlers/proxy.go` | Allowlist: `wibufile`, `samehadaku`, dynamic CDN hosts |
| `FEATURES_DOCUMENTATION.md` | Global feature inventory for `/docs` |
| `SAMEHADAKU_SCRAPER_REPORT.md` | This report (frontend interface guide) |

---

## 12. Suggested frontend task list

| Priority | Task |
|----------|------|
| P0 | Ensure scrape loading UI doesn't assume 30–40s only for Samehadaku (short hint OK) |
| P1 | Update docs/FAQ/home feature text: Samehadaku supported, fast resolve |
| P1 | VideoPlayer host placeholder mentions all supported sites |
| P1 | Update `/docs` page with full list of scrapable websites |
| P2 | Source pill from `metadata.source` |
| P2 | Friendlier `SCRAPE_ERROR` text for samehadaku |
| P3 | Success toast when stream resolves from page URL |

---

*End of report — use this to update DocumentationPage, FAQ, placeholders, and scrape loading UX.*
