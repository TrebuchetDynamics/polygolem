// Package marketresults resolves read-only Polymarket market outcomes with
// causal timestamps. A result is emitted only when CLOB supplies a winning
// token and Gamma supplies a non-zero authoritative closed time.
package marketresults

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/polygolem/pkg/clob"
	"github.com/TrebuchetDynamics/polygolem/pkg/gamma"
	"github.com/TrebuchetDynamics/polygolem/pkg/types"
)

// Result is a causally observed, authoritative market result.
type Result struct {
	ConditionID    string    `json:"condition_id"`
	WinningTokenID string    `json:"winning_token_id"`
	ResolvedAt     time.Time `json:"resolved_at"`
	ObservedAt     time.Time `json:"observed_at"`
	Source         string    `json:"source"`
}

// MarketRef binds Gamma metadata to the exact canonical CLOB token pair.
type MarketRef struct {
	ConditionID string `json:"condition_id"`
	Slug        string `json:"slug"`
	UpTokenID   string `json:"up_token_id"`
	DownTokenID string `json:"down_token_id"`
}

type outcomeClient interface {
	MarketOutcome(context.Context, string, string) (*types.CLOBMarketOutcome, error)
}

type gammaClient interface {
	Markets(context.Context, *types.GetMarketsParams) ([]types.Market, error)
	MarketBySlug(context.Context, string) (*types.Market, error)
}

// Resolver combines Polygolem's typed public CLOB and Gamma clients.
type Resolver struct {
	outcomes     outcomeClient
	gamma        gammaClient
	gammaBaseURL string
	now          func() time.Time
}

const defaultGammaBaseURL = "https://gamma-api.polymarket.com"

// NewResolver creates a read-only production resolver. Empty URLs use the
// Polymarket production defaults.
func NewResolver(clobBaseURL, gammaBaseURL string) *Resolver {
	if strings.TrimSpace(gammaBaseURL) == "" {
		gammaBaseURL = defaultGammaBaseURL
	}
	return newResolver(
		clob.NewClient(clob.Config{BaseURL: clobBaseURL}),
		gamma.NewClient(gammaBaseURL),
		gammaBaseURL,
		time.Now,
	)
}

func newResolver(
	outcomes outcomeClient,
	gamma gammaClient,
	gammaBaseURL string,
	now func() time.Time,
) *Resolver {
	return &Resolver{
		outcomes:     outcomes,
		gamma:        gamma,
		gammaBaseURL: gammaBaseURL,
		now:          now,
	}
}

// Resolve retains the condition-only API for callers that do not have
// canonical token metadata. New collectors should use ResolveMarket.
func (r *Resolver) Resolve(ctx context.Context, conditionID string) (*Result, error) {
	return r.ResolveMarket(ctx, MarketRef{ConditionID: conditionID})
}

// ResolveMarket returns nil until Gamma proves an exact closed 1/0 result for
// the supplied canonical token pair. A resolved CLOB result is cross-checked
// when available; any disagreement fails closed.
func (r *Resolver) ResolveMarket(ctx context.Context, ref MarketRef) (*Result, error) {
	ref.ConditionID = strings.TrimSpace(ref.ConditionID)
	ref.Slug = strings.TrimSpace(ref.Slug)
	if ref.ConditionID == "" {
		return nil, fmt.Errorf("marketresults: condition ID is required")
	}
	var markets []types.Market
	if ref.Slug != "" {
		market, err := r.gamma.MarketBySlug(ctx, ref.Slug)
		if err != nil {
			return nil, fmt.Errorf("marketresults: Gamma market by slug: %w", err)
		}
		if market != nil {
			markets = append(markets, *market)
		}
	} else {
		var err error
		markets, err = r.gamma.Markets(ctx, &types.GetMarketsParams{ConditionIDs: []string{ref.ConditionID}})
		if err != nil {
			return nil, fmt.Errorf("marketresults: Gamma market: %w", err)
		}
	}
	var gammaWinner string
	var resolvedAt time.Time
	for _, market := range markets {
		if market.ConditionID != ref.ConditionID || !market.Closed || market.ClosedTime.IsZero() {
			continue
		}
		gammaWinner = exactGammaWinner(market, ref)
		if gammaWinner != "" {
			resolvedAt = market.ClosedTime.Time().UTC()
			break
		}
	}
	if gammaWinner == "" {
		return nil, nil
	}
	outcome, clobErr := r.outcomes.MarketOutcome(ctx, ref.ConditionID, r.gammaBaseURL)
	if clobErr == nil && outcome != nil && outcome.Status == types.CLOBOutcomeResolved && outcome.Closed && strings.TrimSpace(outcome.WinningTokenID) != "" && outcome.WinningTokenID != gammaWinner {
		return nil, fmt.Errorf("marketresults: CLOB and Gamma winners disagree")
	}
	source := "gamma:closedTime+exact_1_0"
	if clobErr == nil && outcome != nil && outcome.WinningTokenID == gammaWinner {
		source = outcome.Source + "+gamma:closedTime+exact_1_0"
	}
	return &Result{
		ConditionID:    ref.ConditionID,
		WinningTokenID: gammaWinner,
		ResolvedAt:     resolvedAt,
		ObservedAt:     r.now().UTC(),
		Source:         source,
	}, nil
}

func exactGammaWinner(market types.Market, ref MarketRef) string {
	if strings.TrimSpace(ref.UpTokenID) == "" || strings.TrimSpace(ref.DownTokenID) == "" {
		return ""
	}
	var tokenIDs []string
	if err := json.Unmarshal([]byte(market.ClobTokenIDs), &tokenIDs); err != nil || len(tokenIDs) != 2 || len(market.OutcomePrices) != 2 {
		return ""
	}
	winner := -1
	for index, price := range market.OutcomePrices {
		value, err := strconv.ParseFloat(strings.TrimSpace(price), 64)
		if err != nil {
			return ""
		}
		if value == 1 {
			if winner != -1 {
				return ""
			}
			winner = index
		} else if value != 0 {
			return ""
		}
	}
	if winner == -1 || (tokenIDs[winner] != ref.UpTokenID && tokenIDs[winner] != ref.DownTokenID) {
		return ""
	}
	if !((tokenIDs[0] == ref.UpTokenID && tokenIDs[1] == ref.DownTokenID) || (tokenIDs[1] == ref.UpTokenID && tokenIDs[0] == ref.DownTokenID)) {
		return ""
	}
	return tokenIDs[winner]
}
