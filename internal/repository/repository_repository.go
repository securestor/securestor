package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/securestor/securestor/internal/models"
)

type RepositoryRepository struct {
	db *sql.DB
}

func NewRepositoryRepository(db *sql.DB) *RepositoryRepository {
	return &RepositoryRepository{db: db}
}

func (r *RepositoryRepository) Create(req *models.CreateRepositoryRequest) (*models.RepositoryResponse, error) {
	// Get the first (default) tenant for this repository
	var tenantID uuid.UUID
	err := r.db.QueryRow(`SELECT tenant_id FROM tenants ORDER BY created_at LIMIT 1`).Scan(&tenantID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no tenant found - please create a tenant first")
		}
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	// Check if repository name already exists
	exists, err := r.NameExists(req.Name)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("repository with name '%s' already exists", req.Name)
	}

	// Prepare settings
	settings := map[string]interface{}{
		"max_storage_gb":  req.MaxStorageGB,
		"retention_days":  req.RetentionDays,
		"username":        req.Username,
		"repository_type": req.RepositoryType,
	}

	// Don't store password in settings - handle securely
	if req.Password != "" {
		// TODO: Hash and store password securely
		settings["has_credentials"] = true
	}

	// Store encryption key config if encryption is enabled
	if req.EnableEncryption && req.EncryptionKey != "" {
		encryptionConfig := map[string]interface{}{
			"key_type":  "custom",
			"algorithm": "AES-256",
		}
		settings["encryption_config"] = encryptionConfig
		// TODO: Securely store the actual encryption key (vault/secrets manager)
		settings["has_encryption_key"] = true
	}

	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal settings: %w", err)
	}

	// Prepare encryption config
	var replicationTargetsJSON interface{}
	if req.EnableReplication && len(req.ReplicationBuckets) > 0 {
		replicationTargetsBytes, _ := json.Marshal(req.ReplicationBuckets)
		replicationTargetsJSON = replicationTargetsBytes
	} else {
		replicationTargetsJSON = nil
	}

	// Prepare cloud config
	var cloudConfigJSON interface{}
	var cloudProvider, cloudRegion sql.NullString

	if req.RepositoryType == "cloud" {
		cloudConfig := map[string]interface{}{
			"provider": req.CloudProvider,
		}

		switch req.CloudProvider {
		case "s3", "s3-compatible":
			cloudConfig["bucket_name"] = req.BucketName
			cloudConfig["region"] = req.Region
			// TODO: Securely store access credentials
			if req.AccessKeyID != "" {
				cloudConfig["has_credentials"] = true
			}
			if req.CloudProvider == "s3-compatible" {
				cloudConfig["endpoint"] = req.Endpoint
			}
		case "github":
			cloudConfig["org"] = req.GithubOrg
			cloudConfig["repo"] = req.GithubRepo
			// TODO: Securely store GitHub token
			if req.GithubToken != "" {
				cloudConfig["has_token"] = true
			}
		case "aws-ecr":
			cloudConfig["region"] = req.Region
			// TODO: Securely store AWS credentials
			if req.AccessKeyID != "" {
				cloudConfig["has_credentials"] = true
			}
		}

		cloudConfigBytes, _ := json.Marshal(cloudConfig)
		cloudConfigJSON = cloudConfigBytes
		cloudProvider = sql.NullString{String: req.CloudProvider, Valid: true}
		if req.Region != "" {
			cloudRegion = sql.NullString{String: req.Region, Valid: true}
		}
	} else {
		cloudConfigJSON = nil
	}

	// Set default sync frequency if not provided and replication is enabled
	syncFrequency := req.SyncFrequency
	if req.EnableReplication && syncFrequency == "" {
		syncFrequency = "daily"
	} else if !req.EnableReplication {
		syncFrequency = "daily" // Default value for column
	}

	query := `
        INSERT INTO repositories (
            tenant_id, name, type, description, 
            public_access, enable_indexing, remote_url, 
            status, settings, enable_encryption,
            enable_replication, replication_targets, sync_frequency,
            cloud_provider, cloud_region, cloud_config
        )
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
        RETURNING repository_id, created_at, updated_at
    `

	var id uuid.UUID
	var createdAt, updatedAt time.Time

	err = r.db.QueryRow(
		query,
		tenantID,
		req.Name,
		req.Type,
		req.Description,
		req.PublicAccess,
		req.EnableIndexing,
		req.RemoteURL,
		"active",
		settingsJSON,
		req.EnableEncryption,
		req.EnableReplication,
		replicationTargetsJSON,
		syncFrequency,
		cloudProvider,
		cloudRegion,
		cloudConfigJSON,
	).Scan(&id, &createdAt, &updatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create repository: %w", err)
	}

	// Return created repository
	return &models.RepositoryResponse{
		ID:                 id,
		Name:               req.Name,
		Type:               req.Type,
		RepositoryType:     req.RepositoryType,
		Description:        req.Description,
		PublicAccess:       req.PublicAccess,
		EnableIndexing:     req.EnableIndexing,
		RemoteURL:          req.RemoteURL,
		Status:             "active",
		EnableEncryption:   req.EnableEncryption,
		EnableReplication:  req.EnableReplication,
		ReplicationBuckets: req.ReplicationBuckets,
		SyncFrequency:      req.SyncFrequency,
		CloudProvider:      req.CloudProvider,
		Region:             req.Region,
		BucketName:         req.BucketName,
		ArtifactCount:      0,
		TotalSize:          "0 B",
		CreatedAt:          createdAt,
		UpdatedAt:          updatedAt,
		Settings:           settings,
	}, nil
}

