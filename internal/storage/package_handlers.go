package storage

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// PackageHandler defines the interface for package-specific storage operations
type PackageHandler interface {
	// Storage layout operations
	GetStoragePath(name, version string, extra map[string]string) string
	ValidatePackage(reader io.Reader) error
	ExtractMetadata(reader io.Reader) (map[string]interface{}, error)

	// Package-specific operations
	GetContentType() string
	GetFileExtensions() []string
	SupportsVersioning() bool
	RequiresManifest() bool
}

// OCIHandler handles Docker/OCI container images
type OCIHandler struct {
	basePath string
}

func NewOCIHandler(basePath string) *OCIHandler {
	return &OCIHandler{basePath: basePath}
}

func (h *OCIHandler) GetStoragePath(name, version string, extra map[string]string) string {
	// OCI Distribution Spec layout: /v2/<name>/manifests/<tag>
	// Blobs: /v2/<name>/blobs/<algorithm>:<encoded>

	if contentType, ok := extra["content_type"]; ok {
		if contentType == "application/vnd.docker.distribution.manifest.v2+json" ||
			contentType == "application/vnd.oci.image.manifest.v1+json" {
			return filepath.Join(h.basePath, "v2", name, "manifests", version)
		}
		if strings.HasPrefix(contentType, "application/vnd.docker.image.rootfs") {
			if digest, ok := extra["digest"]; ok {
				return filepath.Join(h.basePath, "v2", name, "blobs", digest)
			}
		}
	}

	// Default to manifests for tags
	return filepath.Join(h.basePath, "v2", name, "manifests", version)
}

func (h *OCIHandler) ValidatePackage(reader io.Reader) error {
	// Validate Docker/OCI manifest format
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	var manifest map[string]interface{}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("invalid JSON manifest: %v", err)
	}

	// Check for required OCI/Docker manifest fields
	if _, ok := manifest["schemaVersion"]; !ok {
		return fmt.Errorf("missing schemaVersion in manifest")
	}

	return nil
}

func (h *OCIHandler) ExtractMetadata(reader io.Reader) (map[string]interface{}, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	var manifest map[string]interface{}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}

	metadata := make(map[string]interface{})
	metadata["manifest"] = manifest
	metadata["schema_version"] = manifest["schemaVersion"]
	metadata["media_type"] = manifest["mediaType"]

	if config, ok := manifest["config"].(map[string]interface{}); ok {
		metadata["config_digest"] = config["digest"]
		metadata["config_size"] = config["size"]
	}

	if layers, ok := manifest["layers"].([]interface{}); ok {
		metadata["layer_count"] = len(layers)
		var totalSize int64
		for _, layer := range layers {
			if l, ok := layer.(map[string]interface{}); ok {
				if size, ok := l["size"].(float64); ok {
					totalSize += int64(size)
				}
			}
		}
		metadata["total_layer_size"] = totalSize
	}

	return metadata, nil
}

func (h *OCIHandler) GetContentType() string {
	return "application/vnd.docker.distribution.manifest.v2+json"
}

func (h *OCIHandler) GetFileExtensions() []string {
	return []string{".json", ".tar", ".tar.gz"}
}

func (h *OCIHandler) SupportsVersioning() bool {
	return true
}

func (h *OCIHandler) RequiresManifest() bool {
	return true
}

// NPMHandler handles Node.js packages
type NPMHandler struct {
	basePath string
}

func NewNPMHandler(basePath string) *NPMHandler {
	return &NPMHandler{basePath: basePath}
}

func (h *NPMHandler) GetStoragePath(name, version string, extra map[string]string) string {
	// npm registry layout: /<package-name>/<version>/<package-name>-<version>.tgz
	// For scoped packages: /@scope/<package-name>/<version>/<package-name>-<version>.tgz

	if strings.HasPrefix(name, "@") {
		// Scoped package
		parts := strings.Split(name, "/")
		if len(parts) == 2 {
			scope := parts[0]
			packageName := parts[1]
			filename := fmt.Sprintf("%s-%s.tgz", packageName, version)
			return filepath.Join(h.basePath, scope, packageName, version, filename)
		}
	}

	// Regular package
	filename := fmt.Sprintf("%s-%s.tgz", name, version)
	return filepath.Join(h.basePath, name, version, filename)
}

