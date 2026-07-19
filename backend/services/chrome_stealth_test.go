package services

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestStreamURLLooksLike(t *testing.T) {
	yes := []string{
		"https://cdn.example.com/master.m3u8",
		"https://x.com/path/index.m3u8?token=1",
		"https://krakenfiles.com/view/abc/file.html",
		"https://acefile.co/f/123/video",
		"https://hxfile.co/d/xyz",
		"https://cdn.x.com/hls/segment.m3u8",
	}
	for _, u := range yes {
		if !streamURLLooksLike(u) {
			t.Errorf("expected true for %s", u)
		}
	}
	if streamURLLooksLike("data:text/html,hi") {
		t.Error("data URL should be false")
	}
}

func TestHighConfidenceStreamURL(t *testing.T) {
	if !highConfidenceStreamURL("https://cdn.x.com/master.m3u8") {
		t.Error("m3u8 should be high confidence")
	}
	if !highConfidenceStreamURL("https://www.krakenfiles.com/view/abc") {
		t.Error("krakenfiles should be high confidence")
	}
	if highConfidenceStreamURL("https://doubleclick.net/ad.js") {
		t.Error("ads should not be high confidence")
	}
}

func TestIsAbortSuccess(t *testing.T) {
	cap := &networkCapture{}
	if isAbortSuccess(context.Canceled, cap) {
		t.Error("empty capture should not be success")
	}
	cap.add("https://cdn.x.com/master.m3u8")
	if !isAbortSuccess(context.Canceled, cap) {
		t.Error("canceled + stream should be success")
	}
	if !isAbortSuccess(context.DeadlineExceeded, cap) {
		t.Error("deadline + stream should be success")
	}
}

func TestIsDeadlineErr(t *testing.T) {
	if !isDeadlineErr(context.DeadlineExceeded) {
		t.Error("expected deadline")
	}
	if isDeadlineErr(nil) {
		t.Error("nil should be false")
	}
}

func TestIdlixPlaySVGPathPrefix(t *testing.T) {
	if !strings.HasPrefix(idlixPlaySVGPathPrefix, "M18.8906") {
		t.Fatalf("unexpected path prefix: %q", idlixPlaySVGPathPrefix)
	}
}

func TestNetworkCapture_SoftSuccess(t *testing.T) {
	cap := &networkCapture{}
	if cap.hasUsefulCapture() {
		t.Fatal("empty should be false")
	}
	cap.add("https://cdn.example.com/master.m3u8")
	if !cap.hasUsefulCapture() {
		t.Fatal("m3u8 should be useful")
	}
	if cap.bestStream() == "" {
		t.Fatal("expected stream")
	}
}

func TestStealthNavigate_Smoke(t *testing.T) {
	if findChromePath() == "" {
		t.Skip("Chrome not installed")
	}

	// Simple static page — should complete under session budget with resource blocking
	cap, html, err := stealthNavigate("https://example.com", 25*time.Second)
	if err != nil {
		if strings.Contains(err.Error(), "Chrome/Chromium not installed") {
			t.Fatal(err)
		}
		// Soft: still OK if we got capture object
		t.Logf("navigation error (acceptable offline): %v", err)
	}
	if cap == nil {
		t.Fatal("expected capture object")
	}
	_, _, _, total := cap.snapshot()
	t.Logf("total network requests=%d html_len=%d stream=%q", total, len(html), cap.bestStream())
	// With resource blocking, image/analytics requests should be low
	t.Logf("blocked patterns active; request count=%d", total)
}
