package stream

import (
	"fmt"
	"testing"
	"time"
)

// TestDeduplicatorEnforcesHardCap guards the regression where evictLocked only
// removed expired entries, letting the seen map grow unbounded past its size
// cap during a burst of unique messages within the TTL window.
func TestDeduplicatorEnforcesHardCap(t *testing.T) {
	const size = 2
	d := NewDeduplicator(size, time.Hour) // long TTL: nothing expires during the test

	for i := 0; i < 6; i++ {
		msg := []byte(fmt.Sprintf(`{"event_type":"book","hash":"h%d"}`, i))
		if !d.Process(msg) {
			t.Fatalf("message %d should be treated as new", i)
		}
	}

	d.mu.Lock()
	got := len(d.seen)
	d.mu.Unlock()
	if got > size {
		t.Fatalf("seen map holds %d entries, must not exceed hard cap %d", got, size)
	}
}

// TestDeduplicatorStillDetectsDuplicates confirms the cap change did not break
// duplicate detection within the TTL.
func TestDeduplicatorStillDetectsDuplicates(t *testing.T) {
	d := NewDeduplicator(16, time.Hour)
	msg := []byte(`{"event_type":"book","hash":"abc"}`)
	if !d.Process(msg) {
		t.Fatal("first sighting should be new")
	}
	if d.Process(msg) {
		t.Fatal("second sighting within TTL should be a duplicate")
	}
}
