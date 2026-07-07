// Package clobsimulation estimates taker fills from read-only CLOB books.
package clobsimulation

import (
	"context"
	"fmt"
	"math/big"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/polygolem/internal/polytypes"
)

// Reader is the read-only CLOB interface used by the simulation workflow.
type Reader interface {
	OrderBook(context.Context, string) (*polytypes.OrderBook, error)
}

// Request describes a proposed order to simulate against the current book.
type Request struct {
	TokenID    string
	Side       string
	Amount     string
	LimitPrice string
	Output     string
}

// FillLevel is one book level consumed by the simulated order.
type FillLevel struct {
	Price         string `json:"price"`
	AvailableSize string `json:"available_size"`
	FilledSize    string `json:"filled_size"`
	Notional      string `json:"notional"`
}

// Result reports the expected read-only fill for a proposed order.
type Result struct {
	TokenID           string      `json:"token_id"`
	Market            string      `json:"market,omitempty"`
	Side              string      `json:"side"`
	InputAmount       string      `json:"input_amount"`
	InputAmountType   string      `json:"input_amount_type"`
	LimitPrice        string      `json:"limit_price,omitempty"`
	Complete          bool        `json:"complete"`
	FilledSize        string      `json:"filled_size"`
	Notional          string      `json:"notional"`
	AveragePrice      string      `json:"average_price,omitempty"`
	ExpectedFillPrice string      `json:"expected_fill_price,omitempty"`
	BestPrice         string      `json:"best_price,omitempty"`
	WorstPrice        string      `json:"worst_price,omitempty"`
	Slippage          string      `json:"slippage,omitempty"`
	SlippageBps       string      `json:"slippage_bps,omitempty"`
	UnfilledAmount    string      `json:"unfilled_amount"`
	BookHash          string      `json:"book_hash,omitempty"`
	BookTimestamp     string      `json:"book_timestamp,omitempty"`
	Levels            []FillLevel `json:"levels"`
}

// Runner owns read-only CLOB order simulation.
type Runner struct {
	reader Reader
}

// New creates a read-only CLOB simulation runner.
func New(reader Reader) *Runner {
	return &Runner{reader: reader}
}

// SimulateOrder walks the opposing side of the book without signing or submitting anything.
func (r *Runner) SimulateOrder(ctx context.Context, req Request) (*Result, error) {
	if err := checkOutput(req.Output); err != nil {
		return nil, err
	}
	tokenID := strings.TrimSpace(req.TokenID)
	if tokenID == "" {
		return nil, fmt.Errorf("--token is required")
	}
	side, err := normalizeSide(req.Side)
	if err != nil {
		return nil, err
	}
	amount, err := decimalRat("--amount", req.Amount)
	if err != nil {
		return nil, err
	}
	var limit *big.Rat
	if strings.TrimSpace(req.LimitPrice) != "" {
		limit, err = decimalRat("--limit-price", req.LimitPrice)
		if err != nil {
			return nil, err
		}
	}
	book, err := r.reader.OrderBook(ctx, tokenID)
	if err != nil {
		return nil, err
	}
	return simulateBook(book, tokenID, side, amount, limit), nil
}

type bookLevel struct {
	price *big.Rat
	size  *big.Rat
}

