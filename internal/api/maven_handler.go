package api

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/securestor/securestor/internal/handlers"
	"github.com/securestor/securestor/internal/models"
	"github.com/securestor/securestor/internal/replicate"
	"github.com/securestor/securestor/internal/repository"
	"github.com/securestor/securestor/internal/service"
	"github.com/securestor/securestor/internal/storage"
)

// MavenHandler handles Maven repository operations with enterprise replication
type MavenHandler struct {
	mavenHandler      *storage.MavenRepositoryHandler
	storagePath       string
	artifactService   *service.ArtifactService
	repositoryService *service.RepositoryService
	artifactRepo      *repository.ArtifactRepository
	replicationMixin  *handlers.ReplicationMixin // Enterprise replication
}

// NewMavenHandler creates a new Maven handler with replication support
func NewMavenHandler(storagePath string, artifactService *service.ArtifactService, repositoryService *service.RepositoryService, artifactRepo *repository.ArtifactRepository, replicationMixin *handlers.ReplicationMixin) *MavenHandler {
	return &MavenHandler{
		mavenHandler:      storage.NewMavenRepositoryHandler(storagePath),
		storagePath:       storagePath,
		artifactService:   artifactService,
		repositoryService: repositoryService,
		artifactRepo:      artifactRepo,
		replicationMixin:  replicationMixin,
	}
}

// DeployArtifact handles Maven artifact deployment
// PUT /maven2/<groupId>/<artifactId>/<version>/<filename>
func (h *MavenHandler) DeployArtifact(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract path from URL
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/maven2/")

	// Read file content
	content, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read request body: %v", err), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Determine file type and handle accordingly
	fullPath := filepath.Join(h.storagePath, "maven", path)

	// Create directory structure
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create directories: %v", err), http.StatusInternalServerError)
		return
	}

	// Write file
	if err := os.WriteFile(fullPath, content, 0644); err != nil {
		http.Error(w, fmt.Sprintf("Failed to write file: %v", err), http.StatusInternalServerError)
		return
	}

	// Calculate checksums
	sha1Hash := sha1.Sum(content)
	sha1Checksum := fmt.Sprintf("%x", sha1Hash)
	md5Hash := md5.Sum(content)
	md5Checksum := fmt.Sprintf("%x", md5Hash)

	// Generate checksums for main artifacts (not for checksum files themselves)
	if !strings.HasSuffix(path, ".md5") && !strings.HasSuffix(path, ".sha1") &&
		!strings.HasSuffix(path, ".sha256") && !strings.HasSuffix(path, ".sha512") {
		checksums := h.mavenHandler.CalculateChecksums(content)

		// Write checksum files
		for algo, checksum := range checksums {
			checksumPath := h.mavenHandler.GetChecksumPath(fullPath, algo)
			if err := os.WriteFile(checksumPath, []byte(checksum), 0644); err != nil {
				// Log error but don't fail the request
				fmt.Printf("Warning: Failed to write %s checksum: %v\n", algo, err)
			}
		}
	}

	// Handle JAR files - save to database
	if strings.HasSuffix(path, ".jar") {
		// Parse Maven coordinates from path
		// Path format: groupId/artifactId/version/artifactId-version.jar
		pathParts := strings.Split(path, "/")
		if len(pathParts) >= 4 {
			version := pathParts[len(pathParts)-2]
			artifactId := pathParts[len(pathParts)-3]
			groupId := strings.Join(pathParts[:len(pathParts)-3], ".")

			// Get or create Maven repository
			mavenRepo, err := h.getOrCreateMavenRepository()
			if err != nil {
				fmt.Printf("Warning: Failed to get/create Maven repository: %v\n", err)
			} else {
				// Create artifact record in database
				artifact := &models.Artifact{
					Name:         fmt.Sprintf("%s:%s", groupId, artifactId),
					Version:      version,
					Type:         "maven",
					RepositoryID: mavenRepo.ID,
					Size:         int64(len(content)),
					Checksum:     sha1Checksum,
					UploadedBy:   nil,
					Metadata: map[string]interface{}{
						"groupId":    groupId,
						"artifactId": artifactId,
						"path":       path,
						"md5":        md5Checksum,
						"storage_id": fmt.Sprintf("%s-%s-%s", groupId, artifactId, version),
						"file_path":  fullPath,
					},
				}

				if err := h.artifactRepo.Create(artifact); err != nil {
					fmt.Printf("Warning: Failed to save artifact to database: %v\n", err)
				} else {
					fmt.Printf("Maven artifact saved to database: ID=%d, %s:%s:%s\n", artifact.ID, groupId, artifactId, version)

					// Trigger replication asynchronously
					go h.replicateMavenArtifact(mavenRepo.TenantID.String(), mavenRepo.ID.String(),
						fmt.Sprintf("%s-%s-%s", groupId, artifactId, version), content)
				}
			}
		}
	}

	// Handle POM files - validate and extract metadata
	if strings.HasSuffix(path, ".pom") {
		pom, err := h.mavenHandler.ValidatePOM(content)
		if err != nil {
			// Log validation error but don't fail (some POMs might be slightly malformed)
			fmt.Printf("Warning: POM validation failed: %v\n", err)
		} else {
			// Update metadata
			h.updateMavenMetadata(pom)
		}
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Artifact deployed successfully",
		"path":    path,
	})
}

