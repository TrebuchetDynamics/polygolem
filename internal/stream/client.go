package stream

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// Config holds WebSocket connection configuration.
type Config struct {
	URL                  string
	PingInterval         time.Duration
	PongTimeout          time.Duration
	Reconnect            bool
	ReconnectDelay       time.Duration
	ReconnectMaxDelay    time.Duration
	ReconnectMax         int
	Level                int
	CustomFeatureEnabled bool
}

// DefaultConfig returns sensible defaults.
func DefaultConfig(url string) Config {
	return Config{
		URL:               url,
		PingInterval:      10 * time.Second,
		PongTimeout:       30 * time.Second,
		Reconnect:         true,
		ReconnectDelay:    2 * time.Second,
		ReconnectMaxDelay: 30 * time.Second,
		ReconnectMax:      5,
	}
}

// BookMessage is a WebSocket order book event.
type BookMessage struct {
	EventType string       `json:"event_type"`
	AssetID   string       `json:"asset_id"`
	Market    string       `json:"market"`
	Timestamp string       `json:"timestamp"`
	Hash      string       `json:"hash"`
	Bids      []PriceLevel `json:"bids"`
	Asks      []PriceLevel `json:"asks"`
}

// PriceLevel is a single price level in a WebSocket book message.
type PriceLevel struct {
	Price string `json:"price"`
	Size  string `json:"size"`
}

// PriceChangeMessage is a WebSocket price change event.
type PriceChangeMessage struct {
	EventType string             `json:"event_type"`
	Market    string             `json:"market"`
	Changes   []PriceChangeEntry `json:"price_changes"`
	Timestamp string             `json:"timestamp"`
}

// PriceChangeEntry is a single price change.
type PriceChangeEntry struct {
	AssetID string `json:"asset_id"`
	Price   string `json:"price"`
	Side    string `json:"side"`
	Size    string `json:"size"`
	Hash    string `json:"hash"`
	BestBid string `json:"best_bid,omitempty"`
	BestAsk string `json:"best_ask,omitempty"`
}

// LastTradeMessage is a WebSocket last trade event.
type LastTradeMessage struct {
	EventType       string `json:"event_type"`
	AssetID         string `json:"asset_id"`
	Market          string `json:"market"`
	Price           string `json:"price"`
	Side            string `json:"side"`
	Size            string `json:"size"`
	FeeRateBps      string `json:"fee_rate_bps"`
	Timestamp       string `json:"timestamp"`
	TransactionHash string `json:"transaction_hash,omitempty"`
}

// TickSizeChangeMessage is a WebSocket tick-size update event.
type TickSizeChangeMessage struct {
	EventType   string `json:"event_type"`
	AssetID     string `json:"asset_id"`
	Market      string `json:"market"`
	OldTickSize string `json:"old_tick_size"`
	NewTickSize string `json:"new_tick_size"`
	Timestamp   string `json:"timestamp"`
}

// BestBidAskMessage is a top-of-book update event.
type BestBidAskMessage struct {
	EventType string `json:"event_type"`
	AssetID   string `json:"asset_id"`
	Market    string `json:"market"`
	BestBid   string `json:"best_bid"`
	BestAsk   string `json:"best_ask"`
	Spread    string `json:"spread"`
	Timestamp string `json:"timestamp"`
}

