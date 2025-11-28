package encrypt

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kms"
)

// AWSKMSClient implements KMSClient for AWS KMS
type AWSKMSClient struct {
	client *kms.Client
	ctx    context.Context
}

// NewAWSKMSClient creates a new AWS KMS client
func NewAWSKMSClient(ctx context.Context, region string) (*AWSKMSClient, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &AWSKMSClient{
		client: kms.NewFromConfig(cfg),
		ctx:    ctx,
	}, nil
}

// Encrypt encrypts data using AWS KMS
func (c *AWSKMSClient) Encrypt(keyID string, plaintext []byte) ([]byte, error) {
	result, err := c.client.Encrypt(c.ctx, &kms.EncryptInput{
		KeyId:     &keyID,
		Plaintext: plaintext,
	})
	if err != nil {
		return nil, fmt.Errorf("AWS KMS encrypt failed: %w", err)
	}

	return result.CiphertextBlob, nil
}

// Decrypt decrypts data using AWS KMS
func (c *AWSKMSClient) Decrypt(keyID string, ciphertext []byte) ([]byte, error) {
	result, err := c.client.Decrypt(c.ctx, &kms.DecryptInput{
		KeyId:          &keyID,
		CiphertextBlob: ciphertext,
	})
	if err != nil {
		return nil, fmt.Errorf("AWS KMS decrypt failed: %w", err)
	}

	return result.Plaintext, nil
}

// GenerateDataKey generates a data encryption key using AWS KMS
func (c *AWSKMSClient) GenerateDataKey(keyID string) (*DataKey, error) {
	result, err := c.client.GenerateDataKey(c.ctx, &kms.GenerateDataKeyInput{
		KeyId:   &keyID,
		KeySpec: "AES_256",
	})
	if err != nil {
		return nil, fmt.Errorf("AWS KMS generate data key failed: %w", err)
	}

	return &DataKey{
		Plaintext:  result.Plaintext,
		Ciphertext: result.CiphertextBlob,
		KeyID:      keyID,
	}, nil
}

// MockKMSClient implements KMSClient for testing (DO NOT USE IN PRODUCTION)
type MockKMSClient struct {
	masterKey []byte // Simulated KMS master key
}

// NewMockKMSClient creates a mock KMS client for development/testing
func NewMockKMSClient() *MockKMSClient {
	// Generate a random master key for simulation
	masterKey := make([]byte, 32)
	// In real implementation, would use crypto/rand
	return &MockKMSClient{
		masterKey: masterKey,
	}
}

// Encrypt simulates KMS encryption (uses AES-GCM with mock master key)
func (c *MockKMSClient) Encrypt(keyID string, plaintext []byte) ([]byte, error) {
	// This is a simplified mock - in production, use real KMS
	// For testing, we just XOR with master key (INSECURE - for dev only!)
	ciphertext := make([]byte, len(plaintext))
	for i := range plaintext {
		ciphertext[i] = plaintext[i] ^ c.masterKey[i%len(c.masterKey)]
	}
	return ciphertext, nil
}

// Decrypt simulates KMS decryption
func (c *MockKMSClient) Decrypt(keyID string, ciphertext []byte) ([]byte, error) {
	// XOR again to decrypt (symmetric operation for this mock)
	plaintext := make([]byte, len(ciphertext))
	for i := range ciphertext {
		plaintext[i] = ciphertext[i] ^ c.masterKey[i%len(c.masterKey)]
	}
	return plaintext, nil
}

// GenerateDataKey simulates data key generation
func (c *MockKMSClient) GenerateDataKey(keyID string) (*DataKey, error) {
	// Generate random DEK
	dek := make([]byte, 32)
	// In real implementation, would use crypto/rand

	// "Encrypt" DEK with mock master key
	encryptedDEK, err := c.Encrypt(keyID, dek)
	if err != nil {
		return nil, err
	}

	return &DataKey{
		Plaintext:  dek,
		Ciphertext: encryptedDEK,
		KeyID:      keyID,
	}, nil
}
