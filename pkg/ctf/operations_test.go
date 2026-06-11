package ctf

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestBuildSplitAndMergeCallsTargetConditionalTokens(t *testing.T) {
	req := validOperationRequest(OperationSplit)
	split, err := BuildSplitCall(req)
	if err != nil {
		t.Fatalf("split: %v", err)
	}
	if split.Target != ConditionalTokens.Hex() || split.Value != "0" || !strings.HasPrefix(split.Data, "0x") {
		t.Fatalf("bad split call: %+v", split)
	}

	req.Operation = OperationMerge
	merge, err := BuildMergeCall(req)
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	if merge.Target != ConditionalTokens.Hex() || merge.Value != "0" || !strings.HasPrefix(merge.Data, "0x") {
		t.Fatalf("bad merge call: %+v", merge)
	}
	if split.Data == merge.Data {
		t.Fatal("split and merge calldata should differ by selector")
	}
}

func TestBuildOperationDryRunWarnsAndDoesNotSubmit(t *testing.T) {
	dryRun, err := BuildOperationDryRun(validOperationRequest(OperationSplit))
	if err != nil {
		t.Fatal(err)
	}
	if dryRun.ReadyToSubmit {
		t.Fatal("dry run must not be ready to submit")
	}
	if len(dryRun.SafetyWarnings) == 0 {
		t.Fatal("expected safety warnings")
	}
	if dryRun.Call.Target != ConditionalTokens.Hex() {
		t.Fatalf("target=%s", dryRun.Call.Target)
	}
}

func TestBuildOperationSubmitPlanRequiresReadiness(t *testing.T) {
	_, err := BuildOperationSubmitPlan(validOperationRequest(OperationSplit), ReadinessGate{Status: "missing_adapter_approval"})
	if err == nil {
		t.Fatal("expected readiness error")
	}

	plan, err := BuildOperationSubmitPlan(validOperationRequest(OperationSplit), ReadinessGate{
		Ready:             true,
		Status:            "ready",
		DepositWallet:     "0xwallet",
		RelayerConfigured: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.ReadyToSubmit {
		t.Fatal("submit plan should be marked ready after readiness proof")
	}
	if plan.Readiness.DepositWallet != "0xwallet" {
		t.Fatalf("readiness not preserved: %+v", plan.Readiness)
	}
}

func TestBuildOperationCallValidatesRequiredFields(t *testing.T) {
	cases := []OperationRequest{
		{},
		{Operation: OperationSplit, CollateralToken: USDC.Hex(), ConditionID: common.Hash{}.Hex(), AmountBaseUnits: "1"},
		{Operation: OperationSplit, CollateralToken: USDC.Hex(), ConditionID: common.Hash{}.Hex(), Partition: []string{"1"}, AmountBaseUnits: "0"},
		{Operation: "burn", CollateralToken: USDC.Hex(), ConditionID: common.Hash{}.Hex(), Partition: []string{"1"}, AmountBaseUnits: "1"},
		{Operation: OperationSplit, CollateralToken: "not-address", ConditionID: common.Hash{}.Hex(), Partition: []string{"1"}, AmountBaseUnits: "1"},
		{Operation: OperationSplit, CollateralToken: USDC.Hex(), ConditionID: "0x1234", Partition: []string{"1"}, AmountBaseUnits: "1"},
		{Operation: OperationSplit, CollateralToken: USDC.Hex(), ParentCollectionID: "0x1234", ConditionID: common.Hash{}.Hex(), Partition: []string{"1"}, AmountBaseUnits: "1"},
	}
	for _, req := range cases {
		if _, err := BuildOperationDryRun(req); err == nil {
			t.Fatalf("expected validation error for %+v", req)
		}
	}
}

func validOperationRequest(operation string) OperationRequest {
	return OperationRequest{
		Operation:       operation,
		CollateralToken: USDC.Hex(),
		ConditionID:     "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
		Partition:       []string{"1", "2"},
		AmountBaseUnits: "1000000",
	}
}
