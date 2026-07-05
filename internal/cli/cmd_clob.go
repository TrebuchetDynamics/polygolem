package cli

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/TrebuchetDynamics/polygolem/internal/clob"
	"github.com/TrebuchetDynamics/polygolem/internal/polytypes"
	"github.com/TrebuchetDynamics/polygolem/internal/workflows/clobaccountreads"
	"github.com/TrebuchetDynamics/polygolem/internal/workflows/clobbalances"
	"github.com/TrebuchetDynamics/polygolem/internal/workflows/clobdiagnostics"
	"github.com/TrebuchetDynamics/polygolem/internal/workflows/clobmarketdata"
	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cobra"
)

type clobMarketDataCommandRunner interface {
	Book(context.Context, clobmarketdata.TokenRequest) (*polytypes.OrderBook, error)
	TickSize(context.Context, clobmarketdata.TokenRequest) (*polytypes.TickSize, error)
	PriceHistory(context.Context, clobmarketdata.PriceHistoryRequest) (*polytypes.PriceHistory, error)
	Market(context.Context, clobmarketdata.ConditionRequest) (*polytypes.CLOBMarket, error)
	MarketByToken(context.Context, clobmarketdata.TokenRequest) (*polytypes.CLOBMarketByTokenResponse, error)
	Markets(context.Context, clobmarketdata.MarketsRequest) (*polytypes.CLOBPaginatedMarkets, error)
}

type clobBalanceCommandRunner interface {
	Balance(context.Context, clobbalances.Request) (map[string]interface{}, error)
	UpdateBalance(context.Context, clobbalances.Request) (map[string]interface{}, error)
}

type clobAccountReadCommandRunner interface {
	Orders(context.Context, clobaccountreads.Request) ([]clob.OrderRecord, error)
	Order(context.Context, clobaccountreads.OrderRequest) (*clob.OrderRecord, error)
	Trades(context.Context, clobaccountreads.Request) ([]clob.TradeRecord, error)
}

type clobDiagnosticCommandRunner interface {
	ListBuilderFeeKeys(context.Context, clobdiagnostics.Request) ([]clob.BuilderFeeKeyRecord, error)
	MarketTradesProbe(context.Context, clobdiagnostics.ProbeRequest) (*clob.MarketTradesProbeResult, error)
}

const defaultMaxLiveOrderUSD = "1"

func addCLOBOutputFlag(c *cobra.Command, output *string) {
	c.Flags().StringVar(output, "output", "json", "output format (json)")
}

func checkCLOBOutput(output string) error {
	if output != "" && output != "json" {
		return fmt.Errorf("only --output json is supported")
	}
	return nil
}

