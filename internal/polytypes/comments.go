package polytypes

import (
	"encoding/json"
	"strconv"

	"github.com/TrebuchetDynamics/polygolem/pkg/types"
)

type Comment = types.Comment
type CommentUser = types.CommentUser
type CommentQuery = types.CommentQuery
type CommentByIDQuery = types.CommentByIDQuery
type CommentsByUserQuery = types.CommentsByUserQuery

// --- Rewards (CLOB) ---

// RewardsConfig represents active rewards configuration.
type RewardsConfig struct {
	Market           string  `json:"market"`
	AssetAddress     string  `json:"asset_address"`
	RewardsMinSize   float64 `json:"rewards_min_size"`
	RewardsMaxSpread float64 `json:"rewards_max_spread"`
	Active           bool    `json:"active"`
}

func (r *RewardsConfig) UnmarshalJSON(b []byte) error {
	var raw struct {
		Market                NumericString   `json:"market"`
		AssetAddress          NumericString   `json:"asset_address"`
		AssetAddressCamel     NumericString   `json:"assetAddress"`
		RewardsMinSize        NumericString   `json:"rewards_min_size"`
		RewardsMinSizeCamel   NumericString   `json:"rewardsMinSize"`
		RewardsMaxSpread      NumericString   `json:"rewards_max_spread"`
		RewardsMaxSpreadCamel NumericString   `json:"rewardsMaxSpread"`
		Active                json.RawMessage `json:"active"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	r.Market = string(raw.Market)
	r.AssetAddress = firstNonEmptyString(string(raw.AssetAddress), string(raw.AssetAddressCamel))
	r.RewardsMinSize = parseRewardFloat(firstNonEmptyString(string(raw.RewardsMinSize), string(raw.RewardsMinSizeCamel)))
	r.RewardsMaxSpread = parseRewardFloat(firstNonEmptyString(string(raw.RewardsMaxSpread), string(raw.RewardsMaxSpreadCamel)))
	r.Active = jsonBoolOrFalse(raw.Active)
	return nil
}

// RawRewards represents raw rewards for a market.
type RawRewards struct {
	Market      string  `json:"market"`
	Date        string  `json:"date"`
	RewardsPaid float64 `json:"rewards_paid"`
	Volume      float64 `json:"volume"`
}

func (r *RawRewards) UnmarshalJSON(b []byte) error {
	var raw struct {
		Market           NumericString `json:"market"`
		Date             NumericString `json:"date"`
		RewardsPaid      NumericString `json:"rewards_paid"`
		RewardsPaidCamel NumericString `json:"rewardsPaid"`
		Volume           NumericString `json:"volume"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	r.Market = string(raw.Market)
	r.Date = string(raw.Date)
	r.RewardsPaid = parseRewardFloat(firstNonEmptyString(string(raw.RewardsPaid), string(raw.RewardsPaidCamel)))
	r.Volume = parseRewardFloat(string(raw.Volume))
	return nil
}

// UserEarnings represents earnings for a user.
type UserEarnings struct {
	Date     string  `json:"date"`
	Earnings float64 `json:"earnings"`
	Market   string  `json:"market,omitempty"`
}

func (u *UserEarnings) UnmarshalJSON(b []byte) error {
	var raw struct {
		Date     NumericString `json:"date"`
		Earnings NumericString `json:"earnings"`
		Market   NumericString `json:"market"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	u.Date = string(raw.Date)
	u.Earnings = parseRewardFloat(string(raw.Earnings))
	u.Market = string(raw.Market)
	return nil
}

// TotalEarnings represents total earnings.
type TotalEarnings struct {
	Date     string  `json:"date"`
	Earnings float64 `json:"earnings"`
}

func (t *TotalEarnings) UnmarshalJSON(b []byte) error {
	var raw struct {
		Date     NumericString `json:"date"`
		Earnings NumericString `json:"earnings"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	t.Date = string(raw.Date)
	t.Earnings = parseRewardFloat(string(raw.Earnings))
	return nil
}

// RewardPercentages represents reward percentages.
type RewardPercentages struct {
	Market           string  `json:"market"`
	RewardPercentage float64 `json:"reward_percentage"`
}

func (r *RewardPercentages) UnmarshalJSON(b []byte) error {
	var raw struct {
		Market                NumericString `json:"market"`
		RewardPercentage      NumericString `json:"reward_percentage"`
		RewardPercentageCamel NumericString `json:"rewardPercentage"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	r.Market = string(raw.Market)
	r.RewardPercentage = parseRewardFloat(firstNonEmptyString(string(raw.RewardPercentage), string(raw.RewardPercentageCamel)))
	return nil
}

// UserRewardsMarket represents user rewards by market.
type UserRewardsMarket struct {
	Market           string  `json:"market"`
	TotalRewards     float64 `json:"total_rewards"`
	RewardPercentage float64 `json:"reward_percentage"`
}

func (u *UserRewardsMarket) UnmarshalJSON(b []byte) error {
	var raw struct {
		Market                NumericString `json:"market"`
		TotalRewards          NumericString `json:"total_rewards"`
		TotalRewardsCamel     NumericString `json:"totalRewards"`
		RewardPercentage      NumericString `json:"reward_percentage"`
		RewardPercentageCamel NumericString `json:"rewardPercentage"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	u.Market = string(raw.Market)
	u.TotalRewards = parseRewardFloat(firstNonEmptyString(string(raw.TotalRewards), string(raw.TotalRewardsCamel)))
	u.RewardPercentage = parseRewardFloat(firstNonEmptyString(string(raw.RewardPercentage), string(raw.RewardPercentageCamel)))
	return nil
}

// UserRewardsByMarketRequest represents query params.
type UserRewardsByMarketRequest struct {
	Date          string `json:"date,omitempty"`
	OrderBy       string `json:"order_by,omitempty"`
	NoCompetition bool   `json:"no_competition,omitempty"`
}

// RebatedFees represents current rebated fees for a maker.
type RebatedFees struct {
	MakerAddress string  `json:"maker_address"`
	Market       string  `json:"market,omitempty"`
	TotalRebated float64 `json:"total_rebated"`
	Date         string  `json:"date"`
}

func (r *RebatedFees) UnmarshalJSON(b []byte) error {
	var raw struct {
		MakerAddress      NumericString `json:"maker_address"`
		MakerAddressCamel NumericString `json:"makerAddress"`
		Market            NumericString `json:"market"`
		TotalRebated      NumericString `json:"total_rebated"`
		TotalRebatedCamel NumericString `json:"totalRebated"`
		Date              NumericString `json:"date"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	r.MakerAddress = firstNonEmptyString(string(raw.MakerAddress), string(raw.MakerAddressCamel))
	r.Market = string(raw.Market)
	r.TotalRebated = parseRewardFloat(firstNonEmptyString(string(raw.TotalRebated), string(raw.TotalRebatedCamel)))
	r.Date = string(raw.Date)
	return nil
}

func parseRewardFloat(value string) float64 {
	out, _ := strconv.ParseFloat(value, 64)
	return out
}

type SportsMarketType = types.SportsMarketType
type KeysetParams = types.KeysetParams
type KeysetResponse[T any] = types.KeysetResponse[T]
type MarketByTokenResponse = types.MarketByTokenResponse
type PolymarketCategory = types.PolymarketCategory
type CategoryEventsParams = types.CategoryEventsParams
type CategoryEventsResponse = types.CategoryEventsResponse
