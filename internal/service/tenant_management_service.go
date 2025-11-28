package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// TenantManagementService handles tenant operations and settings
type TenantManagementService struct {
	db *sql.DB
}

// NewTenantManagementService creates a new tenant management service
func NewTenantManagementService(db *sql.DB) *TenantManagementService {
	return &TenantManagementService{
		db: db,
	}
}

// Tenant represents a tenant with settings
type Tenant struct {
	ID              uuid.UUID              `json:"id"`
	Name            string                 `json:"name"`
	Slug            string                 `json:"slug"`
	Description     *string                `json:"description,omitempty"`
	ContactEmail    *string                `json:"contact_email,omitempty"`
	BillingEmail    *string                `json:"billing_email,omitempty"`
	Subdomain       *string                `json:"subdomain,omitempty"`
	Domain          *string                `json:"domain,omitempty"`
	Settings        map[string]interface{} `json:"settings"`
	IsActive        bool                   `json:"is_active"`
	Plan            string                 `json:"plan"`
	MaxUsers        int                    `json:"max_users"`
	MaxRepositories int                    `json:"max_repositories"`
	MaxStorageGb    int                    `json:"max_storage_gb"`
	Features        []string               `json:"features"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`

	// Additional computed fields
	CurrentUsers int     `json:"current_users"`
	UsagePercent float64 `json:"usage_percent"`
}

// TenantSettings represents tenant-specific configuration
type TenantSettings struct {
	TenantID uuid.UUID `json:"tenant_id"`

	// Security settings
	SecuritySettings SecuritySettings `json:"security"`

	// User management settings
	UserSettings UserSettings `json:"user_management"`

	// Storage settings
	StorageSettings StorageSettings `json:"storage"`

	// Notification settings
	NotificationSettings NotificationSettings `json:"notifications"`

	// Integration settings
	IntegrationSettings IntegrationSettings `json:"integrations"`

	// Compliance settings
	ComplianceSettings ComplianceSettings `json:"compliance"`

	// Billing and subscription settings
	BillingSettings BillingSettings `json:"billing"`

	// Feature flags and toggles
	FeatureFlags FeatureFlags `json:"feature_flags"`

	// Monitoring and analytics settings
	MonitoringSettings MonitoringSettings `json:"monitoring"`

	// Advanced security settings
	AdvancedSecuritySettings AdvancedSecuritySettings `json:"advanced_security"`
}

// SecuritySettings represents security-related tenant settings
type SecuritySettings struct {
	MFARequired            bool           `json:"mfa_required"`
	MFAMethods             []string       `json:"mfa_methods"`
	PasswordPolicy         PasswordPolicy `json:"password_policy"`
	SessionTimeoutMinutes  int            `json:"session_timeout_minutes"`
	MaxLoginAttempts       int            `json:"max_login_attempts"`
	LockoutDurationMinutes int            `json:"lockout_duration_minutes"`
	AllowedIPRanges        []string       `json:"allowed_ip_ranges"`
	RequireSSO             bool           `json:"require_sso"`
	AllowAPIKeys           bool           `json:"allow_api_keys"`
	APIKeyExpirationDays   *int           `json:"api_key_expiration_days,omitempty"`
}

// PasswordPolicy represents password requirements
type PasswordPolicy struct {
	MinLength        int  `json:"min_length"`
	RequireUppercase bool `json:"require_uppercase"`
	RequireLowercase bool `json:"require_lowercase"`
	RequireNumbers   bool `json:"require_numbers"`
	RequireSymbols   bool `json:"require_symbols"`
	MaxAge           *int `json:"max_age,omitempty"`
	PreventReuse     int  `json:"prevent_reuse"`
}

// UserSettings represents user management settings
type UserSettings struct {
	AllowSelfRegistration     bool     `json:"allow_self_registration"`
	DefaultRoles              []string `json:"default_roles"`
	EmailVerificationRequired bool     `json:"email_verification_required"`
	AllowUsernameChange       bool     `json:"allow_username_change"`
	AllowEmailChange          bool     `json:"allow_email_change"`
	InvitationExpiryDays      int      `json:"invitation_expiry_days"`
	MaxUsers                  int      `json:"max_users"`
	AllowedDomains            []string `json:"allowed_domains"`
}

// StorageSettings represents storage and data management settings
type StorageSettings struct {
	MaxStorageGB         int      `json:"max_storage_gb"`
	MaxFileSize          int      `json:"max_file_size"`
	AllowedFileTypes     []string `json:"allowed_file_types"`
	RetentionPolicyDays  int      `json:"retention_policy_days"`
	BackupEnabled        bool     `json:"backup_enabled"`
	BackupFrequencyHours int      `json:"backup_frequency_hours"`
	EncryptionEnabled    bool     `json:"encryption_enabled"`
}

// NotificationSettings represents notification preferences
type NotificationSettings struct {
	EmailNotifications bool     `json:"email_notifications"`
	SlackIntegration   bool     `json:"slack_integration"`
	SlackWebhookURL    *string  `json:"slack_webhook_url,omitempty"`
	SecurityAlerts     bool     `json:"security_alerts"`
	SystemAlerts       bool     `json:"system_alerts"`
	ComplianceAlerts   bool     `json:"compliance_alerts"`
	NotificationEmails []string `json:"notification_emails"`
}