func addCLOBMarketDataCommands(cmd *cobra.Command, marketData clobMarketDataCommandRunner) {
	var bookOutput string
	bookCmd := &cobra.Command{Use: "book <token-id>", Short: "Get L2 order book", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			book, err := marketData.Book(cmd.Context(), clobmarketdata.TokenRequest{TokenID: args[0], Output: bookOutput})
			if err != nil {
				return err
			}
			return writeCommandJSON(cmd, book)
		},
	}
	addCLOBOutputFlag(bookCmd, &bookOutput)
	cmd.AddCommand(bookCmd)

	var tickOutput string
	tickCmd := &cobra.Command{Use: "tick-size <token-id>", Short: "Get minimum tick size", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tick, err := marketData.TickSize(cmd.Context(), clobmarketdata.TokenRequest{TokenID: args[0], Output: tickOutput})
			if err != nil {
				return err
			}
			return writeCommandJSON(cmd, tick)
		},
	}
	addCLOBOutputFlag(tickCmd, &tickOutput)
	cmd.AddCommand(tickCmd)

	var priceHistoryOutput, priceHistoryInterval string
	priceHistoryCmd := &cobra.Command{Use: "price-history <token-id>", Short: "Get CLOB token price history", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			history, err := marketData.PriceHistory(cmd.Context(), clobmarketdata.PriceHistoryRequest{
				TokenID:  args[0],
				Interval: priceHistoryInterval,
				Output:   priceHistoryOutput,
			})
			if err != nil {
				return err
			}
			return writeCommandJSON(cmd, history)
		},
	}
	addCLOBOutputFlag(priceHistoryCmd, &priceHistoryOutput)
	priceHistoryCmd.Flags().StringVar(&priceHistoryInterval, "interval", "1m", "history interval")
	cmd.AddCommand(priceHistoryCmd)

	var marketOutput string
	marketCmd := &cobra.Command{Use: "market <condition-id>", Short: "Get CLOB market by condition ID", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			market, err := marketData.Market(cmd.Context(), clobmarketdata.ConditionRequest{ConditionID: args[0], Output: marketOutput})
			if err != nil {
				return err
			}
			return writeCommandJSON(cmd, market)
		},
	}
	addCLOBOutputFlag(marketCmd, &marketOutput)
	cmd.AddCommand(marketCmd)

	var marketByTokenOutput string
	marketByTokenCmd := &cobra.Command{Use: "market-by-token <token-id>", Short: "Resolve CLOB market by token ID", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			market, err := marketData.MarketByToken(cmd.Context(), clobmarketdata.TokenRequest{TokenID: args[0], Output: marketByTokenOutput})
			if err != nil {
				return err
			}
			return writeCommandJSON(cmd, market)
		},
	}
	addCLOBOutputFlag(marketByTokenCmd, &marketByTokenOutput)
	cmd.AddCommand(marketByTokenCmd)

	var marketsOutput, marketsCursor string
	marketsCmd := &cobra.Command{Use: "markets", Short: "List CLOB markets", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			markets, err := marketData.Markets(cmd.Context(), clobmarketdata.MarketsRequest{Cursor: marketsCursor, Output: marketsOutput})
			if err != nil {
				return err
			}
			return writeCommandJSON(cmd, markets)
		},
	}
	addCLOBOutputFlag(marketsCmd, &marketsOutput)
	marketsCmd.Flags().StringVar(&marketsCursor, "cursor", "", "pagination cursor")
	cmd.AddCommand(marketsCmd)
}

