package stream

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/TrebuchetDynamics/polygolem/internal/auth"
	"github.com/gorilla/websocket"
)

// UserCredentials are the L2 API-key triple required by the authenticated
// Polymarket user WebSocket channel.
type UserCredentials struct {
	Key        string `json:"apiKey"`
	Secret     string `json:"secret"`
	Passphrase string `json:"passphrase"`
}

func userCredentialsFromAPIKey(key auth.APIKey) UserCredentials {
	return UserCredentials{Key: key.Key, Secret: key.Secret, Passphrase: key.Passphrase}
}

func (c UserCredentials) validate() error {
	if c.Key == "" || c.Secret == "" || c.Passphrase == "" {
		return fmt.Errorf("user stream credentials require api key, secret, and passphrase")
	}
	return nil
}

// UserOrderMessage is an authenticated user-channel order event. Unknown
// upstream fields are ignored; known ID/status fields are preserved for typed
// consumers.
type UserOrderMessage struct {
	EventType string `json:"event_type"`
	ID        string `json:"id,omitempty"`
	OrderID   string `json:"order_id,omitempty"`
	Market    string `json:"market,omitempty"`
	AssetID   string `json:"asset_id,omitempty"`
	Side      string `json:"side,omitempty"`
	Price     string `json:"price,omitempty"`
	Size      string `json:"size,omitempty"`
	Status    string `json:"status,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
}

// UserTradeMessage is an authenticated user-channel trade/fill event.
type UserTradeMessage struct {
	EventType       string `json:"event_type"`
	ID              string `json:"id,omitempty"`
	TradeID         string `json:"trade_id,omitempty"`
	OrderID         string `json:"order_id,omitempty"`
	Market          string `json:"market,omitempty"`
	AssetID         string `json:"asset_id,omitempty"`
	Side            string `json:"side,omitempty"`
	Price           string `json:"price,omitempty"`
	Size            string `json:"size,omitempty"`
	FeeRateBps      string `json:"fee_rate_bps,omitempty"`
	Timestamp       string `json:"timestamp,omitempty"`
	TransactionHash string `json:"transaction_hash,omitempty"`
}

// UserClient manages an authenticated user WebSocket connection.
type UserClient struct {
	config      Config
	credentials UserCredentials
	conn        *websocket.Conn
	mu          sync.Mutex
	ctx         context.Context
	cancel      context.CancelFunc
	connected   atomic.Bool
	markets     []string

	OnOrder func(UserOrderMessage)
	OnTrade func(UserTradeMessage)
	OnError func(error)
}

// NewUserClient creates an authenticated user-stream client.
func NewUserClient(config Config, credentials auth.APIKey) *UserClient {
	return &UserClient{config: config, credentials: userCredentialsFromAPIKey(credentials)}
}

// Connect establishes the WebSocket connection.
func (uc *UserClient) Connect(ctx context.Context) error {
	if err := uc.credentials.validate(); err != nil {
		return err
	}
	uc.ctx, uc.cancel = context.WithCancel(ctx)
	conn, _, err := websocket.DefaultDialer.Dial(uc.config.URL, nil)
	if err != nil {
		return fmt.Errorf("user ws dial: %w", err)
	}
	uc.mu.Lock()
	uc.conn = conn
	uc.connected.Store(true)
	uc.mu.Unlock()
	go uc.readLoop()
	go uc.pingLoop()
	return nil
}

func (uc *UserClient) readLoop() {
	uc.mu.Lock()
	conn := uc.conn
	uc.mu.Unlock()
	defer func() {
		if conn != nil {
			conn.Close()
		}
		uc.connected.Store(false)
	}()
	for {
		select {
		case <-uc.ctx.Done():
			return
		default:
		}
		_, msg, err := conn.ReadMessage()
		if err != nil {
			if uc.OnError != nil && uc.ctx.Err() == nil {
				uc.OnError(fmt.Errorf("user ws read: %w", err))
			}
			return
		}
		uc.dispatch(msg)
	}
}

func (uc *UserClient) dispatch(msg []byte) {
	var envelope struct {
		EventType string `json:"event_type"`
		Type      string `json:"type"`
	}
	if err := json.Unmarshal(msg, &envelope); err != nil {
		if uc.OnError != nil {
			uc.OnError(err)
		}
		return
	}
	eventType := firstNonEmpty(envelope.EventType, envelope.Type)
	switch eventType {
	case "order":
		if uc.OnOrder != nil {
			var order UserOrderMessage
			if err := json.Unmarshal(msg, &order); err == nil {
				order.EventType = eventType
				uc.OnOrder(order)
			}
		}
	case "trade":
		if uc.OnTrade != nil {
			var trade UserTradeMessage
			if err := json.Unmarshal(msg, &trade); err == nil {
				trade.EventType = eventType
				uc.OnTrade(trade)
			}
		}
	}
}

func (uc *UserClient) pingLoop() {
	interval := uc.config.PingInterval
	if interval == 0 {
		interval = 10 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-uc.ctx.Done():
			return
		case <-ticker.C:
			uc.mu.Lock()
			if uc.conn != nil {
				_ = uc.conn.WriteMessage(websocket.PingMessage, nil)
			}
			uc.mu.Unlock()
		}
	}
}

// SubscribeUser authenticates the user channel and optionally filters by market
// condition IDs when upstream supports market scoping.
func (uc *UserClient) SubscribeUser(ctx context.Context, markets []string) error {
	uc.mu.Lock()
	defer uc.mu.Unlock()
	if uc.conn == nil {
		return fmt.Errorf("user ws subscribe: not connected")
	}
	if err := uc.credentials.validate(); err != nil {
		return err
	}
	payload := map[string]interface{}{
		"type":    "user",
		"markets": markets,
		"auth": map[string]string{
			"apiKey":     uc.credentials.Key,
			"secret":     uc.credentials.Secret,
			"passphrase": uc.credentials.Passphrase,
		},
	}
	if err := uc.conn.WriteJSON(payload); err != nil {
		return err
	}
	uc.markets = append([]string(nil), markets...)
	_ = ctx
	return nil
}

func (uc *UserClient) Close() {
	if uc.cancel != nil {
		uc.cancel()
	}
	uc.mu.Lock()
	if uc.conn != nil {
		_ = uc.conn.Close()
	}
	uc.mu.Unlock()
}

func (uc *UserClient) IsConnected() bool { return uc.connected.Load() }

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
