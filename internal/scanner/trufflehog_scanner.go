package scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

// TruffleHogScanner implements secret detection using TruffleHog
type TruffleHogScanner struct {
	trufflehogPath string
}

func NewTruffleHogScanner() *TruffleHogScanner {
	return &TruffleHogScanner{
		trufflehogPath: "trufflehog",
	}
}

func (s *TruffleHogScanner) Name() string {
	return "TruffleHog Secret Scanner"
}

func (s *TruffleHogScanner) SupportedTypes() []string {
	return []string{"generic", "docker", "git", "archive"}
}

func (s *TruffleHogScanner) IsAvailable() bool {
	_, err := exec.LookPath(s.trufflehogPath)
	return err == nil
}

func (s *TruffleHogScanner) Supports(artifactType string) bool {
	supportedTypes := s.SupportedTypes()
	for _, t := range supportedTypes {
		if t == artifactType {
			return true
		}
	}
	return false
}

func (s *TruffleHogScanner) Scan(ctx context.Context, artifactPath string, artifactType string) (*ScanResult, error) {
	startTime := time.Now()

	// TruffleHog filesystem scan with JSON output
	args := []string{
		"filesystem",
		"--json",
		"--no-update",
		artifactPath,
	}

	cmd := exec.CommandContext(ctx, s.trufflehogPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("trufflehog command failed: %w, output: %s", err, string(output))
	}

	vulnerabilities, err := s.parseTruffleHogOutput(output)
	if err != nil {
		return nil, fmt.Errorf("failed to parse trufflehog output: %w", err)
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
			"scan_type": "secret_detection",
			"tool":      "trufflehog",
		},
	}, nil
}

func (s *TruffleHogScanner) getVersion() string {
	cmd := exec.Command(s.trufflehogPath, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "unknown"
	}
	return string(output)
}

// extractComponentName extracts a meaningful component name from file path
func (s *TruffleHogScanner) extractComponentName(filePath string) string {
	// Remove temporary paths and extract meaningful component name
	if filePath == "" {
		return "Unknown File"
	}

	// If it's a temp path like /app/storage/temp/artifact-37-40, return generic name
	if len(filePath) > 20 && (filePath[:20] == "/app/storage/temp/ar" ||
		(len(filePath) > 15 && filePath[:15] == "/app/storage/te") ||
		(len(filePath) > 5 && filePath[:5] == "/tmp/")) {
		return "Container Image"
	}

	// Extract filename from path for more specific identification
	lastSlash := -1
	for i := len(filePath) - 1; i >= 0; i-- {
		if filePath[i] == '/' {
			lastSlash = i
			break
		}
	}

	if lastSlash >= 0 && lastSlash < len(filePath)-1 {
		filename := filePath[lastSlash+1:]
		if filename != "" {
			// Common file types that might contain secrets
			if len(filename) > 4 {
				ext := filename[len(filename)-4:]
				switch ext {
				case ".pem", ".key", ".crt":
					return "Certificate/Key File"
				case ".env", ".cfg", ".ini":
					return "Configuration File"
				case ".yml", "yaml":
					return "YAML Config"
				case ".xml", ".json":
					return "Structured Data File"
				}
			}

			// Check for common secret-containing filenames
			if filename == "id_rsa" || filename == "id_dsa" || filename == "id_ed25519" {
				return "SSH Private Key"
			}
			if filename == ".env" || filename == "environment" {
				return "Environment File"
			}
			if filename == "docker-compose.yml" || filename == "dockerfile" {
				return "Docker Configuration"
			}

			return filename
		}
	}

	return "File System"
}

// TruffleHogResult represents TruffleHog JSON output
type TruffleHogResult struct {
	SourceMetadata TruffleHogSource       `json:"SourceMetadata"`
	SourceID       int64                  `json:"SourceID"`
	SourceType     int                    `json:"SourceType"`
	SourceName     string                 `json:"SourceName"`
	DetectorType   int                    `json:"DetectorType"`
	DetectorName   string                 `json:"DetectorName"`
	Verified       bool                   `json:"Verified"`
	Raw            string                 `json:"Raw"`
	RawV2          string                 `json:"RawV2"`
	Redacted       string                 `json:"Redacted"`
	ExtraData      map[string]interface{} `json:"ExtraData"`
}

type TruffleHogSource struct {
	Data TruffleHogData `json:"Data"`
}

type TruffleHogData struct {
	Filesystem TruffleHogFilesystem `json:"Filesystem"`
}

type TruffleHogFilesystem struct {
	File string `json:"file"`
	Line int64  `json:"line"`
}

func (s *TruffleHogScanner) parseTruffleHogOutput(output []byte) ([]Vulnerability, error) {
	// TruffleHog outputs one JSON object per line
	lines := make([]string, 0)
	currentLine := ""

	for _, b := range output {
		if b == '\n' {
			if currentLine != "" {
				lines = append(lines, currentLine)
				currentLine = ""
			}
		} else {
			currentLine += string(b)
		}
	}
	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	var vulnerabilities []Vulnerability

	for _, line := range lines {
		if line == "" {
			continue
		}

		var result TruffleHogResult
		if err := json.Unmarshal([]byte(line), &result); err != nil {
			continue // Skip malformed lines
		}

		// Determine severity based on verification and detector type
		severity := "MEDIUM"
		if result.Verified {
			severity = "HIGH"
		}

		// Create vulnerability ID for secrets
		vulnID := fmt.Sprintf("SECRET-%s-%d", result.DetectorName, result.SourceID)

		// Extract meaningful component name from file path
		componentName := s.extractComponentName(result.SourceMetadata.Data.Filesystem.File)

		vulnerability := Vulnerability{
			ID:          vulnID,
			Severity:    severity,
			Title:       fmt.Sprintf("SECRET-%s-%d", result.DetectorName, result.SourceID%1000),
			Description: fmt.Sprintf("Potential secret of type '%s' found in file. Verified: %v", result.DetectorName, result.Verified),
			Package:     componentName,
			Version:     fmt.Sprintf("Line %d", result.SourceMetadata.Data.Filesystem.Line),
			CVSS:        0, // Secrets don't have CVSS scores
			References:  []string{},
		}

		vulnerabilities = append(vulnerabilities, vulnerability)
	}

	return vulnerabilities, nil
}
