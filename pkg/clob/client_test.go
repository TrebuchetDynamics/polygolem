package clob

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/polygolem/pkg/types"
)

const testPrivateKey = "0x4c0883a69102937d6231471b5dbb6204fe5129617082792ae468d01a3f362318"
const testBuilderCode = "0x1111111111111111111111111111111111111111111111111111111111111111"
const testDepositWallet = "0x19bE70b1e4F59C0663a999C0dC6f5b3C68CFCaF3"

// EOA derived from testPrivateKey. Per the 2026-05-08 web-UI capture,
// CLOB POLY_ADDRESS is always the EOA at the HTTP layer; the
// deposit-wallet identity rides on the order body's signatureType=3.
const testEOA = "0x2c7536E3605D9C16a7a3D7b1898e529396a65c23"

func TestPublicOrderRecordDecodesCamelCaseAliases(t *testing.T) {
	var row OrderRecord
	raw := `{"id":"ord-1","assetId":"token-1","originalSize":10,"sizeMatched":5,"orderType":"GTC","signatureType":"3","createdAt":1710000000,"makerAddress":"0xmaker","associateTrades":[1,"trade-2"]}`
	if err := json.Unmarshal([]byte(raw), &row); err != nil {
		t.Fatal(err)
	}
	if row.AssetID != "token-1" || row.OriginalSize != "10" || row.SizeMatched != "5" || row.OrderType != "GTC" {
		t.Fatalf("unexpected decoded order: %+v", row)
	}
	if row.SignatureType != 3 || row.CreatedAt != "1710000000" || row.MakerAddress != "0xmaker" {
		t.Fatalf("unexpected scalar fields: %+v", row)
	}
	if got := strings.Join(row.AssociateTrades, ","); got != "1,trade-2" {
		t.Fatalf("AssociateTrades=%q", got)
	}
}

func TestPublicTradeRecordDecodesCamelCaseAliases(t *testing.T) {
	var row TradeRecord
	raw := `{"id":"trade-1","assetId":"token-1","feeRateBps":10,"matchedAmount":5,"transactionHash":"0xhash","createdAt":1710000000,"lastUpdated":1710000060}`
	if err := json.Unmarshal([]byte(raw), &row); err != nil {
		t.Fatal(err)
	}
	if row.AssetID != "token-1" || row.FeeRateBps != "10" || row.MatchedAmount != "5" || row.TransactionHash != "0xhash" || row.CreatedAt != "1710000000" || row.LastUpdated != "1710000060" {
		t.Fatalf("unexpected decoded trade: %+v", row)
	}
}

func TestPublicOrderRecordDecodesNumericAssociateTrades(t *testing.T) {
	var row OrderRecord
	if err := json.Unmarshal([]byte(`{"id":"ord-1","associate_trades":[1,"trade-2"]}`), &row); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(row.AssociateTrades, ","); got != "1,trade-2" {
		t.Fatalf("AssociateTrades=%q", got)
	}
}

func TestPublicOrderRecordDecodesNumericScalarFields(t *testing.T) {
	var row OrderRecord
	raw := `{"id":"ord-1","original_size":10,"size_matched":5,"price":0.49,"signature_type":"3","created_at":1710000000,"expiration":0}`
	if err := json.Unmarshal([]byte(raw), &row); err != nil {
		t.Fatal(err)
	}
	if row.OriginalSize != "10" || row.SizeMatched != "5" || row.Price != "0.49" {
		t.Fatalf("unexpected amount fields: %+v", row)
	}
	if row.SignatureType != 3 || row.CreatedAt != "1710000000" || row.Expiration != "0" {
		t.Fatalf("unexpected scalar fields: %+v", row)
	}
}

