package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/securestor/securestor/internal/storage"
)

// DockerRegistryHandler handles Docker Registry v2 API endpoints
type DockerRegistryHandler struct {
	registryAPI *storage.RegistryAPI
}

func NewDockerRegistryHandler(basePath string) *DockerRegistryHandler {
	return &DockerRegistryHandler{
		registryAPI: storage.NewRegistryAPI(basePath),
	}
}

// SetupDockerRoutes sets up Docker Registry v2 API routes
func (s *Server) SetupDockerRoutes(r *mux.Router) {
	handler := NewDockerRegistryHandler(s.config.StoragePath + "/docker")

	// Docker Registry v2 API routes
	v2 := r.PathPrefix("/v2").Subrouter()

	// Registry version check
	v2.HandleFunc("/", handler.handleRegistryVersion).Methods("GET")

	// Manifest operations
	v2.HandleFunc("/{name:.*}/manifests/{reference}", handler.handleGetManifest).Methods("GET")
	v2.HandleFunc("/{name:.*}/manifests/{reference}", handler.handlePutManifest).Methods("PUT")
	v2.HandleFunc("/{name:.*}/manifests/{reference}", handler.handleDeleteManifest).Methods("DELETE")

	// Blob operations
	v2.HandleFunc("/{name:.*}/blobs/{digest}", handler.handleGetBlob).Methods("GET")
	v2.HandleFunc("/{name:.*}/blobs/{digest}", handler.handleHeadBlob).Methods("HEAD")
	v2.HandleFunc("/{name:.*}/blobs/{digest}", handler.handleDeleteBlob).Methods("DELETE")

	// Blob upload operations
	v2.HandleFunc("/{name:.*}/blobs/uploads/", handler.handleInitiateBlobUpload).Methods("POST")
	v2.HandleFunc("/{name:.*}/blobs/uploads/{uuid}", handler.handleBlobUpload).Methods("PATCH")
	v2.HandleFunc("/{name:.*}/blobs/uploads/{uuid}", handler.handleCompleteBlobUpload).Methods("PUT")
	v2.HandleFunc("/{name:.*}/blobs/uploads/{uuid}", handler.handleGetBlobUploadStatus).Methods("GET")
	v2.HandleFunc("/{name:.*}/blobs/uploads/{uuid}", handler.handleCancelBlobUpload).Methods("DELETE")

	// Tag operations
	v2.HandleFunc("/{name:.*}/tags/list", handler.handleListTags).Methods("GET")

	// Catalog
	v2.HandleFunc("/_catalog", handler.handleCatalog).Methods("GET")
}

// handleRegistryVersion returns registry version information
func (h *DockerRegistryHandler) handleRegistryVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Docker-Distribution-API-Version", "registry/2.0")

	response := map[string]string{
		"what":    "SecureStore Docker Registry",
		"version": "2.0",
	}

	json.NewEncoder(w).Encode(response)
}

// handleGetManifest retrieves a manifest by reference
func (h *DockerRegistryHandler) handleGetManifest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	reference := vars["reference"]

	data, mediaType, err := h.registryAPI.GetManifest(name, reference)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.respondWithError(w, http.StatusNotFound, "manifest unknown", err.Error())
		} else {
			h.respondWithError(w, http.StatusInternalServerError, "internal server error", err.Error())
		}
		return
	}

	// Set appropriate headers
	w.Header().Set("Content-Type", mediaType)
	w.Header().Set("Docker-Distribution-API-Version", "registry/2.0")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))

	// Calculate and set digest header
	if !strings.Contains(reference, ":") {
		// This is a tag, calculate digest
		digest := h.calculateDigest(data)
		w.Header().Set("Docker-Content-Digest", digest)
	}

	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// handlePutManifest stores a manifest
func (h *DockerRegistryHandler) handlePutManifest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	reference := vars["reference"]

	// Read manifest data
	manifest, err := io.ReadAll(r.Body)
	if err != nil {
		h.respondWithError(w, http.StatusBadRequest, "invalid manifest", err.Error())
		return
	}

	// Store manifest
	if err := h.registryAPI.PutManifest(name, reference, manifest); err != nil {
		if strings.Contains(err.Error(), "validation failed") {
			h.respondWithError(w, http.StatusBadRequest, "manifest invalid", err.Error())
		} else {
			h.respondWithError(w, http.StatusInternalServerError, "internal server error", err.Error())
		}
		return
	}

	// Calculate digest and set location header
	digest := h.calculateDigest(manifest)
	location := fmt.Sprintf("/v2/%s/manifests/%s", name, digest)

	w.Header().Set("Location", location)
	w.Header().Set("Docker-Content-Digest", digest)
	w.Header().Set("Docker-Distribution-API-Version", "registry/2.0")
	w.WriteHeader(http.StatusCreated)
}

// handleDeleteManifest deletes a manifest
func (h *DockerRegistryHandler) handleDeleteManifest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	reference := vars["reference"]

	if err := h.registryAPI.DeleteManifest(name, reference); err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.respondWithError(w, http.StatusNotFound, "manifest unknown", err.Error())
		} else {
			h.respondWithError(w, http.StatusInternalServerError, "internal server error", err.Error())
		}
		return
	}

	w.Header().Set("Docker-Distribution-API-Version", "registry/2.0")
	w.WriteHeader(http.StatusAccepted)
}

