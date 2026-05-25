// Package clobdiagnostics reads authenticated CLOB diagnostic state without Cobra coupling.
package clobdiagnostics

import (
	"context"
	"fmt"

	internalclob "github.com/TrebuchetDynamics/polygolem/internal/clob"
)

// PrivateKeyLoader loads the EOA private key used for authenticated CLOB diagnostics.
type PrivateKeyLoader func() (string, error)

// Reader is the authenticated diagnostic CLOB interface used by the workflow.
type Reader interface {
	ListBuilderFeeKeys(context.Context, string) ([]internalclob.BuilderFeeKeyRecord, error)
	MarketTradesProbe(context.Context, string, internalclob.MarketTradesProbeRequest) (*internalclob.MarketTradesProbeResult, error)
}

// Config contains adapters used by the CLOB diagnostic workflow.
type Config struct {
	Reader     Reader
	PrivateKey PrivateKeyLoader
}

// Request describes an authenticated diagnostic list read.
type Request struct {
	Output string
}

// ProbeRequest describes a market-trades diagnostic probe.
type ProbeRequest struct {
	Market     string
	AssetID    string
	NextCursor string
	Output     string
}

// Runner owns authenticated CLOB diagnostic orchestration behind a small interface.
type Runner struct {
	reader     Reader
	privateKey PrivateKeyLoader
}

// New creates a CLOB diagnostic workflow runner.
func New(cfg Config) *Runner {
	return &Runner{reader: cfg.Reader, privateKey: cfg.PrivateKey}
}

// ListBuilderFeeKeys lists builder fee keys for the authenticated wallet.
func (r *Runner) ListBuilderFeeKeys(ctx context.Context, req Request) ([]internalclob.BuilderFeeKeyRecord, error) {
	key, err := r.privateKeyAfterOutput(req.Output)
	if err != nil {
		return nil, err
	}
	return r.reader.ListBuilderFeeKeys(ctx, key)
}

// MarketTradesProbe probes CLOB trade scope for one market or token.
func (r *Runner) MarketTradesProbe(ctx context.Context, req ProbeRequest) (*internalclob.MarketTradesProbeResult, error) {
	key, err := r.privateKeyAfterOutput(req.Output)
	if err != nil {
		return nil, err
	}
	return r.reader.MarketTradesProbe(ctx, key, internalclob.MarketTradesProbeRequest{
		Market:     req.Market,
		AssetID:    req.AssetID,
		NextCursor: req.NextCursor,
	})
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
