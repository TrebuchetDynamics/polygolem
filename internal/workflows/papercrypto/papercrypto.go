// Package papercrypto discovers crypto markets for paper trading without Cobra coupling.
package papercrypto

import (
	"context"

	"github.com/TrebuchetDynamics/polygolem/internal/polytypes"
	"github.com/TrebuchetDynamics/polygolem/internal/workflows/cryptomarkets"
)

const tradeHelp = "Use 'polygolem paper buy --token-id <ID> --size 1' to paper trade"

// Searcher searches Gamma markets and events.
type Searcher interface {
	Search(ctx context.Context, params *polytypes.SearchParams) (*polytypes.SearchResponse, error)
}

// Request contains filters for paper crypto market discovery.
type Request struct {
	Asset    string
	Interval string
	Limit    int
}

// Market is a JSON-friendly paper-trading market candidate.
type Market struct {
	EventID    string   `json:"event_id"`
	EventTitle string   `json:"event_title"`
	MarketID   string   `json:"market_id"`
	Question   string   `json:"question"`
	TokenID    string   `json:"token_id"`
	Outcomes   []string `json:"outcomes"`
	EndDate    string   `json:"end_date"`
}

// Response is the JSON-friendly discovery result.
type Response struct {
	Query   string   `json:"query"`
	Count   int      `json:"count"`
	Markets []Market `json:"markets"`
	Help    string   `json:"help"`
}

// Runner owns paper-crypto discovery orchestration behind a small interface.
type Runner struct {
	searcher Searcher
}

// New creates a paper-crypto discovery runner.
func New(searcher Searcher) *Runner {
	return &Runner{searcher: searcher}
}

// Run searches Gamma and returns active CLOB-token markets matching the filters.
func (r *Runner) Run(ctx context.Context, req Request) (Response, error) {
	filter := cryptomarkets.Filter{Asset: req.Asset, Interval: req.Interval}
	query := cryptomarkets.Query(filter)
	limit := req.Limit
	if limit == 0 {
		limit = 10
	}

	resp, err := r.searcher.Search(ctx, &polytypes.SearchParams{
		Q:            query,
		LimitPerType: &limit,
	})
	if err != nil {
		return Response{}, err
	}

	markets := filterMarkets(resp, filter)
	return Response{
		Query:   query,
		Count:   len(markets),
		Markets: markets,
		Help:    tradeHelp,
	}, nil
}

func filterMarkets(resp *polytypes.SearchResponse, filter cryptomarkets.Filter) []Market {
	var results []Market
	for _, candidate := range cryptomarkets.Select(resp, filter) {
		if len(candidate.TokenIDs) == 0 {
			continue
		}
		event := candidate.Event
		market := candidate.Market
		results = append(results, Market{
			EventID:    event.ID,
			EventTitle: event.Title,
			MarketID:   market.ID,
			Question:   market.Question,
			TokenID:    candidate.TokenIDs[0],
			Outcomes:   []string(market.Outcomes),
			EndDate:    market.EndDateISO,
		})
	}
	return results
}
