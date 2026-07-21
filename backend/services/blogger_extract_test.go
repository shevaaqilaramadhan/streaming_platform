package services

import (
	"strings"
	"testing"
)

func TestExtractBloggerToken(t *testing.T) {
	u := "https://www.blogger.com/video.g?token=AD6v5dw7z2Y5AOOuCZI_IKI2cPDSdX_quPehOuse64BSpNGJ3DV1A9nb9KCCW2CdB_-ONzVTOt4pN0kvZ4gcWTmZ1pD1UCsGiI7lyOlVN03Ikhfe-VBpPOqC91OTw91IPp4IMc5sE_U"
	tok := extractBloggerToken(u)
	if tok == "" || !strings.HasPrefix(tok, "AD6v5") {
		t.Fatalf("token=%q", tok)
	}
}

func TestUnescapeBloggerURL(t *testing.T) {
	raw := `https://rr1---sn-x.googlevideo.com/videoplayback?expire\u003d123\u0026itag\u003d22`
	got := unescapeBloggerURL(raw)
	if !strings.Contains(got, "expire=123") || !strings.Contains(got, "itag=22") {
		t.Fatalf("got %q", got)
	}
}

func TestPickBestBloggerStream_PrefersItag22(t *testing.T) {
	urls := []string{
		"https://rr1---sn-x.googlevideo.com/videoplayback?itag=18&id=a",
		"https://rr1---sn-x.googlevideo.com/videoplayback?itag=22&id=a",
	}
	best := pickBestBloggerStream(urls)
	if !strings.Contains(best, "itag=22") {
		t.Fatalf("expected itag 22, got %q", best)
	}
}

func TestFetchBloggerViaBatchexecute_Live(t *testing.T) {
	if testing.Short() {
		t.Skip("live network")
	}
	token := "AD6v5dw7z2Y5AOOuCZI_IKI2cPDSdX_quPehOuse64BSpNGJ3DV1A9nb9KCCW2CdB_-ONzVTOt4pN0kvZ4gcWTmZ1pD1UCsGiI7lyOlVN03Ikhfe-VBpPOqC91OTw91IPp4IMc5sE_U"
	url, err := fetchBloggerViaBatchexecute(token, "https://anoboy.si/")
	if err != nil {
		t.Fatalf("batchexecute: %v", err)
	}
	if !strings.Contains(url, "googlevideo.com/videoplayback") {
		t.Fatalf("unexpected stream URL: %s", url)
	}
	t.Logf("stream=%s", truncateURL(url, 120))
}

func TestScrapeAnoboy_Live(t *testing.T) {
	if testing.Short() {
		t.Skip("live network")
	}
	meta, err := scrapeAnoboy("https://anoboy.si/nippon-sangoku-episode-12-subtitle-indonesia/")
	if err != nil {
		t.Fatalf("scrapeAnoboy: %v", err)
	}
	if meta == nil || meta.VideoURL == "" {
		t.Fatal("empty video url")
	}
	if !strings.Contains(meta.VideoURL, "googlevideo.com") && !strings.Contains(meta.VideoURL, ".mp4") && !strings.Contains(meta.VideoURL, ".m3u8") {
		t.Fatalf("unexpected video url: %s", meta.VideoURL)
	}
	t.Logf("title=%q video=%s", meta.Title, truncateURL(meta.VideoURL, 120))
}

func TestUnescapeBloggerURL_DoubleEscaped(t *testing.T) {
	// Nested JSON often has \\u003d → after one pass becomes \=
	raw := `https://rr1---sn-x.googlevideo.com/videoplayback?expire\u003d123\u0026itag\u003d22`
	got := unescapeBloggerURL(raw)
	if strings.Contains(got, `\=`) || strings.Contains(got, `\&`) {
		t.Fatalf("still escaped: %q", got)
	}
	if !strings.Contains(got, "expire=123") || !strings.Contains(got, "itag=22") {
		t.Fatalf("got %q", got)
	}
	// Double-backslash form from batchexecute nested string
	raw2 := `https://rr1---sn-x.googlevideo.com/videoplayback?expire\\u003d123\\u0026ei\\u003dabc`
	got2 := unescapeBloggerURL(raw2)
	if strings.Contains(got2, `\`) {
		t.Fatalf("double-escape residual: %q", got2)
	}
	if !strings.Contains(got2, "expire=123") {
		t.Fatalf("got2 %q", got2)
	}
	// Residual form as seen in production logs: expire\= … \&ei\=
	raw3 := `https://rr2---sn-npoe7ndd.googlevideo.com/videoplayback?expire\=1784552192\&ei\=gKpdavTUB5LX2O8P0LWr`
	got3 := unescapeBloggerURL(raw3)
	if strings.Contains(got3, `\`) {
		t.Fatalf("residual backslash: %q", got3)
	}
	if !strings.Contains(got3, "expire=1784552192") || !strings.Contains(got3, "&ei=") {
		t.Fatalf("got3 %q", got3)
	}
}
