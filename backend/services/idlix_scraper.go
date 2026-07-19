package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"watchparty-backend/models"
)

const (
	idlixDefaultBaseURL = "https://z2.idlixku.com"
	idlixMobileUA       = "Mozilla/5.0 (Linux; Android 6.0; Nexus 5 Build/MRA58N) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Mobile Safari/537.36"
	idlixGateMaxWait    = 20 * time.Second
	// Warm navigation + gate (~15s) + API steps + optional claim retry
	idlixBrowserTimeout = 120 * time.Second
)

// idlixParsedURL holds content coordinates extracted from an IDLIX page URL.
type idlixParsedURL struct {
	BaseURL string // e.g. https://z2.idlixku.com
	Kind    string // "movie" | "series"
	Slug    string
	Season  int
	Episode int
	Referer string
	PageURL string
}

var (
	idlixMoviePathRe  = regexp.MustCompile(`(?i)/movie/([^/?#]+)`)
	idlixSeriesPathRe = regexp.MustCompile(`(?i)/series/([^/?#]+)(?:/season/(\d+))?(?:/episode/(\d+))?`)
)

// parseIdlixURL extracts movie/series slug and episode coordinates from a page URL.
// Exported-style for unit tests (same package).
func parseIdlixURL(pageURL string) (*idlixParsedURL, error) {
	pageURL = strings.TrimSpace(pageURL)
	if pageURL == "" {
		return nil, fmt.Errorf("empty IDLIX URL")
	}

	parsed, err := url.Parse(pageURL)
	if err != nil {
		return nil, fmt.Errorf("invalid IDLIX URL: %w", err)
	}
	if parsed.Scheme == "" {
		parsed, err = url.Parse("https://" + pageURL)
		if err != nil {
			return nil, fmt.Errorf("invalid IDLIX URL: %w", err)
		}
	}

	host := strings.ToLower(parsed.Hostname())
	if host == "" {
		return nil, fmt.Errorf("IDLIX URL missing host")
	}

	base := idlixDefaultBaseURL
	if strings.Contains(host, "idlix") {
		base = parsed.Scheme + "://" + parsed.Host
	}

	path := parsed.EscapedPath()
	if path == "" {
		path = parsed.Path
	}

	out := &idlixParsedURL{
		BaseURL: base,
		PageURL: pageURL,
	}

	if m := idlixMoviePathRe.FindStringSubmatch(path); len(m) > 1 {
		out.Kind = "movie"
		out.Slug = strings.TrimSpace(m[1])
		// strip trailing junk like empty segments
		out.Slug = strings.Trim(out.Slug, "/")
		if out.Slug == "" {
			return nil, fmt.Errorf("IDLIX movie URL missing slug: %s", pageURL)
		}
		out.Referer = base + "/movie/" + out.Slug
		return out, nil
	}

	if m := idlixSeriesPathRe.FindStringSubmatch(path); len(m) > 1 {
		out.Kind = "series"
		out.Slug = strings.TrimSpace(m[1])
		if out.Slug == "" {
			return nil, fmt.Errorf("IDLIX series URL missing slug: %s", pageURL)
		}
		out.Season = 1
		out.Episode = 1
		if len(m) > 2 && m[2] != "" {
			if n, err := strconv.Atoi(m[2]); err == nil && n > 0 {
				out.Season = n
			}
		}
		if len(m) > 3 && m[3] != "" {
			if n, err := strconv.Atoi(m[3]); err == nil && n > 0 {
				out.Episode = n
			}
		}
		out.Referer = fmt.Sprintf("%s/series/%s/season/%d/episode/%d", base, out.Slug, out.Season, out.Episode)
		return out, nil
	}

	return nil, fmt.Errorf("unrecognized IDLIX URL path (expected /movie/{slug} or /series/{slug}/season/N/episode/M): %s", pageURL)
}

