# POLY_1271 Signing Chain — sigtype 3 Full Flow

> **Status:** Implementation contract for polygolem's CLOB V2 deposit-wallet order path.
> **Last updated:** 2026-06-10
> **Companion:** [ONBOARDING.md](./ONBOARDING.md), [CONTRACTS.md](./CONTRACTS.md)
>
> **Critical auth rule:** CLOB L1/L2 auth is **EOA-bound** in polygolem's validated V2 path. `/auth/api-key` and `/auth/derive-api-key` use the EOA address in `POLY_ADDRESS` and a standard 65-byte EOA ECDSA `POLY_SIGNATURE`. Deposit-wallet identity is carried by the order payload (`maker`, `signer`, `signatureType=3`) and the ERC-7739-wrapped order signature, not by deposit-wallet-bound ClobAuth headers.

---

## The Full Chain

For sigtype 3 (POLY_1271 / deposit wallet) to work end-to-end, four conditions must be satisfied:

```
┌─────────────────────────────────────────────────────────────────────────┐
│                     SIGTYPE 3 — FULL CHAIN                              │
│                                                                         │
│  Step 1: EOA-Bound CLOB API Key                                        │
│  ┌──────────────────────────────────────────────────────────────┐      │
│  │ POST /auth/api-key or GET /auth/derive-api-key               │      │
│  │   POLY_ADDRESS = EOA                                         │      │
│  │   POLY_SIGNATURE = standard 65-byte EOA ECDSA ClobAuth       │      │
│  │   → L2 key authenticates the HTTP request layer               │      │
│  └──────────────────────────────────────────────────────────────┘      │
│                              │                                          │
│                              ▼                                          │
│  Step 2: CLOB HTTP Gate Passes                                         │
│  ┌──────────────────────────────────────────────────────────────┐      │
│  │ POST /order (L2 HMAC headers)                                │      │
│  │   POLY_ADDRESS = EOA                                         │      │
│  │   → CLOB accepts the EOA-bound API key                        │      │
│  └──────────────────────────────────────────────────────────────┘      │
│                              │                                          │
│                              ▼                                          │
│  Step 3: Order Struct Carries Deposit-Wallet Identity                  │
│  ┌──────────────────────────────────────────────────────────────┐      │
│  │ signedOrderPayload {                                          │      │
│  │   maker  = depositWallet                                      │      │
│  │   signer = depositWallet   ← must equal maker for sigtype 3   │      │
│  │   signatureType = 3                                          │      │
│  │   signature = ERC-7739 wrapped order (636 hex chars)          │      │
│  │ }                                                             │      │
│  └──────────────────────────────────────────────────────────────┘      │
│                              │                                          │
│                              ▼                                          │
│  Step 4: On-Chain ERC-1271 Validates                                   │
│  ┌──────────────────────────────────────────────────────────────┐      │
│  │ CTF Exchange V2 calls:                                        │      │
│  │   depositWallet.isValidSignature(orderHash, wrappedSig)       │      │
│  │   → wallet validates EOA signature via ERC-1271 ✓             │      │
│  └──────────────────────────────────────────────────────────────┘      │
└─────────────────────────────────────────────────────────────────────────┘
```

## Step 1 — EOA-Bound CLOB API Key

The CLOB API key is created or derived with EOA-bound ClobAuth headers:

```go
headers, err := auth.BuildL1HeadersFromPrivateKey(
    privateKeyHex, // EOA private key signs ClobAuth
    chainID,
    timestamp,
    nonce,
)
// headers["POLY_ADDRESS"] = EOA
// headers["POLY_SIGNATURE"] = 65-byte standard EOA ECDSA signature
```

The compatibility helpers named `CreateOrDeriveAPIKeyForAddress` and `DeriveAPIKeyForAddress` retain their `ownerAddress` parameter for source compatibility, but the validated V2 implementation ignores it and uses EOA-bound auth. `internal/auth.BuildL1HeadersForDepositWallet` is an obsolete guarded helper and intentionally rejects the old deposit-wallet-bound ERC-7739 ClobAuth hypothesis.

## Step 2 — CLOB HTTP Gate

When placing orders, the CLOB HTTP layer checks the L2 HMAC headers against the API key. In polygolem's validated V2 path, `POLY_ADDRESS` in these L2 headers is the EOA address. The deposit wallet is not authenticated at the HTTP layer.

## Step 3 — Order Struct

