package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

const stealthUserAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"

// webdriverBypassJS hides navigator.webdriver before any page script runs
// (required to reduce Cloudflare Turnstile / bot detection hits).
const webdriverBypassJS = `Object.defineProperty(navigator, 'webdriver', {get: () => undefined});`

// Default timeouts for stealth sessions.
const (
	defaultSessionTimeout = 40 * time.Second // overall chromedp session
	defaultNavTimeout     = 18 * time.Second // navigate + wait for body/player only
	defaultStreamWait     = 12 * time.Second // poll for network-captured streams after DOM ready
)

// blockedURLPatterns are CDP URLPattern strings (absolute) for Network.setBlockedURLs.
// Images are intentionally NOT blocked — many players gate streams on poster/thumb loads.
// See: https://urlpattern.spec.whatwg.org/ — example: *://*:*/*.css
var blockedURLPatterns = []string{
	// Fonts
	"*://*/*/*.woff", "*://*/*/*.woff2", "*://*/*/*.ttf", "*://*/*/*.otf", "*://*/*/*.eot",
	// Analytics / ads hosts (explicit + wildcards)
	"*://google-analytics.com/*", "*://*.google-analytics.com/*",
	"*://googletagmanager.com/*", "*://*.googletagmanager.com/*",
	"*://googleadservices.com/*", "*://*.googleadservices.com/*",
	"*://doubleclick.net/*", "*://*.doubleclick.net/*",
	"*://googlesyndication.com/*", "*://*.googlesyndication.com/*",
	"*://*.facebook.net/*", "*://connect.facebook.net/*",
	"*://*.hotjar.com/*",
	"*://*.scorecardresearch.com/*",
	"*://*.quantserve.com/*",
	"*://*.adnxs.com/*",
	"*://*.adsrvr.org/*",
	"*://*.taboola.com/*",
	"*://*.outbrain.com/*",
	"*://*.histats.com/*",
	"*://*.statcounter.com/*",
	"*://mc.yandex.ru/*",
	"*://*.propellerads.com/*",
	"*://*.exoclick.com/*",
	// Samehadaku / Indo anime ad networks
	"*://*.adsterra.com/*", "*://adsterra.com/*",
	"*://*.juicyads.com/*", "*://*.popads.net/*", "*://*.popcash.net/*",
	"*://*.trafficjunky.net/*", "*://*.tsyndicate.com/*",
	"*://*.hilltopads.com/*", "*://*.hilltopads.net/*",
	"*://*.clickadu.com/*", "*://*.adk2x.com/*",
	"*://*.adskeeper.com/*", "*://*.mgid.com/*",
	"*://*.pubmatic.com/*", "*://*.openx.net/*",
	"*://*.rubiconproject.com/*", "*://*.casalemedia.com/*",
	"*://*.moatads.com/*", "*://*.amazon-adsystem.com/*",
	"*://*.2mdn.net/*", "*://*.pagead2.googlesyndication.com/*",
	"*://*.googletagservices.com/*",
	"*://*.adservice.google.com/*",
	"*://*.partner.googleadservices.com/*",
	"*://*.securepubads.g.doubleclick.net/*",
}

// highConfidenceStreamURL is true for playable media hosts we should abort on immediately
// (Samehadaku embeds: krakenfiles, acefile, hxfile, m3u8, etc.).
func highConfidenceStreamURL(raw string) bool {
	if raw == "" || strings.HasPrefix(raw, "data:") || strings.HasPrefix(raw, "blob:") {
		return false
	}
	lower := strings.ToLower(raw)
	// Direct playlists / progressive
	if strings.Contains(lower, ".m3u8") ||
		strings.Contains(lower, "master.m3u8") ||
		strings.Contains(lower, "index.m3u8") ||
		strings.Contains(lower, "playlist.m3u8") {
		return true
	}
	if strings.Contains(lower, ".mp4") && !strings.Contains(lower, ".mp44") {
		// Prefer real media hosts, not tracking pixels named *.mp4
		if strings.Contains(lower, "krakenfiles") ||
			strings.Contains(lower, "acefile") ||
			strings.Contains(lower, "hxfile") ||
			strings.Contains(lower, "streamtape") ||
			strings.Contains(lower, "filemoon") ||
			strings.Contains(lower, "vidhide") ||
			strings.Contains(lower, "mp4upload") ||
			strings.Contains(lower, "pixeldrain") ||
			strings.Contains(lower, "/hls/") ||
			strings.Contains(lower, "/stream/") ||
			strings.Contains(lower, "/video/") ||
			strings.Contains(lower, "cdn") {
			return true
		}
		// Long .mp4 paths are usually real
		if len(raw) > 40 {
			return true
		}
	}
	// Known embed/player hosts used by Samehadaku mirrors
	hosts := []string{
		"krakenfiles.com", "acefile.co", "acefile.cc", "hxfile.co",
		"streamtape.com", "filemoon.", "vidhide", "mp4upload.com",
		"pixeldrain.com", "luluvdo.com", "vidguard", "dood.",
		"mixdrop.", "upstream.", "sbplay", "streamsb",
	}
	for _, h := range hosts {
		if strings.Contains(lower, h) {
			return true
		}
	}
	return false
}