// scrapeIdlix extracts a direct stream URL from an IDLIX movie/episode page
// using the official API chain (not DOM/SVG play-button scraping):
//
//  1. resolve content UUID
//  2. track view (best-effort)
//  3. play-info → gateToken + unlockAt
//  4. wait anti-scrape timer
//  5. claim session → claim JWT + redeemUrl
//  6. redeem on majorplay.net (normal HTTP, text/plain body)
func scrapeIdlix(pageURL string) (*models.VideoMetadata, error) {
	parsed, err := parseIdlixURL(pageURL)
	if err != nil {
		return nil, err
	}

	log.Printf("[IDLIX] Scraping %s kind=%s slug=%s s=%d e=%d base=%s",
		pageURL, parsed.Kind, parsed.Slug, parsed.Season, parsed.Episode, parsed.BaseURL)

	// Concurrency limit shared with other headless scrapes
	select {
	case scrapeSemaphore <- struct{}{}:
		defer func() { <-scrapeSemaphore }()
	case <-time.After(10 * time.Second):
		return nil, fmt.Errorf("IDLIX scrape queue full — try again shortly")
	}

	// Warm on the real page URL first (series often has no /season/N/episode/M path).
	// Origin must be idlix host so session/claim is same-site.
	warm := parsed.PageURL
	if warm == "" {
		warm = parsed.Referer
	}
	sess, err := newIdlixBrowserSession(parsed.BaseURL, warm)
	if err != nil {
		return nil, err
	}
	defer sess.close()

	var meta *models.VideoMetadata
	switch parsed.Kind {
	case "movie":
		meta, err = sess.scrapeMovie(parsed)
	case "series":
		meta, err = sess.scrapeSeries(parsed)
	default:
		return nil, fmt.Errorf("unsupported IDLIX content kind %q", parsed.Kind)
	}
	if err != nil {
		return nil, err
	}
	if meta == nil || meta.VideoURL == "" {
		return nil, fmt.Errorf("IDLIX stream extraction returned empty video URL")
	}
	meta.Source = "idlix"
	if meta.Title == "" {
		meta.Title = humanizeSlug(parsed.Slug)
	}
	if parsed.Kind == "series" {
		meta.Episode = fmt.Sprintf("S%02dE%02d", parsed.Season, parsed.Episode)
	}
	// Dynamic proxy allowlist: stream CDN + thumbnail hosts (rotating domains)
	RegisterStreamMetadata(meta.VideoURL, meta.ThumbnailURL)
	return meta, nil
}

// ── Browser session (CF-protected steps 1–5) ──────────────────────────────────

type idlixBrowserSession struct {
	baseURL     string
	allocCancel context.CancelFunc
	ctx         context.Context
	cancel      context.CancelFunc
}

func newIdlixBrowserSession(baseURL, warmURL string) (*idlixBrowserSession, error) {
	if findChromePath() == "" {
		return nil, fmt.Errorf("Chrome/Chromium not installed — required for IDLIX Cloudflare bypass")
	}

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), stealthAllocatorOpts()...)

	browserCtx, browserCancel := chromedp.NewContext(allocCtx)
	ctx, timeoutCancel := context.WithTimeout(browserCtx, idlixBrowserTimeout)

	// Combined cancel
	cancel := func() {
		timeoutCancel()
		browserCancel()
	}

	s := &idlixBrowserSession{
		baseURL:     baseURL,
		allocCancel: allocCancel,
		ctx:         ctx,
		cancel:      cancel,
	}

	// Must land on the IDLIX origin so subsequent fetch() is same-site.
	// Claiming session from about:blank is treated as cross-site → HTTP 403
	// {"error":"Cross-site request blocked"}.
	if warmURL == "" {
		warmURL = baseURL + "/"
	}
	if err := chromedp.Run(ctx,
		network.Enable(),
		injectWebdriverBypass(),
	); err != nil {
		s.close()
		return nil, fmt.Errorf("IDLIX browser session failed: %w", err)
	}

	if err := s.warmOrigin(warmURL); err != nil {
		s.close()
		return nil, err
	}

	return s, nil
}

