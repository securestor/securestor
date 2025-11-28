package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// User represents a user in the RBAC system
type User struct {
	ID              uuid.UUID  `json:"id"`            // UUID primary key for users table
	Sub             *string    `json:"sub,omitempty"` // OIDC subject identifier - nullable for local auth
	Username        string     `json:"username"`
	Email           string     `json:"email"`
	PasswordHash    *string    `json:"password_hash,omitempty"` // Bcrypt password hash - nullable for OIDC users
	FirstName       *string    `json:"first_name,omitempty"`
	LastName        *string    `json:"last_name,omitempty"`
	DisplayName     *string    `json:"display_name,omitempty"`
	IsActive        bool       `json:"is_active"`
	IsEmailVerified bool       `json:"is_email_verified"`
	LastLoginAt     *time.Time `json:"last_login_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	TenantID        uuid.UUID  `json:"tenant_id"` // Required: multi-tenancy enforcement

	// Populated from joins
	Roles       []Role       `json:"roles,omitempty"`
	Permissions []Permission `json:"permissions,omitempty"`
}

// Scan implements sql.Scanner interface for UUID
func (u *User) Scan(value interface{}) error {
	return nil // UUID scanning handled by individual fields
}

// Value implements driver.Valuer interface for UUID
func (u User) Value() (driver.Value, error) {
	return json.Marshal(u)
}

// IsAdmin checks if the user has admin role
func (u *User) IsAdmin() bool {
	for _, role := range u.Roles {
		if role.Name == "admin" {
			return true
		}
	}
	return false
}

// Role represents a role in the RBAC system
type Role struct {
	ID           uuid.UUID `json:"id"`
	TenantID     uuid.UUID `json:"tenant_id"`
	Name         string    `json:"name"`
	DisplayName  *string   `json:"display_name,omitempty"`
	Description  *string   `json:"description,omitempty"`
	IsSystemRole bool      `json:"is_system_role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// Populated from joins
	Permissions []Permission `json:"permissions,omitempty"`
}

// Permission represents a granular permission in the RBAC system
type Permission struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Resource    string    `json:"resource"`
	Action      string    `json:"action"`
	Description *string   `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// OAuth2Client represents an OAuth2 client for machine-to-machine authentication
type OAuth2Client struct {
	ID               int64      `json:"id"`
	ClientID         string     `json:"client_id"`
	ClientSecretHash string     `json:"-"` // Never expose the hash
	Name             string     `json:"name"`
	Description      *string    `json:"description,omitempty"`
	GrantTypes       []string   `json:"grant_types"`
	Scopes           []string   `json:"scopes"`
	IsActive         bool       `json:"is_active"`
	ExpiresAt        *time.Time `json:"expires_at,omitempty"`
	CreatedBy        int64      `json:"created_by"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	LastUsedAt       *time.Time `json:"last_used_at,omitempty"`

	// Populated from joins
	Creator *User `json:"creator,omitempty"`
}

