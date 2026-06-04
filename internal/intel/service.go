// Package intel composes Polygolem read adapters into wallet-intelligence
// dossiers without adding network, auth, or CLI coupling to pkg/intel.
package intel

import (
	"context"
	"fmt"
	"strings"
	"time"

	sdkintel "github.com/TrebuchetDynamics/polygolem/pkg/intel"
	"github.com/TrebuchetDynamics/polygolem/pkg/types"
)

const defaultLimit = 100

// DataReader is the minimal Polygolem Data API contract needed for wallet dossiers.
type DataReader interface {
	CurrentPositionsWithLimit(context.Context, string, int) ([]types.Position, error)
	ClosedPositionsWithLimit(context.Context, string, int) ([]types.ClosedPosition, error)
	Trades(context.Context, string, int) ([]types.Trade, error)
	MarketTrades(context.Context, string, int) ([]types.Trade, error)
	TopHolders(context.Context, string, int) ([]types.Holder, error)
	OpenInterest(context.Context, string) (*types.OpenInterest, error)
	TraderLeaderboard(context.Context, int) ([]types.LeaderboardRow, error)
}

// Service builds read-only wallet-intelligence dossiers from Polygolem source adapters.
type Service struct {
	data DataReader
	now  func() time.Time
}

// NewService creates a wallet-intelligence service.
func NewService(data DataReader) *Service {
	return &Service{data: data, now: time.Now}
}

// DossierOptions controls source row limits and scoring priors.
type DossierOptions struct {
	Limit     int
	PriorWins float64
	PriorBets float64
	AsOf      time.Time
}

// LeaderboardOptions controls wallet-intelligence leaderboard reads.
type LeaderboardOptions struct {
	Limit int
	AsOf  time.Time
}

// AlertOptions controls batch wallet-intelligence alert reads.
type AlertOptions struct {
	User     string
	Limit    int
	MinScore int
	AsOf     time.Time
}

// MarketFlowOptions controls market-flow source reads.
type MarketFlowOptions struct {
	Limit int
	AsOf  time.Time
}

