# Deposit-wallet-only trading

Polygolem supports Polymarket V2 trading through deposit wallets / POLY_1271 (`signatureType=3`) only. The EOA signs CLOB auth and order payloads, but the deposit wallet is the maker, signer, funder, pUSD holder, CTF position holder, and approval owner for production order transactions.

EOA, proxy, and Safe order modes are not supported for new production accounts. Documentation and code should fail closed rather than silently falling back to another wallet/signature mode.

Wallet setup is first-class in Polygolem: derive, deploy, relayer onboarding, approval batches, funding checks, settlement readiness, and user-directed CLOB order transaction support all assume the deposit-wallet path.