func TestPublicTradeRecordDecodesNumericScalarFields(t *testing.T) {
	var row TradeRecord
	raw := `{"id":"trade-1","price":0.52,"size":10,"fee_rate_bps":0,"matched_amount":10,"created_at":1710000000,"last_updated":1710000060}`
	if err := json.Unmarshal([]byte(raw), &row); err != nil {
		t.Fatal(err)
	}
	if row.Price != "0.52" || row.Size != "10" || row.FeeRateBps != "0" || row.MatchedAmount != "10" {
		t.Fatalf("unexpected amount fields: %+v", row)
	}
	if row.CreatedAt != "1710000000" || row.LastUpdated != "1710000060" {
		t.Fatalf("unexpected timestamp fields: %+v", row)
	}
}

func TestBuilderFeeKeyRecordDecodesCamelCaseTimestamps(t *testing.T) {
	var row BuilderFeeKeyRecord
	if err := json.Unmarshal([]byte(`{"key":"fee-1","createdAt":"2026-05-08T00:00:00Z","updatedAt":"2026-05-08T01:00:00Z"}`), &row); err != nil {
		t.Fatal(err)
	}
	if row.CreatedAt != "2026-05-08T00:00:00Z" || row.UpdatedAt != "2026-05-08T01:00:00Z" {
		t.Fatalf("row=%+v", row)
	}
}