func (r *RepositoryRepository) NameExists(name string) (bool, error) {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM repositories WHERE name = $1)"
	err := r.db.QueryRow(query, name).Scan(&exists)
	return exists, err
}

func (r *RepositoryRepository) GetByIDWithStats(id uuid.UUID) (*models.RepositoryResponse, error) {
	query := `
        SELECT 
            r.repository_id, r.name, r.type, r.description,
            r.public_access, r.enable_indexing, r.remote_url, r.status,
            r.settings, r.created_at, r.updated_at,
            r.enable_encryption, r.enable_replication, r.replication_targets,
            r.sync_frequency, r.cloud_provider, r.cloud_region, r.cloud_config,
            COUNT(a.artifact_id) as artifact_count,
            COALESCE(SUM(a.size), 0) as total_size,
            MAX(a.uploaded_at) as last_activity
        FROM repositories r
        LEFT JOIN artifacts a ON r.repository_id = a.repository_id
        WHERE r.repository_id = $1
        GROUP BY r.repository_id
    `

	var repo models.RepositoryResponse
	var settingsJSON, cloudConfigJSON, replicationTargetsJSON []byte
	var totalSize int64
	var lastActivity sql.NullTime
	var syncFrequency, cloudProvider, cloudRegion sql.NullString

	err := r.db.QueryRow(query, id).Scan(
		&repo.ID,
		&repo.Name,
		&repo.Type,
		&repo.Description,
		&repo.PublicAccess,
		&repo.EnableIndexing,
		&repo.RemoteURL,
		&repo.Status,
		&settingsJSON,
		&repo.CreatedAt,
		&repo.UpdatedAt,
		&repo.EnableEncryption,
		&repo.EnableReplication,
		&replicationTargetsJSON,
		&syncFrequency,
		&cloudProvider,
		&cloudRegion,
		&cloudConfigJSON,
		&repo.ArtifactCount,
		&totalSize,
		&lastActivity,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("repository not found")
		}
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	// Parse settings
	if len(settingsJSON) > 0 {
		json.Unmarshal(settingsJSON, &repo.Settings)
		// Extract repository_type from settings
		if repoType, ok := repo.Settings["repository_type"].(string); ok {
			repo.RepositoryType = repoType
		}
	}

	// Parse replication targets
	if len(replicationTargetsJSON) > 0 {
		json.Unmarshal(replicationTargetsJSON, &repo.ReplicationBuckets)
	}

	// Parse cloud config and extract bucket name
	if len(cloudConfigJSON) > 0 {
		var cloudConfig map[string]interface{}
		json.Unmarshal(cloudConfigJSON, &cloudConfig)
		if bucketName, ok := cloudConfig["bucket_name"].(string); ok {
			repo.BucketName = bucketName
		}
	}

	// Set optional string fields
	if syncFrequency.Valid {
		repo.SyncFrequency = syncFrequency.String
	}
	if cloudProvider.Valid {
		repo.CloudProvider = cloudProvider.String
	}
	if cloudRegion.Valid {
		repo.Region = cloudRegion.String
	}

	// Format size
	repo.TotalSize = formatSize(totalSize)

	// Set last activity
	if lastActivity.Valid {
		repo.LastActivity = &lastActivity.Time
	}

	return &repo, nil
}

