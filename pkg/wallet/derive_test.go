package wallet

import (
	"testing"
)

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
	if info.ProxyWallet == "" {
		t.Fatal("proxy wallet missing")
	}
	if info.SafeWallet == "" {
		t.Fatal("safe wallet missing")
	}
}

func TestReadinessEmptyEOA(t *testing.T) {
	info := Readiness(137, "")
	if info.HasSigner {
		t.Fatal("should not have signer with empty EOA")
	}
}
