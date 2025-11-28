package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/securestor/securestor/internal/logger"
	"github.com/securestor/securestor/internal/replicate"
)

// ReplicationMixin provides reusable replication functionality for all handler types
// This implements the Mixin pattern to avoid code duplication
type ReplicationMixin struct {
	replicator *replicate.UnifiedReplicator
	logger     *logger.Logger
}

// NewReplicationMixin creates a new replication mixin
func NewReplicationMixin(replicator *replicate.UnifiedReplicator, logger *logger.Logger) *ReplicationMixin {
	return &ReplicationMixin{
		replicator: replicator,
		logger:     logger,
	}
}

// ReplicateAsync performs asynchronous artifact replication
// This is the recommended method for most use cases to avoid blocking the upload response
func (rm *ReplicationMixin) ReplicateAsync(req *replicate.ReplicationRequest) {
	if rm.replicator == nil || rm.logger == nil {
		rm.safeLog("WARNING: Replicator or logger not configured, skipping replication")
		return
	}

	go rm.replicateWithRecovery(req)
}

// ReplicateSync performs synchronous artifact replication
// Use this when you need to ensure replication completes before returning
func (rm *ReplicationMixin) ReplicateSync(ctx context.Context, req *replicate.ReplicationRequest) (*replicate.ReplicationResult, error) {
	if rm.replicator == nil {
		return nil, fmt.Errorf("replicator not configured")
	}

	return rm.replicator.ReplicateArtifact(ctx, req)
}

// replicateWithRecovery performs replication with panic recovery
func (rm *ReplicationMixin) replicateWithRecovery(req *replicate.ReplicationRequest) {
	defer func() {
		if r := recover(); r != nil {
			rm.safeLog("PANIC: Replication panic recovered for %s:%s - %v", req.ArtifactType, req.ArtifactID, r)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	result, err := rm.replicator.ReplicateArtifact(ctx, req)

	if err != nil {
		rm.logger.Printf("ERROR: Replication failed for %s artifact %s: %v",
			req.ArtifactType, req.ArtifactID, err)
		return
	}

	if result != nil && result.Success && result.ActualReplicas > 0 {
		rm.logger.Printf("✅ Replication successful: type=%s, artifact=%s, replicas=%d/%d, checksum=%s",
			req.ArtifactType, req.ArtifactID, result.ActualReplicas, result.RequiredReplicas, result.Checksum)
	} else {
		replicas := 0
		if result != nil {
			replicas = result.ActualReplicas
		}
		rm.logger.Printf("⚠️ Replication partial/failed: type=%s, artifact=%s, replicas=%d",
			req.ArtifactType, req.ArtifactID, replicas)
	}
}

// safeLog logs a message safely, handling nil logger
func (rm *ReplicationMixin) safeLog(format string, v ...interface{}) {
	if rm.logger != nil {
		rm.logger.Printf(format, v...)
	}
}

// IsEnabled returns whether replication is enabled
func (rm *ReplicationMixin) IsEnabled() bool {
	return rm.replicator != nil && rm.logger != nil
}
