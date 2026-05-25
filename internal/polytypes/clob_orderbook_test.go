package polytypes

import (
	"encoding/json"
	"testing"
)

func TestOrderBookDecodesNumericFieldsAndCamelAliases(t *testing.T) {
	var book OrderBook
	raw := `{"market":"0xmarket","assetId":12345,"timestamp":1710000000,"hash":"0xhash","bids":[{"price":0.42,"size":100}],"asks":[{"price":"0.43","size":200}],"minOrderSize":5,"tickSize":0.01,"negRisk":"true","lastTradePrice":0.42}`
	if err := json.Unmarshal([]byte(raw), &book); err != nil {
		t.Fatal(err)
	}
	if book.AssetID != "12345" || book.Timestamp != "1710000000" || book.MinOrderSize != "5" || book.TickSize != "0.01" || book.LastTradePrice != "0.42" {
		t.Fatalf("stringified fields not decoded: %+v", book)
	}
	if !book.NegRisk || len(book.Bids) != 1 || book.Bids[0].Price != "0.42" || book.Asks[0].Size != "200" {
		t.Fatalf("nested/bool fields not decoded: %+v", book)
	}
}
