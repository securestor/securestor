package storage

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// MavenRepositoryHandler handles Maven repository layout and metadata
type MavenRepositoryHandler struct {
	basePath string
}

// MavenCoordinates represents Maven GAV coordinates
type MavenCoordinates struct {
	GroupID    string `json:"group_id"`
	ArtifactID string `json:"artifact_id"`
	Version    string `json:"version"`
	Packaging  string `json:"packaging"`
	Classifier string `json:"classifier,omitempty"`
}

// MavenMetadata represents maven-metadata.xml structure
type MavenMetadata struct {
	XMLName    xml.Name         `xml:"metadata"`
	GroupID    string           `xml:"groupId"`
	ArtifactID string           `xml:"artifactId"`
	Version    string           `xml:"version,omitempty"`
	Versioning *MavenVersioning `xml:"versioning,omitempty"`
	Plugins    *MavenPlugins    `xml:"plugins,omitempty"`
}

// MavenVersioning represents versioning information
type MavenVersioning struct {
	Latest           string                 `xml:"latest,omitempty"`
	Release          string                 `xml:"release,omitempty"`
	Versions         *MavenVersions         `xml:"versions,omitempty"`
	LastUpdated      string                 `xml:"lastUpdated"`
	Snapshot         *MavenSnapshot         `xml:"snapshot,omitempty"`
	SnapshotVersions *MavenSnapshotVersions `xml:"snapshotVersions,omitempty"`
}

// MavenVersions represents list of versions
type MavenVersions struct {
	Version []string `xml:"version"`
}

// MavenSnapshot represents snapshot information
type MavenSnapshot struct {
	Timestamp   string `xml:"timestamp"`
	BuildNumber int    `xml:"buildNumber"`
	LocalCopy   bool   `xml:"localCopy,omitempty"`
}

// MavenSnapshotVersions represents snapshot version list
type MavenSnapshotVersions struct {
	SnapshotVersion []MavenSnapshotVersion `xml:"snapshotVersion"`
}

// MavenSnapshotVersion represents individual snapshot version
type MavenSnapshotVersion struct {
	Extension  string `xml:"extension"`
	Value      string `xml:"value"`
	Updated    string `xml:"updated"`
	Classifier string `xml:"classifier,omitempty"`
}

// MavenPlugins represents plugin metadata
type MavenPlugins struct {
	Plugin []MavenPlugin `xml:"plugin"`
}

// MavenPlugin represents plugin information
type MavenPlugin struct {
	Name       string `xml:"name"`
	Prefix     string `xml:"prefix"`
	ArtifactID string `xml:"artifactId"`
}

// POMProject represents a simplified POM structure
type POMProject struct {
	XMLName      xml.Name         `xml:"project"`
	ModelVersion string           `xml:"modelVersion"`
	GroupID      string           `xml:"groupId"`
	ArtifactID   string           `xml:"artifactId"`
	Version      string           `xml:"version"`
	Packaging    string           `xml:"packaging,omitempty"`
	Name         string           `xml:"name,omitempty"`
	Description  string           `xml:"description,omitempty"`
	URL          string           `xml:"url,omitempty"`
	Parent       *POMParent       `xml:"parent,omitempty"`
	Properties   *POMProperties   `xml:"properties,omitempty"`
	Dependencies *POMDependencies `xml:"dependencies,omitempty"`
	Licenses     *POMLicenses     `xml:"licenses,omitempty"`
	Developers   *POMDevelopers   `xml:"developers,omitempty"`
	SCM          *POMSCM          `xml:"scm,omitempty"`
}

// POMParent represents parent project
type POMParent struct {
	GroupID    string `xml:"groupId"`
	ArtifactID string `xml:"artifactId"`
	Version    string `xml:"version"`
}

// POMProperties represents project properties
type POMProperties struct {
	Properties map[string]string `xml:",any"`
}

// POMDependencies represents project dependencies
type POMDependencies struct {
	Dependency []POMDependency `xml:"dependency"`
}

// POMDependency represents a single dependency
type POMDependency struct {
	GroupID    string `xml:"groupId"`
	ArtifactID string `xml:"artifactId"`
	Version    string `xml:"version"`
	Type       string `xml:"type,omitempty"`
	Classifier string `xml:"classifier,omitempty"`
	Scope      string `xml:"scope,omitempty"`
	Optional   bool   `xml:"optional,omitempty"`
}

