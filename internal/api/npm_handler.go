package api

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"

	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/securestor/securestor/internal/tenant"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/securestor/securestor/internal/models"
	"github.com/securestor/securestor/internal/storage"
)

// NPMPublishRequest represents an npm publish request
type NPMPublishRequest struct {
	ID          string                   `json:"_id"`
	Rev         string                   `json:"_rev,omitempty"`
	Name        string                   `json:"name"`
	Description string                   `json:"description,omitempty"`
	DistTags    map[string]string        `json:"dist-tags"`
	Versions    map[string]interface{}   `json:"versions"`
	Readme      string                   `json:"readme,omitempty"`
	Maintainers []NPMMaintainer          `json:"maintainers,omitempty"`
	Time        map[string]string        `json:"time,omitempty"`
	Author      interface{}              `json:"author,omitempty"`
	Repository  interface{}              `json:"repository,omitempty"`
	Keywords    []string                 `json:"keywords,omitempty"`
	License     string                   `json:"license,omitempty"`
	Bugs        interface{}              `json:"bugs,omitempty"`
	Homepage    string                   `json:"homepage,omitempty"`
	Attachments map[string]NPMAttachment `json:"_attachments,omitempty"`
}

// NPMMaintainer represents an npm package maintainer
type NPMMaintainer struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// NPMAttachment represents an npm package attachment (tarball)
type NPMAttachment struct {
	ContentType string `json:"content_type"`
	Data        string `json:"data"` // base64 encoded
	Length      int64  `json:"length"`
}

// NPMPublishResponse represents the response for npm publish
type NPMPublishResponse struct {
	Ok     bool   `json:"ok"`
	ID     string `json:"id"`
	Rev    string `json:"rev"`
	Error  string `json:"error,omitempty"`
	Reason string `json:"reason,omitempty"`
}

// setupNPMRoutes sets up npm registry compatible routes
func (s *Server) setupNPMRoutes(r *mux.Router) {
	// npm registry routes
	npmRouter := r.PathPrefix("/npm").Subrouter()

	// Special routes must come first (before generic package routes)
	// Registry ping
	npmRouter.HandleFunc("/-/ping", s.handleNPMPing).Methods("GET", "HEAD")

	// Package search
	npmRouter.HandleFunc("/-/v1/search", s.handleNPMSearch).Methods("GET")

	// Tarball download
	npmRouter.HandleFunc("/{package}/-/{filename}", s.handleNPMTarballGet).Methods("GET")

	// Specific version metadata
	npmRouter.HandleFunc("/{package}/{version}", s.handleNPMVersionGet).Methods("GET")

	// Package publish and update
	npmRouter.HandleFunc("/{package}", s.handleNPMPublish).Methods("PUT")

	// Package metadata retrieval (this should be last among package routes)
	npmRouter.HandleFunc("/{package}", s.handleNPMPackageGet).Methods("GET")
}

// handleNPMPublish handles npm package publish requests
func (s *Server) handleNPMPublish(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	packageName := vars["package"]

	s.logger.Printf("NPM publish request for package: %s", packageName)

	// Get tenant ID from context (injected by tenant middleware)
	tenantID, err := tenant.GetTenantID(r.Context())
	if err != nil {
		s.sendNPMError(w, http.StatusUnauthorized, "tenant_required", "No tenant context found. Ensure tenant middleware is applied.")
		return
	}

	// Parse the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.sendNPMError(w, http.StatusBadRequest, "invalid_request", "Failed to read request body")
		return
	}

	var publishReq NPMPublishRequest
	if err := json.Unmarshal(body, &publishReq); err != nil {
		s.sendNPMError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON in request body")
		return
	}

	// Validate package name
	npmHandler := storage.NewNPMRegistryHandler(filepath.Join(s.config.StoragePath, "npm"))
	if err := npmHandler.ValidatePackageName(packageName); err != nil {
		s.sendNPMError(w, http.StatusBadRequest, "invalid_package_name", err.Error())
		return
	}

	// Process each version in the request
	for version, versionData := range publishReq.Versions {
		if err := s.processNPMVersion(packageName, version, versionData, publishReq.Attachments, npmHandler, tenantID); err != nil {
			s.logger.Printf("Failed to process npm package %s@%s: %v", packageName, version, err)
			s.sendNPMError(w, http.StatusInternalServerError, "processing_failed", err.Error())
			return
		}
	}

	// Send success response
	response := NPMPublishResponse{
		Ok:  true,
		ID:  packageName,
		Rev: "1-" + generateRandomString(32),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)

	s.logger.Printf("Successfully published npm package: %s", packageName)
}

