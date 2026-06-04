package intel

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	sdkintel "github.com/TrebuchetDynamics/polygolem/pkg/intel"
	"github.com/TrebuchetDynamics/polygolem/pkg/types"
)

func TestWalletDossierUsesClosedPositionsAsPnLAuthority(t *testing.T) {
	asOf := time.Date(2026, 6, 3, 0, 0, 0, 0, time.UTC)
	reader := &fakeDataReader{
		closed: []types.ClosedPosition{
			{TokenID: "won", Size: 10, AvgPrice: 0.4, TotalBought: 4, RealizedPnl: 6, Timestamp: "2026-06-02T10:00:00Z"},
			{TokenID: "lost", Size: 5, AvgPrice: 0.5, TotalBought: 2.5, RealizedPnl: -2.5, Timestamp: "2026-06-02T11:00:00Z"},
		},
		positions: []types.Position{{TokenID: "open", ProxyWallet: "0xwallet", CurrentValue: 12}},
		trades:    []types.Trade{{ID: "trade", ProxyWallet: "0xwallet", Size: 100, Price: 0.9}},
	}
	service := NewService(reader)

	dossier, err := service.WalletDossier(context.Background(), "0xwallet", DossierOptions{AsOf: asOf, Limit: 50})
	if err != nil {
		t.Fatal(err)
	}

	if reader.closedLimit != 50 || reader.positionsLimit != 50 || reader.tradesLimit != 50 {
		t.Fatalf("limits closed=%d positions=%d trades=%d", reader.closedLimit, reader.positionsLimit, reader.tradesLimit)
	}
	if dossier.Status != sdkintel.DossierStatusPartial {
		t.Fatalf("status=%q warnings=%v", dossier.Status, dossier.Warnings)
	}
	if dossier.Summary.Bets != 2 || dossier.Summary.Wins != 1 {
		t.Fatalf("summary bets/wins=%+v", dossier.Summary)
	}
	if dossier.Summary.RealizedPnL != 3.5 || dossier.Summary.Volume != 6.5 {
		t.Fatalf("summary pnl/volume=%+v", dossier.Summary)
	}
	if dossier.Summary.SourceDescription != "data_api.closed_positions" || dossier.Summary.SourceRows != 2 {
		t.Fatalf("source summary=%+v", dossier.Summary)
	}
	if !dossier.Summary.LastActive.Equal(time.Date(2026, 6, 2, 11, 0, 0, 0, time.UTC)) {
		t.Fatalf("lastActive=%s", dossier.Summary.LastActive)
	}
	if dossier.Score.RawMetrics.Bets != 2 || dossier.Score.RawMetrics.Wins != 1 {
		t.Fatalf("score metrics=%+v", dossier.Score.RawMetrics)
	}
	if len(dossier.Sources) != 3 {
		t.Fatalf("sources=%+v", dossier.Sources)
	}
	if !containsWarning(dossier.Warnings, "current positions are present") {
		t.Fatalf("expected current-position warning, got %v", dossier.Warnings)
	}
}

