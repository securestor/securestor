package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/securestor/securestor/internal/models"
)

type ArtifactRepository struct {
	db *sql.DB
}

func NewArtifactRepository(db *sql.DB) *ArtifactRepository {
	return &ArtifactRepository{db: db}
}

func (r *ArtifactRepository) Create(artifact *models.Artifact) error {
	return r.CreateOrUpdate(artifact)
}

func (r *ArtifactRepository) CreateOrUpdate(artifact *models.Artifact) error {
	metadataJSON, err := json.Marshal(artifact.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Log the marshaled JSON for debugging
	fmt.Printf("[DEBUG] Marshaled metadata JSON: %s\n", string(metadataJSON))

	// First try to get existing artifact
	existing, err := r.GetByNameVersionRepo(artifact.Name, artifact.Version, artifact.RepositoryID)
	if err == nil && existing != nil {
		// Artifact exists, update it
		artifact.ID = existing.ID
		return r.Update(artifact)
	}

	// Artifact doesn't exist, create new one
	// Use the tenant_id from the artifact (set during upload), not from the repository
	// This ensures storage path matches the database record
	tenantID := artifact.TenantID
	if tenantID == uuid.Nil {
		// Fallback: get tenant_id from the repository if not set on artifact
		err = r.db.QueryRow("SELECT tenant_id FROM repositories WHERE repository_id = $1", artifact.RepositoryID).Scan(&tenantID)
		if err != nil {
			return fmt.Errorf("failed to get tenant_id from artifact or repository: %w", err)
		}
	}

	// Marshal encryption metadata to JSON if present
	var encryptionMetadataJSON interface{}
	if len(artifact.EncryptionMetadata) > 0 {
		jsonBytes, err := json.Marshal(artifact.EncryptionMetadata)
		if err != nil {
			return fmt.Errorf("failed to marshal encryption metadata: %w", err)
		}
		encryptionMetadataJSON = jsonBytes
	} else {
		// Use NULL for empty encryption metadata (not empty string)
		encryptionMetadataJSON = nil
	}

	query := `
        INSERT INTO artifacts (artifact_id, name, version, type, repository_id, tenant_id, size, checksum, 
                              uploaded_by, license, metadata, uploaded_at, created_at, updated_at,
                              encrypted, encryption_version, encrypted_dek, encryption_algorithm, encryption_metadata)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
        RETURNING artifact_id, uploaded_at, created_at, updated_at
    `

	now := time.Now()
	err = r.db.QueryRow(query,
		uuid.New(), // artifact_id
		artifact.Name,
		artifact.Version,
		artifact.Type,
		artifact.RepositoryID,
		tenantID, // tenant_id - use from artifact (set during upload)
		artifact.Size,
		artifact.Checksum,
		nil, // uploaded_by - TODO: Get user UUID from auth context
		artifact.License,
		metadataJSON,
		now, // uploaded_at
		now, // created_at
		now, // updated_at
		artifact.Encrypted,
		artifact.EncryptionVersion,
		artifact.EncryptedDEK,
		artifact.EncryptionAlgorithm,
		encryptionMetadataJSON,
	).Scan(&artifact.ID, &artifact.UploadedAt, &artifact.CreatedAt, &artifact.UpdatedAt)

	if err != nil {
		// Log the actual PostgreSQL error for debugging
		fmt.Printf("[ERROR] Failed to insert artifact into database: %v\n", err)
		fmt.Printf("[ERROR] Artifact details - Name: %s, Version: %s, Type: %s\n", artifact.Name, artifact.Version, artifact.Type)
		fmt.Printf("[ERROR] Metadata JSON length: %d bytes\n", len(metadataJSON))
		if encryptionMetadataJSON != nil {
			if b, ok := encryptionMetadataJSON.([]byte); ok {
				fmt.Printf("[ERROR] Encryption metadata JSON length: %d bytes\n", len(b))
			}
		} else {
			fmt.Printf("[ERROR] Encryption metadata is NULL\n")
		}

		// Check if it's a duplicate key error - return success
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			fmt.Printf("[INFO] Artifact %s-%s already exists, retrieving existing record\n", artifact.Name, artifact.Version)
			existing, getErr := r.GetByNameVersionRepo(artifact.Name, artifact.Version, artifact.RepositoryID)
			if getErr == nil && existing != nil {
				// Copy existing artifact data
				artifact.ID = existing.ID
				artifact.UploadedAt = existing.UploadedAt
				artifact.CreatedAt = existing.CreatedAt
				artifact.UpdatedAt = time.Now()
				fmt.Printf("[INFO] Successfully retrieved existing artifact with ID: %s\n", artifact.ID)
				return nil // Return success for duplicate
			}
			// Even if we can't fetch the existing record, treat duplicate as success
			fmt.Printf("[WARN] Could not fetch existing artifact but treating duplicate as success\n")
			artifact.ID = uuid.New() // Generate a temporary ID
			artifact.UploadedAt = now
			artifact.CreatedAt = now
			artifact.UpdatedAt = now
			return nil
		}
		return fmt.Errorf("failed to create artifact: %w", err)
	}

	// Add tags
	if len(artifact.Tags) > 0 {
		if err := r.addTags(artifact.ID, artifact.Tags); err != nil {
			return err
		}
	}

	// Initialize indexing (skip if table doesn't exist)
	_ = r.initializeIndexing(tenantID, artifact.ID)

	return nil
}

