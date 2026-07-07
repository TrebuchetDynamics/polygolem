package relayer

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/polygolem/pkg/contracts"
)

func TestBuildAdapterApprovalCallsCalldata(t *testing.T) {
	calls := BuildAdapterApprovalCalls()
	if len(calls) != 4 {
		t.Fatalf("len=%d want 4", len(calls))
	}

	// Calls 0-1: CtfCollateralAdapter
	assertApprove(t, "call0", calls[0], contracts.PUSD, contracts.CtfCollateralAdapter)
	assertSetApprovalForAll(t, "call1", calls[1], contracts.CTF, contracts.CtfCollateralAdapter)

	// Calls 2-3: NegRiskCtfCollateralAdapter
	assertApprove(t, "call2", calls[2], contracts.PUSD, contracts.NegRiskCtfCollateralAdapter)
	assertSetApprovalForAll(t, "call3", calls[3], contracts.CTF, contracts.NegRiskCtfCollateralAdapter)
}

func TestBuildAutoRedeemApprovalCallsCalldata(t *testing.T) {
	calls := BuildAutoRedeemApprovalCalls()
	if len(calls) != 3 {
		t.Fatalf("len=%d want 3", len(calls))
	}

	assertSetApprovalForAll(t, "call0", calls[0], contracts.CTF, contracts.CtfAutoRedeem)
	assertSetApprovalForAll(t, "call1", calls[1], contracts.CTF, contracts.AutoRedeemer)
	assertSetApprovalForAll(t, "call2", calls[2], contracts.PositionManager, contracts.AutoRedeemer)
}

// TestBuildAutoRedeemApprovalCallsMatchesObservedBatch pins the exact
// calldata against a WALLET batch captured from polymarket.com's
// "Get Paid Instantly" enablement flow (2026-07-06).
func TestBuildAutoRedeemApprovalCallsMatchesObservedBatch(t *testing.T) {
	observed := []struct{ target, data string }{
		{
			"0x4D97DCd97eC945f40cF65F87097ACe5EA0476045",
			"0xa22cb465000000000000000000000000f3cfb6a6ebfeb51876289eb235719eb1c65252b00000000000000000000000000000000000000000000000000000000000000001",
		},
		{
			"0x4D97DCd97eC945f40cF65F87097ACe5EA0476045",
			"0xa22cb465000000000000000000000000a1200000d0002264c9a1698e001292d00e1b00af0000000000000000000000000000000000000000000000000000000000000001",
		},
		{
			"0x006F54F7f9A22e0000CC2AB60031000000ae9fEF",
			"0xa22cb465000000000000000000000000a1200000d0002264c9a1698e001292d00e1b00af0000000000000000000000000000000000000000000000000000000000000001",
		},
	}
	calls := BuildAutoRedeemApprovalCalls()
	if len(calls) != len(observed) {
		t.Fatalf("len=%d want %d", len(calls), len(observed))
	}
	for i, want := range observed {
		if !strings.EqualFold(calls[i].Target, want.target) {
			t.Errorf("call%d target=%s want %s", i, calls[i].Target, want.target)
		}
		if !strings.EqualFold(calls[i].Data, want.data) {
			t.Errorf("call%d data=%s want %s", i, calls[i].Data, want.data)
		}
		if calls[i].Value != "0" {
			t.Errorf("call%d value=%s want 0", i, calls[i].Value)
		}
	}
}

