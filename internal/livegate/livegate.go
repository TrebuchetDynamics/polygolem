// Package livegate centralizes the guards that must fire before any polygolem
// command signs or submits a live-money Polymarket transaction: the per-order
// notional cap and the typed confirmation token. Keeping both in one place gives
// a new mutating command a single, discoverable guard to reach for, instead of
// re-deriving ad-hoc checks (the gap that let clob batch-orders ship without a
// cap). Callers must invoke the relevant guard before loading the private key.
package livegate

import (
	"fmt"
	"math/big"
	"os"
	"strings"
)

// MaxLiveOrderEnvVar caps the notional of a single live order and the summed
// notional of a batch. Operators raise it deliberately; there is no flag that
// bypasses it.
const MaxLiveOrderEnvVar = "POLYGOLEM_MAX_LIVE_ORDER_USD"

// DefaultMaxLiveOrderUSD is the cap applied when MaxLiveOrderEnvVar is unset.
const DefaultMaxLiveOrderUSD = "1"

// EnforceNotionalCap rejects a live order whose notional exceeds the configured
// cap. It reads MaxLiveOrderEnvVar (default DefaultMaxLiveOrderUSD) and must be
// called before the private key is loaded.
func EnforceNotionalCap(notional *big.Rat) error {
	capValue := strings.TrimSpace(os.Getenv(MaxLiveOrderEnvVar))
	if capValue == "" {
		capValue = DefaultMaxLiveOrderUSD
	}
	capRat, err := positiveDecimal(MaxLiveOrderEnvVar, capValue)
	if err != nil {
		return err
	}
	if notional.Cmp(capRat) > 0 {
		return fmt.Errorf("live order notional %s exceeds %s=%s", notional.FloatString(6), MaxLiveOrderEnvVar, capRat.FloatString(6))
	}
	return nil
}

// RequireConfirm enforces a typed live-money confirmation token so a command
// that signs and submits real transactions cannot fire from a single mistyped
// flag. The token must match exactly.
func RequireConfirm(confirm, token string) error {
	if confirm != token {
		return fmt.Errorf("this live-money command requires --confirm %s (got %q)", token, confirm)
	}
	return nil
}

// positiveDecimal parses a positive decimal amount, rejecting fractions and
// non-positive values. It mirrors the CLI's decimalRat so the cap env var is
// validated identically wherever it is read.
func positiveDecimal(name, value string) (*big.Rat, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, fmt.Errorf("%s is required", name)
	}
	if strings.Contains(value, "/") {
		return nil, fmt.Errorf("%s must be a decimal", name)
	}
	r, ok := new(big.Rat).SetString(value)
	if !ok || r.Sign() <= 0 {
		return nil, fmt.Errorf("%s must be a positive decimal", name)
	}
	return r, nil
}
