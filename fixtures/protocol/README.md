# Protocol Conformance Fixtures

These fixtures turn the open-source reinforcement plan into executable drift
checks. Each fixture records:

- the protocol family being pinned;
- the trusted source or formula;
- the exact input values;
- the expected bytes/strings that polygolem must reproduce.

Rules for new fixtures:

1. Prefer values copied from a trusted reference repo or protocol contract.
2. Record the source repo, commit/path when available, and why the source is trusted.
3. Keep secrets synthetic and deterministic; never include real API keys or wallet keys.
4. Tests must compare byte-for-byte or string-for-string and print useful diffs.

Implemented families:

- `hmac_headers.json` — CLOB L2 and builder HMAC header canonicalization.
- `eip712_orders.json` — V2 CLOB POLY_1271 order typed-data hashes and ERC-7739 wrapped signatures.
- `clob_auth.json` — validated EOA-bound ClobAuth L1 headers for API-key creation/derivation.
- `ctf_calldata.json` — split, merge, redeem calldata selectors and payloads.
- `deposit_wallet_batch.json` — relayer DepositWallet.Batch typed-data fixtures.
