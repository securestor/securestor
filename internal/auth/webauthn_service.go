package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/securestor/securestor/internal/logger"
	"github.com/securestor/securestor/internal/models"
)

// WebAuthnService handles WebAuthn/FIDO2 authentication
type WebAuthnService struct {
	db       *sql.DB
	logger   *logger.Logger
	rpID     string // Relying Party ID (domain)
	rpName   string // Relying Party Name
	rpOrigin string // Relying Party Origin
}

// NewWebAuthnService creates a new WebAuthn service
func NewWebAuthnService(db *sql.DB, logger *logger.Logger, rpID, rpName, rpOrigin string) *WebAuthnService {
	return &WebAuthnService{
		db:       db,
		logger:   logger,
		rpID:     rpID,
		rpName:   rpName,
		rpOrigin: rpOrigin,
	}
}

// GenerateRegistrationOptions generates WebAuthn registration options
func (s *WebAuthnService) GenerateRegistrationOptions(userID int64, username, displayName string) (*models.WebAuthnRegistrationOptions, error) {
	// Generate challenge
	challenge := make([]byte, 32)
	if _, err := rand.Read(challenge); err != nil {
		return nil, fmt.Errorf("failed to generate challenge: %w", err)
	}

	challengeB64 := base64.RawURLEncoding.EncodeToString(challenge)

	// Get existing credentials for this user to exclude them
	_, err := s.getUserCredentials(userID)
	if err != nil {
		s.logger.Error("Failed to get existing credentials", err)
		// Continue without exclusions
	}

	// Create user ID (use SHA256 hash of user ID for privacy)
	userIDHash := sha256.Sum256([]byte(fmt.Sprintf("%d", userID)))
	userIDB64 := base64.RawURLEncoding.EncodeToString(userIDHash[:])

	options := &models.WebAuthnRegistrationOptions{
		Challenge: challengeB64,
		RelyingParty: models.WebAuthnRelyingParty{
			ID:   s.rpID,
			Name: s.rpName,
		},
		User: models.WebAuthnUser{
			ID:          userIDB64,
			Name:        username,
			DisplayName: displayName,
		},
		PublicKeyCredParams: []models.WebAuthnCredParam{
			{Type: "public-key", Alg: -7},   // ES256
			{Type: "public-key", Alg: -257}, // RS256
		},
		AuthenticatorSelection: models.WebAuthnAuthSelection{
			RequireResidentKey: false,
			UserVerification:   "preferred",
		},
		Timeout:     60000, // 60 seconds
		Attestation: "none",
	}

	// Store challenge in database for verification
	err = s.storeChallenge(userID, challengeB64, "registration")
	if err != nil {
		return nil, fmt.Errorf("failed to store challenge: %w", err)
	}

	return options, nil
}

// VerifyRegistration verifies a WebAuthn registration response
func (s *WebAuthnService) VerifyRegistration(userID int64, response *models.WebAuthnRegistrationResponse) error {
	// Verify challenge
	valid, err := s.verifyAndConsumeChallenge(userID, "registration")
	if err != nil {
		return fmt.Errorf("failed to verify challenge: %w", err)
	}
	if !valid {
		return fmt.Errorf("invalid or expired challenge")
	}

	// In a real implementation, you would:
	// 1. Verify the attestation object
	// 2. Verify the client data JSON
	// 3. Verify the signature
	// 4. Extract the public key

	// For this simplified implementation, we'll store the credential
	// assuming the client-side verification passed

	credential := &models.WebAuthnCredential{
		UserID:            userID,
		CredentialID:      response.ID,
		PublicKey:         response.Response.AttestationObject, // Simplified
		AuthenticatorData: response.Response.ClientDataJSON,    // Simplified
		SignCount:         0,
		Name:              response.DeviceName,
		DeviceType:        "cross-platform", // Default
		IsActive:          true,
		CreatedAt:         time.Now(),
	}

	// Determine device type based on authenticator data
	// This is a simplified heuristic
	if response.DeviceName != "" {
		credential.Name = response.DeviceName
	} else {
		credential.Name = "Security Key"
	}

	err = s.storeCredential(credential)
	if err != nil {
		return fmt.Errorf("failed to store credential: %w", err)
	}

	return nil
}

