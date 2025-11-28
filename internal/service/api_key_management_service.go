package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// APIKeyManagementService handles API key operations
type APIKeyManagementService struct {
	db *sql.DB
}

// NewAPIKeyManagementService creates a new API key management service
func NewAPIKeyManagementService(db *sql.DB) *APIKeyManagementService {
	return &APIKeyManagementService{
		db: db,
	}
}

// APIKey represents an API key
type APIKey struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	Description      *string    `json:"description,omitempty"`
	KeyID            string     `json:"key_id"`
	KeyPrefix        string     `json:"key_prefix"`
	UserID           uuid.UUID  `json:"user_id"`
	TenantID         uuid.UUID  `json:"tenant_id"`
	Scopes           []string   `json:"scopes"`
	IsActive         bool       `json:"is_active"`
	LastUsedAt       *time.Time `json:"last_used_at,omitempty"`
	LastUsedIP       *string    `json:"last_used_ip,omitempty"`
	UsageCount       int64      `json:"usage_count"`
	RateLimitPerHour int        `json:"rate_limit_per_hour"`
	RateLimitPerDay  int        `json:"rate_limit_per_day"`
	ExpiresAt        *time.Time `json:"expires_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`

	// Additional fields for responses
	Username     string     `json:"username,omitempty"`
	ScopeDetails []APIScope `json:"scope_details,omitempty"`
}

// APIKeyWithSecret represents an API key with its secret (only returned on creation)
type APIKeyWithSecret struct {
	APIKey
	KeySecret string `json:"key_secret"`
}

