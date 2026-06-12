// Package opportunities scans read-only Polymarket market data for research candidates.
package opportunities

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/polygolem/internal/polytypes"
	"github.com/TrebuchetDynamics/polygolem/internal/workflows/crypto5m"
	"github.com/TrebuchetDynamics/polygolem/internal/workflows/cryptomarkets"
)

// Type selects the opportunity scanner.
type Type string

const (
	TypeWideSpread             Type = "wide-spread"
	TypeLowLiquidityHighVolume Type = "low-liquidity-high-volume"
	TypeNewMarkets             Type = "new-markets"
	TypeClosingSoon            Type = "closing-soon"
	TypeNegativeRisk           Type = "negative-risk"
	TypeCrypto5M               Type = "crypto-5m"
)

// Config wires read adapters used by the scanner.
type Config struct {
	Gamma  MarketLister
	Events crypto5m.EventFetcher
	Pricer crypto5m.Pricer
}

// Request describes one opportunity scan.
type Request struct {
	Type  Type
	Limit int
	Hours int
	Asset string
}

// Response is the JSON-friendly scanner result.
type Response struct {
	Type          Type          `json:"type"`
	Count         int           `json:"count"`
	Opportunities []Opportunity `json:"opportunities"`
}

// Opportunity is one read-only research candidate, not trading advice.
type Opportunity struct {
	Type          Type     `json:"type"`
	MarketID      string   `json:"market_id"`
	Question      string   `json:"question"`
	Slug          string   `json:"slug,omitempty"`
	ConditionID   string   `json:"condition_id,omitempty"`
	TokenIDs      []string `json:"token_ids,omitempty"`
	EndDate       string   `json:"end_date,omitempty"`
	Asset         string   `json:"asset,omitempty"`
	Volume24hr    float64  `json:"volume_24h,omitempty"`
	Liquidity     float64  `json:"liquidity,omitempty"`
	LiquidityClob float64  `json:"liquidity_clob,omitempty"`
	Spread        float64  `json:"spread,omitempty"`
	Price         string   `json:"price,omitempty"`
	SpreadText    string   `json:"clob_spread,omitempty"`
	Reasons       []string `json:"reasons"`
}

// MarketLister lists Gamma markets.
type MarketLister interface {
	Markets(context.Context, *polytypes.GetMarketsParams) ([]polytypes.Market, error)
}

// Runner owns opportunity scanning orchestration.
type Runner struct {
	gamma  MarketLister
	events crypto5m.EventFetcher
	pricer crypto5m.Pricer
	Now    func() time.Time
}

// New creates a read-only opportunity scanner.
func New(cfg Config) *Runner {
	events := cfg.Events
	if events == nil {
		if candidate, ok := cfg.Gamma.(crypto5m.EventFetcher); ok {
			events = candidate
		}
	}
	return &Runner{
		gamma:  cfg.Gamma,
		events: events,
		pricer: cfg.Pricer,
		Now:    func() time.Time { return time.Now().UTC() },
	}
}

// Run executes the selected scanner.
func (r *Runner) Run(ctx context.Context, req Request) (Response, error) {
	scannerType := req.Type
	if scannerType == "" {
		scannerType = TypeWideSpread
	}
	if scannerType == TypeCrypto5M {
		return r.runCrypto5M(ctx, req)
	}
	markets, err := r.activeMarkets(ctx, req.Limit)
	if err != nil {
		return Response{}, err
	}
	var opps []Opportunity
	switch scannerType {
	case TypeWideSpread:
		opps = wideSpread(markets)
	case TypeLowLiquidityHighVolume:
		opps = lowLiquidityHighVolume(markets)
	case TypeNewMarkets:
		opps = newMarkets(markets)
	case TypeClosingSoon:
		opps = closingSoon(markets, r.now(), req.Hours)
	case TypeNegativeRisk:
		opps = negativeRisk(markets)
	default:
		return Response{}, fmt.Errorf("unknown opportunity type %q", scannerType)
	}
	limit := normalizedLimit(req.Limit)
	if len(opps) > limit {
		opps = opps[:limit]
	}
	return Response{Type: scannerType, Count: len(opps), Opportunities: opps}, nil
}

func (r *Runner) activeMarkets(ctx context.Context, limit int) ([]polytypes.Market, error) {
	active, closed := true, false
	return r.gamma.Markets(ctx, &polytypes.GetMarketsParams{Active: &active, Closed: &closed, Limit: scanFetchLimit(limit)})
}

func wideSpread(markets []polytypes.Market) []Opportunity {
	opps := make([]Opportunity, 0, len(markets))
	for _, market := range markets {
		if !market.Active || market.Closed || market.Spread <= 0 {
			continue
		}
		opps = append(opps, opportunityFromMarket(TypeWideSpread, market, fmt.Sprintf("spread %.4f", market.Spread)))
	}
	sort.SliceStable(opps, func(i, j int) bool { return opps[i].Spread > opps[j].Spread })
	return opps
}

func lowLiquidityHighVolume(markets []polytypes.Market) []Opportunity {
	opps := make([]Opportunity, 0, len(markets))
	for _, market := range markets {
		if !market.Active || market.Closed || market.Volume24hr <= 0 {
			continue
		}
		liquidity := firstPositive(market.LiquidityClob, market.LiquidityNum, market.LiquidityAmm)
		if liquidity <= 0 || market.Volume24hr <= liquidity {
			continue
		}
		opp := opportunityFromMarket(TypeLowLiquidityHighVolume, market, fmt.Sprintf("24h volume %.2f exceeds liquidity %.2f", market.Volume24hr, liquidity))
		opp.Liquidity = liquidity
		opps = append(opps, opp)
	}
	sort.SliceStable(opps, func(i, j int) bool {
		return ratio(opps[i].Volume24hr, firstPositive(opps[i].Liquidity, opps[i].LiquidityClob)) > ratio(opps[j].Volume24hr, firstPositive(opps[j].Liquidity, opps[j].LiquidityClob))
	})
	return opps
}