func simulateBook(book *polytypes.OrderBook, tokenID, side string, amount, limit *big.Rat) *Result {
	if book == nil {
		book = &polytypes.OrderBook{}
	}
	levels := opposingLevels(book, side)
	bestPrice := ""
	if len(levels) > 0 {
		bestPrice = formatRat(levels[0].price, 6)
	}
	remaining := new(big.Rat).Set(amount)
	filledSize := new(big.Rat)
	notional := new(big.Rat)
	var fills []FillLevel
	worstPrice := ""
	for _, level := range levels {
		if limit != nil && ((side == "buy" && level.price.Cmp(limit) > 0) || (side == "sell" && level.price.Cmp(limit) < 0)) {
			break
		}
		fillSize, fillNotional := fillAtLevel(side, remaining, level)
		if fillSize.Sign() == 0 {
			continue
		}
		filledSize.Add(filledSize, fillSize)
		notional.Add(notional, fillNotional)
		fills = append(fills, FillLevel{
			Price:         formatRat(level.price, 6),
			AvailableSize: formatRat(level.size, 6),
			FilledSize:    formatRat(fillSize, 6),
			Notional:      formatRat(fillNotional, 6),
		})
		worstPrice = formatRat(level.price, 6)
		if side == "buy" {
			remaining.Sub(remaining, fillNotional)
		} else {
			remaining.Sub(remaining, fillSize)
		}
		if remaining.Sign() <= 0 {
			remaining.SetInt64(0)
			break
		}
	}
	result := &Result{
		TokenID:         firstNonEmpty(book.AssetID, tokenID),
		Market:          book.Market,
		Side:            side,
		InputAmount:     formatRat(amount, 6),
		InputAmountType: inputAmountType(side),
		Complete:        remaining.Sign() == 0,
		FilledSize:      formatRat(filledSize, 6),
		Notional:        formatRat(notional, 6),
		BestPrice:       bestPrice,
		WorstPrice:      worstPrice,
		UnfilledAmount:  formatRat(remaining, 6),
		BookHash:        book.Hash,
		BookTimestamp:   book.Timestamp,
		Levels:          fills,
	}
	if limit != nil {
		result.LimitPrice = formatRat(limit, 6)
	}
	if filledSize.Sign() > 0 {
		avg := new(big.Rat).Quo(notional, filledSize)
		result.AveragePrice = formatRat(avg, 6)
		result.ExpectedFillPrice = result.AveragePrice
		if len(levels) > 0 && levels[0].price.Sign() > 0 {
			slip := slippage(side, levels[0].price, avg)
			result.Slippage = formatRat(slip, 6)
			result.SlippageBps = formatRat(new(big.Rat).Mul(new(big.Rat).Quo(slip, levels[0].price), big.NewRat(10000, 1)), 4)
		}
	}
	return result
}

func opposingLevels(book *polytypes.OrderBook, side string) []bookLevel {
	raw := book.Asks
	if side == "sell" {
		raw = book.Bids
	}
	levels := make([]bookLevel, 0, len(raw))
	for _, level := range raw {
		price, err := parseRat(level.Price)
		if err != nil || price.Sign() <= 0 {
			continue
		}
		size, err := parseRat(level.Size)
		if err != nil || size.Sign() <= 0 {
			continue
		}
		levels = append(levels, bookLevel{price: price, size: size})
	}
	sort.SliceStable(levels, func(i, j int) bool {
		if side == "buy" {
			return levels[i].price.Cmp(levels[j].price) < 0
		}
		return levels[i].price.Cmp(levels[j].price) > 0
	})
	return levels
}

func fillAtLevel(side string, remaining *big.Rat, level bookLevel) (*big.Rat, *big.Rat) {
	if side == "buy" {
		levelNotional := new(big.Rat).Mul(level.size, level.price)
		if remaining.Cmp(levelNotional) >= 0 {
			return new(big.Rat).Set(level.size), levelNotional
		}
		return new(big.Rat).Quo(remaining, level.price), new(big.Rat).Set(remaining)
	}
	fillSize := new(big.Rat).Set(level.size)
	if remaining.Cmp(level.size) < 0 {
		fillSize.Set(remaining)
	}
	return fillSize, new(big.Rat).Mul(fillSize, level.price)
}

func slippage(side string, best, average *big.Rat) *big.Rat {
	if side == "buy" {
		return new(big.Rat).Sub(average, best)
	}
	return new(big.Rat).Sub(best, average)
}

func normalizeSide(side string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(side)) {
	case "", "buy":
		return "buy", nil
	case "sell":
		return "sell", nil
	default:
		return "", fmt.Errorf("--side must be buy or sell")
	}
}

func inputAmountType(side string) string {
	if side == "buy" {
		return "usdc"
	}
	return "shares"
}

func decimalRat(name, value string) (*big.Rat, error) {
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

func parseRat(value string) (*big.Rat, error) {
	r, ok := new(big.Rat).SetString(strings.TrimSpace(value))
	if !ok {
		return nil, fmt.Errorf("invalid decimal %q", value)
	}
	return r, nil
}

func formatRat(value *big.Rat, places int) string {
	if value == nil {
		return "0"
	}
	s := value.FloatString(places)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	}
	if s == "" || s == "-0" {
		return "0"
	}
	return s
}

func checkOutput(output string) error {
	if output != "" && output != "json" {
		return fmt.Errorf("only --output json is supported")
	}
	return nil
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}
