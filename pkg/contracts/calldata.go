package contracts

import (
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// Calldata builders for the on-chain calls Polymarket trading needs: ERC-20
// approvals/reads, CTF (ERC-1155) operator approvals, and the V2 collateral
// ramp pUSD wrap/unwrap. They return raw calldata bytes so both EOA
// transactions and deposit-wallet batch entries can use them; callers that
// need hex can wrap with hexutil/fmt.

const (
	calldataERC20ABIJSON = `[
	  {"inputs":[{"name":"owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
	  {"inputs":[{"name":"owner","type":"address"},{"name":"spender","type":"address"}],"name":"allowance","outputs":[{"name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
	  {"inputs":[{"name":"spender","type":"address"},{"name":"amount","type":"uint256"}],"name":"approve","outputs":[{"name":"","type":"bool"}],"stateMutability":"nonpayable","type":"function"},
	  {"inputs":[{"name":"to","type":"address"},{"name":"amount","type":"uint256"}],"name":"transfer","outputs":[{"name":"","type":"bool"}],"stateMutability":"nonpayable","type":"function"}
	]`
	calldataERC1155ABIJSON = `[
	  {"inputs":[{"name":"account","type":"address"},{"name":"operator","type":"address"}],"name":"isApprovedForAll","outputs":[{"name":"","type":"bool"}],"stateMutability":"view","type":"function"},
	  {"inputs":[{"name":"operator","type":"address"},{"name":"approved","type":"bool"}],"name":"setApprovalForAll","outputs":[],"stateMutability":"nonpayable","type":"function"}
	]`
	// The V2 collateral on/offramp contracts share the wrap/unwrap shape:
	// wrap(asset, to, amount) on CollateralOnramp converts `asset`
	// (e.g. USDC.e) into pUSD delivered to `to`; unwrap(asset, to, amount)
	// on CollateralOfframp converts pUSD back into `asset`.
	calldataRampABIJSON = `[
	  {"inputs":[{"name":"_asset","type":"address"},{"name":"_to","type":"address"},{"name":"_amount","type":"uint256"}],"name":"wrap","outputs":[],"stateMutability":"nonpayable","type":"function"},
	  {"inputs":[{"name":"_asset","type":"address"},{"name":"_to","type":"address"},{"name":"_amount","type":"uint256"}],"name":"unwrap","outputs":[],"stateMutability":"nonpayable","type":"function"}
	]`
)

var (
	calldataERC20ABI   = mustCalldataABI(calldataERC20ABIJSON)
	calldataERC1155ABI = mustCalldataABI(calldataERC1155ABIJSON)
	calldataRampABI    = mustCalldataABI(calldataRampABIJSON)
)

func mustCalldataABI(raw string) abi.ABI {
	parsed, err := abi.JSON(strings.NewReader(raw))
	if err != nil {
		panic(fmt.Sprintf("contracts: parse embedded ABI: %v", err))
	}
	return parsed
}

// MaxUint256 returns 2^256-1, the conventional unlimited ERC-20 allowance.
func MaxUint256() *big.Int {
	return new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
}

// ERC20ApproveCalldata encodes approve(spender, amount). A nil amount
// approves MaxUint256.
func ERC20ApproveCalldata(spender string, amount *big.Int) ([]byte, error) {
	if amount == nil {
		amount = MaxUint256()
	}
	return calldataERC20ABI.Pack("approve", common.HexToAddress(spender), amount)
}

// ERC20TransferCalldata encodes transfer(to, amount).
func ERC20TransferCalldata(to string, amount *big.Int) ([]byte, error) {
	return calldataERC20ABI.Pack("transfer", common.HexToAddress(to), amount)
}

// ERC20AllowanceCalldata encodes the allowance(owner, spender) view call.
// Decode the result with DecodeUint256Result.
func ERC20AllowanceCalldata(owner, spender string) ([]byte, error) {
	return calldataERC20ABI.Pack("allowance", common.HexToAddress(owner), common.HexToAddress(spender))
}

// ERC20BalanceOfCalldata encodes the balanceOf(owner) view call. Decode the
// result with DecodeUint256Result.
func ERC20BalanceOfCalldata(owner string) ([]byte, error) {
	return calldataERC20ABI.Pack("balanceOf", common.HexToAddress(owner))
}

// ERC1155SetApprovalForAllCalldata encodes setApprovalForAll(operator, approved).
func ERC1155SetApprovalForAllCalldata(operator string, approved bool) ([]byte, error) {
	return calldataERC1155ABI.Pack("setApprovalForAll", common.HexToAddress(operator), approved)
}

// ERC1155IsApprovedForAllCalldata encodes the isApprovedForAll(account,
// operator) view call. Decode the result with DecodeBoolResult.
func ERC1155IsApprovedForAllCalldata(account, operator string) ([]byte, error) {
	return calldataERC1155ABI.Pack("isApprovedForAll", common.HexToAddress(account), common.HexToAddress(operator))
}

// RampWrapCalldata encodes CollateralOnramp.wrap(asset, to, amount): convert
// `amount` of `asset` (e.g. USDC.e) into pUSD delivered to `to`. The sender
// must have approved the onramp to spend `asset` first.
func RampWrapCalldata(asset, to string, amount *big.Int) ([]byte, error) {
	if amount == nil || amount.Sign() <= 0 {
		return nil, errors.New("contracts: wrap amount must be positive")
	}
	return calldataRampABI.Pack("wrap", common.HexToAddress(asset), common.HexToAddress(to), amount)
}

// RampUnwrapCalldata encodes CollateralOfframp.unwrap(asset, to, amount):
// convert `amount` of pUSD back into `asset` (e.g. USDC.e) delivered to `to`.
// The sender must have approved the offramp to spend pUSD first.
func RampUnwrapCalldata(asset, to string, amount *big.Int) ([]byte, error) {
	if amount == nil || amount.Sign() <= 0 {
		return nil, errors.New("contracts: unwrap amount must be positive")
	}
	return calldataRampABI.Pack("unwrap", common.HexToAddress(asset), common.HexToAddress(to), amount)
}

// DecodeUint256Result decodes a single-word uint256 return value (balanceOf,
// allowance).
func DecodeUint256Result(out []byte) (*big.Int, error) {
	if len(out) != 32 {
		return nil, fmt.Errorf("contracts: expected 32-byte uint256 return, got %d bytes", len(out))
	}
	return new(big.Int).SetBytes(out), nil
}

// DecodeBoolResult decodes a single-word bool return value (isApprovedForAll).
func DecodeBoolResult(out []byte) (bool, error) {
	if len(out) != 32 {
		return false, fmt.Errorf("contracts: expected 32-byte bool return, got %d bytes", len(out))
	}
	word := new(big.Int).SetBytes(out)
	switch {
	case word.Sign() == 0:
		return false, nil
	case word.Cmp(big.NewInt(1)) == 0:
		return true, nil
	default:
		return false, fmt.Errorf("contracts: bool return word is neither 0 nor 1")
	}
}