// APIScope represents a predefined API scope
type APIScope struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	Description *string   `json:"description,omitempty"`
	Resource    string    `json:"resource"`
	Actions     []string  `json:"actions"`
	IsDefault   bool      `json:"is_default"`
	IsSensitive bool      `json:"is_sensitive"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// APIKeyUsageLog represents usage analytics
type APIKeyUsageLog struct {
	ID                int64     `json:"id"`
	APIKeyID          string    `json:"api_key_id"`
	Endpoint          string    `json:"endpoint"`
	Method            string    `json:"method"`
	StatusCode        int       `json:"status_code"`
	ResponseTimeMs    *int      `json:"response_time_ms,omitempty"`
	RequestSizeBytes  *int      `json:"request_size_bytes,omitempty"`
	ResponseSizeBytes *int      `json:"response_size_bytes,omitempty"`
	IPAddress         *string   `json:"ip_address,omitempty"`
	UserAgent         *string   `json:"user_agent,omitempty"`
	ErrorMessage      *string   `json:"error_message,omitempty"`
	RequestTimestamp  time.Time `json:"request_timestamp"`
}

// APIKeyFilter represents filtering options for API key queries
type APIKeyFilter struct {
	UserID    *uuid.UUID
	TenantID  *uuid.UUID
	Search    string
	IsActive  *bool
	Scopes    []string
	Limit     int
	Offset    int
	SortBy    string
	SortOrder string
}

// UsageAnalytics represents API key usage analytics
type UsageAnalytics struct {
	TotalRequests      int64           `json:"total_requests"`
	SuccessfulRequests int64           `json:"successful_requests"`
	ErrorRequests      int64           `json:"error_requests"`
	AvgResponseTime    float64         `json:"avg_response_time"`
	TotalDataTransfer  int64           `json:"total_data_transfer"`
	TopEndpoints       []EndpointUsage `json:"top_endpoints"`
	HourlyUsage        []HourlyUsage   `json:"hourly_usage"`
}

// EndpointUsage represents usage statistics for an endpoint
type EndpointUsage struct {
	Endpoint        string  `json:"endpoint"`
	Method          string  `json:"method"`
	RequestCount    int64   `json:"request_count"`
	AvgResponseTime float64 `json:"avg_response_time"`
	ErrorRate       float64 `json:"error_rate"`
}

// HourlyUsage represents hourly usage statistics
type HourlyUsage struct {
	Hour         time.Time `json:"hour"`
	RequestCount int64     `json:"request_count"`
	ErrorCount   int64     `json:"error_count"`
}

// CreateAPIKey creates a new API key
func (s *APIKeyManagementService) CreateAPIKey(ctx context.Context, userID, tenantID uuid.UUID, name, description string, scopes []string, expiresAt *time.Time, rateLimitPerHour, rateLimitPerDay int) (*APIKeyWithSecret, error) {
	// DEBUG: Log received parameters
	fmt.Printf("DEBUG: CreateAPIKey Service - userID = %s, tenantID = %s, name = %s, description = %s, scopes = %v\n", userID.String(), tenantID.String(), name, description, scopes)

	// Generate key ID (UUID) and secret
	keyID := uuid.New().String()

	keySecret, err := s.generateKeySecret()
	if err != nil {
		return nil, fmt.Errorf("failed to generate key secret: %w", err)
	}

	// Create key hash - hash the full key string (keyID:keySecret)
	keyHash := s.hashKey(keyID + ":" + keySecret)
	fmt.Printf("DEBUG: CreateAPIKey Service - keyID = %s, keyHash = %s (first 20 chars), keyPrefix = %s\n", keyID, keyHash[:20], keyID[:8])
	keyPrefix := keyID[:8] // First 8 characters for display

	// Begin transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert API key
	query := `
		INSERT INTO api_keys (name, description, key_id, key_hash, key_prefix, user_id, tenant_id, scopes, 
		                     rate_limit_per_hour, rate_limit_per_day, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING created_at, updated_at
	`

	var createdAt, updatedAt time.Time
	err = tx.QueryRowContext(ctx, query,
		name, description, keyID, keyHash, keyPrefix, userID, tenantID, pq.Array(scopes),
		rateLimitPerHour, rateLimitPerDay, expiresAt,
	).Scan(&createdAt, &updatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create API key: %w", err)
	}

	// Populate response
	apiKey := APIKeyWithSecret{}
	apiKey.Name = name
	apiKey.Description = &description
	apiKey.KeyID = keyID
	apiKey.KeySecret = keyID + ":" + keySecret
	apiKey.KeyPrefix = keyPrefix
	apiKey.UserID = userID
	apiKey.TenantID = tenantID
	apiKey.Scopes = scopes
	apiKey.IsActive = true
	apiKey.RateLimitPerHour = rateLimitPerHour
	apiKey.RateLimitPerDay = rateLimitPerDay
	apiKey.ExpiresAt = expiresAt
	apiKey.CreatedAt = createdAt
	apiKey.UpdatedAt = updatedAt

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &apiKey, nil
}

// GetAPIKeys retrieves API keys with filtering
func (s *APIKeyManagementService) GetAPIKeys(ctx context.Context, filter *APIKeyFilter) ([]*APIKey, int, error) {
	// Build WHERE clause
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argCount := 0

	if filter.UserID != nil {
		argCount++
		whereClause += fmt.Sprintf(" AND ak.user_id = $%d", argCount)
		args = append(args, *filter.UserID)
	}

	if filter.TenantID != nil {
		argCount++
		whereClause += fmt.Sprintf(" AND ak.tenant_id = $%d", argCount)
		args = append(args, *filter.TenantID)
	}

	if filter.Search != "" {
		argCount++
		whereClause += fmt.Sprintf(" AND (ak.name ILIKE $%d OR ak.description ILIKE $%d)", argCount, argCount)
		searchTerm := "%" + filter.Search + "%"
		args = append(args, searchTerm)
	}

	if filter.IsActive != nil {
		argCount++
		whereClause += fmt.Sprintf(" AND ak.is_active = $%d", argCount)
		args = append(args, *filter.IsActive)
	}

	if len(filter.Scopes) > 0 {
		argCount++
		whereClause += fmt.Sprintf(" AND ak.scopes && $%d", argCount)
		args = append(args, pq.Array(filter.Scopes))
	}

	// Get total count
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM api_keys ak 
		LEFT JOIN users u ON ak.user_id = u.user_id 
		%s
	`, whereClause)

	var totalCount int
	err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get API key count: %w", err)
	}

	// Build ORDER BY clause
	orderBy := "ORDER BY ak.created_at DESC"
	if filter.SortBy != "" {
		sortOrder := "ASC"
		if strings.ToUpper(filter.SortOrder) == "DESC" {
			sortOrder = "DESC"
		}
		orderBy = fmt.Sprintf("ORDER BY ak.%s %s", filter.SortBy, sortOrder)
	}

	// Build LIMIT and OFFSET
	limit := 50
	if filter.Limit > 0 && filter.Limit <= 100 {
		limit = filter.Limit
	}
	offset := 0
	if filter.Offset >= 0 {
		offset = filter.Offset
	}

	argCount++
	limitClause := fmt.Sprintf("LIMIT $%d", argCount)
	args = append(args, limit)

	argCount++
	offsetClause := fmt.Sprintf("OFFSET $%d", argCount)
	args = append(args, offset)

	// Main query
	query := fmt.Sprintf(`
		SELECT ak.key_id, ak.name, ak.description, ak.key_id, ak.key_prefix, ak.user_id, ak.tenant_id,
		       ak.scopes, ak.is_active, ak.last_used_at, ak.last_used_ip, ak.usage_count,
		       ak.rate_limit_per_hour, ak.rate_limit_per_day, ak.expires_at, ak.created_at, ak.updated_at,
		       u.username
		FROM api_keys ak
		LEFT JOIN users u ON ak.user_id = u.user_id
		%s %s %s %s
	`, whereClause, orderBy, limitClause, offsetClause)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query API keys: %w", err)
	}
	defer rows.Close()

	var apiKeys []*APIKey
	for rows.Next() {
		apiKey := &APIKey{}
		err := rows.Scan(
			&apiKey.ID, &apiKey.Name, &apiKey.Description, &apiKey.KeyID, &apiKey.KeyPrefix,
			&apiKey.UserID, &apiKey.TenantID, pq.Array(&apiKey.Scopes), &apiKey.IsActive,
			&apiKey.LastUsedAt, &apiKey.LastUsedIP, &apiKey.UsageCount,
			&apiKey.RateLimitPerHour, &apiKey.RateLimitPerDay, &apiKey.ExpiresAt,
			&apiKey.CreatedAt, &apiKey.UpdatedAt, &apiKey.Username,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan API key: %w", err)
		}

		apiKeys = append(apiKeys, apiKey)
	}

	return apiKeys, totalCount, nil
}

