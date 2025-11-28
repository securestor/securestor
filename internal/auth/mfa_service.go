package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"database/sql"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"strings"
	"time"

	"github.com/securestor/securestor/internal/logger"
	"github.com/securestor/securestor/internal/models"
	"golang.org/x/crypto/bcrypt"
)

// MFAService handles Multi-Factor Authentication operations
type MFAService struct {
	db         *sql.DB
	logger     *logger.Logger
	encryptKey []byte
	issuer     string
}

// NewMFAService creates a new MFA service
func NewMFAService(db *sql.DB, logger *logger.Logger, encryptionKey string, issuer string) *MFAService {
	// Create a 32-byte key from the provided string
	hash := sha256.Sum256([]byte(encryptionKey))

	return &MFAService{
		db:         db,
		logger:     logger,
		encryptKey: hash[:],
		issuer:     issuer,
	}
}

// GetUserMFAStatus retrieves the MFA status for a user
func (s *MFAService) GetUserMFAStatus(userID int64) (*models.UserMFAStatus, error) {
	query := `
		SELECT 
			user_id, email, username, mfa_required, mfa_enforced_at,
			is_mfa_enabled, totp_verified, webauthn_count, 
			recovery_codes_count, mfa_setup_at, mfa_updated_at
		FROM user_mfa_status 
		WHERE user_id = $1
	`

	status := &models.UserMFAStatus{}
	err := s.db.QueryRow(query, userID).Scan(
		&status.UserID, &status.Email, &status.Username,
		&status.MFARequired, &status.MFAEnforcedAt,
		&status.IsMFAEnabled, &status.TOTPVerified, &status.WebAuthnCount,
		&status.RecoveryCodesCount, &status.MFASetupAt, &status.MFAUpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get MFA status: %w", err)
	}

	return status, nil
}

// SetupTOTP generates a new TOTP secret for a user
func (s *MFAService) SetupTOTP(userID int64, username string) (*models.TOTPSetupResponse, error) {
	// Generate TOTP secret (32 bytes = 256 bits)
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, fmt.Errorf("failed to generate TOTP secret: %w", err)
	}

	// Convert to base32 for compatibility with authenticator apps
	secretBase32 := base32.StdEncoding.EncodeToString(secret)

	// Encrypt the secret
	encryptedSecret, err := s.encryptString(secretBase32)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt TOTP secret: %w", err)
	}

	// Generate backup codes
	backupCodes, err := s.generateBackupCodes()
	if err != nil {
		return nil, fmt.Errorf("failed to generate backup codes: %w", err)
	}

	// Store in database
	query := `
		INSERT INTO user_totp_secrets (user_id, secret_encrypted, backup_codes)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id) 
		DO UPDATE SET 
			secret_encrypted = EXCLUDED.secret_encrypted,
			backup_codes = EXCLUDED.backup_codes,
			is_verified = false,
			verified_at = NULL,
			created_at = CURRENT_TIMESTAMP
	`

	// Encrypt backup codes
	encryptedBackupCodes := make([]string, len(backupCodes))
	for i, code := range backupCodes {
		encrypted, err := s.encryptString(code)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt backup code: %w", err)
		}
		encryptedBackupCodes[i] = encrypted
	}

	_, err = s.db.Exec(query, userID, encryptedSecret, encryptedBackupCodes)
	if err != nil {
		return nil, fmt.Errorf("failed to store TOTP secret: %w", err)
	}

	// Create QR code URL for Google Authenticator format
	qrURL := fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s",
		s.issuer, username, secretBase32, s.issuer)

	return &models.TOTPSetupResponse{
		Secret:      secretBase32,
		QRCodeURL:   qrURL,
		BackupCodes: backupCodes,
	}, nil
}

