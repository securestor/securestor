package storage

import (
	"archive/zip"
	"fmt"
	"html/template"
	"io"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// PyPIPEP503Handler handles Python Package Index with PEP 503 compliance
type PyPIPEP503Handler struct {
	basePath string
}

// PyPIPackageMetadata represents package metadata from wheel/egg-info
type PyPIPackageMetadata struct {
	Name            string            `json:"name"`
	Version         string            `json:"version"`
	Summary         string            `json:"summary,omitempty"`
	Description     string            `json:"description,omitempty"`
	Author          string            `json:"author,omitempty"`
	AuthorEmail     string            `json:"author_email,omitempty"`
	Maintainer      string            `json:"maintainer,omitempty"`
	MaintainerEmail string            `json:"maintainer_email,omitempty"`
	License         string            `json:"license,omitempty"`
	Homepage        string            `json:"homepage,omitempty"`
	Keywords        []string          `json:"keywords,omitempty"`
	Classifiers     []string          `json:"classifiers,omitempty"`
	RequiresDist    []string          `json:"requires_dist,omitempty"`
	RequiresPython  string            `json:"requires_python,omitempty"`
	ProjectURLs     map[string]string `json:"project_urls,omitempty"`
	PlatformTag     string            `json:"platform_tag,omitempty"`
	PythonTag       string            `json:"python_tag,omitempty"`
	AbiTag          string            `json:"abi_tag,omitempty"`
	Filename        string            `json:"filename"`
	FileType        string            `json:"file_type"` // wheel, sdist
	UploadTime      time.Time         `json:"upload_time"`
	Size            int64             `json:"size"`
	MD5Digest       string            `json:"md5_digest,omitempty"`
	SHA256Digest    string            `json:"sha256_digest"`
	Blake2b256      string            `json:"blake2b_256,omitempty"`
}

// PyPISimpleIndex represents the simple index page structure
type PyPISimpleIndex struct {
	Projects []string `json:"projects"`
}

// PyPIProjectIndex represents a project's file listing
type PyPIProjectIndex struct {
	ProjectName string         `json:"project_name"`
	Files       []PyPIFileInfo `json:"files"`
	LastUpdate  time.Time      `json:"last_update"`
}

// PyPIFileInfo represents file information for simple API
type PyPIFileInfo struct {
	Filename       string            `json:"filename"`
	URL            string            `json:"url"`
	HashDigests    map[string]string `json:"hash_digests"`
	RequiresPython string            `json:"requires_python,omitempty"`
	UploadTime     time.Time         `json:"upload_time"`
	Size           int64             `json:"size"`
}

// WheelMetadata represents wheel-specific metadata
type WheelMetadata struct {
	WheelVersion string `json:"wheel_version"`
	Generator    string `json:"generator"`
	RootIsPrefix bool   `json:"root_is_prefix"`
	Tag          string `json:"tag"`
}

func NewPyPIPEP503Handler(basePath string) *PyPIPEP503Handler {
	return &PyPIPEP503Handler{basePath: basePath}
}

// NormalizeProjectName normalizes project name per PEP 503
func (h *PyPIPEP503Handler) NormalizeProjectName(name string) string {
	// PEP 503: Normalize by lowercasing and replacing runs of [-_.] with a single -
	normalized := strings.ToLower(name)
	re := regexp.MustCompile(`[-_.]+`)
	return re.ReplaceAllString(normalized, "-")
}

// GetSimpleIndexPath returns path for simple index
func (h *PyPIPEP503Handler) GetSimpleIndexPath() string {
	return filepath.Join(h.basePath, "simple", "index.html")
}

// GetProjectIndexPath returns path for project simple index
func (h *PyPIPEP503Handler) GetProjectIndexPath(projectName string) string {
	normalized := h.NormalizeProjectName(projectName)
	return filepath.Join(h.basePath, "simple", normalized, "index.html")
}

// GetPackagePath returns storage path for package files
func (h *PyPIPEP503Handler) GetPackagePath(projectName, filename string) string {
	normalized := h.NormalizeProjectName(projectName)
	return filepath.Join(h.basePath, "packages", normalized, filename)
}

// ValidateProjectName validates Python project name
func (h *PyPIPEP503Handler) ValidateProjectName(name string) error {
	if name == "" {
		return fmt.Errorf("project name cannot be empty")
	}

	// PEP 508 compliant project name
	// Must contain only ASCII letters, numbers, periods, hyphens, and underscores
	nameRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?$`)
	if !nameRegex.MatchString(name) {
		return fmt.Errorf("invalid project name format: %s", name)
	}

	if len(name) > 214 {
		return fmt.Errorf("project name too long (max 214 characters)")
	}

	return nil
}

// ValidateVersion validates Python package version
func (h *PyPIPEP503Handler) ValidateVersion(version string) error {
	// PEP 440 version scheme validation (simplified)
	versionRegex := regexp.MustCompile(`^([1-9][0-9]*!)?(0|[1-9][0-9]*)(\.(0|[1-9][0-9]*))*((a|b|rc)(0|[1-9][0-9]*))?(\.post(0|[1-9][0-9]*))?(\.dev(0|[1-9][0-9]*))?$`)

	if !versionRegex.MatchString(version) {
		return fmt.Errorf("invalid version format (PEP 440): %s", version)
	}

	return nil
}

// ParseWheelFilename parses wheel filename to extract metadata
func (h *PyPIPEP503Handler) ParseWheelFilename(filename string) (*PyPIPackageMetadata, error) {
	// Wheel filename format: {name}-{version}(-{build tag})?-{python tag}-{abi tag}-{platform tag}.whl
	if !strings.HasSuffix(filename, ".whl") {
		return nil, fmt.Errorf("not a wheel file: %s", filename)
	}

	basename := strings.TrimSuffix(filename, ".whl")
	parts := strings.Split(basename, "-")

	if len(parts) < 5 {
		return nil, fmt.Errorf("invalid wheel filename format: %s", filename)
	}

	name := parts[0]
	version := parts[1]

	// Find python tag (usually 3rd or 4th element depending on build tag presence)
	var pythonTag, abiTag, platformTag string
	if len(parts) == 5 {
		// No build tag
		pythonTag = parts[2]
		abiTag = parts[3]
		platformTag = parts[4]
	} else if len(parts) == 6 {
		// With build tag
		pythonTag = parts[3]
		abiTag = parts[4]
		platformTag = parts[5]
	} else {
		return nil, fmt.Errorf("invalid wheel filename format: %s", filename)
	}

	metadata := &PyPIPackageMetadata{
		Name:        name,
		Version:     version,
		Filename:    filename,
		FileType:    "wheel",
		PythonTag:   pythonTag,
		AbiTag:      abiTag,
		PlatformTag: platformTag,
	}

	return metadata, nil
}

// ParseSDISTFilename parses source distribution filename
func (h *PyPIPEP503Handler) ParseSDISTFilename(filename string) (*PyPIPackageMetadata, error) {
	// Common sdist formats: {name}-{version}.tar.gz, {name}-{version}.zip
	var name, version string

	if strings.HasSuffix(filename, ".tar.gz") {
		basename := strings.TrimSuffix(filename, ".tar.gz")
		parts := strings.Split(basename, "-")
		if len(parts) < 2 {
			return nil, fmt.Errorf("invalid sdist filename format: %s", filename)
		}
		name = strings.Join(parts[:len(parts)-1], "-")
		version = parts[len(parts)-1]
	} else if strings.HasSuffix(filename, ".zip") {
		basename := strings.TrimSuffix(filename, ".zip")
		parts := strings.Split(basename, "-")
		if len(parts) < 2 {
			return nil, fmt.Errorf("invalid sdist filename format: %s", filename)
		}
		name = strings.Join(parts[:len(parts)-1], "-")
		version = parts[len(parts)-1]
	} else {
		return nil, fmt.Errorf("unsupported sdist format: %s", filename)
	}

	metadata := &PyPIPackageMetadata{
		Name:     name,
		Version:  version,
		Filename: filename,
		FileType: "sdist",
	}

	return metadata, nil
}

// ExtractWheelMetadata extracts metadata from wheel file
func (h *PyPIPEP503Handler) ExtractWheelMetadata(wheelReader io.ReaderAt, size int64) (*PyPIPackageMetadata, error) {
	zipReader, err := zip.NewReader(wheelReader, size)
	if err != nil {
		return nil, fmt.Errorf("failed to read wheel as zip: %v", err)
	}

	// Find METADATA file in .dist-info directory
	var metadataFile *zip.File
	var wheelFile *zip.File

	for _, file := range zipReader.File {
		if strings.HasSuffix(file.Name, "/METADATA") {
			metadataFile = file
		}
		if strings.HasSuffix(file.Name, "/WHEEL") {
			wheelFile = file
		}
	}

	if metadataFile == nil {
		return nil, fmt.Errorf("METADATA file not found in wheel")
	}

	// Read METADATA file
	metadataReader, err := metadataFile.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open METADATA file: %v", err)
	}
	defer metadataReader.Close()

	metadataContent, err := io.ReadAll(metadataReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read METADATA file: %v", err)
	}

	metadata := h.parseMetadataContent(string(metadataContent))

	// Read WHEEL file if present
	if wheelFile != nil {
		wheelReader, err := wheelFile.Open()
		if err == nil {
			defer wheelReader.Close()
			wheelContent, err := io.ReadAll(wheelReader)
			if err == nil {
				h.parseWheelContent(metadata, string(wheelContent))
			}
		}
	}

	return metadata, nil
}

// parseMetadataContent parses METADATA file content
func (h *PyPIPEP503Handler) parseMetadataContent(content string) *PyPIPackageMetadata {
	metadata := &PyPIPackageMetadata{}
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "Name":
			metadata.Name = value
		case "Version":
			metadata.Version = value
		case "Summary":
			metadata.Summary = value
		case "Author":
			metadata.Author = value
		case "Author-email":
			metadata.AuthorEmail = value
		case "Maintainer":
			metadata.Maintainer = value
		case "Maintainer-email":
			metadata.MaintainerEmail = value
		case "License":
			metadata.License = value
		case "Home-page":
			metadata.Homepage = value
		case "Requires-Python":
			metadata.RequiresPython = value
		case "Classifier":
			metadata.Classifiers = append(metadata.Classifiers, value)
		case "Requires-Dist":
			metadata.RequiresDist = append(metadata.RequiresDist, value)
		case "Keywords":
			metadata.Keywords = strings.Split(value, ",")
			for i, keyword := range metadata.Keywords {
				metadata.Keywords[i] = strings.TrimSpace(keyword)
			}
		}
	}

	return metadata
}

// parseWheelContent parses WHEEL file content
func (h *PyPIPEP503Handler) parseWheelContent(metadata *PyPIPackageMetadata, content string) {
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "Tag":
			// Extract python, abi, and platform tags from Tag field
			tagParts := strings.Split(value, "-")
			if len(tagParts) >= 3 {
				metadata.PythonTag = tagParts[0]
				metadata.AbiTag = tagParts[1]
				metadata.PlatformTag = strings.Join(tagParts[2:], "-")
			}
		}
	}
}

// GenerateSimpleIndex generates PEP 503 compliant simple index HTML
func (h *PyPIPEP503Handler) GenerateSimpleIndex(projects []string) string {
	sort.Strings(projects)

	tmpl := `<!DOCTYPE html>
<html>
<head>
    <meta name="pypi:repository-version" content="1.0">
    <title>Simple index</title>
</head>
<body>
    <h1>Simple index</h1>
{{range .Projects}}    <a href="{{.}}/">{{.}}</a><br/>
{{end}}</body>
</html>`

	t := template.Must(template.New("index").Parse(tmpl))
	var result strings.Builder

	data := struct {
		Projects []string
	}{
		Projects: projects,
	}

	t.Execute(&result, data)
	return result.String()
}

// GenerateProjectIndex generates PEP 503 compliant project index HTML
func (h *PyPIPEP503Handler) GenerateProjectIndex(projectName string, files []PyPIFileInfo) string {
	normalized := h.NormalizeProjectName(projectName)

	tmpl := `<!DOCTYPE html>
<html>
<head>
    <meta name="pypi:repository-version" content="1.0">
    <title>Links for {{.ProjectName}}</title>
</head>
<body>
    <h1>Links for {{.ProjectName}}</h1>
{{range .Files}}    <a href="{{.URL}}"{{if .RequiresPython}} data-requires-python="{{.RequiresPython}}"{{end}}{{range $algo, $hash := .HashDigests}} data-dist-info-metadata="{{$algo}}={{$hash}}"{{end}}>{{.Filename}}</a><br/>
{{end}}</body>
</html>`

	t := template.Must(template.New("project").Parse(tmpl))
	var result strings.Builder

	data := struct {
		ProjectName string
		Files       []PyPIFileInfo
	}{
		ProjectName: normalized,
		Files:       files,
	}

	t.Execute(&result, data)
	return result.String()
}

// CreateFileInfo creates PyPIFileInfo from metadata
func (h *PyPIPEP503Handler) CreateFileInfo(metadata *PyPIPackageMetadata, baseURL string) PyPIFileInfo {
	hashDigests := make(map[string]string)

	if metadata.SHA256Digest != "" {
		hashDigests["sha256"] = metadata.SHA256Digest
	}
	if metadata.MD5Digest != "" {
		hashDigests["md5"] = metadata.MD5Digest
	}
	if metadata.Blake2b256 != "" {
		hashDigests["blake2b_256"] = metadata.Blake2b256
	}

	projectName := h.NormalizeProjectName(metadata.Name)
	url := fmt.Sprintf("%s/packages/%s/%s", baseURL, projectName, metadata.Filename)

	return PyPIFileInfo{
		Filename:       metadata.Filename,
		URL:            url,
		HashDigests:    hashDigests,
		RequiresPython: metadata.RequiresPython,
		UploadTime:     metadata.UploadTime,
		Size:           metadata.Size,
	}
}

// ValidatePackageType validates if file is a supported Python package
func (h *PyPIPEP503Handler) ValidatePackageType(filename string) error {
	validExtensions := []string{".whl", ".tar.gz", ".zip", ".egg"}

	for _, ext := range validExtensions {
		if strings.HasSuffix(filename, ext) {
			return nil
		}
	}

	return fmt.Errorf("unsupported package type: %s", filename)
}

// GetMetadataFromFilename attempts to extract metadata from filename only
func (h *PyPIPEP503Handler) GetMetadataFromFilename(filename string) (*PyPIPackageMetadata, error) {
	if strings.HasSuffix(filename, ".whl") {
		return h.ParseWheelFilename(filename)
	} else if strings.HasSuffix(filename, ".tar.gz") || strings.HasSuffix(filename, ".zip") {
		return h.ParseSDISTFilename(filename)
	}

	return nil, fmt.Errorf("unsupported file type: %s", filename)
}
