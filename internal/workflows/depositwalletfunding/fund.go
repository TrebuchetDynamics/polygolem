package depositwalletfunding

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/TrebuchetDynamics/polygolem/internal/auth"
	"github.com/TrebuchetDynamics/polygolem/internal/rpc"
)

type FundRequest struct {
	AmountPUSD string
	RPCURL     string
}

type FundResult struct {
	TxHash string `json:"txHash"`
	From   string `json:"from"`
	To     string `json:"to"`
	Amount string `json:"amount"`
}

type FundConfig struct {
	PrivateKey   func() (string, error)
	TransferPUSD func(ctx context.Context, privateKeyHex, toAddress string, amount *big.Int, rpcURL string) (string, error)
}

func (c FundConfig) withDefaults() FundConfig {
	if c.TransferPUSD == nil {
		c.TransferPUSD = rpc.TransferPUSD
	}
	return c
}

func Fund(ctx context.Context, cfg FundConfig, req FundRequest) (FundResult, error) {
	cfg = cfg.withDefaults()
	if cfg.PrivateKey == nil {
		return FundResult{}, fmt.Errorf("private key loader is required")
	}
	key, err := cfg.PrivateKey()
	if err != nil {
		return FundResult{}, err
	}
	signer, err := auth.NewPrivateKeySigner(key, 137)
	if err != nil {
		return FundResult{}, fmt.Errorf("init signer: %w", err)
	}
	owner := signer.Address()
	wallet, err := auth.MakerAddressForSignatureType(owner, 137, 3)
	if err != nil {
		return FundResult{}, fmt.Errorf("derive deposit wallet: %w", err)
	}
	if strings.TrimSpace(req.AmountPUSD) == "" {
		return FundResult{}, fmt.Errorf("--amount is required (pUSD to transfer, e.g. 0.71)")
	}
	amountFloat, err := ParsePUSDAmount(req.AmountPUSD)
	if err != nil {
		return FundResult{}, fmt.Errorf("invalid amount: %w", err)
	}
	if amountFloat.Sign() <= 0 {
		return FundResult{}, fmt.Errorf("amount must be positive")
	}
	txHash, err := cfg.TransferPUSD(ctx, key, wallet, amountFloat, req.RPCURL)
	if err != nil {
		return FundResult{}, fmt.Errorf("transfer pUSD: %w", err)
	}
	return FundResult{
		TxHash: txHash,
		From:   owner,
		To:     wallet,
		Amount: req.AmountPUSD,
	}, nil
}