func TestOrderPlacementResponseDecodesSnakeCaseErrorMessage(t *testing.T) {
	var resp OrderPlacementResponse
	if err := json.Unmarshal([]byte(`{"success":false,"orderID":"0xorder","error_msg":"price out of band"}`), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.ErrorMsg != "price out of band" {
		t.Fatalf("ErrorMsg=%q", resp.ErrorMsg)
	}
}

func TestCancelOrdersResponseDecodesCamelCaseNotCanceled(t *testing.T) {
	var resp CancelOrdersResponse
	if err := json.Unmarshal([]byte(`{"canceled":["0x1"],"notCanceled":{"0x2":"already filled"}}`), &resp); err != nil {
		t.Fatal(err)
	}
	if got := resp.NotCanceled["0x2"]; got != "already filled" {
		t.Fatalf("NotCanceled[0x2]=%q", got)
	}
}

func TestCancelOrdersResponseDecodesNumericValues(t *testing.T) {
	var resp CancelOrdersResponse
	if err := json.Unmarshal([]byte(`{"canceled":[1,"0x2"],"not_canceled":{"0x3":404,"0x4":"already filled"}}`), &resp); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(resp.Canceled, ","); got != "1,0x2" {
		t.Fatalf("Canceled=%q", got)
	}
	if got := resp.NotCanceled["0x3"]; got != "404" {
		t.Fatalf("NotCanceled[0x3]=%q", got)
	}
	if got := resp.NotCanceled["0x4"]; got != "already filled" {
		t.Fatalf("NotCanceled[0x4]=%q", got)
	}
}

func TestOrderPlacementResponseDecodesCamelResponseAliases(t *testing.T) {
	var resp OrderPlacementResponse
	if err := json.Unmarshal([]byte(`{"success":true,"orderID":"0xorder","status":"matched","transactionHash":"0xtx","tradeIds":["trade-1"]}`), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.TransactionHash != "0xtx" {
		t.Fatalf("TransactionHash=%q", resp.TransactionHash)
	}
	if got := strings.Join(resp.TradeIDs, ","); got != "trade-1" {
		t.Fatalf("TradeIDs=%q", got)
	}
}

func TestOrderPlacementResponseDecodesTransactionHashesAlias(t *testing.T) {
	var resp OrderPlacementResponse
	if err := json.Unmarshal([]byte(`{"success":true,"orderID":"0xorder","status":"matched","transactionHashes":["0xtx"]}`), &resp); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(resp.TransactionsHashes, ","); got != "0xtx" {
		t.Fatalf("TransactionsHashes=%q", got)
	}
}

func TestOrderPlacementResponseDecodesNumericListFields(t *testing.T) {
	var resp OrderPlacementResponse
	if err := json.Unmarshal([]byte(`{"success":true,"orderID":"0xorder","transactionHashes":[1,"0xtx"],"tradeIDs":[2,"trade-3"]}`), &resp); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(resp.TransactionsHashes, ","); got != "1,0xtx" {
		t.Fatalf("TransactionsHashes=%q", got)
	}
	if got := strings.Join(resp.TradeIDs, ","); got != "2,trade-3" {
		t.Fatalf("TradeIDs=%q", got)
	}
}

func TestOrderPlacementResponseDecodesNumericAmountFields(t *testing.T) {
	var resp OrderPlacementResponse
	if err := json.Unmarshal([]byte(`{"success":true,"orderID":"0xorder","status":"matched","makingAmount":100,"takingAmount":200}`), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.MakingAmount != "100" || resp.TakingAmount != "200" {
		t.Fatalf("unexpected decoded response: %+v", resp)
	}
}

func TestOrderPlacementResponseDecodesSnakeCaseFields(t *testing.T) {
	var resp OrderPlacementResponse
	raw := `{"success":true,"order_id":"0xorder","status":"matched","making_amount":"100","taking_amount":"200","transaction_hash":"0xtx","transaction_hashes":["0xtx2"],"trade_ids":["trade-1"]}`
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.OrderID != "0xorder" || resp.MakingAmount != "100" || resp.TakingAmount != "200" || resp.TransactionHash != "0xtx" {
		t.Fatalf("unexpected decoded response: %+v", resp)
	}
	if got := strings.Join(resp.TransactionsHashes, ","); got != "0xtx2" {
		t.Fatalf("TransactionsHashes=%q", got)
	}
	if got := strings.Join(resp.TradeIDs, ","); got != "trade-1" {
		t.Fatalf("TradeIDs=%q", got)
	}
}

func TestClientOrderBookReturnsPublicDTO(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/book" || r.URL.Query().Get("token_id") != "token-1" {
			t.Fatalf("unexpected request: %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"market":"condition-1",
			"asset_id":"token-1",
			"timestamp":"1710000000",
			"hash":"book-hash",
			"bids":[{"price":"0.44","size":"10"}],
			"asks":[{"price":"0.46","size":"11"}],
			"min_order_size":"5",
			"tick_size":"0.01",
			"neg_risk":true,
			"last_trade_price":"0.45"
		}`))
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL})
	book, err := client.OrderBook(context.Background(), "token-1")
	if err != nil {
		t.Fatalf("OrderBook returned error: %v", err)
	}

	var publicBook *types.CLOBOrderBook = book
	if publicBook.Market != "condition-1" || publicBook.AssetID != "token-1" {
		t.Fatalf("unexpected order book identity: %+v", publicBook)
	}
	if publicBook.TickSize != "0.01" || publicBook.NegRisk != true || publicBook.LastTradePrice != "0.45" {
		t.Fatalf("missing CLOB book metadata: %+v", publicBook)
	}
	if got := publicBook.Bids[0]; got.Price != "0.44" || got.Size != "10" {
		t.Fatalf("unexpected bid level: %+v", got)
	}
}

func TestClientMarketReturnsPublicDTO(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/markets/condition-1" {
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"condition_id":"condition-1",
			"game_start_time":"2026-01-01T00:00:00Z",
			"tokens":[
				{"token_id":"token-yes","outcome":"Yes","price":1,"winner":true},
				{"token_id":"token-no","outcome":"No","price":0,"winner":false}
			],
			"order_min_size":5,
			"order_price_min_tick_size":0.01,
			"maker_base_fee":0,
			"taker_base_fee":0,
			"closed":true,
			"rfq_enabled":true,
			"taker_order_delay":true,
			"blockaid_check_enabled":true,
			"fee_details":{"rate":0.02,"exponent":2,"taker_only":true},
			"minimum_order_age":123
		}`))
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL})
	market, err := client.Market(context.Background(), "condition-1")
	if err != nil {
		t.Fatalf("Market returned error: %v", err)
	}

	var publicMarket *types.CLOBMarket = market
	if publicMarket.ConditionID != "condition-1" || len(publicMarket.Tokens) != 2 {
		t.Fatalf("unexpected market: %+v", publicMarket)
	}
	if got := publicMarket.Tokens[0]; got.TokenID != "token-yes" || got.Outcome != "Yes" || !got.Winner {
		t.Fatalf("unexpected token conversion: %+v", got)
	}
	if !publicMarket.Closed {
		t.Fatalf("closed market status not preserved: %+v", publicMarket)
	}
	if publicMarket.OrderMinSize != 5 || publicMarket.OrderPriceMinTickSize != 0.01 {
		t.Fatalf("unexpected market order constraints: %+v", publicMarket)
	}
	if !publicMarket.RFQEnabled || !publicMarket.TakerOrderDelay || !publicMarket.BlockaidCheckEnabled {
		t.Fatalf("missing current CLOB market flags: %+v", publicMarket)
	}
	if publicMarket.FeeDetails.Rate != 0.02 || publicMarket.FeeDetails.Exponent != 2 || !publicMarket.FeeDetails.TakerOnly {
		t.Fatalf("unexpected fee details: %+v", publicMarket.FeeDetails)
	}
}

