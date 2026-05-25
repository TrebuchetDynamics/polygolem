// Package depositwalletsettlement owns read-only deposit-wallet settlement discovery.
package depositwalletsettlement

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/TrebuchetDynamics/polygolem/internal/auth"
	"github.com/TrebuchetDynamics/polygolem/pkg/data"
	"github.com/TrebuchetDynamics/polygolem/pkg/settlement"
)

// PrivateKeyLoader loads the EOA key used to derive the deposit wallet owner.
type PrivateKeyLoader func() (string, error)

// DataClientProvider provides the Data API client used for settlement reads.
type DataClientProvider func() *data.Client

// RelayerConfiguredChecker reports whether relayer credentials are configured.
type RelayerConfiguredChecker func() bool

// RPCURLProvider supplies the default Polygon RPC URL for settlement checks.
type RPCURLProvider func() string

// ReadinessChecker performs the read-only settlement readiness gate.
type ReadinessChecker func(context.Context, *data.Client, string, string, settlement.ReadinessOptions) (*settlement.Readiness, error)

// RedeemableFinder lists redeemable positions for a deposit wallet.
type RedeemableFinder func(context.Context, *data.Client, string) ([]settlement.RedeemablePosition, error)

// Config contains adapters used by the deposit-wallet settlement workflow.
type Config struct {
	PrivateKey        PrivateKeyLoader
	DataClient        DataClientProvider
	RelayerConfigured RelayerConfiguredChecker
	RPCURL            RPCURLProvider
	CheckReadiness    ReadinessChecker
	FindRedeemable    RedeemableFinder
}

// Runner owns read-only settlement orchestration behind a small interface.
type Runner struct {
	privateKey        PrivateKeyLoader
	dataClient        DataClientProvider
	relayerConfigured RelayerConfiguredChecker
	rpcURL            RPCURLProvider
	checkReadiness    ReadinessChecker
	findRedeemable    RedeemableFinder
}

// StatusRequest configures a settlement-status read.
type StatusRequest struct {
	RPCURL string
}

// RedeemableResult is the JSON payload for deposit-wallet redeemable.
type RedeemableResult struct {
	DepositWallet string                          `json:"depositWallet"`
	Count         int                             `json:"count"`
	Positions     []settlement.RedeemablePosition `json:"positions"`
}

// New creates a deposit-wallet settlement workflow runner.
func New(cfg Config) *Runner {
	return &Runner{
		privateKey:        cfg.PrivateKey,
		dataClient:        cfg.DataClient,
		relayerConfigured: cfg.RelayerConfigured,
		rpcURL:            cfg.RPCURL,
		checkReadiness:    cfg.CheckReadiness,
		findRedeemable:    cfg.FindRedeemable,
	}
}

// SettlementStatus checks read-only V2 settlement readiness for the derived deposit wallet.
func (r *Runner) SettlementStatus(ctx context.Context, req StatusRequest) (*settlement.Readiness, error) {
	key, err := r.loadPrivateKey()
	if err != nil {
		return nil, err
	}
	owner, wallet, err := ownerAndWallet(key)
	if err != nil {
		return nil, err
	}
	return r.readinessChecker()(ctx, r.dataAPIClient(), owner, wallet, settlement.ReadinessOptions{
		RPCURL:            firstNonEmpty(req.RPCURL, r.rpcURLValue()),
		RelayerConfigured: r.relayerReady(),
	})
}

// Redeemable lists redeemable positions held by the derived deposit wallet.
func (r *Runner) Redeemable(ctx context.Context) (RedeemableResult, error) {
	key, err := r.loadPrivateKey()
	if err != nil {
		return RedeemableResult{}, err
	}
	_, wallet, err := ownerAndWallet(key)
	if err != nil {
		return RedeemableResult{}, err
	}
	rows, err := r.redeemableFinder()(ctx, r.dataAPIClient(), wallet)
	if err != nil {
		return RedeemableResult{}, fmt.Errorf("find redeemable: %w", err)
	}
	return RedeemableResult{DepositWallet: wallet, Count: len(rows), Positions: rows}, nil
}

func (r *Runner) loadPrivateKey() (string, error) {
	if r.privateKey == nil {
		return "", fmt.Errorf("private key loader is required")
	}
	return r.privateKey()
}

func (r *Runner) dataAPIClient() *data.Client {
	if r.dataClient != nil {
		return r.dataClient()
	}
	return data.NewClient(data.DefaultConfig())
}

func (r *Runner) readinessChecker() ReadinessChecker {
	if r.checkReadiness != nil {
		return r.checkReadiness
	}
	return settlement.CheckReadiness
}

func (r *Runner) redeemableFinder() RedeemableFinder {
	if r.findRedeemable != nil {
		return r.findRedeemable
	}
	return settlement.FindRedeemable
}

func (r *Runner) relayerReady() bool {
	if r.relayerConfigured == nil {
		return false
	}
	return r.relayerConfigured()
}

func (r *Runner) rpcURLValue() string {
	if r.rpcURL != nil {
		return r.rpcURL()
	}
	return os.Getenv("POLYGON_RPC_URL")
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
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
