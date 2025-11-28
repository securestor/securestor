package models

import (
	"time"

	"github.com/google/uuid"
)

// CompliancePolicy defines data governance and compliance rules
type CompliancePolicy struct {
	ID          uuid.UUID  `json:"id" db:"policy_id"`
	TenantID    uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	Name        string     `json:"name" db:"name"`
	Type        string     `json:"type" db:"type"`     // data_retention, gdpr, legal_hold, audit_logging, data_locality, access_control, encryption, pypi_quality
	Status      string     `json:"status" db:"status"` // active, inactive, draft
	Rules       string     `json:"rules" db:"rules"`   // JSON string of policy rules
	Region      string     `json:"region" db:"region"` // EU, US, IN, GLOBAL
	CreatedBy   *uuid.UUID `json:"created_by" db:"created_by"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
	EnforcedAt  *time.Time `json:"enforced_at" db:"enforced_at"`
	Description string     `json:"description" db:"description"`
}

// DataRetentionRule defines automatic deletion rules
type DataRetentionRule struct {
	ArtifactTypes    []string `json:"artifact_types"`
	RetentionDays    int      `json:"retention_days"`
	DeleteAfterDays  int      `json:"delete_after_days"`
	GracePeriodDays  int      `json:"grace_period_days"`
	NotifyBeforeDays int      `json:"notify_before_days"`
}

// GDPRPolicy defines GDPR compliance rules
type GDPRPolicy struct {
	RightToErasure    bool     `json:"right_to_erasure"`
	DataPortability   bool     `json:"data_portability"`
	ConsentRequired   bool     `json:"consent_required"`
	ProcessingPurpose []string `json:"processing_purpose"`
	DataCategories    []string `json:"data_categories"`
	RetentionPeriod   int      `json:"retention_period"`
}

// LegalHold prevents deletion during investigations
type LegalHold struct {
	ID         uuid.UUID  `json:"id" db:"hold_id"`
	TenantID   uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	ArtifactID uuid.UUID  `json:"artifact_id" db:"artifact_id"`
	CaseNumber string     `json:"case_number" db:"case_number"`
	Reason     string     `json:"reason" db:"reason"`
	StartDate  time.Time  `json:"start_date" db:"start_date"`
	EndDate    *time.Time `json:"end_date" db:"end_date"`
	Status     string     `json:"status" db:"status"` // active, released, expired
	CreatedBy  *uuid.UUID `json:"created_by" db:"created_by"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at" db:"updated_at"`
}

// AuditLog for immutable audit trail
type AuditLog struct {
	ID           int64     `json:"id" db:"id"`
	EventType    string    `json:"event_type" db:"event_type"`
	ResourceID   string    `json:"resource_id" db:"resource_id"`
	ResourceType string    `json:"resource_type" db:"resource_type"`
	UserID       string    `json:"user_id" db:"user_id"`
	Action       string    `json:"action" db:"action"`
	OldValue     string    `json:"old_value" db:"old_value"`
	NewValue     string    `json:"new_value" db:"new_value"`
	IPAddress    string    `json:"ip_address" db:"ip_address"`
	UserAgent    string    `json:"user_agent" db:"user_agent"`
	Success      bool      `json:"success" db:"success"`
	ErrorMsg     string    `json:"error_msg" db:"error_msg"`
	Timestamp    time.Time `json:"timestamp" db:"timestamp"`
	Metadata     string    `json:"metadata" db:"metadata"` // JSON
}

// DataLocality defines regional storage requirements
type DataLocality struct {
	ID             int64     `json:"id" db:"id"`
	ArtifactID     int64     `json:"artifact_id" db:"artifact_id"`
	RequiredRegion string    `json:"required_region" db:"required_region"` // EU, US, IN
	CurrentRegion  string    `json:"current_region" db:"current_region"`
	Compliant      bool      `json:"compliant" db:"compliant"`
	LastChecked    time.Time `json:"last_checked" db:"last_checked"`
}

// AccessControlPolicy defines RBAC and permissions
type AccessControlPolicy struct {
	ID           int64     `json:"id" db:"id"`
	Name         string    `json:"name" db:"name"`
	ResourceType string    `json:"resource_type" db:"resource_type"`
	Permissions  []string  `json:"permissions"`
	Roles        []string  `json:"roles"`
	Conditions   string    `json:"conditions" db:"conditions"` // JSON
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// EncryptionPolicy defines encryption requirements
type EncryptionPolicy struct {
	ID                  int64     `json:"id" db:"id"`
	Name                string    `json:"name" db:"name"`
	EncryptionAtRest    bool      `json:"encryption_at_rest" db:"encryption_at_rest"`
	EncryptionInTransit bool      `json:"encryption_in_transit" db:"encryption_in_transit"`
	KeyManagement       string    `json:"key_management" db:"key_management"` // local, aws_kms, azure_kv, gcp_kms
	Algorithm           string    `json:"algorithm" db:"algorithm"`
	KeyRotationDays     int       `json:"key_rotation_days" db:"key_rotation_days"`
	CreatedAt           time.Time `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time `json:"updated_at" db:"updated_at"`
}

// PyPIQualityPolicy defines Python package quality and governance rules
type PyPIQualityPolicy struct {
	MinQualityScore      int      `json:"min_quality_score"`     // Minimum quality score (0-100)
	RequiredMetadata     []string `json:"required_metadata"`     // Required package metadata fields
	MinTestCoverage      int      `json:"min_test_coverage"`     // Minimum test coverage percentage
	MaxVulnerabilities   int      `json:"max_vulnerabilities"`   // Maximum allowed vulnerabilities
	AllowedLicenses      []string `json:"allowed_licenses"`      // Allowed license types
	BlockedDependencies  []string `json:"blocked_dependencies"`  // Dependencies not allowed
	RequireDocumentation bool     `json:"require_documentation"` // Documentation requirement
	MaxAge               int      `json:"max_age"`               // Maximum package age in days
}

// ComplianceReport aggregates compliance status
type ComplianceReport struct {
	ID              uuid.UUID `json:"id"`
	ArtifactID      uuid.UUID `json:"artifact_id"`
	PolicyID        uuid.UUID `json:"policy_id"`
	Status          string    `json:"status"` // compliant, non_compliant, warning
	Details         string    `json:"details"`
	Recommendations []string  `json:"recommendations"`
	LastChecked     time.Time `json:"last_checked"`
	NextCheck       time.Time `json:"next_check"`
}
