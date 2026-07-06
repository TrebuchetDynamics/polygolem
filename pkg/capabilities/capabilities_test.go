package capabilities

import "testing"

func TestAllIncludesCriticalPolymarketSurfaces(t *testing.T) {
	got := ByID()
	for _, id := range []string{
		"gamma.markets",
		"clob.public_data",
		"clob.trading",
		"data.positions",
		"relayer.deposit_wallet",
		"bridge.funding",
		"websocket.market",
		"websocket.user",
	} {
		if _, ok := got[id]; !ok {
			t.Fatalf("missing capability %s", id)
		}
	}
}

func TestTradingCapabilitiesDeclareAuthAndMutation(t *testing.T) {
	clob := ByID()["clob.trading"]
	if !clob.Mutating {
		t.Fatalf("clob trading must be marked mutating")
	}
	if !clob.Requires(AuthL1) || !clob.Requires(AuthL2) || !clob.Requires(AuthPrivateKey) {
		t.Fatalf("clob trading auth=%v, want L1/L2/private-key", clob.Auth)
	}
	if clob.WalletMode != WalletDepositOnly {
		t.Fatalf("clob trading wallet mode=%q, want deposit-only", clob.WalletMode)
	}
}

func TestReadOnlyCapabilitiesExcludeSecrets(t *testing.T) {
	for _, cap := range All() {
		if !cap.ReadOnly {
			continue
		}
		if cap.Mutating {
			t.Fatalf("%s is both read-only and mutating", cap.ID)
		}
		if cap.Requires(AuthPrivateKey) || cap.Requires(AuthL2) || cap.Requires(AuthSIWE) {
			t.Fatalf("read-only capability %s requires secret-bearing auth: %v", cap.ID, cap.Auth)
		}
	}
}

func TestAllIsDeterministic(t *testing.T) {
	caps := All()
	for i := 1; i < len(caps); i++ {
		if caps[i-1].ID > caps[i].ID {
			t.Fatalf("capabilities not sorted: %s before %s", caps[i-1].ID, caps[i].ID)
		}
	}
}
