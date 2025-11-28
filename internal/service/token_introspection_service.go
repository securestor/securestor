package service

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/securestor/securestor/internal/models"
)

// TokenIntrospectionService handles OAuth2 token introspection
type TokenIntrospectionService struct {
	db *sql.DB
}

// NewTokenIntrospectionService creates a new token introspection service
func NewTokenIntrospectionService(db *sql.DB) *TokenIntrospectionService {
	return &TokenIntrospectionService{
		db: db,
	}
}

// AccessToken represents an OAuth2 access token (simplified model)
type AccessToken struct {
	ID        int64     `json:"id"`
	UserID    string    `json:"user_id"`
	ClientID  string    `json:"client_id"`
	Scope     string    `json:"scope"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// IntrospectionResponse represents the OAuth2 token introspection response
type IntrospectionResponse struct {
	Active     bool                   `json:"active"`
	TokenType  string                 `json:"token_type,omitempty"`
	ClientID   string                 `json:"client_id,omitempty"`
	Username   string                 `json:"username,omitempty"`
	Scope      string                 `json:"scope,omitempty"`
	ExpiresAt  *int64                 `json:"exp,omitempty"`
	IssuedAt   *int64                 `json:"iat,omitempty"`
	NotBefore  *int64                 `json:"nbf,omitempty"`
	Subject    string                 `json:"sub,omitempty"`
	Audience   string                 `json:"aud,omitempty"`
	Issuer     string                 `json:"iss,omitempty"`
	JTI        string                 `json:"jti,omitempty"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

// IntrospectToken performs token introspection with caching
func (s *TokenIntrospectionService) IntrospectToken(ctx context.Context, token string) (*IntrospectionResponse, error) {
	tokenHash := s.hashToken(token)

	// First, check cache
	cached, err := s.getCachedIntrospection(ctx, tokenHash)
	if err == nil && cached != nil {
		// Update last accessed time
		s.updateLastAccessed(ctx, tokenHash)
		return s.buildIntrospectionResponse(cached), nil
	}

	// If not cached or cache miss, perform actual introspection
	response, err := s.performIntrospection(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to introspect token: %w", err)
	}

	// Cache the result
	if response.Active {
		err = s.cacheIntrospection(ctx, tokenHash, response)
		if err != nil {
			// Log error but don't fail the request
			// s.logger.Warn("Failed to cache introspection result", "error", err)
		}
	}

	return response, nil
}

// performIntrospection performs actual token validation
func (s *TokenIntrospectionService) performIntrospection(ctx context.Context, token string) (*IntrospectionResponse, error) {
	// Check if it's a valid API key
	apiKey, err := s.validateAPIKey(ctx, token)
	if err == nil && apiKey != nil {
		issuedAt := apiKey.CreatedAt.Unix()
		return &IntrospectionResponse{
			Active:    true,
			TokenType: "api_key",
			ClientID:  fmt.Sprintf("client_%d", apiKey.UserID),
			Username:  apiKey.Name, // Use API key name as username
			Subject:   fmt.Sprintf("user_%d", apiKey.UserID),
			IssuedAt:  &issuedAt,
			ExpiresAt: func() *int64 {
				if apiKey.ExpiresAt != nil {
					exp := apiKey.ExpiresAt.Unix()
					return &exp
				}
				return nil
			}(),
			Extensions: map[string]interface{}{
				"api_key_id": apiKey.ID,
				"scopes":     apiKey.Scopes,
			},
		}, nil
	}

	// Check if it's a valid OAuth2 access token
	accessToken, err := s.validateAccessToken(ctx, token)
	if err == nil && accessToken != nil {
		issuedAt := accessToken.CreatedAt.Unix()
		expiresAt := accessToken.ExpiresAt.Unix()
		return &IntrospectionResponse{
			Active:    true,
			TokenType: "Bearer",
			ClientID:  accessToken.ClientID,
			Username:  accessToken.UserID,
			Subject:   accessToken.UserID,
			Scope:     accessToken.Scope,
			IssuedAt:  &issuedAt,
			ExpiresAt: &expiresAt,
			Extensions: map[string]interface{}{
				"token_id": accessToken.ID,
			},
		}, nil
	}

	// Token is not valid
	return &IntrospectionResponse{
		Active: false,
	}, nil
}

// validateAPIKey validates an API key token
func (s *TokenIntrospectionService) validateAPIKey(ctx context.Context, token string) (*models.APIKey, error) {
	query := `
		SELECT id, user_id, name, scopes, created_at, expires_at, last_used_at
		FROM api_keys 
		WHERE key_hash = $1 AND is_active = true AND (expires_at IS NULL OR expires_at > NOW())
	`

	tokenHash := s.hashToken(token)
	apiKey := &models.APIKey{}

	err := s.db.QueryRowContext(ctx, query, tokenHash).Scan(
		&apiKey.ID, &apiKey.UserID, &apiKey.Name, &apiKey.Scopes,
		&apiKey.CreatedAt, &apiKey.ExpiresAt, &apiKey.LastUsedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invalid API key")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Update last used timestamp
	s.updateAPIKeyLastUsed(ctx, apiKey.ID)

	return apiKey, nil
}

// validateAccessToken validates an OAuth2 access token
func (s *TokenIntrospectionService) validateAccessToken(ctx context.Context, token string) (*AccessToken, error) {
	// This is a simplified implementation - in a real system you'd parse JWT tokens
	// or lookup tokens in your OAuth2 token store
	query := `
		SELECT id, user_id, client_id, scope, created_at, expires_at
		FROM oauth_access_tokens 
		WHERE token_hash = $1 AND expires_at > NOW()
	`

	tokenHash := s.hashToken(token)
	accessToken := &AccessToken{}

	err := s.db.QueryRowContext(ctx, query, tokenHash).Scan(
		&accessToken.ID, &accessToken.UserID, &accessToken.ClientID,
		&accessToken.Scope, &accessToken.CreatedAt, &accessToken.ExpiresAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invalid access token")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	return accessToken, nil
}

// getCachedIntrospection retrieves cached introspection data
func (s *TokenIntrospectionService) getCachedIntrospection(ctx context.Context, tokenHash string) (*models.TokenIntrospectionCache, error) {
	query := `
		SELECT id, token_hash, is_active, token_type, client_id, username, 
		       scope, expires_at, issued_at, cached_at, last_accessed
		FROM token_introspection_cache 
		WHERE token_hash = $1 AND (expires_at IS NULL OR expires_at > NOW())
	`

	cache := &models.TokenIntrospectionCache{}
	err := s.db.QueryRowContext(ctx, query, tokenHash).Scan(
		&cache.ID, &cache.TokenHash, &cache.IsActive, &cache.TokenType,
		&cache.ClientID, &cache.Username, &cache.Scope, &cache.ExpiresAt,
		&cache.IssuedAt, &cache.CachedAt, &cache.LastAccessed,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("cache miss")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	return cache, nil
}

// cacheIntrospection caches introspection result
func (s *TokenIntrospectionService) cacheIntrospection(ctx context.Context, tokenHash string, response *IntrospectionResponse) error {
	query := `
		INSERT INTO token_introspection_cache (
			token_hash, is_active, token_type, client_id, username,
			scope, expires_at, issued_at, cached_at, last_accessed
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (token_hash) DO UPDATE SET
			is_active = EXCLUDED.is_active,
			token_type = EXCLUDED.token_type,
			client_id = EXCLUDED.client_id,
			username = EXCLUDED.username,
			scope = EXCLUDED.scope,
			expires_at = EXCLUDED.expires_at,
			issued_at = EXCLUDED.issued_at,
			cached_at = EXCLUDED.cached_at,
			last_accessed = EXCLUDED.last_accessed
	`

	now := time.Now()
	var expiresAt, issuedAt *time.Time

	if response.ExpiresAt != nil {
		exp := time.Unix(*response.ExpiresAt, 0)
		expiresAt = &exp
	}

	if response.IssuedAt != nil {
		iat := time.Unix(*response.IssuedAt, 0)
		issuedAt = &iat
	}

	_, err := s.db.ExecContext(ctx, query,
		tokenHash, response.Active, response.TokenType, response.ClientID,
		response.Username, response.Scope, expiresAt, issuedAt, now, now,
	)

	return err
}

// buildIntrospectionResponse builds response from cached data
func (s *TokenIntrospectionService) buildIntrospectionResponse(cache *models.TokenIntrospectionCache) *IntrospectionResponse {
	response := &IntrospectionResponse{
		Active:    cache.IsActive,
		TokenType: cache.TokenType,
		ClientID:  cache.ClientID,
		Username:  cache.Username,
		Scope:     cache.Scope,
	}

	if cache.ExpiresAt != nil {
		exp := cache.ExpiresAt.Unix()
		response.ExpiresAt = &exp
	}

	if cache.IssuedAt != nil {
		iat := cache.IssuedAt.Unix()
		response.IssuedAt = &iat
	}

	return response
}

// updateLastAccessed updates the last accessed timestamp for cached token
func (s *TokenIntrospectionService) updateLastAccessed(ctx context.Context, tokenHash string) error {
	query := `UPDATE token_introspection_cache SET last_accessed = NOW() WHERE token_hash = $1`
	_, err := s.db.ExecContext(ctx, query, tokenHash)
	return err
}

// updateAPIKeyLastUsed updates the last used timestamp for API key
func (s *TokenIntrospectionService) updateAPIKeyLastUsed(ctx context.Context, keyID uuid.UUID) error {
	query := `UPDATE api_keys SET last_used_at = NOW() WHERE key_id = $1`
	_, err := s.db.ExecContext(ctx, query, keyID)
	return err
}

// hashToken creates a SHA-256 hash of the token for storage
func (s *TokenIntrospectionService) hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// RevokeToken revokes a token by removing it from cache and marking as inactive
func (s *TokenIntrospectionService) RevokeToken(ctx context.Context, token string) error {
	tokenHash := s.hashToken(token)

	// Remove from cache
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM token_introspection_cache WHERE token_hash = $1`,
		tokenHash)
	if err != nil {
		return fmt.Errorf("failed to remove from cache: %w", err)
	}

	// Mark API key as inactive if it's an API key
	_, err = s.db.ExecContext(ctx,
		`UPDATE api_keys SET is_active = false WHERE key_hash = $1`,
		tokenHash)
	if err != nil {
		// This might fail if it's not an API key, which is OK
	}

	return nil
}

// CleanupExpiredCache removes expired entries from the introspection cache
func (s *TokenIntrospectionService) CleanupExpiredCache(ctx context.Context) error {
	query := `DELETE FROM token_introspection_cache WHERE expires_at < NOW()`
	_, err := s.db.ExecContext(ctx, query)
	return err
}

// GetTokenStats returns statistics about cached tokens
func (s *TokenIntrospectionService) GetTokenStats(ctx context.Context) (map[string]interface{}, error) {
	query := `
		SELECT 
			COUNT(*) as total_cached,
			COUNT(*) FILTER (WHERE is_active = true) as active_tokens,
			COUNT(*) FILTER (WHERE token_type = 'api_key') as api_keys,
			COUNT(*) FILTER (WHERE token_type = 'Bearer') as access_tokens,
			COUNT(*) FILTER (WHERE expires_at < NOW()) as expired_tokens
		FROM token_introspection_cache
	`

	var stats struct {
		TotalCached   int `db:"total_cached"`
		ActiveTokens  int `db:"active_tokens"`
		APIKeys       int `db:"api_keys"`
		AccessTokens  int `db:"access_tokens"`
		ExpiredTokens int `db:"expired_tokens"`
	}

	err := s.db.QueryRowContext(ctx, query).Scan(
		&stats.TotalCached, &stats.ActiveTokens, &stats.APIKeys,
		&stats.AccessTokens, &stats.ExpiredTokens,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get token stats: %w", err)
	}

	return map[string]interface{}{
		"total_cached":   stats.TotalCached,
		"active_tokens":  stats.ActiveTokens,
		"api_keys":       stats.APIKeys,
		"access_tokens":  stats.AccessTokens,
		"expired_tokens": stats.ExpiredTokens,
	}, nil
}
