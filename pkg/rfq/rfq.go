// Package rfq defines typed request-for-quote (RFQ) models for future
// Polymarket RFQ integration.
//
// RFQ live submission is intentionally not implemented yet. The package gives
// SDK consumers stable DTOs and validation so fixtures and UI can be built
// before authenticated upstream behavior is captured from reference clients.
package rfq

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrSubmitUnsupported is returned by Client methods that would mutate live RFQ
// state. Capture upstream payloads and safety docs before replacing it.
var ErrSubmitUnsupported = errors.New("RFQ live submission is not supported; use ValidateRequest and fixtures until upstream behavior is captured")

const (
	SideBuy  = "BUY"
	SideSell = "SELL"
)

// Request is a typed RFQ intent. Amount is an opaque decimal string so callers
// do not lose precision before protocol-specific rounding rules are known.
type Request struct {
	MarketID   string    `json:"market_id"`
	TokenID    string    `json:"token_id"`
	Side       string    `json:"side"`
	Amount     string    `json:"amount"`
	Expiration time.Time `json:"expiration,omitempty"`
	Maker      string    `json:"maker,omitempty"`
	Metadata   Metadata  `json:"metadata,omitempty"`
}

// Metadata carries optional attribution/debug context without affecting the
// canonical RFQ fields.
type Metadata struct {
	ClientOrderID string `json:"client_order_id,omitempty"`
	BuilderCode   string `json:"builder_code,omitempty"`
	Notes         string `json:"notes,omitempty"`
}

// Quote is the future typed response shape for a received RFQ quote.
type Quote struct {
	ID           string    `json:"id"`
	RequestID    string    `json:"request_id"`
	Price        string    `json:"price"`
	Size         string    `json:"size"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
	Counterparty string    `json:"counterparty,omitempty"`
}

// Response is reserved for future live submission responses.
type Response struct {
	RequestID string  `json:"request_id"`
	Status    string  `json:"status"`
	Quotes    []Quote `json:"quotes,omitempty"`
}

// Client is a placeholder RFQ client. It validates inputs but refuses live
// submission until endpoint shape, auth requirements, and safety gates are
// captured in fixtures.
type Client struct{}

func NewClient() *Client { return &Client{} }

func isPositiveDecimalString(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "+") || strings.HasPrefix(value, "-") {
		return false
	}
	seenDigit := false
	seenDot := false
	seenNonZero := false
	for _, r := range value {
		switch {
		case r == '.':
			if seenDot {
				return false
			}
			seenDot = true
		case r >= '0' && r <= '9':
			seenDigit = true
			if r != '0' {
				seenNonZero = true
			}
		default:
			return false
		}
	}
	return seenDigit && seenNonZero
}

// ValidateRequest checks a request's stable fields without contacting upstream.
func ValidateRequest(req Request) error {
	if strings.TrimSpace(req.MarketID) == "" && strings.TrimSpace(req.TokenID) == "" {
		return fmt.Errorf("market_id or token_id is required")
	}
	if !isPositiveDecimalString(req.Amount) {
		return fmt.Errorf("amount must be a positive decimal string")
	}
	side := strings.ToUpper(strings.TrimSpace(req.Side))
	if side != SideBuy && side != SideSell {
		return fmt.Errorf("side must be BUY or SELL")
	}
	if !req.Expiration.IsZero() && !req.Expiration.After(time.Now()) {
		return fmt.Errorf("expiration must be in the future")
	}
	return nil
}

// Submit validates req and then returns ErrSubmitUnsupported. It exists so
// callers can wire RFQ UX safely before any live mutation path exists.
func (c *Client) Submit(req Request) (*Response, error) {
	if err := ValidateRequest(req); err != nil {
		return nil, err
	}
	return nil, ErrSubmitUnsupported
}
