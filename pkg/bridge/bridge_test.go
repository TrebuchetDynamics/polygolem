package bridge

import (
	"encoding/json"
	"testing"
)

func TestSupportedAssetDecodesStringNumericFields(t *testing.T) {
	var asset SupportedAsset
	raw := `{"chainId":137,"chainName":123,"token":{"name":456,"symbol":789,"address":101112,"decimals":"6"},"minCheckoutUsd":"5.5"}`
	if err := json.Unmarshal([]byte(raw), &asset); err != nil {
		t.Fatal(err)
	}
	if asset.ChainID != "137" || asset.ChainName != "123" || asset.MinCheckoutUsd != 5.5 {
		t.Fatalf("asset fields not decoded: %+v", asset)
	}
	if asset.Token.Name != "456" || asset.Token.Symbol != "789" || asset.Token.Address != "101112" || asset.Token.Decimals != 6 {
		t.Fatalf("token fields not decoded: %+v", asset.Token)
	}
}

func TestQuoteResponseDecodesStringNumericFields(t *testing.T) {
	var quote QuoteResponse
	raw := `{
		"estCheckoutTimeMs":"120000",
		"estInputUsd":"100.5",
		"estOutputUsd":"99.4",
		"estToTokenBaseUnit":99400000,
		"quoteId":123,
		"estFeeBreakdown":{
			"appFeeLabel":123,
			"appFeePercent":"0.01",
			"appFeeUsd":"0.10",
			"fillCostPercent":"0.02",
			"fillCostUsd":"0.20",
			"gasUsd":"0.30",
			"maxSlippage":"0.005",
			"minReceived":"99.40",
			"swapImpact":"0.001",
			"swapImpactUsd":"0.10",
			"totalImpact":"0.006",
			"totalImpactUsd":"0.60"
		}
	}`
	if err := json.Unmarshal([]byte(raw), &quote); err != nil {
		t.Fatal(err)
	}
	if quote.EstCheckoutTimeMs != 120000 || quote.EstInputUsd != 100.5 || quote.EstOutputUsd != 99.4 {
		t.Fatalf("quote scalars not decoded: %+v", quote)
	}
	if quote.EstToTokenBaseUnit != "99400000" || quote.QuoteID != "123" {
		t.Fatalf("quote string fields not stringified: %+v", quote)
	}
	fees := quote.EstFeeBreakdown
	if fees.AppFeeLabel != "123" || fees.AppFeePercent != 0.01 || fees.MinReceived != 99.40 || fees.TotalImpactUsd != 0.60 {
		t.Fatalf("fee breakdown not decoded: %+v", fees)
	}
}

func TestDepositTransactionDecodesNumericStringDrift(t *testing.T) {
	var tx DepositTransaction
	raw := `{"fromChainId":1,"fromTokenAddress":123,"fromAmountBaseUnit":1000000,"toChainId":137,"toTokenAddress":"0x2791","txHash":456,"createdTimeMs":"1714000000123","status":"confirmed"}`
	if err := json.Unmarshal([]byte(raw), &tx); err != nil {
		t.Fatal(err)
	}
	if tx.FromChainID != "1" || tx.FromTokenAddress != "123" || tx.FromAmountBaseUnit != "1000000" {
		t.Fatalf("from fields not stringified: %+v", tx)
	}
	if tx.ToChainID != "137" || tx.TxHash != "456" || tx.CreatedTimeMs != 1714000000123 {
		t.Fatalf("to/hash/time fields not decoded: %+v", tx)
	}
}
