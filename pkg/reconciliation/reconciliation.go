// Package reconciliation defines a read-only operator report for comparing
// Polymarket order, position, relayer, and fill evidence.
package reconciliation

import "strings"

type Status string

const (
	Submitted    Status = "submitted"
	Matched      Status = "matched"
	Mined        Status = "mined"
	PositionSeen Status = "position_seen"
	FillSeen     Status = "fill_seen"
	Conflicted   Status = "conflicted"
	Unresolved   Status = "unresolved"
)

type Input struct {
	Order     *OrderEvidence    `json:"order,omitempty"`
	Position  *PositionEvidence `json:"position,omitempty"`
	RelayerTx *RelayerEvidence  `json:"relayerTx,omitempty"`
	Fill      *FillEvidence     `json:"fill,omitempty"`
}

type OrderEvidence struct {
	ID          string `json:"id,omitempty"`
	TokenID     string `json:"tokenId,omitempty"`
	Status      string `json:"status,omitempty"`
	SizeMatched string `json:"sizeMatched,omitempty"`
}

type PositionEvidence struct {
	TokenID string `json:"tokenId,omitempty"`
	Size    string `json:"size,omitempty"`
}

type RelayerEvidence struct {
	TransactionID string `json:"transactionId,omitempty"`
	State         string `json:"state,omitempty"`
}

type FillEvidence struct {
	TxHash  string `json:"txHash,omitempty"`
	TokenID string `json:"tokenId,omitempty"`
	Size    string `json:"size,omitempty"`
}

type Report struct {
	Statuses  []Status `json:"statuses"`
	Conflicts []string `json:"conflicts,omitempty"`
}

func (r Report) Has(status Status) bool {
	for _, got := range r.Statuses {
		if got == status {
			return true
		}
	}
	return false
}

func BuildReport(in Input) Report {
	report := Report{}
	if in.Order != nil && strings.TrimSpace(in.Order.ID) != "" {
		report.add(Submitted)
		if isMatched(in.Order.Status) || strings.TrimSpace(in.Order.SizeMatched) != "" {
			report.add(Matched)
		}
	}
	if in.RelayerTx != nil && isMined(in.RelayerTx.State) {
		report.add(Mined)
	}
	if in.Position != nil && strings.TrimSpace(in.Position.TokenID) != "" {
		report.add(PositionSeen)
	}
	if in.Fill != nil && strings.TrimSpace(in.Fill.TxHash) != "" {
		report.add(FillSeen)
	}
	report.checkConflicts(in)
	if len(report.Statuses) == 0 || (!report.Has(PositionSeen) && !report.Has(FillSeen) && !report.Has(Conflicted)) {
		report.add(Unresolved)
	}
	return report
}

func (r *Report) add(status Status) {
	if r.Has(status) {
		return
	}
	r.Statuses = append(r.Statuses, status)
}

func (r *Report) checkConflicts(in Input) {
	orderToken := ""
	if in.Order != nil {
		orderToken = strings.TrimSpace(in.Order.TokenID)
	}
	if orderToken == "" {
		return
	}
	if in.Position != nil && strings.TrimSpace(in.Position.TokenID) != "" && strings.TrimSpace(in.Position.TokenID) != orderToken {
		r.addConflict("position token does not match order token")
	}
	if in.Fill != nil && strings.TrimSpace(in.Fill.TokenID) != "" && strings.TrimSpace(in.Fill.TokenID) != orderToken {
		r.addConflict("fill token does not match order token")
	}
}

func (r *Report) addConflict(message string) {
	r.Conflicts = append(r.Conflicts, message)
	r.add(Conflicted)
}

func isMatched(status string) bool {
	status = strings.ToLower(strings.TrimSpace(status))
	return status == "matched" || status == "filled"
}

func isMined(state string) bool {
	state = strings.ToLower(strings.TrimSpace(state))
	return strings.Contains(state, "mined") || strings.Contains(state, "confirmed") || strings.Contains(state, "executed")
}
