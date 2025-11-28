package encrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"os"

	"github.com/securestor/securestor/internal/config"
)

var EncryptionKey string

func init() {
	// Use the centralized environment loader
	config.LoadEnvOnce()
	EncryptionKey = config.GetEnvWithFallback("ENCRYPTION_KEY", "")
}

func EncryptFile(inputFilePath, outputFilePath string, key []byte) error {
	inputFile, err := os.Open(inputFilePath)
	if err != nil {
		return err
	}
	defer inputFile.Close()

	outputFile, err := os.Create(outputFilePath)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	// Generate a new AES cipher using the key
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	// Generate a new GCM (Galois/Counter Mode) cipher
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	// Create a nonce of the appropriate size
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return err
	}

	// Write the nonce to the output file
	if _, err := outputFile.Write(nonce); err != nil {
		return err
	}

	// Create a buffer to hold chunks of the file
	buffer := make([]byte, 1024*1024) // 1 MB buffer

	for {
		// Read a chunk from the input file
		n, err := inputFile.Read(buffer)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			break
		}

		// Encrypt the chunk
		encryptedChunk := gcm.Seal(nil, nonce, buffer[:n], nil)

		// Write the encrypted chunk to the output file
		if _, err := outputFile.Write(encryptedChunk); err != nil {
			return err
		}
	}

	return nil
}

func Encrypt(data []byte, passphrase string) ([]byte, error) {
	block, _ := aes.NewCipher([]byte(passphrase))
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

func Decrypt(data []byte, passphrase string) ([]byte, error) {
	block, err := aes.NewCipher([]byte(passphrase))
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}
