// Package rtds exposes a Polymarket Real-Time Data Service (RTDS) WebSocket
// client for the wss://ws-live-data.polymarket.com host.
//
// The RTDS host is distinct from the CLOB market/user sockets (pkg/stream):
// it broadcasts site-level feeds. This package currently exposes the
// crypto_prices_chainlink topic — the Chainlink oracle prices Polymarket
// uses to resolve its crypto Up/Down markets — at roughly one update per
// second per feed.
//
// When not to use this package:
//   - For CLOB order-book/user events — use pkg/stream.
//   - For a window's open/close resolution reference price — use
//     pkg/cryptoprice (HTTP, authoritative per-window record).
//
// Stability: Client, NewClient, Config, DefaultConfig, DefaultURL,
// ChainlinkPriceEvent, and the Connect/Connected/Close methods are part of
// the polygolem public SDK.
package rtds

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// DefaultURL is the production RTDS WebSocket endpoint.
const DefaultURL = "wss://ws-live-data.polymarket.com"

const chainlinkTopic = "crypto_prices_chainlink"

// Config holds RTDS WebSocket connection configuration.
type Config struct {
	URL               string
	PingInterval      time.Duration
	PongTimeout       time.Duration
	Reconnect         bool
	ReconnectDelay    time.Duration
	ReconnectMaxDelay time.Duration
	ReconnectMax      int
}

// DefaultConfig returns production defaults. Pass an empty URL to use the
// production RTDS endpoint.
func DefaultConfig(url string) Config {
	if url == "" {
		url = DefaultURL
	}
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

// ChainlinkPriceEvent is one Chainlink oracle price update.
type ChainlinkPriceEvent struct {
	// Symbol is the lowercase feed name as broadcast, e.g. "btc/usd".
	Symbol string
	// Value is the oracle price as a float. For a lossless representation
	// use FullAccuracyValue.
	Value float64
	// FullAccuracyValue is the raw fixed-point decimal string as broadcast
	// (1e18-scaled integer), empty if absent.
	FullAccuracyValue string
	// FeedTime is the oracle-side timestamp of the update (second-aligned).
	FeedTime time.Time
	// ObservedAt is the local receive time.
	ObservedAt time.Time
}

// Client manages one RTDS WebSocket connection subscribed to the Chainlink
// crypto price topic. Set callbacks before Connect. Methods are safe for
// concurrent use.
type Client struct {
	config     Config
	feeds      map[string]struct{}
	conn       *websocket.Conn
	mu         sync.Mutex
	ctx        context.Context
	cancel     context.CancelFunc
	connected  atomic.Bool
	reconnects int32

	// OnChainlinkPrice receives each Chainlink price update that passes the
	// feed filter. Called from the read loop; do not block.
	OnChainlinkPrice func(ChainlinkPriceEvent)
	// OnConnected fires after every successful connect or reconnect, once
	// the subscription has been written. Useful for gap accounting.
	OnConnected func()
	// OnError receives read/reconnect errors.
	OnError func(error)
}

// NewClient creates an RTDS client subscribed to the Chainlink price topic.
// feeds filters delivered events by lowercase feed name (e.g. "btc/usd");
// empty delivers every feed the server broadcasts. A zero-valued Config uses
// production defaults.
func NewClient(cfg Config, feeds []string) *Client {
	if cfg.URL == "" {
		cfg = DefaultConfig("")
	}
	set := make(map[string]struct{}, len(feeds))
	for _, feed := range feeds {
		feed = strings.ToLower(strings.TrimSpace(feed))
		if feed != "" {
			set[feed] = struct{}{}
		}
	}
	return &Client{config: cfg, feeds: set}
}

// Connect dials the endpoint, subscribes, and starts the read/ping loops.
func (c *Client) Connect(ctx context.Context) error {
	c.ctx, c.cancel = context.WithCancel(ctx)
	return c.dial()
}

// Connected reports whether the underlying socket is currently up.
func (c *Client) Connected() bool { return c.connected.Load() }

// Close tears down the connection and stops all loops.
func (c *Client) Close() error {
	if c.cancel != nil {
		c.cancel()
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.connected.Store(false)
	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}

func (c *Client) dial() error {
	conn, _, err := websocket.DefaultDialer.Dial(c.config.URL, nil)
	if err != nil {
		return fmt.Errorf("rtds dial: %w", err)
	}
	c.mu.Lock()
	c.conn = conn
	c.connected.Store(true)
	c.mu.Unlock()
	conn.SetPongHandler(func(string) error {
		c.mu.Lock()
		cc := c.conn
		c.mu.Unlock()
		if cc != nil && c.config.PongTimeout > 0 {
			cc.SetReadDeadline(time.Now().Add(c.config.PongTimeout))
		}
		return nil
	})
	// Initial read deadline so a silent dead connection is detected even if
	// no pong ever arrives (same rationale as internal/stream).
	if c.config.PongTimeout > 0 {
		conn.SetReadDeadline(time.Now().Add(c.config.PongTimeout))
	}
	if err := c.writeSubscribe(); err != nil {
		conn.Close()
		c.connected.Store(false)
		return err
	}
	go c.readLoop()
	go c.pingLoop()
	if c.OnConnected != nil {
		c.OnConnected()
	}
	return nil
}

func (c *Client) writeSubscribe() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return fmt.Errorf("rtds subscribe: not connected")
	}
	// Subscribe unfiltered: the chainlink topic's server-side filter accepts
	// only a single feed, so multi-feed clients filter locally instead.
	return c.conn.WriteJSON(map[string]any{
		"action": "subscribe",
		"subscriptions": []map[string]any{
			{"topic": chainlinkTopic, "type": "*"},
		},
	})
}

