package service

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"
)

// SecurityPolicy defines vulnerability scanning and blocking policies
type SecurityPolicy struct {
	ID                   string    `json:"id"`
	TenantID             string    `json:"tenant_id"`
	Name                 string    `json:"name"`
	Description          string    `json:"description"`
	CriticalThreshold    int       `json:"critical_threshold"`    // Block if critical count >= this
	HighThreshold        int       `json:"high_threshold"`        // Block if high count >= this
	MediumThreshold      int       `json:"medium_threshold"`      // Warn if medium count >= this
	AutoBlockEnabled     bool      `json:"auto_block_enabled"`    // Auto-block artifacts
	QuarantineEnabled    bool      `json:"quarantine_enabled"`    // Quarantine suspicious artifacts
	NotifyOnViolation    bool      `json:"notify_on_violation"`   // Send alerts
	RequiredScanners     []string  `json:"required_scanners"`     // e.g., ["trivy", "grype", "syft"]
	ExcludedArtifacts    []string  `json:"excluded_artifacts"`    // Patterns to skip scanning
	ComplianceFrameworks []string  `json:"compliance_frameworks"` // e.g., ["PCI-DSS", "HIPAA", "SOC2"]
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
	IsActive             bool      `json:"is_active"`
}

// VulnerabilityFinding represents a single security finding
type VulnerabilityFinding struct {
	ID               string      `json:"id"`
	ScanID           string      `json:"scan_id"`
	ArtifactID       string      `json:"artifact_id"`
	Severity         string      `json:"severity"` // CRITICAL, HIGH, MEDIUM, LOW
	CveID            string      `json:"cve_id"`   // CVE-XXXX-XXXXX
	Title            string      `json:"title"`
	Description      string      `json:"description"`
	PackageName      string      `json:"package_name"`
	InstalledVersion string      `json:"installed_version"`
	FixedVersion     string      `json:"fixed_version"`
	CVSSScore        float64     `json:"cvss_score"` // 0.0 - 10.0
	EPSSScore        float64     `json:"epss_score"` // 0.0 - 1.0 (Exploit Prediction Scoring System)
	SourceURL        string      `json:"source_url"`
	IsRemediatable   bool        `json:"is_remediatable"`
	RemediationPath  string      `json:"remediation_path"`
	DiscoveredAt     time.Time   `json:"discovered_at"`
	AcknowledgedAt   *time.Time  `json:"acknowledged_at"`
	ResolvedAt       *time.Time  `json:"resolved_at"`
	Metadata         interface{} `json:"metadata"`
}

// SecurityPolicyService manages security scanning policies
type SecurityPolicyService struct {
	db *sql.DB
}

// NewSecurityPolicyService creates a new security policy service
func NewSecurityPolicyService(db *sql.DB) *SecurityPolicyService {
	return &SecurityPolicyService{db: db}
}

// CreateSecurityPolicy creates a new security policy
func (s *SecurityPolicyService) CreateSecurityPolicy(ctx context.Context, policy *SecurityPolicy) error {
	if policy.TenantID == "" || policy.Name == "" {
		return fmt.Errorf("tenant_id and name are required")
	}

	query := `
		INSERT INTO proxy_security_policies (
			tenant_id, name, description, critical_threshold, high_threshold,
			medium_threshold, auto_block_enabled, quarantine_enabled,
			notify_on_violation, required_scanners, excluded_artifacts,
			compliance_frameworks, is_active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`

	err := s.db.QueryRowContext(ctx, query,
		policy.TenantID,
		policy.Name,
		policy.Description,
		policy.CriticalThreshold,
		policy.HighThreshold,
		policy.MediumThreshold,
		policy.AutoBlockEnabled,
		policy.QuarantineEnabled,
		policy.NotifyOnViolation,
		toStringArray(policy.RequiredScanners),
		toStringArray(policy.ExcludedArtifacts),
		toStringArray(policy.ComplianceFrameworks),
		policy.IsActive,
	).Scan(&policy.ID, &policy.CreatedAt, &policy.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create security policy: %w", err)
	}

	log.Printf("✅ Created security policy: %s (ID: %s)", policy.Name, policy.ID)
	return nil
}

