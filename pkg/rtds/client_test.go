package rtds

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// sampleChainlinkMessage is a captured production RTDS frame (2026-07-06).
const sampleChainlinkMessage = `{"connection_id":"gTPq5o-8pWeIKEhV_A==","payload":{"full_accuracy_value":"63621393086946047500000","symbol":"btc/usd","timestamp":1783369349000,"value":63621.393086946046},"timestamp":1783369350000,"topic":"crypto_prices_chainlink","type":"update"}`

func TestProcessMessage_ParsesChainlinkUpdate(t *testing.T) {
	client := NewClient(DefaultConfig(""), nil)
	var got ChainlinkPriceEvent
	client.OnChainlinkPrice = func(ev ChainlinkPriceEvent) { got = ev }

	now := time.Now()
	client.processMessage([]byte(sampleChainlinkMessage), now)

	if got.Symbol != "btc/usd" {
		t.Fatalf("symbol=%q", got.Symbol)
	}
	if got.Value != 63621.393086946046 {
		t.Fatalf("value=%v", got.Value)
	}
	if got.FullAccuracyValue != "63621393086946047500000" {
		t.Fatalf("full_accuracy_value=%q", got.FullAccuracyValue)
	}
	if got.FeedTime != time.UnixMilli(1783369349000).UTC() {
		t.Fatalf("feed time=%v", got.FeedTime)
	}
	if !got.ObservedAt.Equal(now.UTC()) {
		t.Fatalf("observed at=%v", got.ObservedAt)
	}
}

func TestProcessMessage_FeedFilterAndForeignTopics(t *testing.T) {
	client := NewClient(DefaultConfig(""), []string{"ETH/USD"})
	var events []ChainlinkPriceEvent
	client.OnChainlinkPrice = func(ev ChainlinkPriceEvent) { events = append(events, ev) }

	// Filtered-out feed.
	client.processMessage([]byte(sampleChainlinkMessage), time.Now())
	// Foreign topic (binance crypto_prices) must never surface.
	client.processMessage([]byte(`{"payload":{"symbol":"ethusdt","timestamp":1,"value":1789.14},"topic":"crypto_prices","type":"update"}`), time.Now())
	// Keepalive/garbage frames must be tolerated.
	client.processMessage([]byte(""), time.Now())
	client.processMessage([]byte("not json"), time.Now())
	if len(events) != 0 {
		t.Fatalf("expected no events, got %+v", events)
	}

	// Matching feed passes (filter is case-insensitive).
	matching := strings.ReplaceAll(sampleChainlinkMessage, "btc/usd", "eth/usd")
	client.processMessage([]byte(matching), time.Now())
	if len(events) != 1 || events[0].Symbol != "eth/usd" {
		t.Fatalf("expected one eth/usd event, got %+v", events)
	}
}

func TestProcessMessage_BatchedArrayFrame(t *testing.T) {
	client := NewClient(DefaultConfig(""), nil)
	var events []ChainlinkPriceEvent
	client.OnChainlinkPrice = func(ev ChainlinkPriceEvent) { events = append(events, ev) }

	batch := "[" + sampleChainlinkMessage + "," + strings.ReplaceAll(sampleChainlinkMessage, "btc/usd", "sol/usd") + "]"
	client.processMessage([]byte(batch), time.Now())
	if len(events) != 2 || events[0].Symbol != "btc/usd" || events[1].Symbol != "sol/usd" {
		t.Fatalf("expected btc+sol events, got %+v", events)
	}
}

func TestConnect_SubscribesAndDeliversOverWebSocket(t *testing.T) {
	upgrader := websocket.Upgrader{}
	var mu sync.Mutex
	var subscribeRaw []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		defer conn.Close()
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}
		mu.Lock()
		subscribeRaw = msg
		mu.Unlock()
		if err := conn.WriteMessage(websocket.TextMessage, []byte(sampleChainlinkMessage)); err != nil {
			return
		}
		// Hold the connection open until the client disconnects.
		conn.ReadMessage()
	}))
	defer server.Close()

	cfg := DefaultConfig("ws" + strings.TrimPrefix(server.URL, "http"))
	cfg.Reconnect = false
	client := NewClient(cfg, []string{"btc/usd"})
	delivered := make(chan ChainlinkPriceEvent, 1)
	client.OnChainlinkPrice = func(ev ChainlinkPriceEvent) { delivered <- ev }

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer client.Close()

	select {
	case ev := <-delivered:
		if ev.Symbol != "btc/usd" || ev.Value != 63621.393086946046 {
			t.Fatalf("unexpected event: %+v", ev)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no chainlink event delivered")
	}
	if !client.Connected() {
		t.Fatal("client should report connected")
	}

	var sub struct {
		Action        string `json:"action"`
		Subscriptions []struct {
			Topic   string `json:"topic"`
			MsgType string `json:"type"`
		} `json:"subscriptions"`
	}
	mu.Lock()
	raw := subscribeRaw
	mu.Unlock()
	if err := json.Unmarshal(raw, &sub); err != nil {
		t.Fatalf("subscribe payload: %v (%s)", err, raw)
	}
	if sub.Action != "subscribe" || len(sub.Subscriptions) != 1 ||
		sub.Subscriptions[0].Topic != "crypto_prices_chainlink" || sub.Subscriptions[0].MsgType != "*" {
		t.Fatalf("unexpected subscribe payload: %s", raw)
	}
}

func TestDefaultConfig_UsesProductionURL(t *testing.T) {
	cfg := DefaultConfig("")
	if cfg.URL != DefaultURL {
		t.Fatalf("URL=%s", cfg.URL)
	}
	if cfg.PingInterval == 0 || cfg.PongTimeout == 0 {
		t.Fatal("keepalive defaults missing")
	}
}
