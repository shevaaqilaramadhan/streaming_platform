package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"watchparty-backend/services"
)

func TestValidateProxyURL_BlocksSSRF(t *testing.T) {
	blocked := []string{
		"http://127.0.0.1:8080/api/rooms",
		"http://localhost:8080/",
		"http://[::1]/",
		"http://169.254.169.254/latest/meta-data/",
		"http://192.168.1.1/",
		"http://10.0.0.1/",
		"https://evil.example.com/video.m3u8",
		"https://random-site.org/file.mp4",
	}
	for _, u := range blocked {
		if err := validateProxyURL(u); err == nil {
			t.Errorf("expected block for %s", u)
		}
	}
}

func TestValidateProxyURL_AllowsCDN(t *testing.T) {
	allowed := []string{
		"https://cdn.vidhidepro.com/stream/master.m3u8",
		"https://otakudesu.blog/episode/test",
		"https://anoboy.si/anime/x",
		"https://filedon.com/e/abc",
		"https://blogger.com/video.g?token=x",
		"https://z2.idlixku.com/movie/x",
		"https://majorplay.net/api/play",
		"https://cdn.majorplay.net/hls/master.m3u8",
		"https://g5.ruangskill.space/hls/master.m3u8",
		"https://cdn.ruangskill.space/segment.ts",
		"https://x.pancal.space/video.m3u8",
		"https://image.tmdb.org/t/p/w500/poster.jpg",
	}
	for _, u := range allowed {
		if err := validateProxyURL(u); err != nil {
			t.Errorf("expected allow for %s: %v", u, err)
		}
	}
}

func TestValidateProxyURL_RejectsGenericSubstring(t *testing.T) {
	// Old contains-match would allow these via tokens like "anime", "cdn.", "m3u8", "plyr"
	blocked := []string{
		"https://not-anime-evil.com/x.m3u8",
		"https://evil-cdn.example.com/v.mp4",
		"https://plyr-attacker.net/stream",
		"https://m3u8.attacker.example/master.m3u8",
	}
	for _, u := range blocked {
		if err := validateProxyURL(u); err == nil {
			t.Errorf("expected block for overly-broad host %s", u)
		}
	}
}

func TestHandleStreamProxy_SSRFReturns403(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/proxy?url=http://127.0.0.1:1/", nil)
	req.RemoteAddr = "203.0.113.10:12345"
	rr := httptest.NewRecorder()
	HandleStreamProxy(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "Forbidden") {
		t.Fatalf("body = %q", rr.Body.String())
	}
}

func TestHandleStreamProxy_NotAllowlisted403(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/proxy?url=https://example.com/video.m3u8", nil)
	req.RemoteAddr = "203.0.113.11:12345"
	rr := httptest.NewRecorder()
	HandleStreamProxy(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body=%s", rr.Code, rr.Body.String())
	}
}

func TestValidateProxyURL_DynamicAllowlist(t *testing.T) {
	// Rotated CDN not on static list
	u := "https://g5.akademivo.website/hls/master.m3u8"
	if err := validateProxyURL(u); err == nil {
		// Might already be registered from prior tests — clear expectation carefully
	}

	// Simulate successful scrape registration
	services.RegisterStreamURLs(u, "https://e2e.majorplay.net/api/play")

	if err := validateProxyURL(u); err != nil {
		t.Fatalf("expected dynamic allow for %s: %v", u, err)
	}
	// Sibling subdomain under same root
	if err := validateProxyURL("https://cdn.akademivo.website/seg.ts"); err != nil {
		t.Fatalf("expected sibling subdomain allow: %v", err)
	}
	// Private still blocked even if someone tried to register
	services.RegisterStreamDomain("127.0.0.1")
	if err := validateProxyURL("http://127.0.0.1/secret"); err == nil {
		t.Fatal("private IP must stay blocked")
	}
}