// GenerateAssertionOptions generates WebAuthn assertion options for authentication
func (s *WebAuthnService) GenerateAssertionOptions(userID int64) (*models.WebAuthnAssertionOptions, error) {
	// Generate challenge
	challenge := make([]byte, 32)
	if _, err := rand.Read(challenge); err != nil {
		return nil, fmt.Errorf("failed to generate challenge: %w", err)
	}

	challengeB64 := base64.RawURLEncoding.EncodeToString(challenge)

	// Get user's credentials
	credentials, err := s.getUserCredentials(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user credentials: %w", err)
	}

	if len(credentials) == 0 {
		return nil, fmt.Errorf("no WebAuthn credentials found for user")
	}

	// Create allow list
	allowCredentials := make([]models.WebAuthnCredentialDescriptor, len(credentials))
	for i, cred := range credentials {
		allowCredentials[i] = models.WebAuthnCredentialDescriptor{
			Type:       "public-key",
			ID:         cred.CredentialID,
			Transports: []string{"usb", "nfc", "ble", "internal"},
		}
	}

	options := &models.WebAuthnAssertionOptions{
		Challenge:        challengeB64,
		Timeout:          60000, // 60 seconds
		RelyingPartyID:   s.rpID,
		AllowCredentials: allowCredentials,
		UserVerification: "preferred",
	}

	// Store challenge in database for verification
	err = s.storeChallenge(userID, challengeB64, "authentication")
	if err != nil {
		return nil, fmt.Errorf("failed to store challenge: %w", err)
	}

	return options, nil
}

// VerifyAssertion verifies a WebAuthn assertion response
func (s *WebAuthnService) VerifyAssertion(userID int64, response *models.WebAuthnAssertionResponse) error {
	// Verify challenge
	valid, err := s.verifyAndConsumeChallenge(userID, "authentication")
	if err != nil {
		return fmt.Errorf("failed to verify challenge: %w", err)
	}
	if !valid {
		return fmt.Errorf("invalid or expired challenge")
	}

	// Get credential
	credential, err := s.getCredentialByID(response.ID)
	if err != nil {
		return fmt.Errorf("credential not found: %w", err)
	}

	if credential.UserID != userID {
		return fmt.Errorf("credential does not belong to user")
	}

	if !credential.IsActive {
		return fmt.Errorf("credential is inactive")
	}

	// In a real implementation, you would:
	// 1. Verify the signature using the stored public key
	// 2. Verify the authenticator data
	// 3. Verify the client data JSON
	// 4. Check and update the sign count

	// For this simplified implementation, we'll just update the last used time
	err = s.updateCredentialLastUsed(credential.CredentialID)
	if err != nil {
		s.logger.Error("Failed to update credential last used", err)
	}

	return nil
}

// GetUserCredentials returns all active credentials for a user
func (s *WebAuthnService) GetUserCredentials(userID int64) ([]models.WebAuthnCredential, error) {
	return s.getUserCredentials(userID)
}

// DeleteCredential deletes a WebAuthn credential
func (s *WebAuthnService) DeleteCredential(userID int64, credentialID string) error {
	query := `
		UPDATE webauthn_credentials 
		SET is_active = false 
		WHERE credential_id = $1 AND user_id = $2
	`

	result, err := s.db.Exec(query, credentialID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete credential: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("credential not found or already deleted")
	}

	return nil
}