// warmOrigin navigates Chromium onto the IDLIX site so document origin matches
// API host (required for session/claim CSRF + cookie jar).
func (s *idlixBrowserSession) warmOrigin(pageURL string) error {
	log.Printf("[IDLIX] Warming browser origin via %s", pageURL)

	// Bounded wait: CF sometimes hangs; prefer partial load over infinite stall.
	navCtx, navCancel := context.WithTimeout(s.ctx, 25*time.Second)
	defer navCancel()

	err := chromedp.Run(navCtx,
		chromedp.Navigate(pageURL),
		chromedp.WaitReady("body", chromedp.ByQuery),
	)
	if err != nil {
		// Fallback: try site root if episode/series page fails (redirects / CF)
		root := s.baseURL + "/"
		if pageURL != root {
			log.Printf("[IDLIX] Warm page failed (%v); retrying base %s", err, root)
			navCtx2, cancel2 := context.WithTimeout(s.ctx, 20*time.Second)
			defer cancel2()
			if err2 := chromedp.Run(navCtx2,
				chromedp.Navigate(root),
				chromedp.WaitReady("body", chromedp.ByQuery),
			); err2 != nil {
				return fmt.Errorf("IDLIX warm navigation failed: %w (root: %v)", err, err2)
			}
		} else {
			return fmt.Errorf("IDLIX warm navigation failed: %w", err)
		}
	}

	// Brief settle for Set-Cookie / CF clearance
	select {
	case <-time.After(800 * time.Millisecond):
	case <-s.ctx.Done():
		return s.ctx.Err()
	}

	// Confirm we are on the expected origin (not stuck on interstitial with wrong host)
	var loc string
	_ = chromedp.Run(s.ctx, chromedp.Location(&loc))
	if loc != "" && !strings.Contains(strings.ToLower(loc), "idlix") {
		log.Printf("[IDLIX] Warning: browser location after warm is %q (expected idlix host)", loc)
	} else {
		log.Printf("[IDLIX] Browser origin ready at %s", loc)
	}
	return nil
}

func (s *idlixBrowserSession) close() {
	if s.cancel != nil {
		s.cancel()
	}
	if s.allocCancel != nil {
		s.allocCancel()
	}
}

// browserFetchResult mirrors the JS fetch response surface we evaluate.
type browserFetchResult struct {
	Status int    `json:"status"`
	OK     bool   `json:"ok"`
	Text   string `json:"text"`
}

// browserFetch runs fetch() inside Chromium so TLS fingerprint matches CF expectations.
// Caller must have warmed the page onto the IDLIX origin (same-site cookies + CSRF).
func (s *idlixBrowserSession) browserFetch(reqURL, method, body string, headers map[string]string) (*browserFetchResult, error) {
	if s.ctx.Err() != nil {
		return nil, fmt.Errorf("browser session expired: %w", s.ctx.Err())
	}

	if method == "" {
		method = "GET"
	}
	if headers == nil {
		headers = map[string]string{}
	}
	// Defaults similar to IDLIX-API
	if _, ok := headers["accept"]; !ok {
		headers["accept"] = "*/*"
	}
	if _, ok := headers["accept-language"]; !ok {
		headers["accept-language"] = "en-US,en;q=0.9"
	}
	// Never set Origin manually — browser forbids overriding it and IDLIX rejects
	// forged/cross-site origins with "Cross-site request blocked".
	delete(headers, "origin")
	delete(headers, "Origin")

	payload := map[string]interface{}{
		"url":     reqURL,
		"method":  method,
		"body":    body,
		"headers": headers,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	// Async fetch inside Chromium (TLS fingerprint matches CF). AwaitPromise is required.
	// mode: cors + credentials from document origin (must be idlix after warmOrigin).
	js := fmt.Sprintf(`
		(async () => {
			const p = %s;
			const hdrs = Object.assign({}, p.headers || {});
			delete hdrs.origin;
			delete hdrs.Origin;
			const opts = {
				method: p.method || 'GET',
				headers: hdrs,
				credentials: 'include',
				mode: 'cors',
			};
			if (p.body && p.method && p.method.toUpperCase() !== 'GET' && p.method.toUpperCase() !== 'HEAD') {
				opts.body = p.body;
			}
			try {
				const r = await fetch(p.url, opts);
				const t = await r.text();
				return JSON.stringify({ status: r.status, ok: r.ok, text: t });
			} catch (e) {
				return JSON.stringify({ status: 0, ok: false, text: String(e && e.message ? e.message : e) });
			}
		})()
	`, string(payloadJSON))

	var raw string
	eval := chromedp.Evaluate(js, &raw, func(p *runtime.EvaluateParams) *runtime.EvaluateParams {
		return p.WithAwaitPromise(true)
	})
	if err := chromedp.Run(s.ctx, eval); err != nil {
		return nil, fmt.Errorf("browserFetch evaluate failed for %s: %w", reqURL, err)
	}
	if raw == "" {
		return nil, fmt.Errorf("browserFetch empty result for %s", reqURL)
	}

	var res browserFetchResult
	if err := json.Unmarshal([]byte(raw), &res); err != nil {
		return nil, fmt.Errorf("browserFetch bad response JSON: %w (raw=%q)", err, truncate(raw, 200))
	}
	return &res, nil
}

func (s *idlixBrowserSession) browserFetchJSON(reqURL, method, body string, headers map[string]string) (map[string]interface{}, *browserFetchResult, error) {
	res, err := s.browserFetch(reqURL, method, body, headers)
	if err != nil {
		return nil, nil, err
	}
	if !res.OK || res.Text == "" {
		return nil, res, fmt.Errorf("HTTP %d from %s: %s", res.Status, reqURL, truncate(res.Text, 160))
	}
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(res.Text), &data); err != nil {
		return nil, res, fmt.Errorf("JSON parse failed for %s (status %d): %w — body=%s",
			reqURL, res.Status, err, truncate(res.Text, 160))
	}
	return data, res, nil
}

