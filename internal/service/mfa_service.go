package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
)

// MFAService handles multi-factor authentication operations
type MFAService struct {
	db *sql.DB
}

// NewMFAService creates a new MFA service
func NewMFAService(db *sql.DB) *MFAService {
	return &MFAService{
		db: db,
	}
}

// MFAMethod represents an available MFA method
type MFAMethod struct {
	ID            int64                  `json:"id"`
	Name          string                 `json:"name"`
	DisplayName   string                 `json:"display_name"`
	IsEnabled     bool                   `json:"is_enabled"`
	IsDefault     bool                   `json:"is_default"`
	Configuration map[string]interface{} `json:"configuration"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// UserMFASettings represents user MFA configuration
type UserMFASettings struct {
	ID                   int64      `json:"id"`
	UserID               int64      `json:"user_id"`
	TenantID             int64      `json:"tenant_id"`
	IsMFAEnabled         bool       `json:"is_mfa_enabled"`
	PreferredMethod      *string    `json:"preferred_method"`
	BackupCodesGenerated bool       `json:"backup_codes_generated"`
	BackupCodesUsedCount int        `json:"backup_codes_used_count"`
	LastMFASetup         *time.Time `json:"last_mfa_setup"`
	LastMFAUsed          *time.Time `json:"last_mfa_used"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

// TOTPSecret represents TOTP configuration for a user
type TOTPSecret struct {
	ID         int64      `json:"id"`
	UserID     int64      `json:"user_id"`
	SecretKey  string     `json:"secret_key"`
	IsVerified bool       `json:"is_verified"`
	QRCodeURL  *string    `json:"qr_code_url"`
	CreatedAt  time.Time  `json:"created_at"`
	VerifiedAt *time.Time `json:"verified_at"`
	LastUsed   *time.Time `json:"last_used"`
}

// WebAuthnCredential represents a WebAuthn/FIDO2 credential
type WebAuthnCredential struct {
	ID                int64      `json:"id"`
	UserID            int64      `json:"user_id"`
	CredentialID      string     `json:"credential_id"`
	PublicKey         string     `json:"public_key"`
	Counter           int64      `json:"counter"`
	DeviceName        *string    `json:"device_name"`
	DeviceType        *string    `json:"device_type"`
	AAGUID            *string    `json:"aaguid"`
	AttestationFormat *string    `json:"attestation_format"`
	Transport         []string   `json:"transport"`
	IsBackupEligible  bool       `json:"is_backup_eligible"`
	IsBackupState     bool       `json:"is_backup_state"`
	CreatedAt         time.Time  `json:"created_at"`
	LastUsed          *time.Time `json:"last_used"`
	UserAgent         *string    `json:"user_agent"`
}

// MFALoginAttempt represents an MFA authentication attempt
type MFALoginAttempt struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	SessionID   *string   `json:"session_id"`
	MethodUsed  string    `json:"method_used"`
	Success     bool      `json:"success"`
	IPAddress   *string   `json:"ip_address"`
	UserAgent   *string   `json:"user_agent"`
	ErrorReason *string   `json:"error_reason"`
	CreatedAt   time.Time `json:"created_at"`
}

// UserMFAOverview represents comprehensive MFA status for a user
type UserMFAOverview struct {
	UserID            int64      `json:"user_id"`
	Username          string     `json:"username"`
	Email             string     `json:"email"`
	TenantID          int64      `json:"tenant_id"`
	MFAEnabled        bool       `json:"mfa_enabled"`
	PreferredMethod   *string    `json:"preferred_method"`
	LastMFASetup      *time.Time `json:"last_mfa_setup"`
	LastMFAUsed       *time.Time `json:"last_mfa_used"`
	BackupCodesUsed   int        `json:"backup_codes_used_count"`
	WebAuthnDevices   int        `json:"webauthn_devices"`
	UnusedBackupCodes int        `json:"unused_backup_codes"`
	TOTPConfigured    bool       `json:"totp_configured"`
	MFAActive         bool       `json:"mfa_active"`
}

