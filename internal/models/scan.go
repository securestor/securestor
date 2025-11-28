package models

import (
	"time"

	"github.com/google/uuid"
)

// ScanResult represents a workflow-based scan result
type ScanResult struct {
	ID                 uuid.UUID              `json:"id"`
	ArtifactID         uuid.UUID              `json:"artifact_id"`
	ScanType           string                 `json:"scan_type"`
	Status             string                 `json:"status"`
	WorkflowName       string                 `json:"workflow_name"`
	StartTime          time.Time              `json:"start_time"`
	EndTime            *time.Time             `json:"end_time,omitempty"`
	Duration           *int64                 `json:"duration,omitempty"`
	VulnerabilityCount int                    `json:"vulnerability_count"`
	CriticalCount      int                    `json:"critical_count"`
	HighCount          int                    `json:"high_count"`
	MediumCount        int                    `json:"medium_count"`
	LowCount           int                    `json:"low_count"`
	Error              string                 `json:"error,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
}

// SecurityScan represents a security scan operation
type SecurityScan struct {
	ID                uuid.UUID              `json:"id"`
	TenantID          uuid.UUID              `json:"tenant_id"`
	ArtifactID        uuid.UUID              `json:"artifact_id"`
	Status            string                 `json:"status"`    // initiated, running, completed, failed, cancelled
	ScanType          string                 `json:"scan_type"` // full, quick, custom, bulk
	Priority          string                 `json:"priority"`  // low, normal, high, critical
	VulnerabilityScan bool                   `json:"vulnerability_scan"`
	MalwareScan       bool                   `json:"malware_scan"`
	LicenseScan       bool                   `json:"license_scan"`
	DependencyScan    bool                   `json:"dependency_scan"`
	InitiatedBy       *uuid.UUID             `json:"initiated_by,omitempty"`
	StartedAt         time.Time              `json:"started_at"`
	CompletedAt       *time.Time             `json:"completed_at,omitempty"`
	Duration          *int64                 `json:"duration,omitempty"` // seconds
	ErrorMessage      *string                `json:"error_message,omitempty"`
	Results           *ScanResults           `json:"results,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
}

// ScanResults contains the detailed results of a security scan
type ScanResults struct {
	ID                   uuid.UUID             `json:"id"`
	TenantID             uuid.UUID             `json:"tenant_id"`
	ScanID               uuid.UUID             `json:"scan_id"`
	OverallScore         int                   `json:"overall_score"` // 0-100
	RiskLevel            string                `json:"risk_level"`    // low, medium, high, critical
	VulnerabilityResults *VulnerabilityResults `json:"vulnerability_results,omitempty"`
	MalwareResults       *MalwareResults       `json:"malware_results,omitempty"`
	LicenseResults       *LicenseResults       `json:"license_results,omitempty"`
	DependencyResults    *DependencyResults    `json:"dependency_results,omitempty"`
	PolicyDecision       *PolicyDecision       `json:"policy_decision,omitempty"`
	Summary              string                `json:"summary"`
	Recommendations      []string              `json:"recommendations"`
	CreatedAt            time.Time             `json:"created_at"`
	UpdatedAt            time.Time             `json:"updated_at"`
}

// PolicyDecision represents OPA policy evaluation result
type PolicyDecision struct {
	Allow     bool   `json:"allow"`
	Action    string `json:"action"` // allow, warn, quarantine, block
	RiskScore int    `json:"risk_score"`
	RiskLevel string `json:"risk_level"`
	Reason    string `json:"reason"`
	Timestamp int64  `json:"timestamp"`
}

// VulnerabilityResults contains vulnerability scan results
type VulnerabilityResults struct {
	TotalFound      int                 `json:"total_found"`
	Critical        int                 `json:"critical"`
	High            int                 `json:"high"`
	Medium          int                 `json:"medium"`
	Low             int                 `json:"low"`
	Info            int                 `json:"info"`
	Fixed           int                 `json:"fixed"`
	Unfixed         int                 `json:"unfixed"`
	Vulnerabilities []VulnerabilityItem `json:"vulnerabilities"`
}