Before posting a market order, `polygolem exchange market-order --dry-run`
prints a redacted preview of this shape and does not submit the order
(`internal/cli/cmd_clob.go:614` to `internal/cli/cmd_clob.go:619`,
`internal/clob/orders.go:1021` to `internal/clob/orders.go:1059`). Browser
wallets may label the nested deposit-wallet prompt as **Unknown Signature Type**;
for Polymarket V2 this is expected when the raw typed data shows
`name=DepositWallet`, `operation=TypedDataSign`, and order `signatureType=3`.

The signed order payload must have:

```go
order.Maker  = depositWallet  // holds the funds
order.Signer = depositWallet  // must equal maker for sigtype 3
order.SignatureType = 3       // POLY_1271
order.Signature = "0x..."     // ERC-7739 wrapped order, 636 hex chars
```

**The order signature IS ERC-7739 wrapped.** It uses the nested TypedDataSign structure:

```
innerSig(65) || appDomainSep(32) || contents(32) || contentsType(186) || uint16BE(186)
= 317 bytes = 634 hex chars + "0x" = 636 chars total
```

Where:
- `innerSig` = EOA's ECDSA signature over the final hash
- `appDomainSep` = CTF Exchange V2 domain separator
- `contents` = hashStruct(Order)
- `contentsType` = V2 Order type string (186 bytes)

## Step 4 — On-Chain Validation

The CTF Exchange V2 calls `isValidSignature()` on the deposit wallet:

```solidity
// CTFExchangeV2._verifyPoly1271Signature()
bool valid = IDepositWallet(signer).isValidSignature(hash, signature);
```

The deposit wallet:
1. Unwraps the ERC-7739 order envelope
2. Reconstructs the TypedDataSign hash
3. Verifies the EOA's ECDSA signature against it
4. Returns `0x1626ba7e` (ERC-1271 magic value) on success

## Key Distinction: CLOB Auth vs Order Signing

| Aspect | CLOB L1/L2 auth | Order Signing (POLY_1271) |
|--------|------------------|---------------------------|
| Signature type | Standard 65-byte EOA ECDSA ClobAuth | ERC-7739 wrapped order (636 chars) |
| Outer EIP-712 domain | `ClobAuthDomain` v1 | `Polymarket CTF Exchange` v2 |
| Signer | EOA | EOA |
| `POLY_ADDRESS` | EOA | EOA in L2 headers; deposit wallet in order `maker`/`signer` |
| Purpose | Authenticate HTTP requests | Authorize deposit-wallet trade |

## Polygolem Implementation

| Component | File | Purpose |
|-----------|------|---------|
| L1 CLOB auth | `internal/auth/l1.go::BuildL1HeadersFromPrivateKey` | Build EOA-bound ClobAuth headers |
| Source-compatible SDK | `pkg/clob.Client::{CreateOrDeriveAPIKeyForAddress,DeriveAPIKeyForAddress}` | Retain owner-address parameter but use EOA-bound auth |
| Order signing | `internal/clob/orders.go::buildSignedOrderPayload` | Build POLY_1271 order with correct maker/signer |
| ERC-7739 order wrap | `internal/clob/orders.go::wrapPOLY1271Signature` | Wrap EOA order sig in TypedDataSign envelope |
| Obsolete guard | `internal/auth/l1.go::BuildL1HeadersForDepositWallet` | Reject old deposit-wallet-bound ClobAuth hypothesis |

## Verification Checklist

- [ ] `POLY_ADDRESS` in `/auth/api-key` or `/auth/derive-api-key` headers = EOA
- [ ] `POLY_SIGNATURE` in L1 auth headers is standard 65-byte EOA ECDSA
- [ ] `POLY_ADDRESS` in L2 order headers = EOA
- [ ] Order `maker` = order `signer` = deposit wallet
- [ ] Order `signatureType` = 3
- [ ] Order `signature` is ERC-7739 wrapped (636 hex chars)
- [ ] Deposit wallet is deployed (relayer `/deployed` returns true)
- [ ] Deposit wallet has approvals (6 contracts approved)

## Related

- [Deposit Wallets (Polymarket docs)](https://docs.polymarket.com/trading/deposit-wallets)
- [CLOB Authentication](https://docs.polymarket.com/developers/CLOB/authentication)
- [EIP-7739 — TypedDataSign](https://eips.ethereum.org/EIPS/eip-7739)
- [EIP-1271 — Standard Signature Validation](https://eips.ethereum.org/EIPS/eip-1271)
