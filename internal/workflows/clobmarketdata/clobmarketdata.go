// Package clobmarketdata reads CLOB market-data without Cobra coupling.
package clobmarketdata

import (
	"context"
	"fmt"

	"github.com/TrebuchetDynamics/polygolem/internal/polytypes"
)

// Reader is the read-only CLOB market-data interface used by the workflow.
type Reader interface {
	OrderBook(context.Context, string) (*polytypes.OrderBook, error)
	TickSize(context.Context, string) (*polytypes.TickSize, error)
	PricesHistory(context.Context, *polytypes.PriceHistoryParams) (*polytypes.PriceHistory, error)
	Market(context.Context, string) (*polytypes.CLOBMarket, error)
	MarketByToken(context.Context, string) (*polytypes.CLOBMarketByTokenResponse, error)
	Markets(context.Context, string) (*polytypes.CLOBPaginatedMarkets, error)
}

// TokenRequest describes a token-scoped read.
type TokenRequest struct {
	TokenID string
	Output  string
}

// PriceHistoryRequest describes a token price-history read.
type PriceHistoryRequest struct {
	TokenID  string
	Interval string
	Output   string
	Fidelity int
	StartTS  int64
	EndTS    int64
}

// ConditionRequest describes a condition-id scoped read.
type ConditionRequest struct {
	ConditionID string
	Output      string
}

// MarketsRequest describes a paginated CLOB markets read.
type MarketsRequest struct {
	Cursor string
	Output string
}

// Runner owns read-only CLOB market-data orchestration behind a small interface.
type Runner struct {
	reader Reader
}

// New creates a read-only CLOB market-data workflow runner.
func New(reader Reader) *Runner {
	return &Runner{reader: reader}
}

// Book returns L2 order-book depth for a CLOB token.
func (r *Runner) Book(ctx context.Context, req TokenRequest) (*polytypes.OrderBook, error) {
	if err := checkOutput(req.Output); err != nil {
		return nil, err
	}
	return r.reader.OrderBook(ctx, req.TokenID)
}

// TickSize returns tick-size metadata for a CLOB token.
func (r *Runner) TickSize(ctx context.Context, req TokenRequest) (*polytypes.TickSize, error) {
	if err := checkOutput(req.Output); err != nil {
		return nil, err
	}
	return r.reader.TickSize(ctx, req.TokenID)
}

// PriceHistory returns OHLCV price history for a CLOB token.
func (r *Runner) PriceHistory(ctx context.Context, req PriceHistoryRequest) (*polytypes.PriceHistory, error) {
	if err := checkOutput(req.Output); err != nil {
		return nil, err
	}
	return r.reader.PricesHistory(ctx, &polytypes.PriceHistoryParams{
		Market:   req.TokenID,
		Interval: req.Interval,
		Fidelity: req.Fidelity,
		StartTS:  req.StartTS,
		EndTS:    req.EndTS,
	})
}

// Market returns a single CLOB market by condition ID.
func (r *Runner) Market(ctx context.Context, req ConditionRequest) (*polytypes.CLOBMarket, error) {
	if err := checkOutput(req.Output); err != nil {
		return nil, err
	}
	return r.reader.Market(ctx, req.ConditionID)
}

// MarketByToken resolves a CLOB token ID to its parent market identifiers.
func (r *Runner) MarketByToken(ctx context.Context, req TokenRequest) (*polytypes.CLOBMarketByTokenResponse, error) {
	if err := checkOutput(req.Output); err != nil {
		return nil, err
	}
	return r.reader.MarketByToken(ctx, req.TokenID)
}

// Markets returns cursor-paginated CLOB markets.
func (r *Runner) Markets(ctx context.Context, req MarketsRequest) (*polytypes.CLOBPaginatedMarkets, error) {
	if err := checkOutput(req.Output); err != nil {
		return nil, err
	}
	return r.reader.Markets(ctx, req.Cursor)
}

func checkOutput(output string) error {
	if output != "" && output != "json" {
		return fmt.Errorf("only --output json is supported")
	}
	return nil
}
