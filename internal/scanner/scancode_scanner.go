package scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ScancodeScanner implements license analysis using Scancode Toolkit
type ScancodeScanner struct {
	scancodePath string
}

func NewScancodeScanner() *ScancodeScanner {
	return &ScancodeScanner{
		scancodePath: "scancode",
	}
}

func (s *ScancodeScanner) Name() string {
	return "Scancode License Analyzer"
}

func (s *ScancodeScanner) SupportedTypes() []string {
	return []string{"generic", "maven", "npm", "pypi", "docker", "source"}
}

func (s *ScancodeScanner) IsAvailable() bool {
	_, err := exec.LookPath(s.scancodePath)
	return err == nil
}

func (s *ScancodeScanner) Supports(artifactType string) bool {
	supportedTypes := s.SupportedTypes()
	for _, t := range supportedTypes {
		if t == artifactType {
			return true
		}
	}
	return false
}

func (s *ScancodeScanner) Scan(ctx context.Context, artifactPath string, artifactType string) (*ScanResult, error) {
	startTime := time.Now()

	// Scancode with license detection and JSON output
	args := []string{
		"--license",
		"--copyright",
		"--json-pp", "-", // Output to stdout
		"--timeout", "60", // 60 second timeout per file
		artifactPath,
	}

	cmd := exec.CommandContext(ctx, s.scancodePath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("scancode command failed: %w, output: %s", err, string(output))
	}

	vulnerabilities, err := s.parseScancodeOutput(output)
	if err != nil {
		return nil, fmt.Errorf("failed to parse scancode output: %w", err)
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
			"scan_type": "license_analysis",
			"tool":      "scancode",
		},
	}, nil
}

func (s *ScancodeScanner) getVersion() string {
	cmd := exec.Command(s.scancodePath, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "unknown"
	}
	return string(output)
}

// ScancodeResult represents Scancode JSON output
type ScancodeResult struct {
	Headers []ScancodeHeader `json:"headers"`
	Files   []ScancodeFile   `json:"files"`
}

type ScancodeHeader struct {
	ToolName    string `json:"tool_name"`
	ToolVersion string `json:"tool_version"`
}

type ScancodeFile struct {
	Path       string              `json:"path"`
	Type       string              `json:"type"`
	Licenses   []ScancodeLicense   `json:"licenses"`
	Copyrights []ScancodeCopyright `json:"copyrights"`
}

type ScancodeLicense struct {
	Key            string   `json:"key"`
	Score          float64  `json:"score"`
	Name           string   `json:"name"`
	ShortName      string   `json:"short_name"`
	Category       string   `json:"category"`
	IsException    bool     `json:"is_exception"`
	IsUnknown      bool     `json:"is_unknown"`
	Owner          string   `json:"owner"`
	HomepageURL    string   `json:"homepage_url"`
	TextURL        string   `json:"text_url"`
	ReferenceURL   string   `json:"reference_url"`
	SpdxLicenseKey string   `json:"spdx_license_key"`
	TextUrls       []string `json:"text_urls"`
	OtherUrls      []string `json:"other_urls"`
}

type ScancodeCopyright struct {
	Value     string `json:"value"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
}

// Define risky license categories that should be flagged
var riskyLicenseCategories = map[string]string{
	"Copyleft":         "HIGH",
	"Copyleft Limited": "MEDIUM",
	"Commercial":       "HIGH",
	"Proprietary":      "CRITICAL",
	"Free Restricted":  "MEDIUM",
}

var riskyLicenses = map[string]string{
	"GPL-2.0":  "HIGH",
	"GPL-3.0":  "HIGH",
	"AGPL-3.0": "CRITICAL",
	"CC-BY-SA": "MEDIUM",
	"SSPL-1.0": "CRITICAL",
	"BUSL-1.1": "HIGH",
}

func (s *ScancodeScanner) parseScancodeOutput(output []byte) ([]Vulnerability, error) {
	var result ScancodeResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal scancode JSON: %w", err)
	}

	var vulnerabilities []Vulnerability

	for _, file := range result.Files {
		// Skip directories
		if file.Type == "directory" {
			continue
		}

		for _, license := range file.Licenses {
			// Skip unknown or very low confidence licenses
			if license.IsUnknown || license.Score < 50.0 {
				continue
			}

			// Determine severity based on license category and specific licenses
			severity := "LOW" // Default for permissive licenses

			if riskLevel, exists := riskyLicenses[license.SpdxLicenseKey]; exists {
				severity = riskLevel
			} else if riskLevel, exists := riskyLicenseCategories[license.Category]; exists {
				severity = riskLevel
			}

			// Only report licenses that pose compliance risks
			if severity == "LOW" {
				continue
			}

			vulnID := fmt.Sprintf("LICENSE-%s-%s",
				strings.ReplaceAll(license.Key, "-", "_"),
				strings.ReplaceAll(file.Path, "/", "_"))

			vulnerability := Vulnerability{
				ID:       vulnID,
				Severity: severity,
				Title:    fmt.Sprintf("License compliance risk: %s", license.Name),
				Description: fmt.Sprintf("File contains %s license (%s) which may pose compliance risks. Category: %s, Confidence: %.1f%%",
					license.Name, license.Key, license.Category, license.Score),
				Package:    file.Path,
				Version:    fmt.Sprintf("Confidence: %.1f%%", license.Score),
				CVSS:       0, // License issues don't have CVSS scores
				References: append([]string{license.HomepageURL, license.TextURL}, license.TextUrls...),
			}

			vulnerabilities = append(vulnerabilities, vulnerability)
		}
	}

	return vulnerabilities, nil
}