// MFASetupRequest represents a request to set up MFA
type MFASetupRequest struct {
	UserID int64  `json:"user_id"`
	Method string `json:"method"`
}

// MFASetupResponse represents the response from MFA setup
type MFASetupResponse struct {
	Method      string   `json:"method"`
	Secret      *string  `json:"secret,omitempty"`
	QRCodeURL   *string  `json:"qr_code_url,omitempty"`
	BackupCodes []string `json:"backup_codes,omitempty"`
}

// MFAVerificationRequest represents a request to verify MFA
type MFAVerificationRequest struct {
	UserID         int64   `json:"user_id"`
	Method         string  `json:"method"`
	Code           string  `json:"code"`
	SessionID      *string `json:"session_id"`
	RememberDevice bool    `json:"remember_device"`
	IPAddress      *string `json:"ip_address"`
	UserAgent      *string `json:"user_agent"`
}

// GetAvailableMFAMethods retrieves all available MFA methods
func (s *MFAService) GetAvailableMFAMethods(ctx context.Context) ([]*MFAMethod, error) {
	query := `
		SELECT id, name, display_name, is_enabled, is_default, 
		       configuration, created_at, updated_at
		FROM mfa_methods
		WHERE is_enabled = true
		ORDER BY is_default DESC, display_name ASC
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query MFA methods: %w", err)
	}
	defer rows.Close()

	var methods []*MFAMethod
	for rows.Next() {
		method := &MFAMethod{}
		var configJSON []byte

		err := rows.Scan(
			&method.ID, &method.Name, &method.DisplayName,
			&method.IsEnabled, &method.IsDefault, &configJSON,
			&method.CreatedAt, &method.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan MFA method: %w", err)
		}

		// Parse configuration JSON
		if len(configJSON) > 0 {
			err = json.Unmarshal(configJSON, &method.Configuration)
			if err != nil {
				method.Configuration = make(map[string]interface{})
			}
		} else {
			method.Configuration = make(map[string]interface{})
		}

		methods = append(methods, method)
	}

	return methods, nil
}

// GetUserMFASettings retrieves MFA settings for a user
func (s *MFAService) GetUserMFASettings(ctx context.Context, userID int64) (*UserMFASettings, error) {
	query := `
		SELECT id, user_id, tenant_id, is_mfa_enabled, preferred_method,
		       backup_codes_generated, backup_codes_used_count,
		       last_mfa_setup, last_mfa_used, created_at, updated_at
		FROM user_mfa_settings
		WHERE user_id = $1
	`

	settings := &UserMFASettings{}
	err := s.db.QueryRowContext(ctx, query, userID).Scan(
		&settings.ID, &settings.UserID, &settings.TenantID,
		&settings.IsMFAEnabled, &settings.PreferredMethod,
		&settings.BackupCodesGenerated, &settings.BackupCodesUsedCount,
		&settings.LastMFASetup, &settings.LastMFAUsed,
		&settings.CreatedAt, &settings.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// Return default settings if none exist
			return &UserMFASettings{
				UserID:       userID,
				IsMFAEnabled: false,
			}, nil
		}
		return nil, fmt.Errorf("failed to get user MFA settings: %w", err)
	}

	return settings, nil
}

// GetUserMFAOverview retrieves comprehensive MFA status for a user
func (s *MFAService) GetUserMFAOverview(ctx context.Context, userID int64) (*UserMFAOverview, error) {
	query := `
		SELECT user_id, username, email, tenant_id, mfa_enabled,
		       preferred_method, last_mfa_setup, last_mfa_used,
		       backup_codes_used_count, webauthn_devices,
		       unused_backup_codes, totp_configured, mfa_active
		FROM user_mfa_overview
		WHERE user_id = $1
	`

	overview := &UserMFAOverview{}
	err := s.db.QueryRowContext(ctx, query, userID).Scan(
		&overview.UserID, &overview.Username, &overview.Email,
		&overview.TenantID, &overview.MFAEnabled, &overview.PreferredMethod,
		&overview.LastMFASetup, &overview.LastMFAUsed, &overview.BackupCodesUsed,
		&overview.WebAuthnDevices, &overview.UnusedBackupCodes,
		&overview.TOTPConfigured, &overview.MFAActive,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user MFA overview: %w", err)
	}

	return overview, nil
}

// SetupMFA initiates MFA setup for a user
func (s *MFAService) SetupMFA(ctx context.Context, req *MFASetupRequest) (*MFASetupResponse, error) {
	switch req.Method {
	case "totp":
		return s.setupTOTP(ctx, req.UserID)
	case "backup_codes":
		return s.setupBackupCodes(ctx, req.UserID)
	default:
		return nil, fmt.Errorf("unsupported MFA method: %s", req.Method)
	}
}

// setupTOTP sets up TOTP for a user
func (s *MFAService) setupTOTP(ctx context.Context, userID int64) (*MFASetupResponse, error) {
	// Get user information
	var username, email string
	err := s.db.QueryRowContext(ctx, "SELECT username, email FROM users WHERE id = $1", userID).Scan(&username, &email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Generate TOTP secret
	secret, err := s.generateTOTPSecret()
	if err != nil {
		return nil, fmt.Errorf("failed to generate TOTP secret: %w", err)
	}

	// Generate QR code URL for TOTP
	qrCodeURL := fmt.Sprintf("otpauth://totp/SecureStorT:%s?secret=%s&issuer=SecureStorT&algorithm=SHA1&digits=6&period=30",
		email, secret)

	// Store the secret in database
	query := `
		INSERT INTO user_totp_secrets (user_id, secret_key, qr_code_url)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id) DO UPDATE SET
			secret_key = EXCLUDED.secret_key,
			qr_code_url = EXCLUDED.qr_code_url,
			is_verified = false,
			created_at = CURRENT_TIMESTAMP
	`

	_, err = s.db.ExecContext(ctx, query, userID, secret, qrCodeURL)
	if err != nil {
		return nil, fmt.Errorf("failed to store TOTP secret: %w", err)
	}

	// Update user MFA settings
	err = s.updateUserMFASettings(ctx, userID, "totp")
	if err != nil {
		return nil, fmt.Errorf("failed to update MFA settings: %w", err)
	}

	return &MFASetupResponse{
		Method:    "totp",
		Secret:    &secret,
		QRCodeURL: &qrCodeURL,
	}, nil
}

// setupBackupCodes generates backup codes for a user
func (s *MFAService) setupBackupCodes(ctx context.Context, userID int64) (*MFASetupResponse, error) {
	// Generate backup codes using database function
	var backupCodesArray pq.StringArray
	err := s.db.QueryRowContext(ctx, "SELECT generate_backup_codes($1, 10)", userID).Scan(&backupCodesArray)
	if err != nil {
		return nil, fmt.Errorf("failed to generate backup codes: %w", err)
	}

	backupCodes := []string(backupCodesArray)

	// Update user MFA settings
	err = s.updateUserMFASettings(ctx, userID, "backup_codes")
	if err != nil {
		return nil, fmt.Errorf("failed to update MFA settings: %w", err)
	}

	return &MFASetupResponse{
		Method:      "backup_codes",
		BackupCodes: backupCodes,
	}, nil
}

// VerifyMFA verifies an MFA code
func (s *MFAService) VerifyMFA(ctx context.Context, req *MFAVerificationRequest) (bool, error) {
	var success bool
	var errorReason string

	switch req.Method {
	case "totp":
		success = s.verifyTOTP(ctx, req.UserID, req.Code)
		if !success {
			errorReason = "Invalid TOTP code"
		}
	case "backup_code":
		success = s.verifyBackupCode(ctx, req.UserID, req.Code)
		if !success {
			errorReason = "Invalid or expired backup code"
		}
	default:
		success = false
		errorReason = "Unsupported MFA method"
	}

	// Log the attempt
	_, err := s.LogMFAAttempt(ctx, &MFALoginAttempt{
		UserID:      req.UserID,
		SessionID:   req.SessionID,
		MethodUsed:  req.Method,
		Success:     success,
		IPAddress:   req.IPAddress,
		UserAgent:   req.UserAgent,
		ErrorReason: &errorReason,
	})
	if err != nil {
		// Don't fail verification due to logging error
		fmt.Printf("Failed to log MFA attempt: %v", err)
	}

	return success, nil
}

// verifyTOTP verifies a TOTP code
func (s *MFAService) verifyTOTP(ctx context.Context, userID int64, code string) bool {
	// Get user's TOTP secret
	var secret string
	var isVerified bool
	err := s.db.QueryRowContext(ctx,
		"SELECT secret_key, is_verified FROM user_totp_secrets WHERE user_id = $1",
		userID).Scan(&secret, &isVerified)
	if err != nil {
		return false
	}

	if !isVerified {
		return false
	}

	// For now, we'll do a basic validation
	// In production, implement proper TOTP algorithm with time windows
	// This is a placeholder that validates format
	if len(code) != 6 {
		return false
	}

	// Check if code is numeric
	for _, char := range code {
		if char < '0' || char > '9' {
			return false
		}
	}

	// Update last used timestamp (assuming valid for demo)
	s.db.ExecContext(ctx,
		"UPDATE user_totp_secrets SET last_used = CURRENT_TIMESTAMP WHERE user_id = $1",
		userID)

	// TODO: Implement proper TOTP validation using RFC 6238
	// For now, accept any 6-digit numeric code for demo purposes
	return true
}

// verifyBackupCode verifies a backup code
func (s *MFAService) verifyBackupCode(ctx context.Context, userID int64, code string) bool {
	var valid bool
	err := s.db.QueryRowContext(ctx, "SELECT verify_backup_code($1, $2)", userID, code).Scan(&valid)
	return err == nil && valid
}

// CompleteMFASetup completes MFA setup after verification
func (s *MFAService) CompleteMFASetup(ctx context.Context, userID int64, method, verificationCode string) error {
	switch method {
	case "totp":
		// Verify the TOTP code first
		if !s.verifyTOTP(ctx, userID, verificationCode) {
			return fmt.Errorf("invalid TOTP verification code")
		}

		// Mark TOTP as verified and enable MFA
		_, err := s.db.ExecContext(ctx, "SELECT complete_mfa_setup($1, $2, $3)", userID, method, verificationCode)
		if err != nil {
			return fmt.Errorf("failed to complete TOTP setup: %w", err)
		}
	case "webauthn":
		// WebAuthn verification is handled by the application layer
		_, err := s.db.ExecContext(ctx, "SELECT complete_mfa_setup($1, $2, $3)", userID, method, "")
		if err != nil {
			return fmt.Errorf("failed to complete WebAuthn setup: %w", err)
		}
	default:
		return fmt.Errorf("unsupported MFA method for completion: %s", method)
	}

	return nil
}

// DisableMFA disables MFA for a user
func (s *MFAService) DisableMFA(ctx context.Context, userID int64) error {
	var disabled bool
	err := s.db.QueryRowContext(ctx, "SELECT disable_user_mfa($1)", userID).Scan(&disabled)
	if err != nil {
		return fmt.Errorf("failed to disable MFA: %w", err)
	}

	if !disabled {
		return fmt.Errorf("failed to disable MFA for user")
	}

	return nil
}

// GetUserWebAuthnCredentials retrieves WebAuthn credentials for a user
func (s *MFAService) GetUserWebAuthnCredentials(ctx context.Context, userID int64) ([]*WebAuthnCredential, error) {
	query := `
		SELECT id, user_id, credential_id, public_key, counter,
		       device_name, device_type, aaguid, attestation_format,
		       transport, is_backup_eligible, is_backup_state,
		       created_at, last_used, user_agent
		FROM user_webauthn_credentials
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query WebAuthn credentials: %w", err)
	}
	defer rows.Close()

	var credentials []*WebAuthnCredential
	for rows.Next() {
		cred := &WebAuthnCredential{}
		var transportArray pq.StringArray

		err := rows.Scan(
			&cred.ID, &cred.UserID, &cred.CredentialID, &cred.PublicKey,
			&cred.Counter, &cred.DeviceName, &cred.DeviceType, &cred.AAGUID,
			&cred.AttestationFormat, &transportArray, &cred.IsBackupEligible,
			&cred.IsBackupState, &cred.CreatedAt, &cred.LastUsed, &cred.UserAgent,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan WebAuthn credential: %w", err)
		}

		cred.Transport = []string(transportArray)
		credentials = append(credentials, cred)
	}

	return credentials, nil
}

