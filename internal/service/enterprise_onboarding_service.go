package service

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// EnterpriseOnboardingService handles automated enterprise tenant provisioning
type EnterpriseOnboardingService struct {
	db     *sql.DB
	logger *log.Logger
}

// OnboardingTemplate represents a pre-configured onboarding template
type OnboardingTemplate struct {
	TemplateID         string                   `json:"template_id"`
	Name               string                   `json:"name"`
	Description        string                   `json:"description"`
	Category           string                   `json:"category"` // healthcare, finance, government, saas
	Configuration      map[string]interface{}   `json:"configuration"`
	SecurityPolicies   []SecurityPolicyTemplate `json:"security_policies"`
	ComplianceSettings map[string]interface{}   `json:"compliance_settings"`
	BrandingSettings   map[string]interface{}   `json:"branding_settings"`
	SSOProviders       []SSOProviderTemplate    `json:"sso_providers"`
}

type SecurityPolicyTemplate struct {
	PolicyName string                 `json:"policy_name"`
	PolicyType string                 `json:"policy_type"`
	Rules      map[string]interface{} `json:"rules"`
}

type SSOProviderTemplate struct {
	Provider      string                 `json:"provider"`
	ProviderName  string                 `json:"provider_name"`
	Configuration map[string]interface{} `json:"configuration"`
}

// OnboardingRequest represents an enterprise onboarding request
type OnboardingRequest struct {
	TenantName             string                 `json:"tenant_name"`
	ContactEmail           string                 `json:"contact_email"`
	OrganizationSize       string                 `json:"organization_size"` // small, medium, large, enterprise
	Industry               string                 `json:"industry"`
	ComplianceRequirements []string               `json:"compliance_requirements"`
	TemplateID             string                 `json:"template_id,omitempty"`
	CustomConfiguration    map[string]interface{} `json:"custom_configuration,omitempty"`
	BrandingOptions        BrandingOptions        `json:"branding_options"`
	SSOConfiguration       []SSOConfiguration     `json:"sso_configuration,omitempty"`
	AdminUsers             []AdminUser            `json:"admin_users"`
	UseBulkImport          bool                   `json:"use_bulk_import"`
	BulkImportData         []BulkUserData         `json:"bulk_import_data,omitempty"`
}

type BrandingOptions struct {
	LogoURL        string            `json:"logo_url"`
	PrimaryColor   string            `json:"primary_color"`
	SecondaryColor string            `json:"secondary_color"`
	CompanyName    string            `json:"company_name"`
	CustomDomain   string            `json:"custom_domain"`
	CustomCSS      string            `json:"custom_css"`
	FaviconURL     string            `json:"favicon_url"`
	ThemeSettings  map[string]string `json:"theme_settings"`
}

type SSOConfiguration struct {
	Provider      string                 `json:"provider"`
	ProviderName  string                 `json:"provider_name"`
	MetadataURL   string                 `json:"metadata_url,omitempty"`
	EntityID      string                 `json:"entity_id,omitempty"`
	Configuration map[string]interface{} `json:"configuration"`
}

type AdminUser struct {
	Username    string   `json:"username"`
	Email       string   `json:"email"`
	FirstName   string   `json:"first_name"`
	LastName    string   `json:"last_name"`
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
}

type BulkUserData struct {
	Username    string   `json:"username"`
	Email       string   `json:"email"`
	FirstName   string   `json:"first_name"`
	LastName    string   `json:"last_name"`
	Department  string   `json:"department"`
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
}