// GetAPIKeyByID retrieves an API key by ID
func (s *APIKeyManagementService) GetAPIKeyByID(ctx context.Context, keyID string) (*APIKey, error) {
	query := `
		SELECT ak.key_id, ak.name, ak.description, ak.key_id, ak.key_prefix, ak.user_id, ak.tenant_id,
		       ak.scopes, ak.is_active, ak.last_used_at, ak.last_used_ip, ak.usage_count,
		       ak.rate_limit_per_hour, ak.rate_limit_per_day, ak.expires_at, ak.created_at, ak.updated_at,
		       u.username
		FROM api_keys ak
		LEFT JOIN users u ON ak.user_id = u.user_id
		WHERE ak.key_id = $1
	`

	apiKey := &APIKey{}
	err := s.db.QueryRowContext(ctx, query, keyID).Scan(
		&apiKey.ID, &apiKey.Name, &apiKey.Description, &apiKey.KeyID, &apiKey.KeyPrefix,
		&apiKey.UserID, &apiKey.TenantID, pq.Array(&apiKey.Scopes), &apiKey.IsActive,
		&apiKey.LastUsedAt, &apiKey.LastUsedIP, &apiKey.UsageCount,
		&apiKey.RateLimitPerHour, &apiKey.RateLimitPerDay, &apiKey.ExpiresAt,
		&apiKey.CreatedAt, &apiKey.UpdatedAt, &apiKey.Username,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("API key not found")
		}
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	return apiKey, nil
}

