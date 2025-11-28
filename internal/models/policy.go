package models

import (
	"fmt"
	"time"
)

// OPAPolicy represents an OPA Rego policy document
type OPAPolicy struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	PolicyType  string     `json:"policy_type"` // 'rbac', 'abac', 'custom'
	RegoPolicy  string     `json:"rego_policy"`
	Version     int        `json:"version"`
	IsActive    bool       `json:"is_active"`
	CreatedBy   *int64     `json:"created_by,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ActivatedAt *time.Time `json:"activated_at,omitempty"`

	// Populated from joins
	Creator *User `json:"creator,omitempty"`
}

// OPAPolicyData represents data used by OPA policies
type OPAPolicyData struct {
	ID          int64                  `json:"id"`
	DataKey     string                 `json:"data_key"`
	DataValue   map[string]interface{} `json:"data_value"`
	Description string                 `json:"description"`
	CreatedBy   *int64                 `json:"created_by,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`

	// Populated from joins
	Creator *User `json:"creator,omitempty"`
}

// PolicyAuditLog represents a policy decision audit entry
type PolicyAuditLog struct {
	ID              int64                  `json:"id"`
	RequestID       string                 `json:"request_id,omitempty"`
	UserID          *int64                 `json:"user_id,omitempty"`
	Resource        string                 `json:"resource"`
	Action          string                 `json:"action"`
	Decision        string                 `json:"decision"` // 'allow', 'deny'
	PolicyName      string                 `json:"policy_name,omitempty"`
	InputData       map[string]interface{} `json:"input_data,omitempty"`
	PolicyOutput    map[string]interface{} `json:"policy_output,omitempty"`
	ExecutionTimeMs int                    `json:"execution_time_ms"`
	IPAddress       *string                `json:"ip_address,omitempty"`
	UserAgent       *string                `json:"user_agent,omitempty"`
	SessionID       *string                `json:"session_id,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`

	// Populated from joins
	User *User `json:"user,omitempty"`
}

// SecurityPolicyViolation represents a security policy violation
type SecurityPolicyViolation struct {
	ID             int64      `json:"id"`
	AuditLogID     int64      `json:"audit_log_id"`
	ViolationType  string     `json:"violation_type"`
	Severity       string     `json:"severity"` // 'low', 'medium', 'high', 'critical'
	Description    string     `json:"description"`
	IsInvestigated bool       `json:"is_investigated"`
	InvestigatedBy *int64     `json:"investigated_by,omitempty"`
	InvestigatedAt *time.Time `json:"investigated_at,omitempty"`
	Notes          *string    `json:"notes,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`

	// Populated from joins
	AuditLog     *PolicyAuditLog `json:"audit_log,omitempty"`
	Investigator *User           `json:"investigator,omitempty"`
}

