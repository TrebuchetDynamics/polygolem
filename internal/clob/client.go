package clob

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/polygolem/internal/auth"
	"github.com/TrebuchetDynamics/polygolem/internal/gamma"
	"github.com/TrebuchetDynamics/polygolem/internal/jsonx"
	"github.com/TrebuchetDynamics/polygolem/internal/polytypes"
	"github.com/TrebuchetDynamics/polygolem/internal/transport"
)

// Client provides read-only access to the Polymarket CLOB API.
// Base URL: https://clob.polymarket.com
type Client struct {
	transport     *transport.Client
	builderCode   string
	l2Credentials *auth.APIKey
	gate          TradeGate
}

const polygonChainID = 137

// BalanceAllowanceParams are query parameters for CLOB balance-allowance.
// Sigtype is hardcoded to 3 (POLY_1271, deposit wallet) — the only type
// Polymarket V2 supports.
type BalanceAllowanceParams struct {
	Asset     string
	AssetType string
	TokenID   string
}

// BalanceAllowanceResponse is the authenticated CLOB collateral/token balance.
type BalanceAllowanceResponse struct {
	Balance    string            `json:"balance"`
	Allowances map[string]string `json:"allowances,omitempty"`
	Allowance  string            `json:"allowance,omitempty"`
}

