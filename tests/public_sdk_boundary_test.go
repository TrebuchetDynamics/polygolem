package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestPublicDataAPIDoesNotRequireInternalImports(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), ".."))
	tempDir := t.TempDir()

	writeFile(t, filepath.Join(tempDir, "go.mod"), `module example.com/polygolem-public-consumer

go 1.25.0

require github.com/TrebuchetDynamics/polygolem v0.0.0

replace github.com/TrebuchetDynamics/polygolem => `+repoRoot+`
`)
	writeFile(t, filepath.Join(tempDir, "public_sdk_test.go"), `package publicconsumer

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/polygolem/pkg/bridge"
	"github.com/TrebuchetDynamics/polygolem/pkg/builder"
	"github.com/TrebuchetDynamics/polygolem/pkg/capabilities"
	sdkclob "github.com/TrebuchetDynamics/polygolem/pkg/clob"
	"github.com/TrebuchetDynamics/polygolem/pkg/compat"
	"github.com/TrebuchetDynamics/polygolem/pkg/contracts"
	"github.com/TrebuchetDynamics/polygolem/pkg/cryptoprice"
	"github.com/TrebuchetDynamics/polygolem/pkg/ctf"
	"github.com/TrebuchetDynamics/polygolem/pkg/data"
	"github.com/TrebuchetDynamics/polygolem/pkg/enabletrading"
	"github.com/TrebuchetDynamics/polygolem/pkg/funding"
	"github.com/TrebuchetDynamics/polygolem/pkg/gamma"
	"github.com/TrebuchetDynamics/polygolem/pkg/geoblock"
	"github.com/TrebuchetDynamics/polygolem/pkg/intel"
	"github.com/TrebuchetDynamics/polygolem/pkg/marketdata"
	"github.com/TrebuchetDynamics/polygolem/pkg/marketresolver"
	"github.com/TrebuchetDynamics/polygolem/pkg/mcp"
	"github.com/TrebuchetDynamics/polygolem/pkg/openapi"
	"github.com/TrebuchetDynamics/polygolem/pkg/orderbook"
	"github.com/TrebuchetDynamics/polygolem/pkg/orderfills"
	"github.com/TrebuchetDynamics/polygolem/pkg/orderresults"
	"github.com/TrebuchetDynamics/polygolem/pkg/pagination"
	"github.com/TrebuchetDynamics/polygolem/pkg/plugins"
	"github.com/TrebuchetDynamics/polygolem/pkg/polyerrors"
	"github.com/TrebuchetDynamics/polygolem/pkg/reconciliation"
	"github.com/TrebuchetDynamics/polygolem/pkg/relayer"
	"github.com/TrebuchetDynamics/polygolem/pkg/rfq"
	"github.com/TrebuchetDynamics/polygolem/pkg/settlement"
	"github.com/TrebuchetDynamics/polygolem/pkg/signers"
	httpsigner "github.com/TrebuchetDynamics/polygolem/pkg/signers/http"
	kmssigner "github.com/TrebuchetDynamics/polygolem/pkg/signers/kms"
	turnkeysigner "github.com/TrebuchetDynamics/polygolem/pkg/signers/turnkey"
	sdkstream "github.com/TrebuchetDynamics/polygolem/pkg/stream"
	"github.com/TrebuchetDynamics/polygolem/pkg/types"
	"github.com/TrebuchetDynamics/polygolem/pkg/universal"
	"github.com/TrebuchetDynamics/polygolem/pkg/upstreamdrift"
	"github.com/TrebuchetDynamics/polygolem/pkg/wallet"
	"github.com/TrebuchetDynamics/polygolem/pkg/experimental/orders"
	"github.com/TrebuchetDynamics/polygolem/pkg/experimental/auth"
)

func TestPublicSDKSignatures(t *testing.T) {
	var bridgeClient *bridge.Client = bridge.NewClient("", nil)
	var bridgeWithdrawRequest bridge.WithdrawRequest
	var bridgeWithdrawDryRun *bridge.WithdrawDryRun
	var bridgeBuildWithdrawDryRun func(bridge.WithdrawRequest) (*bridge.WithdrawDryRun, error) = bridge.BuildWithdrawDryRun
	var bridgeWithdraw func(*bridge.Client, context.Context, bridge.WithdrawRequest) (*bridge.WithdrawResponse, error) = (*bridge.Client).Withdraw
	var builderSigner builder.Signer
	var builderConfig builder.LocalSignerConfig
	var builderNewLocal func(builder.LocalSignerConfig) (*builder.LocalSigner, error) = builder.NewLocalSigner
	var capabilityList []capabilities.Capability = capabilities.All()
	var readOnlyCapabilities []string = capabilities.ReadOnlyIDs()
	var compatibilityContract compat.CompatibilityContract = compat.Contract()
	var clobClient *sdkclob.Client = sdkclob.NewClient(sdkclob.Config{})
	var clobConfig sdkclob.Config = sdkclob.Config{BuilderCode: "0x1111111111111111111111111111111111111111111111111111111111111111"}
	var clobMarkets func(*sdkclob.Client, context.Context, string) (*types.CLOBPaginatedMarkets, error) = (*sdkclob.Client).Markets
	var clobMarket func(*sdkclob.Client, context.Context, string) (*types.CLOBMarket, error) = (*sdkclob.Client).Market
	var clobMarketByToken func(*sdkclob.Client, context.Context, string) (*types.CLOBMarketByTokenResponse, error) = (*sdkclob.Client).MarketByToken
	var clobOrderBook func(*sdkclob.Client, context.Context, string) (*types.CLOBOrderBook, error) = (*sdkclob.Client).OrderBook
	var clobOrderBooks func(*sdkclob.Client, context.Context, []types.CLOBBookParams) ([]types.CLOBOrderBook, error) = (*sdkclob.Client).OrderBooks
	var clobTickSize func(*sdkclob.Client, context.Context, string) (*types.CLOBTickSize, error) = (*sdkclob.Client).TickSize
	var clobPriceHistory func(*sdkclob.Client, context.Context, *types.CLOBPriceHistoryParams) (*types.CLOBPriceHistory, error) = (*sdkclob.Client).PricesHistory
	var clobAPIKey sdkclob.APIKey
	var clobDeriveAPIKey func(*sdkclob.Client, context.Context, string) (sdkclob.APIKey, error) = (*sdkclob.Client).DeriveAPIKey
	var clobBalanceParams sdkclob.BalanceAllowanceParams
	var clobBalance func(*sdkclob.Client, context.Context, string, sdkclob.BalanceAllowanceParams) (*sdkclob.BalanceAllowanceResponse, error) = (*sdkclob.Client).BalanceAllowance
	var clobOrders func(*sdkclob.Client, context.Context, string) ([]sdkclob.OrderRecord, error) = (*sdkclob.Client).ListOrders
	var clobOrder func(*sdkclob.Client, context.Context, string, string) (*sdkclob.OrderRecord, error) = (*sdkclob.Client).Order
	var clobTrades func(*sdkclob.Client, context.Context, string) ([]sdkclob.TradeRecord, error) = (*sdkclob.Client).ListTrades
	var clobCancel func(*sdkclob.Client, context.Context, string, string) (*sdkclob.CancelOrdersResponse, error) = (*sdkclob.Client).CancelOrder
	var clobCancelMarketParams sdkclob.CancelMarketParams
	var clobCancelMarket func(*sdkclob.Client, context.Context, string, sdkclob.CancelMarketParams) (*sdkclob.CancelOrdersResponse, error) = (*sdkclob.Client).CancelMarket
	var clobCreateParams sdkclob.CreateOrderParams
	var clobCreate func(*sdkclob.Client, context.Context, string, sdkclob.CreateOrderParams) (*sdkclob.OrderPlacementResponse, error) = (*sdkclob.Client).CreateLimitOrder
	var clobMarketOrderParams sdkclob.MarketOrderParams
	var clobMarketOrder func(*sdkclob.Client, context.Context, string, sdkclob.MarketOrderParams) (*sdkclob.OrderPlacementResponse, error) = (*sdkclob.Client).CreateMarketOrder
	var streamClient *sdkstream.MarketClient = sdkstream.NewMarketClient(sdkstream.Config{})
	var streamConfig sdkstream.Config = sdkstream.DefaultConfig("")
	var streamConnect func(*sdkstream.MarketClient, context.Context) error = (*sdkstream.MarketClient).Connect
	var streamSubscribe func(*sdkstream.MarketClient, context.Context, []string) error = (*sdkstream.MarketClient).SubscribeAssets
	var streamClose func(*sdkstream.MarketClient) = (*sdkstream.MarketClient).Close
	var streamConnected func(*sdkstream.MarketClient) bool = (*sdkstream.MarketClient).IsConnected
	var streamBook sdkstream.BookMessage
	var streamPriceChange sdkstream.PriceChangeMessage
	var streamLastTrade sdkstream.LastTradeMessage
	var streamTickSize sdkstream.TickSizeChangeMessage
	var streamBestBidAsk sdkstream.BestBidAskMessage
	var streamNewMarket sdkstream.NewMarketMessage
	var streamMarketResolved sdkstream.MarketResolvedMessage
	var streamDeduplicator *sdkstream.Deduplicator = sdkstream.NewDeduplicator(100, 0)
	var marketDataTracker *marketdata.Tracker = marketdata.NewTracker()
	var marketDataSnapshot marketdata.Snapshot
	var marketDataBestBidAsk func(*marketdata.Tracker, sdkstream.BestBidAskMessage) marketdata.Snapshot = (*marketdata.Tracker).ApplyBestBidAsk
	var marketDataTickSize func(*marketdata.Tracker, sdkstream.TickSizeChangeMessage) marketdata.Snapshot = (*marketdata.Tracker).ApplyTickSizeChange
	var marketResolver *marketresolver.Resolver = marketresolver.NewResolver("")
	var marketResolverResult marketresolver.ResolveResult
	var cryptoPriceClient *cryptoprice.Client = cryptoprice.NewClient(cryptoprice.Config{})
	var cryptoPrice cryptoprice.CryptoPrice
	var enableTradingParams enabletrading.EnableTradingParams
	var enableTradingBuildCalls func() []enabletrading.DepositWalletCall = enabletrading.BuildEnableTradingApprovalCalls
	var fundingTransfer func(context.Context, string, string, *big.Int, string) (string, error) = funding.TransferPUSD
	var intelScore intel.WalletScore = intel.ScoreWallet(intel.ScoreInput{Wallet: "0xabc"})
	var contractsMaxUint *big.Int = contracts.MaxUint256()
	var contractsApprove func(string, *big.Int) ([]byte, error) = contracts.ERC20ApproveCalldata
	var contractsTransfer func(string, *big.Int) ([]byte, error) = contracts.ERC20TransferCalldata
	var contractsAllowance func(string, string) ([]byte, error) = contracts.ERC20AllowanceCalldata
	var contractsBalanceOf func(string) ([]byte, error) = contracts.ERC20BalanceOfCalldata
	var contractsSetApprovalForAll func(string, bool) ([]byte, error) = contracts.ERC1155SetApprovalForAllCalldata
	var contractsIsApprovedForAll func(string, string) ([]byte, error) = contracts.ERC1155IsApprovedForAllCalldata
	var contractsRampWrap func(string, string, *big.Int) ([]byte, error) = contracts.RampWrapCalldata
	var contractsRampUnwrap func(string, string, *big.Int) ([]byte, error) = contracts.RampUnwrapCalldata
	var contractsDecodeUint func([]byte) (*big.Int, error) = contracts.DecodeUint256Result
	var contractsDecodeBool func([]byte) (bool, error) = contracts.DecodeBoolResult
	_, _, _, _, _, _, _, _, _, _, _ = contractsMaxUint, contractsApprove, contractsTransfer, contractsAllowance, contractsBalanceOf, contractsSetApprovalForAll, contractsIsApprovedForAll, contractsRampWrap, contractsRampUnwrap, contractsDecodeUint, contractsDecodeBool
	var geoblockClient *geoblock.Client = geoblock.New("", nil)
	var geoblockCheck func(*geoblock.Client, context.Context) (geoblock.Result, error) = (*geoblock.Client).Check
	var bookBestBid func(types.CLOBOrderBook) (float64, bool) = types.CLOBOrderBook.BestBid
	var bookBestAsk func(types.CLOBOrderBook) (float64, bool) = types.CLOBOrderBook.BestAsk
	var bookAskDepth func(types.CLOBOrderBook, float64) float64 = types.CLOBOrderBook.AvailableAskSize
	var tickSizeValue func(types.CLOBTickSize) (float64, error) = types.CLOBTickSize.Value
	var outcomeForToken func(string, string, string) string = marketresolver.OutcomeForToken
	var normalizeOutcome func(string) string = marketresolver.NormalizeOutcome
	var upDownTokenIDs func([]string, []string) (string, string) = marketresolver.UpDownTokenIDs
	var inferTimeframe func(...string) string = marketresolver.InferTimeframe
	var inferTimeframeFromWindow func(time.Time, time.Time) string = marketresolver.InferTimeframeFromWindow
	var windowFromSlug func(string, string) (time.Time, time.Time, bool) = marketresolver.WindowFromSlug
	var assetSearchQueries func(string) []string = marketresolver.AssetSearchQueries
	var assetMentioned func(string, string) bool = marketresolver.AssetMentioned
	var parseJSONStringList func(string) ([]string, error) = marketresolver.ParseJSONStringList
	_, _, _, _, _, _, _ = upDownTokenIDs, inferTimeframe, inferTimeframeFromWindow, windowFromSlug, assetSearchQueries, assetMentioned, parseJSONStringList
	var marketOutcomeForToken func(marketresolver.CryptoMarket, string) string = marketresolver.CryptoMarket.OutcomeForToken
	_, _, _, _, _, _, _, _, _ = geoblockClient, geoblockCheck, bookBestBid, bookBestAsk, bookAskDepth, tickSizeValue, outcomeForToken, normalizeOutcome, marketOutcomeForToken
	var mcpTools []mcp.Tool = mcp.SafeTools()
	var mcpServer *mcp.Server = mcp.NewServer()
	var openAPISpec map[string]any = openapi.Spec()
	var orderbookReader orderbook.Reader = orderbook.NewReader("")
	var orderbookSnapshot orderbook.OrderBook
	var orderbookLevel orderbook.Level
	var orderFillsReader orderfills.Reader
	var orderFillsQuery orderfills.Query
	var orderFillsValidate func(orderfills.Query) error = orderfills.ValidateQuery
	var orderResultsSource orderresults.Source
	var orderResultsReport *orderresults.Report
	var orderResultsOptions orderresults.Options
	var orderResultsBuild func(context.Context, orderresults.DataReader, string, orderresults.Options) (*orderresults.Report, error) = orderresults.BuildReport
	var paginationCollect func(context.Context, pagination.Page[int]) ([]int, error) = pagination.CollectAll[int]
	var pluginsOrder plugins.Order
	var ctfOperationRequest ctf.OperationRequest
	var ctfOperationDryRun *ctf.OperationDryRun
	var ctfReadinessGate ctf.ReadinessGate
	var ctfSubmitPlan *ctf.OperationSubmitPlan
	var ctfBuildDryRun func(ctf.OperationRequest) (*ctf.OperationDryRun, error) = ctf.BuildOperationDryRun
	var ctfBuildSubmitPlan func(ctf.OperationRequest, ctf.ReadinessGate) (*ctf.OperationSubmitPlan, error) = ctf.BuildOperationSubmitPlan
	var ctfBuildSplit func(ctf.OperationRequest) (relayer.DepositWalletCall, error) = ctf.BuildSplitCall
	var ctfBuildMerge func(ctf.OperationRequest) (relayer.DepositWalletCall, error) = ctf.BuildMergeCall
	var contractsRegistry contracts.Registry = contracts.PolygonMainnet()
	var contractStatus contracts.DeploymentStatus
	var contractDeployed func(context.Context, string, string) (contracts.DeploymentStatus, error) = contracts.ContractDeployed
	var depositWalletDeployed func(context.Context, string, string) (contracts.DeploymentStatus, error) = contracts.DepositWalletDeployed
	var redeemAdapterFor func(bool) string = contracts.RedeemAdapterFor
	var reconciliationReport reconciliation.Report = reconciliation.BuildReport(reconciliation.Input{Order: &reconciliation.OrderEvidence{ID: "order-1"}})
	var rfqClient *rfq.Client = rfq.NewClient()
	var rfqRequest rfq.Request
	var rfqQuote rfq.Quote
	var rfqValidate func(rfq.Request) error = rfq.ValidateRequest
	var rfqSubmit func(*rfq.Client, rfq.Request) (*rfq.Response, error) = (*rfq.Client).Submit
	var settlementPosition settlement.RedeemablePosition
	var settlementResult *settlement.RedeemResult
	var settlementReadiness *settlement.Readiness
	var settlementReadinessOptions settlement.ReadinessOptions
	var settlementAdapterApproval settlement.AdapterApproval
	var settlementFind func(context.Context, *data.Client, string) ([]settlement.RedeemablePosition, error) = settlement.FindRedeemable
	var settlementBuild func(settlement.RedeemablePosition) (relayer.DepositWalletCall, error) = settlement.BuildRedeemCall
	var settlementSubmit func(context.Context, *relayer.Client, string, []settlement.RedeemablePosition, int) (*settlement.RedeemResult, error) = settlement.SubmitRedeem
	var settlementCheck func(context.Context, *data.Client, string, string, settlement.ReadinessOptions) (*settlement.Readiness, error) = settlement.CheckReadiness
	var localSigner *signers.LocalSigner
	var signerInterface signers.Signer
	var newLocalSigner func(string, int64) (*signers.LocalSigner, error) = signers.NewLocalSigner
	var redactSecret func(string) string = signers.RedactSecret
	var httpSigner *httpsigner.Signer
	var httpSignerConfig httpsigner.Config
	var newHTTPSigner func(httpsigner.Config) (*httpsigner.Signer, error) = httpsigner.New
	var kmsSigner *kmssigner.Signer
	var kmsSignerConfig kmssigner.Config
	var newKMSSigner func(kmssigner.Config) (*kmssigner.Signer, error) = kmssigner.New
	var turnkeySigner *turnkeysigner.Signer
	var turnkeySignerConfig turnkeysigner.Config
	var newTurnkeySigner func(turnkeysigner.Config) (*turnkeysigner.Signer, error) = turnkeysigner.New
	var relayerClient *relayer.Client
	var relayerV2Key relayer.V2APIKey
	var relayerOnboardOptions relayer.OnboardOptions
	var relayerOnboard func(context.Context, *relayer.Client, string, relayer.OnboardOptions) (*relayer.OnboardResult, error) = relayer.OnboardDepositWallet
	var relayerNewV2 func(string, relayer.V2APIKey, int64) (*relayer.Client, error) = relayer.NewV2
	var dataPositions func(*data.Client, context.Context, string) ([]types.Position, error) = (*data.Client).CurrentPositions
	var universalPositions func(*universal.Client, context.Context, string) ([]types.Position, error) = (*universal.Client).CurrentPositions
	var dataLeaderboard func(*data.Client, context.Context, int) ([]types.LeaderboardRow, error) = (*data.Client).TraderLeaderboard
	var universalLiveVolume func(*universal.Client, context.Context, int) (*types.LiveVolumeResponse, error) = (*universal.Client).LiveVolume
	var gammaMarkets func(*gamma.Client, context.Context, *types.GetMarketsParams) ([]types.Market, error) = (*gamma.Client).Markets
	var gammaSearch func(*gamma.Client, context.Context, *types.SearchParams) (*types.SearchResponse, error) = (*gamma.Client).Search
	var gammaComments func(*gamma.Client, context.Context, *types.CommentQuery) ([]types.Comment, error) = (*gamma.Client).Comments
	var universalMarkets func(*universal.Client, context.Context, *types.GetMarketsParams) ([]types.Market, error) = (*universal.Client).Markets
	var universalSearch func(*universal.Client, context.Context, *types.SearchParams) (*types.SearchResponse, error) = (*universal.Client).Search
	var universalComments func(*universal.Client, context.Context, *types.CommentQuery) ([]types.Comment, error) = (*universal.Client).Comments
	var universalConfig universal.Config = universal.Config{BuilderCode: "0x1111111111111111111111111111111111111111111111111111111111111111"}
	var driftReport upstreamdrift.Report = upstreamdrift.CheckLLMS("https://docs.polymarket.com/trading/overview")
	var universalCLOBMarkets func(*universal.Client, context.Context, string) (*types.CLOBPaginatedMarkets, error) = (*universal.Client).CLOBMarkets
	var universalCLOBMarket func(*universal.Client, context.Context, string) (*types.CLOBMarket, error) = (*universal.Client).CLOBMarket
	var universalCLOBMarketByToken func(*universal.Client, context.Context, string) (*types.CLOBMarketByTokenResponse, error) = (*universal.Client).CLOBMarketByToken
	var universalOrderBook func(*universal.Client, context.Context, string) (*types.CLOBOrderBook, error) = (*universal.Client).OrderBook
	var universalOrderBooks func(*universal.Client, context.Context, []types.CLOBBookParams) ([]types.CLOBOrderBook, error) = (*universal.Client).OrderBooks
	var universalTickSize func(*universal.Client, context.Context, string) (*types.CLOBTickSize, error) = (*universal.Client).TickSize
	var universalPriceHistory func(*universal.Client, context.Context, *types.CLOBPriceHistoryParams) (*types.CLOBPriceHistory, error) = (*universal.Client).PricesHistory
	var universalDeriveAPIKey func(*universal.Client, context.Context, string) (sdkclob.APIKey, error) = (*universal.Client).DeriveAPIKey
	var universalBalance func(*universal.Client, context.Context, string, sdkclob.BalanceAllowanceParams) (*sdkclob.BalanceAllowanceResponse, error) = (*universal.Client).BalanceAllowance
	var universalOrders func(*universal.Client, context.Context, string) ([]sdkclob.OrderRecord, error) = (*universal.Client).ListOrders
	var universalOrder func(*universal.Client, context.Context, string, string) (*sdkclob.OrderRecord, error) = (*universal.Client).Order
	var universalTrades func(*universal.Client, context.Context, string) ([]sdkclob.TradeRecord, error) = (*universal.Client).ListTrades
	var universalCancel func(*universal.Client, context.Context, string, string) (*sdkclob.CancelOrdersResponse, error) = (*universal.Client).CancelOrder
	var universalCancelMarket func(*universal.Client, context.Context, string, sdkclob.CancelMarketParams) (*sdkclob.CancelOrdersResponse, error) = (*universal.Client).CancelMarket
	var universalCreate func(*universal.Client, context.Context, string, sdkclob.CreateOrderParams) (*sdkclob.OrderPlacementResponse, error) = (*universal.Client).CreateLimitOrder
	var universalMarketOrder func(*universal.Client, context.Context, string, sdkclob.MarketOrderParams) (*sdkclob.OrderPlacementResponse, error) = (*universal.Client).CreateMarketOrder
	var universalStream func(*universal.Client) *sdkstream.MarketClient = (*universal.Client).StreamClient
	var universalStreamWithConfig func(*universal.Client, sdkstream.Config) *sdkstream.MarketClient = (*universal.Client).StreamClientWithConfig

	_, _, _, _, _ = bridgeClient, bridgeWithdrawRequest, bridgeWithdrawDryRun, bridgeBuildWithdrawDryRun, bridgeWithdraw
	_, _, _ = builderSigner, builderConfig, builderNewLocal
	_, _, _ = capabilityList, readOnlyCapabilities, compatibilityContract
	_, _, _, _, _, _, _, _, _ = clobClient, clobConfig, clobMarkets, clobMarket, clobMarketByToken, clobOrderBook, clobOrderBooks, clobTickSize, clobPriceHistory
	_, _, _, _, _, _, _, _, _, _ = clobAPIKey, clobDeriveAPIKey, clobBalanceParams, clobBalance, clobOrders, clobOrder, clobTrades, clobCancel, clobCancelMarketParams, clobCancelMarket
	_, _, _, _ = clobCreateParams, clobCreate, clobMarketOrderParams, clobMarketOrder
	_, _, _, _, _, _, _, _, _, _ = streamClient, streamConfig, streamConnect, streamSubscribe, streamClose, streamConnected, streamBook, streamPriceChange, streamLastTrade, streamTickSize
	_, _, _, _, _, _ = streamBestBidAsk, streamNewMarket, streamMarketResolved, streamDeduplicator, marketDataTracker, marketDataSnapshot
	_, _ = marketDataBestBidAsk, marketDataTickSize
	_, _, _, _ = marketResolver, marketResolverResult, enableTradingParams, enableTradingBuildCalls
	_, _, _, _, _, _, _ = cryptoPriceClient, cryptoPrice, fundingTransfer, intelScore, mcpTools, mcpServer, openAPISpec
	var normalizedError polyerrors.Error = polyerrors.Normalize(polyerrors.Input{HTTPStatus: 429})
	_ = normalizedError
	_, _, _, _, _, _, _, _, _, _, _, _ = orderbookReader, orderbookSnapshot, orderbookLevel, orderFillsReader, orderFillsQuery, orderFillsValidate, orderResultsSource, orderResultsReport, orderResultsOptions, orderResultsBuild, paginationCollect, pluginsOrder
	_, _, _, _, _, _, _, _ = ctfOperationRequest, ctfOperationDryRun, ctfReadinessGate, ctfSubmitPlan, ctfBuildDryRun, ctfBuildSubmitPlan, ctfBuildSplit, ctfBuildMerge
	_, _, _, _, _ = contractsRegistry, contractStatus, contractDeployed, depositWalletDeployed, redeemAdapterFor
	_, _, _, _, _, _ = reconciliationReport, rfqClient, rfqRequest, rfqQuote, rfqValidate, rfqSubmit
	_, _, _, _, _, _, _, _, _ = settlementPosition, settlementResult, settlementReadiness, settlementReadinessOptions, settlementAdapterApproval, settlementFind, settlementBuild, settlementSubmit, settlementCheck
	_, _, _, _, _, _, _ = localSigner, signerInterface, newLocalSigner, redactSecret, httpSigner, httpSignerConfig, newHTTPSigner
	_, _, _ = kmsSigner, kmsSignerConfig, newKMSSigner
	_, _, _ = turnkeySigner, turnkeySignerConfig, newTurnkeySigner
	_, _, _, _, _ = relayerClient, relayerV2Key, relayerOnboardOptions, relayerOnboard, relayerNewV2
	_, _, _, _ = dataPositions, universalPositions, dataLeaderboard, universalLiveVolume
	_, _, _, _, _, _, _ = gammaMarkets, gammaSearch, gammaComments, universalMarkets, universalSearch, universalComments, universalConfig
	_ = driftReport
	_, _, _, _, _, _, _ = universalCLOBMarkets, universalCLOBMarket, universalCLOBMarketByToken, universalOrderBook, universalOrderBooks, universalTickSize, universalPriceHistory
	_, _, _, _, _, _, _, _, _ = universalDeriveAPIKey, universalBalance, universalOrders, universalOrder, universalTrades, universalCancel, universalCancelMarket, universalCreate, universalMarketOrder
	_, _ = universalStream, universalStreamWithConfig

	var walletProxy func(string) string = wallet.DeriveProxyWallet
	var walletSafe func(string) string = wallet.DeriveSafeWallet
	var walletReady func(int64, string) wallet.ReadyInfo = wallet.Readiness
	_ = walletProxy
	_ = walletSafe
	_ = walletReady

	var expOrder orders.OrderIntent = orders.OrderIntent{TokenID: "123", Side: orders.SideBuy}
	var expOrderValidate func() error = expOrder.Validate
	var expAuthDomain auth.EIP712Domain = auth.EIP712Domain{Name: "Test", Version: "1", ChainID: 137, VerifyingContract: "0x123"}
	var expAuthValidType func(int) bool = auth.IsValidSignatureType
	_ = expOrderValidate
	_ = expAuthDomain
	_ = expAuthValidType
}
`)

	cmd := exec.Command("go", "test", "-mod=mod", "./...")
	cmd.Dir = tempDir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("external consumer compile failed: %v\n%s", err, out)
	}
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}
