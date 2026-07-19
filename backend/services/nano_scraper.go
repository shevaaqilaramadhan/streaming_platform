package services

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// NanoProxy configuration — read from environment.
var (
	nanoProxyHTTP = os.Getenv("NANOPROXY_HTTP_URL") // e.g. "http://localhost:8180"
)

// NanoScraperResult holds the result from a nano scrape operation.
type NanoScraperResult struct {
	StreamURL  string
	PageTitle  string
	Thumbnail  string
	Source     string
	M3U8URLs   []string
	MP4URLs    []string
	IFrameURLs []string
}

var (
	scrapeSemaphore = make(chan struct{}, 3) // max 3 concurrent headless scrapes
	nanoTitleRe     = regexp.MustCompile(`meta\s+property=["']og:title["']\s+content=["']([^"']+)["']`)
	nanoTitleTagRe  = regexp.MustCompile(`<title[^>]*>([^<]+)</title>`)
	nanoImageRe     = regexp.MustCompile(`meta\s+property=["']og:image["']\s+content=["']([^"']+)["']`)
	chromePath      string // resolved Chrome/Chromium path
	chromeChecked   bool
	chromePathMu    sync.Mutex
)

// findChromePath returns the path to Chrome/Chromium binary.
// Checks CHROME_PATH env, then common binary names.
func findChromePath() string {
	chromePathMu.Lock()
	defer chromePathMu.Unlock()

	if chromeChecked {
		return chromePath
	}
	chromeChecked = true

	// Check CHROME_PATH environment variable first
	if envPath := os.Getenv("CHROME_PATH"); envPath != "" {
		if _, err := exec.LookPath(envPath); err == nil {
			chromePath = envPath
			log.Printf("[NanoScraper] Using Chrome from CHROME_PATH: %s", chromePath)
			return chromePath
		}
		log.Printf("[NanoScraper] CHROME_PATH=%s not found, trying defaults...", envPath)
	}

	// Try common Chrome/Chromium binary names
	candidates := []string{
		"chromium",
		"chromium-browser",
		"google-chrome",
		"google-chrome-stable",
		"/usr/bin/chromium",
		"/usr/bin/chromium-browser",
		"/usr/bin/google-chrome",
		"/snap/bin/chromium",
	}
	for _, name := range candidates {
		if _, err := exec.LookPath(name); err == nil {
			chromePath = name
			log.Printf("[NanoScraper] Found Chrome: %s", chromePath)
			return chromePath
		}
	}

	log.Printf("[NanoScraper] WARNING: Chrome/Chromium not found. Headless scraping disabled.")
	log.Printf("[NanoScraper] Install with: sudo pacman -S chromium  (Arch)")
	log.Printf("[NanoScraper]             sudo apt install chromium   (Debian/Ubuntu)")
	log.Printf("[NanoScraper] Or set CHROME_PATH=/path/to/chromium")
	return ""
}

// NanoScrapeWithPool wraps NanoScrape with a concurrency limiter.
func NanoScrapeWithPool(pageURL string) (*NanoScraperResult, error) {
	// Check Chrome availability first — fail fast if not installed
	if findChromePath() == "" {
		return nil, fmt.Errorf("Chrome/Chromium not installed — headless scraping unavailable")
	}

	select {
	case scrapeSemaphore <- struct{}{}:
		defer func() { <-scrapeSemaphore }()
		return NanoScrape(pageURL)
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("scrape queue full, too many concurrent requests")
	}
}

