package services

import (
	"strings"
	"testing"
)

func TestParseIdlixURL_Movie(t *testing.T) {
	cases := []struct {
		raw      string
		slug     string
		baseHost string
	}{
		{
			raw:      "https://z2.idlixku.com/movie/per-aspera-ad-astra-2026",
			slug:     "per-aspera-ad-astra-2026",
			baseHost: "z2.idlixku.com",
		},
		{
			raw:      "https://z2.idlixku.com/movie/per-aspera-ad-astra-2026?play=1",
			slug:     "per-aspera-ad-astra-2026",
			baseHost: "z2.idlixku.com",
		},
		{
			raw:      "https://tv.idlixofficial.co/movie/some-title-2024/",
			slug:     "some-title-2024",
			baseHost: "tv.idlixofficial.co",
		},
	}

	for _, tc := range cases {
		p, err := parseIdlixURL(tc.raw)
		if err != nil {
			t.Fatalf("parseIdlixURL(%q): %v", tc.raw, err)
		}
		if p.Kind != "movie" {
			t.Errorf("kind=%q want movie for %s", p.Kind, tc.raw)
		}
		if p.Slug != tc.slug {
			t.Errorf("slug=%q want %q for %s", p.Slug, tc.slug, tc.raw)
		}
		if !strings.Contains(p.BaseURL, tc.baseHost) {
			t.Errorf("baseURL=%q want host %s", p.BaseURL, tc.baseHost)
		}
		if !strings.Contains(p.Referer, "/movie/"+tc.slug) {
			t.Errorf("referer=%q missing movie path", p.Referer)
		}
	}
}

func TestParseIdlixURL_Series(t *testing.T) {
	cases := []struct {
		raw     string
		slug    string
		season  int
		episode int
	}{
		{
			raw:     "https://z2.idlixku.com/series/oasis-2026/season/1/episode/1",
			slug:    "oasis-2026",
			season:  1,
			episode: 1,
		},
		{
			raw:     "https://z2.idlixku.com/series/foo-bar/season/2/episode/12",
			slug:    "foo-bar",
			season:  2,
			episode: 12,
		},
		{
			// season/episode missing → defaults 1/1
			raw:     "https://z2.idlixku.com/series/only-slug",
			slug:    "only-slug",
			season:  1,
			episode: 1,
		},
	}

	for _, tc := range cases {
		p, err := parseIdlixURL(tc.raw)
		if err != nil {
			t.Fatalf("parseIdlixURL(%q): %v", tc.raw, err)
		}
		if p.Kind != "series" {
			t.Errorf("kind=%q want series for %s", p.Kind, tc.raw)
		}
		if p.Slug != tc.slug {
			t.Errorf("slug=%q want %q", p.Slug, tc.slug)
		}
		if p.Season != tc.season || p.Episode != tc.episode {
			t.Errorf("S%dE%d want S%dE%d for %s", p.Season, p.Episode, tc.season, tc.episode, tc.raw)
		}
	}
}

func TestParseIdlixURL_Invalid(t *testing.T) {
	bad := []string{
		"",
		"https://z2.idlixku.com/",
		"https://z2.idlixku.com/genre/action",
		"not-a-url",
	}
	for _, raw := range bad {
		if _, err := parseIdlixURL(raw); err == nil {
			t.Errorf("expected error for %q", raw)
		}
	}
}

func TestDetectSitePlatform_Idlix(t *testing.T) {
	if got := detectSitePlatform("https://z2.idlixku.com/movie/x"); got != "idlix" {
		t.Fatalf("got %q want idlix", got)
	}
	if got := detectSitePlatform("https://tv.idlixofficial.co/series/y"); got != "idlix" {
		t.Fatalf("got %q want idlix", got)
	}
}

func TestHumanizeSlug(t *testing.T) {
	if got := humanizeSlug("per-aspera-ad-astra-2026"); got != "Per Aspera Ad Astra" {
		t.Fatalf("got %q", got)
	}
}

func TestPickHelpers(t *testing.T) {
	m := map[string]interface{}{
		"id":            "abc-uuid",
		"unlockAt":      float64(1_700_000_015_000),
		"serverNow":     float64(1_700_000_000_000),
		"episodeNumber": float64(3),
	}
	if pickString(m, "id") != "abc-uuid" {
		t.Fatal("pickString")
	}
	if pickInt(m, "episodeNumber") != 3 {
		t.Fatal("pickInt")
	}
	if pickFloat(m, "unlockAt")-pickFloat(m, "serverNow") != 15000 {
		t.Fatal("pickFloat countdown")
	}
}
