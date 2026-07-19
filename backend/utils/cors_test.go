package utils

import (
	"os"
	"testing"
)

func TestIsOriginAllowed_EmptyOrigin(t *testing.T) {
	if !IsOriginAllowed("") {
		t.Fatal("empty origin should be allowed")
	}
}

func TestIsOriginAllowed_DevLocalhost(t *testing.T) {
	prev := os.Getenv("CORS_ORIGINS")
	_ = os.Unsetenv("CORS_ORIGINS")
	t.Cleanup(func() {
		if prev == "" {
			_ = os.Unsetenv("CORS_ORIGINS")
		} else {
			_ = os.Setenv("CORS_ORIGINS", prev)
		}
	})

	// force reload
	_ = AllowedOrigins()

	cases := []string{
		"http://localhost:5173",
		"http://localhost:3000",
		"http://127.0.0.1:8080",
		"http://localhost:8080", // Vite proxy rewriteWsOrigin target
	}
	for _, o := range cases {
		if !IsOriginAllowed(o) {
			t.Errorf("expected allow %s when CORS_ORIGINS unset", o)
		}
	}
	if IsOriginAllowed("https://evil.example.com") {
		t.Error("evil origin should be denied in dev defaults")
	}
}

func TestIsOriginAllowed_StrictProd(t *testing.T) {
	prev := os.Getenv("CORS_ORIGINS")
	_ = os.Setenv("CORS_ORIGINS", "https://watchparty.example.com")
	t.Cleanup(func() {
		if prev == "" {
			_ = os.Unsetenv("CORS_ORIGINS")
		} else {
			_ = os.Setenv("CORS_ORIGINS", prev)
		}
	})

	_ = AllowedOrigins()

	if !IsOriginAllowed("https://watchparty.example.com") {
		t.Error("configured origin should be allowed")
	}
	if IsOriginAllowed("http://localhost:5173") {
		t.Error("localhost should be denied when CORS_ORIGINS is set")
	}
	if IsOriginAllowed("https://evil.example.com") {
		t.Error("unlisted origin should be denied")
	}
}
