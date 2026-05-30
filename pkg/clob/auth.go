package clob

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/polygolem/internal/auth"
	internalclob "github.com/TrebuchetDynamics/polygolem/internal/clob"
	"github.com/TrebuchetDynamics/polygolem/internal/jsonx"
)

// APIKey is a Polymarket CLOB L2 credential.
type APIKey struct {
	Key        string `json:"apiKey"`
	Secret     string `json:"secret"`
	Passphrase string `json:"passphrase"`
}

// BalanceAllowanceParams filters CLOB collateral or conditional-token balance
// and allowance reads. V2 calls always use deposit-wallet signature type 3.
type BalanceAllowanceParams struct {
	Asset     string
	AssetType string
	TokenID   string
}

// BalanceAllowanceResponse is the authenticated CLOB balance/allowance state.
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

// jsonStringOrNumber unwraps a JSON value that may be a string or a number,
// returning the underlying lexical text without quotes.
func jsonStringOrNumber(raw json.RawMessage) string {
	s := strings.TrimSpace(string(raw))
	if s == "" || s == "null" {
		return ""
	}
	if s[0] == '"' {
		var v string
		if err := json.Unmarshal(raw, &v); err == nil {
			return v
		}
	}
	return s
}

// CreateOrderParams is the public input to CreateLimitOrder.
type CreateOrderParams struct {
	TokenID    string
	Side       string
	Price      string
	Size       string
	OrderType  string
	Expiration string
	PostOnly   bool
}

// MarketOrderParams is the public input to CreateMarketOrder.
type MarketOrderParams struct {
	TokenID   string
	Side      string
	Amount    string
	Price     string
	OrderType string
}

// OrderPlacementResponse is the CLOB response after posting a signed order.
type OrderPlacementResponse struct {
	Success            bool     `json:"success"`
	OrderID            string   `json:"orderID"`
	Status             string   `json:"status"`
	MakingAmount       string   `json:"makingAmount,omitempty"`
	TakingAmount       string   `json:"takingAmount,omitempty"`
	ErrorMsg           string   `json:"errorMsg,omitempty"`
	TransactionHash    string   `json:"transaction_hash,omitempty"`
	TransactionsHashes []string `json:"transactionsHashes,omitempty"`
	TradeIDs           []string `json:"tradeIDs,omitempty"`
}

