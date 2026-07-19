package services

import (
	"log"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Dynamic stream domain allowlist.
//
// IDLIX and similar scrapers redeem on majorplay then stream from rotating
// CDNs (ruangskill.space, akademivo.website, g5.*, e2e.*, …). Static allowlists
// cannot keep up. When a scrape succeeds we register every media hostname so
// /api/proxy can serve HLS segments without 403.
//
// Security: private/loopback IPs are never registered. Entries expire (TTL).

const streamDomainTTL = 2 * time.Hour

var (
	streamDomainMu    sync.RWMutex
	streamDomainCache = make(map[string]time.Time) // host or root domain → expiresAt
)

func init() {
	go func() {
		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			purgeExpiredStreamDomains()
		}
	}()
}

func purgeExpiredStreamDomains() {
	now := time.Now()
	streamDomainMu.Lock()
	defer streamDomainMu.Unlock()
	before := len(streamDomainCache)
	for h, exp := range streamDomainCache {
		if now.After(exp) {
			delete(streamDomainCache, h)
		}
	}
	after := len(streamDomainCache)
	if before != after {
		log.Printf("[StreamDomain] purge: %d → %d entries", before, after)
	}
}

// RegisterStreamDomain allows host (and its registrable-ish root) for proxying
// until streamDomainTTL elapses.
func RegisterStreamDomain(host string) {
	host = normalizeStreamHost(host)
	if host == "" || !isSafeStreamHost(host) {
		return
	}

	now := time.Now()
	exp := now.Add(streamDomainTTL)

	streamDomainMu.Lock()
	defer streamDomainMu.Unlock()

	// Opportunistic purge of expired entries
	if len(streamDomainCache) > 500 {
		for h, e := range streamDomainCache {
			if now.After(e) {
				delete(streamDomainCache, h)
			}
		}
	}

	streamDomainCache[host] = exp
	// Also register eTLD+1 style root so g5.akademivo.website allows *.akademivo.website
	if root := streamRootDomain(host); root != "" && root != host {
		streamDomainCache[root] = exp
	}
	log.Printf("[StreamDomain] registered allow: %s (root=%s) ttl=%s", host, streamRootDomain(host), streamDomainTTL)
}

// RegisterStreamURLs extracts hostnames from one or more absolute/relative URLs
// and registers them for dynamic proxy allowlisting.
func RegisterStreamURLs(urls ...string) {
	for _, raw := range urls {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		// Protocol-relative
		if strings.HasPrefix(raw, "//") {
			raw = "https:" + raw
		}
		u, err := url.Parse(raw)
		if err != nil || u.Hostname() == "" {
			continue
		}
		RegisterStreamDomain(u.Hostname())
	}
}

// IsDynamicallyAllowedHost reports whether host was registered by a recent scrape
// (exact match or suffix match against a registered root domain).
func IsDynamicallyAllowedHost(host string) bool {
	host = normalizeStreamHost(host)
	if host == "" {
		return false
	}

	now := time.Now()
	streamDomainMu.RLock()
	defer streamDomainMu.RUnlock()

	// Exact
	if exp, ok := streamDomainCache[host]; ok && now.Before(exp) {
		return true
	}
	// Suffix: host ends with ".registered-root" or equals root
	for allowed, exp := range streamDomainCache {
		if now.After(exp) {
			continue
		}
		if host == allowed {
			return true
		}
		// subdomain of allowed root: g5.akademivo.website under akademivo.website
		if strings.HasSuffix(host, "."+allowed) {
			return true
		}
		// allowed is a subdomain pattern stored as full host — also accept sibling
		// subdomains under same root (g5.ruangskill.space vs cdn.ruangskill.space)
		if root := streamRootDomain(allowed); root != "" {
			if host == root || strings.HasSuffix(host, "."+root) {
				return true
			}
		}
	}
	return false
}

// RegisterStreamMetadata registers all hosts found on a successful scrape result.
func RegisterStreamMetadata(videoURL, thumbnailURL string, extraURLs ...string) {
	RegisterStreamURLs(videoURL, thumbnailURL)
	RegisterStreamURLs(extraURLs...)
}

func normalizeStreamHost(host string) string {
	host = strings.TrimSpace(strings.ToLower(host))
	// strip port if present
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	host = strings.TrimPrefix(host, ".")
	return host
}

// isSafeStreamHost rejects private IPs and localhost — never dynamically allow SSRF targets.
func isSafeStreamHost(host string) bool {
	if host == "" {
		return false
	}
	blocked := []string{"localhost", "127.0.0.1", "0.0.0.0", "::1", "host.docker.internal", "metadata.google.internal"}
	for _, b := range blocked {
		if host == b {
			return false
		}
	}
	// If host is an IP literal, only allow public IPs
	if ip := net.ParseIP(host); ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
			return false
		}
		if ip.Equal(net.ParseIP("169.254.169.254")) {
			return false
		}
		return true
	}
	// Hostnames that resolve only to private IPs are still blocked at proxy validate
	// via isPrivateIP — registration of public-looking names is OK.
	return true
}

// streamRootDomain returns a simple eTLD+1 approximation (last two labels).
// e.g. g5.akademivo.website → akademivo.website
//
//	cdn.majorplay.net → majorplay.net
//	foo.co.uk → co.uk (acceptable approximation for our CDN use-case)
func streamRootDomain(host string) string {
	host = normalizeStreamHost(host)
	parts := strings.Split(host, ".")
	if len(parts) < 2 {
		return host
	}
	// multi-part public suffixes we care about lightly
	if len(parts) >= 3 {
		secondLast := parts[len(parts)-2]
		// co.uk, com.au style
		if secondLast == "co" || secondLast == "com" || secondLast == "net" || secondLast == "org" || secondLast == "ac" {
			return strings.Join(parts[len(parts)-3:], ".")
		}
	}
	return strings.Join(parts[len(parts)-2:], ".")
}

// SnapshotStreamDomains returns current non-expired hosts (for tests/debug).
func SnapshotStreamDomains() []string {
	now := time.Now()
	streamDomainMu.RLock()
	defer streamDomainMu.RUnlock()
	out := make([]string, 0, len(streamDomainCache))
	for h, exp := range streamDomainCache {
		if now.Before(exp) {
			out = append(out, h)
		}
	}
	return out
}
