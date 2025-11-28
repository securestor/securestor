package models

import (
	"time"

	"github.com/google/uuid"
)

type CreateRepositoryRequest struct {
	Name           string `json:"name" validate:"required,min=3,max=255"`
	Type           string `json:"type" validate:"required,oneof=docker npm maven pypi helm generic"`
	RepositoryType string `json:"repository_type" validate:"required,oneof=local remote cloud"`
	Description    string `json:"description"`
	PublicAccess   bool   `json:"public_access"`
	EnableIndexing bool   `json:"enable_indexing"`

	// Remote repository settings
	RemoteURL string `json:"remote_url"`
	Username  string `json:"username"`
	Password  string `json:"password"`

	// Encryption settings
	EnableEncryption bool   `json:"enable_encryption"`
	EncryptionKey    string `json:"encryption_key"`

	// Replication settings
	EnableReplication  bool     `json:"enable_replication"`
	ReplicationBuckets []string `json:"replication_buckets"`
	SyncFrequency      string   `json:"sync_frequency"`

	// Cloud storage settings
	CloudProvider   string `json:"cloud_provider"`
	Region          string `json:"region"`
	BucketName      string `json:"bucket_name"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	Endpoint        string `json:"endpoint"`
	GithubToken     string `json:"github_token"`
	GithubOrg       string `json:"github_org"`
	GithubRepo      string `json:"github_repo"`

	MaxStorageGB  int `json:"max_storage_gb"`
	RetentionDays int `json:"retention_days"`
}

type RepositoryResponse struct {
	ID                 uuid.UUID              `json:"id"`
	Name               string                 `json:"name"`
	Type               string                 `json:"type"`
	RepositoryType     string                 `json:"repository_type"`
	Description        string                 `json:"description"`
	PublicAccess       bool                   `json:"public_access"`
	EnableIndexing     bool                   `json:"enable_indexing"`
	RemoteURL          string                 `json:"remote_url,omitempty"`
	Status             string                 `json:"status"`
	EnableEncryption   bool                   `json:"enable_encryption"`
	EnableReplication  bool                   `json:"enable_replication"`
	ReplicationBuckets []string               `json:"replication_buckets,omitempty"`
	SyncFrequency      string                 `json:"sync_frequency,omitempty"`
	CloudProvider      string                 `json:"cloud_provider,omitempty"`
	Region             string                 `json:"region,omitempty"`
	BucketName         string                 `json:"bucket_name,omitempty"`
	ArtifactCount      int                    `json:"artifact_count"`
	TotalSize          string                 `json:"total_size"`
	LastActivity       *time.Time             `json:"last_activity,omitempty"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
	Settings           map[string]interface{} `json:"settings,omitempty"`
}

type RepositoryStats struct {
	TotalRepositories int     `json:"total_repositories"`
	LocalCount        int     `json:"local_count"`
	RemoteCount       int     `json:"remote_count"`
	VirtualCount      int     `json:"virtual_count"`
	TotalArtifacts    int     `json:"total_artifacts"`
	TotalStorageGB    float64 `json:"total_storage_gb"`
}