// NanoScrape uses stealth headless Chrome with network request interception
// to capture dynamically loaded stream URLs. Routes through NanoProxy when
// available. Does not rely on static DOM parsing for primary stream discovery.
func NanoScrape(pageURL string) (*NanoScraperResult, error) {
	// Session budget includes nav + stream wait; abort-on-match cancels Chromium early
	cap, renderedHTML, err := stealthNavigate(pageURL, defaultSessionTimeout)
	if err != nil && cap == nil {
		return nil, fmt.Errorf("nano scrape failed for %s: %w", pageURL, err)
	}
	if err != nil {
		// context.Canceled / deadline + non-empty stream = 100% SUCCESS (abort-on-match)
		if isAbortSuccess(err, cap) {
			log.Printf("[NanoScraper] Abort-on-match SUCCESS for %s: %v", pageURL, err)
			err = nil
		} else if cap.bestStream() != "" || cap.hasUsefulCapture() {
			log.Printf("[NanoScraper] Soft-success for %s after error (network capture hit): %v", pageURL, err)
			err = nil
		} else {
			log.Printf("[NanoScraper] Partial navigation error for %s: %v (using captured network data)", pageURL, err)
		}
	}

	m3u8URLs, mp4URLs, embedURLs, totalReqs := cap.snapshot()

	// Primary path: network-intercepted streams (not static DOM)
	streamURL := cap.bestStream()

	// Secondary: if only embed endpoints were seen, keep them for iframe follow-up
	iframeURLs := make([]string, 0)
	for _, src := range extractIframeSrcs(renderedHTML) {
		iframeURLs = append(iframeURLs, normalizeURL(src, pageURL))
	}
	for _, u := range embedURLs {
		if strings.Contains(strings.ToLower(u), "/embed/") && !nanoContains(iframeURLs, u) {
			iframeURLs = append(iframeURLs, u)
		}
	}

	// Fallback only if network capture found nothing: search rendered HTML / packer
	if streamURL == "" {
		log.Printf("[NanoScraper] No stream URLs in network requests, searching rendered HTML fallback...")
		if s := findStreamURLInHTML(renderedHTML); s != "" {
			streamURL = s
			if strings.Contains(s, ".m3u8") {
				m3u8URLs = append(m3u8URLs, s)
			} else {
				mp4URLs = append(mp4URLs, s)
			}
		}
		if streamURL == "" {
			if s := findStreamURLInScripts(renderedHTML); s != "" {
				streamURL = s
				if strings.Contains(s, ".m3u8") {
					m3u8URLs = append(m3u8URLs, s)
				} else {
					mp4URLs = append(mp4URLs, s)
				}
			}
		}
		if streamURL == "" {
			for _, script := range unpackPackerInHTML(renderedHTML) {
				if match := m3u8Regex.FindString(script); match != "" {
					streamURL = match
					m3u8URLs = append(m3u8URLs, match)
					break
				}
				if match := mp4Regex.FindString(script); match != "" {
					streamURL = match
					mp4URLs = append(mp4URLs, match)
					break
				}
			}
		}
	}

	// If still no direct stream but we have embed iframes, try resolving first embed via deep search
	if streamURL == "" && len(iframeURLs) > 0 {
		for _, iframe := range iframeURLs {
			if s, e := deepSearchStream(iframe, pageURL, 2); e == nil && s != "" {
				streamURL = s
				break
			}
		}
	}

	pageTitle := nanoExtractTitle(renderedHTML)
	ogImage := nanoExtractImage(renderedHTML)
	if pageTitle == "" && renderedHTML != "" {
		if doc, e := goquery.NewDocumentFromReader(strings.NewReader(renderedHTML)); e == nil {
			meta := extractGenericMetadata(doc, pageURL)
			pageTitle = meta.Title
			if ogImage == "" {
				ogImage = meta.ThumbnailURL
			}
		}
	}

	result := &NanoScraperResult{
		StreamURL:  streamURL,
		PageTitle:  pageTitle,
		Thumbnail:  ogImage,
		Source:     extractDomain(pageURL),
		M3U8URLs:   m3u8URLs,
		MP4URLs:    mp4URLs,
		IFrameURLs: iframeURLs,
	}

	log.Printf("[NanoScraper] Found %d m3u8, %d mp4, %d embeds, %d total requests for %s (stream=%v)",
		len(m3u8URLs), len(mp4URLs), len(embedURLs), totalReqs, pageURL, streamURL != "")

	if streamURL == "" {
		return result, fmt.Errorf("could not find stream URL from %s after stealth headless render", pageURL)
	}
	return result, nil
}

func nanoSelectBest(m3u8URLs, mp4URLs []string) string {
	if len(m3u8URLs) > 0 {
		best := m3u8URLs[0]
		for _, u := range m3u8URLs[1:] {
			if len(u) > len(best) {
				best = u
			}
		}
		return best
	}
	if len(mp4URLs) > 0 {
		for _, u := range mp4URLs {
			if len(u) > 50 {
				return u
			}
		}
		return mp4URLs[0]
	}
	return ""
}

func nanoContains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func nanoExtractTitle(html string) string {
	if match := nanoTitleRe.FindStringSubmatch(html); len(match) > 1 {
		return match[1]
	}
	if match := nanoTitleTagRe.FindStringSubmatch(html); len(match) > 1 {
		return match[1]
	}
	return ""
}

func nanoExtractImage(html string) string {
	if match := nanoImageRe.FindStringSubmatch(html); len(match) > 1 {
		return match[1]
	}
	return ""
}
