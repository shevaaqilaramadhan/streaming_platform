package services

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"watchparty-backend/models"
)

// Samehadaku does NOT need headless for stream resolution.
// Player mirrors are loaded via WordPress admin-ajax:
//
//	POST /wp-admin/admin-ajax.php
//	action=player_ajax&post={id}&nume={n}&type=schtml
//
// Response is an <iframe src="..."> pointing at Blogspot / Wibufile / Mega / etc.
// Headless only sees ad iframes and times out — this AJAX path is the real fix.

var (
	samehaPlayerOptionRe = regexp.MustCompile(
		`(?is)class=["'][^"']*east_player_option[^"']*["'][^>]*data-post=["'](\d+)["'][^>]*data-nume=["'](\d+)["'][^>]*data-type=["']([^"']+)["'][^>]*>\s*<span>([^<]*)</span>`,
	)
	// Attribute order sometimes varies
	samehaPlayerOptionRe2 = regexp.MustCompile(
		`(?is)data-post=["'](\d+)["'][^>]*data-nume=["'](\d+)["'][^>]*data-type=["']([^"']+)["'][^>]*class=["'][^"']*east_player_option`,
	)
	samehaIframeSrcRe = regexp.MustCompile(`(?i)<iframe[^>]+src=["']([^"']+)["']`)
)

type samehaMirror struct {
	PostID int
	Nume   int
	Type   string
	Label  string
	Score  int // higher = preferred
}

func scoreSamehaMirror(label string) int {
	l := strings.ToLower(label)
	score := 0
	// Prefer direct progressive/host streams over blogger when possible
	if strings.Contains(l, "1080") {
		score += 300
	} else if strings.Contains(l, "720") {
		score += 200
	} else if strings.Contains(l, "480") {
		score += 100
	}
	if strings.Contains(l, "wibufile") {
		score += 50 // direct mp4 host — works well with proxy
	}
	if strings.Contains(l, "blogspot") || strings.Contains(l, "blogger") {
		score += 40
	}
	if strings.Contains(l, "vip") {
		score += 20
	}
	if strings.Contains(l, "mega") {
		score -= 10 // often download-only / not embed-friendly
	}
	return score
}

func parseSamehaPlayerOptions(html string) []samehaMirror {
	seen := make(map[string]bool)
	var out []samehaMirror

	add := func(post, nume, typ, label string) {
		pid, _ := strconv.Atoi(post)
		n, _ := strconv.Atoi(nume)
		if pid == 0 || n == 0 {
			return
		}
		key := fmt.Sprintf("%d:%d:%s", pid, n, typ)
		if seen[key] {
			return
		}
		seen[key] = true
		label = strings.TrimSpace(label)
		out = append(out, samehaMirror{
			PostID: pid,
			Nume:   n,
			Type:   typ,
			Label:  label,
			Score:  scoreSamehaMirror(label),
		})
	}

	for _, re := range []*regexp.Regexp{samehaPlayerOptionRe, samehaPlayerOptionRe2} {
		for _, m := range re.FindAllStringSubmatch(html, -1) {
			if len(m) >= 5 {
				add(m[1], m[2], m[3], m[4])
			}
		}
	}

	// goquery fallback
	if len(out) == 0 {
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
		if err == nil {
			doc.Find(".east_player_option, [class*='east_player_option']").Each(func(_ int, s *goquery.Selection) {
				post, _ := s.Attr("data-post")
				nume, _ := s.Attr("data-nume")
				typ, _ := s.Attr("data-type")
				label := strings.TrimSpace(s.Find("span").First().Text())
				if label == "" {
					label = strings.TrimSpace(s.Text())
				}
				add(post, nume, typ, label)
			})
		}
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return out[i].Nume < out[j].Nume
	})
	return out
}

func samehaAjaxURL(pageURL string) string {
	parsed, err := url.Parse(pageURL)
	if err != nil || parsed.Host == "" {
		return "https://v2.samehadaku.how/wp-admin/admin-ajax.php"
	}
	return parsed.Scheme + "://" + parsed.Host + "/wp-admin/admin-ajax.php"
}