// VerifyTOTP verifies a TOTP code for a user
func (s *MFAService) VerifyTOTP(userID int64, code string, markAsVerified bool) error {
	// Get encrypted secret
	query := `
		SELECT secret_encrypted, is_verified, last_used_at
		FROM user_totp_secrets 
		WHERE user_id = $1
	`

	var encryptedSecret string
	var isVerified bool
	var lastUsedAt *time.Time

	err := s.db.QueryRow(query, userID).Scan(&encryptedSecret, &isVerified, &lastUsedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("TOTP not setup for user")
		}
		return fmt.Errorf("failed to get TOTP secret: %w", err)
	}

	// Decrypt secret
	secret, err := s.decryptString(encryptedSecret)
	if err != nil {
		return fmt.Errorf("failed to decrypt TOTP secret: %w", err)
	}

	// Verify TOTP code
	valid := s.validateTOTP(code, secret)
	if !valid {
		// Log failed attempt
		s.logMFAAttempt(userID, "totp", "verify", false, "Invalid TOTP code")
		return fmt.Errorf("invalid TOTP code")
	}

	// Check for replay attacks (if code was used very recently)
	if lastUsedAt != nil && time.Since(*lastUsedAt) < 30*time.Second {
		// Generate the current time window code to see if it's the same
		currentCode := s.generateTOTP(secret, time.Now())
		if currentCode == code {
			s.logMFAAttempt(userID, "totp", "verify", false, "Code reuse detected")
			return fmt.Errorf("TOTP code already used")
		}
	}

	// Update verification status and last used time
	updateQuery := `
		UPDATE user_totp_secrets 
		SET last_used_at = CURRENT_TIMESTAMP
	`

	if markAsVerified && !isVerified {
		updateQuery += `, is_verified = true, verified_at = CURRENT_TIMESTAMP`
	}

	updateQuery += ` WHERE user_id = $1`

	_, err = s.db.Exec(updateQuery, userID)
	if err != nil {
		return fmt.Errorf("failed to update TOTP verification: %w", err)
	}

	// Enable MFA if this is the first verification
	if markAsVerified && !isVerified {
		err = s.enableMFAForUser(userID)
		if err != nil {
			s.logger.Error("Failed to enable MFA for user after TOTP verification", err)
		}
	}

	// Log successful attempt
	s.logMFAAttempt(userID, "totp", "verify", true, "")

	return nil
}

// VerifyBackupCode verifies a backup code for a user
func (s *MFAService) VerifyBackupCode(userID int64, code string) error {
	// Get encrypted backup codes
	query := `
		SELECT backup_codes
		FROM user_totp_secrets 
		WHERE user_id = $1 AND is_verified = true
	`

	var encryptedBackupCodes []string
	err := s.db.QueryRow(query, userID).Scan(&encryptedBackupCodes)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("no backup codes found for user")
		}
		return fmt.Errorf("failed to get backup codes: %w", err)
	}

	// Check each backup code
	for _, encryptedCode := range encryptedBackupCodes {
		decryptedCode, err := s.decryptString(encryptedCode)
		if err != nil {
			continue
		}

		if decryptedCode == code {
			// Code is valid, now check if it's already used
			checkQuery := `
				SELECT id FROM mfa_recovery_codes 
				WHERE user_id = $1 AND code_hash = $2 AND is_used = true
			`

			codeHash, _ := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)

			var recoveryID int64
			err = s.db.QueryRow(checkQuery, userID, string(codeHash)).Scan(&recoveryID)
			if err == nil {
				// Code already used
				s.logMFAAttempt(userID, "recovery", "verify", false, "Backup code already used")
				return fmt.Errorf("backup code already used")
			}

			// Mark code as used
			insertQuery := `
				INSERT INTO mfa_recovery_codes (user_id, code_hash, is_used, used_at)
				VALUES ($1, $2, true, CURRENT_TIMESTAMP)
			`

			_, err = s.db.Exec(insertQuery, userID, string(codeHash))
			if err != nil {
				s.logger.Error("Failed to mark backup code as used", err)
			}

			// Log successful attempt
			s.logMFAAttempt(userID, "recovery", "verify", true, "")

			return nil
		}
	}

	// No valid code found
	s.logMFAAttempt(userID, "recovery", "verify", false, "Invalid backup code")
	return fmt.Errorf("invalid backup code")
}

