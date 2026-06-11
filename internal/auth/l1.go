package auth

import (
	"fmt"
	"math/big"
	"strconv"
	"time"

	gethmath "github.com/ethereum/go-ethereum/common/math"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

// BuildL1HeadersFromPrivateKey builds Polymarket CLOB L1 auth headers for
// API-key creation and derivation. It signs the canonical ClobAuth EIP-712
// message with a local EOA private key. POLY_ADDRESS is the EOA address.
func BuildL1HeadersFromPrivateKey(privateKeyHex string, chainID int64, timestamp int64, nonce int64) (map[string]string, error) {
	return BuildL1HeadersForAddress(privateKeyHex, chainID, timestamp, nonce, "")
}

// BuildL1HeadersForDepositWallet is retained only as a defensive compatibility
// guard for the obsolete hypothesis that CLOB L1 auth should be ERC-7739 wrapped
// and bound to the deposit wallet address.
//
// Deprecated: the validated V2 path uses EOA-bound CLOB L1/L2 auth via
// BuildL1HeadersFromPrivateKey or BuildL1HeadersForAddress. Deposit-wallet
// identity is carried by POLY_1271 order fields and ERC-7739 order signatures,
// not by deposit-wallet-bound ClobAuth headers.
func BuildL1HeadersForDepositWallet(privateKeyHex string, chainID int64, timestamp int64, nonce int64, depositWalletAddress string) (map[string]string, error) {
	if depositWalletAddress == "" {
		return nil, fmt.Errorf("depositWalletAddress is required")
	}
	return nil, fmt.Errorf("deposit-wallet-bound ERC-7739 ClobAuth is unsupported; use EOA-bound BuildL1HeadersFromPrivateKey or BuildL1HeadersForAddress")
}

// BuildL1HeadersForAddress builds raw EOA-signed ClobAuth headers. When
// ownerAddress is non-empty it overrides both POLY_ADDRESS and the typed-data
// value.address, but the signature remains a raw EOA signature.
//
// Deposit-wallet trading flows should also use EOA-bound L1 auth; the deposit
// wallet identity belongs in POLY_1271 order fields rather than ClobAuth.
func BuildL1HeadersForAddress(privateKeyHex string, chainID int64, timestamp int64, nonce int64, ownerAddress string) (map[string]string, error) {
	signer, err := NewPrivateKeySigner(privateKeyHex, chainID)
	if err != nil {
		return nil, err
	}
	if timestamp == 0 {
		timestamp = time.Now().Unix()
	}
	polyAddress := ownerAddress
	if polyAddress == "" {
		polyAddress = signer.Address()
	}
	typed := apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": {
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
			},
			"ClobAuth": {
				{Name: "address", Type: "address"},
				{Name: "timestamp", Type: "string"},
				{Name: "nonce", Type: "uint256"},
				{Name: "message", Type: "string"},
			},
		},
		PrimaryType: "ClobAuth",
		Domain: apitypes.TypedDataDomain{
			Name:    clobAuthDomainName,
			Version: clobAuthDomainVersion,
			ChainId: (*gethmath.HexOrDecimal256)(big.NewInt(chainID)),
		},
		Message: apitypes.TypedDataMessage{
			"address":   polyAddress,
			"timestamp": strconv.FormatInt(timestamp, 10),
			"nonce":     (*gethmath.HexOrDecimal256)(big.NewInt(nonce)),
			"message":   clobAuthDefaultMessage,
		},
	}
	hash, _, err := apitypes.TypedDataAndHash(typed)
	if err != nil {
		return nil, err
	}
	sig, err := ethcrypto.Sign(hash, signer.key)
	if err != nil {
		return nil, err
	}
	if sig[64] < 27 {
		sig[64] += 27
	}
	return map[string]string{
		"POLY_ADDRESS":   polyAddress,
		"POLY_SIGNATURE": fmt.Sprintf("0x%x", sig),
		"POLY_TIMESTAMP": strconv.FormatInt(timestamp, 10),
		"POLY_NONCE":     strconv.FormatInt(nonce, 10),
	}, nil
}
