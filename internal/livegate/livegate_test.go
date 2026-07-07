package livegate

import (
	"math/big"
	"strings"
	"testing"
)

func TestEnforceNotionalCapDefault(t *testing.T) {
	t.Setenv(MaxLiveOrderEnvVar, "")

	if err := EnforceNotionalCap(big.NewRat(1, 1)); err != nil {
		t.Fatalf("notional at default cap should pass: %v", err)
	}
	err := EnforceNotionalCap(big.NewRat(101, 100)) // 1.01 > default 1
	if err == nil {
		t.Fatal("notional above default cap must be rejected")
	}
	if !strings.Contains(err.Error(), MaxLiveOrderEnvVar) {
		t.Fatalf("error must name the cap env var: %v", err)
	}
}

func TestEnforceNotionalCapRespectsEnv(t *testing.T) {
	t.Setenv(MaxLiveOrderEnvVar, "0.50")

	if err := EnforceNotionalCap(big.NewRat(1, 2)); err != nil {
		t.Fatalf("0.50 at the 0.50 cap should pass: %v", err)
	}
	if err := EnforceNotionalCap(big.NewRat(51, 100)); err == nil {
		t.Fatal("0.51 above the 0.50 cap must be rejected")
	}
}

func TestEnforceNotionalCapRejectsBadEnv(t *testing.T) {
	t.Setenv(MaxLiveOrderEnvVar, "1/3")
	if err := EnforceNotionalCap(big.NewRat(1, 100)); err == nil {
		t.Fatal("a fraction cap value must be rejected")
	}
	t.Setenv(MaxLiveOrderEnvVar, "-1")
	if err := EnforceNotionalCap(big.NewRat(1, 100)); err == nil {
		t.Fatal("a non-positive cap value must be rejected")
	}
}

func TestRequireConfirm(t *testing.T) {
	if err := RequireConfirm("SUBMIT_BATCH", "SUBMIT_BATCH"); err != nil {
		t.Fatalf("exact token must pass: %v", err)
	}
	err := RequireConfirm("", "SUBMIT_BATCH")
	if err == nil {
		t.Fatal("empty confirm must be rejected")
	}
	if !strings.Contains(err.Error(), "SUBMIT_BATCH") {
		t.Fatalf("error must name the expected token: %v", err)
	}
	if RequireConfirm("submit_batch", "SUBMIT_BATCH") == nil {
		t.Fatal("token match must be exact (case-sensitive)")
	}
}
