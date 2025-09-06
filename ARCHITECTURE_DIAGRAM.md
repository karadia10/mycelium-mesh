# Mycelium Mesh - Architecture Diagram

## System Overview

```mermaid
graph TB
    subgraph "Developer Workflow"
        DEV[Developer] --> BUILD[Build Binary]
        BUILD --> PACK[Pack Spore]
        PACK --> PUB[Publish to Repo]
    end
    
    subgraph "Mycelium Mesh Cluster"
        subgraph "Control Plane"
            FABRIC[Control Fabric<br/>📡 Pub/Sub + Registry]
            BUDGET[Nutrient Ledger<br/>💰 Resource Budgets]
        end
        
        subgraph "Edge Layer"
            EDGE[Edge Gateway<br/>🌐 Reverse Proxy]
        end
        
        subgraph "Node Layer"
            AGENT1[Agent Node-1<br/>🤖 Spore Runner]
            AGENT2[Agent Node-2<br/>🤖 Spore Runner]
            AGENT3[Agent Node-N<br/>🤖 Spore Runner]
        end
        
        subgraph "Storage Layer"
            REPO[Content Repository<br/>📦 SHA-256 Addressed]
        end
    end
    
    subgraph "Running Workloads"
        PROC1[Process 1<br/>🍄 Billing Service]
        PROC2[Process 2<br/>🍄 Frontend Service]
        PROC3[Process N<br/>🍄 Other Services]
    end
    
    subgraph "External Clients"
        CLIENT[HTTP Client<br/>🌍 curl/browser]
    end
    
    %% Developer to Storage
    PUB --> REPO
    
    %% Control Plane connections
    FABRIC -.-> BUDGET
    FABRIC --> AGENT1
    FABRIC --> AGENT2
    FABRIC --> AGENT3
    
    %% Storage to Agents
    REPO --> AGENT1
    REPO --> AGENT2
    REPO --> AGENT3
    
    %% Agents to Processes
    AGENT1 --> PROC1
    AGENT2 --> PROC2
    AGENT3 --> PROC3
    
    %% Edge to Control Plane
    EDGE --> FABRIC
    
    %% Client to Edge
    CLIENT --> EDGE
    
    %% Edge to Processes (via Control Plane)
    EDGE -.-> PROC1
    EDGE -.-> PROC2
    EDGE -.-> PROC3
    
    %% Styling
    classDef controlPlane fill:#e1f5fe
    classDef storage fill:#f3e5f5
    classDef agent fill:#e8f5e8
    classDef process fill:#fff3e0
    classDef client fill:#fce4ec
    
    class FABRIC,BUDGET controlPlane
    class REPO storage
    class AGENT1,AGENT2,AGENT3 agent
    class PROC1,PROC2,PROC3 process
    class CLIENT client
```

## Component Details

### 🍄 Spore Lifecycle

```mermaid
sequenceDiagram
    participant DEV as Developer
    participant CLI as mesh CLI
    participant SPORE as Spore Packer
    participant REPO as Repository
    participant FABRIC as Control Fabric
    participant AGENT as Node Agent
    participant PROC as Process
    participant EDGE as Edge Gateway
    participant CLIENT as Client
    
    %% Build Phase
    DEV->>CLI: go build workload
    CLI->>SPORE: Pack(binary, manifest, key)
    SPORE->>SPORE: Sign with Ed25519
    SPORE-->>CLI: .spore file
    
    %% Publish Phase
    CLI->>REPO: Put(spore)
    REPO->>REPO: Compute SHA-256
    REPO-->>CLI: digest
    
    %% Deploy Phase
    CLI->>FABRIC: PublishPlan(app, digest)
    FABRIC->>AGENT: Plan notification
    AGENT->>REPO: Get spore by digest
    AGENT->>SPORE: Verify signature
    AGENT->>SPORE: Extract to run dir
    AGENT->>PROC: Start process
    AGENT->>FABRIC: Register endpoint
    
    %% Runtime Phase
    CLIENT->>EDGE: GET /app/hello
    EDGE->>FABRIC: Get endpoints
    FABRIC-->>EDGE: endpoint list
    EDGE->>PROC: Proxy request
    PROC-->>EDGE: Response
    EDGE-->>CLIENT: Response
```

### 🏗️ Data Structures

```mermaid
classDiagram
    class Spore {
        +string Name
        +string Version
        +string Command
        +string[] Args
        +map[string]string Env
        +Nutrients Resources
        +SLO Performance
        +Security Security
        +string BinarySHA256
        +string Signature
        +string PublicKey
        +time CreatedAt
    }
    
    class Manifest {
        +string Kind
        +string Name
        +string Version
        +string Command
        +string[] Args
        +map[string]string Env
        +Nutrients Nutrients
        +SLO SLO
        +Security Security
        +time CreatedAt
        +string BinarySHA256
        +string Signature
        +string PublicKey
    }
    
    class Plan {
        +string AppName
        +string Digest
        +int Min
        +int Max
        +int Port
    }
    
    class Budget {
        +string AppName
        +int MaxInstances
        +int CPUmilli
        +int MemoryMB
    }
    
    class Endpoint {
        +string AppName
        +string URL
        +string NodeID
    }
    
    class Agent {
        +string ID
        +Fabric Fab
        +Repo Repo
        +string RunDir
        +time Duration Warmup
        +map[string]procInfo procs
        +Start(ctx) void
        +handlePlan(plan) void
        +sproutProcess(plan) procInfo
    }
    
    class Edge {
        +Fabric Fab
        +map[string]atomic.Int64 counters
        +map[string]atomic.Int64 errors
        +Start(addr) error
        +handleRequest(w, r) void
    }
    
    Spore --> Manifest
    Agent --> Plan
    Agent --> Budget
    Agent --> Endpoint
    Edge --> Plan
    Edge --> Endpoint
```

