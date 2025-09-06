# Mycelium Mesh — Detailed Architecture

## 0. Problem Statement
Ship and run backend services **without containers** or a central API server. Each workload is a **Spore** (signed binary + manifest) that a node agent can verify and run as an **isolated process**. Nodes discover each other and route traffic via a lightweight **Hyphae Router** and coordinate via a decentralized **Control Fabric**.

---

## 1. Domain Model
- **Spore**: signed bundle (`manifest.json` + binary). Digest-addressed. Example fields:
  - `name`, `version`, `command`, `args`, `env`
  - `nutrients { cpu_milli, memory_mb }` (resource envelope)
  - `slo { p99_budget_ms }`
  - `security { lsm_profile, read_only_fs }`
  - `binary_sha256`, `signature`, `public_key`, `created_at`

- **Repository**: content-addressed storage for `.spore` files. API:
  - `Put(path) -> (digest, storedPath)`
  - `Path(digest) -> file path`

- **Control Fabric**: pub/sub + tiny registry.
  - **Plan**: desired state for an app `{ app, digest, min, max, port }`
  - **Budget**: nutrient credits & caps per app
  - **Endpoint**: runtime location `{ app, url, nodeID }`

- **Spored Agent**: node daemon that subscribes to plans, pulls spores, verifies signatures, extracts, launches as OS process, registers endpoint, emits telemetry.

- **Edge Gateway**: minimal HTTP reverse proxy that routes `/app/...` to registered endpoints using round-robin (or policy later).

---

## 2. Core Flows (Step-by-step)
1) **Build**: CI compiles the workload binary.
2) **Pack**: `spore.Pack(binary, manifest, privKey)` → zip with `manifest.json` & binary, signed.
3) **Publish**: repo stores the zip under SHA-256 digest.
4) **Plan**: operator publishes `Plan{app, digest, min, max}` to the fabric.
5) **Schedule**: each Agent decides if it can sprout (demo = one instance per agent).
6) **Verify+Extract**: Agent verifies signature + binary hash, extracts to a run dir.
7) **Launch**: Agent sets env (e.g., `PORT`), starts process, waits for `/health`.
8) **Register**: Agent registers `Endpoint{app, url, nodeID}` with the fabric.
9) **Ingress**: Edge receives `/app/...` and proxies to a live endpoint.
10) **Pulse**: Agents emit logs/metrics (placeholder; wire OTel later).
11) **Recover**: If a process dies, Agent can re-sprout; if node dies, other agents sprout (future).

---

## 3. Non-Functional Requirements (NFRs)
- **Security**: mandatory signature verification; SBOM inclusion (future).
- **Isolation**: cgroups/LSM/eBPF (future; stub interfaces now).
- **Observability**: health endpoint required; later OTel metrics/traces.
- **Scalability**: decentralized plan gossip (demo uses in-proc pub/sub).

---

## 4. Packages (Go)
```
internal/spore   // pack/verify/extract; ed25519
internal/repo    // content-addressed store
internal/fabric  // plans, budgets, endpoints; pub/sub
internal/agent   // node agent: launch & register
internal/edge    // reverse proxy
cmd/mesh         // CLI: build, publish, run
```
**Extension points**:
- `internal/nutrients` for ledgers/credits → cgroups.
- `internal/telemetry` for OTel exporters.
- `internal/crypto` for Sigstore (later).

---

## 5. Data Contracts

### 5.1 `manifest.json` (Spore DNA)
```json
{
  "kind": "Spore",
  "name": "billing",
  "version": "v0.1.0",
  "command": "billing",
  "args": [],
  "env": { "GREETING": "hi" },
  "provides": ["http"],
  "nutrients": { "cpu_milli": 200, "memory_mb": 128 },
  "slo": { "p99_budget_ms": 300 },
  "security": { "lsm_profile": "demo", "read_only_fs": false },
  "binary_sha256": "<hex>",
  "signature": "<base64>",
  "public_key": "<base64>",
  "created_at": "2025-09-06T00:00:00Z"
}
```

### 5.2 Fabric Messages
- `Plan`: `{ "appName": "billing", "digest": "<sha256>", "min": 2, "max": 4, "port": 0 }`
- `Budget`: `{ "appName": "billing", "maxInstances": 4, "cpu_milli": 500, "memory_mb": 256 }`
- `Endpoint`: `{ "appName": "billing", "url": "http://127.0.0.1:8081", "nodeID": "node-1" }`

---

## 6. Implementation Plan (Milestones)

### M1 — Monolith Running Locally
- [ ] Implement `internal/spore` (Pack, Verify, Extract: ed25519, zip).
- [ ] Implement `internal/repo` (Put/Path, content address).
- [ ] Implement `internal/fabric` (in-proc pub/sub; Endpoints; Budgets; Plans).
- [ ] Implement `internal/agent` (subscribe → verify → extract → exec → register).
- [ ] Implement `internal/edge` (reverse proxy `/app/...` round-robin).
- [ ] CLI `cmd/mesh` with `build`, `publish`, `run` commands.
- [ ] Example workloads (`billing`, `frontend`).

### M2 — Hardening
- [ ] Health & readiness with retries/backoff.
- [ ] Structured logging + request IDs at edge.
- [ ] Basic rolling update: publish new digest as `Plan`, blue/green flip.
- [ ] Unit tests for `spore` and `repo` packages.

### M3 — Advanced
- [ ] Nutrient ledger → cgroups enforcement.
- [ ] eBPF-based process/network telemetry.
- [ ] Real gossip/DHT (memberlist, hashicorp/serf or libp2p).
- [ ] OTel metrics/traces export + dashboards.
- [ ] Secrets/config binding via files/env + KMS integration.

---

## 7. Security & Trust
- **Sig**: ed25519 signature over `sha256(manifest_without_sig || binary_hash)`.
- **Verification**: node must validate signature and binary hash before launching.
- **Future**: Sigstore keyless, SBOM embedded, policy checks before sprouting.

---

## 8. Local Dev UX
- `go build` for workloads.
- `mesh build` → signed `.spore` → `mesh publish` → digest.
- `mesh run` → edge + N agents start; visit `http://localhost:8080/<app>/hello`.

---

## 9. Diagrams
See the **step-by-step** and **overview** PNGs you generated earlier for visuals.
