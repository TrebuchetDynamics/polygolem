package orderfills

import (
	"context"
	"math/big"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestValidateQueryRejectsMissingBlockRange(t *testing.T) {
	err := ValidateQuery(Query{})
	if err == nil || !strings.Contains(err.Error(), "block range") {
		t.Fatalf("ValidateQuery(Query{}) error=%v, want missing block range", err)
	}
}

func TestNormalizeFillAcceptsOnlyBuyOrSellSide(t *testing.T) {
	fill := validFill()
	fill.Side = "buy"

	normalized, err := NormalizeFill(fill)
	if err != nil {
		t.Fatalf("NormalizeFill() error=%v", err)
	}
	if normalized.Side != SideBUY {
		t.Fatalf("Side=%q, want %q", normalized.Side, SideBUY)
	}

	fill.Side = "HOLD"
	if _, err := NormalizeFill(fill); err == nil || !strings.Contains(err.Error(), "side") {
		t.Fatalf("NormalizeFill(HOLD) error=%v, want side validation error", err)
	}
}

func TestNormalizeFillRequiresOnchainOrderFilledSource(t *testing.T) {
	fill := validFill()
	fill.Source = ""

	normalized, err := NormalizeFill(fill)
	if err != nil {
		t.Fatalf("NormalizeFill() error=%v", err)
	}
	if normalized.Source != SourceOnchainOrderFilled {
		t.Fatalf("Source=%q, want %q", normalized.Source, SourceOnchainOrderFilled)
	}

	fill.Source = "clob_trades"
	if _, err := NormalizeFill(fill); err == nil || !strings.Contains(err.Error(), "source") {
		t.Fatalf("NormalizeFill(clob_trades) error=%v, want source validation error", err)
	}
}

func TestPublicTypesDoNotExposeSecretsOrAuthHeaders(t *testing.T) {
	for _, typ := range []reflect.Type{
		reflect.TypeOf(Fill{}),
		reflect.TypeOf(Query{}),
		reflect.TypeOf((*Reader)(nil)).Elem(),
	} {
		text := strings.ToLower(typ.String())
		switch typ.Kind() {
		case reflect.Struct:
			for i := 0; i < typ.NumField(); i++ {
				text += " " + strings.ToLower(typ.Field(i).Name)
			}
		case reflect.Interface:
			for i := 0; i < typ.NumMethod(); i++ {
				method := typ.Method(i)
				text += " " + strings.ToLower(method.Name)
				text += " " + strings.ToLower(method.Type.String())
			}
		}
		for _, banned := range []string{"private", "secret", "signature", "auth", "header", "key"} {
			if strings.Contains(text, banned) {
				t.Fatalf("%s exposes banned term %q", typ, banned)
			}
		}
	}
}

type stubReader struct{}

func (stubReader) OrderFilled(context.Context, Query) ([]Fill, error) {
	return []Fill{validFill()}, nil
}

func TestReaderInterfaceUsesTypedQueryAndFills(t *testing.T) {
	var reader Reader = stubReader{}

	fills, err := reader.OrderFilled(context.Background(), Query{FromBlock: 1, ToBlock: 2})
	if err != nil {
		t.Fatalf("OrderFilled() error=%v", err)
	}
	if len(fills) != 1 || fills[0].Source != SourceOnchainOrderFilled {
		t.Fatalf("fills=%#v", fills)
	}
}

func TestRPCReaderDecodesOrderFilledBuyAndSell(t *testing.T) {
	observedAt := time.Date(2026, 5, 16, 12, 30, 0, 0, time.UTC)
	exchange := common.HexToAddress("0xE111180000d2663C0091e4f400237545B87B996B")
	yesTokenID := "111"
	noTokenID := "222"
	client := &fakeOrderFilledLogClient{
		logs: []types.Log{
			orderFilledLog(exchange, 10, 7, yesTokenID, "0", "10000000", "4500000"),
			orderFilledLog(exchange, 10, 8, "0", noTokenID, "3000000", "5000000"),
		},
		blockTimes: map[uint64]time.Time{10: observedAt},
	}
	reader := &rpcReader{
		rpcURL: "http://polygon.invalid",
		newClient: func(context.Context, string) (orderFilledLogClient, error) {
			return client, nil
		},
	}

	fills, err := reader.OrderFilled(context.Background(), Query{
		FromBlock: 10,
		ToBlock:   10,
		Markets: []Market{
			{
				MarketID:    "market-1",
				ConditionID: "condition-1",
				YesTokenID:  yesTokenID,
				NoTokenID:   noTokenID,
			},
		},
	})
	if err != nil {
		t.Fatalf("OrderFilled() error=%v", err)
	}
	if len(fills) != 2 {
		t.Fatalf("fills=%#v, want 2", fills)
	}
	if fills[0].Side != SideBUY || fills[0].MarketID != "market-1" || fills[0].ConditionID != "condition-1" || fills[0].TokenID != yesTokenID {
		t.Fatalf("buy fill=%#v", fills[0])
	}
	if fills[0].Price != "0.45" || fills[0].Size != "10" || !fills[0].FilledAt.Equal(observedAt) {
		t.Fatalf("buy price/size/time=%#v", fills[0])
	}
	if fills[1].Side != SideSELL || fills[1].TokenID != noTokenID {
		t.Fatalf("sell fill=%#v", fills[1])
	}
	if fills[1].Price != "0.6" || fills[1].Size != "5" {
		t.Fatalf("sell price/size=%#v", fills[1])
	}
	if len(client.gotQuery.Addresses) != 2 {
		t.Fatalf("default exchange addresses=%#v, want CTF and neg-risk exchanges", client.gotQuery.Addresses)
	}
}

func TestRPCReaderSkipsUnmappedTokenFills(t *testing.T) {
	exchange := common.HexToAddress("0xE111180000d2663C0091e4f400237545B87B996B")
	client := &fakeOrderFilledLogClient{
		logs: []types.Log{
			orderFilledLog(exchange, 10, 7, "999", "0", "1000000", "500000"),
		},
		blockTimes: map[uint64]time.Time{10: time.Date(2026, 5, 16, 12, 30, 0, 0, time.UTC)},
	}
	reader := &rpcReader{
		rpcURL: "http://polygon.invalid",
		newClient: func(context.Context, string) (orderFilledLogClient, error) {
			return client, nil
		},
	}

	fills, err := reader.OrderFilled(context.Background(), Query{
		FromBlock: 10,
		ToBlock:   10,
		Markets: []Market{{
			MarketID:   "market-1",
			YesTokenID: "111",
			NoTokenID:  "222",
		}},
	})
	if err != nil {
		t.Fatalf("OrderFilled() error=%v", err)
	}
	if len(fills) != 0 {
		t.Fatalf("fills=%#v, want no unmapped-token fills", fills)
	}
}

func TestRPCReaderLatestBlockNumber(t *testing.T) {
	client := &fakeOrderFilledLogClient{latestBlock: 12345}
	reader := &rpcReader{
		rpcURL: "http://polygon.invalid",
		newClient: func(context.Context, string) (orderFilledLogClient, error) {
			return client, nil
		},
	}

	block, err := reader.LatestBlockNumber(context.Background())
	if err != nil {
		t.Fatalf("LatestBlockNumber() error=%v", err)
	}
	if block != 12345 {
		t.Fatalf("LatestBlockNumber()=%d, want 12345", block)
	}
	if !client.closed {
		t.Fatalf("LatestBlockNumber() should close rpc client")
	}
}

func validFill() Fill {
	return Fill{
		TxHash:      "0xtx",
		LogIndex:    7,
		Exchange:    "0xexchange",
		MarketID:    "market-1",
		ConditionID: "condition-1",
		TokenID:     "token-1",
		Side:        SideSELL,
		Price:       "0.42",
		Size:        "10",
		BlockNumber: 12,
		FilledAt:    time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC),
		Source:      SourceOnchainOrderFilled,
	}
}