// GetArtifact retrieves a Maven artifact
// GET /maven2/<groupId>/<artifactId>/<version>/<filename>
func (h *MavenHandler) GetArtifact(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract path from URL
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/maven2/")
	fullPath := filepath.Join(h.storagePath, "maven", path)

	// Check if file exists
	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "Artifact not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to access artifact: %v", err), http.StatusInternalServerError)
		return
	}

	// Set content type based on file extension
	contentType := h.getContentType(path)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.FormatInt(fileInfo.Size(), 10))

	// For HEAD requests, just return headers
	if r.Method == http.MethodHead {
		return
	}

	// Serve the file
	http.ServeFile(w, r, fullPath)
}

// ListVersions lists all versions of an artifact
// GET /api/v1/maven/<groupId>/<artifactId>/versions
func (h *MavenHandler) ListVersions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse URL parameters
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/maven/"), "/")
	if len(pathParts) < 3 {
		http.Error(w, "Invalid path format", http.StatusBadRequest)
		return
	}

	groupID := pathParts[0]
	artifactID := pathParts[1]

	// Read maven-metadata.xml
	metadataPath := h.mavenHandler.GetMetadataPath(groupID, artifactID)
	fullMetadataPath := filepath.Join(h.storagePath, "maven", metadataPath)

	data, err := os.ReadFile(fullMetadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "Artifact not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to read metadata: %v", err), http.StatusInternalServerError)
		return
	}

	var metadata storage.MavenMetadata
	if err := xml.Unmarshal(data, &metadata); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse metadata: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"group_id":     metadata.GroupID,
		"artifact_id":  metadata.ArtifactID,
		"latest":       metadata.Versioning.Latest,
		"release":      metadata.Versioning.Release,
		"versions":     metadata.Versioning.Versions.Version,
		"last_updated": metadata.Versioning.LastUpdated,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// SearchArtifacts searches for Maven artifacts
// GET /api/v1/maven/search?q=<query>
func (h *MavenHandler) SearchArtifacts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	// Search through Maven repository structure
	results := []map[string]interface{}{}
	mavenBasePath := filepath.Join(h.storagePath, "maven")

	err := filepath.Walk(mavenBasePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		if info.Name() == "maven-metadata.xml" {
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}

			var metadata storage.MavenMetadata
			if err := xml.Unmarshal(data, &metadata); err != nil {
				return nil
			}

			// Simple search matching
			if strings.Contains(strings.ToLower(metadata.GroupID), strings.ToLower(query)) ||
				strings.Contains(strings.ToLower(metadata.ArtifactID), strings.ToLower(query)) {
				results = append(results, map[string]interface{}{
					"group_id":    metadata.GroupID,
					"artifact_id": metadata.ArtifactID,
					"latest":      metadata.Versioning.Latest,
					"versions":    len(metadata.Versioning.Versions.Version),
				})
			}
		}

		return nil
	})

	if err != nil {
		http.Error(w, fmt.Sprintf("Search failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"query":   query,
		"results": results,
		"total":   len(results),
	})
}

// DeleteArtifact deletes a Maven artifact version
// DELETE /api/v1/maven/<groupId>/<artifactId>/<version>
func (h *MavenHandler) DeleteArtifact(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse coordinates from path
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/maven/"), "/")
	if len(pathParts) < 3 {
		http.Error(w, "Invalid path format", http.StatusBadRequest)
		return
	}

	groupID := pathParts[0]
	artifactID := pathParts[1]
	version := pathParts[2]

	// Delete version directory
	groupPath := strings.ReplaceAll(groupID, ".", "/")
	versionPath := filepath.Join(h.storagePath, "maven", groupPath, artifactID, version)

	if err := os.RemoveAll(versionPath); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete artifact: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":     "Artifact deleted successfully",
		"group_id":    groupID,
		"artifact_id": artifactID,
		"version":     version,
	})
}

// Helper methods

