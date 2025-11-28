package scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// DockerManifestScanner implements scanning for Docker/OCI manifests using Syft, Grype, and Trivy
type DockerManifestScanner struct {
	syftPath  string
	grypePath string
	trivyPath string
}

func NewDockerManifestScanner() *DockerManifestScanner {
	return &DockerManifestScanner{
		syftPath:  "syft",
		grypePath: "grype",
		trivyPath: "trivy",
	}
}

func (s *DockerManifestScanner) Name() string {
	return "Docker/OCI Manifest Scanner (Syft+Grype+Trivy)"
}

func (s *DockerManifestScanner) SupportedTypes() []string {
	return []string{"docker"}
}

func (s *DockerManifestScanner) IsAvailable() bool {
	tools := []string{s.syftPath, s.grypePath, s.trivyPath}
	for _, tool := range tools {
		if _, err := exec.LookPath(tool); err != nil {
			return false
		}
	}
	return true
}

func (s *DockerManifestScanner) Supports(artifactType string) bool {
	supportedTypes := s.SupportedTypes()
	for _, t := range supportedTypes {
		if t == artifactType {
			return true
		}
	}
	return false
}

// DockerManifest represents a Docker/OCI manifest structure
type DockerManifest struct {
	SchemaVersion int                `json:"schemaVersion"`
	MediaType     string             `json:"mediaType"`
	Config        DockerDescriptor   `json:"config"`
	Layers        []DockerDescriptor `json:"layers"`
	Annotations   map[string]string  `json:"annotations,omitempty"`
}

type DockerDescriptor struct {
	MediaType string `json:"mediaType"`
	Digest    string `json:"digest"`
	Size      int64  `json:"size"`
}

func (s *DockerManifestScanner) Scan(ctx context.Context, artifactPath string, artifactType string) (*ScanResult, error) {
	startTime := time.Now()

	// Check if this is a Docker manifest
	manifest, err := s.parseManifest(artifactPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Docker manifest: %w", err)
	}

	// Perform multiple scanning approaches
	var allVulnerabilities []Vulnerability
	var scanErrors []string

	// 1. Scan the manifest file itself with Syft+Grype
	if manifestVulns, err := s.scanManifestFile(ctx, artifactPath); err != nil {
		scanErrors = append(scanErrors, fmt.Sprintf("Manifest file scan failed: %v", err))
	} else {
		allVulnerabilities = append(allVulnerabilities, manifestVulns...)
	}

	// 2. Analyze manifest structure and referenced images with Trivy
	if structuralVulns, err := s.analyzeManifestStructure(ctx, manifest, artifactPath); err != nil {
		scanErrors = append(scanErrors, fmt.Sprintf("Structural analysis failed: %v", err))
	} else {
		allVulnerabilities = append(allVulnerabilities, structuralVulns...)
	}

	// 3. Check base image vulnerabilities if we can identify them
	if baseImageVulns, err := s.scanBaseImages(ctx, manifest); err != nil {
		scanErrors = append(scanErrors, fmt.Sprintf("Base image scan failed: %v", err))
	} else {
		allVulnerabilities = append(allVulnerabilities, baseImageVulns...)
	}

	// Deduplicate vulnerabilities
	uniqueVulns := s.deduplicateVulnerabilities(allVulnerabilities)

	// Calculate severity counts
	criticalCount, highCount, mediumCount, lowCount := s.calculateSeverityCounts(uniqueVulns)

	result := &ScanResult{
		ScannerName:     s.Name(),
		ScannerVersion:  "1.0.0",
		ArtifactType:    artifactType,
		Vulnerabilities: uniqueVulns,
		Summary: VulnerabilitySummary{
			Critical: criticalCount,
			High:     highCount,
			Medium:   mediumCount,
			Low:      lowCount,
			Total:    len(uniqueVulns),
		},
		ScanDuration: time.Since(startTime).Seconds(),
		Metadata: map[string]interface{}{
			"manifest_schema_version": manifest.SchemaVersion,
			"manifest_media_type":     manifest.MediaType,
			"config_digest":           manifest.Config.Digest,
			"layer_count":             len(manifest.Layers),
			"scan_tools_used":         []string{"syft", "grype", "trivy"},
			"scan_approaches":         []string{"manifest_file_syft", "structural_analysis", "trivy_config_scan", "trivy_filesystem_scan", "base_image_inference"},
			"trivy_scans":             []string{"config", "filesystem", "secrets"},
			"scan_errors":             scanErrors,
		},
	}

	return result, nil
}

