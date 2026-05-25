package orderbookreads

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/polygolem/internal/polytypes"
)

type fakeReader struct {
	priceToken string
	priceSide  string
}

func (f *fakeReader) OrderBook(context.Context, string) (*polytypes.OrderBook, error) {
	return nil, errors.New("not used")
}

func (f *fakeReader) Price(_ context.Context, tokenID, side string) (string, error) {
	f.priceToken = tokenID
	f.priceSide = side
	return "0.42", nil
}

func (f *fakeReader) Midpoint(context.Context, string) (string, error) {
	return "", errors.New("not used")
}

func (f *fakeReader) Spread(context.Context, string) (string, error) {
	return "", errors.New("not used")
}

func (f *fakeReader) TickSize(context.Context, string) (*polytypes.TickSize, error) {
	return nil, errors.New("not used")
}

func (f *fakeReader) FeeRateBps(context.Context, string) (int, error) {
	return 0, errors.New("not used")
}

func (f *fakeReader) LastTradePrice(context.Context, string) (string, error) {
	return "", errors.New("not used")
}

func TestRunnerShapesBestBuyPrice(t *testing.T) {
	reader := &fakeReader{}
	runner := New(reader)

	got, err := runner.Run(context.Background(), Request{Operation: Price, TokenID: "token-1"})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	want := map[string]string{"token_id": "token-1", "price": "0.42"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("result=%+v, want %+v", got, want)
	}
	if reader.priceToken != "token-1" || reader.priceSide != "BUY" {
		t.Fatalf("price called with token=%q side=%q", reader.priceToken, reader.priceSide)
	}
}

func TestRunnerRejectsMissingTokenID(t *testing.T) {
	runner := New(&fakeReader{})

	_, err := runner.Run(context.Background(), Request{Operation: Price})
	if err == nil || !strings.Contains(err.Error(), "--token-id required") {
		t.Fatalf("error=%v, want --token-id required", err)
	}
}
