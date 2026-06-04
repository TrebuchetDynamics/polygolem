package intel

const (
	defaultPriorWins = 10.0
	defaultPriorBets = 20.0

	candidateLanguage = "statistical candidate signal; not a finding of misconduct"
)

// ShrinkageWinRate returns a win rate pulled toward a prior so tiny samples do
// not outrank wallets with durable records. Invalid negative inputs are treated
// as zero and wins are capped at bets.
func ShrinkageWinRate(wins, bets int, priorWins, priorBets float64) float64 {
	if bets < 0 {
		bets = 0
	}
	if wins < 0 {
		wins = 0
	}
	if wins > bets {
		wins = bets
	}
	if priorWins < 0 {
		priorWins = 0
	}
	if priorBets < 0 {
		priorBets = 0
	}
	if priorWins > priorBets && priorBets > 0 {
		priorWins = priorBets
	}
	denominator := float64(bets) + priorBets
	if denominator == 0 {
		return 0
	}
	return (float64(wins) + priorWins) / denominator
}

// ROI returns realized PnL divided by traded volume. Zero or negative volume is
// not meaningful for ROI and returns zero instead of infinity or NaN.
func ROI(realizedPnL, volume float64) float64 {
	if volume <= 0 {
		return 0
	}
	return realizedPnL / volume
}

// ScoreWallet converts public wallet metrics into a deterministic explainable
// research score. It is pure: no network, auth, signing, or mutation.
func ScoreWallet(input ScoreInput) WalletScore {
	priorWins, priorBets := normalizePrior(input.PriorWins, input.PriorBets)
	wins, bets := normalizeRecord(input.Wins, input.Bets)
	rawWinRate := 0.0
	if bets > 0 {
		rawWinRate = float64(wins) / float64(bets)
	}
	roi := ROI(input.RealizedPnL, input.Volume)
	shrinkage := ShrinkageWinRate(wins, bets, priorWins, priorBets)

	score := 0
	reasons := make([]string, 0, 6)

	sampleScore, sampleReason := sampleScore(bets)
	score += sampleScore
	if sampleReason != "" {
		reasons = append(reasons, sampleReason)
	}

	swrScore, swrReason := shrinkageScore(shrinkage)
	score += swrScore
	if swrReason != "" {
		reasons = append(reasons, swrReason)
	}

	if input.RealizedPnL > 0 {
		score += 15
		reasons = append(reasons, "positive realized PnL")
	}

	roiScore, roiReason := roiScore(roi)
	score += roiScore
	if roiReason != "" {
		reasons = append(reasons, roiReason)
	}

	categoryScore, categoryReason := categoryEdgeScore(input.CategoryEdge)
	score += categoryScore
	if categoryReason != "" {
		reasons = append(reasons, categoryReason)
	}

	if input.ConcentrationSignal {
		score += 5
		reasons = append(reasons, "concentrated exposure requires review")
	}
	if input.LateEntrySignal {
		score += 5
		reasons = append(reasons, "late market entry requires review")
	}
	if input.CoPositioningSignal {
		score += 5
		reasons = append(reasons, "repeat co-positioning suggests potential coordination signal")
	}

	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}

	return WalletScore{
		Wallet:         input.Wallet,
		Value:          score,
		Confidence:     confidenceForBets(bets),
		FormulaVersion: FormulaWalletScoreV1,
		AsOf:           input.AsOf,
		SourceRows:     input.SourceRows,
		Reasons:        reasons,
		Language:       candidateLanguage,
		RawMetrics: WalletScoreMetrics{
			Wins:                wins,
			Bets:                bets,
			Volume:              input.Volume,
			RealizedPnL:         input.RealizedPnL,
			ROI:                 roi,
			RawWinRate:          rawWinRate,
			ShrinkageWinRate:    shrinkage,
			CategoryEdge:        input.CategoryEdge,
			ConcentrationSignal: input.ConcentrationSignal,
			LateEntrySignal:     input.LateEntrySignal,
			CoPositioningSignal: input.CoPositioningSignal,
			ShrinkagePriorWins:  priorWins,
			ShrinkagePriorBets:  priorBets,
		},
	}
}

func normalizePrior(priorWins, priorBets float64) (float64, float64) {
	if priorWins <= 0 && priorBets <= 0 {
		return defaultPriorWins, defaultPriorBets
	}
	if priorWins < 0 {
		priorWins = 0
	}
	if priorBets <= 0 {
		priorBets = defaultPriorBets
	}
	if priorWins > priorBets {
		priorWins = priorBets
	}
	return priorWins, priorBets
}

func normalizeRecord(wins, bets int) (int, int) {
	if bets < 0 {
		bets = 0
	}
	if wins < 0 {
		wins = 0
	}
	if wins > bets {
		wins = bets
	}
	return wins, bets
}

func sampleScore(bets int) (int, string) {
	switch {
	case bets >= 100:
		return 20, "large enough sample for high-confidence interpretation"
	case bets >= 30:
		return 12, "moderate sample for interpretation"
	case bets > 0:
		return 5, "small sample; score is heavily discounted"
	default:
		return 0, ""
	}
}

func shrinkageScore(rate float64) (int, string) {
	switch {
	case rate >= 0.65:
		return 25, "shrinkage-adjusted win rate is materially above prior"
	case rate >= 0.58:
		return 18, "shrinkage-adjusted win rate is above prior"
	case rate >= 0.52:
		return 10, "shrinkage-adjusted win rate is slightly above prior"
	default:
		return 0, ""
	}
}

func roiScore(roi float64) (int, string) {
	switch {
	case roi >= 0.10:
		return 15, "ROI is strongly positive"
	case roi >= 0.02:
		return 10, "ROI is positive"
	case roi > 0:
		return 5, "ROI is slightly positive"
	default:
		return 0, ""
	}
}

func categoryEdgeScore(edge float64) (int, string) {
	switch {
	case edge >= 0.08:
		return 10, "category edge is elevated"
	case edge >= 0.03:
		return 5, "category edge is positive"
	default:
		return 0, ""
	}
}

func confidenceForBets(bets int) string {
	switch {
	case bets >= 100:
		return ConfidenceHigh
	case bets >= 30:
		return ConfidenceMedium
	case bets > 0:
		return ConfidenceLow
	default:
		return ConfidenceNone
	}
}
