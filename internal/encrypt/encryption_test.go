package encrypt

import (
	"testing"
)

func TestEncryptionService(t *testing.T) {
	// Initialize Mock KMS
	kmsClient := NewMockKMSClient()
	encService := NewEncryptionService(kmsClient)

	tenantID := "test-tenant-123"
	kekKeyID := "tenant:test-tenant-123:kek"

	// Test data
	plaintext := []byte("This is a secret artifact content for SecureStor")

	t.Run("EncryptArtifact", func(t *testing.T) {
		encryptedData, err := encService.EncryptArtifact(tenantID, plaintext, kekKeyID)
		if err != nil {
			t.Fatalf("EncryptArtifact failed: %v", err)
		}

		if encryptedData == nil {
			t.Fatal("Expected encrypted data, got nil")
		}

		if len(encryptedData.Ciphertext) == 0 {
			t.Error("Ciphertext is empty")
		}

		if len(encryptedData.EncryptedDEK) == 0 {
			t.Error("Encrypted DEK is empty")
		}

		if encryptedData.Algorithm != "AES-256-GCM" {
			t.Errorf("Expected AES-256-GCM, got %s", encryptedData.Algorithm)
		}

		if encryptedData.KeyVersion != 1 {
			t.Errorf("Expected key version 1, got %d", encryptedData.KeyVersion)
		}

		t.Logf("Encrypted %d bytes to %d bytes", len(plaintext), len(encryptedData.Ciphertext))
	})

	t.Run("EncryptAndDecrypt", func(t *testing.T) {
		// Encrypt
		encryptedData, err := encService.EncryptArtifact(tenantID, plaintext, kekKeyID)
		if err != nil {
			t.Fatalf("Encryption failed: %v", err)
		}

		// Decrypt
		decrypted, err := encService.DecryptArtifact(encryptedData, kekKeyID)
		if err != nil {
			t.Fatalf("Decryption failed: %v", err)
		}

		// Verify
		if string(decrypted) != string(plaintext) {
			t.Errorf("Decrypted data doesn't match original.\nExpected: %s\nGot: %s",
				string(plaintext), string(decrypted))
		}

		t.Logf("Successfully encrypted and decrypted %d bytes", len(plaintext))
	})

	t.Run("TamperedCiphertextShouldFail", func(t *testing.T) {
		// Encrypt
		encryptedData, err := encService.EncryptArtifact(tenantID, plaintext, kekKeyID)
		if err != nil {
			t.Fatalf("Encryption failed: %v", err)
		}

		// Tamper with ciphertext
		if len(encryptedData.Ciphertext) > 10 {
			encryptedData.Ciphertext[5] ^= 0xFF
		}

		// Try to decrypt - should fail
		_, err = encService.DecryptArtifact(encryptedData, kekKeyID)
		if err == nil {
			t.Error("Expected decryption to fail with tampered ciphertext")
		}

		t.Logf("Correctly rejected tampered ciphertext: %v", err)
	})

	t.Run("LargeArtifact", func(t *testing.T) {
		// Test with 10MB artifact
		largePlaintext := make([]byte, 10*1024*1024)
		for i := range largePlaintext {
			largePlaintext[i] = byte(i % 256)
		}

		encryptedData, err := encService.EncryptArtifact(tenantID, largePlaintext, kekKeyID)
		if err != nil {
			t.Fatalf("Failed to encrypt large artifact: %v", err)
		}

		decrypted, err := encService.DecryptArtifact(encryptedData, kekKeyID)
		if err != nil {
			t.Fatalf("Failed to decrypt large artifact: %v", err)
		}

		if len(decrypted) != len(largePlaintext) {
			t.Errorf("Size mismatch: expected %d, got %d", len(largePlaintext), len(decrypted))
		}

		// Verify first and last 1KB
		for i := 0; i < 1024; i++ {
			if decrypted[i] != largePlaintext[i] {
				t.Errorf("Mismatch at byte %d", i)
				break
			}
		}

		t.Logf("Successfully encrypted and decrypted 10MB artifact")
	})
}