// processNPMVersion processes a single version of an npm package
func (s *Server) processNPMVersion(packageName, version string, versionData interface{}, attachments map[string]NPMAttachment, npmHandler *storage.NPMRegistryHandler, tenantID uuid.UUID) error {
	// Parse version data
	versionBytes, err := json.Marshal(versionData)
	if err != nil {
		return fmt.Errorf("failed to marshal version data: %v", err)
	}

	var versionMeta storage.NPMVersionMetadata
	if err := json.Unmarshal(versionBytes, &versionMeta); err != nil {
		return fmt.Errorf("failed to parse version metadata: %v", err)
	}

	// Validate version
	if err := npmHandler.ValidateVersion(version); err != nil {
		return fmt.Errorf("invalid version: %v", err)
	}

	// Find the tarball in attachments
	tarballKey := fmt.Sprintf("%s-%s.tgz", getPackageNameFromScoped(packageName), version)
	attachment, exists := attachments[tarballKey]
	if !exists {
		return fmt.Errorf("tarball attachment not found: %s", tarballKey)
	}

	// Decode base64 tarball data
	tarballData, err := decodeBase64(attachment.Data)
	if err != nil {
		return fmt.Errorf("failed to decode tarball: %v", err)
	}

	// Validate tarball and extract package.json
	packageJSON, err := npmHandler.ExtractPackageJSON(strings.NewReader(string(tarballData)))
	if err != nil {
		return fmt.Errorf("failed to extract package.json: %v", err)
	}

	// Verify package name and version match
	if packageJSON.Name != packageName {
		return fmt.Errorf("package name mismatch: expected %s, got %s", packageName, packageJSON.Name)
	}
	if packageJSON.Version != version {
		return fmt.Errorf("version mismatch: expected %s, got %s", version, packageJSON.Version)
	}

	// Calculate checksum
	checksum := calculateSHA256(tarballData)

	// Ensure we have a repository for npm packages
	repositoryID, err := s.ensureNPMRepository()
	if err != nil {
		return fmt.Errorf("failed to ensure npm repository: %v", err)
	}

	// Create artifact record
	artifact := &models.Artifact{
		Name:         packageName,
		Version:      version,
		Type:         "npm",
		RepositoryID: repositoryID,
		Size:         int64(len(tarballData)),
		Checksum:     checksum,
		UploadedBy:   nil, // TODO: Get from authentication
		Metadata: map[string]interface{}{
			"package_type":     "npm",
			"tarball_name":     tarballKey,
			"description":      packageJSON.Description,
			"license":          packageJSON.License,
			"author":           packageJSON.Author,
			"keywords":         packageJSON.Keywords,
			"dependencies":     packageJSON.Dependencies,
			"dev_dependencies": packageJSON.DevDependencies,
			"engines":          packageJSON.Engines,
			"main":             packageJSON.Main,
			"scripts":          packageJSON.Scripts,
		},
	}

	// Store in blob storage
	storageID := generateRandomString(32)
	err = s.blobStorage.UploadArtifact(context.TODO(), tenantID.String(), repositoryID.String(), storageID, strings.NewReader(string(tarballData)), int64(len(tarballData)))
	if err != nil {
		return fmt.Errorf("failed to store tarball: %v", err)
	}

	artifact.Metadata["storage_id"] = storageID

	// Save artifact to database
	if err := s.artifactService.Create(artifact); err != nil {
		return fmt.Errorf("failed to create artifact record: %v", err)
	}

	// Store npm registry metadata
	tarballURL := fmt.Sprintf("/npm/%s/-/%s", packageName, tarballKey)
	if err := s.storeNPMMetadata(packageName, packageJSON, tarballURL, checksum, npmHandler); err != nil {
		s.logger.Printf("Warning: Failed to store npm metadata for %s@%s: %v", packageName, version, err)
	}

	// Initialize security scans
	go s.initializeArtifactScans(artifact.ID.String())

	s.logger.Printf("Stored npm package %s@%s (artifact ID: %s)", packageName, version, artifact.ID)
	return nil
}