func (r *ArtifactRepository) GetByID(id uuid.UUID) (*models.Artifact, error) {
	query := `
        SELECT a.artifact_id, a.tenant_id, a.name, a.version, a.type, a.repository_id, r.name as repository,
               a.size, a.checksum, a.uploaded_by, a.uploaded_at, a.downloads,
               a.license, a.metadata, a.created_at, a.updated_at,
               a.encrypted, a.encryption_version, a.encrypted_dek, a.encryption_algorithm, a.encryption_metadata
        FROM artifacts a
        JOIN repositories r ON a.repository_id = r.repository_id
        WHERE a.artifact_id = $1
    `

	artifact := &models.Artifact{}
	var metadataJSON []byte
	var encryptionMetadataJSON []byte

	err := r.db.QueryRow(query, id).Scan(
		&artifact.ID,
		&artifact.TenantID,
		&artifact.Name,
		&artifact.Version,
		&artifact.Type,
		&artifact.RepositoryID,
		&artifact.Repository,
		&artifact.Size,
		&artifact.Checksum,
		&artifact.UploadedBy,
		&artifact.UploadedAt,
		&artifact.Downloads,
		&artifact.License,
		&metadataJSON,
		&artifact.CreatedAt,
		&artifact.UpdatedAt,
		&artifact.Encrypted,
		&artifact.EncryptionVersion,
		&artifact.EncryptedDEK,
		&artifact.EncryptionAlgorithm,
		&encryptionMetadataJSON,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("artifact not found")
		}
		return nil, fmt.Errorf("failed to get artifact: %w", err)
	}

	if err := json.Unmarshal(metadataJSON, &artifact.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	// Unmarshal encryption metadata if present
	if len(encryptionMetadataJSON) > 0 {
		if err := json.Unmarshal(encryptionMetadataJSON, &artifact.EncryptionMetadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal encryption metadata: %w", err)
		}
	}

	// Load associated data
	artifact.Tags, _ = r.getTags(id)
	artifact.Compliance, _ = r.getCompliance(id)
	artifact.Vulnerabilities, _ = r.getVulnerabilities(id)
	artifact.Indexing, _ = r.getIndexing(id)

	artifact.SizeFormatted = formatSize(artifact.Size)

	return artifact, nil
}

