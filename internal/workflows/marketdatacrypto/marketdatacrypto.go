// Package marketdatacrypto builds crypto market-data snapshots without Cobra coupling.
package marketdatacrypto

import (
	"context"

	"github.com/TrebuchetDynamics/polygolem/internal/polytypes"
	"github.com/TrebuchetDynamics/polygolem/internal/workflows/cryptomarkets"
)

const searchLimit = 50

// Searcher searches Gamma markets and events.
type Searcher interface {
	Search(ctx context.Context, params *polytypes.SearchParams) (*polytypes.SearchResponse, error)
}

// Quoter reads CLOB quote metadata for one token.
type Quoter interface {
	Price(ctx context.Context, tokenID, side string) (string, error)
	Spread(ctx context.Context, tokenID string) (string, error)
	Midpoint(ctx context.Context, tokenID string) (string, error)
	TickSize(ctx context.Context, tokenID string) (*polytypes.TickSize, error)
}

// Request contains filters for crypto market-data snapshots.
type Request struct {
	Asset    string
	Interval string
	Limit    int
}

// Snapshot is a JSON-friendly point-in-time view of one crypto market token.
type Snapshot struct {
	EventID         string  `json:"event_id"`
	EventTitle      string  `json:"event_title"`
	MarketID        string  `json:"market_id"`
	Question        string  `json:"question"`
	TokenID         string  `json:"token_id"`
	Outcome         string  `json:"outcome"`
	Price           string  `json:"price"`
	Spread          string  `json:"spread"`
	Midpoint        string  `json:"midpoint"`
	TickSize        string  `json:"tick_size"`
	Volume24hr      float64 `json:"volume_24h"`
	AcceptingOrders bool    `json:"accepting_orders"`
	EndDate         string  `json:"end_date"`
}

// Response is the JSON-friendly crypto market-data result.
type Response struct {
	Query    string     `json:"query"`
	Asset    string     `json:"asset"`
	Interval string     `json:"interval"`
	Count    int        `json:"count"`
	Markets  []Snapshot `json:"markets"`
}

// Runner owns crypto market-data orchestration behind a small interface.
type Runner struct {
	searcher Searcher
	quoter   Quoter
}

// New creates a crypto market-data runner.
func New(searcher Searcher, quoter Quoter) *Runner {
	return &Runner{searcher: searcher, quoter: quoter}
}

// Run searches Gamma, filters active crypto markets, and fetches one CLOB snapshot per market.
func (r *Runner) Run(ctx context.Context, req Request) (Response, error) {
	filter := cryptomarkets.Filter{Asset: req.Asset, Interval: req.Interval}
	query := cryptomarkets.Query(filter)
	limit := searchLimit
	resp, err := r.searcher.Search(ctx, &polytypes.SearchParams{
		Q:            query,
		LimitPerType: &limit,
	})
	if err != nil {
		return Response{}, err
	}
	snapshots := r.snapshots(ctx, resp, filter, req)
	return Response{
		Query:    query,
		Asset:    req.Asset,
		Interval: req.Interval,
		Count:    len(snapshots),
		Markets:  snapshots,
	}, nil
}

func (r *Runner) snapshots(ctx context.Context, resp *polytypes.SearchResponse, filter cryptomarkets.Filter, req Request) []Snapshot {
	var results []Snapshot
	for _, candidate := range cryptomarkets.Select(resp, filter) {
		event := candidate.Event
		market := candidate.Market
		if len(candidate.TokenIDs) == 0 {
			continue
		}
		snapshot := Snapshot{
			EventID:         event.ID,
			EventTitle:      event.Title,
			MarketID:        market.ID,
			Question:        market.Question,
			TokenID:         candidate.TokenIDs[0],
			Volume24hr:      market.Volume24hr,
			AcceptingOrders: market.AcceptingOrders,
			EndDate:         market.EndDateISO,
		}
		if outcomes := []string(market.Outcomes); len(outcomes) > 0 {
			snapshot.Outcome = outcomes[0]
		}
		if r.quoter != nil {
			fillQuote(ctx, r.quoter, &snapshot)
		}

		results = append(results, snapshot)
		// Only cap when a positive limit is set; req.Limit == 0 means "no limit"
		// (otherwise the >= comparison would truncate to a single snapshot).
		if req.Limit > 0 && len(results) >= req.Limit {
			return results
		}
	}
	return results
}

func fillQuote(ctx context.Context, quoter Quoter, snapshot *Snapshot) {
	if price, err := quoter.Price(ctx, snapshot.TokenID, "BUY"); err == nil {
		snapshot.Price = price
	}
	if spread, err := quoter.Spread(ctx, snapshot.TokenID); err == nil {
		snapshot.Spread = spread
	}
	if midpoint, err := quoter.Midpoint(ctx, snapshot.TokenID); err == nil {
		snapshot.Midpoint = midpoint
	}
	if tick, err := quoter.TickSize(ctx, snapshot.TokenID); err == nil && tick != nil {
		snapshot.TickSize = tick.MinimumTickSize
	}
}
