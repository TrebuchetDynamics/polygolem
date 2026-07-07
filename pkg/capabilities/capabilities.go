// Package capabilities exposes Polymarket surface metadata for docs, agents,
// and safety gates.
package capabilities

import "sort"

type AuthRequirement string

const (
	AuthNone       AuthRequirement = "none"
	AuthL1         AuthRequirement = "l1"
	AuthL2         AuthRequirement = "l2"
	AuthSIWE       AuthRequirement = "siwe"
	AuthPrivateKey AuthRequirement = "private_key"
)

type WalletMode string

const (
	WalletNone        WalletMode = "none"
	WalletDepositOnly WalletMode = "deposit_wallet_only"
)

type Capability struct {
	ID          string
	Service     string
	Summary     string
	ReadOnly    bool
	Mutating    bool
	Auth        []AuthRequirement
	WalletMode  WalletMode
	SDKPackages []string
	CLI         []string
}

func (c Capability) Requires(auth AuthRequirement) bool {
	for _, got := range c.Auth {
		if got == auth {
			return true
		}
	}
	return false
}

func All() []Capability {
	caps := []Capability{
		{
			ID:          "bridge.funding",
			Service:     "Bridge",
			Summary:     "Supported assets, deposit addresses, quotes, and deposit status for pUSD funding.",
			Mutating:    true,
			Auth:        []AuthRequirement{AuthNone},
			WalletMode:  WalletDepositOnly,
			SDKPackages: []string{"pkg/bridge"},
			CLI:         []string{"bridge assets", "bridge deposit", "bridge status", "bridge quote"},
		},
		{
			ID:          "clob.public_data",
			Service:     "CLOB API",
			Summary:     "Public order books, prices, spreads, tick sizes, and market metadata.",
			ReadOnly:    true,
			Auth:        []AuthRequirement{AuthNone},
			WalletMode:  WalletNone,
			SDKPackages: []string{"pkg/clob", "pkg/orderbook", "pkg/marketdata"},
			CLI:         []string{"book", "exchange book", "exchange markets", "exchange price-history"},
		},
		{
			ID:          "clob.trading",
			Service:     "CLOB API",
			Summary:     "Deposit-wallet CLOB V2 order signing, placement, cancellation, account reads, and builder attribution.",
			Mutating:    true,
			Auth:        []AuthRequirement{AuthL1, AuthL2, AuthPrivateKey},
			WalletMode:  WalletDepositOnly,
			SDKPackages: []string{"pkg/clob"},
			CLI:         []string{"exchange create-order", "exchange market-order", "exchange cancel"},
		},
		{
			ID:          "data.positions",
			Service:     "Data API",
			Summary:     "Public wallet-level positions, activity, trades, value, holders, leaderboard, and open interest.",
			ReadOnly:    true,
			Auth:        []AuthRequirement{AuthNone},
			WalletMode:  WalletNone,
			SDKPackages: []string{"pkg/data"},
			CLI:         []string{"analytics positions", "analytics trades", "analytics activity"},
		},
		{
			ID:          "gamma.markets",
			Service:     "Gamma API",
			Summary:     "Public event, market, tag, series, comment, and search discovery.",
			ReadOnly:    true,
			Auth:        []AuthRequirement{AuthNone},
			WalletMode:  WalletNone,
			SDKPackages: []string{"pkg/gamma", "pkg/universal"},
			CLI:         []string{"markets search", "markets markets", "markets market"},
		},
		{
			ID:          "relayer.deposit_wallet",
			Service:     "Relayer V2",
			Summary:     "Deposit-wallet deploy, approvals, gasless transactions, CTF redeem, and transaction lookup.",
			Mutating:    true,
			Auth:        []AuthRequirement{AuthSIWE, AuthPrivateKey},
			WalletMode:  WalletDepositOnly,
			SDKPackages: []string{"pkg/relayer", "pkg/ctf", "pkg/settlement"},
			CLI:         []string{"wallet", "tx transaction"},
		},
		{
			ID:          "websocket.market",
			Service:     "CLOB WebSocket",
			Summary:     "Public real-time book, price, last-trade, tick-size, best-bid-ask, and lifecycle events.",
			ReadOnly:    true,
			Auth:        []AuthRequirement{AuthNone},
			WalletMode:  WalletNone,
			SDKPackages: []string{"pkg/stream", "pkg/marketdata"},
			CLI:         []string{"stream market", "stream crypto", "marketdata live"},
		},
		{
			ID:          "websocket.user",
			Service:     "CLOB WebSocket",
			Summary:     "Authenticated user order and trade stream for inspection and reconciliation.",
			Auth:        []AuthRequirement{AuthL2},
			WalletMode:  WalletDepositOnly,
			SDKPackages: []string{"pkg/stream"},
			CLI:         []string{"stream user"},
		},
	}
	sort.Slice(caps, func(i, j int) bool { return caps[i].ID < caps[j].ID })
	return caps
}

func ByID() map[string]Capability {
	out := map[string]Capability{}
	for _, cap := range All() {
		out[cap.ID] = cap
	}
	return out
}

func ReadOnlyIDs() []string {
	var out []string
	for _, cap := range All() {
		if cap.ReadOnly {
			out = append(out, cap.ID)
		}
	}
	return out
}
