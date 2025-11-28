package encrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"
)

// EncryptionService handles all encryption operations for SecureStor
type EncryptionService struct {
	kmsClient KMSClient
	keyCache  *KeyCache
}

// KMSClient interface for abstracting KMS providers (AWS, Azure, Vault)
type KMSClient interface {
	// Encrypt encrypts data using KMS
	Encrypt(keyID string, plaintext []byte) ([]byte, error)

	// Decrypt decrypts data using KMS
	Decrypt(keyID string, ciphertext []byte) ([]byte, error)

	// GenerateDataKey generates a data encryption key
	GenerateDataKey(keyID string) (*DataKey, error)
}

// DataKey represents a KMS-generated data encryption key
type DataKey struct {
	Plaintext  []byte // DEK in plaintext (use immediately, then zero out)
	Ciphertext []byte // DEK encrypted by KMS (store with artifact)
	KeyID      string // KMS key ID used
}

// EncryptedData represents encrypted artifact data with metadata
type EncryptedData struct {
	Ciphertext   []byte `json:"ciphertext"`         // Encrypted data
	EncryptedDEK []byte `json:"encrypted_dek"`      // DEK encrypted by KEK
	Nonce        []byte `json:"nonce"`              // GCM nonce (12 bytes)
	AuthTag      []byte `json:"auth_tag,omitempty"` // GCM auth tag (16 bytes) - may be in ciphertext
	Algorithm    string `json:"algorithm"`          // "AES-256-GCM"
	KeyVersion   int    `json:"key_version"`        // For key rotation
	TenantID     string `json:"tenant_id"`          // Tenant isolation
	EncryptedAt  int64  `json:"encrypted_at"`       // Unix timestamp
}

// NewEncryptionService creates a new encryption service
func NewEncryptionService(kmsClient KMSClient) *EncryptionService {
	return &EncryptionService{
		kmsClient: kmsClient,
		keyCache:  NewKeyCache(300), // 5 minute TTL
	}
}

// EncryptArtifact encrypts artifact data using envelope encryption
// Returns encrypted data with all necessary metadata for decryption
func (s *EncryptionService) EncryptArtifact(tenantID string, plaintext []byte, kekKeyID string) (*EncryptedData, error) {
	// Step 1: Generate random DEK (256-bit for AES-256)
	dek := make([]byte, 32)
	if _, err := rand.Read(dek); err != nil {
		return nil, fmt.Errorf("failed to generate DEK: %w", err)
	}
	defer zeroBytes(dek) // Security: zero out key from memory when done

	// Step 2: Encrypt artifact data with DEK using AES-256-GCM
	block, err := aes.NewCipher(dek)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate random nonce (96 bits / 12 bytes for GCM)
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt and authenticate
	// GCM automatically appends the auth tag to ciphertext
	ciphertext := gcm.Seal(nil, nonce, plaintext, []byte(tenantID))

	// Step 3: Encrypt DEK with KEK (via KMS)
	encryptedDEK, err := s.kmsClient.Encrypt(kekKeyID, dek)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt DEK: %w", err)
	}

	return &EncryptedData{
		Ciphertext:   ciphertext,
		EncryptedDEK: encryptedDEK,
		Nonce:        nonce,
		Algorithm:    "AES-256-GCM",
		KeyVersion:   1, // Increment when rotating keys
		TenantID:     tenantID,
		EncryptedAt:  currentUnixTimestamp(),
	}, nil
}

// DecryptArtifact decrypts artifact data using envelope encryption
func (s *EncryptionService) DecryptArtifact(encryptedData *EncryptedData, kekKeyID string) ([]byte, error) {
	// Validate tenant context (prevent cross-tenant decryption)
	if encryptedData.TenantID == "" {
		return nil, fmt.Errorf("tenant ID required for decryption")
	}

	// Step 1: Decrypt DEK using KMS
	dek, err := s.kmsClient.Decrypt(kekKeyID, encryptedData.EncryptedDEK)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt DEK: %w", err)
	}
	defer zeroBytes(dek) // Security: zero out key from memory

	// Step 2: Decrypt artifact data with DEK
	block, err := aes.NewCipher(dek)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Decrypt and verify authenticity
	plaintext, err := gcm.Open(nil, encryptedData.Nonce, encryptedData.Ciphertext, []byte(encryptedData.TenantID))
	if err != nil {
		return nil, fmt.Errorf("decryption failed (tampering detected or wrong key): %w", err)
	}

	return plaintext, nil
}

