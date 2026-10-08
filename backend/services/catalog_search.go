package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// SearchItem represents a normalized search result across supported providers.
type SearchItem struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Type          string   `json:"type"`          // "movie" | "series" | "anime"
	Provider      string   `json:"provider"`      // "idlix" | "samehadaku"
	Poster        string   `json:"poster"`
	Overview      string   `json:"overview,omitempty"`
	Year          string   `json:"year,omitempty"`
	Quality       string   `json:"quality,omitempty"`
	LatestEpisode string   `json:"latestEpisode,omitempty"`
	DetailURL     string   `json:"detailUrl"`     // Direct page URL to play or view
	Slug          string   `json:"slug,omitempty"`
	SeasonsCount  int      `json:"seasonsCount,omitempty"`
}

// CatalogEpisode represents an individual playable episode in a series.
type CatalogEpisode struct {
	EpisodeNumber int    `json:"episodeNumber"`
	Title         string `json:"title"`
	PageURL       string `json:"pageUrl"`
	Thumbnail     string `json:"thumbnail,omitempty"`
}

var searchHTTPClient = &http.Client{
	Timeout: 8 * time.Second,
}

// SearchAll queries multiple providers in parallel with timeouts and normalizes results.
func SearchAll(query, providerFilter string) []SearchItem {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}

	var results []SearchItem
	var mu sync.Mutex
	var wg sync.WaitGroup

	runSearch := func(name string, fn func(string) ([]SearchItem, error)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			items, err := fn(query)
			if err != nil {
				return
			}
			mu.Lock()
			results = append(results, items...)
			mu.Unlock()
		}()
	}

	filter := strings.ToLower(providerFilter)

	if filter == "" || filter == "all" || filter == "idlix" {
		runSearch("idlix", searchIDLIX)
	}
	if filter == "" || filter == "all" || filter == "samehadaku" {
		runSearch("samehadaku", searchSamehadaku)
	}

	wg.Wait()
	return results
}

// ── IDLIX Search Adapter ──────────────────────────────────────────────────────

func searchIDLIX(query string) ([]SearchItem, error) {
	apiURL := fmt.Sprintf("https://z2.idlixku.com/api/search?q=%s", url.QueryEscape(query))
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://z2.idlixku.com/")
	req.Header.Set("Accept", "application/json")

	resp, err := searchHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("IDLIX search status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return nil, err
	}

	var data struct {
		Results []struct {
			ID              string   `json:"id"`
			ContentType     string   `json:"contentType"`
			Title           string   `json:"title"`
			Overview        string   `json:"overview"`
			PosterPath      string   `json:"posterPath"`
			ReleaseDate     string   `json:"releaseDate"`
			FirstAirDate    string   `json:"firstAirDate"`
			Quality         string   `json:"quality"`
			Slug            string   `json:"slug"`
			NumberOfSeasons int      `json:"numberOfSeasons"`
		} `json:"results"`
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	var items []SearchItem
	for _, it := range data.Results {
		itemType := "movie"
		pageURL := fmt.Sprintf("https://z2.idlixku.com/movie/%s", it.Slug)
		year := it.ReleaseDate
		if it.ContentType == "tv_series" {
			itemType = "series"
			pageURL = fmt.Sprintf("https://z2.idlixku.com/series/%s", it.Slug)
			year = it.FirstAirDate
		}
		if len(year) > 4 {
			year = year[:4]
		}

		poster := ""
		if it.PosterPath != "" {
			if strings.HasPrefix(it.PosterPath, "http") {
				poster = it.PosterPath
			} else {
				poster = "https://image.tmdb.org/t/p/w500" + it.PosterPath
			}
		}

		items = append(items, SearchItem{
			ID:           "idlix:" + it.Slug,
			Title:        it.Title,
			Type:         itemType,
			Provider:     "IDLIX",
			Poster:       poster,
			Overview:     it.Overview,
			Year:         year,
			Quality:      it.Quality,
			DetailURL:    pageURL,
			Slug:         it.Slug,
			SeasonsCount: it.NumberOfSeasons,
		})
	}

	return items, nil
}

// ── Samehadaku Search Adapter ─────────────────────────────────────────────────

func searchSamehadaku(query string) ([]SearchItem, error) {
	searchURL := fmt.Sprintf("https://v2.samehadaku.how/?s=%s", url.QueryEscape(query))
	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://v2.samehadaku.how/")

	resp, err := searchHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Samehadaku search status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var items []SearchItem
	seen := make(map[string]bool)

	doc.Find("article.animepost").Each(func(_ int, s *goquery.Selection) {
		linkTag := s.Find("a").First()
		href, exists := linkTag.Attr("href")
		if !exists || href == "" || seen[href] {
			return
		}
		seen[href] = true

		title := strings.TrimSpace(linkTag.AttrOr("title", ""))
		if title == "" {
			title = strings.TrimSpace(s.Find(".title, h2, h3").First().Text())
		}
		if title == "" {
			return
		}

		imgTag := s.Find("img").First()
		imgSrc := imgTag.AttrOr("src", "")
		if imgSrc == "" {
			imgSrc = imgTag.AttrOr("data-src", "")
		}

		eps := strings.TrimSpace(s.Find(".type").First().Text())

		items = append(items, SearchItem{
			ID:            "samehadaku:" + href,
			Title:         title,
			Type:          "anime",
			Provider:      "Samehadaku",
			Poster:        imgSrc,
			LatestEpisode: eps,
			DetailURL:     href,
		})
	})

	return items, nil
}

// GetSeriesEpisodes returns episode list for a series from IDLIX or other providers.
func GetSeriesEpisodes(provider, slug string, season int) ([]CatalogEpisode, error) {
	if season <= 0 {
		season = 1
	}

	if strings.ToLower(provider) == "idlix" {
		apiURL := fmt.Sprintf("https://z2.idlixku.com/api/series/%s/season/%d", slug, season)
		req, err := http.NewRequest("GET", apiURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0.0.0 Safari/537.36")
		req.Header.Set("Referer", "https://z2.idlixku.com/")
		req.Header.Set("Accept", "application/json")

		resp, err := searchHTTPClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("IDLIX season episodes status %d", resp.StatusCode)
		}

		var data struct {
			Season struct {
				Episodes []struct {
					EpisodeNumber int    `json:"episodeNumber"`
					Name          string `json:"name"`
					StillPath     string `json:"stillPath"`
				} `json:"episodes"`
			} `json:"season"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			return nil, err
		}

		var epList []CatalogEpisode
		for _, ep := range data.Season.Episodes {
			thumb := ""
			if ep.StillPath != "" {
				thumb = "https://image.tmdb.org/t/p/w300" + ep.StillPath
			}
			epList = append(epList, CatalogEpisode{
				EpisodeNumber: ep.EpisodeNumber,
				Title:         ep.Name,
				PageURL:       fmt.Sprintf("https://z2.idlixku.com/series/%s/season/%d/episode/%d", slug, season, ep.EpisodeNumber),
				Thumbnail:     thumb,
			})
		}
		return epList, nil
	}

	return nil, fmt.Errorf("unsupported series provider %q", provider)
}