// OnboardingProgress tracks the progress of an onboarding process
type OnboardingProgress struct {
	OnboardingID      string                 `json:"onboarding_id"`
	TenantID          string                 `json:"tenant_id"`
	Status            string                 `json:"status"` // pending, in_progress, completed, failed
	CurrentStep       string                 `json:"current_step"`
	TotalSteps        int                    `json:"total_steps"`
	CompletedSteps    int                    `json:"completed_steps"`
	StepDetails       map[string]interface{} `json:"step_details"`
	ErrorDetails      string                 `json:"error_details,omitempty"`
	StartedAt         time.Time              `json:"started_at"`
	CompletedAt       *time.Time             `json:"completed_at,omitempty"`
	EstimatedTimeLeft int                    `json:"estimated_time_left_minutes"`
}

// NewEnterpriseOnboardingService creates a new enterprise onboarding service
func NewEnterpriseOnboardingService(db *sql.DB, logger *log.Logger) *EnterpriseOnboardingService {
	return &EnterpriseOnboardingService{
		db:     db,
		logger: logger,
	}
}

// GetOnboardingTemplates retrieves available onboarding templates
func (e *EnterpriseOnboardingService) GetOnboardingTemplates() ([]OnboardingTemplate, error) {
	templates := []OnboardingTemplate{
		{
			TemplateID:  "healthcare-hipaa",
			Name:        "Healthcare HIPAA Compliant",
			Description: "HIPAA-compliant configuration for healthcare organizations",
			Category:    "healthcare",
			Configuration: map[string]interface{}{
				"encryption_required":   true,
				"audit_retention_years": 7,
				"data_residency":        "US",
				"backup_frequency":      "hourly",
			},
			SecurityPolicies: []SecurityPolicyTemplate{
				{
					PolicyName: "HIPAA Password Policy",
					PolicyType: "password",
					Rules: map[string]interface{}{
						"min_length":               14,
						"require_uppercase":        true,
						"require_lowercase":        true,
						"require_numbers":          true,
						"require_symbols":          true,
						"max_age_days":             60,
						"history_count":            24,
						"lockout_attempts":         3,
						"lockout_duration_minutes": 60,
					},
				},
				{
					PolicyName: "HIPAA Session Policy",
					PolicyType: "session",
					Rules: map[string]interface{}{
						"max_session_duration_hours": 4,
						"idle_timeout_minutes":       15,
						"concurrent_sessions_limit":  1,
						"require_mfa":                true,
						"ip_binding":                 true,
						"device_binding":             true,
					},
				},
			},
			ComplianceSettings: map[string]interface{}{
				"hipaa_enabled":             true,
				"audit_all_access":          true,
				"data_encryption":           "AES-256",
				"breach_notification":       true,
				"risk_assessment_frequency": "quarterly",
			},
			BrandingSettings: map[string]interface{}{
				"show_hipaa_notice": true,
				"compliance_footer": true,
				"security_badge":    true,
			},
		},
		{
			TemplateID:  "finance-sox",
			Name:        "Financial SOX Compliant",
			Description: "SOX-compliant configuration for financial institutions",
			Category:    "finance",
			Configuration: map[string]interface{}{
				"encryption_required":   true,
				"audit_retention_years": 7,
				"data_residency":        "US",
				"segregation_of_duties": true,
			},
			SecurityPolicies: []SecurityPolicyTemplate{
				{
					PolicyName: "SOX Password Policy",
					PolicyType: "password",
					Rules: map[string]interface{}{
						"min_length":               12,
						"require_uppercase":        true,
						"require_lowercase":        true,
						"require_numbers":          true,
						"require_symbols":          true,
						"max_age_days":             9,
						"history_count":            12,
						"lockout_attempts":         5,
						"lockout_duration_minutes": 3,
					},
				},
			},
			ComplianceSettings: map[string]interface{}{
				"sox_enabled":           true,
				"segregation_of_duties": true,
				"change_management":     true,
				"audit_trail_immutable": true,
			},
		},
		{
			TemplateID:  "government-fedramp",
			Name:        "Government FedRAMP",
			Description: "FedRAMP-compliant configuration for government agencies",
			Category:    "government",
			Configuration: map[string]interface{}{
				"encryption_required": true,
				"fips_140_2":          true,
				"data_residency":      "US",
				"security_controls":   "high",
			},
			SecurityPolicies: []SecurityPolicyTemplate{
				{
					PolicyName: "FedRAMP Password Policy",
					PolicyType: "password",
					Rules: map[string]interface{}{
						"min_length":               15,
						"require_uppercase":        true,
						"require_lowercase":        true,
						"require_numbers":          true,
						"require_symbols":          true,
						"max_age_days":             6,
						"history_count":            24,
						"lockout_attempts":         3,
						"lockout_duration_minutes": 6,
					},
				},
			},
			ComplianceSettings: map[string]interface{}{
				"fedramp_enabled":       true,
				"fips_140_2":            true,
				"security_controls":     "high",
				"continuous_monitoring": true,
			},
		},
		{
			TemplateID:  "enterprise-standard",
			Name:        "Enterprise Standard",
			Description: "Standard enterprise configuration with best practices",
			Category:    "enterprise",
			Configuration: map[string]interface{}{
				"encryption_required":   true,
				"audit_retention_years": 3,
				"backup_frequency":      "daily",
				"multi_region":          false,
			},
			SecurityPolicies: []SecurityPolicyTemplate{
				{
					PolicyName: "Enterprise Password Policy",
					PolicyType: "password",
					Rules: map[string]interface{}{
						"min_length":               12,
						"require_uppercase":        true,
						"require_lowercase":        true,
						"require_numbers":          true,
						"require_symbols":          true,
						"max_age_days":             9,
						"history_count":            12,
						"lockout_attempts":         5,
						"lockout_duration_minutes": 3,
					},
				},
			},
		},
	}

	return templates, nil
}