// ── Movie / series chains ─────────────────────────────────────────────────────

func (s *idlixBrowserSession) scrapeMovie(p *idlixParsedURL) (*models.VideoMetadata, error) {
	// Step 1: GET /api/movies/{slug} → UUID
	apiURL := fmt.Sprintf("%s/api/movies/%s", p.BaseURL, p.Slug)
	log.Printf("[IDLIX] Step 1: GET %s", apiURL)

	data, res, err := s.browserFetchJSON(apiURL, "GET", "", map[string]string{
		"referer": p.Referer,
		"accept":  "application/json",
	})
	if err != nil {
		return nil, fmt.Errorf("IDLIX step 1 (movie UUID) failed: %w", err)
	}
	_ = res

	uuid := pickString(data, "id")
	if uuid == "" {
		if nested, ok := data["data"].(map[string]interface{}); ok {
			uuid = pickString(nested, "id")
		}
	}
	if uuid == "" {
		return nil, fmt.Errorf("IDLIX step 1: no content UUID in response for slug %q", p.Slug)
	}
	log.Printf("[IDLIX] Step 1 OK movie UUID=%s", uuid)

	// Step 2: track (best-effort)
	s.trackView("movie", uuid, p.Referer, "")

	// Steps 3–6
	return s.runStreamTail("movie", uuid, p.Referer, "movie/"+p.Slug)
}

func (s *idlixBrowserSession) scrapeSeries(p *idlixParsedURL) (*models.VideoMetadata, error) {
	// Step 1: GET /api/series/{slug}/season/{season}
	apiURL := fmt.Sprintf("%s/api/series/%s/season/%d", p.BaseURL, p.Slug, p.Season)
	log.Printf("[IDLIX] Step 1: GET %s", apiURL)

	data, _, err := s.browserFetchJSON(apiURL, "GET", "", map[string]string{
		"referer": p.Referer,
		"accept":  "application/json",
	})
	if err != nil {
		return nil, fmt.Errorf("IDLIX step 1 (series season) failed: %w", err)
	}

	seriesUUID := ""
	if series, ok := data["series"].(map[string]interface{}); ok {
		seriesUUID = pickString(series, "id")
	}
	episodeUUID := ""
	if season, ok := data["season"].(map[string]interface{}); ok {
		if eps, ok := season["episodes"].([]interface{}); ok {
			for _, raw := range eps {
				ep, ok := raw.(map[string]interface{})
				if !ok {
					continue
				}
				num := pickInt(ep, "episodeNumber")
				if num == p.Episode {
					episodeUUID = pickString(ep, "id")
					break
				}
			}
		}
	}
	if episodeUUID == "" {
		return nil, fmt.Errorf("IDLIX step 1: episode %d not found in season %d for slug %q", p.Episode, p.Season, p.Slug)
	}
	log.Printf("[IDLIX] Step 1 OK seriesUUID=%s episodeUUID=%s", seriesUUID, episodeUUID)

	// Step 2: track (best-effort)
	contentID := seriesUUID
	if contentID == "" {
		contentID = episodeUUID
	}
	s.trackView("tv_series", contentID, p.Referer, episodeUUID)

	// Steps 3–6
	return s.runStreamTail("episode", episodeUUID, p.Referer,
		fmt.Sprintf("series/%s/s%de%d", p.Slug, p.Season, p.Episode))
}