func TestClientMarketByTokenReturnsPublicDTO(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/markets-by-token/token-1" {
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"condition_id":"condition-1",
			"primary_token_id":"token-yes",
			"secondary_token_id":"token-no"
		}`))
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL})
	got, err := client.MarketByToken(context.Background(), "token-1")
	if err != nil {
		t.Fatalf("MarketByToken returned error: %v", err)
	}

	var publicResponse *types.CLOBMarketByTokenResponse = got
	if publicResponse.ConditionID != "condition-1" ||
		publicResponse.PrimaryTokenID != "token-yes" ||
		publicResponse.SecondaryTokenID != "token-no" {
		t.Fatalf("unexpected market-by-token response: %+v", publicResponse)
	}
}

func TestClientScalarMarketDataParsesCurrentNumericDTOs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/price":
			_, _ = w.Write([]byte(`{"price":0.45}`))
		case "/midpoint":
			_, _ = w.Write([]byte(`{"mid_price":0.5}`))
		case "/fee-rate":
			_, _ = w.Write([]byte(`{"base_fee":30}`))
		case "/prices-history":
			_, _ = w.Write([]byte(`{"history":[{"t":123,"p":0.45}]}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL})
	price, err := client.Price(context.Background(), "token-1", "BUY")
	if err != nil {
		t.Fatalf("Price returned error: %v", err)
	}
	if price != "0.45" {
		t.Fatalf("Price = %q, want 0.45", price)
	}
	midpoint, err := client.Midpoint(context.Background(), "token-1")
	if err != nil {
		t.Fatalf("Midpoint returned error: %v", err)
	}
	if midpoint != "0.5" {
		t.Fatalf("Midpoint = %q, want 0.5", midpoint)
	}
	fee, err := client.FeeRateBps(context.Background(), "token-1")
	if err != nil {
		t.Fatalf("FeeRateBps returned error: %v", err)
	}
	if fee != 30 {
		t.Fatalf("FeeRateBps = %d, want 30", fee)
	}
	history, err := client.PricesHistory(context.Background(), &types.CLOBPriceHistoryParams{Market: "token-1"})
	if err != nil {
		t.Fatalf("PricesHistory returned error: %v", err)
	}
	if got := history.History[0]; got.T != "123" || got.P != "0.45" {
		t.Fatalf("unexpected history point: %+v", got)
	}
}

