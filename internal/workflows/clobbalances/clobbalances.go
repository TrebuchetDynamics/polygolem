// Package clobbalances reads authenticated CLOB balances without Cobra coupling.
package clobbalances

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	internalclob "github.com/TrebuchetDynamics/polygolem/internal/clob"
)

// PrivateKeyLoader loads the EOA private key used for authenticated CLOB balance reads.
type PrivateKeyLoader func() (string, error)

// Reader is the authenticated CLOB balance interface used by the workflow.
type Reader interface {
	BalanceAllowance(context.Context, string, internalclob.BalanceAllowanceParams) (*internalclob.BalanceAllowanceResponse, error)
	UpdateBalanceAllowance(context.Context, string, internalclob.BalanceAllowanceParams) (*internalclob.BalanceAllowanceResponse, error)
}

// Config contains adapters used by the CLOB balance workflow.
type Config struct {
	Reader     Reader
	PrivateKey PrivateKeyLoader
}

// Request describes an authenticated CLOB balance read.
type Request struct {
	AssetType string
	TokenID   string
	Output    string
}

// Runner owns authenticated CLOB balance orchestration behind a small interface.
type Runner struct {
	reader     Reader
	privateKey PrivateKeyLoader
}

// New creates a CLOB balance workflow runner.
func New(cfg Config) *Runner {
	return &Runner{reader: cfg.Reader, privateKey: cfg.PrivateKey}
}

// Balance returns current CLOB balance and allowance state.
func (r *Runner) Balance(ctx context.Context, req Request) (map[string]interface{}, error) {
	key, err := r.privateKeyAfterOutput(req.Output)
	if err != nil {
		return nil, err
	}
	res, err := r.reader.BalanceAllowance(ctx, key, params(req))
	if err != nil {
		return nil, err
	}
	return FormatBalanceResponse(res), nil
}

// UpdateBalance refreshes and returns CLOB balance and allowance state.
func (r *Runner) UpdateBalance(ctx context.Context, req Request) (map[string]interface{}, error) {
	key, err := r.privateKeyAfterOutput(req.Output)
	if err != nil {
		return nil, err
	}
	res, err := r.reader.UpdateBalanceAllowance(ctx, key, params(req))
	if err != nil {
		return nil, err
	}
	return FormatBalanceResponse(res), nil
}

// FormatBalanceResponse converts the CLOB response to the CLI JSON shape and scales collateral base units.
func FormatBalanceResponse(res *internalclob.BalanceAllowanceResponse) map[string]interface{} {
	out := map[string]interface{}{}
	if res == nil {
		return out
	}
	out["balance"] = res.Balance
	if len(res.Allowances) > 0 {
		out["allowances"] = res.Allowances
	}
	if strings.TrimSpace(res.Allowance) != "" {
		out["allowance"] = res.Allowance
	}
	if scaled, ok := scaleBaseUnits(res.Balance, 6); ok {
		out["balance"] = scaled
	}
	return out
}

func (r *Runner) privateKeyAfterOutput(output string) (string, error) {
	if err := checkOutput(output); err != nil {
		return "", err
	}
	if r.privateKey == nil {
		return "", fmt.Errorf("private key loader is required")
	}
	return r.privateKey()
}

func params(req Request) internalclob.BalanceAllowanceParams {
	return internalclob.BalanceAllowanceParams{
		AssetType: req.AssetType,
		TokenID:   req.TokenID,
	}
}

func checkOutput(output string) error {
	if output != "" && output != "json" {
		return fmt.Errorf("only --output json is supported")
	}
	return nil
}

func scaleBaseUnits(value string, decimals int) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" || strings.Contains(value, ".") || decimals <= 0 {
		return value, false
	}
	n := new(big.Int)
	if _, ok := n.SetString(value, 10); !ok {
		return value, false
	}
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	whole := new(big.Int).Quo(new(big.Int).Set(n), scale)
	frac := new(big.Int).Mod(new(big.Int).Set(n), scale).String()
	if len(frac) < decimals {
		frac = strings.Repeat("0", decimals-len(frac)) + frac
	}
	return whole.String() + "." + frac, true
}
