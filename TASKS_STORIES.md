# Implementation Tasks (Stories)

## S1 — Spore Packaging
- Create `internal/spore` with Pack/Verify/Extract.
- Add tests: invalid signature; modified binary; missing manifest.

## S2 — Repo
- Create `internal/repo` with Open/Put/Path.
- Tests: idempotent Put; corrupted file error.

## S3 — Fabric
- Create `internal/fabric` with mutex-protected maps and non-blocking pub/sub.
- Tests: PublishPlan fanout; RegisterEndpoint replaces by NodeID.

## S4 — Agent (MVP)
- Implement `Start(ctx)` loop: subscribe → for each plan → if not running → launch.
- Health check with timeout 6s.
- Register endpoint when healthy.

## S5 — Edge
- Reverse proxy `/app/...` with round-robin; header injection; basic error counting.

## S6 — Blue/Green
- Track running digest per app.
- On new digest: launch-new → warmup → stop-old → register-new.

## S7 — Telemetry Hooks
- Add `Logf` helper and periodic edge counters log.

## S8 — Smoke Script
- Makefile targets for workloads, spore, publish, run.

**Definition of Done (MVP):**
- Manual acceptance tests in SPEC pass.
- `go build ./...` clean.
- `go test ./...` passes for spore/repo/fabric.