// DisableTOTP disables TOTP for a user
func (s *MFAService) DisableTOTP(userID int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete TOTP secret
	_, err = tx.Exec("DELETE FROM user_totp_secrets WHERE user_id = $1", userID)
	if err != nil {
		return fmt.Errorf("failed to delete TOTP secret: %w", err)
	}

	// Delete recovery codes
	_, err = tx.Exec("DELETE FROM mfa_recovery_codes WHERE user_id = $1", userID)
	if err != nil {
		return fmt.Errorf("failed to delete recovery codes: %w", err)
	}

	// Check if user has any other MFA methods
	var webauthnCount int
	err = tx.QueryRow("SELECT COUNT(*) FROM webauthn_credentials WHERE user_id = $1 AND is_active = true", userID).Scan(&webauthnCount)
	if err != nil {
		return fmt.Errorf("failed to check WebAuthn credentials: %w", err)
	}

	// If no other MFA methods, disable MFA
	if webauthnCount == 0 {
		_, err = tx.Exec("UPDATE user_mfa_settings SET is_mfa_enabled = false WHERE user_id = $1", userID)
		if err != nil {
			return fmt.Errorf("failed to disable MFA: %w", err)
		}
	}

	return tx.Commit()
}

// GenerateNewBackupCodes generates new backup codes for a user
func (s *MFAService) GenerateNewBackupCodes(userID int64) ([]string, error) {
	// Generate new backup codes
	backupCodes, err := s.generateBackupCodes()
	if err != nil {
		return nil, fmt.Errorf("failed to generate backup codes: %w", err)
	}

	// Encrypt backup codes
	encryptedBackupCodes := make([]string, len(backupCodes))
	for i, code := range backupCodes {
		encrypted, err := s.encryptString(code)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt backup code: %w", err)
		}
		encryptedBackupCodes[i] = encrypted
	}

	// Update in database
	query := `
		UPDATE user_totp_secrets 
		SET backup_codes = $1
		WHERE user_id = $2
	`

	_, err = s.db.Exec(query, encryptedBackupCodes, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to update backup codes: %w", err)
	}

	// Invalidate old recovery codes
	_, err = s.db.Exec("DELETE FROM mfa_recovery_codes WHERE user_id = $1", userID)
	if err != nil {
		s.logger.Error("Failed to delete old recovery codes", err)
	}

	return backupCodes, nil
}

// enableMFAForUser enables MFA for a user
func (s *MFAService) enableMFAForUser(userID int64) error {
	query := `
		INSERT INTO user_mfa_settings (user_id, is_mfa_enabled)
		VALUES ($1, true)
		ON CONFLICT (user_id)
		DO UPDATE SET is_mfa_enabled = true, updated_at = CURRENT_TIMESTAMP
	`

	_, err := s.db.Exec(query, userID)
	return err
}

// logMFAAttempt logs an MFA attempt
func (s *MFAService) logMFAAttempt(userID int64, method, attemptType string, success bool, errorMessage string) {
	query := `
		INSERT INTO mfa_attempts (user_id, mfa_method, attempt_type, success, error_message)
		VALUES ($1, $2, $3, $4, $5)
	`

	var errMsg *string
	if errorMessage != "" {
		errMsg = &errorMessage
	}

	_, err := s.db.Exec(query, userID, method, attemptType, success, errMsg)
	if err != nil {
		s.logger.Error("Failed to log MFA attempt", err)
	}
}