// APIKey represents an API key for programmatic access
type APIKey struct {
	ID          uuid.UUID  `json:"id"`     // Use key_id as primary identifier
	KeyID       string     `json:"key_id"` // Public identifier
	KeyHash     string     `json:"-"`      // Never expose the hash
	Name        string     `json:"name"`
	Description *string    `json:"description,omitempty"`
	Scopes      []string   `json:"scopes"`
	UserID      uuid.UUID  `json:"user_id"`
	IsActive    bool       `json:"is_active"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	// Populated from joins
	User *User `json:"user,omitempty"`
}

// RefreshTokenStore represents stored refresh tokens with rotation support
type RefreshTokenStore struct {
	ID            int64      `json:"id"`
	TokenHash     string     `json:"-"` // Never expose the hash
	UserID        int64      `json:"user_id"`
	SessionID     string     `json:"session_id"`
	ClientID      *string    `json:"client_id,omitempty"` // For OAuth2 clients
	ExpiresAt     time.Time  `json:"expires_at"`
	IsRevoked     bool       `json:"is_revoked"`
	RevokedAt     *time.Time `json:"revoked_at,omitempty"`
	ParentTokenID *int64     `json:"parent_token_id,omitempty"` // For token rotation
	CreatedAt     time.Time  `json:"created_at"`

	// Populated from joins
	User   *User         `json:"user,omitempty"`
	Client *OAuth2Client `json:"client,omitempty"`
}

// UserRole represents the assignment of a role to a user
type UserRole struct {
	ID         int64      `json:"id"`
	UserID     int64      `json:"user_id"`
	RoleID     int64      `json:"role_id"`
	AssignedBy *int64     `json:"assigned_by,omitempty"`
	AssignedAt time.Time  `json:"assigned_at"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`

	// Populated from joins
	User *User `json:"user,omitempty"`
	Role *Role `json:"role,omitempty"`
}

// UserSession represents a user session with OIDC tokens
type UserSession struct {
	ID               string    `json:"id"`
	UserID           uuid.UUID `json:"user_id"`
	AccessToken      string    `json:"-"` // Don't expose in JSON
	RefreshToken     string    `json:"-"` // Don't expose in JSON
	IDToken          string    `json:"-"` // Don't expose in JSON
	AccessTokenHash  *string   `json:"-"` // Don't expose in JSON - for database storage
	RefreshTokenHash *string   `json:"-"` // Don't expose in JSON - for database storage
	IDTokenHash      *string   `json:"-"` // Don't expose in JSON - for database storage
	ExpiresAt        time.Time `json:"expires_at"`
	CreatedAt        time.Time `json:"created_at"`
	LastAccessedAt   time.Time `json:"last_accessed_at"`
	IPAddress        *string   `json:"ip_address,omitempty"`
	UserAgent        *string   `json:"user_agent,omitempty"`

	// Populated from joins
	User *User `json:"user,omitempty"`
}

// AuthContext holds the authentication context for a request
type AuthContext struct {
	UserID      string              `json:"user_id"`
	Username    string              `json:"username"`
	Email       string              `json:"email"`
	User        *User               `json:"user"`
	SessionID   string              `json:"session_id"`
	Roles       []string            `json:"roles"`
	Permissions map[string][]string `json:"permissions"` // resource -> actions
	Claims      *TokenClaims        `json:"claims"`
	IsAdmin     bool                `json:"is_admin"`
	ExpiresAt   time.Time           `json:"expires_at"`

	// OAuth2/M2M specific fields
	ClientID *string  `json:"client_id,omitempty"` // For OAuth2 client credentials
	Scopes   []string `json:"scopes,omitempty"`    // For API keys and OAuth2 clients
	AuthType string   `json:"auth_type"`           // "user", "client_credentials", "api_key"
}

// HasPermission checks if the user has a specific permission
func (ctx *AuthContext) HasPermission(resource, action string) bool {
	// Admin has all permissions
	if ctx.IsAdmin {
		return true
	}

	// Check specific permission
	if actions, exists := ctx.Permissions[resource]; exists {
		for _, a := range actions {
			if a == action || a == "admin" {
				return true
			}
		}
	}

	return false
}

// HasRole checks if the user has a specific role
func (ctx *AuthContext) HasRole(roleName string) bool {
	for _, role := range ctx.Roles {
		if role == roleName {
			return true
		}
	}
	return false
}

// HasAnyRole checks if the user has any of the specified roles
func (ctx *AuthContext) HasAnyRole(roleNames ...string) bool {
	for _, roleName := range roleNames {
		if ctx.HasRole(roleName) {
			return true
		}
	}
	return false
}

// HasScope checks if the context has a specific scope (for API keys and OAuth2 clients)
func (ctx *AuthContext) HasScope(scope string) bool {
	for _, s := range ctx.Scopes {
		if s == scope || s == "*" {
			return true
		}
	}
	return false
}

// HasAnyScope checks if the context has any of the specified scopes
func (ctx *AuthContext) HasAnyScope(scopes ...string) bool {
	for _, scope := range scopes {
		if ctx.HasScope(scope) {
			return true
		}
	}
	return false
}

// IsM2M checks if this is a machine-to-machine authentication
func (ctx *AuthContext) IsM2M() bool {
	return ctx.AuthType == "client_credentials" || ctx.AuthType == "api_key"
}

// IsUserAuth checks if this is a user authentication (OIDC)
func (ctx *AuthContext) IsUserAuth() bool {
	return ctx.AuthType == "user"
}

// OIDCConfig holds OIDC configuration
type OIDCConfig struct {
	IssuerURL     string   `json:"issuer_url"`
	ClientID      string   `json:"client_id"`
	ClientSecret  string   `json:"client_secret"`
	RedirectURL   string   `json:"redirect_url"`
	PostLogoutURL string   `json:"post_logout_url"`
	Scopes        []string `json:"scopes"`
}

// Tenant represents a tenant in the multi-tenant system
type Tenant struct {
	ID          int64     `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Slug        string    `json:"slug" db:"slug"`
	Description *string   `json:"description,omitempty" db:"description"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	Settings    string    `json:"settings" db:"settings"` // JSONB stored as string, will be parsed as needed
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
	CreatedBy   *int64    `json:"created_by,omitempty" db:"created_by"`
}

// TokenClaims represents the claims in an OIDC token
type TokenClaims struct {
	Subject           string `json:"sub"`
	Email             string `json:"email"`
	EmailVerified     bool   `json:"email_verified"`
	PreferredUsername string `json:"preferred_username"`
	Name              string `json:"name"`
	GivenName         string `json:"given_name"`
	FamilyName        string `json:"family_name"`
	RealmAccess       struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`
	Roles        []string `json:"roles"`
	Realm        string   `json:"realm"`
	Iss          string   `json:"iss"`
	Aud          []string `json:"aud"`
	Exp          int64    `json:"exp"`
	Iat          int64    `json:"iat"`
	AuthTime     int64    `json:"auth_time"`
	SessionState string   `json:"session_state"`
}

// UserGroupInfo represents user group and role mappings from SCIM
type UserGroupInfo struct {
	UserID uuid.UUID `json:"user_id"`
	Groups []string  `json:"groups"`
	Roles  []string  `json:"roles"`
}
