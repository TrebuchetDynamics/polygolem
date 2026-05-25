package polytypes

import (
	"encoding/json"
	"strconv"
	"strings"
)

// OrderBookLevel is a single price level in the order book.
type OrderBookLevel struct {
	Price string `json:"price"`
	Size  string `json:"size"`
}

func (l *OrderBookLevel) UnmarshalJSON(b []byte) error {
	var raw struct {
		Price NumericString `json:"price"`
		Size  NumericString `json:"size"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	l.Price = string(raw.Price)
	l.Size = string(raw.Size)
	return nil
}

// OrderBook represents L2 order book depth for a token.
type OrderBook struct {
	Market         string           `json:"market"`
	AssetID        string           `json:"asset_id"`
	Timestamp      string           `json:"timestamp"`
	Hash           string           `json:"hash"`
	Bids           []OrderBookLevel `json:"bids"`
	Asks           []OrderBookLevel `json:"asks"`
	MinOrderSize   string           `json:"min_order_size,omitempty"`
	TickSize       string           `json:"tick_size,omitempty"`
	NegRisk        bool             `json:"neg_risk,omitempty"`
	LastTradePrice string           `json:"last_trade_price,omitempty"`
}

func (o *OrderBook) UnmarshalJSON(b []byte) error {
	var raw struct {
		Market              string           `json:"market"`
		AssetID             NumericString    `json:"asset_id"`
		AssetIDCamel        NumericString    `json:"assetId"`
		Timestamp           NumericString    `json:"timestamp"`
		Hash                string           `json:"hash"`
		Bids                []OrderBookLevel `json:"bids"`
		Asks                []OrderBookLevel `json:"asks"`
		MinOrderSize        NumericString    `json:"min_order_size"`
		MinOrderSizeCamel   NumericString    `json:"minOrderSize"`
		TickSize            NumericString    `json:"tick_size"`
		TickSizeCamel       NumericString    `json:"tickSize"`
		NegRisk             json.RawMessage  `json:"neg_risk"`
		NegRiskCamel        json.RawMessage  `json:"negRisk"`
		LastTradePrice      NumericString    `json:"last_trade_price"`
		LastTradePriceCamel NumericString    `json:"lastTradePrice"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	o.Market = raw.Market
	o.AssetID = firstNonEmptyString(string(raw.AssetID), string(raw.AssetIDCamel))
	o.Timestamp = string(raw.Timestamp)
	o.Hash = raw.Hash
	o.Bids = raw.Bids
	o.Asks = raw.Asks
	o.MinOrderSize = firstNonEmptyString(string(raw.MinOrderSize), string(raw.MinOrderSizeCamel))
	o.TickSize = firstNonEmptyString(string(raw.TickSize), string(raw.TickSizeCamel))
	o.NegRisk = jsonBoolOrFalse(firstNonEmptyRaw(raw.NegRisk, raw.NegRiskCamel))
	o.LastTradePrice = firstNonEmptyString(string(raw.LastTradePrice), string(raw.LastTradePriceCamel))
	return nil
}

// TickSize represents the minimum tick size for a market.
type TickSize struct {
	MinimumTickSize  string `json:"minimum_tick_size"`
	MinimumOrderSize string `json:"minimum_order_size"`
	TickSize         string `json:"tick_size"`
}

func (t *TickSize) UnmarshalJSON(b []byte) error {
	var raw struct {
		MinimumTickSize       NumericString `json:"minimum_tick_size"`
		MinimumTickSizeCamel  NumericString `json:"minimumTickSize"`
		MinimumOrderSize      NumericString `json:"minimum_order_size"`
		MinimumOrderSizeCamel NumericString `json:"minimumOrderSize"`
		TickSize              NumericString `json:"tick_size"`
		TickSizeCamel         NumericString `json:"tickSize"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	t.MinimumTickSize = firstNonEmptyString(string(raw.MinimumTickSize), string(raw.MinimumTickSizeCamel))
	t.MinimumOrderSize = firstNonEmptyString(string(raw.MinimumOrderSize), string(raw.MinimumOrderSizeCamel))
	t.TickSize = firstNonEmptyString(string(raw.TickSize), string(raw.TickSizeCamel))
	return nil
}

// NegRiskInfo represents negative risk market info.
type NegRiskInfo struct {
	NegRisk         bool   `json:"neg_risk"`
	NegRiskMarketID string `json:"neg_risk_market_id,omitempty"`
	NegRiskFeeBips  int    `json:"neg_risk_fee_bips,omitempty"`
}

func (n *NegRiskInfo) UnmarshalJSON(b []byte) error {
	var raw struct {
		NegRisk              json.RawMessage `json:"neg_risk"`
		NegRiskCamel         json.RawMessage `json:"negRisk"`
		NegRiskMarketID      NumericString   `json:"neg_risk_market_id"`
		NegRiskMarketIDCamel NumericString   `json:"negRiskMarketID"`
		NegRiskFeeBips       NumericString   `json:"neg_risk_fee_bips"`
		NegRiskFeeBipsCamel  NumericString   `json:"negRiskFeeBips"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	n.NegRisk = jsonBoolOrFalse(firstNonEmptyRaw(raw.NegRisk, raw.NegRiskCamel))
	n.NegRiskMarketID = firstNonEmptyString(string(raw.NegRiskMarketID), string(raw.NegRiskMarketIDCamel))
	n.NegRiskFeeBips, _ = strconv.Atoi(firstNonEmptyString(string(raw.NegRiskFeeBips), string(raw.NegRiskFeeBipsCamel)))
	return nil
}

// FeeRate represents the fee rate in basis points.
type FeeRate struct {
	FeeRateBps int `json:"fee_rate_bps"`
}

func (f *FeeRate) UnmarshalJSON(b []byte) error {
	var raw struct {
		FeeRateBps      NumericString `json:"fee_rate_bps"`
		FeeRateBpsCamel NumericString `json:"feeRateBps"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	f.FeeRateBps, _ = strconv.Atoi(firstNonEmptyString(string(raw.FeeRateBps), string(raw.FeeRateBpsCamel)))
	return nil
}

// PricePoint represents a single price history data point.
type PricePoint struct {
	T        string `json:"t"` // timestamp
	P        string `json:"p"` // price
	Volume   string `json:"v,omitempty"`
	Interval string `json:"interval,omitempty"`
}

func (p *PricePoint) UnmarshalJSON(b []byte) error {
	var raw struct {
		T              NumericString `json:"t"`
		Timestamp      NumericString `json:"timestamp"`
		P              NumericString `json:"p"`
		Price          NumericString `json:"price"`
		Volume         NumericString `json:"v"`
		VolumeLongName NumericString `json:"volume"`
		Interval       string        `json:"interval"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	p.T = firstNonEmptyString(string(raw.T), string(raw.Timestamp))
	p.P = firstNonEmptyString(string(raw.P), string(raw.Price))
	p.Volume = firstNonEmptyString(string(raw.Volume), string(raw.VolumeLongName))
	p.Interval = raw.Interval
	return nil
}

// PriceHistory represents OHLCV price history.
type PriceHistory struct {
	History []PricePoint `json:"history"`
}

// CLOBFeeDetails is the fee curve metadata returned by CLOB market info.
type CLOBFeeDetails struct {
	Rate      float64 `json:"rate,omitempty"`
	Exponent  float64 `json:"exponent,omitempty"`
	TakerOnly bool    `json:"taker_only,omitempty"`
}

func (f *CLOBFeeDetails) UnmarshalJSON(b []byte) error {
	var raw struct {
		Rate           NumericString   `json:"rate"`
		RateShort      NumericString   `json:"r"`
		Exponent       NumericString   `json:"exponent"`
		ExponentShort  NumericString   `json:"e"`
		TakerOnly      json.RawMessage `json:"taker_only"`
		TakerOnlyCamel json.RawMessage `json:"takerOnly"`
		TakerOnlyShort json.RawMessage `json:"to"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	f.Rate, _ = strconv.ParseFloat(firstNonEmptyString(string(raw.Rate), string(raw.RateShort)), 64)
	f.Exponent, _ = strconv.ParseFloat(firstNonEmptyString(string(raw.Exponent), string(raw.ExponentShort)), 64)
	f.TakerOnly = jsonBoolOrFalse(firstNonEmptyRaw(raw.TakerOnly, raw.TakerOnlyCamel, raw.TakerOnlyShort))
	return nil
}

// CLOBMarket represents a market from the CLOB API.
type CLOBMarket struct {
	ConditionID           string         `json:"condition_id"`
	QuestionID            string         `json:"question_id"`
	Tokens                []Token        `json:"tokens"`
	GameStartTime         string         `json:"game_start_time,omitempty"`
	RewardsMinSize        float64        `json:"rewards_min_size"`
	RewardsMaxSpread      float64        `json:"rewards_max_spread"`
	Spread                float64        `json:"spread"`
	EnableOrderBook       bool           `json:"enable_order_book"`
	OrderPriceMinTickSize float64        `json:"order_price_min_tick_size"`
	OrderMinSize          float64        `json:"order_min_size"`
	Closed                bool           `json:"closed"`
	Archived              bool           `json:"archived"`
	AcceptingOrders       bool           `json:"accepting_orders"`
	NegRisk               bool           `json:"neg_risk"`
	NegRiskMarketID       string         `json:"neg_risk_market_id,omitempty"`
	NegRiskRequestID      string         `json:"neg_risk_request_id,omitempty"`
	MakerBaseFee          int            `json:"maker_base_fee"`
	TakerBaseFee          int            `json:"taker_base_fee"`
	NotificationsEnabled  bool           `json:"notifications_enabled"`
	RFQEnabled            bool           `json:"rfq_enabled,omitempty"`
	TakerOrderDelay       bool           `json:"taker_order_delay,omitempty"`
	BlockaidCheckEnabled  bool           `json:"blockaid_check_enabled,omitempty"`
	FeeDetails            CLOBFeeDetails `json:"fee_details,omitempty"`
	MinimumOrderAge       int            `json:"minimum_order_age,omitempty"`
}

// CLOBMarketByTokenResponse resolves a CLOB token ID to its parent market.
type CLOBMarketByTokenResponse struct {
	ConditionID      string `json:"condition_id"`
	PrimaryTokenID   string `json:"primary_token_id"`
	SecondaryTokenID string `json:"secondary_token_id"`
}

func (m *CLOBMarket) UnmarshalJSON(b []byte) error {
	type alias CLOBMarket
	var raw struct {
		alias
		ConditionIDShort   string `json:"c"`
		GameStartTimeShort string `json:"gst"`
		TokensShort        []struct {
			TokenID string `json:"t"`
			Outcome string `json:"o"`
		} `json:"t"`
		RewardsShort *struct {
			MinSize         *float64 `json:"mi"`
			MaxSpread       *float64 `json:"ma"`
			MinimumOrderAge *int     `json:"moas"`
		} `json:"r"`
		OrderMinSizeShort          *float64        `json:"mos"`
		OrderPriceMinTickSizeShort *float64        `json:"mts"`
		MakerBaseFeeShort          *int            `json:"mbf"`
		TakerBaseFeeShort          *int            `json:"tbf"`
		AcceptingOrdersShort       *bool           `json:"ao"`
		EnableOrderBookShort       *bool           `json:"cbos"`
		NegRiskShort               *bool           `json:"nr"`
		RFQEnabledShort            *bool           `json:"rfqe"`
		TakerOrderDelayShort       *bool           `json:"itode"`
		BlockaidCheckEnabledShort  *bool           `json:"ibce"`
		FeeDetailsCamel            *CLOBFeeDetails `json:"feeDetails"`
		FeeDetailsShort            *CLOBFeeDetails `json:"fd"`
		MinimumOrderAgeShort       *int            `json:"oas"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	*m = CLOBMarket(raw.alias)
	if m.ConditionID == "" {
		m.ConditionID = raw.ConditionIDShort
	}
	if m.GameStartTime == "" {
		m.GameStartTime = raw.GameStartTimeShort
	}
	if len(m.Tokens) == 0 && len(raw.TokensShort) > 0 {
		m.Tokens = make([]Token, len(raw.TokensShort))
		for i, token := range raw.TokensShort {
			m.Tokens[i] = Token{
				TokenID: token.TokenID,
				Outcome: token.Outcome,
			}
		}
	}
	if raw.RewardsShort != nil {
		if raw.RewardsShort.MinSize != nil {
			m.RewardsMinSize = *raw.RewardsShort.MinSize
		}
		if raw.RewardsShort.MaxSpread != nil {
			m.RewardsMaxSpread = *raw.RewardsShort.MaxSpread
		}
		if raw.RewardsShort.MinimumOrderAge != nil {
			m.MinimumOrderAge = *raw.RewardsShort.MinimumOrderAge
		}
	}
	if raw.OrderMinSizeShort != nil {
		m.OrderMinSize = *raw.OrderMinSizeShort
	}
	if raw.OrderPriceMinTickSizeShort != nil {
		m.OrderPriceMinTickSize = *raw.OrderPriceMinTickSizeShort
	}
	if raw.MakerBaseFeeShort != nil {
		m.MakerBaseFee = *raw.MakerBaseFeeShort
	}
	if raw.TakerBaseFeeShort != nil {
		m.TakerBaseFee = *raw.TakerBaseFeeShort
	}
	if raw.AcceptingOrdersShort != nil {
		m.AcceptingOrders = *raw.AcceptingOrdersShort
	}
	if raw.EnableOrderBookShort != nil {
		m.EnableOrderBook = *raw.EnableOrderBookShort
	}
	if raw.NegRiskShort != nil {
		m.NegRisk = *raw.NegRiskShort
	}
	if raw.RFQEnabledShort != nil {
		m.RFQEnabled = *raw.RFQEnabledShort
	}
	if raw.TakerOrderDelayShort != nil {
		m.TakerOrderDelay = *raw.TakerOrderDelayShort
	}
	if raw.BlockaidCheckEnabledShort != nil {
		m.BlockaidCheckEnabled = *raw.BlockaidCheckEnabledShort
	}
	if raw.FeeDetailsCamel != nil {
		m.FeeDetails = *raw.FeeDetailsCamel
	}
	if raw.FeeDetailsShort != nil {
		m.FeeDetails = *raw.FeeDetailsShort
	}
	if raw.MinimumOrderAgeShort != nil {
		m.MinimumOrderAge = *raw.MinimumOrderAgeShort
	}
	return nil
}

// Token represents a CLOB outcome token.
type Token struct {
	TokenID string        `json:"token_id"`
	Outcome string        `json:"outcome"`
	Price   NumericString `json:"price"`
	Winner  bool          `json:"winner"`
}

// NumericString preserves CLOB fields that may be encoded as JSON strings or
// numbers while keeping downstream order math string-based.
type NumericString string

func (s *NumericString) UnmarshalJSON(b []byte) error {
	raw := strings.TrimSpace(string(b))
	if raw == "" || raw == "null" {
		*s = ""
		return nil
	}
	var asString string
	if err := json.Unmarshal(b, &asString); err == nil {
		*s = NumericString(strings.TrimSpace(asString))
		return nil
	}
	*s = NumericString(raw)
	return nil
}

func (s NumericString) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(s))
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func firstNonEmptyRaw(values ...json.RawMessage) json.RawMessage {
	for _, value := range values {
		if len(value) == 0 || strings.TrimSpace(string(value)) == "null" || strings.TrimSpace(string(value)) == "" {
			continue
		}
		return value
	}
	return nil
}

func jsonBoolOrFalse(raw json.RawMessage) bool {
	text := strings.Trim(strings.ToLower(strings.TrimSpace(string(raw))), "\"")
	return text == "true" || text == "1"
}

// CLOBPaginatedMarkets represents cursor-paginated CLOB markets.
type CLOBPaginatedMarkets struct {
	Limit      int          `json:"limit"`
	Count      int          `json:"count"`
	NextCursor string       `json:"next_cursor"`
	Data       []CLOBMarket `json:"data"`
}

// BookParams represents parameters for batch order book requests.
type BookParams struct {
	TokenID string `json:"token_id"`
	Side    string `json:"side,omitempty"` // BUY or SELL (for price requests)
}

// PriceHistoryParams represents parameters for price history requests.
type PriceHistoryParams struct {
	Market   string `json:"market,omitempty"`
	Interval string `json:"interval,omitempty"` // 1m, 1h, 6h, 1d, 1w, max
	Fidelity int    `json:"fidelity,omitempty"`
	StartTS  int64  `json:"start_ts,omitempty"`
	EndTS    int64  `json:"end_ts,omitempty"`
}

// MidpointResponse represents a midpoint price.
type MidpointResponse struct {
	Midpoint string `json:"mid"`
}

// PriceResponse represents a price/spread response.
type PriceResponse struct {
	Price  string `json:"price"`
	Spread string `json:"spread,omitempty"`
}

// ServerTime represents the server time response.
type ServerTime struct {
	Timestamp string `json:"timestamp"`
	ISO       string `json:"iso"`
}

// EnrichedMarket joins Gamma metadata with CLOB details.
type EnrichedMarket struct {
	// From Gamma
	Market Market `json:"market"`
	// From CLOB
	TickSize   TickSize `json:"tick_size"`
	NegRisk    bool     `json:"neg_risk"`
	FeeRateBps int      `json:"fee_rate_bps"`
	// Optional
	OrderBook *OrderBook `json:"order_book,omitempty"`
	LastPrice string     `json:"last_price,omitempty"`
	Midpoint  string     `json:"midpoint,omitempty"`
	Spread    string     `json:"spread,omitempty"`
}
