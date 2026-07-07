# Polygolem is a Polymarket API interface, not a bot

Polygolem is an interface into Polymarket APIs and contracts. It provides typed SDK packages, CLI commands, wallet setup, approvals, signing primitives, CLOB order transaction support, read-only market/data surfaces, and safety gates for user-directed actions.

Polygolem must not choose markets, sides, prices, sizes, timing, or whether to trade. Those decisions belong to the human operator or an external application that calls Polygolem. Documentation and code should avoid describing Polygolem as a bot, strategy engine, autonomous trader, or decision maker.

Stable user-directed order transactions live in `pkg/clob` and `polygolem exchange create-order` / `polygolem exchange market-order`. Experimental order helper packages may validate or shape DTOs, but they do not replace the stable transaction interface and must not introduce autonomous decision logic.
