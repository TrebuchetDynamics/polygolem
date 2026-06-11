// Package kmssigner defines an optional KMS-style signing adapter boundary.
//
// It intentionally does not import an AWS/GCP/HSM SDK. Operators can implement
// Backend with their custody provider of choice and keep cloud dependencies out
// of the default polygolem binary.
package kmssigner

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/polygolem/pkg/signers"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

const defaultTimeout = 10 * time.Second

// Backend is the provider-specific custody implementation. Methods must return
// Ethereum-compatible signatures where applicable: 65 bytes with recovery byte
// 27/28 for SignHash and SignEIP712.
type Backend interface {
	SignHash(ctx context.Context, keyID string, hash [32]byte) ([]byte, error)
	SignTypedData(ctx context.Context, keyID string, domainHash, structHash [32]byte) ([32]byte, error)
	SignEIP712(ctx context.Context, keyID string, typed apitypes.TypedData) ([]byte, error)
}

// Config identifies the remote custody key and its public Ethereum identity.
type Config struct {
	KeyID   string
	Address string
	ChainID int64
	Timeout time.Duration
	Backend Backend
}

// Signer implements signers.Signer through a caller-provided Backend.
type Signer struct {
	config Config
}

var _ signers.Signer = (*Signer)(nil)

// New constructs a KMS-style signer adapter. No provider SDK is initialized
// here; callers own Backend construction and credential loading.
func New(config Config) (*Signer, error) {
	if strings.TrimSpace(config.KeyID) == "" {
		return nil, fmt.Errorf("kms signer key id is required")
	}
	if config.Backend == nil {
		return nil, fmt.Errorf("kms signer backend is required")
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
	return s.config.Backend.SignHash(ctx, s.config.KeyID, hash)
}

func (s *Signer) SignTypedData(domainHash, structHash [32]byte) ([32]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.config.Timeout)
	defer cancel()
	return s.config.Backend.SignTypedData(ctx, s.config.KeyID, domainHash, structHash)
}

func (s *Signer) SignEIP712(typed apitypes.TypedData) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.config.Timeout)
	defer cancel()
	return s.config.Backend.SignEIP712(ctx, s.config.KeyID, typed)
}
