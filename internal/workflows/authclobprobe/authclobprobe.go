// Package authclobprobe owns read-only CLOB credential probing for auth commands.
package authclobprobe

import (
	"context"
	"fmt"

	"github.com/TrebuchetDynamics/polygolem/internal/auth"
	"github.com/TrebuchetDynamics/polygolem/internal/clob"
	"github.com/TrebuchetDynamics/polygolem/internal/jsonx"
)

// PrivateKeyLoader loads the signer key from the caller's environment.
type PrivateKeyLoader func() (string, error)

// L2CredentialsProvider returns configured CLOB L2 credentials, when present.
type L2CredentialsProvider func() (auth.APIKey, bool)

// Reader is the read-only CLOB surface used by the probe.
type Reader interface {
	SetL2Credentials(auth.APIKey)
	ListOrders(context.Context, string) ([]clob.OrderRecord, error)
	ListTrades(context.Context, string) ([]clob.TradeRecord, error)
	BalanceAllowance(context.Context, string, clob.BalanceAllowanceParams) (*clob.BalanceAllowanceResponse, error)
}

// Config contains adapters used by the CLOB credential probe workflow.
type Config struct {
	PrivateKey    PrivateKeyLoader
	L2Credentials L2CredentialsProvider
	CLOB          Reader
}

// Runner owns read-only CLOB credential probe orchestration behind a small interface.
type Runner struct {
	privateKey    PrivateKeyLoader
	l2Credentials L2CredentialsProvider
	clob          Reader
}

// Result is the JSON payload for auth clob-probe.
type Result struct {
	CredentialSource   string          `json:"credentialSource"`
	ReadOnly           bool            `json:"readOnly"`
	DeriveAPIKeyCalled bool            `json:"deriveApiKeyCalled"`
	EOAAddress         string          `json:"eoaAddress"`
	DepositWallet      string          `json:"depositWallet"`
	Orders             CountResult     `json:"orders"`
	Trades             CountResult     `json:"trades"`
	BalanceAllowance   AllowanceResult `json:"balanceAllowance"`
}

// CountResult summarizes one read-only authenticated list endpoint.
type CountResult struct {
	OK       bool   `json:"ok"`
	Endpoint string `json:"endpoint"`
	Count    int    `json:"count"`
}

// AllowanceResult summarizes the read-only balance-allowance endpoint.
type AllowanceResult struct {
	OK        bool   `json:"ok"`
	Endpoint  string `json:"endpoint"`
	Balance   string `json:"balance"`
	Allowance string `json:"allowance,omitempty"`
}

// New creates a CLOB credential probe workflow runner.
func New(cfg Config) *Runner {
	return &Runner{privateKey: cfg.PrivateKey, l2Credentials: cfg.L2Credentials, clob: cfg.CLOB}
}

// Probe checks configured CLOB L2 credentials using only read-only endpoints.
func (r *Runner) Probe(ctx context.Context) (*Result, error) {
	privateKey, err := r.loadPrivateKey()
	if err != nil {
		return nil, err
	}
	key, err := r.configuredCredentials()
	if err != nil {
		return nil, err
	}
	reader, err := r.reader()
	if err != nil {
		return nil, err
	}

	owner, depositWallet, err := ownerAndWallet(privateKey)
	if err != nil {
		return nil, err
	}

	reader.SetL2Credentials(key)
	orders, err := reader.ListOrders(ctx, privateKey)
	if err != nil {
		return nil, fmt.Errorf("read CLOB orders with configured L2 credentials: %w", err)
	}
	trades, err := reader.ListTrades(ctx, privateKey)
	if err != nil {
		return nil, fmt.Errorf("read CLOB trades with configured L2 credentials: %w", err)
	}
	balance, err := reader.BalanceAllowance(ctx, privateKey, clob.BalanceAllowanceParams{AssetType: "COLLATERAL"})
	if err != nil {
		return nil, fmt.Errorf("read CLOB collateral balance with configured L2 credentials: %w", err)
	}
	if balance == nil {
		balance = &clob.BalanceAllowanceResponse{}
	}

	return &Result{
		CredentialSource:   "configured_clob_l2",
		ReadOnly:           true,
		DeriveAPIKeyCalled: false,
		EOAAddress:         owner,
		DepositWallet:      depositWallet,
		Orders: CountResult{
			OK:       true,
			Endpoint: "GET /data/orders",
			Count:    len(orders),
		},
		Trades: CountResult{
			OK:       true,
			Endpoint: "GET /data/trades",
			Count:    len(trades),
		},
		BalanceAllowance: AllowanceResult{
			OK:        true,
			Endpoint:  "GET /balance-allowance",
			Balance:   balance.Balance,
			Allowance: firstNonEmpty(balance.Allowance, balance.Allowances["collateral"], balance.Allowances["COLLATERAL"]),
		},
	}, nil
}

func (r *Runner) loadPrivateKey() (string, error) {
	if r.privateKey == nil {
		return "", fmt.Errorf("private key loader is required")
	}
	return r.privateKey()
}

func (r *Runner) configuredCredentials() (auth.APIKey, error) {
	if r.l2Credentials == nil {
		return auth.APIKey{}, fmt.Errorf("configured CLOB L2 credentials are required: set POLYMARKET_CLOB_API_KEY, POLYMARKET_CLOB_SECRET, and POLYMARKET_CLOB_PASSPHRASE")
	}
	key, ok := r.l2Credentials()
	if !ok {
		return auth.APIKey{}, fmt.Errorf("configured CLOB L2 credentials are required: set POLYMARKET_CLOB_API_KEY, POLYMARKET_CLOB_SECRET, and POLYMARKET_CLOB_PASSPHRASE")
	}
	if err := key.Validate(); err != nil {
		return auth.APIKey{}, fmt.Errorf("configured CLOB L2 credentials invalid: %w", err)
	}
	return key, nil
}

func (r *Runner) reader() (Reader, error) {
	if r.clob == nil {
		return nil, fmt.Errorf("CLOB client is required")
	}
	return r.clob, nil
}

func ownerAndWallet(privateKey string) (string, string, error) {
	signer, err := auth.NewPrivateKeySigner(privateKey, 137)
	if err != nil {
		return "", "", fmt.Errorf("init signer: %w", err)
	}
	owner := signer.Address()
	wallet, err := auth.MakerAddressForSignatureType(owner, signer.ChainID(), 3)
	if err != nil {
		return "", "", fmt.Errorf("derive deposit wallet: %w", err)
	}
	return owner, wallet, nil
}

func firstNonEmpty(values ...string) string {
	return jsonx.FirstString(values...)
}
