package repo

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Repo represents a content-addressed repository
type Repo struct {
	Dir string
}

// Open creates or opens a repository
func Open(dir string) (*Repo, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create repo directory: %w", err)
	}
	return &Repo{Dir: dir}, nil
}

// Put stores a spore file and returns its digest
func (r *Repo) Put(sporePath string) (digest string, storedPath string, err error) {
	// Read the spore file
	file, err := os.Open(sporePath)
	if err != nil {
		return "", "", fmt.Errorf("failed to open spore file: %w", err)
	}
	defer file.Close()

	// Compute SHA256 digest
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", "", fmt.Errorf("failed to compute digest: %w", err)
	}

	digest = fmt.Sprintf("%x", hasher.Sum(nil))
	storedPath = filepath.Join(r.Dir, digest+".spore")

	// Copy file to repository
	if err := r.copyFile(sporePath, storedPath); err != nil {
		return "", "", fmt.Errorf("failed to copy file to repo: %w", err)
	}

	return digest, storedPath, nil
}

// Path returns the file path for a given digest
func (r *Repo) Path(digest string) string {
	return filepath.Join(r.Dir, digest+".spore")
}

// copyFile copies a file from src to dst
func (r *Repo) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}
