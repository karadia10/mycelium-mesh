# Mycelium Mesh — Feature Spec & Requirements (Authoritative)

## Goal
Implement a containerless orchestrator monolith in Go that supports:
1) Packaging a **Spore** (binary + manifest) with signing.
2) Publishing to a content-addressed **Repo**.
3) Running an **Edge**, **Control Fabric**, and multiple **Agents** on one machine.
4) Deploying (sprouting) spores as OS processes, health-checking, and routing traffic.
5) Blue/green update flow (minimal viable).
6) Observability hooks (log lines + counters; OTel later).

---

## Core Features (MVP)

### F1 — Spore Packaging
- Input: `binaryPath`, `manifest` (JSON), `ed25519 private key`.
- Output: `.spore` (zip) containing `manifest.json` + `binary`.
- Manifest MUST include:
  - identity: `{kind:"Spore", name, version}`
  - runtime: `{command, args[], env{}}`
  - resources: `{nutrients: {cpu_milli, memory_mb}}`
  - SLO: `{p99_budget_ms}`
  - security: `{lsm_profile, read_only_fs}` (not enforced in MVP)
  - integrity: `{binary_sha256, signature, public_key, created_at}`
- Signing: `ed25519(Sign( sha256( manifest_without_sig || binary_sha256 ) ))`

### F2 — Repo (Content Addressed)
- `Put(file)` → `digest` (sha256 of file bytes), copies file to `repo/<digest>.spore`.
- `Path(digest)` → absolute file path.

### F3 — Control Fabric (in-process)
- Maintains:
  - `Plan{ AppName, Digest, Min, Max, Port }`
  - `Budget{ AppName, MaxInstances, CPUmilli, MemoryMB }`
  - `Endpoint{ AppName, URL, NodeID }`
- APIs:
  - `PublishPlan(Plan)` → broadcast to subscribers (non-blocking).
  - `SubscribePlans() <-chan Plan`
  - `SetBudget(Budget)`, `GetBudget(app)`
  - `RegisterEndpoint(Endpoint)`, `Endpoints(app) []Endpoint`

### F4 — Agent (Node)
- Subscribes to plans.
- For each plan:
  - If budget allows and instance not running on this node, then:
    1) Pull `.spore` via Repo.Path(Digest).
    2) Verify signature & binary hash.
    3) Extract to `run/<nodeID>/<app>-<rand>/`.
    4) Choose free TCP port; set `PORT` env; launch process.
    5) Wait for `/health` (<= 6s).
    6) Register `Endpoint{ AppName, URL, NodeID }` with Fabric.
- If process exits, agent logs and (MVP) leaves it down (auto-respawn later).

### F5 — Edge (Reverse Proxy)
- Listens on configurable addr (e.g., `:8080`).
- Routes HTTP `/{app}/...` → one of Fabric.Endpoints(app) (round-robin).
- Adds header: `X-Mycelium-Edge: <RFC3339>`.

### F6 — Blue/Green Update (Minimal)
- Operator publishes a new `Plan{Digest: newDigest}` for the same `AppName`.
- Agents receiving a *new digest* should:
  - Launch a new instance (steps above) AND mark it **Ready**.
  - After N seconds of readiness (configurable; default 2s), stop old instance (same app, old digest) on this node.
  - Register only the newest endpoint in Fabric for this node/app.

### F7 — Logging & Counters
- Agent logs:
  - plan received, launch start, health ok/fail, endpoint register, process exit.
- Edge counters (in-memory):
  - requests per app, 5xx per app (exposed via log every 10s).

---

## Non-Goals (MVP)
- Real gossip/DHT (single-process fabric only).
- cgroups, eBPF, LSM enforcement (stub only).
- Secrets/integrations, multi-node networking, TLS termination.
- Persistent workloads/state.

---

## Config & Defaults
- Health endpoint path: `/health` (200 = healthy).
- Edge routing prefix: `/{app}/...`.
- Blue/green warmup: `2s` (can be a flag `-warmup 2s` on `mesh run`).

---

## Directory Contracts
- `internal/spore`: **MUST** export `Pack`, `Verify`, `Extract`.
- `internal/repo`: **MUST** export `Open`, `Put`, `Path`.
- `internal/fabric`: **MUST** export `New`, `PublishPlan`, `SubscribePlans`, `SetBudget`, `GetBudget`, `RegisterEndpoint`, `Endpoints`.
- `internal/agent`: **MUST** export `New`, `Start(ctx)`, and support blue/green per SPEC F6.
- `internal/edge`: **MUST** export `New`, `Start(addr)` and do RR.

---

## Acceptance Tests (Manual)
1) Build workloads, pack, publish, run → `curl /billing/hello` returns 200.
2) Update to a new digest; within ~2s, requests still succeed and old instance is stopped on each node.
3) Kill a workload process; edge returns 5xx briefly until agent removes/reg updates (future improvement).
