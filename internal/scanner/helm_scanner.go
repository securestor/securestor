package scanner

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// HelmChartScanner implements scanning for Helm charts
type HelmChartScanner struct {
	trivyPath string
}

// NewHelmChartScanner creates a new Helm chart scanner
func NewHelmChartScanner() *HelmChartScanner {
	return &HelmChartScanner{
		trivyPath: "trivy",
	}
}

func (s *HelmChartScanner) Name() string {
	return "Helm Chart Scanner"
}

func (s *HelmChartScanner) SupportedTypes() []string {
	return []string{"helm"}
}

func (s *HelmChartScanner) IsAvailable() bool {
	// Check if Trivy is available for enhanced scanning
	if _, err := exec.LookPath(s.trivyPath); err == nil {
		return true
	}
	// Always available for basic static analysis
	return true
}

func (s *HelmChartScanner) Supports(artifactType string) bool {
	for _, t := range s.SupportedTypes() {
		if t == artifactType {
			return true
		}
	}
	return false
}

// HelmChart represents Chart.yaml metadata
type HelmChart struct {
	APIVersion  string   `yaml:"apiVersion"`
	Name        string   `yaml:"name"`
	Version     string   `yaml:"version"`
	KubeVersion string   `yaml:"kubeVersion,omitempty"`
	Description string   `yaml:"description,omitempty"`
	AppVersion  string   `yaml:"appVersion,omitempty"`
	Deprecated  bool     `yaml:"deprecated,omitempty"`
	Keywords    []string `yaml:"keywords,omitempty"`
}

// Scan performs security scanning on a Helm chart
func (s *HelmChartScanner) Scan(ctx context.Context, artifactPath string, artifactType string) (*ScanResult, error) {
	startTime := time.Now()

	result := &ScanResult{
		ScannerName:     s.Name(),
		ArtifactType:    artifactType,
		Vulnerabilities: make([]Vulnerability, 0),
		Metadata:        make(map[string]interface{}),
	}

	// Extract chart to temporary directory
	tempDir, err := os.MkdirTemp("", "helm-scan-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Extract and analyze chart
	chart, err := s.extractChart(artifactPath, tempDir)
	if err != nil {
		return nil, fmt.Errorf("failed to extract chart: %w", err)
	}

	result.Metadata["chart_name"] = chart.Name
	result.Metadata["chart_version"] = chart.Version
	result.Metadata["app_version"] = chart.AppVersion
	result.Metadata["deprecated"] = chart.Deprecated

	// Check for deprecated chart
	if chart.Deprecated {
		result.Vulnerabilities = append(result.Vulnerabilities, Vulnerability{
			ID:          "HELM-DEPRECATED-001",
			Severity:    "MEDIUM",
			Title:       "Chart is deprecated",
			Description: fmt.Sprintf("Helm chart '%s' version '%s' is marked as deprecated", chart.Name, chart.Version),
			Package:     chart.Name,
			Version:     chart.Version,
		})
	}

	// Scan for deprecated Kubernetes APIs
	s.scanDeprecatedAPIs(chart, tempDir, result)

	// Scan for security issues
	s.scanSecurityIssues(chart, tempDir, result)

	// Calculate summary
	result.Summary = s.calculateSummary(result.Vulnerabilities)
	result.ScanDuration = time.Since(startTime).Seconds()

	return result, nil
}

// extractChart extracts Helm chart and parses Chart.yaml
func (s *HelmChartScanner) extractChart(chartPath, destDir string) (*HelmChart, error) {
	file, err := os.Open(chartPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open chart: %w", err)
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	var chart HelmChart

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read tar: %w", err)
		}

		// Security: prevent path traversal
		if strings.Contains(header.Name, "..") {
			continue
		}

		target := filepath.Join(destDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(target, 0755)

		case tar.TypeReg:
			os.MkdirAll(filepath.Dir(target), 0755)

			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY, os.FileMode(header.Mode))
			if err != nil {
				continue
			}
			io.Copy(f, tr)
			f.Close()

			// Parse Chart.yaml
			if strings.HasSuffix(header.Name, "Chart.yaml") {
				data, _ := os.ReadFile(target)
				yaml.Unmarshal(data, &chart)
			}
		}
	}

	if chart.Name == "" {
		return nil, fmt.Errorf("Chart.yaml not found")
	}

	return &chart, nil
}

