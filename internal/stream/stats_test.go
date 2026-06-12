package stream

import (
	"testing"
	"time"
)

func TestStreamStatsRecordsLifecycleCounters(t *testing.T) {
	stats := NewStreamStats("market")
	stats.SetSubscriptions([]string{"token-1", "token-2"}, nil)
	stats.MarkConnected(time.Unix(100, 0).UTC())
	stats.RecordMessage(time.Unix(101, 0).UTC())
	stats.RecordDuplicate()
	stats.RecordInvalid()
	stats.RecordReconnect(time.Unix(102, 0).UTC())
	stats.MarkDisconnected(time.Unix(103, 0).UTC())

	snap := stats.Snapshot()
	if snap.Stream != "market" || snap.State != "disconnected" {
		t.Fatalf("unexpected stream/state: %+v", snap)
	}
	if snap.MessagesReceived != 1 || snap.DuplicateMessages != 1 || snap.InvalidMessages != 1 || snap.Reconnects != 1 {
		t.Fatalf("unexpected counters: %+v", snap)
	}
	if len(snap.AssetIDs) != 2 || snap.AssetIDs[0] != "token-1" {
		t.Fatalf("asset ids not captured: %+v", snap.AssetIDs)
	}
	if snap.ConnectedAt == "" || snap.LastMessageAt == "" || snap.LastReconnectAt == "" || snap.DisconnectedAt == "" {
		t.Fatalf("timestamps missing: %+v", snap)
	}
}
