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

// NPMRegistryHandler handles npm registry compatible storage
type NPMRegistryHandler struct {
	basePath string
}

// NPMPackageJSON represents the package.json structure
type NPMPackageJSON struct {
	Name                 string                 `json:"name"`
	Version              string                 `json:"version"`
	Description          string                 `json:"description,omitempty"`
	Keywords             []string               `json:"keywords,omitempty"`
	Homepage             string                 `json:"homepage,omitempty"`
	Bugs                 interface{}            `json:"bugs,omitempty"`
	License              string                 `json:"license,omitempty"`
	Author               interface{}            `json:"author,omitempty"`
	Contributors         []interface{}          `json:"contributors,omitempty"`
	Files                []string               `json:"files,omitempty"`
	Main                 string                 `json:"main,omitempty"`
	Browser              interface{}            `json:"browser,omitempty"`
	Bin                  interface{}            `json:"bin,omitempty"`
	Man                  interface{}            `json:"man,omitempty"`
	Directories          map[string]string      `json:"directories,omitempty"`
	Repository           interface{}            `json:"repository,omitempty"`
	Scripts              map[string]string      `json:"scripts,omitempty"`
	Config               map[string]interface{} `json:"config,omitempty"`
	Dependencies         map[string]string      `json:"dependencies,omitempty"`
	DevDependencies      map[string]string      `json:"devDependencies,omitempty"`
	PeerDependencies     map[string]string      `json:"peerDependencies,omitempty"`
	OptionalDependencies map[string]string      `json:"optionalDependencies,omitempty"`
	BundledDependencies  []string               `json:"bundledDependencies,omitempty"`
	Engines              map[string]string      `json:"engines,omitempty"`
	Os                   []string               `json:"os,omitempty"`
	Cpu                  []string               `json:"cpu,omitempty"`
	Private              bool                   `json:"private,omitempty"`
	PublishConfig        map[string]interface{} `json:"publishConfig,omitempty"`
	Workspaces           interface{}            `json:"workspaces,omitempty"`
}

// NPMRegistryMetadata represents npm registry metadata document
type NPMRegistryMetadata struct {
	ID             string                        `json:"_id"`
	Rev            string                        `json:"_rev,omitempty"`
	Name           string                        `json:"name"`
	Description    string                        `json:"description,omitempty"`
	DistTags       map[string]string             `json:"dist-tags"`
	Versions       map[string]NPMVersionMetadata `json:"versions"`
	Time           map[string]time.Time          `json:"time"`
	Users          map[string]bool               `json:"users,omitempty"`
	Author         interface{}                   `json:"author,omitempty"`
	Repository     interface{}                   `json:"repository,omitempty"`
	Homepage       string                        `json:"homepage,omitempty"`
	Keywords       []string                      `json:"keywords,omitempty"`
	Bugs           interface{}                   `json:"bugs,omitempty"`
	License        string                        `json:"license,omitempty"`
	ReadmeFilename string                        `json:"readmeFilename,omitempty"`
	Readme         string                        `json:"readme,omitempty"`
}

