# Backend Scraper Task — WatchParty Go Backend (Updated)

> **For the backend agent**: This document specifies the updated requirements for the lightweight scraper service in the Go backend.
> The frontend is already fully integrated to support the resolved stream URLs.
> Read this document to implement the scraper fixes for `otakudesu.blog` and similar sites.

---

## 1. Overview & Current Issue
The previous static scraper failed on `otakudesu.blog` because the default player uses Blogger, which does not expose direct `.m3u8` or `.mp4` URLs. However, Otakudesu provides high-quality mirror streams (like **Vidhide** and **Filedon**) which serve standard HLS `.m3u8` streams. These streams are protected using **Dean Edwards Packer** obfuscation.

To make the scraper work 100% of the time, the backend agent needs to implement:
1. **Otakudesu Mirror Resolver**: Extract mirror endpoints, call WordPress admin-ajax to fetch embed players, and fallback automatically from high-quality (720p) to low-quality (360p).
2. **Dean Edwards Packer Unpacker in Go**: Decode packed script blocks inside mirror player pages to extract the underlying HLS stream URLs.
3. **General Fallback Scraper**: Keep the generic iframe parsing fallback, but with Packer unpacking enabled so that general sites embedding Packer-protected streams are automatically supported.

---

## 2. Updated File Structure
No new external dependencies are required. Use Go's standard library and `github.com/PuerkitoBio/goquery`.

Modify/create:
- `backend/services/scraper.go` — Update with the full implementation below.
- `backend/services/message_handler.go` — Integrate the updated scraper.

---

## 3. Scraper Implementation (`backend/services/scraper.go`)

Replace the contents of `backend/services/scraper.go` with the following Go implementation:

```go
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

// ScrapeStreamURL is the main entry point to resolve direct streams.
func ScrapeStreamURL(pageURL string) (string, error) {
	if strings.Contains(pageURL, "otakudesu") {
		return scrapeOtakudesu(pageURL)
	}
	return scrapeGeneric(pageURL)
}

// ==========================================
// 1. OTAKUDESU MIRROR RESOLVING LOGIC
// ==========================================

type mirror struct {
	host    string
	payload string
	score   int
}

func scrapeOtakudesu(pageURL string) (string, error) {
	doc, err := fetchPageDocument(pageURL, "")
	if err != nil {
		return "", fmt.Errorf("failed to fetch Otakudesu page: %w", err)
	}

	html, _ := doc.Html()

	// Step A: Extract WordPress AJAX Action IDs
	allActions := actionRegex.FindAllStringSubmatch(html, -1)
	if len(allActions) < 2 {
		return "", fmt.Errorf("could not find WordPress AJAX action IDs in page source")
	}

	var nonceAction string
	nonceMatch := nonceRegex.FindStringSubmatch(html)
	if len(nonceMatch) > 1 {
		nonceAction = nonceMatch[1]
	} else {
		nonceAction = allActions[1][1] // fallback
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

	// Step B: Parse all mirror links from the page
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
		return "", fmt.Errorf("no mirrors found on episode page")
	}

	// Step C: Fetch the WordPress admin AJAX Nonce
	ajaxURL := "https://otakudesu.blog/wp-admin/admin-ajax.php"
	nonce, err := fetchNonce(ajaxURL, nonceAction, pageURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch nonce: %w", err)
	}

	// Step D: Iterate mirrors (highest quality first)
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
			return streamURL, nil
		}
	}

	if lastErr != nil {
		return "", fmt.Errorf("failed to resolve stream from mirrors: %w", lastErr)
	}
	return "", fmt.Errorf("failed to resolve stream from any mirror")
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

// ==========================================
// 2. DEAN EDWARDS PACKER UNPACKER LOGIC
// ==========================================

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

// ==========================================
// 3. GENERIC FALLBACK SCRAPER LOGIC
// ==========================================

func scrapeGeneric(pageURL string) (string, error) {
	doc, err := fetchPageDocument(pageURL, "")
	if err != nil {
		return "", err
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
		return streamURL, nil
	}

	for _, iframeSrc := range iframeSrcs {
		iframeHTML, err := fetchPageHTML(iframeSrc, pageURL)
		if err != nil {
			continue
		}

		if streamURL := findStreamURLInHTML(iframeHTML); streamURL != "" {
			return streamURL, nil
		}
	}

	return "", fmt.Errorf("could not find .m3u8 stream in page or its iframes")
}

// ==========================================
// HELPERS
// ==========================================

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
```

---

## 4. Verification Check
Ask the backend agent to verify that the app successfully builds and runs:
```bash
cd backend
go run main.go
```
Verify pasting `https://otakudesu.blog/episode/gcms-episode-8-sub-indo/` resolves properly and broadcasts the stream.
