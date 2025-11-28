package api

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/securestor/securestor/internal/models"
	"github.com/securestor/securestor/internal/scanner"
)

// PolicyService handles OPA integration for SecureStor
type PolicyService struct {
	opaClient *scanner.OPAClient
	enabled   bool
	logger    Logger
}

// Logger interface for logging
type Logger interface {
	Printf(format string, v ...interface{})
}

// NewPolicyService creates a new policy service
func NewPolicyService(logger Logger) *PolicyService {
	enabled := os.Getenv("OPA_ENABLED") == "true"
	if !enabled {
		logger.Printf("OPA policy evaluation is disabled")
		return &PolicyService{enabled: false, logger: logger}
	}

	opaURL := os.Getenv("OPA_URL")
	if opaURL == "" {
		opaURL = "http://opa:8181"
	}

	opaClient := scanner.NewOPAClient(opaURL, "/v1/data/securestor/policy/result")
	logger.Printf("OPA policy service initialized with URL: %s", opaURL)

	return &PolicyService{
		opaClient: opaClient,
		enabled:   true,
		logger:    logger,
	}
}

// ArtifactPolicyInput represents the input data for OPA artifact policy evaluation
type ArtifactPolicyInput struct {
	Artifact        ArtifactInfo      `json:"artifact"`
	Vulnerabilities VulnerabilityInfo `json:"vulnerabilities"`
	ScanResults     ScanResultInfo    `json:"scan_results"`
	Repository      RepositoryInfo    `json:"repository"`
	User            UserInfo          `json:"user"`
}

// ArtifactInfo contains artifact metadata for policy evaluation
type ArtifactInfo struct {
	Name     string                 `json:"name"`
	Version  string                 `json:"version"`
	Type     string                 `json:"type"`
	Size     int64                  `json:"size"`
	License  *string                `json:"license,omitempty"`
	Tags     []string               `json:"tags"`
	Metadata map[string]interface{} `json:"metadata"`
}

// VulnerabilityInfo contains vulnerability scan results
type VulnerabilityInfo struct {
	Critical int `json:"critical"`
	High     int `json:"high"`
	Medium   int `json:"medium"`
	Low      int `json:"low"`
}

// ScanResultInfo contains security scan results
type ScanResultInfo struct {
	MalwareDetected  bool     `json:"malware_detected"`
	LicenseCompliant bool     `json:"license_compliant"`
	DependencyIssues []string `json:"dependency_issues"`
	LastScanned      string   `json:"last_scanned"`
}

// RepositoryInfo contains repository context
type RepositoryInfo struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	Type string    `json:"type"`
}

