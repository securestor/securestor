package scanner

import (
	"context"
	"encoding/json"
	"regexp"
	"time"
)

// Scanner defines the interface for vulnerability scanners
type Scanner interface {
	// Name returns the scanner name
	Name() string

	// SupportedTypes returns artifact types this scanner supports
	SupportedTypes() []string

	// Scan performs vulnerability scanning on an artifact
	Scan(ctx context.Context, artifactPath string, artifactType string) (*ScanResult, error)

	// IsAvailable checks if the scanner binary is available
	IsAvailable() bool

	// Supports checks if scanner supports given artifact type
	Supports(artifactType string) bool
}

// ScanResult represents the output of a vulnerability scan
type ScanResult struct {
	ScannerName     string                 `json:"scanner_name"`
	ScannerVersion  string                 `json:"scanner_version"`
	ArtifactType    string                 `json:"artifact_type"`
	Vulnerabilities []Vulnerability        `json:"vulnerabilities"`
	Summary         VulnerabilitySummary   `json:"summary"`
	ScanDuration    float64                `json:"scan_duration_seconds"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// Vulnerability represents a single vulnerability
type Vulnerability struct {
	ID          string   `json:"id"`            // Internal ID (CVE-2021-1234 or SECRET-Signable-1)
	CVE         string   `json:"cve,omitempty"` // CVE number if this is a CVE vulnerability
	Severity    string   `json:"severity"`      // CRITICAL, HIGH, MEDIUM, LOW
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Package     string   `json:"package"`
	Version     string   `json:"version"`
	FixedIn     string   `json:"fixed_in,omitempty"`
	References  []string `json:"references,omitempty"`
	CVSS        float64  `json:"cvss,omitempty"`
}

// VulnerabilitySummary provides a count of vulnerabilities by severity
type VulnerabilitySummary struct {
	Critical int `json:"critical"`
	High     int `json:"high"`
	Medium   int `json:"medium"`
	Low      int `json:"low"`
	Total    int `json:"total"`
}

// Orchestrator coordinates scanners for a job
type Orchestrator interface {
	Submit(ctx context.Context, job ScanJob) error
}

// PolicyClient queries OPA/Conftest to decide allow/quarantine/block
type PolicyClient interface {
	Evaluate(ctx context.Context, artifactMeta map[string]interface{}) (Decision, error)
}

// Decision is the OPA decision
type Decision struct {
	Allow  bool   `json:"allow"`
	Action string `json:"action"` // allow, warn, quarantine, block
	Reason string `json:"reason"`
}

// ScanJob represents a scan request
type ScanJob struct {
	JobID        string     `json:"job_id"`
	TenantID     string     `json:"tenant_id"`
	ArtifactID   string     `json:"artifact_id"`
	ArtifactType string     `json:"artifact_type"`
	ArtifactPath string     `json:"artifact_path"`
	ScannerNames []string   `json:"scanner_names,omitempty"`
	Priority     int        `json:"priority,omitempty"`
	RequestedAt  time.Time  `json:"requested_at,omitempty"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	Status       string     `json:"status,omitempty"` // pending, running, completed, failed
	Error        string     `json:"error,omitempty"`
}

// ScannerResult represents the result from a specific scanner
type ScannerResult struct {
	Tool      string                 `json:"tool"`
	OutputRaw json.RawMessage        `json:"output_raw"`
	Summary   map[string]interface{} `json:"summary"`
}

// ExtractCVE extracts CVE ID from vulnerability ID if it's a proper CVE
func ExtractCVE(id string) string {
	cvePattern := regexp.MustCompile(`^CVE-\d{4}-\d+$`)
	if cvePattern.MatchString(id) {
		return id
	}
	return ""
}
