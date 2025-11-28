package replicate

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/securestor/securestor/internal/logger"
)

// ArtifactReplicator defines the interface for artifact replication across all repository types
// This provides a unified abstraction for replication operations
type ArtifactReplicator interface {
	// ReplicateArtifact replicates an artifact to configured storage nodes
	ReplicateArtifact(ctx context.Context, req *ReplicationRequest) (*ReplicationResult, error)

	// GetReplicationStatus checks the replication status of an artifact
	GetReplicationStatus(ctx context.Context, artifactID string) (*ReplicationStatus, error)
}

// ReplicationRequest encapsulates all information needed to replicate an artifact
type ReplicationRequest struct {
	TenantID     string            // Tenant identifier
	RepositoryID string            // Repository identifier
	ArtifactID   string            // Artifact identifier (name-version-timestamp)
	ArtifactType string            // Type: maven, npm, pypi, docker, helm
	Data         []byte            // Artifact data to replicate
	Metadata     map[string]string // Additional metadata
	BucketName   string            // Logical bucket/namespace for storage
	FileName     string            // Filename to use in storage
}

// ReplicationStatus represents the current replication state of an artifact
type ReplicationStatus struct {
	ArtifactID      string            // Artifact identifier
	TotalNodes      int               // Total configured nodes
	HealthyReplicas int               // Number of healthy replicas
	Nodes           map[string]string // Node name -> status
	LastChecked     string            // Last status check timestamp
}

// UnifiedReplicator provides a centralized replication service for all artifact types
// It implements the Strategy pattern to handle different repository types uniformly
type UnifiedReplicator struct {
	db           *sql.DB
	logger       *logger.Logger
	mu           sync.RWMutex
	tenantCaches map[string]*ReplicationService // Cached replication services per tenant
}

// NewUnifiedReplicator creates a new unified replication service
func NewUnifiedReplicator(db *sql.DB, logger *logger.Logger) *UnifiedReplicator {
	return &UnifiedReplicator{
		db:           db,
		logger:       logger,
		tenantCaches: make(map[string]*ReplicationService),
	}
}

// ReplicateArtifact performs artifact replication with tenant-aware caching
func (ur *UnifiedReplicator) ReplicateArtifact(ctx context.Context, req *ReplicationRequest) (*ReplicationResult, error) {
	if req == nil {
		return nil, fmt.Errorf("replication request cannot be nil")
	}

	if req.TenantID == "" || req.Data == nil || len(req.Data) == 0 {
		return &ReplicationResult{
			Success:          false,
			ActualReplicas:   0,
			RequiredReplicas: 0,
		}, fmt.Errorf("invalid replication request: missing tenantID or data")
	}

	// Get or create tenant-specific replication service
	rs := ur.getReplicationService(req.TenantID)

	// Perform replication
	result, err := rs.ReplicateFile(ctx, req.BucketName, req.FileName, req.Data)
	if err != nil {
		ur.logger.Printf("ERROR: Replication failed for %s:%s - %v", req.ArtifactType, req.ArtifactID, err)
		return &ReplicationResult{
			Success:          false,
			ActualReplicas:   0,
			RequiredReplicas: 0,
		}, err
	}

	// Return the result as-is (already in correct format)
	return result, nil
}

// GetReplicationStatus checks the replication status (not implemented in this version)
func (ur *UnifiedReplicator) GetReplicationStatus(ctx context.Context, artifactID string) (*ReplicationStatus, error) {
	return nil, fmt.Errorf("status check not yet implemented")
}

// getReplicationService retrieves or creates a cached replication service for a tenant
// This reduces database queries by caching tenant-specific replication configurations
func (ur *UnifiedReplicator) getReplicationService(tenantID string) *ReplicationService {
	ur.mu.RLock()
	rs, exists := ur.tenantCaches[tenantID]
	ur.mu.RUnlock()

	if exists {
		return rs
	}

	// Create new service if not cached
	ur.mu.Lock()
	defer ur.mu.Unlock()

	// Double-check after acquiring write lock
	if rs, exists := ur.tenantCaches[tenantID]; exists {
		return rs
	}

	rs = InitReplicationServiceWithDB(ur.logger, ur.db, tenantID)
	ur.tenantCaches[tenantID] = rs
	return rs
}

// InvalidateCache clears the cached replication service for a tenant
// Call this when replication configuration changes
func (ur *UnifiedReplicator) InvalidateCache(tenantID string) {
	ur.mu.Lock()
	defer ur.mu.Unlock()
	delete(ur.tenantCaches, tenantID)
	ur.logger.Printf("Replication cache invalidated for tenant: %s", tenantID)
}

// InvalidateAllCaches clears all cached replication services
func (ur *UnifiedReplicator) InvalidateAllCaches() {
	ur.mu.Lock()
	defer ur.mu.Unlock()
	ur.tenantCaches = make(map[string]*ReplicationService)
	ur.logger.Printf("All replication caches invalidated")
}