// TokenIntrospectionCache represents cached token introspection data
type TokenIntrospectionCache struct {
	ID           int64      `json:"id"`
	TokenHash    string     `json:"token_hash"`
	IsActive     bool       `json:"is_active"`
	TokenType    string     `json:"token_type,omitempty"`
	ClientID     string     `json:"client_id,omitempty"`
	Username     string     `json:"username,omitempty"`
	Scope        string     `json:"scope,omitempty"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	IssuedAt     *time.Time `json:"issued_at,omitempty"`
	CachedAt     time.Time  `json:"cached_at"`
	LastAccessed time.Time  `json:"last_accessed"`
}

// AuditComplianceReport represents a generated compliance report
type AuditComplianceReport struct {
	ID             int64                  `json:"id"`
	ReportType     string                 `json:"report_type"`
	ReportFormat   string                 `json:"report_format"` // 'json', 'csv', 'pdf'
	Parameters     map[string]interface{} `json:"parameters,omitempty"`
	FilePath       *string                `json:"file_path,omitempty"`
	FileSize       *int64                 `json:"file_size,omitempty"`
	GeneratedBy    int64                  `json:"generated_by"`
	GeneratedAt    time.Time              `json:"generated_at"`
	ExpiresAt      *time.Time             `json:"expires_at,omitempty"`
	DownloadCount  int                    `json:"download_count"`
	LastDownloaded *time.Time             `json:"last_downloaded,omitempty"`

	// Populated from joins
	Generator *User `json:"generator,omitempty"`
}

// PolicyStatistics represents policy usage statistics
type PolicyStatistics struct {
	TotalPolicies  int `json:"total_policies"`
	ActivePolicies int `json:"active_policies"`
	RBACPolicies   int `json:"rbac_policies"`
	ABACPolicies   int `json:"abac_policies"`
	CustomPolicies int `json:"custom_policies"`
}

// RecentPolicyDecision represents a recent policy decision
type RecentPolicyDecision struct {
	ID              int64     `json:"id"`
	UserID          *int64    `json:"user_id,omitempty"`
	Username        *string   `json:"username,omitempty"`
	Resource        string    `json:"resource"`
	Action          string    `json:"action"`
	Decision        string    `json:"decision"`
	PolicyName      *string   `json:"policy_name,omitempty"`
	ExecutionTimeMs int       `json:"execution_time_ms"`
	CreatedAt       time.Time `json:"created_at"`
}

// PolicyViolationsSummary represents a summary of policy violations
type PolicyViolationsSummary struct {
	Severity            string `json:"severity"`
	ViolationCount      int    `json:"violation_count"`
	UninvestigatedCount int    `json:"uninvestigated_count"`
	RecentCount         int    `json:"recent_count"`
}

// Token introspection request/response models

// TokenIntrospectionRequest represents a token introspection request
type TokenIntrospectionRequest struct {
	Token         string `json:"token" form:"token" binding:"required"`
	TokenTypeHint string `json:"token_type_hint,omitempty" form:"token_type_hint"`
}

// TokenIntrospectionResponse represents a token introspection response
type TokenIntrospectionResponse struct {
	Active    bool   `json:"active"`
	ClientID  string `json:"client_id,omitempty"`
	Username  string `json:"username,omitempty"`
	Scope     string `json:"scope,omitempty"`
	TokenType string `json:"token_type,omitempty"`
	ExpiresAt int64  `json:"exp,omitempty"`
	IssuedAt  int64  `json:"iat,omitempty"`
	Subject   string `json:"sub,omitempty"`
	Audience  string `json:"aud,omitempty"`
	Issuer    string `json:"iss,omitempty"`
}

// Audit export models

// AuditExportRequest represents a request to export audit data
type AuditExportRequest struct {
	StartDate    time.Time `json:"start_date" binding:"required"`
	EndDate      time.Time `json:"end_date" binding:"required"`
	Format       string    `json:"format" binding:"required"` // 'csv', 'json', 'siem'
	IncludeTypes []string  `json:"include_types,omitempty"`   // 'policy_decisions', 'violations', 'mfa_attempts'
	UserID       *int64    `json:"user_id,omitempty"`
	ResourcePath *string   `json:"resource_path,omitempty"`
}

// Helper methods

// GetSeverityLevel returns numeric severity level for comparison
func (v *SecurityPolicyViolation) GetSeverityLevel() int {
	switch v.Severity {
	case "low":
		return 1
	case "medium":
		return 2
	case "high":
		return 3
	case "critical":
		return 4
	default:
		return 0
	}
}

// IsExpired checks if a cached token is expired
func (c *TokenIntrospectionCache) IsExpired() bool {
	if c.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*c.ExpiresAt)
}

// IsStale checks if the cache entry is stale (older than 5 minutes)
func (c *TokenIntrospectionCache) IsStale() bool {
	return time.Since(c.LastAccessed) > 5*time.Minute
}

// IsReportExpired checks if a compliance report is expired
func (r *AuditComplianceReport) IsReportExpired() bool {
	if r.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*r.ExpiresAt)
}

// GetFileSizeHuman returns file size in human readable format
func (r *AuditComplianceReport) GetFileSizeHuman() string {
	if r.FileSize == nil {
		return "Unknown"
	}

	size := float64(*r.FileSize)
	units := []string{"B", "KB", "MB", "GB"}

	for _, unit := range units {
		if size < 1024 {
			return fmt.Sprintf("%.1f %s", size, unit)
		}
		size /= 1024
	}

	return fmt.Sprintf("%.1f TB", size)
}