// fetchSamehaPlayerEmbed POSTs player_ajax and returns iframe src (or raw body).
func fetchSamehaPlayerEmbed(ajaxURL, pageURL string, m samehaMirror) (string, error) {
	form := url.Values{}
	form.Set("action", "player_ajax")
	form.Set("post", strconv.Itoa(m.PostID))
	form.Set("nume", strconv.Itoa(m.Nume))
	form.Set("type", m.Type)
	if m.Type == "" {
		form.Set("type", "schtml")
	}

	req, err := http.NewRequest(http.MethodPost, ajaxURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", getRandomUserAgent())
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Referer", pageURL)
	req.Header.Set("Origin", originFromURL(pageURL))
	req.Header.Set("Accept", "*/*")

	client := &http.Client{Timeout: 20 * time.Second}
	if httpClient != nil && httpClient.Transport != nil {
		client.Transport = httpClient.Transport
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxScrapeBodyBytes))
	if err != nil {
		return "", err
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("HTTP %d from player_ajax", resp.StatusCode)
	}

	html := string(body)
	if match := samehaIframeSrcRe.FindStringSubmatch(html); len(match) > 1 {
		return strings.TrimSpace(match[1]), nil
	}
	// Sometimes returns bare URL
	html = strings.TrimSpace(html)
	if strings.HasPrefix(html, "http://") || strings.HasPrefix(html, "https://") {
		return html, nil
	}
	if stream := findStreamURLInHTML(html); stream != "" {
		return stream, nil
	}
	return "", fmt.Errorf("no iframe/src in player_ajax response for nume=%d", m.Nume)
}

func originFromURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

// resolveSamehaEmbed turns an embed iframe URL into a playable stream URL.
func resolveSamehaEmbed(embedURL, pageURL string) (string, error) {
	embedURL = strings.TrimSpace(embedURL)
	if embedURL == "" {
		return "", fmt.Errorf("empty embed")
	}
	// Protocol-relative
	if strings.HasPrefix(embedURL, "//") {
		embedURL = "https:" + embedURL
	}

	lower := strings.ToLower(embedURL)

	// Direct video file in iframe src (Wibufile style)
	if strings.HasSuffix(strings.Split(lower, "?")[0], ".mp4") ||
		strings.HasSuffix(strings.Split(lower, "?")[0], ".m3u8") ||
		strings.Contains(lower, ".mp4") && (strings.Contains(lower, "wibufile") || strings.Contains(lower, "video")) {
		return embedURL, nil
	}

	// Blogger video player
	if isBloggerVideoURL(embedURL) {
		return extractBloggerVideoURL(embedURL, pageURL)
	}

	// Generic: fetch embed page and search for stream
	html, err := fetchPageHTML(embedURL, pageURL)
	if err == nil {
		if s := findStreamURLInHTML(html); s != "" {
			return s, nil
		}
		if s := findStreamURLInScripts(html); s != "" {
			return s, nil
		}
		// Nested iframe
		if match := samehaIframeSrcRe.FindStringSubmatch(html); len(match) > 1 {
			nested := match[1]
			if strings.HasPrefix(nested, "//") {
				nested = "https:" + nested
			}
			if isBloggerVideoURL(nested) {
				return extractBloggerVideoURL(nested, embedURL)
			}
			if strings.Contains(strings.ToLower(nested), ".mp4") {
				return nested, nil
			}
		}
	}

	// If embed host itself is a known player, return embed for client/proxy try
	if highConfidenceStreamURL(embedURL) {
		return embedURL, nil
	}

	return "", fmt.Errorf("could not resolve embed: %s", truncateURL(embedURL, 80))
}

