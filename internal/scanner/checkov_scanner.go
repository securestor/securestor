package scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

// CheckovScanner implements infrastructure as code security scanning using Checkov
type CheckovScanner struct {
	checkovPath string
}

func NewCheckovScanner() *CheckovScanner {
	return &CheckovScanner{
		checkovPath: "checkov",
	}
}

func (s *CheckovScanner) Name() string {
	return "Checkov IaC Security Scanner"
}

func (s *CheckovScanner) SupportedTypes() []string {
	return []string{"terraform", "cloudformation", "kubernetes", "dockerfile", "helm", "yaml", "json", "generic"}
}

func (s *CheckovScanner) IsAvailable() bool {
	// Check if checkov is available as a Python module
	cmd := exec.Command("python3", "-c", "import checkov")
	err := cmd.Run()
	if err == nil {
		return true
	}

	// Fallback: check if checkov binary is available
	_, err = exec.LookPath(s.checkovPath)
	return err == nil
}

func (s *CheckovScanner) Supports(artifactType string) bool {
	supportedTypes := s.SupportedTypes()
	for _, t := range supportedTypes {
		if t == artifactType {
			return true
		}
	}
	return false
}

func (s *CheckovScanner) Scan(ctx context.Context, artifactPath string, artifactType string) (*ScanResult, error) {
	startTime := time.Now()

	// Checkov scan with JSON output
	args := []string{
		"-f", artifactPath,
		"-o", "json",
		"--quiet",
	}

	// Try python module first, then binary
	var cmd *exec.Cmd
	if s.isAvailableAsPythonModule() {
		pythonArgs := append([]string{"-m", "checkov.main"}, args...)
		cmd = exec.CommandContext(ctx, "python3", pythonArgs...)
	} else {
		cmd = exec.CommandContext(ctx, s.checkovPath, args...)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Checkov returns non-zero exit code when issues are found
		if exitErr, ok := err.(*exec.ExitError); ok && (exitErr.ExitCode() == 1 || exitErr.ExitCode() == 2) {
			// This is expected when issues are found
		} else {
			return nil, fmt.Errorf("checkov command failed: %w, output: %s", err, string(output))
		}
	}

	vulnerabilities, err := s.parseCheckovOutput(output)
	if err != nil {
		return nil, fmt.Errorf("failed to parse checkov output: %w", err)
	}

	summary := calculateSummary(vulnerabilities)
	duration := time.Since(startTime).Seconds()

	return &ScanResult{
		ScannerName:     s.Name(),
		ScannerVersion:  s.getVersion(),
		ArtifactType:    artifactType,
		Vulnerabilities: vulnerabilities,
		Summary:         summary,
		ScanDuration:    duration,
		Metadata: map[string]interface{}{
			"scan_type": "iac_security",
			"tool":      "checkov",
		},
	}, nil
}

func (s *CheckovScanner) isAvailableAsPythonModule() bool {
	cmd := exec.Command("python3", "-c", "import checkov")
	return cmd.Run() == nil
}

func (s *CheckovScanner) getVersion() string {
	var cmd *exec.Cmd
	if s.isAvailableAsPythonModule() {
		cmd = exec.Command("python3", "-m", "checkov.main", "--version")
	} else {
		cmd = exec.Command(s.checkovPath, "--version")
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "unknown"
	}
	return string(output)
}

// CheckovResult represents Checkov JSON output
type CheckovResult struct {
	Results CheckovResults `json:"results"`
}

type CheckovResults struct {
	FailedChecks  []CheckovCheck `json:"failed_checks"`
	PassedChecks  []CheckovCheck `json:"passed_checks"`
	SkippedChecks []CheckovCheck `json:"skipped_checks"`
}

type CheckovCheck struct {
	CheckId             string      `json:"check_id"`
	CheckName           string      `json:"check_name"`
	CheckResult         string      `json:"check_result"`
	CodeBlock           [][]string  `json:"code_block"`
	FilePath            string      `json:"file_path"`
	FileLineRange       []int       `json:"file_line_range"`
	Resource            string      `json:"resource"`
	Evaluations         interface{} `json:"evaluations"`
	CheckClass          string      `json:"check_class"`
	FixedDefinition     interface{} `json:"fixed_definition"`
	CallerFilePath      string      `json:"caller_file_path,omitempty"`
	CallerFileLineRange []int       `json:"caller_file_line_range,omitempty"`
}

// Map Checkov check types to severity levels
var checkovSeverityMap = map[string]string{
	"CKV_DOCKER_": "MEDIUM",
	"CKV_K8S_":    "HIGH",
	"CKV_AWS_":    "HIGH",
	"CKV_AZURE_":  "HIGH",
	"CKV_GCP_":    "HIGH",
	"CKV_TF_":     "MEDIUM",
	"CKV_SECRET_": "CRITICAL",
}

func (s *CheckovScanner) parseCheckovOutput(output []byte) ([]Vulnerability, error) {
	var result CheckovResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal checkov JSON: %w", err)
	}

	var vulnerabilities []Vulnerability

	for _, check := range result.Results.FailedChecks {
		// Determine severity based on check ID prefix
		severity := "MEDIUM" // Default
		for prefix, sev := range checkovSeverityMap {
			if len(check.CheckId) >= len(prefix) && check.CheckId[:len(prefix)] == prefix {
				severity = sev
				break
			}
		}

		// Special case for secret-related checks
		if check.CheckClass == "SecretScanner" {
			severity = "CRITICAL"
		}

		vulnerability := Vulnerability{
			ID:          check.CheckId,
			Severity:    severity,
			Title:       check.CheckName,
			Description: fmt.Sprintf("Infrastructure security check failed: %s. Resource: %s", check.CheckName, check.Resource),
			Package:     check.FilePath,
			Version:     fmt.Sprintf("Lines %d-%d", check.FileLineRange[0], check.FileLineRange[1]),
			CVSS:        0, // IaC issues don't have CVSS scores
			References:  []string{fmt.Sprintf("https://docs.bridgecrew.io/docs/%s", check.CheckId)},
		}

		vulnerabilities = append(vulnerabilities, vulnerability)
	}

	return vulnerabilities, nil
}
