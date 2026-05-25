package polytypes

import (
	"encoding/json"
	"testing"
)

func TestRewardsDTOsDecodeCamelAliasesAndStringNumerics(t *testing.T) {
	var cfg RewardsConfig
	if err := json.Unmarshal([]byte(`{"market":"m1","assetAddress":"0xasset","rewardsMinSize":"100","rewardsMaxSpread":"0.02","active":"true"}`), &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.AssetAddress != "0xasset" || cfg.RewardsMinSize != 100 || cfg.RewardsMaxSpread != 0.02 || !cfg.Active {
		t.Fatalf("cfg=%+v", cfg)
	}

	var raw RawRewards
	if err := json.Unmarshal([]byte(`{"market":"m1","date":"2026-05-07","rewardsPaid":"1.5","volume":"12"}`), &raw); err != nil {
		t.Fatal(err)
	}
	if raw.RewardsPaid != 1.5 || raw.Volume != 12 {
		t.Fatalf("raw=%+v", raw)
	}

	var pct RewardPercentages
	if err := json.Unmarshal([]byte(`{"market":"m1","rewardPercentage":"0.15"}`), &pct); err != nil {
		t.Fatal(err)
	}
	if pct.RewardPercentage != 0.15 {
		t.Fatalf("pct=%+v", pct)
	}

	var byMarket UserRewardsMarket
	if err := json.Unmarshal([]byte(`{"market":"m1","totalRewards":"5.5","rewardPercentage":"0.25"}`), &byMarket); err != nil {
		t.Fatal(err)
	}
	if byMarket.TotalRewards != 5.5 || byMarket.RewardPercentage != 0.25 {
		t.Fatalf("byMarket=%+v", byMarket)
	}

	var rebates RebatedFees
	if err := json.Unmarshal([]byte(`{"makerAddress":"0xmaker","market":"m1","totalRebated":"2","date":"2026-05-07"}`), &rebates); err != nil {
		t.Fatal(err)
	}
	if rebates.MakerAddress != "0xmaker" || rebates.TotalRebated != 2 {
		t.Fatalf("rebates=%+v", rebates)
	}
}