// VulnerabilityItem represents a single vulnerability
type VulnerabilityItem struct {
	ID            string    `json:"id"`
	CVE           string    `json:"cve,omitempty"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	Severity      string    `json:"severity"`
	Score         float64   `json:"score,omitempty"`
	Vector        string    `json:"vector,omitempty"`
	Component     string    `json:"component"`
	Version       string    `json:"version,omitempty"`
	FixedVersion  string    `json:"fixed_version,omitempty"`
	PublishedDate time.Time `json:"published_date,omitempty"`
	ModifiedDate  time.Time `json:"modified_date,omitempty"`
	References    []string  `json:"references,omitempty"`
	Tags          []string  `json:"tags,omitempty"`
	Status        string    `json:"status"` // open, fixed, ignored, false_positive
}

// VulnerabilityRecord represents a vulnerability record for workflow-based scans
type VulnerabilityRecord struct {
	ID          int64     `json:"id"`
	ArtifactID  int64     `json:"artifact_id"`
	ScanID      int64     `json:"scan_id"`
	CVE         string    `json:"cve"`
	Severity    string    `json:"severity"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Package     string    `json:"package"`
	Version     string    `json:"version"`
	FixedIn     string    `json:"fixed_in"`
	CVSS        float64   `json:"cvss"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// MalwareResults contains malware scan results
type MalwareResults struct {
	TotalScanned    int                `json:"total_scanned"`
	ThreatsFound    int                `json:"threats_found"`
	CleanFiles      int                `json:"clean_files"`
	SuspiciousFiles int                `json:"suspicious_files"`
	InfectedFiles   int                `json:"infected_files"`
	Threats         []MalwareThreat    `json:"threats"`
	ScanEngines     []ScanEngineResult `json:"scan_engines"`
}

// MalwareThreat represents a detected malware threat
type MalwareThreat struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"` // virus, trojan, rootkit, adware, etc.
	Severity    string    `json:"severity"`
	FilePath    string    `json:"file_path"`
	FileHash    string    `json:"file_hash"`
	DetectedBy  []string  `json:"detected_by"`
	Quarantined bool      `json:"quarantined"`
	DetectedAt  time.Time `json:"detected_at"`
}

// ScanEngineResult represents results from individual scan engines
type ScanEngineResult struct {
	Name       string `json:"name"`
	Version    string `json:"version"`
	Result     string `json:"result"` // clean, infected, suspicious, error
	ThreatName string `json:"threat_name,omitempty"`
	UpdateDate string `json:"update_date,omitempty"`
}

// LicenseResults contains license compliance scan results
type LicenseResults struct {
	TotalLicenses        int               `json:"total_licenses"`
	CompliantLicenses    int               `json:"compliant_licenses"`
	NonCompliantLicenses int               `json:"non_compliant_licenses"`
	UnknownLicenses      int               `json:"unknown_licenses"`
	RiskLevel            string            `json:"risk_level"`
	Licenses             []LicenseItem     `json:"licenses"`
	PolicyViolations     []PolicyViolation `json:"policy_violations"`
}

// LicenseItem represents a detected license
type LicenseItem struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	SPDXID      string   `json:"spdx_id,omitempty"`
	Type        string   `json:"type"` // permissive, copyleft, proprietary, etc.
	RiskLevel   string   `json:"risk_level"`
	Components  []string `json:"components"`
	FilePaths   []string `json:"file_paths"`
	Compliant   bool     `json:"compliant"`
	Obligations []string `json:"obligations,omitempty"`
}

// PolicyViolation represents a license policy violation
type PolicyViolation struct {
	ID          string   `json:"id"`
	Type        string   `json:"type"` // forbidden_license, missing_license, etc.
	Severity    string   `json:"severity"`
	Description string   `json:"description"`
	Component   string   `json:"component"`
	License     string   `json:"license"`
	Actions     []string `json:"actions"`
}

