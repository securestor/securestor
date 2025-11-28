package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/securestor/securestor/internal/models"
)

// ReplicationConfigService manages global replication configuration
type ReplicationConfigService struct {
	db *sql.DB
}

// NewReplicationConfigService creates a new service instance
func NewReplicationConfigService(db *sql.DB) *ReplicationConfigService {
	return &ReplicationConfigService{db: db}
}

// GetGlobalConfig retrieves the global replication configuration for a tenant
func (s *ReplicationConfigService) GetGlobalConfig(tenantID uuid.UUID) (*models.TenantReplicationConfig, error) {
	config := &models.TenantReplicationConfig{}

	err := s.db.QueryRow(`
		SELECT id, tenant_id, enable_replication_default, default_quorum_size, 
		       sync_frequency_default, node_health_check_interval, failover_timeout, 
		       created_at, updated_at
		FROM tenant_replication_config
		WHERE tenant_id = $1
	`, tenantID).Scan(
		&config.ID, &config.TenantID, &config.EnableReplicationDefault,
		&config.DefaultQuorumSize, &config.SyncFrequencyDefault,
		&config.NodeHealthCheckInterval, &config.FailoverTimeout,
		&config.CreatedAt, &config.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		// Return default config if not found
		return s.createDefaultConfig(tenantID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query global replication config: %w", err)
	}

	// Load associated nodes
	nodes, err := s.GetReplicationNodes(tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to load replication nodes: %w", err)
	}
	config.Nodes = nodes

	return config, nil
}

// UpdateGlobalConfig updates the global replication configuration
func (s *ReplicationConfigService) UpdateGlobalConfig(tenantID uuid.UUID, req *models.CreateTenantReplicationConfigRequest) (*models.TenantReplicationConfig, error) {
	// Get current config for audit trail
	oldConfig, err := s.GetGlobalConfig(tenantID)
	if err != nil && err.Error() != "no rows in result set" {
		return nil, err
	}

	now := time.Now()

	// Update or insert configuration
	result, err := s.db.Exec(`
		INSERT INTO tenant_replication_config 
		(tenant_id, enable_replication_default, default_quorum_size, 
		 sync_frequency_default, node_health_check_interval, failover_timeout, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (tenant_id) DO UPDATE SET
			enable_replication_default = $2,
			default_quorum_size = $3,
			sync_frequency_default = $4,
			node_health_check_interval = $5,
			failover_timeout = $6,
			updated_at = $8
	`, tenantID, req.EnableReplicationDefault, req.DefaultQuorumSize,
		req.SyncFrequencyDefault, req.NodeHealthCheckInterval,
		req.FailoverTimeout, now, now,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update global replication config: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return nil, fmt.Errorf("failed to update global replication config")
	}

	// Log to audit trail
	s.logConfigChange(tenantID, nil, "GLOBAL_CONFIG", "UPDATE", oldConfig, req)

	// Return updated config
	return s.GetGlobalConfig(tenantID)
}

// GetReplicationNodes retrieves all active replication nodes for a tenant
func (s *ReplicationConfigService) GetReplicationNodes(tenantID uuid.UUID) ([]models.ReplicationNode, error) {
	rows, err := s.db.Query(`
		SELECT id, tenant_id, node_name, node_path, node_type, priority, is_active, 
		       last_health_check, is_healthy, health_status, storage_available_gb, 
		       storage_total_gb, error_count, response_time_ms, s3_endpoint, s3_region,
		       s3_bucket, s3_access_key, s3_secret_key, s3_use_ssl, s3_path_prefix,
		       created_at, updated_at
		FROM replication_nodes
		WHERE tenant_id = $1
		ORDER BY priority DESC
	`, tenantID)

	if err != nil {
		return nil, fmt.Errorf("failed to query replication nodes: %w", err)
	}
	defer rows.Close()

	nodes := make([]models.ReplicationNode, 0)
	for rows.Next() {
		var node models.ReplicationNode
		var storageAvail, storageTotal sql.NullInt64
		var s3Endpoint, s3Region, s3Bucket, s3AccessKey, s3SecretKey, s3PathPrefix sql.NullString
		var s3UseSSL sql.NullBool

		err := rows.Scan(
			&node.ID, &node.TenantID, &node.NodeName, &node.NodePath, &node.NodeType, &node.Priority,
			&node.IsActive, &node.LastHealthCheck, &node.IsHealthy, &node.HealthStatus,
			&storageAvail, &storageTotal, &node.ErrorCount, &node.ResponseTimeMs,
			&s3Endpoint, &s3Region, &s3Bucket, &s3AccessKey, &s3SecretKey, &s3UseSSL, &s3PathPrefix,
			&node.CreatedAt, &node.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan replication node: %w", err)
		}

		// Handle NULL values
		if storageAvail.Valid {
			node.StorageAvailableGB = storageAvail.Int64
		}
		if storageTotal.Valid {
			node.StorageTotalGB = storageTotal.Int64
		}
		if s3Endpoint.Valid {
			node.S3Endpoint = s3Endpoint.String
		}
		if s3Region.Valid {
			node.S3Region = s3Region.String
		}
		if s3Bucket.Valid {
			node.S3Bucket = s3Bucket.String
		}
		if s3AccessKey.Valid {
			node.S3AccessKey = s3AccessKey.String
		}
		if s3SecretKey.Valid {
			node.S3SecretKey = s3SecretKey.String
		}
		if s3UseSSL.Valid {
			node.S3UseSSL = s3UseSSL.Bool
		}
		if s3PathPrefix.Valid {
			node.S3PathPrefix = s3PathPrefix.String
		}

		nodes = append(nodes, node)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating replication nodes: %w", err)
	}

	return nodes, nil
}

// GetReplicationNodeByID retrieves a specific replication node
func (s *ReplicationConfigService) GetReplicationNodeByID(tenantID, nodeID uuid.UUID) (*models.ReplicationNode, error) {
	node := &models.ReplicationNode{}
	var storageAvail, storageTotal sql.NullInt64
	var s3Endpoint, s3Region, s3Bucket, s3AccessKey, s3SecretKey, s3PathPrefix sql.NullString
	var s3UseSSL sql.NullBool

	err := s.db.QueryRow(`
		SELECT id, tenant_id, node_name, node_path, node_type, priority, is_active, 
		       last_health_check, is_healthy, health_status, storage_available_gb, 
		       storage_total_gb, error_count, response_time_ms, s3_endpoint, s3_region,
		       s3_bucket, s3_access_key, s3_secret_key, s3_use_ssl, s3_path_prefix,
		       created_at, updated_at
		FROM replication_nodes
		WHERE id = $1 AND tenant_id = $2
	`, nodeID, tenantID).Scan(
		&node.ID, &node.TenantID, &node.NodeName, &node.NodePath, &node.NodeType, &node.Priority,
		&node.IsActive, &node.LastHealthCheck, &node.IsHealthy, &node.HealthStatus,
		&storageAvail, &storageTotal, &node.ErrorCount, &node.ResponseTimeMs,
		&s3Endpoint, &s3Region, &s3Bucket, &s3AccessKey, &s3SecretKey, &s3UseSSL, &s3PathPrefix,
		&node.CreatedAt, &node.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("replication node not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query replication node: %w", err)
	}

	// Handle NULL values
	if storageAvail.Valid {
		node.StorageAvailableGB = storageAvail.Int64
	}
	if storageTotal.Valid {
		node.StorageTotalGB = storageTotal.Int64
	}
	if s3Endpoint.Valid {
		node.S3Endpoint = s3Endpoint.String
	}
	if s3Region.Valid {
		node.S3Region = s3Region.String
	}
	if s3Bucket.Valid {
		node.S3Bucket = s3Bucket.String
	}
	if s3AccessKey.Valid {
		node.S3AccessKey = s3AccessKey.String
	}
	if s3SecretKey.Valid {
		node.S3SecretKey = s3SecretKey.String
	}
	if s3UseSSL.Valid {
		node.S3UseSSL = s3UseSSL.Bool
	}
	if s3PathPrefix.Valid {
		node.S3PathPrefix = s3PathPrefix.String
	}

	return node, nil
}

// CreateReplicationNode creates a new replication node
func (s *ReplicationConfigService) CreateReplicationNode(tenantID uuid.UUID, req *models.CreateReplicationNodeRequest) (*models.ReplicationNode, error) {
	nodeID := uuid.New()
	now := time.Now()

	// Default node_type to 'local' if not specified
	nodeType := req.NodeType
	if nodeType == "" {
		nodeType = "local"
	}

	var node models.ReplicationNode
	var storageAvail, storageTotal sql.NullInt64
	var s3Endpoint, s3Region, s3Bucket, s3AccessKey, s3SecretKey, s3PathPrefix sql.NullString
	var s3UseSSL sql.NullBool

	err := s.db.QueryRow(`
		INSERT INTO replication_nodes 
		(id, tenant_id, node_name, node_path, node_type, priority, is_active, is_healthy, 
		 health_status, s3_endpoint, s3_region, s3_bucket, s3_access_key, s3_secret_key, 
		 s3_use_ssl, s3_path_prefix, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, true, true, 'unknown', $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id, tenant_id, node_name, node_path, node_type, priority, is_active, 
		          last_health_check, is_healthy, health_status, storage_available_gb, 
		          storage_total_gb, error_count, response_time_ms, s3_endpoint, s3_region, 
		          s3_bucket, s3_access_key, s3_secret_key, s3_use_ssl, s3_path_prefix, 
		          created_at, updated_at
	`, nodeID, tenantID, req.NodeName, req.NodePath, nodeType, req.Priority,
		req.S3Endpoint, req.S3Region, req.S3Bucket, req.S3AccessKey, req.S3SecretKey,
		req.S3UseSSL, req.S3PathPrefix, now, now).Scan(
		&node.ID, &node.TenantID, &node.NodeName, &node.NodePath, &node.NodeType, &node.Priority,
		&node.IsActive, &node.LastHealthCheck, &node.IsHealthy, &node.HealthStatus,
		&storageAvail, &storageTotal, &node.ErrorCount, &node.ResponseTimeMs,
		&s3Endpoint, &s3Region, &s3Bucket, &s3AccessKey, &s3SecretKey, &s3UseSSL, &s3PathPrefix,
		&node.CreatedAt, &node.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create replication node: %w", err)
	}

	// Handle NULL values
	if storageAvail.Valid {
		node.StorageAvailableGB = storageAvail.Int64
	}
	if storageTotal.Valid {
		node.StorageTotalGB = storageTotal.Int64
	}
	if s3Endpoint.Valid {
		node.S3Endpoint = s3Endpoint.String
	}
	if s3Region.Valid {
		node.S3Region = s3Region.String
	}
	if s3Bucket.Valid {
		node.S3Bucket = s3Bucket.String
	}
	if s3AccessKey.Valid {
		node.S3AccessKey = s3AccessKey.String
	}
	if s3SecretKey.Valid {
		node.S3SecretKey = s3SecretKey.String
	}
	if s3UseSSL.Valid {
		node.S3UseSSL = s3UseSSL.Bool
	}
	if s3PathPrefix.Valid {
		node.S3PathPrefix = s3PathPrefix.String
	}

	// Log to audit trail
	s.logConfigChange(tenantID, nil, "NODE_CONFIG", "CREATE", nil, req)

	return &node, nil
}

// UpdateReplicationNode updates a replication node
func (s *ReplicationConfigService) UpdateReplicationNode(tenantID, nodeID uuid.UUID, req *models.UpdateReplicationNodeRequest) (*models.ReplicationNode, error) {
	// Get current node for audit trail
	oldNode, err := s.GetReplicationNodeByID(tenantID, nodeID)
	if err != nil {
		return nil, err
	}

	now := time.Now()

	result, err := s.db.Exec(`
		UPDATE replication_nodes
		SET node_path = $1, priority = $2, is_active = $3, 
		    s3_endpoint = $4, s3_region = $5, s3_bucket = $6, 
		    s3_access_key = $7, s3_secret_key = $8, s3_use_ssl = $9, 
		    s3_path_prefix = $10, updated_at = $11
		WHERE id = $12 AND tenant_id = $13
	`, req.NodePath, req.Priority, req.IsActive,
		req.S3Endpoint, req.S3Region, req.S3Bucket,
		req.S3AccessKey, req.S3SecretKey, req.S3UseSSL,
		req.S3PathPrefix, now, nodeID, tenantID)

	if err != nil {
		return nil, fmt.Errorf("failed to update replication node: %w", err)
	}

	if affected, _ := result.RowsAffected(); affected == 0 {
		return nil, fmt.Errorf("replication node not found")
	}

	// Log to audit trail
	s.logConfigChange(tenantID, nil, "NODE_CONFIG", "UPDATE", oldNode, req)

	return s.GetReplicationNodeByID(tenantID, nodeID)
}

// DeleteReplicationNode deletes a replication node
func (s *ReplicationConfigService) DeleteReplicationNode(tenantID, nodeID uuid.UUID) error {
	// Get node for audit trail
	node, err := s.GetReplicationNodeByID(tenantID, nodeID)
	if err != nil {
		return err
	}

	result, err := s.db.Exec(`
		DELETE FROM replication_nodes
		WHERE id = $1 AND tenant_id = $2
	`, nodeID, tenantID)

	if err != nil {
		return fmt.Errorf("failed to delete replication node: %w", err)
	}

	if affected, _ := result.RowsAffected(); affected == 0 {
		return fmt.Errorf("replication node not found")
	}

	// Log to audit trail
	s.logConfigChange(tenantID, nil, "NODE_CONFIG", "DELETE", node, nil)

	return nil
}

// UpdateNodeHealth updates the health status of a replication node
func (s *ReplicationConfigService) UpdateNodeHealth(tenantID, nodeID uuid.UUID, isHealthy bool, healthStatus string, responseTimeMs int) error {
	now := time.Now()

	_, err := s.db.Exec(`
		UPDATE replication_nodes
		SET is_healthy = $1, health_status = $2, response_time_ms = $3, 
		    last_health_check = $4, error_count = CASE WHEN $1 = false THEN error_count + 1 ELSE 0 END
		WHERE id = $5 AND tenant_id = $6
	`, isHealthy, healthStatus, responseTimeMs, now, nodeID, tenantID)

	return err
}

// GetRepositoryReplicationSettings retrieves replication settings for a specific repository
func (s *ReplicationConfigService) GetRepositoryReplicationSettings(tenantID, repositoryID uuid.UUID) (*models.RepositoryReplicationSettings, error) {
	settings := &models.RepositoryReplicationSettings{
		RepositoryID: repositoryID,
	}

	var nodeIDsJSON sql.NullString
	var syncFreq sql.NullString
	var customQuorum sql.NullInt64

	err := s.db.QueryRow(`
		SELECT enable_replication, replication_node_ids, sync_frequency, 
		       override_global_replication, custom_quorum_size, last_replication_sync, 
		       replication_status
		FROM repositories
		WHERE id = $1 AND tenant_id = $2
	`, repositoryID, tenantID).Scan(
		&settings.EnableReplication, &nodeIDsJSON, &syncFreq,
		&settings.OverrideGlobalSettings, &customQuorum, &settings.LastReplicationSync,
		&settings.ReplicationStatus,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("repository not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query repository replication settings: %w", err)
	}

	// Parse node IDs
	if nodeIDsJSON.Valid {
		if err := json.Unmarshal([]byte(nodeIDsJSON.String), &settings.ReplicationNodeIDs); err != nil {
			return nil, fmt.Errorf("failed to parse node IDs: %w", err)
		}
	}

	if syncFreq.Valid {
		settings.SyncFrequency = syncFreq.String
	}

	if customQuorum.Valid {
		customQuorumInt := int(customQuorum.Int64)
		settings.CustomQuorumSize = &customQuorumInt
	}

	// Calculate effective configuration
	effectiveConfig, err := s.GetEffectiveReplicationConfig(tenantID, repositoryID)
	if err == nil {
		settings.EffectiveConfig = effectiveConfig
	}

	return settings, nil
}

// GetEffectiveReplicationConfig calculates the effective replication config (global or repo override)
func (s *ReplicationConfigService) GetEffectiveReplicationConfig(tenantID, repositoryID uuid.UUID) (*models.EffectiveReplicationConfig, error) {
	settings, err := s.GetRepositoryReplicationSettings(tenantID, repositoryID)
	if err != nil {
		return nil, err
	}

	globalConfig, err := s.GetGlobalConfig(tenantID)
	if err != nil {
		return nil, err
	}

	effective := &models.EffectiveReplicationConfig{
		IsUsingGlobalDefaults: !settings.OverrideGlobalSettings,
	}

	if settings.OverrideGlobalSettings {
		// Use repository-specific settings
		effective.EnableReplication = settings.EnableReplication
		effective.SyncFrequency = settings.SyncFrequency
		if settings.CustomQuorumSize != nil {
			effective.QuorumSize = *settings.CustomQuorumSize
		} else {
			effective.QuorumSize = globalConfig.DefaultQuorumSize
		}

		// Load specified nodes
		if len(settings.ReplicationNodeIDs) > 0 {
			for _, nodeID := range settings.ReplicationNodeIDs {
				node, err := s.GetReplicationNodeByID(tenantID, nodeID)
				if err == nil {
					effective.Nodes = append(effective.Nodes, *node)
				}
			}
		}
	} else {
		// Use global defaults
		effective.EnableReplication = globalConfig.EnableReplicationDefault
		effective.SyncFrequency = globalConfig.SyncFrequencyDefault
		effective.QuorumSize = globalConfig.DefaultQuorumSize
		effective.Nodes = globalConfig.Nodes
	}

	effective.RequiredReplicas = effective.QuorumSize
	effective.ReplicaCount = len(effective.Nodes)
	effective.LastSync = settings.LastReplicationSync
	effective.SyncStatus = settings.ReplicationStatus

	return effective, nil
}

// UpdateRepositoryReplicationSettings updates replication settings for a repository
func (s *ReplicationConfigService) UpdateRepositoryReplicationSettings(tenantID, repositoryID uuid.UUID, req *models.UpdateRepositoryReplicationRequest) error {
	// Get current settings for audit trail
	oldSettings, _ := s.GetRepositoryReplicationSettings(tenantID, repositoryID)

	nodeIDsJSON, _ := json.Marshal(req.ReplicationNodeIDs)

	now := time.Now()
	_, err := s.db.Exec(`
		UPDATE repositories
		SET enable_replication = $1, replication_node_ids = $2::uuid[], 
		    sync_frequency = $3, override_global_replication = $4, 
		    custom_quorum_size = $5, updated_at = $6
		WHERE id = $7 AND tenant_id = $8
	`, req.EnableReplication, string(nodeIDsJSON), req.SyncFrequency,
		req.OverrideGlobalSettings, req.CustomQuorumSize, now, repositoryID, tenantID)

	if err != nil {
		return fmt.Errorf("failed to update repository replication settings: %w", err)
	}

	// Log to audit trail
	s.logConfigChange(tenantID, &repositoryID, "REPO_CONFIG", "UPDATE", oldSettings, req)

	return nil
}

// GetConfigHistory retrieves the audit trail for replication configuration changes
func (s *ReplicationConfigService) GetConfigHistory(tenantID uuid.UUID, limit, offset int) ([]models.ReplicationConfigHistory, error) {
	rows, err := s.db.Query(`
		SELECT id, tenant_id, repository_id, entity_type, change_type, 
		       old_value, new_value, changed_by, ip_address, changed_at
		FROM replication_config_history
		WHERE tenant_id = $1
		ORDER BY changed_at DESC
		LIMIT $2 OFFSET $3
	`, tenantID, limit, offset)

	if err != nil {
		return nil, fmt.Errorf("failed to query config history: %w", err)
	}
	defer rows.Close()

	history := make([]models.ReplicationConfigHistory, 0)
	for rows.Next() {
		var record models.ReplicationConfigHistory
		var oldValueJSON, newValueJSON sql.NullString

		err := rows.Scan(
			&record.ID, &record.TenantID, &record.RepositoryID, &record.EntityType,
			&record.ChangeType, &oldValueJSON, &newValueJSON, &record.ChangedBy,
			&record.IPAddress, &record.ChangedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan config history: %w", err)
		}

		if oldValueJSON.Valid {
			_ = json.Unmarshal([]byte(oldValueJSON.String), &record.OldValue)
		}
		if newValueJSON.Valid {
			_ = json.Unmarshal([]byte(newValueJSON.String), &record.NewValue)
		}

		history = append(history, record)
	}

	return history, rows.Err()
}

// Private helper functions

func (s *ReplicationConfigService) createDefaultConfig(tenantID uuid.UUID) (*models.TenantReplicationConfig, error) {
	now := time.Now()
	config := &models.TenantReplicationConfig{
		ID:                       uuid.New(),
		TenantID:                 tenantID,
		EnableReplicationDefault: true,
		DefaultQuorumSize:        2,
		SyncFrequencyDefault:     "realtime",
		NodeHealthCheckInterval:  30,
		FailoverTimeout:          20,
		CreatedAt:                now,
		UpdatedAt:                now,
	}

	// Insert default config - use ON CONFLICT DO UPDATE to ensure we get the ID
	err := s.db.QueryRow(`
		INSERT INTO tenant_replication_config 
		(id, tenant_id, enable_replication_default, default_quorum_size, 
		 sync_frequency_default, node_health_check_interval, failover_timeout, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (tenant_id) DO UPDATE SET
			tenant_id = EXCLUDED.tenant_id
		RETURNING id, tenant_id, enable_replication_default, default_quorum_size, 
		          sync_frequency_default, node_health_check_interval, failover_timeout, 
		          created_at, updated_at
	`, config.ID, tenantID, config.EnableReplicationDefault, config.DefaultQuorumSize,
		config.SyncFrequencyDefault, config.NodeHealthCheckInterval, config.FailoverTimeout,
		now, now).Scan(
		&config.ID, &config.TenantID, &config.EnableReplicationDefault,
		&config.DefaultQuorumSize, &config.SyncFrequencyDefault,
		&config.NodeHealthCheckInterval, &config.FailoverTimeout,
		&config.CreatedAt, &config.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create default config: %w", err)
	}

	// Load associated nodes
	nodes, err := s.GetReplicationNodes(tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to load replication nodes: %w", err)
	}
	config.Nodes = nodes

	return config, nil
}

// CheckNodeHealth checks the filesystem health and storage capacity of a replication node
func (s *ReplicationConfigService) CheckNodeHealth(tenantID, nodeID uuid.UUID) (*models.ReplicationNode, error) {
	// Get the node
	node, err := s.GetReplicationNodeByID(tenantID, nodeID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	isHealthy := true
	healthStatus := "healthy"
	var storageAvailGB, storageTotalGB int64

	// Check based on node type
	if node.NodeType == "local" || node.NodeType == "" {
		// Local filesystem check
		fileInfo, err := os.Stat(node.NodePath)
		if err != nil {
			isHealthy = false
			if os.IsNotExist(err) {
				healthStatus = "path_not_found"
			} else {
				healthStatus = "path_inaccessible"
			}
		} else if !fileInfo.IsDir() {
			isHealthy = false
			healthStatus = "path_not_directory"
		} else {
			// Get disk usage using syscall
			var stat syscall.Statfs_t
			err = syscall.Statfs(node.NodePath, &stat)
			if err != nil {
				isHealthy = false
				healthStatus = "filesystem_error"
			} else {
				// Calculate storage in GB
				totalBytes := stat.Blocks * uint64(stat.Bsize)
				availBytes := stat.Bavail * uint64(stat.Bsize)
				storageTotalGB = int64(totalBytes / (1024 * 1024 * 1024))
				storageAvailGB = int64(availBytes / (1024 * 1024 * 1024))
			}
		}
	} else if node.NodeType == "s3" || node.NodeType == "minio" {
		// S3/MinIO health check
		ctx := context.Background()

		// Create S3 client configuration
		var cfg aws.Config

		if node.S3AccessKey != "" && node.S3SecretKey != "" {
			// Use provided credentials
			cfg, err = awsconfig.LoadDefaultConfig(ctx,
				awsconfig.WithRegion(node.S3Region),
				awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
					node.S3AccessKey,
					node.S3SecretKey,
					"",
				)),
			)
		} else {
			// Use default credential chain
			cfg, err = awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(node.S3Region))
		}

		if err != nil {
			isHealthy = false
			healthStatus = "s3_config_error"
		} else {
			// Create S3 client with custom endpoint if specified
			s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
				if node.S3Endpoint != "" {
					o.BaseEndpoint = aws.String(node.S3Endpoint)
					o.UsePathStyle = true // Required for MinIO
				}
			})

			// Try to list bucket to verify access
			_, err = s3Client.HeadBucket(ctx, &s3.HeadBucketInput{
				Bucket: aws.String(node.S3Bucket),
			})

			if err != nil {
				isHealthy = false
				healthStatus = "s3_bucket_inaccessible"
			} else {
				// Try to get bucket size (this is an approximation)
				// For production, you might want to use CloudWatch metrics or S3 analytics
				healthStatus = "healthy"
				// Note: S3 doesn't provide direct storage metrics without additional API calls
				// Setting placeholder values - in production, integrate with CloudWatch or S3 analytics
				storageTotalGB = 1000 // Placeholder: 1TB (unlimited for S3)
				storageAvailGB = 900  // Placeholder: 90% available
			}
		}
	} else {
		// Unsupported node type
		isHealthy = false
		healthStatus = "unsupported_node_type"
	}

	// Update the node in database
	_, err = s.db.Exec(`
		UPDATE replication_nodes
		SET last_health_check = $1, is_healthy = $2, health_status = $3,
		    storage_total_gb = $4, storage_available_gb = $5, updated_at = $6
		WHERE id = $7 AND tenant_id = $8
	`, now, isHealthy, healthStatus, storageTotalGB, storageAvailGB, now, nodeID, tenantID)

	if err != nil {
		return nil, fmt.Errorf("failed to update node health: %w", err)
	}

	// Return updated node
	node.LastHealthCheck = &now
	node.IsHealthy = isHealthy
	node.HealthStatus = healthStatus
	node.StorageTotalGB = storageTotalGB
	node.StorageAvailableGB = storageAvailGB
	node.UpdatedAt = now

	return node, nil
}

func (s *ReplicationConfigService) logConfigChange(tenantID uuid.UUID, repoID *uuid.UUID, entityType, changeType string, oldValue, newValue interface{}) {
	oldJSON, _ := json.Marshal(oldValue)
	newJSON, _ := json.Marshal(newValue)

	_, _ = s.db.Exec(`
		INSERT INTO replication_config_history 
		(id, tenant_id, repository_id, entity_type, change_type, old_value, new_value, changed_by, changed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, uuid.New(), tenantID, repoID, entityType, changeType,
		string(oldJSON), string(newJSON), "system", time.Now())
}
