# Copilot Instructions for Mycelium Mesh

## Project Overview
Mycelium Mesh is a containerless orchestrator written in Go. It replaces containers and central API servers with signed binaries (Spores), node agents, and a decentralized control fabric. The project is a monolith demo, showing the full flow: build → sign → publish → schedule → sprout → route → observe.

## Architecture & Major Components
- **Spore**: Signed bundle (binary + manifest). Implements packing, verification, extraction, and Ed25519 signing. See `internal/spore/`.
- **Repo**: Content-addressed storage for spores. See `internal/repo/`.
- **Fabric**: In-process control plane for plans, budgets, endpoints. See `internal/fabric/`.
- **Agent**: Node daemon that subscribes to plans, verifies, extracts, launches spores as OS processes. See `internal/agent/`.
- **Edge**: Reverse proxy that routes `/app/...` requests to endpoints using round-robin. See `internal/edge/`.
- **Workloads**: Example HTTP servers in `cmd/workload-billing/` and `cmd/workload-frontend/`.
- **CLI**: Entrypoint in `cmd/mesh/` for build, publish, and run commands.

## Developer Workflows
- **Build all**: `go build ./...`
- **Run all tests**: `go test ./...`
- **Build workloads**: `go build -o bin/billing ./cmd/workload-billing`
- **Pack spore**: `go run ./cmd/mesh build -manifest ./examples/billing.json -binary ./bin/billing -out ./out`
- **Publish spore**: `go run ./cmd/mesh publish -spore ./out/*.spore -repo ./repo`
- **Run mesh**: `go run ./cmd/mesh run -repo ./repo -digest <DIGEST> -app billing -instances 2 -edge :8080 -nodes 3`
- **Test routing**: `curl http://localhost:8080/billing/hello`

## Project-Specific Conventions
- **Single Go module** (see `go.mod`).
- **Ed25519** for signing spores.
- **Manifest fields**: See `internal/spore/Manifest` and `examples/*.json`.
- **Health checks**: Workloads must respond to `/health` (HTTP 200 = healthy).
- **Round-robin routing**: Edge proxies requests to endpoints in a round-robin fashion.
- **Resource budgets**: Managed via Fabric and enforced by Agent.
- **No central API server**: All coordination is in-process and decentralized.

## Integration Points & Patterns
- **Spore <-> Repo**: Spores are packed and published to the repo, then pulled by agents.
- **Agent <-> Fabric**: Agents subscribe to plans, register endpoints, and update budgets.
- **Edge <-> Fabric**: Edge queries Fabric for endpoints to route traffic.
- **Workloads**: Must implement `/health` and `/hello` endpoints for integration tests.

## Key Files & Directories
- `internal/spore/`: Spore logic and signing
- `internal/repo/`: Content-addressed storage
- `internal/fabric/`: Control plane logic
- `internal/agent/`: Node agent orchestration
- `internal/edge/`: Reverse proxy
- `cmd/mesh/`: CLI entrypoint
- `examples/`: Workload manifests
- `README.md`: Quickstart, architecture, and philosophy

## Example Patterns
- **Packing a spore**:
  ```go
  sporePath, manifest, err := spore.Pack(binaryPath, manifest, privKey, outDir)
  ```
- **Publishing to repo**:
  ```go
  digest, storedPath, err := repo.Put(sporePath)
  ```
- **Agent launching a workload**:
  ```go
  agent := agent.New(id, fabric, repo, runDir)
  agent.Start(ctx)
  ```
- **Edge proxy routing**:
  ```go
  edge := edge.New(fabric)
  edge.Start(":8080")
  ```

## Testing & Verification
- Unit tests for spore, repo, and fabric are required to pass (`go test ./...`).
- Manual acceptance: build, pack, publish, run mesh, and verify with curl as described in README.

---

If any section is unclear or missing, please provide feedback for iterative improvement.
