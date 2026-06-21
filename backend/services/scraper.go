package services

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
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

var (
	httpClient = &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}
	m3u8Regex   = regexp.MustCompile(`https?://[^\s"'<>]+?\.m3u8[^\s"'<>]*`)
	mp4Regex    = regexp.MustCompile(`https?://[^\s"'<>]+?\.mp4[^\s"'<>]*`)
	actionRegex = regexp.MustCompile(`action:\s*["']([a-f0-9]{32})["']`)
	nonceRegex  = regexp.MustCompile(`data:\s*\{\s*action:\s*["']([a-f0-9]{32})["']\s*\}`)
	iframeRegex = regexp.MustCompile(`<iframe[^>]+src="([^"]+)"`)
	packerRegex = regexp.MustCompile(`(?s)eval\(function\(p,a,c,k,e,d\).*?\}\('(.*?)',\s*(\d+),\s*(\d+),\s*'(.*?)'\.split\('\|'\)`)
)

func ScrapeStreamURL(pageURL string) (*models.VideoMetadata, error) {
	if strings.Contains(pageURL, "otakudesu") {
		return scrapeOtakudesu(pageURL)
	}
	return scrapeGeneric(pageURL)
}

type mirror struct {
	host    string
	payload string
	score   int
}

func scrapeOtakudesu(pageURL string) (*models.VideoMetadata, error) {
	doc, err := fetchPageDocument(pageURL, "")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Otakudesu page: %w", err)
	}

	html, _ := doc.Html()

	allActions := actionRegex.FindAllStringSubmatch(html, -1)
	if len(allActions) < 2 {
		return nil, fmt.Errorf("could not find WordPress AJAX action IDs in page source")
	}

	var nonceAction string
	nonceMatch := nonceRegex.FindStringSubmatch(html)
	if len(nonceMatch) > 1 {
		nonceAction = nonceMatch[1]
	} else {
		nonceAction = allActions[1][1]
	}

	var iframeAction string
	for _, match := range allActions {
		if match[1] != nonceAction {
			iframeAction = match[1]
			break
		}
	}
	if iframeAction == "" {
		iframeAction = allActions[0][1]
	}

	var mirrors []mirror
	doc.Find(".mirrorstream a").Each(func(_ int, s *goquery.Selection) {
		b64Payload, exists := s.Attr("data-content")
		if !exists || b64Payload == "" {
			return
		}
		hostName := strings.TrimSpace(strings.ToLower(s.Text()))
		
		decoded, err := base64.StdEncoding.DecodeString(b64Payload)
		if err != nil {
			return
		}
		var pl struct {
			Q string `json:"q"`
		}
		json.Unmarshal(decoded, &pl)
		quality := pl.Q

		score := 0
		if strings.Contains(quality, "720p") {
			score += 100
		} else if strings.Contains(quality, "480p") {
			score += 50
		} else if strings.Contains(quality, "360p") {
			score += 10
		}

		if strings.Contains(hostName, "vidhide") {
			score += 5
		} else if strings.Contains(hostName, "filedon") {
			score += 4
		} else if strings.Contains(hostName, "mega") {
			score += 1
		}

		mirrors = append(mirrors, mirror{
			host:    hostName,
			payload: b64Payload,
			score:   score,
		})
	})

	sort.Slice(mirrors, func(i, j int) bool {
		return mirrors[i].score > mirrors[j].score
	})

	if len(mirrors) == 0 {
		return nil, fmt.Errorf("no mirrors found on episode page")
	}

	ajaxURL := "https://otakudesu.blog/wp-admin/admin-ajax.php"
	nonce, err := fetchNonce(ajaxURL, nonceAction, pageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch nonce: %w", err)
	}

	var lastErr error
	for _, m := range mirrors {
		iframeSrc, err := fetchMirrorIframeSrc(ajaxURL, m.payload, nonce, iframeAction, pageURL)
		if err != nil {
			lastErr = err
			continue
		}

		embedHTML, err := fetchPageHTML(iframeSrc, pageURL)
		if err != nil {
			lastErr = err
			continue
		}

		if streamURL := findStreamURLInHTML(embedHTML); streamURL != "" {
			metadata := extractOtakudesuMetadata(doc, pageURL)
			metadata.VideoURL = streamURL
			return metadata, nil
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("failed to resolve stream from mirrors: %w", lastErr)
	}
	return nil, fmt.Errorf("failed to resolve stream from any mirror")
}

