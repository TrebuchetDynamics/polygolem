package marketresults

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/polygolem/pkg/types"
)

type fakeOutcomeClient struct {
	outcome *types.CLOBMarketOutcome
	err     error
}

func (f fakeOutcomeClient) MarketOutcome(context.Context, string, string) (*types.CLOBMarketOutcome, error) {
	return f.outcome, f.err
}

type fakeGammaClient struct {
	markets []types.Market
	err     error
}

func (f fakeGammaClient) Markets(context.Context, *types.GetMarketsParams) ([]types.Market, error) {
	return f.markets, f.err
}

func TestNewResolverUsesProductionGammaDefault(t *testing.T) {
	resolver := NewResolver("", "")
	if resolver.gammaBaseURL != defaultGammaBaseURL {
		t.Fatalf("unexpected Gamma default %q", resolver.gammaBaseURL)
	}
}

func resolvedMarket(at time.Time) types.Market {
	return types.Market{
		ConditionID:   "condition-1",
		Slug:          "btc-updown",
		Closed:        true,
		ClosedTime:    types.NormalizedTime(at),
		ClobTokenIDs:  `["token-up","token-down"]`,
		OutcomePrices: types.StringOrArray{"1", "0"},
	}
}

func marketRef() MarketRef {
	return MarketRef{
		ConditionID: "condition-1",
		Slug:        "btc-updown",
		UpTokenID:   "token-up",
		DownTokenID: "token-down",
	}
}

func TestResolveRequiresWinnerAndAuthoritativeClosedTime(t *testing.T) {
	resolvedAt := time.Date(2026, 7, 10, 12, 5, 3, 0, time.UTC)
	observedAt := resolvedAt.Add(2 * time.Second)
	resolver := newResolver(
		fakeOutcomeClient{outcome: &types.CLOBMarketOutcome{
			Status:         types.CLOBOutcomeResolved,
			ConditionID:    "condition-1",
			WinningTokenID: "token-up",
			Closed:         true,
			Source:         "clob:/markets/condition-1",
		}},
		fakeGammaClient{markets: []types.Market{resolvedMarket(resolvedAt)}},
		"https://gamma.example",
		func() time.Time { return observedAt },
	)

	got, err := resolver.ResolveMarket(context.Background(), marketRef())
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.WinningTokenID != "token-up" {
		t.Fatalf("unexpected result: %#v", got)
	}
	if !got.ResolvedAt.Equal(resolvedAt) || !got.ObservedAt.Equal(observedAt) {
		t.Fatalf("timestamps lost: %#v", got)
	}
}

func TestResolveAcceptsExactGammaResultWhenClobHasNotPublishedWinner(t *testing.T) {
	resolvedAt := time.Date(2026, 7, 10, 12, 5, 3, 0, time.UTC)
	resolver := newResolver(
		fakeOutcomeClient{outcome: &types.CLOBMarketOutcome{
			Status:      types.CLOBOutcomeUnresolved,
			ConditionID: "condition-1",
			Closed:      true,
		}},
		fakeGammaClient{markets: []types.Market{resolvedMarket(resolvedAt)}},
		"https://gamma.example",
		func() time.Time { return resolvedAt.Add(time.Second) },
	)
	got, err := resolver.ResolveMarket(context.Background(), marketRef())
	if err != nil || got == nil || got.WinningTokenID != "token-up" {
		t.Fatalf("unexpected Gamma result %#v, %v", got, err)
	}
}

func TestResolveFailsClosedWithoutGammaClosedTimeOrExactPrices(t *testing.T) {
	market := resolvedMarket(time.Time{})
	resolver := newResolver(fakeOutcomeClient{}, fakeGammaClient{markets: []types.Market{market}}, "https://gamma.example", time.Now)
	got, err := resolver.ResolveMarket(context.Background(), marketRef())
	if err != nil || got != nil {
		t.Fatalf("missing closed time must remain unresolved: %#v, %v", got, err)
	}
	market = resolvedMarket(time.Now())
	market.OutcomePrices = types.StringOrArray{"0.99", "0.01"}
	resolver = newResolver(fakeOutcomeClient{}, fakeGammaClient{markets: []types.Market{market}}, "https://gamma.example", time.Now)
	got, err = resolver.ResolveMarket(context.Background(), marketRef())
	if err != nil || got != nil {
		t.Fatalf("nonterminal prices must remain unresolved: %#v, %v", got, err)
	}
}

func TestResolvePropagatesGammaSourceErrors(t *testing.T) {
	resolver := newResolver(
		fakeOutcomeClient{},
		fakeGammaClient{err: errors.New("gamma unavailable")},
		"https://gamma.example",
		time.Now,
	)
	if _, err := resolver.ResolveMarket(context.Background(), marketRef()); err == nil {
		t.Fatal("expected source error")
	}
}
