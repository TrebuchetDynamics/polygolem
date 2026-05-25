package clobmarketdata

import (
	"context"
	"errors"
	"testing"

	"github.com/TrebuchetDynamics/polygolem/internal/polytypes"
)

func TestRunnerBookValidatesOutputBeforeCallingReader(t *testing.T) {
	reader := &fakeReader{}
	runner := New(reader)

	_, err := runner.Book(context.Background(), TokenRequest{TokenID: "token-1", Output: "table"})
	if err == nil || err.Error() != "only --output json is supported" {
		t.Fatalf("error=%v, want only --output json is supported", err)
	}
	if reader.bookToken != "" {
		t.Fatalf("reader called with token %q before output validation", reader.bookToken)
	}
}

func TestRunnerReadsBookTickHistoryAndMarkets(t *testing.T) {
	reader := &fakeReader{
		book:    &polytypes.OrderBook{AssetID: "token-1"},
		tick:    &polytypes.TickSize{MinimumTickSize: "0.01"},
		history: &polytypes.PriceHistory{History: []polytypes.PricePoint{{T: "1", P: "0.5"}}},
		market:  &polytypes.CLOBMarket{ConditionID: "0xmarket"},
		byToken: &polytypes.CLOBMarketByTokenResponse{ConditionID: "0xmarket", PrimaryTokenID: "token-1"},
		markets: &polytypes.CLOBPaginatedMarkets{NextCursor: "next", Count: 1},
	}
	runner := New(reader)

	book, err := runner.Book(context.Background(), TokenRequest{TokenID: "token-1", Output: "json"})
	if err != nil || book.AssetID != "token-1" || reader.bookToken != "token-1" {
		t.Fatalf("Book result=%+v token=%q err=%v", book, reader.bookToken, err)
	}
	tick, err := runner.TickSize(context.Background(), TokenRequest{TokenID: "token-2"})
	if err != nil || tick.MinimumTickSize != "0.01" || reader.tickToken != "token-2" {
		t.Fatalf("TickSize result=%+v token=%q err=%v", tick, reader.tickToken, err)
	}
	history, err := runner.PriceHistory(context.Background(), PriceHistoryRequest{TokenID: "token-3", Interval: "1h"})
	if err != nil || len(history.History) != 1 || reader.historyParams.Market != "token-3" || reader.historyParams.Interval != "1h" {
		t.Fatalf("PriceHistory result=%+v params=%+v err=%v", history, reader.historyParams, err)
	}
	market, err := runner.Market(context.Background(), ConditionRequest{ConditionID: "0xmarket"})
	if err != nil || market.ConditionID != "0xmarket" || reader.marketConditionID != "0xmarket" {
		t.Fatalf("Market result=%+v condition=%q err=%v", market, reader.marketConditionID, err)
	}
	byToken, err := runner.MarketByToken(context.Background(), TokenRequest{TokenID: "token-4"})
	if err != nil || byToken.PrimaryTokenID != "token-1" || reader.byTokenID != "token-4" {
		t.Fatalf("MarketByToken result=%+v token=%q err=%v", byToken, reader.byTokenID, err)
	}
	markets, err := runner.Markets(context.Background(), MarketsRequest{Cursor: "cursor-1"})
	if err != nil || markets.NextCursor != "next" || reader.marketsCursor != "cursor-1" {
		t.Fatalf("Markets result=%+v cursor=%q err=%v", markets, reader.marketsCursor, err)
	}
}

func TestRunnerPropagatesReaderErrors(t *testing.T) {
	wantErr := errors.New("clob down")
	runner := New(&fakeReader{err: wantErr})
	_, err := runner.Markets(context.Background(), MarketsRequest{})
	if !errors.Is(err, wantErr) {
		t.Fatalf("error=%v, want %v", err, wantErr)
	}
}

type fakeReader struct {
	book    *polytypes.OrderBook
	tick    *polytypes.TickSize
	history *polytypes.PriceHistory
	market  *polytypes.CLOBMarket
	byToken *polytypes.CLOBMarketByTokenResponse
	markets *polytypes.CLOBPaginatedMarkets
	err     error

	bookToken         string
	tickToken         string
	historyParams     polytypes.PriceHistoryParams
	marketConditionID string
	byTokenID         string
	marketsCursor     string
}

func (f *fakeReader) OrderBook(_ context.Context, tokenID string) (*polytypes.OrderBook, error) {
	f.bookToken = tokenID
	return f.book, f.err
}

func (f *fakeReader) TickSize(_ context.Context, tokenID string) (*polytypes.TickSize, error) {
	f.tickToken = tokenID
	return f.tick, f.err
}

func (f *fakeReader) PricesHistory(_ context.Context, params *polytypes.PriceHistoryParams) (*polytypes.PriceHistory, error) {
	if params != nil {
		f.historyParams = *params
	}
	return f.history, f.err
}

func (f *fakeReader) Market(_ context.Context, conditionID string) (*polytypes.CLOBMarket, error) {
	f.marketConditionID = conditionID
	return f.market, f.err
}

func (f *fakeReader) MarketByToken(_ context.Context, tokenID string) (*polytypes.CLOBMarketByTokenResponse, error) {
	f.byTokenID = tokenID
	return f.byToken, f.err
}

func (f *fakeReader) Markets(_ context.Context, cursor string) (*polytypes.CLOBPaginatedMarkets, error) {
	f.marketsCursor = cursor
	return f.markets, f.err
}
