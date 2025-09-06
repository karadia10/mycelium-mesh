# Mycelium Mesh (Demo, Go)

A **from-scratch experiment** in building a **containerless orchestrator** written in Go.  
Instead of containers and a central Kubernetes API server, Mycelium Mesh introduces:

- **Spore** → the unit of deployment (signed binary + DNA manifest).  
- **Spored agent** → runs on each node, verifies & sprouts spores as processes.  
- **Control Fabric** → gossip + DHT style control plane (no etcd / API server).  
- **Nutrient Ledger** → resource budgets & credits instead of static limits.  
- **Hyphae Router** → peer-to-peer encrypted overlay for service discovery & traffic.  

This repo is a **monolith demo** — a single Go module showing the full end-to-end flow:  
**build → sign → publish → schedule → sprout → route → observe.**

---

## 🚀 Quick Start

### 1. Clone & enter
```bash
git clone https://github.com/karadia10/mycelium-mesh.git
cd mycelium-mesh
```

### 2. Build example workloads
```bash
go build -o bin/billing ./cmd/workload-billing
go build -o bin/frontend ./cmd/workload-frontend
```

### 3. Build a spore
```bash
go run ./cmd/mesh build   -manifest ./examples/billing.json   -binary ./bin/billing   -out ./out
```
This creates a signed `.spore` bundle in `./out/`.

### 4. Publish to the local repo
```bash
go run ./cmd/mesh publish -spore $(ls out/*.spore) -repo ./repo
```
Note the printed **digest** (a SHA-256 string).

### 5. Run the mesh
```bash
go run ./cmd/mesh run   -repo ./repo   -digest <DIGEST>   -app billing   -instances 2   -edge :8080   -nodes 3
```

### 6. Test it
```bash
curl http://localhost:8080/billing/hello
```
You should see round-robin responses from different sprouted spores.

---

## 📂 Repo Structure
```
cmd/mesh/              # CLI: build, publish, run
cmd/workload-billing/  # Example workload (HTTP server)
cmd/workload-frontend/ # Another example workload
internal/agent/        # Node agent (sprouts spores as processes)
internal/edge/         # Reverse proxy edge gateway
internal/fabric/       # Control fabric (pub/sub, registry, budgets)
internal/repo/         # Content-addressed repo for spores
internal/spore/        # Pack/verify/extract spores, ed25519 signing
examples/              # DNA manifests for workloads
```

---

## 🧩 What’s Implemented
- Spore packaging (zip with manifest + binary, signed with Ed25519).  
- Local content-addressed repo for spores.  
- In-process “control fabric” and simple budgets.  
- Node agents that verify & run spores as OS processes.  
- Edge proxy that routes `/app/...` requests to live spores.  
- Example workloads (`billing`, `frontend`) with `/health` and `/hello`.  

---

## 🛠️ What’s Missing (Future Work)
- Real decentralized gossip/DHT (currently in-process pub/sub).  
- Nutrient ledger with true cgroups/LSM/eBPF enforcement.  
- Rolling updates, blue/green deployment, SLO-aware autoscaling.  
- Secrets/config binding.  
- Multi-language workload support.  

---

## ✨ Philosophy
- Kubernetes = **bureaucracy** (central API server + declarative state).  
- Mycelium Mesh = **ecosystem** (self-organizing spores + nutrient economy).  
