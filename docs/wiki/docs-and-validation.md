# Docs and Validation

## What this is

A compact map of which docs are generated, which docs are canonical, and which
validation commands prove common changes are safe.

## Start here

`docs/README.md` is the canonical docs index. It states that historical notes
under `docs/history/` preserve audit context and may contain old wording, and
that generated docs-site output is not edited by hand (`docs/README.md:7` to
`docs/README.md:10`).

## Generated docs

`docs/COMMANDS.md` is generated from Cobra metadata. The generator writes the
header and warning from `internal/cli/docs_generation.go:23` to
`internal/cli/docs_generation.go:31`, then renders conventions, environment
variables, command tree, and command reference from `internal/cli/docs_generation.go:32`
to `internal/cli/docs_generation.go:47`.

Run this after command metadata changes:

```bash
go run ./cmd/polygolem_docs
```

## Validation gates

| Change type | Small check | Full check | Source evidence |
|---|---|---|---|
| CLI command rename/add/remove | `go test ./internal/cli` | `go test ./...` | `tests/cli_rename_e2e_test.go:10` verifies renamed commands through a built binary. |
| Public SDK package inventory | `go test ./tests -run TestPublicPackageInventoryIsDocumented` | `go test ./...` | `tests/repository_hygiene_test.go:90` requires README, architecture, and docs-site SDK docs for every `pkg/` package. |
| Docs site pages | `npm --prefix docs/docs-site run build` | CI docs-site job | CI installs and builds the docs site (`.github/workflows/ci.yml:9` to `.github/workflows/ci.yml:19`). |
| Go behavior | package-scoped `go test` | `go test ./...` plus `go vet ./...` | CI runs gofmt check, vet, short tests, race, and coverage (`.github/workflows/ci.yml:31` to `.github/workflows/ci.yml:56`). |
| Markdown/patch hygiene | `git diff --check` | same | Delivery workflow uses whitespace diff checks before commit. |

## Historical vs current docs

When auditing old command names, do not treat every match as stale:

- Current user/operator docs should use renamed commands (`ping`, `markets`,
  `book`, `exchange`, `wallet`, `credentials`, etc.).
- `docs/history/` and old planning artifacts can intentionally preserve old
  command names as audit evidence.
- `CHANGELOG.md` can intentionally preserve old names in older release entries.

## Update triggers

Refresh this page when:

- `docs/README.md` changes docs ownership/status rules;
- CI validation changes;
- command generation changes;
- new docs tests are added under `tests/` or `internal/cli/*_test.go`.
