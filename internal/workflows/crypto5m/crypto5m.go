// Package crypto5m resolves the active 5-minute crypto market sweep without Cobra coupling.
package crypto5m

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/polygolem/internal/polytypes"
	"github.com/TrebuchetDynamics/polygolem/internal/workflows/cryptomarkets"
	"github.com/TrebuchetDynamics/polygolem/pkg/marketresolver"
)

// SupportedAssets is the canonical crypto-5m sweep order.
var SupportedAssets = []string{"BTC", "ETH", "SOL", "XRP", "BNB", "DOGE", "HYPE"}

// EventFetcher fetches a Gamma event by deterministic slug.
type EventFetcher interface {
	EventBySlug(ctx context.Context, slug string) (*polytypes.Event, error)
}

// Pricer enriches CLOB token market data.
type Pricer interface {
	Price(ctx context.Context, tokenID, side string) (string, error)
	Spread(ctx context.Context, tokenID string) (string, error)
}

// Request describes one crypto-5m sweep.
type Request struct {
	Assets []string
	Enrich bool
}

// MarketResult is a JSON-friendly per-asset sweep result.
type MarketResult struct {
	Asset       string   `json:"asset"`
	EventID     string   `json:"event_id,omitempty"`
	EventTitle  string   `json:"event_title,omitempty"`
	EventSlug   string   `json:"event_slug,omitempty"`
	MarketID    string   `json:"market_id,omitempty"`
	Question    string   `json:"question,omitempty"`
	ConditionID string   `json:"condition_id,omitempty"`
	TokenIDs    []string `json:"token_ids,omitempty"`
	Outcomes    []string `json:"outcomes,omitempty"`
	WindowStart string   `json:"window_start,omitempty"`
	WindowEnd   string   `json:"window_end,omitempty"`
	Price       string   `json:"price,omitempty"`
	Spread      string   `json:"spread,omitempty"`
	Status      string   `json:"status"`
	Error       string   `json:"error,omitempty"`
}

// Response is the JSON-friendly crypto-5m sweep result.
type Response struct {
	Interval    string         `json:"interval"`
	WindowStart string         `json:"window_start"`
	Assets      []string       `json:"assets"`
	Count       int            `json:"count"`
	Markets     []MarketResult `json:"markets"`
}

// Runner owns crypto-5m sweep orchestration behind a small interface.
type Runner struct {
	events EventFetcher
	pricer Pricer
	Now    func() time.Time
}

// New creates a crypto-5m runner.
func New(events EventFetcher, pricer Pricer) *Runner {
	return &Runner{
		events: events,
		pricer: pricer,
		Now:    func() time.Time { return time.Now().UTC() },
	}
}

// Run resolves the current 5-minute window for every requested asset.
func (r *Runner) Run(ctx context.Context, req Request) (Response, error) {
	assets := append([]string(nil), req.Assets...)
	if len(assets) == 0 {
		assets = append([]string(nil), SupportedAssets...)
	}
	windowStart, err := windowStartAt("5m", r.now())
	if err != nil {
		return Response{}, err
	}

	results := make([]MarketResult, 0, len(assets))
	for _, asset := range assets {
		results = append(results, r.resolveAsset(ctx, asset, windowStart, req.Enrich))
	}
	return Response{
		Interval:    "5m",
		WindowStart: windowStart.UTC().Format(time.RFC3339),
		Assets:      assets,
		Count:       len(results),
		Markets:     results,
	}, nil
}

func (r *Runner) resolveAsset(ctx context.Context, asset string, windowStart time.Time, enrich bool) MarketResult {
	asset = strings.TrimSpace(asset)
	slug := marketresolver.CryptoWindowSlug(asset, "5m", windowStart)
	if slug == "" {
		return MarketResult{Asset: asset, Status: "error", Error: "unsupported asset"}
	}

	event, err := r.events.EventBySlug(ctx, slug)
	if err != nil {
		return MarketResult{Asset: asset, Status: "not_found", Error: err.Error()}
	}
	if event == nil {
		return MarketResult{Asset: asset, Status: "no_active_market"}
	}
	for _, market := range event.Markets {
		if !market.Active || market.Closed {
			continue
		}
		tokenIDs := cryptomarkets.ParseTokenIDs(market.ClobTokenIDs)
		result := MarketResult{
			Asset:       asset,
			EventID:     event.ID,
			EventTitle:  event.Title,
			EventSlug:   event.Slug,
			MarketID:    market.ID,
			Question:    market.Question,
			ConditionID: market.ConditionID,
			TokenIDs:    tokenIDs,
			Outcomes:    []string(market.Outcomes),
			WindowStart: windowStart.UTC().Format(time.RFC3339),
			WindowEnd:   market.EndDateISO,
			Status:      "active",
		}
		if enrich && len(tokenIDs) > 0 && r.pricer != nil {
			if price, err := r.pricer.Price(ctx, tokenIDs[0], "BUY"); err == nil {
				result.Price = price
			}
			if spread, err := r.pricer.Spread(ctx, tokenIDs[0]); err == nil {
				result.Spread = spread
			}
		}
		return result
	}
	return MarketResult{Asset: asset, Status: "no_active_market"}
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
	default:
		return time.Time{}, fmt.Errorf("unsupported interval: %s (use 5m)", interval)
	}
	unix := now.UTC().Unix()
	windowUnix := unix - (unix % seconds)
	return time.Unix(windowUnix, 0).UTC(), nil
}
