package marketdatacrypto

import (
	"context"
	"errors"
	"testing"

	"github.com/TrebuchetDynamics/polygolem/internal/polytypes"
)

type fakeSearcher struct {
	params *polytypes.SearchParams
	resp   *polytypes.SearchResponse
	err    error
}

func (f *fakeSearcher) Search(ctx context.Context, params *polytypes.SearchParams) (*polytypes.SearchResponse, error) {
	f.params = params
	return f.resp, f.err
}

type fakeQuoter struct {
	priceTokens    []string
	spreadTokens   []string
	midpointTokens []string
	tickTokens     []string
}

func (f *fakeQuoter) Price(ctx context.Context, tokenID, side string) (string, error) {
	f.priceTokens = append(f.priceTokens, tokenID+":"+side)
	return "0.62", nil
}

func (f *fakeQuoter) Spread(ctx context.Context, tokenID string) (string, error) {
	f.spreadTokens = append(f.spreadTokens, tokenID)
	return "0.03", nil
}

func (f *fakeQuoter) Midpoint(ctx context.Context, tokenID string) (string, error) {
	f.midpointTokens = append(f.midpointTokens, tokenID)
	return "0.605", nil
}

func (f *fakeQuoter) TickSize(ctx context.Context, tokenID string) (*polytypes.TickSize, error) {
	f.tickTokens = append(f.tickTokens, tokenID)
	return &polytypes.TickSize{MinimumTickSize: "0.01"}, nil
}

func TestRunnerSearchesFiltersQuotesAndLimitsCryptoSnapshots(t *testing.T) {
	searcher := &fakeSearcher{resp: &polytypes.SearchResponse{Events: []polytypes.Event{
		{
			ID:     "event-btc",
			Title:  "BTC 5m event",
			Active: true,
			Markets: []polytypes.Market{
				{
					ID:              "market-btc",
					Question:        "BTC up or down in 5m?",
					Active:          true,
					AcceptingOrders: true,
					ClobTokenIDs:    `["btc-up","btc-down"]`,
					Outcomes:        polytypes.StringOrArray{"Up", "Down"},
					EndDateISO:      "2026-05-23T12:35:00Z",
					Volume24hr:      123.45,
				},
				{
					ID:           "market-second",
					Question:     "BTC second 5m?",
					Active:       true,
					ClobTokenIDs: `["btc-second"]`,
				},
				{
					ID:           "market-closed",
					Question:     "BTC stale 5m?",
					Active:       true,
					Closed:       true,
					ClobTokenIDs: `["closed"]`,
				},
			},
		},
		{
			ID:     "event-eth",
			Title:  "ETH 5m event",
			Active: true,
			Markets: []polytypes.Market{{
				ID:           "market-eth",
				Question:     "ETH up or down in 5m?",
				Active:       true,
				ClobTokenIDs: `["eth-up","eth-down"]`,
			}},
		},
		{
			ID:     "event-closed",
			Title:  "BTC 5m closed event",
			Active: true,
			Closed: true,
			Markets: []polytypes.Market{{
				ID:           "hidden",
				Question:     "BTC hidden 5m?",
				Active:       true,
				ClobTokenIDs: `["hidden"]`,
			}},
		},
	}}}
	quoter := &fakeQuoter{}

	got, err := New(searcher, quoter).Run(context.Background(), Request{Asset: "BTC", Interval: "5m", Limit: 1})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if searcher.params == nil {
		t.Fatal("Search was not called")
	}
	if searcher.params.Q != "BTC 5m" {
		t.Fatalf("query=%q, want BTC 5m", searcher.params.Q)
	}
	if searcher.params.LimitPerType == nil || *searcher.params.LimitPerType != 50 {
		t.Fatalf("LimitPerType=%v, want 50", searcher.params.LimitPerType)
	}
	if got.Query != "BTC 5m" || got.Asset != "BTC" || got.Interval != "5m" || got.Count != 1 {
		t.Fatalf("unexpected summary: %+v", got)
	}
	if len(got.Markets) != 1 {
		t.Fatalf("markets=%d, want 1: %+v", len(got.Markets), got.Markets)
	}
	market := got.Markets[0]
	if market.EventID != "event-btc" || market.MarketID != "market-btc" || market.TokenID != "btc-up" || market.Outcome != "Up" {
		t.Fatalf("unexpected market identity: %+v", market)
	}
	if market.Price != "0.62" || market.Spread != "0.03" || market.Midpoint != "0.605" || market.TickSize != "0.01" {
		t.Fatalf("unexpected quote data: %+v", market)
	}
	if !market.AcceptingOrders || market.Volume24hr != 123.45 || market.EndDate != "2026-05-23T12:35:00Z" {
		t.Fatalf("unexpected market metadata: %+v", market)
	}
	if len(quoter.priceTokens) != 1 || quoter.priceTokens[0] != "btc-up:BUY" {
		t.Fatalf("unexpected price calls: %+v", quoter.priceTokens)
	}
	if len(quoter.spreadTokens) != 1 || quoter.spreadTokens[0] != "btc-up" || len(quoter.midpointTokens) != 1 || quoter.midpointTokens[0] != "btc-up" || len(quoter.tickTokens) != 1 || quoter.tickTokens[0] != "btc-up" {
		t.Fatalf("unexpected quote calls: %+v", quoter)
	}
}

func TestRunnerDefaultsEmptyQueryToCrypto(t *testing.T) {
	searcher := &fakeSearcher{resp: &polytypes.SearchResponse{}}
	got, err := New(searcher, nil).Run(context.Background(), Request{})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if searcher.params == nil || searcher.params.Q != "crypto" {
		t.Fatalf("query=%v, want crypto", searcher.params)
	}
	if got.Query != "crypto" {
		t.Fatalf("response query=%q, want crypto", got.Query)
	}
}

func TestRunnerPropagatesSearchErrors(t *testing.T) {
	wantErr := errors.New("gamma down")
	_, err := New(&fakeSearcher{err: wantErr}, nil).Run(context.Background(), Request{Asset: "BTC"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("error=%v, want %v", err, wantErr)
	}
}
