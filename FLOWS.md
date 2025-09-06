# Mycelium Mesh — Request & Sequence Flows

## Legend
- **CLI**: mesh CLI
- **Repo**: content-addressed store
- **Fabric**: in-proc control plane
- **Agent[n]**: node agent
- **Edge**: reverse proxy
- **Svc**: running spore process (HTTP server)

---

## Flow A — Build & Publish

1. **CLI** → reads manifest JSON and workload binary path.
2. **CLI** → `spore.Pack(binary, manifest, privKey)`
   - Compute `binary_sha256`.
   - Fill manifest fields (timestamps).
   - Compute signature; write zip: `manifest.json` + `binary`.
3. **CLI** → `repo.Put(sporeFile)`
   - Compute file sha256; copy as `repo/<digest>.spore`.
4. **CLI** → prints `<digest>`.

**Artifacts**: `.spore` file in `repo/`, digest string.

---

## Flow B — Deploy (Run)

1. **CLI** → `Fabric.SetBudget({App, MaxInstances, CPU, Mem})`
2. **CLI** → `Fabric.PublishPlan({App, Digest, Min, Max})`
3. **Edge** starts (listening on `:8080`).
4. **Agent[n]** subscribe to plans.
5. On receiving a plan:
   - Check budget and if app already running on this node.
   - Pull path from `Repo.Path(Digest)`; verify via `spore.Verify`.
   - Extract via `spore.Extract(destDir)` → `binPath`.
   - Pick free `PORT`; start process; wait `/health` OK.
   - `Fabric.RegisterEndpoint({App, URL, NodeID})`.

**Result**: Fabric has endpoints for the app from N agents.

---

## Flow C — Request Routing

1. **Client** → `GET /{app}/hello` to **Edge**.
2. **Edge** → lookup `Fabric.Endpoints(app)`.
3. Select backend (round-robin); proxy request.
4. **Svc** → returns 200; **Edge** adds `X-Mycelium-Edge` header; respond to client.

---

## Flow D — Blue/Green Update

1. **Operator** publishes plan with **new Digest** for same App.
2. **Agent[n]** sees a plan with new Digest:
   - Launch **new** process (green) → wait healthy.
   - Wait `warmup` duration.
   - Stop **old** process (blue) on this node.
   - Register endpoint for green; ensure old endpoint removed/overwritten.
3. **Edge** continues round-robin across the now-updated endpoints.

---

## Flow E — Failure

- **Svc** process exits:
  - Agent logs exit; removes endpoint (MVP may leave stale until next register).
  - Future: auto-respawn using the same Digest.
- **Agent** exits:
  - Endpoints for that Node become stale until re-registration (future: TTL).

---

## State Machines (per Agent/App)

### States
- `Idle` → `Launching` → `Ready` → `Draining` → `Stopped`

### Transitions
- `Idle` + Plan(Digest) → `Launching`
- `Launching` + HealthOK → `Ready` (register endpoint)
- `Ready` + NewDigest → `Launching(new)`; after warmup → `Draining(old)` → `Stopped(old)`
- `Ready` + ProcessExit → `Stopped` (future: respawn)