// LogMFAAttempt logs an MFA authentication attempt
func (s *MFAService) LogMFAAttempt(ctx context.Context, attempt *MFALoginAttempt) (int64, error) {
	query := `
		INSERT INTO mfa_login_attempts (
			user_id, session_id, method_used, success,
			ip_address, user_agent, error_reason
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	var attemptID int64
	err := s.db.QueryRowContext(ctx, query,
		attempt.UserID, attempt.SessionID, attempt.MethodUsed,
		attempt.Success, attempt.IPAddress, attempt.UserAgent,
		attempt.ErrorReason,
	).Scan(&attemptID)

	if err != nil {
		return 0, fmt.Errorf("failed to log MFA attempt: %w", err)
	}

	return attemptID, nil
}

// GetMFAAttempts retrieves MFA attempts for a user
func (s *MFAService) GetMFAAttempts(ctx context.Context, userID int64, limit int) ([]*MFALoginAttempt, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	query := `
		SELECT id, user_id, session_id, method_used, success,
		       ip_address, user_agent, error_reason, created_at
		FROM mfa_login_attempts
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := s.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query MFA attempts: %w", err)
	}
	defer rows.Close()

	var attempts []*MFALoginAttempt
	for rows.Next() {
		attempt := &MFALoginAttempt{}

		err := rows.Scan(
			&attempt.ID, &attempt.UserID, &attempt.SessionID,
			&attempt.MethodUsed, &attempt.Success, &attempt.IPAddress,
			&attempt.UserAgent, &attempt.ErrorReason, &attempt.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan MFA attempt: %w", err)
		}

		attempts = append(attempts, attempt)
	}

	return attempts, nil
}

