package data

import (
	"context"
	"encoding/json"
	goerrors "errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TrebuchetDynamics/polygolem/pkg/polyerrors"
	"github.com/TrebuchetDynamics/polygolem/pkg/types"
)

func TestClientReturnsNormalizedDataAPIErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`geoblock: restricted jurisdiction`))
	}))
	defer server.Close()

	_, err := NewClient(Config{BaseURL: server.URL}).CurrentPositions(context.Background(), "0xuser")
	var normalized polyerrors.Error
	if !goerrors.As(err, &normalized) {
		t.Fatalf("expected polyerrors.Error, got %T %v", err, err)
	}
	if normalized.Kind != polyerrors.Geoblocked {
		t.Fatalf("normalized=%#v", normalized)
	}
}

func TestClientCurrentPositionsReturnsPublicTypes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/positions" {
			t.Fatalf("path=%s want /positions", r.URL.Path)
		}
		if r.URL.Query().Get("user") != "0xuser" || r.URL.Query().Get("limit") != "2" {
			t.Fatalf("query=%s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"asset":         "token-1",
			"conditionId":   "condition-1",
			"market_id":     "market-1",
			"side":          "YES",
			"size":          7.5,
			"cashPnl":       1.25,
			"unrealizedPnl": 2.5,
			"redeemable":    true,
		}})
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL})
	rows, err := client.CurrentPositionsWithLimit(context.Background(), "0xuser", 2)
	if err != nil {
		t.Fatal(err)
	}
	var publicRows []types.Position = rows
	if len(publicRows) != 1 || publicRows[0].TokenID != "token-1" || publicRows[0].CashPnl != 1.25 {
		t.Fatalf("rows=%+v", publicRows)
	}
	if publicRows[0].MarketID != "market-1" || publicRows[0].Side != "YES" {
		t.Fatalf("market/side not threaded: %+v", publicRows[0])
	}
	if publicRows[0].UnrealizedPnl != 2.5 {
		t.Fatalf("unrealized pnl not threaded: %+v", publicRows[0])
	}
	if !publicRows[0].Redeemable {
		t.Fatalf("redeemable not threaded: %+v", publicRows[0])
	}
}

func TestClientCurrentPositionsDecodesStringNumericFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"asset":              "token-1",
			"conditionId":        "condition-1",
			"size":               "7.5",
			"avgPrice":           "0.42",
			"curPrice":           "0.55",
			"unrealizedPnl":      "1.25",
			"outcomeIndex":       "1",
			"redeemable":         "true",
			"negativeRisk":       "true",
			"percentRealizedPnl": "3.25",
		}})
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL})
	rows, err := client.CurrentPositionsWithLimit(context.Background(), "0xuser", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Size != 7.5 || rows[0].AvgPrice != 0.42 || rows[0].OutcomeIndex != 1 {
		t.Fatalf("rows=%+v", rows)
	}
	if !rows[0].Redeemable || !rows[0].NegativeRisk || rows[0].PercentRealized != 3.25 {
		t.Fatalf("bool/pnl fields not threaded: %+v", rows[0])
	}
}

func TestClientCurrentPositionsDecodesSnakeCaseAliases(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/positions" {
			t.Fatalf("path=%s want /positions", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"token_id":       "token-1",
			"condition_id":   "condition-1",
			"market_id":      "market-1",
			"avg_price":      0.42,
			"current_price":  0.55,
			"unrealized_pnl": 3.25,
		}})
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL})
	rows, err := client.CurrentPositionsWithLimit(context.Background(), "0xuser", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows=%+v", rows)
	}
	if rows[0].TokenID != "token-1" || rows[0].ConditionID != "condition-1" || rows[0].MarketID != "market-1" {
		t.Fatalf("id fields not decoded: %+v", rows[0])
	}
	if rows[0].AvgPrice != 0.42 || rows[0].CurrentPrice != 0.55 || rows[0].UnrealizedPnl != 3.25 {
		t.Fatalf("price/pnl fields not decoded: %+v", rows[0])
	}
}

func TestClientClosedPositionsDecodesStringOutcomeIndex(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"asset":        "token-closed",
			"conditionId":  "condition-closed",
			"size":         "2.5",
			"avgPrice":     "0.45",
			"realizedPnl":  "1.25",
			"curPrice":     "0.62",
			"outcomeIndex": "1",
		}})
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL})
	rows, err := client.ClosedPositionsWithLimit(context.Background(), "0xuser", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].OutcomeIndex != 1 || rows[0].Size != 2.5 || rows[0].RealizedPnl != 1.25 {
		t.Fatalf("rows=%+v", rows)
	}
}

