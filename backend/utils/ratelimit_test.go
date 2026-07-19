package utils

import "testing"

func TestTokenBucket_BurstThenBlock(t *testing.T) {
	tb := NewTokenBucket(0.001, 3) // very slow refill
	for i := 0; i < 3; i++ {
		if !tb.Allow("ip1") {
			t.Fatalf("token %d should be allowed", i+1)
		}
	}
	if tb.Allow("ip1") {
		t.Fatal("4th request should be blocked")
	}
	// different key independent
	if !tb.Allow("ip2") {
		t.Fatal("other key should be allowed")
	}
}

func TestClientIP(t *testing.T) {
	if got := ClientIP("1.2.3.4:5678", "", ""); got != "1.2.3.4" {
		t.Fatalf("got %q", got)
	}
	if got := ClientIP("1.2.3.4:5678", "9.9.9.9, 8.8.8.8", ""); got != "9.9.9.9" {
		t.Fatalf("xff got %q", got)
	}
	if got := ClientIP("1.2.3.4:5678", "", "7.7.7.7"); got != "7.7.7.7" {
		t.Fatalf("xri got %q", got)
	}
}
