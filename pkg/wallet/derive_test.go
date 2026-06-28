package wallet

import (
	"strings"
	"testing"
)

func TestDeriveDepositWallet(t *testing.T) {
	eoa := "0x2c7536E3605D9C16a7a3D7b1898e529396a65c23"
	result, err := DeriveDepositWallet(eoa)
	if err != nil {
		t.Fatal(err)
	}
	want := "0xfd5041047Be8C192C725A66228F141196Fa3cF9C"
	if result != want {
		t.Fatalf("deposit wallet = %s, want %s", result, want)
	}
}

func TestDeriveDepositWalletRejectsInvalidEOA(t *testing.T) {
	if _, err := DeriveDepositWallet("not-an-address"); err == nil {
		t.Fatal("expected invalid EOA error")
	}
}

func TestDeriveProxyWallet(t *testing.T) {
	eoa := "0x2c7536E3605D9C16a7a3D7b1898e529396a65c23"
	result := DeriveProxyWallet(eoa)
	want := "0x96a9892de6a11fe0b18cf63373b9763055eca8a6"
	if result != want {
		t.Fatalf("proxy wallet = %s, want %s", result, want)
	}
}

func TestDeriveSafeWallet(t *testing.T) {
	eoa := "0x2c7536E3605D9C16a7a3D7b1898e529396a65c23"
	result := DeriveSafeWallet(eoa)
	want := "0x907c14d6cea8e8fc78dd3db152f0a93f43276b4d"
	if result != want {
		t.Fatalf("safe wallet = %s, want %s", result, want)
	}
}

func TestProxyAndSafeAreDifferent(t *testing.T) {
	eoa := "0x2c7536E3605D9C16a7a3D7b1898e529396a65c23"
	proxy := DeriveProxyWallet(eoa)
	safe := DeriveSafeWallet(eoa)
	if proxy == safe {
		t.Fatal("proxy and safe should be different addresses")
	}
}

func TestDeriveDeterministic(t *testing.T) {
	eoa := "0x2c7536E3605D9C16a7a3D7b1898e529396a65c23"
	a := DeriveProxyWallet(eoa)
	b := DeriveProxyWallet(eoa)
	if a != b {
		t.Fatal("derivation should be deterministic")
	}
}

func TestReadiness(t *testing.T) {
	info := Readiness(137, "0x2c7536E3605D9C16a7a3D7b1898e529396a65c23")
	if !info.HasSigner {
		t.Fatal("should have signer")
	}
	if info.ChainID != 137 {
		t.Fatalf("chainID = %d", info.ChainID)
	}
	if info.DepositWallet != "0xfd5041047Be8C192C725A66228F141196Fa3cF9C" {
		t.Fatalf("deposit wallet = %s", info.DepositWallet)
	}
}

func TestReadinessEmptyEOA(t *testing.T) {
	info := Readiness(137, "")
	if info.HasSigner {
		t.Fatal("should not have signer with empty EOA")
	}
}

// TestDeriveDoesNotPanicOnInvalidHexEOA guards the regression where a 40-char
// non-hex EOA panicked in hexToBytes via proxySalt/safeSalt.
func TestDeriveDoesNotPanicOnInvalidHexEOA(t *testing.T) {
	bad := "0x" + strings.Repeat("z", 40) // 40 chars, not hex
	if got := DeriveProxyWallet(bad); got == "" {
		t.Fatal("DeriveProxyWallet returned empty")
	}
	if got := DeriveSafeWallet(bad); got == "" {
		t.Fatal("DeriveSafeWallet returned empty")
	}
	// Readiness must also not panic.
	_ = Readiness(PolygonChainID, bad)
}