func (r *OrderPlacementResponse) UnmarshalJSON(data []byte) error {
	type alias OrderPlacementResponse
	aux := struct {
		*alias
		OrderIDSnake            string            `json:"order_id"`
		MakingAmountRaw         json.RawMessage   `json:"makingAmount"`
		MakingAmountSnake       json.RawMessage   `json:"making_amount"`
		TakingAmountRaw         json.RawMessage   `json:"takingAmount"`
		TakingAmountSnake       json.RawMessage   `json:"taking_amount"`
		TransactionHashCamel    string            `json:"transactionHash"`
		ErrorMsgSnake           string            `json:"error_msg"`
		TransactionsHashesRaw   []json.RawMessage `json:"transactionsHashes"`
		TransactionHashesAlias  []json.RawMessage `json:"transactionHashes"`
		TransactionsHashesSnake []json.RawMessage `json:"transaction_hashes"`
		TradeIDsRaw             []json.RawMessage `json:"tradeIDs"`
		TradeIDsAlias           []json.RawMessage `json:"tradeIds"`
		TradeIDsSnake           []json.RawMessage `json:"trade_ids"`
	}{alias: (*alias)(r)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if r.OrderID == "" {
		r.OrderID = aux.OrderIDSnake
	}
	if r.MakingAmount == "" {
		r.MakingAmount = firstNonEmptyString(jsonStringOrNumber(aux.MakingAmountRaw), jsonStringOrNumber(aux.MakingAmountSnake))
	}
	if r.TakingAmount == "" {
		r.TakingAmount = firstNonEmptyString(jsonStringOrNumber(aux.TakingAmountRaw), jsonStringOrNumber(aux.TakingAmountSnake))
	}
	if r.TransactionHash == "" {
		r.TransactionHash = aux.TransactionHashCamel
	}
	if r.ErrorMsg == "" {
		r.ErrorMsg = aux.ErrorMsgSnake
	}
	if len(r.TransactionsHashes) == 0 && aux.TransactionsHashesRaw != nil {
		r.TransactionsHashes = rawStringList(aux.TransactionsHashesRaw)
	}
	if len(r.TransactionsHashes) == 0 && aux.TransactionHashesAlias != nil {
		r.TransactionsHashes = rawStringList(aux.TransactionHashesAlias)
	}
	if len(r.TransactionsHashes) == 0 && aux.TransactionsHashesSnake != nil {
		r.TransactionsHashes = rawStringList(aux.TransactionsHashesSnake)
	}
	if len(r.TradeIDs) == 0 && aux.TradeIDsRaw != nil {
		r.TradeIDs = rawStringList(aux.TradeIDsRaw)
	}
	if len(r.TradeIDs) == 0 && aux.TradeIDsAlias != nil {
		r.TradeIDs = rawStringList(aux.TradeIDsAlias)
	}
	if len(r.TradeIDs) == 0 && aux.TradeIDsSnake != nil {
		r.TradeIDs = rawStringList(aux.TradeIDsSnake)
	}
	return nil
}

// BatchOrderResponse is returned by CreateBatchOrders.
type BatchOrderResponse struct {
	Orders []OrderPlacementResponse `json:"orders"`
}

// CancelOrdersResponse is returned by CLOB cancellation endpoints.
type CancelOrdersResponse struct {
	Canceled    []string          `json:"canceled"`
	NotCanceled map[string]string `json:"not_canceled"`
}

func (r *CancelOrdersResponse) UnmarshalJSON(data []byte) error {
	var raw struct {
		Canceled         []json.RawMessage          `json:"canceled"`
		NotCanceled      map[string]json.RawMessage `json:"not_canceled"`
		NotCanceledCamel map[string]json.RawMessage `json:"notCanceled"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if raw.Canceled != nil {
		r.Canceled = make([]string, 0, len(raw.Canceled))
		for _, item := range raw.Canceled {
			r.Canceled = append(r.Canceled, jsonStringOrNumber(item))
		}
	}
	r.NotCanceled = rawStringMap(raw.NotCanceled)
	if len(r.NotCanceled) == 0 && len(raw.NotCanceledCamel) > 0 {
		r.NotCanceled = rawStringMap(raw.NotCanceledCamel)
	}
	return nil
}

func rawStringMap(raw map[string]json.RawMessage) map[string]string {
	if raw == nil {
		return nil
	}
	out := make(map[string]string, len(raw))
	for key, value := range raw {
		out[key] = jsonStringOrNumber(value)
	}
	return out
}

func rawStringList(raw []json.RawMessage) []string {
	out := make([]string, 0, len(raw))
	for _, value := range raw {
		out = append(out, jsonStringOrNumber(value))
	}
	return out
}

func firstNonEmptyRaw(values ...json.RawMessage) json.RawMessage {
	for _, value := range values {
		if jsonStringOrNumber(value) != "" {
			return value
		}
	}
	return nil
}

func jsonIntOrNumberString(raw json.RawMessage) int {
	value := jsonStringOrNumber(raw)
	if value == "" {
		return 0
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return n
}

// CancelMarketParams filters cancel-market requests by condition ID or token ID.
type CancelMarketParams struct {
	Market string
	Asset  string
}

// OrderRecord is an authenticated CLOB order record.
type OrderRecord struct {
	ID              string   `json:"id"`
	Status          string   `json:"status"`
	Owner           string   `json:"owner"`
	Market          string   `json:"market"`
	AssetID         string   `json:"asset_id"`
	Side            string   `json:"side"`
	OriginalSize    string   `json:"original_size"`
	SizeMatched     string   `json:"size_matched"`
	Price           string   `json:"price"`
	Outcome         string   `json:"outcome"`
	Type            string   `json:"type,omitempty"`
	OrderType       string   `json:"order_type,omitempty"`
	SignatureType   int      `json:"signature_type,omitempty"`
	CreatedAt       string   `json:"created_at"`
	Expiration      string   `json:"expiration"`
	MakerAddress    string   `json:"maker_address"`
	AssociateTrades []string `json:"associate_trades,omitempty"`
}

func (o *OrderRecord) UnmarshalJSON(data []byte) error {
	type alias OrderRecord
	aux := struct {
		*alias
		AssetIDCamel      json.RawMessage   `json:"assetId"`
		OriginalSize      json.RawMessage   `json:"original_size"`
		OriginalSizeCamel json.RawMessage   `json:"originalSize"`
		SizeMatched       json.RawMessage   `json:"size_matched"`
		SizeMatchedCamel  json.RawMessage   `json:"sizeMatched"`
		Price             json.RawMessage   `json:"price"`
		OrderTypeCamel    json.RawMessage   `json:"orderType"`
		CreatedAt         json.RawMessage   `json:"created_at"`
		CreatedAtCamel    json.RawMessage   `json:"createdAt"`
		Expiration        json.RawMessage   `json:"expiration"`
		SignatureType     json.RawMessage   `json:"signature_type"`
		SignatureCamel    json.RawMessage   `json:"signatureType"`
		MakerAddrCamel    json.RawMessage   `json:"makerAddress"`
		AssociateTrades   []json.RawMessage `json:"associate_trades"`
		AssocTradesCamel  []json.RawMessage `json:"associateTrades"`
	}{alias: (*alias)(o)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if o.AssetID == "" {
		o.AssetID = jsonStringOrNumber(aux.AssetIDCamel)
	}
	o.OriginalSize = firstNonEmptyString(jsonStringOrNumber(aux.OriginalSize), jsonStringOrNumber(aux.OriginalSizeCamel))
	o.SizeMatched = firstNonEmptyString(jsonStringOrNumber(aux.SizeMatched), jsonStringOrNumber(aux.SizeMatchedCamel))
	o.Price = jsonStringOrNumber(aux.Price)
	if o.OrderType == "" {
		o.OrderType = jsonStringOrNumber(aux.OrderTypeCamel)
	}
	o.CreatedAt = firstNonEmptyString(jsonStringOrNumber(aux.CreatedAt), jsonStringOrNumber(aux.CreatedAtCamel))
	o.Expiration = jsonStringOrNumber(aux.Expiration)
	if len(aux.SignatureType) > 0 || len(aux.SignatureCamel) > 0 {
		o.SignatureType = jsonIntOrNumberString(firstNonEmptyRaw(aux.SignatureType, aux.SignatureCamel))
	}
	if o.MakerAddress == "" {
		o.MakerAddress = jsonStringOrNumber(aux.MakerAddrCamel)
	}
	if aux.AssociateTrades != nil {
		o.AssociateTrades = rawStringList(aux.AssociateTrades)
	}
	if len(o.AssociateTrades) == 0 && aux.AssocTradesCamel != nil {
		o.AssociateTrades = rawStringList(aux.AssocTradesCamel)
	}
	return nil
}

// TradeRecord is an authenticated CLOB trade record.
type TradeRecord struct {
	ID              string `json:"id"`
	Status          string `json:"status"`
	Market          string `json:"market"`
	AssetID         string `json:"asset_id"`
	Side            string `json:"side"`
	Price           string `json:"price"`
	Size            string `json:"size"`
	FeeRateBps      string `json:"fee_rate_bps"`
	Outcome         string `json:"outcome"`
	Owner           string `json:"owner"`
	Builder         string `json:"builder"`
	MatchedAmount   string `json:"matched_amount"`
	TransactionHash string `json:"transaction_hash"`
	CreatedAt       string `json:"created_at"`
	LastUpdated     string `json:"last_updated"`
}

func (t *TradeRecord) UnmarshalJSON(data []byte) error {
	type alias TradeRecord
	aux := struct {
		*alias
		AssetIDCamel     json.RawMessage `json:"assetId"`
		Price            json.RawMessage `json:"price"`
		Size             json.RawMessage `json:"size"`
		FeeRateBps       json.RawMessage `json:"fee_rate_bps"`
		FeeRateBpsCamel  json.RawMessage `json:"feeRateBps"`
		MatchedAmount    json.RawMessage `json:"matched_amount"`
		MatchedAmtCamel  json.RawMessage `json:"matchedAmount"`
		TxHashCamel      json.RawMessage `json:"transactionHash"`
		CreatedAt        json.RawMessage `json:"created_at"`
		CreatedAtCamel   json.RawMessage `json:"createdAt"`
		LastUpdated      json.RawMessage `json:"last_updated"`
		LastUpdatedCamel json.RawMessage `json:"lastUpdated"`
	}{alias: (*alias)(t)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if t.AssetID == "" {
		t.AssetID = jsonStringOrNumber(aux.AssetIDCamel)
	}
	t.Price = jsonStringOrNumber(aux.Price)
	t.Size = jsonStringOrNumber(aux.Size)
	t.FeeRateBps = firstNonEmptyString(jsonStringOrNumber(aux.FeeRateBps), jsonStringOrNumber(aux.FeeRateBpsCamel))
	t.MatchedAmount = firstNonEmptyString(jsonStringOrNumber(aux.MatchedAmount), jsonStringOrNumber(aux.MatchedAmtCamel))
	if t.TransactionHash == "" {
		t.TransactionHash = jsonStringOrNumber(aux.TxHashCamel)
	}
	t.CreatedAt = firstNonEmptyString(jsonStringOrNumber(aux.CreatedAt), jsonStringOrNumber(aux.CreatedAtCamel))
	t.LastUpdated = firstNonEmptyString(jsonStringOrNumber(aux.LastUpdated), jsonStringOrNumber(aux.LastUpdatedCamel))
	return nil
}

type ProbeClassification string

const (
	ProbeMarketWide              ProbeClassification = "market_wide"
	ProbeAccountScoped           ProbeClassification = "account_scoped"
	ProbeEmptyInconclusive       ProbeClassification = "empty_inconclusive"
	ProbeUnauthorizedUnavailable ProbeClassification = "unauthorized_or_unavailable"
	ProbeSchemaUnknown           ProbeClassification = "schema_unknown"
)

type MarketTradesProbeRequest struct {
	Market     string `json:"market,omitempty"`
	AssetID    string `json:"asset_id,omitempty"`
	NextCursor string `json:"next_cursor,omitempty"`
}

type MarketTradesProbeResult struct {
	Classification ProbeClassification `json:"classification"`
	Endpoint       string              `json:"endpoint"`
	SelectorType   string              `json:"selector_type"`
	Selector       string              `json:"selector"`
	HTTPStatus     int                 `json:"http_status"`
	RowCount       int                 `json:"row_count"`
	CursorPresent  bool                `json:"cursor_present"`
	NextCursor     string              `json:"next_cursor,omitempty"`
	TimestampMin   string              `json:"timestamp_min,omitempty"`
	TimestampMax   string              `json:"timestamp_max,omitempty"`
	ObservedFields []string            `json:"observed_fields,omitempty"`
	Warning        string              `json:"warning,omitempty"`
}

// BuilderFeeKeyRecord represents one CLOB builder-fee key.
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

// CreateOrDeriveAPIKey creates new CLOB L2 credentials, falling back to
// deterministic derivation when a key already exists.
func (c *Client) CreateOrDeriveAPIKey(ctx context.Context, privateKey string) (APIKey, error) {
	key, err := c.inner.CreateOrDeriveAPIKey(ctx, privateKey)
	if err != nil {
		return APIKey{}, err
	}
	return apiKeyFromInternal(key), nil
}

// CreateOrDeriveAPIKeyForAddress creates or derives CLOB L2 credentials for
// a deposit/smart wallet address while signing L1 auth with the controlling
// EOA private key.
func (c *Client) CreateOrDeriveAPIKeyForAddress(ctx context.Context, privateKey, ownerAddress string) (APIKey, error) {
	key, err := c.inner.CreateOrDeriveAPIKeyForAddress(ctx, privateKey, ownerAddress)
	if err != nil {
		return APIKey{}, err
	}
	return apiKeyFromInternal(key), nil
}

// CreateAPIKeyForAddress creates CLOB L2 credentials for a deposit/smart
// wallet address while signing L1 auth with the controlling EOA private key.
func (c *Client) CreateAPIKeyForAddress(ctx context.Context, privateKey, ownerAddress string) (APIKey, error) {
	key, err := c.inner.CreateAPIKeyForAddress(ctx, privateKey, ownerAddress)
	if err != nil {
		return APIKey{}, err
	}
	return apiKeyFromInternal(key), nil
}

// DeriveAPIKeyForAddress derives existing CLOB L2 credentials for a
// deposit/smart wallet address while signing L1 auth with the controlling EOA
// private key.
func (c *Client) DeriveAPIKeyForAddress(ctx context.Context, privateKey, ownerAddress string) (APIKey, error) {
	key, err := c.inner.DeriveAPIKeyForAddress(ctx, privateKey, ownerAddress)
	if err != nil {
		return APIKey{}, err
	}
	return apiKeyFromInternal(key), nil
}

// DeriveAPIKey derives existing CLOB L2 credentials.
func (c *Client) DeriveAPIKey(ctx context.Context, privateKey string) (APIKey, error) {
	key, err := c.inner.DeriveAPIKey(ctx, privateKey)
	if err != nil {
		return APIKey{}, err
	}
	return apiKeyFromInternal(key), nil
}

// CreateBuilderFeeKey mints a CLOB builder-fee key via L2 auth.
func (c *Client) CreateBuilderFeeKey(ctx context.Context, privateKey string) (APIKey, error) {
	key, err := c.inner.CreateBuilderFeeKey(ctx, privateKey)
	if err != nil {
		return APIKey{}, err
	}
	return apiKeyFromInternal(key), nil
}

// ListBuilderFeeKeys lists CLOB builder-fee keys for the authenticated wallet.
func (c *Client) ListBuilderFeeKeys(ctx context.Context, privateKey string) ([]BuilderFeeKeyRecord, error) {
	rows, err := c.inner.ListBuilderFeeKeys(ctx, privateKey)
	if err != nil {
		return nil, err
	}
	out := make([]BuilderFeeKeyRecord, len(rows))
	for i, row := range rows {
		out[i] = BuilderFeeKeyRecord{
			Key:        row.Key,
			Secret:     row.Secret,
			Passphrase: row.Passphrase,
			CreatedAt:  row.CreatedAt,
			UpdatedAt:  row.UpdatedAt,
		}
	}
	return out, nil
}

// RevokeBuilderFeeKey deletes a CLOB builder-fee key.
func (c *Client) RevokeBuilderFeeKey(ctx context.Context, privateKey, builderKey string) error {
	return c.inner.RevokeBuilderFeeKey(ctx, privateKey, builderKey)
}

// BalanceAllowance returns CLOB collateral or conditional token balance and allowances.
func (c *Client) BalanceAllowance(ctx context.Context, privateKey string, params BalanceAllowanceParams) (*BalanceAllowanceResponse, error) {
	row, err := c.inner.BalanceAllowance(ctx, privateKey, balanceAllowanceParamsToInternal(params))
	if err != nil {
		return nil, err
	}
	return balanceAllowanceFromInternal(row), nil
}

// UpdateBalanceAllowance refreshes the CLOB balance/allowance cache.
func (c *Client) UpdateBalanceAllowance(ctx context.Context, privateKey string, params BalanceAllowanceParams) (*BalanceAllowanceResponse, error) {
	row, err := c.inner.UpdateBalanceAllowance(ctx, privateKey, balanceAllowanceParamsToInternal(params))
	if err != nil {
		return nil, err
	}
	return balanceAllowanceFromInternal(row), nil
}

// ListOrders returns the authenticated user's open CLOB orders.
func (c *Client) ListOrders(ctx context.Context, privateKey string) ([]OrderRecord, error) {
	rows, err := c.inner.ListOrders(ctx, privateKey)
	if err != nil {
		return nil, err
	}
	return orderRecordsFromInternal(rows), nil
}

// Order returns one authenticated CLOB order by order ID.
func (c *Client) Order(ctx context.Context, privateKey, orderID string) (*OrderRecord, error) {
	row, err := c.inner.Order(ctx, privateKey, orderID)
	if err != nil {
		return nil, err
	}
	return orderRecordFromInternal(row), nil
}

// ListTrades returns the authenticated user's CLOB trade history.
func (c *Client) ListTrades(ctx context.Context, privateKey string) ([]TradeRecord, error) {
	rows, err := c.inner.ListTrades(ctx, privateKey)
	if err != nil {
		return nil, err
	}
	return tradeRecordsFromInternal(rows), nil
}

func (c *Client) MarketTradesProbe(ctx context.Context, privateKey string, params MarketTradesProbeRequest) (*MarketTradesProbeResult, error) {
	row, err := c.inner.MarketTradesProbe(ctx, privateKey, internalclob.MarketTradesProbeRequest{
		Market:     params.Market,
		AssetID:    params.AssetID,
		NextCursor: params.NextCursor,
	})
	if err != nil {
		return nil, err
	}
	return marketTradesProbeFromInternal(row), nil
}

// CancelOrder cancels a single open CLOB order.
func (c *Client) CancelOrder(ctx context.Context, privateKey, orderID string) (*CancelOrdersResponse, error) {
	row, err := c.inner.CancelOrder(ctx, privateKey, orderID)
	if err != nil {
		return nil, err
	}
	return cancelOrdersFromInternal(row), nil
}

// CancelOrders cancels multiple open CLOB orders by order ID.
func (c *Client) CancelOrders(ctx context.Context, privateKey string, orderIDs []string) (*CancelOrdersResponse, error) {
	row, err := c.inner.CancelOrders(ctx, privateKey, orderIDs)
	if err != nil {
		return nil, err
	}
	return cancelOrdersFromInternal(row), nil
}

// CancelAll cancels all open CLOB orders for the authenticated user.
func (c *Client) CancelAll(ctx context.Context, privateKey string) (*CancelOrdersResponse, error) {
	row, err := c.inner.CancelAll(ctx, privateKey)
	if err != nil {
		return nil, err
	}
	return cancelOrdersFromInternal(row), nil
}

// CancelMarket cancels open CLOB orders matching a market or asset filter.
func (c *Client) CancelMarket(ctx context.Context, privateKey string, params CancelMarketParams) (*CancelOrdersResponse, error) {
	row, err := c.inner.CancelMarket(ctx, privateKey, cancelMarketParamsToInternal(params))
	if err != nil {
		return nil, err
	}
	return cancelOrdersFromInternal(row), nil
}

// CreateLimitOrder signs and submits a V2 limit order.
func (c *Client) CreateLimitOrder(ctx context.Context, privateKey string, params CreateOrderParams) (*OrderPlacementResponse, error) {
	row, err := c.inner.CreateLimitOrder(ctx, privateKey, createOrderParamsToInternal(params))
	if err != nil {
		return nil, err
	}
	return orderPlacementFromInternal(row), nil
}

// CreateBatchOrders signs and posts multiple V2 limit orders to POST /orders.
func (c *Client) CreateBatchOrders(ctx context.Context, privateKey string, params []CreateOrderParams) (*BatchOrderResponse, error) {
	row, err := c.inner.CreateBatchOrders(ctx, privateKey, createOrderParamsListToInternal(params))
	if err != nil {
		return nil, err
	}
	return batchOrderResponseFromInternal(row), nil
}

// CreateMarketOrder signs and submits a V2 market order.
// For BUY, Amount is a USDC budget. For SELL, Amount is the share size to sell.
func (c *Client) CreateMarketOrder(ctx context.Context, privateKey string, params MarketOrderParams) (*OrderPlacementResponse, error) {
	row, err := c.inner.CreateMarketOrder(ctx, privateKey, marketOrderParamsToInternal(params))
	if err != nil {
		return nil, err
	}
	return orderPlacementFromInternal(row), nil
}

// Heartbeat sends a CLOB heartbeat ping for open-order keepalive behavior.
func (c *Client) Heartbeat(ctx context.Context, privateKey, heartbeatID string) error {
	return c.inner.Heartbeat(ctx, privateKey, heartbeatID)
}

// AutoHeartbeat sends periodic heartbeats until the returned cancel function
// is called.
func (c *Client) AutoHeartbeat(ctx context.Context, privateKey string, interval time.Duration) context.CancelFunc {
	return c.inner.AutoHeartbeat(ctx, privateKey, interval)
}

func apiKeyFromInternal(row auth.APIKey) APIKey {
	return APIKey{
		Key:        row.Key,
		Secret:     row.Secret,
		Passphrase: row.Passphrase,
	}
}

func apiKeyToInternal(row APIKey) auth.APIKey {
	return auth.APIKey{
		Key:        row.Key,
		Secret:     row.Secret,
		Passphrase: row.Passphrase,
	}
}

func apiKeyConfigured(row APIKey) bool {
	return strings.TrimSpace(row.Key) != "" ||
		strings.TrimSpace(row.Secret) != "" ||
		strings.TrimSpace(row.Passphrase) != ""
}

func balanceAllowanceParamsToInternal(params BalanceAllowanceParams) internalclob.BalanceAllowanceParams {
	return internalclob.BalanceAllowanceParams{
		Asset:     params.Asset,
		AssetType: params.AssetType,
		TokenID:   params.TokenID,
	}
}

func balanceAllowanceFromInternal(row *internalclob.BalanceAllowanceResponse) *BalanceAllowanceResponse {
	if row == nil {
		return nil
	}
	return &BalanceAllowanceResponse{
		Balance:    row.Balance,
		Allowances: row.Allowances,
		Allowance:  row.Allowance,
	}
}

func orderRecordsFromInternal(rows []internalclob.OrderRecord) []OrderRecord {
	out := make([]OrderRecord, len(rows))
	for i, row := range rows {
		out[i] = orderRecordValueFromInternal(row)
	}
	return out
}

func orderRecordFromInternal(row *internalclob.OrderRecord) *OrderRecord {
	if row == nil {
		return nil
	}
	out := orderRecordValueFromInternal(*row)
	return &out
}

func orderRecordValueFromInternal(row internalclob.OrderRecord) OrderRecord {
	return OrderRecord{
		ID:              row.ID,
		Status:          row.Status,
		Owner:           row.Owner,
		Market:          row.Market,
		AssetID:         row.AssetID,
		Side:            row.Side,
		OriginalSize:    row.OriginalSize,
		SizeMatched:     row.SizeMatched,
		Price:           row.Price,
		Outcome:         row.Outcome,
		Type:            row.Type,
		OrderType:       firstNonEmptyString(row.OrderType, row.Type),
		SignatureType:   row.SignatureType,
		CreatedAt:       row.CreatedAt,
		Expiration:      row.Expiration,
		MakerAddress:    row.MakerAddress,
		AssociateTrades: row.AssociateTrades,
	}
}

func firstNonEmptyString(values ...string) string {
	return jsonx.FirstString(values...)
}

func tradeRecordsFromInternal(rows []internalclob.TradeRecord) []TradeRecord {
	out := make([]TradeRecord, len(rows))
	for i, row := range rows {
		out[i] = TradeRecord{
			ID:              row.ID,
			Status:          row.Status,
			Market:          row.Market,
			AssetID:         row.AssetID,
			Side:            row.Side,
			Price:           row.Price,
			Size:            row.Size,
			FeeRateBps:      row.FeeRateBps,
			Outcome:         row.Outcome,
			Owner:           row.Owner,
			Builder:         row.Builder,
			MatchedAmount:   row.MatchedAmount,
			TransactionHash: row.TransactionHash,
			CreatedAt:       row.CreatedAt,
			LastUpdated:     row.LastUpdated,
		}
	}
	return out
}

func marketTradesProbeFromInternal(row *internalclob.MarketTradesProbeResult) *MarketTradesProbeResult {
	if row == nil {
		return nil
	}
	return &MarketTradesProbeResult{
		Classification: ProbeClassification(row.Classification),
		Endpoint:       row.Endpoint,
		SelectorType:   row.SelectorType,
		Selector:       row.Selector,
		HTTPStatus:     row.HTTPStatus,
		RowCount:       row.RowCount,
		CursorPresent:  row.CursorPresent,
		NextCursor:     row.NextCursor,
		TimestampMin:   row.TimestampMin,
		TimestampMax:   row.TimestampMax,
		ObservedFields: append([]string(nil), row.ObservedFields...),
		Warning:        row.Warning,
	}
}

func cancelOrdersFromInternal(row *internalclob.CancelOrdersResponse) *CancelOrdersResponse {
	if row == nil {
		return nil
	}
	return &CancelOrdersResponse{
		Canceled:    row.Canceled,
		NotCanceled: row.NotCanceled,
	}
}

func cancelMarketParamsToInternal(params CancelMarketParams) internalclob.CancelMarketParams {
	return internalclob.CancelMarketParams{
		Market: params.Market,
		Asset:  params.Asset,
	}
}

func createOrderParamsToInternal(params CreateOrderParams) internalclob.CreateOrderParams {
	return internalclob.CreateOrderParams{
		TokenID:    params.TokenID,
		Side:       params.Side,
		Price:      params.Price,
		Size:       params.Size,
		OrderType:  params.OrderType,
		Expiration: params.Expiration,
		PostOnly:   params.PostOnly,
	}
}

func createOrderParamsListToInternal(params []CreateOrderParams) []internalclob.CreateOrderParams {
	out := make([]internalclob.CreateOrderParams, len(params))
	for i, param := range params {
		out[i] = createOrderParamsToInternal(param)
	}
	return out
}

func marketOrderParamsToInternal(params MarketOrderParams) internalclob.MarketOrderParams {
	return internalclob.MarketOrderParams{
		TokenID:   params.TokenID,
		Side:      params.Side,
		Amount:    params.Amount,
		Price:     params.Price,
		OrderType: params.OrderType,
	}
}

func batchOrderResponseFromInternal(row *internalclob.BatchOrderResponse) *BatchOrderResponse {
	if row == nil {
		return nil
	}
	out := &BatchOrderResponse{Orders: make([]OrderPlacementResponse, len(row.Orders))}
	for i := range row.Orders {
		converted := orderPlacementFromInternal(&row.Orders[i])
		if converted != nil {
			out.Orders[i] = *converted
		}
	}
	return out
}

func orderPlacementFromInternal(row *internalclob.OrderPlacementResponse) *OrderPlacementResponse {
	if row == nil {
		return nil
	}
	return &OrderPlacementResponse{
		Success:            row.Success,
		OrderID:            row.OrderID,
		Status:             row.Status,
		MakingAmount:       row.MakingAmount,
		TakingAmount:       row.TakingAmount,
		ErrorMsg:           row.ErrorMsg,
		TransactionHash:    row.TransactionHash,
		TransactionsHashes: row.TransactionsHashes,
		TradeIDs:           row.TradeIDs,
	}
}