func TestWalletDossierMarksPartialWhenClosedPositionsUnavailable(t *testing.T) {
	reader := &fakeDataReader{
		closedErr: errors.New("upstream closed down"),
		trades:    []types.Trade{{ID: "trade", ProxyWallet: "0xwallet"}},
	}
	service := NewService(reader)

	dossier, err := service.WalletDossier(context.Background(), "0xwallet", DossierOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if dossier.Status != sdkintel.DossierStatusPartial {
		t.Fatalf("status=%q", dossier.Status)
	}
	if dossier.Score.Confidence != sdkintel.ConfidenceNone || dossier.Score.Value != 0 {
		t.Fatalf("score=%+v", dossier.Score)
	}
	if !containsWarning(dossier.Warnings, "closed positions unavailable") {
		t.Fatalf("warnings=%v", dossier.Warnings)
	}
}

func TestWalletDossierMarksConflictedSourceRows(t *testing.T) {
	reader := &fakeDataReader{
		closed:    []types.ClosedPosition{{TokenID: "won", Size: 1, AvgPrice: 0.4, RealizedPnl: 0.6}},
		positions: []types.Position{{TokenID: "open", ProxyWallet: "0xother"}},
		trades:    []types.Trade{{ID: "trade", ProxyWallet: "0xwallet"}},
	}
	service := NewService(reader)

	dossier, err := service.WalletDossier(context.Background(), "0xwallet", DossierOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if dossier.Status != sdkintel.DossierStatusConflicted {
		t.Fatalf("status=%q conflicts=%+v", dossier.Status, dossier.Conflicts)
	}
	if len(dossier.Conflicts) != 1 || dossier.Conflicts[0].Other != "0xother" {
		t.Fatalf("conflicts=%+v", dossier.Conflicts)
	}
}

func TestLeaderboardUsesDataAPILeaderboardWithoutInventingWins(t *testing.T) {
	asOf := time.Date(2026, 6, 3, 0, 0, 0, 0, time.UTC)
	reader := &fakeDataReader{leaderboard: []types.LeaderboardRow{{Rank: 3, User: "0xwallet", Volume: 1000, Pnl: 125, ROI: 0.125}}}
	service := NewService(reader)

	rows, err := service.Leaderboard(context.Background(), LeaderboardOptions{Limit: 7, AsOf: asOf})
	if err != nil {
		t.Fatal(err)
	}

	if reader.leaderboardLimit != 7 {
		t.Fatalf("limit=%d", reader.leaderboardLimit)
	}
	if len(rows) != 1 {
		t.Fatalf("rows=%+v", rows)
	}
	row := rows[0]
	if row.Rank != 3 || row.Wallet != "0xwallet" {
		t.Fatalf("row identity=%+v", row)
	}
	if row.Summary.Volume != 1000 || row.Summary.RealizedPnL != 125 || row.Summary.ROI != 0.125 {
		t.Fatalf("summary=%+v", row.Summary)
	}
	if row.Score.RawMetrics.Bets != 0 || row.Score.RawMetrics.Wins != 0 {
		t.Fatalf("leaderboard must not invent wins/bets: %+v", row.Score.RawMetrics)
	}
}

func TestAlertsReturnsBatchSignalWhenDossierScorePassesThreshold(t *testing.T) {
	reader := &fakeDataReader{closed: []types.ClosedPosition{
		{TokenID: "a", Size: 10, AvgPrice: 0.5, TotalBought: 5, RealizedPnl: 10},
		{TokenID: "b", Size: 10, AvgPrice: 0.5, TotalBought: 5, RealizedPnl: 10},
	}}
	service := NewService(reader)

	signals, err := service.Alerts(context.Background(), AlertOptions{User: "0xwallet", MinScore: 40})
	if err != nil {
		t.Fatal(err)
	}
	if len(signals) != 1 {
		t.Fatalf("signals=%+v", signals)
	}
	if signals[0].Wallet != "0xwallet" || signals[0].Score < 40 {
		t.Fatalf("signal=%+v", signals[0])
	}
	if !strings.Contains(signals[0].Language, "not a finding") {
		t.Fatalf("unsafe language=%q", signals[0].Language)
	}
}

func TestAlertsFiltersBelowThreshold(t *testing.T) {
	reader := &fakeDataReader{closed: []types.ClosedPosition{{TokenID: "a", Size: 10, AvgPrice: 0.5, TotalBought: 5, RealizedPnl: -2}}}
	service := NewService(reader)

	signals, err := service.Alerts(context.Background(), AlertOptions{User: "0xwallet", MinScore: 70})
	if err != nil {
		t.Fatal(err)
	}
	if len(signals) != 0 {
		t.Fatalf("signals=%+v", signals)
	}
}

func TestMarketFlowSummarizesBoundedDataAPIReads(t *testing.T) {
	asOf := time.Date(2026, 6, 3, 0, 0, 0, 0, time.UTC)
	reader := &fakeDataReader{
		holders:      []types.Holder{{Shares: 10, Volume: 100}, {Shares: 3, Volume: 25}},
		marketTrades: []types.Trade{{Price: 0.5, Size: 10}, {Price: 0.25, Size: 4}},
		openInterest: &types.OpenInterest{Market: "0xmarket", OpenValue: 200},
	}
	service := NewService(reader)

	flow, err := service.MarketFlow(context.Background(), "0xmarket", MarketFlowOptions{Limit: 12, AsOf: asOf})
	if err != nil {
		t.Fatal(err)
	}
	if reader.holdersLimit != 12 || reader.marketTradesLimit != 12 {
		t.Fatalf("limits holders=%d trades=%d", reader.holdersLimit, reader.marketTradesLimit)
	}
	if flow.HolderCount != 2 || flow.HolderShares != 13 || flow.HolderVolume != 125 {
		t.Fatalf("holder flow=%+v", flow)
	}
	if flow.TradeCount != 2 || flow.TradeNotional != 6 {
		t.Fatalf("trade flow=%+v", flow)
	}
	if flow.OpenInterest != 200 || !flow.CandidateSignal {
		t.Fatalf("open interest flow=%+v", flow)
	}
	if len(flow.Sources) != 3 {
		t.Fatalf("sources=%+v", flow.Sources)
	}
}

func TestWalletDossierRequiresWalletAndReader(t *testing.T) {
	if _, err := NewService(&fakeDataReader{}).WalletDossier(context.Background(), " ", DossierOptions{}); err == nil {
		t.Fatal("expected wallet error")
	}
	if _, err := NewService(nil).WalletDossier(context.Background(), "0xwallet", DossierOptions{}); err == nil {
		t.Fatal("expected reader error")
	}
}

func containsWarning(warnings []string, needle string) bool {
	for _, warning := range warnings {
		if strings.Contains(warning, needle) {
			return true
		}
	}
	return false
}

type fakeDataReader struct {
	positions    []types.Position
	closed       []types.ClosedPosition
	trades       []types.Trade
	marketTrades []types.Trade
	holders      []types.Holder
	openInterest *types.OpenInterest
	leaderboard  []types.LeaderboardRow

	positionsErr error
	closedErr    error
	tradesErr    error

	positionsLimit    int
	closedLimit       int
	tradesLimit       int
	marketTradesLimit int
	holdersLimit      int
	leaderboardLimit  int
}

func (f *fakeDataReader) CurrentPositionsWithLimit(_ context.Context, _ string, limit int) ([]types.Position, error) {
	f.positionsLimit = limit
	if f.positionsErr != nil {
		return nil, f.positionsErr
	}
	return f.positions, nil
}

func (f *fakeDataReader) ClosedPositionsWithLimit(_ context.Context, _ string, limit int) ([]types.ClosedPosition, error) {
	f.closedLimit = limit
	if f.closedErr != nil {
		return nil, f.closedErr
	}
	return f.closed, nil
}

func (f *fakeDataReader) Trades(_ context.Context, _ string, limit int) ([]types.Trade, error) {
	f.tradesLimit = limit
	if f.tradesErr != nil {
		return nil, f.tradesErr
	}
	return f.trades, nil
}

func (f *fakeDataReader) MarketTrades(_ context.Context, _ string, limit int) ([]types.Trade, error) {
	f.marketTradesLimit = limit
	return f.marketTrades, nil
}

func (f *fakeDataReader) TopHolders(_ context.Context, _ string, limit int) ([]types.Holder, error) {
	f.holdersLimit = limit
	return f.holders, nil
}

func (f *fakeDataReader) OpenInterest(_ context.Context, _ string) (*types.OpenInterest, error) {
	return f.openInterest, nil
}

func (f *fakeDataReader) TraderLeaderboard(_ context.Context, limit int) ([]types.LeaderboardRow, error) {
	f.leaderboardLimit = limit
	return f.leaderboard, nil
}
