package services

import (
	"testing"
)

func TestRegisterAndAllowDynamicDomain(t *testing.T) {
	// Fresh-ish: register rotated IDLIX CDN hosts
	RegisterStreamURLs(
		"https://e2e.majorplay.net/api/play",
		"https://g5.akademivo.website/hls/master.m3u8",
		"https://g5.ruangskill.space/seg.ts",
	)

	cases := []struct {
		host string
		want bool
	}{
		{"e2e.majorplay.net", true},
		{"cdn.majorplay.net", true}, // same root
		{"majorplay.net", true},
		{"g5.akademivo.website", true},
		{"cdn.akademivo.website", true}, // sibling subdomain under root
		{"akademivo.website", true},
		{"g5.ruangskill.space", true},
		{"evil.example.com", false},
		{"127.0.0.1", false},
		{"localhost", false},
	}
	for _, tc := range cases {
		got := IsDynamicallyAllowedHost(tc.host)
		if got != tc.want {
			t.Errorf("IsDynamicallyAllowedHost(%q) = %v, want %v", tc.host, got, tc.want)
		}
	}
}

func TestRegisterStreamDomain_RejectsPrivate(t *testing.T) {
	RegisterStreamDomain("127.0.0.1")
	RegisterStreamDomain("localhost")
	RegisterStreamDomain("10.0.0.5")
	if IsDynamicallyAllowedHost("127.0.0.1") {
		t.Error("private IP must never be dynamically allowed")
	}
	if IsDynamicallyAllowedHost("localhost") {
		t.Error("localhost must never be dynamically allowed")
	}
}

func TestStreamRootDomain(t *testing.T) {
	if got := streamRootDomain("g5.akademivo.website"); got != "akademivo.website" {
		t.Errorf("root = %q", got)
	}
	if got := streamRootDomain("e2e.majorplay.net"); got != "majorplay.net" {
		t.Errorf("root = %q", got)
	}
}
