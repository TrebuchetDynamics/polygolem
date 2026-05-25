package cli

import (
	"context"

	"github.com/TrebuchetDynamics/polygolem/internal/workflows/healthcheck"
	"github.com/spf13/cobra"
)

type healthRunner interface {
	Run(context.Context) healthcheck.Result
}

func healthCmd(jsonOut bool) *cobra.Command {
	w := newWire(jsonOut)
	runner := healthcheck.New(healthcheck.Config{
		Gamma: func(ctx context.Context) error {
			_, err := w.gamma.HealthCheck(ctx)
			return err
		},
		CLOB: w.clob.Health,
	})
	return newHealthCommand(runner)
}

func newHealthCommand(runner healthRunner) *cobra.Command {
	return &cobra.Command{
		Use: "health", Short: "Check Gamma and CLOB API reachability", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return writeCommandJSON(cmd, runner.Run(cmd.Context()))
		},
	}
}