// handleGetBlob retrieves a blob
func (h *DockerRegistryHandler) handleGetBlob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	digest := vars["digest"]

	blob, err := h.registryAPI.GetBlob(name, digest)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.respondWithError(w, http.StatusNotFound, "blob unknown", err.Error())
		} else {
			h.respondWithError(w, http.StatusInternalServerError, "internal server error", err.Error())
		}
		return
	}
	defer blob.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Docker-Distribution-API-Version", "registry/2.0")
	w.Header().Set("Docker-Content-Digest", digest)

	w.WriteHeader(http.StatusOK)
	io.Copy(w, blob)
}

// handleHeadBlob checks if a blob exists
func (h *DockerRegistryHandler) handleHeadBlob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	digest := vars["digest"]

	exists, size, err := h.registryAPI.CheckBlobExists(name, digest)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "internal server error", err.Error())
		return
	}

	if !exists {
		h.respondWithError(w, http.StatusNotFound, "blob unknown", "blob not found")
		return
	}

	w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	w.Header().Set("Docker-Distribution-API-Version", "registry/2.0")
	w.Header().Set("Docker-Content-Digest", digest)
	w.WriteHeader(http.StatusOK)
}

// handleInitiateBlobUpload initiates a blob upload
func (h *DockerRegistryHandler) handleInitiateBlobUpload(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	// Generate upload UUID
	uploadUUID := h.generateUploadUUID()

	// Set location header for upload continuation
	location := fmt.Sprintf("/v2/%s/blobs/uploads/%s", name, uploadUUID)

	w.Header().Set("Location", location)
	w.Header().Set("Range", "0-0")
	w.Header().Set("Docker-Upload-UUID", uploadUUID)
	w.Header().Set("Docker-Distribution-API-Version", "registry/2.0")
	w.WriteHeader(http.StatusAccepted)
}

// handleBlobUpload handles chunked blob upload
func (h *DockerRegistryHandler) handleBlobUpload(w http.ResponseWriter, r *http.Request) {
	// For simplicity, we'll implement this as a simple upload
	// In a production system, you'd handle chunked uploads
	vars := mux.Vars(r)
	name := vars["name"]
	uuid := vars["uuid"]

	location := fmt.Sprintf("/v2/%s/blobs/uploads/%s", name, uuid)

	w.Header().Set("Location", location)
	w.Header().Set("Range", "0-0")
	w.Header().Set("Docker-Upload-UUID", uuid)
	w.Header().Set("Docker-Distribution-API-Version", "registry/2.0")
	w.WriteHeader(http.StatusAccepted)
}

// handleCompleteBlobUpload completes a blob upload
func (h *DockerRegistryHandler) handleCompleteBlobUpload(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	digest := r.URL.Query().Get("digest")

	if digest == "" {
		h.respondWithError(w, http.StatusBadRequest, "digest invalid", "digest parameter required")
		return
	}

	// Store the blob
	if err := h.registryAPI.PutBlob(name, digest, r.Body); err != nil {
		if strings.Contains(err.Error(), "digest mismatch") {
			h.respondWithError(w, http.StatusBadRequest, "digest invalid", err.Error())
		} else {
			h.respondWithError(w, http.StatusInternalServerError, "internal server error", err.Error())
		}
		return
	}

	location := fmt.Sprintf("/v2/%s/blobs/%s", name, digest)

	w.Header().Set("Location", location)
	w.Header().Set("Docker-Content-Digest", digest)
	w.Header().Set("Docker-Distribution-API-Version", "registry/2.0")
	w.WriteHeader(http.StatusCreated)
}

// handleListTags lists all tags for a repository
func (h *DockerRegistryHandler) handleListTags(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	tags, err := h.registryAPI.ListTags(name)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "internal server error", err.Error())
		return
	}

	response := map[string]interface{}{
		"name": name,
		"tags": tags,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Docker-Distribution-API-Version", "registry/2.0")
	json.NewEncoder(w).Encode(response)
}

// handleCatalog lists all repositories
func (h *DockerRegistryHandler) handleCatalog(w http.ResponseWriter, r *http.Request) {
	// This would require scanning the storage directory
	// For now, return empty catalog
	response := map[string]interface{}{
		"repositories": []string{},
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Docker-Distribution-API-Version", "registry/2.0")
	json.NewEncoder(w).Encode(response)
}

// Helper methods

func (h *DockerRegistryHandler) respondWithError(w http.ResponseWriter, code int, errorCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Docker-Distribution-API-Version", "registry/2.0")
	w.WriteHeader(code)

	errorResponse := map[string]interface{}{
		"errors": []map[string]interface{}{
			{
				"code":    errorCode,
				"message": message,
			},
		},
	}

	json.NewEncoder(w).Encode(errorResponse)
}

func (h *DockerRegistryHandler) calculateDigest(data []byte) string {
	hasher := sha256.New()
	hasher.Write(data)
	return "sha256:" + hex.EncodeToString(hasher.Sum(nil))
}

func (h *DockerRegistryHandler) generateUploadUUID() string {
	// Simple UUID generation for uploads
	return fmt.Sprintf("upload-%d", time.Now().UnixNano())
}

// Placeholder methods for blob upload operations
func (h *DockerRegistryHandler) handleGetBlobUploadStatus(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *DockerRegistryHandler) handleCancelBlobUpload(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *DockerRegistryHandler) handleDeleteBlob(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusMethodNotAllowed)
}