// POMLicenses represents license information
type POMLicenses struct {
	License []POMLicense `xml:"license"`
}

// POMLicense represents a single license
type POMLicense struct {
	Name         string `xml:"name"`
	URL          string `xml:"url"`
	Distribution string `xml:"distribution,omitempty"`
}

// POMDevelopers represents developer information
type POMDevelopers struct {
	Developer []POMDeveloper `xml:"developer"`
}

// POMDeveloper represents a single developer
type POMDeveloper struct {
	ID    string `xml:"id,omitempty"`
	Name  string `xml:"name"`
	Email string `xml:"email,omitempty"`
	URL   string `xml:"url,omitempty"`
}

// POMSCM represents source control information
type POMSCM struct {
	Connection          string `xml:"connection,omitempty"`
	DeveloperConnection string `xml:"developerConnection,omitempty"`
	Tag                 string `xml:"tag,omitempty"`
	URL                 string `xml:"url,omitempty"`
}

func NewMavenRepositoryHandler(basePath string) *MavenRepositoryHandler {
	return &MavenRepositoryHandler{basePath: basePath}
}

// ParseCoordinates parses Maven coordinates from various formats
func (h *MavenRepositoryHandler) ParseCoordinates(coords string) (*MavenCoordinates, error) {
	// Support multiple formats:
	// groupId:artifactId:version
	// groupId:artifactId:packaging:version
	// groupId:artifactId:packaging:classifier:version

	parts := strings.Split(coords, ":")
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid Maven coordinates format: %s", coords)
	}

	coordinates := &MavenCoordinates{
		GroupID:    parts[0],
		ArtifactID: parts[1],
		Packaging:  "jar", // default
	}

	switch len(parts) {
	case 3:
		// groupId:artifactId:version
		coordinates.Version = parts[2]
	case 4:
		// groupId:artifactId:packaging:version
		coordinates.Packaging = parts[2]
		coordinates.Version = parts[3]
	case 5:
		// groupId:artifactId:packaging:classifier:version
		coordinates.Packaging = parts[2]
		coordinates.Classifier = parts[3]
		coordinates.Version = parts[4]
	default:
		return nil, fmt.Errorf("invalid Maven coordinates format: %s", coords)
	}

	return coordinates, nil
}

// GetArtifactPath returns the storage path for an artifact
func (h *MavenRepositoryHandler) GetArtifactPath(coords *MavenCoordinates) string {
	// Maven repository layout:
	// /<groupId-as-path>/<artifactId>/<version>/<artifactId>-<version>[-classifier].<packaging>

	groupPath := strings.ReplaceAll(coords.GroupID, ".", "/")
	filename := coords.ArtifactID + "-" + coords.Version

	if coords.Classifier != "" {
		filename += "-" + coords.Classifier
	}

	filename += "." + coords.Packaging

	return filepath.Join(h.basePath, groupPath, coords.ArtifactID, coords.Version, filename)
}

// GetPOMPath returns the path for the POM file
func (h *MavenRepositoryHandler) GetPOMPath(coords *MavenCoordinates) string {
	groupPath := strings.ReplaceAll(coords.GroupID, ".", "/")
	filename := fmt.Sprintf("%s-%s.pom", coords.ArtifactID, coords.Version)
	return filepath.Join(h.basePath, groupPath, coords.ArtifactID, coords.Version, filename)
}

// GetMetadataPath returns the path for maven-metadata.xml
func (h *MavenRepositoryHandler) GetMetadataPath(groupID, artifactID string) string {
	groupPath := strings.ReplaceAll(groupID, ".", "/")
	return filepath.Join(h.basePath, groupPath, artifactID, "maven-metadata.xml")
}

// GetVersionMetadataPath returns the path for version-specific metadata
func (h *MavenRepositoryHandler) GetVersionMetadataPath(coords *MavenCoordinates) string {
	groupPath := strings.ReplaceAll(coords.GroupID, ".", "/")
	return filepath.Join(h.basePath, groupPath, coords.ArtifactID, coords.Version, "maven-metadata.xml")
}

