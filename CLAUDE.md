# cg-proto — CLAUDE.md (session handoff)

Entry point for a new Claude session in this repo. Read before editing.

## What this is

**Centralized Protocol Buffers definitions for all CTOgram services.** Module: `github.com/4ubak/cg-proto`. One package per domain. Generated Go code under `gen/go/`. Consumers pin a tagged release (latest is `v1.271.0` as of 2026-07-06) in their `go.mod`.

## Layout

One subdir per domain — each contains `.proto` files and (after `buf generate`) the corresponding `gen/go/<domain>/**/*.pb.go` (version suffix varies — most are `.../v1/`, some subpackages have none):

`agents/`, `agreement/`, `ai/`, `billing/`, `booking/`, `communication/`, `crm/`, `jobs/`, `orders/`, `parser/`, `parts/`, `payments/`, `platform/`, `services/`, `signing/`, `subscriptions/`, `telephony/`, `users/`, `warehouse/`, `workshop/`

Top-level: `buf.yaml`, `buf.gen.yaml`, `go.mod`, `gen/`.

## Consumer map

| Proto package | Consumed by |
|---|---|
| `users` | every service (auth/JWT) |
| `ai` | cg-bff, cg-services (matching), cg-orders |
| `services` (request/bid/matching/search) | cg-bff, cg-ai, cg-orders, cg-workshop |
| `orders` | cg-bff, cg-payments, cg-billing, cg-workshop, cg-ai |
| `payments` | cg-bff, cg-orders, cg-billing |
| `billing` | cg-bff, cg-orders, cg-subscriptions |
| `workshop` | cg-bff, cg-orders, cg-services |
| `communication` | cg-bff, cg-ai, every event-publishing service |
| `platform` | cg-bff, every service (nsi/counter) |
| `crm` | cg-bff, cg-crm-web |
| `parts` | cg-bff, cg-ai, cg-orders |
| `warehouse` | cg-bff, cg-orders |
| `booking` | cg-bff, cg-services |
| `agreement` | cg-bff, cg-orders, cg-workshop |
| `subscriptions` | cg-bff, cg-billing |
| `signing` | cg-bff, cg-workshop (АВР/ЭЦП via cg-signing) |
| `telephony` | cg-bff (bff-admin thin-proxy) via cg-asterisk (Asterisk/ARI control-plane, Go service in repo root) |

## Domain rules (locked)

- **One coherent package per domain.** Don't cross-define: orders proto cannot import workshop proto types — duplicate the fields or reference them via well-known scalar IDs.
- **Backward compatibility**: never delete/renumber fields. Only add new ones with new tags.
- **Tag every release**: `v1.X.Y` SemVer, then push with `--tags` to GitHub.
- **Consumers never use local `replace` directives.** `replace github.com/4ubak/cg-proto => /Users/...` breaks CI. Direct GitHub pin only.
- `GOPRIVATE=github.com/4ubak/*` required in consumer envs and CI.

## Workflow

```bash
# Inside cg-proto:
# 1. edit .proto files
# 2. regenerate
buf generate

# 3. local validation
go build ./...

# 4. tag + push
git add . && git commit -m "feat: <change>"
git tag v1.X.Y
git push --tags

# 5. in each consuming service:
GONOSUMDB='github.com/4ubak/*' GOPROXY=direct go get github.com/4ubak/cg-proto@v1.X.Y
go mod tidy && go build ./... && go vet ./...
```

## Gotchas

- `buf generate` must run from repo root, not from a subdir.
- After regen, `git status` should show `gen/...` changes — these are committed (not gitignored).
- If a consumer's CI fails with "missing go.sum entry," they forgot `go mod tidy`.
- New domain dirs are auto-discovered — `buf.yaml` is a single `version: v1` module with no modules list to edit. Just add `<domain>/**/*.proto` and run `buf generate` (output lands under `gen/go/` per `buf.gen.yaml`).

## Critical files

- `buf.yaml`, `buf.gen.yaml` — codegen config
- `<domain>/v1/*.proto` — actual contracts
- `gen/go/<domain>/**/*.pb.go` — generated; never edit by hand

## Commands

```bash
buf generate          # regenerate Go bindings
buf lint              # protobuf lint
go build ./...        # validate generated code compiles
go test ./...         # any helpers that have tests
```

## Commit & deploy

- No deployment — this is a code-gen target repo.
- Releases = git tags.
- Coordinate consumer bumps in a single sweep after a breaking change.
