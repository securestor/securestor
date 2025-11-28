package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
	"time"

	"github.com/securestor/securestor/internal/models"
)

// RefreshTokenService handles refresh token rotation and management
type RefreshTokenService struct {
	db     *sql.DB
	logger *log.Logger
}

// NewRefreshTokenService creates a new refresh token service
func NewRefreshTokenService(db *sql.DB, logger *log.Logger) *RefreshTokenService {
	return &RefreshTokenService{
		db:     db,
		logger: logger,
	}
}

// StoreRefreshToken stores a refresh token in the database
func (s *RefreshTokenService) StoreRefreshToken(userID int64, sessionID string, refreshToken string, clientID *string, expiresAt time.Time) (*models.RefreshTokenStore, error) {
	// Hash the refresh token
	tokenHash := s.hashToken(refreshToken)

	// Insert into database
	query := `
		INSERT INTO refresh_token_store (token_hash, user_id, session_id, client_id, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`

	tokenStore := &models.RefreshTokenStore{
		TokenHash: tokenHash,
		UserID:    userID,
		SessionID: sessionID,
		ClientID:  clientID,
		ExpiresAt: expiresAt,
		IsRevoked: false,
		CreatedAt: time.Now(),
	}

	err := s.db.QueryRow(query,
		tokenStore.TokenHash, tokenStore.UserID, tokenStore.SessionID,
		tokenStore.ClientID, tokenStore.ExpiresAt, tokenStore.CreatedAt,
	).Scan(&tokenStore.ID)

	if err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	return tokenStore, nil
}

// ValidateRefreshToken validates a refresh token and returns the stored token info
func (s *RefreshTokenService) ValidateRefreshToken(refreshToken string) (*models.RefreshTokenStore, error) {
	tokenHash := s.hashToken(refreshToken)

	tokenStore := &models.RefreshTokenStore{}
	query := `
		SELECT id, token_hash, user_id, session_id, client_id, expires_at, 
		       is_revoked, revoked_at, parent_token_id, created_at
		FROM refresh_token_store 
		WHERE token_hash = $1 AND is_revoked = false AND expires_at > NOW()`

	err := s.db.QueryRow(query, tokenHash).Scan(
		&tokenStore.ID, &tokenStore.TokenHash, &tokenStore.UserID, &tokenStore.SessionID,
		&tokenStore.ClientID, &tokenStore.ExpiresAt, &tokenStore.IsRevoked,
		&tokenStore.RevokedAt, &tokenStore.ParentTokenID, &tokenStore.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invalid or expired refresh token")
		}
		return nil, fmt.Errorf("failed to validate refresh token: %w", err)
	}

	return tokenStore, nil
}

// RotateRefreshToken creates a new refresh token and revokes the old one
func (s *RefreshTokenService) RotateRefreshToken(oldRefreshToken string, newRefreshToken string, expiresAt time.Time) (*models.RefreshTokenStore, error) {
	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Validate and get the old token
	oldTokenStore, err := s.ValidateRefreshToken(oldRefreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid old refresh token: %w", err)
	}

	// Revoke the old token
	revokeQuery := `
		UPDATE refresh_token_store 
		SET is_revoked = true, revoked_at = NOW() 
		WHERE id = $1`

	_, err = tx.Exec(revokeQuery, oldTokenStore.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to revoke old token: %w", err)
	}

	// Create new token hash
	newTokenHash := s.hashToken(newRefreshToken)

	// Insert new token
	insertQuery := `
		INSERT INTO refresh_token_store (token_hash, user_id, session_id, client_id, expires_at, parent_token_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`

	newTokenStore := &models.RefreshTokenStore{
		TokenHash:     newTokenHash,
		UserID:        oldTokenStore.UserID,
		SessionID:     oldTokenStore.SessionID,
		ClientID:      oldTokenStore.ClientID,
		ExpiresAt:     expiresAt,
		IsRevoked:     false,
		ParentTokenID: &oldTokenStore.ID,
		CreatedAt:     time.Now(),
	}

	err = tx.QueryRow(insertQuery,
		newTokenStore.TokenHash, newTokenStore.UserID, newTokenStore.SessionID,
		newTokenStore.ClientID, newTokenStore.ExpiresAt, newTokenStore.ParentTokenID,
		newTokenStore.CreatedAt,
	).Scan(&newTokenStore.ID)

	if err != nil {
		return nil, fmt.Errorf("failed to create new refresh token: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit token rotation: %w", err)
	}

	s.logger.Printf("Rotated refresh token for user %d (session: %s)", newTokenStore.UserID, newTokenStore.SessionID)

	return newTokenStore, nil
}

