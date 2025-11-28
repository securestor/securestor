package storage

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// HelmOCIHandler handles Helm charts using OCI format
type HelmOCIHandler struct {
	basePath string
}

// HelmChart represents Chart.yaml structure
type HelmChart struct {
	APIVersion   string            `yaml:"apiVersion" json:"apiVersion"`
	Name         string            `yaml:"name" json:"name"`
	Version      string            `yaml:"version" json:"version"`
	KubeVersion  string            `yaml:"kubeVersion,omitempty" json:"kubeVersion,omitempty"`
	Description  string            `yaml:"description,omitempty" json:"description,omitempty"`
	Type         string            `yaml:"type,omitempty" json:"type,omitempty"`
	Keywords     []string          `yaml:"keywords,omitempty" json:"keywords,omitempty"`
	Home         string            `yaml:"home,omitempty" json:"home,omitempty"`
	Sources      []string          `yaml:"sources,omitempty" json:"sources,omitempty"`
	Dependencies []HelmDependency  `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`
	Maintainers  []HelmMaintainer  `yaml:"maintainers,omitempty" json:"maintainers,omitempty"`
	Icon         string            `yaml:"icon,omitempty" json:"icon,omitempty"`
	AppVersion   string            `yaml:"appVersion,omitempty" json:"appVersion,omitempty"`
	Deprecated   bool              `yaml:"deprecated,omitempty" json:"deprecated,omitempty"`
	Annotations  map[string]string `yaml:"annotations,omitempty" json:"annotations,omitempty"`
}

// HelmDependency represents chart dependency
type HelmDependency struct {
	Name         string   `yaml:"name" json:"name"`
	Version      string   `yaml:"version" json:"version"`
	Repository   string   `yaml:"repository" json:"repository"`
	Condition    string   `yaml:"condition,omitempty" json:"condition,omitempty"`
	Tags         []string `yaml:"tags,omitempty" json:"tags,omitempty"`
	Enabled      bool     `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	ImportValues []string `yaml:"import-values,omitempty" json:"import-values,omitempty"`
	Alias        string   `yaml:"alias,omitempty" json:"alias,omitempty"`
}

// HelmMaintainer represents chart maintainer
type HelmMaintainer struct {
	Name  string `yaml:"name" json:"name"`
	Email string `yaml:"email,omitempty" json:"email,omitempty"`
	URL   string `yaml:"url,omitempty" json:"url,omitempty"`
}

// HelmOCIManifest represents Helm chart OCI manifest
type HelmOCIManifest struct {
	SchemaVersion int                 `json:"schemaVersion"`
	MediaType     string              `json:"mediaType"`
	Config        HelmOCIDescriptor   `json:"config"`
	Layers        []HelmOCIDescriptor `json:"layers"`
	Annotations   map[string]string   `json:"annotations,omitempty"`
}

// HelmOCIDescriptor represents OCI content descriptor for Helm
type HelmOCIDescriptor struct {
	MediaType   string            `json:"mediaType"`
	Digest      string            `json:"digest"`
	Size        int64             `json:"size"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// HelmOCIConfig represents Helm chart configuration
type HelmOCIConfig struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Description string            `json:"description,omitempty"`
	Keywords    []string          `json:"keywords,omitempty"`
	Home        string            `json:"home,omitempty"`
	Sources     []string          `json:"sources,omitempty"`
	Maintainers []HelmMaintainer  `json:"maintainers,omitempty"`
	Icon        string            `json:"icon,omitempty"`
	AppVersion  string            `json:"appVersion,omitempty"`
	Deprecated  bool              `json:"deprecated,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	CreatedAt   time.Time         `json:"created"`
}

// Helm OCI media types
const (
	HelmChartConfigMediaType     = "application/vnd.cncf.helm.config.v1+json"
	HelmChartContentMediaType    = "application/vnd.cncf.helm.chart.content.v1.tar+gzip"
	HelmChartProvenanceMediaType = "application/vnd.cncf.helm.chart.provenance.v1.prov"
)

func NewHelmOCIHandler(basePath string) *HelmOCIHandler {
	return &HelmOCIHandler{basePath: basePath}
}

// GetManifestPath returns OCI manifest path for Helm chart
func (h *HelmOCIHandler) GetManifestPath(name, version string) string {
	// OCI format: /v2/<name>/manifests/<version>
	return filepath.Join(h.basePath, "v2", name, "manifests", version)
}

// GetBlobPath returns blob storage path
func (h *HelmOCIHandler) GetBlobPath(digest string) string {
	// OCI format: /v2/<name>/blobs/<digest>
	// For Helm, we can use a shared blob storage
	return filepath.Join(h.basePath, "blobs", digest)
}

// GetTraditionalPath returns traditional Helm chart path
func (h *HelmOCIHandler) GetTraditionalPath(name, version string) string {
	// Traditional format: /charts/<name>-<version>.tgz
	filename := fmt.Sprintf("%s-%s.tgz", name, version)
	return filepath.Join(h.basePath, "charts", filename)
}

// ValidateChartName validates Helm chart name
func (h *HelmOCIHandler) ValidateChartName(name string) error {
	if name == "" {
		return fmt.Errorf("chart name cannot be empty")
	}

	// Helm chart name rules (DNS-1123 subdomain)
	nameRegex := regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
	if !nameRegex.MatchString(name) {
		return fmt.Errorf("invalid chart name format: %s", name)
	}

	if len(name) > 63 {
		return fmt.Errorf("chart name too long (max 63 characters)")
	}

	return nil
}

// ValidateVersion validates Helm chart version (semantic versioning)
func (h *HelmOCIHandler) ValidateVersion(version string) error {
	// Semantic versioning for Helm charts
	semverRegex := regexp.MustCompile(`^v?(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)

	if !semverRegex.MatchString(version) {
		return fmt.Errorf("invalid semantic version format: %s", version)
	}

	return nil
}

