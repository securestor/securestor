package cache

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/securestor/securestor/internal/models"
)

// CacheService provides caching operations for SecureStor
type CacheService struct {
	redis  *RedisClient
	logger *log.Logger
	config CacheConfig
}

// CacheConfig holds cache configuration
type CacheConfig struct {
	DefaultTTL    time.Duration
	ArtifactTTL   time.Duration
	ScanResultTTL time.Duration
	WorkflowTTL   time.Duration
	SessionTTL    time.Duration
}

// NewCacheService creates a new cache service
func NewCacheService(redis *RedisClient, config CacheConfig, logger *log.Logger) *CacheService {
	// Set default TTL values if not provided
	if config.DefaultTTL == 0 {
		config.DefaultTTL = 1 * time.Hour
	}
	if config.ArtifactTTL == 0 {
		config.ArtifactTTL = 24 * time.Hour
	}
	if config.ScanResultTTL == 0 {
		config.ScanResultTTL = 12 * time.Hour
	}
	if config.WorkflowTTL == 0 {
		config.WorkflowTTL = 6 * time.Hour
	}
	if config.SessionTTL == 0 {
		config.SessionTTL = 24 * time.Hour
	}

	return &CacheService{
		redis:  redis,
		logger: logger,
		config: config,
	}
}

// Artifact caching operations

// CacheArtifact stores an artifact in cache
func (c *CacheService) CacheArtifact(ctx context.Context, artifact *models.Artifact) error {
	key := fmt.Sprintf("artifact:%s", artifact.ID.String())
	return c.redis.Set(ctx, key, artifact, c.config.ArtifactTTL)
}