func (r *ArtifactRepository) GetByNameVersionRepo(name, version string, repositoryID uuid.UUID) (*models.Artifact, error) {
	query := `
        SELECT a.artifact_id, a.tenant_id, a.name, a.version, a.type, a.repository_id, r.name as repository,
               a.size, a.checksum, a.uploaded_by, a.uploaded_at, a.downloads,
               a.license, a.metadata, a.created_at, a.updated_at
        FROM artifacts a
        JOIN repositories r ON a.repository_id = r.repository_id
        WHERE a.name = $1 AND a.version = $2 AND a.repository_id = $3
    `

	artifact := &models.Artifact{}
	var metadataJSON []byte

	err := r.db.QueryRow(query, name, version, repositoryID).Scan(
		&artifact.ID,
		&artifact.TenantID,
		&artifact.Name,
		&artifact.Version,
		&artifact.Type,
		&artifact.RepositoryID,
		&artifact.Repository,
		&artifact.Size,
		&artifact.Checksum,
		&artifact.UploadedBy,
		&artifact.UploadedAt,
		&artifact.Downloads,
		&artifact.License,
		&metadataJSON,
		&artifact.CreatedAt,
		&artifact.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get artifact: %w", err)
	}

	if err := json.Unmarshal(metadataJSON, &artifact.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	// Load associated data
	artifact.Tags, _ = r.getTags(artifact.ID)
	artifact.Compliance, _ = r.getCompliance(artifact.ID)
	artifact.Vulnerabilities, _ = r.getVulnerabilities(artifact.ID)
	artifact.Indexing, _ = r.getIndexing(artifact.ID)

	artifact.SizeFormatted = formatSize(artifact.Size)

	return artifact, nil
}

func (r *ArtifactRepository) Update(artifact *models.Artifact) error {
	metadataJSON, err := json.Marshal(artifact.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Marshal encryption metadata to JSON if present
	var encryptionMetadataJSON []byte
	if artifact.EncryptionMetadata != nil {
		encryptionMetadataJSON, err = json.Marshal(artifact.EncryptionMetadata)
		if err != nil {
			return fmt.Errorf("failed to marshal encryption metadata: %w", err)
		}
	}

	query := `
        UPDATE artifacts 
        SET type = $2, size = $3, checksum = $4, uploaded_by = $5, 
            license = $6, metadata = $7, updated_at = CURRENT_TIMESTAMP,
            encrypted = $8, encryption_version = $9, encrypted_dek = $10, 
            encryption_algorithm = $11, encryption_metadata = $12
        WHERE artifact_id = $1
        RETURNING uploaded_at, created_at, updated_at
    `

	err = r.db.QueryRow(query,
		artifact.ID,
		artifact.Type,
		artifact.Size,
		artifact.Checksum,
		artifact.UploadedBy,
		artifact.License,
		metadataJSON,
		artifact.Encrypted,
		artifact.EncryptionVersion,
		artifact.EncryptedDEK,
		artifact.EncryptionAlgorithm,
		encryptionMetadataJSON,
	).Scan(&artifact.UploadedAt, &artifact.CreatedAt, &artifact.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to update artifact: %w", err)
	}

	// Update tags if provided
	if len(artifact.Tags) > 0 {
		// Delete existing tags
		_, err := r.db.Exec("DELETE FROM artifact_tags WHERE artifact_id = $1", artifact.ID)
		if err != nil {
			return fmt.Errorf("failed to delete existing tags: %w", err)
		}

		// Add new tags
		if err := r.addTags(artifact.ID, artifact.Tags); err != nil {
			return err
		}
	}

	return nil
}

func (r *ArtifactRepository) List(filter *models.ArtifactFilter) ([]models.Artifact, int, error) {
	whereClause, args := r.buildWhereClause(filter)

	countQuery := fmt.Sprintf(`
        SELECT COUNT(a.artifact_id)
        FROM artifacts a
        JOIN repositories r ON a.repository_id = r.repository_id
        %s
    `, whereClause)

	var total int
	if err := r.db.QueryRow(countQuery, args[:len(args)-2]...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count artifacts: %w", err)
	}

	// Build ORDER BY clause
	orderBy := "ORDER BY a.uploaded_at DESC, a.artifact_id DESC"
	if filter.SortBy != "" {
		direction := "ASC"
		if filter.SortOrder == "desc" {
			direction = "DESC"
		}
		switch filter.SortBy {
		case "name":
			orderBy = fmt.Sprintf("ORDER BY a.name %s, a.artifact_id DESC", direction)
		case "version":
			orderBy = fmt.Sprintf("ORDER BY a.version %s, a.artifact_id DESC", direction)
		case "size":
			orderBy = fmt.Sprintf("ORDER BY a.size %s, a.artifact_id DESC", direction)
		case "uploaded_at":
			orderBy = fmt.Sprintf("ORDER BY a.uploaded_at %s, a.artifact_id DESC", direction)
		case "downloads":
			orderBy = fmt.Sprintf("ORDER BY a.downloads %s, a.artifact_id DESC", direction)
		case "type":
			orderBy = fmt.Sprintf("ORDER BY a.type %s, a.artifact_id DESC", direction)
		default:
			// Default to uploaded_at DESC if invalid sort field
			orderBy = "ORDER BY a.uploaded_at DESC, a.artifact_id DESC"
		}
	}

	query := fmt.Sprintf(`
        SELECT a.artifact_id, a.tenant_id, a.name, a.version, a.type, a.repository_id, r.name as repository,
               a.size, a.checksum, a.uploaded_by, a.uploaded_at, a.downloads,
               a.license, a.metadata, a.created_at, a.updated_at
        FROM artifacts a
        JOIN repositories r ON a.repository_id = r.repository_id
        %s
        %s
        LIMIT $%d OFFSET $%d
    `, whereClause, orderBy, len(args)-1, len(args))

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list artifacts: %w", err)
	}
	defer rows.Close()

	var artifacts []models.Artifact
	for rows.Next() {
		var artifact models.Artifact
		var metadataJSON []byte

		err := rows.Scan(
			&artifact.ID,
			&artifact.TenantID,
			&artifact.Name,
			&artifact.Version,
			&artifact.Type,
			&artifact.RepositoryID,
			&artifact.Repository,
			&artifact.Size,
			&artifact.Checksum,
			&artifact.UploadedBy,
			&artifact.UploadedAt,
			&artifact.Downloads,
			&artifact.License,
			&metadataJSON,
			&artifact.CreatedAt,
			&artifact.UpdatedAt,
		)

		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan artifact: %w", err)
		}

		if err := json.Unmarshal(metadataJSON, &artifact.Metadata); err != nil {
			return nil, 0, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		// Load associated data
		artifact.Tags, _ = r.getTags(artifact.ID)
		artifact.Compliance, _ = r.getCompliance(artifact.ID)
		artifact.Vulnerabilities, _ = r.getVulnerabilities(artifact.ID)
		artifact.SizeFormatted = formatSize(artifact.Size)

		artifacts = append(artifacts, artifact)
	}

	return artifacts, total, nil
}

func (r *ArtifactRepository) buildWhereClause(filter *models.ArtifactFilter) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	argPos := 1

	// Repository ID filter
	if filter.RepositoryID != nil {
		conditions = append(conditions, fmt.Sprintf("a.repository_id = $%d", argPos))
		args = append(args, *filter.RepositoryID)
		argPos++
	}

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf(`(
            a.name ILIKE $%d OR 
            a.version ILIKE $%d OR 
            a.checksum ILIKE $%d
        )`, argPos, argPos, argPos))
		args = append(args, "%"+filter.Search+"%")
		argPos++
	}

	if len(filter.Types) > 0 {
		placeholders := make([]string, len(filter.Types))
		for i, t := range filter.Types {
			placeholders[i] = fmt.Sprintf("$%d", argPos)
			args = append(args, t)
			argPos++
		}
		conditions = append(conditions, fmt.Sprintf("a.type IN (%s)", strings.Join(placeholders, ",")))
	}

	if len(filter.Repositories) > 0 {
		placeholders := make([]string, len(filter.Repositories))
		for i, repo := range filter.Repositories {
			placeholders[i] = fmt.Sprintf("$%d", argPos)
			args = append(args, repo)
			argPos++
		}
		conditions = append(conditions, fmt.Sprintf("r.name IN (%s)", strings.Join(placeholders, ",")))
	}

	// TODO: Re-implement compliance status filtering with proper subquery
	// if len(filter.ComplianceStatus) > 0 {
	//     placeholders := make([]string, len(filter.ComplianceStatus))
	//     for i, s := range filter.ComplianceStatus {
	//         placeholders[i] = fmt.Sprintf("$%d", argPos)
	//         args = append(args, s)
	//         argPos++
	//     }
	//     conditions = append(conditions, fmt.Sprintf("ca.status IN (%s)", strings.Join(placeholders, ",")))
	// }

	// Date range filters
	if filter.CreatedAfter != nil {
		conditions = append(conditions, fmt.Sprintf("a.created_at >= $%d", argPos))
		args = append(args, *filter.CreatedAfter)
		argPos++
	}

	if filter.CreatedBefore != nil {
		conditions = append(conditions, fmt.Sprintf("a.created_at <= $%d", argPos))
		args = append(args, *filter.CreatedBefore)
		argPos++
	}

	// Size filters
	if filter.MinSize != nil {
		conditions = append(conditions, fmt.Sprintf("a.size >= $%d", argPos))
		args = append(args, *filter.MinSize)
		argPos++
	}

	if filter.MaxSize != nil {
		conditions = append(conditions, fmt.Sprintf("a.size <= $%d", argPos))
		args = append(args, *filter.MaxSize)
		argPos++
	}

	// Add limit and offset
	if filter.Limit == 0 {
		filter.Limit = 50
	}
	args = append(args, filter.Limit, filter.Offset)

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	return whereClause, args
}