func (s *idlixBrowserSession) trackView(contentType, contentID, referer, episodeID string) {
	bodyMap := map[string]string{
		"contentType": contentType,
		"contentId":   contentID,
	}
	if episodeID != "" {
		bodyMap["episodeId"] = episodeID
	}
	bodyBytes, _ := json.Marshal(bodyMap)
	apiURL := s.baseURL + "/api/views/track"
	res, err := s.browserFetch(apiURL, "POST", string(bodyBytes), map[string]string{
		"content-type": "application/json",
		"referer":      referer,
	})
	if err != nil {
		log.Printf("[IDLIX] Step 2 track error (ignored): %v", err)
		return
	}
	log.Printf("[IDLIX] Step 2 views/track status=%d", res.Status)
}

func (s *idlixBrowserSession) runStreamTail(playInfoType, uuid, referer, label string) (*models.VideoMetadata, error) {
	// Step 3: play-info
	apiURL := fmt.Sprintf("%s/api/watch/play-info/%s/%s", s.baseURL, playInfoType, uuid)
	log.Printf("[IDLIX] Step 3: GET %s", apiURL)

	playInfo, _, err := s.browserFetchJSON(apiURL, "GET", "", map[string]string{
		"referer": referer,
		"accept":  "application/json",
	})
	if err != nil {
		return nil, fmt.Errorf("IDLIX step 3 (play-info) failed: %w", err)
	}

	kind := pickString(playInfo, "kind")
	if kind != "gate" {
		return nil, fmt.Errorf("IDLIX step 3: unexpected play-info kind %q (want gate)", kind)
	}
	gateToken := pickString(playInfo, "gateToken")
	if gateToken == "" {
		return nil, fmt.Errorf("IDLIX step 3: missing gateToken")
	}

	// Step 4: honor anti-scrape timer
	unlockAt := pickFloat(playInfo, "unlockAt")
	serverNow := pickFloat(playInfo, "serverNow")
	countdownMs := unlockAt - serverNow
	waitMs := countdownMs + 500
	if waitMs < 0 {
		waitMs = 0
	}
	if waitMs > float64(idlixGateMaxWait.Milliseconds()) {
		waitMs = float64(idlixGateMaxWait.Milliseconds())
	}
	if waitMs > 0 {
		log.Printf("[IDLIX] Step 4: waiting %.0fms for gate unlock (%s)", waitMs, label)
		select {
		case <-time.After(time.Duration(waitMs) * time.Millisecond):
		case <-s.ctx.Done():
			return nil, fmt.Errorf("IDLIX step 4: session cancelled during gate wait: %w", s.ctx.Err())
		}
	} else {
		log.Printf("[IDLIX] Step 4: gate already unlocked")
	}

	// Step 5: claim session
	claimURL := s.baseURL + "/api/watch/session/claim"
	claimBody, _ := json.Marshal(map[string]string{"gateToken": gateToken})
	log.Printf("[IDLIX] Step 5: POST %s", claimURL)

	// Same-origin POST: Origin is set by the browser after warmOrigin (do not forge).
	claimData, _, err := s.browserFetchJSON(claimURL, "POST", string(claimBody), map[string]string{
		"content-type":     "application/json",
		"referer":          referer,
		"x-requested-with": "XMLHttpRequest",
	})
	if err != nil {
		// One retry after re-warming origin (CF cookie may have expired mid-gate wait)
		log.Printf("[IDLIX] Step 5 claim failed (%v); re-warming origin and retrying once", err)
		if warmErr := s.warmOrigin(referer); warmErr != nil {
			return nil, fmt.Errorf("IDLIX step 5 (session claim) failed: %w (re-warm: %v)", err, warmErr)
		}
		claimData, _, err = s.browserFetchJSON(claimURL, "POST", string(claimBody), map[string]string{
			"content-type":     "application/json",
			"referer":          referer,
			"x-requested-with": "XMLHttpRequest",
		})
		if err != nil {
			return nil, fmt.Errorf("IDLIX step 5 (session claim) failed: %w", err)
		}
	}

	claim := pickString(claimData, "claim")
	redeemURL := pickString(claimData, "redeemUrl")
	if claim == "" || redeemURL == "" {
		return nil, fmt.Errorf("IDLIX step 5: missing claim or redeemUrl in response")
	}
	title := pickString(claimData, "title")
	log.Printf("[IDLIX] Step 5 OK redeemUrl=%s title=%q", redeemURL, title)

	// Allow majorplay redeem host + any CDN subdomain under that root immediately
	RegisterStreamURLs(redeemURL)

	// Step 6: majorplay redeem — normal HTTP, Content-Type text/plain
	playData, err := redeemMajorplay(redeemURL, claim, s.baseURL)
	if err != nil {
		return nil, fmt.Errorf("IDLIX step 6 (majorplay redeem) failed: %w", err)
	}

	streamURL := pickString(playData, "url")
	if streamURL == "" {
		return nil, fmt.Errorf("IDLIX step 6: no stream url in redeem response")
	}
	log.Printf("[IDLIX] Step 6 OK stream ready — %s", label)

	// Register actual media host (often g5.ruangskill.space / g5.akademivo.website / …)
	RegisterStreamURLs(streamURL)
	// Subtitle tracks may live on yet another CDN host
	if subs, ok := playData["subtitles"].([]interface{}); ok {
		for _, s := range subs {
			if m, ok := s.(map[string]interface{}); ok {
				for _, key := range []string{"path", "url", "file"} {
					if u, _ := m[key].(string); u != "" {
						RegisterStreamURLs(u)
					}
				}
			}
		}
	}

	meta := &models.VideoMetadata{
		VideoURL: streamURL,
		Title:    title,
		Source:   "idlix",
	}
	if thumb := pickString(claimData, "poster"); thumb != "" {
		meta.ThumbnailURL = thumb
		RegisterStreamURLs(thumb)
	}
	return meta, nil
}

