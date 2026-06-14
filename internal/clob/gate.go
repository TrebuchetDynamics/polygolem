package clob

import "errors"

// TradeGate decides whether new orders may be submitted. A nil gate means
// "always allowed". *risk.Breaker satisfies this interface via its CanProceed
// method, but any type with the same method works.
type TradeGate interface {
	CanProceed() bool
}

// ErrTradingHalted is returned by the order-creation methods (CreateLimitOrder,
// CreateMarketOrder, CreateBatchOrders) when an attached TradeGate reports that
// trading is halted. Detect it with errors.Is. Cancellation methods are never
// gated and never return this error.
var ErrTradingHalted = errors.New("clob: trading halted by risk gate")

// Option configures a Client at construction time.
type Option func(*Client)

// WithTradeGate attaches a TradeGate that is consulted before each order
// submission. When the gate reports CanProceed()==false, order-creation methods
// return ErrTradingHalted without signing or sending anything. Passing a nil
// gate is a no-op (equivalent to attaching no gate: trading is always allowed).
func WithTradeGate(g TradeGate) Option {
	return func(c *Client) { c.gate = g }
}

// ensureCanTrade returns ErrTradingHalted when an attached gate reports halted.
// A nil gate always permits trading.
func (c *Client) ensureCanTrade() error {
	if c.gate != nil && !c.gate.CanProceed() {
		return ErrTradingHalted
	}
	return nil
}
