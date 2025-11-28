package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/securestor/securestor/internal/models"
	"golang.org/x/crypto/bcrypt"
)

// OAuth2Service handles OAuth2 client credentials flow
type OAuth2Service struct {
	db     *sql.DB
	logger *log.Logger
}

// NewOAuth2Service creates a new OAuth2 service
func NewOAuth2Service(db *sql.DB, logger *log.Logger) *OAuth2Service {
	return &OAuth2Service{
		db:     db,
		logger: logger,
	}
}

// ClientCredentialsRequest represents a client credentials grant request
type ClientCredentialsRequest struct {
	GrantType    string `json:"grant_type" binding:"required"`
	ClientID     string `json:"client_id" binding:"required"`
	ClientSecret string `json:"client_secret" binding:"required"`
	Scope        string `json:"scope,omitempty"`
}

// TokenResponse represents an OAuth2 token response
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
	Scope       string `json:"scope,omitempty"`
}

// CreateOAuth2Client creates a new OAuth2 client
func (s *OAuth2Service) CreateOAuth2Client(name, description string, scopes []string, createdBy int64) (*models.OAuth2Client, string, error) {
	// Generate client ID and secret
	clientID, err := s.generateClientID()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate client ID: %w", err)
	}

	clientSecret, err := s.generateClientSecret()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate client secret: %w", err)
	}

	// Hash the client secret
	secretHash, err := bcrypt.GenerateFromPassword([]byte(clientSecret), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", fmt.Errorf("failed to hash client secret: %w", err)
	}

	// Create client record
	client := &models.OAuth2Client{
		ClientID:         clientID,
		ClientSecretHash: string(secretHash),
		Name:             name,
		Description:      &description,
		GrantTypes:       []string{"client_credentials"},
		Scopes:           scopes,
		IsActive:         true,
		CreatedBy:        createdBy,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	// Insert into database
	query := `
		INSERT INTO oauth2_clients (client_id, client_secret_hash, name, description, grant_types, scopes, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id`

	err = s.db.QueryRow(query,
		client.ClientID, client.ClientSecretHash, client.Name, client.Description,
		client.GrantTypes, client.Scopes, client.CreatedBy, client.CreatedAt, client.UpdatedAt,
	).Scan(&client.ID)

	if err != nil {
		return nil, "", fmt.Errorf("failed to create OAuth2 client: %w", err)
	}

	s.logger.Printf("Created OAuth2 client: %s (ID: %s)", client.Name, client.ClientID)

	return client, clientSecret, nil
}

// ValidateClientCredentials validates client credentials and returns the client
func (s *OAuth2Service) ValidateClientCredentials(clientID, clientSecret string) (*models.OAuth2Client, error) {
	client := &models.OAuth2Client{}

	query := `
		SELECT id, client_id, client_secret_hash, name, description, grant_types, scopes, 
		       is_active, expires_at, created_by, created_at, updated_at, last_used_at
		FROM oauth2_clients 
		WHERE client_id = $1 AND is_active = true`

	err := s.db.QueryRow(query, clientID).Scan(
		&client.ID, &client.ClientID, &client.ClientSecretHash, &client.Name,
		&client.Description, &client.GrantTypes, &client.Scopes, &client.IsActive,
		&client.ExpiresAt, &client.CreatedBy, &client.CreatedAt, &client.UpdatedAt,
		&client.LastUsedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invalid client credentials")
		}
		return nil, fmt.Errorf("failed to validate client: %w", err)
	}

	// Check if client is expired
	if client.ExpiresAt != nil && client.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("client credentials expired")
	}

	// Verify client secret
	err = bcrypt.CompareHashAndPassword([]byte(client.ClientSecretHash), []byte(clientSecret))
	if err != nil {
		return nil, fmt.Errorf("invalid client credentials")
	}

	// Update last used timestamp
	s.updateClientLastUsed(clientID)

	return client, nil
}