// StartOnboarding initiates the enterprise onboarding process
func (e *EnterpriseOnboardingService) StartOnboarding(request OnboardingRequest) (*OnboardingProgress, error) {
	onboardingID := uuid.New().String()

	// Insert onboarding record
	_, err := e.db.Exec(`
		INSERT INTO enterprise_onboarding (
			onboarding_id, tenant_name, contact_email, organization_size, 
			industry, compliance_requirements, template_id, 
			custom_configuration, branding_options, sso_configuration,
			admin_users, use_bulk_import, bulk_import_data,
			status, current_step, total_steps, completed_steps,
			started_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)`,
		onboardingID, request.TenantName, request.ContactEmail,
		request.OrganizationSize, request.Industry,
		pq.Array(request.ComplianceRequirements), request.TemplateID,
		jsonMarshal(request.CustomConfiguration),
		jsonMarshal(request.BrandingOptions),
		jsonMarshal(request.SSOConfiguration),
		jsonMarshal(request.AdminUsers),
		request.UseBulkImport,
		jsonMarshal(request.BulkImportData),
		"pending", "validation", 10, 0, time.Now(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create onboarding record: %w", err)
	}

	// Start the onboarding process asynchronously
	go e.processOnboarding(onboardingID, request)

	progress := &OnboardingProgress{
		OnboardingID:      onboardingID,
		Status:            "pending",
		CurrentStep:       "validation",
		TotalSteps:        10,
		CompletedSteps:    0,
		StartedAt:         time.Now(),
		EstimatedTimeLeft: 30,
	}

	return progress, nil
}

// processOnboarding handles the actual onboarding process
func (e *EnterpriseOnboardingService) processOnboarding(onboardingID string, request OnboardingRequest) {
	steps := []struct {
		name     string
		function func() error
	}{
		{"validation", func() error { return e.validateOnboardingRequest(request) }},
		{"tenant_creation", func() error { return e.createTenant(onboardingID, request) }},
		{"security_policies", func() error { return e.setupSecurityPolicies(onboardingID, request) }},
		{"compliance_setup", func() error { return e.setupCompliance(onboardingID, request) }},
		{"branding_setup", func() error { return e.setupBranding(onboardingID, request) }},
		{"sso_configuration", func() error { return e.setupSSO(onboardingID, request) }},
		{"admin_users", func() error { return e.createAdminUsers(onboardingID, request) }},
		{"bulk_import", func() error { return e.performBulkImport(onboardingID, request) }},
		{"finalization", func() error { return e.finalizeOnboarding(onboardingID) }},
		{"notification", func() error { return e.sendCompletionNotification(onboardingID, request) }},
	}

	for i, step := range steps {
		e.updateOnboardingProgress(onboardingID, step.name, i+1, len(steps))

		if err := step.function(); err != nil {
			e.updateOnboardingError(onboardingID, step.name, err.Error())
			e.logger.Printf("Onboarding %s failed at step %s: %v", onboardingID, step.name, err)
			return
		}

		time.Sleep(2 * time.Second) // Simulate processing time
	}

	e.updateOnboardingComplete(onboardingID)
}

// validateOnboardingRequest validates the onboarding request
func (e *EnterpriseOnboardingService) validateOnboardingRequest(request OnboardingRequest) error {
	if request.TenantName == "" {
		return fmt.Errorf("tenant name is required")
	}
	if request.ContactEmail == "" {
		return fmt.Errorf("contact email is required")
	}
	if len(request.AdminUsers) == 0 {
		return fmt.Errorf("at least one admin user is required")
	}

	// Check if tenant name already exists
	var exists bool
	err := e.db.QueryRow("SELECT EXISTS(SELECT 1 FROM tenants WHERE name = $1)", request.TenantName).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check tenant name availability: %w", err)
	}
	if exists {
		return fmt.Errorf("tenant name '%s' is already taken", request.TenantName)
	}

	return nil
}

