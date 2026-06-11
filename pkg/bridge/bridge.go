// Package bridge is a client for the Polymarket Bridge API — supported
// assets, deposit addresses, deposit-status polling, and quotes.
//
// Use bridge to discover which assets can be bridged into Polymarket and
// to surface a deposit address for an EOA. The client is HTTP-only and
// performs no signing; it is safe to use in read-only mode.
//
// When not to use this package:
//   - For on-chain transfers — use a Polygon RPC client directly.
//   - For order placement — see the polygolem CLOB surface.
//
// Stability: the Client constructor, methods, and request/response types
// are part of the polygolem public SDK and follow semver. Pass a nil
// transport to NewClient to use the package default; advanced callers may
// supply their own.
package bridge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/TrebuchetDynamics/polygolem/internal/transport"
)

const defaultBridgeBaseURL = "https://bridge.polymarket.com"

// Client provides read-only access to the Polymarket Bridge API.
// Construct via NewClient. Methods are safe for concurrent use.
type Client struct {
	transport *transport.Client
}

// NewClient returns a Bridge API client.
// If baseURL is empty, the production Bridge URL is used.
// If tc is nil, a default transport with retry and rate limiting is
// constructed.
func NewClient(baseURL string, tc *transport.Client) *Client {
	if baseURL == "" {
		baseURL = defaultBridgeBaseURL
	}
	if tc == nil {
		tc = transport.New(nil, transport.DefaultConfig(baseURL))
	}
	return &Client{transport: tc}
}

// --- Types ---

// DepositAddress carries the per-chain deposit addresses returned by the
// Bridge for a given Polymarket account.
type DepositAddress struct {
	EVM string `json:"evm"`
	SVM string `json:"svm"`
	BTC string `json:"btc"`
}

// CreateDepositAddressResponse is the response shape for POST /deposit.
type CreateDepositAddressResponse struct {
	Address DepositAddress `json:"address"`
	Note    string         `json:"note"`
}

// TokenInfo describes one token (name, symbol, address, decimals) as
// reported by the Bridge.
type TokenInfo struct {
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	Address  string `json:"address"`
	Decimals int    `json:"decimals"`
}