func addCLOBAuthenticatedReadCommands(cmd *cobra.Command, balances clobBalanceCommandRunner, accountReads clobAccountReadCommandRunner, diagnostics clobDiagnosticCommandRunner) {
	var listBuilderFeeKeysOutput string
	listBuilderFeeKeysCmd := &cobra.Command{
		Use:   "list-builder-fee-keys",
		Short: "List builder fee keys (GET /auth/builder-api-keys)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			records, err := diagnostics.ListBuilderFeeKeys(cmd.Context(), clobdiagnostics.Request{Output: listBuilderFeeKeysOutput})
			if err != nil {
				return err
			}
			return writeCommandJSON(cmd, records)
		},
	}
	addCLOBOutputFlag(listBuilderFeeKeysCmd, &listBuilderFeeKeysOutput)
	cmd.AddCommand(listBuilderFeeKeysCmd)

	var balanceOutput, balanceAssetType, balanceTokenID string
	balanceCmd := &cobra.Command{Use: "balance", Short: "Get CLOB balance and allowances", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := balances.Balance(cmd.Context(), clobbalances.Request{AssetType: balanceAssetType, TokenID: balanceTokenID, Output: balanceOutput})
			if err != nil {
				return err
			}
			return writeCommandJSON(cmd, res)
		},
	}
	addCLOBOutputFlag(balanceCmd, &balanceOutput)
	balanceCmd.Flags().StringVar(&balanceAssetType, "asset-type", "collateral", "asset type")
	balanceCmd.Flags().StringVar(&balanceTokenID, "token-id", "", "conditional token id")
	cmd.AddCommand(balanceCmd)

	var updateBalanceOutput, updateBalanceAssetType, updateBalanceTokenID string
	updateBalanceCmd := &cobra.Command{Use: "update-balance", Short: "Refresh CLOB balance and allowances", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := balances.UpdateBalance(cmd.Context(), clobbalances.Request{AssetType: updateBalanceAssetType, TokenID: updateBalanceTokenID, Output: updateBalanceOutput})
			if err != nil {
				return err
			}
			return writeCommandJSON(cmd, res)
		},
	}
	addCLOBOutputFlag(updateBalanceCmd, &updateBalanceOutput)
	updateBalanceCmd.Flags().StringVar(&updateBalanceAssetType, "asset-type", "collateral", "asset type")
	updateBalanceCmd.Flags().StringVar(&updateBalanceTokenID, "token-id", "", "conditional token id")
	cmd.AddCommand(updateBalanceCmd)

	var ordersOutput string
	ordersCmd := &cobra.Command{Use: "orders", Short: "List authenticated CLOB orders", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			rows, err := accountReads.Orders(cmd.Context(), clobaccountreads.Request{Output: ordersOutput})
			if err != nil {
				return err
			}
			return writeCommandJSON(cmd, rows)
		},
	}
	addCLOBOutputFlag(ordersCmd, &ordersOutput)
	cmd.AddCommand(ordersCmd)

	var orderOutput string
	orderCmd := &cobra.Command{Use: "order <order-id>", Short: "Get a single authenticated CLOB order", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			row, err := accountReads.Order(cmd.Context(), clobaccountreads.OrderRequest{OrderID: args[0], Output: orderOutput})
			if err != nil {
				return err
			}
			return writeCommandJSON(cmd, row)
		},
	}
	addCLOBOutputFlag(orderCmd, &orderOutput)
	cmd.AddCommand(orderCmd)

	var tradesOutput string
	tradesCmd := &cobra.Command{Use: "trades", Short: "List authenticated CLOB trades", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			rows, err := accountReads.Trades(cmd.Context(), clobaccountreads.Request{Output: tradesOutput})
			if err != nil {
				return err
			}
			return writeCommandJSON(cmd, rows)
		},
	}
	addCLOBOutputFlag(tradesCmd, &tradesOutput)
	cmd.AddCommand(tradesCmd)

	var probeOutput, probeMarket, probeAssetID, probeCursor string
	probeCmd := &cobra.Command{
		Use:   "market-trades-probe",
		Short: "Probe CLOB trade scope for one market or token",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := diagnostics.MarketTradesProbe(cmd.Context(), clobdiagnostics.ProbeRequest{
				Market:     probeMarket,
				AssetID:    probeAssetID,
				NextCursor: probeCursor,
				Output:     probeOutput,
			})
			if err != nil {
				return err
			}
			return writeCommandJSON(cmd, res)
		},
	}
	addCLOBOutputFlag(probeCmd, &probeOutput)
	probeCmd.Flags().StringVar(&probeMarket, "market", "", "market condition ID")
	probeCmd.Flags().StringVar(&probeAssetID, "asset-id", "", "CLOB token ID")
	probeCmd.Flags().StringVar(&probeCursor, "cursor", "", "optional next_cursor for diagnostics")
	cmd.AddCommand(probeCmd)
}

