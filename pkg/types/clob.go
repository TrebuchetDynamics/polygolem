package types

import (
	"encoding/json"
	"strconv"

	"github.com/TrebuchetDynamics/polygolem/internal/jsonx"
)

// CLOBServerTime is the CLOB server-time response.
type CLOBServerTime struct {
	Timestamp string `json:"timestamp"`
	ISO       string `json:"iso"`
}

// CLOBOrderBookLevel is one price level in a CLOB order book.
type CLOBOrderBookLevel struct {
	Price string `json:"price"`
	Size  string `json:"size"`
}

func (l *CLOBOrderBookLevel) UnmarshalJSON(b []byte) error {
	var raw struct {
		Price json.RawMessage `json:"price"`
		Size  json.RawMessage `json:"size"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	l.Price = clobStringOrNumber(raw.Price)
	l.Size = clobStringOrNumber(raw.Size)
	return nil
}

// CLOBOrderBook is a public CLOB order-book snapshot for one outcome token.
type CLOBOrderBook struct {
	Market         string               `json:"market"`
	AssetID        string               `json:"asset_id"`
	Timestamp      string               `json:"timestamp"`
	Hash           string               `json:"hash"`
	Bids           []CLOBOrderBookLevel `json:"bids"`
	Asks           []CLOBOrderBookLevel `json:"asks"`
	MinOrderSize   string               `json:"min_order_size,omitempty"`
	TickSize       string               `json:"tick_size,omitempty"`
	NegRisk        bool                 `json:"neg_risk,omitempty"`
	LastTradePrice string               `json:"last_trade_price,omitempty"`
}

func (o *CLOBOrderBook) UnmarshalJSON(b []byte) error {
	var raw struct {
		Market              string               `json:"market"`
		AssetID             json.RawMessage      `json:"asset_id"`
		AssetIDCamel        json.RawMessage      `json:"assetId"`
		Timestamp           json.RawMessage      `json:"timestamp"`
		Hash                string               `json:"hash"`
		Bids                []CLOBOrderBookLevel `json:"bids"`
		Asks                []CLOBOrderBookLevel `json:"asks"`
		MinOrderSize        json.RawMessage      `json:"min_order_size"`
		MinOrderSizeCamel   json.RawMessage      `json:"minOrderSize"`
		TickSize            json.RawMessage      `json:"tick_size"`
		TickSizeCamel       json.RawMessage      `json:"tickSize"`
		NegRisk             json.RawMessage      `json:"neg_risk"`
		NegRiskCamel        json.RawMessage      `json:"negRisk"`
		LastTradePrice      json.RawMessage      `json:"last_trade_price"`
		LastTradePriceCamel json.RawMessage      `json:"lastTradePrice"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	o.Market = raw.Market
	o.AssetID = firstNonEmptyClobString(clobStringOrNumber(raw.AssetID), clobStringOrNumber(raw.AssetIDCamel))
	o.Timestamp = clobStringOrNumber(raw.Timestamp)
	o.Hash = raw.Hash
	o.Bids = raw.Bids
	o.Asks = raw.Asks
	o.MinOrderSize = firstNonEmptyClobString(clobStringOrNumber(raw.MinOrderSize), clobStringOrNumber(raw.MinOrderSizeCamel))
	o.TickSize = firstNonEmptyClobString(clobStringOrNumber(raw.TickSize), clobStringOrNumber(raw.TickSizeCamel))
	o.NegRisk = clobBoolOrFalse(firstNonEmptyClobRaw(raw.NegRisk, raw.NegRiskCamel))
	o.LastTradePrice = firstNonEmptyClobString(clobStringOrNumber(raw.LastTradePrice), clobStringOrNumber(raw.LastTradePriceCamel))
	return nil
}

// CLOBTickSize is the minimum size and price increment metadata for a token.
type CLOBTickSize struct {
	MinimumTickSize  string `json:"minimum_tick_size"`
	MinimumOrderSize string `json:"minimum_order_size"`
	TickSize         string `json:"tick_size"`
}

func (t *CLOBTickSize) UnmarshalJSON(b []byte) error {
	var raw struct {
		MinimumTickSize       json.RawMessage `json:"minimum_tick_size"`
		MinimumTickSizeCamel  json.RawMessage `json:"minimumTickSize"`
		MinimumOrderSize      json.RawMessage `json:"minimum_order_size"`
		MinimumOrderSizeCamel json.RawMessage `json:"minimumOrderSize"`
		TickSize              json.RawMessage `json:"tick_size"`
		TickSizeCamel         json.RawMessage `json:"tickSize"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	t.MinimumTickSize = firstNonEmptyClobString(clobStringOrNumber(raw.MinimumTickSize), clobStringOrNumber(raw.MinimumTickSizeCamel))
	t.MinimumOrderSize = firstNonEmptyClobString(clobStringOrNumber(raw.MinimumOrderSize), clobStringOrNumber(raw.MinimumOrderSizeCamel))
	t.TickSize = firstNonEmptyClobString(clobStringOrNumber(raw.TickSize), clobStringOrNumber(raw.TickSizeCamel))
	return nil
}

// CLOBNegRiskInfo is negative-risk metadata for a token.
type CLOBNegRiskInfo struct {
	NegRisk         bool   `json:"neg_risk"`
	NegRiskMarketID string `json:"neg_risk_market_id,omitempty"`
	NegRiskFeeBips  int    `json:"neg_risk_fee_bips,omitempty"`
}

func (n *CLOBNegRiskInfo) UnmarshalJSON(b []byte) error {
	var raw struct {
		NegRisk              json.RawMessage `json:"neg_risk"`
		NegRiskCamel         json.RawMessage `json:"negRisk"`
		NegRiskMarketID      json.RawMessage `json:"neg_risk_market_id"`
		NegRiskMarketIDCamel json.RawMessage `json:"negRiskMarketID"`
		NegRiskFeeBips       json.RawMessage `json:"neg_risk_fee_bips"`
		NegRiskFeeBipsCamel  json.RawMessage `json:"negRiskFeeBips"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	n.NegRisk = clobBoolOrFalse(firstNonEmptyClobRaw(raw.NegRisk, raw.NegRiskCamel))
	n.NegRiskMarketID = firstNonEmptyClobString(clobStringOrNumber(raw.NegRiskMarketID), clobStringOrNumber(raw.NegRiskMarketIDCamel))
	n.NegRiskFeeBips, _ = strconv.Atoi(firstNonEmptyClobString(clobStringOrNumber(raw.NegRiskFeeBips), clobStringOrNumber(raw.NegRiskFeeBipsCamel)))
	return nil
}

// CLOBFeeDetails is the fee curve metadata returned by CLOB market info.
type CLOBFeeDetails struct {
	Rate      float64 `json:"rate,omitempty"`
	Exponent  float64 `json:"exponent,omitempty"`
	TakerOnly bool    `json:"taker_only,omitempty"`
}

func (f *CLOBFeeDetails) UnmarshalJSON(b []byte) error {
	var raw struct {
		Rate           json.RawMessage `json:"rate"`
		RateShort      json.RawMessage `json:"r"`
		Exponent       json.RawMessage `json:"exponent"`
		ExponentShort  json.RawMessage `json:"e"`
		TakerOnly      json.RawMessage `json:"taker_only"`
		TakerOnlyCamel json.RawMessage `json:"takerOnly"`
		TakerOnlyShort json.RawMessage `json:"to"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	f.Rate, _ = strconv.ParseFloat(firstNonEmptyClobString(clobStringOrNumber(raw.Rate), clobStringOrNumber(raw.RateShort)), 64)
	f.Exponent, _ = strconv.ParseFloat(firstNonEmptyClobString(clobStringOrNumber(raw.Exponent), clobStringOrNumber(raw.ExponentShort)), 64)
	f.TakerOnly = clobBoolOrFalse(firstNonEmptyClobRaw(raw.TakerOnly, raw.TakerOnlyCamel, raw.TakerOnlyShort))
	return nil
}

// CLOBMarket is a market from the CLOB API.
type CLOBMarket struct {
	ConditionID           string         `json:"condition_id"`
	QuestionID            string         `json:"question_id"`
	Tokens                []CLOBToken    `json:"tokens"`
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

func clobStringOrNumber(raw json.RawMessage) string {
	return jsonx.StringOrNumber(raw)
}

func firstNonEmptyClobString(values ...string) string {
	return jsonx.FirstString(values...)
}

func firstNonEmptyClobRaw(values ...json.RawMessage) json.RawMessage {
	return jsonx.FirstRaw(values...)
}

func clobBoolOrFalse(raw json.RawMessage) bool {
	return jsonx.BoolOrFalse(raw)
}

func (m *CLOBMarket) UnmarshalJSON(b []byte) error {
	type alias CLOBMarket
	var raw struct {
		alias
		ConditionIDShort   string `json:"c"`
		QuestionIDShort    string `json:"q"`
		GameStartTimeShort string `json:"gst"`
		TokensShort        []struct {
			TokenID string `json:"t"`
			Outcome string `json:"o"`
			Price   string `json:"p"`
			Winner  bool   `json:"w"`
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
	if m.QuestionID == "" {
		m.QuestionID = raw.QuestionIDShort
	}
	if m.GameStartTime == "" {
		m.GameStartTime = raw.GameStartTimeShort
	}
	if len(m.Tokens) == 0 && len(raw.TokensShort) > 0 {
		m.Tokens = make([]CLOBToken, len(raw.TokensShort))
		for i, token := range raw.TokensShort {
			m.Tokens[i] = CLOBToken{
				TokenID: token.TokenID,
				Outcome: token.Outcome,
				Price:   token.Price,
				Winner:  token.Winner,
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

// CLOBMarketByTokenResponse resolves a CLOB token ID to its parent market.
type CLOBMarketByTokenResponse struct {
	ConditionID      string `json:"condition_id"`
	PrimaryTokenID   string `json:"primary_token_id"`
	SecondaryTokenID string `json:"secondary_token_id"`
}

// CLOBToken is an outcome token listed on a CLOB market.
type CLOBToken struct {
	TokenID string `json:"token_id"`
	Outcome string `json:"outcome"`
	Price   string `json:"price"`
	Winner  bool   `json:"winner"`
}

// CLOBPaginatedMarkets is the cursor-paginated CLOB market-list response.
type CLOBPaginatedMarkets struct {
	Limit      int          `json:"limit"`
	Count      int          `json:"count"`
	NextCursor string       `json:"next_cursor"`
	Data       []CLOBMarket `json:"data"`
}

// CLOBBookParams identifies one token, and optionally a side, for batch CLOB
// book and price requests.
type CLOBBookParams struct {
	TokenID string `json:"token_id"`
	Side    string `json:"side,omitempty"`
}

// CLOBPricePoint is one price-history point.
type CLOBPricePoint struct {
	T        string `json:"t"`
	P        string `json:"p"`
	Volume   string `json:"v,omitempty"`
	Interval string `json:"interval,omitempty"`
}

func (p *CLOBPricePoint) UnmarshalJSON(b []byte) error {
	var raw struct {
		T              json.RawMessage `json:"t"`
		Timestamp      json.RawMessage `json:"timestamp"`
		P              json.RawMessage `json:"p"`
		Price          json.RawMessage `json:"price"`
		Volume         json.RawMessage `json:"v"`
		VolumeLongName json.RawMessage `json:"volume"`
		Interval       string          `json:"interval"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	p.T = firstNonEmptyClobString(clobStringOrNumber(raw.T), clobStringOrNumber(raw.Timestamp))
	p.P = firstNonEmptyClobString(clobStringOrNumber(raw.P), clobStringOrNumber(raw.Price))
	p.Volume = firstNonEmptyClobString(clobStringOrNumber(raw.Volume), clobStringOrNumber(raw.VolumeLongName))
	p.Interval = raw.Interval
	return nil
}

// CLOBPriceHistory is the CLOB price-history response.
type CLOBPriceHistory struct {
	History []CLOBPricePoint `json:"history"`
}

// CLOBPriceHistoryParams filters CLOB price-history requests.
type CLOBPriceHistoryParams struct {
	Market   string `json:"market,omitempty"`
	Interval string `json:"interval,omitempty"`
	Fidelity int    `json:"fidelity,omitempty"`
	StartTS  int64  `json:"start_ts,omitempty"`
	EndTS    int64  `json:"end_ts,omitempty"`
}

// CLOBMarketOutcomeStatus describes the resolution state of a CLOB market.
type CLOBMarketOutcomeStatus string

const (
	CLOBOutcomeResolved   CLOBMarketOutcomeStatus = "resolved"
	CLOBOutcomeUnresolved CLOBMarketOutcomeStatus = "unresolved"
)

// CLOBMarketOutcome is the result of resolving a market's outcome.
type CLOBMarketOutcome struct {
	Status         CLOBMarketOutcomeStatus `json:"status"`
	ConditionID    string                  `json:"condition_id"`
	WinningTokenID string                  `json:"winning_token_id,omitempty"`
	Closed         bool                    `json:"closed"`
	Source         string                  `json:"source"`
}