// ExtractChartYAML extracts and validates Chart.yaml from tarball
func (h *HelmOCIHandler) ExtractChartYAML(tarballReader io.Reader) (*HelmChart, error) {
	// Open gzip stream
	gzReader, err := gzip.NewReader(tarballReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %v", err)
	}
	defer gzReader.Close()

	// Open tar stream
	tarReader := tar.NewReader(gzReader)

	// Look for Chart.yaml
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read tar header: %v", err)
		}

		// Chart.yaml should be at the root or in a single directory
		if strings.HasSuffix(header.Name, "/Chart.yaml") || header.Name == "Chart.yaml" {
			chartData, err := io.ReadAll(tarReader)
			if err != nil {
				return nil, fmt.Errorf("failed to read Chart.yaml: %v", err)
			}

			var chart HelmChart
			// Note: In a real implementation, you'd use a YAML parser
			// For now, we'll assume JSON format for simplicity
			if err := json.Unmarshal(chartData, &chart); err != nil {
				return nil, fmt.Errorf("invalid Chart.yaml format: %v", err)
			}

			return &chart, nil
		}
	}

	return nil, fmt.Errorf("Chart.yaml not found in tarball")
}

// ValidateChartStructure validates Helm chart directory structure
func (h *HelmOCIHandler) ValidateChartStructure(tarballReader io.Reader) error {
	// Open gzip stream
	gzReader, err := gzip.NewReader(tarballReader)
	if err != nil {
		return fmt.Errorf("invalid gzip format: %v", err)
	}
	defer gzReader.Close()

	// Open tar stream
	tarReader := tar.NewReader(gzReader)

	hasChartYAML := false
	hasTemplates := false
	chartDir := ""

	// Validate chart structure
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %v", err)
		}

		// Determine chart directory
		if chartDir == "" {
			parts := strings.Split(header.Name, "/")
			if len(parts) > 0 {
				chartDir = parts[0]
			}
		}

		// Check for required files
		if strings.HasSuffix(header.Name, "/Chart.yaml") || header.Name == "Chart.yaml" {
			hasChartYAML = true
		}

		if strings.Contains(header.Name, "/templates/") {
			hasTemplates = true
		}
	}

	if !hasChartYAML {
		return fmt.Errorf("Chart.yaml not found in chart")
	}

	// Templates directory is optional but common
	if !hasTemplates {
		// This is just a warning, not an error
		// Some charts might not have templates (e.g., library charts)
	}

	return nil
}

// CreateOCIManifest creates OCI manifest for Helm chart
func (h *HelmOCIHandler) CreateOCIManifest(chart *HelmChart, chartTarball []byte, configJSON []byte) (*HelmOCIManifest, error) {
	// Calculate digests (simplified - in real implementation use proper hash calculation)
	chartDigest := fmt.Sprintf("sha256:%x", len(chartTarball)) // Placeholder
	configDigest := fmt.Sprintf("sha256:%x", len(configJSON))  // Placeholder

	manifest := &HelmOCIManifest{
		SchemaVersion: 2,
		MediaType:     "application/vnd.oci.image.manifest.v1+json",
		Config: HelmOCIDescriptor{
			MediaType: HelmChartConfigMediaType,
			Digest:    configDigest,
			Size:      int64(len(configJSON)),
		},
		Layers: []HelmOCIDescriptor{
			{
				MediaType: HelmChartContentMediaType,
				Digest:    chartDigest,
				Size:      int64(len(chartTarball)),
			},
		},
		Annotations: map[string]string{
			"org.opencontainers.image.title":       chart.Name,
			"org.opencontainers.image.version":     chart.Version,
			"org.opencontainers.image.description": chart.Description,
		},
	}

	if chart.AppVersion != "" {
		manifest.Annotations["org.opencontainers.image.source"] = chart.AppVersion
	}

	return manifest, nil
}

