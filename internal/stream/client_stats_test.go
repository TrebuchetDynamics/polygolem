package stream

import "testing"

func TestMarketClientStatsCountsInvalidAndDuplicateMessages(t *testing.T) {
	client := NewMarketClient(DefaultConfig("ws://127.0.0.1/unused"))
	var books int
	client.OnBook = func(BookMessage) { books++ }

	msg := []byte(`{"event_type":"book","asset_id":"token-1","market":"market-1","timestamp":"1","hash":"same","bids":[],"asks":[]}`)
	client.processMessage(msg)
	client.processMessage(msg)
	client.processMessage([]byte(`{"event_type":"unknown"}`))

	stats := client.Stats()
	if books != 1 {
		t.Fatalf("books=%d, want 1 after duplicate suppression", books)
	}
	if stats.MessagesReceived != 1 || stats.DuplicateMessages != 1 || stats.InvalidMessages != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}
