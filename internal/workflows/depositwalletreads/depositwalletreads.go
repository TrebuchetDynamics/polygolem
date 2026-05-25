// Package depositwalletreads owns read-only deposit-wallet CLI orchestration without Cobra coupling.
package depositwalletreads

import (
	"context"
	"fmt"
	"os"

	"github.com/TrebuchetDynamics/polygolem/internal/auth"
	"github.com/TrebuchetDynamics/polygolem/internal/relayer"
	"github.com/TrebuchetDynamics/polygolem/pkg/contracts"
)

// PrivateKeyLoader loads the EOA key used to derive the deposit wallet owner.
type PrivateKeyLoader func() (string, error)

// RelayerFactory builds the relayer reader for authenticated read operations.
type RelayerFactory func(context.Context, string) (RelayerReader, error)

// DeploymentChecker checks whether the derived deposit wallet has on-chain code.
type DeploymentChecker func(context.Context, string, string) (contracts.DeploymentStatus, error)

// RPCURLProvider supplies the Polygon RPC URL for on-chain code checks.
type RPCURLProvider func() string

// EnableTradingValidator optionally validates the read-only enable-trading status details.
type EnableTradingValidator func(context.Context, string, string, string) (map[string]interface{}, error)

// RelayerReader is the read-only relayer surface used by deposit-wallet read commands.
type RelayerReader interface {
	GetNonce(context.Context, string) (string, error)
	IsDeployed(context.Context, string) (bool, error)
	GetTransaction(context.Context, string) (*relayer.RelayerTransaction, error)
}

// Config contains adapters used by the deposit-wallet read workflow.
type Config struct {
	PrivateKey    PrivateKeyLoader
	Relayer       RelayerFactory
	Deployment    DeploymentChecker
	RPCURL        RPCURLProvider
	EnableTrading EnableTradingValidator
}

// Runner owns read-only deposit-wallet orchestration behind a small interface.
type Runner struct {
	privateKey    PrivateKeyLoader
	relayer       RelayerFactory
	deployment    DeploymentChecker
	rpcURL        RPCURLProvider
	enableTrading EnableTradingValidator
}

// DeriveResult is the JSON payload for deposit-wallet derive.
type DeriveResult struct {
	Owner         string `json:"owner"`
	DepositWallet string `json:"depositWallet"`
}

// NonceResult is the JSON payload for deposit-wallet nonce.
type NonceResult struct {
	Address string `json:"address"`
	Type    string `json:"type"`
	Nonce   string `json:"nonce"`
}

// StatusRequest describes optional status checks.
type StatusRequest struct {
	CheckEnableTrading bool
}

// StatusResult is the JSON payload for deposit-wallet status.
type StatusResult struct {
	Owner                  string                 `json:"owner"`
	DepositWallet          string                 `json:"depositWallet"`
	Deployed               bool                   `json:"deployed"`
	RelayerDeployed        bool                   `json:"relayerDeployed"`
	OnchainCodeDeployed    bool                   `json:"onchainCodeDeployed"`
	DeploymentStatusSource string                 `json:"deploymentStatusSource"`
	WalletNonce            string                 `json:"walletNonce"`
	EnableTrading          map[string]interface{} `json:"enableTrading,omitempty"`
}

// New creates a deposit-wallet read workflow runner.
func New(cfg Config) *Runner {
	return &Runner{
		privateKey:    cfg.PrivateKey,
		relayer:       cfg.Relayer,
		deployment:    cfg.Deployment,
		rpcURL:        cfg.RPCURL,
		enableTrading: cfg.EnableTrading,
	}
}

// Derive returns the deterministic POLY_1271 deposit-wallet address for the EOA.
func (r *Runner) Derive(_ context.Context) (DeriveResult, error) {
	key, err := r.loadPrivateKey()
	if err != nil {
		return DeriveResult{}, err
	}
	owner, wallet, err := ownerAndWallet(key)
	if err != nil {
		return DeriveResult{}, err
	}
	return DeriveResult{Owner: owner, DepositWallet: wallet}, nil
}