// DependencyResults contains dependency analysis results
type DependencyResults struct {
	TotalDependencies      int                `json:"total_dependencies"`
	OutdatedDependencies   int                `json:"outdated_dependencies"`
	VulnerableDependencies int                `json:"vulnerable_dependencies"`
	UnknownDependencies    int                `json:"unknown_dependencies"`
	Dependencies           []DependencyItem   `json:"dependencies"`
	SecurityAdvisories     []SecurityAdvisory `json:"security_advisories"`
}

// DependencyItem represents a single dependency
type DependencyItem struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	CurrentVersion  string   `json:"current_version"`
	LatestVersion   string   `json:"latest_version,omitempty"`
	Type            string   `json:"type"`      // direct, transitive
	Ecosystem       string   `json:"ecosystem"` // npm, maven, pypi, etc.
	License         string   `json:"license,omitempty"`
	Vulnerabilities int      `json:"vulnerabilities"`
	IsOutdated      bool     `json:"is_outdated"`
	RiskScore       int      `json:"risk_score"`
	UsedBy          []string `json:"used_by,omitempty"`
}

// SecurityAdvisory represents a security advisory for dependencies
type SecurityAdvisory struct {
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	Severity        string    `json:"severity"`
	Package         string    `json:"package"`
	VulnerableRange string    `json:"vulnerable_range"`
	PatchedVersions []string  `json:"patched_versions"`
	PublishedAt     time.Time `json:"published_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	References      []string  `json:"references"`
}

// ScanConfig represents scan configuration
type ScanConfig struct {
	VulnerabilityScan bool     `json:"vulnerability_scan"`
	MalwareScan       bool     `json:"malware_scan"`
	LicenseScan       bool     `json:"license_scan"`
	DependencyScan    bool     `json:"dependency_scan"`
	Priority          string   `json:"priority"`
	Timeout           int      `json:"timeout,omitempty"` // minutes
	ScanEngines       []string `json:"scan_engines,omitempty"`
	ExcludePatterns   []string `json:"exclude_patterns,omitempty"`
}

// ScannerInfo represents information about available scanners
type ScannerInfo struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Type         string                 `json:"type"` // vulnerability, malware, license, dependency
	Version      string                 `json:"version"`
	Enabled      bool                   `json:"enabled"`
	Status       string                 `json:"status"` // healthy, unhealthy, unknown
	LastUpdate   time.Time              `json:"last_update"`
	Capabilities []string               `json:"capabilities"`
	Config       map[string]interface{} `json:"config,omitempty"`
}

// ScannerHealth represents health status of a scanner
type ScannerHealth struct {
	ScannerID    string    `json:"scanner_id"`
	Name         string    `json:"name"`
	Status       string    `json:"status"` // healthy, unhealthy, disabled
	LastCheck    time.Time `json:"last_check"`
	ErrorMessage string    `json:"error_message,omitempty"`
	ResponseTime int64     `json:"response_time"` // milliseconds
}

// ScanStatistics represents scan statistics
type ScanStatistics struct {
	TotalScans           int64            `json:"total_scans"`
	CompletedScans       int64            `json:"completed_scans"`
	FailedScans          int64            `json:"failed_scans"`
	ActiveScans          int64            `json:"active_scans"`
	AverageTime          float64          `json:"average_time"` // minutes
	ScansByType          map[string]int64 `json:"scans_by_type"`
	ScansByPriority      map[string]int64 `json:"scans_by_priority"`
	VulnerabilitiesFound int64            `json:"vulnerabilities_found"`
	ThreatsDetected      int64            `json:"threats_detected"`
	PolicyViolations     int64            `json:"policy_violations"`
}

// Severity levels for vulnerabilities
const (
	SeverityCritical = "critical"
	SeverityHigh     = "high"
	SeverityMedium   = "medium"
	SeverityLow      = "low"
	SeverityInfo     = "info"
)

// VulnerabilityDetail represents a detailed security vulnerability detected during scanning
type VulnerabilityDetail struct {
	VulnerabilityID uuid.UUID  `json:"vulnerability_id"`
	TenantID        uuid.UUID  `json:"tenant_id"`
	ArtifactID      uuid.UUID  `json:"artifact_id"`
	ScanID          *uuid.UUID `json:"scan_id,omitempty"`
	CVEID           *string    `json:"cve_id,omitempty"`
	Severity        string     `json:"severity"` // critical, high, medium, low, info
	Title           string     `json:"title"`
	Description     *string    `json:"description,omitempty"`
	AffectedPackage *string    `json:"affected_package,omitempty"`
	AffectedVersion *string    `json:"affected_version,omitempty"`
	FixedVersion    *string    `json:"fixed_version,omitempty"`
	CVSSScore       *float64   `json:"cvss_score,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// VulnerabilityDetailList represents paginated vulnerability results
type VulnerabilityDetailList struct {
	Total           int64                 `json:"total"`
	Page            int                   `json:"page"`
	PageSize        int                   `json:"page_size"`
	Vulnerabilities []VulnerabilityDetail `json:"vulnerabilities"`
}

// VulnerabilityDetailSummary represents vulnerability statistics
type VulnerabilityDetailSummary struct {
	TotalCount    int64 `json:"total_count"`
	CriticalCount int64 `json:"critical_count"`
	HighCount     int64 `json:"high_count"`
	MediumCount   int64 `json:"medium_count"`
	LowCount      int64 `json:"low_count"`
	InfoCount     int64 `json:"info_count"`
}

// ArtifactVulnerabilityReport contains vulnerability data for an artifact
type ArtifactVulnerabilityReport struct {
	ArtifactID      uuid.UUID                  `json:"artifact_id"`
	TenantID        uuid.UUID                  `json:"tenant_id"`
	Summary         VulnerabilityDetailSummary `json:"summary"`
	Vulnerabilities []VulnerabilityDetail      `json:"vulnerabilities"`
	LastScanTime    *time.Time                 `json:"last_scan_time,omitempty"`
	HighestSeverity string                     `json:"highest_severity"`
}

// VulnerabilityDetailFilter represents filter criteria for vulnerability queries
type VulnerabilityDetailFilter struct {
	TenantID        uuid.UUID
	ArtifactID      *uuid.UUID
	ScanID          *uuid.UUID
	Severity        *string
	CVEID           *string
	AffectedPackage *string
	MinCVSSScore    *float64
	MaxCVSSScore    *float64
	Page            int
	PageSize        int
	SortBy          string // created_at, severity, cvss_score
	SortOrder       string // asc, desc
}

// CreateVulnerabilityDetailRequest represents the request to create a vulnerability
type CreateVulnerabilityDetailRequest struct {
	ArtifactID      uuid.UUID  `json:"artifact_id" binding:"required"`
	ScanID          *uuid.UUID `json:"scan_id"`
	CVEID           *string    `json:"cve_id"`
	Severity        string     `json:"severity" binding:"required"`
	Title           string     `json:"title" binding:"required"`
	Description     *string    `json:"description"`
	AffectedPackage *string    `json:"affected_package"`
	AffectedVersion *string    `json:"affected_version"`
	FixedVersion    *string    `json:"fixed_version"`
	CVSSScore       *float64   `json:"cvss_score"`
}

// UpdateVulnerabilityDetailRequest represents the request to update a vulnerability
type UpdateVulnerabilityDetailRequest struct {
	Severity        *string  `json:"severity"`
	Title           *string  `json:"title"`
	Description     *string  `json:"description"`
	AffectedPackage *string  `json:"affected_package"`
	AffectedVersion *string  `json:"affected_version"`
	FixedVersion    *string  `json:"fixed_version"`
	CVSSScore       *float64 `json:"cvss_score"`
}

// BulkCreateVulnerabilityDetailsRequest represents request to create multiple vulnerabilities
type BulkCreateVulnerabilityDetailsRequest struct {
	ArtifactID      uuid.UUID                          `json:"artifact_id" binding:"required"`
	ScanID          *uuid.UUID                         `json:"scan_id"`
	Vulnerabilities []CreateVulnerabilityDetailRequest `json:"vulnerabilities" binding:"required"`
}
