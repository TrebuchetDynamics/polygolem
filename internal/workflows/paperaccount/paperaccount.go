// Package paperaccount owns local paper-account actions without Cobra coupling.
package paperaccount

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/TrebuchetDynamics/polygolem/internal/paper"
)

// Pricer fetches a CLOB token price for a side.
type Pricer interface {
	Price(ctx context.Context, tokenID, side string) (string, error)
}

// Config wires local paper-account state and optional market pricing.
type Config struct {
	State  *paper.State
	Pricer Pricer
}

// TradeRequest contains a local paper buy/sell request.
type TradeRequest struct {
	TokenID string
	Price   string
	Size    string
}

// TradeResponse is the JSON-friendly result of a local paper buy/sell.
type TradeResponse struct {
	Action      string     `json:"action"`
	TokenID     string     `json:"token_id"`
	Price       float64    `json:"price"`
	Size        float64    `json:"size"`
	Cost        float64    `json:"cost,omitempty"`
	Proceeds    float64    `json:"proceeds,omitempty"`
	RealizedPnL float64    `json:"realized_pnl,omitempty"`
	Cash        float64    `json:"cash"`
	Fill        paper.Fill `json:"fill"`
}

// PositionsResponse is the JSON-friendly local paper-account snapshot.
type PositionsResponse struct {
	Cash      float64                   `json:"cash"`
	Positions map[string]paper.Position `json:"positions"`
	Fills     []paper.Fill              `json:"fills"`
}

// ResetResponse is the JSON-friendly result of resetting local paper state.
type ResetResponse struct {
	Status string  `json:"status"`
	Cash   float64 `json:"cash"`
}

// Runner executes local paper-account actions behind a small interface.
type Runner struct {
	state  *paper.State
	pricer Pricer
}

// New creates a local paper-account workflow runner.
func New(cfg Config) *Runner {
	return &Runner{state: cfg.State, pricer: cfg.Pricer}
}

// Buy records a local paper buy, using best ask pricing when no price is set.
func (r *Runner) Buy(ctx context.Context, req TradeRequest) (TradeResponse, error) {
	return r.trade(ctx, "buy", "SELL", req)
}

// Sell records a local paper sell-shaped response while preserving existing local accounting.
func (r *Runner) Sell(ctx context.Context, req TradeRequest) (TradeResponse, error) {
	return r.trade(ctx, "sell", "BUY", req)
}

// Positions returns the current local paper-account snapshot.
func (r *Runner) Positions() PositionsResponse {
	return PositionsResponse{Cash: r.state.Cash, Positions: r.state.Positions, Fills: r.state.Fills}
}

// Reset replaces the current local paper-account state in place.
func (r *Runner) Reset(cash float64) ResetResponse {
	*r.state = *paper.NewState("USD", cash)
	return ResetResponse{Status: "reset", Cash: cash}
}

func (r *Runner) trade(ctx context.Context, action, clobSide string, req TradeRequest) (TradeResponse, error) {
	if r.state == nil {
		return TradeResponse{}, fmt.Errorf("paper state is required")
	}
	tokenID := strings.TrimSpace(req.TokenID)
	if tokenID == "" {
		return TradeResponse{}, fmt.Errorf("--token-id required")
	}

	price, err := r.price(ctx, tokenID, clobSide, req.Price)
	if err != nil {
		return TradeResponse{}, err
	}
	size, err := parseSize(req.Size)
	if err != nil {
		return TradeResponse{}, err
	}

	order := paper.Order{TokenID: tokenID, Price: price, Size: size}
	var fill paper.Fill
	if action == "sell" {
		fill, err = r.state.Sell(order)
	} else {
		fill, err = r.state.Buy(order)
	}
	if err != nil {
		return TradeResponse{}, err
	}
	res := TradeResponse{Action: action, TokenID: tokenID, Price: price, Size: size, Cash: r.state.Cash, Fill: fill}
	if action == "sell" {
		res.Proceeds = price * size
		res.RealizedPnL = fill.RealizedPnL
	} else {
		res.Cost = price * size
	}
	return res, nil
}

func (r *Runner) price(ctx context.Context, tokenID, clobSide, priceValue string) (float64, error) {
	priceValue = strings.TrimSpace(priceValue)
	if priceValue != "" {
		price, err := strconv.ParseFloat(priceValue, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid price: %w", err)
		}
		return price, nil
	}

	price := 0.5
	if r.pricer == nil {
		return price, nil
	}
	quoted, err := r.pricer.Price(ctx, tokenID, clobSide)
	if err != nil {
		return price, nil
	}
	parsed, err := strconv.ParseFloat(quoted, 64)
	if err != nil {
		return price, nil
	}
	return parsed, nil
}

func parseSize(value string) (float64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 1.0, nil
	}
	size, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size: %w", err)
	}
	return size, nil
}
