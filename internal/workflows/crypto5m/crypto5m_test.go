package crypto5m

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/polygolem/internal/polytypes"
	"github.com/TrebuchetDynamics/polygolem/pkg/marketresolver"
)

type fakeEvents struct {
	events map[string]*polytypes.Event
	errs   map[string]error
	slugs  []string
}

func (f *fakeEvents) EventBySlug(ctx context.Context, slug string) (*polytypes.Event, error) {
	f.slugs = append(f.slugs, slug)
	if err := f.errs[slug]; err != nil {
		return nil, err
	}
	return f.events[slug], nil
}

type fakePricer struct {
	priceToken  string
	priceSide   string
	spreadToken string
}

func (f *fakePricer) Price(ctx context.Context, tokenID, side string) (string, error) {
	f.priceToken = tokenID
	f.priceSide = side
	return "0.55", nil
}

func (f *fakePricer) Spread(ctx context.Context, tokenID string) (string, error) {
	f.spreadToken = tokenID
	return "0.03", nil
}

func TestRunnerScansAssetsAndKeepsPerAssetStatus(t *testing.T) {
	now := time.Date(2026, 5, 23, 12, 34, 56, 0, time.UTC)
	windowStart := time.Date(2026, 5, 23, 12, 30, 0, 0, time.UTC)
	btcSlug := marketresolver.CryptoWindowSlug("BTC", "5m", windowStart)
	ethSlug := marketresolver.CryptoWindowSlug("ETH", "5m", windowStart)
	solSlug := marketresolver.CryptoWindowSlug("SOL", "5m", windowStart)
	events := &fakeEvents{
		events: map[string]*polytypes.Event{
			btcSlug: {
				ID:    "event-btc",
				Title: "BTC 5m",
				Slug:  btcSlug,
				Markets: []polytypes.Market{
					{ID: "btc-closed", Active: true, Closed: true, ClobTokenIDs: `["closed"]`},
					{
						ID:              "btc-market",
						Question:        "BTC up or down?",
						ConditionID:     "btc-condition",
						Active:          true,
						AcceptingOrders: true,
						LiquidityClob:   12000,
						BestBid:         0.50,
						BestAsk:         0.51,
						Spread:          0.01,
						ClobTokenIDs:    `["btc-up","btc-down"]`,
						Outcomes:        polytypes.StringOrArray{"Up", "Down"},
						EndDateISO:      "2026-05-23T12:35:00Z",
					},
				},
			},
			solSlug: {
				ID:    "event-sol",
				Title: "SOL 5m",
				Slug:  solSlug,
				Markets: []polytypes.Market{
					{ID: "sol-closed", Active: true, Closed: true, ClobTokenIDs: `["sol"]`},
				},
			},
		},
		errs: map[string]error{ethSlug: errors.New("gamma 404")},
	}
	pricer := &fakePricer{}
	runner := New(events, pricer)
	runner.Now = func() time.Time { return now }

	got, err := runner.Run(context.Background(), Request{Assets: []string{"BTC", "ETH", "BAD", "SOL"}, Enrich: true})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if got.Interval != "5m" || got.WindowStart != windowStart.Format(time.RFC3339) {
		t.Fatalf("unexpected summary: %+v", got)
	}
	if got.Count != 4 || len(got.Markets) != 4 {
		t.Fatalf("count/markets=%d/%d, want 4/4: %+v", got.Count, len(got.Markets), got.Markets)
	}
	if got.Markets[0].Asset != "BTC" || got.Markets[0].Status != "active" || got.Markets[0].MarketID != "btc-market" {
		t.Fatalf("unexpected BTC result: %+v", got.Markets[0])
	}
	if got.Markets[0].Price != "0.55" || got.Markets[0].Spread != "0.03" {
		t.Fatalf("BTC enrichment missing: %+v", got.Markets[0])
	}
	if !got.Markets[0].AcceptingOrders || got.Markets[0].LiquidityClob != 12000 || got.Markets[0].BookSpread != 0.01 {
		t.Fatalf("BTC book fields missing: %+v", got.Markets[0])
	}
	if got.Markets[1].Asset != "ETH" || got.Markets[1].Status != "not_found" || got.Markets[1].Error != "gamma 404" {
		t.Fatalf("unexpected ETH result: %+v", got.Markets[1])
	}
	if got.Markets[2].Asset != "BAD" || got.Markets[2].Status != "error" || got.Markets[2].Error != "unsupported asset" {
		t.Fatalf("unexpected BAD result: %+v", got.Markets[2])
	}
	if got.Markets[3].Asset != "SOL" || got.Markets[3].Status != "no_active_market" {
		t.Fatalf("unexpected SOL result: %+v", got.Markets[3])
	}
	if pricer.priceToken != "btc-up" || pricer.priceSide != "BUY" || pricer.spreadToken != "btc-up" {
		t.Fatalf("unexpected price calls: %+v", pricer)
	}
}

func TestRunnerAddsLocalTimezoneFields(t *testing.T) {
	now := time.Date(2026, 6, 28, 4, 26, 0, 0, time.UTC)
	windowStart := time.Date(2026, 6, 28, 4, 25, 0, 0, time.UTC)
	slug := marketresolver.CryptoWindowSlug("BTC", "5m", windowStart)
	events := &fakeEvents{events: map[string]*polytypes.Event{slug: {
		ID:    "event-btc",
		Title: "BTC 5m",
		Slug:  slug,
		Markets: []polytypes.Market{{
			ID:              "btc-market",
			Question:        "BTC up or down?",
			Active:          true,
			AcceptingOrders: true,
			ClobTokenIDs:    `["btc-up","btc-down"]`,
		}},
	}}, errs: map[string]error{}}
	runner := New(events, nil)
	runner.Now = func() time.Time { return now }

	got, err := runner.Run(context.Background(), Request{Assets: []string{"BTC"}, Timezone: "America/Chicago"})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if got.Timezone != "America/Chicago" || got.Markets[0].WindowStartLocal != "2026-06-27T23:25:00-05:00" {
		t.Fatalf("unexpected timezone fields: %+v", got)
	}
}

func TestRunnerIncludesFutureWindows(t *testing.T) {
	now := time.Date(2026, 5, 23, 12, 34, 56, 0, time.UTC)
	windowStart := time.Date(2026, 5, 23, 12, 30, 0, 0, time.UTC)
	events := &fakeEvents{events: map[string]*polytypes.Event{}, errs: map[string]error{}}
	runner := New(events, nil)
	runner.Now = func() time.Time { return now }

	got, err := runner.Run(context.Background(), Request{Assets: []string{"BTC"}, HoursAhead: 1})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if got.Count != 13 || len(events.slugs) != 13 {
		t.Fatalf("count/slugs=%d/%d, want 13/13", got.Count, len(events.slugs))
	}
	wantLast := marketresolver.CryptoWindowSlug("BTC", "5m", windowStart.Add(time.Hour))
	if events.slugs[12] != wantLast {
		t.Fatalf("last slug=%q, want %q", events.slugs[12], wantLast)
	}
}

func TestRunnerDefaultsToSupportedAssets(t *testing.T) {
	runner := New(&fakeEvents{events: map[string]*polytypes.Event{}, errs: map[string]error{}}, nil)
	runner.Now = func() time.Time { return time.Date(2026, 5, 23, 12, 34, 56, 0, time.UTC) }

	got, err := runner.Run(context.Background(), Request{})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(got.Assets) != len(SupportedAssets) {
		t.Fatalf("assets=%d, want %d", len(got.Assets), len(SupportedAssets))
	}
}