func (r *RepositoryRepository) ListWithStats() ([]models.RepositoryResponse, error) {
	query := `
        SELECT 
            r.repository_id, r.name, r.type, r.description,
            r.public_access, r.enable_indexing, r.remote_url, r.status,
            r.settings, r.created_at, r.updated_at,
            r.enable_encryption, r.enable_replication, r.replication_targets,
            r.sync_frequency, r.cloud_provider, r.cloud_region, r.cloud_config,
            COUNT(a.artifact_id) as artifact_count,
            COALESCE(SUM(a.size), 0) as total_size,
            MAX(a.uploaded_at) as last_activity
        FROM repositories r
        LEFT JOIN artifacts a ON r.repository_id = a.repository_id
        GROUP BY r.repository_id
        ORDER BY r.created_at DESC
    `

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}
	defer rows.Close()

	var repositories []models.RepositoryResponse

	for rows.Next() {
		var repo models.RepositoryResponse
		var settingsJSON, cloudConfigJSON, replicationTargetsJSON []byte
		var totalSize int64
		var lastActivity sql.NullTime
		var remoteURL, syncFrequency, cloudProvider, cloudRegion sql.NullString

		err := rows.Scan(
			&repo.ID,
			&repo.Name,
			&repo.Type,
			&repo.Description,
			&repo.PublicAccess,
			&repo.EnableIndexing,
			&remoteURL,
			&repo.Status,
			&settingsJSON,
			&repo.CreatedAt,
			&repo.UpdatedAt,
			&repo.EnableEncryption,
			&repo.EnableReplication,
			&replicationTargetsJSON,
			&syncFrequency,
			&cloudProvider,
			&cloudRegion,
			&cloudConfigJSON,
			&repo.ArtifactCount,
			&totalSize,
			&lastActivity,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan repository: %w", err)
		}

		// Parse settings
		if len(settingsJSON) > 0 {
			json.Unmarshal(settingsJSON, &repo.Settings)
			// Extract repository_type from settings
			if repoType, ok := repo.Settings["repository_type"].(string); ok {
				repo.RepositoryType = repoType
			}
		}

		// Parse replication targets
		if len(replicationTargetsJSON) > 0 {
			json.Unmarshal(replicationTargetsJSON, &repo.ReplicationBuckets)
		}

		// Parse cloud config and extract bucket name
		if len(cloudConfigJSON) > 0 {
			var cloudConfig map[string]interface{}
			json.Unmarshal(cloudConfigJSON, &cloudConfig)
			if bucketName, ok := cloudConfig["bucket_name"].(string); ok {
				repo.BucketName = bucketName
			}
		}

		// Set optional string fields
		if remoteURL.Valid {
			repo.RemoteURL = remoteURL.String
		}
		if syncFrequency.Valid {
			repo.SyncFrequency = syncFrequency.String
		}
		if cloudProvider.Valid {
			repo.CloudProvider = cloudProvider.String
		}
		if cloudRegion.Valid {
			repo.Region = cloudRegion.String
		}

		// Format size
		repo.TotalSize = formatSize(totalSize)

		// Set last activity
		if lastActivity.Valid {
			repo.LastActivity = &lastActivity.Time
		}

		repositories = append(repositories, repo)
	}

	return repositories, nil
}