func (h *NPMHandler) ValidatePackage(reader io.Reader) error {
	// Validate npm package (tarball with package.json)
	// This would require extracting and checking package.json
	return nil // Simplified for now
}

func (h *NPMHandler) ExtractMetadata(reader io.Reader) (map[string]interface{}, error) {
	// Extract package.json from tarball and parse metadata
	metadata := make(map[string]interface{})
	metadata["package_type"] = "npm"
	metadata["registry_compatible"] = true
	return metadata, nil
}

func (h *NPMHandler) GetContentType() string {
	return "application/x-tar"
}

func (h *NPMHandler) GetFileExtensions() []string {
	return []string{".tgz", ".tar.gz"}
}

func (h *NPMHandler) SupportsVersioning() bool {
	return true
}

func (h *NPMHandler) RequiresManifest() bool {
	return false
}

// MavenHandler handles Java Maven artifacts
type MavenHandler struct {
	basePath string
}

func NewMavenHandler(basePath string) *MavenHandler {
	return &MavenHandler{basePath: basePath}
}

func (h *MavenHandler) GetStoragePath(name, version string, extra map[string]string) string {
	// Maven repository layout: /<groupId-as-path>/<artifactId>/<version>/<artifactId>-<version>.<packaging>
	// Example: org/springframework/spring-core/5.3.21/spring-core-5.3.21.jar

	groupId := extra["group_id"]
	artifactId := extra["artifact_id"]
	packaging := extra["packaging"]

	if groupId == "" || artifactId == "" {
		// Fallback if GAV not properly parsed
		return filepath.Join(h.basePath, name, version, fmt.Sprintf("%s-%s.jar", name, version))
	}

	if packaging == "" {
		packaging = "jar"
	}

	// Convert groupId dots to path separators
	groupPath := strings.ReplaceAll(groupId, ".", "/")
	filename := fmt.Sprintf("%s-%s.%s", artifactId, version, packaging)

	return filepath.Join(h.basePath, groupPath, artifactId, version, filename)
}

func (h *MavenHandler) ValidatePackage(reader io.Reader) error {
	// Validate JAR/WAR structure or POM XML
	return nil // Simplified for now
}

func (h *MavenHandler) ExtractMetadata(reader io.Reader) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	metadata["package_type"] = "maven"
	metadata["layout"] = "maven2"
	return metadata, nil
}

func (h *MavenHandler) GetContentType() string {
	return "application/java-archive"
}

func (h *MavenHandler) GetFileExtensions() []string {
	return []string{".jar", ".war", ".ear", ".pom", ".aar"}
}

func (h *MavenHandler) SupportsVersioning() bool {
	return true
}

func (h *MavenHandler) RequiresManifest() bool {
	return false
}

// PyPIHandler handles Python packages with PEP 503 compliance
type PyPIHandler struct {
	basePath string
}

func NewPyPIHandler(basePath string) *PyPIHandler {
	return &PyPIHandler{basePath: basePath}
}

func (h *PyPIHandler) GetStoragePath(name, version string, extra map[string]string) string {
	// PEP 503 Simple Repository API layout: /simple/<project>/
	// Files: /<project>/<filename>

	// Normalize package name per PEP 503
	normalizedName := strings.ToLower(strings.ReplaceAll(name, "_", "-"))

	if filename, ok := extra["filename"]; ok {
		return filepath.Join(h.basePath, "packages", normalizedName, filename)
	}

	// Default wheel filename pattern
	filename := fmt.Sprintf("%s-%s-py3-none-any.whl", normalizedName, version)
	return filepath.Join(h.basePath, "packages", normalizedName, filename)
}

func (h *PyPIHandler) ValidatePackage(reader io.Reader) error {
	// Validate wheel or sdist format
	return nil // Simplified for now
}

