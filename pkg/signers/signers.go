// Package signers defines public signing interfaces and safe local adapters
// for Polymarket auth, order, and deposit-wallet typed-data flows.
//
// The default polygolem CLI still signs locally from POLYMARKET_PRIVATE_KEY.
// This package exists so SDK consumers can depend on a stable seam before
// optional HTTP/KMS/Turnkey adapters are added. Remote/cloud adapters should
// stay optional and must preserve timeout and redaction guarantees.
package signers

import (
	"github.com/TrebuchetDynamics/polygolem/internal/auth"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

// Signer is the stable public signing seam for SDK consumers.
//
// Implementations must never log private material. SignHash and SignTypedData
// preserve the lower-level interface used by current auth flows; SignEIP712
// signs canonical go-ethereum typed data and returns a full 65-byte Ethereum
// signature with a 27/28 recovery byte.
type Signer interface {
	Address() string
	ChainID() int64
	SignHash(hash [32]byte) ([]byte, error)
	SignTypedData(domainHash, structHash [32]byte) ([32]byte, error)
	SignEIP712(typed apitypes.TypedData) ([]byte, error)
}

// LocalSigner implements Signer with an in-process secp256k1 private key.
// Prefer this adapter unless an operator explicitly needs external custody.
type LocalSigner struct {
	inner *auth.PrivateKeySigner
}

// NewLocalSigner creates a local signer from a 0x-prefixed or bare hex private
// key. The private key is parsed once and is never exposed by this type.
func NewLocalSigner(privateKeyHex string, chainID int64) (*LocalSigner, error) {
	inner, err := auth.NewPrivateKeySigner(privateKeyHex, chainID)
	if err != nil {
		return nil, err
	}
	return &LocalSigner{inner: inner}, nil
}

// NewLocal is a short alias for NewLocalSigner.
func NewLocal(privateKeyHex string, chainID int64) (*LocalSigner, error) {
	return NewLocalSigner(privateKeyHex, chainID)
}

func (s *LocalSigner) Address() string { return s.inner.Address() }
func (s *LocalSigner) ChainID() int64  { return s.inner.ChainID() }

func (s *LocalSigner) SignHash(hash [32]byte) ([]byte, error) {
	return s.inner.SignHash(hash)
}

func (s *LocalSigner) SignTypedData(domainHash, structHash [32]byte) ([32]byte, error) {
	return s.inner.SignTypedData(domainHash, structHash)
}

func (s *LocalSigner) SignEIP712(typed apitypes.TypedData) ([]byte, error) {
	return s.inner.SignEIP712(typed)
}

// RedactSecret returns the same redacted representation used by internal auth
// logging. It is provided for adapter authors and tests.
func RedactSecret(value string) string {
	return auth.Redact(value)
}
