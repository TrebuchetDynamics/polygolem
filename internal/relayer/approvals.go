package relayer

import (
	"encoding/json"
	"strings"

	"github.com/TrebuchetDynamics/polygolem/pkg/contracts"
	"github.com/ethereum/go-ethereum/common"
)

const (
	erc20ApproveSelector        = "095ea7b3"
	erc1155SetApprovalForAllSel = "a22cb465"
	maxUint256                  = "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
)

func pad32Bytes(hexAddr string) string {
	hexAddr = strings.TrimPrefix(strings.TrimSpace(hexAddr), "0x")
	// A 32-byte EVM word holds at most 64 hex chars. Keep the low-order 32
	// bytes for over-long input (the correct truncation for a uint256) rather
	// than panicking on strings.Repeat with a negative count.
	if len(hexAddr) > 64 {
		hexAddr = hexAddr[len(hexAddr)-64:]
	}
	return strings.Repeat("0", 64-len(hexAddr)) + hexAddr
}

func buildApproveCall(tokenAddress, spenderAddress string) DepositWalletCall {
	token := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(tokenAddress), "0x"))
	spender := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(spenderAddress), "0x"))
	data := "0x" + erc20ApproveSelector + pad32Bytes(spender) + maxUint256
	return DepositWalletCall{
		Target: common.HexToAddress(token).Hex(),
		Value:  "0",
		Data:   data,
	}
}

func buildCTFApprovalCall(operatorAddress string) DepositWalletCall {
	operator := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(operatorAddress), "0x"))
	data := "0x" + erc1155SetApprovalForAllSel + pad32Bytes(operator) +
		"0000000000000000000000000000000000000000000000000000000000000001"
	return DepositWalletCall{
		Target: common.HexToAddress(contracts.CTF).Hex(),
		Value:  "0",
		Data:   data,
	}
}

func buildTransferCall(tokenAddress, toAddress, amountHex string) DepositWalletCall {
	token := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(tokenAddress), "0x"))
	to := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(toAddress), "0x"))
	amount := strings.TrimPrefix(strings.TrimSpace(amountHex), "0x")
	data := "0xa9059cbb" + pad32Bytes(to) + pad32Bytes(amount)
	return DepositWalletCall{
		Target: common.HexToAddress(token).Hex(),
		Value:  "0",
		Data:   data,
	}
}

// BuildApprovalCalls returns the 6 calls needed to approve pUSD and CTF for
// all V2 exchange spenders. These must be submitted via WALLET batch from
// the deposit wallet.
func BuildApprovalCalls() []DepositWalletCall {
	return buildApprovalCalls(contracts.TradingApprovals())
}

// BuildApprovalCallsJSON returns the approval calls as a JSON-marshalable
// slice for CLI --calls-json. It also returns the raw bytes for validation.
func BuildApprovalCallsJSON() (string, error) {
	calls := BuildApprovalCalls()
	raw, err := marshalCalls(calls)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// BuildAdapterApprovalCalls returns the 4 calls a deposit wallet must
// submit before V2 split/merge/redeem can succeed: pUSD approve and
// CTF setApprovalForAll for both the standard and neg-risk V2 collateral
// adapters. Idempotent: re-issuing on a wallet that already approved is
// a no-op (max-uint approve sticks; setApprovalForAll(true) sticks).
//
// Required because CtfCollateralAdapter.redeemPositions calls
// safeBatchTransferFrom(msg.sender, address(this), ...) on the CTF.
// Without setApprovalForAll the redeem path reverts.
func BuildAdapterApprovalCalls() []DepositWalletCall {
	return buildApprovalCalls(contracts.SettlementApprovals())
}

// BuildAdapterApprovalCallsJSON mirrors BuildApprovalCallsJSON for the
// adapter approval set so a CLI dry-run path can print the calldata
// without signing.
func BuildAdapterApprovalCallsJSON() (string, error) {
	raw, err := marshalCalls(BuildAdapterApprovalCalls())
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// BuildEnableTradingApprovalCalls returns the two ERC-20 approvals observed
// in polymarket.com's "Enable Trading" UI after deposit-wallet deployment:
// pUSD -> CTF and USDC.e -> CollateralOnramp, both max uint256. This batch is
// distinct from the six-call exchange trading approval set and the four-call
// V2 collateral-adapter approval set.
func BuildEnableTradingApprovalCalls() []DepositWalletCall {
	return buildApprovalCalls(contracts.EnableTradingApprovals())
}

func buildApprovalCalls(approvals []contracts.Approval) []DepositWalletCall {
	calls := make([]DepositWalletCall, 0, len(approvals))
	for _, approval := range approvals {
		switch approval.Kind {
		case contracts.ApprovalERC20Approve:
			calls = append(calls, buildApproveCall(approval.Token, approval.Spender))
		case contracts.ApprovalERC1155ForAll:
			calls = append(calls, buildCTFApprovalCall(approval.Spender))
		}
	}
	return calls
}

// BuildEnableTradingApprovalCallsJSON mirrors BuildApprovalCallsJSON for the
// UI Enable Trading approval set.
func BuildEnableTradingApprovalCallsJSON() (string, error) {
	raw, err := marshalCalls(BuildEnableTradingApprovalCalls())
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func marshalCalls(calls []DepositWalletCall) ([]byte, error) {
	return json.Marshal(calls)
}
