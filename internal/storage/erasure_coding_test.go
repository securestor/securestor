package storage

import (
	"bytes"
	"testing"
)

func TestErasureCoding_BasicEncodeDecode(t *testing.T) {
	// Create erasure coder with k=4 data shards and m=2 parity shards
	k, m := 4, 2
	ec, err := NewErasureCoder(k, m)
	if err != nil {
		t.Fatalf("Failed to create erasure coder: %v", err)
	}

	// Original data
	originalData := []byte("Hello, this is a test of Reed-Solomon erasure coding!")
	originalSize := len(originalData)

	// Encode: Split into k data shards + m parity shards
	shards, err := ec.Encode(originalData)
	if err != nil {
		t.Fatalf("Failed to encode: %v", err)
	}

	// Verify we have k+m shards
	if len(shards) != k+m {
		t.Errorf("Expected %d shards, got %d", k+m, len(shards))
	}

	t.Logf("✓ Successfully split data into %d data shards and %d parity shards", k, m)

	// Test 1: Decode with all shards intact
	t.Run("Decode with all shards", func(t *testing.T) {
		decoded, err := ec.Decode(shards, originalSize)
		if err != nil {
			t.Fatalf("Failed to decode: %v", err)
		}

		if !bytes.Equal(decoded, originalData) {
			t.Errorf("Decoded data doesn't match original")
		}
		t.Logf("✓ Successfully reconstructed data with all %d shards", k+m)
	})

	// Test 2: Decode with exactly k shards (minimum required)
	t.Run("Decode with minimum k shards", func(t *testing.T) {
		// Make a copy and simulate losing m parity shards
		shardsWithLoss := make([][]byte, k+m)
		copy(shardsWithLoss, shards)

		// Remove parity shards (simulate disk/node failure)
		for i := k; i < k+m; i++ {
			shardsWithLoss[i] = nil
		}

		decoded, err := ec.Decode(shardsWithLoss, originalSize)
		if err != nil {
			t.Fatalf("Failed to decode with k shards: %v", err)
		}

		if !bytes.Equal(decoded, originalData) {
			t.Errorf("Decoded data doesn't match original")
		}
		t.Logf("✓ Successfully reconstructed data with only %d data shards (lost %d parity shards)", k, m)
	})

	// Test 3: Decode with mixed loss (some data + some parity shards missing)
	t.Run("Decode with mixed shard loss", func(t *testing.T) {
		// Make a copy and simulate losing up to m shards
		shardsWithLoss := make([][]byte, k+m)
		copy(shardsWithLoss, shards)

		// Lose shard 0 (data) and shard 4 (parity) - total of 2 shards lost
		shardsWithLoss[0] = nil
		shardsWithLoss[k] = nil

		decoded, err := ec.Decode(shardsWithLoss, originalSize)
		if err != nil {
			t.Fatalf("Failed to decode with mixed loss: %v", err)
		}

		if !bytes.Equal(decoded, originalData) {
			t.Errorf("Decoded data doesn't match original")
		}
		t.Logf("✓ Successfully reconstructed data after losing 1 data shard and 1 parity shard")
	})

	// Test 4: Verify we can lose up to m shards and still recover
	t.Run("Decode with maximum loss tolerance", func(t *testing.T) {
		// Make a copy and simulate losing exactly m shards
		shardsWithLoss := make([][]byte, k+m)
		copy(shardsWithLoss, shards)

		// Lose first m shards (could be data or parity)
		for i := 0; i < m; i++ {
			shardsWithLoss[i] = nil
		}

		decoded, err := ec.Decode(shardsWithLoss, originalSize)
		if err != nil {
			t.Fatalf("Failed to decode with max loss: %v", err)
		}

		if !bytes.Equal(decoded, originalData) {
			t.Errorf("Decoded data doesn't match original")
		}
		t.Logf("✓ Successfully reconstructed data after losing maximum %d shards", m)
	})

	// Test 5: Verify failure when losing more than m shards
	t.Run("Fail when losing too many shards", func(t *testing.T) {
		// Make a copy and simulate losing m+1 shards (more than tolerance)
		shardsWithLoss := make([][]byte, k+m)
		copy(shardsWithLoss, shards)

		// Lose more than m shards
		for i := 0; i < m+1; i++ {
			shardsWithLoss[i] = nil
		}

		_, err := ec.Decode(shardsWithLoss, originalSize)
		if err == nil {
			t.Errorf("Expected error when losing more than %d shards, but decode succeeded", m)
		} else {
			t.Logf("✓ Correctly failed when losing %d shards (more than tolerance of %d)", m+1, m)
		}
	})
}

func TestErasureCoding_LargeData(t *testing.T) {
	// Test with larger data (1MB)
	k, m := 10, 4
	ec, err := NewErasureCoder(k, m)
	if err != nil {
		t.Fatalf("Failed to create erasure coder: %v", err)
	}

	// Create 1MB of data
	originalData := make([]byte, 1024*1024)
	for i := range originalData {
		originalData[i] = byte(i % 256)
	}
	originalSize := len(originalData)

	// Encode
	shards, err := ec.Encode(originalData)
	if err != nil {
		t.Fatalf("Failed to encode: %v", err)
	}

	// Simulate losing 4 shards (at the tolerance limit)
	shardsWithLoss := make([][]byte, k+m)
	copy(shardsWithLoss, shards)
	shardsWithLoss[0] = nil
	shardsWithLoss[3] = nil
	shardsWithLoss[7] = nil
	shardsWithLoss[k] = nil // First parity shard

	// Decode
	decoded, err := ec.Decode(shardsWithLoss, originalSize)
	if err != nil {
		t.Fatalf("Failed to decode large data: %v", err)
	}

	if !bytes.Equal(decoded, originalData) {
		t.Errorf("Decoded large data doesn't match original")
	}

	t.Logf("✓ Successfully encoded/decoded 1MB with k=%d, m=%d, losing %d shards", k, m, 4)
}

func TestErasureCoding_DifferentConfigurations(t *testing.T) {
	testCases := []struct {
		name string
		k    int
		m    int
	}{
		{"Small k=2, m=1", 2, 1},
		{"Balanced k=4, m=2", 4, 2},
		{"Standard k=6, m=3", 6, 3},
		{"High redundancy k=4, m=4", 4, 4},
		{"Large k=10, m=4", 10, 4},
	}

	data := []byte("Test data for different configurations")

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ec, err := NewErasureCoder(tc.k, tc.m)
			if err != nil {
				t.Fatalf("Failed to create coder: %v", err)
			}

			shards, err := ec.Encode(data)
			if err != nil {
				t.Fatalf("Failed to encode: %v", err)
			}

			// Lose m shards
			shardsWithLoss := make([][]byte, tc.k+tc.m)
			copy(shardsWithLoss, shards)
			for i := 0; i < tc.m; i++ {
				shardsWithLoss[i] = nil
			}

			decoded, err := ec.Decode(shardsWithLoss, len(data))
			if err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}

			if !bytes.Equal(decoded, data) {
				t.Errorf("Mismatch in config k=%d, m=%d", tc.k, tc.m)
			}

			t.Logf("✓ Config k=%d, m=%d works correctly", tc.k, tc.m)
		})
	}
}
