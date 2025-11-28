package storage

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/securestor/securestor/internal/logger"
	"github.com/securestor/securestor/internal/replicate"
)

// BlobStorage provides custom blob storage with erasure coding support
type BlobStorage struct {
	basePath            string
	erasureCoder        *ErasureCoder
	mu                  sync.RWMutex
	db                  *sql.DB
	logger              *logger.Logger
	replicationCallback func(tenantID, repositoryID, artifactID string, data []byte)
}

// StorageConfig contains configuration for blob storage
type StorageConfig struct {
	BasePath     string // Root directory for storage
	DataShards   int    // Number of data shards (k)
	ParityShards int    // Number of parity shards (m)
}

// BlobMetadata contains metadata about stored blobs
type BlobMetadata struct {
	ArtifactID     string         `json:"artifact_id"`
	OriginalSize   int            `json:"original_size"`
	TotalShards    int            `json:"total_shards"`
	DataShards     int            `json:"data_shards"`
	ParityShards   int            `json:"parity_shards"`
	Checksum       string         `json:"checksum"`
	UploadedAt     time.Time      `json:"uploaded_at"`
	ShardChecksums map[int]string `json:"shard_checksums"`
}

// ShardInfo contains information about individual shards
type ShardInfo struct {
	Index    int    `json:"index"`
	Size     int    `json:"size"`
	Checksum string `json:"checksum"`
	Path     string `json:"path"`
}

