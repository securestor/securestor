package scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

// OSVScanner implements SBOM-based vulnerability scanning using OSV-Scanner
type OSVScanner struct {
	osvPath string
}

func NewOSVScanner() *OSVScanner {
	return &OSVScanner{
		osvPath: "osv-scanner",
	}
}

func (s *OSVScanner) Name() string {
	return "OSV-Scanner (SBOM-based CVE)"
}

func (s *OSVScanner) SupportedTypes() []string {
	return []string{"sbom", "generic", "maven", "npm", "pypi", "docker"}
}

func (s *OSVScanner) IsAvailable() bool {
	_, err := exec.LookPath(s.osvPath)
	return err == nil
}

func (s *OSVScanner) Supports(artifactType string) bool {
	supportedTypes := s.SupportedTypes()
	for _, t := range supportedTypes {
		if t == artifactType {
			return true
		}
	}
	return false
}

func (s *OSVScanner) Scan(ctx context.Context, artifactPath string, artifactType string) (*ScanResult, error) {
	startTime := time.Now()

	// OSV-Scanner with JSON output
	args := []string{
		"--format", "json",
		"--lockfile", artifactPath,
	}

	cmd := exec.CommandContext(ctx, s.osvPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// OSV-Scanner returns non-zero exit code when vulnerabilities are found
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			// This is expected when vulnerabilities are found
		} else {
			return nil, fmt.Errorf("osv-scanner command failed: %w, output: %s", err, string(output))
		}
	}

	vulnerabilities, err := s.parseOSVOutput(output)
	if err != nil {
		return nil, fmt.Errorf("failed to parse osv-scanner output: %w", err)
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
			"scan_type": "sbom_cve",
			"tool":      "osv-scanner",
		},
	}, nil
}

func (s *OSVScanner) getVersion() string {
	cmd := exec.Command(s.osvPath, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "unknown"
	}
	return string(output)
}

// OSVResult represents OSV-Scanner JSON output
type OSVResult struct {
	Results []OSVPackageResult `json:"results"`
}

type OSVPackageResult struct {
	Source   OSVSource    `json:"source"`
	Packages []OSVPackage `json:"packages"`
}

type OSVSource struct {
	Path string `json:"path"`
	Type string `json:"type"`
}

type OSVPackage struct {
	Package         OSVPackageInfo     `json:"package"`
	Vulnerabilities []OSVVulnerability `json:"vulnerabilities"`
}

type OSVPackageInfo struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Ecosystem string `json:"ecosystem"`
}

type OSVVulnerability struct {
	ID         string         `json:"id"`
	Summary    string         `json:"summary"`
	Details    string         `json:"details"`
	Severity   []OSVSeverity  `json:"severity"`
	Affected   []OSVAffected  `json:"affected"`
	References []OSVReference `json:"references"`
}

type OSVSeverity struct {
	Type  string `json:"type"`
	Score string `json:"score"`
}

type OSVAffected struct {
	Package          OSVPackageInfo         `json:"package"`
	Ranges           []OSVRange             `json:"ranges"`
	DatabaseSpecific map[string]interface{} `json:"database_specific"`
}

type OSVRange struct {
	Type   string     `json:"type"`
	Events []OSVEvent `json:"events"`
}

type OSVEvent struct {
	Introduced string `json:"introduced,omitempty"`
	Fixed      string `json:"fixed,omitempty"`
}

type OSVReference struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

func (s *OSVScanner) parseOSVOutput(output []byte) ([]Vulnerability, error) {
	var result OSVResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal OSV JSON: %w", err)
	}

	var vulnerabilities []Vulnerability

	for _, packageResult := range result.Results {
		for _, pkg := range packageResult.Packages {
			for _, vuln := range pkg.Vulnerabilities {
				// Extract CVSS score
				var cvssScore float64
				var severity string = "MEDIUM" // Default

				for _, sev := range vuln.Severity {
					if sev.Type == "CVSS_V3" {
						// Parse CVSS score from string like "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H"
						if len(sev.Score) > 10 {
							// Extract base score - simplified parsing
							if sev.Score[9:11] == "H/" {
								cvssScore = 7.5 // High severity approximation
								severity = "HIGH"
							}
						}
					}
				}

				// Extract fixed version
				var fixedVersion string
				for _, affected := range vuln.Affected {
					for _, rng := range affected.Ranges {
						for _, event := range rng.Events {
							if event.Fixed != "" {
								fixedVersion = event.Fixed
								break
							}
						}
					}
				}

				// Extract references
				var references []string
				for _, ref := range vuln.References {
					references = append(references, ref.URL)
				}

				vulnerability := Vulnerability{
					ID:          vuln.ID,
					Severity:    severity,
					Title:       vuln.Summary,
					Description: vuln.Details,
					Package:     pkg.Package.Name,
					Version:     pkg.Package.Version,
					FixedIn:     fixedVersion,
					CVSS:        cvssScore,
					References:  references,
				}

				vulnerabilities = append(vulnerabilities, vulnerability)
			}
		}
	}

	return vulnerabilities, nil
}