// TestBuildEnableTradingApprovalCallsMatchesObservedBatch pins the exact
// calldata against a WALLET batch captured from polymarket.com's Enable
// Trading "Approve Tokens" flow (2026-07-06).
func TestBuildEnableTradingApprovalCallsMatchesObservedBatch(t *testing.T) {
	observed := []struct{ target, data string }{
		{
			"0xC011a7E12a19f7B1f670d46F03B03f3342E82DFB",
			"0x095ea7b30000000000000000000000004d97dcd97ec945f40cf65f87097ace5ea0476045ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
		{
			"0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174",
			"0x095ea7b300000000000000000000000093070a847efef7f70739046a929d47a521f5b8eeffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
		{
			"0x006F54F7f9A22e0000CC2AB60031000000ae9fEF",
			"0xa22cb46500000000000000000000000012121212006e4cd160d18e3f00711da5c33726000000000000000000000000000000000000000000000000000000000000000001",
		},
		{
			"0xC011a7E12a19f7B1f670d46F03B03f3342E82DFB",
			"0x095ea7b3000000000000000000000000e3333700ca9d93003f00f0f71f8515005f6c00aaffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
		{
			"0xC011a7E12a19f7B1f670d46F03B03f3342E82DFB",
			"0x095ea7b300000000000000000000000012121212006e4cd160d18e3f00711da5c3372600ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
		{
			"0x006F54F7f9A22e0000CC2AB60031000000ae9fEF",
			"0xa22cb465000000000000000000000000e3333700ca9d93003f00f0f71f8515005f6c00aa0000000000000000000000000000000000000000000000000000000000000001",
		},
	}
	calls := BuildEnableTradingApprovalCalls()
	if len(calls) != len(observed) {
		t.Fatalf("len=%d want %d", len(calls), len(observed))
	}
	for i, want := range observed {
		if !strings.EqualFold(calls[i].Target, want.target) {
			t.Errorf("call%d target=%s want %s", i, calls[i].Target, want.target)
		}
		if !strings.EqualFold(calls[i].Data, want.data) {
			t.Errorf("call%d data=%s want %s", i, calls[i].Data, want.data)
		}
		if calls[i].Value != "0" {
			t.Errorf("call%d value=%s want 0", i, calls[i].Value)
		}
	}
}

func TestBuildAdapterApprovalCallsIdempotent(t *testing.T) {
	a := BuildAdapterApprovalCalls()
	b := BuildAdapterApprovalCalls()
	if len(a) != len(b) {
		t.Fatalf("len mismatch a=%d b=%d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("call %d differs", i)
		}
	}
}

func assertApprove(t *testing.T, label string, call DepositWalletCall, expectToken, expectSpender string) {
	t.Helper()
	if !strings.EqualFold(call.Target, expectToken) {
		t.Errorf("%s target=%s want %s", label, call.Target, expectToken)
	}
	data := strings.ToLower(call.Data)
	if !strings.HasPrefix(data, "0x"+erc20ApproveSelector) {
		t.Errorf("%s data does not start with approve selector: %s", label, data[:10])
	}
	wantSpender := strings.ToLower(strings.TrimPrefix(expectSpender, "0x"))
	if !strings.Contains(data, wantSpender) {
		t.Errorf("%s spender %s not encoded in calldata: %s", label, wantSpender, data)
	}
	if !strings.HasSuffix(data, maxUint256) {
		t.Errorf("%s amount is not MaxUint256: %s", label, data)
	}
}

func assertSetApprovalForAll(t *testing.T, label string, call DepositWalletCall, expectCTF, expectOperator string) {
	t.Helper()
	if !strings.EqualFold(call.Target, expectCTF) {
		t.Errorf("%s target=%s want %s", label, call.Target, expectCTF)
	}
	data := strings.ToLower(call.Data)
	if !strings.HasPrefix(data, "0x"+erc1155SetApprovalForAllSel) {
		t.Errorf("%s data does not start with setApprovalForAll selector: %s", label, data[:10])
	}
	wantOp := strings.ToLower(strings.TrimPrefix(expectOperator, "0x"))
	if !strings.Contains(data, wantOp) {
		t.Errorf("%s operator %s not encoded in calldata: %s", label, wantOp, data)
	}
	if !strings.HasSuffix(data, "0000000000000000000000000000000000000000000000000000000000000001") {
		t.Errorf("%s approved=true not encoded: %s", label, data)
	}
}

// TestPad32BytesDoesNotPanicOnOverlongInput guards the regression where a hex
// string longer than 64 chars caused strings.Repeat to panic with a negative
// count. Over-long input keeps the low-order 32 bytes; the result is always a
// 64-char word.
func TestPad32BytesDoesNotPanicOnOverlongInput(t *testing.T) {
	cases := []string{
		"abc",                          // short → left-padded
		strings.Repeat("f", 64),        // exact 32 bytes
		"0x" + strings.Repeat("a", 80), // over-long (was a panic)
	}
	for _, in := range cases {
		got := pad32Bytes(in)
		if len(got) != 64 {
			t.Fatalf("pad32Bytes(%q) length = %d, want 64", in, len(got))
		}
	}
	// Over-long input keeps the low-order 64 hex chars.
	over := strings.Repeat("a", 80)
	if got := pad32Bytes(over); got != over[len(over)-64:] {
		t.Fatalf("pad32Bytes overlong = %q, want low 64 chars", got)
	}
}