func TestClientTradesDecodeStringOutcomeIndex(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"id":              "trade-1",
			"conditionId":     "condition-1",
			"asset":           "token-1",
			"price":           "0.55",
			"size":            "12.5",
			"feeRateBps":      "25",
			"outcome":         "Yes",
			"outcomeIndex":    "1",
			"transactionHash": "0xtx",
			"timestamp":       "1714001234",
		}})
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL})
	rows, err := client.Trades(context.Background(), "0xuser", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].OutcomeIndex != 1 || rows[0].FeeRateBps != 25 || rows[0].CreatedAt != "1714001234" {
		t.Fatalf("rows=%+v", rows)
	}
}

func TestClientTopHoldersReturnsCurrentHolderFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/holders" {
			t.Fatalf("path=%s want /holders", r.URL.Path)
		}
		if r.URL.Query().Get("market") != "condition-1" || r.URL.Query().Get("limit") != "2" {
			t.Fatalf("query=%s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"token": "token-1",
			"holders": []map[string]any{{
				"proxyWallet": "0xholder",
				"amount":      7.5,
				"pnl":         1.25,
				"volume":      100.0,
			}},
		}})
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL})
	rows, err := client.TopHolders(context.Background(), "condition-1", 2)
	if err != nil {
		t.Fatal(err)
	}
	var publicRows []types.Holder = rows
	if len(publicRows) != 1 {
		t.Fatalf("rows=%+v", publicRows)
	}
	if publicRows[0].Address != "0xholder" || publicRows[0].ProxyWallet != "0xholder" {
		t.Fatalf("holder address fields not threaded: %+v", publicRows[0])
	}
	if publicRows[0].Shares != 7.5 || publicRows[0].Amount != 7.5 {
		t.Fatalf("holder amount fields not threaded: %+v", publicRows[0])
	}
}

func TestClientTopHoldersDecodesStringNumericFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"token": "token-1",
			"holders": []map[string]any{{
				"proxyWallet": "0xholder",
				"amount":      "7.5",
				"pnl":         "1.25",
				"volume":      "100",
			}},
		}})
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL})
	rows, err := client.TopHolders(context.Background(), "condition-1", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Address != "0xholder" || rows[0].Shares != 7.5 || rows[0].Amount != 7.5 || rows[0].Pnl != 1.25 || rows[0].Volume != 100 {
		t.Fatalf("rows=%+v", rows)
	}
}

func TestClientAggregateDTOsDecodeStringNumericFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/value":
			_ = json.NewEncoder(w).Encode(map[string]any{"user": "0xuser", "value": "1234.56", "timestamp": 1714000000})
		case "/traded":
			_ = json.NewEncoder(w).Encode(map[string]any{"user": "0xuser", "traded": "42"})
		case "/oi":
			_ = json.NewEncoder(w).Encode([]map[string]any{{"market": "0xcondition", "asset_id": "token-1", "open_value": "9999.99"}})
		case "/v1/leaderboard":
			_ = json.NewEncoder(w).Encode([]map[string]any{{"rank": "1", "proxyWallet": "0xleader", "vol": "1000000.5", "pnl": "50000.25", "roi": "0.05"}})
		case "/live-volume":
			_ = json.NewEncoder(w).Encode(map[string]any{"total": "42.5", "markets": []map[string]any{{"market": "0xcondition", "value": "42.5"}}, "events": []map[string]any{{"event_id": "event-1", "event_slug": "event", "title": "Event", "volume": "7.25"}}})
		default:
			t.Fatalf("unexpected path=%s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL})
	value, err := client.TotalValue(context.Background(), "0xuser")
	if err != nil {
		t.Fatal(err)
	}
	if value.Value != 1234.56 || value.Timestamp != "1714000000" {
		t.Fatalf("value=%+v", value)
	}
	traded, err := client.MarketsTraded(context.Background(), "0xuser")
	if err != nil {
		t.Fatal(err)
	}
	if traded.MarketsTraded != 42 || traded.Traded != 42 {
		t.Fatalf("traded=%+v", traded)
	}
	oi, err := client.OpenInterest(context.Background(), "0xcondition")
	if err != nil {
		t.Fatal(err)
	}
	if oi.OpenValue != 9999.99 || oi.AssetID != "token-1" {
		t.Fatalf("oi=%+v", oi)
	}
	leaders, err := client.TraderLeaderboard(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(leaders) != 1 || leaders[0].Volume != 1000000.5 || leaders[0].Pnl != 50000.25 || leaders[0].ROI != 0.05 {
		t.Fatalf("leaders=%+v", leaders)
	}
	live, err := client.LiveVolume(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if live.Total != 42.5 || live.Markets[0].Value != 42.5 || live.Events[0].Volume != 7.25 {
		t.Fatalf("live=%+v", live)
	}
}

func TestClientTotalValueDefaultsMissingUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/value" {
			t.Fatalf("path=%s want /value", r.URL.Path)
		}
		if r.URL.Query().Get("user") != "0xuser" {
			t.Fatalf("query=%s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"value":     12.5,
			"timestamp": "2026-01-01T00:00:00Z",
		})
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL})
	row, err := client.TotalValue(context.Background(), "0xuser")
	if err != nil {
		t.Fatal(err)
	}
	if row.User != "0xuser" || row.Value != 12.5 {
		t.Fatalf("row=%+v", row)
	}
}

