package fabric

import (
	"sync"
)

// Plan represents a deployment plan
type Plan struct {
	AppName string
	Digest  string
	Min     int
	Max     int
	Port    int
}

// Budget represents resource budget for an app
type Budget struct {
	AppName      string
	MaxInstances int
	CPUmilli     int
	MemoryMB     int
}

// Endpoint represents a running service endpoint
type Endpoint struct {
	AppName string
	URL     string // http://127.0.0.1:PORT
	NodeID  string
}

// Fabric represents the control fabric
type Fabric struct {
	mu          sync.RWMutex
	plans       chan Plan
	budgets     map[string]Budget
	endpoints   map[string][]Endpoint // appName -> endpoints
	subscribers []chan Plan
}

// New creates a new fabric
func New() *Fabric {
	return &Fabric{
		plans:       make(chan Plan, 100),
		budgets:     make(map[string]Budget),
		endpoints:   make(map[string][]Endpoint),
		subscribers: make([]chan Plan, 0),
	}
}

// PublishPlan publishes a plan to all subscribers
func (f *Fabric) PublishPlan(p Plan) {
	select {
	case f.plans <- p:
		// Plan published successfully
	default:
		// Channel is full, drop the plan (non-blocking)
	}
}

// SubscribePlans returns a channel for receiving plans
func (f *Fabric) SubscribePlans() <-chan Plan {
	f.mu.Lock()
	defer f.mu.Unlock()

	ch := make(chan Plan, 10)
	f.subscribers = append(f.subscribers, ch)

	// Start goroutine to forward plans to this subscriber
	go func() {
		for plan := range f.plans {
			select {
			case ch <- plan:
			default:
				// Subscriber is not ready, drop the plan
			}
		}
	}()

	return ch
}

// SetBudget sets the budget for an app
func (f *Fabric) SetBudget(b Budget) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.budgets[b.AppName] = b
}

// GetBudget gets the budget for an app
func (f *Fabric) GetBudget(app string) (Budget, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	budget, exists := f.budgets[app]
	return budget, exists
}

// RegisterEndpoint registers an endpoint for an app
func (f *Fabric) RegisterEndpoint(e Endpoint) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Remove any existing endpoint for this node/app combination
	endpoints := f.endpoints[e.AppName]
	for i, ep := range endpoints {
		if ep.NodeID == e.NodeID {
			// Replace existing endpoint
			endpoints[i] = e
			f.endpoints[e.AppName] = endpoints
			return
		}
	}

	// Add new endpoint
	f.endpoints[e.AppName] = append(endpoints, e)
}

// Endpoints returns all endpoints for an app
func (f *Fabric) Endpoints(app string) []Endpoint {
	f.mu.RLock()
	defer f.mu.RUnlock()

	endpoints, exists := f.endpoints[app]
	if !exists {
		return nil
	}

	// Return a copy to prevent external mutation
	result := make([]Endpoint, len(endpoints))
	copy(result, endpoints)
	return result
}