// streamURLLooksLike reports whether a network URL is likely a video stream
// or player embed endpoint worth capturing.
func streamURLLooksLike(raw string) bool {
	if highConfidenceStreamURL(raw) {
		return true
	}
	if raw == "" || strings.HasPrefix(raw, "data:") {
		return false
	}
	lower := strings.ToLower(raw)
	if strings.Contains(lower, "/embed/") ||
		strings.Contains(lower, "/hls/") ||
		strings.Contains(lower, "/stream/") {
		return true
	}
	return false
}

// stealthAllocatorOpts returns production-safe full-headless Chrome flags
// for Linux VPS environments.
func stealthAllocatorOpts() []chromedp.ExecAllocatorOption {
	opts := []chromedp.ExecAllocatorOption{
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.Headless,
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("disable-features", "IsolateOrigins,site-per-process"),
		chromedp.Flag("disable-web-security", true),
		chromedp.UserAgent(stealthUserAgent),
		chromedp.WindowSize(1920, 1080),
	}

	if cp := findChromePath(); cp != "" {
		opts = append(opts, chromedp.ExecPath(cp))
	}

	if nanoProxyHTTP != "" {
		log.Printf("[Chrome] Routing through NanoProxy: %s", nanoProxyHTTP)
		opts = append(opts, chromedp.ProxyServer(nanoProxyHTTP))
	}

	return opts
}

// injectWebdriverBypass registers a script that runs on every new document
// before page JS, spoofing navigator.webdriver.
func injectWebdriverBypass() chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		_, err := page.AddScriptToEvaluateOnNewDocument(webdriverBypassJS).Do(ctx)
		return err
	})
}

// enableResourceBlocking blocks ads/analytics/fonts so navigation
// finishes faster and is less likely to hit context deadline.
func enableResourceBlocking() chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		patterns := make([]*network.BlockPattern, 0, len(blockedURLPatterns))
		for _, p := range blockedURLPatterns {
			patterns = append(patterns, &network.BlockPattern{
				URLPattern: p,
				Block:      true,
			})
		}
		return network.SetBlockedURLs().WithURLPatterns(patterns).Do(ctx)
	})
}

// fastNavigate issues page.Navigate without waiting for full NetworkIdle /
// complete load (which stalls on heavy ad networks). Callers should wait for
// body/player separately with a short timeout.
func fastNavigate(pageURL string) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		_, _, errorText, _, err := page.Navigate(pageURL).Do(ctx)
		if err != nil {
			return err
		}
		if errorText != "" {
			return fmt.Errorf("page navigate: %s", errorText)
		}
		return nil
	})
}

// waitForBodyOrPlayer polls until body, video, iframe player, or common
// player containers appear — not the full page load event.
func waitForBodyOrPlayer(timeout time.Duration) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		deadline := time.Now().Add(timeout)
		selectors := []string{
			`path[d^="M18.8906"]`, // IDLIX play SVG overlay
			`path[d*="M18.8906 12.846"]`,
			"video",
			"iframe",
			"#player",
			".player",
			".video-js",
			".jwplayer",
			".plyr",
			`div[class*="play"]`,
			`div[id*="player"]`,
			"[class*='player']",
			"body",
		}
		for time.Now().Before(deadline) {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			for _, sel := range selectors {
				var n int
				// count matching nodes
				err := chromedp.Evaluate(fmt.Sprintf(
					`document.querySelectorAll(%q).length`, sel,
				), &n).Do(ctx)
				if err == nil && n > 0 {
					if sel != "body" {
						log.Printf("[Chrome] Early DOM ready via selector %s", sel)
					}
					return nil
				}
			}
			time.Sleep(200 * time.Millisecond)
		}
		// Soft: body may still be usable even if poll timed out
		return nil
	})
}

