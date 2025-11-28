package scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// SyftScanner implements vulnerability scanning using Syft + Grype
type SyftScanner struct {
	syftPath  string
	grypePath string
}

func NewSyftScanner() *SyftScanner {
	return &SyftScanner{
		syftPath:  "syft",  // assumes syft is in PATH
		grypePath: "grype", // assumes grype is in PATH
	}
}

func (s *SyftScanner) Name() string {
	return "Syft/Grype"
}

func (s *SyftScanner) SupportedTypes() []string {
	return []string{"docker", "npm", "maven", "pypi", "generic"}
}

func (s *SyftScanner) IsAvailable() bool {
	_, err := exec.LookPath(s.syftPath)
	if err != nil {
		return false
	}
	_, err = exec.LookPath(s.grypePath)
	return err == nil
}

func (s *SyftScanner) Supports(artifactType string) bool {
	supportedTypes := s.SupportedTypes()
	for _, t := range supportedTypes {
		if t == artifactType {
			return true
		}
	}
	return false
}

func (s *SyftScanner) Scan(ctx context.Context, artifactPath string, artifactType string) (*ScanResult, error) {
	startTime := time.Now()

	// Generate SBOM using Syft
	sbomData, err := s.generateSBOM(ctx, artifactPath, artifactType)
	if err != nil {
		return nil, fmt.Errorf("failed to generate SBOM: %w", err)
	}

	// Scan SBOM using Grype
	vulnerabilities, err := s.scanWithGrype(ctx, sbomData)
	if err != nil {
		return nil, fmt.Errorf("failed to scan with Grype: %w", err)
	}

	// Calculate summary
	summary := calculateSummary(vulnerabilities)

	duration := time.Since(startTime).Seconds()

	return &ScanResult{
		ScannerName:     s.Name(),
		ScannerVersion:  s.getVersion(),
		ArtifactType:    artifactType,
		Vulnerabilities: vulnerabilities,
		Summary:         summary,
		ScanDuration:    duration,
	}, nil
}

func (s *SyftScanner) generateSBOM(ctx context.Context, artifactPath string, artifactType string) ([]byte, error) {
	args := []string{
		artifactPath,
		"-o", "syft-json",
	}

	cmd := exec.CommandContext(ctx, s.syftPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("syft command failed: %w, output: %s", err, string(output))
	}

	return output, nil
}

func (s *SyftScanner) scanWithGrype(ctx context.Context, sbomData []byte) ([]Vulnerability, error) {
	// Write SBOM to a temporary file
	tmpFile, err := os.CreateTemp("", "sbom-*.json")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp SBOM file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.Write(sbomData); err != nil {
		return nil, fmt.Errorf("failed to write SBOM data: %w", err)
	}
	tmpFile.Close()

	// Run Grype on the SBOM file
	cmd := exec.CommandContext(ctx, s.grypePath, tmpFile.Name(), "-o", "json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("grype command failed: %w, output: %s", err, string(output))
	}

	// Parse Grype output
	var grypeOutput struct {
		Matches []struct {
			Vulnerability struct {
				ID          string `json:"id"`
				Severity    string `json:"severity"`
				Description string `json:"description"`
			} `json:"vulnerability"`
			Artifact struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"artifact"`
			RelatedVulnerabilities []struct {
				ID         string `json:"id"`
				DataSource string `json:"dataSource"`
			} `json:"relatedVulnerabilities"`
		} `json:"matches"`
	}

	if err := json.Unmarshal(output, &grypeOutput); err != nil {
		return nil, fmt.Errorf("failed to parse Grype output: %w", err)
	}

	// Convert to our format
	var vulnerabilities []Vulnerability
	for _, match := range grypeOutput.Matches {
		vuln := Vulnerability{
			ID:          match.Vulnerability.ID,
			CVE:         ExtractCVE(match.Vulnerability.ID),
			Severity:    normalizeSeverity(match.Vulnerability.Severity),
			Description: match.Vulnerability.Description,
			Package:     match.Artifact.Name,
			Version:     match.Artifact.Version,
		}

		// Add references
		for _, related := range match.RelatedVulnerabilities {
			vuln.References = append(vuln.References, related.DataSource)
		}

		vulnerabilities = append(vulnerabilities, vuln)
	}

	return vulnerabilities, nil
}

func (s *SyftScanner) getVersion() string {
	cmd := exec.Command(s.grypePath, "version")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return string(output)
}
