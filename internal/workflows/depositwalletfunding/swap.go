package depositwalletfunding

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/TrebuchetDynamics/polygolem/internal/auth"
	"github.com/TrebuchetDynamics/polygolem/internal/rpc"
)

type SwapRequest struct {
	AmountPUSDOut string
	MaxPOLIn      string
	RPCURL        string
}

type SwapResult struct {
	TxHash        string `json:"txHash"`
	Recipient     string `json:"recipient"`
	AmountPUSDOut string `json:"amountPUSDOut"`
	MaxPOLIn      string `json:"maxPOLIn"`
}

type SwapConfig struct {
	PrivateKey          func() (string, error)
	SwapPOLForExactPUSD func(ctx context.Context, privateKeyHex string, amountOutPUSD, maxPOLInWei *big.Int, rpcURL string) (string, error)
}

func (c SwapConfig) withDefaults() SwapConfig {
	if c.SwapPOLForExactPUSD == nil {
		c.SwapPOLForExactPUSD = rpc.SwapPOLForExactPUSD
	}
	return c
}

func Swap(ctx context.Context, cfg SwapConfig, req SwapRequest) (SwapResult, error) {
	cfg = cfg.withDefaults()
	if cfg.PrivateKey == nil {
		return SwapResult{}, fmt.Errorf("private key loader is required")
	}
	key, err := cfg.PrivateKey()
	if err != nil {
		return SwapResult{}, err
	}
	signer, err := auth.NewPrivateKeySigner(key, 137)
	if err != nil {
		return SwapResult{}, fmt.Errorf("init signer: %w", err)
	}
	if strings.TrimSpace(req.AmountPUSDOut) == "" {
		return SwapResult{}, fmt.Errorf("--out-pusd is required (pUSD to receive, e.g. 0.72)")
	}
	if strings.TrimSpace(req.MaxPOLIn) == "" {
		return SwapResult{}, fmt.Errorf("--max-pol-in is required (max POL to spend, e.g. 10)")
	}
	outPUSD, err := ParsePUSDAmount(req.AmountPUSDOut)
	if err != nil {
		return SwapResult{}, fmt.Errorf("invalid --out-pusd: %w", err)
	}
	if outPUSD.Sign() <= 0 {
		return SwapResult{}, fmt.Errorf("--out-pusd must be positive")
	}
	maxPOLWei, err := ParsePOLAmount(req.MaxPOLIn)
	if err != nil {
		return SwapResult{}, fmt.Errorf("invalid --max-pol-in: %w", err)
	}
	if maxPOLWei.Sign() <= 0 {
		return SwapResult{}, fmt.Errorf("--max-pol-in must be positive")
	}
	txHash, err := cfg.SwapPOLForExactPUSD(ctx, key, outPUSD, maxPOLWei, req.RPCURL)
	if err != nil {
		return SwapResult{}, fmt.Errorf("swap POL→pUSD: %w", err)
	}
	return SwapResult{
		TxHash:        txHash,
		Recipient:     signer.Address(),
		AmountPUSDOut: req.AmountPUSDOut,
		MaxPOLIn:      req.MaxPOLIn,
	}, nil
}

// ParsePOLAmount converts a human POL string (e.g. "10", "0.5") to wei
// (18-decimal *big.Int).
func ParsePOLAmount(s string) (*big.Int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("empty amount")
	}
	parts := strings.Split(s, ".")
	if len(parts) > 2 {
		return nil, fmt.Errorf("invalid POL amount %q", s)
	}
	whole, ok := new(big.Int).SetString(parts[0], 10)
	if !ok {
		return nil, fmt.Errorf("invalid integer part: %s", parts[0])
	}
	weiPerPOL := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	result := new(big.Int).Mul(whole, weiPerPOL)
	if len(parts) == 2 {
		frac := parts[1]
		if len(frac) > 18 {
			frac = frac[:18]
		}
		for len(frac) < 18 {
			frac += "0"
		}
		fracInt, ok := new(big.Int).SetString(frac, 10)
		if !ok {
			return nil, fmt.Errorf("invalid fractional part: %s", parts[1])
		}
		result.Add(result, fracInt)
	}
	return result, nil
}
