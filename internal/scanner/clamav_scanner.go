package scanner

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ClamAVScanner implements malware detection using ClamAV
type ClamAVScanner struct {
	clamscanPath string
}

func NewClamAVScanner() *ClamAVScanner {
	return &ClamAVScanner{
		clamscanPath: "clamscan",
	}
}

func (s *ClamAVScanner) Name() string {
	return "ClamAV Malware Scanner"
}

func (s *ClamAVScanner) SupportedTypes() []string {
	return []string{"generic", "binary", "docker", "executable", "archive"}
}

func (s *ClamAVScanner) IsAvailable() bool {
	_, err := exec.LookPath(s.clamscanPath)
	return err == nil
}

func (s *ClamAVScanner) Supports(artifactType string) bool {
	supportedTypes := s.SupportedTypes()
	for _, t := range supportedTypes {
		if t == artifactType {
			return true
		}
	}
	return false
}

func (s *ClamAVScanner) Scan(ctx context.Context, artifactPath string, artifactType string) (*ScanResult, error) {
	startTime := time.Now()

	// ClamAV scan with verbose output
	args := []string{
		"--recursive",
		"--infected", // Only show infected files
		"--no-summary",
		artifactPath,
	}

	cmd := exec.CommandContext(ctx, s.clamscanPath, args...)
	output, err := cmd.CombinedOutput()

	// ClamAV returns exit code 1 when infections are found, which is expected
	var hasInfections bool
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			hasInfections = true // Infections found
		} else {
			return nil, fmt.Errorf("clamscan command failed: %w, output: %s", err, string(output))
		}
	}

	vulnerabilities, err := s.parseClamAVOutput(output, hasInfections)
	if err != nil {
		return nil, fmt.Errorf("failed to parse clamav output: %w", err)
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
			"scan_type":      "malware_detection",
			"tool":           "clamav",
			"has_infections": hasInfections,
		},
	}, nil
}

func (s *ClamAVScanner) getVersion() string {
	cmd := exec.Command(s.clamscanPath, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "unknown"
	}
	return string(output)
}

func (s *ClamAVScanner) parseClamAVOutput(output []byte, hasInfections bool) ([]Vulnerability, error) {
	var vulnerabilities []Vulnerability

	if !hasInfections {
		// No infections found, return empty list
		return vulnerabilities, nil
	}

	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// ClamAV output format: "/path/to/file: VirusName FOUND"
		if strings.Contains(line, " FOUND") {
			parts := strings.Split(line, ": ")
			if len(parts) >= 2 {
				filePath := parts[0]
				detection := strings.TrimSuffix(parts[1], " FOUND")

				vulnID := fmt.Sprintf("MALWARE-%s", strings.ReplaceAll(detection, ".", "_"))

				vulnerability := Vulnerability{
					ID:          vulnID,
					Severity:    "CRITICAL", // All malware detections are critical
					Title:       fmt.Sprintf("Malware detected: %s", detection),
					Description: fmt.Sprintf("ClamAV detected potential malware '%s' in file '%s'. This could indicate a compromised or malicious artifact.", detection, filePath),
					Package:     filePath,
					Version:     "detected",
					CVSS:        10.0, // Maximum CVSS score for malware
					References:  []string{"https://www.clamav.net/"},
				}

				vulnerabilities = append(vulnerabilities, vulnerability)
			}
		}
	}

	return vulnerabilities, nil
}
