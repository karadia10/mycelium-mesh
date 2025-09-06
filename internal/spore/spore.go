package spore

import (
	"archive/zip"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// Manifest represents the DNA of a spore
type Manifest struct {
	Kind         string            `json:"kind"` // "Spore"
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Command      string            `json:"command"`
	Args         []string          `json:"args"`
	Env          map[string]string `json:"env"`
	Provides     []string          `json:"provides"`
	Nutrients    Nutrients         `json:"nutrients"`
	SLO          SLO               `json:"slo"`
	Security     Security          `json:"security"`
	CreatedAt    time.Time         `json:"created_at"`
	BinarySHA256 string            `json:"binary_sha256"`
	Signature    string            `json:"signature"`  // base64
	PublicKey    string            `json:"public_key"` // base64
}

type Nutrients struct {
	CPUMilli int `json:"cpu_milli"`
	MemoryMB int `json:"memory_mb"`
}

type SLO struct {
	P99BudgetMs int `json:"p99_budget_ms"`
}

type Security struct {
	LSMProfile string `json:"lsm_profile"`
	ReadOnlyFS bool   `json:"read_only_fs"`
}

// Pack creates a signed spore bundle
func Pack(binaryPath string, m Manifest, priv ed25519.PrivateKey, outDir string) (sporePath string, out *Manifest, err error) {
	// Read the binary file
	binaryData, err := os.ReadFile(binaryPath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read binary: %w", err)
	}

	// Compute binary SHA256
	hash := sha256.Sum256(binaryData)
	m.BinarySHA256 = fmt.Sprintf("%x", hash)

	// Set creation time
	m.CreatedAt = time.Now()
	m.Kind = "Spore"

	// Get public key
	pubKey := priv.Public().(ed25519.PublicKey)
	m.PublicKey = base64.StdEncoding.EncodeToString(pubKey)

	// Create manifest without signature for signing
	manifestData, err := json.Marshal(m)
	if err != nil {
		return "", nil, fmt.Errorf("failed to marshal manifest: %w", err)
	}

	// Create signing data: sha256(manifest_without_sig || binary_hash)
	signingData := append(manifestData, hash[:]...)
	signingHash := sha256.Sum256(signingData)

	// Sign the hash
	signature := ed25519.Sign(priv, signingHash[:])
	m.Signature = base64.StdEncoding.EncodeToString(signature)

	// Create spore file
	sporeName := fmt.Sprintf("%s-%s.spore", m.Name, m.Version)
	sporePath = filepath.Join(outDir, sporeName)

	// Create zip file
	zipFile, err := os.Create(sporePath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create spore file: %w", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Add manifest.json
	manifestWriter, err := zipWriter.Create("manifest.json")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create manifest in zip: %w", err)
	}

	finalManifest, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return "", nil, fmt.Errorf("failed to marshal final manifest: %w", err)
	}

	if _, err := manifestWriter.Write(finalManifest); err != nil {
		return "", nil, fmt.Errorf("failed to write manifest: %w", err)
	}

	// Add binary
	binaryWriter, err := zipWriter.Create("binary")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create binary in zip: %w", err)
	}

	if _, err := binaryWriter.Write(binaryData); err != nil {
		return "", nil, fmt.Errorf("failed to write binary: %w", err)
	}

	return sporePath, &m, nil
}

// Verify verifies a spore's signature
func Verify(sporePath string) (*Manifest, error) {
	// Open zip file
	zipReader, err := zip.OpenReader(sporePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open spore file: %w", err)
	}
	defer zipReader.Close()

	var manifestData []byte
	var binaryData []byte

	// Extract manifest and binary
	for _, file := range zipReader.File {
		switch file.Name {
		case "manifest.json":
			rc, err := file.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to open manifest: %w", err)
			}
			manifestData, err = io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return nil, fmt.Errorf("failed to read manifest: %w", err)
			}
		case "binary":
			rc, err := file.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to open binary: %w", err)
			}
			binaryData, err = io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return nil, fmt.Errorf("failed to read binary: %w", err)
			}
		}
	}

	if manifestData == nil {
		return nil, fmt.Errorf("manifest.json not found in spore")
	}
	if binaryData == nil {
		return nil, fmt.Errorf("binary not found in spore")
	}

	// Parse manifest
	var manifest Manifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	// Verify binary hash
	hash := sha256.Sum256(binaryData)
	expectedHash := fmt.Sprintf("%x", hash)
	if manifest.BinarySHA256 != expectedHash {
		return nil, fmt.Errorf("binary hash mismatch: expected %s, got %s", expectedHash, manifest.BinarySHA256)
	}

	// Decode public key
	pubKeyData, err := base64.StdEncoding.DecodeString(manifest.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode public key: %w", err)
	}

	// Decode signature
	signature, err := base64.StdEncoding.DecodeString(manifest.Signature)
	if err != nil {
		return nil, fmt.Errorf("failed to decode signature: %w", err)
	}

	// Create manifest without signature for verification
	manifestCopy := manifest
	manifestCopy.Signature = ""
	manifestWithoutSig, err := json.Marshal(manifestCopy)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal manifest without signature: %w", err)
	}

	// Create signing data: sha256(manifest_without_sig || binary_hash)
	signingData := append(manifestWithoutSig, hash[:]...)
	signingHash := sha256.Sum256(signingData)

	// Verify signature
	if !ed25519.Verify(ed25519.PublicKey(pubKeyData), signingHash[:], signature) {
		return nil, fmt.Errorf("signature verification failed")
	}

	return &manifest, nil
}

// Extract extracts a spore to a destination directory
func Extract(sporePath, destDir string) (*Manifest, string, error) {
	// Verify the spore first
	manifest, err := Verify(sporePath)
	if err != nil {
		return nil, "", fmt.Errorf("spore verification failed: %w", err)
	}

	// Open zip file
	zipReader, err := zip.OpenReader(sporePath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to open spore file: %w", err)
	}
	defer zipReader.Close()

	// Create destination directory
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, "", fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Extract files
	for _, file := range zipReader.File {
		rc, err := file.Open()
		if err != nil {
			return nil, "", fmt.Errorf("failed to open file %s: %w", file.Name, err)
		}

		destPath := filepath.Join(destDir, file.Name)
		destFile, err := os.Create(destPath)
		if err != nil {
			rc.Close()
			return nil, "", fmt.Errorf("failed to create file %s: %w", destPath, err)
		}

		_, err = io.Copy(destFile, rc)
		rc.Close()
		destFile.Close()

		if err != nil {
			return nil, "", fmt.Errorf("failed to extract file %s: %w", file.Name, err)
		}

		// Make binary executable
		if file.Name == "binary" {
			if err := os.Chmod(destPath, 0755); err != nil {
				return nil, "", fmt.Errorf("failed to make binary executable: %w", err)
			}
		}
	}

	// Rename binary to match the command name
	oldBinaryPath := filepath.Join(destDir, "binary")
	newBinaryPath := filepath.Join(destDir, manifest.Command)
	if err := os.Rename(oldBinaryPath, newBinaryPath); err != nil {
		return nil, "", fmt.Errorf("failed to rename binary to %s: %w", manifest.Command, err)
	}

	return manifest, newBinaryPath, nil
}
