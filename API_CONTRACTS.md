# Mycelium Mesh — API Contracts (Go-first)

## internal/spore

```go
type Manifest struct {
    Kind        string            `json:"kind"` // "Spore"
    Name        string            `json:"name"`
    Version     string            `json:"version"`
    Command     string            `json:"command"`
    Args        []string          `json:"args"`
    Env         map[string]string `json:"env"`
    Provides    []string          `json:"provides"`
    Nutrients   Nutrients         `json:"nutrients"`
    SLO         SLO               `json:"slo"`
    Security    Security          `json:"security"`
    CreatedAt   time.Time         `json:"created_at"`
    BinarySHA256 string           `json:"binary_sha256"`
    Signature    string           `json:"signature"`  // base64
    PublicKey    string           `json:"public_key"` // base64
}

func Pack(binaryPath string, m Manifest, priv ed25519.PrivateKey, outDir string) (sporePath string, out *Manifest, err error)
func Verify(sporePath string) (*Manifest, error)
func Extract(sporePath, destDir string) (*Manifest, string /*binPath*/, error)
```

## internal/repo
```go
type Repo struct{ Dir string }
func Open(dir string) (*Repo, error)
func (r *Repo) Put(sporePath string) (digest string, storedPath string, err error)
func (r *Repo) Path(digest string) string
```

## internal/fabric
```go
type Plan struct {
    AppName string
    Digest  string
    Min, Max int
    Port    int
}
type Budget struct {
    AppName string
    MaxInstances int
    CPUmilli int
    MemoryMB int
}
type Endpoint struct {
    AppName string
    URL     string // http://127.0.0.1:PORT
    NodeID  string
}

type Fabric struct { /* internal fields */ }
func New() *Fabric
func (f *Fabric) PublishPlan(p Plan)
func (f *Fabric) SubscribePlans() <-chan Plan
func (f *Fabric) SetBudget(b Budget)
func (f *Fabric) GetBudget(app string) (Budget, bool)
func (f *Fabric) RegisterEndpoint(e Endpoint)
func (f *Fabric) Endpoints(app string) []Endpoint
```

## internal/agent
```go
type Agent struct {
    ID     string
    Fab    *fabric.Fabric
    Repo   *repo.Repo
    RunDir string

    // track processes by app and digest
    procs map[string]procInfo
    Warmup time.Duration // default 2s
}

func New(id string, fab *fabric.Fabric, repo *repo.Repo, runDir string) *Agent
func (a *Agent) Start(ctx context.Context)
```

**Blue/Green behavior** (per app):
- If a plan arrives with a new digest:
  - Launch `newDigest`, wait health, sleep `Warmup`, then stop `oldDigest`.
  - Register endpoint for `newDigest`. Only one endpoint per node/app should be visible at a time.

## internal/edge
```go
type Edge struct {
    Fab *fabric.Fabric
    // in-memory counters per app (requests, errors)
}
func New(fab *fabric.Fabric) *Edge
func (e *Edge) Start(addr string) error // blocks in a goroutine
```