// NPMVersionMetadata represents version-specific metadata
type NPMVersionMetadata struct {
	Name                 string                 `json:"name"`
	Version              string                 `json:"version"`
	Description          string                 `json:"description,omitempty"`
	Keywords             []string               `json:"keywords,omitempty"`
	Homepage             string                 `json:"homepage,omitempty"`
	Bugs                 interface{}            `json:"bugs,omitempty"`
	License              string                 `json:"license,omitempty"`
	Author               interface{}            `json:"author,omitempty"`
	Contributors         []interface{}          `json:"contributors,omitempty"`
	Files                []string               `json:"files,omitempty"`
	Main                 string                 `json:"main,omitempty"`
	Browser              interface{}            `json:"browser,omitempty"`
	Bin                  interface{}            `json:"bin,omitempty"`
	Man                  interface{}            `json:"man,omitempty"`
	Directories          map[string]string      `json:"directories,omitempty"`
	Repository           interface{}            `json:"repository,omitempty"`
	Scripts              map[string]string      `json:"scripts,omitempty"`
	Config               map[string]interface{} `json:"config,omitempty"`
	Dependencies         map[string]string      `json:"dependencies,omitempty"`
	DevDependencies      map[string]string      `json:"devDependencies,omitempty"`
	PeerDependencies     map[string]string      `json:"peerDependencies,omitempty"`
	OptionalDependencies map[string]string      `json:"optionalDependencies,omitempty"`
	BundledDependencies  []string               `json:"bundledDependencies,omitempty"`
	Engines              map[string]string      `json:"engines,omitempty"`
	Os                   []string               `json:"os,omitempty"`
	Cpu                  []string               `json:"cpu,omitempty"`
	Dist                 NPMDistInfo            `json:"dist"`
	ID                   string                 `json:"_id"`
	NodeVersion          string                 `json:"_nodeVersion,omitempty"`
	NpmVersion           string                 `json:"_npmVersion,omitempty"`
	HasShrinkwrap        bool                   `json:"_hasShrinkwrap,omitempty"`
}

// NPMDistInfo represents distribution information
type NPMDistInfo struct {
	Integrity    string         `json:"integrity,omitempty"`
	Shasum       string         `json:"shasum"`
	Tarball      string         `json:"tarball"`
	FileCount    int            `json:"fileCount,omitempty"`
	UnpackedSize int64          `json:"unpackedSize,omitempty"`
	Signatures   []NPMSignature `json:"signatures,omitempty"`
}

// NPMSignature represents package signature
type NPMSignature struct {
	Keyid string `json:"keyid"`
	Sig   string `json:"sig"`
}

func NewNPMRegistryHandler(basePath string) *NPMRegistryHandler {
	return &NPMRegistryHandler{basePath: basePath}
}

// GetPackagePath returns the path for storing package metadata
func (h *NPMRegistryHandler) GetPackagePath(name string) string {
	// npm registry layout: /<package-name> for metadata
	if strings.HasPrefix(name, "@") {
		// Scoped package: /@scope%2F<package-name>
		return filepath.Join(h.basePath, strings.ReplaceAll(name, "/", "%2F"))
	}
	return filepath.Join(h.basePath, name)
}

// GetTarballPath returns the path for storing package tarballs
func (h *NPMRegistryHandler) GetTarballPath(name, version string) string {
	// npm registry layout for tarballs: /<package-name>/-/<package-name>-<version>.tgz
	if strings.HasPrefix(name, "@") {
		// Scoped package
		parts := strings.SplitN(name, "/", 2)
		if len(parts) == 2 {
			scope := parts[0]
			packageName := parts[1]
			filename := fmt.Sprintf("%s-%s.tgz", packageName, version)
			return filepath.Join(h.basePath, scope, packageName, "-", filename)
		}
	}

	// Regular package
	filename := fmt.Sprintf("%s-%s.tgz", name, version)
	return filepath.Join(h.basePath, name, "-", filename)
}

// ValidatePackageName validates npm package name
func (h *NPMRegistryHandler) ValidatePackageName(name string) error {
	// npm package name rules:
	// - lowercase only
	// - can contain hyphens and dots
	// - cannot start with dot or underscore
	// - scoped packages: @scope/package-name

	if name == "" {
		return fmt.Errorf("package name cannot be empty")
	}

	if len(name) > 214 {
		return fmt.Errorf("package name too long (max 214 characters)")
	}

	// Check for scoped package
	if strings.HasPrefix(name, "@") {
		scopedRegex := regexp.MustCompile(`^@[a-z0-9]([a-z0-9._-])*[a-z0-9]/[a-z0-9]([a-z0-9._-])*[a-z0-9]$`)
		if !scopedRegex.MatchString(name) {
			return fmt.Errorf("invalid scoped package name format")
		}
	} else {
		// Regular package name
		nameRegex := regexp.MustCompile(`^[a-z0-9]([a-z0-9._-])*[a-z0-9]$`)
		if !nameRegex.MatchString(name) {
			return fmt.Errorf("invalid package name format")
		}
	}

	// Check for URL-unsafe characters
	if strings.ContainsAny(name, " ~)('!*") {
		return fmt.Errorf("package name contains invalid characters")
	}

	return nil
}

