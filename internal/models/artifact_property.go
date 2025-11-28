package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ArtifactProperty represents a key-value property attached to an artifact
type ArtifactProperty struct {
	ID           uuid.UUID `json:"id" db:"id"`
	TenantID     uuid.UUID `json:"tenant_id" db:"tenant_id"`
	RepositoryID string    `json:"repository_id" db:"repository_id"`
	ArtifactID   string    `json:"artifact_id" db:"artifact_id"`

	// Property data
	Key       string `json:"key" db:"key"`
	Value     string `json:"value" db:"value"`
	ValueType string `json:"value_type" db:"value_type"` // string, number, boolean, json, array

	// Flags
	IsSensitive  bool `json:"is_sensitive" db:"is_sensitive"`
	IsSystem     bool `json:"is_system" db:"is_system"`
	IsMultiValue bool `json:"is_multi_value" db:"is_multi_value"`

	// Encryption metadata (for sensitive properties)
	EncryptedValue      *string `json:"encrypted_value,omitempty" db:"encrypted_value"`
	EncryptionKeyID     *string `json:"encryption_key_id,omitempty" db:"encryption_key_id"`
	EncryptionAlgorithm *string `json:"encryption_algorithm,omitempty" db:"encryption_algorithm"`
	Nonce               *string `json:"nonce,omitempty" db:"nonce"`

	// Audit fields
	CreatedBy *uuid.UUID `json:"created_by,omitempty" db:"created_by"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedBy *uuid.UUID `json:"updated_by,omitempty" db:"updated_by"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	Version   int        `json:"version" db:"version"`

	// Additional metadata
	Tags        StringArray `json:"tags,omitempty" db:"tags"`
	Description *string     `json:"description,omitempty" db:"description"`
}

// PropertyTemplate represents a pre-defined property template
type PropertyTemplate struct {
	ID          uuid.UUID       `json:"id" db:"id"`
	TenantID    uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	Name        string          `json:"name" db:"name"`
	Description *string         `json:"description,omitempty" db:"description"`
	Category    *string         `json:"category,omitempty" db:"category"`
	Properties  json.RawMessage `json:"properties" db:"properties"`
	IsSystem    bool            `json:"is_system" db:"is_system"`
	IsActive    bool            `json:"is_active" db:"is_active"`
	CreatedBy   *uuid.UUID      `json:"created_by,omitempty" db:"created_by"`
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at" db:"updated_at"`
}