func (r *RepositoryRepository) GetStats() (*models.RepositoryStats, error) {
	query := `
        SELECT 
            COUNT(*) as total,
            COUNT(CASE WHEN settings->>'repository_type' = 'local' THEN 1 END) as local_count,
            COUNT(CASE WHEN settings->>'repository_type' = 'remote' THEN 1 END) as remote_count,
            COUNT(CASE WHEN settings->>'repository_type' = 'cloud' THEN 1 END) as virtual_count
        FROM repositories
    `

	var stats models.RepositoryStats
	err := r.db.QueryRow(query).Scan(
		&stats.TotalRepositories,
		&stats.LocalCount,
		&stats.RemoteCount,
		&stats.VirtualCount,
	)

	if err != nil {
		return nil, err
	}

	// Get artifact stats
	artifactQuery := `
        SELECT 
            COUNT(*) as total_artifacts,
            COALESCE(SUM(size), 0) / (1024.0 * 1024.0 * 1024.0) as total_storage_gb
        FROM artifacts
    `

	err = r.db.QueryRow(artifactQuery).Scan(
		&stats.TotalArtifacts,
		&stats.TotalStorageGB,
	)

	return &stats, err
}

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func (r *RepositoryRepository) GetByID(id uuid.UUID) (*models.Repository, error) {
	query := `
        SELECT repository_id, name, type, description, public_access, 
               enable_indexing, COALESCE(remote_url, ''), created_at, updated_at
        FROM repositories
        WHERE repository_id = $1
    `

	repo := &models.Repository{}
	err := r.db.QueryRow(query, id).Scan(
		&repo.ID,
		&repo.Name,
		&repo.Type,
		&repo.Description,
		&repo.PublicAccess,
		&repo.EnableIndexing,
		&repo.RemoteURL,
		&repo.CreatedAt,
		&repo.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("repository not found")
		}
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	return repo, nil
}

func (r *RepositoryRepository) GetByName(name string) (*models.Repository, error) {
	query := `
        SELECT repository_id, name, type, description, public_access, 
               enable_indexing, COALESCE(remote_url, ''), created_at, updated_at
        FROM repositories
        WHERE name = $1
    `

	repo := &models.Repository{}
	err := r.db.QueryRow(query, name).Scan(
		&repo.ID,
		&repo.Name,
		&repo.Type,
		&repo.Description,
		&repo.PublicAccess,
		&repo.EnableIndexing,
		&repo.RemoteURL,
		&repo.CreatedAt,
		&repo.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("repository not found")
		}
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	return repo, nil
}

func (r *RepositoryRepository) List() ([]models.Repository, error) {
	query := `
        SELECT repository_id, name, type, description, public_access, 
               enable_indexing, COALESCE(remote_url, ''), created_at, updated_at
        FROM repositories
        ORDER BY name
    `

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}
	defer rows.Close()

	var repositories []models.Repository
	for rows.Next() {
		var repo models.Repository
		err := rows.Scan(
			&repo.ID,
			&repo.Name,
			&repo.Type,
			&repo.Description,
			&repo.PublicAccess,
			&repo.EnableIndexing,
			&repo.RemoteURL,
			&repo.CreatedAt,
			&repo.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan repository: %w", err)
		}

		repositories = append(repositories, repo)
	}

	return repositories, nil
}

