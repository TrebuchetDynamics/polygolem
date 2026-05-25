// Package clobaccountreads reads authenticated CLOB account state without Cobra coupling.
package clobaccountreads

import (
	"context"
	"fmt"

	internalclob "github.com/TrebuchetDynamics/polygolem/internal/clob"
)

// PrivateKeyLoader loads the EOA private key used for authenticated CLOB reads.
type PrivateKeyLoader func() (string, error)

// Reader is the authenticated read-only CLOB account interface used by the workflow.
type Reader interface {
	ListOrders(context.Context, string) ([]internalclob.OrderRecord, error)
	Order(context.Context, string, string) (*internalclob.OrderRecord, error)
	ListTrades(context.Context, string) ([]internalclob.TradeRecord, error)
}

// Config contains adapters used by the CLOB account-read workflow.
type Config struct {
	Reader     Reader
	PrivateKey PrivateKeyLoader
}

// Request describes an authenticated account list read.
type Request struct {
	Output string
}

// OrderRequest describes a single authenticated order read.
type OrderRequest struct {
	OrderID string
	Output  string
}

// Runner owns authenticated CLOB account-read orchestration behind a small interface.
type Runner struct {
	reader     Reader
	privateKey PrivateKeyLoader
}

// New creates a CLOB account-read workflow runner.
func New(cfg Config) *Runner {
	return &Runner{reader: cfg.Reader, privateKey: cfg.PrivateKey}
}

// Orders lists authenticated CLOB orders.
func (r *Runner) Orders(ctx context.Context, req Request) ([]internalclob.OrderRecord, error) {
	key, err := r.privateKeyAfterOutput(req.Output)
	if err != nil {
		return nil, err
	}
	return r.reader.ListOrders(ctx, key)
}

// Order reads one authenticated CLOB order.
func (r *Runner) Order(ctx context.Context, req OrderRequest) (*internalclob.OrderRecord, error) {
	key, err := r.privateKeyAfterOutput(req.Output)
	if err != nil {
		return nil, err
	}
	return r.reader.Order(ctx, key, req.OrderID)
}

// Trades lists authenticated CLOB trades.
func (r *Runner) Trades(ctx context.Context, req Request) ([]internalclob.TradeRecord, error) {
	key, err := r.privateKeyAfterOutput(req.Output)
	if err != nil {
		return nil, err
	}
	return r.reader.ListTrades(ctx, key)
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

func checkOutput(output string) error {
	if output != "" && output != "json" {
		return fmt.Errorf("only --output json is supported")
	}
	return nil
}
