package dataapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/TrebuchetDynamics/polygolem/internal/jsonx"
	"github.com/TrebuchetDynamics/polygolem/internal/transport"
)

// Client provides read-only Data API access.
type Client struct {
	transport *transport.Client
}

func NewClient(baseURL string, tc *transport.Client) *Client {
	if tc == nil {
		tc = transport.New(nil, transport.DefaultConfig(baseURL))
	}
	return &Client{transport: tc}
}

// --- Types ---

// Position is a current Data API position for a user (the proxy/deposit
// wallet, not the EOA). Field names follow Polymarket's documented camelCase
// schema; see https://docs.polymarket.com/api-reference/core/get-current-positions-for-a-user.md.
type Position struct {
	TokenID         string  `json:"asset"`
	ConditionID     string  `json:"conditionId"`
	MarketID        string  `json:"market"`
	Side            string  `json:"side"`
	EventID         string  `json:"eventId"`
	ProxyWallet     string  `json:"proxyWallet"`
	Size            float64 `json:"size"`
	AvgPrice        float64 `json:"avgPrice"`
	InitialValue    float64 `json:"initialValue"`
	CurrentValue    float64 `json:"currentValue"`
	CurrentPrice    float64 `json:"curPrice"`
	UnrealizedPnl   float64 `json:"unrealizedPnl"`
	CashPnl         float64 `json:"cashPnl"`
	PercentPnl      float64 `json:"percentPnl"`
	TotalBought     float64 `json:"totalBought"`
	RealizedPnl     float64 `json:"realizedPnl"`
	PercentRealized float64 `json:"percentRealizedPnl"`
	// V2 redemption-relevant fields.
	Redeemable      bool   `json:"redeemable"`
	Mergeable       bool   `json:"mergeable"`
	NegativeRisk    bool   `json:"negativeRisk"`
	Outcome         string `json:"outcome"`
	OutcomeIndex    int    `json:"outcomeIndex"`
	OppositeOutcome string `json:"oppositeOutcome"`
	OppositeAsset   string `json:"oppositeAsset"`
	EndDate         string `json:"endDate"`
	Title           string `json:"title"`
	Slug            string `json:"slug"`
	EventSlug       string `json:"eventSlug"`
	Icon            string `json:"icon"`
}

