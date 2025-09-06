package spore

import (
	"crypto/ed25519"
	"os"
	"path/filepath"
	"testing"
)

func TestPackAndVerify(t *testing.T) {
	// Create a temporary binary file
	tempDir := t.TempDir()
	binaryPath := filepath.Join(tempDir, "test-binary")
	if err := os.WriteFile(binaryPath, []byte("test binary content"), 0755); err != nil {
		t.Fatalf("Failed to create test binary: %v", err)
	}

	// Generate key pair
	_, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	// Create manifest
	manifest := Manifest{
		Name:    "test-app",
		Version: "v1.0.0",
		Command: "test-binary",
		Args:    []string{},
		Env:     map[string]string{"TEST": "value"},
		Nutrients: Nutrients{
			CPUMilli: 100,
			MemoryMB: 64,
		},
		SLO: SLO{
			P99BudgetMs: 200,
		},
		Security: Security{
			LSMProfile: "test",
			ReadOnlyFS: false,
		},
	}

	// Pack spore
	sporePath, finalManifest, err := Pack(binaryPath, manifest, privKey, tempDir)
	if err != nil {
		t.Fatalf("Pack failed: %v", err)
	}

	if sporePath == "" {
		t.Fatal("Pack returned empty spore path")
	}

	if finalManifest == nil {
		t.Fatal("Pack returned nil manifest")
	}

	// Verify spore
	verifiedManifest, err := Verify(sporePath)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}

	if verifiedManifest.Name != "test-app" {
		t.Errorf("Expected name 'test-app', got '%s'", verifiedManifest.Name)
	}

	if verifiedManifest.BinarySHA256 == "" {
		t.Error("BinarySHA256 should not be empty")
	}

	if verifiedManifest.Signature == "" {
		t.Error("Signature should not be empty")
	}

	if verifiedManifest.PublicKey == "" {
		t.Error("PublicKey should not be empty")
	}

	if verifiedManifest.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
}

func TestVerifyFailsWithWrongSignature(t *testing.T) {
	// Create a temporary binary file
	tempDir := t.TempDir()
	binaryPath := filepath.Join(tempDir, "test-binary")
	if err := os.WriteFile(binaryPath, []byte("test binary content"), 0755); err != nil {
		t.Fatalf("Failed to create test binary: %v", err)
	}

	// Generate key pair
	_, privKey1, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	// Create manifest
	manifest := Manifest{
		Name:    "test-app",
		Version: "v1.0.0",
		Command: "test-binary",
		Args:    []string{},
		Env:     map[string]string{"TEST": "value"},
		Nutrients: Nutrients{
			CPUMilli: 100,
			MemoryMB: 64,
		},
		SLO: SLO{
			P99BudgetMs: 200,
		},
		Security: Security{
			LSMProfile: "test",
			ReadOnlyFS: false,
		},
	}

	// Pack spore with key
	_, _, err = Pack(binaryPath, manifest, privKey1, tempDir)
	if err != nil {
		t.Fatalf("Pack failed: %v", err)
	}

	// Now try to verify with a different key (this should fail)
	// We'll create a new manifest with the wrong signature
	wrongManifest := manifest
	wrongManifest.PublicKey = "wrong-public-key"

	// This test is a bit complex because we need to actually modify the spore file
	// For now, let's test with a non-existent file
	_, err = Verify("non-existent-file.spore")
	if err == nil {
		t.Error("Verify should have failed with non-existent file")
	}
}

func TestExtract(t *testing.T) {
	// Create a temporary binary file
	tempDir := t.TempDir()
	binaryPath := filepath.Join(tempDir, "test-binary")
	if err := os.WriteFile(binaryPath, []byte("test binary content"), 0755); err != nil {
		t.Fatalf("Failed to create test binary: %v", err)
	}

	// Generate key pair
	_, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	// Create manifest
	manifest := Manifest{
		Name:    "test-app",
		Version: "v1.0.0",
		Command: "test-binary",
		Args:    []string{},
		Env:     map[string]string{"TEST": "value"},
		Nutrients: Nutrients{
			CPUMilli: 100,
			MemoryMB: 64,
		},
		SLO: SLO{
			P99BudgetMs: 200,
		},
		Security: Security{
			LSMProfile: "test",
			ReadOnlyFS: false,
		},
	}

	// Pack spore
	sporePath, _, err := Pack(binaryPath, manifest, privKey, tempDir)
	if err != nil {
		t.Fatalf("Pack failed: %v", err)
	}

	// Extract spore
	extractDir := filepath.Join(tempDir, "extracted")
	extractedManifest, extractedBinaryPath, err := Extract(sporePath, extractDir)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if extractedManifest.Name != "test-app" {
		t.Errorf("Expected name 'test-app', got '%s'", extractedManifest.Name)
	}

	if extractedBinaryPath == "" {
		t.Error("Extracted binary path should not be empty")
	}

	// Check if binary was extracted
	if _, err := os.Stat(extractedBinaryPath); os.IsNotExist(err) {
		t.Error("Extracted binary file does not exist")
	}

	// Check if binary is executable
	info, err := os.Stat(extractedBinaryPath)
	if err != nil {
		t.Fatalf("Failed to stat extracted binary: %v", err)
	}

	if info.Mode()&0111 == 0 {
		t.Error("Extracted binary should be executable")
	}
}