func (r *RepositoryRepository) Update(repo *models.Repository) error {
	query := `
        UPDATE repositories
        SET name = $1, type = $2, description = $3,
            public_access = $4, enable_indexing = $5, remote_url = $6, updated_at = $7
        WHERE repository_id = $8
    `

	_, err := r.db.Exec(query,
		repo.Name,
		repo.Type,
		repo.Description,
		repo.PublicAccess,
		repo.EnableIndexing,
		repo.RemoteURL,
		time.Now(),
		repo.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update repository: %w", err)
	}

	return nil
}

func (r *RepositoryRepository) Delete(id uuid.UUID) error {
	_, err := r.db.Exec("DELETE FROM repositories WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete repository: %w", err)
	}
	return nil
}

func (r *RepositoryRepository) GetRepositoryTenantID(repositoryID uuid.UUID) (uuid.UUID, error) {
	var tenantID uuid.UUID
	err := r.db.QueryRow("SELECT tenant_id FROM repositories WHERE repository_id = $1", repositoryID).Scan(&tenantID)
	if err != nil {
		if err == sql.ErrNoRows {
			return uuid.Nil, fmt.Errorf("repository not found")
		}
		return uuid.Nil, fmt.Errorf("failed to get repository tenant_id: %w", err)
	}
	return tenantID, nil
}

// CreateWithTenant creates a repository for a specific tenant
func (r *RepositoryRepository) CreateWithTenant(tenantID uuid.UUID, req *models.CreateRepositoryRequest) (*models.RepositoryResponse, error) {
	// Check if repository name already exists for this tenant
	var exists bool
	err := r.db.QueryRow("SELECT EXISTS(SELECT 1 FROM repositories WHERE tenant_id = $1 AND name = $2)", tenantID, req.Name).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("repository with name '%s' already exists", req.Name)
	}

	// Prepare settings
	settings := map[string]interface{}{
		"max_storage_gb":  req.MaxStorageGB,
		"retention_days":  req.RetentionDays,
		"username":        req.Username,
		"repository_type": req.RepositoryType,
	}

	if req.Password != "" {
		settings["has_credentials"] = true
	}

	if req.EnableEncryption && req.EncryptionKey != "" {
		encryptionConfig := map[string]interface{}{
			"key_type":  "custom",
			"algorithm": "AES-256",
		}
		settings["encryption_config"] = encryptionConfig
		settings["has_encryption_key"] = true
	}

	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal settings: %w", err)
	}

	// Prepare replication targets
	var replicationTargetsJSON interface{}
	if req.EnableReplication && len(req.ReplicationBuckets) > 0 {
		replicationTargetsBytes, _ := json.Marshal(req.ReplicationBuckets)
		replicationTargetsJSON = replicationTargetsBytes
	} else {
		replicationTargetsJSON = nil
	}

	// Prepare cloud config
	var cloudConfigJSON interface{}
	var cloudProvider, cloudRegion sql.NullString

	if req.RepositoryType == "cloud" {
		cloudConfig := map[string]interface{}{
			"provider": req.CloudProvider,
		}

		switch req.CloudProvider {
		case "s3", "s3-compatible":
			cloudConfig["bucket_name"] = req.BucketName
			cloudConfig["region"] = req.Region
			if req.AccessKeyID != "" {
				cloudConfig["has_credentials"] = true
			}
			if req.CloudProvider == "s3-compatible" {
				cloudConfig["endpoint"] = req.Endpoint
			}
		case "github":
			cloudConfig["org"] = req.GithubOrg
			cloudConfig["repo"] = req.GithubRepo
			if req.GithubToken != "" {
				cloudConfig["has_token"] = true
			}
		case "aws-ecr":
			cloudConfig["region"] = req.Region
			if req.AccessKeyID != "" {
				cloudConfig["has_credentials"] = true
			}
		}

		cloudConfigBytes, _ := json.Marshal(cloudConfig)
		cloudConfigJSON = cloudConfigBytes
		cloudProvider = sql.NullString{String: req.CloudProvider, Valid: true}
		if req.Region != "" {
			cloudRegion = sql.NullString{String: req.Region, Valid: true}
		}
	} else {
		cloudConfigJSON = nil
	}

	syncFrequency := req.SyncFrequency
	if req.EnableReplication && syncFrequency == "" {
		syncFrequency = "daily"
	} else if !req.EnableReplication {
		syncFrequency = "daily"
	}

	query := `
        INSERT INTO repositories (
            tenant_id, name, type, description, 
            public_access, enable_indexing, remote_url, 
            status, settings, enable_encryption,
            enable_replication, replication_targets, sync_frequency,
            cloud_provider, cloud_region, cloud_config
        )
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
        RETURNING repository_id, created_at, updated_at
    `

	var id uuid.UUID
	var createdAt, updatedAt time.Time

	err = r.db.QueryRow(
		query,
		tenantID,
		req.Name,
		req.Type,
		req.Description,
		req.PublicAccess,
		req.EnableIndexing,
		req.RemoteURL,
		"active",
		settingsJSON,
		req.EnableEncryption,
		req.EnableReplication,
		replicationTargetsJSON,
		syncFrequency,
		cloudProvider,
		cloudRegion,
		cloudConfigJSON,
	).Scan(&id, &createdAt, &updatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create repository: %w", err)
	}

	return &models.RepositoryResponse{
		ID:                 id,
		Name:               req.Name,
		Type:               req.Type,
		RepositoryType:     req.RepositoryType,
		Description:        req.Description,
		PublicAccess:       req.PublicAccess,
		EnableIndexing:     req.EnableIndexing,
		RemoteURL:          req.RemoteURL,
		Status:             "active",
		EnableEncryption:   req.EnableEncryption,
		EnableReplication:  req.EnableReplication,
		ReplicationBuckets: req.ReplicationBuckets,
		SyncFrequency:      req.SyncFrequency,
		CloudProvider:      req.CloudProvider,
		Region:             req.Region,
		BucketName:         req.BucketName,
		ArtifactCount:      0,
		TotalSize:          "0 B",
		CreatedAt:          createdAt,
		UpdatedAt:          updatedAt,
		Settings:           settings,
	}, nil
}

