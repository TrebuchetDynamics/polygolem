package papertrade

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/polygolem/internal/paper"
	"github.com/TrebuchetDynamics/polygolem/pkg/marketresolver"
)

type fakeResolver struct {
	asset       string
	interval    string
	windowStart time.Time
	result      marketresolver.ResolveResult
	calls       int
}

func (f *fakeResolver) ResolveTokenIDsForWindow(ctx context.Context, asset, timeframe string, windowStart time.Time) marketresolver.ResolveResult {
	f.calls++
	f.asset = asset
	f.interval = timeframe
	f.windowStart = windowStart
	return f.result
}

type fakePricer struct {
	tokenID string
	side    string
	price   string
	err     error
	calls   int
}

func (f *fakePricer) Price(ctx context.Context, tokenID, side string) (string, error) {
	f.calls++
	f.tokenID = tokenID
	f.side = side
	return f.price, f.err
}

func TestRunnerResolvesCryptoWindowAndBuysUpOutcome(t *testing.T) {
	now := time.Date(2026, 5, 23, 12, 34, 56, 0, time.UTC)
	resolver := &fakeResolver{result: marketresolver.ResolveResult{
		Status:      marketresolver.StatusAvailable,
		UpTokenID:   "up-token",
		DownTokenID: "down-token",
	}}
	pricer := &fakePricer{price: "0.42"}
	state := paper.NewState("USD", 100)

	runner := New(resolver, pricer, state)
	runner.Now = func() time.Time { return now }

	got, err := runner.Run(context.Background(), Request{
		Asset:    "BTC",
		Interval: "5m",
		Side:     "up",
		Size:     2,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if resolver.calls != 1 {
		t.Fatalf("resolver calls=%d, want 1", resolver.calls)
	}
	if resolver.asset != "BTC" || resolver.interval != "5m" {
		t.Fatalf("resolver got asset=%q interval=%q", resolver.asset, resolver.interval)
	}
	wantWindow := time.Date(2026, 5, 23, 12, 30, 0, 0, time.UTC)
	if !resolver.windowStart.Equal(wantWindow) {
		t.Fatalf("windowStart=%s, want %s", resolver.windowStart, wantWindow)
	}
	if pricer.tokenID != "up-token" || pricer.side != "SELL" {
		t.Fatalf("price lookup token=%q side=%q, want up-token SELL", pricer.tokenID, pricer.side)
	}
	if got.Action != "paper_trade" || got.TokenID != "up-token" {
		t.Fatalf("unexpected action/token: %+v", got)
	}
	if got.Price != 0.42 || got.Size != 2 || got.Cost != 0.84 {
		t.Fatalf("unexpected fill math: %+v", got)
	}
	if got.Cash != 99.16 {
		t.Fatalf("cash=%v, want 99.16", got.Cash)
	}
	if got.Timestamp != now.Format(time.RFC3339) {
		t.Fatalf("timestamp=%q, want %q", got.Timestamp, now.Format(time.RFC3339))
	}
	if state.Positions["up-token"].Size != 2 {
		t.Fatalf("paper position not updated: %+v", state.Positions)
	}
}

func TestRunnerUsesTokenIDBypassWithoutResolverOrPricerWhenPriceProvided(t *testing.T) {
	resolver := &fakeResolver{result: marketresolver.ResolveResult{Status: marketresolver.StatusUnresolved}}
	pricer := &fakePricer{err: errors.New("price should not be called")}
	state := paper.NewState("USD", 10)

	runner := New(resolver, pricer, state)
	got, err := runner.Run(context.Background(), Request{
		TokenID: "direct-token",
		Price:   "0.25",
		Size:    4,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if resolver.calls != 0 {
		t.Fatalf("resolver calls=%d, want 0", resolver.calls)
	}
	if pricer.calls != 0 {
		t.Fatalf("pricer calls=%d, want 0", pricer.calls)
	}
	if got.TokenID != "direct-token" || got.Price != 0.25 || got.Cash != 9 {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestRunnerRefusesUnavailableWindow(t *testing.T) {
	resolver := &fakeResolver{result: marketresolver.ResolveResult{
		Status: marketresolver.StatusWindowMismatch,
		Source: "gamma:slug_hit_window_mismatch",
	}}
	runner := New(resolver, &fakePricer{price: "0.42"}, paper.NewState("USD", 100))
	runner.Now = func() time.Time { return time.Date(2026, 5, 23, 12, 34, 56, 0, time.UTC) }

	_, err := runner.Run(context.Background(), Request{
		Asset:    "BTC",
		Interval: "5m",
		Side:     "up",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "window not found") || !strings.Contains(err.Error(), "window_mismatch") {
		t.Fatalf("error=%q, want window status detail", err.Error())
	}
}
