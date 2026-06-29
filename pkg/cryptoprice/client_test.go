package cryptoprice

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientFetchesOpenPrice(t *testing.T) {
	var gotPath, gotSymbol, gotStart, gotVariant, gotEnd string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotSymbol = r.URL.Query().Get("symbol")
		gotStart = r.URL.Query().Get("eventStartTime")
		gotVariant = r.URL.Query().Get("variant")
		gotEnd = r.URL.Query().Get("endDate")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"openPrice": 79798.44899532206,
			"closePrice": null,
			"timestamp": 1778745536990,
			"completed": false,
			"incomplete": true,
			"cached": true
		}`))
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL})
	start := time.Date(2026, 5, 14, 7, 55, 0, 0, time.UTC)
	end := start.Add(5 * time.Minute)
	price, err := client.CryptoPrice(context.Background(), "btc", start, "fiveminute", end)
	if err != nil {
		t.Fatalf("CryptoPrice returned error: %v", err)
	}

	if gotPath != "/api/crypto/crypto-price" || gotSymbol != "BTC" || gotStart != "2026-05-14T07:55:00Z" || gotVariant != "fiveminute" || gotEnd != "2026-05-14T08:00:00Z" {
		t.Fatalf("request path/query = %s symbol=%s start=%s variant=%s end=%s", gotPath, gotSymbol, gotStart, gotVariant, gotEnd)
	}
	if price.OpenPrice != 79798.44899532206 || price.ClosePrice != nil || price.TimestampMillis != 1778745536990 || !price.Incomplete || !price.Cached {
		t.Fatalf("price = %+v", price)
	}
}

func TestClientRejectsInvalidOpenPrice(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"openPrice":0}`))
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL})
	_, err := client.CryptoPrice(context.Background(), "BTC", time.Date(2026, 5, 14, 7, 55, 0, 0, time.UTC), "fiveminute", time.Time{})
	if err == nil {
		t.Fatalf("CryptoPrice error = nil, want invalid open price error")
	}
}

func TestClientRequiresInputs(t *testing.T) {
	client := NewClient(Config{})
	if _, err := client.CryptoPrice(context.Background(), "", time.Now(), "fiveminute", time.Time{}); err == nil {
		t.Fatal("empty symbol: error = nil, want error")
	}
	if _, err := client.CryptoPrice(context.Background(), "BTC", time.Time{}, "fiveminute", time.Time{}); err == nil {
		t.Fatal("zero eventStart: error = nil, want error")
	}
}