// GetSecurityPolicy retrieves a security policy by ID
func (s *SecurityPolicyService) GetSecurityPolicy(ctx context.Context, policyID string) (*SecurityPolicy, error) {
	policy := &SecurityPolicy{}

	query := `
		SELECT id, tenant_id, name, description, critical_threshold, high_threshold,
		       medium_threshold, auto_block_enabled, quarantine_enabled,
		       notify_on_violation, required_scanners, excluded_artifacts,
		       compliance_frameworks, is_active, created_at, updated_at
		FROM proxy_security_policies WHERE id = $1
	`

	err := s.db.QueryRowContext(ctx, query, policyID).Scan(
		&policy.ID,
		&policy.TenantID,
		&policy.Name,
		&policy.Description,
		&policy.CriticalThreshold,
		&policy.HighThreshold,
		&policy.MediumThreshold,
		&policy.AutoBlockEnabled,
		&policy.QuarantineEnabled,
		&policy.NotifyOnViolation,
		&policy.RequiredScanners,
		&policy.ExcludedArtifacts,
		&policy.ComplianceFrameworks,
		&policy.IsActive,
		&policy.CreatedAt,
		&policy.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("security policy not found: %s", policyID)
	} else if err != nil {
		return nil, fmt.Errorf("failed to get security policy: %w", err)
	}

	return policy, nil
}

// ListSecurityPolicies lists all security policies for a tenant
func (s *SecurityPolicyService) ListSecurityPolicies(ctx context.Context, tenantID string) ([]*SecurityPolicy, error) {
	query := `
		SELECT id, tenant_id, name, description, critical_threshold, high_threshold,
		       medium_threshold, auto_block_enabled, quarantine_enabled,
		       notify_on_violation, required_scanners, excluded_artifacts,
		       compliance_frameworks, is_active, created_at, updated_at
		FROM proxy_security_policies
		WHERE tenant_id = $1 AND is_active = true
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list security policies: %w", err)
	}
	defer rows.Close()

	var policies []*SecurityPolicy
	for rows.Next() {
		policy := &SecurityPolicy{}
		err := rows.Scan(
			&policy.ID,
			&policy.TenantID,
			&policy.Name,
			&policy.Description,
			&policy.CriticalThreshold,
			&policy.HighThreshold,
			&policy.MediumThreshold,
			&policy.AutoBlockEnabled,
			&policy.QuarantineEnabled,
			&policy.NotifyOnViolation,
			&policy.RequiredScanners,
			&policy.ExcludedArtifacts,
			&policy.ComplianceFrameworks,
			&policy.IsActive,
			&policy.CreatedAt,
			&policy.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan policy row: %w", err)
		}
		policies = append(policies, policy)
	}

	return policies, rows.Err()
}

// EvaluateVulnerabilities evaluates if artifacts should be blocked based on policy
func (s *SecurityPolicyService) EvaluateVulnerabilities(ctx context.Context, policyID string, findings []*VulnerabilityFinding) (bool, []string, error) {
	policy, err := s.GetSecurityPolicy(ctx, policyID)
	if err != nil {
		return false, nil, err
	}

	if !policy.IsActive {
		return false, nil, nil
	}

	var criticalCount, highCount, mediumCount int
	var violations []string

	for _, finding := range findings {
		switch finding.Severity {
		case "CRITICAL":
			criticalCount++
		case "HIGH":
			highCount++
		case "MEDIUM":
			mediumCount++
		}
	}

	// Check thresholds
	if policy.CriticalThreshold > 0 && criticalCount >= policy.CriticalThreshold {
		violations = append(violations, fmt.Sprintf("Critical vulnerability threshold exceeded: %d >= %d", criticalCount, policy.CriticalThreshold))
	}

	if policy.HighThreshold > 0 && highCount >= policy.HighThreshold {
		violations = append(violations, fmt.Sprintf("High vulnerability threshold exceeded: %d >= %d", highCount, policy.HighThreshold))
	}

	if policy.MediumThreshold > 0 && mediumCount >= policy.MediumThreshold {
		violations = append(violations, fmt.Sprintf("Medium vulnerability threshold exceeded: %d >= %d", mediumCount, policy.MediumThreshold))
	}

	shouldBlock := len(violations) > 0 && policy.AutoBlockEnabled
	return shouldBlock, violations, nil
}

// RecordScanFinding records a vulnerability finding in the database
func (s *SecurityPolicyService) RecordScanFinding(ctx context.Context, finding *VulnerabilityFinding) error {
	query := `
		INSERT INTO proxy_security_scan_findings (
			scan_id, artifact_id, severity, cve_id, title, description,
			package_name, installed_version, fixed_version, cvss_score,
			epss_score, source_url, is_remediatable, discovered_at, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW(), $14)
		RETURNING id
	`

	err := s.db.QueryRowContext(ctx, query,
		finding.ScanID,
		finding.ArtifactID,
		finding.Severity,
		finding.CveID,
		finding.Title,
		finding.Description,
		finding.PackageName,
		finding.InstalledVersion,
		finding.FixedVersion,
		finding.CVSSScore,
		finding.EPSSScore,
		finding.SourceURL,
		finding.IsRemediatable,
		finding.Metadata,
	).Scan(&finding.ID)

	if err != nil {
		return fmt.Errorf("failed to record security finding: %w", err)
	}

	return nil
}

// GetVulnerabilitiesForArtifact retrieves all vulnerabilities for an artifact
func (s *SecurityPolicyService) GetVulnerabilitiesForArtifact(ctx context.Context, artifactID string) ([]*VulnerabilityFinding, error) {
	query := `
		SELECT id, scan_id, artifact_id, severity, cve_id, title, description,
		       package_name, installed_version, fixed_version, cvss_score, epss_score,
		       source_url, is_remediatable, discovered_at, acknowledged_at, resolved_at, metadata
		FROM proxy_security_scan_findings
		WHERE artifact_id = $1 AND resolved_at IS NULL
		ORDER BY cvss_score DESC, discovered_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, artifactID)
	if err != nil {
		return nil, fmt.Errorf("failed to get vulnerabilities: %w", err)
	}
	defer rows.Close()

	var findings []*VulnerabilityFinding
	for rows.Next() {
		finding := &VulnerabilityFinding{}
		err := rows.Scan(
			&finding.ID,
			&finding.ScanID,
			&finding.ArtifactID,
			&finding.Severity,
			&finding.CveID,
			&finding.Title,
			&finding.Description,
			&finding.PackageName,
			&finding.InstalledVersion,
			&finding.FixedVersion,
			&finding.CVSSScore,
			&finding.EPSSScore,
			&finding.SourceURL,
			&finding.IsRemediatable,
			&finding.DiscoveredAt,
			&finding.AcknowledgedAt,
			&finding.ResolvedAt,
			&finding.Metadata,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan finding row: %w", err)
		}
		findings = append(findings, finding)
	}

	return findings, rows.Err()
}