func fetchNonce(ajaxURL, action, referer string) (string, error) {
	data := url.Values{}
	data.Set("action", action)

	req, err := http.NewRequest("POST", ajaxURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Referer", referer)

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("nonce request HTTP %d", resp.StatusCode)
	}

	var result struct {
		Data string `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.Data, nil
}

func fetchMirrorIframeSrc(ajaxURL, b64Payload, nonce, action, referer string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(b64Payload)
	if err != nil {
		return "", err
	}
	var payloadMap map[string]interface{}
	if err := json.Unmarshal(decoded, &payloadMap); err != nil {
		return "", err
	}

	data := url.Values{}
	for k, v := range payloadMap {
		data.Set(k, fmt.Sprintf("%v", v))
	}
	data.Set("nonce", nonce)
	data.Set("action", action)

	req, err := http.NewRequest("POST", ajaxURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Referer", referer)

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("iframe request HTTP %d", resp.StatusCode)
	}

	var result struct {
		Data string `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	decodedHTML, err := base64.StdEncoding.DecodeString(result.Data)
	if err != nil {
		return "", err
	}

	matches := iframeRegex.FindStringSubmatch(string(decodedHTML))
	if len(matches) < 2 {
		return "", fmt.Errorf("no iframe tag found in response HTML")
	}

	src := matches[1]
	if strings.HasPrefix(src, "//") {
		src = "https:" + src
	}
	return src, nil
}

func getWord(number int, base int) string {
	const alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	if number < base {
		return string(alphabet[number])
	}
	return getWord(number/base, base) + string(alphabet[number%base])
}

func unpack(p string, a int, c int, k []string) string {
	for i := c - 1; i >= 0; i-- {
		if i < len(k) && k[i] != "" {
			word := k[i]
			baseARep := getWord(i, a)
			re := regexp.MustCompile(`\b` + regexp.QuoteMeta(baseARep) + `\b`)
			p = re.ReplaceAllString(p, word)
		}
	}
	return p
}

func unpackPackerInHTML(html string) []string {
	matches := packerRegex.FindAllStringSubmatch(html, -1)
	var unpacked []string
	for _, match := range matches {
		if len(match) < 5 {
			continue
		}
		p := match[1]
		a, err1 := strconv.Atoi(match[2])
		c, err2 := strconv.Atoi(match[3])
		k := strings.Split(match[4], "|")
		if err1 != nil || err2 != nil {
			continue
		}
		
		p = strings.ReplaceAll(p, `\'`, `'`)
		p = strings.ReplaceAll(p, `\"`, `"`)
		
		unpacked = append(unpacked, unpack(p, a, c, k))
	}
	return unpacked
}

func scrapeGeneric(pageURL string) (*models.VideoMetadata, error) {
	doc, err := fetchPageDocument(pageURL, "")
	if err != nil {
		return nil, err
	}

	var iframeSrcs []string
	doc.Find("iframe").Each(func(_ int, s *goquery.Selection) {
		if src, exists := s.Attr("src"); exists && src != "" {
			if strings.HasPrefix(src, "//") {
				src = "https:" + src
			}
			iframeSrcs = append(iframeSrcs, src)
		}
	})

	html, _ := doc.Html()
	if streamURL := findStreamURLInHTML(html); streamURL != "" {
		metadata := extractGenericMetadata(doc, pageURL)
		metadata.VideoURL = streamURL
		return metadata, nil
	}

	for _, iframeSrc := range iframeSrcs {
		iframeHTML, err := fetchPageHTML(iframeSrc, pageURL)
		if err != nil {
			continue
		}

		if streamURL := findStreamURLInHTML(iframeHTML); streamURL != "" {
			metadata := extractGenericMetadata(doc, pageURL)
			metadata.VideoURL = streamURL
			return metadata, nil
		}
	}

	return nil, fmt.Errorf("could not find .m3u8 stream in page or its iframes")
}

func fetchPageDocument(url, referer string) (*goquery.Document, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	if referer != "" {
		req.Header.Set("Referer", referer)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	return goquery.NewDocumentFromReader(resp.Body)
}

func fetchPageHTML(url, referer string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	if referer != "" {
		req.Header.Set("Referer", referer)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(bodyBytes), nil
}

func findStreamURLInHTML(html string) string {
	if match := m3u8Regex.FindString(html); match != "" {
		return match
	}

	unpacked := unpackPackerInHTML(html)
	for _, script := range unpacked {
		if match := m3u8Regex.FindString(script); match != "" {
			return match
		}
		if match := mp4Regex.FindString(script); match != "" {
			return match
		}
	}

	if match := mp4Regex.FindString(html); match != "" {
		return match
	}

	return ""
}

func extractDomain(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return parsed.Host
}

func cleanEpisodeString(raw string) string {
	raw = strings.TrimSpace(raw)
	lower := strings.ToLower(raw)
	
	if idx := strings.Index(lower, "subtitle"); idx != -1 {
		raw = strings.TrimSpace(raw[:idx])
	}
	if idx := strings.Index(lower, "sub indo"); idx != -1 {
		raw = strings.TrimSpace(raw[:idx])
	}
	
	if !strings.Contains(strings.ToLower(raw), "episode") {
		re := regexp.MustCompile(`(?i)(?:ep\.?\s*|episode\s*)(\d+)`)
		if match := re.FindStringSubmatch(raw); len(match) > 1 {
			return "Episode " + match[1]
		}
	}
	
	return raw
}

func extractOtakudesuMetadata(doc *goquery.Document, pageURL string) *models.VideoMetadata {
	metadata := &models.VideoMetadata{Source: "otakudesu.blog"}
	
	title := doc.Find(".venutama .post-title h1").First().Text()
	if title == "" {
		title = doc.Find(".fotoanime").Next().Find("h1").First().Text()
	}
	if title == "" {
		title = doc.Find(".entry-content h1").First().Text()
	}
	metadata.Title = strings.TrimSpace(title)
	
	episodeText := doc.Find(".venutama .epztitle").First().Text()
	if episodeText == "" {
		episodeText = doc.Find("h1.entry-title").First().Text()
	}
	if episodeText != "" {
		metadata.Episode = cleanEpisodeString(episodeText)
	}
	
	thumbnail, exists := doc.Find(".fotoanime img").First().Attr("src")
	if exists && thumbnail != "" {
		if strings.HasPrefix(thumbnail, "//") {
			thumbnail = "https:" + thumbnail
		}
		metadata.ThumbnailURL = thumbnail
	}
	
	if metadata.Title == "" {
		ogTitle := doc.Find("meta[property='og:title']").AttrOr("content", "")
		if ogTitle != "" {
			metadata.Title = ogTitle
		} else {
			metadata.Title = doc.Find("title").First().Text()
		}
	}
	
	return metadata
}

func extractGenericMetadata(doc *goquery.Document, pageURL string) *models.VideoMetadata {
	metadata := &models.VideoMetadata{Source: extractDomain(pageURL)}
	
	ogTitle := doc.Find("meta[property='og:title']").AttrOr("content", "")
	if ogTitle != "" {
		metadata.Title = ogTitle
	} else {
		metadata.Title = doc.Find("title").First().Text()
	}
	
	ogImage := doc.Find("meta[property='og:image']").AttrOr("content", "")
	if ogImage != "" {
		if strings.HasPrefix(ogImage, "//") {
			ogImage = "https:" + ogImage
		}
		metadata.ThumbnailURL = ogImage
	}
	
	return metadata
}
