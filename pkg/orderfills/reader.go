package orderfills

import (
	"context"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/polygolem/pkg/contracts"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

const orderFilledEventSignature = "OrderFilled(bytes32,address,address,uint256,uint256,uint256,uint256,uint256)"

var (
	orderFilledEventID       = crypto.Keccak256Hash([]byte(orderFilledEventSignature))
	orderFilledEventDataArgs = abi.Arguments{
		{Name: "makerAssetId", Type: mustABIType("uint256")},
		{Name: "takerAssetId", Type: mustABIType("uint256")},
		{Name: "makerAmountFilled", Type: mustABIType("uint256")},
		{Name: "takerAmountFilled", Type: mustABIType("uint256")},
		{Name: "fee", Type: mustABIType("uint256")},
	}
)

type ReaderOptions struct {
	RPCURL            string
	ExchangeAddresses []string
}

func NewReader(rpcURL string) Reader {
	return NewReaderWithOptions(ReaderOptions{RPCURL: rpcURL})
}

func NewReaderWithOptions(options ReaderOptions) Reader {
	return &rpcReader{
		rpcURL:            strings.TrimSpace(options.RPCURL),
		exchangeAddresses: append([]string(nil), options.ExchangeAddresses...),
		newClient: func(ctx context.Context, rpcURL string) (orderFilledLogClient, error) {
			return ethclient.DialContext(ctx, rpcURL)
		},
	}
}

type orderFilledLogClient interface {
	FilterLogs(ctx context.Context, query ethereum.FilterQuery) ([]types.Log, error)
	BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)
	BlockNumber(ctx context.Context) (uint64, error)
	Close()
}

type rpcReader struct {
	rpcURL            string
	exchangeAddresses []string
	newClient         func(ctx context.Context, rpcURL string) (orderFilledLogClient, error)
}

func (r *rpcReader) OrderFilled(ctx context.Context, query Query) ([]Fill, error) {
	if err := ValidateQuery(query); err != nil {
		return nil, err
	}
	client, err := r.connect(ctx)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	exchanges, err := r.filterAddresses(query.ExchangeAddresses)
	if err != nil {
		return nil, err
	}
	logs, err := client.FilterLogs(ctx, ethereum.FilterQuery{
		FromBlock: new(big.Int).SetUint64(query.FromBlock),
		ToBlock:   new(big.Int).SetUint64(query.ToBlock),
		Addresses: exchanges,
		Topics:    [][]common.Hash{{orderFilledEventID}},
	})
	if err != nil {
		return nil, fmt.Errorf("orderfills filter OrderFilled logs: %w", err)
	}

	markets := newMarketIndex(query.Markets)
	marketFilter := strings.TrimSpace(query.MarketID)
	tokenFilter := newStringSet(query.TokenIDs, normalizeTokenID)
	conditionFilter := newStringSet(query.ConditionIDs, normalizeConditionID)
	blockTimes := map[uint64]time.Time{}
	fills := make([]Fill, 0, len(logs))
	for _, log := range logs {
		decoded, ok, err := decodeOrderFilledLog(log)
		if err != nil {
			return nil, fmt.Errorf("orderfills decode tx=%s log_index=%d: %w", log.TxHash.Hex(), log.Index, err)
		}
		if !ok {
			continue
		}
		if len(tokenFilter) > 0 {
			if _, ok := tokenFilter[normalizeTokenID(decoded.tokenID)]; !ok {
				continue
			}
		}
		market, mapped := markets.byToken[normalizeTokenID(decoded.tokenID)]
		if len(query.Markets) > 0 && !mapped {
			continue
		}
		if marketFilter != "" {
			if !mapped || strings.TrimSpace(market.MarketID) != marketFilter {
				continue
			}
		}
		if len(conditionFilter) > 0 {
			if !mapped {
				continue
			}
			if _, ok := conditionFilter[normalizeConditionID(market.ConditionID)]; !ok {
				continue
			}
		}
		filledAt, err := blockTimestamp(ctx, client, log.BlockNumber, blockTimes)
		if err != nil {
			return nil, err
		}
		fill, err := NormalizeFill(Fill{
			TxHash:      log.TxHash.Hex(),
			LogIndex:    log.Index,
			Exchange:    log.Address.Hex(),
			MarketID:    strings.TrimSpace(market.MarketID),
			ConditionID: strings.TrimSpace(market.ConditionID),
			TokenID:     decoded.tokenID,
			Side:        decoded.side,
			Price:       decoded.price,
			Size:        decoded.size,
			BlockNumber: log.BlockNumber,
			FilledAt:    filledAt,
			Source:      SourceOnchainOrderFilled,
		})
		if err != nil {
			return nil, err
		}
		fills = append(fills, fill)
	}
	sort.SliceStable(fills, func(i, j int) bool {
		if fills[i].BlockNumber != fills[j].BlockNumber {
			return fills[i].BlockNumber < fills[j].BlockNumber
		}
		return fills[i].LogIndex < fills[j].LogIndex
	})
	return fills, nil
}

func (r *rpcReader) LatestBlockNumber(ctx context.Context) (uint64, error) {
	client, err := r.connect(ctx)
	if err != nil {
		return 0, err
	}
	defer client.Close()
	block, err := client.BlockNumber(ctx)
	if err != nil {
		return 0, fmt.Errorf("orderfills latest polygon block: %w", err)
	}
	return block, nil
}

func (r *rpcReader) connect(ctx context.Context) (orderFilledLogClient, error) {
	rpcURL := strings.TrimSpace(r.rpcURL)
	if rpcURL == "" {
		rpcURL = contracts.PolygonRPC
	}
	newClient := r.newClient
	if newClient == nil {
		newClient = func(ctx context.Context, rpcURL string) (orderFilledLogClient, error) {
			return ethclient.DialContext(ctx, rpcURL)
		}
	}
	client, err := newClient(ctx, rpcURL)
	if err != nil {
		return nil, fmt.Errorf("orderfills dial polygon rpc: %w", err)
	}
	return client, nil
}