// ValidateVersion validates semantic version
func (h *NPMRegistryHandler) ValidateVersion(version string) error {
	// Semantic versioning pattern
	semverRegex := regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)

	if !semverRegex.MatchString(version) {
		return fmt.Errorf("invalid semantic version format: %s", version)
	}

	return nil
}

// ExtractPackageJSON extracts and validates package.json from tarball
func (h *NPMRegistryHandler) ExtractPackageJSON(tarballReader io.Reader) (*NPMPackageJSON, error) {
	// Open gzip stream
	gzReader, err := gzip.NewReader(tarballReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %v", err)
	}
	defer gzReader.Close()

	// Open tar stream
	tarReader := tar.NewReader(gzReader)

	// Look for package/package.json
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read tar header: %v", err)
		}

		// npm tarballs have package.json at package/package.json
		if header.Name == "package/package.json" {
			packageJSONData, err := io.ReadAll(tarReader)
			if err != nil {
				return nil, fmt.Errorf("failed to read package.json: %v", err)
			}

			var packageJSON NPMPackageJSON
			if err := json.Unmarshal(packageJSONData, &packageJSON); err != nil {
				return nil, fmt.Errorf("invalid package.json format: %v", err)
			}

			return &packageJSON, nil
		}
	}

	return nil, fmt.Errorf("package.json not found in tarball")
}

// CreateRegistryMetadata creates npm registry metadata document
func (h *NPMRegistryHandler) CreateRegistryMetadata(packageJSON *NPMPackageJSON, tarballURL string, shasum string) *NPMRegistryMetadata {
	now := time.Now()
	versionID := fmt.Sprintf("%s@%s", packageJSON.Name, packageJSON.Version)

	versionMetadata := NPMVersionMetadata{
		Name:                 packageJSON.Name,
		Version:              packageJSON.Version,
		Description:          packageJSON.Description,
		Keywords:             packageJSON.Keywords,
		Homepage:             packageJSON.Homepage,
		Bugs:                 packageJSON.Bugs,
		License:              packageJSON.License,
		Author:               packageJSON.Author,
		Contributors:         packageJSON.Contributors,
		Files:                packageJSON.Files,
		Main:                 packageJSON.Main,
		Browser:              packageJSON.Browser,
		Bin:                  packageJSON.Bin,
		Man:                  packageJSON.Man,
		Directories:          packageJSON.Directories,
		Repository:           packageJSON.Repository,
		Scripts:              packageJSON.Scripts,
		Config:               packageJSON.Config,
		Dependencies:         packageJSON.Dependencies,
		DevDependencies:      packageJSON.DevDependencies,
		PeerDependencies:     packageJSON.PeerDependencies,
		OptionalDependencies: packageJSON.OptionalDependencies,
		BundledDependencies:  packageJSON.BundledDependencies,
		Engines:              packageJSON.Engines,
		Os:                   packageJSON.Os,
		Cpu:                  packageJSON.Cpu,
		ID:                   versionID,
		Dist: NPMDistInfo{
			Shasum:  shasum,
			Tarball: tarballURL,
		},
	}

	metadata := &NPMRegistryMetadata{
		ID:          packageJSON.Name,
		Name:        packageJSON.Name,
		Description: packageJSON.Description,
		DistTags: map[string]string{
			"latest": packageJSON.Version,
		},
		Versions: map[string]NPMVersionMetadata{
			packageJSON.Version: versionMetadata,
		},
		Time: map[string]time.Time{
			"created":           now,
			"modified":          now,
			packageJSON.Version: now,
		},
		Author:     packageJSON.Author,
		Repository: packageJSON.Repository,
		Homepage:   packageJSON.Homepage,
		Keywords:   packageJSON.Keywords,
		Bugs:       packageJSON.Bugs,
		License:    packageJSON.License,
	}

	return metadata
}

