// Package cryptowindow resolves deterministic crypto prediction windows without Cobra coupling.
package cryptowindow

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/polygolem/internal/polytypes"
	"github.com/TrebuchetDynamics/polygolem/internal/workflows/cryptomarkets"
	"github.com/TrebuchetDynamics/polygolem/pkg/marketresolver"
)

// EventFetcher fetches a Gamma event by deterministic slug.
type EventFetcher interface {
	EventBySlug(ctx context.Context, slug string) (*polytypes.Event, error)
}

// Pricer enriches CLOB token market data.
type Pricer interface {
	Price(ctx context.Context, tokenID, side string) (string, error)
	Spread(ctx context.Context, tokenID string) (string, error)
}

// Request describes one deterministic crypto-window lookup.
type Request struct {
	Asset    string
	Interval string
	Enrich   bool
}

// Market is a JSON-friendly active market in the resolved window.
type Market struct {
	EventID       string   `json:"event_id"`
	EventTitle    string   `json:"event_title"`
	EventSlug     string   `json:"event_slug"`
	MarketID      string   `json:"market_id"`
	Question      string   `json:"question"`
	ConditionID   string   `json:"condition_id"`
	TokenIDs      []string `json:"token_ids"`
	Outcomes      []string `json:"outcomes"`
	OutcomePrices []string `json:"outcome_prices"`
	WindowStart   string   `json:"window_start"`
	WindowEnd     string   `json:"window_end"`
	Price         string   `json:"price,omitempty"`
	Spread        string   `json:"spread,omitempty"`
}

// Response is the JSON-friendly deterministic window result.
type Response struct {
	Asset       string   `json:"asset"`
	Interval    string   `json:"interval"`
	WindowStart string   `json:"window_start"`
	Slug        string   `json:"slug"`
	Count       int      `json:"count"`
	Markets     []Market `json:"markets"`
}

// Runner owns deterministic crypto-window orchestration behind a small interface.
type Runner struct {
	events EventFetcher
	pricer Pricer
	Now    func() time.Time
}

// New creates a crypto-window runner.
func New(events EventFetcher, pricer Pricer) *Runner {
	return &Runner{
		events: events,
		pricer: pricer,
		Now:    func() time.Time { return time.Now().UTC() },
	}
}

// Run resolves the current deterministic window and optionally enriches token data.
func (r *Runner) Run(ctx context.Context, req Request) (Response, error) {
	asset := strings.TrimSpace(req.Asset)
	interval := strings.TrimSpace(req.Interval)
	if asset == "" {
		return Response{}, fmt.Errorf("--asset required (BTC, ETH, SOL, etc.)")
	}
	if interval == "" {
		return Response{}, fmt.Errorf("--interval required (5m, 15m, 1h, 4h)")
	}

	windowStart, err := windowStartAt(interval, r.now())
	if err != nil {
		return Response{}, err
	}
	slug := marketresolver.CryptoWindowSlug(asset, interval, windowStart)
	if slug == "" {
		return Response{}, fmt.Errorf("unable to construct slug for asset=%s interval=%s", asset, interval)
	}

	event, err := r.events.EventBySlug(ctx, slug)
	if err != nil {
		return Response{}, fmt.Errorf("window not found (may not be created yet): slug=%s: %w", slug, err)
	}

	markets := r.markets(ctx, event, windowStart, req.Enrich)
	return Response{
		Asset:       asset,
		Interval:    interval,
		WindowStart: windowStart.UTC().Format(time.RFC3339),
		Slug:        slug,
		Count:       len(markets),
		Markets:     markets,
	}, nil
}

func (r *Runner) markets(ctx context.Context, event *polytypes.Event, windowStart time.Time, enrich bool) []Market {
	if event == nil {
		return nil
	}
	var results []Market
	for _, market := range event.Markets {
		if !market.Active || market.Closed {
			continue
		}
		tokenIDs := cryptomarkets.ParseTokenIDs(market.ClobTokenIDs)
		wm := Market{
			EventID:       event.ID,
			EventTitle:    event.Title,
			EventSlug:     event.Slug,
			MarketID:      market.ID,
			Question:      market.Question,
			ConditionID:   market.ConditionID,
			TokenIDs:      tokenIDs,
			Outcomes:      []string(market.Outcomes),
			OutcomePrices: []string(market.OutcomePrices),
			WindowStart:   windowStart.UTC().Format(time.RFC3339),
			WindowEnd:     market.EndDateISO,
		}
		if enrich && len(tokenIDs) > 0 && r.pricer != nil {
			if price, err := r.pricer.Price(ctx, tokenIDs[0], "BUY"); err == nil {
				wm.Price = price
			}
			if spread, err := r.pricer.Spread(ctx, tokenIDs[0]); err == nil {
				wm.Spread = spread
			}
		}
		results = append(results, wm)
	}
	return results
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
