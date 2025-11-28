package auth

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/securestor/securestor/internal/models"
	"golang.org/x/crypto/bcrypt"
)

// APIKeyService handles API key management
type APIKeyService struct {
	db     *sql.DB
	logger *log.Logger
}

// NewAPIKeyService creates a new API key service
func NewAPIKeyService(db *sql.DB, logger *log.Logger) *APIKeyService {
	return &APIKeyService{
		db:     db,
		logger: logger,
	}
}

// CreateAPIKeyRequest represents a request to create an API key
type CreateAPIKeyRequest struct {
	Name        string     `json:"name" binding:"required"`
	Description string     `json:"description,omitempty"`
	Scopes      []string   `json:"scopes" binding:"required"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// CreateAPIKey creates a new API key for a user
func (s *APIKeyService) CreateAPIKey(userID uuid.UUID, req *CreateAPIKeyRequest) (*models.APIKey, string, error) {
	return s.CreateAPIKeyDirect(userID, req.Name, req.Description, req.Scopes, req.ExpiresAt)
}

// CreateAPIKeyDirect creates a new API key with individual parameters
func (s *APIKeyService) CreateAPIKeyDirect(userID uuid.UUID, name, description string, scopes []string, expiresAt *time.Time) (*models.APIKey, string, error) {
	// Generate key ID and secret
	keyID, err := s.generateKeyID()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate key ID: %w", err)
	}

	keySecret, err := s.generateKeySecret()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate key secret: %w", err)
	}

	// Create the full API key (keyID.keySecret)
	fullAPIKey := fmt.Sprintf("%s.%s", keyID, keySecret)

	// Hash the full API key for storage
	keyHash, err := bcrypt.GenerateFromPassword([]byte(fullAPIKey), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", fmt.Errorf("failed to hash API key: %w", err)
	}

	// Validate scopes
	validScopes, err := s.validateAPIKeyScopes(scopes)
	if err != nil {
		return nil, "", fmt.Errorf("invalid scopes: %w", err)
	}

	// Create API key record
	apiKey := &models.APIKey{
		KeyID:       keyID,
		KeyHash:     string(keyHash),
		Name:        name,
		Description: &description,
		Scopes:      validScopes,
		UserID:      userID,
		IsActive:    true,
		ExpiresAt:   expiresAt,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Insert into database
	query := `
		INSERT INTO api_keys (key_id, key_hash, name, description, scopes, user_id, is_active, expires_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id`

	err = s.db.QueryRow(query,
		apiKey.KeyID, apiKey.KeyHash, apiKey.Name, apiKey.Description,
		apiKey.Scopes, apiKey.UserID, apiKey.IsActive, apiKey.ExpiresAt,
		apiKey.CreatedAt, apiKey.UpdatedAt,
	).Scan(&apiKey.ID)

	if err != nil {
		return nil, "", fmt.Errorf("failed to create API key: %w", err)
	}

	s.logger.Printf("Created API key: %s for user %d", apiKey.Name, userID)

	return apiKey, fullAPIKey, nil
}

// ValidateAPIKey validates an API key and returns the associated user context
func (s *APIKeyService) ValidateAPIKey(apiKey string) (*models.APIKey, error) {
	// Parse API key (format: keyID.keySecret)
	parts := strings.Split(apiKey, ".")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid API key format")
	}

	keyID := parts[0]

	// Get API key record from database
	key := &models.APIKey{}
	query := `
		SELECT id, key_id, key_hash, name, description, scopes, user_id, 
		       is_active, expires_at, last_used_at, created_at, updated_at
		FROM api_keys 
		WHERE key_id = $1 AND is_active = true`

	err := s.db.QueryRow(query, keyID).Scan(
		&key.ID, &key.KeyID, &key.KeyHash, &key.Name, &key.Description,
		&key.Scopes, &key.UserID, &key.IsActive, &key.ExpiresAt,
		&key.LastUsedAt, &key.CreatedAt, &key.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invalid API key")
		}
		return nil, fmt.Errorf("failed to validate API key: %w", err)
	}

	// Check if API key is expired
	if key.ExpiresAt != nil && key.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("API key expired")
	}

	// Verify API key hash
	err = bcrypt.CompareHashAndPassword([]byte(key.KeyHash), []byte(apiKey))
	if err != nil {
		return nil, fmt.Errorf("invalid API key")
	}

	// Update last used timestamp
	s.updateAPIKeyLastUsed(keyID)

	return key, nil
}

// ListAPIKeys retrieves all active API keys for a user
func (s *APIKeyService) ListAPIKeys(userID uuid.UUID) ([]*models.APIKey, error) {
	query := `
		SELECT id, key_id, name, description, scopes, user_id, 
		       is_active, expires_at, last_used_at, created_at, updated_at
		FROM api_keys 
		WHERE user_id = $1 
		ORDER BY created_at DESC`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}
	defer rows.Close()

	var keys []*models.APIKey
	for rows.Next() {
		key := &models.APIKey{}
		err := rows.Scan(
			&key.ID, &key.KeyID, &key.Name, &key.Description, &key.Scopes,
			&key.UserID, &key.IsActive, &key.ExpiresAt, &key.LastUsedAt,
			&key.CreatedAt, &key.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan API key: %w", err)
		}
		keys = append(keys, key)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating API keys: %w", err)
	}

	return keys, nil
}

// RevokeAPIKey revokes an API key so it cannot be used
func (s *APIKeyService) RevokeAPIKey(keyID string, userID uuid.UUID) error {
	query := `
		UPDATE api_keys 
		SET is_active = false, updated_at = NOW() 
		WHERE key_id = $1 AND user_id = $2`

	result, err := s.db.Exec(query, keyID, userID)
	if err != nil {
		return fmt.Errorf("failed to revoke API key: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("API key not found or not owned by user")
	}

	s.logger.Printf("Revoked API key: %s for user %d", keyID, userID)

	return nil
}

// UpdateAPIKey updates an API key (name, description, scopes)
func (s *APIKeyService) UpdateAPIKey(keyID string, userID uuid.UUID, name, description string, scopes []string) error {
	// Validate scopes
	validScopes, err := s.validateAPIKeyScopes(scopes)
	if err != nil {
		return fmt.Errorf("invalid scopes: %w", err)
	}

	query := `
		UPDATE api_keys 
		SET name = $3, description = $4, scopes = $5, updated_at = NOW() 
		WHERE key_id = $1 AND user_id = $2`

	result, err := s.db.Exec(query, keyID, userID, name, description, validScopes)
	if err != nil {
		return fmt.Errorf("failed to update API key: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("API key not found or not owned by user")
	}

	s.logger.Printf("Updated API key: %s for user %d", keyID, userID)

	return nil
}

// GetAPIKeyStats returns usage statistics for API keys
func (s *APIKeyService) GetAPIKeyStats(userID int64) (map[string]interface{}, error) {
	query := `
		SELECT 
			COUNT(*) as total_keys,
			COUNT(CASE WHEN is_active = true THEN 1 END) as active_keys,
			COUNT(CASE WHEN expires_at IS NOT NULL AND expires_at < NOW() THEN 1 END) as expired_keys,
			COUNT(CASE WHEN last_used_at > NOW() - INTERVAL '24 hours' THEN 1 END) as recently_used
		FROM api_keys 
		WHERE user_id = $1`

	var totalKeys, activeKeys, expiredKeys, recentlyUsed int
	err := s.db.QueryRow(query, userID).Scan(&totalKeys, &activeKeys, &expiredKeys, &recentlyUsed)
	if err != nil {
		return nil, fmt.Errorf("failed to get API key stats: %w", err)
	}

	stats := map[string]interface{}{
		"total_keys":    totalKeys,
		"active_keys":   activeKeys,
		"expired_keys":  expiredKeys,
		"recently_used": recentlyUsed,
	}

	return stats, nil
}

// Helper functions

func (s *APIKeyService) generateKeyID() (string, error) {
	b := make([]byte, 8)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("sk_%s", hex.EncodeToString(b)), nil
}

func (s *APIKeyService) generateKeySecret() (string, error) {
	b := make([]byte, 24)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func (s *APIKeyService) validateAPIKeyScopes(scopes []string) ([]string, error) {
	if len(scopes) == 0 {
		return []string{"read"}, nil
	}

	// Get available scopes from database
	availableScopes, err := s.getAvailableScopes()
	if err != nil {
		return nil, fmt.Errorf("failed to get available scopes: %w", err)
	}

	availableScopeMap := make(map[string]bool)
	for _, scope := range availableScopes {
		availableScopeMap[scope] = true
	}

	var validScopes []string
	for _, scope := range scopes {
		if availableScopeMap[scope] {
			validScopes = append(validScopes, scope)
		} else {
			return nil, fmt.Errorf("invalid scope: %s", scope)
		}
	}

	if len(validScopes) == 0 {
		return []string{"read"}, nil
	}

	return validScopes, nil
}

func (s *APIKeyService) getAvailableScopes() ([]string, error) {
	query := `SELECT name FROM oauth2_scopes ORDER BY name`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get available scopes: %w", err)
	}
	defer rows.Close()

	var scopes []string
	for rows.Next() {
		var scope string
		if err := rows.Scan(&scope); err != nil {
			return nil, fmt.Errorf("failed to scan scope: %w", err)
		}
		scopes = append(scopes, scope)
	}

	return scopes, nil
}

func (s *APIKeyService) updateAPIKeyLastUsed(keyID string) {
	query := `UPDATE api_keys SET last_used_at = NOW() WHERE key_id = $1`
	_, err := s.db.Exec(query, keyID)
	if err != nil {
		s.logger.Printf("Failed to update API key last used timestamp: %v", err)
	}
}
