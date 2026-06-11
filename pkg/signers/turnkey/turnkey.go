// Package turnkeysigner defines an optional Turnkey-style signer adapter seam.
//
// The package intentionally avoids importing the Turnkey SDK so the default
// polygolem module stays lightweight. Callers provide a Backend that performs
// Turnkey API calls and returns Ethereum-compatible signatures.
package turnkeysigner

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/polygolem/pkg/signers"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

const defaultTimeout = 10 * time.Second

// Backend is implemented by a Turnkey API adapter owned by the caller.
// Implementations must keep organization/user credentials out of logs.
type Backend interface {
	SignHash(ctx context.Context, organizationID, walletID, address string, hash [32]byte) ([]byte, error)
	SignTypedData(ctx context.Context, organizationID, walletID, address string, domainHash, structHash [32]byte) ([32]byte, error)
	SignEIP712(ctx context.Context, organizationID, walletID, address string, typed apitypes.TypedData) ([]byte, error)
}

// Config identifies a Turnkey-controlled Ethereum account.
type Config struct {
	OrganizationID string
	WalletID       string
	Address        string
	ChainID        int64
	Timeout        time.Duration
	Backend        Backend
}

// Signer implements signers.Signer through a caller-provided Turnkey backend.
type Signer struct {
	config Config
}

var _ signers.Signer = (*Signer)(nil)

// New constructs a Turnkey-style signer. No Turnkey SDK or credentials are
// loaded by this package.
func New(config Config) (*Signer, error) {
	if strings.TrimSpace(config.OrganizationID) == "" {
		return nil, fmt.Errorf("turnkey organization id is required")
	}
	if strings.TrimSpace(config.WalletID) == "" {
		return nil, fmt.Errorf("turnkey wallet id is required")
	}
	if strings.TrimSpace(config.Address) == "" {
		return nil, fmt.Errorf("turnkey address is required")
	}
	if config.Backend == nil {
		return nil, fmt.Errorf("turnkey backend is required")
	}
	if config.Timeout <= 0 {
		config.Timeout = defaultTimeout
	}
	return &Signer{config: config}, nil
}

func (s *Signer) Address() string { return s.config.Address }
func (s *Signer) ChainID() int64  { return s.config.ChainID }

func (s *Signer) SignHash(hash [32]byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.config.Timeout)
	defer cancel()
	return s.config.Backend.SignHash(ctx, s.config.OrganizationID, s.config.WalletID, s.config.Address, hash)
}

func (s *Signer) SignTypedData(domainHash, structHash [32]byte) ([32]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.config.Timeout)
	defer cancel()
	return s.config.Backend.SignTypedData(ctx, s.config.OrganizationID, s.config.WalletID, s.config.Address, domainHash, structHash)
}

func (s *Signer) SignEIP712(typed apitypes.TypedData) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.config.Timeout)
	defer cancel()
	return s.config.Backend.SignEIP712(ctx, s.config.OrganizationID, s.config.WalletID, s.config.Address, typed)
}
