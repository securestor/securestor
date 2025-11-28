package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/securestor/securestor/internal/models"
)

// PropertyRepository handles database operations for artifact properties
type PropertyRepository struct {
	db *sql.DB
}

// NewPropertyRepository creates a new property repository
func NewPropertyRepository(db *sql.DB) *PropertyRepository {
	return &PropertyRepository{db: db}
}

// Create creates a new property
func (r *PropertyRepository) Create(ctx context.Context, property *models.ArtifactProperty) error {
	query := `
		INSERT INTO artifact_properties (
			id, tenant_id, repository_id, artifact_id, key, value, value_type,
			is_sensitive, is_system, is_multi_value,
			encrypted_value, encryption_key_id, encryption_algorithm, nonce,
			created_by, created_at, updated_by, updated_at, version, tags, description
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14,
			$15, $16, $17, $18, $19, $20, $21
		)
		RETURNING id, created_at, updated_at
	`

	if property.ID == uuid.Nil {
		property.ID = uuid.New()
	}
	if property.CreatedAt.IsZero() {
		property.CreatedAt = time.Now()
	}
	if property.UpdatedAt.IsZero() {
		property.UpdatedAt = time.Now()
	}
	if property.Version == 0 {
		property.Version = 1
	}

	err := r.db.QueryRowContext(ctx, query,
		property.ID, property.TenantID, property.RepositoryID, property.ArtifactID,
		property.Key, property.Value, property.ValueType,
		property.IsSensitive, property.IsSystem, property.IsMultiValue,
		property.EncryptedValue, property.EncryptionKeyID, property.EncryptionAlgorithm, property.Nonce,
		property.CreatedBy, property.CreatedAt, property.UpdatedBy, property.UpdatedAt,
		property.Version, property.Tags, property.Description,
	).Scan(&property.ID, &property.CreatedAt, &property.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create property: %w", err)
	}

	return nil
}

// Get retrieves a property by ID
func (r *PropertyRepository) Get(ctx context.Context, tenantID, propertyID uuid.UUID) (*models.ArtifactProperty, error) {
	query := `
		SELECT id, tenant_id, repository_id, artifact_id, key, value, value_type,
			   is_sensitive, is_system, is_multi_value,
			   encrypted_value, encryption_key_id, encryption_algorithm, nonce,
			   created_by, created_at, updated_by, updated_at, version, tags, description
		FROM artifact_properties
		WHERE id = $1 AND tenant_id = $2
	`

	property := &models.ArtifactProperty{}
	err := r.db.QueryRowContext(ctx, query, propertyID, tenantID).Scan(
		&property.ID, &property.TenantID, &property.RepositoryID, &property.ArtifactID,
		&property.Key, &property.Value, &property.ValueType,
		&property.IsSensitive, &property.IsSystem, &property.IsMultiValue,
		&property.EncryptedValue, &property.EncryptionKeyID, &property.EncryptionAlgorithm, &property.Nonce,
		&property.CreatedBy, &property.CreatedAt, &property.UpdatedBy, &property.UpdatedAt,
		&property.Version, &property.Tags, &property.Description,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("property not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get property: %w", err)
	}

	return property, nil
}

// GetByKey retrieves a property by artifact ID and key
func (r *PropertyRepository) GetByKey(ctx context.Context, tenantID uuid.UUID, artifactID, key string) (*models.ArtifactProperty, error) {
	query := `
		SELECT id, tenant_id, repository_id, artifact_id, key, value, value_type,
			   is_sensitive, is_system, is_multi_value,
			   encrypted_value, encryption_key_id, encryption_algorithm, nonce,
			   created_by, created_at, updated_by, updated_at, version, tags, description
		FROM artifact_properties
		WHERE tenant_id = $1 AND artifact_id = $2 AND key = $3
		LIMIT 1
	`

	property := &models.ArtifactProperty{}
	err := r.db.QueryRowContext(ctx, query, tenantID, artifactID, key).Scan(
		&property.ID, &property.TenantID, &property.RepositoryID, &property.ArtifactID,
		&property.Key, &property.Value, &property.ValueType,
		&property.IsSensitive, &property.IsSystem, &property.IsMultiValue,
		&property.EncryptedValue, &property.EncryptionKeyID, &property.EncryptionAlgorithm, &property.Nonce,
		&property.CreatedBy, &property.CreatedAt, &property.UpdatedBy, &property.UpdatedAt,
		&property.Version, &property.Tags, &property.Description,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("property not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get property: %w", err)
	}

	return property, nil
}

