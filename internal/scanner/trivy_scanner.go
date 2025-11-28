package scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// TrivyScanner implements OS and container scanning using Trivy
type TrivyScanner struct {
	trivyPath string
}

func NewTrivyScanner() *TrivyScanner {
	return &TrivyScanner{
		trivyPath: "trivy",
	}
}

func (s *TrivyScanner) Name() string {
	return "Trivy OS/Container Scanner"
}

func (s *TrivyScanner) SupportedTypes() []string {
	return []string{"docker", "generic", "maven", "npm", "pypi"}
}

func (s *TrivyScanner) IsAvailable() bool {
	_, err := exec.LookPath(s.trivyPath)
	return err == nil
}

func (s *TrivyScanner) Supports(artifactType string) bool {
	supportedTypes := s.SupportedTypes()
	for _, t := range supportedTypes {
		if t == artifactType {
			return true
		}
	}
	return false
}

func (s *TrivyScanner) Scan(ctx context.Context, artifactPath string, artifactType string) (*ScanResult, error) {
	startTime := time.Now()

	// Trivy scan with JSON output
	args := []string{
		"fs", // Filesystem scan
		"--format", "json",
		"--severity", "UNKNOWN,LOW,MEDIUM,HIGH,CRITICAL",
		"--quiet", // Suppress progress messages and logs
		artifactPath,
	}

	cmd := exec.CommandContext(ctx, s.trivyPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("trivy command failed: %w, output: %s", err, string(output))
	}

	// Debug logging to see the actual output
	fmt.Printf("[TRIVY_DEBUG] Raw output length: %d bytes\n", len(output))
	previewLen := 500
	if len(output) < previewLen {
		previewLen = len(output)
	}
	fmt.Printf("[TRIVY_DEBUG] First 500 chars: %q\n", string(output[:previewLen]))
	if len(output) > 500 {
		fmt.Printf("[TRIVY_DEBUG] Last 200 chars: %q\n", string(output[len(output)-200:]))
	}

	vulnerabilities, err := s.parseTrivyOutput(output)
	if err != nil {
		return nil, fmt.Errorf("failed to parse trivy output: %w", err)
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
			"scan_type": "filesystem",
			"tool":      "trivy",
		},
	}, nil
}

func (s *TrivyScanner) getVersion() string {
	cmd := exec.Command(s.trivyPath, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "unknown"
	}
	return string(output)
}

// TrivyResult represents Trivy JSON output structure
type TrivyResult struct {
	SchemaVersion int           `json:"SchemaVersion"`
	ArtifactName  string        `json:"ArtifactName"`
	ArtifactType  string        `json:"ArtifactType"`
	Results       []TrivyTarget `json:"Results"`
}

type TrivyTarget struct {
	Target          string               `json:"Target"`
	Class           string               `json:"Class"`
	Type            string               `json:"Type"`
	Vulnerabilities []TrivyVulnerability `json:"Vulnerabilities"`
}

type TrivyVulnerability struct {
	VulnerabilityID  string                 `json:"VulnerabilityID"`
	PkgName          string                 `json:"PkgName"`
	PkgPath          string                 `json:"PkgPath"`
	InstalledVersion string                 `json:"InstalledVersion"`
	FixedVersion     string                 `json:"FixedVersion"`
	Severity         string                 `json:"Severity"`
	Title            string                 `json:"Title"`
	Description      string                 `json:"Description"`
	References       []string               `json:"References"`
	CVSS             map[string]interface{} `json:"CVSS"`
}

func (s *TrivyScanner) parseTrivyOutput(output []byte) ([]Vulnerability, error) {
	// Try to clean the output first - Trivy sometimes adds extra content
	cleanedOutput := s.cleanTrivyOutput(string(output))

	fmt.Printf("[TRIVY_DEBUG] Cleaned output length: %d bytes\n", len(cleanedOutput))
	if len(cleanedOutput) > 100 {
		fmt.Printf("[TRIVY_DEBUG] Cleaned output (first 100 chars): %q\n", cleanedOutput[:100])
		fmt.Printf("[TRIVY_DEBUG] Cleaned output (last 100 chars): %q\n", cleanedOutput[len(cleanedOutput)-100:])
	} else {
		fmt.Printf("[TRIVY_DEBUG] Cleaned output (full): %q\n", cleanedOutput)
	}

	var result TrivyResult
	if err := json.Unmarshal([]byte(cleanedOutput), &result); err != nil {
		fmt.Printf("[TRIVY_DEBUG] JSON unmarshal failed: %v\n", err)
		return nil, fmt.Errorf("failed to unmarshal trivy JSON: %w", err)
	}

	var vulnerabilities []Vulnerability

	for _, target := range result.Results {
		for _, vuln := range target.Vulnerabilities {
			// Extract CVSS score
			var cvssScore float64
			if vuln.CVSS != nil {
				if nvd, ok := vuln.CVSS["nvd"].(map[string]interface{}); ok {
					if v3, ok := nvd["V3Score"].(float64); ok {
						cvssScore = v3
					} else if v2, ok := nvd["V2Score"].(float64); ok {
						cvssScore = v2
					}
				}
			}

			vulnerability := Vulnerability{
				ID:          vuln.VulnerabilityID,
				CVE:         ExtractCVE(vuln.VulnerabilityID),
				Severity:    normalizeSeverity(vuln.Severity),
				Title:       vuln.Title,
				Description: vuln.Description,
				Package:     vuln.PkgName,
				Version:     vuln.InstalledVersion,
				FixedIn:     vuln.FixedVersion,
				CVSS:        cvssScore,
				References:  vuln.References,
			}

			vulnerabilities = append(vulnerabilities, vulnerability)
		}
	}

	return vulnerabilities, nil
}