func TestClientCreateAPIKeyForAddressReturnsPublicAPIKey(t *testing.T) {
	var sawAddress string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/auth/api-key" || r.Method != http.MethodPost {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		sawAddress = r.Header.Get("POLY_ADDRESS")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"apiKey":"owner-key","secret":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=","passphrase":"owner-pass"}`))
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL})
	key, err := client.CreateAPIKeyForAddress(context.Background(), testPrivateKey, testDepositWallet)
	if err != nil {
		t.Fatalf("CreateAPIKeyForAddress returned error: %v", err)
	}
	if !strings.EqualFold(sawAddress, testEOA) {
		t.Fatalf("POLY_ADDRESS = %s, want EOA %s", sawAddress, testEOA)
	}
	if key.Key != "owner-key" || key.Passphrase != "owner-pass" {
		t.Fatalf("unexpected key: %+v", key)
	}
}

func TestClientDeriveAPIKeyForAddressReturnsPublicAPIKey(t *testing.T) {
	var sawAddress string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/auth/derive-api-key" || r.Method != http.MethodGet {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		sawAddress = r.Header.Get("POLY_ADDRESS")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"apiKey":"owner-key","secret":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=","passphrase":"owner-pass"}`))
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL})
	key, err := client.DeriveAPIKeyForAddress(context.Background(), testPrivateKey, testDepositWallet)
	if err != nil {
		t.Fatalf("DeriveAPIKeyForAddress returned error: %v", err)
	}
	if !strings.EqualFold(sawAddress, testEOA) {
		t.Fatalf("POLY_ADDRESS = %s, want EOA %s", sawAddress, testEOA)
	}
	if key.Key != "owner-key" || key.Passphrase != "owner-pass" {
		t.Fatalf("unexpected key: %+v", key)
	}
}

func TestClientBatchMarketDataParsesCurrentDTOs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/prices":
			_, _ = w.Write([]byte(`{"token-1":{"BUY":0.45},"token-2":{"SELL":0.52}}`))
		case "/midpoints":
			_, _ = w.Write([]byte(`{"token-1":0.5,"token-2":"0.51"}`))
		case "/last-trades-prices":
			_, _ = w.Write([]byte(`[{"token_id":"token-1","price":"0.44","side":"BUY"},{"token_id":"token-2","price":"0.53","side":"SELL"}]`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL})
	params := []types.CLOBBookParams{
		{TokenID: "token-1", Side: "BUY"},
		{TokenID: "token-2", Side: "SELL"},
	}
	prices, err := client.Prices(context.Background(), params)
	if err != nil {
		t.Fatalf("Prices returned error: %v", err)
	}
	if prices["token-1"] != "0.45" || prices["token-2"] != "0.52" {
		t.Fatalf("unexpected prices: %+v", prices)
	}
	midpoints, err := client.Midpoints(context.Background(), params)
	if err != nil {
		t.Fatalf("Midpoints returned error: %v", err)
	}
	if midpoints["token-1"] != "0.5" || midpoints["token-2"] != "0.51" {
		t.Fatalf("unexpected midpoints: %+v", midpoints)
	}
	lastTrades, err := client.LastTradesPrices(context.Background(), params)
	if err != nil {
		t.Fatalf("LastTradesPrices returned error: %v", err)
	}
	if lastTrades["token-1"] != "0.44" || lastTrades["token-2"] != "0.53" {
		t.Fatalf("unexpected last trades: %+v", lastTrades)
	}
}