// ValidateAPIKey validates an API key and returns the key info if valid
func (s *APIKeyManagementService) ValidateAPIKey(ctx context.Context, keyString string) (*APIKey, error) {
	// Parse key string (format: keyID:keySecret)
	parts := strings.Split(keyString, ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid key format")
	}

	keyID := parts[0]
	keyHash := s.hashKey(keyString)

	// Query for the API key
	query := `
		SELECT ak.key_id, ak.name, ak.description, ak.key_id, ak.key_prefix, ak.user_id, ak.tenant_id,
		       ak.scopes, ak.is_active, ak.last_used_at, ak.last_used_ip, ak.usage_count,
		       ak.rate_limit_per_hour, ak.rate_limit_per_day, ak.expires_at, ak.created_at, ak.updated_at,
		       u.username
		FROM api_keys ak
		LEFT JOIN users u ON ak.user_id = u.user_id
		WHERE ak.key_id = $1 AND ak.key_hash = $2 AND ak.is_active = true
		  AND (ak.expires_at IS NULL OR ak.expires_at > NOW())
	`

	apiKey := &APIKey{}
	err := s.db.QueryRowContext(ctx, query, keyID, keyHash).Scan(
		&apiKey.ID, &apiKey.Name, &apiKey.Description, &apiKey.KeyID, &apiKey.KeyPrefix,
		&apiKey.UserID, &apiKey.TenantID, pq.Array(&apiKey.Scopes), &apiKey.IsActive,
		&apiKey.LastUsedAt, &apiKey.LastUsedIP, &apiKey.UsageCount,
		&apiKey.RateLimitPerHour, &apiKey.RateLimitPerDay, &apiKey.ExpiresAt,
		&apiKey.CreatedAt, &apiKey.UpdatedAt, &apiKey.Username,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invalid or expired API key")
		}
		return nil, fmt.Errorf("failed to validate API key: %w", err)
	}

	return apiKey, nil
}

// UpdateAPIKeyUsage updates the last usage information for an API key
func (s *APIKeyManagementService) UpdateAPIKeyUsage(ctx context.Context, keyID string, ipAddress string) error {
	query := `
		UPDATE api_keys 
		SET last_used_at = NOW(), last_used_ip = $2, usage_count = usage_count + 1, updated_at = NOW()
		WHERE key_id = $1
	`

	_, err := s.db.ExecContext(ctx, query, keyID, ipAddress)
	if err != nil {
		return fmt.Errorf("failed to update API key usage: %w", err)
	}

	return nil
}

// LogAPIKeyUsage logs API key usage for analytics
func (s *APIKeyManagementService) LogAPIKeyUsage(ctx context.Context, log *APIKeyUsageLog) error {
	query := `
		INSERT INTO api_key_usage_logs (api_key_id, endpoint, method, status_code, response_time_ms,
		                               request_size_bytes, response_size_bytes, ip_address, user_agent, error_message)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := s.db.ExecContext(ctx, query,
		log.APIKeyID, log.Endpoint, log.Method, log.StatusCode, log.ResponseTimeMs,
		log.RequestSizeBytes, log.ResponseSizeBytes, log.IPAddress, log.UserAgent, log.ErrorMessage,
	)

	if err != nil {
		return fmt.Errorf("failed to log API key usage: %w", err)
	}

	return nil
}

// CheckRateLimit checks if an API key is within its rate limits
func (s *APIKeyManagementService) CheckRateLimit(ctx context.Context, apiKey *APIKey) (bool, error) {
	// Check hourly limit
	hourlyOK, err := s.checkRateLimitWindow(ctx, apiKey.KeyID, "hour", apiKey.RateLimitPerHour)
	if err != nil {
		return false, err
	}

	if !hourlyOK {
		return false, nil
	}

	// Check daily limit
	dailyOK, err := s.checkRateLimitWindow(ctx, apiKey.KeyID, "day", apiKey.RateLimitPerDay)
	if err != nil {
		return false, err
	}

	return dailyOK, nil
}

// RevokeAPIKey revokes an API key
func (s *APIKeyManagementService) RevokeAPIKey(ctx context.Context, keyID string, reason string) error {
	// Verify the key exists before revoking
	query := `SELECT key_id FROM api_keys WHERE key_id = $1`
	var existingKeyID string
	err := s.db.QueryRowContext(ctx, query, keyID).Scan(&existingKeyID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("API key not found")
		}
		return fmt.Errorf("failed to verify API key: %w", err)
	}

	// Deactivate the API key
	updateQuery := `UPDATE api_keys SET is_active = false, updated_at = NOW() WHERE key_id = $1`
	result, err := s.db.ExecContext(ctx, updateQuery, keyID)
	if err != nil {
		return fmt.Errorf("failed to revoke API key: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("API key not found")
	}

	return nil
}

// GetAPIScopes retrieves available API scopes
func (s *APIKeyManagementService) GetAPIScopes(ctx context.Context) ([]APIScope, error) {
	query := `
		SELECT scope_id, name, name as display_name, description, resource, actions, is_default, false as is_sensitive, created_at, created_at as updated_at
		FROM oauth2_scopes
		ORDER BY resource, name
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query API scopes: %w", err)
	}
	defer rows.Close()

	var scopes []APIScope
	for rows.Next() {
		var scope APIScope
		err := rows.Scan(
			&scope.ID, &scope.Name, &scope.DisplayName, &scope.Description,
			&scope.Resource, pq.Array(&scope.Actions), &scope.IsDefault, &scope.IsSensitive,
			&scope.CreatedAt, &scope.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan API scope: %w", err)
		}
		scopes = append(scopes, scope)
	}

	return scopes, nil
}

