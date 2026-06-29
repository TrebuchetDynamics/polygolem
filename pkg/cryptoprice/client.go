// Package cryptoprice is a read-only client for the Polymarket web
// crypto reference-price endpoint (GET /api/crypto/crypto-price).
//
// It returns the open/close reference price that backs resolution of crypto
// Up/Down markets, for a given asset symbol and market window. It performs no
// signing and is safe in read-only contexts.
//
// When not to use this package:
//   - For order book reads — use pkg/orderbook.
//   - For user positions/trades or other Data API reads — use pkg/data.
//   - For market metadata, search, or tags — use pkg/gamma.
//
// This package targets the Polymarket web host (https://polymarket.com), which
// is distinct from the Gamma and Data API hosts.
//
// Stability: Client, NewClient, Config, DefaultConfig, CryptoPrice, and the
// CryptoPrice method are part of the polygolem public SDK.
package cryptoprice

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/polygolem/internal/transport"
)

const defaultBaseURL = "https://polymarket.com"

// CryptoPrice is the reference price for one crypto Up/Down market window.
type CryptoPrice struct {
	OpenPrice       float64
	ClosePrice      *float64
	TimestampMillis int64
	Completed       bool
	Incomplete      bool
	Cached          bool
}

// Config holds crypto-price client settings.
type Config struct {
	// BaseURL overrides the Polymarket web host. Empty uses production.
	BaseURL string
}

// DefaultConfig returns production defaults (the Polymarket web host).
func DefaultConfig() Config { return Config{BaseURL: defaultBaseURL} }

// Client is a read-only Polymarket crypto reference-price client.
// Construct via NewClient. Methods are safe for concurrent use.
type Client struct {
	inner *transport.Client
}

// NewClient creates a crypto-price client. A zero-valued config targets
// production. The client uses the package default HTTP transport with retry
// and rate limiting.
func NewClient(cfg Config) *Client {
	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Client{inner: transport.New(nil, transport.DefaultConfig(baseURL))}
}

// CryptoPrice fetches the reference price for one crypto market window.
//
// symbol is the asset (e.g. "BTC"); eventStart is the window start; variant is
// the market cadence (e.g. "fiveminute"); endDate is the optional window end.
// The returned open price is validated to be a positive, finite number; a
// present close price is validated the same way.
func (c *Client) CryptoPrice(ctx context.Context, symbol string, eventStart time.Time, variant string, endDate time.Time) (CryptoPrice, error) {
	if strings.TrimSpace(symbol) == "" || eventStart.IsZero() || strings.TrimSpace(variant) == "" {
		return CryptoPrice{}, fmt.Errorf("cryptoprice: symbol, eventStart, and variant are required")
	}

	query := url.Values{}
	query.Set("symbol", strings.ToUpper(strings.TrimSpace(symbol)))
	query.Set("eventStartTime", eventStart.UTC().Format(time.RFC3339))
	query.Set("variant", strings.TrimSpace(variant))
	if !endDate.IsZero() {
		query.Set("endDate", endDate.UTC().Format(time.RFC3339))
	}
	path := "/api/crypto/crypto-price?" + query.Encode()

	var out struct {
		OpenPrice       float64  `json:"openPrice"`
		ClosePrice      *float64 `json:"closePrice"`
		TimestampMillis int64    `json:"timestamp"`
		Completed       bool     `json:"completed"`
		Incomplete      bool     `json:"incomplete"`
		Cached          bool     `json:"cached"`
	}
	if err := c.inner.Get(ctx, path, &out); err != nil {
		return CryptoPrice{}, fmt.Errorf("cryptoprice: %w", err)
	}

	if out.OpenPrice <= 0 || math.IsNaN(out.OpenPrice) || math.IsInf(out.OpenPrice, 0) {
		return CryptoPrice{}, fmt.Errorf("cryptoprice: invalid open price %v", out.OpenPrice)
	}
	if out.ClosePrice != nil && (*out.ClosePrice <= 0 || math.IsNaN(*out.ClosePrice) || math.IsInf(*out.ClosePrice, 0)) {
		return CryptoPrice{}, fmt.Errorf("cryptoprice: invalid close price %v", *out.ClosePrice)
	}

	return CryptoPrice(out), nil
}
