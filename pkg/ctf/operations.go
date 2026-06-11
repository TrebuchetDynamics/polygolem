package ctf

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"unicode"

	"github.com/TrebuchetDynamics/polygolem/pkg/relayer"
	"github.com/ethereum/go-ethereum/common"
)

const (
	OperationSplit = "split"
	OperationMerge = "merge"
)

// OperationRequest describes a high-level CTF split/merge wallet action.
// AmountBaseUnits must already be scaled to the collateral token decimals.
type OperationRequest struct {
	Operation          string   `json:"operation"`
	CollateralToken    string   `json:"collateralToken"`
	ParentCollectionID string   `json:"parentCollectionId,omitempty"`
	ConditionID        string   `json:"conditionId"`
	Partition          []string `json:"partition"`
	AmountBaseUnits    string   `json:"amountBaseUnits"`
}

// OperationDryRun is the safe preview artifact for a CTF split/merge action.
// It contains the exact deposit-wallet call a caller may later place in a
// relayer WALLET batch after operator approval and allowance checks.
type OperationDryRun struct {
	Request        OperationRequest          `json:"request"`
	Call           relayer.DepositWalletCall `json:"call"`
	ReadyToSubmit  bool                      `json:"readyToSubmit"`
	SafetyWarnings []string                  `json:"safetyWarnings"`
}

// ReadinessGate is the minimal readiness proof required before a caller may
// treat a split/merge dry-run as submit-ready. It intentionally mirrors the
// relevant fields from settlement.Readiness without importing settlement and
// creating a package cycle.
type ReadinessGate struct {
	Ready             bool     `json:"ready"`
	Status            string   `json:"status"`
	DepositWallet     string   `json:"depositWallet"`
	MissingApprovals  []string `json:"missingApprovals,omitempty"`
	RelayerConfigured bool     `json:"relayerConfigured"`
}

// OperationSubmitPlan is still a dry-run artifact; it marks when a caller has
// supplied enough readiness proof to place Call in a relayer batch under their
// own explicit live-confirmation gate.
type OperationSubmitPlan struct {
	OperationDryRun
	Readiness ReadinessGate `json:"readiness"`
}

// BuildSplitCall returns a deposit-wallet call that invokes CTF splitPosition.
func BuildSplitCall(req OperationRequest) (relayer.DepositWalletCall, error) {
	req.Operation = OperationSplit
	return buildOperationCall(req)
}

// BuildMergeCall returns a deposit-wallet call that invokes CTF mergePositions.
func BuildMergeCall(req OperationRequest) (relayer.DepositWalletCall, error) {
	req.Operation = OperationMerge
	return buildOperationCall(req)
}

// BuildOperationDryRun validates req and returns a no-submit preview of the
// encoded split/merge call.
func BuildOperationDryRun(req OperationRequest) (*OperationDryRun, error) {
	call, err := buildOperationCall(req)
	if err != nil {
		return nil, err
	}
	return &OperationDryRun{
		Request:        req,
		Call:           call,
		ReadyToSubmit:  false,
		SafetyWarnings: operationSafetyWarnings(),
	}, nil
}

// BuildOperationSubmitPlan validates req and readiness, then returns a
// submit-ready dry-run artifact. It still does not sign or submit; callers must
// enforce explicit operator confirmation before using the call in a live batch.
func BuildOperationSubmitPlan(req OperationRequest, readiness ReadinessGate) (*OperationSubmitPlan, error) {
	if !readiness.Ready {
		return nil, fmt.Errorf("ctf operation readiness failed: %s", firstNonEmpty(readiness.Status, "not_ready"))
	}
	if strings.TrimSpace(readiness.DepositWallet) == "" {
		return nil, fmt.Errorf("depositWallet readiness proof is required")
	}
	if !readiness.RelayerConfigured {
		return nil, fmt.Errorf("relayer credentials are required for submit-ready CTF operation plan")
	}
	if len(readiness.MissingApprovals) > 0 {
		return nil, fmt.Errorf("missing CTF adapter approvals: %s", strings.Join(readiness.MissingApprovals, ","))
	}
	dryRun, err := BuildOperationDryRun(req)
	if err != nil {
		return nil, err
	}
	dryRun.ReadyToSubmit = true
	return &OperationSubmitPlan{OperationDryRun: *dryRun, Readiness: readiness}, nil
}