func (h *MavenHandler) updateMavenMetadata(pom *storage.POMProject) error {
	coords := &storage.MavenCoordinates{
		GroupID:    pom.GroupID,
		ArtifactID: pom.ArtifactID,
		Version:    pom.Version,
	}

	metadataPath := h.mavenHandler.GetMetadataPath(coords.GroupID, coords.ArtifactID)
	fullMetadataPath := filepath.Join(h.storagePath, "maven", metadataPath)

	var metadata *storage.MavenMetadata

	// Try to read existing metadata
	if data, err := os.ReadFile(fullMetadataPath); err == nil {
		var existing storage.MavenMetadata
		if err := xml.Unmarshal(data, &existing); err == nil {
			metadata = h.mavenHandler.UpdateMetadata(&existing, coords.Version)
		}
	}

	// Create new metadata if doesn't exist
	if metadata == nil {
		metadata = h.mavenHandler.CreateMetadata(coords, []string{coords.Version})
	}

	// Marshal and write metadata
	output, err := xml.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}

	// Add XML header
	xmlData := []byte(xml.Header + string(output))

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(fullMetadataPath), 0755); err != nil {
		return err
	}

	return os.WriteFile(fullMetadataPath, xmlData, 0644)
}

// getOrCreateMavenRepository gets or creates the Maven repository in the database
func (h *MavenHandler) getOrCreateMavenRepository() (*models.Repository, error) {
	// Try to find existing Maven repository
	repos, err := h.repositoryService.List()
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}

	// Look for Maven repository
	for _, repo := range repos {
		if repo.Type == "maven" {
			return &repo, nil
		}
	}

	// Create Maven repository if it doesn't exist
	newRepoReq := &models.CreateRepositoryRequest{
		Name:           "Maven Repository",
		Type:           "maven",
		RepositoryType: "local",
		Description:    "Default Maven repository for JAR artifacts",
		PublicAccess:   false,
		EnableIndexing: true,
	}

	repoResp, err := h.repositoryService.Create(newRepoReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create Maven repository: %w", err)
	}

	// Convert response to Repository model
	repo := &models.Repository{
		ID:             repoResp.ID,
		Name:           repoResp.Name,
		Type:           repoResp.Type,
		RepositoryType: repoResp.RepositoryType,
		Description:    repoResp.Description,
	}

	return repo, nil
}

func (h *MavenHandler) getContentType(filename string) string {
	ext := filepath.Ext(filename)
	switch ext {
	case ".pom", ".xml":
		return "application/xml"
	case ".jar", ".war", ".ear":
		return "application/java-archive"
	case ".md5":
		return "text/plain"
	case ".sha1", ".sha256", ".sha512":
		return "text/plain"
	case ".json":
		return "application/json"
	default:
		return "application/octet-stream"
	}
}

// GetArtifactInfo returns detailed information about a Maven artifact
func (h *MavenHandler) GetArtifactInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse coordinates from path
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/maven/"), "/")
	if len(pathParts) < 4 || pathParts[3] != "info" {
		http.Error(w, "Invalid path format", http.StatusBadRequest)
		return
	}

	groupID := pathParts[0]
	artifactID := pathParts[1]
	version := pathParts[2]

	// Read POM file
	coords := &storage.MavenCoordinates{
		GroupID:    groupID,
		ArtifactID: artifactID,
		Version:    version,
	}

	pomPath := h.mavenHandler.GetPOMPath(coords)
	fullPomPath := filepath.Join(h.storagePath, "maven", pomPath)

	data, err := os.ReadFile(fullPomPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "Artifact not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to read POM: %v", err), http.StatusInternalServerError)
		return
	}

	pom, err := h.mavenHandler.ValidatePOM(data)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse POM: %v", err), http.StatusInternalServerError)
		return
	}

	// Extract metadata
	metadata := h.mavenHandler.ExtractPOMMetadata(pom)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metadata)
}

// replicateMavenArtifact triggers replication for a Maven artifact using the replication mixin
func (h *MavenHandler) replicateMavenArtifact(tenantID, repositoryID, artifactID string, data []byte) {
	if h.replicationMixin == nil || !h.replicationMixin.IsEnabled() {
		return // Replication not configured
	}

	// Trigger async replication using the mixin
	h.replicationMixin.ReplicateAsync(&replicate.ReplicationRequest{
		TenantID:     tenantID,
		RepositoryID: repositoryID,
		ArtifactID:   artifactID,
		ArtifactType: "maven",
		Data:         data,
		Metadata: map[string]string{
			"artifact_type": "jar",
		},
		BucketName: fmt.Sprintf("%s/maven", repositoryID),
		FileName:   fmt.Sprintf("%s.jar", artifactID),
	})
}
