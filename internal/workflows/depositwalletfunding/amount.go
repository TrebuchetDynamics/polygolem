package depositwalletfunding

import (
	"fmt"
	"math/big"
	"strings"
)

// ParsePUSDAmount parses a human pUSD decimal amount into 6-decimal base units.
func ParsePUSDAmount(s string) (*big.Int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("empty amount")
	}
	if strings.HasPrefix(s, "-") || strings.HasPrefix(s, "+") {
		return nil, fmt.Errorf("amount must be unsigned decimal")
	}
	parts := strings.Split(s, ".")
	if len(parts) > 2 {
		return nil, fmt.Errorf("invalid amount %q", s)
	}
	wholePart := parts[0]
	if wholePart == "" {
		wholePart = "0"
	}
	if !decimalDigitsOnly(wholePart) {
		return nil, fmt.Errorf("invalid integer part: %s", parts[0])
	}
	fracPart := ""
	if len(parts) == 2 {
		fracPart = parts[1]
		if !decimalDigitsOnly(fracPart) {
			return nil, fmt.Errorf("invalid fractional part: %s", fracPart)
		}
		for len(fracPart) > 6 && strings.HasSuffix(fracPart, "0") {
			fracPart = strings.TrimSuffix(fracPart, "0")
		}
		if len(fracPart) > 6 {
			return nil, fmt.Errorf("pUSD supports at most 6 decimals")
		}
	}
	for len(fracPart) < 6 {
		fracPart += "0"
	}
	whole, ok := new(big.Int).SetString(wholePart, 10)
	if !ok {
		return nil, fmt.Errorf("invalid integer part: %s", wholePart)
	}
	result := new(big.Int).Mul(whole, big.NewInt(1000000))
	if fracPart != "" {
		frac, ok := new(big.Int).SetString(fracPart, 10)
		if !ok {
			return nil, fmt.Errorf("invalid fractional part: %s", fracPart)
		}
		result.Add(result, frac)
	}
	return result, nil
}

func decimalDigitsOnly(s string) bool {
	if s == "" {
		return true
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
