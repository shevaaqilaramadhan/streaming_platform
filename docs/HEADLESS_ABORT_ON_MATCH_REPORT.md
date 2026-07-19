# Report: Headless Abort-on-Match (Samehadaku / SPA Scraper Optimization)

**Date:** 17 July 2026  
**Status:** Implemented & verified (`go test` 21 passed, build OK)  
**Primary files:** `backend/services/chrome_stealth.go`, `nano_scraper.go`, `scraper.go`

---

## 1. Problem

Logs for Samehadaku (and similar iframe-heavy sites) showed:

```
Early DOM ready via selector iframe
…
context deadline exceeded
```

### Root cause

1. Headless Chrome navigated and detected the **player iframe** quickly.
2. The page (and nested iframe) kept loading **ads / analytics** indefinitely.
3. Chromium stayed alive until the **session timeout** (e.g. 40s) → `context deadline exceeded`.
4. Even when a valid stream (`.m3u8`, Krakenfiles, Acefile, Hxfile, …) had already appeared on the network, the scrape path often treated the deadline as a **hard failure**.

Samehadaku’s video iframes are notorious for background ad scripts (`doubleclick`, `adsterra`, popunders, etc.) that prevent a clean “page idle” lifecycle.

---

## 2. Solution: Abort-on-Match

### 2.1 Cancel context as soon as a stream is found

```
Navigate → block ads → listen network
                │
                ▼
     RequestWillBeSent / ResponseReceived
                │
     high-confidence URL? (.m3u8, krakenfiles, acefile, hxfile, …)
                │
                ▼
     capture URL + abortOnce() → context.Cancel
                │
                ▼
     Chromium killed early (ads cannot hold the process)
                │
                ▼
     err == context.Canceled + stream != ""  →  SUCCESS
```

### 2.2 Implementation details

| Piece | Behavior |
|-------|----------|
| `networkCapture.abortFn` | `context.CancelFunc` wired to kill the headless session |
| `cap.add(url)` | On high-confidence match → `abortOnce()` |
| `attachNetworkListener` | Still listens to request/response/WS frames |
| `waitForStreamOrTimeout` | Exits immediately if stream already present / context canceled |
| `isAbortSuccess(err, cap)` | `Canceled` or deadline **and** non-empty stream → success |
| `NanoScrape` / `scrapeWithHeadless` | Treat abort success as clean result (no error) |

### 2.3 High-confidence URL rules

Abort is **not** triggered for every `/embed/` (too noisy). Abort when:

- URL contains `.m3u8` / `master.m3u8` / `index.m3u8` / `playlist.m3u8`
- Real `.mp4` on known media patterns (cdn, hls, stream, video, long path)
- Host matches: **krakenfiles**, **acefile**, **hxfile**, streamtape, filemoon, vidhide, mp4upload, pixeldrain, luluvdo, vidguard, dood, mixdrop, upstream, sbplay, streamsb

### 2.4 Expanded ad / analytics blocking

Additional CDP `SetBlockedURLs` patterns (on top of images/fonts):

- `*.doubleclick.net`, `*.google-analytics.com`, GTM, googlesyndication  
- **adsterra**, juicyads, popads, popcash, hilltopads, clickadu  
- trafficjunky, tsyndicate, adskeeper, mgid, pubmatic, openx, rubicon, moat, amazon-adsystem, 2mdn, securepubads, etc.

Goal: less bandwidth, fewer hung connections inside the player iframe.

---

## 3. Success criteria (new)

| Condition | Result |
|-----------|--------|
| Stream captured + `context.Canceled` (abort) | **SUCCESS** |
| Stream captured + `context.DeadlineExceeded` | **SUCCESS** (soft) |
| No stream + deadline/cancel | **FAILURE** |
| Stream captured + no error | **SUCCESS** |

User-visible effect: Samehadaku / generic headless scrapes finish **as soon as the mirror URL is seen**, instead of waiting for the full timeout while ads run.

---

## 4. Samehadaku path (unchanged routing, faster headless)

```
scrapeSamehadaku
  → scrapeGeneric (fast static HTML)
  → on fail: scrapeWithHeadless
       → stealthNavigate (abort-on-match)
       → NanoScrape may run first via generic SPA fallback chain
```

No change to platform detection; only the **headless engine** is smarter.

---

## 5. Expected log patterns

**Before (bad):**

```
[Chrome] Early DOM ready via selector iframe
… long silence / ad traffic …
context deadline exceeded
scrape failed
```

**After (good):**

```
[Chrome] Early DOM ready via selector iframe
[Chrome] High-confidence stream URL captured — aborting session: https://…m3u8…
[Chrome] Abort-on-match: canceling headless context (stream found)
[Headless] Abort-on-match SUCCESS for https://samehadaku.…
```

---

## 6. Verification

```bash
cd backend
go test ./services/ ./handlers/ -count=1 -timeout 60s
go build -o /tmp/watchparty-server .
```

- **21 tests passed** (including `TestHighConfidenceStreamURL`, `TestIsAbortSuccess`)
- Build OK

Manual check:

1. Paste a Samehadaku episode URL in a room  
2. Confirm stream resolves without waiting full 40s when mirror appears  
3. Confirm dynamic proxy allowlist still registers media host  

---

## 7. Limitations

| Limitation | Note |
|------------|------|
| Abort only on high-confidence hosts | Unknown CDNs may still wait until soft timeout (still soft-success if URL captured) |
| Nested iframe CF / captcha | Abort cannot help if stream never hits the network |
| Chromium still required | No Chrome → headless path disabled |
| First request cold start | Chrome launch cost remains (~1–2s) |

---

## 8. Docs / product notes

For `FEATURES_DOCUMENTATION.md` / `/docs`:

> **Faster anime page scrapes**  
> When the backend detects a real stream URL (HLS playlist or known hosts like Krakenfiles), it **stops the headless browser immediately** instead of waiting for ad-heavy pages to finish loading. This especially helps Samehadaku and similar sites.

---

## 9. Files touched

| File | Change |
|------|--------|
| `services/chrome_stealth.go` | Abort-on-match, `isAbortSuccess`, ad block expansion, high-confidence URL rules |
| `services/nano_scraper.go` | Treat abort as success |
| `services/scraper.go` (`scrapeWithHeadless`) | Treat abort as success |
| `services/chrome_stealth_test.go` | Unit tests for confidence + abort success |
| `HEADLESS_ABORT_ON_MATCH_REPORT.md` | This report |

---

*End of report.*