// idlixPlaySVGPathPrefix is the start of the IDLIX play-overlay SVG path "d"
// attribute. We never click the <path>/<svg> itself — only a clickable parent.
const idlixPlaySVGPathPrefix = "M18.8906 12.846C"

// clickIdlixSVGPlayOverlay waits for the IDLIX SVG play overlay (if present)
// and clicks the closest clickable parent (div/button), not the SVG/path.
// Falls back to JS injection when chromedp.Click is unreliable on overlays.
func clickIdlixSVGPlayOverlay() chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		// Wait briefly for the play SVG path to appear in the DOM
		deadline := time.Now().Add(4 * time.Second)
		for time.Now().Before(deadline) {
			if ctx.Err() != nil {
				break
			}
			var found bool
			_ = chromedp.Evaluate(fmt.Sprintf(`(() => {
				const paths = document.querySelectorAll('path[d^="%s"], path[d*="M18.8906"]');
				return paths.length > 0;
			})()`, idlixPlaySVGPathPrefix), &found).Do(ctx)
			if found {
				break
			}
			time.Sleep(150 * time.Millisecond)
		}

		// JS injection: find SVG path → click closest div/button parent (not path/svg)
		var clicked bool
		_ = chromedp.Evaluate(fmt.Sprintf(`(() => {
			const selectors = [
				'path[d^="%s"]',
				'path[d*="M18.8906 12.846"]',
				'path[d*="M18.8906"]',
			];
			let pathEl = null;
			for (const sel of selectors) {
				pathEl = document.querySelector(sel);
				if (pathEl) break;
			}
			if (!pathEl) {
				// Broader: any large play-triangle path inside a player container
				const all = document.querySelectorAll('svg path');
				for (const p of all) {
					const d = p.getAttribute('d') || '';
					if (d.startsWith('M18.8906') || d.includes('12.846C')) {
						pathEl = p;
						break;
					}
				}
			}
			if (!pathEl) return false;

			// Prefer button / role=button / clickable div — never the path or svg itself
			const clickable =
				pathEl.closest('button') ||
				pathEl.closest('[role="button"]') ||
				pathEl.closest('div[class*="play"]') ||
				pathEl.closest('div[id*="play"]') ||
				pathEl.closest('div[class*="player"]') ||
				pathEl.closest('div[id*="player"]') ||
				pathEl.closest('a') ||
				pathEl.closest('div') ||
				pathEl.parentElement;

			if (!clickable || clickable.tagName === 'PATH' || clickable.tagName === 'SVG') {
				return false;
			}

			// Dispatch full click sequence (some players ignore .click() alone)
			const opts = { bubbles: true, cancelable: true, view: window };
			clickable.dispatchEvent(new MouseEvent('mousedown', opts));
			clickable.dispatchEvent(new MouseEvent('mouseup', opts));
			clickable.dispatchEvent(new MouseEvent('click', opts));
			try { clickable.click(); } catch (e) {}
			console.log('[WatchParty] IDLIX play overlay clicked via parent', clickable.tagName, clickable.className);
			return true;
		})()`, idlixPlaySVGPathPrefix), &clicked).Do(ctx)

		if clicked {
			log.Printf("[Chrome] IDLIX SVG play overlay parent clicked via JS injection")
		}
		return nil
	})
}

