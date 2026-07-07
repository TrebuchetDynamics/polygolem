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
	Assets     []string
	Enrich     bool
	HoursAhead int
	Timezone   string
}

// MarketResult is a JSON-friendly per-asset sweep result.
type MarketResult struct {
	Asset            string   `json:"asset"`
	EventID          string   `json:"event_id,omitempty"`
	EventTitle       string   `json:"event_title,omitempty"`
	EventSlug        string   `json:"event_slug,omitempty"`
	MarketID         string   `json:"market_id,omitempty"`
	Question         string   `json:"question,omitempty"`
	ConditionID      string   `json:"condition_id,omitempty"`
	TokenIDs         []string `json:"token_ids,omitempty"`
	Outcomes         []string `json:"outcomes,omitempty"`
	WindowStart      string   `json:"window_start,omitempty"`
	WindowEnd        string   `json:"window_end,omitempty"`
	WindowStartLocal string   `json:"window_start_local,omitempty"`
	WindowEndLocal   string   `json:"window_end_local,omitempty"`
	AcceptingOrders  bool     `json:"accepting_orders"`
	LiquidityClob    float64  `json:"liquidity_clob,omitempty"`
	Volume24hrClob   float64  `json:"volume_24h_clob,omitempty"`
	BestBid          float64  `json:"best_bid,omitempty"`
	BestAsk          float64  `json:"best_ask,omitempty"`
	BookSpread       float64  `json:"book_spread,omitempty"`
	Price            string   `json:"price,omitempty"`
	Spread           string   `json:"spread,omitempty"`
	Status           string   `json:"status"`
	Error            string   `json:"error,omitempty"`
}

// Response is the JSON-friendly crypto-5m sweep result.
type Response struct {
	Interval    string         `json:"interval"`
	WindowStart string         `json:"window_start"`
	Timezone    string         `json:"timezone,omitempty"`
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
	loc, tz, err := loadLocation(req.Timezone)
	if err != nil {
		return Response{}, err
	}

	windows := req.HoursAhead*12 + 1
	if windows < 1 {
		windows = 1
	}
	results := make([]MarketResult, 0, len(assets)*windows)
	for i := 0; i < windows; i++ {
		start := windowStart.Add(time.Duration(i) * 5 * time.Minute)
		for _, asset := range assets {
			results = append(results, r.resolveAsset(ctx, asset, start, req.Enrich, loc))
		}
	}
	return Response{
		Interval:    "5m",
		WindowStart: windowStart.UTC().Format(time.RFC3339),
		Timezone:    tz,
		Assets:      assets,
		Count:       len(results),
		Markets:     results,
	}, nil
}

func (r *Runner) resolveAsset(ctx context.Context, asset string, windowStart time.Time, enrich bool, loc *time.Location) MarketResult {
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
		if !market.Active || market.Closed || !market.AcceptingOrders {
			continue
		}
		tokenIDs := cryptomarkets.ParseTokenIDs(market.ClobTokenIDs)
		result := MarketResult{
			Asset:            asset,
			EventID:          event.ID,
			EventTitle:       event.Title,
			EventSlug:        event.Slug,
			MarketID:         market.ID,
			Question:         market.Question,
			ConditionID:      market.ConditionID,
			TokenIDs:         tokenIDs,
			Outcomes:         []string(market.Outcomes),
			WindowStart:      windowStart.UTC().Format(time.RFC3339),
			WindowEnd:        market.EndDateISO,
			WindowStartLocal: windowStart.In(loc).Format(time.RFC3339),
			WindowEndLocal:   windowStart.Add(5 * time.Minute).In(loc).Format(time.RFC3339),
			AcceptingOrders:  market.AcceptingOrders,
			LiquidityClob:    market.LiquidityClob,
			Volume24hrClob:   market.Volume24hrClob,
			BestBid:          market.BestBid,
			BestAsk:          market.BestAsk,
			BookSpread:       market.Spread,
			Status:           "active",
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

func loadLocation(name string) (*time.Location, string, error) {
	name = strings.TrimSpace(name)
	if name == "" || strings.EqualFold(name, "UTC") {
		return time.UTC, "UTC", nil
	}
	if strings.EqualFold(name, "local") {
		return time.Local, "Local", nil
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return nil, "", fmt.Errorf("invalid timezone %q (use IANA names like America/Chicago)", name)
	}
	return loc, name, nil
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