func (c *Client) readLoop() {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()
	defer func() {
		if conn != nil {
			conn.Close()
		}
	}()
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}
		c.mu.Lock()
		conn = c.conn
		c.mu.Unlock()
		if conn == nil {
			return
		}
		_, msg, err := conn.ReadMessage()
		if err != nil {
			select {
			case <-c.ctx.Done():
				// Close() interrupted the read; not an error.
				return
			default:
			}
			if c.OnError != nil {
				c.OnError(fmt.Errorf("rtds read: %w", err))
			}
			c.connected.Store(false)
			c.reconnect()
			return
		}
		c.processMessage(msg, time.Now())
	}
}

func (c *Client) processMessage(msg []byte, observedAt time.Time) {
	// The server intersperses empty keepalive frames; skip anything that is
	// not a JSON object or array.
	trimmed := strings.TrimSpace(string(msg))
	if trimmed == "" {
		return
	}
	var envelopes []rtdsEnvelope
	if strings.HasPrefix(trimmed, "[") {
		if json.Unmarshal([]byte(trimmed), &envelopes) != nil {
			return
		}
	} else {
		var one rtdsEnvelope
		if json.Unmarshal([]byte(trimmed), &one) != nil {
			return
		}
		envelopes = append(envelopes, one)
	}
	for _, envelope := range envelopes {
		if envelope.Topic != chainlinkTopic || len(envelope.Payload) == 0 {
			continue
		}
		var payload struct {
			Symbol            string  `json:"symbol"`
			Timestamp         int64   `json:"timestamp"`
			Value             float64 `json:"value"`
			FullAccuracyValue string  `json:"full_accuracy_value"`
		}
		if json.Unmarshal(envelope.Payload, &payload) != nil {
			continue
		}
		symbol := strings.ToLower(strings.TrimSpace(payload.Symbol))
		if symbol == "" {
			continue
		}
		if len(c.feeds) > 0 {
			if _, ok := c.feeds[symbol]; !ok {
				continue
			}
		}
		if c.OnChainlinkPrice != nil {
			c.OnChainlinkPrice(ChainlinkPriceEvent{
				Symbol:            symbol,
				Value:             payload.Value,
				FullAccuracyValue: payload.FullAccuracyValue,
				FeedTime:          time.UnixMilli(payload.Timestamp).UTC(),
				ObservedAt:        observedAt.UTC(),
			})
		}
	}
}

type rtdsEnvelope struct {
	Topic   string          `json:"topic"`
	MsgType string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

func (c *Client) pingLoop() {
	ticker := time.NewTicker(c.config.PingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.mu.Lock()
			if c.conn != nil {
				c.conn.WriteMessage(websocket.PingMessage, nil)
			}
			c.mu.Unlock()
		}
	}
}

func (c *Client) reconnect() {
	if !c.config.Reconnect || atomic.LoadInt32(&c.reconnects) >= int32(c.config.ReconnectMax) {
		return
	}
	atomic.AddInt32(&c.reconnects, 1)
	delay := c.config.ReconnectDelay
	for i := int32(0); i < atomic.LoadInt32(&c.reconnects); i++ {
		delay *= 2
		if delay > c.config.ReconnectMaxDelay {
			delay = c.config.ReconnectMaxDelay
		}
	}
	time.Sleep(delay)
	if c.ctx.Err() != nil {
		return
	}
	if err := c.dial(); err != nil {
		if c.OnError != nil {
			c.OnError(fmt.Errorf("rtds reconnect: %w", err))
		}
		return
	}
	// Reconnect succeeded; restore the full retry budget for future drops
	// (same rationale as internal/stream).
	atomic.StoreInt32(&c.reconnects, 0)
}