type fakeOrderFilledLogClient struct {
	logs        []types.Log
	blockTimes  map[uint64]time.Time
	latestBlock uint64
	gotQuery    ethereum.FilterQuery
	closed      bool
}

func (f *fakeOrderFilledLogClient) FilterLogs(_ context.Context, query ethereum.FilterQuery) ([]types.Log, error) {
	f.gotQuery = query
	return f.logs, nil
}

func (f *fakeOrderFilledLogClient) BlockByNumber(_ context.Context, number *big.Int) (*types.Block, error) {
	at := f.blockTimes[number.Uint64()]
	return types.NewBlockWithHeader(&types.Header{
		Number: number,
		Time:   uint64(at.Unix()),
	}), nil
}

func (f *fakeOrderFilledLogClient) BlockNumber(context.Context) (uint64, error) {
	return f.latestBlock, nil
}

func (f *fakeOrderFilledLogClient) Close() {
	f.closed = true
}

func orderFilledLog(exchange common.Address, blockNumber uint64, logIndex uint, makerAssetID string, takerAssetID string, makerAmount string, takerAmount string) types.Log {
	data, err := orderFilledEventDataArgs.Pack(
		mustBigInt(makerAssetID),
		mustBigInt(takerAssetID),
		mustBigInt(makerAmount),
		mustBigInt(takerAmount),
		big.NewInt(0),
	)
	if err != nil {
		panic(err)
	}
	return types.Log{
		Address:     exchange,
		Topics:      []common.Hash{orderFilledEventID, common.HexToHash("0x01"), common.HexToHash("0x02"), common.HexToHash("0x03")},
		Data:        data,
		BlockNumber: blockNumber,
		TxHash:      common.HexToHash("0xabc"),
		Index:       logIndex,
	}
}

func mustBigInt(raw string) *big.Int {
	value, ok := new(big.Int).SetString(raw, 10)
	if !ok {
		panic("invalid big.Int test value")
	}
	return value
}