func TestClientAuthenticatedMethodsReturnPublicDTOs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/auth/derive-api-key":
			_, _ = w.Write([]byte(`{"apiKey":"api-key","secret":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=","passphrase":"pass"}`))
		case "/data/orders":
			_, _ = w.Write([]byte(`[{
				"id":"0xorder",
				"status":"ORDER_STATUS_LIVE",
				"market":"0xmarket",
				"asset_id":"token-1",
				"side":"BUY",
				"original_size":10,
				"size_matched":2,
				"price":0.45,
				"outcome":"Yes",
				"type":"limit",
				"order_type":"GTC",
				"maker_address":"0xmaker",
				"owner":"api-key",
				"associate_trades":["trade-1"],
				"expiration":"0",
				"created_at":"1710000000"
			}]`))
		case "/data/trades":
			_, _ = w.Write([]byte(`[{
				"id":"trade-1",
				"status":"MATCHED",
				"market":"0xmarket",
				"asset_id":"token-1",
				"side":"BUY",
				"price":0.45,
				"size":2,
				"fee_rate_bps":0,
				"outcome":"Yes",
				"owner":"api-key",
				"builder":"builder",
				"matched_amount":2,
				"transaction_hash":"0xtx",
				"created_at":"1710000000",
				"last_updated":"1710000001"
			}]`))
		case "/order/0xorder":
			_, _ = w.Write([]byte(`{"id":"0xorder","status":"ORDER_STATUS_LIVE","order_type":"GTC"}`))
		case "/balance-allowance":
			if got := r.URL.Query().Get("signature_type"); got != "3" {
				t.Fatalf("signature_type = %q, want 3", got)
			}
			_, _ = w.Write([]byte(`{"balance":"1000000","allowance":"999"}`))
		case "/order":
			if r.Method != http.MethodDelete {
				t.Fatalf("method = %s, want DELETE", r.Method)
			}
			_, _ = w.Write([]byte(`{"canceled":["0xorder"],"not_canceled":{"0xother":"not found"}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL})
	orders, err := client.ListOrders(context.Background(), testPrivateKey)
	if err != nil {
		t.Fatalf("ListOrders returned error: %v", err)
	}
	var publicOrders []OrderRecord = orders
	if len(publicOrders) != 1 || publicOrders[0].Type != "limit" || publicOrders[0].OrderType != "GTC" || publicOrders[0].AssetID != "token-1" || publicOrders[0].OriginalSize != "10" || publicOrders[0].SizeMatched != "2" || publicOrders[0].Price != "0.45" {
		t.Fatalf("unexpected public orders: %+v", publicOrders)
	}

	trades, err := client.ListTrades(context.Background(), testPrivateKey)
	if err != nil {
		t.Fatalf("ListTrades returned error: %v", err)
	}
	var publicTrades []TradeRecord = trades
	if len(publicTrades) != 1 || publicTrades[0].TransactionHash != "0xtx" || publicTrades[0].Price != "0.45" || publicTrades[0].Size != "2" || publicTrades[0].MatchedAmount != "2" {
		t.Fatalf("unexpected public trades: %+v", publicTrades)
	}

	order, err := client.Order(context.Background(), testPrivateKey, "0xorder")
	if err != nil {
		t.Fatalf("Order returned error: %v", err)
	}
	var publicOrder *OrderRecord = order
	if publicOrder.ID != "0xorder" || publicOrder.OrderType != "GTC" {
		t.Fatalf("unexpected public order: %+v", publicOrder)
	}

	balance, err := client.BalanceAllowance(context.Background(), testPrivateKey, BalanceAllowanceParams{AssetType: "COLLATERAL"})
	if err != nil {
		t.Fatalf("BalanceAllowance returned error: %v", err)
	}
	var publicBalance *BalanceAllowanceResponse = balance
	if publicBalance.Balance != "1000000" || publicBalance.Allowance != "999" {
		t.Fatalf("unexpected public balance: %+v", publicBalance)
	}

	cancel, err := client.CancelOrder(context.Background(), testPrivateKey, "0xorder")
	if err != nil {
		t.Fatalf("CancelOrder returned error: %v", err)
	}
	var publicCancel *CancelOrdersResponse = cancel
	if len(publicCancel.Canceled) != 1 || publicCancel.NotCanceled["0xother"] != "not found" {
		t.Fatalf("unexpected public cancel response: %+v", publicCancel)
	}
}

func TestClientPublicTrades(t *testing.T) {
	var gotMarket string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/trades" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		gotMarket = r.URL.Query().Get("market")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"trades":[{"id":"trade-1","status":"MATCHED","market":"condition-1","asset_id":"token-1","side":"BUY","price":"0.52","size":"10","transaction_hash":"0xtx"}]}`))
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL})
	trades, err := client.PublicTrades(context.Background(), " condition-1 ")
	if err != nil {
		t.Fatalf("PublicTrades returned error: %v", err)
	}
	if gotMarket != "condition-1" {
		t.Fatalf("market query=%q want condition-1", gotMarket)
	}
	if len(trades) != 1 || trades[0].ID != "trade-1" || trades[0].TransactionHash != "0xtx" {
		t.Fatalf("unexpected trades: %+v", trades)
	}
}

