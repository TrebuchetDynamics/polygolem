package reconciliation

import "testing"

func TestBuildReportClassifiesObservedOrderLifecycle(t *testing.T) {
	report := BuildReport(Input{
		Order:     &OrderEvidence{ID: "order-1", TokenID: "token-1", Status: "MATCHED", SizeMatched: "10"},
		RelayerTx: &RelayerEvidence{TransactionID: "tx-1", State: "STATE_MINED"},
		Position:  &PositionEvidence{TokenID: "token-1", Size: "10"},
		Fill:      &FillEvidence{TxHash: "0xfill", TokenID: "token-1", Size: "10"},
	})

	want := []Status{Submitted, Matched, Mined, PositionSeen, FillSeen}
	for _, status := range want {
		if !report.Has(status) {
			t.Fatalf("report missing %s: %+v", status, report.Statuses)
		}
	}
	if report.Has(Conflicted) || report.Has(Unresolved) {
		t.Fatalf("settled report should not be conflicted/unresolved: %+v", report.Statuses)
	}
}

func TestBuildReportMarksUnresolvedWhenOnlySubmitIsKnown(t *testing.T) {
	report := BuildReport(Input{Order: &OrderEvidence{ID: "order-1", Status: "OPEN"}})
	if !report.Has(Submitted) || !report.Has(Unresolved) {
		t.Fatalf("expected submitted unresolved: %+v", report.Statuses)
	}
}

func TestBuildReportMarksConflictWhenSourcesDisagree(t *testing.T) {
	report := BuildReport(Input{
		Order:    &OrderEvidence{ID: "order-1", TokenID: "token-1", Status: "MATCHED", SizeMatched: "10"},
		Position: &PositionEvidence{TokenID: "other-token", Size: "10"},
		Fill:     &FillEvidence{TxHash: "0xfill", TokenID: "token-1", Size: "10"},
	})
	if !report.Has(Conflicted) {
		t.Fatalf("expected conflict: %+v", report)
	}
	if len(report.Conflicts) == 0 {
		t.Fatalf("expected conflict details: %+v", report)
	}
}

func TestBuildReportIgnoresEmptyInputAsUnresolved(t *testing.T) {
	report := BuildReport(Input{})
	if !report.Has(Unresolved) || len(report.Statuses) != 1 {
		t.Fatalf("empty input should only be unresolved: %+v", report.Statuses)
	}
}
