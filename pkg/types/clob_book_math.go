package types

import (
	"errors"
	"math"
	"strconv"
	"strings"
)

// BestBid returns the highest executable bid price in the book. The second
// return value is false when the book has no executable bid levels.
func (o CLOBOrderBook) BestBid() (float64, bool) {
	best := 0.0
	found := false
	for _, lvl := range o.Bids {
		price, _, ok := parseBookLevel(lvl)
		if !ok {
			continue
		}
		if !found || price > best {
			best = price
			found = true
		}
	}
	return best, found
}

// BestAsk returns the lowest executable ask price in the book. The second
// return value is false when the book has no executable ask levels.
func (o CLOBOrderBook) BestAsk() (float64, bool) {
	best := 0.0
	found := false
	for _, lvl := range o.Asks {
		price, _, ok := parseBookLevel(lvl)
		if !ok {
			continue
		}
		if !found || price < best {
			best = price
			found = true
		}
	}
	return best, found
}

// AvailableAskSize returns the total executable ask size at or below maxPrice.
// Callers use this to detect thin books before submitting taker orders. A
// non-positive maxPrice returns 0.
func (o CLOBOrderBook) AvailableAskSize(maxPrice float64) float64 {
	if !positiveFiniteBookValue(maxPrice) {
		return 0
	}
	total := 0.0
	for _, lvl := range o.Asks {
		price, size, ok := parseBookLevel(lvl)
		if !ok || price > maxPrice {
			continue
		}
		total += size
	}
	return total
}

// ErrTickSizeUnavailable is returned by CLOBTickSize.Value when the payload
// carries no parseable positive tick size.
var ErrTickSizeUnavailable = errors.New("clob tick size unavailable")

// Value returns the tick size as a float, preferring TickSize and falling
// back to MinimumTickSize. It returns ErrTickSizeUnavailable when neither
// field parses to a positive finite number.
func (t CLOBTickSize) Value() (float64, error) {
	for _, raw := range []string{t.TickSize, t.MinimumTickSize} {
		v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
		if err == nil && positiveFiniteBookValue(v) {
			return v, nil
		}
	}
	return 0, ErrTickSizeUnavailable
}

// parseBookLevel parses one string-typed book level. A level is executable
// when its price is a positive finite share price (0, 1] and its size is
// positive finite.
func parseBookLevel(lvl CLOBOrderBookLevel) (price, size float64, ok bool) {
	price, err := strconv.ParseFloat(strings.TrimSpace(lvl.Price), 64)
	if err != nil {
		return 0, 0, false
	}
	size, err = strconv.ParseFloat(strings.TrimSpace(lvl.Size), 64)
	if err != nil {
		return 0, 0, false
	}
	if !positiveFiniteBookValue(price) || price > 1 || !positiveFiniteBookValue(size) {
		return 0, 0, false
	}
	return price, size, true
}

func positiveFiniteBookValue(v float64) bool {
	return v > 0 && !math.IsNaN(v) && !math.IsInf(v, 0)
}
