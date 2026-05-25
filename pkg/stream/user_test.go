package stream

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestUserClientSubscribeAndDispatchesUserDTOs(t *testing.T) {
	upgrader := websocket.Upgrader{}
	receivedSubscribe := make(chan map[string]interface{}, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		defer conn.Close()

		var sub map[string]interface{}
		if err := conn.ReadJSON(&sub); err != nil {
			t.Errorf("read subscribe: %v", err)
			return
		}
		receivedSubscribe <- sub

		_ = conn.WriteJSON(map[string]interface{}{
			"event_type": "order",
			"order_id":   "ord-1",
			"market":     "condition-1",
			"asset_id":   "token-1",
			"side":       "BUY",
			"price":      "0.5",
			"size":       "10",
			"status":     "live",
			"timestamp":  "1757908892351",
		})
		_ = conn.WriteJSON(map[string]interface{}{
			"event_type":       "trade",
			"trade_id":         "trade-1",
			"order_id":         "ord-1",
			"market":           "condition-1",
			"asset_id":         "token-1",
			"side":             "BUY",
			"price":            "0.5",
			"size":             "10",
			"transaction_hash": "0xtx",
			"timestamp":        "1757908892352",
		})
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	client := NewUserClient(Config{
		URL:          wsURL,
		PingInterval: time.Hour,
		PongTimeout:  time.Second,
		Reconnect:    false,
	}, UserCredentials{Key: "key", Secret: "secret", Passphrase: "pass"})

	gotOrder := make(chan UserOrderMessage, 1)
	gotTrade := make(chan UserTradeMessage, 1)
	client.OnOrder = func(msg UserOrderMessage) { gotOrder <- msg }
	client.OnTrade = func(msg UserTradeMessage) { gotTrade <- msg }

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect returned error: %v", err)
	}
	defer client.Close()
	if err := client.SubscribeUser(ctx, []string{"condition-1"}); err != nil {
		t.Fatalf("SubscribeUser returned error: %v", err)
	}

	select {
	case sub := <-receivedSubscribe:
		if sub["type"] != "user" {
			t.Fatalf("type = %v, want user", sub["type"])
		}
		markets, ok := sub["markets"].([]interface{})
		if !ok || len(markets) != 1 || markets[0] != "condition-1" {
			t.Fatalf("markets = %#v", sub["markets"])
		}
		auth, ok := sub["auth"].(map[string]interface{})
		if !ok || auth["apiKey"] != "key" || auth["secret"] != "secret" || auth["passphrase"] != "pass" {
			b, _ := json.Marshal(sub["auth"])
			t.Fatalf("auth = %s", b)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for subscribe payload")
	}

	select {
	case msg := <-gotOrder:
		if msg.OrderID != "ord-1" || msg.Status != "live" || msg.AssetID != "token-1" {
			t.Fatalf("unexpected order: %+v", msg)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for order event")
	}

	select {
	case msg := <-gotTrade:
		if msg.TradeID != "trade-1" || msg.TransactionHash != "0xtx" {
			t.Fatalf("unexpected trade: %+v", msg)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for trade event")
	}
}

func TestUserClientRejectsMissingCredentials(t *testing.T) {
	client := NewUserClient(Config{URL: "ws://127.0.0.1/unused"}, UserCredentials{Key: "key"})
	if err := client.Connect(context.Background()); err == nil {
		t.Fatal("expected missing credentials error")
	}
}

func TestDefaultUserConfigUsesProductionUserURL(t *testing.T) {
	cfg := DefaultUserConfig("")
	if cfg.URL != "wss://ws-subscriptions-clob.polymarket.com/ws/user" {
		t.Fatalf("URL=%s", cfg.URL)
	}
}