// storeNPMMetadata stores npm registry compatible metadata
func (s *Server) storeNPMMetadata(packageName string, packageJSON *storage.NPMPackageJSON, tarballURL, checksum string, npmHandler *storage.NPMRegistryHandler) error {
	metadataPath := npmHandler.GetPackagePath(packageName)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(metadataPath), 0755); err != nil {
		return fmt.Errorf("failed to create metadata directory: %v", err)
	}

	var metadata *storage.NPMRegistryMetadata

	// Check if metadata already exists
	if _, err := os.Stat(metadataPath); err == nil {
		// Load existing metadata
		data, err := os.ReadFile(metadataPath)
		if err != nil {
			return fmt.Errorf("failed to read existing metadata: %v", err)
		}

		var existing storage.NPMRegistryMetadata
		if err := json.Unmarshal(data, &existing); err != nil {
			return fmt.Errorf("failed to parse existing metadata: %v", err)
		}

		metadata = npmHandler.UpdateRegistryMetadata(&existing, packageJSON, tarballURL, checksum)
	} else {
		// Create new metadata
		metadata = npmHandler.CreateRegistryMetadata(packageJSON, tarballURL, checksum)
	}

	// Save metadata
	metadataBytes, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %v", err)
	}

	if err := os.WriteFile(metadataPath, metadataBytes, 0644); err != nil {
		return fmt.Errorf("failed to write metadata file: %v", err)
	}

	return nil
}