// GetChecksumPath returns the path for checksum files
func (h *MavenRepositoryHandler) GetChecksumPath(filePath, algorithm string) string {
	switch algorithm {
	case "md5":
		return filePath + ".md5"
	case "sha1":
		return filePath + ".sha1"
	case "sha256":
		return filePath + ".sha256"
	case "sha512":
		return filePath + ".sha512"
	default:
		return filePath + ".sha1" // default to SHA1
	}
}

// ValidateCoordinates validates Maven coordinates
func (h *MavenRepositoryHandler) ValidateCoordinates(coords *MavenCoordinates) error {
	if coords.GroupID == "" {
		return fmt.Errorf("groupId cannot be empty")
	}
	if coords.ArtifactID == "" {
		return fmt.Errorf("artifactId cannot be empty")
	}
	if coords.Version == "" {
		return fmt.Errorf("version cannot be empty")
	}

	// Validate groupId format
	groupIDRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-_.])*[a-zA-Z0-9]$`)
	if !groupIDRegex.MatchString(coords.GroupID) {
		return fmt.Errorf("invalid groupId format: %s", coords.GroupID)
	}

	// Validate artifactId format
	artifactIDRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-_.])*[a-zA-Z0-9]$`)
	if !artifactIDRegex.MatchString(coords.ArtifactID) {
		return fmt.Errorf("invalid artifactId format: %s", coords.ArtifactID)
	}

	// Validate version format (semantic versioning or Maven version)
	versionRegex := regexp.MustCompile(`^[0-9]([0-9a-zA-Z\-_.])*[a-zA-Z0-9]$`)
	if !versionRegex.MatchString(coords.Version) {
		return fmt.Errorf("invalid version format: %s", coords.Version)
	}

	return nil
}

// IsSnapshot checks if version is a snapshot
func (h *MavenRepositoryHandler) IsSnapshot(version string) bool {
	return strings.HasSuffix(version, "-SNAPSHOT")
}

// CreateMetadata creates maven-metadata.xml
func (h *MavenRepositoryHandler) CreateMetadata(coords *MavenCoordinates, versions []string) *MavenMetadata {
	latest := h.findLatestVersion(versions)
	release := h.findLatestRelease(versions)

	metadata := &MavenMetadata{
		GroupID:    coords.GroupID,
		ArtifactID: coords.ArtifactID,
		Versioning: &MavenVersioning{
			Latest:      latest,
			Release:     release,
			Versions:    &MavenVersions{Version: versions},
			LastUpdated: h.getCurrentTimestamp(),
		},
	}

	return metadata
}

// UpdateMetadata updates existing metadata with new version
func (h *MavenRepositoryHandler) UpdateMetadata(metadata *MavenMetadata, newVersion string) *MavenMetadata {
	// Add new version if not already present
	versionExists := false
	for _, version := range metadata.Versioning.Versions.Version {
		if version == newVersion {
			versionExists = true
			break
		}
	}

	if !versionExists {
		metadata.Versioning.Versions.Version = append(metadata.Versioning.Versions.Version, newVersion)
	}

	// Update latest and release
	metadata.Versioning.Latest = h.findLatestVersion(metadata.Versioning.Versions.Version)
	metadata.Versioning.Release = h.findLatestRelease(metadata.Versioning.Versions.Version)
	metadata.Versioning.LastUpdated = h.getCurrentTimestamp()

	return metadata
}

// CreateSnapshotMetadata creates snapshot-specific metadata
func (h *MavenRepositoryHandler) CreateSnapshotMetadata(coords *MavenCoordinates, timestamp string, buildNumber int) *MavenMetadata {
	metadata := &MavenMetadata{
		GroupID:    coords.GroupID,
		ArtifactID: coords.ArtifactID,
		Version:    coords.Version,
		Versioning: &MavenVersioning{
			Snapshot: &MavenSnapshot{
				Timestamp:   timestamp,
				BuildNumber: buildNumber,
			},
			LastUpdated: h.getCurrentTimestamp(),
			SnapshotVersions: &MavenSnapshotVersions{
				SnapshotVersion: []MavenSnapshotVersion{
					{
						Extension:  coords.Packaging,
						Value:      fmt.Sprintf("%s-%s-%d", strings.TrimSuffix(coords.Version, "-SNAPSHOT"), timestamp, buildNumber),
						Updated:    h.getCurrentTimestamp(),
						Classifier: coords.Classifier,
					},
				},
			},
		},
	}

	return metadata
}

