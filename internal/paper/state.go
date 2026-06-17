package paper

import "fmt"

type Order struct {
	MarketID string  `json:"market_id"`
	TokenID  string  `json:"token_id"`
	Price    float64 `json:"price"`
	Size     float64 `json:"size"`
}

type Fill struct {
	MarketID    string  `json:"market_id"`
	TokenID     string  `json:"token_id"`
	Price       float64 `json:"price"`
	Size        float64 `json:"size"`
	Live        bool    `json:"live"`
	RealizedPnL float64 `json:"realized_pnl,omitempty"`
}

type Position struct {
	TokenID string  `json:"token_id"`
	Size    float64 `json:"size"`
	Cost    float64 `json:"cost"`
}

type State struct {
	Currency  string              `json:"currency"`
	Cash      float64             `json:"cash"`
	Positions map[string]Position `json:"positions"`
	Fills     []Fill              `json:"fills"`
}

func NewState(currency string, cash float64) *State {
	return &State{Currency: currency, Cash: cash, Positions: map[string]Position{}}
}

func (s *State) Buy(order Order) (Fill, error) {
	cost := order.Price * order.Size
	if cost > s.Cash {
		return Fill{}, fmt.Errorf("insufficient paper cash")
	}
	s.Cash -= cost
	pos := s.Positions[order.TokenID]
	pos.TokenID = order.TokenID
	pos.Size += order.Size
	pos.Cost += cost
	s.Positions[order.TokenID] = pos
	fill := Fill{MarketID: order.MarketID, TokenID: order.TokenID, Price: order.Price, Size: order.Size, Live: false}
	s.Fills = append(s.Fills, fill)
	return fill, nil
}

const sizeEpsilon = 1e-9

// Sell reduces a held position using average-cost accounting, credits cash with
// the proceeds, and reports realized PnL on the returned fill. Selling more than
// is held, selling a token with no position, or a non-positive size is rejected
// (no shorting). State is not mutated on any error path.
func (s *State) Sell(order Order) (Fill, error) {
	if order.Size <= 0 {
		return Fill{}, fmt.Errorf("sell size must be positive")
	}
	pos := s.Positions[order.TokenID]
	if pos.Size <= 0 || order.Size > pos.Size+sizeEpsilon {
		return Fill{}, fmt.Errorf("insufficient paper position")
	}
	avgCost := pos.Cost / pos.Size
	proceeds := order.Price * order.Size
	realized := proceeds - avgCost*order.Size

	s.Cash += proceeds
	pos.Size -= order.Size
	// Re-derive the remaining cost from the unchanged average cost so float
	// rounding can't accumulate dust across repeated partial sells.
	pos.Cost = avgCost * pos.Size
	if pos.Size <= sizeEpsilon {
		delete(s.Positions, order.TokenID)
	} else {
		s.Positions[order.TokenID] = pos
	}

	fill := Fill{MarketID: order.MarketID, TokenID: order.TokenID, Price: order.Price, Size: order.Size, Live: false, RealizedPnL: realized}
	s.Fills = append(s.Fills, fill)
	return fill, nil
}
