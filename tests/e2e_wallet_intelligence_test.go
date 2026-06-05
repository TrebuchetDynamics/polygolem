package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	internalintel "github.com/TrebuchetDynamics/polygolem/internal/intel"
	"github.com/TrebuchetDynamics/polygolem/pkg/data"
)

func TestWalletIntelligenceE2EAgainstLocalDataAPI(t *testing.T) {
	hits := map[string]int{}
	dataServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits[r.URL.Path]++
		routeWalletIntelDataAPI(t, w, r)
	}))
	defer dataServer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dataClient := data.NewClient(data.Config{BaseURL: dataServer.URL})
	service := internalintel.NewService(dataClient)
	asOf := time.Date(2026, 6, 3, 0, 0, 0, 0, time.UTC)

	dossier, err := service.WalletDossier(ctx, "0xwallet", internalintel.DossierOptions{Limit: 50, AsOf: asOf})
	if err != nil {
		t.Fatalf("WalletDossier: %v", err)
	}
	if dossier.Wallet != "0xwallet" || dossier.Status != "partial" {
		t.Fatalf("dossier identity/status = %+v", dossier)
	}
	if dossier.Summary.Bets != 2 || dossier.Summary.Wins != 2 || dossier.Summary.Volume != 10 || dossier.Summary.RealizedPnL != 20 {
		t.Fatalf("summary = %+v", dossier.Summary)
	}
	if dossier.Score.Value != 45 || dossier.Score.Language == "" {
		t.Fatalf("score = %+v", dossier.Score)
	}
	if !containsString(dossier.Warnings, "current positions are present but not included in realized PnL") {
		t.Fatalf("warnings = %+v", dossier.Warnings)
	}

	alerts, err := service.Alerts(ctx, internalintel.AlertOptions{User: "0xwallet", Limit: 50, MinScore: 40, AsOf: asOf})
	if err != nil {
		t.Fatalf("Alerts: %v", err)
	}
	if len(alerts) != 1 || alerts[0].Wallet != "0xwallet" || alerts[0].Score != 45 {
		t.Fatalf("alerts = %+v", alerts)
	}

	leaderboard, err := service.Leaderboard(ctx, internalintel.LeaderboardOptions{Limit: 5, AsOf: asOf})
	if err != nil {
		t.Fatalf("Leaderboard: %v", err)
	}
	if len(leaderboard) != 1 || leaderboard[0].Rank != 7 || leaderboard[0].Wallet != "0xwallet" {
		t.Fatalf("leaderboard = %+v", leaderboard)
	}
	if leaderboard[0].Score.RawMetrics.Bets != 0 || leaderboard[0].Score.RawMetrics.Wins != 0 {
		t.Fatalf("leaderboard invented wins/bets: %+v", leaderboard[0].Score.RawMetrics)
	}

	flow, err := service.MarketFlow(ctx, "0xmarket", internalintel.MarketFlowOptions{Limit: 3, AsOf: asOf})
	if err != nil {
		t.Fatalf("MarketFlow: %v", err)
	}
	if flow.HolderCount != 2 || flow.HolderShares != 13 || flow.HolderVolume != 125 {
		t.Fatalf("holder flow = %+v", flow)
	}
	if flow.TradeCount != 2 || flow.TradeNotional != 6 || flow.OpenInterest != 200 || !flow.CandidateSignal {
		t.Fatalf("market flow = %+v", flow)
	}

	for _, path := range []string{"/closed-positions", "/positions", "/trades", "/holders", "/oi", "/v1/leaderboard"} {
		if hits[path] == 0 {
			t.Fatalf("expected e2e route %s to be exercised; hits=%+v", path, hits)
		}
	}
}

func routeWalletIntelDataAPI(t *testing.T, w http.ResponseWriter, r *http.Request) {
	t.Helper()
	switch r.URL.Path {
	case "/closed-positions":
		if r.URL.Query().Get("user") != "0xwallet" || r.URL.Query().Get("limit") != "50" {
			failRequest(t, w, "closed positions query = %s", r.URL.RawQuery)
			return
		}
		respondJSON(t, w, []map[string]interface{}{
			{"asset": "won-1", "conditionId": "condition-1", "proxyWallet": "0xwallet", "avgPrice": "0.50", "size": "10", "totalBought": "5", "realizedPnl": "10", "timestamp": "2026-06-02T10:00:00Z"},
			{"asset": "won-2", "conditionId": "condition-2", "proxyWallet": "0xwallet", "avgPrice": "0.50", "size": "10", "totalBought": "5", "realizedPnl": "10", "timestamp": "2026-06-02T11:00:00Z"},
		})
	case "/positions":
		if r.URL.Query().Get("user") != "0xwallet" || r.URL.Query().Get("limit") != "50" {
			failRequest(t, w, "positions query = %s", r.URL.RawQuery)
			return
		}
		respondJSON(t, w, []map[string]interface{}{
			{"asset": "open-1", "conditionId": "condition-open", "proxyWallet": "0xwallet", "size": "4", "avgPrice": "0.25", "curPrice": "0.40", "currentValue": "1.60"},
		})
	case "/trades":
		if user := r.URL.Query().Get("user"); user != "" {
			if user != "0xwallet" || r.URL.Query().Get("limit") != "50" {
				failRequest(t, w, "user trades query = %s", r.URL.RawQuery)
				return
			}
			respondJSON(t, w, []map[string]interface{}{
				{"id": "user-trade-1", "conditionId": "condition-1", "asset": "won-1", "proxyWallet": "0xwallet", "side": "BUY", "price": "0.50", "size": "10"},
			})
			return
		}
		if r.URL.Query().Get("market") != "0xmarket" || r.URL.Query().Get("limit") != "3" {
			failRequest(t, w, "market trades query = %s", r.URL.RawQuery)
			return
		}
		respondJSON(t, w, []map[string]interface{}{
			{"id": "market-trade-1", "conditionId": "0xmarket", "asset": "yes-token", "price": "0.50", "size": "10"},
			{"id": "market-trade-2", "conditionId": "0xmarket", "asset": "no-token", "price": "0.25", "size": "4"},
		})
	case "/v1/leaderboard":
		if r.URL.Query().Get("limit") != "5" {
			failRequest(t, w, "leaderboard query = %s", r.URL.RawQuery)
			return
		}
		respondJSON(t, w, []map[string]interface{}{
			{"rank": 7, "user": "0xwallet", "volume": "1000", "pnl": "125", "roi": "0.125"},
		})
	case "/holders":
		if r.URL.Query().Get("market") != "0xmarket" || r.URL.Query().Get("limit") != "3" {
			failRequest(t, w, "holders query = %s", r.URL.RawQuery)
			return
		}
		respondJSON(t, w, []map[string]interface{}{
			{
				"token": "yes-token",
				"holders": []map[string]interface{}{
					{"proxyWallet": "0xa", "shares": "10", "volume": "100"},
					{"proxyWallet": "0xb", "shares": "3", "volume": "25"},
				},
			},
		})
	case "/oi":
		if r.URL.Query().Get("market") != "0xmarket" {
			failRequest(t, w, "open interest query = %s", r.URL.RawQuery)
			return
		}
		respondJSON(t, w, []map[string]interface{}{
			{"market": "0xmarket", "open_value": "200"},
		})
	default:
		failRequest(t, w, "unexpected wallet-intel e2e path %s", r.URL.Path)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
