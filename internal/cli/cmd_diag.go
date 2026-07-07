package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/TrebuchetDynamics/polygolem/internal/auth"
	"github.com/TrebuchetDynamics/polygolem/internal/preflight"
	"github.com/TrebuchetDynamics/polygolem/internal/workflows/localpreflight"
	"github.com/spf13/cobra"
)

type diagReport struct {
	Version   string                  `json:"version"`
	Endpoints map[string]string       `json:"endpoints"`
	Env       map[string]diagEnvValue `json:"env"`
	Preflight preflight.Result        `json:"preflight"`
}

type diagEnvValue struct {
	Set   bool   `json:"set"`
	Value string `json:"value,omitempty"`
}

func diagCmd(_ bool, version string) *cobra.Command {
	cmd := &cobra.Command{Use: "diag", Short: "Print redacted local diagnostics", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			runner := localpreflight.New(localpreflight.Config{Version: version, BuilderCode: builderCodeFromFlagOrEnv("")})
			report := buildDiagReport(cmd.Context(), version, runner)
			if jsonEnabled(cmd) {
				return writeCommandJSON(cmd, report)
			}
			return writeDiagText(cmd.OutOrStdout(), report)
		},
	}
	return cmd
}

func buildDiagReport(ctx context.Context, version string, runner *localpreflight.Runner) diagReport {
	return diagReport{
		Version: version,
		Endpoints: map[string]string{
			"gamma":         gammaBaseURL,
			"clob":          firstNonEmptyCLI(firstEnv("POLYMARKET_CLOB_URL", "CLOB_URL"), clobBaseURL),
			"data":          dataBaseURL,
			"market_stream": marketStreamBaseURL,
			"user_stream":   userStreamBaseURL,
		},
		Env: map[string]diagEnvValue{
			"SIGNER_PRIVATE_KEY":      redactedEnv("SIGNER_PRIVATE_KEY"),
			"POLYMARKET_PRIVATE_KEY":  redactedEnv("POLYMARKET_PRIVATE_KEY"),
			"POLYMARKET_BUILDER_CODE": redactedEnv("POLYMARKET_BUILDER_CODE"),
			"POLYMARKET_CLOB_URL":     redactedEnv("POLYMARKET_CLOB_URL"),
			"CLOB_URL":                redactedEnv("CLOB_URL"),
		},

		Preflight: runner.Run(ctx),
	}
}

func redactedEnv(key string) diagEnvValue {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return diagEnvValue{Set: false}
	}
	return diagEnvValue{Set: true, Value: auth.Redact(value)}
}

func writeDiagText(w io.Writer, report diagReport) error {
	if _, err := fmt.Fprintf(w, "diag: polygolem %s\n", report.Version); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "endpoints:"); err != nil {
		return err
	}
	for _, key := range []string{"gamma", "clob", "data", "market_stream", "user_stream"} {
		if _, err := fmt.Fprintf(w, "- %s: %s\n", key, report.Endpoints[key]); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w, "env:"); err != nil {
		return err
	}
	for _, key := range []string{"SIGNER_PRIVATE_KEY", "POLYMARKET_PRIVATE_KEY", "POLYMARKET_BUILDER_CODE", "POLYMARKET_CLOB_URL", "CLOB_URL"} {
		value := report.Env[key]
		if !value.Set {
			if _, err := fmt.Fprintf(w, "- %s: unset\n", key); err != nil {
				return err
			}
			continue
		}
		if _, err := fmt.Fprintf(w, "- %s: %s\n", key, value.Value); err != nil {
			return err
		}
	}
	return nil
}
