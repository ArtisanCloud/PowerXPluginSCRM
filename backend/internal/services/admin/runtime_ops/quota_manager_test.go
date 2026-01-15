package runtime_ops

import (
	"testing"
	"time"
)

func TestTokenBucketAllow(t *testing.T) {
	bucket := newTokenBucket(10, 1) // 1 token per second, capacity 10
	if !bucket.Allow(5) {
		t.Fatalf("expected to allow initial tokens")
	}
	if bucket.Allow(6) {
		t.Fatalf("should not allow exceeding capacity")
	}
	time.Sleep(1200 * time.Millisecond)
	if !bucket.Allow(1) {
		t.Fatalf("expected tokens after refill")
	}
}
