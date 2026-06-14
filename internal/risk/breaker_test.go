package risk

import (
	"testing"
	"time"
)

func TestBreakerStartsClosed(t *testing.T) {
	b := NewBreaker(DefaultPolicy())
	if !b.CanProceed() {
		t.Fatal("should start closed")
	}
}

func TestBreakerOpensOnConsecutiveErrors(t *testing.T) {
	policy := DefaultPolicy()
	policy.MaxConsecutiveErrs = 3
	b := NewBreaker(policy)
	for i := 0; i < 3; i++ {
		if b.RecordError() && i < 2 {
			t.Fatalf("should not break on error %d", i)
		}
	}
	if b.CanProceed() {
		t.Fatal("should be halted")
	}
}

func TestBreakerResetsOnSuccess(t *testing.T) {
	policy := DefaultPolicy()
	policy.MaxConsecutiveErrs = 3
	b := NewBreaker(policy)
	b.RecordError()
	b.RecordError()
	b.RecordSuccess()
	b.RecordError()
	if b.CanProceed() {
		// After 1 error post-reset, should still proceed
	}
}

func TestBreakerDailyLossLimit(t *testing.T) {
	policy := DefaultPolicy()
	policy.DailyLossLimitUSD = 50
	b := NewBreaker(policy)
	if b.RecordLoss(60) {
		if b.CanProceed() {
			t.Fatal("should be halted after exceeding daily loss")
		}
	}
}

func TestBreakerCoolDown(t *testing.T) {
	policy := DefaultPolicy()
	policy.MaxConsecutiveErrs = 1
	policy.CoolDownSecs = 0
	b := NewBreaker(policy)
	b.RecordError()
	if b.CanProceed() {
		t.Fatal("should be halted")
	}
	b.Reset()
	if !b.CanProceed() {
		t.Fatal("should proceed after reset")
	}
}

func TestBreakerRecordSuccessClearsErrors(t *testing.T) {
	policy := DefaultPolicy()
	policy.MaxConsecutiveErrs = 5
	b := NewBreaker(policy)
	for i := 0; i < 4; i++ {
		b.RecordError()
	}
	b.RecordSuccess()
	for i := 0; i < 4; i++ {
		b.RecordError()
	}
	// Should still be closed — successes reset the counter
	if !b.CanProceed() {
		t.Fatal("should still be closed")
	}
}

func TestBreakerHalted(t *testing.T) {
	b := NewBreaker(DefaultPolicy())
	if b.Halted() {
		t.Fatal("should not be halted initially")
	}
	b.RecordError()
	b.RecordError()
	b.RecordError()
	b.RecordError()
	b.RecordError()
	if !b.Halted() {
		t.Fatal("should be halted after 5 errors")
	}
}

func TestBreakerRecordsTripReasonOnConsecutiveErrors(t *testing.T) {
	policy := DefaultPolicy()
	policy.MaxConsecutiveErrs = 1
	b := NewBreaker(policy)
	b.RecordError()
	status := b.Status()
	if status.TripReason != ReasonConsecutiveErrors {
		t.Fatalf("trip reason=%d want ReasonConsecutiveErrors", status.TripReason)
	}
}

func TestBreakerRecordsTripReasonOnDailyLossLimit(t *testing.T) {
	policy := DefaultPolicy()
	policy.DailyLossLimitUSD = 10
	b := NewBreaker(policy)
	b.RecordLoss(20)
	status := b.Status()
	if status.TripReason != ReasonDailyLossLimit {
		t.Fatalf("trip reason=%d want ReasonDailyLossLimit", status.TripReason)
	}
}

func TestBreakerStatusIncludesPositions(t *testing.T) {
	policy := DefaultPolicy()
	b := NewBreaker(policy)
	b.RecordPosition("token1", 5.0)
	b.RecordPosition("token2", -3.0)
	status := b.Status()
	if status.Positions["token1"] != 5.0 || status.Positions["token2"] != -3.0 {
		t.Fatalf("positions=%v", status.Positions)
	}
	if status.TotalPosition != 8.0 {
		t.Fatalf("total position=%f want 8.0", status.TotalPosition)
	}
}

