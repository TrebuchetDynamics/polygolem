// Package orderfills defines Polymarket on-chain OrderFilled truth models.
package orderfills

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const (
	SideBUY  = "BUY"
	SideSELL = "SELL"

	SourceOnchainOrderFilled = "onchain_order_filled"
)

type Fill struct {
	TxHash      string
	LogIndex    uint
	Exchange    string
	MarketID    string
	ConditionID string
	TokenID     string
	Side        string
	Price       string
	Size        string
	BlockNumber uint64
	FilledAt    time.Time
	Source      string
}

type Market struct {
	MarketID    string
	ConditionID string
	YesTokenID  string
	NoTokenID   string
}

type Query struct {
	ExchangeAddresses []string
	FromBlock         uint64
	ToBlock           uint64
	MarketID          string
	ConditionIDs      []string
	TokenIDs          []string
	Markets           []Market
}

type Reader interface {
	OrderFilled(ctx context.Context, query Query) ([]Fill, error)
}

type BlockNumberReader interface {
	LatestBlockNumber(ctx context.Context) (uint64, error)
}

func ValidateQuery(query Query) error {
	if query.FromBlock == 0 && query.ToBlock == 0 {
		return fmt.Errorf("orderfills query block range is required")
	}
	if query.FromBlock == 0 {
		return fmt.Errorf("orderfills query from block is required")
	}
	if query.ToBlock == 0 {
		return fmt.Errorf("orderfills query to block is required")
	}
	if query.FromBlock > query.ToBlock {
		return fmt.Errorf("orderfills query from block must be <= to block")
	}
	return nil
}

func NormalizeFill(fill Fill) (Fill, error) {
	side := strings.ToUpper(strings.TrimSpace(fill.Side))
	if side != SideBUY && side != SideSELL {
		return Fill{}, fmt.Errorf("orderfills fill side must be BUY or SELL")
	}
	fill.Side = side

	source := strings.TrimSpace(fill.Source)
	if source == "" {
		source = SourceOnchainOrderFilled
	}
	if source != SourceOnchainOrderFilled {
		return Fill{}, fmt.Errorf("orderfills fill source must be %s", SourceOnchainOrderFilled)
	}
	fill.Source = source

	return fill, nil
}

func ValidateFill(fill Fill) error {
	_, err := NormalizeFill(fill)
	return err
}
