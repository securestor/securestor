package models

import (
	"time"
)

// MFAMethod represents a supported MFA method
type MFAMethod struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	Description string    `json:"description"`
	IsEnabled   bool      `json:"is_enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// UserMFASettings represents a user's MFA configuration
type UserMFASettings struct {
	ID                     int64      `json:"id"`
	UserID                 int64      `json:"user_id"`
	IsMFAEnabled           bool       `json:"is_mfa_enabled"`
	BackupCodesGeneratedAt *time.Time `json:"backup_codes_generated_at,omitempty"`
	BackupCodesUsedCount   int        `json:"backup_codes_used_count"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`

	// Populated from joins
	User          *User                `json:"user,omitempty"`
	TOTPSecret    *UserTOTPSecret      `json:"totp_secret,omitempty"`
	WebAuthnCreds []WebAuthnCredential `json:"webauthn_credentials,omitempty"`
}

// UserTOTPSecret represents a user's TOTP secret
type UserTOTPSecret struct {
	ID              int64      `json:"id"`
	UserID          int64      `json:"user_id"`
	SecretEncrypted string     `json:"-"` // Don't expose in JSON
	IsVerified      bool       `json:"is_verified"`
	BackupCodes     []string   `json:"-"` // Don't expose in JSON
	CreatedAt       time.Time  `json:"created_at"`
	VerifiedAt      *time.Time `json:"verified_at,omitempty"`
	LastUsedAt      *time.Time `json:"last_used_at,omitempty"`

	// Populated from joins
	User *User `json:"user,omitempty"`
}

// WebAuthnCredential represents a WebAuthn/FIDO2 credential
type WebAuthnCredential struct {
	ID                string     `json:"id"`
	UserID            int64      `json:"user_id"`
	CredentialID      string     `json:"credential_id"`
	PublicKey         string     `json:"-"` // Don't expose in JSON
	AuthenticatorData string     `json:"-"` // Don't expose in JSON
	SignCount         int64      `json:"sign_count"`
	Name              string     `json:"name"`
	DeviceType        string     `json:"device_type"` // 'platform' or 'cross-platform'
	IsActive          bool       `json:"is_active"`
	CreatedAt         time.Time  `json:"created_at"`
	LastUsedAt        *time.Time `json:"last_used_at,omitempty"`

	// Populated from joins
	User *User `json:"user,omitempty"`
}

// MFAAttempt represents an MFA authentication attempt
type MFAAttempt struct {
	ID           int64     `json:"id"`
	UserID       int64     `json:"user_id"`
	MFAMethod    string    `json:"mfa_method"`
	AttemptType  string    `json:"attempt_type"` // 'verify', 'setup', 'recovery'
	Success      bool      `json:"success"`
	IPAddress    *string   `json:"ip_address,omitempty"`
	UserAgent    *string   `json:"user_agent,omitempty"`
	ErrorMessage *string   `json:"error_message,omitempty"`
	CreatedAt    time.Time `json:"created_at"`

	// Populated from joins
	User *User `json:"user,omitempty"`
}

// MFARecoveryCode represents a recovery code for MFA
type MFARecoveryCode struct {
	ID        int64      `json:"id"`
	UserID    int64      `json:"user_id"`
	CodeHash  string     `json:"-"` // Don't expose in JSON
	IsUsed    bool       `json:"is_used"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	UsedIP    *string    `json:"used_ip,omitempty"`
	CreatedAt time.Time  `json:"created_at"`

	// For display purposes only (not stored)
	Code string `json:"code,omitempty"`

	// Populated from joins
	User *User `json:"user,omitempty"`
}

// UserMFAStatus represents a comprehensive view of user's MFA status
type UserMFAStatus struct {
	UserID             int64      `json:"user_id"`
	Email              string     `json:"email"`
	Username           string     `json:"username"`
	MFARequired        bool       `json:"mfa_required"`
	MFAEnforcedAt      *time.Time `json:"mfa_enforced_at,omitempty"`
	IsMFAEnabled       bool       `json:"is_mfa_enabled"`
	TOTPVerified       *bool      `json:"totp_verified,omitempty"`
	WebAuthnCount      int        `json:"webauthn_count"`
	RecoveryCodesCount int        `json:"recovery_codes_count"`
	MFASetupAt         *time.Time `json:"mfa_setup_at,omitempty"`
	MFAUpdatedAt       *time.Time `json:"mfa_updated_at,omitempty"`
}

// MFASetupRequest represents a request to setup MFA
type MFASetupRequest struct {
	Method string `json:"method" binding:"required"` // 'totp', 'webauthn'
}

// TOTPSetupResponse represents the response for TOTP setup
type TOTPSetupResponse struct {
	Secret      string   `json:"secret"`
	QRCodeURL   string   `json:"qr_code_url"`
	BackupCodes []string `json:"backup_codes"`
}