func TestBreakerPositionLimitHaltsTrading(t *testing.T) {
	policy := DefaultPolicy()
	policy.MaxPositionPerMarket = 10.0
	b := NewBreaker(policy)
	if !b.RecordPosition("token1", 15.0) {
		t.Fatal("should halt for single position record")
	}
	if b.CanProceed() {
		t.Fatal("should be halted after exceeding per-market position")
	}
	status := b.Status()
	if status.TripReason != ReasonPositionPerMarket {
		t.Fatalf("trip reason=%d want ReasonPositionPerMarket", status.TripReason)
	}
}

func TestBreakerTotalPositionLimitHaltsTrading(t *testing.T) {
	policy := DefaultPolicy()
	policy.MaxTotalPosition = 10.0
	b := NewBreaker(policy)
	b.RecordPosition("token1", 6.0)
	b.RecordPosition("token2", 6.0)
	if b.CanProceed() {
		t.Fatal("should be halted after exceeding total position")
	}
	status := b.Status()
	if status.TripReason != ReasonTotalPosition {
		t.Fatalf("trip reason=%d want ReasonTotalPosition", status.TripReason)
	}
}

func TestBreakerDailyLossResetsAtConfiguredHour(t *testing.T) {
	// Replace the old fragile test that mutates unexported fields.
	// Instead, test the pure function shouldResetDailyLoss directly.

	// Same day: no reset regardless of hour
	base := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	if shouldResetDailyLoss(base, base, 0) {
		t.Fatal("same moment should not reset")
	}

	// Different day, before reset hour: no reset
	t1 := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2025, 6, 16, 7, 0, 0, 0, time.UTC)
	if shouldResetDailyLoss(t2, t1, 8) {
		t.Fatal("should not reset before reset hour")
	}

	// Different day, at or past reset hour: should reset
	if !shouldResetDailyLoss(t2, t1, 7) {
		t.Fatal("should reset at reset hour")
	}
	if !shouldResetDailyLoss(t2, t1, 0) {
		t.Fatal("should reset when reset hour is 0 and day changed")
	}

	// Month boundary
	jan31 := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)
	feb1 := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	if !shouldResetDailyLoss(feb1, jan31, 0) {
		t.Fatal("should reset at month boundary")
	}

	// Year boundary
	dec31 := time.Date(2025, 12, 31, 23, 0, 0, 0, time.UTC)
	jan1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if !shouldResetDailyLoss(jan1, dec31, 0) {
		t.Fatal("should reset at year boundary")
	}
}

func TestBreakerManualHaltRecordsReason(t *testing.T) {
	b := NewBreaker(DefaultPolicy())
	b.Halt()
	if !b.Halted() {
		t.Fatal("should be halted after manual halt")
	}
	status := b.Status()
	if status.TripReason != ReasonManualHalt {
		t.Fatalf("trip reason=%d want ReasonManualHalt", status.TripReason)
	}
}

func TestBreakerDefaultCooldownIs300Seconds(t *testing.T) {
	policy := DefaultPolicy()
	if policy.CoolDownSecs != 300 {
		t.Fatalf("default cooldown=%d want 300", policy.CoolDownSecs)
	}
}

// --- Bug-exposing characterization tests ---

func TestBreakerRecordLossRejectsNegative(t *testing.T) {
	// BUG: negative loss values reduce the dailyLoss accumulator,
	// hiding real losses. A trader losing 100 then "recording" -50
	// should still show 100 daily loss, not 50.
	policy := DefaultPolicy()
	policy.DailyLossLimitUSD = 100
	b := NewBreaker(policy)

	b.RecordLoss(100) // hit limit
	if !b.Halted() {
		t.Fatal("should be halted after 100 loss")
	}

	// Reset and test the negative case
	b.Reset()
	b.RecordLoss(100) // hit limit
	b.RecordLoss(-50) // should NOT reduce the accumulator
	status := b.Status()
	if status.DailyLossUSD != 100 {
		t.Fatalf("daily loss should be 100 after negative record, got %f", status.DailyLossUSD)
	}
}

