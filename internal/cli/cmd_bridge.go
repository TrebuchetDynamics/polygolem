package cli

import (
	"context"

	"github.com/TrebuchetDynamics/polygolem/internal/workflows/bridgeassets"
	"github.com/TrebuchetDynamics/polygolem/pkg/bridge"
	"github.com/spf13/cobra"
)

type bridgeAssetsRunner interface {
	Run(context.Context) (*bridge.SupportedAssetsResponse, error)
}

type bridgeClient interface {
	CreateDepositAddress(context.Context, string) (*bridge.CreateDepositAddressResponse, error)
	GetDepositStatus(context.Context, string) (*bridge.DepositStatusResponse, error)
	GetQuote(context.Context, bridge.QuoteRequest) (*bridge.QuoteResponse, error)
}

func bridgeCmd(jsonOut bool) *cobra.Command {
	bc := bridge.NewClient("", nil)
	return newBridgeCommand(bridgeassets.New(bc), bc)
}

func newBridgeCommand(assetsRunner bridgeAssetsRunner, bc bridgeClient) *cobra.Command {
	cmd := commandGroup("bridge", "Polymarket Bridge API")
	cmd.Long = `Cross-chain deposits into Polymarket USD (pUSD) via the bridge.

Read-only: assets (supported tokens/chains), status, quote. Deposit-address
creation is supported. Live withdrawal/offramp submission is intentionally
unsupported and returns an explicit error rather than moving funds.`
	cmd.Example = `  polygolem bridge assets --json
  polygolem bridge status <address> --json`
	cmd.AddCommand(&cobra.Command{
		Use: "assets", Short: "List supported bridge assets", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := assetsRunner.Run(cmd.Context())
			if err != nil {
				return err
			}
			return writeCommandJSON(cmd, a)
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use: "deposit <address>", Short: "Create deposit addresses", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := bc.CreateDepositAddress(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return writeCommandJSON(cmd, d)
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use: "status <deposit-address>", Short: "Get bridge deposit status", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := bc.GetDepositStatus(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return writeCommandJSON(cmd, s)
		},
	})
	var quoteReq bridge.QuoteRequest
	quoteCmd := &cobra.Command{
		Use:   "quote",
		Short: "Quote a bridge deposit move",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			q, err := bc.GetQuote(cmd.Context(), quoteReq)
			if err != nil {
				return err
			}
			return writeCommandJSON(cmd, q)
		},
	}
	quoteCmd.Flags().StringVar(&quoteReq.FromAmountBaseUnit, "from-amount-base-unit", "", "source amount in base units")
	quoteCmd.Flags().StringVar(&quoteReq.FromChainID, "from-chain-id", "", "source chain ID")
	quoteCmd.Flags().StringVar(&quoteReq.FromTokenAddress, "from-token-address", "", "source token address")
	quoteCmd.Flags().StringVar(&quoteReq.RecipientAddress, "recipient-address", "", "recipient Polymarket address")
	quoteCmd.Flags().StringVar(&quoteReq.ToChainID, "to-chain-id", "", "destination chain ID")
	quoteCmd.Flags().StringVar(&quoteReq.ToTokenAddress, "to-token-address", "", "destination token address")
	_ = quoteCmd.MarkFlagRequired("from-amount-base-unit")
	_ = quoteCmd.MarkFlagRequired("from-chain-id")
	_ = quoteCmd.MarkFlagRequired("from-token-address")
	_ = quoteCmd.MarkFlagRequired("recipient-address")
	_ = quoteCmd.MarkFlagRequired("to-chain-id")
	_ = quoteCmd.MarkFlagRequired("to-token-address")
	cmd.AddCommand(quoteCmd)
	return cmd
}
