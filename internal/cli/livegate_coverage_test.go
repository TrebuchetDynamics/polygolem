package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLiveMoneyCommandsFailClosed is the coverage net for the livegate seam: it
// enumerates every CLI command that signs or submits a live-money transaction
// and proves each one refuses a violating invocation BEFORE the private key is
// loaded. A cap-guarded order command must reject an over-cap notional; a
// confirm-guarded wallet command must reject a missing --confirm token.
//
// This is the regression guard for the class of bug the skeptical review found
// (batch-orders shipping without a cap). When you add a new live-money command,
// add a row here — a command that reaches signing without appearing in this
// table is exactly the gap this test exists to catch.
//
// deposit-wallet redeem is deliberately absent: its confirm gate fires only
// after fetching redeemable positions and printing the dry-run calldata (you
// cannot confirm a redeem before knowing what is redeemable), so it cannot
// fail-closed before the key loads like the commands below. Its confirm gate is
// covered by TestDepositWalletRedeemRequiresConfirm, which supplies the key and
// a mock Data API.
func TestLiveMoneyCommandsFailClosed(t *testing.T) {
	dir := t.TempDir()

	overCapOrders := filepath.Join(dir, "orders.json")
	// One order at 1.20 notional (0.60 * 2), above the default $1 cap.
	if err := os.WriteFile(overCapOrders, []byte(`[{"tokenID":"1","side":"buy","price":"0.60","size":"2"}]`), 0o600); err != nil {
		t.Fatalf("write orders file: %v", err)
	}

	cases := []struct {
		name       string
		args       []string
		wantErrSub string // proves the specific guard fired
	}{
		// Cap-guarded order commands (default cap = $1).
		{"exchange create-order", []string{"exchange", "create-order", "--token", "1", "--side", "buy", "--price", "0.60", "--size", "2"}, "POLYGOLEM_MAX_LIVE_ORDER_USD"},
		{"exchange market-order", []string{"exchange", "market-order", "--token", "1", "--amount", "2"}, "POLYGOLEM_MAX_LIVE_ORDER_USD"},
		{"exchange batch-orders", []string{"exchange", "batch-orders", "--orders-file", overCapOrders}, "POLYGOLEM_MAX_LIVE_ORDER_USD"},

		// Confirm-guarded deposit-wallet commands (no --confirm token supplied).
		{"wallet batch", []string{"wallet", "batch", "--calls-json", `[{"target":"0x0000000000000000000000000000000000000001","value":"0","data":"0x"}]`}, "SUBMIT_BATCH"},
		{"wallet approve --submit", []string{"wallet", "approve", "--submit"}, "APPROVE_TRADING"},
		{"wallet onboard", []string{"wallet", "onboard", "--skip-deploy"}, "ONBOARD_WALLET"},
		{"wallet approve-adapters --submit", []string{"wallet", "approve-adapters", "--submit"}, "APPROVE_ADAPTERS"},
		{"wallet approve-auto-redeem --submit", []string{"wallet", "approve-auto-redeem", "--submit"}, "APPROVE_AUTO_REDEEM"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Ensure the default cap applies and no key is set, so a guard that
			// failed to fire would surface as a private-key error instead.
			t.Setenv("POLYGOLEM_MAX_LIVE_ORDER_USD", "")
			t.Setenv("POLYMARKET_PRIVATE_KEY", "")
			t.Setenv("SIGNER_PRIVATE_KEY", "")

			_, _, err := executeRootForTest(tc.args...)
			if err == nil {
				t.Fatalf("%s must fail closed on a violating invocation", tc.name)
			}
			if !strings.Contains(err.Error(), tc.wantErrSub) {
				t.Fatalf("%s error must prove the guard fired (want %q): %v", tc.name, tc.wantErrSub, err)
			}
			if strings.Contains(err.Error(), "PRIVATE_KEY") {
				t.Fatalf("%s loaded the private key before its live-money guard: %v", tc.name, err)
			}
		})
	}
}