func (r *BalanceAllowanceResponse) UnmarshalJSON(data []byte) error {
	var raw struct {
		Balance    json.RawMessage            `json:"balance"`
		Allowances map[string]json.RawMessage `json:"allowances"`
		Allowance  json.RawMessage            `json:"allowance"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	r.Balance = jsonStringOrNumber(raw.Balance)
	r.Allowance = jsonStringOrNumber(raw.Allowance)
	if raw.Allowances != nil {
		r.Allowances = make(map[string]string, len(raw.Allowances))
		for key, value := range raw.Allowances {
			r.Allowances[key] = jsonStringOrNumber(value)
		}
	}
	return nil
}

// NewClient creates a CLOB API client.
func NewClient(baseURL string, tc *transport.Client, opts ...Option) *Client {
	if tc == nil {
		tc = transport.New(nil, transport.DefaultConfig(baseURL))
	}
	c := &Client{transport: tc, builderCode: bytes32Zero}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// SetBuilderCode configures the V2 order builder attribution bytes32.
func (c *Client) SetBuilderCode(builderCode string) {
	c.builderCode = builderCode
}

// SetL2Credentials configures pre-provisioned CLOB L2 credentials for
// authenticated deposit-wallet requests. When set, authenticated CLOB calls
// use these HMAC credentials instead of deriving a key through L1 auth.
func (c *Client) SetL2Credentials(key auth.APIKey) {
	c.l2Credentials = &key
}

// Health checks the CLOB API is reachable.
func (c *Client) Health(ctx context.Context) error {
	return c.transport.Get(ctx, "/", nil)
}

// ServerTime returns the current server time.
func (c *Client) ServerTime(ctx context.Context) (*polytypes.ServerTime, error) {
	var result polytypes.ServerTime
	if err := c.transport.Get(ctx, "/time", &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateOrDeriveAPIKey creates new CLOB L2 credentials with L1 auth, falling
// back to deterministic derivation when a key already exists.
func (c *Client) CreateOrDeriveAPIKey(ctx context.Context, privateKey string) (auth.APIKey, error) {
	key, err := c.createAPIKey(ctx, privateKey)
	if err == nil {
		return key, nil
	}
	return c.DeriveAPIKey(ctx, privateKey)
}

// CreateOrDeriveAPIKeyForAddress creates CLOB L2 credentials, falling back
// to derivation when a key already exists. The ownerAddress parameter is
// retained for source-compat but ignored (see [DeriveAPIKeyForAddress]).
func (c *Client) CreateOrDeriveAPIKeyForAddress(ctx context.Context, privateKey, ownerAddress string) (auth.APIKey, error) {
	_ = ownerAddress
	return c.CreateOrDeriveAPIKey(ctx, privateKey)
}

// DeriveAPIKey derives existing CLOB L2 credentials with L1 auth.
func (c *Client) DeriveAPIKey(ctx context.Context, privateKey string) (auth.APIKey, error) {
	headers, err := auth.BuildL1HeadersFromPrivateKey(privateKey, polygonChainID, time.Now().Unix(), 0)
	if err != nil {
		return auth.APIKey{}, err
	}
	var raw apiKeyResponse
	if err := c.transport.GetWithHeaders(ctx, "/auth/derive-api-key", headers, &raw); err != nil {
		return auth.APIKey{}, err
	}
	return raw.apiKey(), nil
}

// DeriveAPIKeyForAddress derives existing CLOB L2 credentials. The 2026-05-08
// web-UI capture (docs/history/BLOCKERS.md) showed that even for sigtype-3 deposit-wallet
// users, the CLOB API key is **EOA-bound**: POLY_ADDRESS=EOA, raw 65-byte
// ECDSA POLY_SIGNATURE — no ERC-7739 wrap. The deposit-wallet identity rides
// on the order's `signatureType=3` field at the EIP-712 layer plus the
// `signature_type=3` query param on read endpoints; HTTP-layer auth is EOA.
//
// The ownerAddress parameter is retained for source-compat but is ignored:
// L1 always signs with the EOA derived from privateKey.
func (c *Client) DeriveAPIKeyForAddress(ctx context.Context, privateKey, ownerAddress string) (auth.APIKey, error) {
	_ = ownerAddress
	return c.DeriveAPIKey(ctx, privateKey)
}

func (c *Client) createAPIKey(ctx context.Context, privateKey string) (auth.APIKey, error) {
	headers, err := auth.BuildL1HeadersFromPrivateKey(privateKey, polygonChainID, time.Now().Unix(), 0)
	if err != nil {
		return auth.APIKey{}, err
	}
	var raw apiKeyResponse
	if err := c.transport.PostWithHeaders(ctx, "/auth/api-key", nil, headers, &raw); err != nil {
		return auth.APIKey{}, err
	}
	return raw.apiKey(), nil
}

// CreateAPIKeyForAddress creates CLOB L2 credentials. Since the 2026-05-08
// capture proved the web UI uses EOA-bound CLOB keys for both sigtype-1 and
// sigtype-3 paths, this is now a thin wrapper over [createAPIKey] — the
// ownerAddress parameter is retained for source-compat but ignored. See
// DeriveAPIKeyForAddress for the rationale.
func (c *Client) CreateAPIKeyForAddress(ctx context.Context, privateKey, ownerAddress string) (auth.APIKey, error) {
	_ = ownerAddress
	return c.createAPIKey(ctx, privateKey)
}

type apiKeyResponse struct {
	APIKey         string `json:"apiKey"`
	APIKeySnake    string `json:"api_key"`
	Key            string `json:"key"` // builder-fee key endpoint uses bare "key"
	Secret         string `json:"secret"`
	Passphrase     string `json:"passphrase"`
	PassphraseAlt  string `json:"passPhrase"`
	PassphraseAlt2 string `json:"pass_phrase"`
}

func (r apiKeyResponse) apiKey() auth.APIKey {
	passphrase := firstNonEmpty(r.Passphrase, r.PassphraseAlt, r.PassphraseAlt2)
	return auth.APIKey{
		Key:        firstNonEmpty(r.APIKey, r.APIKeySnake, r.Key),
		Secret:     r.Secret,
		Passphrase: passphrase,
	}
}

// CreateBuilderFeeKey mints a builder fee key via L2 auth.
// The returned creds are used for V2 order builder field attribution.
// Requires existing L2 credentials (from builder auto or CreateOrDeriveAPIKey).
// This endpoint is fully headless — no cookie, no browser, no relayer.
//
// See docs/HEADLESS-BUILDER-KEYS-INVESTIGATION.md for the V2 split-cred
// model that distinguishes this CLOB-side builder fee key from the
// browser-gated Relayer API Key.
func (c *Client) CreateBuilderFeeKey(ctx context.Context, privateKey string) (auth.APIKey, error) {
	key, err := c.DeriveAPIKey(ctx, privateKey)
	if err != nil {
		return auth.APIKey{}, fmt.Errorf("derive api key: %w", err)
	}
	body := map[string]string{}
	bodyBytes, _ := json.Marshal(body)
	bodyStr := string(bodyBytes)
	headers, err := c.l2Headers(privateKey, &key, http.MethodPost, "/auth/builder-api-key", &bodyStr)
	if err != nil {
		return auth.APIKey{}, err
	}
	var raw apiKeyResponse
	if err := c.transport.PostWithHeaders(ctx, "/auth/builder-api-key", body, headers, &raw); err != nil {
		return auth.APIKey{}, fmt.Errorf("create builder fee key: %w", err)
	}
	return raw.apiKey(), nil
}

// BuilderFeeKeyRecord represents one row from `GET /auth/builder-api-keys`.
// Fields are best-effort — the upstream shape is loosely typed.
type BuilderFeeKeyRecord struct {
	Key        string `json:"key"`
	Secret     string `json:"secret,omitempty"`
	Passphrase string `json:"passphrase,omitempty"`
	CreatedAt  string `json:"created_at,omitempty"`
	UpdatedAt  string `json:"updated_at,omitempty"`
}

func (r *BuilderFeeKeyRecord) UnmarshalJSON(data []byte) error {
	type alias BuilderFeeKeyRecord
	aux := struct {
		*alias
		CreatedAtCamel string `json:"createdAt"`
		UpdatedAtCamel string `json:"updatedAt"`
	}{alias: (*alias)(r)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if r.CreatedAt == "" {
		r.CreatedAt = aux.CreatedAtCamel
	}
	if r.UpdatedAt == "" {
		r.UpdatedAt = aux.UpdatedAtCamel
	}
	return nil
}

// ListBuilderFeeKeys returns every builder fee key minted for the
// authenticated wallet via `GET /auth/builder-api-keys`.
func (c *Client) ListBuilderFeeKeys(ctx context.Context, privateKey string) ([]BuilderFeeKeyRecord, error) {
	key, err := c.DeriveAPIKey(ctx, privateKey)
	if err != nil {
		return nil, fmt.Errorf("derive api key: %w", err)
	}
	headers, err := c.l2Headers(privateKey, &key, http.MethodGet, "/auth/builder-api-keys", nil)
	if err != nil {
		return nil, err
	}
	var result []BuilderFeeKeyRecord
	if err := c.transport.GetWithHeaders(ctx, "/auth/builder-api-keys", headers, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// RevokeBuilderFeeKey deletes a builder fee key via
// `DELETE /auth/builder-api-key/{key}`.
func (c *Client) RevokeBuilderFeeKey(ctx context.Context, privateKey, builderKey string) error {
	if strings.TrimSpace(builderKey) == "" {
		return fmt.Errorf("builderKey is required")
	}
	key, err := c.DeriveAPIKey(ctx, privateKey)
	if err != nil {
		return fmt.Errorf("derive api key: %w", err)
	}
	path := "/auth/builder-api-key/" + url.PathEscape(builderKey)
	headers, err := c.l2Headers(privateKey, &key, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	return c.transport.DeleteWithHeaders(ctx, path, nil, headers, nil)
}

// BalanceAllowance returns collateral or conditional token balance and allowances.
func (c *Client) BalanceAllowance(ctx context.Context, privateKey string, params BalanceAllowanceParams) (*BalanceAllowanceResponse, error) {
	key, polyAddress, err := c.depositWalletAPIKey(ctx, privateKey)
	if err != nil {
		return nil, fmt.Errorf("derive deposit-wallet api key: %w", err)
	}
	path := balanceAllowancePath("/balance-allowance", params)
	headers, err := c.l2HeadersForAddress(&key, http.MethodGet, "/balance-allowance", nil, polyAddress)
	if err != nil {
		return nil, err
	}
	var result BalanceAllowanceResponse
	if err := c.transport.GetWithHeaders(ctx, path, headers, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateBalanceAllowance refreshes the CLOB balance/allowance cache.
func (c *Client) UpdateBalanceAllowance(ctx context.Context, privateKey string, params BalanceAllowanceParams) (*BalanceAllowanceResponse, error) {
	key, polyAddress, err := c.depositWalletAPIKey(ctx, privateKey)
	if err != nil {
		return nil, fmt.Errorf("derive deposit-wallet api key: %w", err)
	}
	path := balanceAllowancePath("/balance-allowance/update", params)
	headers, err := c.l2HeadersForAddress(&key, http.MethodGet, "/balance-allowance/update", nil, polyAddress)
	if err != nil {
		return nil, err
	}
	// The endpoint is fire-and-forget: HTTP 200 with an empty body signals
	// that the refresh has been queued. Read raw bytes so an empty body is
	// not treated as a JSON decode error.
	raw, err := c.transport.GetRawWithHeaders(ctx, path, headers)
	if err != nil {
		return nil, err
	}
	var result BalanceAllowanceResponse
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &result); err != nil {
			return nil, fmt.Errorf("decode update-balance: %w", err)
		}
	}
	return &result, nil
}

func (c *Client) l2Headers(privateKey string, key *auth.APIKey, method, path string, body *string) (map[string]string, error) {
	signer, err := auth.NewPrivateKeySigner(privateKey, polygonChainID)
	if err != nil {
		return nil, err
	}
	return c.l2HeadersForAddress(key, method, path, body, signer.Address())
}

func (c *Client) l2HeadersForAddress(key *auth.APIKey, method, path string, body *string, polyAddress string) (map[string]string, error) {
	headers, err := auth.BuildL2Headers(key, time.Now().Unix(), method, path, body)
	if err != nil {
		return nil, err
	}
	headers["POLY_ADDRESS"] = polyAddress
	return headers, nil
}

func signerAndDepositWallet(privateKey string) (*auth.PrivateKeySigner, string, error) {
	signer, err := auth.NewPrivateKeySigner(privateKey, polygonChainID)
	if err != nil {
		return nil, "", err
	}
	depositWallet, err := auth.MakerAddressForSignatureType(signer.Address(), polygonChainID, signatureTypePoly1271)
	if err != nil {
		return nil, "", err
	}
	return signer, depositWallet, nil
}

// depositWalletAPIKey returns the EOA-bound CLOB credentials and the EOA
// address to use as POLY_ADDRESS in HMAC headers. Per the 2026-05-08 web-UI
// capture, V2 CLOB authentication is EOA-bound at the HTTP layer for both
// sigtype-1 and sigtype-3 orders; the deposit-wallet identity travels in
// the order's signatureType=3 field at the EIP-712 layer.
func (c *Client) depositWalletAPIKey(ctx context.Context, privateKey string) (auth.APIKey, string, error) {
	signer, _, err := signerAndDepositWallet(privateKey)
	if err != nil {
		return auth.APIKey{}, "", err
	}
	key, err := c.depositWalletAPIKeyForAddress(ctx, privateKey, "")
	if err != nil {
		return auth.APIKey{}, "", err
	}
	return key, signer.Address(), nil
}

// depositWalletAPIKeyForAddress retains the depositWallet parameter for
// source-compat with callers in orders.go, but only the EOA-bound key is
// returned (see [DeriveAPIKeyForAddress]). Tries derive first, falls back
// to create — matches the order the web UI uses (captured 2026-05-08).
func (c *Client) depositWalletAPIKeyForAddress(ctx context.Context, privateKey, depositWallet string) (auth.APIKey, error) {
	_ = depositWallet
	if key, ok := c.configuredL2Credentials(); ok {
		return key, nil
	}
	key, err := c.DeriveAPIKey(ctx, privateKey)
	if err == nil {
		return key, nil
	}
	return c.createAPIKey(ctx, privateKey)
}

func (c *Client) configuredL2Credentials() (auth.APIKey, bool) {
	if c.l2Credentials == nil {
		return auth.APIKey{}, false
	}
	key := *c.l2Credentials
	if strings.TrimSpace(key.Key) == "" &&
		strings.TrimSpace(key.Secret) == "" &&
		strings.TrimSpace(key.Passphrase) == "" {
		return auth.APIKey{}, false
	}
	return key, true
}

func balanceAllowancePath(base string, params BalanceAllowanceParams) string {
	q := url.Values{}
	if params.Asset != "" {
		q.Set("asset", params.Asset)
	}
	if params.AssetType != "" {
		q.Set("asset_type", strings.ToUpper(params.AssetType))
	}
	if params.TokenID != "" {
		q.Set("token_id", params.TokenID)
	}
	q.Set("signature_type", strconv.Itoa(signatureTypePoly1271))
	return base + "?" + q.Encode()
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		v := strings.TrimSpace(value)
		if v != "" && v != "<nil>" {
			return value
		}
	}
	return ""
}

func parseTokenValueMap(raw json.RawMessage, params []polytypes.BookParams, valueKeys ...string) (map[string]string, error) {
	var rows map[string]json.RawMessage
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, err
	}
	sidesByToken := make(map[string]string, len(params))
	for _, param := range params {
		if param.TokenID != "" && param.Side != "" {
			sidesByToken[param.TokenID] = strings.ToUpper(param.Side)
		}
	}
	out := make(map[string]string, len(rows))
	for tokenID, payload := range rows {
		if value, ok := decodeNumericString(payload); ok {
			out[tokenID] = value
			continue
		}
		var nested map[string]json.RawMessage
		if err := json.Unmarshal(payload, &nested); err != nil {
			return nil, fmt.Errorf("decode token %s value: %w", tokenID, err)
		}
		if side := sidesByToken[tokenID]; side != "" {
			if value, ok := decodeNumericString(nested[side]); ok {
				out[tokenID] = value
				continue
			}
			if value, ok := decodeNumericString(nested[strings.ToLower(side)]); ok {
				out[tokenID] = value
				continue
			}
		}
		for _, key := range valueKeys {
			if value, ok := decodeNumericString(nested[key]); ok {
				out[tokenID] = value
				break
			}
		}
		if out[tokenID] != "" {
			continue
		}
		for _, key := range []string{"BUY", "SELL", "buy", "sell"} {
			if value, ok := decodeNumericString(nested[key]); ok {
				out[tokenID] = value
				break
			}
		}
	}
	return out, nil
}

func parseLastTradePrices(raw json.RawMessage, params []polytypes.BookParams) (map[string]string, error) {
	var rows []struct {
		TokenID string                  `json:"token_id"`
		Price   polytypes.NumericString `json:"price"`
	}
	if err := json.Unmarshal(raw, &rows); err == nil {
		out := make(map[string]string, len(rows))
		for _, row := range rows {
			if row.TokenID != "" {
				out[row.TokenID] = string(row.Price)
			}
		}
		return out, nil
	}
	return parseTokenValueMap(raw, params, "price")
}

func decodeNumericString(raw json.RawMessage) (string, bool) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" || strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		return "", false
	}
	value := jsonx.StringOrNumber(raw)
	return value, value != ""
}

// Markets lists CLOB markets with cursor pagination.
func (c *Client) Markets(ctx context.Context, nextCursor string) (*polytypes.CLOBPaginatedMarkets, error) {
	path := "/markets"
	if nextCursor != "" {
		path += "?next_cursor=" + url.QueryEscape(nextCursor)
	}
	var result polytypes.CLOBPaginatedMarkets
	if err := c.transport.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Market returns a single CLOB market by condition ID.
func (c *Client) Market(ctx context.Context, conditionID string) (*polytypes.CLOBMarket, error) {
	var result polytypes.CLOBMarket
	if err := c.transport.Get(ctx, "/markets/"+url.PathEscape(conditionID), &result); err != nil {
		return nil, err
	}
	if result.ConditionID == "" {
		result.ConditionID = conditionID
	}
	return &result, nil
}

// winningTokenID scans the market tokens and returns the token ID marked as
// winner, or empty string if no token has Winner=true or there are multiple.
func winningTokenID(market *polytypes.CLOBMarket) string {
	if market == nil {
		return ""
	}
	var winner string
	winnerCount := 0
	for _, token := range market.Tokens {
		if token.Winner {
			winner = token.TokenID
			winnerCount++
		}
	}
	if winnerCount == 1 {
		return winner
	}
	return ""
}

// MarketOutcome tries to resolve a market's outcome. It first queries the
// CLOB API by condition ID. If CLOB returns the market as closed with a
// winning token, the outcome is resolved. If CLOB cannot find the market
// (e.g. the market is archived or does not appear on the CLOB), it falls
// back to the Polymarket Gamma API to check whether the market has resolved.
//
// The gammaBaseURL parameter is optional; when empty, no Gamma fallback is
// attempted and a CLOB-not-found result returns an error as before.
func (c *Client) MarketOutcome(ctx context.Context, conditionID string, gammaBaseURL string) (*polytypes.CLOBMarketOutcome, error) {
	conditionID = strings.TrimSpace(conditionID)
	if conditionID == "" {
		return nil, fmt.Errorf("clob: conditionID is required")
	}

	// First try CLOB.
	market, err := c.Market(ctx, conditionID)
	if err == nil && market != nil {
		winner := winningTokenID(market)
		if market.Closed && winner != "" {
			return &polytypes.CLOBMarketOutcome{
				Status:         polytypes.CLOBOutcomeResolved,
				ConditionID:    conditionID,
				WinningTokenID: winner,
				Closed:         true,
				Source:         "clob:/markets/" + conditionID,
			}, nil
		}
		// CLOB knows the market but it is not yet closed or has no winner.
		return &polytypes.CLOBMarketOutcome{
			Status:      polytypes.CLOBOutcomeUnresolved,
			ConditionID: conditionID,
			Closed:      market.Closed,
			Source:      "clob:/markets/" + conditionID + ":not_closed_or_no_winner",
		}, nil
	}

	// Fall back to Gamma when CLOB returned an error (e.g. 404).
	gammaBaseURL = strings.TrimSpace(gammaBaseURL)
	if gammaBaseURL == "" {
		// No Gamma fallback configured — return the CLOB error.
		return nil, err
	}
	outcome, gammaErr := resolveViaGamma(ctx, gammaBaseURL, conditionID)
	if gammaErr == nil && outcome != nil {
		return outcome, nil
	}
	// Both CLOB and Gamma failed — return original CLOB error.
	return nil, err
}

// resolveViaGamma queries the Gamma API for a market by condition ID.
func resolveViaGamma(ctx context.Context, gammaBaseURL, conditionID string) (*polytypes.CLOBMarketOutcome, error) {
	g := newGammaClient(gammaBaseURL)
	markets, err := g.Markets(ctx, &polytypes.GetMarketsParams{ConditionIDs: []string{conditionID}})
	if err != nil {
		return nil, fmt.Errorf("gamma: markets query: %w", err)
	}
	for _, m := range markets {
		if !m.Closed {
			continue
		}
		return &polytypes.CLOBMarketOutcome{
			Status:      polytypes.CLOBOutcomeUnresolved,
			ConditionID: conditionID,
			Closed:      true,
			Source:      fmt.Sprintf("gamma:closed_condition_id=%s", conditionID),
		}, nil
	}
	return nil, fmt.Errorf("gamma: no closed market found for condition_id=%s", conditionID)
}

// newGammaClient creates a Gamma API client for the given base URL.
func newGammaClient(gammaBaseURL string) *gamma.Client {
	return gamma.NewClient(gammaBaseURL, nil)
}

// MarketByToken returns the parent CLOB market identifiers for a token ID.
func (c *Client) MarketByToken(ctx context.Context, tokenID string) (*polytypes.CLOBMarketByTokenResponse, error) {
	var result polytypes.CLOBMarketByTokenResponse
	if err := c.transport.Get(ctx, "/markets-by-token/"+url.PathEscape(tokenID), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// OrderBook returns L2 order book depth for a token.
func (c *Client) OrderBook(ctx context.Context, tokenID string) (*polytypes.OrderBook, error) {
	path := fmt.Sprintf("/book?token_id=%s", url.QueryEscape(tokenID))
	var result polytypes.OrderBook
	if err := c.transport.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// OrderBooks returns order books for multiple tokens (POST).
func (c *Client) OrderBooks(ctx context.Context, params []polytypes.BookParams) ([]polytypes.OrderBook, error) {
	var result []polytypes.OrderBook
	if err := c.transport.Post(ctx, "/books", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// Price returns the best bid/ask for a token.
func (c *Client) Price(ctx context.Context, tokenID, side string) (string, error) {
	path := fmt.Sprintf("/price?token_id=%s&side=%s", url.QueryEscape(tokenID), url.QueryEscape(side))
	var wrapper struct {
		Price polytypes.NumericString `json:"price"`
	}
	if err := c.transport.Get(ctx, path, &wrapper); err != nil {
		return "", err
	}
	return string(wrapper.Price), nil
}

// Prices returns prices for multiple tokens (POST).
func (c *Client) Prices(ctx context.Context, params []polytypes.BookParams) (map[string]string, error) {
	var raw json.RawMessage
	if err := c.transport.Post(ctx, "/prices", params, &raw); err != nil {
		return nil, err
	}
	return parseTokenValueMap(raw, params, "price")
}

// Midpoint returns the midpoint price for a token.
func (c *Client) Midpoint(ctx context.Context, tokenID string) (string, error) {
	path := fmt.Sprintf("/midpoint?token_id=%s", url.QueryEscape(tokenID))
	var wrapper struct {
		Mid      polytypes.NumericString `json:"mid"`
		MidPrice polytypes.NumericString `json:"mid_price"`
	}
	if err := c.transport.Get(ctx, path, &wrapper); err != nil {
		return "", err
	}
	return firstNonEmpty(string(wrapper.Mid), string(wrapper.MidPrice)), nil
}

// Midpoints returns midpoints for multiple tokens (POST).
func (c *Client) Midpoints(ctx context.Context, params []polytypes.BookParams) (map[string]string, error) {
	var raw json.RawMessage
	if err := c.transport.Post(ctx, "/midpoints", params, &raw); err != nil {
		return nil, err
	}
	return parseTokenValueMap(raw, params, "mid", "mid_price")
}

// Spread returns the spread for a token.
func (c *Client) Spread(ctx context.Context, tokenID string) (string, error) {
	path := fmt.Sprintf("/spread?token_id=%s", url.QueryEscape(tokenID))
	var wrapper struct {
		Spread string `json:"spread"`
	}
	if err := c.transport.Get(ctx, path, &wrapper); err != nil {
		return "", err
	}
	return wrapper.Spread, nil
}

// TickSize returns the tick size for a token.
func (c *Client) TickSize(ctx context.Context, tokenID string) (*polytypes.TickSize, error) {
	path := fmt.Sprintf("/tick-size?token_id=%s", url.QueryEscape(tokenID))
	var raw struct {
		MinimumTickSize  interface{} `json:"minimum_tick_size"`
		MinimumOrderSize interface{} `json:"minimum_order_size"`
		TickSize         interface{} `json:"tick_size"`
	}
	if err := c.transport.Get(ctx, path, &raw); err != nil {
		return nil, err
	}
	toString := func(v interface{}) string {
		if v == nil {
			return ""
		}
		switch val := v.(type) {
		case string:
			return val
		case float64:
			return strconv.FormatFloat(val, 'f', -1, 64)
		case json.Number:
			return val.String()
		default:
			return fmt.Sprintf("%v", val)
		}
	}
	return &polytypes.TickSize{
		MinimumTickSize:  toString(raw.MinimumTickSize),
		MinimumOrderSize: toString(raw.MinimumOrderSize),
		TickSize:         toString(raw.TickSize),
	}, nil
}

// NegRisk returns neg risk info for a token.
func (c *Client) NegRisk(ctx context.Context, tokenID string) (*polytypes.NegRiskInfo, error) {
	path := fmt.Sprintf("/neg-risk?token_id=%s", url.QueryEscape(tokenID))
	var result polytypes.NegRiskInfo
	if err := c.transport.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// FeeRateBps returns the fee rate in basis points for a token.
func (c *Client) FeeRateBps(ctx context.Context, tokenID string) (int, error) {
	path := fmt.Sprintf("/fee-rate?token_id=%s", url.QueryEscape(tokenID))
	var wrapper struct {
		FeeRateBps      polytypes.NumericString `json:"fee_rate_bps"`
		FeeRateBpsCamel polytypes.NumericString `json:"feeRateBps"`
		BaseFee         polytypes.NumericString `json:"base_fee"`
		BaseFeeCamel    polytypes.NumericString `json:"baseFee"`
	}
	if err := c.transport.Get(ctx, path, &wrapper); err != nil {
		return 0, err
	}
	value := firstNonEmpty(string(wrapper.FeeRateBps), string(wrapper.FeeRateBpsCamel), string(wrapper.BaseFee), string(wrapper.BaseFeeCamel))
	if value == "" {
		return 0, nil
	}
	fee, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	return fee, nil
}

// --- Rewards ---

func (c *Client) RewardsConfig(ctx context.Context) ([]polytypes.RewardsConfig, error) {
	var result []polytypes.RewardsConfig
	if err := c.transport.Get(ctx, "/rewards/config", &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) RawRewards(ctx context.Context, market string) ([]polytypes.RawRewards, error) {
	path := fmt.Sprintf("/rewards/raw?market=%s", url.QueryEscape(market))
	var result []polytypes.RawRewards
	if err := c.transport.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) UserEarnings(ctx context.Context, date string) ([]polytypes.UserEarnings, error) {
	path := fmt.Sprintf("/rewards/earnings?date=%s", url.QueryEscape(date))
	var result []polytypes.UserEarnings
	if err := c.transport.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) TotalEarnings(ctx context.Context, date string) (*polytypes.TotalEarnings, error) {
	var result polytypes.TotalEarnings
	if err := c.transport.Get(ctx, "/rewards/total-earnings?date="+url.QueryEscape(date), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) RewardPercentages(ctx context.Context) ([]polytypes.RewardPercentages, error) {
	var result []polytypes.RewardPercentages
	if err := c.transport.Get(ctx, "/rewards/percentages", &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) UserRewardsByMarket(ctx context.Context, params *polytypes.UserRewardsByMarketRequest) ([]polytypes.UserRewardsMarket, error) {
	var result []polytypes.UserRewardsMarket
	if err := c.transport.Get(ctx, "/rewards/markets", &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) RebatedFees(ctx context.Context) ([]polytypes.RebatedFees, error) {
	var result []polytypes.RebatedFees
	if err := c.transport.Get(ctx, "/rebates", &result); err != nil {
		return nil, err
	}
	return result, nil
}

// --- Simplified & Sampling Markets ---

func (c *Client) SimplifiedMarkets(ctx context.Context, nextCursor string) (*polytypes.CLOBPaginatedMarkets, error) {
	path := "/simplified-markets"
	if nextCursor != "" {
		path += "?next_cursor=" + url.QueryEscape(nextCursor)
	}
	var result polytypes.CLOBPaginatedMarkets
	if err := c.transport.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) SamplingMarkets(ctx context.Context, nextCursor string) (*polytypes.CLOBPaginatedMarkets, error) {
	path := "/sampling-markets"
	if nextCursor != "" {
		path += "?next_cursor=" + url.QueryEscape(nextCursor)
	}
	var result polytypes.CLOBPaginatedMarkets
	if err := c.transport.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) SamplingSimplifiedMarkets(ctx context.Context, nextCursor string) (*polytypes.CLOBPaginatedMarkets, error) {
	path := "/sampling-simplified-markets"
	if nextCursor != "" {
		path += "?next_cursor=" + url.QueryEscape(nextCursor)
	}
	var result polytypes.CLOBPaginatedMarkets
	if err := c.transport.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// --- Order Scoring ---

// PublicTrades returns unauthenticated CLOB trades, optionally filtered by market.
func (c *Client) PublicTrades(ctx context.Context, market string) ([]TradeRecord, error) {
	path := "/trades"
	if strings.TrimSpace(market) != "" {
		path += "?market=" + url.QueryEscape(strings.TrimSpace(market))
	}
	var raw json.RawMessage
	if err := c.transport.Get(ctx, path, &raw); err != nil {
		return nil, err
	}
	var rows []TradeRecord
	if err := json.Unmarshal(raw, &rows); err == nil {
		return rows, nil
	}
	var wrapped struct {
		Trades []TradeRecord `json:"trades"`
		Data   []TradeRecord `json:"data"`
	}
	if err := json.Unmarshal(raw, &wrapped); err != nil {
		return nil, err
	}
	if wrapped.Trades != nil {
		return wrapped.Trades, nil
	}
	return wrapped.Data, nil
}

// BuilderTrade represents a trade attributed to a builder code.
type BuilderTrade struct {
	TradeID   string `json:"trade_id"`
	OrderID   string `json:"order_id"`
	Market    string `json:"market"`
	AssetID   string `json:"asset_id"`
	Side      string `json:"side"`
	Size      string `json:"size"`
	Price     string `json:"price"`
	Timestamp string `json:"timestamp"`
}

func (t *BuilderTrade) UnmarshalJSON(b []byte) error {
	var raw struct {
		TradeID        polytypes.NumericString `json:"trade_id"`
		TradeIDCamel   polytypes.NumericString `json:"tradeId"`
		OrderID        polytypes.NumericString `json:"order_id"`
		OrderIDCamel   polytypes.NumericString `json:"orderId"`
		Market         polytypes.NumericString `json:"market"`
		AssetID        polytypes.NumericString `json:"asset_id"`
		AssetIDCamel   polytypes.NumericString `json:"assetId"`
		Side           string                  `json:"side"`
		Size           polytypes.NumericString `json:"size"`
		Price          polytypes.NumericString `json:"price"`
		Timestamp      polytypes.NumericString `json:"timestamp"`
		TimestampCamel polytypes.NumericString `json:"createdAt"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	t.TradeID = firstNonEmpty(string(raw.TradeID), string(raw.TradeIDCamel))
	t.OrderID = firstNonEmpty(string(raw.OrderID), string(raw.OrderIDCamel))
	t.Market = string(raw.Market)
	t.AssetID = firstNonEmpty(string(raw.AssetID), string(raw.AssetIDCamel))
	t.Side = raw.Side
	t.Size = string(raw.Size)
	t.Price = string(raw.Price)
	t.Timestamp = firstNonEmpty(string(raw.Timestamp), string(raw.TimestampCamel))
	return nil
}

// BuilderTrades returns trades attributed to the configured builder code.
func (c *Client) BuilderTrades(ctx context.Context, limit int) ([]BuilderTrade, error) {
	path := fmt.Sprintf("/builder-trades?limit=%d", limit)
	var result struct {
		Trades []BuilderTrade `json:"trades"`
	}
	if err := c.transport.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return result.Trades, nil
}

func (c *Client) OrderScoring(ctx context.Context, orderID string) (bool, error) {
	var wrapper struct {
		Scoring json.RawMessage `json:"scoring"`
	}
	if err := c.transport.Get(ctx, "/orders/scoring?order_id="+url.QueryEscape(orderID), &wrapper); err != nil {
		return false, err
	}
	return boolishRaw(wrapper.Scoring), nil
}

func (c *Client) OrdersScoring(ctx context.Context, orderIDs []string) ([]bool, error) {
	var raw []json.RawMessage
	body := map[string]interface{}{"order_ids": orderIDs}
	if err := c.transport.Post(ctx, "/orders/scoring", body, &raw); err != nil {
		return nil, err
	}
	result := make([]bool, len(raw))
	for i, value := range raw {
		result[i] = boolishRaw(value)
	}
	return result, nil
}

func boolishRaw(raw json.RawMessage) bool {
	text := strings.Trim(strings.ToLower(strings.TrimSpace(string(raw))), "\"")
	return text == "true" || text == "1"
}

// LastTradePrice returns the last trade price for a token.
func (c *Client) LastTradePrice(ctx context.Context, tokenID string) (string, error) {
	path := fmt.Sprintf("/last-trade-price?token_id=%s", url.QueryEscape(tokenID))
	var wrapper struct {
		Price polytypes.NumericString `json:"price"`
	}
	if err := c.transport.Get(ctx, path, &wrapper); err != nil {
		return "", err
	}
	return string(wrapper.Price), nil
}

// LastTradesPrices returns last trade prices for multiple tokens (POST).
func (c *Client) LastTradesPrices(ctx context.Context, params []polytypes.BookParams) (map[string]string, error) {
	var raw json.RawMessage
	if err := c.transport.Post(ctx, "/last-trades-prices", params, &raw); err != nil {
		return nil, err
	}
	return parseLastTradePrices(raw, params)
}

// PricesHistory returns OHLCV price history.
func (c *Client) PricesHistory(ctx context.Context, params *polytypes.PriceHistoryParams) (*polytypes.PriceHistory, error) {
	q := url.Values{}
	if params.Market != "" {
		q.Set("market", params.Market)
	}
	if params.Interval != "" {
		q.Set("interval", params.Interval)
	}
	if params.Fidelity > 0 {
		q.Set("fidelity", strconv.Itoa(params.Fidelity))
	}
	if params.StartTS > 0 {
		q.Set("startTs", strconv.FormatInt(params.StartTS, 10))
	}
	if params.EndTS > 0 {
		q.Set("endTs", strconv.FormatInt(params.EndTS, 10))
	}
	path := "/prices-history"
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	var result polytypes.PriceHistory
	if err := c.transport.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