// scanDeprecatedAPIs checks for deprecated Kubernetes API versions
func (s *HelmChartScanner) scanDeprecatedAPIs(chart *HelmChart, chartDir string, result *ScanResult) {
	deprecatedAPIs := map[string]string{
		"extensions/v1beta1":           "Use apps/v1 (deprecated in Kubernetes 1.16)",
		"apps/v1beta1":                 "Use apps/v1 (deprecated in Kubernetes 1.16)",
		"apps/v1beta2":                 "Use apps/v1 (deprecated in Kubernetes 1.16)",
		"apiextensions.k8s.io/v1beta1": "Use apiextensions.k8s.io/v1 (deprecated in Kubernetes 1.22)",
		"batch/v1beta1":                "Use batch/v1 (deprecated in Kubernetes 1.21)",
	}

	templatesDir := filepath.Join(chartDir, chart.Name, "templates")
	entries, err := os.ReadDir(templatesDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(templatesDir, entry.Name()))
		if err != nil {
			continue
		}

		content := string(data)
		for api, message := range deprecatedAPIs {
			if strings.Contains(content, api) {
				result.Vulnerabilities = append(result.Vulnerabilities, Vulnerability{
					ID:          fmt.Sprintf("HELM-API-%s", strings.ReplaceAll(api, "/", "-")),
					Severity:    "HIGH",
					Title:       "Deprecated Kubernetes API",
					Description: fmt.Sprintf("Template '%s' uses deprecated API '%s'. %s", entry.Name(), api, message),
					Package:     chart.Name,
					Version:     chart.Version,
				})
			}
		}
	}
}

// scanSecurityIssues checks for common security misconfigurations
func (s *HelmChartScanner) scanSecurityIssues(chart *HelmChart, chartDir string, result *ScanResult) {
	templatesDir := filepath.Join(chartDir, chart.Name, "templates")
	entries, err := os.ReadDir(templatesDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(templatesDir, entry.Name()))
		if err != nil {
			continue
		}

		content := string(data)

		// Check for privileged containers
		if strings.Contains(content, "privileged: true") {
			result.Vulnerabilities = append(result.Vulnerabilities, Vulnerability{
				ID:          "HELM-SEC-001",
				Severity:    "HIGH",
				Title:       "Privileged container",
				Description: fmt.Sprintf("Template '%s' defines a privileged container", entry.Name()),
				Package:     chart.Name,
				Version:     chart.Version,
			})
		}

		// Check for hostPath volumes
		if strings.Contains(content, "hostPath:") {
			result.Vulnerabilities = append(result.Vulnerabilities, Vulnerability{
				ID:          "HELM-SEC-002",
				Severity:    "MEDIUM",
				Title:       "hostPath volume usage",
				Description: fmt.Sprintf("Template '%s' uses hostPath volumes", entry.Name()),
				Package:     chart.Name,
				Version:     chart.Version,
			})
		}

		// Check for missing resource limits
		if (strings.Contains(content, "kind: Deployment") || strings.Contains(content, "kind: StatefulSet")) &&
			(!strings.Contains(content, "limits:") || !strings.Contains(content, "requests:")) {
			result.Vulnerabilities = append(result.Vulnerabilities, Vulnerability{
				ID:          "HELM-SEC-003",
				Severity:    "LOW",
				Title:       "Missing resource limits",
				Description: fmt.Sprintf("Template '%s' should define resource limits and requests", entry.Name()),
				Package:     chart.Name,
				Version:     chart.Version,
			})
		}
	}
}

// calculateSummary counts vulnerabilities by severity
func (s *HelmChartScanner) calculateSummary(vulns []Vulnerability) VulnerabilitySummary {
	summary := VulnerabilitySummary{}

	for _, vuln := range vulns {
		switch strings.ToUpper(vuln.Severity) {
		case "CRITICAL":
			summary.Critical++
		case "HIGH":
			summary.High++
		case "MEDIUM":
			summary.Medium++
		case "LOW":
			summary.Low++
		}
		summary.Total++
	}

	return summary
}