// NewBlobStorage creates a new custom blob storage instance
func NewBlobStorage(config StorageConfig) (*BlobStorage, error) {
	// Validate base path
	if config.BasePath == "" {
		return nil, fmt.Errorf("base path is required")
	}

	// Create base directory if it doesn't exist
	if err := os.MkdirAll(config.BasePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	// Initialize erasure coder
	erasureCoder, err := NewErasureCoder(config.DataShards, config.ParityShards)
	if err != nil {
		return nil, fmt.Errorf("failed to create erasure coder: %w", err)
	}

	bs := &BlobStorage{
		basePath:     config.BasePath,
		erasureCoder: erasureCoder,
	}

	// Create required subdirectories
	if err := bs.ensureDirectories(); err != nil {
		return nil, err
	}

	return bs, nil
}

// SetReplicationCallback sets the callback function for replication
func (bs *BlobStorage) SetReplicationCallback(callback func(tenantID, repositoryID, artifactID string, data []byte)) {
	bs.replicationCallback = callback
}

// SetDatabaseAndLogger sets database connection and logger for replication
func (bs *BlobStorage) SetDatabaseAndLogger(db *sql.DB, l *logger.Logger) {
	bs.db = db
	bs.logger = l
}

// ensureDirectories creates required storage subdirectories
func (bs *BlobStorage) ensureDirectories() error {
	dirs := []string{
		filepath.Join(bs.basePath, "artifacts"),
		filepath.Join(bs.basePath, "metadata"),
		filepath.Join(bs.basePath, "temp"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// calculateChecksum calculates SHA-256 checksum of data
func calculateChecksum(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// getArtifactPath returns the base path for an artifact
// Structure: {basePath}/{tenantID}/{repositoryID}/{artifactID}
func (bs *BlobStorage) getArtifactPath(tenantID, repositoryID, artifactID string) string {
	return filepath.Join(bs.basePath, tenantID, repositoryID, artifactID)
}

// getMetadataPath returns the metadata file path for an artifact
func (bs *BlobStorage) getMetadataPath(tenantID, repositoryID, artifactID string) string {
	return filepath.Join(bs.basePath, tenantID, repositoryID, artifactID, "metadata.json")
}

// getShardPath returns the path for a specific shard
func (bs *BlobStorage) getShardPath(tenantID, repositoryID, artifactID string, shardIndex int) string {
	return filepath.Join(bs.getArtifactPath(tenantID, repositoryID, artifactID), fmt.Sprintf("shard-%d.bin", shardIndex))
}

// UploadArtifact uploads an artifact with erasure coding
// Storage path: {basePath}/{tenantID}/{repositoryID}/{artifactID}
func (bs *BlobStorage) UploadArtifact(ctx context.Context, tenantID, repositoryID, artifactID string, reader io.Reader, size int64) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	// Read all data
	data, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read data: %w", err)
	}

	// Calculate checksum of original data
	checksum := calculateChecksum(data)

	// Encode data into shards
	shards, err := bs.erasureCoder.Encode(data)
	if err != nil {
		return fmt.Errorf("failed to encode data: %w", err)
	}

	// Create artifact directory
	artifactPath := bs.getArtifactPath(tenantID, repositoryID, artifactID)
	if err := os.MkdirAll(artifactPath, 0755); err != nil {
		return fmt.Errorf("failed to create artifact directory: %w", err)
	}

	// Store shard checksums
	shardChecksums := make(map[int]string)

	// Upload each shard
	for i, shard := range shards {
		shardPath := bs.getShardPath(tenantID, repositoryID, artifactID, i)

		// Calculate shard checksum
		shardChecksum := calculateChecksum(shard)
		shardChecksums[i] = shardChecksum

		// Write shard to disk
		if err := os.WriteFile(shardPath, shard, 0644); err != nil {
			// Cleanup on failure
			bs.cleanupArtifact(tenantID, repositoryID, artifactID)
			return fmt.Errorf("failed to write shard %d: %w", i, err)
		}
	}

	// Create metadata
	metadata := BlobMetadata{
		ArtifactID:     artifactID,
		OriginalSize:   len(data),
		TotalShards:    len(shards),
		DataShards:     bs.erasureCoder.dataShards,
		ParityShards:   bs.erasureCoder.parityShards,
		Checksum:       checksum,
		UploadedAt:     time.Now(),
		ShardChecksums: shardChecksums,
	}

	// Write metadata file
	metadataPath := bs.getMetadataPath(tenantID, repositoryID, artifactID)
	metadataJSON, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		bs.cleanupArtifact(tenantID, repositoryID, artifactID)
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(metadataPath, metadataJSON, 0644); err != nil {
		bs.cleanupArtifact(tenantID, repositoryID, artifactID)
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	// Trigger replication asynchronously
	if bs.replicationCallback != nil {
		go bs.replicationCallback(tenantID, repositoryID, artifactID, data)
	} else if bs.db != nil && bs.logger != nil {
		// Fallback: trigger replication directly
		go bs.replicateArtifact(tenantID, repositoryID, artifactID, data)
	}

	return nil
}

// replicateArtifact triggers replication for the artifact
func (bs *BlobStorage) replicateArtifact(tenantID, repositoryID, artifactID string, data []byte) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Initialize replication service
	rs := replicate.InitReplicationServiceWithDB(bs.logger, bs.db, tenantID)

	// Replicate file
	bucketName := fmt.Sprintf("%s/%s", repositoryID, artifactID)
	filename := artifactID

	result, err := rs.ReplicateFile(ctx, bucketName, filename, data)
	if err != nil {
		bs.logger.Printf("WARNING: Replication failed for %s: %v", artifactID, err)
	} else {
		bs.logger.Printf("Replication successful: artifact=%s, replicas=%d, checksum=%s",
			artifactID, result.ActualReplicas, result.Checksum)
	}
}

// cleanupArtifact removes all files associated with an artifact (used on upload failure)
func (bs *BlobStorage) cleanupArtifact(tenantID, repositoryID, artifactID string) {
	// Remove artifact directory
	artifactPath := bs.getArtifactPath(tenantID, repositoryID, artifactID)
	os.RemoveAll(artifactPath)
}

// DownloadArtifact downloads and reconstructs an artifact
func (bs *BlobStorage) DownloadArtifact(ctx context.Context, tenantID, repositoryID, artifactID string) ([]byte, error) {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	// Read metadata
	metadataPath := bs.getMetadataPath(tenantID, repositoryID, artifactID)
	metadataBytes, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	var metadata BlobMetadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	// Download all shards
	totalShards := metadata.TotalShards
	shards := make([][]byte, totalShards)
	shardsRead := 0

	for i := 0; i < totalShards; i++ {
		shardPath := bs.getShardPath(tenantID, repositoryID, artifactID, i)

		shardData, err := os.ReadFile(shardPath)
		if err != nil {
			// Shard might be missing - erasure coding can handle this
			shards[i] = nil
			continue
		}

		// Verify shard checksum
		if expectedChecksum, ok := metadata.ShardChecksums[i]; ok {
			actualChecksum := calculateChecksum(shardData)
			if actualChecksum != expectedChecksum {
				return nil, fmt.Errorf("shard %d checksum mismatch: expected %s, got %s",
					i, expectedChecksum, actualChecksum)
			}
		}

		shards[i] = shardData
		shardsRead++
	}

	// Ensure we have enough shards to reconstruct
	minShardsRequired := metadata.DataShards
	if shardsRead < minShardsRequired {
		return nil, fmt.Errorf("insufficient shards: need at least %d, have %d",
			minShardsRequired, shardsRead)
	}

	// Decode shards back to original data
	data, err := bs.erasureCoder.Decode(shards, metadata.OriginalSize)
	if err != nil {
		return nil, fmt.Errorf("failed to decode shards: %w", err)
	}

	// Verify checksum of reconstructed data
	actualChecksum := calculateChecksum(data)
	if actualChecksum != metadata.Checksum {
		return nil, fmt.Errorf("data checksum mismatch: expected %s, got %s",
			metadata.Checksum, actualChecksum)
	}

	return data, nil
}

// DeleteArtifact removes all shards and metadata of an artifact
// DeleteArtifact removes an artifact and all its shards
func (bs *BlobStorage) DeleteArtifact(ctx context.Context, tenantID, repositoryID, artifactID string) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	// Remove artifact directory and all shards
	artifactPath := bs.getArtifactPath(tenantID, repositoryID, artifactID)
	if err := os.RemoveAll(artifactPath); err != nil {
		return fmt.Errorf("failed to remove artifact directory: %w", err)
	}

	return nil
}

// GetMetadata retrieves metadata for an artifact
func (bs *BlobStorage) GetMetadata(ctx context.Context, tenantID, repositoryID, artifactID string) (*BlobMetadata, error) {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	metadataPath := bs.getMetadataPath(tenantID, repositoryID, artifactID)
	metadataBytes, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	var metadata BlobMetadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	return &metadata, nil
}

// ListArtifacts lists all stored artifact IDs
func (bs *BlobStorage) ListArtifacts(ctx context.Context) ([]string, error) {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	metadataDir := filepath.Join(bs.basePath, "metadata")
	entries, err := os.ReadDir(metadataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata directory: %w", err)
	}

	var artifactIDs []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			// Remove .json extension to get artifact ID
			artifactID := entry.Name()[:len(entry.Name())-5]
			artifactIDs = append(artifactIDs, artifactID)
		}
	}

	return artifactIDs, nil
}