// Nonce reads the current WALLET nonce for the EOA owner.
func (r *Runner) Nonce(ctx context.Context) (NonceResult, error) {
	key, err := r.loadPrivateKey()
	if err != nil {
		return NonceResult{}, err
	}
	owner, _, err := ownerAndWallet(key)
	if err != nil {
		return NonceResult{}, err
	}
	rc, err := r.relayerReader(ctx, key)
	if err != nil {
		return NonceResult{}, err
	}
	nonce, err := rc.GetNonce(ctx, owner)
	if err != nil {
		return NonceResult{}, fmt.Errorf("get nonce: %w", err)
	}
	return NonceResult{Address: owner, Type: "WALLET", Nonce: nonce}, nil
}

// Transaction reads a relayer transaction by ID through the same status auth path.
func (r *Runner) Transaction(ctx context.Context, txID string) (*relayer.RelayerTransaction, error) {
	key, err := r.loadPrivateKey()
	if err != nil {
		return nil, err
	}
	rc, err := r.relayerReader(ctx, key)
	if err != nil {
		return nil, err
	}
	if _, _, err := ownerAndWallet(key); err != nil {
		return nil, err
	}
	tx, err := rc.GetTransaction(ctx, txID)
	if err != nil {
		return nil, fmt.Errorf("get transaction: %w", err)
	}
	return tx, nil
}

// Status checks deployment state and current WALLET nonce for the derived deposit wallet.
func (r *Runner) Status(ctx context.Context, req StatusRequest) (StatusResult, error) {
	key, err := r.loadPrivateKey()
	if err != nil {
		return StatusResult{}, err
	}
	rc, err := r.relayerReader(ctx, key)
	if err != nil {
		return StatusResult{}, err
	}
	owner, wallet, err := ownerAndWallet(key)
	if err != nil {
		return StatusResult{}, err
	}
	relayerDeployed, err := rc.IsDeployed(ctx, owner)
	if err != nil {
		return StatusResult{}, fmt.Errorf("deployed check: %w", err)
	}
	onchainDeployed := relayerDeployed
	if !relayerDeployed {
		codeStatus, err := r.deploymentChecker()(ctx, wallet, r.rpcURLValue())
		if err != nil {
			return StatusResult{}, fmt.Errorf("on-chain deposit wallet code check: %w", err)
		}
		onchainDeployed = codeStatus.Deployed
	}
	nonce, err := rc.GetNonce(ctx, owner)
	if err != nil {
		nonce = "error: " + err.Error()
	}
	result := StatusResult{
		Owner:                  owner,
		DepositWallet:          wallet,
		Deployed:               relayerDeployed || onchainDeployed,
		RelayerDeployed:        relayerDeployed,
		OnchainCodeDeployed:    onchainDeployed,
		DeploymentStatusSource: deploymentStatusSource(relayerDeployed, onchainDeployed),
		WalletNonce:            nonce,
	}
	if req.CheckEnableTrading {
		if r.enableTrading == nil {
			return StatusResult{}, fmt.Errorf("enable trading validator is required")
		}
		validation, err := r.enableTrading(ctx, key, wallet, nonce)
		if err != nil {
			return StatusResult{}, err
		}
		result.EnableTrading = validation
	}
	return result, nil
}

func (r *Runner) loadPrivateKey() (string, error) {
	if r.privateKey == nil {
		return "", fmt.Errorf("private key loader is required")
	}
	return r.privateKey()
}

func (r *Runner) relayerReader(ctx context.Context, privateKey string) (RelayerReader, error) {
	if r.relayer == nil {
		return nil, fmt.Errorf("relayer reader is required")
	}
	rc, err := r.relayer(ctx, privateKey)
	if err != nil {
		return nil, fmt.Errorf("init relayer client: %w", err)
	}
	return rc, nil
}

func (r *Runner) deploymentChecker() DeploymentChecker {
	if r.deployment != nil {
		return r.deployment
	}
	return contracts.DepositWalletDeployed
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

func deploymentStatusSource(relayerDeployed bool, onchainDeployed bool) string {
	if relayerDeployed {
		return "relayer"
	}
	if onchainDeployed {
		return "polygon_code"
	}
	return "relayer_and_polygon_code"
}