// encryptString encrypts a string using AES-GCM
func (s *MFAService) encryptString(plaintext string) (string, error) {
	block, err := aes.NewCipher(s.encryptKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decryptString decrypts a string using AES-GCM
func (s *MFAService) decryptString(ciphertext string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(s.encryptKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// generateTOTP generates a TOTP code for the given secret and time
func (s *MFAService) generateTOTP(secret string, t time.Time) string {
	// Decode base32 secret
	key, err := base32.StdEncoding.DecodeString(secret)
	if err != nil {
		return ""
	}

	// Calculate time counter (30-second intervals)
	counter := uint64(t.Unix() / 30)

	// Convert counter to bytes
	counterBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(counterBytes, counter)

	// HMAC-SHA1
	h := hmac.New(sha1.New, key)
	h.Write(counterBytes)
	hash := h.Sum(nil)

	// Dynamic truncation
	offset := hash[len(hash)-1] & 0x0f
	code := binary.BigEndian.Uint32(hash[offset:offset+4]) & 0x7fffffff

	// Convert to 6-digit string
	return fmt.Sprintf("%06d", code%1000000)
}

// validateTOTP validates a TOTP code against the secret
func (s *MFAService) validateTOTP(code, secret string) bool {
	// Check current time window
	if s.generateTOTP(secret, time.Now()) == code {
		return true
	}

	// Check previous time window (allow 30 second clock skew)
	if s.generateTOTP(secret, time.Now().Add(-30*time.Second)) == code {
		return true
	}

	// Check next time window (allow 30 second clock skew)
	if s.generateTOTP(secret, time.Now().Add(30*time.Second)) == code {
		return true
	}

	return false
}

// generateBackupCodes generates secure backup codes
func (s *MFAService) generateBackupCodes() ([]string, error) {
	codes := make([]string, 10) // Generate 10 backup codes

	for i := 0; i < 10; i++ {
		// Generate 8-character alphanumeric code
		bytes := make([]byte, 6)
		if _, err := rand.Read(bytes); err != nil {
			return nil, err
		}

		// Convert to base32 and take first 8 characters
		code := base32.StdEncoding.EncodeToString(bytes)
		code = strings.ToUpper(code[:8])

		// Format as XXXX-XXXX for readability
		codes[i] = code[:4] + "-" + code[4:]
	}

	return codes, nil
}

// GetMFAMethods returns available MFA methods
func (s *MFAService) GetMFAMethods() ([]models.MFAMethod, error) {
	query := `
		SELECT id, name, display_name, description, is_enabled, created_at, updated_at
		FROM mfa_methods
		WHERE is_enabled = true
		ORDER BY name
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query MFA methods: %w", err)
	}
	defer rows.Close()

	var methods []models.MFAMethod
	for rows.Next() {
		var method models.MFAMethod
		err := rows.Scan(
			&method.ID, &method.Name, &method.DisplayName,
			&method.Description, &method.IsEnabled,
			&method.CreatedAt, &method.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan MFA method: %w", err)
		}
		methods = append(methods, method)
	}

	return methods, rows.Err()
}

// GetMFAAttempts returns recent MFA attempts for a user
func (s *MFAService) GetMFAAttempts(userID int64, limit int) ([]models.MFAAttempt, error) {
	query := `
		SELECT id, user_id, mfa_method, attempt_type, success, 
		       ip_address, user_agent, error_message, created_at
		FROM mfa_attempts
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := s.db.Query(query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query MFA attempts: %w", err)
	}
	defer rows.Close()

	var attempts []models.MFAAttempt
	for rows.Next() {
		var attempt models.MFAAttempt
		err := rows.Scan(
			&attempt.ID, &attempt.UserID, &attempt.MFAMethod,
			&attempt.AttemptType, &attempt.Success, &attempt.IPAddress,
			&attempt.UserAgent, &attempt.ErrorMessage, &attempt.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan MFA attempt: %w", err)
		}
		attempts = append(attempts, attempt)
	}

	return attempts, rows.Err()
}