func TestClientMarketsTradedReturnsTradedAlias(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/traded" {
			t.Fatalf("path=%s want /traded", r.URL.Path)
		}
		if r.URL.Query().Get("user") != "0xuser" {
			t.Fatalf("query=%s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"user":   "0xuser",
			"traded": 7,
		})
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL})
	row, err := client.MarketsTraded(context.Background(), "0xuser")
	if err != nil {
		t.Fatal(err)
	}
	if row.MarketsTraded != 7 || row.Traded != 7 {
		t.Fatalf("row=%+v", row)
	}
}

func TestClientOpenInterestDecodesOpenValueAlias(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oi" {
			t.Fatalf("path=%s want /oi", r.URL.Path)
		}
		if r.URL.Query().Get("market") != "condition-1" {
			t.Fatalf("query=%s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"market":     "condition-1",
			"asset_id":   "token-1",
			"open_value": 42.5,
		}})
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL})
	row, err := client.OpenInterest(context.Background(), "condition-1")
	if err != nil {
		t.Fatal(err)
	}
	if row.Market != "condition-1" || row.AssetID != "token-1" || row.OpenValue != 42.5 {
		t.Fatalf("row=%+v", row)
	}
}

func TestClientTraderLeaderboardDecodesCurrentAliases(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/leaderboard" {
			t.Fatalf("path=%s want /v1/leaderboard", r.URL.Path)
		}
		if r.URL.Query().Get("limit") != "2" {
			t.Fatalf("query=%s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"rank":        1,
			"proxyWallet": "0xleader",
			"vol":         123.5,
			"pnl":         4.25,
			"roi":         0.12,
		}})
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL})
	rows, err := client.TraderLeaderboard(context.Background(), 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].User != "0xleader" || rows[0].Volume != 123.5 {
		t.Fatalf("rows=%+v", rows)
	}
}

func TestClientLiveVolumeReturnsPublicTypes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/live-volume" {
			t.Fatalf("path=%s want /live-volume", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"total": 1,
			"events": []map[string]any{{
				"event_id": "event-1",
				"title":    "Volume event",
				"volume":   42.0,
			}},
		})
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL})
	volume, err := client.LiveVolume(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	var publicVolume *types.LiveVolumeResponse = volume
	if publicVolume.Total != 1 || len(publicVolume.Events) != 1 || publicVolume.Events[0].EventID != "event-1" {
		t.Fatalf("volume=%+v", publicVolume)
	}
}

func TestClientActivityDecodesCamelAssetID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/activity" {
			t.Fatalf("path=%s want /activity", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"type":    "TRADE",
			"market":  "0xmarket",
			"assetId": "token-1",
			"side":    "BUY",
			"price":   0.45,
			"size":    2,
		}})
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL})
	rows, err := client.Activity(context.Background(), "0xuser", 1)
	if err != nil {
		t.Fatal(err)
	}
	var publicRows []types.Activity = rows
	if len(publicRows) != 1 || publicRows[0].AssetID != "token-1" || publicRows[0].Price != "0.45" || publicRows[0].Size != "2" {
		t.Fatalf("rows=%+v", publicRows)
	}
}

func TestClientMarketTradesReturnsPublicTypes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/trades" {
			t.Fatalf("path=%s want /trades", r.URL.Path)
		}
		if r.URL.Query().Get("market") != "0xmarket" || r.URL.Query().Get("limit") != "10" {
			t.Fatalf("query=%s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"conditionId":     "0xmarket",
			"asset":           "yes-token",
			"price":           0.98,
			"size":            3.5,
			"side":            "BUY",
			"outcome":         "Yes",
			"timestamp":       1778314880,
			"transactionHash": "0xtx",
		}})
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL})
	rows, err := client.MarketTrades(context.Background(), "0xmarket", 10)
	if err != nil {
		t.Fatal(err)
	}
	var publicRows []types.Trade = rows
	if len(publicRows) != 1 || publicRows[0].Market != "0xmarket" || publicRows[0].AssetID != "yes-token" {
		t.Fatalf("rows=%+v", publicRows)
	}
}