func newMarkets(markets []polytypes.Market) []Opportunity {
	opps := make([]Opportunity, 0, len(markets))
	for _, market := range markets {
		if !market.Active || market.Closed || !market.New {
			continue
		}
		opps = append(opps, opportunityFromMarket(TypeNewMarkets, market, "market is flagged new by Gamma"))
	}
	sort.SliceStable(opps, func(i, j int) bool {
		return opps[i].MarketID > opps[j].MarketID
	})
	return opps
}

func closingSoon(markets []polytypes.Market, now time.Time, hours int) []Opportunity {
	if hours <= 0 {
		hours = 24
	}
	deadline := now.UTC().Add(time.Duration(hours) * time.Hour)
	opps := make([]Opportunity, 0, len(markets))
	for _, market := range markets {
		if !market.Active || market.Closed {
			continue
		}
		end, ok := marketEndTime(market)
		if !ok || end.Before(now.UTC()) || end.After(deadline) {
			continue
		}
		opps = append(opps, opportunityFromMarket(TypeClosingSoon, market, fmt.Sprintf("ends within %d hours", hours)))
	}
	sort.SliceStable(opps, func(i, j int) bool {
		left, _ := marketEndTime(polytypes.Market{EndDateISO: opps[i].EndDate})
		right, _ := marketEndTime(polytypes.Market{EndDateISO: opps[j].EndDate})
		return left.Before(right)
	})
	return opps
}

func negativeRisk(markets []polytypes.Market) []Opportunity {
	opps := make([]Opportunity, 0, len(markets))
	for _, market := range markets {
		if !market.Active || market.Closed || !market.NegRiskOther {
			continue
		}
		opps = append(opps, opportunityFromMarket(TypeNegativeRisk, market, "market is marked as negative-risk related"))
	}
	return opps
}

func (r *Runner) runCrypto5M(ctx context.Context, req Request) (Response, error) {
	if r.events == nil {
		return Response{}, fmt.Errorf("crypto-5m opportunities require Gamma event lookup")
	}
	assets := []string(nil)
	if strings.TrimSpace(req.Asset) != "" {
		assets = []string{strings.ToUpper(strings.TrimSpace(req.Asset))}
	}
	runner := crypto5m.New(r.events, r.pricer)
	runner.Now = r.now
	result, err := runner.Run(ctx, crypto5m.Request{Assets: assets, Enrich: true})
	if err != nil {
		return Response{}, err
	}
	opps := make([]Opportunity, 0, len(result.Markets))
	for _, market := range result.Markets {
		if market.Status != "active" {
			continue
		}
		opps = append(opps, Opportunity{
			Type:        TypeCrypto5M,
			Asset:       market.Asset,
			MarketID:    market.MarketID,
			Question:    market.Question,
			Slug:        market.EventSlug,
			ConditionID: market.ConditionID,
			TokenIDs:    market.TokenIDs,
			EndDate:     market.WindowEnd,
			Price:       market.Price,
			SpreadText:  market.Spread,
			Reasons:     []string{"active 5-minute crypto market"},
		})
	}
	limit := normalizedLimit(req.Limit)
	if len(opps) > limit {
		opps = opps[:limit]
	}
	return Response{Type: TypeCrypto5M, Count: len(opps), Opportunities: opps}, nil
}

func (r *Runner) now() time.Time {
	if r.Now == nil {
		return time.Now().UTC()
	}
	return r.Now().UTC()
}

func marketEndTime(market polytypes.Market) (time.Time, bool) {
	if strings.TrimSpace(market.EndDateISO) != "" {
		if parsed, err := time.Parse(time.RFC3339, market.EndDateISO); err == nil {
			return parsed.UTC(), true
		}
	}
	if !market.EndDate.IsZero() {
		return market.EndDate.Time().UTC(), true
	}
	return time.Time{}, false
}

func firstPositive(values ...float64) float64 {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func ratio(numerator, denominator float64) float64 {
	if denominator <= 0 {
		return 0
	}
	return numerator / denominator
}

func opportunityFromMarket(scannerType Type, market polytypes.Market, reasons ...string) Opportunity {
	return Opportunity{
		Type:          scannerType,
		MarketID:      market.ID,
		Question:      market.Question,
		Slug:          market.Slug,
		ConditionID:   market.ConditionID,
		TokenIDs:      cryptomarkets.ParseTokenIDs(market.ClobTokenIDs),
		EndDate:       market.EndDateISO,
		Volume24hr:    market.Volume24hr,
		LiquidityClob: market.LiquidityClob,
		Spread:        market.Spread,
		Reasons:       compactReasons(reasons),
	}
}

func compactReasons(reasons []string) []string {
	out := make([]string, 0, len(reasons))
	for _, reason := range reasons {
		if trimmed := strings.TrimSpace(reason); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func normalizedLimit(limit int) int {
	if limit <= 0 {
		return 20
	}
	return limit
}

func scanFetchLimit(limit int) int {
	resultLimit := normalizedLimit(limit)
	fetchLimit := resultLimit * 5
	if fetchLimit < 100 {
		return 100
	}
	if fetchLimit > 500 {
		return 500
	}
	return fetchLimit
}