// IntegrationSettings represents third-party integrations
type IntegrationSettings struct {
	SSOProviders        []SSOProvider  `json:"sso_providers"`
	WebhookEndpoints    []string       `json:"webhook_endpoints"`
	AllowedIntegrations []string       `json:"allowed_integrations"`
	APIRateLimits       map[string]int `json:"api_rate_limits"`
}

// SSOProvider represents SSO configuration
type SSOProvider struct {
	Name          string                 `json:"name"`
	Type          string                 `json:"type"` // oidc, saml, etc.
	Enabled       bool                   `json:"enabled"`
	Configuration map[string]interface{} `json:"configuration"`
}

// ComplianceSettings represents compliance and audit settings
type ComplianceSettings struct {
	AuditLogsEnabled   bool   `json:"audit_logs_enabled"`
	AuditRetentionDays int    `json:"audit_retention_days"`
	ComplianceMode     string `json:"compliance_mode"` // none, basic, strict
	DataResidency      string `json:"data_residency"`
	GDPRCompliance     bool   `json:"gdpr_compliance"`
	SOC2Compliance     bool   `json:"soc2_compliance"`
	HIPAACompliance    bool   `json:"hipaa_compliance"`
}

// BillingSettings represents billing and subscription settings
type BillingSettings struct {
	PlanType              string      `json:"plan_type"`     // basic, professional, enterprise
	BillingCycle          string      `json:"billing_cycle"` // monthly, annually
	UsageLimits           UsageLimits `json:"usage_limits"`
	OverageChargesEnabled bool        `json:"overage_charges_enabled"`
	BillingContact        string      `json:"billing_contact"`
	TaxID                 *string     `json:"tax_id,omitempty"`
	BillingAddress        Address     `json:"billing_address"`
	PaymentMethods        []string    `json:"payment_methods"`
	InvoiceDelivery       string      `json:"invoice_delivery"` // email, portal, both
	AutoRenewal           bool        `json:"auto_renewal"`
}

// UsageLimits represents usage limits for different resources
type UsageLimits struct {
	MaxUsers        int `json:"max_users"`
	MaxAPIRequests  int `json:"max_api_requests"` // per month
	MaxStorageGB    int `json:"max_storage_gb"`
	MaxIntegrations int `json:"max_integrations"`
	MaxProjects     int `json:"max_projects"`
	MaxScanJobs     int `json:"max_scan_jobs"` // per month
}