// GetUsageAnalytics retrieves usage analytics for an API key
func (s *APIKeyManagementService) GetUsageAnalytics(ctx context.Context, keyID string, days int) (*UsageAnalytics, error) {
	if days <= 0 {
		days = 30
	}

	// Get overall statistics
	var analytics UsageAnalytics
	query := `
		SELECT 
			COUNT(*) as total_requests,
			COUNT(CASE WHEN status_code >= 200 AND status_code < 300 THEN 1 END) as successful_requests,
			COUNT(CASE WHEN status_code >= 400 THEN 1 END) as error_requests,
			COALESCE(AVG(response_time_ms), 0) as avg_response_time,
			COALESCE(SUM(request_size_bytes + response_size_bytes), 0) as total_data_transfer
		FROM api_key_usage_logs
		WHERE api_key_id = $1 AND request_timestamp >= NOW() - INTERVAL '%d days'
	`

	err := s.db.QueryRowContext(ctx, fmt.Sprintf(query, days), keyID).Scan(
		&analytics.TotalRequests, &analytics.SuccessfulRequests, &analytics.ErrorRequests,
		&analytics.AvgResponseTime, &analytics.TotalDataTransfer,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage analytics: %w", err)
	}

	// Get top endpoints
	endpointQuery := `
		SELECT endpoint, method, COUNT(*) as request_count,
		       COALESCE(AVG(response_time_ms), 0) as avg_response_time,
		       (COUNT(CASE WHEN status_code >= 400 THEN 1 END)::float / COUNT(*)::float * 100) as error_rate
		FROM api_key_usage_logs
		WHERE api_key_id = $1 AND request_timestamp >= NOW() - INTERVAL '%d days'
		GROUP BY endpoint, method
		ORDER BY request_count DESC
		LIMIT 10
	`

	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(endpointQuery, days), keyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get endpoint usage: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var endpoint EndpointUsage
		err := rows.Scan(&endpoint.Endpoint, &endpoint.Method, &endpoint.RequestCount, &endpoint.AvgResponseTime, &endpoint.ErrorRate)
		if err != nil {
			return nil, fmt.Errorf("failed to scan endpoint usage: %w", err)
		}
		analytics.TopEndpoints = append(analytics.TopEndpoints, endpoint)
	}

	// Get hourly usage
	hourlyQuery := `
		SELECT DATE_TRUNC('hour', request_timestamp) as hour,
		       COUNT(*) as request_count,
		       COUNT(CASE WHEN status_code >= 400 THEN 1 END) as error_count
		FROM api_key_usage_logs
		WHERE api_key_id = $1 AND request_timestamp >= NOW() - INTERVAL '%d days'
		GROUP BY DATE_TRUNC('hour', request_timestamp)
		ORDER BY hour
	`

	rows, err = s.db.QueryContext(ctx, fmt.Sprintf(hourlyQuery, days), keyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get hourly usage: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var hourly HourlyUsage
		err := rows.Scan(&hourly.Hour, &hourly.RequestCount, &hourly.ErrorCount)
		if err != nil {
			return nil, fmt.Errorf("failed to scan hourly usage: %w", err)
		}
		analytics.HourlyUsage = append(analytics.HourlyUsage, hourly)
	}

	return &analytics, nil
}

// Helper methods

func (s *APIKeyManagementService) generateKeyID() (string, error) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (s *APIKeyManagementService) generateKeySecret() (string, error) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (s *APIKeyManagementService) hashKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

func (s *APIKeyManagementService) checkRateLimitWindow(ctx context.Context, keyID string, windowType string, limit int) (bool, error) {
	query := `SELECT check_rate_limit($1, $2, $3)`

	var allowed bool
	err := s.db.QueryRowContext(ctx, query, keyID, windowType, limit).Scan(&allowed)
	if err != nil {
		return false, fmt.Errorf("failed to check rate limit: %w", err)
	}

	return allowed, nil
}