// createTenant creates the new tenant
func (e *EnterpriseOnboardingService) createTenant(onboardingID string, request OnboardingRequest) error {
	// Create tenant and get the auto-generated ID
	var tenantID int
	err := e.db.QueryRow(`
		INSERT INTO tenants (name, slug, description, is_active)
		VALUES ($1, $2, $3, true)
		RETURNING id`,
		request.TenantName, strings.ToLower(strings.ReplaceAll(request.TenantName, " ", "-")), "Enterprise onboarded tenant",
	).Scan(&tenantID)
	if err != nil {
		return fmt.Errorf("failed to create tenant: %w", err)
	}

	// Update onboarding record with tenant ID
	_, err = e.db.Exec(`
		UPDATE enterprise_onboarding SET tenant_id = $1 WHERE onboarding_id = $2`,
		tenantID, onboardingID,
	)
	if err != nil {
		return fmt.Errorf("failed to update onboarding with tenant ID: %w", err)
	}

	return nil
}

// setupSecurityPolicies sets up security policies based on template or custom configuration
func (e *EnterpriseOnboardingService) setupSecurityPolicies(onboardingID string, request OnboardingRequest) error {
	tenantID, err := e.getTenantIDFromOnboarding(onboardingID)
	if err != nil {
		return err
	}

	// Get template if specified
	var policies []SecurityPolicyTemplate
	if request.TemplateID != "" {
		templates, err := e.GetOnboardingTemplates()
		if err != nil {
			return err
		}

		for _, template := range templates {
			if template.TemplateID == request.TemplateID {
				policies = template.SecurityPolicies
				break
			}
		}
	}

	// Create default admin user for policy creation
	adminUserID := uuid.New().String()

	// Create security policies
	for _, policy := range policies {
		_, err := e.db.Exec(`
			INSERT INTO security_policies_advanced (
				tenant_id, policy_name, policy_type, policy_rules, created_by
			) VALUES ($1, $2, $3, $4, $5)`,
			tenantID, policy.PolicyName, policy.PolicyType,
			jsonMarshal(policy.Rules), adminUserID,
		)
		if err != nil {
			return fmt.Errorf("failed to create security policy %s: %w", policy.PolicyName, err)
		}
	}

	return nil
}