// handleNPMPackageGet handles npm package metadata retrieval
func (s *Server) handleNPMPackageGet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	packageName := vars["package"]

	npmHandler := storage.NewNPMRegistryHandler(filepath.Join(s.config.StoragePath, "npm"))
	metadataPath := npmHandler.GetPackagePath(packageName)

	// Check if package exists locally
	if _, err := os.Stat(metadataPath); err == nil {
		// Read metadata from local storage
		data, err := os.ReadFile(metadataPath)
		if err != nil {
			s.sendNPMError(w, http.StatusInternalServerError, "read_error", "Failed to read package metadata")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(data)
		return
	}

	// Package not found locally, try to fetch from remote npm registries
	remoteURL := s.fetchFromRemoteNPMRegistry(packageName)
	if remoteURL == "" {
		s.sendNPMError(w, http.StatusNotFound, "not_found", "Package not found")
		return
	}

	// Proxy request to remote registry
	resp, err := http.Get(remoteURL)
	if err != nil {
		s.logger.Printf("Failed to fetch from remote npm registry: %v", err)
		s.sendNPMError(w, http.StatusBadGateway, "upstream_error", "Failed to fetch from upstream registry")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		s.sendNPMError(w, http.StatusNotFound, "not_found", "Package not found in upstream registry")
		return
	}

	// Read and forward the response
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		s.sendNPMError(w, http.StatusInternalServerError, "read_error", "Failed to read upstream response")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// handleNPMVersionGet handles npm specific version metadata retrieval
func (s *Server) handleNPMVersionGet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	packageName := vars["package"]
	version := vars["version"]

	npmHandler := storage.NewNPMRegistryHandler(filepath.Join(s.config.StoragePath, "npm"))
	metadataPath := npmHandler.GetPackagePath(packageName)

	// Read package metadata
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		s.sendNPMError(w, http.StatusNotFound, "not_found", "Package not found")
		return
	}

	var metadata storage.NPMRegistryMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		s.sendNPMError(w, http.StatusInternalServerError, "parse_error", "Failed to parse package metadata")
		return
	}

	// Find specific version
	versionMeta, exists := metadata.Versions[version]
	if !exists {
		s.sendNPMError(w, http.StatusNotFound, "version_not_found", "Version not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(versionMeta)
}

// handleNPMTarballGet handles npm tarball download
func (s *Server) handleNPMTarballGet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	packageName := vars["package"]
	filename := vars["filename"]

	// Extract version from filename
	version, err := extractVersionFromFilename(filename, packageName)
	if err != nil {
		s.sendNPMError(w, http.StatusBadRequest, "invalid_filename", "Invalid tarball filename")
		return
	}

	// Find artifact in database
	artifacts, err := s.artifactService.GetByNameVersionType(packageName, version, "npm")
	if err == nil && len(artifacts) > 0 {
		// Artifact found locally, serve it
		artifact := artifacts[0]
		storageID, ok := artifact.Metadata["storage_id"].(string)
		if !ok {
			s.sendNPMError(w, http.StatusInternalServerError, "storage_error", "Storage ID not found")
			return
		}

		// Download from blob storage
		data, err := s.blobStorage.DownloadArtifact(r.Context(), artifact.TenantID.String(), artifact.RepositoryID.String(), storageID)
		if err != nil {
			s.sendNPMError(w, http.StatusInternalServerError, "download_error", "Failed to download tarball")
			return
		}

		// Set appropriate headers
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))

		w.WriteHeader(http.StatusOK)
		w.Write(data)
		return
	}

	// Not found locally, try to proxy from remote npm registry
	remoteURL := s.fetchFromRemoteNPMRegistry(packageName)
	if remoteURL == "" {
		s.sendNPMError(w, http.StatusNotFound, "not_found", "Tarball not found")
		return
	}

	// Get package metadata to find tarball URL
	resp, err := http.Get(remoteURL)
	if err != nil {
		s.logger.Printf("Failed to fetch package metadata from remote: %v", err)
		s.sendNPMError(w, http.StatusBadGateway, "upstream_error", "Failed to fetch from upstream registry")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		s.sendNPMError(w, http.StatusNotFound, "not_found", "Package not found in upstream registry")
		return
	}

	// Parse metadata to get tarball URL
	var metadata map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		s.sendNPMError(w, http.StatusInternalServerError, "parse_error", "Failed to parse package metadata")
		return
	}

	// Extract tarball URL for this version
	versions, ok := metadata["versions"].(map[string]interface{})
	if !ok {
		s.sendNPMError(w, http.StatusNotFound, "not_found", "Version not found in package metadata")
		return
	}

	versionMeta, ok := versions[version].(map[string]interface{})
	if !ok {
		s.sendNPMError(w, http.StatusNotFound, "not_found", "Version not found in package metadata")
		return
	}

	dist, ok := versionMeta["dist"].(map[string]interface{})
	if !ok {
		s.sendNPMError(w, http.StatusNotFound, "not_found", "Distribution metadata not found")
		return
	}

	tarballURL, ok := dist["tarball"].(string)
	if !ok {
		s.sendNPMError(w, http.StatusNotFound, "not_found", "Tarball URL not found")
		return
	}

	// Proxy the tarball download
	tarballResp, err := http.Get(tarballURL)
	if err != nil {
		s.logger.Printf("Failed to download tarball from remote: %v", err)
		s.sendNPMError(w, http.StatusBadGateway, "upstream_error", "Failed to download tarball from upstream")
		return
	}
	defer tarballResp.Body.Close()

	if tarballResp.StatusCode != http.StatusOK {
		s.sendNPMError(w, http.StatusNotFound, "not_found", "Tarball not found in upstream registry")
		return
	}

	// Stream the tarball to the client
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	if tarballResp.ContentLength > 0 {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", tarballResp.ContentLength))
	}

	w.WriteHeader(http.StatusOK)
	io.Copy(w, tarballResp.Body)
}