func TestClientUsesConfiguredCredentialsForAuthenticatedMethods(t *testing.T) {
	var derived bool
	var orderAPIKey string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/auth/derive-api-key":
			derived = true
			http.Error(w, "derive should not be called", http.StatusTeapot)
		case "/data/orders":
			orderAPIKey = r.Header.Get("POLY_API_KEY")
			_, _ = w.Write([]byte(`[{"id":"0xorder","status":"ORDER_STATUS_LIVE","signature_type":"3"}]`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL: server.URL,
		Credentials: APIKey{
			Key:        "configured-key",
			Secret:     "c2VjcmV0",
			Passphrase: "pass",
		},
	})
	orders, err := client.ListOrders(context.Background(), testPrivateKey)
	if err != nil {
		t.Fatalf("ListOrders returned error: %v", err)
	}
	if derived {
		t.Fatal("ListOrders called /auth/derive-api-key despite configured credentials")
	}
	if orderAPIKey != "configured-key" {
		t.Fatalf("POLY_API_KEY=%q want configured-key", orderAPIKey)
	}
	if len(orders) != 1 || orders[0].ID != "0xorder" || orders[0].SignatureType != 3 {
		t.Fatalf("orders=%+v", orders)
	}
}

func TestClientBalanceAllowanceDecodesNumericAllowanceValues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/balance-allowance" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		_, _ = w.Write([]byte(`{"balance":1000000,"allowance":999,"allowances":{"0xCtfExchangeV2":999999999}}`))
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL: server.URL,
		Credentials: APIKey{
			Key:        "configured-key",
			Secret:     "c2VjcmV0",
			Passphrase: "pass",
		},
	})
	balance, err := client.BalanceAllowance(context.Background(), testPrivateKey, BalanceAllowanceParams{AssetType: "COLLATERAL"})
	if err != nil {
		t.Fatalf("BalanceAllowance returned error: %v", err)
	}
	if balance.Balance != "1000000" || balance.Allowance != "999" || balance.Allowances["0xCtfExchangeV2"] != "999999999" {
		t.Fatalf("unexpected balance allowance: %+v", balance)
	}
}