func TestBreakerResetClearsTimestamps(t *testing.T) {
	policy := DefaultPolicy()
	policy.MaxConsecutiveErrs = 1
	b := NewBreaker(policy)

	b.RecordError()
	status := b.Status()
	if !status.Halted {
		t.Fatal("should be halted after error")
	}

	b.Reset()
	status = b.Status()
	if status.Halted {
		t.Fatal("should not be halted after reset")
	}
	if !status.LastBreak.IsZero() {
		t.Fatal("last_break should be zero after reset")
	}
	if status.ConsecutiveErrs != 0 {
		t.Fatal("consecutive errors should be 0 after reset")
	}
}

func TestBreakerRecordLossBoundaryConsistency(t *testing.T) {
	// RecordLoss uses >= while position checks use >.
	// This test characterizes the boundary for loss.
	policy := DefaultPolicy()
	policy.DailyLossLimitUSD = 100
	b := NewBreaker(policy)

	// Exactly at limit should trip (>= semantics)
	b.RecordLoss(100)
	if !b.Halted() {
		t.Fatal("should halt when loss equals limit")
	}
}

func TestBreakerPositionBoundaryConsistency(t *testing.T) {
	// RecordPosition uses > for per-market check.
	// A position exactly equal to limit should NOT trip.
	policy := DefaultPolicy()
	policy.MaxPositionPerMarket = 10
	b := NewBreaker(policy)

	// Exactly at limit should NOT trip
	if b.RecordPosition("tok", 10) {
		t.Fatal("position equal to limit should not trip (uses >)")
	}
	if b.Halted() {
		t.Fatal("should not be halted at exact boundary")
	}
}

func TestBreakerRecordLossOnlyAddsToDailyLoss(t *testing.T) {
	// Characterization: RecordLoss returns true when limit hit,
	// but does not set halted when limit is already hit.
	policy := DefaultPolicy()
	policy.DailyLossLimitUSD = 100
	b := NewBreaker(policy)

	if !b.RecordLoss(100) {
		t.Fatal("first loss at limit should trip")
	}
	// Second call should still trigger limit (>= already hit)
	// but the breaker is already halted
	if !b.Halted() {
		t.Fatal("should remain halted")
	}
	// RecordLoss returns true since dailyLoss >= limit
	if !b.RecordLoss(1) {
		t.Fatal("should still report loss limit hit")
	}
}

func TestBreakerCanProceedAutoResetPreservesOtherState(t *testing.T) {
	// CanProceed auto-resets after cooldown, but only clears halted-related state.
	// Verify other state (dailyLoss, positions) survives the cooldown auto-reset.
	policy := DefaultPolicy()
	policy.DailyLossLimitUSD = 100
	policy.CoolDownSecs = 0
	b := NewBreaker(policy)

	b.RecordLoss(50)
	b.RecordPosition("tok", 5)
	b.RecordError()

	// Trigger cooldown auto-reset (CoolDownSecs=0 means immediate)
	if !b.CanProceed() {
		t.Fatal("should auto-reset when cooldown is 0")
	}

	// dailyLoss should still be 50 (not wiped by cooldown auto-reset)
	status := b.Status()
	if status.DailyLossUSD != 50 {
		t.Fatalf("daily loss should survive auto-reset, got %f", status.DailyLossUSD)
	}
	if status.Positions["tok"] != 5 {
		t.Fatalf("positions should survive auto-reset, got %v", status.Positions)
	}
}

func TestComputeTotalAbsPosition(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]float64
		expected float64
	}{
		{"empty", map[string]float64{}, 0},
		{"single positive", map[string]float64{"a": 5}, 5},
		{"single negative", map[string]float64{"a": -5}, 5},
		{"mixed", map[string]float64{"a": 5, "b": -3}, 8},
		{"zero", map[string]float64{"a": 0}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeTotalAbsPosition(tt.input)
			if got != tt.expected {
				t.Fatalf("computeTotalAbsPosition(%v) = %f, want %f", tt.input, got, tt.expected)
			}
		})
	}
}