func (r *ArtifactRepository) addTags(artifactID uuid.UUID, tags []string) error {
	// Get tenant_id from the artifact
	var tenantID uuid.UUID
	err := r.db.QueryRow("SELECT tenant_id FROM artifacts WHERE artifact_id = $1", artifactID).Scan(&tenantID)
	if err != nil {
		return fmt.Errorf("failed to get tenant_id for artifact: %w", err)
	}

	for _, tag := range tags {
		_, err := r.db.Exec(
			"INSERT INTO artifact_tags (artifact_id, tenant_id, tag) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING",
			artifactID, tenantID, tag,
		)
		if err != nil {
			return fmt.Errorf("failed to add tag: %w", err)
		}
	}
	return nil
}

func (r *ArtifactRepository) getTags(artifactID uuid.UUID) ([]string, error) {
	rows, err := r.db.Query("SELECT tag FROM artifact_tags WHERE artifact_id = $1", artifactID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, nil
}

func (r *ArtifactRepository) getCompliance(artifactID uuid.UUID) (*models.ComplianceAudit, error) {
	query := `
        SELECT audit_id, tenant_id, artifact_id, status, score, auditor, license_compliance,
               security_scan, code_quality, data_privacy, audited_at, created_at, updated_at
        FROM compliance_audits
        WHERE artifact_id = $1
        ORDER BY audited_at DESC
        LIMIT 1
    `

	compliance := &models.ComplianceAudit{}
	err := r.db.QueryRow(query, artifactID).Scan(
		&compliance.AuditID,
		&compliance.TenantID,
		&compliance.ArtifactID,
		&compliance.Status,
		&compliance.Score,
		&compliance.Auditor,
		&compliance.LicenseCompliance,
		&compliance.SecurityScan,
		&compliance.CodeQuality,
		&compliance.DataPrivacy,
		&compliance.AuditedAt,
		&compliance.CreatedAt,
		&compliance.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	compliance.LastAudit = compliance.AuditedAt.Format(time.RFC3339)
	compliance.Checks = map[string]string{
		"licenseCompliance": compliance.LicenseCompliance,
		"securityScan":      compliance.SecurityScan,
		"codeQuality":       compliance.CodeQuality,
		"dataPrivacy":       compliance.DataPrivacy,
	}

	return compliance, nil
}

func (r *ArtifactRepository) getVulnerabilities(artifactID uuid.UUID) (*models.Vulnerability, error) {
	// vulnerabilities table doesn't exist in schema, return nil
	return nil, nil
}

func (r *ArtifactRepository) getIndexing(artifactID uuid.UUID) (*models.ArtifactIndexing, error) {
	query := `SELECT indexing_id, tenant_id, artifact_id, index_status, search_content, keywords, indexed_at, created_at, updated_at
              FROM artifact_indexing WHERE artifact_id = $1`

	indexing := &models.ArtifactIndexing{}
	err := r.db.QueryRow(query, artifactID).Scan(
		&indexing.IndexingID,
		&indexing.TenantID,
		&indexing.ArtifactID,
		&indexing.IndexStatus,
		&indexing.SearchContent,
		&indexing.Keywords,
		&indexing.IndexedAt,
		&indexing.CreatedAt,
		&indexing.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return indexing, nil
}

func (r *ArtifactRepository) initializeIndexing(tenantID, artifactID uuid.UUID) error {
	_, err := r.db.Exec(`
        INSERT INTO artifact_indexing (tenant_id, artifact_id, index_status, created_at, updated_at)
        VALUES ($1, $2, 'pending', NOW(), NOW())
    `, tenantID, artifactID)
	return err
}

func (r *ArtifactRepository) IncrementDownloads(id uuid.UUID) error {
	_, err := r.db.Exec("UPDATE artifacts SET downloads = downloads + 1 WHERE artifact_id = $1", id)
	return err
}

func (r *ArtifactRepository) Delete(id uuid.UUID) error {
	_, err := r.db.Exec("DELETE FROM artifacts WHERE artifact_id = $1", id)
	return err
}

// Tenant-aware methods for multi-tenant isolation

func (r *ArtifactRepository) ListByTenant(tenantID uuid.UUID, filter *models.ArtifactFilter) ([]models.Artifact, int, error) {
	queryBuilder := strings.Builder{}
	countBuilder := strings.Builder{}
	args := []interface{}{tenantID}
	argIndex := 2

	// Base query with tenant filter
	queryBuilder.WriteString(`
        SELECT a.artifact_id, a.tenant_id, a.name, a.version, a.type, a.repository_id, r.name as repository,
               a.size, a.checksum, a.uploaded_by, a.uploaded_at, a.downloads,
               a.license, a.metadata, a.created_at, a.updated_at
        FROM artifacts a
        JOIN repositories r ON a.repository_id = r.repository_id
        WHERE a.tenant_id = $1`)

	countBuilder.WriteString(`
        SELECT COUNT(*)
        FROM artifacts a
        WHERE a.tenant_id = $1`)

	// Log the query parameters for debugging
	log.Printf("[ArtifactRepository] ListByTenant called - TenantID: %s, RepositoryID: %v, Search: %s, Limit: %d, Offset: %d",
		tenantID,
		func() string {
			if filter != nil && filter.RepositoryID != nil {
				return filter.RepositoryID.String()
			}
			return "nil"
		}(),
		func() string {
			if filter != nil {
				return filter.Search
			}
			return ""
		}(),
		func() int {
			if filter != nil {
				return filter.Limit
			}
			return 0
		}(),
		func() int {
			if filter != nil {
				return filter.Offset
			}
			return 0
		}(),
	)

	// Add search filter
	if filter != nil && filter.Search != "" {
		queryBuilder.WriteString(fmt.Sprintf(" AND (a.name ILIKE $%d OR a.version ILIKE $%d)", argIndex, argIndex))
		countBuilder.WriteString(fmt.Sprintf(" AND (a.name ILIKE $%d OR a.version ILIKE $%d)", argIndex, argIndex))
		args = append(args, "%"+filter.Search+"%")
		argIndex++
	}

	// Add repository filter - CRITICAL for repository isolation
	if filter != nil && filter.RepositoryID != nil {
		queryBuilder.WriteString(fmt.Sprintf(" AND a.repository_id = $%d", argIndex))
		countBuilder.WriteString(fmt.Sprintf(" AND a.repository_id = $%d", argIndex))
		args = append(args, *filter.RepositoryID)
		argIndex++
	}

	// Add type filter
	if filter != nil && len(filter.Types) > 0 {
		queryBuilder.WriteString(fmt.Sprintf(" AND a.type = ANY($%d)", argIndex))
		countBuilder.WriteString(fmt.Sprintf(" AND a.type = ANY($%d)", argIndex))
		args = append(args, filter.Types)
		argIndex++
	}

	// Get total count
	var total int
	err := r.db.QueryRow(countBuilder.String(), args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count artifacts: %w", err)
	}

	// Add ordering and pagination
	queryBuilder.WriteString(" ORDER BY a.uploaded_at DESC")
	if filter != nil && filter.Limit > 0 {
		queryBuilder.WriteString(fmt.Sprintf(" LIMIT $%d", argIndex))
		args = append(args, filter.Limit)
		argIndex++
	}
	if filter != nil && filter.Offset > 0 {
		queryBuilder.WriteString(fmt.Sprintf(" OFFSET $%d", argIndex))
		args = append(args, filter.Offset)
		argIndex++
	}

	// Execute query
	rows, err := r.db.Query(queryBuilder.String(), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list artifacts: %w", err)
	}
	defer rows.Close()

	var artifacts []models.Artifact
	for rows.Next() {
		var artifact models.Artifact
		var metadataJSON []byte

		err := rows.Scan(
			&artifact.ID,
			&artifact.TenantID,
			&artifact.Name,
			&artifact.Version,
			&artifact.Type,
			&artifact.RepositoryID,
			&artifact.Repository,
			&artifact.Size,
			&artifact.Checksum,
			&artifact.UploadedBy,
			&artifact.UploadedAt,
			&artifact.Downloads,
			&artifact.License,
			&metadataJSON,
			&artifact.CreatedAt,
			&artifact.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan artifact: %w", err)
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &artifact.Metadata); err != nil {
				return nil, 0, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		// Load associated data
		artifact.Tags, _ = r.getTags(artifact.ID)
		artifact.Compliance, _ = r.getCompliance(artifact.ID)
		artifact.Vulnerabilities, _ = r.getVulnerabilities(artifact.ID)

		artifacts = append(artifacts, artifact)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating artifacts: %w", err)
	}

	return artifacts, total, nil
}

func (r *ArtifactRepository) GetByIDAndTenant(id, tenantID uuid.UUID) (*models.Artifact, error) {
	query := `
        SELECT a.artifact_id, a.tenant_id, a.name, a.version, a.type, a.repository_id, r.name as repository,
               a.size, a.checksum, a.uploaded_by, a.uploaded_at, a.downloads,
               a.license, a.metadata, a.created_at, a.updated_at
        FROM artifacts a
        JOIN repositories r ON a.repository_id = r.repository_id
        WHERE a.artifact_id = $1 AND a.tenant_id = $2
    `

	artifact := &models.Artifact{}
	var metadataJSON []byte

	err := r.db.QueryRow(query, id, tenantID).Scan(
		&artifact.ID,
		&artifact.TenantID,
		&artifact.Name,
		&artifact.Version,
		&artifact.Type,
		&artifact.RepositoryID,
		&artifact.Repository,
		&artifact.Size,
		&artifact.Checksum,
		&artifact.UploadedBy,
		&artifact.UploadedAt,
		&artifact.Downloads,
		&artifact.License,
		&metadataJSON,
		&artifact.CreatedAt,
		&artifact.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("artifact not found or access denied")
		}
		return nil, fmt.Errorf("failed to get artifact: %w", err)
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &artifact.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	// Load associated data
	artifact.Tags, _ = r.getTags(id)
	artifact.Compliance, _ = r.getCompliance(id)
	artifact.Vulnerabilities, _ = r.getVulnerabilities(id)

	return artifact, nil
}

func (r *ArtifactRepository) DeleteByTenant(id, tenantID uuid.UUID) error {
	result, err := r.db.Exec("DELETE FROM artifacts WHERE artifact_id = $1 AND tenant_id = $2", id, tenantID)
	if err != nil {
		return fmt.Errorf("failed to delete artifact: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("artifact not found or access denied")
	}

	return nil
}