// scrapeSamehadaku resolves streams via direct embedded iframe or WP player_ajax.
func scrapeSamehadaku(pageURL string) (*models.VideoMetadata, error) {
	log.Printf("[Samehadaku] Scraping: %s", pageURL)

	html, err := fetchPageHTML(pageURL, "")
	if err != nil {
		// Fall back to generic then headless only if page fetch fails
		log.Printf("[Samehadaku] page fetch failed: %v — trying generic/headless", err)
		return scrapeSamehadakuFallback(pageURL)
	}

	// Layer 0: Check for direct iframe in page (e.g. Blogger / Blogspot embed already rendered)
	doc, docErr := goquery.NewDocumentFromReader(strings.NewReader(html))
	if docErr == nil {
		directIframeSrc := ""
		doc.Find(".player-embed iframe, #pembed iframe, #player-frame-wrapper iframe, .video-content iframe").Each(func(_ int, s *goquery.Selection) {
			if src, exists := s.Attr("src"); exists && src != "" && directIframeSrc == "" {
				directIframeSrc = normalizeURL(src, pageURL)
			}
		})
		if directIframeSrc == "" {
			if match := samehaIframeSrcRe.FindStringSubmatch(html); len(match) > 1 {
				directIframeSrc = normalizeURL(match[1], pageURL)
			}
		}

		if directIframeSrc != "" {
			log.Printf("[Samehadaku] Found direct iframe in page: %s", truncateURL(directIframeSrc, 100))
			streamURL, err := resolveSamehaEmbed(directIframeSrc, pageURL)
			if err == nil && isPlayableStreamURL(streamURL) {
				meta := extractSamehaMetadata(html, pageURL)
				meta.VideoURL = streamURL
				meta.Source = "samehadaku"
				RegisterStreamMetadata(meta.VideoURL, meta.ThumbnailURL)
				RegisterStreamURLs(directIframeSrc)
				log.Printf("[Samehadaku] SUCCESS via direct iframe → %s", truncateURL(streamURL, 100))
				return meta, nil
			}
		}
	}

	mirrors := parseSamehaPlayerOptions(html)
	if len(mirrors) == 0 {
		log.Printf("[Samehadaku] no east_player_option found — trying generic/headless")
		return scrapeSamehadakuFallback(pageURL)
	}

	log.Printf("[Samehadaku] found %d player mirrors (best first)", len(mirrors))
	for i, m := range mirrors {
		if i < 8 {
			log.Printf("[Samehadaku]   #%d post=%d nume=%d label=%q score=%d", i+1, m.PostID, m.Nume, m.Label, m.Score)
		}
	}

	ajaxURL := samehaAjaxURL(pageURL)
	var lastErr error
	// Try top mirrors until one resolves
	maxTry := 6
	if len(mirrors) < maxTry {
		maxTry = len(mirrors)
	}

	for i := 0; i < maxTry; i++ {
		m := mirrors[i]
		log.Printf("[Samehadaku] Trying mirror %q (nume=%d)…", m.Label, m.Nume)

		embed, err := fetchSamehaPlayerEmbed(ajaxURL, pageURL, m)
		if err != nil {
			lastErr = err
			log.Printf("[Samehadaku] player_ajax failed for nume=%d: %v", m.Nume, err)
			continue
		}
		log.Printf("[Samehadaku] embed: %s", truncateURL(embed, 100))

		streamURL, err := resolveSamehaEmbed(embed, pageURL)
		if err != nil {
			lastErr = err
			log.Printf("[Samehadaku] resolve embed failed: %v", err)
			// Still try using raw embed if it looks like media
			if strings.Contains(strings.ToLower(embed), ".mp4") || strings.Contains(strings.ToLower(embed), ".m3u8") {
				streamURL = embed
			} else {
				continue
			}
		}

		meta := extractSamehaMetadata(html, pageURL)
		meta.VideoURL = streamURL
		meta.Source = "samehadaku"
		if meta.Title == "" {
			meta.Title = m.Label
		}
		// Prefer showing quality label in episode field when useful
		if meta.Episode == "" && m.Label != "" {
			meta.Episode = m.Label
		}

		RegisterStreamMetadata(meta.VideoURL, meta.ThumbnailURL)
		RegisterStreamURLs(embed)
		log.Printf("[Samehadaku] SUCCESS via %q → %s", m.Label, truncateURL(streamURL, 100))
		return meta, nil
	}

	log.Printf("[Samehadaku] all AJAX mirrors failed (last=%v) — fallback generic/headless", lastErr)
	return scrapeSamehadakuFallback(pageURL)
}

func scrapeSamehadakuFallback(pageURL string) (*models.VideoMetadata, error) {
	result, err := scrapeGeneric(pageURL)
	if err == nil {
		result.Source = "samehadaku"
		RegisterStreamMetadata(result.VideoURL, result.ThumbnailURL)
		return result, nil
	}
	log.Printf("[Samehadaku] generic failed, headless last resort: %v", err)
	meta, err2 := scrapeWithHeadless(pageURL)
	if err2 == nil && meta != nil {
		meta.Source = "samehadaku"
		RegisterStreamMetadata(meta.VideoURL, meta.ThumbnailURL)
		return meta, nil
	}
	if last := err2; last != nil {
		return nil, fmt.Errorf("samehadaku scrape failed (ajax+generic+headless): %w", last)
	}
	return nil, fmt.Errorf("samehadaku scrape failed: %w", err)
}

func extractSamehaMetadata(html, pageURL string) *models.VideoMetadata {
	meta := &models.VideoMetadata{Source: "samehadaku"}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return meta
	}
	if t := strings.TrimSpace(doc.Find("h1.entry-title, h1").First().Text()); t != "" {
		meta.Title = t
	} else if og := doc.Find("meta[property='og:title']").AttrOr("content", ""); og != "" {
		meta.Title = og
	}
	if og := doc.Find("meta[property='og:image']").AttrOr("content", ""); og != "" {
		meta.ThumbnailURL = og
	}
	if ep := strings.TrimSpace(doc.Find("[itemprop='episodeNumber']").First().Text()); ep != "" {
		meta.Episode = "Episode " + ep
	}
	meta.NextEpisodeURL = extractNextEpisodeURL(doc, pageURL)
	return meta
}
