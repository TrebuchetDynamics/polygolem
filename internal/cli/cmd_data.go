package cli

import (
	"context"

	"github.com/TrebuchetDynamics/polygolem/internal/workflows/dataorderresults"
	"github.com/TrebuchetDynamics/polygolem/internal/workflows/datareads"
	sdkclob "github.com/TrebuchetDynamics/polygolem/pkg/clob"
	"github.com/spf13/cobra"
)

type dataReadRunner interface {
	Run(context.Context, datareads.Request) (any, error)
}

func dataCmd(jsonOut bool) *cobra.Command {
	w := newWire(jsonOut)
	return newDataCommand(datareads.New(w.data))
}

func newDataCommand(reads dataReadRunner) *cobra.Command {
	cmd := commandGroup("data", "Polymarket Data API analytics")

	var user string
	var tokenID string
	var limit int

	addUser := func(c *cobra.Command) {
		c.Flags().StringVar(&user, "user", "", "user wallet address")
	}
	addUserLimit := func(c *cobra.Command) {
		addUser(c)
		c.Flags().IntVar(&limit, "limit", 20, "max rows")
	}
	addToken := func(c *cobra.Command) {
		c.Flags().StringVar(&tokenID, "token-id", "", "CLOB token ID")
	}
	addTokenLimit := func(c *cobra.Command) {
		addToken(c)
		c.Flags().IntVar(&limit, "limit", 20, "max rows")
	}
	runRead := func(operation datareads.Operation) func(*cobra.Command, []string) error {
		return func(cmd *cobra.Command, args []string) error {
			result, err := reads.Run(cmd.Context(), datareads.Request{
				Operation: operation,
				User:      user,
				TokenID:   tokenID,
				Limit:     limit,
			})
			if err != nil {
				return err
			}
			return writeCommandJSON(cmd, result)
		}
	}

	positionsCmd := &cobra.Command{Use: "positions", Short: "List open positions for a user", Args: cobra.NoArgs,
		RunE: runRead(datareads.Positions),
	}
	addUserLimit(positionsCmd)
	cmd.AddCommand(positionsCmd)

	closedPositionsCmd := &cobra.Command{Use: "closed-positions", Short: "List closed positions for a user", Args: cobra.NoArgs,
		RunE: runRead(datareads.ClosedPositions),
	}
	addUserLimit(closedPositionsCmd)
	cmd.AddCommand(closedPositionsCmd)

	tradesCmd := &cobra.Command{Use: "trades", Short: "List public Data API trades for a user", Args: cobra.NoArgs,
		RunE: runRead(datareads.Trades),
	}
	addUserLimit(tradesCmd)
	cmd.AddCommand(tradesCmd)

	var includeCLOB bool
	orderResultsCmd := &cobra.Command{Use: "order-results", Short: "Join positions, trades, and results for a user", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			runner := dataorderresults.New(dataorderresults.Config{
				DataBaseURL: dataBaseURL,
				CLOBBaseURL: clobBaseURL,
				PrivateKey:  privateKeyFromEnv,
				CLOBCredentials: func() (sdkclob.APIKey, bool) {
					creds, ok := clobL2CredentialsFromEnv()
					if !ok {
						return sdkclob.APIKey{}, false
					}
					return sdkclob.APIKey{
						Key:        creds.Key,
						Secret:     creds.Secret,
						Passphrase: creds.Passphrase,
					}, true
				},
			})
			report, err := runner.Run(cmd.Context(), dataorderresults.Request{
				User:        user,
				Limit:       limit,
				IncludeCLOB: includeCLOB,
			})
			if err != nil {
				return err
			}
			return writeCommandJSON(cmd, report)
		},
	}
	addUserLimit(orderResultsCmd)
	orderResultsCmd.Flags().BoolVar(&includeCLOB, "include-clob", false, "include authenticated CLOB open orders and trade history")
	cmd.AddCommand(orderResultsCmd)

	activityCmd := &cobra.Command{Use: "activity", Short: "List public activity for a user", Args: cobra.NoArgs,
		RunE: runRead(datareads.Activity),
	}
	addUserLimit(activityCmd)
	cmd.AddCommand(activityCmd)

	holdersCmd := &cobra.Command{Use: "holders", Short: "List top holders for a token", Args: cobra.NoArgs,
		RunE: runRead(datareads.Holders),
	}
	addTokenLimit(holdersCmd)
	cmd.AddCommand(holdersCmd)

	valueCmd := &cobra.Command{Use: "value", Short: "Get total portfolio value for a user", Args: cobra.NoArgs,
		RunE: runRead(datareads.Value),
	}
	addUser(valueCmd)
	cmd.AddCommand(valueCmd)

	marketsTradedCmd := &cobra.Command{Use: "markets-traded", Short: "Get total markets traded for a user", Args: cobra.NoArgs,
		RunE: runRead(datareads.MarketsTraded),
	}
	addUser(marketsTradedCmd)
	cmd.AddCommand(marketsTradedCmd)

	openInterestCmd := &cobra.Command{Use: "open-interest", Short: "Get open interest for a token", Args: cobra.NoArgs,
		RunE: runRead(datareads.OpenInterest),
	}
	addToken(openInterestCmd)
	cmd.AddCommand(openInterestCmd)

	leaderboardCmd := &cobra.Command{Use: "leaderboard", Short: "List trader leaderboard rows", Args: cobra.NoArgs,
		RunE: runRead(datareads.Leaderboard),
	}
	leaderboardCmd.Flags().IntVar(&limit, "limit", 20, "max rows")
	cmd.AddCommand(leaderboardCmd)

	liveVolumeCmd := &cobra.Command{Use: "live-volume", Short: "Get live volume summary", Args: cobra.NoArgs,
		RunE: runRead(datareads.LiveVolume),
	}
	liveVolumeCmd.Flags().IntVar(&limit, "limit", 20, "max rows")
	cmd.AddCommand(liveVolumeCmd)

	return cmd
}
