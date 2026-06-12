package cli

import (
	"fmt"

	"github.com/TrebuchetDynamics/polygolem/internal/workflows/streamcrypto"
	"github.com/TrebuchetDynamics/polygolem/internal/workflows/streammarket"
	"github.com/TrebuchetDynamics/polygolem/internal/workflows/streamuser"
	"github.com/spf13/cobra"
)

func streamCmd(jsonOut bool) *cobra.Command {
	w := newWire(jsonOut)
	cmd := commandGroup("stream", "Polymarket WebSocket streams")

	var assetsRaw string
	var url string
	var maxMessages int
	var customFeatures bool
	var level int
	var stats bool
	marketCmd := &cobra.Command{Use: "market", Short: "Stream public CLOB market events", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return streammarket.New(streammarket.NewInternalStreamer).Run(
				cmd.Context(),
				streammarket.Request{
					AssetIDsRaw:    assetsRaw,
					URL:            url,
					MaxMessages:    maxMessages,
					CustomFeatures: customFeatures,
					Level:          level,
					Stats:          stats,
				},
				func(v interface{}) { _ = w.printJSON(cmd, v) },
				func(err error) { _, _ = fmt.Fprintf(cmd.ErrOrStderr(), "stream error: %v\n", err) },
			)
		},
	}
	marketCmd.Flags().StringVar(&assetsRaw, "asset-ids", "", "comma-separated CLOB token IDs")
	marketCmd.Flags().StringVar(&url, "url", marketStreamBaseURL, "WebSocket URL")
	marketCmd.Flags().IntVar(&maxMessages, "max-messages", 0, "stop after this many messages; 0 streams until interrupted")
	marketCmd.Flags().BoolVar(&customFeatures, "custom-features", false, "request best-bid-ask and market lifecycle events")
	marketCmd.Flags().IntVar(&level, "level", 0, "optional Polymarket market-stream subscription level")
	marketCmd.Flags().BoolVar(&stats, "stats", false, "emit stream lifecycle and message counters when the stream exits")
	cmd.AddCommand(marketCmd)

	var cryptoStreamAsset string
	var cryptoStreamInterval string
	var cryptoStreamMaxMsgs int
	cryptoCmd := &cobra.Command{
		Use:   "crypto",
		Short: "Stream live crypto market events",
		Long: `Discover active crypto markets and stream their WebSocket events in real-time.

Auto-discovers crypto markets by asset and interval, extracts token IDs, and
subscribes to the CLOB market stream for live order book and price updates.

Examples:
  polygolem stream crypto --asset BTC --interval 5m          # Stream BTC 5m markets
  polygolem stream crypto --asset ETH --max-messages 100     # Stream ETH markets
  polygolem stream crypto --asset SOL --custom-features      # With best-bid-ask events`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return streamcrypto.New(w.gamma, streamcrypto.NewInternalStreamer).Run(
				cmd.Context(),
				streamcrypto.Request{
					Asset:          cryptoStreamAsset,
					Interval:       cryptoStreamInterval,
					URL:            url,
					MaxMessages:    cryptoStreamMaxMsgs,
					CustomFeatures: customFeatures,
					Stats:          stats,
				},
				func(v interface{}) { _ = w.printJSON(cmd, v) },
				func(err error) { _, _ = fmt.Fprintf(cmd.ErrOrStderr(), "stream error: %v\n", err) },
			)
		},
	}
	cryptoCmd.Flags().StringVar(&cryptoStreamAsset, "asset", "", "crypto asset filter (BTC, ETH, SOL, XRP, DOGE, BNB, HYPE)")
	cryptoCmd.Flags().StringVar(&cryptoStreamInterval, "interval", "", "interval filter (5m, 15m, 1h)")
	cryptoCmd.Flags().IntVar(&cryptoStreamMaxMsgs, "max-messages", 0, "stop after this many messages; 0 streams until interrupted")
	cryptoCmd.Flags().BoolVar(&customFeatures, "custom-features", false, "request best-bid-ask and market lifecycle events")
	cryptoCmd.Flags().BoolVar(&stats, "stats", false, "emit stream lifecycle and message counters when the stream exits")
	cmd.AddCommand(cryptoCmd)

	var userMarketsRaw string
	var userURL string
	var userMaxMessages int
	var userStats bool
	userCmd := &cobra.Command{
		Use:   "user",
		Short: "Stream authenticated CLOB user order/trade events",
		Long: `Stream authenticated user-channel events from Polymarket CLOB.

Requires CLOB L2 credentials from POLYMARKET_CLOB_API_KEY,
POLYMARKET_CLOB_SECRET, and POLYMARKET_CLOB_PASSPHRASE (short CLOB_*
aliases are also accepted). Emits typed order and trade events as JSON.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			credentials, ok := clobL2CredentialsFromEnv()
			if !ok {
				return fmt.Errorf("configured CLOB L2 credentials are required: set POLYMARKET_CLOB_API_KEY, POLYMARKET_CLOB_SECRET, and POLYMARKET_CLOB_PASSPHRASE")
			}
			return streamuser.New(streamuser.NewInternalStreamer).Run(
				cmd.Context(),
				streamuser.Request{
					MarketsRaw:  userMarketsRaw,
					URL:         userURL,
					MaxMessages: userMaxMessages,
					Stats:       userStats,
					Credentials: credentials,
				},
				func(v interface{}) { _ = w.printJSON(cmd, v) },
				func(err error) { _, _ = fmt.Fprintf(cmd.ErrOrStderr(), "stream error: %v\n", err) },
			)
		},
	}
	userCmd.Flags().StringVar(&userMarketsRaw, "markets", "", "optional comma-separated market condition IDs")
	userCmd.Flags().StringVar(&userURL, "url", userStreamBaseURL, "WebSocket URL")
	userCmd.Flags().IntVar(&userMaxMessages, "max-messages", 0, "stop after this many messages; 0 streams until interrupted")
	userCmd.Flags().BoolVar(&userStats, "stats", false, "emit stream lifecycle and message counters when the stream exits")
	cmd.AddCommand(userCmd)

	return cmd
}