// cleanTrivyOutput extracts valid JSON from Trivy's raw output using robust line-by-line parsing
// Trivy outputs log messages with timestamps that contain problematic characters for JSON parsing
// This function filters out log lines and extracts only valid JSON content
func (s *TrivyScanner) cleanTrivyOutput(rawOutput string) string {
	lines := strings.Split(rawOutput, "\n")
	var jsonLines []string
	var potentialJson strings.Builder
	inJsonBlock := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines
		if line == "" {
			continue
		}

		// Skip obvious log lines (contain timestamps, info messages, etc.)
		if s.isLogLine(line) {
			continue
		}

		// Check if this line starts a JSON object
		if strings.HasPrefix(line, "{") {
			inJsonBlock = true
			potentialJson.Reset()
			potentialJson.WriteString(line)

			// If it's a complete JSON object on one line, validate and add it
			if strings.HasSuffix(line, "}") && s.isValidSingleLineJSON(line) {
				jsonLines = append(jsonLines, line)
				inJsonBlock = false
			}
			continue
		}

		// If we're in a JSON block, accumulate lines
		if inJsonBlock {
			potentialJson.WriteString("\n")
			potentialJson.WriteString(line)

			// Check if this completes the JSON object
			if strings.HasSuffix(line, "}") {
				jsonContent := potentialJson.String()
				if s.isValidJSON(jsonContent) {
					jsonLines = append(jsonLines, jsonContent)
				}
				inJsonBlock = false
			}
		}
	}

	// If no valid JSON found through line parsing, try fallback extraction
	if len(jsonLines) == 0 {
		return s.fallbackJSONExtraction(rawOutput)
	}

	// Join all valid JSON lines
	result := strings.Join(jsonLines, "\n")

	// Final cleanup of any remaining ANSI codes
	return s.removeANSICodes(result)
}

// isLogLine detects log lines that should be filtered out
func (s *TrivyScanner) isLogLine(line string) bool {
	// Common log patterns in Trivy output
	logPatterns := []string{
		"INFO",
		"WARN",
		"ERROR",
		"DEBUG",
		"Downloading",
		"Loading",
		"Scanning",
		"Need to update",
		"DB Repository:",
	}

	// Check for timestamp patterns (ISO format, common log timestamps)
	if strings.Contains(line, "T") && (strings.Contains(line, "-05:00") || strings.Contains(line, "Z")) {
		return true
	}

	// Check for common log keywords
	for _, pattern := range logPatterns {
		if strings.Contains(line, pattern) {
			return true
		}
	}

	return false
}

// isValidSingleLineJSON validates a single line as complete JSON
func (s *TrivyScanner) isValidSingleLineJSON(line string) bool {
	var temp interface{}
	return json.Unmarshal([]byte(line), &temp) == nil
}

// isValidJSON validates multi-line JSON content
func (s *TrivyScanner) isValidJSON(content string) bool {
	var temp interface{}
	return json.Unmarshal([]byte(content), &temp) == nil
}

// fallbackJSONExtraction provides fallback JSON extraction using brace counting
func (s *TrivyScanner) fallbackJSONExtraction(rawOutput string) string {
	startIndex := strings.Index(rawOutput, "{")
	if startIndex == -1 {
		return rawOutput
	}

	braceCount := 0
	endIndex := -1

	for i := startIndex; i < len(rawOutput); i++ {
		char := rawOutput[i]
		if char == '{' {
			braceCount++
		} else if char == '}' {
			braceCount--
			if braceCount == 0 {
				endIndex = i
				break
			}
		}
	}

	if endIndex == -1 {
		lastBrace := strings.LastIndex(rawOutput, "}")
		if lastBrace > startIndex {
			endIndex = lastBrace
		} else {
			return rawOutput
		}
	}

	return s.removeANSICodes(rawOutput[startIndex : endIndex+1])
}

// removeANSICodes removes ANSI color codes from text
func (s *TrivyScanner) removeANSICodes(text string) string {
	cleanedOutput := text
	for {
		ansiStart := strings.Index(cleanedOutput, "\x1b[")
		if ansiStart == -1 {
			break
		}
		ansiEnd := strings.Index(cleanedOutput[ansiStart:], "m")
		if ansiEnd == -1 {
			break
		}
		ansiEnd += ansiStart + 1
		cleanedOutput = cleanedOutput[:ansiStart] + cleanedOutput[ansiEnd:]
	}
	return cleanedOutput
}