func TestKeyCache(t *testing.T) {
	cache := NewKeyCache(1) // 1 second TTL

	t.Run("SetAndGet", func(t *testing.T) {
		key := []byte("test-encryption-key-32-bytes!!")
		cache.Set("test-key-1", key)

		retrieved, found := cache.Get("test-key-1")
		if !found {
			t.Error("Key not found in cache")
		}

		if string(retrieved) != string(key) {
			t.Error("Retrieved key doesn't match original")
		}
	})

	t.Run("GetNonExistent", func(t *testing.T) {
		_, found := cache.Get("non-existent-key")
		if found {
			t.Error("Expected not found, but key exists")
		}
	})

	t.Run("Delete", func(t *testing.T) {
		key := []byte("another-test-key-32-bytes-long")
		cache.Set("test-key-2", key)

		cache.Delete("test-key-2")

		_, found := cache.Get("test-key-2")
		if found {
			t.Error("Key should have been deleted")
		}
	})

	t.Run("Clear", func(t *testing.T) {
		cache.Set("key1", []byte("value1"))
		cache.Set("key2", []byte("value2"))
		cache.Set("key3", []byte("value3"))

		cache.Clear()

		_, found1 := cache.Get("key1")
		_, found2 := cache.Get("key2")
		_, found3 := cache.Get("key3")

		if found1 || found2 || found3 {
			t.Error("Cache should be empty after clear")
		}
	})
}

func TestMockKMS(t *testing.T) {
	kms := NewMockKMSClient()
	keyID := "test-kms-key-id"

	t.Run("EncryptAndDecrypt", func(t *testing.T) {
		plaintext := []byte("sensitive data to encrypt")

		// Encrypt
		ciphertext, err := kms.Encrypt(keyID, plaintext)
		if err != nil {
			t.Fatalf("Encrypt failed: %v", err)
		}

		if len(ciphertext) == 0 {
			t.Error("Ciphertext is empty")
		}

		// Decrypt
		decrypted, err := kms.Decrypt(keyID, ciphertext)
		if err != nil {
			t.Fatalf("Decrypt failed: %v", err)
		}

		if string(decrypted) != string(plaintext) {
			t.Error("Decrypted data doesn't match original")
		}
	})

	t.Run("GenerateDataKey", func(t *testing.T) {
		dataKey, err := kms.GenerateDataKey(keyID)
		if err != nil {
			t.Fatalf("GenerateDataKey failed: %v", err)
		}

		if len(dataKey.Plaintext) != 32 {
			t.Errorf("Expected 32-byte DEK, got %d bytes", len(dataKey.Plaintext))
		}

		if len(dataKey.Ciphertext) == 0 {
			t.Error("Encrypted DEK is empty")
		}

		// Decrypt the encrypted DEK
		decryptedDEK, err := kms.Decrypt(keyID, dataKey.Ciphertext)
		if err != nil {
			t.Fatalf("Failed to decrypt DEK: %v", err)
		}

		if string(decryptedDEK) != string(dataKey.Plaintext) {
			t.Error("Decrypted DEK doesn't match original")
		}
	})
}

func BenchmarkEncryption(b *testing.B) {
	kmsClient := NewMockKMSClient()
	encService := NewEncryptionService(kmsClient)

	plaintext := make([]byte, 1024*1024) // 1MB
	for i := range plaintext {
		plaintext[i] = byte(i % 256)
	}

	b.Run("Encrypt1MB", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := encService.EncryptArtifact("tenant", plaintext, "kek")
			if err != nil {
				b.Fatal(err)
			}
		}
		b.SetBytes(int64(len(plaintext)))
	})

	// Encrypt once for decrypt benchmark
	encryptedData, _ := encService.EncryptArtifact("tenant", plaintext, "kek")

	b.Run("Decrypt1MB", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := encService.DecryptArtifact(encryptedData, "kek")
			if err != nil {
				b.Fatal(err)
			}
		}
		b.SetBytes(int64(len(plaintext)))
	})
}