// tryClickPlayTriggers attempts IDLIX SVG overlay first, then common
// play-button / player selectors, to force lazy-loaded media network requests.
func tryClickPlayTriggers() chromedp.Action {
	selectors := []string{
		// IDLIX / generic play wrappers (parent of SVG, not the path)
		`div[class*="play"]`,
		`div[id*="play"]`,
		`div[class*="player"]`,
		`div[id*="player"]`,
		`button[class*="play"]`,
		`.play-button`,
		`.play-btn`,
		`.vjs-big-play-button`,
		`.jw-icon-display`,
		`.plyr__control--overlaid`,
		`button[aria-label*="Play" i]`,
		`button[title*="Play" i]`,
		`.video-js`,
		`#player`,
		`.player`,
		`video`,
		`.embed-responsive`,
	}
	return chromedp.ActionFunc(func(ctx context.Context) error {
		// 1) IDLIX-specific SVG overlay → click parent div/button
		_ = clickIdlixSVGPlayOverlay().Do(ctx)

		// 2) Standard visible clickable parents (never path/svg)
		for _, sel := range selectors {
			_ = chromedp.Click(sel, chromedp.ByQuery, chromedp.NodeVisible).Do(ctx)
		}

		// 3) Generic JS: video.play + iframe click + any remaining play wrappers
		_ = chromedp.Evaluate(`(() => {
			// Click div/button wrappers that contain play SVGs (generic sites)
			document.querySelectorAll('svg path').forEach(path => {
				const d = path.getAttribute('d') || '';
				if (!d.includes('M18') && !d.toLowerCase().includes('play')) return;
				const parent = path.closest('button, [role="button"], div[class*="play"], div[id*="player"], div') || path.parentElement;
				if (parent && parent.tagName !== 'PATH' && parent.tagName !== 'SVG') {
					try { parent.click(); } catch (e) {}
				}
			});
			document.querySelectorAll('video').forEach(v => {
				try { v.muted = true; v.play().catch(() => {}); } catch (e) {}
			});
			document.querySelectorAll('iframe').forEach(f => {
				try { f.click(); } catch (e) {}
			});
		})()`, nil).Do(ctx)
		return nil
	})
}

// waitForStreamOrTimeout polls the network capture until a stream URL appears
// or the wait budget expires. Returns early on success to avoid long sleeps.
// If context was canceled because of abort-on-match, returns immediately (success path).
func waitForStreamOrTimeout(cap *networkCapture, timeout time.Duration) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		deadline := time.Now().Add(timeout)
		for time.Now().Before(deadline) {
			if ctx.Err() != nil {
				// Abort-on-match cancels context — not a failure if we have a URL
				return ctx.Err()
			}
			if cap.bestStream() != "" {
				log.Printf("[Chrome] Stream captured early, stopping wait")
				cap.abortOnce()
				return context.Canceled
			}
			// Periodic play clicks while waiting
			_ = tryClickPlayTriggers().Do(ctx)
			time.Sleep(300 * time.Millisecond)
		}
		return nil
	})
}

// networkCapture holds thread-safe intercepted stream URLs.
// When a high-confidence stream is found, abort() kills the Chromium session
// so ad iframes cannot hold the process until deadline.
type networkCapture struct {
	mu       sync.Mutex
	m3u8     []string
	mp4      []string
	embed    []string
	all      []string
	aborted  bool
	abortFn  context.CancelFunc
	abortOnceFn sync.Once
}

func (c *networkCapture) setAbort(fn context.CancelFunc) {
	c.mu.Lock()
	c.abortFn = fn
	c.mu.Unlock()
}

// abortOnce cancels the headless session exactly once (kill Chromium gracefully).
func (c *networkCapture) abortOnce() {
	c.abortOnceFn.Do(func() {
		c.mu.Lock()
		c.aborted = true
		fn := c.abortFn
		c.mu.Unlock()
		if fn != nil {
			log.Printf("[Chrome] Abort-on-match: canceling headless context (stream found)")
			fn()
		}
	})
}

func (c *networkCapture) wasAborted() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.aborted
}

