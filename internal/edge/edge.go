package edge

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/karadia10/mycelium-mesh/internal/fabric"
)

// Edge represents the reverse proxy edge gateway
type Edge struct {
	Fab      *fabric.Fabric
	counters map[string]*atomic.Int64 // appName -> request count
	errors   map[string]*atomic.Int64 // appName -> error count
}

// New creates a new edge
func New(fab *fabric.Fabric) *Edge {
	return &Edge{
		Fab:      fab,
		counters: make(map[string]*atomic.Int64),
		errors:   make(map[string]*atomic.Int64),
	}
}

// Start starts the edge server
func (e *Edge) Start(addr string) error {
	// Start counter logging goroutine
	go e.logCounters()

	// Create HTTP mux
	mux := http.NewServeMux()
	mux.HandleFunc("/", e.handleRequest)

	// Start server
	log.Printf("Edge server starting on %s", addr)
	return http.ListenAndServe(addr, mux)
}

// handleRequest handles incoming requests
func (e *Edge) handleRequest(w http.ResponseWriter, r *http.Request) {
	// Parse app name from path
	path := r.URL.Path
	if len(path) < 2 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	// Extract app name (first segment after /)
	var appName string
	for i := 1; i < len(path); i++ {
		if path[i] == '/' {
			appName = path[1:i]
			break
		}
	}

	if appName == "" {
		http.Error(w, "No app name in path", http.StatusBadRequest)
		return
	}

	// Get endpoints for this app
	endpoints := e.Fab.Endpoints(appName)
	if len(endpoints) == 0 {
		http.Error(w, fmt.Sprintf("No endpoints available for app %s", appName), http.StatusServiceUnavailable)
		e.incrementError(appName)
		return
	}

	// Select endpoint using round-robin
	endpoint := e.selectEndpoint(endpoints)

	// Increment counter
	e.incrementCounter(appName)

	// Add Mycelium-Edge header
	w.Header().Set("X-Mycelium-Edge", time.Now().Format(time.RFC3339))

	// Create reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(&url.URL{
		Scheme: "http",
		Host:   endpoint.URL[7:], // Remove "http://" prefix
	})

	// Update request path to remove app name
	r.URL.Path = "/" + path[len(appName)+2:]
	if r.URL.Path == "//" {
		r.URL.Path = "/"
	}

	// Proxy the request
	proxy.ServeHTTP(w, r)
}

// selectEndpoint selects an endpoint using round-robin
func (e *Edge) selectEndpoint(endpoints []fabric.Endpoint) fabric.Endpoint {
	// Simple round-robin using atomic counter
	// In a real implementation, this would be more sophisticated
	return endpoints[0] // For now, just return the first endpoint
}

// incrementCounter increments the request counter for an app
func (e *Edge) incrementCounter(appName string) {
	if counter, exists := e.counters[appName]; exists {
		counter.Add(1)
	} else {
		newCounter := &atomic.Int64{}
		newCounter.Add(1)
		e.counters[appName] = newCounter
	}
}

// incrementError increments the error counter for an app
func (e *Edge) incrementError(appName string) {
	if counter, exists := e.errors[appName]; exists {
		counter.Add(1)
	} else {
		newCounter := &atomic.Int64{}
		newCounter.Add(1)
		e.errors[appName] = newCounter
	}
}

// logCounters logs counters every 10 seconds
func (e *Edge) logCounters() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		log.Println("=== Edge Counters ===")
		for appName, counter := range e.counters {
			errorCount := int64(0)
			if errorCounter, exists := e.errors[appName]; exists {
				errorCount = errorCounter.Load()
			}
			log.Printf("App %s: %d requests, %d errors", appName, counter.Load(), errorCount)
		}
		log.Println("===================")
	}
}