// CreateOCIConfig creates OCI config for Helm chart
func (h *HelmOCIHandler) CreateOCIConfig(chart *HelmChart) *HelmOCIConfig {
	config := &HelmOCIConfig{
		Name:        chart.Name,
		Version:     chart.Version,
		Description: chart.Description,
		Keywords:    chart.Keywords,
		Home:        chart.Home,
		Sources:     chart.Sources,
		Maintainers: chart.Maintainers,
		Icon:        chart.Icon,
		AppVersion:  chart.AppVersion,
		Deprecated:  chart.Deprecated,
		Annotations: chart.Annotations,
		CreatedAt:   time.Now().UTC(),
	}

	return config
}

// ExtractMetadata extracts metadata from Helm chart
func (h *HelmOCIHandler) ExtractMetadata(chart *HelmChart) map[string]interface{} {
	metadata := map[string]interface{}{
		"name":         chart.Name,
		"version":      chart.Version,
		"app_version":  chart.AppVersion,
		"description":  chart.Description,
		"type":         chart.Type,
		"keywords":     chart.Keywords,
		"home":         chart.Home,
		"sources":      chart.Sources,
		"maintainers":  chart.Maintainers,
		"icon":         chart.Icon,
		"deprecated":   chart.Deprecated,
		"annotations":  chart.Annotations,
		"api_version":  chart.APIVersion,
		"kube_version": chart.KubeVersion,
	}

	if len(chart.Dependencies) > 0 {
		deps := make([]map[string]interface{}, len(chart.Dependencies))
		for i, dep := range chart.Dependencies {
			deps[i] = map[string]interface{}{
				"name":       dep.Name,
				"version":    dep.Version,
				"repository": dep.Repository,
				"condition":  dep.Condition,
				"tags":       dep.Tags,
				"enabled":    dep.Enabled,
				"alias":      dep.Alias,
			}
		}
		metadata["dependencies"] = deps
	}

	return metadata
}

// IsOCISupported checks if the chart supports OCI format
func (h *HelmOCIHandler) IsOCISupported(chart *HelmChart) bool {
	// Helm v3+ supports OCI, indicated by apiVersion
	return chart.APIVersion == "v2"
}

// GetStorageMode determines storage mode based on configuration
func (h *HelmOCIHandler) GetStorageMode(preferOCI bool, chart *HelmChart) string {
	if preferOCI && h.IsOCISupported(chart) {
		return "oci"
	}
	return "traditional"
}

// ValidateChartContent performs content validation on chart tarball
func (h *HelmOCIHandler) ValidateChartContent(tarballReader io.Reader) (*HelmChart, error) {
	// Extract Chart.yaml
	chart, err := h.ExtractChartYAML(tarballReader)
	if err != nil {
		return nil, err
	}

	// Validate chart metadata
	if err := h.ValidateChartName(chart.Name); err != nil {
		return nil, fmt.Errorf("invalid chart name: %v", err)
	}

	if err := h.ValidateVersion(chart.Version); err != nil {
		return nil, fmt.Errorf("invalid chart version: %v", err)
	}

	// Validate API version
	if chart.APIVersion != "v1" && chart.APIVersion != "v2" {
		return nil, fmt.Errorf("unsupported API version: %s", chart.APIVersion)
	}

	return chart, nil
}

// ConvertToOCI converts traditional Helm chart to OCI format
func (h *HelmOCIHandler) ConvertToOCI(chart *HelmChart, chartTarball []byte) (*HelmOCIManifest, []byte, []byte, error) {
	// Create OCI config
	config := h.CreateOCIConfig(chart)
	configJSON, err := json.Marshal(config)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to marshal config: %v", err)
	}

	// Create OCI manifest
	manifest, err := h.CreateOCIManifest(chart, chartTarball, configJSON)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create manifest: %v", err)
	}

	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to marshal manifest: %v", err)
	}

	return manifest, manifestJSON, configJSON, nil
}