func clobCmd(jsonOut bool) *cobra.Command {
	w := newWire(jsonOut)
	cmd := commandGroup("clob", "CLOB market data and authenticated account commands")

	addOutput := addCLOBOutputFlag
	checkOutput := checkCLOBOutput
	privateKey := func() (string, error) {
		return privateKeyFromEnv()
	}
	marketData := clobmarketdata.New(w.clob)
	accountReads := clobaccountreads.New(clobaccountreads.Config{Reader: w.clob, PrivateKey: privateKey})
	balances := clobbalances.New(clobbalances.Config{Reader: w.clob, PrivateKey: privateKey})
	diagnostics := clobdiagnostics.New(clobdiagnostics.Config{Reader: w.clob, PrivateKey: privateKey})
	addCLOBMarketDataCommands(cmd, marketData)
	addCLOBAuthenticatedReadCommands(cmd, balances, accountReads, diagnostics)

	var createKeyOutput string
	createKeyCmd := &cobra.Command{Use: "create-api-key", Short: "Create or derive CLOB API credentials", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkOutput(createKeyOutput); err != nil {
				return err
			}
			key, err := privateKey()
			if err != nil {
				return err
			}
			apiKey, err := w.clob.CreateOrDeriveAPIKey(cmd.Context(), key)
			if err != nil {
				return err
			}
			return w.printJSON(cmd, map[string]string{"api_key": apiKey.Key})
		},
	}
	addOutput(createKeyCmd, &createKeyOutput)
	cmd.AddCommand(createKeyCmd)

	var createKeyForAddressOutput, createKeyForAddressOwner string
	createKeyForAddressCmd := &cobra.Command{Use: "create-api-key-for-address", Short: "Create CLOB API credentials while reporting a maker address", Args: cobra.NoArgs,
		Long: `Creates CLOB L2 credentials using EOA L1 auth and echoes the
configured maker address. Polymarket login and CLOB HTTP authentication sign
with the EOA; the deposit wallet remains the POLY_1271 trading wallet inside
orders, balances, approvals, and settlement.

The --owner flag is retained for source compatibility with older automation
and is returned in the JSON output. It is not used as POLY_ADDRESS for the
CLOB L1 auth headers.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkOutput(createKeyForAddressOutput); err != nil {
				return err
			}
			owner := strings.TrimSpace(createKeyForAddressOwner)
			if owner == "" {
				return fmt.Errorf("--owner is required")
			}
			if !common.IsHexAddress(owner) {
				return fmt.Errorf("--owner must be an Ethereum address")
			}
			key, err := privateKey()
			if err != nil {
				return err
			}
			apiKey, err := w.clob.CreateAPIKeyForAddress(cmd.Context(), key, owner)
			if err != nil {
				return fmt.Errorf("%w\n\nNote: Polymarket login and CLOB HTTP auth sign with the EOA; the deposit wallet remains the POLY_1271 trading wallet. Run `polygolem auth login` first if this EOA has not been profiled. See docs/ONBOARDING.md", err)
			}
			return w.printJSON(cmd, map[string]string{"api_key": apiKey.Key, "owner": common.HexToAddress(owner).Hex()})
		},
	}
	addOutput(createKeyForAddressCmd, &createKeyForAddressOutput)
	createKeyForAddressCmd.Flags().StringVar(&createKeyForAddressOwner, "owner", "", "deposit wallet owner address")
	cmd.AddCommand(createKeyForAddressCmd)

	var createBuilderFeeKeyOutput string
	createBuilderFeeKeyCmd := &cobra.Command{
		Use:   "create-builder-fee-key",
		Short: "Mint a CLOB builder fee key (POST /auth/builder-api-key)",
		Long: `Mints a builder fee key by signing an L2 HMAC-authenticated
POST to /auth/builder-api-key. The returned triple is the fee
attribution key — attach its 'key' to the 'builder' bytes32 field of V2
orders to claim integrator fees.

This is a different credential from the L2 trading triple minted by
'create-api-key'; both are needed for full V2 integrator setup. See
docs/HEADLESS-BUILDER-KEYS-INVESTIGATION.md.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkOutput(createBuilderFeeKeyOutput); err != nil {
				return err
			}
			key, err := privateKey()
			if err != nil {
				return err
			}
			feeKey, err := w.clob.CreateBuilderFeeKey(cmd.Context(), key)
			if err != nil {
				return err
			}
			return w.printJSON(cmd, map[string]string{"builder_fee_key": feeKey.Key})
		},
	}
	addOutput(createBuilderFeeKeyCmd, &createBuilderFeeKeyOutput)
	cmd.AddCommand(createBuilderFeeKeyCmd)

	var revokeBuilderFeeKeyOutput, revokeBuilderFeeKey string
	revokeBuilderFeeKeyCmd := &cobra.Command{
		Use:   "revoke-builder-fee-key",
		Short: "Revoke a builder fee key (DELETE /auth/builder-api-key/{key})",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkOutput(revokeBuilderFeeKeyOutput); err != nil {
				return err
			}
			if strings.TrimSpace(revokeBuilderFeeKey) == "" {
				return fmt.Errorf("--key is required")
			}
			key, err := privateKey()
			if err != nil {
				return err
			}
			if err := w.clob.RevokeBuilderFeeKey(cmd.Context(), key, revokeBuilderFeeKey); err != nil {
				return err
			}
			return w.printJSON(cmd, map[string]string{"revoked": revokeBuilderFeeKey})
		},
	}
	addOutput(revokeBuilderFeeKeyCmd, &revokeBuilderFeeKeyOutput)
	revokeBuilderFeeKeyCmd.Flags().StringVar(&revokeBuilderFeeKey, "key", "", "builder fee key to revoke")
	cmd.AddCommand(revokeBuilderFeeKeyCmd)

	var cancelOutput string
	cancelCmd := &cobra.Command{Use: "cancel <order-id>", Short: "Cancel a single open CLOB order", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkOutput(cancelOutput); err != nil {
				return err
			}
			key, err := privateKey()
			if err != nil {
				return err
			}
			resp, err := w.clob.CancelOrder(cmd.Context(), key, args[0])
			if err != nil {
				return err
			}
			return w.printJSON(cmd, resp)
		},
	}
	addOutput(cancelCmd, &cancelOutput)
	cmd.AddCommand(cancelCmd)

	var cancelOrdersOutput string
	cancelOrdersCmd := &cobra.Command{Use: "cancel-orders <order-ids>", Short: "Cancel multiple open CLOB orders", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkOutput(cancelOrdersOutput); err != nil {
				return err
			}
			key, err := privateKey()
			if err != nil {
				return err
			}
			ids := splitCSV(args[0])
			resp, err := w.clob.CancelOrders(cmd.Context(), key, ids)
			if err != nil {
				return err
			}
			return w.printJSON(cmd, resp)
		},
	}
	addOutput(cancelOrdersCmd, &cancelOrdersOutput)
	cmd.AddCommand(cancelOrdersCmd)

	var cancelAllOutput string
	cancelAllCmd := &cobra.Command{Use: "cancel-all", Short: "Cancel all open CLOB orders", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkOutput(cancelAllOutput); err != nil {
				return err
			}
			key, err := privateKey()
			if err != nil {
				return err
			}
			resp, err := w.clob.CancelAll(cmd.Context(), key)
			if err != nil {
				return err
			}
			return w.printJSON(cmd, resp)
		},
	}
	addOutput(cancelAllCmd, &cancelAllOutput)
	cmd.AddCommand(cancelAllCmd)

	var cancelMarketOutput, cancelMarketID, cancelMarketAsset string
	cancelMarketCmd := &cobra.Command{Use: "cancel-market", Short: "Cancel open CLOB orders for a market or asset", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkOutput(cancelMarketOutput); err != nil {
				return err
			}
			key, err := privateKey()
			if err != nil {
				return err
			}
			resp, err := w.clob.CancelMarket(cmd.Context(), key, clob.CancelMarketParams{
				Market: cancelMarketID,
				Asset:  cancelMarketAsset,
			})
			if err != nil {
				return err
			}
			return w.printJSON(cmd, resp)
		},
	}
	addOutput(cancelMarketCmd, &cancelMarketOutput)
	cancelMarketCmd.Flags().StringVar(&cancelMarketID, "market", "", "market condition ID")
	cancelMarketCmd.Flags().StringVar(&cancelMarketAsset, "asset", "", "asset/token ID")
	cmd.AddCommand(cancelMarketCmd)

	var createOrderOutput, createOrderToken, createOrderSide, createOrderPrice, createOrderSize, createOrderType, createOrderExpiration, createOrderBuilderCode string
	var createOrderPostOnly bool
	createOrderCmd := &cobra.Command{Use: "create-order", Short: "Create a signed CLOB limit order", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkOutput(createOrderOutput); err != nil {
				return err
			}
			builderCode := builderCodeFromFlagOrEnv(createOrderBuilderCode)
			if err := validateBuilderCodeForCLI(builderCode); err != nil {
				return err
			}
			w.clob.SetBuilderCode(builderCode)
			if err := enforceLimitOrderCap(createOrderPrice, createOrderSize); err != nil {
				return err
			}
			key, err := privateKey()
			if err != nil {
				return err
			}
			warnIfNoDepositKey(cmd.Context(), cmd.ErrOrStderr(), key)

			res, err := w.clob.CreateLimitOrder(cmd.Context(), key, clob.CreateOrderParams{
				TokenID:    createOrderToken,
				Side:       createOrderSide,
				Price:      createOrderPrice,
				Size:       createOrderSize,
				OrderType:  createOrderType,
				Expiration: createOrderExpiration,
				PostOnly:   createOrderPostOnly,
			})
			if err != nil {
				return err
			}
			return w.printJSON(cmd, res)
		},
	}
	addOutput(createOrderCmd, &createOrderOutput)
	createOrderCmd.Flags().StringVar(&createOrderToken, "token", "", "CLOB token id")
	createOrderCmd.Flags().StringVar(&createOrderSide, "side", "buy", "order side")
	createOrderCmd.Flags().StringVar(&createOrderPrice, "price", "", "limit price")
	createOrderCmd.Flags().StringVar(&createOrderSize, "size", "", "order size")
	createOrderCmd.Flags().StringVar(&createOrderType, "order-type", "GTC", "order type")
	createOrderCmd.Flags().StringVar(&createOrderExpiration, "expiration", "0", "unix timestamp for GTD orders (0 = no expiration)")
	createOrderCmd.Flags().StringVar(&createOrderBuilderCode, "builder-code", "", "0x-prefixed bytes32 builder attribution code")
	createOrderCmd.Flags().BoolVar(&createOrderPostOnly, "post-only", false, "post-only order (maker-only, rejected if it would take)")
	cmd.AddCommand(createOrderCmd)

	var batchOrdersOutput, batchOrdersFile, batchOrdersBuilderCode string
	batchOrdersCmd := &cobra.Command{Use: "batch-orders", Short: "Create multiple signed CLOB limit orders", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkOutput(batchOrdersOutput); err != nil {
				return err
			}
			builderCode := builderCodeFromFlagOrEnv(batchOrdersBuilderCode)
			if err := validateBuilderCodeForCLI(builderCode); err != nil {
				return err
			}
			if strings.TrimSpace(batchOrdersFile) == "" {
				return fmt.Errorf("--orders-file is required")
			}
			reader, closeReader, err := openBatchOrdersInput(cmd, batchOrdersFile)
			if err != nil {
				return err
			}
			if closeReader != nil {
				defer closeReader()
			}
			orders, err := parseBatchOrderParams(reader)
			if err != nil {
				return err
			}
			w.clob.SetBuilderCode(builderCode)
			key, err := privateKey()
			if err != nil {
				return err
			}
			warnIfNoDepositKey(cmd.Context(), cmd.ErrOrStderr(), key)

			res, err := w.clob.CreateBatchOrders(cmd.Context(), key, orders)
			if err != nil {
				return err
			}
			return w.printJSON(cmd, res)
		},
	}
	addOutput(batchOrdersCmd, &batchOrdersOutput)
	batchOrdersCmd.Flags().StringVar(&batchOrdersFile, "orders-file", "", "JSON array of limit orders, or '-' for stdin")
	batchOrdersCmd.Flags().StringVar(&batchOrdersBuilderCode, "builder-code", "", "0x-prefixed bytes32 builder attribution code")
	cmd.AddCommand(batchOrdersCmd)

	var marketOrderOutput, marketOrderToken, marketOrderSide, marketOrderAmount, marketOrderPrice, marketOrderType, marketOrderBuilderCode string
	marketOrderCmd := &cobra.Command{Use: "market-order", Short: "Create a signed CLOB market/FOK order", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkOutput(marketOrderOutput); err != nil {
				return err
			}
			builderCode := builderCodeFromFlagOrEnv(marketOrderBuilderCode)
			if err := validateBuilderCodeForCLI(builderCode); err != nil {
				return err
			}
			w.clob.SetBuilderCode(builderCode)
			if err := enforceMarketOrderCap(marketOrderAmount); err != nil {
				return err
			}
			key, err := privateKey()
			if err != nil {
				return err
			}
			warnIfNoDepositKey(cmd.Context(), cmd.ErrOrStderr(), key)

			res, err := w.clob.CreateMarketOrder(cmd.Context(), key, clob.MarketOrderParams{
				TokenID:   marketOrderToken,
				Side:      marketOrderSide,
				Amount:    marketOrderAmount,
				Price:     marketOrderPrice,
				OrderType: marketOrderType,
			})
			if err != nil {
				return err
			}
			return w.printJSON(cmd, res)
		},
	}
	addOutput(marketOrderCmd, &marketOrderOutput)
	marketOrderCmd.Flags().StringVar(&marketOrderToken, "token", "", "CLOB token id")
	marketOrderCmd.Flags().StringVar(&marketOrderSide, "side", "buy", "order side")
	marketOrderCmd.Flags().StringVar(&marketOrderAmount, "amount", "", "USDC amount")
	marketOrderCmd.Flags().StringVar(&marketOrderPrice, "price", "", "limit price")
	marketOrderCmd.Flags().StringVar(&marketOrderType, "order-type", "FOK", "order type")
	marketOrderCmd.Flags().StringVar(&marketOrderBuilderCode, "builder-code", "", "0x-prefixed bytes32 builder attribution code")
	cmd.AddCommand(marketOrderCmd)

	var heartbeatOutput, heartbeatID string
	heartbeatCmd := &cobra.Command{Use: "heartbeat", Short: "Send one CLOB heartbeat ping", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkOutput(heartbeatOutput); err != nil {
				return err
			}
			key, err := privateKey()
			if err != nil {
				return err
			}
			if err := w.clob.Heartbeat(cmd.Context(), key, heartbeatID); err != nil {
				return err
			}
			return w.printJSON(cmd, map[string]interface{}{
				"ok":           true,
				"heartbeat_id": heartbeatID,
			})
		},
	}
	addOutput(heartbeatCmd, &heartbeatOutput)
	heartbeatCmd.Flags().StringVar(&heartbeatID, "id", "", "optional heartbeat id")
	cmd.AddCommand(heartbeatCmd)

	return cmd
}

