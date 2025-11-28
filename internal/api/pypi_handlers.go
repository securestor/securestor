package api

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gorilla/mux"
	"github.com/securestor/securestor/internal/models"
)

// handlePyPISimpleIndex handles PEP 503 Simple Repository API index
// GET /simple/
func (s *Server) handlePyPISimpleIndex(w http.ResponseWriter, r *http.Request) {
	s.logger.Printf("PyPI Simple Index request")

	// Get all repositories
	repos, err := s.repositoryService.List()
	if err != nil {
		s.logger.Printf("Error getting repositories: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Filter for PyPI repositories and collect package names
	packageNames := make(map[string]bool)
	for _, repo := range repos {
		if repo.Type != "pypi" {
			continue
		}

		// Get artifacts for this repository using the existing List method with filter
		filter := &models.ArtifactFilter{
			RepositoryID: &repo.ID,
			Types:        []string{"pypi"},
			Limit:        1000, // Large limit to get all packages
		}
		artifacts, _, err := s.artifactService.List(filter)
		if err != nil {
			continue
		}

		for _, artifact := range artifacts {
			// Normalize package name per PEP 503
			normalizedName := normalizePyPIName(artifact.Name)
			packageNames[normalizedName] = true
		}
	}

	// Generate HTML response per PEP 503
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
    <meta name="pypi:repository-version" content="1.0">
    <title>Simple index</title>
</head>
<body>
<h1>Simple index</h1>
`)

	// Sort package names for consistent output
	names := make([]string, 0, len(packageNames))
	for name := range packageNames {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		fmt.Fprintf(w, `<a href="%s/">%s</a><br/>
`, html.EscapeString(name), html.EscapeString(name))
	}

	fmt.Fprintf(w, `</body>
</html>`)
}

// handlePyPISimplePackage handles PEP 503 Simple Repository API package files listing
// GET /simple/{package}/
func (s *Server) handlePyPISimplePackage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	packageName := vars["package"]

	s.logger.Printf("PyPI Simple Package request for: %s", packageName)

	// Normalize package name per PEP 503
	normalizedName := normalizePyPIName(packageName)

	// Get all repositories
	repos, err := s.repositoryService.List()
	if err != nil {
		s.logger.Printf("Error getting repositories: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Collect all versions of the package from PyPI repositories
	var packageFiles []PyPIPackageFile
	for _, repo := range repos {
		if repo.Type != "pypi" {
			continue
		}

		// Get artifacts for this repository using search
		filter := &models.ArtifactFilter{
			RepositoryID: &repo.ID,
			Types:        []string{"pypi"},
			Search:       packageName, // This should help find the package
			Limit:        100,
		}
		artifacts, _, err := s.artifactService.List(filter)
		if err != nil {
			continue
		}

		for _, artifact := range artifacts {
			if normalizePyPIName(artifact.Name) == normalizedName {
				// Extract file information from artifact
				filename := extractPyPIFilename(&artifact)
				if filename == "" {
					filename = fmt.Sprintf("%s-%s.tar.gz", normalizedName, artifact.Version)
				}

				packageFiles = append(packageFiles, PyPIPackageFile{
					Filename:     filename,
					URL:          fmt.Sprintf("/api/v1/artifacts/%d/download", artifact.ID),
					Checksum:     artifact.Checksum,
					RequiresDist: extractRequiresDist(artifact.Metadata),
				})
			}
		}
	}

	if len(packageFiles) == 0 {
		http.NotFound(w, r)
		return
	}

	// Generate HTML response per PEP 503
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
    <meta name="pypi:repository-version" content="1.0">
    <title>Links for %s</title>
</head>
<body>
<h1>Links for %s</h1>
`, html.EscapeString(packageName), html.EscapeString(packageName))

	for _, file := range packageFiles {
		fmt.Fprintf(w, `<a href="%s"`, html.EscapeString(file.URL))

		// Add data-dist-info-metadata attribute if available
		if file.RequiresDist != "" {
			fmt.Fprintf(w, ` data-dist-info-metadata="%s"`, html.EscapeString(file.RequiresDist))
		}

		fmt.Fprintf(w, `>%s</a><br/>
`, html.EscapeString(file.Filename))
	}

	fmt.Fprintf(w, `</body>
</html>`)
}

// handlePyPIPackageDownload handles PyPI package file downloads
// GET /packages/{path:.*}
func (s *Server) handlePyPIPackageDownload(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	path := vars["path"]

	s.logger.Printf("PyPI Package download request for path: %s", path)

	// Extract filename from path
	filename := filepath.Base(path)

	// Find the artifact by searching for matching filename across PyPI repositories
	repos, err := s.repositoryService.List()
	if err != nil {
		s.logger.Printf("Error getting repositories: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var targetArtifact *models.Artifact
	for _, repo := range repos {
		if repo.Type != "pypi" {
			continue
		}

		// Search for artifacts by filename in this repository
		filter := &models.ArtifactFilter{
			RepositoryID: &repo.ID,
			Types:        []string{"pypi"},
			Search:       strings.TrimSuffix(filename, filepath.Ext(filename)), // Search by base name
			Limit:        50,
		}
		artifacts, _, err := s.artifactService.List(filter)
		if err != nil {
			continue
		}

		for _, artifact := range artifacts {
			artifactFilename := extractPyPIFilename(&artifact)
			if artifactFilename == filename {
				targetArtifact = &artifact
				break
			}
		}
		if targetArtifact != nil {
			break
		}
	}

	if targetArtifact == nil {
		http.NotFound(w, r)
		return
	}

	// Redirect to the unified artifact download endpoint
	downloadURL := fmt.Sprintf("/api/v1/artifacts/%d/download", targetArtifact.ID)
	http.Redirect(w, r, downloadURL, http.StatusTemporaryRedirect)
}

// PyPIPackageFile represents a PyPI package file for PEP 503 API
type PyPIPackageFile struct {
	Filename     string
	URL          string
	Checksum     string
	RequiresDist string
}

// normalizePyPIName normalizes package names per PEP 503
func normalizePyPIName(name string) string {
	// Per PEP 503: "All comparisons of distribution names MUST be case insensitive,
	// and MUST consider hyphens and underscores to be equivalent."
	normalized := strings.ToLower(name)
	normalized = strings.ReplaceAll(normalized, "_", "-")
	return normalized
}

// extractPyPIFilename extracts the filename for a PyPI artifact
func extractPyPIFilename(artifact *models.Artifact) string {
	if artifact.Metadata != nil {
		if filename, ok := artifact.Metadata["filename"].(string); ok {
			return filename
		}
		if originalName, ok := artifact.Metadata["original_filename"].(string); ok {
			return originalName
		}
	}

	// Generate default filename based on type
	if strings.Contains(artifact.Name, ".whl") || strings.Contains(artifact.Name, ".tar.gz") {
		return artifact.Name
	}

	// Default to wheel format
	normalizedName := normalizePyPIName(artifact.Name)
	return fmt.Sprintf("%s-%s-py3-none-any.whl", normalizedName, artifact.Version)
}

// extractRequiresDist extracts requires_dist information from artifact metadata
func extractRequiresDist(metadata map[string]interface{}) string {
	if metadata == nil {
		return ""
	}

	if requiresDist, ok := metadata["requires_dist"]; ok {
		if requiresDistSlice, ok := requiresDist.([]interface{}); ok {
			var deps []string
			for _, dep := range requiresDistSlice {
				if depStr, ok := dep.(string); ok {
					deps = append(deps, depStr)
				}
			}
			if len(deps) > 0 {
				jsonBytes, _ := json.Marshal(deps)
				return string(jsonBytes)
			}
		}
	}

	return ""
}
