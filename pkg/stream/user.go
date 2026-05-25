package stream

import (
	"context"

	"github.com/TrebuchetDynamics/polygolem/internal/auth"
	internalstream "github.com/TrebuchetDynamics/polygolem/internal/stream"
)

const defaultUserURL = "wss://ws-subscriptions-clob.polymarket.com/ws/user"

// UserCredentials is the L2 API-key triple required for the authenticated
// Polymarket user WebSocket channel.
type UserCredentials struct {
	Key        string `json:"apiKey"`
	Secret     string `json:"secret"`
	Passphrase string `json:"passphrase"`
}

// UserOrderMessage is an authenticated user-channel order event.
type UserOrderMessage = internalstream.UserOrderMessage

// UserTradeMessage is an authenticated user-channel trade/fill event.
type UserTradeMessage = internalstream.UserTradeMessage

// DefaultUserConfig returns production user-stream defaults. Pass an empty URL
// to use the Polymarket production authenticated user-channel endpoint.
func DefaultUserConfig(url string) Config {
	if url == "" {
		url = defaultUserURL
	}
	cfg := internalstream.DefaultConfig(url)
	return configFromInternal(cfg)
}

// UserClient manages one authenticated user WebSocket connection.
type UserClient struct {
	inner *internalstream.UserClient

	OnOrder func(UserOrderMessage)
	OnTrade func(UserTradeMessage)
	OnError func(error)
}

// NewUserClient creates an authenticated user-stream client. A zero-valued
// Config uses production user-channel defaults. Credentials must be a complete
// CLOB L2 API-key triple.
func NewUserClient(cfg Config, credentials UserCredentials) *UserClient {
	if cfg.URL == "" {
		cfg = DefaultUserConfig("")
	}
	client := &UserClient{}
	inner := internalstream.NewUserClient(configToInternal(cfg), auth.APIKey{
		Key:        credentials.Key,
		Secret:     credentials.Secret,
		Passphrase: credentials.Passphrase,
	})
	inner.OnOrder = func(msg internalstream.UserOrderMessage) {
		if client.OnOrder != nil {
			client.OnOrder(msg)
		}
	}
	inner.OnTrade = func(msg internalstream.UserTradeMessage) {
		if client.OnTrade != nil {
			client.OnTrade(msg)
		}
	}
	inner.OnError = func(err error) {
		if client.OnError != nil {
			client.OnError(err)
		}
	}
	client.inner = inner
	return client
}

func (c *UserClient) Connect(ctx context.Context) error { return c.inner.Connect(ctx) }

func (c *UserClient) SubscribeUser(ctx context.Context, markets []string) error {
	return c.inner.SubscribeUser(ctx, markets)
}

func (c *UserClient) Close() { c.inner.Close() }

func (c *UserClient) IsConnected() bool { return c.inner.IsConnected() }