// ListByTenant lists all repositories for a specific tenant
func (r *RepositoryRepository) ListByTenant(tenantID uuid.UUID) ([]models.Repository, error) {
	query := `
        SELECT repository_id, tenant_id, name, type, description, public_access, 
               enable_indexing, COALESCE(remote_url, ''), created_at, updated_at
        FROM repositories
        WHERE tenant_id = $1
        ORDER BY name
    `

	rows, err := r.db.Query(query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}
	defer rows.Close()

	var repositories []models.Repository
	for rows.Next() {
		var repo models.Repository
		err := rows.Scan(
			&repo.ID,
			&repo.TenantID,
			&repo.Name,
			&repo.Type,
			&repo.Description,
			&repo.PublicAccess,
			&repo.EnableIndexing,
			&repo.RemoteURL,
			&repo.CreatedAt,
			&repo.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan repository: %w", err)
		}

		repositories = append(repositories, repo)
	}

	return repositories, nil
}

// ListWithStatsByTenant lists repositories with stats for a specific tenant
func (r *RepositoryRepository) ListWithStatsByTenant(tenantID uuid.UUID) ([]models.RepositoryResponse, error) {
	query := `
        SELECT 
            r.repository_id, r.name, r.type, r.description,
            r.public_access, r.enable_indexing, r.remote_url, r.status,
            r.settings, r.created_at, r.updated_at,
            r.enable_encryption, r.enable_replication, r.replication_targets,
            r.sync_frequency, r.cloud_provider, r.cloud_region, r.cloud_config,
            COUNT(a.artifact_id) as artifact_count,
            COALESCE(SUM(a.size), 0) as total_size,
            MAX(a.uploaded_at) as last_activity
        FROM repositories r
        LEFT JOIN artifacts a ON r.repository_id = a.repository_id AND a.tenant_id = $1
        WHERE r.tenant_id = $1
        GROUP BY r.repository_id
        ORDER BY r.created_at DESC
    `

	rows, err := r.db.Query(query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}
	defer rows.Close()

	var repositories []models.RepositoryResponse

	for rows.Next() {
		var repo models.RepositoryResponse
		var settingsJSON, cloudConfigJSON, replicationTargetsJSON []byte
		var totalSize int64
		var lastActivity sql.NullTime
		var remoteURL, syncFrequency, cloudProvider, cloudRegion sql.NullString

		err := rows.Scan(
			&repo.ID,
			&repo.Name,
			&repo.Type,
			&repo.Description,
			&repo.PublicAccess,
			&repo.EnableIndexing,
			&remoteURL,
			&repo.Status,
			&settingsJSON,
			&repo.CreatedAt,
			&repo.UpdatedAt,
			&repo.EnableEncryption,
			&repo.EnableReplication,
			&replicationTargetsJSON,
			&syncFrequency,
			&cloudProvider,
			&cloudRegion,
			&cloudConfigJSON,
			&repo.ArtifactCount,
			&totalSize,
			&lastActivity,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan repository: %w", err)
		}

		if len(settingsJSON) > 0 {
			json.Unmarshal(settingsJSON, &repo.Settings)
			if repoType, ok := repo.Settings["repository_type"].(string); ok {
				repo.RepositoryType = repoType
			}
		}

		if len(replicationTargetsJSON) > 0 {
			json.Unmarshal(replicationTargetsJSON, &repo.ReplicationBuckets)
		}

		if len(cloudConfigJSON) > 0 {
			var cloudConfig map[string]interface{}
			json.Unmarshal(cloudConfigJSON, &cloudConfig)
			if bucketName, ok := cloudConfig["bucket_name"].(string); ok {
				repo.BucketName = bucketName
			}
		}

		if remoteURL.Valid {
			repo.RemoteURL = remoteURL.String
		}
		if syncFrequency.Valid {
			repo.SyncFrequency = syncFrequency.String
		}
		if cloudProvider.Valid {
			repo.CloudProvider = cloudProvider.String
		}
		if cloudRegion.Valid {
			repo.Region = cloudRegion.String
		}

		repo.TotalSize = formatSize(totalSize)

		if lastActivity.Valid {
			repo.LastActivity = &lastActivity.Time
		}

		repositories = append(repositories, repo)
	}

	return repositories, nil
}

