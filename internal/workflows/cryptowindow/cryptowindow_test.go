package cryptowindow

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/polygolem/internal/polytypes"
	"github.com/TrebuchetDynamics/polygolem/pkg/marketresolver"
)

type fakeEvents struct {
	slug  string
	event *polytypes.Event
	err   error
}

func (f *fakeEvents) EventBySlug(ctx context.Context, slug string) (*polytypes.Event, error) {
	f.slug = slug
	return f.event, f.err
}

type fakeMarketPricer struct {
	priceToken  string
	priceSide   string
	spreadToken string
}

func (f *fakeMarketPricer) Price(ctx context.Context, tokenID, side string) (string, error) {
	f.priceToken = tokenID
	f.priceSide = side
	return "0.51", nil
}

func (f *fakeMarketPricer) Spread(ctx context.Context, tokenID string) (string, error) {
	f.spreadToken = tokenID
	return "0.02", nil
}

func TestRunnerResolvesCurrentWindowAndEnrichesFirstActiveMarket(t *testing.T) {
	now := time.Date(2026, 5, 23, 12, 34, 56, 0, time.UTC)
	windowStart := time.Date(2026, 5, 23, 12, 30, 0, 0, time.UTC)
	events := &fakeEvents{event: &polytypes.Event{
		ID:    "event-1",
		Title: "BTC 5m",
		Slug:  "btc-updown-5m-expected",
		Markets: []polytypes.Market{
			{
				ID:            "market-closed",
				Active:        true,
				Closed:        true,
				ClobTokenIDs:  `["closed"]`,
				Outcomes:      polytypes.StringOrArray{"Up", "Down"},
				OutcomePrices: polytypes.StringOrArray{"0.1", "0.9"},
			},
			{
				ID:            "market-1",
				Question:      "BTC up or down?",
				ConditionID:   "condition-1",
				Active:        true,
				ClobTokenIDs:  `["up-token","down-token"]`,
				Outcomes:      polytypes.StringOrArray{"Up", "Down"},
				OutcomePrices: polytypes.StringOrArray{"0.51", "0.49"},
				EndDateISO:    "2026-05-23T12:35:00Z",
			},
		},
	}}
	pricer := &fakeMarketPricer{}
	runner := New(events, pricer)
	runner.Now = func() time.Time { return now }

	got, err := runner.Run(context.Background(), Request{Asset: "BTC", Interval: "5m", Enrich: true})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	wantSlug := marketresolver.CryptoWindowSlug("BTC", "5m", windowStart)
	if events.slug != wantSlug {
		t.Fatalf("slug=%q, want %q", events.slug, wantSlug)
	}
	if got.Asset != "BTC" || got.Interval != "5m" || got.Slug != wantSlug {
		t.Fatalf("unexpected summary: %+v", got)
	}
	if got.WindowStart != windowStart.Format(time.RFC3339) {
		t.Fatalf("window_start=%q, want %q", got.WindowStart, windowStart.Format(time.RFC3339))
	}
	if got.Count != 1 || len(got.Markets) != 1 {
		t.Fatalf("count/markets=%d/%d, want 1/1: %+v", got.Count, len(got.Markets), got.Markets)
	}
	market := got.Markets[0]
	if market.EventID != "event-1" || market.MarketID != "market-1" || market.ConditionID != "condition-1" {
		t.Fatalf("unexpected market IDs: %+v", market)
	}
	if len(market.TokenIDs) != 2 || market.TokenIDs[0] != "up-token" || market.TokenIDs[1] != "down-token" {
		t.Fatalf("unexpected token IDs: %+v", market.TokenIDs)
	}
	if market.Price != "0.51" || market.Spread != "0.02" {
		t.Fatalf("unexpected enrichment: %+v", market)
	}
	if pricer.priceToken != "up-token" || pricer.priceSide != "BUY" || pricer.spreadToken != "up-token" {
		t.Fatalf("unexpected price calls: %+v", pricer)
	}
}

func TestRunnerReportsWindowFetchFailureWithSlug(t *testing.T) {
	events := &fakeEvents{err: errors.New("not found")}
	runner := New(events, &fakeMarketPricer{})
	runner.Now = func() time.Time { return time.Date(2026, 5, 23, 12, 34, 56, 0, time.UTC) }

	_, err := runner.Run(context.Background(), Request{Asset: "BTC", Interval: "5m"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "window not found") || !strings.Contains(err.Error(), "btc-updown-5m") {
		t.Fatalf("error=%q, want window slug detail", err.Error())
	}
}
