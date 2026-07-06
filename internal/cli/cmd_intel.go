package cli

import (
	"context"
	"fmt"

	intelworkflow "github.com/TrebuchetDynamics/polygolem/internal/intel"
	"github.com/TrebuchetDynamics/polygolem/pkg/data"
	sdkintel "github.com/TrebuchetDynamics/polygolem/pkg/intel"
	"github.com/spf13/cobra"
)

type intelRunner interface {
	WalletDossier(context.Context, string, intelworkflow.DossierOptions) (*sdkintel.WalletDossier, error)
	Leaderboard(context.Context, intelworkflow.LeaderboardOptions) ([]sdkintel.LeaderboardRow, error)
	Alerts(context.Context, intelworkflow.AlertOptions) ([]sdkintel.Signal, error)
	MarketFlow(context.Context, string, intelworkflow.MarketFlowOptions) (*sdkintel.MarketFlow, error)
}

func intelCmd(jsonOut bool) *cobra.Command {
	_ = jsonOut
	return newIntelCommand(intelworkflow.NewService(data.NewClient(data.Config{BaseURL: dataBaseURL})))
}

func newIntelCommand(runner intelRunner) *cobra.Command {
	cmd := commandGroup("intel", "Read-only wallet intelligence")
	cmd.Long = `Reproducible, read-only statistical signals about public wallet activity.

Scores are computed only from public Data API rows (trades, activity, closed
positions) with a named formula version and the source rows exposed. A signal is
research context, not trading advice and not a finding of misconduct.`
	cmd.Example = `  polygolem intel wallet <address> --json`

	var walletLimit int
	walletCmd := &cobra.Command{
		Use:   "wallet <wallet>",
		Short: "Build a read-only wallet intelligence dossier",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dossier, err := runner.WalletDossier(cmd.Context(), args[0], intelworkflow.DossierOptions{Limit: walletLimit})
			if err != nil {
				return err
			}
			return writeCommandJSON(cmd, dossier)
		},
	}
	walletCmd.Flags().IntVar(&walletLimit, "limit", 100, "max source rows per Data API read")
	cmd.AddCommand(walletCmd)

	var leaderboardLimit int
	var leaderboardSort string
	leaderboardCmd := &cobra.Command{
		Use:   "leaderboard",
		Short: "List Data-API-ranked wallet intelligence rows",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if leaderboardSort != "" && leaderboardSort != "data-api-rank" {
				return fmt.Errorf("intel leaderboard: unsupported --sort %q; V1 supports data-api-rank because leaderboard rows do not expose wins/bets", leaderboardSort)
			}
			rows, err := runner.Leaderboard(cmd.Context(), intelworkflow.LeaderboardOptions{Limit: leaderboardLimit})
			if err != nil {
				return err
			}
			return writeCommandJSON(cmd, rows)
		},
	}
	leaderboardCmd.Flags().IntVar(&leaderboardLimit, "limit", 20, "max leaderboard rows")
	leaderboardCmd.Flags().StringVar(&leaderboardSort, "sort", "data-api-rank", "sort mode (data-api-rank)")
	cmd.AddCommand(leaderboardCmd)

	var alertsUser string
	var alertsLimit int
	var alertsMinScore int
	alertsCmd := &cobra.Command{
		Use:   "alerts",
		Short: "List user-scoped wallet dossier alerts",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			signals, err := runner.Alerts(cmd.Context(), intelworkflow.AlertOptions{User: alertsUser, Limit: alertsLimit, MinScore: alertsMinScore})
			if err != nil {
				return err
			}
			return writeCommandJSON(cmd, sdkintel.DossierAlerts{DossierAlerts: signals})
		},
	}
	alertsCmd.Flags().StringVar(&alertsUser, "user", "", "user wallet address")
	alertsCmd.Flags().IntVar(&alertsLimit, "limit", 100, "max source rows per Data API read")
	alertsCmd.Flags().IntVar(&alertsMinScore, "min-score", 70, "minimum candidate score")
	cmd.AddCommand(alertsCmd)

	var marketFlowLimit int
	marketFlowCmd := &cobra.Command{
		Use:   "market-flow <market-or-token-id>",
		Short: "Summarize read-only market holder, trade, and open-interest flow",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flow, err := runner.MarketFlow(cmd.Context(), args[0], intelworkflow.MarketFlowOptions{Limit: marketFlowLimit})
			if err != nil {
				return err
			}
			return writeCommandJSON(cmd, flow)
		},
	}
	marketFlowCmd.Flags().IntVar(&marketFlowLimit, "limit", 100, "max holder/trade rows")
	cmd.AddCommand(marketFlowCmd)

	return cmd
}
