# ChatGPT-in-VSCode Context

> Paste this into a file named `CONTEXT.md` at the repo root and pin it in your VS Code Chat as the primary context file.

## Project Goal
Implement a **containerless orchestrator** called **Mycelium Mesh** in Go, as described in `ARCHITECTURE.md`.

## Coding Conventions
- Language: Go 1.24+
- Module: single Go module
- Package names: lowercase, short
- Logs: `log.Printf` with prefixes (later: structured logs)
- Errors: return wrapped errors, avoid panics in libraries
- HTTP: stdlib `net/http`

## Definitions (Glossary)
- **Spore**: signed bundle with `manifest.json` + binary. See `internal/spore`.
- **Fabric**: in-proc pub/sub for Plans/Budgets/Endpoints. See `internal/fabric`.
- **Agent**: node daemon that sprouts spores. See `internal/agent`.
- **Edge**: reverse proxy routing `/app/...` to endpoints. See `internal/edge`.

## Acceptance Criteria (M1)
- `mesh build` creates a signed `.spore` file.
- `mesh publish` stores it in a content-addressed repo and prints the digest.
- `mesh run` starts edge + N agents, pulls/launches spores, registers endpoints.
- `curl http://localhost:8080/<app>/hello` returns 200 from an instance.
- Repository passes `go vet` and `go test ./...` (when tests added).

## What ChatGPT Should Generate Next
1. Unit tests for `internal/spore` and `internal/repo`.
2. Blue/green rollout in `internal/fabric` + `agent`.
3. Minimal `internal/telemetry` with an interface and a no-op implementation.
4. GitHub Action: build workloads, package spore, upload artifact.

## Ground Rules for Refactors
- Keep public APIs stable for `spore`, `repo`, and `fabric` packages.
- Add new features behind interfaces; do not break `cmd/mesh` CLI flags.
