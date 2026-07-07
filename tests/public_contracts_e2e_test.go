package tests

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/polygolem/pkg/contracts"
	"github.com/TrebuchetDynamics/polygolem/pkg/relayer"
)

func TestPublicContractApprovalBatchesDecodeToContractMetadata(t *testing.T) {
	cases := []struct {
		name      string
		approvals []contracts.Approval
		calls     []relayer.DepositWalletCall
	}{
		{"trading", contracts.TradingApprovals(), relayer.BuildApprovalCalls()},
		{"settlement", contracts.SettlementApprovals(), relayer.BuildAdapterApprovalCalls()},
		{"enable-trading", contracts.EnableTradingApprovals(), relayer.BuildEnableTradingApprovalCalls()},
		{"auto-redeem", contracts.AutoRedeemApprovals(), relayer.BuildAutoRedeemApprovalCalls()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if len(tc.calls) != len(tc.approvals) {
				t.Fatalf("calls=%d approvals=%d", len(tc.calls), len(tc.approvals))
			}
			for i, approval := range tc.approvals {
				call := tc.calls[i]
				if !strings.EqualFold(call.Target, approval.Token) {
					t.Fatalf("call %d target=%s want token %s", i, call.Target, approval.Token)
				}
				if call.Value != "0" {
					t.Fatalf("call %d value=%s want 0", i, call.Value)
				}
				decoded := decodeApprovalCall(t, call.Data)
				if decoded.kind != approval.Kind || !strings.EqualFold(decoded.spender, approval.Spender) {
					t.Fatalf("call %d decoded=(%s,%s) want (%s,%s)", i, decoded.kind, decoded.spender, approval.Kind, approval.Spender)
				}
			}
		})
	}
}

type decodedApprovalCall struct {
	kind    string
	spender string
}

func decodeApprovalCall(t *testing.T, data string) decodedApprovalCall {
	t.Helper()
	raw, err := hex.DecodeString(strings.TrimPrefix(data, "0x"))
	if err != nil {
		t.Fatalf("decode calldata %q: %v", data, err)
	}
	if len(raw) != 4+32+32 {
		t.Fatalf("calldata len=%d want 68 bytes", len(raw))
	}
	spender := "0x" + hex.EncodeToString(raw[4+12:4+32])
	tail := hex.EncodeToString(raw[36:68])
	switch selector := hex.EncodeToString(raw[:4]); selector {
	case "095ea7b3":
		if tail != strings.Repeat("f", 64) {
			t.Fatalf("approve amount=%s want MaxUint256", tail)
		}
		return decodedApprovalCall{kind: contracts.ApprovalERC20Approve, spender: spender}
	case "a22cb465":
		if tail != strings.Repeat("0", 63)+"1" {
			t.Fatalf("setApprovalForAll approved word=%s want true", tail)
		}
		return decodedApprovalCall{kind: contracts.ApprovalERC1155ForAll, spender: spender}
	default:
		t.Fatalf("unknown approval selector %s", selector)
		return decodedApprovalCall{}
	}
}