// Address represents a billing address
type Address struct {
	Street1    string `json:"street1"`
	Street2    string `json:"street2,omitempty"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
}

// FeatureFlags represents feature toggles and experimental features
type FeatureFlags struct {
	EnableAdvancedScanning bool            `json:"enable_advanced_scanning"`
	EnableMLAnalysis       bool            `json:"enable_ml_analysis"`
	EnableCustomReporting  bool            `json:"enable_custom_reporting"`
	EnableAPIv2            bool            `json:"enable_api_v2"`
	EnableBetaFeatures     bool            `json:"enable_beta_features"`
	CustomFeatureFlags     map[string]bool `json:"custom_feature_flags"`
	FeatureLimits          map[string]int  `json:"feature_limits"`
	ExperimentalFeatures   []string        `json:"experimental_features"`
}

// MonitoringSettings represents monitoring and analytics configuration
type MonitoringSettings struct {
	MetricsEnabled        bool            `json:"metrics_enabled"`
	LogLevel              string          `json:"log_level"` // debug, info, warn, error
	PerformanceMonitoring bool            `json:"performance_monitoring"`
	ErrorTracking         bool            `json:"error_tracking"`
	UsageAnalytics        bool            `json:"usage_analytics"`
	RealTimeAlerts        bool            `json:"real_time_alerts"`
	AlertThresholds       AlertThresholds `json:"alert_thresholds"`
	CustomDashboards      []string        `json:"custom_dashboards"`
	MetricsRetentionDays  int             `json:"metrics_retention_days"`
	LogRetentionDays      int             `json:"log_retention_days"`
}

// AlertThresholds represents various alert thresholds
type AlertThresholds struct {
	CPUUsagePercent    int `json:"cpu_usage_percent"`
	MemoryUsagePercent int `json:"memory_usage_percent"`
	DiskUsagePercent   int `json:"disk_usage_percent"`
	ErrorRatePercent   int `json:"error_rate_percent"`
	ResponseTimeMs     int `json:"response_time_ms"`
	FailedLoginsCount  int `json:"failed_logins_count"`
}

// AdvancedSecuritySettings represents advanced security features
type AdvancedSecuritySettings struct {
	ZeroTrustMode      bool                       `json:"zero_trust_mode"`
	ThreatDetection    ThreatDetectionSettings    `json:"threat_detection"`
	DataLossPrevention DLPSettings                `json:"data_loss_prevention"`
	NetworkSecurity    NetworkSecuritySettings    `json:"network_security"`
	EncryptionSettings EncryptionSettings         `json:"encryption_settings"`
	AccessControls     AccessControlSettings      `json:"access_controls"`
	SecurityMonitoring SecurityMonitoringSettings `json:"security_monitoring"`
}

// ThreatDetectionSettings represents threat detection configuration
type ThreatDetectionSettings struct {
	Enabled                 bool     `json:"enabled"`
	AnomalyDetection        bool     `json:"anomaly_detection"`
	BehavioralAnalysis      bool     `json:"behavioral_analysis"`
	ThreatIntelligence      bool     `json:"threat_intelligence"`
	AutoQuarantine          bool     `json:"auto_quarantine"`
	ThreatFeedSources       []string `json:"threat_feed_sources"`
	SuspiciousActivityAlert bool     `json:"suspicious_activity_alert"`
}

// DLPSettings represents Data Loss Prevention settings
type DLPSettings struct {
	Enabled            bool     `json:"enabled"`
	ScanUploads        bool     `json:"scan_uploads"`
	BlockSensitiveData bool     `json:"block_sensitive_data"`
	DataClassification []string `json:"data_classification"`
	CustomPolicyRules  []string `json:"custom_policy_rules"`
	NotifyOnViolation  bool     `json:"notify_on_violation"`
}

// NetworkSecuritySettings represents network security configuration
type NetworkSecuritySettings struct {
	FirewallEnabled    bool     `json:"firewall_enabled"`
	IPWhitelisting     []string `json:"ip_whitelisting"`
	GeoBlocking        []string `json:"geo_blocking"` // blocked countries
	VPNRequired        bool     `json:"vpn_required"`
	TLSVersion         string   `json:"tls_version"` // minimum TLS version
	CertificatePinning bool     `json:"certificate_pinning"`
}

// EncryptionSettings represents encryption configuration
type EncryptionSettings struct {
	EncryptionAtRest     bool   `json:"encryption_at_rest"`
	EncryptionInTransit  bool   `json:"encryption_in_transit"`
	KeyRotationEnabled   bool   `json:"key_rotation_enabled"`
	KeyRotationDays      int    `json:"key_rotation_days"`
	EncryptionAlgorithm  string `json:"encryption_algorithm"`
	KeyManagementService string `json:"key_management_service"` // internal, aws_kms, azure_kv, etc.
}

// AccessControlSettings represents access control configuration
type AccessControlSettings struct {
	RoleBasedAccess      bool              `json:"role_based_access"`
	AttributeBasedAccess bool              `json:"attribute_based_access"`
	JustInTimeAccess     bool              `json:"just_in_time_access"`
	PrivilegedAccessMgmt bool              `json:"privileged_access_management"`
	AccessReviewRequired bool              `json:"access_review_required"`
	AccessReviewDays     int               `json:"access_review_days"`
	ConditionalAccess    ConditionalAccess `json:"conditional_access"`
}

// ConditionalAccess represents conditional access policies
type ConditionalAccess struct {
	Enabled             bool     `json:"enabled"`
	LocationBasedAccess bool     `json:"location_based_access"`
	DeviceBasedAccess   bool     `json:"device_based_access"`
	TimeBasedAccess     bool     `json:"time_based_access"`
	RiskBasedAccess     bool     `json:"risk_based_access"`
	AllowedLocations    []string `json:"allowed_locations"`
	BusinessHoursOnly   bool     `json:"business_hours_only"`
}

// SecurityMonitoringSettings represents security monitoring configuration
type SecurityMonitoringSettings struct {
	ContinuousMonitoring  bool                     `json:"continuous_monitoring"`
	SecurityEventLogging  bool                     `json:"security_event_logging"`
	IntrusionDetection    bool                     `json:"intrusion_detection"`
	VulnerabilityScanning bool                     `json:"vulnerability_scanning"`
	SecurityScoreTracking bool                     `json:"security_score_tracking"`
	ComplianceMonitoring  bool                     `json:"compliance_monitoring"`
	IncidentResponse      IncidentResponseSettings `json:"incident_response"`
}

// IncidentResponseSettings represents incident response configuration
type IncidentResponseSettings struct {
	AutoIncidentCreation   bool     `json:"auto_incident_creation"`
	SeverityClassification bool     `json:"severity_classification"`
	AutoNotification       bool     `json:"auto_notification"`
	EscalationRules        []string `json:"escalation_rules"`
	ResponseTeamContacts   []string `json:"response_team_contacts"`
	MaxResponseTimeMinutes int      `json:"max_response_time_minutes"`
}

// TenantFilter represents filtering options for tenant queries
type TenantFilter struct {
	Search    string
	IsActive  *bool
	Plan      string
	Limit     int
	Offset    int
	SortBy    string
	SortOrder string
}

// TenantUsage represents tenant usage statistics
type TenantUsage struct {
	TenantID           uuid.UUID `json:"tenant_id"`
	UserCount          int       `json:"user_count"`
	APIKeyCount        int       `json:"api_key_count"`
	StorageUsedGB      float64   `json:"storage_used_gb"`
	RequestsLast30Days int64     `json:"requests_last_30_days"`
	ActiveSessions     int       `json:"active_sessions"`
}

// GetTenants retrieves tenants with filtering
func (s *TenantManagementService) GetTenants(ctx context.Context, filter *TenantFilter) ([]*Tenant, int, error) {
	// Build WHERE clause
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argCount := 0

	if filter.Search != "" {
		argCount++
		whereClause += fmt.Sprintf(" AND (name ILIKE $%d OR subdomain ILIKE $%d)", argCount, argCount)
		searchTerm := "%" + filter.Search + "%"
		args = append(args, searchTerm)
	}

	if filter.IsActive != nil {
		argCount++
		whereClause += fmt.Sprintf(" AND is_active = $%d", argCount)
		args = append(args, *filter.IsActive)
	}

	if filter.Plan != "" {
		argCount++
		whereClause += fmt.Sprintf(" AND plan = $%d", argCount)
		args = append(args, filter.Plan)
	}

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM tenants %s", whereClause)
	var totalCount int
	err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get tenant count: %w", err)
	}

	// Build ORDER BY clause
	orderBy := "ORDER BY created_at DESC"
	if filter.SortBy != "" {
		sortOrder := "ASC"
		if strings.ToUpper(filter.SortOrder) == "DESC" {
			sortOrder = "DESC"
		}
		orderBy = fmt.Sprintf("ORDER BY %s %s", filter.SortBy, sortOrder)
	}

	// Build LIMIT and OFFSET
	limit := 50
	if filter.Limit > 0 && filter.Limit <= 100 {
		limit = filter.Limit
	}
	offset := 0
	if filter.Offset >= 0 {
		offset = filter.Offset
	}

	argCount++
	limitClause := fmt.Sprintf("LIMIT $%d", argCount)
	args = append(args, limit)

	argCount++
	offsetClause := fmt.Sprintf("OFFSET $%d", argCount)
	args = append(args, offset)

	// Main query with user count
	query := fmt.Sprintf(`
		SELECT t.tenant_id, t.name, t.slug, t.description, t.contact_email, t.settings, t.is_active, 
		       t.plan, t.max_users, t.max_repositories, t.features, t.created_at, t.updated_at,
		       COALESCE(uc.user_count, 0) as current_users
		FROM tenants t
		LEFT JOIN (
			SELECT tenant_id, COUNT(*) as user_count
			FROM users
			WHERE is_active = true
			GROUP BY tenant_id
		) uc ON t.tenant_id = uc.tenant_id
		%s %s %s %s
	`, whereClause, orderBy, limitClause, offsetClause)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query tenants: %w", err)
	}
	defer rows.Close()

	var tenants []*Tenant
	for rows.Next() {
		tenant := &Tenant{}
		var settingsJSON []byte
		var featuresArray pq.StringArray
		var tenantUUID uuid.UUID

		err := rows.Scan(
			&tenantUUID, &tenant.Name, &tenant.Slug, &tenant.Description, &tenant.ContactEmail,
			&settingsJSON, &tenant.IsActive, &tenant.Plan, &tenant.MaxUsers, &tenant.MaxRepositories,
			&featuresArray, &tenant.CreatedAt, &tenant.UpdatedAt, &tenant.CurrentUsers,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan tenant: %w", err)
		}

		// Convert UUID to int64 hash for backward compatibility
		tenantBytes := tenantUUID[:]
		var id int64
		for i := 0; i < 8; i++ {
			id = (id << 8) | int64(tenantBytes[i])
		}
		tenant.ID = tenantUUID

		// Parse settings JSON
		if len(settingsJSON) > 0 {
			err = json.Unmarshal(settingsJSON, &tenant.Settings)
			if err != nil {
				tenant.Settings = make(map[string]interface{})
			}
		} else {
			tenant.Settings = make(map[string]interface{})
		}

		// Convert features array
		tenant.Features = []string(featuresArray)

		// Calculate usage percentage
		if tenant.MaxUsers > 0 {
			tenant.UsagePercent = (float64(tenant.CurrentUsers) / float64(tenant.MaxUsers)) * 100
		}

		tenants = append(tenants, tenant)
	}

	return tenants, totalCount, nil
}

// GetTenantByID retrieves a tenant by ID
func (s *TenantManagementService) GetTenantByID(ctx context.Context, tenantID uuid.UUID) (*Tenant, error) {
	query := `
		SELECT t.tenant_id, t.name, t.slug, t.description, t.contact_email, t.settings, t.is_active, 
		       t.plan, t.max_users, t.max_repositories, t.features, t.created_at, t.updated_at,
		       COALESCE(uc.user_count, 0) as current_users
		FROM tenants t
		LEFT JOIN (
			SELECT tenant_id, COUNT(*) as user_count
			FROM users
			WHERE is_active = true AND tenant_id = $1
			GROUP BY tenant_id
		) uc ON t.tenant_id = uc.tenant_id
		WHERE t.tenant_id = $1
	`

	tenant := &Tenant{}
	var settingsJSON []byte
	var featuresArray pq.StringArray

	err := s.db.QueryRowContext(ctx, query, tenantID).Scan(
		&tenant.ID, &tenant.Name, &tenant.Slug, &tenant.Description, &tenant.ContactEmail,
		&settingsJSON, &tenant.IsActive, &tenant.Plan, &tenant.MaxUsers, &tenant.MaxRepositories,
		&featuresArray, &tenant.CreatedAt, &tenant.UpdatedAt, &tenant.CurrentUsers,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("tenant not found")
		}
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	// Parse settings JSON
	if len(settingsJSON) > 0 {
		err = json.Unmarshal(settingsJSON, &tenant.Settings)
		if err != nil {
			tenant.Settings = make(map[string]interface{})
		}
	} else {
		tenant.Settings = make(map[string]interface{})
	}

	// Convert features array
	tenant.Features = []string(featuresArray)

	// Calculate usage percentage
	if tenant.MaxUsers > 0 {
		tenant.UsagePercent = (float64(tenant.CurrentUsers) / float64(tenant.MaxUsers)) * 100
	}

	return tenant, nil
}

// GetTenantBySlug retrieves a tenant by slug
func (s *TenantManagementService) GetTenantBySlug(ctx context.Context, slug string) (*Tenant, error) {
	query := `
		SELECT t.tenant_id, t.name, t.slug, t.description, t.contact_email, t.settings, t.is_active, 
		       t.plan, t.max_users, t.max_repositories, t.features, t.created_at, t.updated_at,
		       COALESCE(uc.user_count, 0) as current_users
		FROM tenants t
		LEFT JOIN (
			SELECT tenant_id, COUNT(*) as user_count
			FROM users
			WHERE is_active = true
			GROUP BY tenant_id
		) uc ON t.tenant_id = uc.tenant_id
		WHERE t.slug = $1
	`

	tenant := &Tenant{}
	var settingsJSON []byte
	var featuresArray pq.StringArray

	err := s.db.QueryRowContext(ctx, query, slug).Scan(
		&tenant.ID, &tenant.Name, &tenant.Slug, &tenant.Description, &tenant.ContactEmail,
		&settingsJSON, &tenant.IsActive, &tenant.Plan, &tenant.MaxUsers, &tenant.MaxRepositories,
		&featuresArray, &tenant.CreatedAt, &tenant.UpdatedAt, &tenant.CurrentUsers,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("tenant not found")
		}
		return nil, fmt.Errorf("failed to get tenant by slug: %w", err)
	}

	// Parse settings JSON
	if len(settingsJSON) > 0 {
		err = json.Unmarshal(settingsJSON, &tenant.Settings)
		if err != nil {
			tenant.Settings = make(map[string]interface{})
		}
	} else {
		tenant.Settings = make(map[string]interface{})
	}

	// Convert features array
	tenant.Features = []string(featuresArray)

	// Calculate usage percentage
	if tenant.MaxUsers > 0 {
		tenant.UsagePercent = (float64(tenant.CurrentUsers) / float64(tenant.MaxUsers)) * 100
	}

	return tenant, nil
}

// CreateTenant creates a new tenant
func (s *TenantManagementService) CreateTenant(ctx context.Context, tenant *Tenant) (*Tenant, error) {
	// Use provided slug/subdomain if available, otherwise generate from name
	var slug string
	if tenant.Slug != "" {
		slug = strings.ToLower(strings.TrimSpace(tenant.Slug))
	} else if tenant.Subdomain != nil && *tenant.Subdomain != "" {
		slug = strings.ToLower(strings.TrimSpace(*tenant.Subdomain))
	} else {
		// Generate slug from tenant name as fallback
		slug = s.generateSlug(tenant.Name)
	}

	// Validate slug format
	if !isValidSlug(slug) {
		return nil, fmt.Errorf("invalid slug format: must be lowercase alphanumeric with hyphens only")
	}

	// Convert settings to JSON
	settingsJSON, err := json.Marshal(tenant.Settings)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal settings: %w", err)
	}

	// Generate new UUID for tenant
	tenantID := uuid.New()

	query := `
		INSERT INTO tenants (tenant_id, name, slug, description, contact_email, is_active, plan, max_users, max_repositories, features, settings)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING tenant_id, created_at, updated_at
	`

	err = s.db.QueryRowContext(ctx, query,
		tenantID, tenant.Name, slug, tenant.Description, tenant.ContactEmail,
		tenant.IsActive, tenant.Plan, tenant.MaxUsers, tenant.MaxRepositories, pq.Array(tenant.Features), settingsJSON,
	).Scan(&tenant.ID, &tenant.CreatedAt, &tenant.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create tenant: %w", err)
	}

	// Set the slug in return value
	tenant.Slug = slug

	return tenant, nil
}

// UpdateTenant updates an existing tenant
func (s *TenantManagementService) UpdateTenant(ctx context.Context, tenantID uuid.UUID, updates map[string]interface{}) (*Tenant, error) {
	if len(updates) == 0 {
		return s.GetTenantByID(ctx, tenantID)
	}

	// Build dynamic update query
	setParts := []string{}
	args := []interface{}{}
	argCount := 0

	for field, value := range updates {
		argCount++
		if field == "settings" {
			// Handle settings as JSON
			settingsJSON, err := json.Marshal(value)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal settings: %w", err)
			}
			setParts = append(setParts, fmt.Sprintf("settings = $%d", argCount))
			args = append(args, settingsJSON)
		} else if field == "features" {
			// Handle features as array
			setParts = append(setParts, fmt.Sprintf("features = $%d", argCount))
			args = append(args, pq.Array(value))
		} else {
			setParts = append(setParts, fmt.Sprintf("%s = $%d", field, argCount))
			args = append(args, value)
		}
	}

	argCount++
	setParts = append(setParts, fmt.Sprintf("updated_at = $%d", argCount))
	args = append(args, time.Now())

	argCount++
	args = append(args, tenantID)

	query := fmt.Sprintf(`
		UPDATE tenants SET %s WHERE tenant_id = $%d
	`, strings.Join(setParts, ", "), argCount)

	_, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update tenant: %w", err)
	}

	return s.GetTenantByID(ctx, tenantID)
}

// GetTenantSettings retrieves tenant settings with defaults
func (s *TenantManagementService) GetTenantSettings(ctx context.Context, tenantID uuid.UUID) (*TenantSettings, error) {
	tenant, err := s.GetTenantByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	settings := &TenantSettings{
		TenantID: tenantID,
	}

	// Extract settings from tenant settings map or use defaults
	if securityData, ok := tenant.Settings["security"]; ok {
		if securityBytes, err := json.Marshal(securityData); err == nil {
			json.Unmarshal(securityBytes, &settings.SecuritySettings)
		}
	}

	if userData, ok := tenant.Settings["user_management"]; ok {
		if userBytes, err := json.Marshal(userData); err == nil {
			json.Unmarshal(userBytes, &settings.UserSettings)
		}
	}

	if storageData, ok := tenant.Settings["storage"]; ok {
		if storageBytes, err := json.Marshal(storageData); err == nil {
			json.Unmarshal(storageBytes, &settings.StorageSettings)
		}
	}

	if notificationData, ok := tenant.Settings["notifications"]; ok {
		if notificationBytes, err := json.Marshal(notificationData); err == nil {
			json.Unmarshal(notificationBytes, &settings.NotificationSettings)
		}
	}

	if integrationData, ok := tenant.Settings["integrations"]; ok {
		if integrationBytes, err := json.Marshal(integrationData); err == nil {
			json.Unmarshal(integrationBytes, &settings.IntegrationSettings)
		}
	}

	if complianceData, ok := tenant.Settings["compliance"]; ok {
		if complianceBytes, err := json.Marshal(complianceData); err == nil {
			json.Unmarshal(complianceBytes, &settings.ComplianceSettings)
		}
	}

	// Extract enterprise settings sections
	if billingData, ok := tenant.Settings["billing"]; ok {
		if billingBytes, err := json.Marshal(billingData); err == nil {
			json.Unmarshal(billingBytes, &settings.BillingSettings)
		}
	}

	if featureData, ok := tenant.Settings["features"]; ok {
		if featureBytes, err := json.Marshal(featureData); err == nil {
			json.Unmarshal(featureBytes, &settings.FeatureFlags)
		}
	}

	if monitoringData, ok := tenant.Settings["monitoring"]; ok {
		if monitoringBytes, err := json.Marshal(monitoringData); err == nil {
			json.Unmarshal(monitoringBytes, &settings.MonitoringSettings)
		}
	}

	if advSecData, ok := tenant.Settings["advanced-security"]; ok {
		if advSecBytes, err := json.Marshal(advSecData); err == nil {
			json.Unmarshal(advSecBytes, &settings.AdvancedSecuritySettings)
		}
	}

	// Apply defaults for any missing settings
	s.applyDefaultSettings(settings)

	return settings, nil
}

// UpdateTenantSettings updates tenant settings
func (s *TenantManagementService) UpdateTenantSettings(ctx context.Context, tenantID uuid.UUID, settings *TenantSettings) error {
	// Build settings map with all enterprise sections
	settingsMap := map[string]interface{}{
		"security":          settings.SecuritySettings,
		"user_management":   settings.UserSettings,
		"storage":           settings.StorageSettings,
		"notifications":     settings.NotificationSettings,
		"integrations":      settings.IntegrationSettings,
		"compliance":        settings.ComplianceSettings,
		"billing":           settings.BillingSettings,
		"features":          settings.FeatureFlags,
		"monitoring":        settings.MonitoringSettings,
		"advanced-security": settings.AdvancedSecuritySettings,
	}

	updates := map[string]interface{}{
		"settings": settingsMap,
	}

	_, err := s.UpdateTenant(ctx, tenantID, updates)
	return err
}

// GetTenantUsage retrieves tenant usage statistics
func (s *TenantManagementService) GetTenantUsage(ctx context.Context, tenantID uuid.UUID) (*TenantUsage, error) {
	usage := &TenantUsage{
		TenantID: tenantID,
	}

	// Get user count
	err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM users WHERE tenant_id = $1 AND is_active = true",
		tenantID).Scan(&usage.UserCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get user count: %w", err)
	}

	// Get API key count
	err = s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM api_keys WHERE tenant_id = $1 AND is_active = true",
		tenantID).Scan(&usage.APIKeyCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get API key count: %w", err)
	}

	// Get requests in last 30 days (if api_key_usage_logs table exists)
	err = s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM api_key_usage_logs aul 
		 JOIN api_keys ak ON aul.api_key_id = ak.id 
		 WHERE ak.tenant_id = $1 AND aul.request_timestamp >= NOW() - INTERVAL '30 days'`,
		tenantID).Scan(&usage.RequestsLast30Days)
	if err != nil {
		// Table might not exist, set to 0
		usage.RequestsLast30Days = 0
	}

	// Get active sessions (if user_sessions table exists)
	err = s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM user_sessions us 
		 JOIN users u ON us.user_id = u.id 
		 WHERE u.tenant_id = $1 AND us.is_active = true AND us.expires_at > NOW()`,
		tenantID).Scan(&usage.ActiveSessions)
	if err != nil {
		// Table might not exist, set to 0
		usage.ActiveSessions = 0
	}

	return usage, nil
}

// Helper method to apply default settings
func (s *TenantManagementService) applyDefaultSettings(settings *TenantSettings) {
	// Apply security defaults
	if settings.SecuritySettings.SessionTimeoutMinutes == 0 {
		settings.SecuritySettings.SessionTimeoutMinutes = 480 // 8 hours
	}
	if settings.SecuritySettings.MaxLoginAttempts == 0 {
		settings.SecuritySettings.MaxLoginAttempts = 5
	}
	if settings.SecuritySettings.LockoutDurationMinutes == 0 {
		settings.SecuritySettings.LockoutDurationMinutes = 30
	}
	if len(settings.SecuritySettings.MFAMethods) == 0 {
		settings.SecuritySettings.MFAMethods = []string{"totp", "webauthn"}
	}
	if settings.SecuritySettings.PasswordPolicy.MinLength == 0 {
		settings.SecuritySettings.PasswordPolicy.MinLength = 8
	}

	// Apply user management defaults
	if settings.UserSettings.InvitationExpiryDays == 0 {
		settings.UserSettings.InvitationExpiryDays = 7
	}
	if settings.UserSettings.MaxUsers == 0 {
		settings.UserSettings.MaxUsers = 100
	}

	// Apply storage defaults
	if settings.StorageSettings.MaxStorageGB == 0 {
		settings.StorageSettings.MaxStorageGB = 10
	}
	if settings.StorageSettings.MaxFileSize == 0 {
		settings.StorageSettings.MaxFileSize = 100 // MB
	}
	if settings.StorageSettings.RetentionPolicyDays == 0 {
		settings.StorageSettings.RetentionPolicyDays = 365
	}
	if len(settings.StorageSettings.AllowedFileTypes) == 0 {
		settings.StorageSettings.AllowedFileTypes = []string{".jar", ".war", ".zip", ".tar.gz"}
	}

	// Apply compliance defaults
	if settings.ComplianceSettings.AuditRetentionDays == 0 {
		settings.ComplianceSettings.AuditRetentionDays = 90
	}
	if settings.ComplianceSettings.ComplianceMode == "" {
		settings.ComplianceSettings.ComplianceMode = "basic"
	}
}

// DeleteTenant deletes a tenant by ID
func (s *TenantManagementService) DeleteTenant(ctx context.Context, tenantID uuid.UUID) error {
	query := `DELETE FROM tenants WHERE tenant_id = $1`
	result, err := s.db.ExecContext(ctx, query, tenantID)
	if err != nil {
		return fmt.Errorf("failed to delete tenant: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// PatchTenantSettings partially updates tenant settings
func (s *TenantManagementService) PatchTenantSettings(ctx context.Context, tenantID uuid.UUID, patch map[string]interface{}) (*TenantSettings, error) {
	// Get current settings
	settings, err := s.GetTenantSettings(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// Apply patch for each section
	for section, sectionData := range patch {
		switch section {
		case "security":
			if securityUpdates, ok := sectionData.(map[string]interface{}); ok {
				if mfaRequired, exists := securityUpdates["mfa_required"]; exists {
					if val, ok := mfaRequired.(bool); ok {
						settings.SecuritySettings.MFARequired = val
					}
				}
				if passwordPolicy, exists := securityUpdates["password_policy"]; exists {
					if policy, ok := passwordPolicy.(map[string]interface{}); ok {
						if minLength, exists := policy["min_length"]; exists {
							if val, ok := minLength.(float64); ok {
								settings.SecuritySettings.PasswordPolicy.MinLength = int(val)
							}
						}
					}
				}
			}
		case "compliance":
			if complianceUpdates, ok := sectionData.(map[string]interface{}); ok {
				if auditRetention, exists := complianceUpdates["audit_retention_days"]; exists {
					if val, ok := auditRetention.(float64); ok {
						settings.ComplianceSettings.AuditRetentionDays = int(val)
					}
				}
				if auditLogging, exists := complianceUpdates["audit_logs_enabled"]; exists {
					if val, ok := auditLogging.(bool); ok {
						settings.ComplianceSettings.AuditLogsEnabled = val
					}
				}
				if complianceMode, exists := complianceUpdates["compliance_mode"]; exists {
					if val, ok := complianceMode.(string); ok {
						settings.ComplianceSettings.ComplianceMode = val
					}
				}
			}
		case "billing":
			if billingUpdates, ok := sectionData.(map[string]interface{}); ok {
				if planType, exists := billingUpdates["plan_type"]; exists {
					if val, ok := planType.(string); ok {
						settings.BillingSettings.PlanType = val
					}
				}
				if autoRenewal, exists := billingUpdates["auto_renewal"]; exists {
					if val, ok := autoRenewal.(bool); ok {
						settings.BillingSettings.AutoRenewal = val
					}
				}
				if billingCycle, exists := billingUpdates["billing_cycle"]; exists {
					if val, ok := billingCycle.(string); ok {
						settings.BillingSettings.BillingCycle = val
					}
				}
				if overageCharges, exists := billingUpdates["overage_charges_enabled"]; exists {
					if val, ok := overageCharges.(bool); ok {
						settings.BillingSettings.OverageChargesEnabled = val
					}
				}
				if billingContact, exists := billingUpdates["billing_contact"]; exists {
					if val, ok := billingContact.(string); ok {
						settings.BillingSettings.BillingContact = val
					}
				}
			}
		case "features":
			if featureUpdates, ok := sectionData.(map[string]interface{}); ok {
				if advancedScanning, exists := featureUpdates["enable_advanced_scanning"]; exists {
					if val, ok := advancedScanning.(bool); ok {
						settings.FeatureFlags.EnableAdvancedScanning = val
					}
				}
				if customReporting, exists := featureUpdates["enable_custom_reporting"]; exists {
					if val, ok := customReporting.(bool); ok {
						settings.FeatureFlags.EnableCustomReporting = val
					}
				}
				if apiv2, exists := featureUpdates["enable_api_v2"]; exists {
					if val, ok := apiv2.(bool); ok {
						settings.FeatureFlags.EnableAPIv2 = val
					}
				}
				if betaFeatures, exists := featureUpdates["enable_beta_features"]; exists {
					if val, ok := betaFeatures.(bool); ok {
						settings.FeatureFlags.EnableBetaFeatures = val
					}
				}
				if mlAnalysis, exists := featureUpdates["enable_ml_analysis"]; exists {
					if val, ok := mlAnalysis.(bool); ok {
						settings.FeatureFlags.EnableMLAnalysis = val
					}
				}
			}
		case "monitoring":
			if monitoringUpdates, ok := sectionData.(map[string]interface{}); ok {
				if metricsEnabled, exists := monitoringUpdates["metrics_enabled"]; exists {
					if val, ok := metricsEnabled.(bool); ok {
						settings.MonitoringSettings.MetricsEnabled = val
					}
				}
				if performanceMonitoring, exists := monitoringUpdates["performance_monitoring"]; exists {
					if val, ok := performanceMonitoring.(bool); ok {
						settings.MonitoringSettings.PerformanceMonitoring = val
					}
				}
				if realTimeAlerts, exists := monitoringUpdates["real_time_alerts"]; exists {
					if val, ok := realTimeAlerts.(bool); ok {
						settings.MonitoringSettings.RealTimeAlerts = val
					}
				}
				if logLevel, exists := monitoringUpdates["log_level"]; exists {
					if val, ok := logLevel.(string); ok {
						settings.MonitoringSettings.LogLevel = val
					}
				}
				if usageAnalytics, exists := monitoringUpdates["usage_analytics"]; exists {
					if val, ok := usageAnalytics.(bool); ok {
						settings.MonitoringSettings.UsageAnalytics = val
					}
				}
				if errorTracking, exists := monitoringUpdates["error_tracking"]; exists {
					if val, ok := errorTracking.(bool); ok {
						settings.MonitoringSettings.ErrorTracking = val
					}
				}
			}
		case "advanced-security":
			if advSecUpdates, ok := sectionData.(map[string]interface{}); ok {
				if threatDetection, exists := advSecUpdates["threat_detection"]; exists {
					if threatData, ok := threatDetection.(map[string]interface{}); ok {
						if enabled, exists := threatData["enabled"]; exists {
							if val, ok := enabled.(bool); ok {
								settings.AdvancedSecuritySettings.ThreatDetection.Enabled = val
							}
						}
					}
				}
			}
		}
	}

	// Update settings
	err = s.UpdateTenantSettings(ctx, tenantID, settings)
	if err != nil {
		return nil, err
	}

	return settings, nil
}

// CheckUsageLimits checks if tenant can perform a specific action
func (s *TenantManagementService) CheckUsageLimits(ctx context.Context, tenantID uuid.UUID, action string) (bool, string, error) {
	tenant, err := s.GetTenantByID(ctx, tenantID)
	if err != nil {
		return false, "Tenant not found", err
	}

	// Get current usage
	usage, err := s.GetTenantUsage(ctx, tenantID)
	if err != nil {
		return false, "Failed to get usage", err
	}

	// Check user limit
	if usage.UserCount >= tenant.MaxUsers {
		return false, fmt.Sprintf("User limit exceeded (%d/%d)", usage.UserCount, tenant.MaxUsers), nil
	}

	return true, "OK", nil
}

// TenantStats represents overall tenant statistics
type TenantStats struct {
	TotalTenants   int     `json:"total_tenants"`
	ActiveTenants  int     `json:"active_tenants"`
	TotalUsers     int     `json:"total_users"`
	AverageStorage float64 `json:"average_storage"`
}

// GetTenantStats gets overall tenant statistics
func (s *TenantManagementService) GetTenantStats(ctx context.Context) (*TenantStats, error) {
	query := `
		SELECT 
			COUNT(*) as total_tenants,
			COUNT(CASE WHEN is_active = true THEN 1 END) as active_tenants
		FROM tenants
	`

	var stats TenantStats
	err := s.db.QueryRowContext(ctx, query).Scan(&stats.TotalTenants, &stats.ActiveTenants)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant stats: %w", err)
	}

	return &stats, nil
}

// SetTenantStatus sets the active status of a tenant
func (s *TenantManagementService) SetTenantStatus(ctx context.Context, tenantID uuid.UUID, isActive bool) error {
	query := `UPDATE tenants SET is_active = $1, updated_at = NOW() WHERE tenant_id = $2`
	result, err := s.db.ExecContext(ctx, query, isActive, tenantID)
	if err != nil {
		return fmt.Errorf("failed to update tenant status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// generateSlug creates a URL-friendly slug from a tenant name
func (s *TenantManagementService) generateSlug(name string) string {
	// Convert to lowercase
	slug := strings.ToLower(name)

	// Replace spaces and special characters with hyphens
	re := regexp.MustCompile(`[^a-z0-9]+`)
	slug = re.ReplaceAllString(slug, "-")

	// Remove leading/trailing hyphens
	slug = strings.Trim(slug, "-")

	// Limit length to 50 characters (database constraint)
	if len(slug) > 50 {
		slug = slug[:50]
		// Remove trailing hyphen if truncation created one
		slug = strings.TrimSuffix(slug, "-")
	}

	return slug
}

// isValidSlug validates a tenant slug format
func isValidSlug(slug string) bool {
	if slug == "" || len(slug) > 63 {
		return false
	}

	// Must be lowercase alphanumeric with hyphens only
	// Must start and end with alphanumeric
	matched, _ := regexp.MatchString(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`, slug)
	return matched
}
