package models

import (
	"time"

	"github.com/google/uuid"
)

// TenantReplicationConfig holds global replication settings for a tenant
type TenantReplicationConfig struct {
	ID                       uuid.UUID         `json:"id"`
	TenantID                 uuid.UUID         `json:"tenant_id"`
	EnableReplicationDefault bool              `json:"enable_replication_default"`
	DefaultQuorumSize        int               `json:"default_quorum_size"`
	SyncFrequencyDefault     string            `json:"sync_frequency_default"`
	NodeHealthCheckInterval  int               `json:"node_health_check_interval"`
	FailoverTimeout          int               `json:"failover_timeout"`
	CreatedAt                time.Time         `json:"created_at"`
	UpdatedAt                time.Time         `json:"updated_at"`
	Nodes                    []ReplicationNode `json:"nodes,omitempty"`
}

// ReplicationNode represents a storage node in the replication cluster
type ReplicationNode struct {
	ID                 uuid.UUID  `json:"id"`
	TenantID           uuid.UUID  `json:"tenant_id"`
	NodeName           string     `json:"node_name"`
	NodePath           string     `json:"node_path"`
	NodeType           string     `json:"node_type"` // local, s3, minio, gcs, azure
	Priority           int        `json:"priority"`
	IsActive           bool       `json:"is_active"`
	LastHealthCheck    *time.Time `json:"last_health_check,omitempty"`
	IsHealthy          bool       `json:"is_healthy"`
	HealthStatus       string     `json:"health_status"`
	StorageAvailableGB int64      `json:"storage_available_gb,omitempty"`
	StorageTotalGB     int64      `json:"storage_total_gb,omitempty"`
	ErrorCount         int        `json:"error_count"`
	ResponseTimeMs     int        `json:"response_time_ms"`
	// S3-compatible storage fields
	S3Endpoint   string    `json:"s3_endpoint,omitempty"`
	S3Region     string    `json:"s3_region,omitempty"`
	S3Bucket     string    `json:"s3_bucket,omitempty"`
	S3AccessKey  string    `json:"s3_access_key,omitempty"`
	S3SecretKey  string    `json:"s3_secret_key,omitempty"`
	S3UseSSL     bool      `json:"s3_use_ssl"`
	S3PathPrefix string    `json:"s3_path_prefix,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ReplicationConfigHistory tracks changes to replication configuration
type ReplicationConfigHistory struct {
	ID           uuid.UUID              `json:"id"`
	TenantID     uuid.UUID              `json:"tenant_id"`
	RepositoryID *uuid.UUID             `json:"repository_id,omitempty"`
	EntityType   string                 `json:"entity_type"` // GLOBAL_CONFIG, REPO_CONFIG, NODE_CONFIG
	ChangeType   string                 `json:"change_type"` // CREATE, UPDATE, DELETE
	OldValue     map[string]interface{} `json:"old_value,omitempty"`
	NewValue     map[string]interface{} `json:"new_value,omitempty"`
	ChangedBy    string                 `json:"changed_by,omitempty"`
	IPAddress    string                 `json:"ip_address,omitempty"`
	ChangedAt    time.Time              `json:"changed_at"`
}

// CreateTenantReplicationConfigRequest is the request body for creating/updating global replication config
type CreateTenantReplicationConfigRequest struct {
	EnableReplicationDefault bool   `json:"enable_replication_default" validate:"required"`
	DefaultQuorumSize        int    `json:"default_quorum_size" validate:"required,min=1,max=5"`
	SyncFrequencyDefault     string `json:"sync_frequency_default" validate:"required,oneof=realtime hourly daily weekly"`
	NodeHealthCheckInterval  int    `json:"node_health_check_interval" validate:"required,min=10,max=300"`
	FailoverTimeout          int    `json:"failover_timeout" validate:"required,min=5,max=300"`
}

// CreateReplicationNodeRequest is the request body for creating a replication node
type CreateReplicationNodeRequest struct {
	NodeName string `json:"node_name" validate:"required,min=1,max=255"`
	NodePath string `json:"node_path" validate:"required,min=1,max=1024"`
	NodeType string `json:"node_type" validate:"required,oneof=local s3 minio gcs azure"`
	Priority int    `json:"priority" validate:"required,min=1,max=100"`
	IsActive bool   `json:"is_active"` // Always created as active, but accepted from frontend
	// S3-compatible storage fields (required when node_type is s3/minio)
	S3Endpoint   string `json:"s3_endpoint"`
	S3Region     string `json:"s3_region"`
	S3Bucket     string `json:"s3_bucket"`
	S3AccessKey  string `json:"s3_access_key"`
	S3SecretKey  string `json:"s3_secret_key"`
	S3UseSSL     bool   `json:"s3_use_ssl"`
	S3PathPrefix string `json:"s3_path_prefix"`
}

// UpdateReplicationNodeRequest is the request body for updating a replication node
type UpdateReplicationNodeRequest struct {
	NodePath string `json:"node_path" validate:"required,min=1,max=1024"`
	Priority int    `json:"priority" validate:"required,min=1,max=100"`
	IsActive bool   `json:"is_active" validate:"required"`
	// S3-compatible storage fields
	S3Endpoint   string `json:"s3_endpoint"`
	S3Region     string `json:"s3_region"`
	S3Bucket     string `json:"s3_bucket"`
	S3AccessKey  string `json:"s3_access_key"`
	S3SecretKey  string `json:"s3_secret_key"`
	S3UseSSL     bool   `json:"s3_use_ssl"`
	S3PathPrefix string `json:"s3_path_prefix"`
}

// ReplicationNodeHealthResponse contains node health status
type ReplicationNodeHealthResponse struct {
	ID                 uuid.UUID  `json:"id"`
	Name               string     `json:"name"`
	IsHealthy          bool       `json:"is_healthy"`
	HealthStatus       string     `json:"health_status"`
	LastCheck          *time.Time `json:"last_check,omitempty"`
	StorageAvailableGB int64      `json:"storage_available_gb,omitempty"`
	StorageTotalGB     int64      `json:"storage_total_gb,omitempty"`
	ErrorCount         int        `json:"error_count"`
	ResponseTimeMs     int        `json:"response_time_ms"`
}

// EffectiveReplicationConfig represents the final replication config for a repository
type EffectiveReplicationConfig struct {
	EnableReplication     bool              `json:"enable_replication"`
	Nodes                 []ReplicationNode `json:"nodes"`
	QuorumSize            int               `json:"quorum_size"`
	SyncFrequency         string            `json:"sync_frequency"`
	IsUsingGlobalDefaults bool              `json:"is_using_global_defaults"`
	LastSync              *time.Time        `json:"last_sync,omitempty"`
	SyncStatus            string            `json:"sync_status"`
	ReplicaCount          int               `json:"replica_count"`
	RequiredReplicas      int               `json:"required_replicas"`
}

// RepositoryReplicationSettings represents replication config for a specific repository
type RepositoryReplicationSettings struct {
	RepositoryID           uuid.UUID                   `json:"repository_id"`
	EnableReplication      bool                        `json:"enable_replication"`
	ReplicationNodeIDs     []uuid.UUID                 `json:"replication_node_ids"`
	SyncFrequency          string                      `json:"sync_frequency"`
	OverrideGlobalSettings bool                        `json:"override_global_replication"`
	CustomQuorumSize       *int                        `json:"custom_quorum_size,omitempty"`
	LastReplicationSync    *time.Time                  `json:"last_replication_sync,omitempty"`
	ReplicationStatus      string                      `json:"replication_status"`
	EffectiveConfig        *EffectiveReplicationConfig `json:"effective_config,omitempty"`
}

// UpdateRepositoryReplicationRequest is the request body for updating repo replication settings
type UpdateRepositoryReplicationRequest struct {
	EnableReplication      bool        `json:"enable_replication"`
	ReplicationNodeIDs     []uuid.UUID `json:"replication_node_ids"`
	SyncFrequency          string      `json:"sync_frequency" validate:"oneof=realtime hourly daily weekly"`
	OverrideGlobalSettings bool        `json:"override_global_replication"`
	CustomQuorumSize       *int        `json:"custom_quorum_size" validate:"omitempty,min=1,max=5"`
}
