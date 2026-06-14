package risk

import (
	"sync"
	"time"
)

// TripReason explains why the breaker tripped.
type TripReason int

const (
	ReasonConsecutiveErrors TripReason = iota
	ReasonDailyLossLimit
	ReasonPositionPerMarket
	ReasonTotalPosition
	ReasonManualHalt
)

func (r TripReason) String() string {
	switch r {
	case ReasonConsecutiveErrors:
		return "consecutive_errors"
	case ReasonDailyLossLimit:
		return "daily_loss_limit"
	case ReasonPositionPerMarket:
		return "position_per_market"
	case ReasonTotalPosition:
		return "total_position"
	case ReasonManualHalt:
		return "manual_halt"
	default:
		return "unknown"
	}
}

// Policy defines risk limits for trading.
type Policy struct {
	DailyLossLimitUSD    float64 `json:"daily_loss_limit_usd"`
	DailyPnLResetHour    int     `json:"daily_pnl_reset_hour"`
	MaxConsecutiveErrs   int     `json:"max_consecutive_errors"`
	CoolDownSecs         int     `json:"cooldown_secs"`
	MaxPositionPerMarket float64 `json:"max_position_per_market"`
	MaxTotalPosition     float64 `json:"max_total_position"`
}

// DefaultPolicy returns conservative defaults.
func DefaultPolicy() Policy {
	return Policy{
		DailyLossLimitUSD:    100.0,
		DailyPnLResetHour:    0,
		MaxConsecutiveErrs:   5,
		CoolDownSecs:         300,
		MaxPositionPerMarket: 50.0,
		MaxTotalPosition:     200.0,
	}
}

// Status is a snapshot of the breaker's current state.
type Status struct {
	Halted          bool               `json:"halted"`
	TripReason      TripReason         `json:"trip_reason"`
	TripReasonMsg   string             `json:"trip_reason_message"`
	LastBreak       time.Time          `json:"last_break"`
	ConsecutiveErrs int                `json:"consecutive_errors"`
	DailyLossUSD    float64            `json:"daily_loss_usd"`
	TotalPosition   float64            `json:"total_position_usd"`
	Positions       map[string]float64 `json:"positions"`
	CoolDownReady   bool               `json:"cooldown_ready"`
}

// Breaker tracks violations and can halt trading.
type Breaker struct {
	policy          Policy
	mu              sync.Mutex
	consecutiveErrs int
	dailyLoss       float64
	dailyLossReset  time.Time
	lastBreak       time.Time
	halted          bool
	tripReason      TripReason
	positions       map[string]float64
}

// NewBreaker creates a risk circuit breaker.
func NewBreaker(policy Policy) *Breaker {
	return &Breaker{
		policy:    policy,
		positions: make(map[string]float64),
	}
}

// RecordError increments the error counter and returns true if we should break.
func (b *Breaker) RecordError() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.checkDailyResetLocked()
	b.consecutiveErrs++
	if b.consecutiveErrs >= b.policy.MaxConsecutiveErrs {
		b.halted = true
		b.tripReason = ReasonConsecutiveErrors
		b.lastBreak = time.Now()
		return true
	}
	return false
}

// RecordSuccess resets the error counter.
func (b *Breaker) RecordSuccess() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.consecutiveErrs = 0
}

// RecordLoss adds to daily PnL. Returns true if daily limit hit.
// Negative amounts are treated as zero (breakers track losses, not net PnL).
func (b *Breaker) RecordLoss(amount float64) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.checkDailyResetLocked()
	b.dailyLoss += clampNonNegative(amount)
	if b.dailyLoss >= b.policy.DailyLossLimitUSD {
		b.halted = true
		b.tripReason = ReasonDailyLossLimit
		b.lastBreak = time.Now()
		return true
	}
	return false
}

// RecordPosition updates the position for a token and checks limits.
// Returns true if a limit was breached and the breaker tripped.
func (b *Breaker) RecordPosition(tokenID string, size float64) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.positions[tokenID] = size
	total := computeTotalAbsPosition(b.positions)
	if shouldTripPerMarket(size, b.policy.MaxPositionPerMarket) {
		b.halted = true
		b.tripReason = ReasonPositionPerMarket
		b.lastBreak = time.Now()
		return true
	}
	if shouldTripTotalPosition(total, b.policy.MaxTotalPosition) {
		b.halted = true
		b.tripReason = ReasonTotalPosition
		b.lastBreak = time.Now()
		return true
	}
	return false
}

// Halt manually trips the breaker.
func (b *Breaker) Halt() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.halted = true
	b.tripReason = ReasonManualHalt
	b.lastBreak = time.Now()
}

