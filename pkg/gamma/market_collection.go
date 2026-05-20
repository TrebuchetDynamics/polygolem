package gamma

import (
	"context"
	"strings"

	"github.com/TrebuchetDynamics/polygolem/pkg/pagination"
	"github.com/TrebuchetDynamics/polygolem/pkg/types"
)

const (
	defaultMarketPageSize = 100
	defaultMaxMarketPages = 50
)

// ActiveMarketsAll collects active, non-closed Gamma markets across offset
// pages and deduplicates by condition ID. It is intended for discovery/indexing
// surfaces that need more than the first Gamma page.
func (c *Client) ActiveMarketsAll(ctx context.Context) ([]types.Market, error) {
	active := true
	closed := false
	items, err := pagination.CollectOffset(ctx, func(ctx context.Context, offset, limit int) ([]types.Market, int, error) {
		if offset >= defaultMarketPageSize*defaultMaxMarketPages {
			return []types.Market{}, 0, nil
		}
		markets, err := c.Markets(ctx, &types.GetMarketsParams{
			Active: &active,
			Closed: &closed,
			Limit:  limit,
			Offset: offset,
		})
		if err != nil {
			return nil, 0, err
		}
		return markets, len(markets), nil
	}, defaultMarketPageSize)
	if err != nil {
		return nil, err
	}
	return DeduplicateMarketsByConditionID(items), nil
}

// DeduplicateMarketsByConditionID returns markets in input order, dropping
// markets with empty or repeated condition IDs.
func DeduplicateMarketsByConditionID(markets []types.Market) []types.Market {
	seen := make(map[string]struct{}, len(markets))
	out := make([]types.Market, 0, len(markets))
	for _, market := range markets {
		conditionID := strings.TrimSpace(market.ConditionID)
		if conditionID == "" {
			continue
		}
		if _, ok := seen[conditionID]; ok {
			continue
		}
		seen[conditionID] = struct{}{}
		out = append(out, market)
	}
	return out
}

// FilterMarketsByCategory applies Polymarket-style category aliases over a
// market slice. Empty and "All" selections return the input markets.
func FilterMarketsByCategory(markets []types.Market, category string) []types.Market {
	selected := strings.TrimSpace(strings.ToLower(category))
	if selected == "" || selected == "all" {
		return markets
	}
	out := make([]types.Market, 0, len(markets))
	for _, market := range markets {
		if MarketMatchesCategory(market.Category, selected) {
			out = append(out, market)
		}
	}
	return out
}

// MarketMatchesCategory reports whether a provider market category matches a
// user-facing Polymarket-like category label.
func MarketMatchesCategory(marketCategory, selectedCategory string) bool {
	market := strings.TrimSpace(strings.ToLower(marketCategory))
	selected := strings.TrimSpace(strings.ToLower(selectedCategory))
	if selected == "" || selected == "all" {
		return true
	}
	if market != "" && (strings.Contains(market, selected) || strings.Contains(selected, market)) {
		return true
	}
	for _, alias := range categoryAliases(selected) {
		if strings.Contains(market, alias) {
			return true
		}
	}
	return false
}

func categoryAliases(category string) []string {
	switch category {
	case "finance", "economy":
		return []string{"finance", "business", "economy", "markets"}
	case "technology", "tech":
		return []string{"technology", "tech", "science", "ai"}
	case "entertainment", "culture", "pop culture":
		return []string{"entertainment", "culture", "pop culture", "movies"}
	case "elections":
		return []string{"elections", "election", "politics"}
	case "world":
		return []string{"world", "global", "geopolitics", "politics"}
	case "weather":
		return []string{"weather", "climate", "science"}
	default:
		return []string{category}
	}
}
