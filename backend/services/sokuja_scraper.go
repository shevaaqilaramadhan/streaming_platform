package services

import (
	"encoding/json"
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

	"watchparty-backend/models"
)

// Sokuja is a Next.js site. Streams are NOT in static HTML — the player loads
// progressive MP4s via:
//
//	GET {origin}/api/video-mirrors?e={episodeId}
//	→ {"mirrors":[{"embedUrl":"https://storages.sokuja.uk/...480p....mp4","quality":"480p"}, ...]}
//
// episodeId appears in RSC payloads as "episodeId":12345.
// Headless is unnecessary and was aborting on ad GIFs under /sda/.

var (
	// RSC payloads escape quotes as \" or \\\" — accept optional backslashes
	sokujaEpisodeIDRe = regexp.MustCompile(`\\*"episodeId\\*"\s*:\s*(\d+)`)
	sokujaEpisodeIDRe2 = regexp.MustCompile(`episodeId\\*"?\s*:\s*(\d+)`)
	sokujaTitleRe     = regexp.MustCompile(`(?i)<title[^>]*>([^<]+)</title>`)
	sokujaOGImageRe   = regexp.MustCompile(`(?i)property=["']og:image["']\s+content=["']([^"']+)["']`)
	sokujaOGTitleRe   = regexp.MustCompile(`(?i)property=["']og:title["']\s+content=["']([^"']+)["']`)
)

type sokujaMirror struct {
	ID         int    `json:"id"`
	ServerName string `json:"serverName"`
	EmbedURL   string `json:"embedUrl"`
	EmbedType  string `json:"embedType"`
	Quality    string `json:"quality"`
}

type sokujaMirrorsResp struct {
	Mirrors []sokujaMirror `json:"mirrors"`
}

func scrapeSokuja(pageURL string) (*models.VideoMetadata, error) {
	start := time.Now()
	parsed, err := url.Parse(pageURL)
	if err != nil || parsed.Host == "" {
		return nil, fmt.Errorf("invalid sokuja url")
	}
	origin := parsed.Scheme + "://" + parsed.Host

	html, err := fetchPageHTML(pageURL, "")
	if err != nil {
		return nil, fmt.Errorf("sokuja page fetch: %w", err)
	}

	epID := extractSokujaEpisodeID(html)
	if epID == 0 {
		return nil, fmt.Errorf("sokuja: episodeId not found in page")
	}

	mirrors, err := fetchSokujaMirrors(origin, epID, pageURL)
	if err != nil {
		return nil, err
	}
	if len(mirrors) == 0 {
		return nil, fmt.Errorf("sokuja: no mirrors for episode %d", epID)
	}

	// Prefer mid quality for faster start (720 > 480 > 1080 for load time balance)
	sort.SliceStable(mirrors, func(i, j int) bool {
		return sokujaQualityScore(mirrors[i].Quality) > sokujaQualityScore(mirrors[j].Quality)
	})

	best := mirrors[0]
	if best.EmbedURL == "" {
		return nil, fmt.Errorf("sokuja: empty embed url")
	}

	meta := extractSokujaMetadata(html, pageURL)
	meta.VideoURL = best.EmbedURL
	if meta.Episode == "" {
		meta.Episode = cleanEpisodeString(meta.Title)
	}

	// Register CDN for proxy
	RegisterStreamMetadata(meta.VideoURL, meta.ThumbnailURL)
	for _, m := range mirrors {
		RegisterStreamURLs(m.EmbedURL)
	}

	log.Printf("[Sokuja] OK episodeId=%d quality=%s mirrors=%d in %s",
		epID, best.Quality, len(mirrors), time.Since(start).Round(time.Millisecond))
	return meta, nil
}

func extractSokujaEpisodeID(html string) int {
	// Prefer occurrence near qualityOptions (player context) when present
	if m := regexp.MustCompile(`episodeId\\*"?\s*:\s*(\d+)[^}]{0,40}qualityOptions`).FindStringSubmatch(html); len(m) > 1 {
		n, _ := strconv.Atoi(m[1])
		if n > 0 {
			return n
		}
	}
	if m := sokujaEpisodeIDRe.FindStringSubmatch(html); len(m) > 1 {
		n, _ := strconv.Atoi(m[1])
		if n > 0 {
			return n
		}
	}
	if m := sokujaEpisodeIDRe2.FindStringSubmatch(html); len(m) > 1 {
		n, _ := strconv.Atoi(m[1])
		return n
	}
	return 0
}

func fetchSokujaMirrors(origin string, episodeID int, referer string) ([]sokujaMirror, error) {
	apiURL := fmt.Sprintf("%s/api/video-mirrors?e=%d", strings.TrimRight(origin, "/"), episodeID)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", getRandomUserAgent())
	req.Header.Set("Accept", "application/json")
	if referer != "" {
		req.Header.Set("Referer", referer)
	} else {
		req.Header.Set("Referer", origin+"/")
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sokuja mirrors: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("sokuja mirrors HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	var out sokujaMirrorsResp
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("sokuja mirrors json: %w", err)
	}
	// Keep only playable progressive/hls embeds
	var playable []sokujaMirror
	for _, m := range out.Mirrors {
		u := strings.TrimSpace(m.EmbedURL)
		if u == "" {
			continue
		}
		lu := strings.ToLower(u)
		if strings.Contains(lu, "/sda/") || strings.HasSuffix(lu, ".gif") {
			continue
		}
		if strings.Contains(lu, ".mp4") || strings.Contains(lu, ".m3u8") ||
			strings.EqualFold(m.EmbedType, "mp4") || strings.EqualFold(m.EmbedType, "hls") {
			playable = append(playable, m)
		}
	}
	return playable, nil
}

func sokujaQualityScore(q string) int {
	l := strings.ToLower(q)
	switch {
	case strings.Contains(l, "720"):
		return 300 // best balance: quality vs start buffer
	case strings.Contains(l, "480"):
		return 200
	case strings.Contains(l, "1080"):
		return 150 // larger file → slower first paint via proxy
	case strings.Contains(l, "360"):
		return 100
	default:
		return 50
	}
}

func extractSokujaMetadata(html, pageURL string) *models.VideoMetadata {
	meta := &models.VideoMetadata{Source: extractDomain(pageURL)}
	if m := sokujaOGTitleRe.FindStringSubmatch(html); len(m) > 1 {
		meta.Title = strings.TrimSpace(m[1])
	}
	if meta.Title == "" {
		if m := sokujaTitleRe.FindStringSubmatch(html); len(m) > 1 {
			t := strings.TrimSpace(m[1])
			t = strings.TrimSuffix(t, " | SOKUJA")
			t = strings.TrimSuffix(t, " | Sokuja")
			meta.Title = strings.TrimSpace(t)
		}
	}
	if m := sokujaOGImageRe.FindStringSubmatch(html); len(m) > 1 {
		meta.ThumbnailURL = m[1]
	}
	meta.Episode = cleanEpisodeString(meta.Title)
	return meta
}