// Status returns a snapshot of the current breaker state.
func (b *Breaker) Status() Status {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.checkDailyResetLocked()
	coolDownReady := false
	if b.halted && b.policy.CoolDownSecs > 0 {
		coolDownReady = time.Since(b.lastBreak) > time.Duration(b.policy.CoolDownSecs)*time.Second
	}
	total := computeTotalAbsPosition(b.positions)
	posCopy := make(map[string]float64, len(b.positions))
	for k, v := range b.positions {
		posCopy[k] = v
	}
	return Status{
		Halted:          b.halted,
		TripReason:      b.tripReason,
		TripReasonMsg:   b.tripReason.String(),
		LastBreak:       b.lastBreak,
		ConsecutiveErrs: b.consecutiveErrs,
		DailyLossUSD:    b.dailyLoss,
		TotalPosition:   total,
		Positions:       posCopy,
		CoolDownReady:   coolDownReady,
	}
}

// CanProceed returns true if trading is allowed.
func (b *Breaker) CanProceed() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.checkDailyResetLocked()
	if b.halted {
		if b.policy.CoolDownSecs > 0 {
			if time.Since(b.lastBreak) > time.Duration(b.policy.CoolDownSecs)*time.Second {
				// Cooling down only clears transient trips. A loss- or
				// position-limit trip must not auto-resume while the account
				// is still over the limit, otherwise the guard fails open.
				if b.tripConditionStillActiveLocked() {
					return false
				}
				b.halted = false
				b.consecutiveErrs = 0
				b.tripReason = 0
				return true
			}
		}
		return false
	}
	return true
}

// tripConditionStillActiveLocked reports whether the limit that tripped the
// breaker is still breached. Must be called with b.mu held.
func (b *Breaker) tripConditionStillActiveLocked() bool {
	switch b.tripReason {
	case ReasonDailyLossLimit:
		return b.policy.DailyLossLimitUSD > 0 && b.dailyLoss >= b.policy.DailyLossLimitUSD
	case ReasonPositionPerMarket:
		for _, size := range b.positions {
			if shouldTripPerMarket(size, b.policy.MaxPositionPerMarket) {
				return true
			}
		}
		return false
	case ReasonTotalPosition:
		return shouldTripTotalPosition(computeTotalAbsPosition(b.positions), b.policy.MaxTotalPosition)
	default:
		// Consecutive-error and manual-halt trips are transient and may
		// clear after the cooldown elapses.
		return false
	}
}

// Halted returns whether the breaker is currently tripped.
func (b *Breaker) Halted() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.halted
}

// Reset clears all breaker state including timestamps.
func (b *Breaker) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.halted = false
	b.tripReason = 0
	b.consecutiveErrs = 0
	b.dailyLoss = 0
	b.positions = make(map[string]float64)
	b.dailyLossReset = time.Time{}
	b.lastBreak = time.Time{}
}

// checkDailyResetLocked resets daily loss if we've crossed the configured UTC reset hour.
// Must be called with b.mu held.
func (b *Breaker) checkDailyResetLocked() {
	if b.policy.DailyLossLimitUSD <= 0 {
		return
	}
	now := time.Now().UTC()
	if b.dailyLossReset.IsZero() {
		b.dailyLossReset = now
		return
	}
	if shouldResetDailyLoss(now, b.dailyLossReset, b.policy.DailyPnLResetHour) {
		b.dailyLoss = 0
		b.dailyLossReset = now
	}
}

// clampNonNegative returns 0 if v is negative, otherwise v.
// RecordLoss should not accept negative values since it tracks losses, not net PnL.
func clampNonNegative(v float64) float64 {
	if v < 0 {
		return 0
	}
	return v
}

// shouldResetDailyLoss returns true when enough calendar time has passed since lastReset
// that the daily loss counter should be cleared. Reset triggers when:
//  1. now and lastReset are on different calendar days, AND
//  2. now.Hour() >= resetHour (e.g., reset happens at or after the configured UTC hour)
func shouldResetDailyLoss(now time.Time, lastReset time.Time, resetHour int) bool {
	if now.Year() != lastReset.Year() || now.Month() != lastReset.Month() || now.Day() != lastReset.Day() {
		return now.Hour() >= resetHour
	}
	return false
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

// computeTotalAbsPosition returns sum of abs values of all tracked positions.
func computeTotalAbsPosition(positions map[string]float64) float64 {
	total := 0.0
	for _, v := range positions {
		total += abs(v)
	}
	return total
}

// shouldTripPerMarket returns true if size exceeds maxPerMarket (strictly greater).
func shouldTripPerMarket(size float64, maxPerMarket float64) bool {
	return maxPerMarket > 0 && abs(size) > maxPerMarket
}

// shouldTripTotalPosition returns true if total exceeds maxTotal (strictly greater).
func shouldTripTotalPosition(total float64, maxTotal float64) bool {
	return maxTotal > 0 && total > maxTotal
}
