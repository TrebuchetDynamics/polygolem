package opportunities

import (
	"context"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/polygolem/internal/polytypes"
)

type fakeMarketLister struct {
	params *polytypes.GetMarketsParams
	items  []polytypes.Market
}

func (f *fakeMarketLister) Markets(ctx context.Context, params *polytypes.GetMarketsParams) ([]polytypes.Market, error) {
	f.params = params
	return f.items, nil
}

type fakeCrypto5MSource struct {
	events map[string]*polytypes.Event
}

func (f fakeCrypto5MSource) Markets(ctx context.Context, params *polytypes.GetMarketsParams) ([]polytypes.Market, error) {
	return nil, nil
}

func (f fakeCrypto5MSource) EventBySlug(ctx context.Context, slug string) (*polytypes.Event, error) {
	return f.events[slug], nil
}

type fakeOpportunityPricer struct{}

func (fakeOpportunityPricer) Price(ctx context.Context, tokenID, side string) (string, error) {
	return "0.64", nil
}

func (fakeOpportunityPricer) Spread(ctx context.Context, tokenID string) (string, error) {
	return "0.05", nil
}

func TestRunnerFindsWideSpreadOpportunities(t *testing.T) {
	lister := &fakeMarketLister{items: []polytypes.Market{
		{ID: "tight", Question: "Tight book?", Active: true, Closed: false, Spread: 0.01, Volume24hr: 50, LiquidityClob: 100, ClobTokenIDs: `["tight-yes","tight-no"]`},
		{ID: "wide", Question: "Wide book?", Active: true, Closed: false, Spread: 0.22, Volume24hr: 250, LiquidityClob: 40, ClobTokenIDs: `["wide-yes","wide-no"]`},
		{ID: "closed", Question: "Closed wide?", Active: true, Closed: true, Spread: 0.5},
	}}

	got, err := New(Config{Gamma: lister}).Run(context.Background(), Request{Type: TypeWideSpread, Limit: 1})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if lister.params == nil || lister.params.Active == nil || !*lister.params.Active || lister.params.Closed == nil || *lister.params.Closed {
		t.Fatalf("Markets params should request active, open markets: %+v", lister.params)
	}
	if lister.params.Limit != 100 {
		t.Fatalf("Markets fetch limit=%d, want broad scan of 100", lister.params.Limit)
	}
	if got.Type != TypeWideSpread || got.Count != 1 || len(got.Opportunities) != 1 {
		t.Fatalf("unexpected response summary: %+v", got)
	}
	opp := got.Opportunities[0]
	if opp.MarketID != "wide" || opp.Question != "Wide book?" || opp.Spread != 0.22 {
		t.Fatalf("unexpected opportunity: %+v", opp)
	}
	if len(opp.TokenIDs) != 2 || opp.TokenIDs[0] != "wide-yes" {
		t.Fatalf("token IDs not parsed: %+v", opp.TokenIDs)
	}
	if len(opp.Reasons) == 0 || opp.Reasons[0] == "" {
		t.Fatalf("expected human-readable reason: %+v", opp.Reasons)
	}
}

func TestRunnerFindsMarketOpportunityTypes(t *testing.T) {
	now := time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)
	lister := &fakeMarketLister{items: []polytypes.Market{
		{ID: "volume", Question: "High volume thin book?", Active: true, Volume24hr: 500, LiquidityClob: 25},
		{ID: "deep", Question: "Deep book?", Active: true, Volume24hr: 500, LiquidityClob: 1000},
		{ID: "new", Question: "New market?", Active: true, New: true, CreatedAt: polytypes.NormalizedTime(now.Add(-time.Hour))},
		{ID: "old", Question: "Old market?", Active: true, New: false, CreatedAt: polytypes.NormalizedTime(now.Add(-48 * time.Hour))},
		{ID: "soon", Question: "Closing soon?", Active: true, EndDateISO: now.Add(2 * time.Hour).Format(time.RFC3339)},
		{ID: "later", Question: "Closing later?", Active: true, EndDateISO: now.Add(48 * time.Hour).Format(time.RFC3339)},
		{ID: "neg", Question: "Negative risk?", Active: true, NegRiskOther: true},
	}}
	runner := New(Config{Gamma: lister})
	runner.Now = func() time.Time { return now }

	cases := []struct {
		name       string
		scanner    Type
		request    Request
		wantMarket string
	}{
		{name: "low liquidity high volume", scanner: TypeLowLiquidityHighVolume, request: Request{Type: TypeLowLiquidityHighVolume, Limit: 1}, wantMarket: "volume"},
		{name: "new markets", scanner: TypeNewMarkets, request: Request{Type: TypeNewMarkets, Limit: 1}, wantMarket: "new"},
		{name: "closing soon", scanner: TypeClosingSoon, request: Request{Type: TypeClosingSoon, Hours: 6, Limit: 1}, wantMarket: "soon"},
		{name: "negative risk", scanner: TypeNegativeRisk, request: Request{Type: TypeNegativeRisk, Limit: 1}, wantMarket: "neg"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := runner.Run(context.Background(), tc.request)
			if err != nil {
				t.Fatalf("Run returned error: %v", err)
			}
			if got.Type != tc.scanner || got.Count != 1 || len(got.Opportunities) != 1 {
				t.Fatalf("unexpected response: %+v", got)
			}
			if got.Opportunities[0].MarketID != tc.wantMarket {
				t.Fatalf("market=%q, want %q in %+v", got.Opportunities[0].MarketID, tc.wantMarket, got.Opportunities)
			}
		})
	}
}

func TestRunnerFindsCrypto5MOpportunitiesForAsset(t *testing.T) {
	now := time.Date(2026, 6, 12, 12, 3, 0, 0, time.UTC)
	source := fakeCrypto5MSource{events: map[string]*polytypes.Event{
		"btc-updown-5m-1781265600": {
			ID:     "event-btc",
			Title:  "BTC Up or Down - June 12, 12:00PM",
			Slug:   "btc-updown-5m-1781265600",
			Active: true,
			Markets: []polytypes.Market{{
				ID:              "market-btc",
				Question:        "BTC up or down?",
				ConditionID:     "condition-btc",
				Active:          true,
				AcceptingOrders: true,
				ClobTokenIDs:    `["btc-up","btc-down"]`,
				Outcomes:        polytypes.StringOrArray{"Up", "Down"},
				EndDateISO:      "2026-06-12T12:05:00Z",
			}},
		},
	}}
	runner := New(Config{Gamma: source, Pricer: fakeOpportunityPricer{}})
	runner.Now = func() time.Time { return now }

	got, err := runner.Run(context.Background(), Request{Type: TypeCrypto5M, Asset: "BTC", Limit: 1})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if got.Type != TypeCrypto5M || got.Count != 1 || len(got.Opportunities) != 1 {
		t.Fatalf("unexpected response: %+v", got)
	}
	opp := got.Opportunities[0]
	if opp.MarketID != "market-btc" || opp.Asset != "BTC" || opp.Price != "0.64" || opp.SpreadText != "0.05" {
		t.Fatalf("unexpected crypto opportunity: %+v", opp)
	}
}