// WalletDossier builds a source-backed wallet dossier. Closed-position rows are
// authoritative for realized PnL and win/loss counts in V1. Trade/current-position
// failures produce partial dossiers instead of synthetic zeroes.
func (s *Service) WalletDossier(ctx context.Context, wallet string, opts DossierOptions) (*sdkintel.WalletDossier, error) {
	wallet = strings.TrimSpace(wallet)
	if wallet == "" {
		return nil, fmt.Errorf("intel: wallet is required")
	}
	if s == nil || s.data == nil {
		return nil, fmt.Errorf("intel: data reader is required")
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	asOf := opts.AsOf
	if asOf.IsZero() {
		asOf = s.currentTime()
	}

	var warnings []string
	var conflicts []sdkintel.SourceConflict
	var sources []sdkintel.SourceRef

	closed, closedErr := s.data.ClosedPositionsWithLimit(ctx, wallet, limit)
	if closedErr != nil {
		warnings = append(warnings, "closed positions unavailable: "+closedErr.Error())
	} else {
		sources = append(sources, sdkintel.SourceRef{Kind: "data_api.closed_positions", Rows: len(closed), AsOf: asOf.Format(time.RFC3339)})
	}

	positions, positionsErr := s.data.CurrentPositionsWithLimit(ctx, wallet, limit)
	if positionsErr != nil {
		warnings = append(warnings, "current positions unavailable: "+positionsErr.Error())
	} else {
		sources = append(sources, sdkintel.SourceRef{Kind: "data_api.current_positions", Rows: len(positions), AsOf: asOf.Format(time.RFC3339)})
		conflicts = append(conflicts, walletConflicts(wallet, positions)...)
	}

	trades, tradesErr := s.data.Trades(ctx, wallet, limit)
	if tradesErr != nil {
		warnings = append(warnings, "trades unavailable: "+tradesErr.Error())
	} else {
		sources = append(sources, sdkintel.SourceRef{Kind: "data_api.trades", Rows: len(trades), AsOf: asOf.Format(time.RFC3339)})
		conflicts = append(conflicts, tradeWalletConflicts(wallet, trades)...)
	}

	if closedErr != nil {
		// Without closed positions, V1 has no authoritative realized PnL or win/loss source.
		return &sdkintel.WalletDossier{
			Wallet:    wallet,
			AsOf:      asOf,
			Status:    statusFor(warnings, conflicts),
			Sources:   sources,
			Conflicts: conflicts,
			Warnings:  warnings,
			Score: sdkintel.WalletScore{
				Wallet:         wallet,
				FormulaVersion: sdkintel.FormulaWalletScoreV1,
				AsOf:           asOf,
				Confidence:     sdkintel.ConfidenceNone,
				Language:       "statistical candidate signal; not a finding of misconduct",
			},
		}, nil
	}

	summary := summarizeClosed(wallet, closed, asOf)
	summary.SourceRows = len(closed)
	summary.SourceDescription = "data_api.closed_positions"
	if len(trades) > 0 && len(closed) == 0 {
		warnings = append(warnings, "trades exist but closed-position authority has no rows")
	}
	if len(positions) > 0 {
		warnings = append(warnings, "current positions are present but not included in realized PnL")
	}

	score := sdkintel.ScoreWallet(sdkintel.ScoreInput{
		Wallet:      wallet,
		Wins:        summary.Wins,
		Bets:        summary.Bets,
		Volume:      summary.Volume,
		RealizedPnL: summary.RealizedPnL,
		PriorWins:   opts.PriorWins,
		PriorBets:   opts.PriorBets,
		AsOf:        asOf,
		SourceRows:  summary.SourceRows,
	})

	return &sdkintel.WalletDossier{
		Wallet:    wallet,
		AsOf:      asOf,
		Status:    statusFor(warnings, conflicts),
		Summary:   summary,
		Score:     score,
		Sources:   sources,
		Conflicts: conflicts,
		Warnings:  warnings,
	}, nil
}

// Leaderboard returns Data-API-ranked wallet intelligence rows. V1 does not
// invent shrinkage win rates because the Data API leaderboard row does not
// expose wins/bets.
func (s *Service) Leaderboard(ctx context.Context, opts LeaderboardOptions) ([]sdkintel.LeaderboardRow, error) {
	if s == nil || s.data == nil {
		return nil, fmt.Errorf("intel: data reader is required")
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = 20
	}
	asOf := opts.AsOf
	if asOf.IsZero() {
		asOf = s.currentTime()
	}
	rows, err := s.data.TraderLeaderboard(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("intel: leaderboard: %w", err)
	}
	out := make([]sdkintel.LeaderboardRow, 0, len(rows))
	for i, row := range rows {
		rank := row.Rank
		if rank == 0 {
			rank = i + 1
		}
		summary := sdkintel.WalletSummary{
			Wallet:            row.User,
			Volume:            row.Volume,
			RealizedPnL:       row.Pnl,
			ROI:               row.ROI,
			AsOf:              asOf,
			FormulaVersion:    sdkintel.FormulaWalletScoreV1,
			SourceRows:        1,
			SourceDescription: "data_api.trader_leaderboard",
		}
		score := sdkintel.ScoreWallet(sdkintel.ScoreInput{
			Wallet:      row.User,
			Volume:      row.Volume,
			RealizedPnL: row.Pnl,
			AsOf:        asOf,
			SourceRows:  1,
		})
		out = append(out, sdkintel.LeaderboardRow{Rank: rank, Wallet: row.User, Summary: summary, Score: score})
	}
	return out, nil
}

// Alerts builds batch intelligence alerts from a wallet dossier. V1 requires a
// wallet/user because Polygolem's current Data API adapters do not expose a
// global recent-trades feed.
func (s *Service) Alerts(ctx context.Context, opts AlertOptions) ([]sdkintel.Signal, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	asOf := opts.AsOf
	if asOf.IsZero() {
		asOf = s.currentTime()
	}
	dossier, err := s.WalletDossier(ctx, opts.User, DossierOptions{Limit: limit, AsOf: asOf})
	if err != nil {
		return nil, err
	}
	minScore := opts.MinScore
	if minScore <= 0 {
		minScore = 70
	}
	if dossier.Score.Value < minScore {
		return nil, nil
	}
	return []sdkintel.Signal{{
		Score:          dossier.Score.Value,
		Wallet:         dossier.Wallet,
		Confidence:     dossier.Score.Confidence,
		FormulaVersion: sdkintel.FormulaWalletScoreV1,
		AsOf:           asOf,
		Language:       dossier.Score.Language,
		Reasons:        append([]string(nil), dossier.Score.Reasons...),
		Sources:        append([]sdkintel.SourceRef(nil), dossier.Sources...),
	}}, nil
}

// MarketFlow summarizes bounded Data API holder, trade, and open-interest reads
// for one market/token. Missing optional sources produce warnings, not synthetic
// zeros hidden as complete data.
func (s *Service) MarketFlow(ctx context.Context, market string, opts MarketFlowOptions) (*sdkintel.MarketFlow, error) {
	market = strings.TrimSpace(market)
	if market == "" {
		return nil, fmt.Errorf("intel: market is required")
	}
	if s == nil || s.data == nil {
		return nil, fmt.Errorf("intel: data reader is required")
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	asOf := opts.AsOf
	if asOf.IsZero() {
		asOf = s.currentTime()
	}
	flow := &sdkintel.MarketFlow{Market: market, AsOf: asOf, FormulaVersion: "market_flow_v1"}

	holders, err := s.data.TopHolders(ctx, market, limit)
	if err != nil {
		flow.Warnings = append(flow.Warnings, "holders unavailable: "+err.Error())
	} else {
		flow.Sources = append(flow.Sources, sdkintel.SourceRef{Kind: "data_api.holders", Rows: len(holders), AsOf: asOf.Format(time.RFC3339)})
		flow.HolderCount = len(holders)
		for _, holder := range holders {
			flow.HolderShares += holder.Shares
			flow.HolderVolume += holder.Volume
		}
	}

	trades, err := s.data.MarketTrades(ctx, market, limit)
	if err != nil {
		flow.Warnings = append(flow.Warnings, "market trades unavailable: "+err.Error())
	} else {
		flow.Sources = append(flow.Sources, sdkintel.SourceRef{Kind: "data_api.market_trades", Rows: len(trades), AsOf: asOf.Format(time.RFC3339)})
		flow.TradeCount = len(trades)
		for _, trade := range trades {
			flow.TradeNotional += trade.Price * trade.Size
		}
	}

	openInterest, err := s.data.OpenInterest(ctx, market)
	if err != nil {
		flow.Warnings = append(flow.Warnings, "open interest unavailable: "+err.Error())
	} else if openInterest != nil {
		flow.Sources = append(flow.Sources, sdkintel.SourceRef{Kind: "data_api.open_interest", Rows: 1, AsOf: asOf.Format(time.RFC3339)})
		flow.OpenInterest = openInterest.OpenValue
	}
	flow.CandidateSignal = flow.TradeNotional > 0 || flow.HolderVolume > 0 || flow.OpenInterest > 0
	return flow, nil
}

func (s *Service) currentTime() time.Time {
	if s != nil && s.now != nil {
		return s.now().UTC()
	}
	return time.Now().UTC()
}

func summarizeClosed(wallet string, rows []types.ClosedPosition, asOf time.Time) sdkintel.WalletSummary {
	summary := sdkintel.WalletSummary{
		Wallet:         wallet,
		AsOf:           asOf,
		FormulaVersion: sdkintel.FormulaShrinkageWinRateV1,
	}
	for _, row := range rows {
		if emptyClosed(row) {
			continue
		}
		summary.Bets++
		if row.RealizedPnl > 0 {
			summary.Wins++
		}
		summary.RealizedPnL += row.RealizedPnl
		volume := row.TotalBought
		if volume == 0 {
			volume = row.Size * row.AvgPrice
		}
		summary.Volume += volume
		if ts, ok := parseRowTime(row.Timestamp); ok && ts.After(summary.LastActive) {
			summary.LastActive = ts
		}
	}
	if summary.Bets > 0 {
		summary.RawWinRate = float64(summary.Wins) / float64(summary.Bets)
	}
	summary.ROI = sdkintel.ROI(summary.RealizedPnL, summary.Volume)
	summary.ShrinkageWinRate = sdkintel.ShrinkageWinRate(summary.Wins, summary.Bets, 0, 0)
	return summary
}

func statusFor(warnings []string, conflicts []sdkintel.SourceConflict) string {
	if len(conflicts) > 0 {
		return sdkintel.DossierStatusConflicted
	}
	if len(warnings) > 0 {
		return sdkintel.DossierStatusPartial
	}
	return sdkintel.DossierStatusComplete
}

func walletConflicts(wallet string, rows []types.Position) []sdkintel.SourceConflict {
	var out []sdkintel.SourceConflict
	for _, row := range rows {
		if row.ProxyWallet == "" || strings.EqualFold(row.ProxyWallet, wallet) {
			continue
		}
		out = append(out, sdkintel.SourceConflict{Field: "proxyWallet", Primary: wallet, Other: row.ProxyWallet, Reason: "current position belongs to a different proxy wallet"})
	}
	return out
}

func tradeWalletConflicts(wallet string, rows []types.Trade) []sdkintel.SourceConflict {
	var out []sdkintel.SourceConflict
	for _, row := range rows {
		if row.ProxyWallet == "" || strings.EqualFold(row.ProxyWallet, wallet) {
			continue
		}
		out = append(out, sdkintel.SourceConflict{Field: "proxyWallet", Primary: wallet, Other: row.ProxyWallet, Reason: "trade belongs to a different proxy wallet"})
	}
	return out
}

func emptyClosed(row types.ClosedPosition) bool {
	return row.TokenID == "" && row.ConditionID == "" && row.MarketID == "" && row.Title == "" && row.RealizedPnl == 0 && row.Size == 0
}

func parseRowTime(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}
	if ts, err := time.Parse(time.RFC3339, raw); err == nil {
		return ts, true
	}
	return time.Time{}, false
}
