// Package papertrade runs one-command paper trades without Cobra coupling.
package papertrade

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/polygolem/internal/paper"
	"github.com/TrebuchetDynamics/polygolem/pkg/marketresolver"
)

// Resolver resolves a crypto decision window to outcome token IDs.
type Resolver interface {
	ResolveTokenIDsForWindow(ctx context.Context, asset, timeframe string, windowStart time.Time) marketresolver.ResolveResult
}

// Pricer fetches a CLOB token price for a side.
type Pricer interface {
	Price(ctx context.Context, tokenID, side string) (string, error)
}

// Request contains the inputs needed to execute one paper trade.
type Request struct {
	Asset    string
	Interval string
	Side     string
	TokenID  string
	Price    string
	Size     float64
}

// Response is the JSON-friendly result returned by a paper trade.
type Response struct {
	Action    string     `json:"action"`
	Asset     string     `json:"asset"`
	Interval  string     `json:"interval"`
	Side      string     `json:"side"`
	TokenID   string     `json:"token_id"`
	Price     float64    `json:"price"`
	Size      float64    `json:"size"`
	Cost      float64    `json:"cost"`
	Cash      float64    `json:"cash"`
	Fill      paper.Fill `json:"fill"`
	Timestamp string     `json:"timestamp"`
}

// Runner owns paper-trade orchestration behind a small interface.
type Runner struct {
	resolver Resolver
	pricer   Pricer
	state    *paper.State
	Now      func() time.Time
}

// New creates a paper-trade runner.
func New(resolver Resolver, pricer Pricer, state *paper.State) *Runner {
	return &Runner{
		resolver: resolver,
		pricer:   pricer,
		state:    state,
		Now:      func() time.Time { return time.Now().UTC() },
	}
}

// Run resolves the target token, chooses a price, and records a local paper fill.
func (r *Runner) Run(ctx context.Context, req Request) (Response, error) {
	if r.state == nil {
		return Response{}, fmt.Errorf("paper state is required")
	}

	side := strings.ToLower(strings.TrimSpace(req.Side))
	tokenID := strings.TrimSpace(req.TokenID)
	if tokenID == "" {
		resolvedTokenID, err := r.resolveTokenID(ctx, req.Asset, req.Interval, side)
		if err != nil {
			return Response{}, err
		}
		tokenID = resolvedTokenID
	}

	price, err := r.price(ctx, tokenID, side, req.Price)
	if err != nil {
		return Response{}, err
	}
	size := req.Size
	if size == 0 {
		size = 1.0
	}

	fill, err := r.state.Buy(paper.Order{
		TokenID: tokenID,
		Price:   price,
		Size:    size,
	})
	if err != nil {
		return Response{}, err
	}

	now := r.now().UTC()
	return Response{
		Action:    "paper_trade",
		Asset:     req.Asset,
		Interval:  req.Interval,
		Side:      req.Side,
		TokenID:   tokenID,
		Price:     price,
		Size:      size,
		Cost:      price * size,
		Cash:      r.state.Cash,
		Fill:      fill,
		Timestamp: now.Format(time.RFC3339),
	}, nil
}

func (r *Runner) resolveTokenID(ctx context.Context, asset, interval, side string) (string, error) {
	asset = strings.TrimSpace(asset)
	interval = strings.TrimSpace(interval)
	if asset == "" {
		return "", fmt.Errorf("--asset required (or use --token-id)")
	}
	if interval == "" {
		return "", fmt.Errorf("--interval required (or use --token-id)")
	}
	if side == "" {
		return "", fmt.Errorf("--side required (up or down)")
	}
	if r.resolver == nil {
		return "", fmt.Errorf("market resolver is required")
	}

	windowStart, err := windowStartAt(interval, r.now())
	if err != nil {
		return "", err
	}
	result := r.resolver.ResolveTokenIDsForWindow(ctx, asset, interval, windowStart)
	if result.Status != marketresolver.StatusAvailable {
		return "", fmt.Errorf("window not found: asset=%s interval=%s status=%s source=%s", asset, interval, result.Status, result.Source)
	}

	switch side {
	case "up", "yes":
		if result.UpTokenID == "" {
			return "", fmt.Errorf("no active market with %s outcome found for %s %s", side, asset, interval)
		}
		return result.UpTokenID, nil
	case "down", "no":
		if result.DownTokenID == "" {
			return "", fmt.Errorf("no active market with %s outcome found for %s %s", side, asset, interval)
		}
		return result.DownTokenID, nil
	default:
		return "", fmt.Errorf("--side must be up or down")
	}
}

func (r *Runner) price(ctx context.Context, tokenID, side, priceValue string) (float64, error) {
	priceValue = strings.TrimSpace(priceValue)
	if priceValue != "" {
		price, err := strconv.ParseFloat(priceValue, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid price: %w", err)
		}
		return price, nil
	}

	price := 0.5
	clobSide := "SELL"
	if side == "down" || side == "no" {
		clobSide = "BUY"
	}
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

func (r *Runner) now() time.Time {
	if r.Now == nil {
		return time.Now().UTC()
	}
	return r.Now()
}

func windowStartAt(interval string, now time.Time) (time.Time, error) {
	var seconds int64
	switch interval {
	case "5m":
		seconds = 300
	case "15m":
		seconds = 900
	case "1h":
		seconds = 3600
	case "4h":
		seconds = 14400
	default:
		return time.Time{}, fmt.Errorf("unsupported interval: %s (use 5m, 15m, 1h, 4h)", interval)
	}
	unix := now.UTC().Unix()
	windowUnix := unix - (unix % seconds)
	return time.Unix(windowUnix, 0).UTC(), nil
}