// NewMarketMessage is a market lifecycle creation event.
type NewMarketMessage struct {
	EventType             string                 `json:"event_type"`
	ID                    string                 `json:"id"`
	Question              string                 `json:"question"`
	Market                string                 `json:"market"`
	Slug                  string                 `json:"slug"`
	Description           string                 `json:"description"`
	AssetIDs              []string               `json:"assets_ids"`
	Outcomes              []string               `json:"outcomes"`
	EventMessage          map[string]interface{} `json:"event_message,omitempty"`
	Timestamp             string                 `json:"timestamp"`
	Tags                  []string               `json:"tags"`
	ConditionID           string                 `json:"condition_id"`
	CLOBTokenIDs          []string               `json:"clob_token_ids"`
	Active                bool                   `json:"active"`
	SportsMarketType      string                 `json:"sports_market_type,omitempty"`
	Line                  string                 `json:"line,omitempty"`
	GameStartTime         string                 `json:"game_start_time,omitempty"`
	OrderPriceMinTickSize string                 `json:"order_price_min_tick_size,omitempty"`
	GroupItemTitle        string                 `json:"group_item_title,omitempty"`
	TakerBaseFee          string                 `json:"taker_base_fee,omitempty"`
	FeesEnabled           bool                   `json:"fees_enabled,omitempty"`
	FeeSchedule           map[string]interface{} `json:"fee_schedule,omitempty"`
}

// MarketResolvedMessage is a market lifecycle resolution event.
type MarketResolvedMessage struct {
	EventType      string   `json:"event_type"`
	ID             string   `json:"id"`
	Market         string   `json:"market"`
	AssetIDs       []string `json:"assets_ids"`
	WinningAssetID string   `json:"winning_asset_id"`
	WinningOutcome string   `json:"winning_outcome"`
	Timestamp      string   `json:"timestamp"`
	Tags           []string `json:"tags"`
}

// RawMessage is a deduplicated market-channel payload plus commonly indexed
// fields. It is useful for append-only WebSocket recorders that want JSONB
// payloads without depending on one typed event shape.
type RawMessage struct {
	ObservedAt time.Time       `json:"observed_at"`
	EventType  string          `json:"event_type,omitempty"`
	Market     string          `json:"market,omitempty"`
	AssetID    string          `json:"asset_id,omitempty"`
	Payload    json.RawMessage `json:"payload"`
}

// MarketClient manages a public market WebSocket connection.
type MarketClient struct {
	config     Config
	conn       *websocket.Conn
	mu         sync.Mutex
	ctx        context.Context
	cancel     context.CancelFunc
	connected  atomic.Bool
	reconnects int32
	assets     []string
	stats      *StreamStats
	dedup      *Deduplicator

	// Callbacks
	OnBook           func(BookMessage)
	OnPriceChange    func(PriceChangeMessage)
	OnLastTrade      func(LastTradeMessage)
	OnTickSizeChange func(TickSizeChangeMessage)
	OnBestBidAsk     func(BestBidAskMessage)
	OnNewMarket      func(NewMarketMessage)
	OnMarketResolved func(MarketResolvedMessage)
	OnRawMessage     func(RawMessage)
	OnError          func(error)
}

// NewMarketClient creates a public market WebSocket client.
func NewMarketClient(config Config) *MarketClient {
	return &MarketClient{config: config, stats: NewStreamStats("market"), dedup: NewDeduplicator(4096, 2*time.Minute)}
}

// Connect establishes the WebSocket connection.
func (mc *MarketClient) Connect(ctx context.Context) error {
	mc.ctx, mc.cancel = context.WithCancel(ctx)
	return mc.dial()
}

func (mc *MarketClient) dial() error {
	conn, _, err := websocket.DefaultDialer.Dial(mc.config.URL, nil)
	if err != nil {
		return fmt.Errorf("ws dial: %w", err)
	}
	mc.mu.Lock()
	mc.conn = conn
	mc.connected.Store(true)
	mc.stats.MarkConnected(time.Now())
	mc.mu.Unlock()
	conn.SetPongHandler(func(string) error {
		mc.mu.Lock()
		c := mc.conn
		mc.mu.Unlock()
		if c != nil && mc.config.PongTimeout > 0 {
			c.SetReadDeadline(time.Now().Add(mc.config.PongTimeout))
		}
		return nil
	})
	// Set an initial read deadline so a server that goes silent (never sending
	// a pong) is detected: without this the deadline is only ever set inside the
	// pong handler, so a dead-but-open connection would block ReadMessage forever
	// and the reconnect machinery would never fire.
	if mc.config.PongTimeout > 0 {
		conn.SetReadDeadline(time.Now().Add(mc.config.PongTimeout))
	}
	go mc.readLoop()
	go mc.pingLoop()
	return nil
}