// GetStatsByTenant gets repository stats for a specific tenant
func (r *RepositoryRepository) GetStatsByTenant(tenantID uuid.UUID) (*models.RepositoryStats, error) {
	query := `
        SELECT 
            COUNT(*) as total,
            COUNT(CASE WHEN settings->>'repository_type' = 'local' THEN 1 END) as local_count,
            COUNT(CASE WHEN settings->>'repository_type' = 'remote' THEN 1 END) as remote_count,
            COUNT(CASE WHEN settings->>'repository_type' = 'cloud' THEN 1 END) as virtual_count
        FROM repositories
        WHERE tenant_id = $1
    `

	var stats models.RepositoryStats
	err := r.db.QueryRow(query, tenantID).Scan(
		&stats.TotalRepositories,
		&stats.LocalCount,
		&stats.RemoteCount,
		&stats.VirtualCount,
	)

	if err != nil {
		return nil, err
	}

	// Get artifact stats for this tenant
	artifactQuery := `
        SELECT 
            COUNT(*) as total_artifacts,
            COALESCE(SUM(a.size), 0) / (1024.0 * 1024.0 * 1024.0) as total_storage_gb
        FROM artifacts a
        JOIN repositories r ON a.repository_id = r.repository_id
        WHERE r.tenant_id = $1
    `

	err = r.db.QueryRow(artifactQuery, tenantID).Scan(
		&stats.TotalArtifacts,
		&stats.TotalStorageGB,
	)

	return &stats, err
}
