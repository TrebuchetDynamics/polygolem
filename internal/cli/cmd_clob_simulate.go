package cli

import (
	"context"

	"github.com/TrebuchetDynamics/polygolem/internal/workflows/clobsimulation"
	"github.com/spf13/cobra"
)

type clobSimulationCommandRunner interface {
	SimulateOrder(context.Context, clobsimulation.Request) (*clobsimulation.Result, error)
}

func addCLOBSimulateOrderCommand(cmd *cobra.Command, runner clobSimulationCommandRunner) {
	var output, token, side, amount, limitPrice string
	simulateCmd := &cobra.Command{
		Use:   "simulate-order",
		Short: "Simulate a read-only CLOB order fill from the current book",
		Long: `Walks the opposing side of the live CLOB book and estimates the fill,
average price, and slippage for a proposed order. This command is read-only:
it does not load a private key, sign, or submit an order.

For buys, --amount is USDC notional to spend. For sells, --amount is shares to sell.
Use --limit-price to stop the walk at a worst acceptable price.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := runner.SimulateOrder(cmd.Context(), clobsimulation.Request{
				TokenID:    token,
				Side:       side,
				Amount:     amount,
				LimitPrice: limitPrice,
				Output:     output,
			})
			if err != nil {
				return err
			}
			return writeCommandJSON(cmd, res)
		},
	}
	addCLOBOutputFlag(simulateCmd, &output)
	simulateCmd.Flags().StringVar(&token, "token", "", "CLOB token id")
	simulateCmd.Flags().StringVar(&side, "side", "buy", "order side: buy or sell")
	simulateCmd.Flags().StringVar(&amount, "amount", "", "buy: USDC notional to spend; sell: number of shares to sell")
	simulateCmd.Flags().StringVar(&limitPrice, "limit-price", "", "optional worst acceptable price for simulated taker fill")
	cmd.AddCommand(simulateCmd)
}