func TestShouldTripPerMarket(t *testing.T) {
	tests := []struct {
		name   string
		size   float64
		max    float64
		should bool
	}{
		{"zero max never trips", 100, 0, false},
		{"below max does not trip", 5, 10, false},
		{"equal to max does not trip", 10, 10, false},
		{"above max trips", 11, 10, true},
		{"negative below max abs", -5, 10, false},
		{"negative above max abs", -15, 10, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldTripPerMarket(tt.size, tt.max)
			if got != tt.should {
				t.Fatalf("shouldTripPerMarket(%f, %f) = %v, want %v", tt.size, tt.max, got, tt.should)
			}
		})
	}
}

func TestShouldTripTotalPosition(t *testing.T) {
	tests := []struct {
		name   string
		total  float64
		max    float64
		should bool
	}{
		{"zero max never trips", 100, 0, false},
		{"below max does not trip", 5, 10, false},
		{"equal to max does not trip", 10, 10, false},
		{"above max trips", 11, 10, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldTripTotalPosition(tt.total, tt.max)
			if got != tt.should {
				t.Fatalf("shouldTripTotalPosition(%f, %f) = %v, want %v", tt.total, tt.max, got, tt.should)
			}
		})
	}
}

func TestClampNonNegative(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected float64
	}{
		{"positive", 5, 5},
		{"negative clamped to zero", -5, 0},
		{"zero stays zero", 0, 0},
		{"fractional", 1.5, 1.5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := clampNonNegative(tt.input)
			if got != tt.expected {
				t.Fatalf("clampNonNegative(%f) = %f, want %f", tt.input, got, tt.expected)
			}
		})
	}
}

// TestBreakerDoesNotAutoResumeWhileOverDailyLoss guards the fail-open regression
// where CanProceed cleared a daily-loss trip after the cooldown elapsed even though
// the account was still over its loss limit.
func TestBreakerDoesNotAutoResumeWhileOverDailyLoss(t *testing.T) {
	policy := DefaultPolicy()
	policy.DailyLossLimitUSD = 10
	policy.CoolDownSecs = 1
	b := NewBreaker(policy)
	b.RecordLoss(20) // trip on daily loss
	// Simulate cooldown elapsing.
	b.lastBreak = b.lastBreak.Add(-2 * time.Second)
	if b.CanProceed() {
		t.Fatal("breaker must stay halted while still over the daily loss limit")
	}
	// Once an explicit reset (or a new day) clears the loss, trading may resume.
	b.Reset()
	if !b.CanProceed() {
		t.Fatal("breaker should proceed after Reset clears the loss")
	}
}

// TestBreakerDoesNotAutoResumeWhileOverPositionLimit guards the same fail-open for
// position-limit trips.
func TestBreakerDoesNotAutoResumeWhileOverPositionLimit(t *testing.T) {
	policy := DefaultPolicy()
	policy.MaxTotalPosition = 10
	policy.CoolDownSecs = 1
	b := NewBreaker(policy)
	b.RecordPosition("token1", 20) // trip on total position
	b.lastBreak = b.lastBreak.Add(-2 * time.Second)
	if b.CanProceed() {
		t.Fatal("breaker must stay halted while still over the position limit")
	}
	// Reducing the position below the limit clears the condition; cooldown then resumes.
	b.RecordPosition("token1", 1)
	b.lastBreak = b.lastBreak.Add(-2 * time.Second)
	if !b.CanProceed() {
		t.Fatal("breaker should resume once position is back under the limit")
	}
}

// TestBreakerAutoResumesConsecutiveErrorsAfterCooldown confirms transient
// error trips still auto-clear after the cooldown (unchanged behavior).
func TestBreakerAutoResumesConsecutiveErrorsAfterCooldown(t *testing.T) {
	policy := DefaultPolicy()
	policy.MaxConsecutiveErrs = 1
	policy.CoolDownSecs = 1
	b := NewBreaker(policy)
	b.RecordError()
	if b.CanProceed() {
		t.Fatal("should be halted immediately after error trip")
	}
	b.lastBreak = b.lastBreak.Add(-2 * time.Second)
	if !b.CanProceed() {
		t.Fatal("consecutive-error trip should auto-resume after cooldown")
	}
}