func (h *PyPIHandler) ExtractMetadata(reader io.Reader) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	metadata["package_type"] = "pypi"
	metadata["pep503_compatible"] = true
	return metadata, nil
}

func (h *PyPIHandler) GetContentType() string {
	return "application/x-wheel+zip"
}

func (h *PyPIHandler) GetFileExtensions() []string {
	return []string{".whl", ".tar.gz", ".zip"}
}

func (h *PyPIHandler) SupportsVersioning() bool {
	return true
}

func (h *PyPIHandler) RequiresManifest() bool {
	return false
}

// HelmHandler handles Helm charts (OCI format for v3+)
type HelmHandler struct {
	basePath string
}

func NewHelmHandler(basePath string) *HelmHandler {
	return &HelmHandler{basePath: basePath}
}

func (h *HelmHandler) GetStoragePath(name, version string, extra map[string]string) string {
	// Helm OCI format (similar to Docker OCI)
	// For traditional: /charts/<name>-<version>.tgz

	if useOCI, ok := extra["use_oci"]; ok && useOCI == "true" {
		// Use OCI format for Helm v3+
		return filepath.Join(h.basePath, "v2", name, "manifests", version)
	}

	// Traditional Helm chart format
	filename := fmt.Sprintf("%s-%s.tgz", name, version)
	return filepath.Join(h.basePath, "charts", filename)
}

func (h *HelmHandler) ValidatePackage(reader io.Reader) error {
	// Validate Helm chart structure (Chart.yaml, templates/, etc.)
	return nil // Simplified for now
}

func (h *HelmHandler) ExtractMetadata(reader io.Reader) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	metadata["package_type"] = "helm"
	metadata["chart_format"] = "v2"
	return metadata, nil
}

func (h *HelmHandler) GetContentType() string {
	return "application/vnd.cncf.helm.chart.content.v1.tar+gzip"
}

func (h *HelmHandler) GetFileExtensions() []string {
	return []string{".tgz", ".tar.gz"}
}

func (h *HelmHandler) SupportsVersioning() bool {
	return true
}

func (h *HelmHandler) RequiresManifest() bool {
	return false
}

// GenericHandler handles generic binary packages
type GenericHandler struct {
	basePath string
}

func NewGenericHandler(basePath string) *GenericHandler {
	return &GenericHandler{basePath: basePath}
}

func (h *GenericHandler) GetStoragePath(name, version string, extra map[string]string) string {
	// Simple layout: /<name>/<version>/<filename>
	if filename, ok := extra["filename"]; ok {
		return filepath.Join(h.basePath, name, version, filename)
	}
	return filepath.Join(h.basePath, name, version, name)
}

func (h *GenericHandler) ValidatePackage(reader io.Reader) error {
	// No specific validation for generic packages
	return nil
}

func (h *GenericHandler) ExtractMetadata(reader io.Reader) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	metadata["package_type"] = "generic"
	return metadata, nil
}

func (h *GenericHandler) GetContentType() string {
	return "application/octet-stream"
}

func (h *GenericHandler) GetFileExtensions() []string {
	return []string{} // Any extension allowed
}

func (h *GenericHandler) SupportsVersioning() bool {
	return true
}

func (h *GenericHandler) RequiresManifest() bool {
	return false
}

// PackageHandlerFactory creates appropriate handlers for package types
type PackageHandlerFactory struct {
	basePath string
}

func NewPackageHandlerFactory(basePath string) *PackageHandlerFactory {
	return &PackageHandlerFactory{basePath: basePath}
}

func (f *PackageHandlerFactory) GetHandler(packageType string) PackageHandler {
	switch packageType {
	case "docker":
		return NewOCIHandler(f.basePath)
	case "npm":
		return NewNPMHandler(f.basePath)
	case "maven":
		return NewMavenHandler(f.basePath)
	case "pypi":
		return NewPyPIHandler(f.basePath)
	case "helm":
		return NewHelmHandler(f.basePath)
	case "generic":
		return NewGenericHandler(f.basePath)
	default:
		return NewGenericHandler(f.basePath) // Default fallback
	}
}