func TestClientCreateLimitOrderUsesConfiguredBuilderCode(t *testing.T) {
	var posted map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/auth/derive-api-key":
			_, _ = w.Write([]byte(`{"apiKey":"api-key","secret":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=","passphrase":"pass"}`))
		case "/tick-size":
			_, _ = w.Write([]byte(`{"minimum_tick_size":"0.001"}`))
		case "/neg-risk":
			_, _ = w.Write([]byte(`{"neg_risk":false}`))
		case "/order":
			if err := json.NewDecoder(r.Body).Decode(&posted); err != nil {
				t.Fatalf("decode order body: %v", err)
			}
			_, _ = w.Write([]byte(`{"success":true,"orderID":"0xabc","status":"matched"}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL, BuilderCode: testBuilderCode})
	_, err := client.CreateLimitOrder(context.Background(), testPrivateKey, CreateOrderParams{
		TokenID:   "12345",
		Side:      "BUY",
		Price:     "0.500000",
		Size:      "1.400000",
		OrderType: "GTC",
		PostOnly:  true,
	})
	if err != nil {
		t.Fatal(err)
	}

	order, ok := posted["order"].(map[string]any)
	if !ok {
		t.Fatalf("posted order missing: %#v", posted)
	}
	if order["builder"] != testBuilderCode {
		t.Fatalf("posted builder=%#v want %s", order["builder"], testBuilderCode)
	}
	if posted["postOnly"] != true {
		t.Fatalf("postOnly=%#v want true", posted["postOnly"])
	}
}

func TestClientCreateBatchOrdersReturnsPublicResponses(t *testing.T) {
	var posted []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/tick-size":
			_, _ = w.Write([]byte(`{"minimum_tick_size":"0.01"}`))
		case "/auth/derive-api-key":
			_, _ = w.Write([]byte(`{"apiKey":"api-key","secret":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=","passphrase":"pass"}`))
		case "/neg-risk":
			_, _ = w.Write([]byte(`{"neg_risk":false}`))
		case "/orders":
			if r.Method != http.MethodPost {
				t.Fatalf("method=%s want POST", r.Method)
			}
			if err := json.NewDecoder(r.Body).Decode(&posted); err != nil {
				t.Fatalf("decode orders body: %v", err)
			}
			_, _ = w.Write([]byte(`[{"success":true,"orderID":"0x1","status":"live"},{"success":true,"orderID":"0x2","status":"live"}]`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL})
	res, err := client.CreateBatchOrders(context.Background(), testPrivateKey, []CreateOrderParams{
		{TokenID: "12345", Side: "BUY", Price: "0.500000", Size: "1.000000", OrderType: "GTC"},
		{TokenID: "12346", Side: "SELL", Price: "0.600000", Size: "2.000000", OrderType: "GTC", PostOnly: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Orders) != 2 || res.Orders[0].OrderID != "0x1" || res.Orders[1].OrderID != "0x2" {
		t.Fatalf("unexpected batch response: %+v", res)
	}
	if len(posted) != 2 || posted[1]["postOnly"] != true {
		t.Fatalf("unexpected posted batch: %#v", posted)
	}
}

func TestClientOrdersScoringPostsOrderIDs(t *testing.T) {
	var posted map[string][]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/orders/scoring" || r.Method != http.MethodPost {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		if err := json.NewDecoder(r.Body).Decode(&posted); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[true,false,true]`))
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL})
	got, err := client.OrdersScoring(context.Background(), []string{"order-1", "order-2", "order-3"})
	if err != nil {
		t.Fatalf("OrdersScoring returned error: %v", err)
	}
	if len(got) != 3 || !got[0] || got[1] || !got[2] {
		t.Fatalf("unexpected scoring result: %+v", got)
	}
	want := []string{"order-1", "order-2", "order-3"}
	if len(posted["order_ids"]) != len(want) {
		t.Fatalf("posted order_ids=%+v", posted["order_ids"])
	}
	for i := range want {
		if posted["order_ids"][i] != want[i] {
			t.Fatalf("posted order_ids=%+v want %+v", posted["order_ids"], want)
		}
	}
}

func TestClientHeartbeatPostsPublicRoute(t *testing.T) {
	var posted map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/auth/derive-api-key":
			_, _ = w.Write([]byte(`{"apiKey":"api-key","secret":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=","passphrase":"pass"}`))
		case "/v1/heartbeats":
			if r.Method != http.MethodPost {
				t.Fatalf("method=%s want POST", r.Method)
			}
			if err := json.NewDecoder(r.Body).Decode(&posted); err != nil {
				t.Fatalf("decode heartbeat body: %v", err)
			}
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL})
	if err := client.Heartbeat(context.Background(), testPrivateKey, "hb-123"); err != nil {
		t.Fatal(err)
	}
	if posted["heartbeat_id"] != "hb-123" {
		t.Fatalf("heartbeat_id=%#v want hb-123", posted["heartbeat_id"])
	}
}

func TestMarketTradesProbeExposesRedactedSummary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/auth/derive-api-key":
			_, _ = w.Write([]byte(`{"apiKey":"api-key","secret":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=","passphrase":"pass"}`))
		case "/data/trades":
			_, _ = w.Write([]byte(`{"limit":100,"next_cursor":"LTE=","count":1,"data":[{"id":"trade-1","market":"0x1111111111111111111111111111111111111111111111111111111111111111","asset_id":"12345","owner":"secret-owner","match_time":"1700000000"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL})
	res, err := client.MarketTradesProbe(context.Background(), testPrivateKey, MarketTradesProbeRequest{
		AssetID: "12345",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Classification != ProbeAccountScoped || res.RowCount != 1 {
		t.Fatalf("probe=%+v", res)
	}
	if strings.Contains(fmt.Sprintf("%+v", res), "secret-owner") {
		t.Fatalf("probe leaked raw row identity: %+v", res)
	}
}
