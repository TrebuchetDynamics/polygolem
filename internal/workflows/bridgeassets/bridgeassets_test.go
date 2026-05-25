package bridgeassets

import (
	"context"
	"reflect"
	"testing"

	"github.com/TrebuchetDynamics/polygolem/pkg/bridge"
)

type fakeReader struct {
	called bool
}

func (f *fakeReader) GetSupportedAssets(context.Context) (*bridge.SupportedAssetsResponse, error) {
	f.called = true
	return &bridge.SupportedAssetsResponse{SupportedAssets: []bridge.SupportedAsset{{
		ChainID:   "137",
		ChainName: "Polygon",
		Token: bridge.TokenInfo{
			Name:     "USD Coin",
			Symbol:   "USDC",
			Address:  "0xusdc",
			Decimals: 6,
		},
		MinCheckoutUsd: 1,
	}}}, nil
}

func TestRunnerReturnsSupportedAssets(t *testing.T) {
	reader := &fakeReader{}
	runner := New(reader)

	got, err := runner.Run(context.Background())
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !reader.called {
		t.Fatal("supported assets reader was not called")
	}
	want := &bridge.SupportedAssetsResponse{SupportedAssets: []bridge.SupportedAsset{{
		ChainID:   "137",
		ChainName: "Polygon",
		Token: bridge.TokenInfo{
			Name:     "USD Coin",
			Symbol:   "USDC",
			Address:  "0xusdc",
			Decimals: 6,
		},
		MinCheckoutUsd: 1,
	}}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("assets=%+v, want %+v", got, want)
	}
}
