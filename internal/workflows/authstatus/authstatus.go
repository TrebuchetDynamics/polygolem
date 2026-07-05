// Package authstatus owns read-only authentication readiness orchestration.
package authstatus

import (
	"context"
	"fmt"

	"github.com/TrebuchetDynamics/polygolem/internal/auth"
	"github.com/TrebuchetDynamics/polygolem/internal/clob"
)

const onboardingHelpURL = "https://github.com/TrebuchetDynamics/polygolem/blob/main/docs/ONBOARDING.md"

// PrivateKeyLoader loads the signer key from the caller's environment.
type PrivateKeyLoader func() (string, error)

// RelayerFactory builds the relayer reader used for deployment status.
type RelayerFactory func(context.Context) (RelayerReader, error)

// L2CredentialsProvider returns configured CLOB L2 credentials, when present.
type L2CredentialsProvider func() (auth.APIKey, bool)

// RelayerReader is the read-only relayer surface used by auth status.
type RelayerReader interface {
	IsDeployed(context.Context, string) (bool, error)
}

// CLOBReader is the read-only CLOB surface used by auth status.
type CLOBReader interface {
	DeriveAPIKey(context.Context, string) (auth.APIKey, error)
	SetL2Credentials(auth.APIKey)
	ListOrders(context.Context, string) ([]clob.OrderRecord, error)
}

// Config contains adapters used by the auth-status workflow.
type Config struct {
	PrivateKey    PrivateKeyLoader
	Relayer       RelayerFactory
	CLOB          CLOBReader
	L2Credentials L2CredentialsProvider
}

// Runner owns auth-status orchestration behind a small interface.
type Runner struct {
	privateKey    PrivateKeyLoader
	relayer       RelayerFactory
	clob          CLOBReader
	l2Credentials L2CredentialsProvider
}

// Request configures an auth-status read.
type Request struct {
	CheckDepositKey bool
}

// Status is the JSON payload for auth status.
type Status struct {
	EOAAddress                string `json:"eoaAddress"`
	DepositWallet             string `json:"depositWallet"`
	DepositWalletDeployed     bool   `json:"depositWalletDeployed"`
	EOAAPIKeyExists           bool   `json:"eoaApiKeyExists"`
	DepositWalletAPIKeyExists bool   `json:"depositWalletApiKeyExists"`
	CanTrade                  bool   `json:"canTrade"`
	NextStep                  string `json:"nextStep,omitempty"`
	Help                      string `json:"help,omitempty"`
}

// New creates an auth-status workflow runner.
func New(cfg Config) *Runner {
	return &Runner{
		privateKey:    cfg.PrivateKey,
		relayer:       cfg.Relayer,
		clob:          cfg.CLOB,
		l2Credentials: cfg.L2Credentials,
	}
}

// Status reports authentication readiness for the configured EOA and deposit wallet.
func (r *Runner) Status(ctx context.Context, req Request) (Status, error) {
	privateKey, err := r.loadPrivateKey()
	if err != nil {
		return Status{}, err
	}
	owner, depositWallet, err := ownerAndWallet(privateKey)
	if err != nil {
		return Status{}, err
	}

	deployed := r.depositWalletDeployed(ctx, owner)
	clobClient, err := r.clobReader()
	if err != nil {
		return Status{}, err
	}
	eoaKeyExists := false
	if _, err := clobClient.DeriveAPIKey(ctx, privateKey); err == nil {
		eoaKeyExists = true
	}

	depositKeyExists := false
	if r.l2Credentials != nil {
		if key, ok := r.l2Credentials(); ok {
			clobClient.SetL2Credentials(key)
			depositKeyExists = key.Validate() == nil
		}
	}
	if deployed && req.CheckDepositKey {
		_, err := clobClient.ListOrders(ctx, privateKey)
		depositKeyExists = err == nil
	}

	out := Status{
		EOAAddress:                owner,
		DepositWallet:             depositWallet,
		DepositWalletDeployed:     deployed,
		EOAAPIKeyExists:           eoaKeyExists,
		DepositWalletAPIKeyExists: depositKeyExists,
		CanTrade:                  deployed && depositKeyExists,
	}
	if !out.CanTrade {
		if !deployed {
			out.NextStep = "Run: polygolem deposit-wallet deploy --wait"
		} else if !depositKeyExists {
			out.NextStep = "Run: polygolem auth login, then polygolem builder auto or polygolem clob create-api-key"
		}
		out.Help = onboardingHelpURL
	}
	return out, nil
}

func (r *Runner) loadPrivateKey() (string, error) {
	if r.privateKey == nil {
		return "", fmt.Errorf("private key loader is required")
	}
	return r.privateKey()
}

func (r *Runner) clobReader() (CLOBReader, error) {
	if r.clob == nil {
		return nil, fmt.Errorf("CLOB reader is required")
	}
	return r.clob, nil
}

func (r *Runner) depositWalletDeployed(ctx context.Context, owner string) bool {
	if r.relayer == nil {
		return false
	}
	rc, err := r.relayer(ctx)
	if err != nil || rc == nil {
		return false
	}
	deployed, err := rc.IsDeployed(ctx, owner)
	if err != nil {
		return false
	}
	return deployed
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
