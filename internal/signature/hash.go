package signature

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
)

// HashComputer computes cryptographic hashes for artifacts
type HashComputer struct {
	sha256Hash hash.Hash
	sha512Hash hash.Hash
	writers    []io.Writer
}

// NewHashComputer creates a new hash computer
func NewHashComputer() *HashComputer {
	sha256Hash := sha256.New()
	sha512Hash := sha512.New()

	return &HashComputer{
		sha256Hash: sha256Hash,
		sha512Hash: sha512Hash,
		writers:    []io.Writer{sha256Hash, sha512Hash},
	}
}

// Writer returns an io.Writer that computes hashes
func (hc *HashComputer) Writer() io.Writer {
	return io.MultiWriter(hc.writers...)
}

// Compute computes hashes from a reader
func (hc *HashComputer) Compute(r io.Reader) (*IntegrityHash, error) {
	if _, err := io.Copy(hc.Writer(), r); err != nil {
		return nil, fmt.Errorf("failed to compute hashes: %w", err)
	}

	return hc.GetHashes(), nil
}

// GetHashes returns the computed hashes
func (hc *HashComputer) GetHashes() *IntegrityHash {
	return &IntegrityHash{
		SHA256:    hex.EncodeToString(hc.sha256Hash.Sum(nil)),
		SHA512:    hex.EncodeToString(hc.sha512Hash.Sum(nil)),
		Algorithm: "sha256", // SHA256 is canonical
	}
}

// ComputeHashes computes SHA-256 and SHA-512 hashes from a reader
func ComputeHashes(r io.Reader) (*IntegrityHash, error) {
	hc := NewHashComputer()
	return hc.Compute(r)
}

// VerifyHash verifies that data matches the expected hash
func VerifyHash(data []byte, expectedSHA256 string) (bool, error) {
	h := sha256.Sum256(data)
	actual := hex.EncodeToString(h[:])
	return actual == expectedSHA256, nil
}

// VerifyHashReader verifies that a reader's content matches the expected hash
func VerifyHashReader(r io.Reader, expectedSHA256 string) (bool, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return false, fmt.Errorf("failed to read data: %w", err)
	}

	actual := hex.EncodeToString(h.Sum(nil))
	return actual == expectedSHA256, nil
}

// FormatHashWithPrefix formats a hash with its algorithm prefix
func FormatHashWithPrefix(algorithm, hash string) string {
	return fmt.Sprintf("%s:%s", algorithm, hash)
}

// ParseHashWithPrefix parses a hash with algorithm prefix (e.g., "sha256:abc123")
func ParseHashWithPrefix(hashStr string) (algorithm, hash string, err error) {
	var algo, h string
	n, err := fmt.Sscanf(hashStr, "%s:%s", &algo, &h)
	if err != nil || n != 2 {
		return "", "", fmt.Errorf("invalid hash format: expected 'algorithm:hash'")
	}
	return algo, h, nil
}