// RenameCredential renames a WebAuthn credential
func (s *WebAuthnService) RenameCredential(userID int64, credentialID, newName string) error {
	query := `
		UPDATE webauthn_credentials 
		SET name = $1 
		WHERE credential_id = $2 AND user_id = $3 AND is_active = true
	`

	result, err := s.db.Exec(query, newName, credentialID, userID)
	if err != nil {
		return fmt.Errorf("failed to rename credential: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("credential not found")
	}

	return nil
}

// getUserCredentials gets all active credentials for a user
func (s *WebAuthnService) getUserCredentials(userID int64) ([]models.WebAuthnCredential, error) {
	query := `
		SELECT id, user_id, credential_id, public_key, authenticator_data, 
		       sign_count, name, device_type, is_active, created_at, last_used_at
		FROM webauthn_credentials 
		WHERE user_id = $1 AND is_active = true
		ORDER BY created_at DESC
	`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query credentials: %w", err)
	}
	defer rows.Close()

	var credentials []models.WebAuthnCredential
	for rows.Next() {
		var cred models.WebAuthnCredential
		err := rows.Scan(
			&cred.ID, &cred.UserID, &cred.CredentialID,
			&cred.PublicKey, &cred.AuthenticatorData, &cred.SignCount,
			&cred.Name, &cred.DeviceType, &cred.IsActive,
			&cred.CreatedAt, &cred.LastUsedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan credential: %w", err)
		}

		// Don't expose sensitive data
		cred.PublicKey = ""
		cred.AuthenticatorData = ""

		credentials = append(credentials, cred)
	}

	return credentials, rows.Err()
}

// getCredentialByID gets a credential by its ID
func (s *WebAuthnService) getCredentialByID(credentialID string) (*models.WebAuthnCredential, error) {
	query := `
		SELECT id, user_id, credential_id, public_key, authenticator_data, 
		       sign_count, name, device_type, is_active, created_at, last_used_at
		FROM webauthn_credentials 
		WHERE credential_id = $1
	`

	var cred models.WebAuthnCredential
	err := s.db.QueryRow(query, credentialID).Scan(
		&cred.ID, &cred.UserID, &cred.CredentialID,
		&cred.PublicKey, &cred.AuthenticatorData, &cred.SignCount,
		&cred.Name, &cred.DeviceType, &cred.IsActive,
		&cred.CreatedAt, &cred.LastUsedAt,
	)

	if err != nil {
		return nil, err
	}

	return &cred, nil
}

// storeCredential stores a new WebAuthn credential
func (s *WebAuthnService) storeCredential(cred *models.WebAuthnCredential) error {
	query := `
		INSERT INTO webauthn_credentials 
		(user_id, credential_id, public_key, authenticator_data, sign_count, 
		 name, device_type, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := s.db.Exec(query,
		cred.UserID, cred.CredentialID, cred.PublicKey,
		cred.AuthenticatorData, cred.SignCount, cred.Name,
		cred.DeviceType, cred.IsActive, cred.CreatedAt,
	)

	return err
}

// updateCredentialLastUsed updates the last used timestamp for a credential
func (s *WebAuthnService) updateCredentialLastUsed(credentialID string) error {
	query := `
		UPDATE webauthn_credentials 
		SET last_used_at = CURRENT_TIMESTAMP, sign_count = sign_count + 1
		WHERE credential_id = $1
	`

	_, err := s.db.Exec(query, credentialID)
	return err
}

// storeChallenge stores a challenge for later verification
func (s *WebAuthnService) storeChallenge(userID int64, challenge, challengeType string) error {
	// Store in a temporary table or Redis
	// For this implementation, we'll use a simple approach with database
	query := `
		INSERT INTO webauthn_challenges (user_id, challenge, challenge_type, expires_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, challenge_type) 
		DO UPDATE SET 
			challenge = EXCLUDED.challenge,
			expires_at = EXCLUDED.expires_at,
			created_at = CURRENT_TIMESTAMP
	`

	expiresAt := time.Now().Add(5 * time.Minute) // 5 minute expiry

	_, err := s.db.Exec(query, userID, challenge, challengeType, expiresAt)
	return err
}

// verifyAndConsumeChallenge verifies and consumes a challenge
func (s *WebAuthnService) verifyAndConsumeChallenge(userID int64, challengeType string) (bool, error) {
	// Get and delete the challenge in one operation
	query := `
		DELETE FROM webauthn_challenges 
		WHERE user_id = $1 AND challenge_type = $2 AND expires_at > CURRENT_TIMESTAMP
		RETURNING challenge
	`

	var challenge string
	err := s.db.QueryRow(query, userID, challengeType).Scan(&challenge)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}

	return true, nil
}
