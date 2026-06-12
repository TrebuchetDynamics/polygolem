# Protocol Conformance Fixtures

These are the named conformance artifacts used by `go test ./tests -run Conformance` and package-level golden tests. They pin deterministic, synthetic vectors for protocol-sensitive bytes.

Rules:

1. Use synthetic keys and credentials only.
2. Record the reference repos and paths used for each family.
3. Compare exact strings/bytes in tests; no live network or live funds.
4. Keep these fixtures in sync with `fixtures/protocol/` while older tests still consume that directory.

Artifacts:

- `clob_auth_v2.json` — CLOB V2 ClobAuth EIP-712 L1 headers plus L2 HMAC canonicalization.
- `order_v2_poly1271.json` — V2 order typed-data struct hash and POLY_1271/ERC-7739 wrapped signature vectors.
- `builder_headers.json` — builder attribution HMAC header canonicalization.
- `ctf_calldata.json` — full CTF split, merge, and redeem calldata vectors.
