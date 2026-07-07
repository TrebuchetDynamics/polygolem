package cli

import (
	"github.com/TrebuchetDynamics/polygolem/internal/paper"
	"github.com/TrebuchetDynamics/polygolem/internal/workflows/paperaccount"
	"github.com/TrebuchetDynamics/polygolem/internal/workflows/papercrypto"
	"github.com/TrebuchetDynamics/polygolem/internal/workflows/papertrade"
	"github.com/TrebuchetDynamics/polygolem/pkg/marketresolver"
	"github.com/spf13/cobra"
)

func paperCmd(jsonOut bool) *cobra.Command {
	w := newWire(jsonOut)
	return newPaperCommand(jsonOut, w.clob)
}

func newPaperCommand(jsonOut bool, pricer paperaccount.Pricer) *cobra.Command {
	cmd := commandGroup("sim", "Paper trading simulation for crypto markets")
	cmd.Long = `Simulate crypto up/down trading with no wallet and no risk.

Paper mode holds a local cash/position ledger and prices simulations against live
public market data. It never loads a private key, signs, or submits anything — a
safe way to rehearse the flow before trading real funds.

  polygolem sim reset --cash 100
  polygolem sim trade --asset BTC --interval 5m --side up --size 1
  polygolem sim positions`

	var paperCash float64
	var tokenID string
	var priceStr string
	var sizeStr string

	paperState := paper.NewState("USD", 10000.0)
	account := paperaccount.New(paperaccount.Config{State: paperState, Pricer: pricer})

	buyCmd := &cobra.Command{
		Use:   "buy",
		Short: "Simulate a buy order (paper trading)",
		Long: `Simulate a buy order against live market data.
Uses current best ask price if --price is not specified.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := account.Buy(cmd.Context(), paperaccount.TradeRequest{TokenID: tokenID, Price: priceStr, Size: sizeStr})
			if err != nil {
				return err
			}
			return writeCommandJSON(cmd, result)
		},
	}
	buyCmd.Flags().StringVar(&tokenID, "token-id", "", "CLOB token ID to buy")
	buyCmd.Flags().StringVar(&priceStr, "price", "", "limit price (default: best ask)")
	buyCmd.Flags().StringVar(&sizeStr, "size", "1", "number of shares")
	cmd.AddCommand(buyCmd)

	sellCmd := &cobra.Command{
		Use:   "sell",
		Short: "Simulate a sell order (paper trading)",
		Long: `Simulate a sell order against live market data.
Uses current best bid price if --price is not specified.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := account.Sell(cmd.Context(), paperaccount.TradeRequest{TokenID: tokenID, Price: priceStr, Size: sizeStr})
			if err != nil {
				return err
			}
			return writeCommandJSON(cmd, result)
		},
	}
	sellCmd.Flags().StringVar(&tokenID, "token-id", "", "CLOB token ID to sell")
	sellCmd.Flags().StringVar(&priceStr, "price", "", "limit price (default: best bid)")
	sellCmd.Flags().StringVar(&sizeStr, "size", "1", "number of shares")
	cmd.AddCommand(sellCmd)

	positionsCmd := &cobra.Command{
		Use:   "positions",
		Short: "Show current paper trading positions",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return writeCommandJSON(cmd, account.Positions())
		},
	}
	cmd.AddCommand(positionsCmd)

	resetCmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset paper trading state",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return writeCommandJSON(cmd, account.Reset(paperCash))
		},
	}
	resetCmd.Flags().Float64Var(&paperCash, "cash", 10000.0, "initial paper cash")
	cmd.AddCommand(resetCmd)

	var cryptoAsset, cryptoInterval string
	var cryptoLimit int
	cryptoCmd := &cobra.Command{
		Use:   "crypto",
		Short: "Discover crypto markets and paper trade",
		Long: `Find active crypto markets and get token IDs for paper trading.

Examples:
  polygolem sim crypto --asset BTC --interval 5m
  polygolem sim crypto --asset ETH --limit 10`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			w := newWire(jsonOut)
			result, err := papercrypto.New(w.gamma).Run(cmd.Context(), papercrypto.Request{
				Asset:    cryptoAsset,
				Interval: cryptoInterval,
				Limit:    cryptoLimit,
			})
			if err != nil {
				return err
			}
			return writeCommandJSON(cmd, result)
		},
	}
	cryptoCmd.Flags().StringVar(&cryptoAsset, "asset", "", "crypto asset filter (BTC, ETH, SOL, etc.)")
	cryptoCmd.Flags().StringVar(&cryptoInterval, "interval", "", "interval filter (5m, 15m, 1h)")
	cryptoCmd.Flags().IntVar(&cryptoLimit, "limit", 10, "max markets")
	cmd.AddCommand(cryptoCmd)

	var tradeAsset, tradeInterval, tradeSide string
	var tradeSize float64
	tradeCmd := &cobra.Command{
		Use:   "trade",
		Short: "Paper trade the current crypto window in one command",
		Long: `Resolve the current crypto window, fetch live price, and execute a paper trade.

Examples:
  polygolem sim trade --asset BTC --interval 5m --side up --size 1
  polygolem sim trade --asset ETH --interval 15m --side down --size 2 --price 0.48`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			runner := papertrade.New(marketresolver.NewResolver(gammaBaseURL), pricer, paperState)
			result, err := runner.Run(cmd.Context(), papertrade.Request{
				Asset:    tradeAsset,
				Interval: tradeInterval,
				Side:     tradeSide,
				TokenID:  tokenID,
				Price:    priceStr,
				Size:     tradeSize,
			})
			if err != nil {
				return err
			}
			return writeCommandJSON(cmd, result)
		},
	}
	tradeCmd.Flags().StringVar(&tradeAsset, "asset", "", "crypto asset (BTC, ETH, SOL, XRP, DOGE, BNB)")
	tradeCmd.Flags().StringVar(&tradeInterval, "interval", "", "time interval (5m, 15m, 1h, 4h)")
	tradeCmd.Flags().StringVar(&tradeSide, "side", "", "trade side: up or down")
	tradeCmd.Flags().Float64Var(&tradeSize, "size", 1.0, "number of shares")
	tradeCmd.Flags().StringVar(&tokenID, "token-id", "", "bypass resolution and trade this token ID directly")
	tradeCmd.Flags().StringVar(&priceStr, "price", "", "limit price (default: best ask/bid)")
	cmd.AddCommand(tradeCmd)

	return cmd
}