func operationSafetyWarnings() []string {
	return []string{
		"CTF split/merge changes on-chain token balances",
		"review calldata, collateral token, condition id, partition, and amount before signing",
		"verify deposit-wallet approvals and relayer allowlist before any future submit path",
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func buildOperationCall(req OperationRequest) (relayer.DepositWalletCall, error) {
	collateral, parent, condition, partition, amount, err := parseOperationRequest(req)
	if err != nil {
		return relayer.DepositWalletCall{}, err
	}
	var data []byte
	switch strings.ToLower(strings.TrimSpace(req.Operation)) {
	case OperationSplit:
		data, err = SplitPositionData(collateral, parent, condition, partition, amount)
	case OperationMerge:
		data, err = MergePositionsData(collateral, parent, condition, partition, amount)
	default:
		return relayer.DepositWalletCall{}, fmt.Errorf("operation must be split or merge")
	}
	if err != nil {
		return relayer.DepositWalletCall{}, err
	}
	return relayer.DepositWalletCall{
		Target: ConditionalTokens.Hex(),
		Value:  "0",
		Data:   "0x" + hex.EncodeToString(data),
	}, nil
}

func parseOperationRequest(req OperationRequest) (common.Address, common.Hash, common.Hash, []*big.Int, *big.Int, error) {
	if strings.TrimSpace(req.CollateralToken) == "" {
		return common.Address{}, common.Hash{}, common.Hash{}, nil, nil, fmt.Errorf("collateralToken is required")
	}
	if !common.IsHexAddress(req.CollateralToken) {
		return common.Address{}, common.Hash{}, common.Hash{}, nil, nil, fmt.Errorf("collateralToken must be a hex address")
	}
	if strings.TrimSpace(req.ConditionID) == "" {
		return common.Address{}, common.Hash{}, common.Hash{}, nil, nil, fmt.Errorf("conditionId is required")
	}
	if !isHexBytes32(req.ConditionID) {
		return common.Address{}, common.Hash{}, common.Hash{}, nil, nil, fmt.Errorf("conditionId must be a 0x-prefixed bytes32 hex string")
	}
	amount, ok := new(big.Int).SetString(strings.TrimSpace(req.AmountBaseUnits), 10)
	if !ok || amount.Sign() <= 0 {
		return common.Address{}, common.Hash{}, common.Hash{}, nil, nil, fmt.Errorf("amountBaseUnits must be a positive base-10 integer")
	}
	if len(req.Partition) == 0 {
		return common.Address{}, common.Hash{}, common.Hash{}, nil, nil, fmt.Errorf("partition is required")
	}
	partition := make([]*big.Int, len(req.Partition))
	for i, raw := range req.Partition {
		value, ok := new(big.Int).SetString(strings.TrimSpace(raw), 10)
		if !ok || value.Sign() <= 0 {
			return common.Address{}, common.Hash{}, common.Hash{}, nil, nil, fmt.Errorf("partition[%d] must be a positive base-10 integer", i)
		}
		partition[i] = value
	}
	parent := common.Hash{}
	if strings.TrimSpace(req.ParentCollectionID) != "" {
		if !isHexBytes32(req.ParentCollectionID) {
			return common.Address{}, common.Hash{}, common.Hash{}, nil, nil, fmt.Errorf("parentCollectionId must be a 0x-prefixed bytes32 hex string")
		}
		parent = common.HexToHash(req.ParentCollectionID)
	}
	return common.HexToAddress(req.CollateralToken), parent, common.HexToHash(req.ConditionID), partition, amount, nil
}

func isHexBytes32(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) != 66 || !strings.HasPrefix(value, "0x") {
		return false
	}
	for _, r := range value[2:] {
		if !unicode.Is(unicode.ASCII_Hex_Digit, r) {
			return false
		}
	}
	return true
}