// GetByArtifact retrieves all properties for an artifact
func (r *PropertyRepository) GetByArtifact(ctx context.Context, tenantID uuid.UUID, artifactID string, limit, offset int) ([]*models.ArtifactProperty, int, error) {
	if limit <= 0 {
		limit = 100
	}

	// Get total count
	var total int
	countQuery := `SELECT COUNT(*) FROM artifact_properties WHERE tenant_id = $1 AND artifact_id = $2`
	err := r.db.QueryRowContext(ctx, countQuery, tenantID, artifactID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count properties: %w", err)
	}

	// Get properties
	query := `
		SELECT id, tenant_id, repository_id, artifact_id, key, value, value_type,
			   is_sensitive, is_system, is_multi_value,
			   encrypted_value, encryption_key_id, encryption_algorithm, nonce,
			   created_by, created_at, updated_by, updated_at, version, tags, description
		FROM artifact_properties
		WHERE tenant_id = $1 AND artifact_id = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID, artifactID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query properties: %w", err)
	}
	defer rows.Close()

	var properties []*models.ArtifactProperty
	for rows.Next() {
		property := &models.ArtifactProperty{}
		err := rows.Scan(
			&property.ID, &property.TenantID, &property.RepositoryID, &property.ArtifactID,
			&property.Key, &property.Value, &property.ValueType,
			&property.IsSensitive, &property.IsSystem, &property.IsMultiValue,
			&property.EncryptedValue, &property.EncryptionKeyID, &property.EncryptionAlgorithm, &property.Nonce,
			&property.CreatedBy, &property.CreatedAt, &property.UpdatedBy, &property.UpdatedAt,
			&property.Version, &property.Tags, &property.Description,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan property: %w", err)
		}
		properties = append(properties, property)
	}

	return properties, total, nil
}

// Update updates a property
func (r *PropertyRepository) Update(ctx context.Context, property *models.ArtifactProperty) error {
	// Optimistic locking: increment version
	query := `
		UPDATE artifact_properties
		SET value = $1, value_type = $2, is_sensitive = $3, is_multi_value = $4,
			encrypted_value = $5, encryption_key_id = $6, encryption_algorithm = $7, nonce = $8,
			updated_by = $9, updated_at = $10, version = version + 1,
			tags = $11, description = $12
		WHERE id = $13 AND tenant_id = $14 AND version = $15
		RETURNING version, updated_at
	`

	property.UpdatedAt = time.Now()

	err := r.db.QueryRowContext(ctx, query,
		property.Value, property.ValueType, property.IsSensitive, property.IsMultiValue,
		property.EncryptedValue, property.EncryptionKeyID, property.EncryptionAlgorithm, property.Nonce,
		property.UpdatedBy, property.UpdatedAt,
		property.Tags, property.Description,
		property.ID, property.TenantID, property.Version,
	).Scan(&property.Version, &property.UpdatedAt)

	if err == sql.ErrNoRows {
		return fmt.Errorf("property not found or version conflict (concurrent update)")
	}
	if err != nil {
		return fmt.Errorf("failed to update property: %w", err)
	}

	return nil
}

// Delete deletes a property
func (r *PropertyRepository) Delete(ctx context.Context, tenantID, propertyID uuid.UUID) error {
	query := `DELETE FROM artifact_properties WHERE id = $1 AND tenant_id = $2`

	result, err := r.db.ExecContext(ctx, query, propertyID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to delete property: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("property not found")
	}

	return nil
}

// DeleteByKey deletes a property by artifact ID and key
func (r *PropertyRepository) DeleteByKey(ctx context.Context, tenantID uuid.UUID, artifactID, key string) error {
	query := `DELETE FROM artifact_properties WHERE tenant_id = $1 AND artifact_id = $2 AND key = $3`

	result, err := r.db.ExecContext(ctx, query, tenantID, artifactID, key)
	if err != nil {
		return fmt.Errorf("failed to delete property: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("property not found")
	}

	return nil
}

// Search searches for properties based on criteria
func (r *PropertyRepository) Search(ctx context.Context, tenantID uuid.UUID, req *models.SearchPropertiesRequest) ([]*models.PropertySearchResult, int, error) {
	if req.Limit <= 0 {
		req.Limit = models.DefaultLimit
	}
	if req.Limit > models.MaxLimit {
		req.Limit = models.MaxLimit
	}

	// Build query dynamically based on search criteria
	var conditions []string
	var args []interface{}
	argIndex := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argIndex))
	args = append(args, tenantID)
	argIndex++

	if req.Key != nil {
		conditions = append(conditions, fmt.Sprintf("key = $%d", argIndex))
		args = append(args, *req.Key)
		argIndex++
	}

	if req.Value != nil {
		conditions = append(conditions, fmt.Sprintf("value = $%d", argIndex))
		args = append(args, *req.Value)
		argIndex++
	}

	if req.KeyPattern != nil {
		conditions = append(conditions, fmt.Sprintf("key LIKE $%d", argIndex))
		args = append(args, *req.KeyPattern)
		argIndex++
	}

	if req.ValuePattern != nil {
		conditions = append(conditions, fmt.Sprintf("value LIKE $%d", argIndex))
		args = append(args, *req.ValuePattern)
		argIndex++
	}

	if len(req.Keys) > 0 {
		placeholders := make([]string, len(req.Keys))
		for i, key := range req.Keys {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, key)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("key IN (%s)", strings.Join(placeholders, ",")))
	}

	if req.ValueType != nil {
		conditions = append(conditions, fmt.Sprintf("value_type = $%d", argIndex))
		args = append(args, *req.ValueType)
		argIndex++
	}

	if req.IsSensitive != nil {
		conditions = append(conditions, fmt.Sprintf("is_sensitive = $%d", argIndex))
		args = append(args, *req.IsSensitive)
		argIndex++
	}

	if req.IsSystem != nil {
		conditions = append(conditions, fmt.Sprintf("is_system = $%d", argIndex))
		args = append(args, *req.IsSystem)
		argIndex++
	}

	if req.IsMultiValue != nil {
		conditions = append(conditions, fmt.Sprintf("is_multi_value = $%d", argIndex))
		args = append(args, *req.IsMultiValue)
		argIndex++
	}

	if req.RepositoryID != nil {
		conditions = append(conditions, fmt.Sprintf("repository_id = $%d", argIndex))
		args = append(args, *req.RepositoryID)
		argIndex++
	}

	if req.ArtifactID != nil {
		conditions = append(conditions, fmt.Sprintf("artifact_id = $%d", argIndex))
		args = append(args, *req.ArtifactID)
		argIndex++
	}

	if req.CreatedAfter != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIndex))
		args = append(args, *req.CreatedAfter)
		argIndex++
	}

	if req.CreatedBefore != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIndex))
		args = append(args, *req.CreatedBefore)
		argIndex++
	}

	// Full-text search
	if req.FullText != nil {
		conditions = append(conditions, fmt.Sprintf("to_tsvector('english', key || ' ' || value) @@ plainto_tsquery('english', $%d)", argIndex))
		args = append(args, *req.FullText)
		argIndex++
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count total
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM artifact_properties WHERE %s", whereClause)
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count search results: %w", err)
	}

	// Query results
	query := fmt.Sprintf(`
		SELECT id, tenant_id, repository_id, artifact_id, key, value, value_type,
			   is_sensitive, is_system, created_at, updated_at
		FROM artifact_properties
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	args = append(args, req.Limit, req.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search properties: %w", err)
	}
	defer rows.Close()

	var results []*models.PropertySearchResult
	for rows.Next() {
		result := &models.PropertySearchResult{}
		err := rows.Scan(
			&result.ID, &result.TenantID, &result.RepositoryID, &result.ArtifactID,
			&result.Key, &result.Value, &result.ValueType,
			&result.IsSensitive, &result.IsSystem, &result.CreatedAt, &result.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan search result: %w", err)
		}
		results = append(results, result)
	}

	return results, total, nil
}

