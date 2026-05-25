// Package dataorderresults builds the read-only data order-results report without Cobra coupling.
package dataorderresults

import (
	"context"
	"fmt"
	"strings"

	sdkclob "github.com/TrebuchetDynamics/polygolem/pkg/clob"
	sdkdata "github.com/TrebuchetDynamics/polygolem/pkg/data"
	sdkorderresults "github.com/TrebuchetDynamics/polygolem/pkg/orderresults"
)

// PrivateKeyLoader loads the private key only when authenticated CLOB reads are requested.
type PrivateKeyLoader func() (string, error)

// CLOBCredentialsLoader loads optional CLOB L2 API credentials for authenticated reads.
type CLOBCredentialsLoader func() (sdkclob.APIKey, bool)

// CLOBFactory creates an authenticated CLOB reader from SDK config.
type CLOBFactory func(sdkclob.Config) sdkorderresults.CLOBReader

// Config contains adapters used by the order-results workflow.
type Config struct {
	Data            sdkorderresults.DataReader
	DataBaseURL     string
	CLOBBaseURL     string
	PrivateKey      PrivateKeyLoader
	CLOBCredentials CLOBCredentialsLoader
	CLOBFactory     CLOBFactory
}

// Request describes one order-results report run.
type Request struct {
	User        string
	Limit       int
	IncludeCLOB bool
}

// Runner owns CLI-level order-results orchestration behind a small interface.
type Runner struct {
	data            sdkorderresults.DataReader
	clobBaseURL     string
	privateKey      PrivateKeyLoader
	clobCredentials CLOBCredentialsLoader
	clobFactory     CLOBFactory
}

// New creates an order-results workflow runner.
func New(cfg Config) *Runner {
	data := cfg.Data
	if data == nil {
		data = sdkdata.NewClient(sdkdata.Config{BaseURL: cfg.DataBaseURL})
	}
	clobFactory := cfg.CLOBFactory
	if clobFactory == nil {
		clobFactory = func(cfg sdkclob.Config) sdkorderresults.CLOBReader {
			return sdkclob.NewClient(cfg)
		}
	}
	return &Runner{
		data:            data,
		clobBaseURL:     cfg.CLOBBaseURL,
		privateKey:      cfg.PrivateKey,
		clobCredentials: cfg.CLOBCredentials,
		clobFactory:     clobFactory,
	}
}

// Run builds the report, loading CLOB auth material only for IncludeCLOB requests.
func (r *Runner) Run(ctx context.Context, req Request) (*sdkorderresults.Report, error) {
	user := strings.TrimSpace(req.User)
	if user == "" {
		return nil, fmt.Errorf("--user required")
	}

	source := sdkorderresults.Source{Data: r.data}
	opts := sdkorderresults.Options{Limit: req.Limit}
	if req.IncludeCLOB {
		if r.privateKey == nil {
			return nil, fmt.Errorf("private key loader is required")
		}
		privateKey, err := r.privateKey()
		if err != nil {
			return nil, err
		}
		opts.IncludeCLOB = true
		opts.PrivateKey = privateKey

		cfg := sdkclob.Config{BaseURL: r.clobBaseURL}
		if r.clobCredentials != nil {
			if creds, ok := r.clobCredentials(); ok {
				cfg.Credentials = creds
			}
		}
		source.CLOB = r.clobFactory(cfg)
	}

	return sdkorderresults.BuildReport(ctx, source, user, opts)
}
