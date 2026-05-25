// Package localpreflight owns local CLI readiness checks without Cobra coupling.
package localpreflight

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"github.com/TrebuchetDynamics/polygolem/internal/preflight"
)

// Config contains local readiness inputs.
type Config struct {
	Version     string
	BuilderCode string
}

// Runner owns local preflight check assembly behind a small interface.
type Runner struct {
	version     string
	builderCode string
}

// New creates a local preflight workflow runner.
func New(cfg Config) *Runner {
	return &Runner{version: cfg.Version, builderCode: cfg.BuilderCode}
}

// Run executes local, non-network readiness checks.
func (r *Runner) Run(ctx context.Context) preflight.Result {
	return preflight.Run(ctx, []preflight.Check{
		{Name: "version", Probe: r.checkVersion},
		{Name: "output", Probe: func(context.Context) error { return nil }},
		{Name: "clob_builder_code", Probe: r.checkBuilderCode},
	})
}

func (r *Runner) checkVersion(context.Context) error {
	if r.version == "" {
		return fmt.Errorf("version is empty")
	}
	return nil
}

func (r *Runner) checkBuilderCode(context.Context) error {
	return validateBuilderCode(r.builderCode)
}

// WriteText writes the human-readable local preflight report.
func WriteText(w io.Writer, result preflight.Result) error {
	status := "ok"
	if !result.OK {
		status = "failed"
	}
	if _, err := fmt.Fprintf(w, "preflight: %s\n", status); err != nil {
		return err
	}
	for _, check := range result.Checks {
		if check.Message == "" {
			if _, err := fmt.Fprintf(w, "- %s: %s\n", check.Name, check.Status); err != nil {
				return err
			}
			continue
		}
		if _, err := fmt.Fprintf(w, "- %s: %s (%s)\n", check.Name, check.Status, check.Message); err != nil {
			return err
		}
	}
	return nil
}

func validateBuilderCode(builderCode string) error {
	value := strings.TrimSpace(builderCode)
	if value == "" {
		return nil
	}
	if !strings.HasPrefix(value, "0x") {
		return fmt.Errorf("builder code must be a 0x-prefixed bytes32 hex string")
	}
	hexValue := value[2:]
	if len(hexValue) != 64 {
		return fmt.Errorf("builder code must be 32 bytes, got %d hex characters", len(hexValue))
	}
	if _, err := hex.DecodeString(hexValue); err != nil {
		return fmt.Errorf("builder code must be hex: %w", err)
	}
	return nil
}
