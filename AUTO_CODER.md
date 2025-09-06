# AUTO CODER PROMPT (Paste into VS Code Chat as a single message)

You are the lead engineer implementing the **Mycelium Mesh** monolith in Go.
Read `ARCHITECTURE.md` and `CONTEXT.md`. Then perform the following, step-by-step:

1) **Scaffold & Verify**
- Ensure the repo layout exactly matches ARCHITECTURE.md §4.
- Create missing folders/files with minimal compilable code.
- Add `go.mod`, run `go mod tidy`, and ensure `go build ./...` is clean.

2) **Implement internal/spore**
- Functions: `Pack(binaryPath, Manifest, privKey, outDir)`, `Verify(sporePath)`, `Extract(sporePath, destDir)`.
- Ed25519 signatures; manifest includes `binary_sha256`, `signature`, `public_key`.
- Add unit tests: corrupted binary -> verify fails; signature mismatch -> fails.

3) **Implement internal/repo**
- Content-addressed Put/Path with SHA-256 of file contents.
- Tests: Put twice produces same digest and overrides safely.

4) **Implement internal/fabric**
- Types: `Plan`, `Budget`, `Endpoint`.
- Functions: `PublishPlan`, `SubscribePlans`, `SetBudget`, `GetBudget`, `RegisterEndpoint`, `Endpoints`.
- Concurrency-safe (mutex), buffered channels, non-blocking publish.

5) **Implement internal/agent**
- Subscribe to plans; for each app not running: pull verify extract; pick free port; launch with `PORT`; wait `/health`; register endpoint.
- Log PID; add retry on health for up to 6s.
- Add `Stop()` logic stub.

6) **Implement internal/edge**
- HTTP mux handling `/app/...`; reverse proxy to endpoints in round-robin using atomic counter.
- Set header `X-Mycelium-Edge: <timestamp>`.

7) **Implement cmd/mesh**
- Subcommands: `build`, `publish`, `run` (flags per ARCHITECTURE.md).
- `run` starts edge + several agents and publishes a plan/budget.

8) **Examples**
- Two workloads with `/health` and `/hello` handlers.

9) **Smoke Test**
- Build workloads; build/publish spore; run mesh; `curl` returns success.

10) **Deliverables**
- `Makefile` targets: `build`, `workloads`, `spore`, `run`.
- `.vscode/tasks.json` tasks to build, spore, run.
- Document usage in `README.md`.

While coding:
- Keep packages small and focused.
- Prefer stdlib; no external deps unless necessary.
- Write clear comments where behavior is non-obvious.
