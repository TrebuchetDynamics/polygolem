// Package polyerrors normalizes upstream Polymarket errors into stable kinds
// for operators, agents, and UI adapters.
package polyerrors

import (
	"fmt"
	"strings"
)

type Kind string

const (
	Unknown             Kind = "unknown"
	RateLimited         Kind = "rate_limited"
	Geoblocked          Kind = "geoblocked"
	AuthRejected        Kind = "auth_rejected"
	TickSizeMismatch    Kind = "tick_size_mismatch"
	MarketClosed        Kind = "market_closed"
	InsufficientFunds   Kind = "insufficient_funds"
	UpstreamUnavailable Kind = "upstream_unavailable"
)

func Kinds() []Kind {
	return []Kind{
		Unknown,
		RateLimited,
		Geoblocked,
		AuthRejected,
		TickSizeMismatch,
		MarketClosed,
		InsufficientFunds,
		UpstreamUnavailable,
	}
}

type Input struct {
	Source     string
	HTTPStatus int
	Message    string
}

type Error struct {
	Kind       Kind   `json:"kind"`
	Source     string `json:"source,omitempty"`
	HTTPStatus int    `json:"http_status,omitempty"`
	Message    string `json:"message,omitempty"`
}

func (e Error) Error() string {
	if e.HTTPStatus != 0 && e.Message != "" {
		return fmt.Sprintf("polymarket %s: HTTP %d: %s", e.Kind, e.HTTPStatus, e.Message)
	}
	if e.HTTPStatus != 0 {
		return fmt.Sprintf("polymarket %s: HTTP %d", e.Kind, e.HTTPStatus)
	}
	if e.Message != "" {
		return fmt.Sprintf("polymarket %s: %s", e.Kind, e.Message)
	}
	return fmt.Sprintf("polymarket %s", e.Kind)
}

func Normalize(in Input) Error {
	message := safeMessage(in.Message)
	return Error{
		Kind:       classify(in.HTTPStatus, strings.ToLower(in.Message)),
		Source:     in.Source,
		HTTPStatus: in.HTTPStatus,
		Message:    message,
	}
}

func classify(status int, lower string) Kind {
	switch status {
	case 401, 407:
		return AuthRejected
	case 403:
		return Geoblocked
	case 429:
		return RateLimited
	}
	if status >= 500 && status <= 599 {
		return UpstreamUnavailable
	}
	return classifyMessage(lower)
}

func classifyMessage(lower string) Kind {
	switch {
	case containsAny(lower, "rate limit", "too many requests"):
		return RateLimited
	case containsAny(lower, "geoblock", "geo block", "restricted jurisdiction"):
		return Geoblocked
	case containsAny(lower, "unauthorized", "forbidden", "api key", "signature", "expired"):
		return AuthRejected
	case containsAny(lower, "tick size", "min tick", "price increment"):
		return TickSizeMismatch
	case containsAny(lower, "market is closed", "market closed", "closed market", "resolved"):
		return MarketClosed
	case containsAny(lower, "insufficient funds", "insufficient balance", "not enough balance", "not enough funds"):
		return InsufficientFunds
	default:
		return Unknown
	}
}

func safeMessage(message string) string {
	lower := strings.ToLower(message)
	if containsAny(lower,
		"poly_api_key",
		"poly_signature",
		"poly_passphrase",
		"private_key",
		"signer_private_key",
		"bearer ",
	) {
		return "[redacted]"
	}
	return message
}

func containsAny(value string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}
