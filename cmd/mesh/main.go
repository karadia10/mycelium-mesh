package main

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/karadia10/mycelium-mesh/internal/agent"
	"github.com/karadia10/mycelium-mesh/internal/edge"
	"github.com/karadia10/mycelium-mesh/internal/fabric"
	"github.com/karadia10/mycelium-mesh/internal/repo"
	"github.com/karadia10/mycelium-mesh/internal/spore"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	os.Args = os.Args[1:]

	switch command {
	case "build":
		buildCommand()
	case "publish":
		publishCommand()
	case "run":
		runCommand()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: mesh <command> [flags]")
	fmt.Println("Commands:")
	fmt.Println("  build    - Build a spore from binary and manifest")
	fmt.Println("  publish  - Publish a spore to repository")
	fmt.Println("  run      - Run the mesh with edge and agents")
	fmt.Println("")
	fmt.Println("Use 'mesh <command> -h' for command-specific help")
}

func buildCommand() {
	var (
		manifestPath = flag.String("manifest", "", "Path to manifest JSON file")
		binaryPath   = flag.String("binary", "", "Path to binary file")
		outDir       = flag.String("out", "./out", "Output directory for spore")
		keyPath      = flag.String("key", "", "Path to private key file (generates new if not provided)")
	)
	flag.Parse()

	if *manifestPath == "" || *binaryPath == "" {
		fmt.Println("Error: -manifest and -binary are required")
		flag.Usage()
		os.Exit(1)
	}

	// Read manifest
	manifestData, err := os.ReadFile(*manifestPath)
	if err != nil {
		log.Fatalf("Failed to read manifest: %v", err)
	}

	var manifest spore.Manifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		log.Fatalf("Failed to parse manifest: %v", err)
	}

	// Generate or load private key
	var privKey ed25519.PrivateKey
	if *keyPath != "" {
		keyData, err := os.ReadFile(*keyPath)
		if err != nil {
			log.Fatalf("Failed to read key file: %v", err)
		}
		privKey = ed25519.PrivateKey(keyData)
	} else {
		_, privKey, err = ed25519.GenerateKey(nil)
		if err != nil {
			log.Fatalf("Failed to generate key: %v", err)
		}
		// Save key for future use
		keyPath := filepath.Join(*outDir, "private.key")
		if err := os.MkdirAll(*outDir, 0755); err != nil {
			log.Fatalf("Failed to create output directory: %v", err)
		}
		if err := os.WriteFile(keyPath, privKey, 0600); err != nil {
			log.Fatalf("Failed to save private key: %v", err)
		}
		log.Printf("Generated private key saved to %s", keyPath)
	}

	// Pack spore
	sporePath, finalManifest, err := spore.Pack(*binaryPath, manifest, privKey, *outDir)
	if err != nil {
		log.Fatalf("Failed to pack spore: %v", err)
	}

	log.Printf("Spore created: %s", sporePath)
	log.Printf("Manifest: %+v", finalManifest)
}

func publishCommand() {
	var (
		sporePath = flag.String("spore", "", "Path to spore file")
		repoDir   = flag.String("repo", "./repo", "Repository directory")
	)
	flag.Parse()

	if *sporePath == "" {
		fmt.Println("Error: -spore is required")
		flag.Usage()
		os.Exit(1)
	}

	// Open repository
	repo, err := repo.Open(*repoDir)
	if err != nil {
		log.Fatalf("Failed to open repository: %v", err)
	}

	// Publish spore
	digest, storedPath, err := repo.Put(*sporePath)
	if err != nil {
		log.Fatalf("Failed to publish spore: %v", err)
	}

	log.Printf("Spore published successfully")
	log.Printf("Digest: %s", digest)
	log.Printf("Stored at: %s", storedPath)
}

func runCommand() {
	var (
		repoDir   = flag.String("repo", "./repo", "Repository directory")
		digest    = flag.String("digest", "", "Spore digest to run")
		appName   = flag.String("app", "", "App name")
		instances = flag.Int("instances", 2, "Number of instances to run")
		edgeAddr  = flag.String("edge", ":8080", "Edge server address")
		nodes     = flag.Int("nodes", 3, "Number of agent nodes")
		warmup    = flag.Duration("warmup", 2*time.Second, "Blue/green warmup duration")
	)
	flag.Parse()

	if *digest == "" || *appName == "" {
		fmt.Println("Error: -digest and -app are required")
		flag.Usage()
		os.Exit(1)
	}

	// Open repository
	repo, err := repo.Open(*repoDir)
	if err != nil {
		log.Fatalf("Failed to open repository: %v", err)
	}

	// Create fabric
	fab := fabric.New()

	// Set budget
	fab.SetBudget(fabric.Budget{
		AppName:      *appName,
		MaxInstances: *instances * *nodes,
		CPUmilli:     1000,
		MemoryMB:     512,
	})

	// Create edge
	edge := edge.New(fab)

	// Start edge in goroutine
	go func() {
		if err := edge.Start(*edgeAddr); err != nil {
			log.Fatalf("Edge server failed: %v", err)
		}
	}()

	// Create and start agents
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for i := 0; i < *nodes; i++ {
		agentID := fmt.Sprintf("node-%d", i+1)
		runDir := filepath.Join("./run", agentID)

		ag := agent.New(agentID, fab, repo, runDir)
		ag.Warmup = *warmup

		go ag.Start(ctx)
	}

	// Publish plan
	plan := fabric.Plan{
		AppName: *appName,
		Digest:  *digest,
		Min:     *instances,
		Max:     *instances * *nodes,
		Port:    0, // Let agents choose ports
	}

	log.Printf("Publishing plan: %+v", plan)
	fab.PublishPlan(plan)

	log.Printf("Mesh running with %d agents", *nodes)
	log.Printf("Edge server: http://localhost%s", *edgeAddr)
	log.Printf("Test with: curl http://localhost%s/%s/hello", *edgeAddr, *appName)

	// Wait for interrupt
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down...")
	cancel()
	time.Sleep(1 * time.Second)
}