// PropertyAuditLog represents an audit entry for property operations
type PropertyAuditLog struct {
	ID            int64      `json:"id" db:"id"`
	TenantID      uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	PropertyID    *uuid.UUID `json:"property_id,omitempty" db:"property_id"`
	ArtifactID    string     `json:"artifact_id" db:"artifact_id"`
	Action        string     `json:"action" db:"action"` // CREATE, UPDATE, DELETE, READ
	Key           string     `json:"key" db:"key"`
	OldValue      *string    `json:"old_value,omitempty" db:"old_value"`
	NewValue      *string    `json:"new_value,omitempty" db:"new_value"`
	UserID        *uuid.UUID `json:"user_id,omitempty" db:"user_id"`
	Username      *string    `json:"username,omitempty" db:"username"`
	IPAddress     *string    `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent     *string    `json:"user_agent,omitempty" db:"user_agent"`
	Timestamp     time.Time  `json:"timestamp" db:"timestamp"`
	CorrelationID *uuid.UUID `json:"correlation_id,omitempty" db:"correlation_id"`
	Metadata      *string    `json:"metadata,omitempty" db:"metadata"`
}

// PropertySearchResult represents a search result
type PropertySearchResult struct {
	ID           uuid.UUID `json:"id"`
	TenantID     uuid.UUID `json:"tenant_id"`
	RepositoryID string    `json:"repository_id"`
	ArtifactID   string    `json:"artifact_id"`
	Key          string    `json:"key"`
	Value        string    `json:"value"`
	ValueType    string    `json:"value_type"`
	IsSensitive  bool      `json:"is_sensitive"`
	IsSystem     bool      `json:"is_system"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// PropertyStatistics represents property statistics for a tenant
type PropertyStatistics struct {
	TenantID                uuid.UUID `json:"tenant_id" db:"tenant_id"`
	TotalProperties         int       `json:"total_properties" db:"total_properties"`
	ArtifactsWithProperties int       `json:"artifacts_with_properties" db:"artifacts_with_properties"`
	UniqueKeys              int       `json:"unique_keys" db:"unique_keys"`
	SensitiveProperties     int       `json:"sensitive_properties" db:"sensitive_properties"`
	SystemProperties        int       `json:"system_properties" db:"system_properties"`
	MultiValueProperties    int       `json:"multi_value_properties" db:"multi_value_properties"`
	LastPropertyAdded       time.Time `json:"last_property_added" db:"last_property_added"`
}

// CreatePropertyRequest represents a request to create a property
type CreatePropertyRequest struct {
	Key          string   `json:"key" validate:"required,propertykey"`
	Value        string   `json:"value" validate:"required,max=65535"`
	ValueType    string   `json:"value_type,omitempty"` // default: string
	IsSensitive  bool     `json:"is_sensitive,omitempty"`
	IsMultiValue bool     `json:"is_multi_value,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	Description  string   `json:"description,omitempty"`
}

// UpdatePropertyRequest represents a request to update a property
type UpdatePropertyRequest struct {
	Value       *string  `json:"value,omitempty"`
	ValueType   *string  `json:"value_type,omitempty"`
	IsSensitive *bool    `json:"is_sensitive,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Description *string  `json:"description,omitempty"`
}

// SearchPropertiesRequest represents a property search request
type SearchPropertiesRequest struct {
	Key           *string    `json:"key,omitempty"`
	Value         *string    `json:"value,omitempty"`
	KeyPattern    *string    `json:"key_pattern,omitempty"`   // SQL LIKE pattern
	ValuePattern  *string    `json:"value_pattern,omitempty"` // SQL LIKE pattern
	Keys          []string   `json:"keys,omitempty"`          // Multiple keys (OR)
	ValueType     *string    `json:"value_type,omitempty"`
	IsSensitive   *bool      `json:"is_sensitive,omitempty"`
	IsSystem      *bool      `json:"is_system,omitempty"`
	IsMultiValue  *bool      `json:"is_multi_value,omitempty"`
	Tags          []string   `json:"tags,omitempty"`
	FullText      *string    `json:"full_text,omitempty"` // Full-text search
	RepositoryID  *string    `json:"repository_id,omitempty"`
	ArtifactID    *string    `json:"artifact_id,omitempty"`
	CreatedAfter  *time.Time `json:"created_after,omitempty"`
	CreatedBefore *time.Time `json:"created_before,omitempty"`
	Limit         int        `json:"limit,omitempty"` // default: 100
	Offset        int        `json:"offset,omitempty"`
}

// BatchPropertyRequest represents a batch property operation
type BatchPropertyRequest struct {
	ArtifactIDs []string                `json:"artifact_ids" validate:"required,min=1,max=1000"`
	Properties  []CreatePropertyRequest `json:"properties" validate:"required,min=1"`
}

// PropertyResponse represents the API response for a property
type PropertyResponse struct {
	ID           uuid.UUID  `json:"id"`
	TenantID     uuid.UUID  `json:"tenant_id"`
	RepositoryID string     `json:"repository_id"`
	ArtifactID   string     `json:"artifact_id"`
	Key          string     `json:"key"`
	Value        string     `json:"value"` // Masked if sensitive and user lacks permission
	ValueType    string     `json:"value_type"`
	IsSensitive  bool       `json:"is_sensitive"`
	IsSystem     bool       `json:"is_system"`
	IsMultiValue bool       `json:"is_multi_value"`
	CreatedBy    *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedBy    *uuid.UUID `json:"updated_by,omitempty"`
	UpdatedAt    time.Time  `json:"updated_at"`
	Version      int        `json:"version"`
	Tags         []string   `json:"tags,omitempty"`
	Description  *string    `json:"description,omitempty"`
	Masked       bool       `json:"masked,omitempty"` // Indicates if value was masked
}

// MaskValue returns a masked version of the property response if sensitive
func (p *PropertyResponse) MaskValue(canReadSensitive bool) {
	if p.IsSensitive && !canReadSensitive {
		p.Value = "***REDACTED***"
		p.Masked = true
	}
}

// PropertiesResponse represents a paginated list of properties
type PropertiesResponse struct {
	Properties []PropertyResponse `json:"properties"`
	Total      int                `json:"total"`
	Limit      int                `json:"limit"`
	Offset     int                `json:"offset"`
	HasMore    bool               `json:"has_more"`
}

// PropertyKey validation constants
const (
	PropertyKeyPattern  = `^[a-zA-Z0-9._-]+$`
	MaxPropertyKeyLen   = 255
	MaxPropertyValueLen = 65535
	DefaultLimit        = 100
	MaxLimit            = 1000
)

// Common property key prefixes used by the system
const (
	PropertyPrefixScan       = "scan."
	PropertyPrefixCompliance = "compliance."
	PropertyPrefixSBOM       = "sbom."
	PropertyPrefixSecurity   = "security."
	PropertyPrefixQuality    = "quality."
	PropertyPrefixCustom     = "custom."
)

// Standard system property keys
const (
	PropertyScanStatus                = "scan.status"
	PropertyScanDate                  = "scan.date"
	PropertyScanVulnerabilities       = "scan.vulnerabilities"
	PropertyComplianceLicense         = "compliance.license"
	PropertyComplianceApproved        = "compliance.approved"
	PropertySBOMID                    = "sbom.id"
	PropertySecuritySignatureVerified = "security.signature.verified"
	PropertySecurityRiskScore         = "security.risk.score"
)
