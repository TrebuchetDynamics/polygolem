// Package orderbookreads owns read-only CLOB order-book command behavior without Cobra coupling.
package orderbookreads

import (
	"context"
	"fmt"
	"strconv"

	"github.com/TrebuchetDynamics/polygolem/internal/polytypes"
)

// Operation selects one read-only order-book request.
type Operation string

const (
	Get       Operation = "get"
	Price     Operation = "price"
	Midpoint  Operation = "midpoint"
	Spread    Operation = "spread"
	TickSize  Operation = "tick-size"
	FeeRate   Operation = "fee-rate"
	LastTrade Operation = "last-trade"
)

// Request contains one read-only order-book request.
type Request struct {
	Operation Operation
	TokenID   string
}

// Reader is the CLOB read adapter used by this workflow.
type Reader interface {
	OrderBook(context.Context, string) (*polytypes.OrderBook, error)
	Price(context.Context, string, string) (string, error)
	Midpoint(context.Context, string) (string, error)
	Spread(context.Context, string) (string, error)
	TickSize(context.Context, string) (*polytypes.TickSize, error)
	FeeRateBps(context.Context, string) (int, error)
	LastTradePrice(context.Context, string) (string, error)
}

// Runner executes read-only order-book requests.
type Runner struct {
	reader Reader
}

// New creates an order-book reads workflow runner.
func New(reader Reader) *Runner {
	return &Runner{reader: reader}
}

// Run executes req and returns the command payload shape.
func (r *Runner) Run(ctx context.Context, req Request) (any, error) {
	if req.TokenID == "" {
		return nil, fmt.Errorf("--token-id required")
	}
	switch req.Operation {
	case Get:
		return r.reader.OrderBook(ctx, req.TokenID)
	case Price:
		p, err := r.reader.Price(ctx, req.TokenID, "BUY")
		return map[string]string{"token_id": req.TokenID, "price": p}, err
	case Midpoint:
		m, err := r.reader.Midpoint(ctx, req.TokenID)
		return map[string]string{"token_id": req.TokenID, "midpoint": m}, err
	case Spread:
		s, err := r.reader.Spread(ctx, req.TokenID)
		return map[string]string{"token_id": req.TokenID, "spread": s}, err
	case TickSize:
		return r.reader.TickSize(ctx, req.TokenID)
	case FeeRate:
		f, err := r.reader.FeeRateBps(ctx, req.TokenID)
		return map[string]string{"token_id": req.TokenID, "fee_rate_bps": strconv.Itoa(f)}, err
	case LastTrade:
		p, err := r.reader.LastTradePrice(ctx, req.TokenID)
		return map[string]string{"token_id": req.TokenID, "price": p}, err
	default:
		return nil, fmt.Errorf("unknown orderbook operation %q", req.Operation)
	}
}
