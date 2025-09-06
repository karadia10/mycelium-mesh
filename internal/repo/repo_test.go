package repo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPutAndPath(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Open repository
	repo, err := Open(tempDir)
	if err != nil {
		t.Fatalf("Failed to open repository: %v", err)
	}

	// Create a test file
	testFile := filepath.Join(tempDir, "test-file.txt")
	content := "test content"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Put file in repository
	digest, storedPath, err := repo.Put(testFile)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	if digest == "" {
		t.Error("Digest should not be empty")
	}

	if storedPath == "" {
		t.Error("Stored path should not be empty")
	}

	// Check if file was stored
	if _, err := os.Stat(storedPath); os.IsNotExist(err) {
		t.Error("Stored file does not exist")
	}

	// Test Path method
	retrievedPath := repo.Path(digest)
	if retrievedPath != storedPath {
		t.Errorf("Path returned %s, expected %s", retrievedPath, storedPath)
	}

	// Test idempotent Put (same content should produce same digest)
	digest2, storedPath2, err := repo.Put(testFile)
	if err != nil {
		t.Fatalf("Second Put failed: %v", err)
	}

	if digest != digest2 {
		t.Error("Same content should produce same digest")
	}

	if storedPath != storedPath2 {
		t.Error("Same content should produce same stored path")
	}
}

func TestPutWithDifferentContent(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Open repository
	repo, err := Open(tempDir)
	if err != nil {
		t.Fatalf("Failed to open repository: %v", err)
	}

	// Create first test file
	testFile1 := filepath.Join(tempDir, "test-file1.txt")
	content1 := "test content 1"
	if err := os.WriteFile(testFile1, []byte(content1), 0644); err != nil {
		t.Fatalf("Failed to create test file 1: %v", err)
	}

	// Create second test file with different content
	testFile2 := filepath.Join(tempDir, "test-file2.txt")
	content2 := "test content 2"
	if err := os.WriteFile(testFile2, []byte(content2), 0644); err != nil {
		t.Fatalf("Failed to create test file 2: %v", err)
	}

	// Put both files
	digest1, _, err := repo.Put(testFile1)
	if err != nil {
		t.Fatalf("Put file 1 failed: %v", err)
	}

	digest2, _, err := repo.Put(testFile2)
	if err != nil {
		t.Fatalf("Put file 2 failed: %v", err)
	}

	// Digests should be different
	if digest1 == digest2 {
		t.Error("Different content should produce different digests")
	}
}