// CalculateChecksums calculates various checksums for a file
func (h *MavenRepositoryHandler) CalculateChecksums(data []byte) map[string]string {
	checksums := make(map[string]string)

	// MD5
	md5Hash := md5.Sum(data)
	checksums["md5"] = hex.EncodeToString(md5Hash[:])

	// SHA1
	sha1Hash := sha1.Sum(data)
	checksums["sha1"] = hex.EncodeToString(sha1Hash[:])

	// SHA256
	sha256Hash := sha256.Sum256(data)
	checksums["sha256"] = hex.EncodeToString(sha256Hash[:])

	// SHA512
	sha512Hash := sha512.Sum512(data)
	checksums["sha512"] = hex.EncodeToString(sha512Hash[:])

	return checksums
}

// ValidatePOM validates POM XML structure
func (h *MavenRepositoryHandler) ValidatePOM(data []byte) (*POMProject, error) {
	var pom POMProject
	if err := xml.Unmarshal(data, &pom); err != nil {
		return nil, fmt.Errorf("invalid POM XML: %v", err)
	}

	// Validate required fields
	if pom.GroupID == "" && (pom.Parent == nil || pom.Parent.GroupID == "") {
		return nil, fmt.Errorf("POM missing groupId")
	}
	if pom.ArtifactID == "" {
		return nil, fmt.Errorf("POM missing artifactId")
	}
	if pom.Version == "" && (pom.Parent == nil || pom.Parent.Version == "") {
		return nil, fmt.Errorf("POM missing version")
	}

	return &pom, nil
}

// ExtractPOMMetadata extracts metadata from POM
func (h *MavenRepositoryHandler) ExtractPOMMetadata(pom *POMProject) map[string]interface{} {
	metadata := map[string]interface{}{
		"group_id":    pom.GroupID,
		"artifact_id": pom.ArtifactID,
		"version":     pom.Version,
		"packaging":   pom.Packaging,
		"name":        pom.Name,
		"description": pom.Description,
		"url":         pom.URL,
	}

	if pom.Parent != nil {
		metadata["parent"] = map[string]interface{}{
			"group_id":    pom.Parent.GroupID,
			"artifact_id": pom.Parent.ArtifactID,
			"version":     pom.Parent.Version,
		}
	}

	if pom.Dependencies != nil && len(pom.Dependencies.Dependency) > 0 {
		deps := make([]map[string]interface{}, len(pom.Dependencies.Dependency))
		for i, dep := range pom.Dependencies.Dependency {
			deps[i] = map[string]interface{}{
				"group_id":    dep.GroupID,
				"artifact_id": dep.ArtifactID,
				"version":     dep.Version,
				"type":        dep.Type,
				"classifier":  dep.Classifier,
				"scope":       dep.Scope,
				"optional":    dep.Optional,
			}
		}
		metadata["dependencies"] = deps
	}

	if pom.Licenses != nil && len(pom.Licenses.License) > 0 {
		licenses := make([]map[string]interface{}, len(pom.Licenses.License))
		for i, license := range pom.Licenses.License {
			licenses[i] = map[string]interface{}{
				"name":         license.Name,
				"url":          license.URL,
				"distribution": license.Distribution,
			}
		}
		metadata["licenses"] = licenses
	}

	return metadata
}

// Helper methods

func (h *MavenRepositoryHandler) findLatestVersion(versions []string) string {
	if len(versions) == 0 {
		return ""
	}
	// In a real implementation, you'd use proper version comparison
	// For now, return the last version (assuming they're sorted)
	return versions[len(versions)-1]
}

func (h *MavenRepositoryHandler) findLatestRelease(versions []string) string {
	for i := len(versions) - 1; i >= 0; i-- {
		if !h.IsSnapshot(versions[i]) {
			return versions[i]
		}
	}
	return ""
}

func (h *MavenRepositoryHandler) getCurrentTimestamp() string {
	return time.Now().UTC().Format("20060102150405")
}
