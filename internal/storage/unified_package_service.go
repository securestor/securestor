package storage

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// UnifiedPackageService provides a unified interface for all package types
type UnifiedPackageService struct {
	basePath string
	handlers map[string]PackageTypeHandler
}

// PackageTypeHandler defines the interface for package-specific operations
type PackageTypeHandler interface {
	ValidatePackage(reader io.Reader, filename string) error
	ExtractMetadata(reader io.Reader, filename string) (map[string]interface{}, error)
	GetStoragePath(name, version string, extra map[string]string) string
	GetContentType(filename string) string
	SupportsVersioning() bool
	RequiresManifest() bool
}

// PackageInfo contains unified package information
type PackageInfo struct {
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Type        string                 `json:"type"`
	Filename    string                 `json:"filename"`
	Size        int64                  `json:"size"`
	ContentType string                 `json:"content_type"`
	StoragePath string                 `json:"storage_path"`
	Metadata    map[string]interface{} `json:"metadata"`
	Checksums   map[string]string      `json:"checksums"`
	UploadedAt  string                 `json:"uploaded_at"`
	UploadedBy  string                 `json:"uploaded_by"`
}

// DockerOCIPackageHandler wraps OCI handler for unified interface
type DockerOCIPackageHandler struct {
	handler *OCIDistributionHandler
}

// NPMPackageHandler wraps npm handler for unified interface
type NPMPackageHandler struct {
	handler *NPMRegistryHandler
}

// MavenPackageHandler wraps maven handler for unified interface
type MavenPackageHandler struct {
	handler *MavenRepositoryHandler
}

// PyPIPackageHandler wraps pypi handler for unified interface
type PyPIPackageHandler struct {
	handler *PyPIPEP503Handler
}

// HelmPackageHandler wraps helm handler for unified interface
type HelmPackageHandler struct {
	handler *HelmOCIHandler
}

// GenericPackageHandler provides basic handling for generic packages
type GenericPackageHandler struct {
	basePath string
}

func NewUnifiedPackageService(basePath string) *UnifiedPackageService {
	service := &UnifiedPackageService{
		basePath: basePath,
		handlers: make(map[string]PackageTypeHandler),
	}

	// Initialize handlers for each package type
	service.handlers["docker"] = &DockerOCIPackageHandler{
		handler: NewOCIDistributionHandler(filepath.Join(basePath, "docker")),
	}
	service.handlers["npm"] = &NPMPackageHandler{
		handler: NewNPMRegistryHandler(filepath.Join(basePath, "npm")),
	}
	service.handlers["maven"] = &MavenPackageHandler{
		handler: NewMavenRepositoryHandler(filepath.Join(basePath, "maven")),
	}
	service.handlers["pypi"] = &PyPIPackageHandler{
		handler: NewPyPIPEP503Handler(filepath.Join(basePath, "pypi")),
	}
	service.handlers["helm"] = &HelmPackageHandler{
		handler: NewHelmOCIHandler(filepath.Join(basePath, "helm")),
	}
	service.handlers["generic"] = &GenericPackageHandler{
		basePath: filepath.Join(basePath, "generic"),
	}

	return service
}

// GetHandler returns the appropriate handler for a package type
func (s *UnifiedPackageService) GetHandler(packageType string) (PackageTypeHandler, error) {
	handler, exists := s.handlers[packageType]
	if !exists {
		return nil, fmt.Errorf("unsupported package type: %s", packageType)
	}
	return handler, nil
}

// ProcessPackage processes a package upload with the appropriate handler
func (s *UnifiedPackageService) ProcessPackage(packageType, name, version, filename string, reader io.Reader, size int64, extra map[string]string) (*PackageInfo, error) {
	handler, err := s.GetHandler(packageType)
	if err != nil {
		return nil, err
	}

	// Validate package
	if err := handler.ValidatePackage(reader, filename); err != nil {
		return nil, fmt.Errorf("package validation failed: %v", err)
	}

	// Extract metadata
	metadata, err := handler.ExtractMetadata(reader, filename)
	if err != nil {
		return nil, fmt.Errorf("metadata extraction failed: %v", err)
	}

	// Get storage path
	storagePath := handler.GetStoragePath(name, version, extra)

	// Create package info
	packageInfo := &PackageInfo{
		Name:        name,
		Version:     version,
		Type:        packageType,
		Filename:    filename,
		Size:        size,
		ContentType: handler.GetContentType(filename),
		StoragePath: storagePath,
		Metadata:    metadata,
	}

	return packageInfo, nil
}

