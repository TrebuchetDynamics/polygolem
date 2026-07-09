package cli

import (
	"context"

	"github.com/TrebuchetDynamics/polygolem/internal/workflows/crypto5m"
	"github.com/TrebuchetDynamics/polygolem/internal/workflows/cryptowindow"
	"github.com/TrebuchetDynamics/polygolem/internal/workflows/discovercrypto"
	"github.com/TrebuchetDynamics/polygolem/internal/workflows/discoverreads"
	"github.com/TrebuchetDynamics/polygolem/internal/workflows/opportunities"
	"github.com/spf13/cobra"
)

type discoverReadRunner interface {
	Run(context.Context, discoverreads.Request) (any, error)
}

func discoverCmd(jsonOut bool) *cobra.Command {
	w := newWire(jsonOut)
	return newDiscoverCommand(w, discoverreads.New(discoverreads.Config{Gamma: w.gamma, Enricher: w.discover}))
}

func newDiscoverCommand(w *wire, reads discoverReadRunner) *cobra.Command {
	var query, marketID, marketSlug string
	var limit int

	cmd := commandGroup("markets", "Market discovery via Polymarket Gamma API")
	cmd.Long = `Find Polymarket markets and events. Read-only; no credentials required.

Search and list markets, look one up by id/slug/token, enrich with live CLOB
quotes, browse tags/series/comments, and resolve crypto up/down windows
(crypto-5m, crypto-window). This is the usual starting point: use it to find the
token id you then pass to book, exchange, or sim.`
	cmd.Example = `  polygolem markets search --query "Will BTC" --limit 5
  polygolem markets crypto-5m --asset BTC --hours-ahead 1 --enrich
  polygolem markets market --slug <market-slug>`

	var marketsLimit, marketsOffset, marketsTagID int
	var marketsOrder string
	var marketsActive, marketsClosed, marketsAscending bool
	marketsCmd := &cobra.Command{Use: "markets", Short: "List Gamma markets", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := reads.Run(cmd.Context(), discoverreads.Request{
				Operation: discoverreads.Markets,
				Limit:     marketsLimit,
				Offset:    marketsOffset,
				Order:     marketsOrder,
				Active:    marketsActive,
				Closed:    marketsClosed,
				Ascending: marketsAscending,
				TagID:     marketsTagID,
			})
			if err != nil {
				return err
			}
			return writeCommandJSON(cmd, result)
		},
	}
	marketsCmd.Flags().IntVar(&marketsLimit, "limit", 20, "max markets")
	marketsCmd.Flags().IntVar(&marketsOffset, "offset", 0, "pagination offset")
	marketsCmd.Flags().StringVar(&marketsOrder, "order", "", "Gamma order field")
	marketsCmd.Flags().BoolVar(&marketsActive, "active", true, "filter active markets")
	marketsCmd.Flags().BoolVar(&marketsClosed, "closed", false, "filter closed markets")
	marketsCmd.Flags().BoolVar(&marketsAscending, "ascending", false, "sort ascending")
	marketsCmd.Flags().IntVar(&marketsTagID, "tag-id", 0, "filter by tag id")
	cmd.AddCommand(marketsCmd)

	searchCmd := &cobra.Command{
		Use: "search", Short: "Search markets and events", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := reads.Run(cmd.Context(), discoverreads.Request{Operation: discoverreads.Search, Query: query, Limit: limit})
			if err != nil {
				return err
			}
			return writeCommandJSON(cmd, result)
		},
	}
	searchCmd.Flags().StringVar(&query, "query", "", "text query")
	searchCmd.Flags().IntVar(&limit, "limit", 10, "max results")
	cmd.AddCommand(searchCmd)

	marketCmd := &cobra.Command{
		Use: "market", Short: "Get market details", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := reads.Run(cmd.Context(), discoverreads.Request{Operation: discoverreads.Market, ID: marketID, Slug: marketSlug})
			if err != nil {
				return err
			}
			return writeCommandJSON(cmd, result)
		},
	}
	marketCmd.Flags().StringVar(&marketID, "id", "", "market Gamma ID")
	marketCmd.Flags().StringVar(&marketSlug, "slug", "", "market slug")
	cmd.AddCommand(marketCmd)

	enrichCmd := &cobra.Command{
		Use: "enrich", Short: "Enrich market with CLOB data", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := reads.Run(cmd.Context(), discoverreads.Request{Operation: discoverreads.Enrich, ID: marketID})
			if err != nil {
				return err
			}
			return writeCommandJSON(cmd, result)
		},
	}
	enrichCmd.Flags().StringVar(&marketID, "id", "", "market Gamma ID")
	cmd.AddCommand(enrichCmd)

	var tagsLimit, tagsOffset int
	var tagID, tagSlug string
	tagsCmd := &cobra.Command{Use: "tags", Short: "List or fetch Gamma tags/categories", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := reads.Run(cmd.Context(), discoverreads.Request{
				Operation: discoverreads.Tags,
				ID:        tagID,
				Slug:      tagSlug,
				Limit:     tagsLimit,
				Offset:    tagsOffset,
			})
			if err != nil {
				return err
			}
			return writeCommandJSON(cmd, result)
		},
	}
	tagsCmd.Flags().StringVar(&tagID, "id", "", "tag ID")
	tagsCmd.Flags().StringVar(&tagSlug, "slug", "", "tag slug")
	tagsCmd.Flags().IntVar(&tagsLimit, "limit", 100, "max tags")
	tagsCmd.Flags().IntVar(&tagsOffset, "offset", 0, "pagination offset")
	cmd.AddCommand(tagsCmd)

	var categorySlug, categoryCursor, categoryOrder string
	var categoryLimit int
	var categoryEvents, categoryClosed, categoryAscending bool
	categoriesCmd := &cobra.Command{Use: "categories", Short: "List curated polymarket.com categories or fetch a category feed", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			operation := discoverreads.Categories
			if categoryEvents {
				operation = discoverreads.CategoryEvents
			}
			result, err := reads.Run(cmd.Context(), discoverreads.Request{
				Operation: operation,
				Slug:      categorySlug,
				Limit:     categoryLimit,
				Cursor:    categoryCursor,
				Order:     categoryOrder,
				Closed:    categoryClosed,
				Ascending: categoryAscending,
			})
			if err != nil {
				return err
			}
			return writeCommandJSON(cmd, result)
		},
	}
	categoriesCmd.Flags().StringVar(&categorySlug, "slug", "", "curated category slug (for example politics, world-cup, mentions)")
	categoriesCmd.Flags().BoolVar(&categoryEvents, "events", false, "fetch Gamma events/keyset feed for --slug")
	categoriesCmd.Flags().IntVar(&categoryLimit, "limit", 20, "max category events")
	categoriesCmd.Flags().StringVar(&categoryCursor, "cursor", "", "Gamma next_cursor for category events")
	categoriesCmd.Flags().StringVar(&categoryOrder, "order", "volume24hr", "Gamma category event order")
	categoriesCmd.Flags().BoolVar(&categoryClosed, "closed", false, "include closed category events")
	categoriesCmd.Flags().BoolVar(&categoryAscending, "ascending", false, "sort category events ascending")
	cmd.AddCommand(categoriesCmd)

	var seriesLimit, seriesOffset int
	var seriesID string
	var seriesClosed bool
	seriesCmd := &cobra.Command{Use: "series", Short: "List or fetch Gamma series", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := reads.Run(cmd.Context(), discoverreads.Request{
				Operation: discoverreads.Series,
				ID:        seriesID,
				Limit:     seriesLimit,
				Offset:    seriesOffset,
				Closed:    seriesClosed,
			})
			if err != nil {
				return err
			}
			return writeCommandJSON(cmd, result)
		},
	}
	seriesCmd.Flags().StringVar(&seriesID, "id", "", "series ID")
	seriesCmd.Flags().IntVar(&seriesLimit, "limit", 20, "max series")
	seriesCmd.Flags().IntVar(&seriesOffset, "offset", 0, "pagination offset")
	seriesCmd.Flags().BoolVar(&seriesClosed, "closed", false, "filter closed series")
	cmd.AddCommand(seriesCmd)

	var commentID, commentEntityType, commentUser string
	var commentEntityID, commentLimit, commentOffset int
	commentsCmd := &cobra.Command{Use: "comments", Short: "List or fetch public Gamma comments", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := reads.Run(cmd.Context(), discoverreads.Request{
				Operation:  discoverreads.Comments,
				ID:         commentID,
				User:       commentUser,
				EntityID:   commentEntityID,
				EntityType: commentEntityType,
				Limit:      commentLimit,
				Offset:     commentOffset,
			})
			if err != nil {
				return err
			}
			return writeCommandJSON(cmd, result)
		},
	}
	commentsCmd.Flags().StringVar(&commentID, "id", "", "comment ID")
	commentsCmd.Flags().StringVar(&commentUser, "user", "", "user wallet address")
	commentsCmd.Flags().IntVar(&commentEntityID, "entity-id", 0, "comment parent entity ID")
	commentsCmd.Flags().StringVar(&commentEntityType, "entity-type", "", "comment parent entity type")
	commentsCmd.Flags().IntVar(&commentLimit, "limit", 20, "max comments")
	commentsCmd.Flags().IntVar(&commentOffset, "offset", 0, "pagination offset")
	cmd.AddCommand(commentsCmd)

	var cryptoInterval string
	var cryptoAsset string
	var cryptoEnrich bool
	cryptoCmd := &cobra.Command{
		Use:   "crypto",
		Short: "Discover active crypto prediction markets",
		Long: `Search for active Polymarket crypto markets by asset and interval.

Extracts markets from events and filters by title patterns. Returns token IDs
ready for orderbook inspection or trading.

Examples:
  polygolem markets crypto --asset BTC --interval 5m    # BTC Up/Down 5m markets
  polygolem markets crypto --asset ETH --interval 15m   # ETH Up/Down 15m markets
  polygolem markets crypto --asset BTC --interval 5m --enrich  # With CLOB prices
  polygolem markets crypto --limit 50                   # All crypto markets

Assets: BTC, ETH, SOL, XRP, DOGE, BNB, HYPE, etc.
Intervals: 5m, 15m, 1h, daily, weekly (matches title patterns)`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := discovercrypto.New(w.gamma, w.clob).Run(cmd.Context(), discovercrypto.Request{
				Asset:    cryptoAsset,
				Interval: cryptoInterval,
				Limit:    limit,
				Enrich:   cryptoEnrich,
			})
			if err != nil {
				return err
			}
			return writeCommandJSON(cmd, result)
		},
	}
	cryptoCmd.Flags().StringVar(&cryptoAsset, "asset", "", "crypto asset filter (BTC, ETH, SOL, XRP, DOGE, BNB, HYPE)")
	cryptoCmd.Flags().StringVar(&cryptoInterval, "interval", "", "interval filter (5m, 15m, 1h, daily, weekly)")
	cryptoCmd.Flags().IntVar(&limit, "limit", 20, "max results")
	cryptoCmd.Flags().BoolVar(&cryptoEnrich, "enrich", false, "enrich with CLOB price and spread (slower, one API call per market)")
	cmd.AddCommand(cryptoCmd)

	var windowAsset, windowInterval string
	windowCmd := &cobra.Command{
		Use:   "crypto-window",
		Short: "Resolve the current crypto prediction window deterministically",
		Long: `Resolve the current active crypto up/down market using the deterministic
slug pattern (<asset>-updown-<interval>-<unix_timestamp>).

This bypasses search and hits the exact current window directly — much faster
and more reliable than discovery via public search.

Examples:
  polygolem markets crypto-window --asset BTC --interval 5m
  polygolem markets crypto-window --asset ETH --interval 15m --enrich`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := cryptowindow.New(w.gamma, w.clob).Run(cmd.Context(), cryptowindow.Request{
				Asset:    windowAsset,
				Interval: windowInterval,
				Enrich:   cryptoEnrich,
			})
			if err != nil {
				return err
			}
			return writeCommandJSON(cmd, result)
		},
	}
	windowCmd.Flags().StringVar(&windowAsset, "asset", "", "crypto asset (BTC, ETH, SOL, XRP, BNB, DOGE, HYPE)")
	windowCmd.Flags().StringVar(&windowInterval, "interval", "", "time interval (5m, 15m, 1h, 4h)")
	windowCmd.Flags().BoolVar(&cryptoEnrich, "enrich", false, "enrich with CLOB price and spread")
	cmd.AddCommand(windowCmd)

	var fiveMinEnrich bool
	var fiveMinHoursAhead int
	var fiveMinTimezone string
	var fiveMinAssets []string
	fiveMinCmd := &cobra.Command{
		Use:   "crypto-5m",
		Short: "List active 5-minute crypto markets",
		Long: `Resolve current and near-future 5-minute windows for supported crypto assets
and return a consolidated view of every open accepting market.

Assets scanned by default: BTC, ETH, SOL, XRP, BNB, DOGE, HYPE

Use --hours-ahead 1 for the current window plus the next hour.
Use --timezone America/Denver to add local window fields.
Use --asset BTC --asset ETH to narrow the sweep.
Use --enrich to fetch live CLOB prices and spreads (slower).`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := crypto5m.New(w.gamma, w.clob).Run(cmd.Context(), crypto5m.Request{Assets: fiveMinAssets, Enrich: fiveMinEnrich, HoursAhead: fiveMinHoursAhead, Timezone: fiveMinTimezone})
			if err != nil {
				return err
			}
			return writeCommandJSON(cmd, result)
		},
	}
	fiveMinCmd.Flags().StringSliceVar(&fiveMinAssets, "asset", nil, "crypto asset(s) to scan (repeat or comma-separate: BTC,ETH,SOL)")
	fiveMinCmd.Flags().BoolVar(&fiveMinEnrich, "enrich", false, "enrich with CLOB price and spread")
	fiveMinCmd.Flags().IntVar(&fiveMinHoursAhead, "hours-ahead", 0, "include future 5m windows this many hours ahead")
	fiveMinCmd.Flags().StringVar(&fiveMinTimezone, "timezone", "UTC", "display local window fields in this IANA timezone (example: America/Chicago)")
	cmd.AddCommand(fiveMinCmd)

	var opportunityType, opportunityAsset string
	var opportunityLimit, opportunityHours int
	opportunitiesCmd := &cobra.Command{
		Use:   "opportunities",
		Short: "Scan read-only market opportunity candidates",
		Long: `Scan public Polymarket data for read-only research candidates.

Scanner types:
  wide-spread
  low-liquidity-high-volume
  new-markets
  closing-soon
  negative-risk
  crypto-5m

Examples:
  polygolem markets opportunities --type wide-spread --limit 20
  polygolem markets opportunities --type closing-soon --hours 6
  polygolem markets opportunities --type low-liquidity-high-volume
  polygolem markets opportunities --type crypto-5m --asset BTC`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			runner := opportunities.New(opportunities.Config{Gamma: w.gamma, Pricer: w.clob})
			result, err := runner.Run(cmd.Context(), opportunities.Request{
				Type:  opportunities.Type(opportunityType),
				Limit: opportunityLimit,
				Hours: opportunityHours,
				Asset: opportunityAsset,
			})
			if err != nil {
				return err
			}
			return writeCommandJSON(cmd, result)
		},
	}
	opportunitiesCmd.Flags().StringVar(&opportunityType, "type", string(opportunities.TypeWideSpread), "scanner type: wide-spread, low-liquidity-high-volume, new-markets, closing-soon, negative-risk, crypto-5m")
	opportunitiesCmd.Flags().IntVar(&opportunityLimit, "limit", 20, "max opportunities")
	opportunitiesCmd.Flags().IntVar(&opportunityHours, "hours", 24, "closing-soon lookahead window in hours")
	opportunitiesCmd.Flags().StringVar(&opportunityAsset, "asset", "", "crypto asset for crypto-5m scanner (BTC, ETH, SOL, XRP, BNB, DOGE, HYPE)")
	cmd.AddCommand(opportunitiesCmd)

	return cmd
}
