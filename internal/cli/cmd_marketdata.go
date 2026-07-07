package cli

import (
	"fmt"

	"github.com/TrebuchetDynamics/polygolem/internal/workflows/marketdatacrypto"
	"github.com/TrebuchetDynamics/polygolem/internal/workflows/marketdatalive"
	"github.com/spf13/cobra"
)

func marketDataCmd(jsonOut bool) *cobra.Command {
	w := newWire(jsonOut)
	cmd := commandGroup("prices", "Live CLOB orderbook and share-price snapshots")
	cmd.Long = `Live, normalized market-data snapshots per token. Read-only; no credentials.

'prices live' subscribes to the public CLOB stream and reports the latest best
bid/ask, spread, midpoint, tick size, last trade, and book levels as they update —
a higher-level, normalized view than the raw 'stream' events.`
	cmd.Example = `  polygolem prices live --asset-ids <id>`

	var assetsRaw string
	var url string
	var maxMessages int
	var customFeatures bool
	var level int
	liveCmd := &cobra.Command{Use: "live", Short: "Stream enriched CLOB market-data snapshots", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return marketdatalive.New(marketdatalive.NewSDKStreamer).Run(
				cmd.Context(),
				marketdatalive.Request{
					AssetIDsRaw:    assetsRaw,
					URL:            url,
					MaxMessages:    maxMessages,
					CustomFeatures: customFeatures,
					Level:          level,
				},
				func(v interface{}) { _ = w.printJSON(cmd, v) },
				func(err error) { _, _ = fmt.Fprintf(cmd.ErrOrStderr(), "prices stream error: %v\n", err) },
			)
		},
	}
	liveCmd.Flags().StringVar(&assetsRaw, "asset-ids", "", "comma-separated CLOB token IDs")
	liveCmd.Flags().StringVar(&url, "url", marketStreamBaseURL, "WebSocket URL")
	liveCmd.Flags().IntVar(&maxMessages, "max-messages", 0, "stop after this many snapshots; 0 streams until interrupted")
	liveCmd.Flags().BoolVar(&customFeatures, "custom-features", true, "request best-bid-ask and market lifecycle events")
	liveCmd.Flags().IntVar(&level, "level", 0, "optional Polymarket market-stream subscription level")
	cmd.AddCommand(liveCmd)

	var cryptoAsset string
	var cryptoInterval string
	var cryptoLimit int
	cryptoCmd := &cobra.Command{
		Use:   "crypto",
		Short: "Get live marketdata snapshots for crypto markets",
		Long: `Discover crypto markets and fetch current CLOB snapshots (price, spread,
order book) for each. Returns a single snapshot per market — no continuous stream.

Examples:
  polygolem prices crypto --asset BTC --interval 5m    # BTC 5m snapshots
  polygolem prices crypto --asset ETH --limit 10       # ETH market snapshots`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := marketdatacrypto.New(w.gamma, w.clob).Run(cmd.Context(), marketdatacrypto.Request{
				Asset:    cryptoAsset,
				Interval: cryptoInterval,
				Limit:    cryptoLimit,
			})
			if err != nil {
				return err
			}
			return w.printJSON(cmd, result)
		},
	}
	cryptoCmd.Flags().StringVar(&cryptoAsset, "asset", "", "crypto asset filter (BTC, ETH, SOL, XRP, DOGE, BNB, HYPE)")
	cryptoCmd.Flags().StringVar(&cryptoInterval, "interval", "", "interval filter (5m, 15m, 1h)")
	cryptoCmd.Flags().IntVar(&cryptoLimit, "limit", 20, "max markets")
	cmd.AddCommand(cryptoCmd)

	return cmd
}