// BatchCreate creates multiple properties in a transaction
func (r *PropertyRepository) BatchCreate(ctx context.Context, properties []*models.ArtifactProperty) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO artifact_properties (
			id, tenant_id, repository_id, artifact_id, key, value, value_type,
			is_sensitive, is_system, is_multi_value,
			encrypted_value, encryption_key_id, encryption_algorithm, nonce,
			created_by, created_at, updated_by, updated_at, version, tags, description
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14,
			$15, $16, $17, $18, $19, $20, $21
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, property := range properties {
		if property.ID == uuid.Nil {
			property.ID = uuid.New()
		}
		if property.CreatedAt.IsZero() {
			property.CreatedAt = time.Now()
		}
		if property.UpdatedAt.IsZero() {
			property.UpdatedAt = time.Now()
		}
		if property.Version == 0 {
			property.Version = 1
		}

		_, err := stmt.ExecContext(ctx,
			property.ID, property.TenantID, property.RepositoryID, property.ArtifactID,
			property.Key, property.Value, property.ValueType,
			property.IsSensitive, property.IsSystem, property.IsMultiValue,
			property.EncryptedValue, property.EncryptionKeyID, property.EncryptionAlgorithm, property.Nonce,
			property.CreatedBy, property.CreatedAt, property.UpdatedBy, property.UpdatedAt,
			property.Version, property.Tags, property.Description,
		)
		if err != nil {
			return fmt.Errorf("failed to insert property %s: %w", property.Key, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// BatchDelete deletes multiple properties in a transaction
func (r *PropertyRepository) BatchDelete(ctx context.Context, tenantID uuid.UUID, propertyIDs []uuid.UUID) error {
	if len(propertyIDs) == 0 {
		return nil
	}

	placeholders := make([]string, len(propertyIDs))
	args := []interface{}{tenantID}
	for i, id := range propertyIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args = append(args, id)
	}

	query := fmt.Sprintf(`
		DELETE FROM artifact_properties
		WHERE tenant_id = $1 AND id IN (%s)
	`, strings.Join(placeholders, ","))

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to batch delete properties: %w", err)
	}

	return nil
}

// GetStatistics returns property statistics for a tenant
func (r *PropertyRepository) GetStatistics(ctx context.Context, tenantID uuid.UUID) (*models.PropertyStatistics, error) {
	query := `
		SELECT 
			tenant_id,
			total_properties,
			artifacts_with_properties,
			unique_keys,
			sensitive_properties,
			system_properties,
			multi_value_properties,
			last_property_added
		FROM v_property_statistics
		WHERE tenant_id = $1
	`

	stats := &models.PropertyStatistics{}
	err := r.db.QueryRowContext(ctx, query, tenantID).Scan(
		&stats.TenantID,
		&stats.TotalProperties,
		&stats.ArtifactsWithProperties,
		&stats.UniqueKeys,
		&stats.SensitiveProperties,
		&stats.SystemProperties,
		&stats.MultiValueProperties,
		&stats.LastPropertyAdded,
	)

	if err == sql.ErrNoRows {
		// Return empty stats if no properties exist
		stats.TenantID = tenantID
		return stats, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get statistics: %w", err)
	}

	return stats, nil
}

// RefreshSearchIndex refreshes the materialized view for property search
func (r *PropertyRepository) RefreshSearchIndex(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, "SELECT refresh_property_search_index()")
	if err != nil {
		return fmt.Errorf("failed to refresh search index: %w", err)
	}
	return nil
}
