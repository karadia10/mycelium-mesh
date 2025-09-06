# Mycelium Mesh - Project Summary & Reference

## 🎯 Project Overview
**Mycelium Mesh** is a **containerless orchestrator** written in Go that eliminates containers and central API servers. It's inspired by biological mycelium networks and uses an economic model for resource management.

## 🧬 Core Concepts

### Spores (Deployment Units)
- **What**: Signed binary + DNA manifest bundle
- **Security**: Ed25519 signature verification
- **Content**: `manifest.json` + executable binary
- **Trust**: Cryptographically verified before execution

### Spored Agents (Node Daemons)
- **Role**: Run on each machine, verify and execute spores
- **Process**: Verify signature → Extract → Launch as OS process → Register endpoint
- **Health**: Monitor `/health` endpoint, emit telemetry

### Control Fabric (Decentralized Control Plane)
- **Type**: Gossip + DHT style coordination (no etcd/API server)
- **Manages**: Plans, Budgets, Endpoints
- **Communication**: Pub/sub with non-blocking publish

### Nutrient Ledger (Resource Management)
- **Model**: Economic credits instead of static resource limits
- **Future**: cgroups/LSM/eBPF enforcement
- **Allocation**: Dynamic based on available credits

### Hyphae Router (P2P Networking)
- **Purpose**: Service discovery and traffic routing
- **Style**: Peer-to-peer encrypted overlay
- **No**: Central load balancer needed

## 🏗️ Architecture Flow
1. **Build** → Compile workload binary
2. **Pack** → Create signed `.spore` bundle
3. **Publish** → Store in content-addressed repo
4. **Plan** → Publish deployment plan to fabric
5. **Schedule** → Agents decide to sprout spores
6. **Verify** → Agents verify signatures and extract
7. **Launch** → Run as isolated OS processes
8. **Register** → Register endpoints with fabric
9. **Route** → Edge gateway routes traffic
10. **Observe** → Monitor health and emit telemetry

## 📁 Project Structure
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

## 🔧 Key APIs

### internal/spore
```go
func Pack(binaryPath string, m Manifest, priv ed25519.PrivateKey, outDir string) (sporePath string, out *Manifest, err error)
func Verify(sporePath string) (*Manifest, error)
func Extract(sporePath, destDir string) (*Manifest, string /*binPath*/, error)
```

### internal/fabric
```go
func (f *Fabric) PublishPlan(p Plan)
func (f *Fabric) SubscribePlans() <-chan Plan
func (f *Fabric) SetBudget(b Budget)
func (f *Fabric) GetBudget(app string) (Budget, bool)
func (f *Fabric) RegisterEndpoint(e Endpoint)
func (f *Fabric) Endpoints(app string) []Endpoint
```

### internal/agent
```go
func New(id string, fab *fabric.Fabric, repo *repo.Repo, runDir string) *Agent
func (a *Agent) Start(ctx context.Context)
```

## 🚀 Quick Start Commands
```bash
# Build workloads
go build -o bin/billing ./cmd/workload-billing
go build -o bin/frontend ./cmd/workload-frontend

# Build a spore
go run ./cmd/mesh build -manifest ./examples/billing.json -binary ./bin/billing -out ./out

# Publish to repo
go run ./cmd/mesh publish -spore $(ls out/*.spore) -repo ./repo

# Run the mesh
go run ./cmd/mesh run -repo ./repo -digest <DIGEST> -app billing -instances 2 -edge :8080 -nodes 3

# Test
curl http://localhost:8080/billing/hello
```

## ✅ Current Status (MVP)
- [x] Spore packaging with Ed25519 signing
- [x] Content-addressed repository
- [x] In-process control fabric
- [x] Node agents that verify and run spores
- [x] Edge proxy for request routing
- [x] Example workloads with health endpoints

## 🚧 Future Work
- [ ] Real decentralized gossip/DHT
- [ ] Nutrient ledger with cgroups/LSM/eBPF enforcement
- [ ] Rolling updates and SLO-aware autoscaling
- [ ] Secrets/config binding
- [ ] Multi-language workload support
- [ ] OTel metrics and tracing

## 🎯 Philosophy
- **Kubernetes = Bureaucracy** (central API server + declarative state)
- **Mycelium Mesh = Ecosystem** (self-organizing spores + nutrient economy)

## 🔒 Security Model
- **Mandatory signature verification** before execution
- **Ed25519 cryptography** for spore signing
- **Binary integrity** checking via SHA-256
- **Future**: Sigstore keyless, SBOM embedded, policy checks

## 📊 Blue/Green Deployment
- **Process**: Launch new spore → Wait for health → Warmup period → Stop old spore
- **State Machine**: Idle → Launching → Ready → Draining → Stopped
- **Zero-downtime** updates through spore versioning

## 🧪 Testing Strategy
- **Unit Tests**: spore, repo, fabric packages
- **Integration Tests**: Build → Publish → Run → curl workflow
- **Update Tests**: Blue/green deployment verification

## 💡 Key Innovations
1. **Containerless**: Native OS processes instead of containers
2. **Decentralized**: No central API server or etcd
3. **Economic**: Resource management through credits
4. **Biological**: Inspired by mycelium network patterns
5. **Security-first**: Cryptographic verification of all deployments