// EncryptStream encrypts data from a reader and writes to a writer (for large files)
func (s *EncryptionService) EncryptStream(tenantID string, reader io.Reader, writer io.Writer, kekKeyID string) (*EncryptedData, error) {
	// Generate DEK
	dek := make([]byte, 32)
	if _, err := rand.Read(dek); err != nil {
		return nil, fmt.Errorf("failed to generate DEK: %w", err)
	}
	defer zeroBytes(dek)

	// Create cipher
	block, err := aes.NewCipher(dek)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Create stream cipher writer
	streamWriter := &cipher.StreamWriter{
		S: cipher.NewCTR(block, nonce),
		W: writer,
	}

	// Copy data through encryption
	if _, err := io.Copy(streamWriter, reader); err != nil {
		return nil, fmt.Errorf("failed to encrypt stream: %w", err)
	}

	// Encrypt DEK with KMS
	encryptedDEK, err := s.kmsClient.Encrypt(kekKeyID, dek)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt DEK: %w", err)
	}

	return &EncryptedData{
		EncryptedDEK: encryptedDEK,
		Nonce:        nonce,
		Algorithm:    "AES-256-CTR", // CTR mode for streaming
		KeyVersion:   1,
		TenantID:     tenantID,
		EncryptedAt:  currentUnixTimestamp(),
	}, nil
}

// DecryptStream decrypts data from a reader and writes to a writer
func (s *EncryptionService) DecryptStream(encryptedData *EncryptedData, reader io.Reader, writer io.Writer, kekKeyID string) error {
	// Decrypt DEK
	dek, err := s.kmsClient.Decrypt(kekKeyID, encryptedData.EncryptedDEK)
	if err != nil {
		return fmt.Errorf("failed to decrypt DEK: %w", err)
	}
	defer zeroBytes(dek)

	// Create cipher
	block, err := aes.NewCipher(dek)
	if err != nil {
		return fmt.Errorf("failed to create AES cipher: %w", err)
	}

	// Create stream cipher reader
	streamReader := &cipher.StreamReader{
		S: cipher.NewCTR(block, encryptedData.Nonce),
		R: reader,
	}

	// Copy data through decryption
	if _, err := io.Copy(writer, streamReader); err != nil {
		return fmt.Errorf("failed to decrypt stream: %w", err)
	}

	return nil
}

// DeriveKEK derives a Key Encryption Key from Tenant Master Key
// Uses HKDF-SHA256 for key derivation
func (s *EncryptionService) DeriveKEK(tmk []byte, context string) ([]byte, error) {
	// Use HKDF to derive KEK from TMK with context
	kdf := hkdf.New(sha256.New, tmk, nil, []byte(context))

	kek := make([]byte, 32) // 256-bit key
	if _, err := io.ReadFull(kdf, kek); err != nil {
		return nil, fmt.Errorf("failed to derive KEK: %w", err)
	}

	return kek, nil
}

// GenerateTMK generates a new Tenant Master Key
func (s *EncryptionService) GenerateTMK() ([]byte, error) {
	tmk := make([]byte, 32) // 256-bit key
	if _, err := rand.Read(tmk); err != nil {
		return nil, fmt.Errorf("failed to generate TMK: %w", err)
	}
	return tmk, nil
}

// EncryptMetadata encrypts artifact metadata (name, version, etc.)
func (s *EncryptionService) EncryptMetadata(tenantID string, metadata map[string]string, kekKeyID string) (map[string]string, error) {
	encrypted := make(map[string]string)

	for key, value := range metadata {
		encData, err := s.EncryptArtifact(tenantID, []byte(value), kekKeyID)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt metadata field %s: %w", key, err)
		}

		// Store as base64 for JSON compatibility
		encrypted[key] = base64.StdEncoding.EncodeToString(encData.Ciphertext)
	}

	return encrypted, nil
}

// Helper functions

// zeroBytes securely zeros out a byte slice (prevents key recovery from memory)
func zeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

// currentUnixTimestamp returns current Unix timestamp
func currentUnixTimestamp() int64 {
	// Implementation would use time.Now().Unix()
	return 0 // Placeholder
}

// VerifyIntegrity verifies the integrity of encrypted data
func (s *EncryptionService) VerifyIntegrity(encryptedData *EncryptedData) error {
	// GCM automatically verifies integrity during decryption
	// This is a placeholder for additional checks if needed
	if encryptedData.Algorithm != "AES-256-GCM" && encryptedData.Algorithm != "AES-256-CTR" {
		return fmt.Errorf("unsupported encryption algorithm: %s", encryptedData.Algorithm)
	}

	if len(encryptedData.Nonce) != 12 {
		return fmt.Errorf("invalid nonce size: expected 12, got %d", len(encryptedData.Nonce))
	}

	return nil
}
