package scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// DepScanScanner implements scanning using OWASP dep-scan
type DepScanScanner struct {
	depscanPath string
}

func NewDepScanScanner() *DepScanScanner {
	return &DepScanScanner{
		depscanPath: "depscan", // assumes depscan is in PATH
	}
}

func (s *DepScanScanner) Name() string {
	return "OWASP Dep-Scan"
}

func (s *DepScanScanner) SupportedTypes() []string {
	return []string{"npm", "maven", "pypi", "docker"}
}

func (s *DepScanScanner) IsAvailable() bool {
	_, err := exec.LookPath(s.depscanPath)
	return err == nil
}

func (s *DepScanScanner) Supports(artifactType string) bool {
	supportedTypes := s.SupportedTypes()
	for _, t := range supportedTypes {
		if t == artifactType {
			return true
		}
	}
	return false
}

func (s *DepScanScanner) Scan(ctx context.Context, artifactPath string, artifactType string) (*ScanResult, error) {
	startTime := time.Now()

	// Check if the artifact is an SBOM file (CycloneDX JSON)
	isSBOM := strings.HasSuffix(strings.ToLower(artifactPath), "bom.json") ||
		filepath.Base(artifactPath) == "bom.json"

	// Create a temporary reports directory
	reportsDir := filepath.Join(os.TempDir(), fmt.Sprintf("depscan-reports-%d", time.Now().UnixNano()))
	if err := os.MkdirAll(reportsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create reports directory: %w", err)
	}
	defer os.RemoveAll(reportsDir)

	var args []string
	if isSBOM {
		// Scan using SBOM file directly
		args = []string{
			"--bom", artifactPath,
			"--no-banner",
			"--reports-dir", reportsDir,
		}
	} else {
		// Traditional source code scanning
		args = []string{
			"--src", artifactPath,
			"--reports-dir", reportsDir,
			"--type", s.mapArtifactType(artifactType),
		}
	}

	// Use python3 -m depscan.cli for better compatibility
	pythonArgs := append([]string{"-m", "depscan.cli"}, args...)
	cmd := exec.CommandContext(ctx, "python3", pythonArgs...)

	// Run depscan (it returns non-zero even on success if vulnerabilities are found)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if it's just a non-zero exit code (common with vulnerabilities found)
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Exit codes 1-2 are normal when vulnerabilities are found
			if exitErr.ExitCode() > 2 {
				return nil, fmt.Errorf("depscan command failed with exit code %d: %s", exitErr.ExitCode(), string(output))
			}
		} else {
			return nil, fmt.Errorf("depscan command failed: %w, output: %s", err, string(output))
		}
	}

	// Look for the generated report file - when scanning with --bom, it creates depscan-bom.json
	reportFilePath := filepath.Join(reportsDir, "depscan-bom.json")
	if _, err := os.Stat(reportFilePath); os.IsNotExist(err) {
		// Try alternative name
		reportFilePath = filepath.Join(reportsDir, "depscan.json")
		if _, err := os.Stat(reportFilePath); os.IsNotExist(err) {
			// List all files in the reports directory for debugging
			files, _ := os.ReadDir(reportsDir)
			fileNames := make([]string, len(files))
			for i, f := range files {
				fileNames[i] = f.Name()
			}
			return nil, fmt.Errorf("no report file found in %s, found files: %v", reportsDir, fileNames)
		}
	}

	// Read the generated report file
	reportData, err := os.ReadFile(reportFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read report file %s: %w", reportFilePath, err)
	}

	fmt.Printf("[DEPSCAN] Read report file %s, size: %d bytes\n", reportFilePath, len(reportData))

	vulnerabilities, err := s.parseDepScanOutput(reportData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse depscan report: %w", err)
	}

	summary := calculateSummary(vulnerabilities)
	duration := time.Since(startTime).Seconds()

	return &ScanResult{
		ScannerName:     s.Name(),
		ScannerVersion:  "latest",
		ArtifactType:    artifactType,
		Vulnerabilities: vulnerabilities,
		Summary:         summary,
		ScanDuration:    duration,
	}, nil
}

func (s *DepScanScanner) mapArtifactType(artifactType string) string {
	typeMap := map[string]string{
		"npm":    "nodejs",
		"maven":  "java",
		"pypi":   "python",
		"docker": "container",
	}

	if mapped, ok := typeMap[artifactType]; ok {
		return mapped
	}
	return artifactType
}

func (s *DepScanScanner) parseDepScanOutput(reportData []byte) ([]Vulnerability, error) {
	// Depscan report format: newline-delimited JSON objects (NDJSON)
	// Each line is a separate vulnerability JSON object
	var vulnerabilities []Vulnerability

	// Split by newlines and parse each JSON object
	lines := strings.Split(string(reportData), "\n")

	fmt.Printf("[DEPSCAN PARSER] Processing %d lines from report\n", len(lines))

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var vuln struct {
			ID               string   `json:"id"`
			Package          string   `json:"package"`
			PackageType      string   `json:"package_type"`
			Version          string   `json:"version"`
			FixVersion       string   `json:"fix_version"`
			Severity         string   `json:"severity"`
			CVSSScore        string   `json:"cvss_score"` // Can be string or number
			ShortDescription string   `json:"short_description"`
			RelatedURLs      []string `json:"related_urls"`
		}

		if err := json.Unmarshal([]byte(line), &vuln); err != nil {
			// Skip invalid JSON lines
			fmt.Printf("[DEPSCAN PARSER] Line %d: Failed to parse JSON: %v\n", i+1, err)
			continue
		}

		if vuln.ID == "" {
			fmt.Printf("[DEPSCAN PARSER] Line %d: Skipping entry with empty ID\n", i+1)
			continue
		}

		fmt.Printf("[DEPSCAN PARSER] Line %d: Found %s (%s) in %s\n", i+1, vuln.ID, vuln.Severity, vuln.Package)

		// Extract package name from full package identifier
		packageName := vuln.Package
		if vuln.Package != "" {
			// Format: "com.fasterxml.jackson.core:jackson-databind"
			parts := strings.Split(vuln.Package, ":")
			if len(parts) > 1 {
				packageName = parts[len(parts)-1]
			}
		}

		// Parse CVSS score - it can be a string or number
		var cvssScore float64
		if vuln.CVSSScore != "" {
			if parsed, err := strconv.ParseFloat(vuln.CVSSScore, 64); err == nil {
				cvssScore = parsed
			}
		}

		vulnerabilities = append(vulnerabilities, Vulnerability{
			ID:          vuln.ID,
			Severity:    normalizeSeverity(vuln.Severity),
			Title:       vuln.ID + " in " + packageName,
			Description: vuln.ShortDescription,
			Package:     packageName,
			Version:     vuln.Version,
			FixedIn:     vuln.FixVersion,
			CVSS:        cvssScore,
			References:  vuln.RelatedURLs,
		})
	}

	fmt.Printf("[DEPSCAN PARSER] Successfully parsed %d vulnerabilities\n", len(vulnerabilities))
	return vulnerabilities, nil
}