// RevokeRefreshToken revokes a refresh token
func (s *RefreshTokenService) RevokeRefreshToken(refreshToken string) error {
	tokenHash := s.hashToken(refreshToken)

	query := `
		UPDATE refresh_token_store 
		SET is_revoked = true, revoked_at = NOW() 
		WHERE token_hash = $1 AND is_revoked = false`

	result, err := s.db.Exec(query, tokenHash)
	if err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("refresh token not found or already revoked")
	}

	return nil
}

// RevokeAllUserTokens revokes all refresh tokens for a user
func (s *RefreshTokenService) RevokeAllUserTokens(userID int64) error {
	query := `
		UPDATE refresh_token_store 
		SET is_revoked = true, revoked_at = NOW() 
		WHERE user_id = $1 AND is_revoked = false`

	result, err := s.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("failed to revoke user tokens: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	s.logger.Printf("Revoked %d refresh tokens for user %d", rowsAffected, userID)

	return nil
}

// RevokeSessionTokens revokes all refresh tokens for a session
func (s *RefreshTokenService) RevokeSessionTokens(sessionID string) error {
	query := `
		UPDATE refresh_token_store 
		SET is_revoked = true, revoked_at = NOW() 
		WHERE session_id = $1 AND is_revoked = false`

	result, err := s.db.Exec(query, sessionID)
	if err != nil {
		return fmt.Errorf("failed to revoke session tokens: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	s.logger.Printf("Revoked %d refresh tokens for session %s", rowsAffected, sessionID)

	return nil
}

// CleanupExpiredTokens removes expired refresh tokens from the database
func (s *RefreshTokenService) CleanupExpiredTokens() (int64, error) {
	query := `
		DELETE FROM refresh_token_store 
		WHERE expires_at < NOW() - INTERVAL '7 days'`

	result, err := s.db.Exec(query)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired tokens: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected > 0 {
		s.logger.Printf("Cleaned up %d expired refresh tokens", rowsAffected)
	}

	return rowsAffected, nil
}

// GetTokenStats returns statistics about refresh tokens
func (s *RefreshTokenService) GetTokenStats() (map[string]interface{}, error) {
	query := `
		SELECT 
			COUNT(*) as total_tokens,
			COUNT(CASE WHEN is_revoked = false THEN 1 END) as active_tokens,
			COUNT(CASE WHEN is_revoked = true THEN 1 END) as revoked_tokens,
			COUNT(CASE WHEN expires_at < NOW() THEN 1 END) as expired_tokens,
			COUNT(CASE WHEN created_at > NOW() - INTERVAL '24 hours' THEN 1 END) as recent_tokens
		FROM refresh_token_store`

	var totalTokens, activeTokens, revokedTokens, expiredTokens, recentTokens int
	err := s.db.QueryRow(query).Scan(&totalTokens, &activeTokens, &revokedTokens, &expiredTokens, &recentTokens)
	if err != nil {
		return nil, fmt.Errorf("failed to get token stats: %w", err)
	}

	stats := map[string]interface{}{
		"total_tokens":   totalTokens,
		"active_tokens":  activeTokens,
		"revoked_tokens": revokedTokens,
		"expired_tokens": expiredTokens,
		"recent_tokens":  recentTokens,
	}

	return stats, nil
}

// GetUserTokens returns active refresh tokens for a user
func (s *RefreshTokenService) GetUserTokens(userID int64) ([]*models.RefreshTokenStore, error) {
	query := `
		SELECT id, token_hash, user_id, session_id, client_id, expires_at, 
		       is_revoked, revoked_at, parent_token_id, created_at
		FROM refresh_token_store 
		WHERE user_id = $1 AND is_revoked = false 
		ORDER BY created_at DESC`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user tokens: %w", err)
	}
	defer rows.Close()

	var tokens []*models.RefreshTokenStore
	for rows.Next() {
		token := &models.RefreshTokenStore{}
		err := rows.Scan(
			&token.ID, &token.TokenHash, &token.UserID, &token.SessionID,
			&token.ClientID, &token.ExpiresAt, &token.IsRevoked, &token.RevokedAt,
			&token.ParentTokenID, &token.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan token: %w", err)
		}
		tokens = append(tokens, token)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tokens: %w", err)
	}

	return tokens, nil
}

// GenerateRefreshToken generates a new cryptographically secure refresh token
func (s *RefreshTokenService) GenerateRefreshToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("failed to generate refresh token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// hashToken creates a hash of the refresh token for storage
func (s *RefreshTokenService) hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return base64.URLEncoding.EncodeToString(hash[:])
}