// AcknowledgeVulnerability marks a vulnerability as acknowledged
func (s *SecurityPolicyService) AcknowledgeVulnerability(ctx context.Context, findingID string) error {
	query := `
		UPDATE proxy_security_scan_findings
		SET acknowledged_at = NOW()
		WHERE id = $1 AND acknowledged_at IS NULL
	`

	result, err := s.db.ExecContext(ctx, query, findingID)
	if err != nil {
		return fmt.Errorf("failed to acknowledge vulnerability: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("vulnerability not found or already acknowledged: %s", findingID)
	}

	log.Printf("✅ Vulnerability acknowledged: %s", findingID)
	return nil
}

// ResolveVulnerability marks a vulnerability as resolved
func (s *SecurityPolicyService) ResolveVulnerability(ctx context.Context, findingID string) error {
	query := `
		UPDATE proxy_security_scan_findings
		SET resolved_at = NOW()
		WHERE id = $1 AND resolved_at IS NULL
	`

	result, err := s.db.ExecContext(ctx, query, findingID)
	if err != nil {
		return fmt.Errorf("failed to resolve vulnerability: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("vulnerability not found or already resolved: %s", findingID)
	}

	log.Printf("✅ Vulnerability resolved: %s", findingID)
	return nil
}

// Helper function to convert string slice to SQL array format
func toStringArray(arr []string) []interface{} {
	if len(arr) == 0 {
		return nil
	}
	result := make([]interface{}, len(arr))
	for i, v := range arr {
		result[i] = v
	}
	return result
}