func (t *TokenInfo) UnmarshalJSON(data []byte) error {
	var raw struct {
		Name     json.RawMessage `json:"name"`
		Symbol   json.RawMessage `json:"symbol"`
		Address  json.RawMessage `json:"address"`
		Decimals json.RawMessage `json:"decimals"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	t.Name = bridgeStringOrNumber(raw.Name)
	t.Symbol = bridgeStringOrNumber(raw.Symbol)
	t.Address = bridgeStringOrNumber(raw.Address)
	t.Decimals = int(bridgeInt64OrZero(raw.Decimals))
	return nil
}

// SupportedAsset is one entry in the Bridge's supported-assets list,
// pairing a chain with the token usable as deposit collateral.
type SupportedAsset struct {
	ChainID        string    `json:"chainId"`
	ChainName      string    `json:"chainName"`
	Token          TokenInfo `json:"token"`
	MinCheckoutUsd float64   `json:"minCheckoutUsd"`
}

func (s *SupportedAsset) UnmarshalJSON(data []byte) error {
	var raw struct {
		ChainID        json.RawMessage `json:"chainId"`
		ChainName      json.RawMessage `json:"chainName"`
		Token          TokenInfo       `json:"token"`
		MinCheckoutUsd json.RawMessage `json:"minCheckoutUsd"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	s.ChainID = bridgeStringOrNumber(raw.ChainID)
	s.ChainName = bridgeStringOrNumber(raw.ChainName)
	s.Token = raw.Token
	s.MinCheckoutUsd = bridgeFloat64OrZero(raw.MinCheckoutUsd)
	return nil
}

// SupportedAssetsResponse is the response shape for GET /supported-assets.
type SupportedAssetsResponse struct {
	SupportedAssets []SupportedAsset `json:"supportedAssets"`
}

// DepositTransaction describes one deposit attempt observed by the Bridge.
// Status is a Bridge-defined string; clients should treat unknown values
// as opaque.
type DepositTransaction struct {
	FromChainID        string `json:"fromChainId"`
	FromTokenAddress   string `json:"fromTokenAddress"`
	FromAmountBaseUnit string `json:"fromAmountBaseUnit"`
	ToChainID          string `json:"toChainId"`
	ToTokenAddress     string `json:"toTokenAddress"`
	TxHash             string `json:"txHash,omitempty"`
	CreatedTimeMs      int64  `json:"createdTimeMs,omitempty"`
	Status             string `json:"status"`
}

func (d *DepositTransaction) UnmarshalJSON(data []byte) error {
	var raw struct {
		FromChainID        json.RawMessage `json:"fromChainId"`
		FromTokenAddress   json.RawMessage `json:"fromTokenAddress"`
		FromAmountBaseUnit json.RawMessage `json:"fromAmountBaseUnit"`
		ToChainID          json.RawMessage `json:"toChainId"`
		ToTokenAddress     json.RawMessage `json:"toTokenAddress"`
		TxHash             json.RawMessage `json:"txHash"`
		CreatedTimeMs      json.RawMessage `json:"createdTimeMs"`
		Status             json.RawMessage `json:"status"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	d.FromChainID = bridgeStringOrNumber(raw.FromChainID)
	d.FromTokenAddress = bridgeStringOrNumber(raw.FromTokenAddress)
	d.FromAmountBaseUnit = bridgeStringOrNumber(raw.FromAmountBaseUnit)
	d.ToChainID = bridgeStringOrNumber(raw.ToChainID)
	d.ToTokenAddress = bridgeStringOrNumber(raw.ToTokenAddress)
	d.TxHash = bridgeStringOrNumber(raw.TxHash)
	d.CreatedTimeMs = bridgeInt64OrZero(raw.CreatedTimeMs)
	d.Status = bridgeStringOrNumber(raw.Status)
	return nil
}

func bridgeStringOrNumber(raw json.RawMessage) string {
	s := strings.TrimSpace(string(raw))
	if s == "" || s == "null" {
		return ""
	}
	if s[0] == '"' {
		var value string
		if err := json.Unmarshal(raw, &value); err == nil {
			return value
		}
	}
	return s
}

func bridgeInt64OrZero(raw json.RawMessage) int64 {
	value := bridgeStringOrNumber(raw)
	if value == "" {
		return 0
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0
	}
	return parsed
}

func bridgeFloat64OrZero(raw json.RawMessage) float64 {
	value := bridgeStringOrNumber(raw)
	if value == "" {
		return 0
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	return parsed
}

// DepositStatusResponse is the response shape for GET /status/{address}.
type DepositStatusResponse struct {
	Transactions []DepositTransaction `json:"transactions"`
}

// QuoteRequest is the input to GetQuote — the source token and amount,
// recipient, and target token on the Polymarket side.
type QuoteRequest struct {
	FromAmountBaseUnit string `json:"fromAmountBaseUnit"`
	FromChainID        string `json:"fromChainId"`
	FromTokenAddress   string `json:"fromTokenAddress"`
	RecipientAddress   string `json:"recipientAddress"`
	ToChainID          string `json:"toChainId"`
	ToTokenAddress     string `json:"toTokenAddress"`
}

// FeeBreakdown enumerates the fee components a Bridge quote includes.
// All percent fields are expressed as a fraction (0.01 = 1%).
type FeeBreakdown struct {
	AppFeeLabel     string  `json:"appFeeLabel"`
	AppFeePercent   float64 `json:"appFeePercent"`
	AppFeeUsd       float64 `json:"appFeeUsd"`
	FillCostPercent float64 `json:"fillCostPercent"`
	FillCostUsd     float64 `json:"fillCostUsd"`
	GasUsd          float64 `json:"gasUsd"`
	MaxSlippage     float64 `json:"maxSlippage"`
	MinReceived     float64 `json:"minReceived"`
	SwapImpact      float64 `json:"swapImpact"`
	SwapImpactUsd   float64 `json:"swapImpactUsd"`
	TotalImpact     float64 `json:"totalImpact"`
	TotalImpactUsd  float64 `json:"totalImpactUsd"`
}

func (f *FeeBreakdown) UnmarshalJSON(data []byte) error {
	var raw struct {
		AppFeeLabel     json.RawMessage `json:"appFeeLabel"`
		AppFeePercent   json.RawMessage `json:"appFeePercent"`
		AppFeeUsd       json.RawMessage `json:"appFeeUsd"`
		FillCostPercent json.RawMessage `json:"fillCostPercent"`
		FillCostUsd     json.RawMessage `json:"fillCostUsd"`
		GasUsd          json.RawMessage `json:"gasUsd"`
		MaxSlippage     json.RawMessage `json:"maxSlippage"`
		MinReceived     json.RawMessage `json:"minReceived"`
		SwapImpact      json.RawMessage `json:"swapImpact"`
		SwapImpactUsd   json.RawMessage `json:"swapImpactUsd"`
		TotalImpact     json.RawMessage `json:"totalImpact"`
		TotalImpactUsd  json.RawMessage `json:"totalImpactUsd"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	f.AppFeeLabel = bridgeStringOrNumber(raw.AppFeeLabel)
	f.AppFeePercent = bridgeFloat64OrZero(raw.AppFeePercent)
	f.AppFeeUsd = bridgeFloat64OrZero(raw.AppFeeUsd)
	f.FillCostPercent = bridgeFloat64OrZero(raw.FillCostPercent)
	f.FillCostUsd = bridgeFloat64OrZero(raw.FillCostUsd)
	f.GasUsd = bridgeFloat64OrZero(raw.GasUsd)
	f.MaxSlippage = bridgeFloat64OrZero(raw.MaxSlippage)
	f.MinReceived = bridgeFloat64OrZero(raw.MinReceived)
	f.SwapImpact = bridgeFloat64OrZero(raw.SwapImpact)
	f.SwapImpactUsd = bridgeFloat64OrZero(raw.SwapImpactUsd)
	f.TotalImpact = bridgeFloat64OrZero(raw.TotalImpact)
	f.TotalImpactUsd = bridgeFloat64OrZero(raw.TotalImpactUsd)
	return nil
}

// QuoteResponse is the response shape for POST /quote — estimated input
// and output USD, an estimated time, the fee breakdown, and a quote ID
// that the caller must echo when accepting the quote.
type QuoteResponse struct {
	EstCheckoutTimeMs  int64        `json:"estCheckoutTimeMs"`
	EstFeeBreakdown    FeeBreakdown `json:"estFeeBreakdown"`
	EstInputUsd        float64      `json:"estInputUsd"`
	EstOutputUsd       float64      `json:"estOutputUsd"`
	EstToTokenBaseUnit string       `json:"estToTokenBaseUnit"`
	QuoteID            string       `json:"quoteId"`
}

// ErrWithdrawSubmitUnsupported is returned by Withdraw until the upstream
// withdrawal/offramp endpoint shape and custody constraints have been captured.
// Use BuildWithdrawDryRun to validate and preview the payload operators intend
// to submit.
var ErrWithdrawSubmitUnsupported = errors.New("bridge withdrawal submit is not supported; use BuildWithdrawDryRun until endpoint shape and custody risk are verified")

// WithdrawRequest describes a proposed Bridge withdrawal/offramp. It is kept
// explicit rather than reusing QuoteRequest because withdrawals have different
// custody and recipient-risk semantics than deposits.
type WithdrawRequest struct {
	FromChainID        string `json:"fromChainId"`
	FromTokenAddress   string `json:"fromTokenAddress"`
	FromAmountBaseUnit string `json:"fromAmountBaseUnit"`
	ToChainID          string `json:"toChainId"`
	ToTokenAddress     string `json:"toTokenAddress"`
	RecipientAddress   string `json:"recipientAddress"`
	QuoteID            string `json:"quoteId,omitempty"`
}

// WithdrawDryRun is the safe preview artifact for a proposed withdrawal. It is
// intentionally not submitted to the Bridge API.
type WithdrawDryRun struct {
	Request        WithdrawRequest `json:"request"`
	ReadyToSubmit  bool            `json:"readyToSubmit"`
	Unsupported    bool            `json:"unsupported"`
	SafetyWarnings []string        `json:"safetyWarnings"`
}

// WithdrawResponse is reserved for the future live submit response shape.
type WithdrawResponse struct {
	QuoteID string `json:"quoteId,omitempty"`
	TxHash  string `json:"txHash,omitempty"`
	Status  string `json:"status,omitempty"`
}

func (q *QuoteResponse) UnmarshalJSON(data []byte) error {
	var raw struct {
		EstCheckoutTimeMs  json.RawMessage `json:"estCheckoutTimeMs"`
		EstFeeBreakdown    FeeBreakdown    `json:"estFeeBreakdown"`
		EstInputUsd        json.RawMessage `json:"estInputUsd"`
		EstOutputUsd       json.RawMessage `json:"estOutputUsd"`
		EstToTokenBaseUnit json.RawMessage `json:"estToTokenBaseUnit"`
		QuoteID            json.RawMessage `json:"quoteId"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	q.EstCheckoutTimeMs = bridgeInt64OrZero(raw.EstCheckoutTimeMs)
	q.EstFeeBreakdown = raw.EstFeeBreakdown
	q.EstInputUsd = bridgeFloat64OrZero(raw.EstInputUsd)
	q.EstOutputUsd = bridgeFloat64OrZero(raw.EstOutputUsd)
	q.EstToTokenBaseUnit = bridgeStringOrNumber(raw.EstToTokenBaseUnit)
	q.QuoteID = bridgeStringOrNumber(raw.QuoteID)
	return nil
}

// --- Methods ---

// BuildWithdrawDryRun validates and previews a withdrawal/offramp request
// without submitting it. This is the only supported withdrawal path until the
// live endpoint contract is captured in fixtures and safety docs.
func BuildWithdrawDryRun(req WithdrawRequest) (*WithdrawDryRun, error) {
	if err := req.validate(); err != nil {
		return nil, err
	}
	return &WithdrawDryRun{
		Request:       req,
		ReadyToSubmit: false,
		Unsupported:   true,
		SafetyWarnings: []string{
			"bridge withdrawal/offramp live submission is intentionally disabled",
			"verify recipient address, token, chain, quote, and custody route before any future submit path",
			"use explicit operator confirmation before moving funds",
		},
	}, nil
}

// Withdraw is a guarded placeholder for future live withdrawal submission. It
// always returns ErrWithdrawSubmitUnsupported after validating the request so
// callers can wire safe UX without accidentally moving funds.
func (c *Client) Withdraw(ctx context.Context, req WithdrawRequest) (*WithdrawResponse, error) {
	if _, err := BuildWithdrawDryRun(req); err != nil {
		return nil, err
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	return nil, ErrWithdrawSubmitUnsupported
}

func (req WithdrawRequest) validate() error {
	checks := map[string]string{
		"fromChainId":        req.FromChainID,
		"fromTokenAddress":   req.FromTokenAddress,
		"fromAmountBaseUnit": req.FromAmountBaseUnit,
		"toChainId":          req.ToChainID,
		"toTokenAddress":     req.ToTokenAddress,
		"recipientAddress":   req.RecipientAddress,
	}
	for field, value := range checks {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", field)
		}
	}
	amount, ok := new(big.Int).SetString(strings.TrimSpace(req.FromAmountBaseUnit), 10)
	if !ok || amount.Sign() <= 0 {
		return fmt.Errorf("fromAmountBaseUnit must be a positive base-10 integer")
	}
	return nil
}

// CreateDepositAddress requests the Bridge mint a deposit address for the
// given Polymarket-side address. The Bridge returns a per-chain address
// set; only one of EVM/SVM/BTC is typically populated per request.
func (c *Client) CreateDepositAddress(ctx context.Context, address string) (*CreateDepositAddressResponse, error) {
	var result CreateDepositAddressResponse
	if err := c.transport.Post(ctx, "/deposit", map[string]string{"address": address}, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetSupportedAssets returns the assets the Bridge currently accepts as
// deposit collateral.
func (c *Client) GetSupportedAssets(ctx context.Context) (*SupportedAssetsResponse, error) {
	var result SupportedAssetsResponse
	if err := c.transport.Get(ctx, "/supported-assets", &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetDepositStatus polls the Bridge for outstanding and recent deposit
// transactions targeting depositAddress.
func (c *Client) GetDepositStatus(ctx context.Context, depositAddress string) (*DepositStatusResponse, error) {
	var result DepositStatusResponse
	if err := c.transport.Get(ctx, "/status/"+depositAddress, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetQuote asks the Bridge to price a deposit move described by req.
// The returned QuoteID is the handle the caller will echo in a follow-up
// accept call.
func (c *Client) GetQuote(ctx context.Context, req QuoteRequest) (*QuoteResponse, error) {
	var result QuoteResponse
	if err := c.transport.Post(ctx, "/quote", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