func (mc *MarketClient) readLoop() {
	mc.mu.Lock()
	conn := mc.conn
	mc.mu.Unlock()
	defer func() {
		if conn != nil {
			conn.Close()
		}
	}()
	for {
		select {
		case <-mc.ctx.Done():
			return
		default:
		}
		mc.mu.Lock()
		conn = mc.conn
		mc.mu.Unlock()
		if conn == nil {
			return
		}
		_, msg, err := conn.ReadMessage()
		if err != nil {
			if mc.OnError != nil {
				mc.OnError(fmt.Errorf("ws read: %w", err))
			}
			mc.connected.Store(false)
			mc.reconnect()
			return
		}
		mc.processMessage(msg)
	}
}

func (mc *MarketClient) processMessage(msg []byte) {
	if mc.dedup != nil && !mc.dedup.Process(msg) {
		mc.stats.RecordDuplicate()
		return
	}
	observedAt := time.Now()
	if mc.OnRawMessage != nil {
		mc.OnRawMessage(rawMessageFromBytes(msg, observedAt))
	}
	mc.dispatch(msg, observedAt)
}

func (mc *MarketClient) dispatch(msg []byte, observedAt time.Time) {
	var envelope struct {
		EventType string `json:"event_type"`
	}
	if err := json.Unmarshal(msg, &envelope); err != nil {
		mc.stats.RecordInvalid()
		return
	}

	switch envelope.EventType {
	case "book":
		var book BookMessage
		if json.Unmarshal(msg, &book) != nil {
			mc.stats.RecordInvalid()
			return
		}
		mc.stats.RecordEvent(envelope.EventType, observedAt)
		if mc.OnBook != nil {
			mc.OnBook(book)
		}
	case "price_change":
		var pc PriceChangeMessage
		if json.Unmarshal(msg, &pc) != nil {
			mc.stats.RecordInvalid()
			return
		}
		mc.stats.RecordEvent(envelope.EventType, observedAt)
		if mc.OnPriceChange != nil {
			mc.OnPriceChange(pc)
		}
	case "last_trade_price":
		var lt LastTradeMessage
		if json.Unmarshal(msg, &lt) != nil {
			mc.stats.RecordInvalid()
			return
		}
		mc.stats.RecordEvent(envelope.EventType, observedAt)
		if mc.OnLastTrade != nil {
			mc.OnLastTrade(lt)
		}
	case "tick_size_change":
		var tick TickSizeChangeMessage
		if json.Unmarshal(msg, &tick) != nil {
			mc.stats.RecordInvalid()
			return
		}
		mc.stats.RecordEvent(envelope.EventType, observedAt)
		if mc.OnTickSizeChange != nil {
			mc.OnTickSizeChange(tick)
		}
	case "best_bid_ask":
		var best BestBidAskMessage
		if json.Unmarshal(msg, &best) != nil {
			mc.stats.RecordInvalid()
			return
		}
		mc.stats.RecordEvent(envelope.EventType, observedAt)
		if mc.OnBestBidAsk != nil {
			mc.OnBestBidAsk(best)
		}
	case "new_market":
		var market NewMarketMessage
		if json.Unmarshal(msg, &market) != nil {
			mc.stats.RecordInvalid()
			return
		}
		mc.stats.RecordEvent(envelope.EventType, observedAt)
		if mc.OnNewMarket != nil {
			mc.OnNewMarket(market)
		}
	case "market_resolved":
		var resolved MarketResolvedMessage
		if json.Unmarshal(msg, &resolved) != nil {
			mc.stats.RecordInvalid()
			return
		}
		mc.stats.RecordEvent(envelope.EventType, observedAt)
		if mc.OnMarketResolved != nil {
			mc.OnMarketResolved(resolved)
		}
	default:
		mc.stats.RecordInvalid()
	}
}