### 🔄 Process State Machine

```mermaid
stateDiagram-v2
    [*] --> Idle : Agent starts
    
    Idle --> Launching : Plan received
    Launching --> Ready : Health check passed
    Launching --> Failed : Health check failed
    Failed --> Idle : Cleanup
    
    Ready --> Launching : New digest plan
    Ready --> Draining : New digest plan
    Draining --> Stopped : Old process killed
    Stopped --> Ready : New process ready
    
    Ready --> Stopped : Process exit
    Stopped --> Idle : Cleanup
    
    note right of Ready
        Process running
        Endpoint registered
        Serving requests
    end note
    
    note right of Draining
        New process ready
        Old process stopping
        Blue/Green deployment
    end note
```

### 🌐 Network Topology

```mermaid
graph LR
    subgraph "External Network"
        CLIENT[Client Applications]
    end
    
    subgraph "Edge Layer"
        EDGE[Edge Gateway<br/>:8080]
    end
    
    subgraph "Control Plane"
        FABRIC[Control Fabric<br/>In-Process Pub/Sub]
    end
    
    subgraph "Node 1"
        AGENT1[Agent]
        PROC1[Process :8081]
    end
    
    subgraph "Node 2"
        AGENT2[Agent]
        PROC2[Process :8082]
    end
    
    subgraph "Node 3"
        AGENT3[Agent]
        PROC3[Process :8083]
    end
    
    subgraph "Storage"
        REPO[Content Repository<br/>Local Filesystem]
    end
    
    CLIENT -->|HTTP| EDGE
    EDGE -->|Round Robin| PROC1
    EDGE -->|Round Robin| PROC2
    EDGE -->|Round Robin| PROC3
    
    FABRIC -.->|Plans| AGENT1
    FABRIC -.->|Plans| AGENT2
    FABRIC -.->|Plans| AGENT3
    
    AGENT1 -->|Register| FABRIC
    AGENT2 -->|Register| FABRIC
    AGENT3 -->|Register| FABRIC
    
    AGENT1 -->|Pull| REPO
    AGENT2 -->|Pull| REPO
    AGENT3 -->|Pull| REPO
    
    AGENT1 -->|Spawn| PROC1
    AGENT2 -->|Spawn| PROC2
    AGENT3 -->|Spawn| PROC3
```

### 🔐 Security Model

```mermaid
graph TB
    subgraph "Spore Creation"
        BINARY[Binary File]
        MANIFEST[Manifest JSON]
        PRIVKEY[Private Key]
        PACKER[Spore Packer]
    end
    
    subgraph "Spore Verification"
        SPORE[.spore File]
        VERIFIER[Signature Verifier]
        PUBKEY[Public Key]
        HASH[Binary Hash Check]
    end
    
    subgraph "Process Execution"
        EXTRACT[Extract to Run Dir]
        CHMOD[Make Executable]
        SPAWN[Start Process]
    end
    
    BINARY --> PACKER
    MANIFEST --> PACKER
    PRIVKEY --> PACKER
    PACKER --> SPORE
    
    SPORE --> VERIFIER
    VERIFIER --> PUBKEY
    VERIFIER --> HASH
    VERIFIER --> EXTRACT
    
    EXTRACT --> CHMOD
    CHMOD --> SPAWN
    
    %% Security checks
    PACKER -.->|Ed25519 Sign| SPORE
    VERIFIER -.->|Ed25519 Verify| PUBKEY
    VERIFIER -.->|SHA-256 Check| HASH
```

### 📊 Resource Management

```mermaid
graph LR
    subgraph "Nutrient Economy"
        BUDGET[Budget Definition<br/>Max Instances: 6<br/>CPU: 1000m<br/>Memory: 512MB]
        CREDITS[Resource Credits<br/>Per App Allocation]
        ENFORCEMENT[Future: cgroups<br/>eBPF, LSM]
    end
    
    subgraph "Agent Decision"
        PLAN[Deployment Plan]
        CHECK[Budget Check]
        LAUNCH[Launch Process]
        REGISTER[Register Endpoint]
    end
    
    subgraph "Process Lifecycle"
        SPAWN[Spawn Process]
        MONITOR[Monitor Health]
        RECYCLE[Recycle on Exit]
    end
    
    BUDGET --> PLAN
    PLAN --> CHECK
    CHECK --> LAUNCH
    LAUNCH --> REGISTER
    
    LAUNCH --> SPAWN
    SPAWN --> MONITOR
    MONITOR --> RECYCLE
    RECYCLE --> SPAWN
    
    CREDITS -.->|Future| ENFORCEMENT
    ENFORCEMENT -.->|Future| SPAWN
```

## Key Architectural Principles

### 🎯 **Decentralized Design**
- No single point of failure
- Self-organizing spore ecosystem
- Peer-to-peer communication patterns

### 🔒 **Security First**
- Mandatory cryptographic verification
- Ed25519 signature validation
- Binary integrity checking

### 💰 **Economic Resource Model**
- Credit-based resource allocation
- Dynamic scaling based on budgets
- Future: cgroups/eBPF enforcement

### 🍄 **Biological Inspiration**
- Mycelium network patterns
- Self-healing and adaptive
- Organic growth and evolution

### ⚡ **Containerless Architecture**
- Native OS processes
- Reduced overhead
- Simplified deployment model

---

*This architecture represents a fundamental shift from container-based orchestration to a more organic, decentralized approach inspired by biological systems.*
