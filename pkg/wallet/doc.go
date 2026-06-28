// Package wallet provides public deposit-wallet primitives.
//
// The canonical Polymarket V2 trading wallet is the deterministic deposit
// wallet used for POLY_1271 / signature type 3 orders. Address derivation is
// non-mutating and safe for read-only flows. Deploy, approvals, funding, and
// batch operations live in pkg/relayer, pkg/enabletrading, pkg/funding, and
// pkg/settlement so this package stays a small identity/readiness module.
package wallet
