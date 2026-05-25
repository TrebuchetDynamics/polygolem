// Package datareads owns public Data API command behavior without Cobra coupling.
package datareads

import (
	"context"
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/polygolem/internal/dataapi"
)

// Operation selects one public Data API read.
type Operation string

const (
	Positions       Operation = "positions"
	ClosedPositions Operation = "closed-positions"
	Trades          Operation = "trades"
	Activity        Operation = "activity"
	Holders         Operation = "holders"
	Value           Operation = "value"
	MarketsTraded   Operation = "markets-traded"
	OpenInterest    Operation = "open-interest"
	Leaderboard     Operation = "leaderboard"
	LiveVolume      Operation = "live-volume"
)

// Request contains one public Data API read request.
type Request struct {
	Operation Operation
	User      string
	TokenID   string
	Limit     int
}

// Reader is the Data API adapter used by this workflow.
type Reader interface {
	CurrentPositions(context.Context, string) ([]dataapi.Position, error)
	ClosedPositions(context.Context, string) ([]dataapi.ClosedPosition, error)
	Trades(context.Context, string, int) ([]dataapi.Trade, error)
	Activity(context.Context, string, int) ([]dataapi.Activity, error)
	TopHolders(context.Context, string, int) ([]dataapi.MetaHolder, error)
	TotalValue(context.Context, string) (*dataapi.TotalValue, error)
	MarketsTraded(context.Context, string) (*dataapi.TotalMarketsTraded, error)
	OpenInterest(context.Context, string) (*dataapi.OpenInterest, error)
	TraderLeaderboard(context.Context, int) ([]dataapi.TraderLeaderboardEntry, error)
	LiveVolume(context.Context, int) (*dataapi.LiveVolumeResponse, error)
}

// Runner executes public Data API reads.
type Runner struct {
	reader Reader
}

// New creates a public Data API reads workflow runner.
func New(reader Reader) *Runner {
	return &Runner{reader: reader}
}

// Run executes req and returns the command payload shape.
func (r *Runner) Run(ctx context.Context, req Request) (any, error) {
	switch req.Operation {
	case Positions:
		if err := requireUser(req.User); err != nil {
			return nil, err
		}
		return r.reader.CurrentPositions(ctx, req.User)
	case ClosedPositions:
		if err := requireUser(req.User); err != nil {
			return nil, err
		}
		return r.reader.ClosedPositions(ctx, req.User)
	case Trades:
		if err := requireUser(req.User); err != nil {
			return nil, err
		}
		return r.reader.Trades(ctx, req.User, req.Limit)
	case Activity:
		if err := requireUser(req.User); err != nil {
			return nil, err
		}
		return r.reader.Activity(ctx, req.User, req.Limit)
	case Holders:
		if err := requireToken(req.TokenID); err != nil {
			return nil, err
		}
		return r.reader.TopHolders(ctx, req.TokenID, req.Limit)
	case Value:
		if err := requireUser(req.User); err != nil {
			return nil, err
		}
		return r.reader.TotalValue(ctx, req.User)
	case MarketsTraded:
		if err := requireUser(req.User); err != nil {
			return nil, err
		}
		return r.reader.MarketsTraded(ctx, req.User)
	case OpenInterest:
		if err := requireToken(req.TokenID); err != nil {
			return nil, err
		}
		return r.reader.OpenInterest(ctx, req.TokenID)
	case Leaderboard:
		return r.reader.TraderLeaderboard(ctx, req.Limit)
	case LiveVolume:
		return r.reader.LiveVolume(ctx, req.Limit)
	default:
		return nil, fmt.Errorf("unknown data operation %q", req.Operation)
	}
}

func requireUser(user string) error {
	if strings.TrimSpace(user) == "" {
		return fmt.Errorf("--user required")
	}
	return nil
}

func requireToken(tokenID string) error {
	if strings.TrimSpace(tokenID) == "" {
		return fmt.Errorf("--token-id required")
	}
	return nil
}