// GetSupportedTypes returns list of supported package types
func (s *UnifiedPackageService) GetSupportedTypes() []string {
	types := make([]string, 0, len(s.handlers))
	for packageType := range s.handlers {
		types = append(types, packageType)
	}
	return types
}

// Implementation of PackageTypeHandler for Docker OCI

func (h *DockerOCIPackageHandler) ValidatePackage(reader io.Reader, filename string) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	return h.handler.ValidateManifest(data)
}

func (h *DockerOCIPackageHandler) ExtractMetadata(reader io.Reader, filename string) (map[string]interface{}, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return h.handler.ExtractManifestMetadata(data)
}

func (h *DockerOCIPackageHandler) GetStoragePath(name, version string, extra map[string]string) string {
	return h.handler.GetManifestPath(name, version)
}

func (h *DockerOCIPackageHandler) GetContentType(filename string) string {
	return "application/vnd.docker.distribution.manifest.v2+json"
}

func (h *DockerOCIPackageHandler) SupportsVersioning() bool {
	return true
}

func (h *DockerOCIPackageHandler) RequiresManifest() bool {
	return true
}

// Implementation of PackageTypeHandler for NPM

func (h *NPMPackageHandler) ValidatePackage(reader io.Reader, filename string) error {
	return h.handler.ValidateTarball(reader)
}

func (h *NPMPackageHandler) ExtractMetadata(reader io.Reader, filename string) (map[string]interface{}, error) {
	packageJSON, err := h.handler.ExtractPackageJSON(reader)
	if err != nil {
		return nil, err
	}

	metadata := map[string]interface{}{
		"name":         packageJSON.Name,
		"version":      packageJSON.Version,
		"description":  packageJSON.Description,
		"keywords":     packageJSON.Keywords,
		"homepage":     packageJSON.Homepage,
		"license":      packageJSON.License,
		"author":       packageJSON.Author,
		"dependencies": packageJSON.Dependencies,
		"engines":      packageJSON.Engines,
	}

	return metadata, nil
}

func (h *NPMPackageHandler) GetStoragePath(name, version string, extra map[string]string) string {
	return h.handler.GetTarballPath(name, version)
}

func (h *NPMPackageHandler) GetContentType(filename string) string {
	return "application/x-tar"
}

func (h *NPMPackageHandler) SupportsVersioning() bool {
	return true
}

func (h *NPMPackageHandler) RequiresManifest() bool {
	return false
}

// Implementation of PackageTypeHandler for Maven

func (h *MavenPackageHandler) ValidatePackage(reader io.Reader, filename string) error {
	if strings.HasSuffix(filename, ".pom") {
		data, err := io.ReadAll(reader)
		if err != nil {
			return err
		}
		_, err = h.handler.ValidatePOM(data)
		return err
	}
	// For JAR/WAR files, basic validation would go here
	return nil
}

func (h *MavenPackageHandler) ExtractMetadata(reader io.Reader, filename string) (map[string]interface{}, error) {
	if strings.HasSuffix(filename, ".pom") {
		data, err := io.ReadAll(reader)
		if err != nil {
			return nil, err
		}
		pom, err := h.handler.ValidatePOM(data)
		if err != nil {
			return nil, err
		}
		return h.handler.ExtractPOMMetadata(pom), nil
	}

	// For other Maven artifacts, return basic metadata
	return map[string]interface{}{
		"type":      "maven",
		"filename":  filename,
		"packaging": filepath.Ext(filename)[1:], // Remove the dot
	}, nil
}

