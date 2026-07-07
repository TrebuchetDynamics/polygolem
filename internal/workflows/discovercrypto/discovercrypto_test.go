package discovercrypto

import (
	"context"
	"errors"
	"testing"

	"github.com/TrebuchetDynamics/polygolem/internal/polytypes"
)

type fakeSearcher struct {
	params *polytypes.SearchParams
	calls  int
	resps  []*polytypes.SearchResponse
	resp   *polytypes.SearchResponse
	err    error
}

func (f *fakeSearcher) Search(ctx context.Context, params *polytypes.SearchParams) (*polytypes.SearchResponse, error) {
	f.params = params
	f.calls++
	if len(f.resps) >= f.calls {
		return f.resps[f.calls-1], f.err
	}
	return f.resp, f.err
}

type fakePricer struct {
	priceToken  string
	priceSide   string
	spreadToken string
}

func (f *fakePricer) Price(ctx context.Context, tokenID, side string) (string, error) {
	f.priceToken = tokenID
	f.priceSide = side
	return "0.61", nil
}

func (f *fakePricer) Spread(ctx context.Context, tokenID string) (string, error) {
	f.spreadToken = tokenID
	return "0.04", nil
}

func TestRunnerSearchesFiltersAndEnrichesCryptoMarkets(t *testing.T) {
	searcher := &fakeSearcher{resp: &polytypes.SearchResponse{Events: []polytypes.Event{
		{
			ID:     "event-btc",
			Title:  "BTC 5m event",
			Slug:   "btc-updown-5m",
			Active: true,
			Markets: []polytypes.Market{
				{
					ID:            "market-btc",
					Question:      "BTC up or down in 5m?",
					ConditionID:   "condition-btc",
					Active:        true,
					ClobTokenIDs:  `["btc-up","btc-down"]`,
					Outcomes:      polytypes.StringOrArray{"Up", "Down"},
					OutcomePrices: polytypes.StringOrArray{"0.61", "0.39"},
					EndDateISO:    "2026-05-23T12:35:00Z",
					Volume24hr:    123.45,
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
			Slug:   "eth-updown-5m",
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
	pricer := &fakePricer{}

	got, err := New(searcher, pricer).Run(context.Background(), Request{Asset: "BTC", Interval: "5m", Limit: 7, Enrich: true})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if searcher.params == nil {
		t.Fatal("Search was not called")
	}
	if searcher.params.Q != "bitcoin 5m updown" {
		t.Fatalf("query=%q, want bitcoin 5m updown", searcher.params.Q)
	}
	if searcher.params.LimitPerType == nil || *searcher.params.LimitPerType != 7 {
		t.Fatalf("LimitPerType=%v, want 7", searcher.params.LimitPerType)
	}
	if searcher.params.EventsStatus != "active" {
		t.Fatalf("EventsStatus=%q, want active", searcher.params.EventsStatus)
	}
	if got.Query != "bitcoin 5m updown" || got.Asset != "BTC" || got.Interval != "5m" || got.Count != 1 {
		t.Fatalf("unexpected summary: %+v", got)
	}
	if len(got.Markets) != 1 {
		t.Fatalf("markets=%d, want 1: %+v", len(got.Markets), got.Markets)
	}
	market := got.Markets[0]
	if market.EventID != "event-btc" || market.MarketID != "market-btc" || market.TokenID != `["btc-up","btc-down"]` {
		t.Fatalf("unexpected market IDs: %+v", market)
	}
	if len(market.Outcomes) != 2 || market.Outcomes[0] != "Up" || market.OutcomePrices[0] != "0.61" {
		t.Fatalf("unexpected outcomes/prices: %+v", market)
	}
	if market.Volume24hr != 123.45 || market.Price != "0.61" || market.Spread != "0.04" {
		t.Fatalf("unexpected market values: %+v", market)
	}
	if pricer.priceToken != "btc-up" || pricer.priceSide != "BUY" || pricer.spreadToken != "btc-up" {
		t.Fatalf("unexpected price calls: %+v", pricer)
	}
}

func TestRunnerPaginatesUntilLimit(t *testing.T) {
	searcher := &fakeSearcher{resps: []*polytypes.SearchResponse{
		{Pagination: polytypes.Pagination{HasMore: true}, Events: []polytypes.Event{{ID: "e1", Title: "Bitcoin Up or Down", Slug: "btc-updown-5m-1", Active: true, Markets: []polytypes.Market{{ID: "m1", Slug: "btc-updown-5m-1", Question: "Bitcoin Up or Down", Active: true}}}}},
		{Events: []polytypes.Event{{ID: "e2", Title: "Bitcoin Up or Down", Slug: "btc-updown-5m-2", Active: true, Markets: []polytypes.Market{{ID: "m2", Slug: "btc-updown-5m-2", Question: "Bitcoin Up or Down", Active: true}}}}},
	}}
	got, err := New(searcher, nil).Run(context.Background(), Request{Asset: "BTC", Interval: "5m", Limit: 2})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if searcher.calls != 2 || got.Count != 2 {
		t.Fatalf("calls=%d count=%d, want 2/2", searcher.calls, got.Count)
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