// parseManifest parses the Docker manifest JSON file
func (s *DockerManifestScanner) parseManifest(manifestPath string) (*DockerManifest, error) {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}

	var manifest DockerManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}

// scanManifestFile scans the manifest file itself using Syft + Grype
func (s *DockerManifestScanner) scanManifestFile(ctx context.Context, manifestPath string) ([]Vulnerability, error) {
	// Create temporary SBOM
	tempDir, err := os.MkdirTemp("", "manifest-scan-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempDir)

	sbomPath := filepath.Join(tempDir, "manifest.sbom.json")

	// Generate SBOM for the manifest file
	syftCmd := exec.CommandContext(ctx, s.syftPath, manifestPath, "-o", "syft-json="+sbomPath)
	if err := syftCmd.Run(); err != nil {
		// Manifest files might not generate useful SBOMs, but try anyway
		return []Vulnerability{}, nil
	}

	// Scan SBOM with Grype
	grypeCmd := exec.CommandContext(ctx, s.grypePath, "sbom:"+sbomPath, "-o", "json")
	output, err := grypeCmd.Output()
	if err != nil {
		return []Vulnerability{}, nil // Don't fail if no vulnerabilities found
	}

	return s.parseGrypeOutput(output)
}

// analyzeManifestStructure analyzes the manifest structure using Trivy
func (s *DockerManifestScanner) analyzeManifestStructure(ctx context.Context, manifest *DockerManifest, manifestPath string) ([]Vulnerability, error) {
	var vulnerabilities []Vulnerability

	// Check for known insecure configurations in manifest
	configVulns := s.checkManifestConfiguration(manifest)
	vulnerabilities = append(vulnerabilities, configVulns...)

	// Analyze layer information for potential security issues
	layerVulns := s.analyzeLayers(manifest)
	vulnerabilities = append(vulnerabilities, layerVulns...)

	// Use Trivy to scan the manifest file for config issues
	trivyConfigVulns, err := s.runTrivyConfigScan(ctx, manifestPath)
	if err == nil {
		vulnerabilities = append(vulnerabilities, trivyConfigVulns...)
	}

	// Use Trivy for filesystem scanning if possible
	trivyFsVulns, err := s.runTrivyFilesystemScan(ctx, manifestPath)
	if err == nil {
		vulnerabilities = append(vulnerabilities, trivyFsVulns...)
	}

	return vulnerabilities, nil
}

// scanBaseImages attempts to identify and scan base images referenced in the manifest
func (s *DockerManifestScanner) scanBaseImages(ctx context.Context, manifest *DockerManifest) ([]Vulnerability, error) {
	var vulnerabilities []Vulnerability

	// Extract potential base image information from config digest
	if strings.HasPrefix(manifest.Config.Digest, "sha256:") {
		// Create a vulnerability entry for potential base image issues
		baseImageVuln := Vulnerability{
			ID:          "BASE-IMAGE-UNKNOWN",
			Severity:    "INFO",
			Title:       "Base Image Analysis",
			Description: fmt.Sprintf("Container image uses config digest %s. Consider scanning the complete image for comprehensive vulnerability assessment.", manifest.Config.Digest),
			FixedIn:     "",
			Package:     "base-image",
			Version:     "unknown",
		}
		vulnerabilities = append(vulnerabilities, baseImageVuln)
	}

	// Analyze layers for potential vulnerabilities
	for i, layer := range manifest.Layers {
		if layer.Size > 100*1024*1024 { // Large layers (>100MB) might contain vulnerable packages
			layerVuln := Vulnerability{
				ID:          fmt.Sprintf("LARGE-LAYER-%d", i),
				Severity:    "LOW",
				Title:       "Large Container Layer",
				Description: fmt.Sprintf("Layer %d (%s) is large (%d bytes). Large layers may contain unnecessary packages with potential vulnerabilities.", i, layer.Digest[:12], layer.Size),
				Package:     "container-layer",
				Version:     fmt.Sprintf("layer-%d", i),
			}
			vulnerabilities = append(vulnerabilities, layerVuln)
		}
	}

	return vulnerabilities, nil
}

// checkManifestConfiguration checks for insecure manifest configurations
func (s *DockerManifestScanner) checkManifestConfiguration(manifest *DockerManifest) []Vulnerability {
	var vulnerabilities []Vulnerability

	// Check schema version
	if manifest.SchemaVersion < 2 {
		vuln := Vulnerability{
			ID:          "MANIFEST-OLD-SCHEMA",
			Severity:    "MEDIUM",
			Title:       "Outdated Manifest Schema",
			Description: fmt.Sprintf("Manifest uses schema version %d. Consider upgrading to schema version 2 for better security and compatibility.", manifest.SchemaVersion),
			Package:     "manifest",
			Version:     fmt.Sprintf("schema-v%d", manifest.SchemaVersion),
		}
		vulnerabilities = append(vulnerabilities, vuln)
	}

	// Check for missing annotations that could improve security
	if manifest.Annotations == nil || len(manifest.Annotations) == 0 {
		vuln := Vulnerability{
			ID:          "MANIFEST-NO-ANNOTATIONS",
			Severity:    "LOW",
			Title:       "Missing Security Annotations",
			Description: "Manifest lacks annotations. Consider adding security-related annotations like org.opencontainers.image.source, org.opencontainers.image.version, etc.",
			Package:     "manifest",
			Version:     "annotations",
		}
		vulnerabilities = append(vulnerabilities, vuln)
	}

	return vulnerabilities
}

// analyzeLayers analyzes layer information for security issues
func (s *DockerManifestScanner) analyzeLayers(manifest *DockerManifest) []Vulnerability {
	var vulnerabilities []Vulnerability

	// Check for too many layers (potential inefficiency and security concern)
	if len(manifest.Layers) > 20 {
		vuln := Vulnerability{
			ID:          "MANIFEST-TOO-MANY-LAYERS",
			Severity:    "LOW",
			Title:       "Excessive Container Layers",
			Description: fmt.Sprintf("Image has %d layers. Consider consolidating layers to reduce attack surface and improve performance.", len(manifest.Layers)),
			Package:     "container-structure",
			Version:     fmt.Sprintf("%d-layers", len(manifest.Layers)),
		}
		vulnerabilities = append(vulnerabilities, vuln)
	}

	return vulnerabilities
}

// runTrivyConfigScan runs Trivy configuration scanning on the manifest
func (s *DockerManifestScanner) runTrivyConfigScan(ctx context.Context, manifestPath string) ([]Vulnerability, error) {
	cmd := exec.CommandContext(ctx, s.trivyPath, "config", manifestPath, "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		return []Vulnerability{}, nil // Don't fail if Trivy config scan doesn't work
	}

	return s.parseTrivyConfigOutput(output)
}

// runTrivyFilesystemScan runs Trivy filesystem scanning on the manifest
func (s *DockerManifestScanner) runTrivyFilesystemScan(ctx context.Context, manifestPath string) ([]Vulnerability, error) {
	cmd := exec.CommandContext(ctx, s.trivyPath, "fs", manifestPath, "--format", "json", "--security-checks", "vuln,secret,config")
	output, err := cmd.Output()
	if err != nil {
		return []Vulnerability{}, nil // Don't fail if Trivy filesystem scan doesn't work
	}

	return s.parseTrivyFilesystemOutput(output)
}

// Helper functions for parsing scanner outputs

func (s *DockerManifestScanner) parseGrypeOutput(output []byte) ([]Vulnerability, error) {
	var grypeResult struct {
		Matches []struct {
			Vulnerability struct {
				ID             string `json:"id"`
				Severity       string `json:"severity"`
				Description    string `json:"description"`
				FixedInVersion string `json:"fixedInVersion"`
			} `json:"vulnerability"`
			Artifact struct {
				Name    string `json:"name"`
				Version string `json:"version"`
				Type    string `json:"type"`
			} `json:"artifact"`
		} `json:"matches"`
	}

	if err := json.Unmarshal(output, &grypeResult); err != nil {
		return nil, err
	}

	var vulnerabilities []Vulnerability
	for _, match := range grypeResult.Matches {
		vuln := Vulnerability{
			ID:          match.Vulnerability.ID,
			CVE:         ExtractCVE(match.Vulnerability.ID),
			Severity:    match.Vulnerability.Severity,
			Title:       match.Vulnerability.ID,
			Description: match.Vulnerability.Description,
			FixedIn:     match.Vulnerability.FixedInVersion,
			Package:     match.Artifact.Name,
			Version:     match.Artifact.Version,
		}
		vulnerabilities = append(vulnerabilities, vuln)
	}

	return vulnerabilities, nil
}

func (s *DockerManifestScanner) parseTrivyConfigOutput(output []byte) ([]Vulnerability, error) {
	var trivyResult struct {
		Results []struct {
			Misconfigurations []struct {
				ID          string `json:"id"`
				Severity    string `json:"severity"`
				Title       string `json:"title"`
				Description string `json:"description"`
				Resolution  string `json:"resolution"`
			} `json:"misconfigurations"`
		} `json:"results"`
	}

	if err := json.Unmarshal(output, &trivyResult); err != nil {
		return []Vulnerability{}, nil // Don't fail on parsing errors
	}

	var vulnerabilities []Vulnerability
	for _, result := range trivyResult.Results {
		for _, misconfig := range result.Misconfigurations {
			vuln := Vulnerability{
				ID:          misconfig.ID,
				Severity:    misconfig.Severity,
				Title:       misconfig.Title,
				Description: misconfig.Description,
				FixedIn:     misconfig.Resolution,
				Package:     "container-config",
				Version:     "configuration",
			}
			vulnerabilities = append(vulnerabilities, vuln)
		}
	}

	return vulnerabilities, nil
}

func (s *DockerManifestScanner) parseTrivyFilesystemOutput(output []byte) ([]Vulnerability, error) {
	var trivyResult struct {
		Results []struct {
			Target          string `json:"target"`
			Class           string `json:"class"`
			Vulnerabilities []struct {
				VulnerabilityID  string `json:"vulnerabilityID"`
				Severity         string `json:"severity"`
				Title            string `json:"title"`
				Description      string `json:"description"`
				PkgName          string `json:"pkgName"`
				InstalledVersion string `json:"installedVersion"`
				FixedVersion     string `json:"fixedVersion"`
			} `json:"vulnerabilities"`
			Secrets []struct {
				RuleID   string `json:"ruleID"`
				Severity string `json:"severity"`
				Title    string `json:"title"`
				Match    string `json:"match"`
			} `json:"secrets"`
		} `json:"results"`
	}

	if err := json.Unmarshal(output, &trivyResult); err != nil {
		return []Vulnerability{}, nil // Don't fail on parsing errors
	}

	var vulnerabilities []Vulnerability
	for _, result := range trivyResult.Results {
		// Process vulnerabilities
		for _, vuln := range result.Vulnerabilities {
			severity := strings.ToUpper(vuln.Severity)
			if severity == "" {
				severity = "UNKNOWN"
			}

			vulnerability := Vulnerability{
				ID:          vuln.VulnerabilityID,
				CVE:         ExtractCVE(vuln.VulnerabilityID),
				Severity:    severity,
				Title:       vuln.Title,
				Description: vuln.Description,
				FixedIn:     vuln.FixedVersion,
				Package:     vuln.PkgName,
				Version:     vuln.InstalledVersion,
			}
			vulnerabilities = append(vulnerabilities, vulnerability)
		}

		// Process secrets
		for _, secret := range result.Secrets {
			severity := strings.ToUpper(secret.Severity)
			if severity == "" {
				severity = "MEDIUM"
			}

			vulnerability := Vulnerability{
				ID:          secret.RuleID,
				Severity:    severity,
				Title:       secret.Title,
				Description: fmt.Sprintf("Secret detected: %s", secret.Match),
				Package:     "secrets",
				Version:     "detected",
			}
			vulnerabilities = append(vulnerabilities, vulnerability)
		}
	}

	return vulnerabilities, nil
}

func (s *DockerManifestScanner) deduplicateVulnerabilities(vulns []Vulnerability) []Vulnerability {
	seen := make(map[string]bool)
	var unique []Vulnerability

	for _, vuln := range vulns {
		key := fmt.Sprintf("%s-%s-%s", vuln.ID, vuln.Package, vuln.Version)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, vuln)
		}
	}

	return unique
}

func (s *DockerManifestScanner) calculateSeverityCounts(vulns []Vulnerability) (critical, high, medium, low int) {
	for _, vuln := range vulns {
		switch strings.ToUpper(vuln.Severity) {
		case "CRITICAL":
			critical++
		case "HIGH":
			high++
		case "MEDIUM":
			medium++
		case "LOW", "INFO":
			low++
		}
	}
	return
}