// redeemMajorplay posts claim to majorplay.net with Content-Type text/plain
// (JSON body). This domain is NOT Cloudflare-protected.
func redeemMajorplay(redeemURL, claim, baseURL string) (map[string]interface{}, error) {
	body, _ := json.Marshal(map[string]string{"claim": claim})

	req, err := http.NewRequest(http.MethodPost, redeemURL, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	// Critical: text/plain (not application/json) — matches browser CORS simple request
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Origin", baseURL)
	req.Header.Set("Referer", baseURL+"/")
	req.Header.Set("User-Agent", idlixMobileUA)
	req.Header.Set("sec-ch-ua", `"Not/A)Brand";v="99", "Chromium";v="131"`)
	req.Header.Set("sec-ch-ua-mobile", "?1")
	req.Header.Set("sec-ch-ua-platform", `"Android"`)
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-site", "cross-site")

	// Dedicated client — do not route through scraper timeout that is too short for redeem
	client := &http.Client{Timeout: 25 * time.Second}
	// Reuse global transport (NanoProxy) if configured
	if httpClient.Transport != nil {
		client.Transport = httpClient.Transport
	}

	log.Printf("[IDLIX] Step 6: POST %s (text/plain)", redeemURL)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(string(raw), 160))
	}

	var data map[string]interface{}
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("JSON parse: %w — body=%s", err, truncate(string(raw), 160))
	}
	code := pickString(data, "code")
	if code != "" && code != "ok" {
		return nil, fmt.Errorf("unexpected redeem code %q", code)
	}
	return data, nil
}

// ── JSON helpers ──────────────────────────────────────────────────────────────

func pickString(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	default:
		return fmt.Sprintf("%v", t)
	}
}

func pickFloat(m map[string]interface{}, key string) float64 {
	if m == nil {
		return 0
	}
	v, ok := m[key]
	if !ok || v == nil {
		return 0
	}
	switch t := v.(type) {
	case float64:
		return t
	case float32:
		return float64(t)
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case json.Number:
		f, _ := t.Float64()
		return f
	case string:
		f, _ := strconv.ParseFloat(t, 64)
		return f
	default:
		return 0
	}
}

func pickInt(m map[string]interface{}, key string) int {
	return int(pickFloat(m, key))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func humanizeSlug(slug string) string {
	slug = strings.TrimSpace(slug)
	// drop trailing year: title-2026
	if m := regexp.MustCompile(`^(.*)-(\d{4})$`).FindStringSubmatch(slug); len(m) == 3 {
		slug = m[1]
	}
	parts := strings.Split(slug, "-")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, " ")
}
