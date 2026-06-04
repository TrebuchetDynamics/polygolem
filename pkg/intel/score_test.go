package intel

import (
	"math"
	"strings"
	"testing"
	"time"
)

func TestShrinkageWinRate(t *testing.T) {
	tests := []struct {
		name      string
		wins      int
		bets      int
		priorWins float64
		priorBets float64
		want      float64
	}{
		{
			name:      "zero bets returns prior",
			priorWins: 10,
			priorBets: 20,
			want:      0.5,
		},
		{
			name:      "small perfect record is pulled toward prior",
			wins:      2,
			bets:      2,
			priorWins: 10,
			priorBets: 20,
			want:      12.0 / 22.0,
		},
		{
			name:      "large winning record moves the rate",
			wins:      95,
			bets:      150,
			priorWins: 10,
			priorBets: 20,
			want:      105.0 / 170.0,
		},
		{
			name:      "losing record remains below prior",
			wins:      10,
			bets:      40,
			priorWins: 10,
			priorBets: 20,
			want:      20.0 / 60.0,
		},
		{
			name:      "wins are capped at bets",
			wins:      5,
			bets:      2,
			priorWins: 10,
			priorBets: 20,
			want:      12.0 / 22.0,
		},
		{
			name: "no record and no prior returns zero",
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShrinkageWinRate(tt.wins, tt.bets, tt.priorWins, tt.priorBets)
			if !closeEnough(got, tt.want) {
				t.Fatalf("ShrinkageWinRate()=%v want %v", got, tt.want)
			}
		})
	}
}

func TestROI(t *testing.T) {
	tests := []struct {
		name string
		pnl  float64
		vol  float64
		want float64
	}{
		{name: "positive ROI", pnl: 25, vol: 100, want: 0.25},
		{name: "negative ROI", pnl: -5, vol: 100, want: -0.05},
		{name: "zero volume returns zero", pnl: 25, vol: 0, want: 0},
		{name: "negative volume returns zero", pnl: 25, vol: -10, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ROI(tt.pnl, tt.vol); !closeEnough(got, tt.want) {
				t.Fatalf("ROI()=%v want %v", got, tt.want)
			}
		})
	}
}

func TestScoreWalletProducesDeterministicExplainableScore(t *testing.T) {
	asOf := time.Date(2026, 6, 3, 0, 0, 0, 0, time.UTC)
	score := ScoreWallet(ScoreInput{
		Wallet:              "0xwallet",
		Wins:                95,
		Bets:                150,
		Volume:              10000,
		RealizedPnL:         1200,
		CategoryEdge:        0.09,
		ConcentrationSignal: true,
		LateEntrySignal:     true,
		CoPositioningSignal: true,
		AsOf:                asOf,
		SourceRows:          300,
	})

	if score.Wallet != "0xwallet" {
		t.Fatalf("wallet=%q", score.Wallet)
	}
	if score.Value != 93 {
		t.Fatalf("score=%d want 93: %+v", score.Value, score)
	}
	if score.Confidence != ConfidenceHigh {
		t.Fatalf("confidence=%q", score.Confidence)
	}
	if score.FormulaVersion != FormulaWalletScoreV1 {
		t.Fatalf("formula=%q", score.FormulaVersion)
	}
	if !score.AsOf.Equal(asOf) || score.SourceRows != 300 {
		t.Fatalf("source metadata asOf=%s rows=%d", score.AsOf, score.SourceRows)
	}
	if !closeEnough(score.RawMetrics.ROI, 0.12) {
		t.Fatalf("roi=%v", score.RawMetrics.ROI)
	}
	if !closeEnough(score.RawMetrics.ShrinkageWinRate, 105.0/170.0) {
		t.Fatalf("shrinkage=%v", score.RawMetrics.ShrinkageWinRate)
	}
	joined := strings.Join(score.Reasons, " | ")
	for _, want := range []string{
		"large enough sample",
		"shrinkage-adjusted win rate is above prior",
		"positive realized PnL",
		"ROI is strongly positive",
		"category edge is elevated",
		"potential coordination signal",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing reason %q in %q", want, joined)
		}
	}
	if strings.Contains(strings.ToLower(score.Language), "finding of misconduct") && !strings.Contains(strings.ToLower(score.Language), "not a finding") {
		t.Fatalf("unsafe language=%q", score.Language)
	}
}

func TestScoreWalletDiscountsSmallSamples(t *testing.T) {
	small := ScoreWallet(ScoreInput{
		Wallet:      "0xsmall",
		Wins:        2,
		Bets:        2,
		Volume:      100,
		RealizedPnL: 10,
	})
	durable := ScoreWallet(ScoreInput{
		Wallet:      "0xdurable",
		Wins:        95,
		Bets:        150,
		Volume:      10000,
		RealizedPnL: 1200,
	})

	if small.Confidence != ConfidenceLow {
		t.Fatalf("small confidence=%q", small.Confidence)
	}
	if small.Value != 45 {
		t.Fatalf("small score=%d want 45", small.Value)
	}
	if small.Value >= durable.Value {
		t.Fatalf("small sample should not outrank durable record: small=%d durable=%d", small.Value, durable.Value)
	}
}

func TestScoreWalletHandlesEmptyAndLosingRecords(t *testing.T) {
	empty := ScoreWallet(ScoreInput{Wallet: "0xempty"})
	if empty.Value != 0 || empty.Confidence != ConfidenceNone || len(empty.Reasons) != 0 {
		t.Fatalf("empty score=%+v", empty)
	}

	losing := ScoreWallet(ScoreInput{
		Wallet:      "0xlosing",
		Wins:        10,
		Bets:        40,
		Volume:      1000,
		RealizedPnL: -100,
	})
	if losing.Value != 12 {
		t.Fatalf("losing score=%d want sample-only 12", losing.Value)
	}
	if losing.Confidence != ConfidenceMedium {
		t.Fatalf("losing confidence=%q", losing.Confidence)
	}
}

func closeEnough(a, b float64) bool {
	return math.Abs(a-b) < 0.000000001
}
