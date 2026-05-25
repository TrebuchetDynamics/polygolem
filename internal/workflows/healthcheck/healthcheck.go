// Package healthcheck owns read-only upstream reachability checks without Cobra coupling.
package healthcheck

import "context"

// Probe checks one upstream dependency.
type Probe func(context.Context) error

// Config contains the read-only health probes.
type Config struct {
	Gamma Probe
	CLOB  Probe
}

// Result maps dependency names to "ok" or an error message.
type Result map[string]string

// Runner executes read-only upstream reachability checks.
type Runner struct {
	gamma Probe
	clob  Probe
}

// New creates a health-check workflow runner.
func New(cfg Config) *Runner {
	return &Runner{gamma: cfg.Gamma, clob: cfg.CLOB}
}

// Run executes all configured health checks and does not short-circuit on errors.
func (r *Runner) Run(ctx context.Context) Result {
	status := Result{"gamma": "ok", "clob": "ok"}
	if err := r.gamma(ctx); err != nil {
		status["gamma"] = err.Error()
	}
	if err := r.clob(ctx); err != nil {
		status["clob"] = err.Error()
	}
	return status
}
