package auth

import (
	"strings"
	"testing"
)

// TestDeriveDepositWalletAddress_BeaconProxy pins the sigtype-3 deposit
// wallet derivation against a REAL on-chain pair created by Polymarket's
// production DepositWalletFactory on 2026-07-05:
//
//	owner  0xAebC1bD62A183FCcd576b2ddC227624e31a0e0C3
//	wallet 0x60Ea46AF8d0EF3C306D7D57C7C4F94eb0C2dE7b7
//
// Evidence: WALLET-CREATE tx
// 0xea02be349844667ea7d82ff413d7e7e50cdd2fc8ccbcb6e98a3e4051fd6a549f
// (Polygon block 89709180) — the factory event names this wallet+owner, the
// proxy's EIP-1967 beacon slot holds
// 0x7a18EDfE055488A3128f01F563E5B479D92FFc3A, and the relayer accepted a
// WALLET batch for this wallet (mined
// 0x4b4cefd8d088ee0dc3c7fc9b4368e481888ac217a44ad03db934afaae8478470).
//
// The factory deploys ERC-1967 BEACON proxies (runtime `363d3d373d3d363d
// 6020366004 36635c60da1b…` staticcalling implementation() on the beacon
// read from the beacon slot). The old UUPS template (embedded
// implementation) derives 0xcE5718c85BD70913011ddc36f4CA92188Af9728F for
// this owner — an address with no code that Polymarket's registry rejects.
func TestDeriveDepositWalletAddress_BeaconProxy(t *testing.T) {
	got, err := MakerAddressForSignatureType("0xAebC1bD62A183FCcd576b2ddC227624e31a0e0C3", 137, 3)
	if err != nil {
		t.Fatalf("MakerAddressForSignatureType: %v", err)
	}
	want := "0x60Ea46AF8d0EF3C306D7D57C7C4F94eb0C2dE7b7"
	if !strings.EqualFold(got, want) {
		t.Fatalf("deposit wallet derivation mismatch:\n got  %s\n want %s (on-chain factory event)", got, want)
	}
	// The stale UUPS derivation must NOT come back: orders carrying that
	// maker reference a codeless address the registry rejects.
	if strings.EqualFold(got, "0xcE5718c85BD70913011ddc36f4CA92188Af9728F") {
		t.Fatal("derivation regressed to the pre-2026 UUPS template address")
	}
}
