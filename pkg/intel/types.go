// Package intel provides read-only wallet intelligence DTOs and pure scoring
// helpers for Polymarket research surfaces.
package intel

import "time"

const (
	FormulaWalletScoreV1      = "wallet_score_v1"
	FormulaShrinkageWinRateV1 = "shrinkage_win_rate_v1"

	ConfidenceNone   = "none"
	ConfidenceLow    = "low"
	ConfidenceMedium = "medium"
	ConfidenceHigh   = "high"

	DossierStatusComplete   = "complete"
	DossierStatusPartial    = "partial"
	DossierStatusConflicted = "conflicted"
)

// WalletSummary is a source-backed summary of a Polymarket wallet's public
// activity. Values are read-only research facts; they are not trading advice.
type WalletSummary struct {
	Wallet            string    `json:"wallet"`
	Volume            float64   `json:"volume"`
	RealizedPnL       float64   `json:"realized_pnl"`
	ROI               float64   `json:"roi"`
	Bets              int       `json:"bets"`
	Wins              int       `json:"wins"`
	RawWinRate        float64   `json:"raw_win_rate"`
	ShrinkageWinRate  float64   `json:"shrinkage_win_rate"`
	LastActive        time.Time `json:"last_active,omitempty"`
	AsOf              time.Time `json:"as_of,omitempty"`
	FormulaVersion    string    `json:"formula_version"`
	SourceRows        int       `json:"source_rows"`
	SourceDescription string    `json:"source_description,omitempty"`
}

// WalletScore explains why a wallet was flagged as an intelligence candidate.
// A score is a statistical signal, not a finding of misconduct.
type WalletScore struct {
	Wallet         string             `json:"wallet"`
	Value          int                `json:"value"`
	Confidence     string             `json:"confidence"`
	FormulaVersion string             `json:"formula_version"`
	AsOf           time.Time          `json:"as_of,omitempty"`
	SourceRows     int                `json:"source_rows"`
	Reasons        []string           `json:"reasons"`
	RawMetrics     WalletScoreMetrics `json:"raw_metrics"`
	Language       string             `json:"language"`
}

// WalletScoreMetrics records the raw inputs and deterministic derived values
// used by ScoreWallet so callers can reproduce the score.
type WalletScoreMetrics struct {
	Wins                int     `json:"wins"`
	Bets                int     `json:"bets"`
	Volume              float64 `json:"volume"`
	RealizedPnL         float64 `json:"realized_pnl"`
	ROI                 float64 `json:"roi"`
	RawWinRate          float64 `json:"raw_win_rate"`
	ShrinkageWinRate    float64 `json:"shrinkage_win_rate"`
	CategoryEdge        float64 `json:"category_edge,omitempty"`
	ConcentrationSignal bool    `json:"concentration_signal,omitempty"`
	LateEntrySignal     bool    `json:"late_entry_signal,omitempty"`
	CoPositioningSignal bool    `json:"co_positioning_signal,omitempty"`
	ShrinkagePriorWins  float64 `json:"shrinkage_prior_wins"`
	ShrinkagePriorBets  float64 `json:"shrinkage_prior_bets"`
}

// WalletDossier is the future SDK shape for a wallet research dossier. Slice 1
// keeps it as a DTO; network-backed assembly belongs in internal/intel.
type WalletDossier struct {
	Wallet    string           `json:"wallet"`
	AsOf      time.Time        `json:"as_of,omitempty"`
	Status    string           `json:"status"`
	Summary   WalletSummary    `json:"summary"`
	Score     WalletScore      `json:"score"`
	Sources   []SourceRef      `json:"sources,omitempty"`
	Conflicts []SourceConflict `json:"conflicts,omitempty"`
	Warnings  []string         `json:"warnings,omitempty"`
}

// LeaderboardRow is a ranked wallet row suitable for CLI table or JSON output.
type LeaderboardRow struct {
	Rank    int           `json:"rank"`
	Wallet  string        `json:"wallet"`
	Summary WalletSummary `json:"summary"`
	Score   WalletScore   `json:"score"`
}

// DossierAlerts is the CLI/API payload shape for user-scoped dossier alerts.
type DossierAlerts struct {
	DossierAlerts []Signal `json:"dossier_alerts"`
}

// Signal is an explainable alert candidate. It intentionally uses candidate /
// potential language rather than claiming wrongdoing.
type Signal struct {
	Score          int         `json:"score"`
	Wallet         string      `json:"wallet"`
	Market         string      `json:"market,omitempty"`
	Side           string      `json:"side,omitempty"`
	Size           float64     `json:"size,omitempty"`
	Price          float64     `json:"price,omitempty"`
	Confidence     string      `json:"confidence"`
	FormulaVersion string      `json:"formula_version"`
	AsOf           time.Time   `json:"as_of,omitempty"`
	Language       string      `json:"language"`
	Reasons        []string    `json:"reasons"`
	Sources        []SourceRef `json:"sources,omitempty"`
}

// MarketFlow summarizes read-only market activity for research views.
type MarketFlow struct {
	Market          string      `json:"market"`
	AsOf            time.Time   `json:"as_of,omitempty"`
	FormulaVersion  string      `json:"formula_version,omitempty"`
	OpenInterest    float64     `json:"open_interest,omitempty"`
	HolderCount     int         `json:"holder_count"`
	HolderShares    float64     `json:"holder_shares,omitempty"`
	HolderVolume    float64     `json:"holder_volume,omitempty"`
	TradeCount      int         `json:"trade_count"`
	TradeNotional   float64     `json:"trade_notional,omitempty"`
	CandidateSignal bool        `json:"candidate_signal,omitempty"`
	Sources         []SourceRef `json:"sources,omitempty"`
	Warnings        []string    `json:"warnings,omitempty"`
}

// SourceRef records source provenance for reproducible intelligence output.
type SourceRef struct {
	Kind string `json:"kind"`
	Rows int    `json:"rows,omitempty"`
	AsOf string `json:"as_of,omitempty"`
}

// SourceConflict describes a reproducibility issue between Polygolem source adapters.
type SourceConflict struct {
	Field   string `json:"field"`
	Primary string `json:"primary"`
	Other   string `json:"other"`
	Reason  string `json:"reason"`
}

// ScoreInput is the pure input contract for ScoreWallet.
type ScoreInput struct {
	Wallet              string
	Wins                int
	Bets                int
	Volume              float64
	RealizedPnL         float64
	CategoryEdge        float64
	ConcentrationSignal bool
	LateEntrySignal     bool
	CoPositioningSignal bool
	PriorWins           float64
	PriorBets           float64
	AsOf                time.Time
	SourceRows          int
}