func rawMessageFromBytes(msg []byte, observedAt time.Time) RawMessage {
	var envelope struct {
		EventType string `json:"event_type"`
		Market    string `json:"market"`
		AssetID   string `json:"asset_id"`
	}
	_ = json.Unmarshal(msg, &envelope)
	return RawMessage{
		ObservedAt: observedAt.UTC(),
		EventType:  envelope.EventType,
		Market:     envelope.Market,
		AssetID:    envelope.AssetID,
		Payload:    append(json.RawMessage(nil), msg...),
	}
}

func (mc *MarketClient) pingLoop() {
	ticker := time.NewTicker(mc.config.PingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-mc.ctx.Done():
			return
		case <-ticker.C:
			mc.mu.Lock()
			if mc.conn != nil {
				mc.conn.WriteMessage(websocket.PingMessage, nil)
			}
			mc.mu.Unlock()
		}
	}
}

func (mc *MarketClient) reconnect() {
	if !mc.config.Reconnect || atomic.LoadInt32(&mc.reconnects) >= int32(mc.config.ReconnectMax) {
		return
	}
	atomic.AddInt32(&mc.reconnects, 1)
	mc.stats.RecordReconnect(time.Now())
	delay := mc.config.ReconnectDelay
	for i := int32(0); i < atomic.LoadInt32(&mc.reconnects); i++ {
		delay *= 2
		if delay > mc.config.ReconnectMaxDelay {
			delay = mc.config.ReconnectMaxDelay
		}
	}
	time.Sleep(delay)
	if mc.ctx.Err() != nil {
		return
	}
	if err := mc.dial(); err != nil {
		if mc.OnError != nil {
			mc.OnError(fmt.Errorf("ws reconnect: %w", err))
		}
		return
	}
	if err := mc.resubscribe(); err != nil && mc.OnError != nil {
		mc.OnError(err)
	}
	// Reconnect succeeded; reset the attempt counter so a later transient drop
	// gets the full ReconnectMax budget again. Without this the client would
	// permanently stop reconnecting after ReconnectMax cumulative drops over
	// its whole lifetime, even with healthy periods in between.
	atomic.StoreInt32(&mc.reconnects, 0)
}

// SubscribeAssets subscribes to order book updates for given token IDs.
func (mc *MarketClient) SubscribeAssets(ctx context.Context, assetIDs []string) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	if err := mc.writeSubscribeLocked(assetIDs); err != nil {
		return err
	}
	mc.assets = append([]string(nil), assetIDs...)
	mc.stats.SetSubscriptions(assetIDs, nil)
	return nil
}

func (mc *MarketClient) resubscribe() error {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	if len(mc.assets) == 0 {
		return nil
	}
	return mc.writeSubscribeLocked(mc.assets)
}

func (mc *MarketClient) writeSubscribeLocked(assetIDs []string) error {
	if mc.conn == nil {
		return fmt.Errorf("ws subscribe: not connected")
	}
	msg := map[string]interface{}{
		"type":       "market",
		"assets_ids": assetIDs,
	}
	if mc.config.Level > 0 {
		msg["level"] = mc.config.Level
	}
	if mc.config.CustomFeatureEnabled {
		msg["custom_feature_enabled"] = true
	}
	return mc.conn.WriteJSON(msg)
}

// Close shuts down the WebSocket connection.
func (mc *MarketClient) Close() {
	if mc.cancel != nil {
		mc.cancel()
	}
	mc.mu.Lock()
	if mc.conn != nil {
		mc.conn.Close()
	}
	mc.stats.MarkDisconnected(time.Now())
	mc.mu.Unlock()
}

// IsConnected returns the current connection state.
func (mc *MarketClient) IsConnected() bool {
	return mc.connected.Load()
}

// Stats returns a snapshot of stream lifecycle and message counters.
func (mc *MarketClient) Stats() StreamStatsSnapshot {
	return mc.stats.Snapshot()
}