func enforceLimitOrderCap(price, size string) error {
	p, err := decimalRat("--price", price)
	if err != nil {
		return err
	}
	s, err := decimalRat("--size", size)
	if err != nil {
		return err
	}
	return enforceLiveOrderCap(new(big.Rat).Mul(p, s))
}

func enforceMarketOrderCap(amount string) error {
	a, err := decimalRat("--amount", amount)
	if err != nil {
		return err
	}
	return enforceLiveOrderCap(a)
}

func enforceLiveOrderCap(notional *big.Rat) error {
	capValue := strings.TrimSpace(os.Getenv("POLYGOLEM_MAX_LIVE_ORDER_USD"))
	if capValue == "" {
		capValue = defaultMaxLiveOrderUSD
	}
	capRat, err := decimalRat("POLYGOLEM_MAX_LIVE_ORDER_USD", capValue)
	if err != nil {
		return err
	}
	if notional.Cmp(capRat) > 0 {
		return fmt.Errorf("live order notional %s exceeds POLYGOLEM_MAX_LIVE_ORDER_USD=%s", notional.FloatString(6), capRat.FloatString(6))
	}
	return nil
}

func decimalRat(name, value string) (*big.Rat, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, fmt.Errorf("%s is required", name)
	}
	if strings.Contains(value, "/") {
		return nil, fmt.Errorf("%s must be a decimal", name)
	}
	r, ok := new(big.Rat).SetString(value)
	if !ok || r.Sign() <= 0 {
		return nil, fmt.Errorf("%s must be a positive decimal", name)
	}
	return r, nil
}
