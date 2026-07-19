package utils

import (
	"os"
	"strings"
	"sync"
)

var (
	corsOnce    sync.Once
	corsAllowed map[string]struct{}
	corsEnvRaw  string
)

func loadAllowedOrigins() map[string]struct{} {
	raw := os.Getenv("CORS_ORIGINS")
	out := make(map[string]struct{})
	if raw == "" {
		for _, o := range []string{
			"http://localhost:5173",
			"http://localhost:5174",
			"http://127.0.0.1:5173",
			"http://127.0.0.1:5174",
		} {
			out[o] = struct{}{}
		}
		return out
	}
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out[part] = struct{}{}
		}
	}
	return out
}

// AllowedOrigins returns the CORS allowlist (dev defaults when CORS_ORIGINS unset).
func AllowedOrigins() map[string]struct{} {
	raw := os.Getenv("CORS_ORIGINS")
	corsOnce.Do(func() {
		corsEnvRaw = raw
		corsAllowed = loadAllowedOrigins()
	})
	// Reload if env changed (tests); production env is fixed at process start.
	if raw != corsEnvRaw {
		corsEnvRaw = raw
		corsAllowed = loadAllowedOrigins()
	}
	return corsAllowed
}

// IsOriginAllowed mirrors HTTP CORS rules for WebSocket CheckOrigin.
// Empty origin is allowed (non-browser / same-origin tooling).
// When CORS_ORIGINS is unset, any localhost / 127.0.0.1 port is allowed.
func IsOriginAllowed(origin string) bool {
	if origin == "" {
		return true
	}
	allowed := AllowedOrigins()
	if _, ok := allowed[origin]; ok {
		return true
	}
	if os.Getenv("CORS_ORIGINS") == "" {
		if strings.HasPrefix(origin, "http://localhost:") ||
			strings.HasPrefix(origin, "http://127.0.0.1:") {
			return true
		}
	}
	return false
}