// MFAVerifyRequest represents a request to verify MFA
type MFAVerifyRequest struct {
	Method string `json:"method" binding:"required"` // 'totp', 'webauthn', 'recovery'
	Code   string `json:"code,omitempty"`            // For TOTP and recovery codes
	Token  string `json:"token,omitempty"`           // For WebAuthn
}

// WebAuthnRegistrationOptions represents WebAuthn registration options
type WebAuthnRegistrationOptions struct {
	Challenge              string                `json:"challenge"`
	RelyingParty           WebAuthnRelyingParty  `json:"rp"`
	User                   WebAuthnUser          `json:"user"`
	PublicKeyCredParams    []WebAuthnCredParam   `json:"pubKeyCredParams"`
	AuthenticatorSelection WebAuthnAuthSelection `json:"authenticatorSelection"`
	Timeout                int                   `json:"timeout"`
	Attestation            string                `json:"attestation"`
}

// WebAuthnRelyingParty represents the relying party information
type WebAuthnRelyingParty struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// WebAuthnUser represents the user information for WebAuthn
type WebAuthnUser struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

// WebAuthnCredParam represents a credential parameter
type WebAuthnCredParam struct {
	Type string `json:"type"`
	Alg  int    `json:"alg"`
}

// WebAuthnAuthSelection represents authenticator selection criteria
type WebAuthnAuthSelection struct {
	AuthenticatorAttachment string `json:"authenticatorAttachment,omitempty"`
	RequireResidentKey      bool   `json:"requireResidentKey"`
	UserVerification        string `json:"userVerification"`
}

// WebAuthnRegistrationResponse represents a WebAuthn registration response
type WebAuthnRegistrationResponse struct {
	ID         string                        `json:"id"`
	RawID      string                        `json:"rawId"`
	Type       string                        `json:"type"`
	Response   WebAuthnAuthenticatorResponse `json:"response"`
	DeviceName string                        `json:"deviceName,omitempty"`
}

// WebAuthnAuthenticatorResponse represents the authenticator response
type WebAuthnAuthenticatorResponse struct {
	AttestationObject string `json:"attestationObject"`
	ClientDataJSON    string `json:"clientDataJSON"`
}

// WebAuthnAssertionOptions represents WebAuthn assertion options
type WebAuthnAssertionOptions struct {
	Challenge        string                         `json:"challenge"`
	Timeout          int                            `json:"timeout"`
	RelyingPartyID   string                         `json:"rpId"`
	AllowCredentials []WebAuthnCredentialDescriptor `json:"allowCredentials"`
	UserVerification string                         `json:"userVerification"`
}

// WebAuthnCredentialDescriptor represents a credential descriptor
type WebAuthnCredentialDescriptor struct {
	Type       string   `json:"type"`
	ID         string   `json:"id"`
	Transports []string `json:"transports,omitempty"`
}

// WebAuthnAssertionResponse represents a WebAuthn assertion response
type WebAuthnAssertionResponse struct {
	ID       string                                 `json:"id"`
	RawID    string                                 `json:"rawId"`
	Type     string                                 `json:"type"`
	Response WebAuthnAuthenticatorAssertionResponse `json:"response"`
}

// WebAuthnAuthenticatorAssertionResponse represents the authenticator assertion response
type WebAuthnAuthenticatorAssertionResponse struct {
	AuthenticatorData string `json:"authenticatorData"`
	ClientDataJSON    string `json:"clientDataJSON"`
	Signature         string `json:"signature"`
	UserHandle        string `json:"userHandle,omitempty"`
}

// MFARequiredError represents an error when MFA is required
type MFARequiredError struct {
	Message          string   `json:"message"`
	RequiredMethods  []string `json:"required_methods"`
	AvailableMethods []string `json:"available_methods"`
	ChallengeToken   string   `json:"challenge_token,omitempty"`
}

func (e MFARequiredError) Error() string {
	return e.Message
}

// HasMFAMethod checks if the user has a specific MFA method configured
func (s *UserMFAStatus) HasMFAMethod(method string) bool {
	switch method {
	case "totp":
		return s.TOTPVerified != nil && *s.TOTPVerified
	case "webauthn":
		return s.WebAuthnCount > 0
	case "recovery":
		return s.RecoveryCodesCount > 0
	default:
		return false
	}
}

// GetAvailableMethods returns the list of available MFA methods for the user
func (s *UserMFAStatus) GetAvailableMethods() []string {
	var methods []string

	if s.HasMFAMethod("totp") {
		methods = append(methods, "totp")
	}
	if s.HasMFAMethod("webauthn") {
		methods = append(methods, "webauthn")
	}
	if s.HasMFAMethod("recovery") {
		methods = append(methods, "recovery")
	}

	return methods
}

// IsFullyConfigured checks if the user has MFA fully configured
func (s *UserMFAStatus) IsFullyConfigured() bool {
	return s.IsMFAEnabled && (s.HasMFAMethod("totp") || s.HasMFAMethod("webauthn"))
}