// GenerateAccessToken creates a JWT access token for client credentials
func (s *OAuth2Service) GenerateAccessToken(client *models.OAuth2Client, scopes []string) (string, int64, error) {
	// Validate requested scopes against client's allowed scopes
	validScopes := s.validateScopes(scopes, client.Scopes)

	// For now, we'll create a simple signed token
	// In production, you'd use proper JWT with signing
	expiresIn := int64(3600) // 1 hour
	expiresAt := time.Now().Add(time.Duration(expiresIn) * time.Second)

	// Create token payload
	tokenData := fmt.Sprintf("%s:%s:%d:%s",
		client.ClientID,
		fmt.Sprintf("%v", validScopes),
		expiresAt.Unix(),
		s.generateTokenSalt())

	// Create a simple token (in production, use proper JWT)
	hash := sha256.Sum256([]byte(tokenData))
	token := base64.URLEncoding.EncodeToString(hash[:])

	// Store token reference for validation (optional)
	// You might want to store active tokens for revocation

	return token, expiresIn, nil
}

// ListOAuth2Clients returns a list of OAuth2 clients for a user
func (s *OAuth2Service) ListOAuth2Clients(createdBy int64) ([]*models.OAuth2Client, error) {
	query := `
		SELECT id, client_id, name, description, grant_types, scopes, 
		       is_active, expires_at, created_by, created_at, updated_at, last_used_at
		FROM oauth2_clients 
		WHERE created_by = $1 
		ORDER BY created_at DESC`

	rows, err := s.db.Query(query, createdBy)
	if err != nil {
		return nil, fmt.Errorf("failed to list OAuth2 clients: %w", err)
	}
	defer rows.Close()

	var clients []*models.OAuth2Client
	for rows.Next() {
		client := &models.OAuth2Client{}
		err := rows.Scan(
			&client.ID, &client.ClientID, &client.Name, &client.Description,
			&client.GrantTypes, &client.Scopes, &client.IsActive, &client.ExpiresAt,
			&client.CreatedBy, &client.CreatedAt, &client.UpdatedAt, &client.LastUsedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan OAuth2 client: %w", err)
		}
		clients = append(clients, client)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating OAuth2 clients: %w", err)
	}

	return clients, nil
}

// RevokeOAuth2Client deactivates an OAuth2 client
func (s *OAuth2Service) RevokeOAuth2Client(clientID string, userID int64) error {
	query := `
		UPDATE oauth2_clients 
		SET is_active = false, updated_at = NOW() 
		WHERE client_id = $1 AND created_by = $2`

	result, err := s.db.Exec(query, clientID, userID)
	if err != nil {
		return fmt.Errorf("failed to revoke OAuth2 client: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("OAuth2 client not found or not owned by user")
	}

	s.logger.Printf("Revoked OAuth2 client: %s", clientID)

	return nil
}

// GetAvailableScopes returns available OAuth2 scopes
func (s *OAuth2Service) GetAvailableScopes() ([]string, error) {
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

// Helper functions

func (s *OAuth2Service) generateClientID() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("scs_%s", hex.EncodeToString(b)), nil
}

func (s *OAuth2Service) generateClientSecret() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func (s *OAuth2Service) generateTokenSalt() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *OAuth2Service) validateScopes(requested, allowed []string) []string {
	if len(requested) == 0 {
		// Return default scope if none requested
		return []string{"read"}
	}

	allowedMap := make(map[string]bool)
	for _, scope := range allowed {
		allowedMap[scope] = true
	}

	var validScopes []string
	for _, scope := range requested {
		if allowedMap[scope] || allowedMap["*"] {
			validScopes = append(validScopes, scope)
		}
	}

	if len(validScopes) == 0 {
		// Return default scope if no valid scopes
		return []string{"read"}
	}

	return validScopes
}

func (s *OAuth2Service) updateClientLastUsed(clientID string) {
	query := `UPDATE oauth2_clients SET last_used_at = NOW() WHERE client_id = $1`
	_, err := s.db.Exec(query, clientID)
	if err != nil {
		s.logger.Printf("Failed to update client last used timestamp: %v", err)
	}
}