func (r *rpcReader) filterAddresses(queryAddresses []string) ([]common.Address, error) {
	addresses := queryAddresses
	if len(addresses) == 0 {
		addresses = r.exchangeAddresses
	}
	if len(addresses) == 0 {
		addresses = []string{contracts.CTFExchangeV2, contracts.NegRiskExchangeV2}
	}
	out := make([]common.Address, 0, len(addresses))
	for _, raw := range addresses {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		if !common.IsHexAddress(raw) {
			return nil, fmt.Errorf("orderfills exchange address %q is not a hex address", raw)
		}
		out = append(out, common.HexToAddress(raw))
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("orderfills exchange address is required")
	}
	return out, nil
}

type decodedOrderFilled struct {
	tokenID string
	side    string
	price   string
	size    string
}

func decodeOrderFilledLog(log types.Log) (decodedOrderFilled, bool, error) {
	if len(log.Topics) == 0 || log.Topics[0] != orderFilledEventID {
		return decodedOrderFilled{}, false, nil
	}
	values, err := orderFilledEventDataArgs.Unpack(log.Data)
	if err != nil {
		return decodedOrderFilled{}, false, err
	}
	makerAssetID, ok := values[0].(*big.Int)
	if !ok {
		return decodedOrderFilled{}, false, fmt.Errorf("makerAssetId is %T", values[0])
	}
	takerAssetID, ok := values[1].(*big.Int)
	if !ok {
		return decodedOrderFilled{}, false, fmt.Errorf("takerAssetId is %T", values[1])
	}
	makerAmountFilled, ok := values[2].(*big.Int)
	if !ok {
		return decodedOrderFilled{}, false, fmt.Errorf("makerAmountFilled is %T", values[2])
	}
	takerAmountFilled, ok := values[3].(*big.Int)
	if !ok {
		return decodedOrderFilled{}, false, fmt.Errorf("takerAmountFilled is %T", values[3])
	}

	var tokenID *big.Int
	var collateral *big.Int
	var shares *big.Int
	side := ""
	switch {
	case makerAssetID.Sign() != 0 && takerAssetID.Sign() == 0:
		tokenID = makerAssetID
		collateral = takerAmountFilled
		shares = makerAmountFilled
		side = SideBUY
	case makerAssetID.Sign() == 0 && takerAssetID.Sign() != 0:
		tokenID = takerAssetID
		collateral = makerAmountFilled
		shares = takerAmountFilled
		side = SideSELL
	default:
		return decodedOrderFilled{}, false, nil
	}
	if tokenID == nil || collateral == nil || shares == nil || tokenID.Sign() == 0 || collateral.Sign() <= 0 || shares.Sign() <= 0 {
		return decodedOrderFilled{}, false, nil
	}
	return decodedOrderFilled{
		tokenID: tokenID.String(),
		side:    side,
		price:   formatRatio(collateral, shares, 8),
		size:    formatScaled(shares, 6),
	}, true, nil
}

func blockTimestamp(ctx context.Context, client orderFilledLogClient, blockNumber uint64, cache map[uint64]time.Time) (time.Time, error) {
	if at, ok := cache[blockNumber]; ok {
		return at, nil
	}
	block, err := client.BlockByNumber(ctx, new(big.Int).SetUint64(blockNumber))
	if err != nil {
		return time.Time{}, fmt.Errorf("orderfills fetch block timestamp block=%d: %w", blockNumber, err)
	}
	at := time.Unix(int64(block.Time()), 0).UTC()
	cache[blockNumber] = at
	return at, nil
}

type marketIndex struct {
	byToken map[string]Market
}

func newMarketIndex(markets []Market) marketIndex {
	index := marketIndex{byToken: map[string]Market{}}
	for _, market := range markets {
		market.MarketID = strings.TrimSpace(market.MarketID)
		market.ConditionID = strings.TrimSpace(market.ConditionID)
		for _, tokenID := range []string{market.YesTokenID, market.NoTokenID} {
			tokenID = normalizeTokenID(tokenID)
			if tokenID == "" {
				continue
			}
			index.byToken[tokenID] = market
		}
	}
	return index
}

func newStringSet(values []string, normalize func(string) string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, value := range values {
		value = normalize(value)
		if value == "" {
			continue
		}
		out[value] = struct{}{}
	}
	return out
}

func normalizeTokenID(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	lower := strings.ToLower(raw)
	if strings.HasPrefix(lower, "0x") {
		if value, ok := new(big.Int).SetString(lower[2:], 16); ok {
			return value.String()
		}
		return raw
	}
	if value, ok := new(big.Int).SetString(raw, 10); ok {
		return value.String()
	}
	return raw
}

func normalizeConditionID(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func formatRatio(numerator *big.Int, denominator *big.Int, precision int) string {
	if denominator == nil || denominator.Sign() == 0 {
		return ""
	}
	return trimDecimal(new(big.Rat).SetFrac(numerator, denominator).FloatString(precision))
}

func formatScaled(value *big.Int, decimals int) string {
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	return trimDecimal(new(big.Rat).SetFrac(value, scale).FloatString(decimals))
}

func trimDecimal(value string) string {
	value = strings.TrimRight(value, "0")
	value = strings.TrimRight(value, ".")
	if value == "" {
		return "0"
	}
	return value
}

func mustABIType(name string) abi.Type {
	typ, err := abi.NewType(name, "", nil)
	if err != nil {
		panic(err)
	}
	return typ
}
