package services

import (
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// SegmentCache provides an in-memory sliding window cache for HLS video segments
// (.ts, .m4s, audio/video data fragments) and master manifests.
//
// In a watch party room where 5-10 people watch in sync, without caching the proxy
// makes 5-10 duplicate upstream requests for every 2-second segment.
// With SegmentCache:
// 1. Singleflight deduplicates simultaneous requests for the exact same segment.
// 2. RAM ring buffer caches segments for 90s (plenty for room members around the same playback time).
// 3. Upstream bandwidth is reduced up to 90%, eliminating CDN rate-limiting/429.

const (
	defaultSegmentTTL  = 90 * time.Second
	manifestTTL        = 10 * time.Second
	maxSegmentCacheRAM = 160 * 1024 * 1024 // 160 MB safety cap for Render free tier
	maxSingleSegmentSz = 8 * 1024 * 1024   // 8 MB max per individual chunk
)

type CachedSegment struct {
	Body        []byte
	ContentType string
	StatusCode  int
	Headers     http.Header
	ExpiresAt   time.Time
	Size        int64
}

var (
	segmentCacheMu    sync.RWMutex
	segmentCacheMap   = make(map[string]*CachedSegment)
	segmentTotalBytes int64
	segmentFlight     singleflight.Group
)

func init() {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			purgeExpiredSegments()
		}
	}()
}

// IsCacheableSegment checks if a target URL qualifies for HLS segment or manifest caching.
func IsCacheableSegment(targetURL string) bool {
	u, err := url.Parse(targetURL)
	if err != nil {
		return false
	}
	path := strings.ToLower(u.Path)
	// TS and fMP4 / CMAF segments
	if strings.HasSuffix(path, ".ts") || strings.HasSuffix(path, ".m4s") ||
		strings.HasSuffix(path, ".aac") || strings.HasSuffix(path, ".vtt") {
		return true
	}
	// Majorplay / Shaka packager video/audio fragments
	if strings.Contains(strings.ToLower(u.Host), "majorplay.net") &&
		(strings.Contains(path, "/data-") || strings.Contains(path, "/config-")) {
		return true
	}
	// Generic query-param fragments (e.g. ?segment=... or .ts?...)
	if strings.Contains(strings.ToLower(targetURL), ".ts?") || strings.Contains(strings.ToLower(targetURL), ".m4s?") {
		return true
	}
	return false
}

// GetCachedSegment retrieves a segment from memory if present and not expired.
func GetCachedSegment(key string) (*CachedSegment, bool) {
	segmentCacheMu.RLock()
	seg, found := segmentCacheMap[key]
	if !found {
		segmentCacheMu.RUnlock()
		return nil, false
	}
	if time.Now().After(seg.ExpiresAt) {
		segmentCacheMu.RUnlock()
		// Lazy delete
		segmentCacheMu.Lock()
		delete(segmentCacheMap, key)
		segmentTotalBytes -= seg.Size
		segmentCacheMu.Unlock()
		return nil, false
	}
	segmentCacheMu.RUnlock()
	return seg, true
}

// PutCachedSegment stores a segment into memory with LRU eviction if memory exceeds cap.
func PutCachedSegment(key string, seg *CachedSegment) {
	if seg == nil || len(seg.Body) == 0 || int64(len(seg.Body)) > maxSingleSegmentSz {
		return
	}

	seg.Size = int64(len(seg.Body))
	if seg.ExpiresAt.IsZero() {
		seg.ExpiresAt = time.Now().Add(defaultSegmentTTL)
	}

	segmentCacheMu.Lock()
	defer segmentCacheMu.Unlock()

	// Evict oldest if exceeding RAM cap
	now := time.Now()
	for (segmentTotalBytes+seg.Size > maxSegmentCacheRAM) && len(segmentCacheMap) > 0 {
		var oldestKey string
		var oldestExp time.Time
		first := true
		for k, v := range segmentCacheMap {
			if first || v.ExpiresAt.Before(oldestExp) {
				oldestKey = k
				oldestExp = v.ExpiresAt
				first = false
			}
			if now.After(v.ExpiresAt) {
				oldestKey = k
				break
			}
		}
		if oldestKey != "" {
			old := segmentCacheMap[oldestKey]
			delete(segmentCacheMap, oldestKey)
			if old != nil {
				segmentTotalBytes -= old.Size
			}
		} else {
			break
		}
	}

	if existing, ok := segmentCacheMap[key]; ok {
		segmentTotalBytes -= existing.Size
	}
	segmentCacheMap[key] = seg
	segmentTotalBytes += seg.Size
}

func purgeExpiredSegments() {
	now := time.Now()
	segmentCacheMu.Lock()
	defer segmentCacheMu.Unlock()

	for k, v := range segmentCacheMap {
		if now.After(v.ExpiresAt) {
			delete(segmentCacheMap, k)
			segmentTotalBytes -= v.Size
		}
	}
}

// DoWithSingleflight deduplicates upstream requests for the same segment key.
func DoWithSingleflight(key string, fn func() (*CachedSegment, error)) (*CachedSegment, error) {
	v, err, _ := segmentFlight.Do(key, func() (interface{}, error) {
		return fn()
	})
	if err != nil {
		return nil, err
	}
	return v.(*CachedSegment), nil
}