func (c *networkCapture) add(reqURL string) {
	if reqURL == "" {
		return
	}
	c.mu.Lock()
	c.all = append(c.all, reqURL)
	lower := strings.ToLower(reqURL)
	matched := false
	if strings.Contains(lower, ".m3u8") || strings.Contains(lower, "master.m3u8") {
		if !nanoContains(c.m3u8, reqURL) {
			c.m3u8 = append(c.m3u8, reqURL)
		}
		matched = true
	} else if strings.Contains(lower, ".mp4") && !strings.Contains(lower, ".mp44") {
		if !nanoContains(c.mp4, reqURL) {
			c.mp4 = append(c.mp4, reqURL)
		}
		matched = true
	} else if streamURLLooksLike(reqURL) {
		if !nanoContains(c.embed, reqURL) {
			c.embed = append(c.embed, reqURL)
		}
		// Only abort on high-confidence embeds (krakenfiles etc.), not every /embed/
		matched = highConfidenceStreamURL(reqURL)
	}
	shouldAbort := matched && highConfidenceStreamURL(reqURL)
	c.mu.Unlock()

	if shouldAbort {
		log.Printf("[Chrome] High-confidence stream URL captured — aborting session: %s", truncateURL(reqURL, 120))
		c.abortOnce()
	}
}

func (c *networkCapture) bestStream() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if s := nanoSelectBest(c.m3u8, c.mp4); s != "" {
		return s
	}
	// Prefer high-confidence embed hosts (krakenfiles, acefile, …)
	for _, u := range c.embed {
		if highConfidenceStreamURL(u) {
			return u
		}
	}
	for _, u := range c.embed {
		lower := strings.ToLower(u)
		if strings.Contains(lower, ".m3u8") || strings.Contains(lower, ".mp4") {
			return u
		}
	}
	return ""
}

// hasUsefulCapture is true when we already have stream or embed URLs worth keeping
// even if the overall page navigation later times out.
func (c *networkCapture) hasUsefulCapture() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.m3u8) > 0 || len(c.mp4) > 0 || len(c.embed) > 0
}

func (c *networkCapture) snapshot() (m3u8, mp4, embed []string, total int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	m3u8 = append([]string(nil), c.m3u8...)
	mp4 = append([]string(nil), c.mp4...)
	embed = append([]string(nil), c.embed...)
	total = len(c.all)
	return
}

// attachNetworkListener registers ListenTarget handlers that capture stream URLs
// and abort the session on the first high-confidence match.
func attachNetworkListener(ctx context.Context, cap *networkCapture) {
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		switch e := ev.(type) {
		case *network.EventRequestWillBeSent:
			cap.add(e.Request.URL)
			if strings.HasPrefix(e.Request.URL, "blob:") {
				log.Printf("[Chrome] Detected blob URL: %s", e.Request.URL)
			}
		case *network.EventResponseReceived:
			cap.add(e.Response.URL)
		case *network.EventWebSocketFrameReceived:
			if e.Response == nil {
				return
			}
			payload := string(e.Response.PayloadData)
			if match := m3u8Regex.FindString(payload); match != "" {
				cap.add(match)
			}
			if match := mp4Regex.FindString(payload); match != "" {
				cap.add(match)
			}
		}
	})
}

// isDeadlineErr reports context deadline / cancellation from chromedp.
func isDeadlineErr(err error) bool {
	if err == nil {
		return false
	}
	if err == context.DeadlineExceeded || err == context.Canceled {
		return true
	}
	s := err.Error()
	return strings.Contains(s, "context deadline exceeded") ||
		strings.Contains(s, "context canceled") ||
		strings.Contains(s, "timeout")
}

// isAbortSuccess is true when context was canceled intentionally after a stream hit
// (or deadline hit with a captured URL) — treat as 100% success, not failure.
func isAbortSuccess(err error, cap *networkCapture) bool {
	if cap == nil {
		return false
	}
	if cap.bestStream() == "" && !cap.hasUsefulCapture() {
		return false
	}
	if err == nil {
		return true
	}
	return err == context.Canceled || isDeadlineErr(err) || cap.wasAborted()
}