func (h *MavenPackageHandler) GetStoragePath(name, version string, extra map[string]string) string {
	coords := &MavenCoordinates{
		GroupID:    extra["group_id"],
		ArtifactID: extra["artifact_id"],
		Version:    version,
		Packaging:  extra["packaging"],
		Classifier: extra["classifier"],
	}

	if coords.GroupID == "" || coords.ArtifactID == "" {
		// Fallback path
		return filepath.Join(h.handler.basePath, name, version, name+"-"+version+".jar")
	}

	return h.handler.GetArtifactPath(coords)
}

func (h *MavenPackageHandler) GetContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jar":
		return "application/java-archive"
	case ".war":
		return "application/java-archive"
	case ".pom":
		return "application/xml"
	default:
		return "application/octet-stream"
	}
}

func (h *MavenPackageHandler) SupportsVersioning() bool {
	return true
}

func (h *MavenPackageHandler) RequiresManifest() bool {
	return false
}

// Implementation of PackageTypeHandler for PyPI

func (h *PyPIPackageHandler) ValidatePackage(reader io.Reader, filename string) error {
	return h.handler.ValidatePackageType(filename)
}

func (h *PyPIPackageHandler) ExtractMetadata(reader io.Reader, filename string) (map[string]interface{}, error) {
	metadata, err := h.handler.GetMetadataFromFilename(filename)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"name":         metadata.Name,
		"version":      metadata.Version,
		"filename":     metadata.Filename,
		"file_type":    metadata.FileType,
		"python_tag":   metadata.PythonTag,
		"abi_tag":      metadata.AbiTag,
		"platform_tag": metadata.PlatformTag,
	}

	return result, nil
}

func (h *PyPIPackageHandler) GetStoragePath(name, version string, extra map[string]string) string {
	filename := extra["filename"]
	if filename == "" {
		filename = fmt.Sprintf("%s-%s.whl", name, version)
	}
	return h.handler.GetPackagePath(name, filename)
}

func (h *PyPIPackageHandler) GetContentType(filename string) string {
	if strings.HasSuffix(filename, ".whl") {
		return "application/x-wheel+zip"
	}
	return "application/x-tar"
}

func (h *PyPIPackageHandler) SupportsVersioning() bool {
	return true
}

func (h *PyPIPackageHandler) RequiresManifest() bool {
	return false
}

// Implementation of PackageTypeHandler for Helm

func (h *HelmPackageHandler) ValidatePackage(reader io.Reader, filename string) error {
	_, err := h.handler.ValidateChartContent(reader)
	return err
}

func (h *HelmPackageHandler) ExtractMetadata(reader io.Reader, filename string) (map[string]interface{}, error) {
	chart, err := h.handler.ValidateChartContent(reader)
	if err != nil {
		return nil, err
	}
	return h.handler.ExtractMetadata(chart), nil
}

func (h *HelmPackageHandler) GetStoragePath(name, version string, extra map[string]string) string {
	useOCI := extra["use_oci"] == "true"
	if useOCI {
		return h.handler.GetManifestPath(name, version)
	}
	return h.handler.GetTraditionalPath(name, version)
}

func (h *HelmPackageHandler) GetContentType(filename string) string {
	return "application/vnd.cncf.helm.chart.content.v1.tar+gzip"
}

func (h *HelmPackageHandler) SupportsVersioning() bool {
	return true
}

func (h *HelmPackageHandler) RequiresManifest() bool {
	return false
}

// Implementation of PackageTypeHandler for Generic

func (h *GenericPackageHandler) ValidatePackage(reader io.Reader, filename string) error {
	// No specific validation for generic packages
	return nil
}

func (h *GenericPackageHandler) ExtractMetadata(reader io.Reader, filename string) (map[string]interface{}, error) {
	return map[string]interface{}{
		"type":     "generic",
		"filename": filename,
	}, nil
}

func (h *GenericPackageHandler) GetStoragePath(name, version string, extra map[string]string) string {
	filename := extra["filename"]
	if filename == "" {
		filename = name
	}
	return filepath.Join(h.basePath, name, version, filename)
}

func (h *GenericPackageHandler) GetContentType(filename string) string {
	return "application/octet-stream"
}

func (h *GenericPackageHandler) SupportsVersioning() bool {
	return true
}

func (h *GenericPackageHandler) RequiresManifest() bool {
	return false
}
