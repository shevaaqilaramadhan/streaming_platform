package services

import (
	"strings"
	"testing"
)

func TestExtractSokujaEpisodeID(t *testing.T) {
	cases := []string{
		`{"episodeId":10112,"qualityOptions":["480p","720p"]}`,
		// Next.js RSC escaped form
		`\\\"$L3d\\\",null,{\\\"episodeId\\\":10112}],false`,
		`\"episodeId\":10112,\"qualityOptions\":[\"480p\"]`,
	}
	for _, html := range cases {
		if id := extractSokujaEpisodeID(html); id != 10112 {
			t.Fatalf("got %d for %q", id, html)
		}
	}
}

func TestSokujaQualityScore_Prefers720(t *testing.T) {
	if sokujaQualityScore("720p") <= sokujaQualityScore("1080p") {
		t.Fatal("720 should score higher than 1080 for faster start")
	}
	if sokujaQualityScore("720p") <= sokujaQualityScore("480p") {
		t.Fatal("720 should score higher than 480")
	}
}

func TestScrapeSokuja_Live(t *testing.T) {
	if testing.Short() {
		t.Skip("live network")
	}
	page := "https://x6.sokuja.uk/koori-no-jouheki-episode-10-subtitle-indonesia/"
	meta, err := scrapeSokuja(page)
	if err != nil {
		t.Fatalf("scrapeSokuja: %v", err)
	}
	if meta == nil || meta.VideoURL == "" {
		t.Fatal("empty result")
	}
	if !strings.Contains(meta.VideoURL, "storages.sokuja.uk") || !strings.Contains(meta.VideoURL, ".mp4") {
		t.Fatalf("unexpected video: %s", meta.VideoURL)
	}
	if strings.Contains(meta.VideoURL, "/sda/") || strings.HasSuffix(meta.VideoURL, ".gif") {
		t.Fatalf("picked ad asset: %s", meta.VideoURL)
	}
	t.Logf("title=%q video=%s", meta.Title, truncateURL(meta.VideoURL, 120))
}

func TestDetectSitePlatform_Sokuja(t *testing.T) {
	if got := detectSitePlatform("https://x6.sokuja.uk/foo"); got != "sokuja" {
		t.Fatalf("got %s", got)
	}
}