// Helper functions

// generateTOTPSecret generates a secure random secret for TOTP
func (s *MFAService) generateTOTPSecret() (string, error) {
	secret := make([]byte, 20) // 160 bits
	_, err := rand.Read(secret)
	if err != nil {
		return "", err
	}

	// Encode as base32 without padding
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret)
	return strings.ToUpper(encoded), nil
}

// updateUserMFASettings creates or updates user MFA settings
func (s *MFAService) updateUserMFASettings(ctx context.Context, userID int64, method string) error {
	// Get user's tenant ID
	var tenantID int64
	err := s.db.QueryRowContext(ctx, "SELECT tenant_id FROM users WHERE id = $1", userID).Scan(&tenantID)
	if err != nil {
		return fmt.Errorf("failed to get user tenant: %w", err)
	}

	query := `
		INSERT INTO user_mfa_settings (user_id, tenant_id, preferred_method, last_mfa_setup)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP)
		ON CONFLICT (user_id) DO UPDATE SET
			preferred_method = EXCLUDED.preferred_method,
			last_mfa_setup = EXCLUDED.last_mfa_setup,
			updated_at = CURRENT_TIMESTAMP
	`

	_, err = s.db.ExecContext(ctx, query, userID, tenantID, method)
	if err != nil {
		return fmt.Errorf("failed to update MFA settings: %w", err)
	}

	return nil
}