func (p *Position) UnmarshalJSON(data []byte) error {
	var aux struct {
		TokenID            string          `json:"asset"`
		TokenIDSnake       string          `json:"token_id"`
		ConditionID        string          `json:"conditionId"`
		ConditionIDSnake   string          `json:"condition_id"`
		MarketID           string          `json:"market"`
		MarketIDSnake      string          `json:"market_id"`
		Side               string          `json:"side"`
		EventID            string          `json:"eventId"`
		ProxyWallet        string          `json:"proxyWallet"`
		Size               json.RawMessage `json:"size"`
		AvgPrice           json.RawMessage `json:"avgPrice"`
		AvgPriceSnake      json.RawMessage `json:"avg_price"`
		InitialValue       json.RawMessage `json:"initialValue"`
		CurrentValue       json.RawMessage `json:"currentValue"`
		CurrentPrice       json.RawMessage `json:"curPrice"`
		CurrentPriceSnake  json.RawMessage `json:"current_price"`
		UnrealizedPnl      json.RawMessage `json:"unrealizedPnl"`
		UnrealizedPnlSnake json.RawMessage `json:"unrealized_pnl"`
		CashPnl            json.RawMessage `json:"cashPnl"`
		PercentPnl         json.RawMessage `json:"percentPnl"`
		TotalBought        json.RawMessage `json:"totalBought"`
		RealizedPnl        json.RawMessage `json:"realizedPnl"`
		PercentRealized    json.RawMessage `json:"percentRealizedPnl"`
		Redeemable         json.RawMessage `json:"redeemable"`
		Mergeable          json.RawMessage `json:"mergeable"`
		NegativeRisk       json.RawMessage `json:"negativeRisk"`
		Outcome            string          `json:"outcome"`
		OutcomeIndex       json.RawMessage `json:"outcomeIndex"`
		OppositeOutcome    string          `json:"oppositeOutcome"`
		OppositeAsset      string          `json:"oppositeAsset"`
		EndDate            string          `json:"endDate"`
		Title              string          `json:"title"`
		Slug               string          `json:"slug"`
		EventSlug          string          `json:"eventSlug"`
		Icon               string          `json:"icon"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	var err error
	*p = Position{
		TokenID:         firstNonEmpty(aux.TokenID, aux.TokenIDSnake),
		ConditionID:     firstNonEmpty(aux.ConditionID, aux.ConditionIDSnake),
		MarketID:        firstNonEmpty(aux.MarketID, aux.MarketIDSnake),
		Side:            aux.Side,
		EventID:         aux.EventID,
		ProxyWallet:     aux.ProxyWallet,
		Redeemable:      jsonBoolOrFalse(aux.Redeemable),
		Mergeable:       jsonBoolOrFalse(aux.Mergeable),
		NegativeRisk:    jsonBoolOrFalse(aux.NegativeRisk),
		Outcome:         aux.Outcome,
		OppositeOutcome: aux.OppositeOutcome,
		OppositeAsset:   aux.OppositeAsset,
		EndDate:         aux.EndDate,
		Title:           aux.Title,
		Slug:            aux.Slug,
		EventSlug:       aux.EventSlug,
		Icon:            aux.Icon,
	}
	if p.Size, err = jsonFloatOrZero(aux.Size); err != nil {
		return fmt.Errorf("decode position size: %w", err)
	}
	if p.AvgPrice, err = jsonFloatOrZero(firstRaw(aux.AvgPrice, aux.AvgPriceSnake)); err != nil {
		return fmt.Errorf("decode position avgPrice: %w", err)
	}
	if p.InitialValue, err = jsonFloatOrZero(aux.InitialValue); err != nil {
		return fmt.Errorf("decode position initialValue: %w", err)
	}
	if p.CurrentValue, err = jsonFloatOrZero(aux.CurrentValue); err != nil {
		return fmt.Errorf("decode position currentValue: %w", err)
	}
	if p.CurrentPrice, err = jsonFloatOrZero(firstRaw(aux.CurrentPrice, aux.CurrentPriceSnake)); err != nil {
		return fmt.Errorf("decode position curPrice: %w", err)
	}
	if p.UnrealizedPnl, err = jsonFloatOrZero(firstRaw(aux.UnrealizedPnl, aux.UnrealizedPnlSnake)); err != nil {
		return fmt.Errorf("decode position unrealizedPnl: %w", err)
	}
	if p.CashPnl, err = jsonFloatOrZero(aux.CashPnl); err != nil {
		return fmt.Errorf("decode position cashPnl: %w", err)
	}
	if p.PercentPnl, err = jsonFloatOrZero(aux.PercentPnl); err != nil {
		return fmt.Errorf("decode position percentPnl: %w", err)
	}
	if p.TotalBought, err = jsonFloatOrZero(aux.TotalBought); err != nil {
		return fmt.Errorf("decode position totalBought: %w", err)
	}
	if p.RealizedPnl, err = jsonFloatOrZero(aux.RealizedPnl); err != nil {
		return fmt.Errorf("decode position realizedPnl: %w", err)
	}
	if p.PercentRealized, err = jsonFloatOrZero(aux.PercentRealized); err != nil {
		return fmt.Errorf("decode position percentRealizedPnl: %w", err)
	}
	if p.OutcomeIndex, err = jsonIntOrZero(aux.OutcomeIndex); err != nil {
		return fmt.Errorf("decode position outcomeIndex: %w", err)
	}
	return nil
}

type ClosedPosition struct {
	TokenID         string  `json:"asset"`
	ConditionID     string  `json:"conditionId"`
	ProxyWallet     string  `json:"proxyWallet,omitempty"`
	MarketID        string  `json:"market_id,omitempty"`
	Side            string  `json:"side,omitempty"`
	AvgPrice        float64 `json:"avgPrice"`
	AvgPriceBuy     float64 `json:"avg_price_buy,omitempty"`
	AvgPriceSell    float64 `json:"avg_price_sell,omitempty"`
	Size            float64 `json:"size"`
	TotalBought     float64 `json:"totalBought,omitempty"`
	RealizedPnl     float64 `json:"realizedPnl"`
	CurrentPrice    float64 `json:"curPrice,omitempty"`
	Timestamp       string  `json:"timestamp,omitempty"`
	Title           string  `json:"title,omitempty"`
	Slug            string  `json:"slug,omitempty"`
	Icon            string  `json:"icon,omitempty"`
	EventSlug       string  `json:"eventSlug,omitempty"`
	Outcome         string  `json:"outcome,omitempty"`
	OutcomeIndex    int     `json:"outcomeIndex,omitempty"`
	OppositeOutcome string  `json:"oppositeOutcome,omitempty"`
	OppositeAsset   string  `json:"oppositeAsset,omitempty"`
	EndDate         string  `json:"endDate,omitempty"`
}

type Trade struct {
	ID              string  `json:"id"`
	Market          string  `json:"market"`
	AssetID         string  `json:"asset_id"`
	ProxyWallet     string  `json:"proxyWallet,omitempty"`
	Side            string  `json:"side"`
	Price           float64 `json:"price"`
	Size            float64 `json:"size"`
	FeeRateBps      int     `json:"fee_rate_bps"`
	Outcome         string  `json:"outcome,omitempty"`
	OutcomeIndex    int     `json:"outcomeIndex,omitempty"`
	Title           string  `json:"title,omitempty"`
	Slug            string  `json:"slug,omitempty"`
	EventSlug       string  `json:"eventSlug,omitempty"`
	Icon            string  `json:"icon,omitempty"`
	Status          string  `json:"status,omitempty"`
	TransactionHash string  `json:"transaction_hash,omitempty"`
	TakerOrderID    string  `json:"taker_order_id,omitempty"`
	TraderSide      string  `json:"trader_side,omitempty"`
	CreatedAt       string  `json:"created_at"`
}

func (p *ClosedPosition) UnmarshalJSON(data []byte) error {
	var aux struct {
		TokenID          string          `json:"asset"`
		TokenIDSnake     string          `json:"token_id"`
		ConditionID      string          `json:"conditionId"`
		ConditionIDSnake string          `json:"condition_id"`
		ProxyWallet      string          `json:"proxyWallet"`
		MarketID         string          `json:"market_id"`
		Side             string          `json:"side"`
		AvgPrice         json.RawMessage `json:"avgPrice"`
		AvgPriceBuy      json.RawMessage `json:"avg_price_buy"`
		AvgPriceSell     json.RawMessage `json:"avg_price_sell"`
		Size             json.RawMessage `json:"size"`
		TotalBought      json.RawMessage `json:"totalBought"`
		RealizedPnl      json.RawMessage `json:"realizedPnl"`
		RealizedPnlSnake json.RawMessage `json:"realized_pnl"`
		CurrentPrice     json.RawMessage `json:"curPrice"`
		Timestamp        json.RawMessage `json:"timestamp"`
		Title            string          `json:"title"`
		Slug             string          `json:"slug"`
		Icon             string          `json:"icon"`
		EventSlug        string          `json:"eventSlug"`
		Outcome          string          `json:"outcome"`
		OutcomeIndex     json.RawMessage `json:"outcomeIndex"`
		OppositeOutcome  string          `json:"oppositeOutcome"`
		OppositeAsset    string          `json:"oppositeAsset"`
		EndDate          string          `json:"endDate"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	avgPrice, err := jsonFloatOrZero(aux.AvgPrice)
	if err != nil {
		return fmt.Errorf("decode closed position avgPrice: %w", err)
	}
	avgPriceBuy, err := jsonFloatOrZero(aux.AvgPriceBuy)
	if err != nil {
		return fmt.Errorf("decode closed position avg_price_buy: %w", err)
	}
	avgPriceSell, err := jsonFloatOrZero(aux.AvgPriceSell)
	if err != nil {
		return fmt.Errorf("decode closed position avg_price_sell: %w", err)
	}
	size, err := jsonFloatOrZero(aux.Size)
	if err != nil {
		return fmt.Errorf("decode closed position size: %w", err)
	}
	totalBought, err := jsonFloatOrZero(aux.TotalBought)
	if err != nil {
		return fmt.Errorf("decode closed position totalBought: %w", err)
	}
	realizedPnl, err := jsonFloatOrZero(firstRaw(aux.RealizedPnl, aux.RealizedPnlSnake))
	if err != nil {
		return fmt.Errorf("decode closed position realizedPnl: %w", err)
	}
	currentPrice, err := jsonFloatOrZero(aux.CurrentPrice)
	if err != nil {
		return fmt.Errorf("decode closed position curPrice: %w", err)
	}
	outcomeIndex, err := jsonIntOrZero(aux.OutcomeIndex)
	if err != nil {
		return fmt.Errorf("decode closed position outcomeIndex: %w", err)
	}
	if avgPrice == 0 {
		avgPrice = avgPriceBuy
	}
	if size == 0 {
		size = totalBought
	}
	*p = ClosedPosition{
		TokenID:         firstNonEmpty(aux.TokenID, aux.TokenIDSnake),
		ConditionID:     firstNonEmpty(aux.ConditionID, aux.ConditionIDSnake),
		ProxyWallet:     aux.ProxyWallet,
		MarketID:        aux.MarketID,
		Side:            aux.Side,
		AvgPrice:        avgPrice,
		AvgPriceBuy:     avgPriceBuy,
		AvgPriceSell:    avgPriceSell,
		Size:            size,
		TotalBought:     totalBought,
		RealizedPnl:     realizedPnl,
		CurrentPrice:    currentPrice,
		Timestamp:       jsonStringOrNumber(aux.Timestamp),
		Title:           aux.Title,
		Slug:            aux.Slug,
		Icon:            aux.Icon,
		EventSlug:       aux.EventSlug,
		Outcome:         aux.Outcome,
		OutcomeIndex:    outcomeIndex,
		OppositeOutcome: aux.OppositeOutcome,
		OppositeAsset:   aux.OppositeAsset,
		EndDate:         aux.EndDate,
	}
	return nil
}

func (t *Trade) UnmarshalJSON(data []byte) error {
	var aux struct {
		ID                   string          `json:"id"`
		Market               string          `json:"market"`
		ConditionID          string          `json:"conditionId"`
		AssetID              string          `json:"asset_id"`
		AssetIDCamel         string          `json:"assetId"`
		Asset                string          `json:"asset"`
		ProxyWallet          string          `json:"proxyWallet"`
		Side                 string          `json:"side"`
		Price                json.RawMessage `json:"price"`
		Size                 json.RawMessage `json:"size"`
		FeeRateBps           json.RawMessage `json:"fee_rate_bps"`
		FeeRateBpsCamel      json.RawMessage `json:"feeRateBps"`
		Outcome              string          `json:"outcome"`
		OutcomeIndex         json.RawMessage `json:"outcomeIndex"`
		Title                string          `json:"title"`
		Slug                 string          `json:"slug"`
		EventSlug            string          `json:"eventSlug"`
		Icon                 string          `json:"icon"`
		Status               string          `json:"status"`
		TransactionHash      string          `json:"transaction_hash"`
		TransactionHashCamel string          `json:"transactionHash"`
		TakerOrderID         string          `json:"taker_order_id"`
		TakerOrderIDCamel    string          `json:"takerOrderId"`
		TraderSide           string          `json:"trader_side"`
		TraderSideCamel      string          `json:"traderSide"`
		CreatedAt            json.RawMessage `json:"created_at"`
		Timestamp            json.RawMessage `json:"timestamp"`
		MatchTime            json.RawMessage `json:"match_time"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	price, err := jsonFloatOrZero(aux.Price)
	if err != nil {
		return fmt.Errorf("decode trade price: %w", err)
	}
	size, err := jsonFloatOrZero(aux.Size)
	if err != nil {
		return fmt.Errorf("decode trade size: %w", err)
	}
	feeRateBps, err := jsonIntOrZero(firstRaw(aux.FeeRateBps, aux.FeeRateBpsCamel))
	if err != nil {
		return fmt.Errorf("decode trade fee_rate_bps: %w", err)
	}
	outcomeIndex, err := jsonIntOrZero(aux.OutcomeIndex)
	if err != nil {
		return fmt.Errorf("decode trade outcomeIndex: %w", err)
	}
	*t = Trade{
		ID:              aux.ID,
		Market:          firstNonEmpty(aux.Market, aux.ConditionID),
		AssetID:         firstNonEmpty(aux.AssetID, aux.AssetIDCamel, aux.Asset),
		ProxyWallet:     aux.ProxyWallet,
		Side:            aux.Side,
		Price:           price,
		Size:            size,
		FeeRateBps:      feeRateBps,
		Outcome:         aux.Outcome,
		OutcomeIndex:    outcomeIndex,
		Title:           aux.Title,
		Slug:            aux.Slug,
		EventSlug:       aux.EventSlug,
		Icon:            aux.Icon,
		Status:          aux.Status,
		TransactionHash: firstNonEmpty(aux.TransactionHash, aux.TransactionHashCamel),
		TakerOrderID:    firstNonEmpty(aux.TakerOrderID, aux.TakerOrderIDCamel),
		TraderSide:      firstNonEmpty(aux.TraderSide, aux.TraderSideCamel),
		CreatedAt:       firstNonEmpty(jsonStringOrNumber(aux.CreatedAt), jsonStringOrNumber(aux.Timestamp), jsonStringOrNumber(aux.MatchTime)),
	}
	return nil
}

type Activity struct {
	Type      string `json:"type"`
	Market    string `json:"market"`
	AssetID   string `json:"asset_id"`
	Side      string `json:"side"`
	Price     string `json:"price"`
	Size      string `json:"size"`
	Timestamp string `json:"timestamp"`
}

func (a *Activity) UnmarshalJSON(data []byte) error {
	var aux struct {
		Type         string          `json:"type"`
		Market       string          `json:"market"`
		AssetID      string          `json:"asset_id"`
		AssetIDCamel string          `json:"assetId"`
		Side         string          `json:"side"`
		Price        json.RawMessage `json:"price"`
		Size         json.RawMessage `json:"size"`
		Timestamp    json.RawMessage `json:"timestamp"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	a.Type = aux.Type
	a.Market = aux.Market
	a.AssetID = firstNonEmpty(aux.AssetID, aux.AssetIDCamel)
	a.Side = aux.Side
	a.Price = jsonStringOrNumber(aux.Price)
	a.Size = jsonStringOrNumber(aux.Size)
	a.Timestamp = jsonStringOrNumber(aux.Timestamp)
	return nil
}

type MetaHolder struct {
	Address     string  `json:"address"`
	ProxyWallet string  `json:"proxyWallet"`
	Shares      float64 `json:"shares"`
	Amount      float64 `json:"amount"`
	Pnl         float64 `json:"pnl"`
	Volume      float64 `json:"volume"`
}

func (h *MetaHolder) UnmarshalJSON(data []byte) error {
	var aux struct {
		Address     string          `json:"address"`
		ProxyWallet string          `json:"proxyWallet"`
		Shares      json.RawMessage `json:"shares"`
		Amount      json.RawMessage `json:"amount"`
		Pnl         json.RawMessage `json:"pnl"`
		Volume      json.RawMessage `json:"volume"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	var err error
	h.Address = firstNonEmpty(aux.Address, aux.ProxyWallet)
	h.ProxyWallet = firstNonEmpty(aux.ProxyWallet, aux.Address)
	if h.Shares, err = jsonFloatOrZero(firstRaw(aux.Shares, aux.Amount)); err != nil {
		return fmt.Errorf("decode holder shares: %w", err)
	}
	if h.Amount, err = jsonFloatOrZero(firstRaw(aux.Amount, aux.Shares)); err != nil {
		return fmt.Errorf("decode holder amount: %w", err)
	}
	if h.Pnl, err = jsonFloatOrZero(aux.Pnl); err != nil {
		return fmt.Errorf("decode holder pnl: %w", err)
	}
	if h.Volume, err = jsonFloatOrZero(aux.Volume); err != nil {
		return fmt.Errorf("decode holder volume: %w", err)
	}
	return nil
}

type holdersByToken struct {
	Token   string       `json:"token"`
	Holders []MetaHolder `json:"holders"`
}

type TotalValue struct {
	User      string  `json:"user"`
	Value     float64 `json:"value"`
	Timestamp string  `json:"timestamp"`
}

func (t *TotalValue) UnmarshalJSON(data []byte) error {
	var aux struct {
		User      string          `json:"user"`
		Value     json.RawMessage `json:"value"`
		Timestamp json.RawMessage `json:"timestamp"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	value, err := jsonFloatOrZero(aux.Value)
	if err != nil {
		return fmt.Errorf("decode total value: %w", err)
	}
	t.User = aux.User
	t.Value = value
	t.Timestamp = jsonStringOrNumber(aux.Timestamp)
	return nil
}

type TotalMarketsTraded struct {
	User          string `json:"user"`
	MarketsTraded int    `json:"markets_traded"`
	Traded        int    `json:"traded,omitempty"`
}

func (t *TotalMarketsTraded) UnmarshalJSON(data []byte) error {
	var aux struct {
		User          string          `json:"user"`
		MarketsTraded json.RawMessage `json:"markets_traded"`
		Traded        json.RawMessage `json:"traded"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	marketsTraded, err := jsonIntOrZero(aux.MarketsTraded)
	if err != nil {
		return fmt.Errorf("decode markets_traded: %w", err)
	}
	traded, err := jsonIntOrZero(aux.Traded)
	if err != nil {
		return fmt.Errorf("decode traded: %w", err)
	}
	t.User = aux.User
	t.MarketsTraded = marketsTraded
	t.Traded = traded
	return nil
}

type OpenInterest struct {
	Market    string  `json:"market"`
	AssetID   string  `json:"asset_id,omitempty"`
	OpenValue float64 `json:"value"`
}

func (o *OpenInterest) UnmarshalJSON(data []byte) error {
	var aux struct {
		Market         string          `json:"market"`
		AssetID        string          `json:"asset_id"`
		OpenValue      json.RawMessage `json:"value"`
		OpenValueSnake json.RawMessage `json:"open_value"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	openValue, err := jsonFloatOrZero(firstRaw(aux.OpenValue, aux.OpenValueSnake))
	if err != nil {
		return fmt.Errorf("decode open interest value: %w", err)
	}
	o.Market = aux.Market
	o.AssetID = aux.AssetID
	o.OpenValue = openValue
	return nil
}

type TraderLeaderboardEntry struct {
	Rank   int     `json:"rank"`
	User   string  `json:"user"`
	Volume float64 `json:"volume"`
	Pnl    float64 `json:"pnl"`
	ROI    float64 `json:"roi"`
}

type LiveVolumeEntry struct {
	EventID   string  `json:"event_id"`
	EventSlug string  `json:"event_slug"`
	Title     string  `json:"title"`
	Volume    float64 `json:"volume"`
}

func (l *LiveVolumeEntry) UnmarshalJSON(data []byte) error {
	var aux struct {
		EventID   string          `json:"event_id"`
		EventSlug string          `json:"event_slug"`
		Title     string          `json:"title"`
		Volume    json.RawMessage `json:"volume"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	volume, err := jsonFloatOrZero(aux.Volume)
	if err != nil {
		return fmt.Errorf("decode live volume event volume: %w", err)
	}
	l.EventID = aux.EventID
	l.EventSlug = aux.EventSlug
	l.Title = aux.Title
	l.Volume = volume
	return nil
}

type LiveVolumeMarket struct {
	Market string  `json:"market"`
	Value  float64 `json:"value"`
}

func (l *LiveVolumeMarket) UnmarshalJSON(data []byte) error {
	var aux struct {
		Market string          `json:"market"`
		Value  json.RawMessage `json:"value"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	value, err := jsonFloatOrZero(aux.Value)
	if err != nil {
		return fmt.Errorf("decode live volume market value: %w", err)
	}
	l.Market = aux.Market
	l.Value = value
	return nil
}

type LiveVolumeResponse struct {
	Total   float64            `json:"total"`
	Markets []LiveVolumeMarket `json:"markets,omitempty"`
	Events  []LiveVolumeEntry  `json:"events,omitempty"`
}

func (l *LiveVolumeResponse) UnmarshalJSON(data []byte) error {
	var aux struct {
		Total   json.RawMessage    `json:"total"`
		Markets []LiveVolumeMarket `json:"markets"`
		Events  []LiveVolumeEntry  `json:"events"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	total, err := jsonFloatOrZero(aux.Total)
	if err != nil {
		return fmt.Errorf("decode live volume total: %w", err)
	}
	l.Total = total
	l.Markets = aux.Markets
	l.Events = aux.Events
	return nil
}

// --- Methods ---

func (c *Client) Health(ctx context.Context) error {
	return c.transport.Get(ctx, "/", nil)
}

func (c *Client) CurrentPositions(ctx context.Context, user string) ([]Position, error) {
	return c.CurrentPositionsWithLimit(ctx, user, 0)
}

func (c *Client) CurrentPositionsWithLimit(ctx context.Context, user string, limit int) ([]Position, error) {
	params := map[string]string{"user": user}
	if limit > 0 {
		params["limit"] = strconv.Itoa(limit)
	}
	path := buildPath("/positions", params)
	var result []Position
	if err := c.transport.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) ClosedPositions(ctx context.Context, user string) ([]ClosedPosition, error) {
	return c.ClosedPositionsWithLimit(ctx, user, 0)
}

func (c *Client) ClosedPositionsWithLimit(ctx context.Context, user string, limit int) ([]ClosedPosition, error) {
	params := map[string]string{"user": user}
	if limit > 0 {
		params["limit"] = strconv.Itoa(limit)
	}
	path := buildPath("/closed-positions", params)
	var result []ClosedPosition
	if err := c.transport.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) Trades(ctx context.Context, user string, limit int) ([]Trade, error) {
	path := buildPath("/trades", map[string]string{
		"user":  user,
		"limit": strconv.Itoa(limit),
	})
	var result []Trade
	if err := c.transport.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) MarketTrades(ctx context.Context, market string, limit int) ([]Trade, error) {
	path := buildPath("/trades", map[string]string{
		"market": market,
		"limit":  strconv.Itoa(limit),
	})
	var result []Trade
	if err := c.transport.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) Activity(ctx context.Context, user string, limit int) ([]Activity, error) {
	path := buildPath("/activity", map[string]string{
		"user":  user,
		"limit": strconv.Itoa(limit),
	})
	var result []Activity
	if err := c.transport.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) TopHolders(ctx context.Context, market string, limit int) ([]MetaHolder, error) {
	path := buildPath("/holders", map[string]string{
		"market": market,
		"limit":  strconv.Itoa(limit),
	})
	var result []holdersByToken
	if err := c.transport.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	var holders []MetaHolder
	for _, token := range result {
		for _, holder := range token.Holders {
			if holder.Address == "" {
				holder.Address = holder.ProxyWallet
			}
			if holder.Shares == 0 {
				holder.Shares = holder.Amount
			}
			holders = append(holders, holder)
		}
	}
	return holders, nil
}

func (c *Client) TotalValue(ctx context.Context, user string) (*TotalValue, error) {
	path := buildPath("/value", map[string]string{"user": user})
	raw, err := c.transport.GetRaw(ctx, path)
	if err != nil {
		return nil, err
	}
	var result TotalValue
	if err := json.Unmarshal(raw, &result); err == nil {
		if result.User == "" {
			result.User = user
		}
		return &result, nil
	}
	var rows []TotalValue
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return &TotalValue{User: user}, nil
	}
	result = rows[0]
	if result.User == "" {
		result.User = user
	}
	return &result, nil
}

func (c *Client) MarketsTraded(ctx context.Context, user string) (*TotalMarketsTraded, error) {
	path := buildPath("/traded", map[string]string{"user": user})
	var result TotalMarketsTraded
	if err := c.transport.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	if result.MarketsTraded == 0 {
		result.MarketsTraded = result.Traded
	}
	return &result, nil
}

func (c *Client) OpenInterest(ctx context.Context, market string) (*OpenInterest, error) {
	path := buildPath("/oi", map[string]string{"market": market})
	var result []OpenInterest
	if err := c.transport.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return &OpenInterest{Market: market}, nil
	}
	return &result[0], nil
}

func (c *Client) TraderLeaderboard(ctx context.Context, limit int) ([]TraderLeaderboardEntry, error) {
	path := buildPath("/v1/leaderboard", map[string]string{"limit": strconv.Itoa(limit)})
	raw, err := c.transport.GetRaw(ctx, path)
	if err != nil {
		return nil, err
	}
	var rows []leaderboardWire
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, err
	}
	out := make([]TraderLeaderboardEntry, len(rows))
	for i, row := range rows {
		rank, err := parseLeaderboardRank(row.Rank)
		if err != nil {
			return nil, err
		}
		volume, err := jsonFloatOrZero(firstRaw(row.Volume, row.Vol))
		if err != nil {
			return nil, fmt.Errorf("decode leaderboard volume: %w", err)
		}
		pnl, err := jsonFloatOrZero(row.Pnl)
		if err != nil {
			return nil, fmt.Errorf("decode leaderboard pnl: %w", err)
		}
		roi, err := jsonFloatOrZero(row.ROI)
		if err != nil {
			return nil, fmt.Errorf("decode leaderboard roi: %w", err)
		}
		out[i] = TraderLeaderboardEntry{
			Rank:   rank,
			User:   firstNonEmpty(row.User, row.ProxyWallet, row.UserName),
			Volume: volume,
			Pnl:    pnl,
			ROI:    roi,
		}
	}
	return out, nil
}

type leaderboardWire struct {
	Rank        json.RawMessage `json:"rank"`
	User        string          `json:"user"`
	ProxyWallet string          `json:"proxyWallet"`
	UserName    string          `json:"userName"`
	Volume      json.RawMessage `json:"volume"`
	Vol         json.RawMessage `json:"vol"`
	Pnl         json.RawMessage `json:"pnl"`
	ROI         json.RawMessage `json:"roi"`
}

func parseLeaderboardRank(raw json.RawMessage) (int, error) {
	if len(raw) == 0 {
		return 0, nil
	}
	var n int
	if err := json.Unmarshal(raw, &n); err == nil {
		return n, nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return 0, fmt.Errorf("decode leaderboard rank: %w", err)
	}
	if s == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("decode leaderboard rank %q: %w", s, err)
	}
	return n, nil
}

func firstNonEmpty(values ...string) string {
	return jsonx.FirstString(values...)
}

func firstRaw(values ...json.RawMessage) json.RawMessage {
	return jsonx.FirstRaw(values...)
}

func jsonFloatOrZero(raw json.RawMessage) (float64, error) {
	value := jsonStringOrNumber(raw)
	if value == "" {
		return 0, nil
	}
	n, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func jsonIntOrZero(raw json.RawMessage) (int, error) {
	value := jsonStringOrNumber(raw)
	if value == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func jsonBoolOrFalse(raw json.RawMessage) bool {
	switch strings.ToLower(jsonStringOrNumber(raw)) {
	case "true", "1":
		return true
	default:
		return false
	}
}

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

func (c *Client) LiveVolume(ctx context.Context, eventID int) (*LiveVolumeResponse, error) {
	path := buildPath("/live-volume", map[string]string{"id": strconv.Itoa(eventID)})
	raw, err := c.transport.GetRaw(ctx, path)
	if err != nil {
		return nil, err
	}
	var result LiveVolumeResponse
	if err := json.Unmarshal(raw, &result); err == nil {
		return &result, nil
	}
	var rows []LiveVolumeResponse
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return &LiveVolumeResponse{}, nil
	}
	result = rows[0]
	return &result, nil
}

func buildPath(base string, params map[string]string) string {
	q := url.Values{}
	for k, v := range params {
		q.Set(k, v)
	}
	if len(q) > 0 {
		return base + "?" + q.Encode()
	}
	return base
}