// UserInfo contains user context for access control
type UserInfo struct {
	ID       string   `json:"id"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
}

// PolicyDecision represents the result of OPA policy evaluation
type PolicyDecision struct {
	Allow     bool   `json:"allow"`
	Action    string `json:"action"` // allow, warn, quarantine, block
	RiskScore int    `json:"risk_score"`
	RiskLevel string `json:"risk_level"`
	Reason    string `json:"reason"`
	Timestamp int64  `json:"timestamp"`
}

// EvaluateArtifactPolicy evaluates OPA policy for artifact operations
func (ps *PolicyService) EvaluateArtifactPolicy(ctx context.Context, input ArtifactPolicyInput) (*PolicyDecision, error) {
	if !ps.enabled {
		// Default allow when OPA is disabled
		return &PolicyDecision{
			Allow:     true,
			Action:    "allow",
			RiskScore: 0,
			RiskLevel: "low",
			Reason:    "Policy evaluation disabled",
			Timestamp: time.Now().UnixNano(),
		}, nil
	}

	// Convert input to map for OPA
	inputMap := map[string]interface{}{
		"artifact":        input.Artifact,
		"vulnerabilities": input.Vulnerabilities,
		"scan_results":    input.ScanResults,
		"repository":      input.Repository,
		"user":            input.User,
	}

	ps.logger.Printf("Evaluating OPA policy for artifact: %s:%s", input.Artifact.Name, input.Artifact.Version)
	ps.logger.Printf("Vulnerability data: Critical=%d, High=%d, Medium=%d, Low=%d",
		input.Vulnerabilities.Critical, input.Vulnerabilities.High, input.Vulnerabilities.Medium, input.Vulnerabilities.Low)

	// Call OPA
	decision, err := ps.opaClient.Evaluate(ctx, inputMap)
	if err != nil {
		ps.logger.Printf("OPA policy evaluation failed: %v", err)
		return nil, fmt.Errorf("policy evaluation failed: %w", err)
	}

	// Convert OPA decision to our format
	policyDecision := &PolicyDecision{
		Allow:     decision.Allow,
		Action:    decision.Action,
		Reason:    decision.Reason,
		Timestamp: time.Now().UnixNano(),
	}

	// Extract additional fields if available
	if decision.Action != "" {
		policyDecision.Action = decision.Action
	}

	ps.logger.Printf("OPA policy decision for %s:%s - Allow: %t, Action: %s, Reason: %s",
		input.Artifact.Name, input.Artifact.Version,
		policyDecision.Allow, policyDecision.Action, policyDecision.Reason)

	return policyDecision, nil
}

// EvaluateRepositoryAccess evaluates OPA policy for repository access
func (ps *PolicyService) EvaluateRepositoryAccess(ctx context.Context, repositoryID uuid.UUID, userID string, operation string) (*PolicyDecision, error) {
	if !ps.enabled {
		return &PolicyDecision{
			Allow:     true,
			Action:    "allow",
			RiskScore: 0,
			RiskLevel: "low",
			Reason:    "Policy evaluation disabled",
			Timestamp: time.Now().UnixNano(),
		}, nil
	}

	input := map[string]interface{}{
		"repository": map[string]interface{}{
			"id": repositoryID,
		},
		"user": map[string]interface{}{
			"id": userID,
		},
		"operation": operation,
	}

	decision, err := ps.opaClient.Evaluate(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("repository access policy evaluation failed: %w", err)
	}

	return &PolicyDecision{
		Allow:     decision.Allow,
		Action:    decision.Action,
		Reason:    decision.Reason,
		Timestamp: time.Now().UnixNano(),
	}, nil
}

// ApplyPolicyDecision applies the policy decision to the artifact workflow
func (ps *PolicyService) ApplyPolicyDecision(decision *PolicyDecision, artifact *models.Artifact) error {
	if !ps.enabled {
		return nil
	}

	// Update artifact metadata with policy decision
	if artifact.Metadata == nil {
		artifact.Metadata = make(map[string]interface{})
	}

	artifact.Metadata["policy_evaluation"] = map[string]interface{}{
		"evaluated_at": time.Now().Format(time.RFC3339),
		"decision":     decision,
	}

	// Apply action-specific logic
	switch decision.Action {
	case "block":
		artifact.Metadata["status"] = "blocked"
		artifact.Metadata["blocked_reason"] = decision.Reason
		ps.logger.Printf("Artifact %s:%s blocked by policy: %s", artifact.Name, artifact.Version, decision.Reason)
		return fmt.Errorf("artifact blocked by security policy: %s", decision.Reason)

	case "quarantine":
		artifact.Metadata["status"] = "quarantined"
		artifact.Metadata["quarantine_reason"] = decision.Reason
		ps.logger.Printf("Artifact %s:%s quarantined by policy: %s", artifact.Name, artifact.Version, decision.Reason)

	case "warn":
		artifact.Metadata["status"] = "warning"
		artifact.Metadata["warning_reason"] = decision.Reason
		ps.logger.Printf("Artifact %s:%s flagged with warning: %s", artifact.Name, artifact.Version, decision.Reason)

	case "allow":
		artifact.Metadata["status"] = "approved"
		ps.logger.Printf("Artifact %s:%s approved by policy", artifact.Name, artifact.Version)

	default:
		artifact.Metadata["status"] = "unknown"
	}

	return nil
}

// CreateArtifactPolicyInput creates policy input from artifact and context
func CreateArtifactPolicyInput(artifact *models.Artifact, repository *models.Repository, userID string) ArtifactPolicyInput {
	// Get vulnerability info from artifact's Vulnerabilities field
	vuln := VulnerabilityInfo{}
	if artifact.Vulnerabilities != nil {
		vuln.Critical = artifact.Vulnerabilities.Critical
		vuln.High = artifact.Vulnerabilities.High
		vuln.Medium = artifact.Vulnerabilities.Medium
		vuln.Low = artifact.Vulnerabilities.Low
	}

	// If no vulnerability data from vulnerabilities table, try parsing from compliance.security_scan
	if vuln.Critical == 0 && vuln.High == 0 && vuln.Medium == 0 && vuln.Low == 0 && artifact.Compliance != nil && artifact.Compliance.SecurityScan != "" {
		vuln = parseVulnerabilityFromSecurityScan(artifact.Compliance.SecurityScan)
	}

	// Get scan results from artifact metadata if available
	scanResults := ScanResultInfo{
		MalwareDetected:  false,
		LicenseCompliant: true,
		DependencyIssues: []string{},
		LastScanned:      "",
	}
	if scanData, ok := artifact.Metadata["scan_results"].(map[string]interface{}); ok {
		if malware, ok := scanData["malware_detected"].(bool); ok {
			scanResults.MalwareDetected = malware
		}
		if license, ok := scanData["license_compliant"].(bool); ok {
			scanResults.LicenseCompliant = license
		}
		if deps, ok := scanData["dependency_issues"].([]interface{}); ok {
			for _, dep := range deps {
				if depStr, ok := dep.(string); ok {
					scanResults.DependencyIssues = append(scanResults.DependencyIssues, depStr)
				}
			}
		}
		if lastScanned, ok := scanData["last_scanned"].(string); ok {
			scanResults.LastScanned = lastScanned
		}
	}

	return ArtifactPolicyInput{
		Artifact: ArtifactInfo{
			Name:     artifact.Name,
			Version:  artifact.Version,
			Type:     artifact.Type,
			Size:     artifact.Size,
			License:  artifact.License,
			Tags:     artifact.Tags,
			Metadata: artifact.Metadata,
		},
		Vulnerabilities: vuln,
		ScanResults:     scanResults,
		Repository: RepositoryInfo{
			ID:   repository.ID,
			Name: repository.Name,
			Type: repository.Type,
		},
		User: UserInfo{
			ID:       userID,
			Username: userID,           // TODO: Get actual username from auth context
			Roles:    []string{"user"}, // TODO: Get actual roles from auth context
		},
	}
}

// parseVulnerabilityFromSecurityScan parses vulnerability counts from security scan string
// Expected format: "Critical: 13, High: 48, Medium: 10, Low: 0"
func parseVulnerabilityFromSecurityScan(securityScan string) VulnerabilityInfo {
	vuln := VulnerabilityInfo{}

	// Use regex to extract numbers for each severity level
	criticalRegex := regexp.MustCompile(`Critical:\s*(\d+)`)
	highRegex := regexp.MustCompile(`High:\s*(\d+)`)
	mediumRegex := regexp.MustCompile(`Medium:\s*(\d+)`)
	lowRegex := regexp.MustCompile(`Low:\s*(\d+)`)

	if matches := criticalRegex.FindStringSubmatch(securityScan); len(matches) > 1 {
		if val, err := strconv.Atoi(matches[1]); err == nil {
			vuln.Critical = val
		}
	}

	if matches := highRegex.FindStringSubmatch(securityScan); len(matches) > 1 {
		if val, err := strconv.Atoi(matches[1]); err == nil {
			vuln.High = val
		}
	}

	if matches := mediumRegex.FindStringSubmatch(securityScan); len(matches) > 1 {
		if val, err := strconv.Atoi(matches[1]); err == nil {
			vuln.Medium = val
		}
	}

	if matches := lowRegex.FindStringSubmatch(securityScan); len(matches) > 1 {
		if val, err := strconv.Atoi(matches[1]); err == nil {
			vuln.Low = val
		}
	}

	return vuln
}