// GetArtifact retrieves an artifact from cache
func (c *CacheService) GetArtifact(ctx context.Context, artifactID uuid.UUID) (*models.Artifact, error) {
	key := fmt.Sprintf("artifact:%s", artifactID.String())
	var artifact models.Artifact

	err := c.redis.Get(ctx, key, &artifact)
	if err != nil {
		if err == ErrKeyNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &artifact, nil
}

// InvalidateArtifact removes an artifact from cache
func (c *CacheService) InvalidateArtifact(ctx context.Context, artifactID uuid.UUID) error {
	key := fmt.Sprintf("artifact:%s", artifactID.String())
	return c.redis.Delete(ctx, key)
}

// Scan result caching operations

// CacheScanResult stores a scan result in cache
func (c *CacheService) CacheScanResult(ctx context.Context, scanResult *models.ScanResult) error {
	key := fmt.Sprintf("scan_result:%d", scanResult.ID)
	return c.redis.Set(ctx, key, scanResult, c.config.ScanResultTTL)
}

// GetScanResult retrieves a scan result from cache
func (c *CacheService) GetScanResult(ctx context.Context, scanID int64) (*models.ScanResult, error) {
	key := fmt.Sprintf("scan_result:%d", scanID)
	var scanResult models.ScanResult

	err := c.redis.Get(ctx, key, &scanResult)
	if err != nil {
		if err == ErrKeyNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &scanResult, nil
}

// CacheScanStatus stores scan status for quick lookup
func (c *CacheService) CacheScanStatus(ctx context.Context, artifactID int64, scanID int64, status string) error {
	key := fmt.Sprintf("scan_status:%d", artifactID)
	statusData := map[string]interface{}{
		"scan_id":   scanID,
		"status":    status,
		"timestamp": time.Now(),
	}
	return c.redis.Set(ctx, key, statusData, c.config.ScanResultTTL)
}

// GetScanStatus retrieves current scan status for an artifact
func (c *CacheService) GetScanStatus(ctx context.Context, artifactID int64) (map[string]interface{}, error) {
	key := fmt.Sprintf("scan_status:%d", artifactID)
	var status map[string]interface{}

	err := c.redis.Get(ctx, key, &status)
	if err != nil {
		if err == ErrKeyNotFound {
			return nil, nil
		}
		return nil, err
	}

	return status, nil
}

// Workflow caching operations

// CacheWorkflowResult stores workflow execution result
func (c *CacheService) CacheWorkflowResult(ctx context.Context, artifactID int64, workflowName string, result interface{}) error {
	key := fmt.Sprintf("workflow_result:%d:%s", artifactID, workflowName)
	return c.redis.Set(ctx, key, result, c.config.WorkflowTTL)
}

// GetWorkflowResult retrieves workflow execution result
func (c *CacheService) GetWorkflowResult(ctx context.Context, artifactID int64, workflowName string, dest interface{}) error {
	key := fmt.Sprintf("workflow_result:%d:%s", artifactID, workflowName)
	return c.redis.Get(ctx, key, dest)
}

// Session management

// CacheSession stores user session data
func (c *CacheService) CacheSession(ctx context.Context, sessionID string, sessionData interface{}) error {
	key := fmt.Sprintf("session:%s", sessionID)
	return c.redis.Set(ctx, key, sessionData, c.config.SessionTTL)
}

// GetSession retrieves user session data
func (c *CacheService) GetSession(ctx context.Context, sessionID string, dest interface{}) error {
	key := fmt.Sprintf("session:%s", sessionID)
	return c.redis.Get(ctx, key, dest)
}

// InvalidateSession removes a user session
func (c *CacheService) InvalidateSession(ctx context.Context, sessionID string) error {
	key := fmt.Sprintf("session:%s", sessionID)
	return c.redis.Delete(ctx, key)
}

// Rate limiting

// IncrementRateLimit increments rate limit counter for an IP/user
func (c *CacheService) IncrementRateLimit(ctx context.Context, identifier string, window time.Duration) (int64, error) {
	key := fmt.Sprintf("rate_limit:%s", identifier)

	// Check if key exists
	exists, err := c.redis.Exists(ctx, key)
	if err != nil {
		return 0, err
	}

	count, err := c.redis.Increment(ctx, key)
	if err != nil {
		return 0, err
	}

	// Set expiration on first increment
	if !exists {
		err = c.redis.SetExpiration(ctx, key, window)
		if err != nil {
			c.logger.Printf("Failed to set rate limit expiration: %v", err)
		}
	}

	return count, nil
}

// Statistics caching

// CacheStats stores dashboard statistics
func (c *CacheService) CacheStats(ctx context.Context, statsType string, data interface{}) error {
	key := fmt.Sprintf("stats:%s", statsType)
	return c.redis.Set(ctx, key, data, 10*time.Minute) // Stats cache for 10 minutes
}

// GetStats retrieves dashboard statistics
func (c *CacheService) GetStats(ctx context.Context, statsType string, dest interface{}) error {
	key := fmt.Sprintf("stats:%s", statsType)
	return c.redis.Get(ctx, key, dest)
}

// Real-time notifications

// PublishScanUpdate publishes scan progress update
func (c *CacheService) PublishScanUpdate(ctx context.Context, artifactID int64, update interface{}) error {
	channel := fmt.Sprintf("scan_updates:%d", artifactID)
	return c.redis.PublishMessage(ctx, channel, update)
}

// PublishWorkflowUpdate publishes workflow progress update
func (c *CacheService) PublishWorkflowUpdate(ctx context.Context, artifactID int64, workflowName string, update interface{}) error {
	channel := fmt.Sprintf("workflow_updates:%d:%s", artifactID, workflowName)
	return c.redis.PublishMessage(ctx, channel, update)
}

// SubscribeToScanUpdates subscribes to scan updates for an artifact
func (c *CacheService) SubscribeToScanUpdates(ctx context.Context, artifactID int64) *RedisSubscription {
	channel := fmt.Sprintf("scan_updates:%d", artifactID)
	pubsub := c.redis.Subscribe(ctx, channel)
	return &RedisSubscription{pubsub: pubsub}
}

// Job queue operations (for async processing)

// EnqueueJob adds a job to the processing queue
func (c *CacheService) EnqueueJob(ctx context.Context, queueName string, job interface{}) error {
	key := fmt.Sprintf("queue:%s", queueName)
	return c.redis.AddToList(ctx, key, job)
}

// DequeueJob removes and returns a job from the queue
func (c *CacheService) DequeueJob(ctx context.Context, queueName string, dest interface{}) error {
	key := fmt.Sprintf("queue:%s", queueName)
	var jobs []interface{}

	err := c.redis.GetListRange(ctx, key, -1, -1, &jobs)
	if err != nil {
		return err
	}

	if len(jobs) == 0 {
		return ErrKeyNotFound
	}

	// Remove the job from the queue
	// Note: This is a simplified implementation. In production, consider using BRPOP or Lua scripts for atomicity

	return nil
}

// Utility operations

// FlushAll clears all cache data (use with extreme caution!)
func (c *CacheService) FlushAll(ctx context.Context) error {
	c.logger.Println("WARNING: Flushing all cache data")
	return c.redis.FlushDB(ctx)
}

// Health check
func (c *CacheService) Health(ctx context.Context) error {
	return c.redis.Health(ctx)
}

// GetRedisClient returns the underlying Redis client for advanced operations
func (c *CacheService) GetRedisClient() *RedisClient {
	return c.redis
}

// RedisSubscription wraps Redis pub/sub subscription
type RedisSubscription struct {
	pubsub interface{} // redis.PubSub - using interface to avoid circular imports
}

// Close closes the subscription
func (rs *RedisSubscription) Close() error {
	// Implementation would close the pub/sub connection
	return nil
}
