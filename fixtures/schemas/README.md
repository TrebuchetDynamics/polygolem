# Public JSON Schemas

These schemas pin public JSON shapes used by the CLI and SDK-facing fixtures.
They are intentionally checked in so docs, examples, and compatibility tests can
fail loudly when an envelope or DTO shape changes.

Current schemas:

- `cli_envelope.schema.json` — shared CLI success/error envelope (`internal/output.Envelope`).
- `rfq_request.schema.json` — public `pkg/rfq.Request` shape used before live RFQ submission exists.
- `bridge_withdraw_request.schema.json` — public `pkg/bridge.WithdrawRequest` dry-run shape.
- `ctf_operation_request.schema.json` — public `pkg/ctf.OperationRequest` split/merge dry-run shape.

Rules:

1. Update schemas only with an intentional compatibility decision.
2. Keep examples and tests generated from the same public structs where possible.
3. Never include secrets in schema examples or fixtures.
