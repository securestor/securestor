package storage

import (
	"bytes"
	"fmt"

	"github.com/klauspost/reedsolomon"
)

type ErasureCoder struct {
	dataShards   int
	parityShards int
	encoder      reedsolomon.Encoder
}

// NewErasureCoder creates a new erasure coder
// dataShards: number of data shards (e.g., 4)
// parityShards: number of parity shards for redundancy (e.g., 2)
// This allows losing up to parityShards before data is unrecoverable
func NewErasureCoder(dataShards, parityShards int) (*ErasureCoder, error) {
	encoder, err := reedsolomon.New(dataShards, parityShards)
	if err != nil {
		return nil, fmt.Errorf("failed to create erasure encoder: %w", err)
	}

	return &ErasureCoder{
		dataShards:   dataShards,
		parityShards: parityShards,
		encoder:      encoder,
	}, nil
}

// Encode splits data into shards using erasure coding
func (ec *ErasureCoder) Encode(data []byte) ([][]byte, error) {
	// Calculate shard size
	shardSize := (len(data) + ec.dataShards - 1) / ec.dataShards

	// Create shards
	shards := make([][]byte, ec.dataShards+ec.parityShards)

	// Fill data shards
	for i := 0; i < ec.dataShards; i++ {
		start := i * shardSize
		end := start + shardSize

		if start >= len(data) {
			shards[i] = make([]byte, shardSize)
		} else if end > len(data) {
			shards[i] = make([]byte, shardSize)
			copy(shards[i], data[start:])
		} else {
			shards[i] = data[start:end]
		}
	}

	// Create parity shards
	for i := ec.dataShards; i < ec.dataShards+ec.parityShards; i++ {
		shards[i] = make([]byte, shardSize)
	}

	// Encode parity shards
	if err := ec.encoder.Encode(shards); err != nil {
		return nil, fmt.Errorf("failed to encode shards: %w", err)
	}

	return shards, nil
}

// Decode reconstructs data from shards (even with missing shards)
// Missing shards should be nil in the shards slice
func (ec *ErasureCoder) Decode(shards [][]byte, originalSize int) ([]byte, error) {
	// Validate shard count
	if len(shards) != ec.dataShards+ec.parityShards {
		return nil, fmt.Errorf("expected %d shards, got %d", ec.dataShards+ec.parityShards, len(shards))
	}

	// Reconstruct if necessary
	if err := ec.encoder.Reconstruct(shards); err != nil {
		return nil, fmt.Errorf("failed to reconstruct data: %w", err)
	}

	// Verify reconstruction
	ok, err := ec.encoder.Verify(shards)
	if err != nil {
		return nil, fmt.Errorf("failed to verify shards: %w", err)
	}
	if !ok {
		return nil, fmt.Errorf("shard verification failed after reconstruction")
	}

	// Join data shards
	var buf bytes.Buffer
	for i := 0; i < ec.dataShards; i++ {
		if shards[i] != nil {
			buf.Write(shards[i])
		}
	}

	// Trim to original size
	data := buf.Bytes()
	if len(data) > originalSize {
		data = data[:originalSize]
	}

	return data, nil
}