func truncateURL(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// stealthNavigate runs a stealth headless session with:
//   - resource blocking (ads/fonts/analytics; images kept for player gate)
//   - fast navigate (no full NetworkIdle wait)
//   - short nav timeout separate from session timeout
//   - abort-on-match: cancel Chromium as soon as .m3u8 / krakenfiles / etc. is seen
//   - context.Canceled + non-empty stream = SUCCESS
func stealthNavigate(pageURL string, sessionTimeout time.Duration) (*networkCapture, string, error) {
	if findChromePath() == "" {
		return nil, "", fmt.Errorf("Chrome/Chromium not installed — headless scraping unavailable")
	}
	if sessionTimeout <= 0 {
		sessionTimeout = defaultSessionTimeout
	}

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), stealthAllocatorOpts()...)
	defer allocCancel()

	// Outer session: timeout + explicit cancel for abort-on-match
	baseCtx, baseCancel := context.WithCancel(allocCtx)
	defer baseCancel()

	sessionCtx, sessionCancel := chromedp.NewContext(baseCtx)
	defer sessionCancel()

	sessionCtx, sessionTimeoutCancel := context.WithTimeout(sessionCtx, sessionTimeout)
	defer sessionTimeoutCancel()

	// abort cancel kills the whole headless tree (stops ad iframes holding Chrome)
	abortCtx, abortCancel := context.WithCancel(sessionCtx)
	defer abortCancel()

	// Re-bind chromedp context under abortCtx so cancel stops Run()
	// Listen on the chromedp context derived from abortCtx
	runCtx, runCancel := chromedp.NewContext(abortCtx)
	defer runCancel()

	cap := &networkCapture{}
	cap.setAbort(func() {
		abortCancel()
		runCancel()
		sessionCancel()
	})
	attachNetworkListener(runCtx, cap)

	// Phase 1: enable network + blocking + webdriver bypass
	if err := chromedp.Run(runCtx,
		network.Enable(),
		enableResourceBlocking(),
		injectWebdriverBypass(),
	); err != nil {
		if isAbortSuccess(err, cap) {
			log.Printf("[Chrome] Setup aborted but stream already captured — SUCCESS")
			return cap, "", nil
		}
		return cap, "", fmt.Errorf("chrome setup failed: %w", err)
	}

	// Phase 2: navigate with a shorter dedicated timeout (does not wait for ads)
	navCtx, navCancel := context.WithTimeout(runCtx, defaultNavTimeout)
	navErr := chromedp.Run(navCtx,
		fastNavigate(pageURL),
		waitForBodyOrPlayer(defaultNavTimeout-2*time.Second),
	)
	navCancel()

	if isAbortSuccess(navErr, cap) && cap.bestStream() != "" {
		log.Printf("[Chrome] Abort-on-match during nav for %s — SUCCESS stream=%s",
			pageURL, truncateURL(cap.bestStream(), 100))
		return cap, bestEffortHTML(runCtx), nil
	}

	if navErr != nil {
		if cap.bestStream() != "" || cap.hasUsefulCapture() {
			log.Printf("[Chrome] Nav timeout/error for %s but capture has data — SUCCESS: %v", pageURL, navErr)
			return cap, bestEffortHTML(runCtx), nil
		}
		if isDeadlineErr(navErr) {
			log.Printf("[Chrome] Navigation deadline for %s: %v", pageURL, navErr)
		} else {
			log.Printf("[Chrome] Navigation error for %s: %v", pageURL, navErr)
		}
	}

	// Phase 3: play clicks + poll network; abort-on-match may cancel mid-wait
	var renderedHTML string
	streamErr := chromedp.Run(runCtx,
		clickIdlixSVGPlayOverlay(),
		tryClickPlayTriggers(),
		waitForStreamOrTimeout(cap, defaultStreamWait),
		clickIdlixSVGPlayOverlay(),
		chromedp.OuterHTML("html", &renderedHTML, chromedp.ByQuery),
	)

	// context.Canceled + non-empty URL = 100% SUCCESS (abort-on-match)
	if isAbortSuccess(streamErr, cap) {
		if streamErr != nil {
			log.Printf("[Chrome] Abort/cancel for %s treated as SUCCESS (stream=%v): %v",
				pageURL, cap.bestStream() != "", streamErr)
		}
		if renderedHTML == "" {
			renderedHTML = bestEffortHTML(runCtx)
		}
		return cap, renderedHTML, nil
	}

	if streamErr != nil {
		return cap, renderedHTML, streamErr
	}

	return cap, renderedHTML, nil
}

// bestEffortHTML tries to grab OuterHTML without failing the scrape.
func bestEffortHTML(ctx context.Context) string {
	var html string
	// Short attempt only
	short, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	_ = chromedp.Run(short, chromedp.OuterHTML("html", &html, chromedp.ByQuery))
	return html
}