// handleNPMPing handles npm registry ping requests
func (s *Server) handleNPMPing(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"ok": true,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleNPMSearch handles npm package search requests
func (s *Server) handleNPMSearch(w http.ResponseWriter, r *http.Request) {
	_ = r.URL.Query().Get("text") // query text
	_ = r.URL.Query().Get("size") // result size
	_ = r.URL.Query().Get("from") // pagination offset

	// For now, return empty results
	// TODO: Implement search functionality
	response := map[string]interface{}{
		"objects": []interface{}{},
		"total":   0,
		"time":    "0ms",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// sendNPMError sends npm-compatible error response
func (s *Server) sendNPMError(w http.ResponseWriter, statusCode int, error, reason string) {
	response := NPMPublishResponse{
		Ok:     false,
		Error:  error,
		Reason: reason,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// Helper functions

func (s *Server) ensureNPMRepository() (uuid.UUID, error) {
	// Check if npm repository already exists
	repositories, err := s.repositoryService.List()
	if err != nil {
		return uuid.UUID{}, err
	}

	// Look for existing npm repository
	for _, repo := range repositories {
		if repo.Type == "npm" {
			return repo.ID, nil
		}
	}

	// Create default npm repository
	createReq := &models.CreateRepositoryRequest{
		Name:           "npm-registry",
		Type:           "npm",
		RepositoryType: "local",
		Description:    "Default NPM registry repository",
		PublicAccess:   false,
		EnableIndexing: true,
	}

	repoResponse, err := s.repositoryService.Create(createReq)
	if err != nil {
		return uuid.UUID{}, err
	}

	return repoResponse.ID, nil
}

func generateRandomString(length int) string {
	bytes := make([]byte, length/2)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func decodeBase64(data string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(data)
}

// fetchFromRemoteNPMRegistry finds a remote npm repository and returns the upstream URL for the package
func (s *Server) fetchFromRemoteNPMRegistry(packageName string) string {
	// Get all npm repositories with remote URLs
	repos, err := s.repositoryService.List()
	if err != nil {
		s.logger.Printf("Failed to list repositories: %v", err)
		return ""
	}

	// Find an npm repository with a remote_url
	for _, repo := range repos {
		if repo.Type == "npm" && repo.RemoteURL != "" {
			// Construct the upstream URL
			// npm registry URL format: https://registry.npmjs.org/packagename
			upstreamURL := strings.TrimSuffix(repo.RemoteURL, "/") + "/" + packageName
			s.logger.Printf("Proxying npm package %s to upstream: %s", packageName, upstreamURL)
			return upstreamURL
		}
	}

	return ""
}

func calculateSHA256(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func getPackageNameFromScoped(name string) string {
	if strings.HasPrefix(name, "@") {
		parts := strings.SplitN(name, "/", 2)
		if len(parts) == 2 {
			return parts[1]
		}
	}
	return name
}

func extractVersionFromFilename(filename, packageName string) (string, error) {
	baseName := getPackageNameFromScoped(packageName)
	prefix := baseName + "-"
	suffix := ".tgz"

	if !strings.HasPrefix(filename, prefix) || !strings.HasSuffix(filename, suffix) {
		return "", fmt.Errorf("invalid filename format")
	}

	version := strings.TrimPrefix(filename, prefix)
	version = strings.TrimSuffix(version, suffix)

	return version, nil
}
