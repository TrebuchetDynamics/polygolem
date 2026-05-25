package papercrypto

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

func TestRunnerBuildsQueryAndFiltersActiveCryptoMarkets(t *testing.T) {
	searcher := &fakeSearcher{resp: &polytypes.SearchResponse{Events: []polytypes.Event{
		{
			ID:     "event-1",
			Title:  "BTC 5m window",
			Active: true,
			Markets: []polytypes.Market{
				{
					ID:           "market-1",
					Question:     "BTC up or down in 5m?",
					Active:       true,
					ClobTokenIDs: `["token-up","token-down"]`,
					Outcomes:     polytypes.StringOrArray{"Up", "Down"},
					EndDateISO:   "2026-05-23T12:35:00Z",
				},
				{
					ID:           "market-closed",
					Question:     "BTC stale 5m?",
					Active:       true,
					Closed:       true,
					ClobTokenIDs: `["stale"]`,
				},
			},
		},
		{
			ID:     "event-eth",
			Title:  "ETH 5m window",
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
				ID:           "market-hidden",
				Question:     "BTC hidden 5m?",
				Active:       true,
				ClobTokenIDs: `["hidden"]`,
			}},
		},
	}}}

	got, err := New(searcher).Run(context.Background(), Request{Asset: "btc", Interval: "5m", Limit: 0})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if searcher.params == nil {
		t.Fatal("Search was not called")
	}
	if searcher.params.Q != "btc 5m" {
		t.Fatalf("query=%q, want btc 5m", searcher.params.Q)
	}
	if searcher.params.LimitPerType == nil || *searcher.params.LimitPerType != 10 {
		t.Fatalf("LimitPerType=%v, want 10", searcher.params.LimitPerType)
	}
	if got.Query != "btc 5m" || got.Count != 1 {
		t.Fatalf("unexpected response summary: %+v", got)
	}
	if len(got.Markets) != 1 {
		t.Fatalf("markets=%d, want 1: %+v", len(got.Markets), got.Markets)
	}
	market := got.Markets[0]
	if market.EventID != "event-1" || market.MarketID != "market-1" || market.TokenID != "token-up" {
		t.Fatalf("unexpected market: %+v", market)
	}
	if len(market.Outcomes) != 2 || market.Outcomes[0] != "Up" || market.Outcomes[1] != "Down" {
		t.Fatalf("unexpected outcomes: %+v", market.Outcomes)
	}
}

func TestRunnerPropagatesSearchErrors(t *testing.T) {
	wantErr := errors.New("gamma down")
	_, err := New(&fakeSearcher{err: wantErr}).Run(context.Background(), Request{})
	if !errors.Is(err, wantErr) {
		t.Fatalf("error=%v, want %v", err, wantErr)
	}
}