// setupCompliance configures compliance settings
func (e *EnterpriseOnboardingService) setupCompliance(onboardingID string, request OnboardingRequest) error {
	tenantID, err := e.getTenantIDFromOnboarding(onboardingID)
	if err != nil {
		return err
	}

	// Get compliance settings from template
	var complianceSettings map[string]interface{}
	if request.TemplateID != "" {
		templates, err := e.GetOnboardingTemplates()
		if err != nil {
			return err
		}

		for _, template := range templates {
			if template.TemplateID == request.TemplateID {
				complianceSettings = template.ComplianceSettings
				break
			}
		}
	}

	if len(complianceSettings) > 0 {
		_, err := e.db.Exec(`
			UPDATE tenant_settings 
			SET compliance_settings = $1
			WHERE tenant_id = $2`,
			jsonMarshal(complianceSettings), tenantID,
		)
		if err != nil {
			return fmt.Errorf("failed to update compliance settings: %w", err)
		}
	}

	return nil
}

// setupBranding configures branding and white-label options
func (e *EnterpriseOnboardingService) setupBranding(onboardingID string, request OnboardingRequest) error {
	tenantID, err := e.getTenantIDFromOnboarding(onboardingID)
	if err != nil {
		return err
	}

	// Create branding configuration
	_, err = e.db.Exec(`
		INSERT INTO tenant_branding (
			tenant_id, logo_url, primary_color, secondary_color,
			company_name, custom_domain, custom_css, favicon_url,
			theme_settings, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		tenantID, request.BrandingOptions.LogoURL,
		request.BrandingOptions.PrimaryColor,
		request.BrandingOptions.SecondaryColor,
		request.BrandingOptions.CompanyName,
		request.BrandingOptions.CustomDomain,
		request.BrandingOptions.CustomCSS,
		request.BrandingOptions.FaviconURL,
		jsonMarshal(request.BrandingOptions.ThemeSettings),
		time.Now(),
	)
	if err != nil {
		return fmt.Errorf("failed to setup branding: %w", err)
	}

	return nil
}

// setupSSO configures single sign-on providers
func (e *EnterpriseOnboardingService) setupSSO(onboardingID string, request OnboardingRequest) error {
	tenantID, err := e.getTenantIDFromOnboarding(onboardingID)
	if err != nil {
		return err
	}

	for _, ssoConfig := range request.SSOConfiguration {
		_, err := e.db.Exec(`
			INSERT INTO sso_configurations (
				tenant_id, provider, provider_name, configuration, is_active
			) VALUES ($1, $2, $3, $4, $5)`,
			tenantID, ssoConfig.Provider, ssoConfig.ProviderName,
			jsonMarshal(ssoConfig.Configuration), true,
		)
		if err != nil {
			return fmt.Errorf("failed to setup SSO provider %s: %w", ssoConfig.ProviderName, err)
		}
	}

	return nil
}

// createAdminUsers creates initial admin users
func (e *EnterpriseOnboardingService) createAdminUsers(onboardingID string, request OnboardingRequest) error {
	tenantID, err := e.getTenantIDFromOnboarding(onboardingID)
	if err != nil {
		return err
	}

	for _, admin := range request.AdminUsers {
		userID := uuid.New().String()

		_, err := e.db.Exec(`
			INSERT INTO users (
				user_id, tenant_id, username, email, first_name, last_name, status
			) VALUES ($1, $2, $3, $4, $5, $6, 'active')`,
			userID, tenantID, admin.Username, admin.Email,
			admin.FirstName, admin.LastName,
		)
		if err != nil {
			return fmt.Errorf("failed to create admin user %s: %w", admin.Username, err)
		}

		// Assign admin role
		_, err = e.db.Exec(`
			INSERT INTO user_roles (user_id, role_id, tenant_id)
			SELECT $1, role_id, $2 FROM roles WHERE role_name = 'admin'`,
			userID, tenantID,
		)
		if err != nil {
			return fmt.Errorf("failed to assign admin role to user %s: %w", admin.Username, err)
		}
	}

	return nil
}

// performBulkImport handles bulk user import if requested
func (e *EnterpriseOnboardingService) performBulkImport(onboardingID string, request OnboardingRequest) error {
	if !request.UseBulkImport || len(request.BulkImportData) == 0 {
		return nil // Skip if not requested
	}

	tenantID, err := e.getTenantIDFromOnboarding(onboardingID)
	if err != nil {
		return err
	}

	for _, userData := range request.BulkImportData {
		userID := uuid.New().String()

		_, err := e.db.Exec(`
			INSERT INTO users (
				user_id, tenant_id, username, email, first_name, last_name, status
			) VALUES ($1, $2, $3, $4, $5, $6, 'active')`,
			userID, tenantID, userData.Username, userData.Email,
			userData.FirstName, userData.LastName,
		)
		if err != nil {
			e.logger.Printf("Failed to import user %s: %v", userData.Username, err)
			continue // Continue with other users
		}

		// Assign role if specified
		if userData.Role != "" {
			_, err = e.db.Exec(`
				INSERT INTO user_roles (user_id, role_id, tenant_id)
				SELECT $1, role_id, $2 FROM roles WHERE role_name = $3`,
				userID, tenantID, userData.Role,
			)
			if err != nil {
				e.logger.Printf("Failed to assign role %s to user %s: %v", userData.Role, userData.Username, err)
			}
		}
	}

	return nil
}

// finalizeOnboarding performs final setup steps
func (e *EnterpriseOnboardingService) finalizeOnboarding(onboardingID string) error {
	tenantID, err := e.getTenantIDFromOnboarding(onboardingID)
	if err != nil {
		return err
	}

	// Create initial tenant settings if not exists
	_, err = e.db.Exec(`
		INSERT INTO tenant_settings (tenant_id, security_settings, user_settings, storage_settings)
		VALUES ($1, '{}', '{}', '{}')
		ON CONFLICT (tenant_id) DO NOTHING`,
		tenantID,
	)
	if err != nil {
		return fmt.Errorf("failed to create tenant settings: %w", err)
	}

	// Log successful onboarding
	_, err = e.db.Exec(`
		INSERT INTO security_audit_events (
			tenant_id, event_type, event_category, severity, source,
			event_data, risk_score, retention_date
		) VALUES ($1, 'tenant_onboarded', 'admin', 'low', 'onboarding_system',
			$2, 0.0, $3)`,
		tenantID,
		jsonMarshal(map[string]interface{}{
			"onboarding_id": onboardingID,
			"completed_at":  time.Now(),
		}),
		time.Now().AddDate(7, 0, 0), // 7 year retention
	)

	return err
}

// sendCompletionNotification sends completion notification
func (e *EnterpriseOnboardingService) sendCompletionNotification(onboardingID string, request OnboardingRequest) error {
	// In a real implementation, this would send emails, Slack notifications, etc.
	e.logger.Printf("Onboarding completed for tenant: %s (ID: %s)", request.TenantName, onboardingID)
	return nil
}

// GetOnboardingProgress retrieves the current progress of an onboarding process
func (e *EnterpriseOnboardingService) GetOnboardingProgress(onboardingID string) (*OnboardingProgress, error) {
	var progress OnboardingProgress
	var stepDetailsJSON, errorDetails sql.NullString
	var completedAt sql.NullTime

	err := e.db.QueryRow(`
		SELECT onboarding_id, tenant_id, status, current_step, total_steps,
			   completed_steps, step_details, error_details, started_at,
			   completed_at, estimated_time_left
		FROM enterprise_onboarding 
		WHERE onboarding_id = $1`, onboardingID).Scan(
		&progress.OnboardingID, &progress.TenantID, &progress.Status,
		&progress.CurrentStep, &progress.TotalSteps, &progress.CompletedSteps,
		&stepDetailsJSON, &errorDetails, &progress.StartedAt,
		&completedAt, &progress.EstimatedTimeLeft,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get onboarding progress: %w", err)
	}

	if stepDetailsJSON.Valid {
		json.Unmarshal([]byte(stepDetailsJSON.String), &progress.StepDetails)
	}
	if errorDetails.Valid {
		progress.ErrorDetails = errorDetails.String
	}
	if completedAt.Valid {
		progress.CompletedAt = &completedAt.Time
	}

	return &progress, nil
}

// ParseBulkImportCSV parses CSV data for bulk user import
func (e *EnterpriseOnboardingService) ParseBulkImportCSV(csvData io.Reader) ([]BulkUserData, error) {
	reader := csv.NewReader(csvData)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("CSV must have at least a header row and one data row")
	}

	// Parse header
	header := records[0]
	var users []BulkUserData

	for i, record := range records[1:] {
		if len(record) != len(header) {
			e.logger.Printf("Skipping row %d: column count mismatch", i+2)
			continue
		}

		user := BulkUserData{}
		for j, value := range record {
			switch strings.ToLower(header[j]) {
			case "username":
				user.Username = value
			case "email":
				user.Email = value
			case "first_name", "firstname":
				user.FirstName = value
			case "last_name", "lastname":
				user.LastName = value
			case "department":
				user.Department = value
			case "role":
				user.Role = value
			case "permissions":
				if value != "" {
					user.Permissions = strings.Split(value, ";")
				}
			}
		}

		if user.Username != "" && user.Email != "" {
			users = append(users, user)
		}
	}

	return users, nil
}

// Helper functions

func (e *EnterpriseOnboardingService) getTenantIDFromOnboarding(onboardingID string) (string, error) {
	var tenantID string
	err := e.db.QueryRow("SELECT tenant_id FROM enterprise_onboarding WHERE onboarding_id = $1", onboardingID).Scan(&tenantID)
	if err != nil {
		return "", fmt.Errorf("failed to get tenant ID from onboarding: %w", err)
	}
	return tenantID, nil
}

func (e *EnterpriseOnboardingService) updateOnboardingProgress(onboardingID, step string, completed, total int) {
	estimatedTimeLeft := (total - completed) * 3 // 3 minutes per step

	_, err := e.db.Exec(`
		UPDATE enterprise_onboarding 
		SET status = 'in_progress', current_step = $1, completed_steps = $2,
			estimated_time_left = $3
		WHERE onboarding_id = $4`,
		step, completed, estimatedTimeLeft, onboardingID,
	)
	if err != nil {
		e.logger.Printf("Failed to update onboarding progress: %v", err)
	}
}

func (e *EnterpriseOnboardingService) updateOnboardingError(onboardingID, step, errorMsg string) {
	_, err := e.db.Exec(`
		UPDATE enterprise_onboarding 
		SET status = 'failed', current_step = $1, error_details = $2
		WHERE onboarding_id = $3`,
		step, errorMsg, onboardingID,
	)
	if err != nil {
		e.logger.Printf("Failed to update onboarding error: %v", err)
	}
}

func (e *EnterpriseOnboardingService) updateOnboardingComplete(onboardingID string) {
	_, err := e.db.Exec(`
		UPDATE enterprise_onboarding 
		SET status = 'completed', current_step = 'completed', 
			completed_steps = total_steps, completed_at = $1,
			estimated_time_left = 0
		WHERE onboarding_id = $2`,
		time.Now(), onboardingID,
	)
	if err != nil {
		e.logger.Printf("Failed to update onboarding completion: %v", err)
	}
}

func jsonMarshal(v interface{}) string {
	if v == nil {
		return "{}"
	}
	data, _ := json.Marshal(v)
	return string(data)
}
