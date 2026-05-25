// Package discovercrypto searches active crypto prediction markets without Cobra coupling.
package discovercrypto

import (
	"context"

	"github.com/TrebuchetDynamics/polygolem/internal/polytypes"
	"github.com/TrebuchetDynamics/polygolem/internal/workflows/cryptomarkets"
)

// Searcher searches Gamma markets and events.
type Searcher interface {
	Search(ctx context.Context, params *polytypes.SearchParams) (*polytypes.SearchResponse, error)
}

// Pricer enriches CLOB token market data.
type Pricer interface {
	Price(ctx context.Context, tokenID, side string) (string, error)
	Spread(ctx context.Context, tokenID string) (string, error)
}

// Request contains filters for crypto market discovery.
type Request struct {
	Asset    string
	Interval string
	Limit    int
	Enrich   bool
}

// Market is a JSON-friendly crypto market candidate.
type Market struct {
	EventID       string   `json:"event_id"`
	EventTitle    string   `json:"event_title"`
	EventSlug     string   `json:"event_slug"`
	MarketID      string   `json:"market_id"`
	Question      string   `json:"question"`
	ConditionID   string   `json:"condition_id"`
	TokenID       string   `json:"token_id"`
	Outcomes      []string `json:"outcomes"`
	OutcomePrices []string `json:"outcome_prices"`
	EndDate       string   `json:"end_date"`
	Volume24hr    float64  `json:"volume_24h"`
	Price         string   `json:"price,omitempty"`
	Spread        string   `json:"spread,omitempty"`
}

// Response is the JSON-friendly discovery result.
type Response struct {
	Query    string   `json:"query"`
	Asset    string   `json:"asset"`
	Interval string   `json:"interval"`
	Count    int      `json:"count"`
	Markets  []Market `json:"markets"`
}

// Runner owns broad crypto discovery orchestration behind a small interface.
type Runner struct {
	searcher Searcher
	pricer   Pricer
}

// New creates a broad crypto discovery runner.
func New(searcher Searcher, pricer Pricer) *Runner {
	return &Runner{searcher: searcher, pricer: pricer}
}

// Run searches Gamma, filters active crypto markets, and optionally enriches token data.
func (r *Runner) Run(ctx context.Context, req Request) (Response, error) {
	filter := cryptomarkets.Filter{Asset: req.Asset, Interval: req.Interval}
	query := cryptomarkets.Query(filter)
	limit := req.Limit
	resp, err := r.searcher.Search(ctx, &polytypes.SearchParams{
		Q:            query,
		LimitPerType: &limit,
	})
	if err != nil {
		return Response{}, err
	}
	markets := r.markets(ctx, resp, filter, req)
	return Response{
		Query:    query,
		Asset:    req.Asset,
		Interval: req.Interval,
		Count:    len(markets),
		Markets:  markets,
	}, nil
}

func (r *Runner) markets(ctx context.Context, resp *polytypes.SearchResponse, filter cryptomarkets.Filter, req Request) []Market {
	var results []Market
	for _, candidate := range cryptomarkets.Select(resp, filter) {
		event := candidate.Event
		market := candidate.Market
		cm := Market{
			EventID:       event.ID,
			EventTitle:    event.Title,
			EventSlug:     event.Slug,
			MarketID:      market.ID,
			Question:      market.Question,
			ConditionID:   market.ConditionID,
			TokenID:       market.ClobTokenIDs,
			Outcomes:      []string(market.Outcomes),
			OutcomePrices: []string(market.OutcomePrices),
			EndDate:       market.EndDateISO,
			Volume24hr:    market.Volume24hr,
		}
		if req.Enrich && r.pricer != nil && len(candidate.TokenIDs) > 0 {
			if price, err := r.pricer.Price(ctx, candidate.TokenIDs[0], "BUY"); err == nil {
				cm.Price = price
			}
			if spread, err := r.pricer.Spread(ctx, candidate.TokenIDs[0]); err == nil {
				cm.Spread = spread
			}
		}
		results = append(results, cm)
	}
	return results
}
