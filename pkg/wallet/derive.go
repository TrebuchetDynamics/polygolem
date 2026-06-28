package wallet

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/polygolem/internal/auth"
	"golang.org/x/crypto/sha3"
)

// Contract addresses for Polygon mainnet (chainID 137).
const (
	ProxyFactoryAddr = "0xaB45c5A4B0c941a2F231C04C3f49182e1A254052"
	SafeFactoryAddr  = "0xaacFeEa03eb1561C4e67d661e40682Bd20E3541b"
	PolygonChainID   = 137

	// CREATE2 init code hashes
	proxyInitCodeHash = "0xd21df8dc65880a8606f09fe0ce3df9b8869287ab0b058be05aa9e8af6330a00b"
	safeInitCodeHash  = "0x2bce2127ff07fb632d16c8347c4ebf501f4841168bed00d9e6ef715ddb6fcecf"
)

// DeriveDepositWallet computes the deterministic Polymarket V2 deposit-wallet
// address for an EOA. This is the canonical wallet for POLY_1271 trading.
func DeriveDepositWallet(eoa string) (string, error) {
	if !validHexAddress(eoa) {
		return "", fmt.Errorf("invalid EOA address")
	}
	return auth.MakerAddressForSignatureType(eoa, PolygonChainID, 3)
}

// DeriveProxyWallet computes the deterministic proxy wallet address via CREATE2.
// Deprecated: Polygolem supports deposit-wallet / POLY_1271 trading only.
func DeriveProxyWallet(eoa string) string {
	return deriveCreate2(ProxyFactoryAddr, proxySalt(eoa), proxyInitCodeHash)
}

// DeriveSafeWallet computes the deterministic Gnosis Safe address via CREATE2.
// Deprecated: Polygolem supports deposit-wallet / POLY_1271 trading only.
func DeriveSafeWallet(eoa string) string {
	return deriveCreate2(SafeFactoryAddr, safeSalt(eoa), safeInitCodeHash)
}

func proxySalt(eoa string) []byte {
	clean := strip0x(eoa)
	b, err := hex.DecodeString(clean)
	if len(clean) != 40 || err != nil {
		// Invalid (wrong-length or non-hex) EOA: zero salt rather than panic.
		return make([]byte, 32)
	}
	return keccak256(b)
}

func safeSalt(eoa string) []byte {
	clean := strip0x(eoa)
	b, err := hex.DecodeString(clean)
	if len(clean) != 40 || err != nil {
		// Invalid (wrong-length or non-hex) EOA: zero salt rather than panic.
		return make([]byte, 32)
	}
	padded := make([]byte, 32)
	copy(padded[12:], b)
	return keccak256(padded)
}

func keccak256(b []byte) []byte {
	hash := sha3.NewLegacyKeccak256()
	hash.Write(b)
	return hash.Sum(nil)
}

// deriveCreate2 implements EIP-1014 CREATE2 address derivation.
// address = keccak256(0xff + factory + salt + keccak256(initCode))[12:]
func deriveCreate2(factory string, salt []byte, initCodeHash string) string {
	hash := sha3.NewLegacyKeccak256()
	hash.Write([]byte{0xff})
	hash.Write(hexToBytes(strip0x(factory)))
	hash.Write(salt)
	hash.Write(hexToBytes(strip0x(initCodeHash)))
	result := hash.Sum(nil)
	return "0x" + toHex(result[12:])
}

// ReadyInfo holds wallet readiness information.
type ReadyInfo struct {
	ChainID       int64  `json:"chain_id"`
	EOA           string `json:"eoa,omitempty"`
	DepositWallet string `json:"deposit_wallet,omitempty"`
	ProxyWallet   string `json:"proxy_wallet,omitempty"`
	SafeWallet    string `json:"safe_wallet,omitempty"`
	HasSigner     bool   `json:"has_signer"`
}

// Readiness returns wallet readiness info for the given EOA.
func Readiness(chainID int64, eoa string) ReadyInfo {
	info := ReadyInfo{
		ChainID: chainID,
		EOA:     eoa,
	}
	if eoa != "" {
		info.HasSigner = true
		info.DepositWallet, _ = DeriveDepositWallet(eoa)
		info.ProxyWallet = DeriveProxyWallet(eoa)
		info.SafeWallet = DeriveSafeWallet(eoa)
	}
	return info
}

// helpers

func validHexAddress(s string) bool {
	s = strip0x(s)
	if len(s) != 40 {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}

func strip0x(s string) string {
	if len(s) >= 2 && strings.EqualFold(s[:2], "0x") {
		return s[2:]
	}
	return s
}

func hexToBytes(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(fmt.Sprintf("hexToBytes: invalid hex %q: %v", s, err))
	}
	return b
}

func toHex(b []byte) string {
	h := ""
	for _, v := range b {
		h += string("0123456789abcdef"[v>>4])
		h += string("0123456789abcdef"[v&0xf])
	}
	return h
}
