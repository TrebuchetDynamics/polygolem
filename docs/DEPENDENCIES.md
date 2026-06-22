# Dependency Story

Polygolem is distributed as a single Go binary with no Python, npm, Docker, or runtime sidecars. The dependency tree is still crypto-heavy because the protocol surface is crypto-heavy.

## Why go-ethereum is here

`github.com/ethereum/go-ethereum` provides the EIP-712, ABI, address, signing, and Polygon RPC primitives used for:

- CLOB V2 order typed-data signing;
- POLY_1271 / ERC-7739 deposit-wallet order wrappers;
- ERC-20 pUSD transfers and allowance checks;
- deposit-wallet bytecode checks and relayer batch calldata;
- Polygon transaction submission for funding/swap paths.

That dependency pulls indirect cryptography packages such as KZG, secp256k1, uint256, and low-level system packages. They increase module size, but they do not create a Python/npm runtime requirement.

## Runtime promise

- Build/install: Go toolchain only.
- Run read-only commands: no wallet, no API key, no local service.
- Run live wallet/trading commands: environment credentials and Polygon/CLOB/relayer network access.

## Review checklist for new dependencies

Before adding a dependency, prefer the standard library or an existing package. Add a new module only when it:

1. removes security-sensitive custom code;
2. supports a protocol primitive we cannot safely maintain;
3. is used by production code, not just a one-off script;
4. keeps the single-binary runtime story intact.
