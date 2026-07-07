package clobsimulation

import (
	"context"
	"errors"
	"testing"

	"github.com/TrebuchetDynamics/polygolem/internal/polytypes"
)

func TestRunnerSimulateBuyWalksAsksBestFirst(t *testing.T) {
	reader := &fakeReader{book: &polytypes.OrderBook{
		Market:    "0xmarket",
		AssetID:   "token-1",
		Hash:      "hash-1",
		Timestamp: "123",
		Asks: []polytypes.OrderBookLevel{
			{Price: "0.60", Size: "10"},
			{Price: "0.50", Size: "10"},
		},
	}}
	runner := New(reader)

	got, err := runner.SimulateOrder(context.Background(), Request{TokenID: "token-1", Side: "buy", Amount: "11", Output: "json"})
	if err != nil {
		t.Fatalf("SimulateOrder returned error: %v", err)
	}
	if reader.tokenID != "token-1" {
		t.Fatalf("tokenID=%q", reader.tokenID)
	}
	if !got.Complete || got.FilledSize != "20" || got.Notional != "11" || got.AveragePrice != "0.55" || got.ExpectedFillPrice != "0.55" {
		t.Fatalf("result=%+v", got)
	}
	if got.BestPrice != "0.5" || got.WorstPrice != "0.6" || got.Slippage != "0.05" || got.SlippageBps != "1000" {
		t.Fatalf("prices/slippage=%+v", got)
	}
	if len(got.Levels) != 2 || got.Levels[0].Price != "0.5" || got.Levels[1].Price != "0.6" {
		t.Fatalf("levels=%+v", got.Levels)
	}
	if got.InputAmountType != "usdc" || got.BookHash != "hash-1" || got.BookTimestamp != "123" {
		t.Fatalf("metadata=%+v", got)
	}
}

func TestRunnerSimulateSellHonorsLimitPriceAndReportsUnfilled(t *testing.T) {
	reader := &fakeReader{book: &polytypes.OrderBook{Bids: []polytypes.OrderBookLevel{
		{Price: "0.30", Size: "10"},
		{Price: "0.40", Size: "5"},
	}}}
	runner := New(reader)

	got, err := runner.SimulateOrder(context.Background(), Request{TokenID: "token-1", Side: "sell", Amount: "8", LimitPrice: "0.35"})
	if err != nil {
		t.Fatalf("SimulateOrder returned error: %v", err)
	}
	if got.Complete || got.FilledSize != "5" || got.Notional != "2" || got.UnfilledAmount != "3" {
		t.Fatalf("result=%+v", got)
	}
	if got.InputAmountType != "shares" || got.LimitPrice != "0.35" || got.BestPrice != "0.4" || got.WorstPrice != "0.4" {
		t.Fatalf("prices=%+v", got)
	}
}

func TestRunnerValidatesBeforeCallingReader(t *testing.T) {
	for _, tc := range []struct {
		name string
		req  Request
		want string
	}{
		{name: "output", req: Request{TokenID: "token-1", Amount: "1", Output: "table"}, want: "only --output json is supported"},
		{name: "token", req: Request{Amount: "1"}, want: "--token is required"},
		{name: "side", req: Request{TokenID: "token-1", Side: "hold", Amount: "1"}, want: "--side must be buy or sell"},
		{name: "amount", req: Request{TokenID: "token-1", Amount: "0"}, want: "--amount must be a positive decimal"},
		{name: "limit", req: Request{TokenID: "token-1", Amount: "1", LimitPrice: "x"}, want: "--limit-price must be a positive decimal"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reader := &fakeReader{}
			_, err := New(reader).SimulateOrder(context.Background(), tc.req)
			if err == nil || err.Error() != tc.want {
				t.Fatalf("error=%v, want %q", err, tc.want)
			}
			if reader.tokenID != "" {
				t.Fatalf("reader called before validation with token %q", reader.tokenID)
			}
		})
	}
}

func TestRunnerPropagatesReaderError(t *testing.T) {
	wantErr := errors.New("clob down")
	_, err := New(&fakeReader{err: wantErr}).SimulateOrder(context.Background(), Request{TokenID: "token-1", Amount: "1"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("error=%v, want %v", err, wantErr)
	}
}

type fakeReader struct {
	book    *polytypes.OrderBook
	err     error
	tokenID string
}

func (f *fakeReader) OrderBook(_ context.Context, tokenID string) (*polytypes.OrderBook, error) {
	f.tokenID = tokenID
	return f.book, f.err
}
