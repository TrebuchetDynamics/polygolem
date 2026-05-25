package datareads

import (
	"context"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/polygolem/internal/dataapi"
)

type fakeReader struct {
	tradesUser  string
	tradesLimit int
	called      bool
}

func (f *fakeReader) CurrentPositions(context.Context, string) ([]dataapi.Position, error) {
	f.called = true
	return []dataapi.Position{{TokenID: "pos-token"}}, nil
}

func (f *fakeReader) ClosedPositions(context.Context, string) ([]dataapi.ClosedPosition, error) {
	f.called = true
	return []dataapi.ClosedPosition{{TokenID: "closed-token"}}, nil
}

func (f *fakeReader) Trades(_ context.Context, user string, limit int) ([]dataapi.Trade, error) {
	f.called = true
	f.tradesUser = user
	f.tradesLimit = limit
	return []dataapi.Trade{{ID: "trade-1", AssetID: "asset-1"}}, nil
}

func (f *fakeReader) Activity(context.Context, string, int) ([]dataapi.Activity, error) {
	f.called = true
	return []dataapi.Activity{{Type: "trade"}}, nil
}

func (f *fakeReader) TopHolders(context.Context, string, int) ([]dataapi.MetaHolder, error) {
	f.called = true
	return []dataapi.MetaHolder{{Address: "0xholder"}}, nil
}

func (f *fakeReader) TotalValue(context.Context, string) (*dataapi.TotalValue, error) {
	f.called = true
	return &dataapi.TotalValue{User: "0xuser", Value: 12.3}, nil
}

func (f *fakeReader) MarketsTraded(context.Context, string) (*dataapi.TotalMarketsTraded, error) {
	f.called = true
	return &dataapi.TotalMarketsTraded{User: "0xuser", MarketsTraded: 2}, nil
}

func (f *fakeReader) OpenInterest(context.Context, string) (*dataapi.OpenInterest, error) {
	f.called = true
	return &dataapi.OpenInterest{Market: "token-1", OpenValue: 45.6}, nil
}

func (f *fakeReader) TraderLeaderboard(context.Context, int) ([]dataapi.TraderLeaderboardEntry, error) {
	f.called = true
	return []dataapi.TraderLeaderboardEntry{{Rank: 1, User: "0xleader"}}, nil
}

func (f *fakeReader) LiveVolume(context.Context, int) (*dataapi.LiveVolumeResponse, error) {
	f.called = true
	return &dataapi.LiveVolumeResponse{Total: 100}, nil
}

func TestRunnerRoutesPublicUserTrades(t *testing.T) {
	reader := &fakeReader{}
	runner := New(reader)

	got, err := runner.Run(context.Background(), Request{Operation: Trades, User: "0xuser", Limit: 7})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	rows, ok := got.([]dataapi.Trade)
	if !ok || len(rows) != 1 || rows[0].ID != "trade-1" {
		t.Fatalf("result=%#v", got)
	}
	if reader.tradesUser != "0xuser" || reader.tradesLimit != 7 {
		t.Fatalf("trades called with user=%q limit=%d", reader.tradesUser, reader.tradesLimit)
	}
}

func TestRunnerRejectsMissingUserBeforeRead(t *testing.T) {
	reader := &fakeReader{}
	runner := New(reader)

	_, err := runner.Run(context.Background(), Request{Operation: Positions})
	if err == nil || !strings.Contains(err.Error(), "--user required") {
		t.Fatalf("error=%v, want --user required", err)
	}
	if reader.called {
		t.Fatal("reader should not be called without --user")
	}
}

func TestRunnerRejectsMissingTokenBeforeRead(t *testing.T) {
	reader := &fakeReader{}
	runner := New(reader)

	_, err := runner.Run(context.Background(), Request{Operation: Holders})
	if err == nil || !strings.Contains(err.Error(), "--token-id required") {
		t.Fatalf("error=%v, want --token-id required", err)
	}
	if reader.called {
		t.Fatal("reader should not be called without --token-id")
	}
}
