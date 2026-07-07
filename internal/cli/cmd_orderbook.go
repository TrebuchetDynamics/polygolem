package cli

import (
	"context"

	"github.com/TrebuchetDynamics/polygolem/internal/workflows/orderbookreads"
	"github.com/spf13/cobra"
)

type orderbookRunner interface {
	Run(context.Context, orderbookreads.Request) (any, error)
}

func orderbookCmd(jsonOut bool) *cobra.Command {
	w := newWire(jsonOut)
	return newOrderbookCommand(orderbookreads.New(w.clob))
}

func newOrderbookCommand(runner orderbookRunner) *cobra.Command {
	var tokenID string

	cmd := commandGroup("book", "Read CLOB order book data")
	cmd.Long = `Read the live CLOB order book for a token. Read-only; no credentials.

Fetch bids/asks, best bid/ask, midpoint, and spread for a token id (get one from
'polygolem markets'). For a continuously updated view, see 'polygolem prices
live'; for the raw event stream, see 'polygolem stream'.`
	cmd.Example = `  polygolem markets search --query "Will BTC" --limit 1   # find a token id
  polygolem book get --token-id <id> --json`

	for _, spec := range []struct {
		use, short string
		op         orderbookreads.Operation
	}{
		{"get", "Get L2 order book", orderbookreads.Get},
		{"price", "Get best price (BUY side)", orderbookreads.Price},
		{"midpoint", "Get midpoint price", orderbookreads.Midpoint},
		{"spread", "Get bid-ask spread", orderbookreads.Spread},
		{"tick-size", "Get minimum tick size", orderbookreads.TickSize},
		{"fee-rate", "Get fee rate in bps", orderbookreads.FeeRate},
		{"last-trade", "Get last trade price", orderbookreads.LastTrade},
	} {
		sub := spec
		c := &cobra.Command{Use: sub.use, Short: sub.short, Args: cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				result, err := runner.Run(cmd.Context(), orderbookreads.Request{Operation: sub.op, TokenID: tokenID})
				if err != nil {
					return err
				}
				return writeCommandJSON(cmd, result)
			},
		}
		c.Flags().StringVar(&tokenID, "token-id", "", "CLOB token ID")
		cmd.AddCommand(c)
	}
	return cmd
}