// UpdateRegistryMetadata updates existing metadata with new version
func (h *NPMRegistryHandler) UpdateRegistryMetadata(existing *NPMRegistryMetadata, packageJSON *NPMPackageJSON, tarballURL string, shasum string) *NPMRegistryMetadata {
	now := time.Now()
	versionID := fmt.Sprintf("%s@%s", packageJSON.Name, packageJSON.Version)

	versionMetadata := NPMVersionMetadata{
		Name:                 packageJSON.Name,
		Version:              packageJSON.Version,
		Description:          packageJSON.Description,
		Keywords:             packageJSON.Keywords,
		Homepage:             packageJSON.Homepage,
		Bugs:                 packageJSON.Bugs,
		License:              packageJSON.License,
		Author:               packageJSON.Author,
		Contributors:         packageJSON.Contributors,
		Files:                packageJSON.Files,
		Main:                 packageJSON.Main,
		Browser:              packageJSON.Browser,
		Bin:                  packageJSON.Bin,
		Man:                  packageJSON.Man,
		Directories:          packageJSON.Directories,
		Repository:           packageJSON.Repository,
		Scripts:              packageJSON.Scripts,
		Config:               packageJSON.Config,
		Dependencies:         packageJSON.Dependencies,
		DevDependencies:      packageJSON.DevDependencies,
		PeerDependencies:     packageJSON.PeerDependencies,
		OptionalDependencies: packageJSON.OptionalDependencies,
		BundledDependencies:  packageJSON.BundledDependencies,
		Engines:              packageJSON.Engines,
		Os:                   packageJSON.Os,
		Cpu:                  packageJSON.Cpu,
		ID:                   versionID,
		Dist: NPMDistInfo{
			Shasum:  shasum,
			Tarball: tarballURL,
		},
	}

	// Add new version to existing metadata
	existing.Versions[packageJSON.Version] = versionMetadata
	existing.Time[packageJSON.Version] = now
	existing.Time["modified"] = now

	// Update latest tag if this is the newest version
	// In a real implementation, you'd use semver comparison
	existing.DistTags["latest"] = packageJSON.Version

	return existing
}

// GetVersions returns all versions for a package
func (h *NPMRegistryHandler) GetVersions(metadata *NPMRegistryMetadata) []string {
	versions := make([]string, 0, len(metadata.Versions))
	for version := range metadata.Versions {
		versions = append(versions, version)
	}
	return versions
}

// GetDistTags returns distribution tags for a package
func (h *NPMRegistryHandler) GetDistTags(metadata *NPMRegistryMetadata) map[string]string {
	return metadata.DistTags
}

// ValidateTarball validates npm package tarball structure
func (h *NPMRegistryHandler) ValidateTarball(tarballReader io.Reader) error {
	// Open gzip stream
	gzReader, err := gzip.NewReader(tarballReader)
	if err != nil {
		return fmt.Errorf("invalid gzip format: %v", err)
	}
	defer gzReader.Close()

	// Open tar stream
	tarReader := tar.NewReader(gzReader)

	hasPackageJSON := false
	topLevelDir := ""

	// Validate tarball structure
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %v", err)
		}

		// All files should be under package/ directory
		if !strings.HasPrefix(header.Name, "package/") {
			return fmt.Errorf("invalid file path in tarball: %s", header.Name)
		}

		// Set top level directory
		if topLevelDir == "" {
			parts := strings.Split(header.Name, "/")
			if len(parts) > 0 {
				topLevelDir = parts[0]
			}
		}

		// Check for package.json
		if header.Name == "package/package.json" {
			hasPackageJSON = true
		}
	}

	if !hasPackageJSON {
		return fmt.Errorf("package.json not found in tarball")
	}

	if topLevelDir != "package" {
		return fmt.Errorf("tarball must have all files under package/ directory")
	}

	return nil
}
