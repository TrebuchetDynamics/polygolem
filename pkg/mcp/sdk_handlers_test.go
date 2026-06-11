package mcp

import (
	"context"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/polygolem/pkg/marketdata"
	"github.com/TrebuchetDynamics/polygolem/pkg/stream"
)

func TestNewSDKReadOnlyHandlersSkipsNilClients(t *testing.T) {
	handlers := NewSDKReadOnlyHandlers(HandlerConfig{}, nil, nil, nil)
	if len(handlers) != 0 {
		t.Fatalf("expected no handlers for nil clients: %#v", handlers)
	}
}

func TestNewSDKReadOnlyHandlersPreservesExplicitAdapters(t *testing.T) {
	handlers := NewSDKReadOnlyHandlers(HandlerConfig{
		Timeout: time.Second,
		Adapters: ReadOnlyAdapters{
			MarketDataSnapshot: func(ctx context.Context, args map[string]any) (any, error) {
				return map[string]string{"custom": "ok"}, nil
			},
		},
	}, nil, nil, nil)
	if _, ok := handlers["polygolem.marketdata_snapshot"]; !ok {
		t.Fatalf("expected custom marketdata handler: %#v", handlers)
	}
}

func TestNewMarketDataSnapshotHandlerReadsTracker(t *testing.T) {
	tracker := marketdata.NewTracker()
	tracker.ApplyBestBidAsk(stream.BestBidAskMessage{AssetID: "token-1", BestBid: "0.40", BestAsk: "0.42"})
	handler := NewMarketDataSnapshotHandler(tracker)
	if handler == nil {
		t.Fatal("expected handler")
	}
	got, err := handler(context.Background(), map[string]any{"token_id": "token-1"})
	if err != nil {
		t.Fatal(err)
	}
	snapshot := got.(marketdata.Snapshot)
	if snapshot.BestBid != "0.40" || snapshot.BestAsk != "0.42" {
		t.Fatalf("snapshot=%+v", snapshot)
	}
	if _, err := handler(context.Background(), map[string]any{"token_id": "missing"}); err == nil {
		t.Fatal("expected missing snapshot error")
	}
}

func TestSDKHandlerArgumentParsing(t *testing.T) {
	if got := stringArg(map[string]any{"query": "btc"}, "query"); got != "btc" {
		t.Fatalf("stringArg=%q", got)
	}
	for _, value := range []any{float64(7), 7, "7"} {
		if got := intArg(map[string]any{"limit": value}, "limit"); got != 7 {
			t.Fatalf("intArg(%T)=%d", value, got)
		}
	}
}
