package scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

// BlintScanner implements scanning using OWASP BlInt
type BlintScanner struct {
	blintPath string
}

func NewBlintScanner() *BlintScanner {
	return &BlintScanner{
		blintPath: "blint", // assumes blint is in PATH
	}
}

func (s *BlintScanner) Name() string {
	return "OWASP BlInt"
}

func (s *BlintScanner) SupportedTypes() []string {
	return []string{"generic", "docker"} // Binary analysis
}

func (s *BlintScanner) IsAvailable() bool {
	_, err := exec.LookPath(s.blintPath)
	return err == nil
}

func (s *BlintScanner) Supports(artifactType string) bool {
	supportedTypes := s.SupportedTypes()
	for _, t := range supportedTypes {
		if t == artifactType {
			return true
		}
	}
	return false
}

func (s *BlintScanner) Scan(ctx context.Context, artifactPath string, artifactType string) (*ScanResult, error) {
	startTime := time.Now()

	args := []string{
		"sbom",
		"--json",
		artifactPath,
	}

	cmd := exec.CommandContext(ctx, s.blintPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("blint command failed: %w, output: %s", err, string(output))
	}

	vulnerabilities, err := s.parseBlintOutput(output)
	if err != nil {
		return nil, fmt.Errorf("failed to parse blint output: %w", err)
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

func (s *BlintScanner) parseBlintOutput(output []byte) ([]Vulnerability, error) {
	var blintOutput struct {
		Vulnerabilities []struct {
			ID          string `json:"id"`
			Severity    string `json:"severity"`
			Title       string `json:"title"`
			Description string `json:"description"`
			Component   string `json:"component"`
		} `json:"vulnerabilities"`
	}

	if err := json.Unmarshal(output, &blintOutput); err != nil {
		return nil, err
	}

	var vulnerabilities []Vulnerability
	for _, v := range blintOutput.Vulnerabilities {
		vulnerabilities = append(vulnerabilities, Vulnerability{
			ID:          v.ID,
			Severity:    normalizeSeverity(v.Severity),
			Title:       v.Title,
			Description: v.Description,
			Package:     v.Component,
		})
	}

	return vulnerabilities, nil
}