// VerifyIntegrity checks the integrity of all shards for an artifact
func (bs *BlobStorage) VerifyIntegrity(ctx context.Context, tenantID, repositoryID, artifactID string) (bool, []int, error) {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	// Read metadata
	metadata, err := bs.GetMetadata(ctx, tenantID, repositoryID, artifactID)
	if err != nil {
		return false, nil, err
	}

	var corruptedShards []int

	// Check each shard
	for i := 0; i < metadata.TotalShards; i++ {
		shardPath := bs.getShardPath(tenantID, repositoryID, artifactID, i)

		shardData, err := os.ReadFile(shardPath)
		if err != nil {
			corruptedShards = append(corruptedShards, i)
			continue
		}

		// Verify checksum
		if expectedChecksum, ok := metadata.ShardChecksums[i]; ok {
			actualChecksum := calculateChecksum(shardData)
			if actualChecksum != expectedChecksum {
				corruptedShards = append(corruptedShards, i)
			}
		}
	}

	isValid := len(corruptedShards) == 0
	return isValid, corruptedShards, nil
}

// GetStorageStats returns storage statistics
// Note: Currently returns empty stats - needs refactor for hierarchical storage
func (bs *BlobStorage) GetStorageStats(ctx context.Context) (*StorageStats, error) {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	stats := &StorageStats{
		ArtifactCount: 0,
		TotalSize:     0,
		ShardCount:    0,
	}

	// TODO: Implement traversal of tenant/repository/artifact hierarchy
	// to calculate statistics

	return stats, nil
}

// StorageStats contains storage statistics
type StorageStats struct {
	ArtifactCount int   `json:"artifact_count"`
	TotalSize     int64 `json:"total_size"`
	ShardCount    int   `json:"shard_count"`
}
