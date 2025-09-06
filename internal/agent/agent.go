package agent

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/karadia10/mycelium-mesh/internal/fabric"
	"github.com/karadia10/mycelium-mesh/internal/repo"
	"github.com/karadia10/mycelium-mesh/internal/spore"
)

// procInfo tracks a running process
type procInfo struct {
	AppName string
	Digest  string
	Process *exec.Cmd
	URL     string
	Port    int
}

// Agent represents a node agent
type Agent struct {
	ID     string
	Fab    *fabric.Fabric
	Repo   *repo.Repo
	RunDir string
	Warmup time.Duration

	mu    sync.RWMutex
	procs map[string]procInfo // appName -> procInfo
}

// New creates a new agent
func New(id string, fab *fabric.Fabric, repo *repo.Repo, runDir string) *Agent {
	return &Agent{
		ID:     id,
		Fab:    fab,
		Repo:   repo,
		RunDir: runDir,
		Warmup: 2 * time.Second,
		procs:  make(map[string]procInfo),
	}
}

// Start starts the agent
func (a *Agent) Start(ctx context.Context) {
	log.Printf("Agent %s starting", a.ID)

	// Subscribe to plans
	planCh := a.Fab.SubscribePlans()

	// Create run directory
	if err := os.MkdirAll(a.RunDir, 0755); err != nil {
		log.Printf("Failed to create run directory: %v", err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			log.Printf("Agent %s stopping", a.ID)
			a.stopAllProcesses()
			return
		case plan := <-planCh:
			a.handlePlan(plan)
		}
	}
}

// handlePlan handles a deployment plan
func (a *Agent) handlePlan(plan fabric.Plan) {
	log.Printf("Agent %s received plan for app %s, digest %s", a.ID, plan.AppName, plan.Digest)

	a.mu.Lock()
	defer a.mu.Unlock()

	// Check if we already have this app running
	if proc, exists := a.procs[plan.AppName]; exists {
		if proc.Digest == plan.Digest {
			log.Printf("App %s already running with digest %s", plan.AppName, plan.Digest)
			return
		}

		// Blue/green deployment: new digest
		log.Printf("Starting blue/green deployment for app %s", plan.AppName)
		go a.blueGreenDeploy(plan, proc)
		return
	}

	// Check budget
	budget, exists := a.Fab.GetBudget(plan.AppName)
	if !exists {
		log.Printf("No budget found for app %s", plan.AppName)
		return
	}

	// Check if we can run more instances
	if len(a.procs) >= budget.MaxInstances {
		log.Printf("Budget limit reached for app %s", plan.AppName)
		return
	}

	// Launch new process
	go a.launchProcess(plan)
}

// blueGreenDeploy handles blue/green deployment
func (a *Agent) blueGreenDeploy(plan fabric.Plan, oldProc procInfo) {
	// Launch new process
	newProc, err := a.sproutProcess(plan)
	if err != nil {
		log.Printf("Failed to launch new process for app %s: %v", plan.AppName, err)
		return
	}

	// Wait for warmup period
	time.Sleep(a.Warmup)

	// Update fabric with new endpoint
	a.Fab.RegisterEndpoint(fabric.Endpoint{
		AppName: plan.AppName,
		URL:     newProc.URL,
		NodeID:  a.ID,
	})

	// Update our process tracking
	a.mu.Lock()
	a.procs[plan.AppName] = newProc
	a.mu.Unlock()

	// Stop old process
	log.Printf("Stopping old process for app %s", plan.AppName)
	if oldProc.Process != nil && oldProc.Process.Process != nil {
		oldProc.Process.Process.Kill()
	}
}

// launchProcess launches a new process
func (a *Agent) launchProcess(plan fabric.Plan) {
	proc, err := a.sproutProcess(plan)
	if err != nil {
		log.Printf("Failed to launch process for app %s: %v", plan.AppName, err)
		return
	}

	// Update process tracking
	a.mu.Lock()
	a.procs[plan.AppName] = proc
	a.mu.Unlock()

	// Register endpoint
	a.Fab.RegisterEndpoint(fabric.Endpoint{
		AppName: plan.AppName,
		URL:     proc.URL,
		NodeID:  a.ID,
	})

	log.Printf("Successfully launched app %s on %s", plan.AppName, proc.URL)
}

// sproutProcess sprouts a spore as a process
func (a *Agent) sproutProcess(plan fabric.Plan) (procInfo, error) {
	// Get spore path from repo
	sporePath := a.Repo.Path(plan.Digest)
	if _, err := os.Stat(sporePath); err != nil {
		return procInfo{}, fmt.Errorf("spore not found: %w", err)
	}

	// Verify spore
	manifest, err := spore.Verify(sporePath)
	if err != nil {
		return procInfo{}, fmt.Errorf("spore verification failed: %w", err)
	}

	// Extract spore
	extractDir := filepath.Join(a.RunDir, fmt.Sprintf("%s-%s-%d", plan.AppName, plan.Digest[:8], time.Now().Unix()))
	_, binaryPath, err := spore.Extract(sporePath, extractDir)
	if err != nil {
		return procInfo{}, fmt.Errorf("spore extraction failed: %w", err)
	}

	// Find free port
	port, err := a.findFreePort()
	if err != nil {
		return procInfo{}, fmt.Errorf("failed to find free port: %w", err)
	}

	// Prepare environment
	env := os.Environ()
	for k, v := range manifest.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	env = append(env, fmt.Sprintf("PORT=%d", port))

	// Create command - ensure binary path is absolute
	absBinaryPath, err := filepath.Abs(binaryPath)
	if err != nil {
		return procInfo{}, fmt.Errorf("failed to get absolute path for binary: %w", err)
	}

	log.Printf("Launching binary: %s in dir: %s", absBinaryPath, extractDir)
	cmd := exec.Command(absBinaryPath, manifest.Args...)
	cmd.Dir = extractDir
	cmd.Env = env

	// Start process
	if err := cmd.Start(); err != nil {
		return procInfo{}, fmt.Errorf("failed to start process: %w", err)
	}

	url := fmt.Sprintf("http://127.0.0.1:%d", port)

	// Wait for health check
	if err := a.waitForHealth(url, 6*time.Second); err != nil {
		cmd.Process.Kill()
		return procInfo{}, fmt.Errorf("health check failed: %w", err)
	}

	return procInfo{
		AppName: plan.AppName,
		Digest:  plan.Digest,
		Process: cmd,
		URL:     url,
		Port:    port,
	}, nil
}

// findFreePort finds a free TCP port
func (a *Agent) findFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()

	return l.Addr().(*net.TCPAddr).Port, nil
}

// waitForHealth waits for a service to become healthy
func (a *Agent) waitForHealth(url string, timeout time.Duration) error {
	client := &http.Client{Timeout: 1 * time.Second}
	healthURL := url + "/health"

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := client.Get(healthURL)
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			return nil
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("health check timeout after %v", timeout)
}

// stopAllProcesses stops all running processes
func (a *Agent) stopAllProcesses() {
	a.mu.Lock()
	defer a.mu.Unlock()

	for appName, proc := range a.procs {
		log.Printf("Stopping process for app %s", appName)
		if proc.Process != nil && proc.Process.Process != nil {
			proc.Process.Process.Kill()
		}
	}
}
