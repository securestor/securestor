package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// OCIDistributionHandler handles Docker Registry v2 API and OCI Distribution Spec
type OCIDistributionHandler struct {
	basePath string
}

// OCIManifest represents an OCI image manifest
type OCIManifest struct {
	SchemaVersion int               `json:"schemaVersion"`
	MediaType     string            `json:"mediaType"`
	Config        OCIDescriptor     `json:"config"`
	Layers        []OCIDescriptor   `json:"layers"`
	Annotations   map[string]string `json:"annotations,omitempty"`
}

// OCIDescriptor represents a content descriptor
type OCIDescriptor struct {
	MediaType   string            `json:"mediaType"`
	Digest      string            `json:"digest"`
	Size        int64             `json:"size"`
	URLs        []string          `json:"urls,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// OCIIndex represents an OCI image index (manifest list)
type OCIIndex struct {
	SchemaVersion int               `json:"schemaVersion"`
	MediaType     string            `json:"mediaType"`
	Manifests     []OCIDescriptor   `json:"manifests"`
	Annotations   map[string]string `json:"annotations,omitempty"`
}

func NewOCIDistributionHandler(basePath string) *OCIDistributionHandler {
	return &OCIDistributionHandler{basePath: basePath}
}

// GetManifestPath returns the path for storing manifests
func (h *OCIDistributionHandler) GetManifestPath(name, reference string) string {
	// OCI Distribution API: /v2/<name>/manifests/<reference>
	return filepath.Join(h.basePath, "v2", name, "manifests", reference)
}

// GetBlobPath returns the path for storing blobs
func (h *OCIDistributionHandler) GetBlobPath(name, digest string) string {
	// OCI Distribution API: /v2/<name>/blobs/<digest>
	return filepath.Join(h.basePath, "v2", name, "blobs", digest)
}

// GetTagsPath returns the path for tag listing
func (h *OCIDistributionHandler) GetTagsPath(name string) string {
	// OCI Distribution API: /v2/<name>/tags/list
	return filepath.Join(h.basePath, "v2", name, "tags")
}

// ValidateManifest validates an OCI/Docker manifest
func (h *OCIDistributionHandler) ValidateManifest(data []byte) error {
	var manifest OCIManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("invalid manifest JSON: %v", err)
	}

	// Validate schema version
	if manifest.SchemaVersion != 2 {
		return fmt.Errorf("unsupported schema version: %d", manifest.SchemaVersion)
	}

	// Validate media type
	validMediaTypes := []string{
		"application/vnd.docker.distribution.manifest.v2+json",
		"application/vnd.oci.image.manifest.v1+json",
		"application/vnd.docker.distribution.manifest.list.v2+json",
		"application/vnd.oci.image.index.v1+json",
	}

	if !contains(validMediaTypes, manifest.MediaType) {
		return fmt.Errorf("invalid media type: %s", manifest.MediaType)
	}

	// Validate digest format
	digestRegex := regexp.MustCompile(`^[a-z0-9]+([+.-_][a-z0-9]+)*:[a-zA-Z0-9=_-]+$`)
	if !digestRegex.MatchString(manifest.Config.Digest) {
		return fmt.Errorf("invalid config digest format: %s", manifest.Config.Digest)
	}

	for _, layer := range manifest.Layers {
		if !digestRegex.MatchString(layer.Digest) {
			return fmt.Errorf("invalid layer digest format: %s", layer.Digest)
		}
	}

	return nil
}

// ValidateReference validates a tag or digest reference
func (h *OCIDistributionHandler) ValidateReference(reference string) error {
	// Tag pattern: [a-zA-Z0-9_][a-zA-Z0-9._-]{0,127}
	tagRegex := regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9._-]{0,127}$`)

	// Digest pattern: algorithm:encoded
	digestRegex := regexp.MustCompile(`^[a-z0-9]+([+.-_][a-z0-9]+)*:[a-zA-Z0-9=_-]+$`)

	if tagRegex.MatchString(reference) || digestRegex.MatchString(reference) {
		return nil
	}

	return fmt.Errorf("invalid reference format: %s", reference)
}

// ExtractManifestMetadata extracts metadata from an OCI manifest
func (h *OCIDistributionHandler) ExtractManifestMetadata(data []byte) (map[string]interface{}, error) {
	var manifest OCIManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}

	metadata := map[string]interface{}{
		"schema_version": manifest.SchemaVersion,
		"media_type":     manifest.MediaType,
		"config": map[string]interface{}{
			"digest":     manifest.Config.Digest,
			"size":       manifest.Config.Size,
			"media_type": manifest.Config.MediaType,
		},
		"layer_count": len(manifest.Layers),
	}

	// Calculate total size
	var totalSize int64 = manifest.Config.Size
	layers := make([]map[string]interface{}, len(manifest.Layers))

	for i, layer := range manifest.Layers {
		totalSize += layer.Size
		layers[i] = map[string]interface{}{
			"digest":     layer.Digest,
			"size":       layer.Size,
			"media_type": layer.MediaType,
		}
	}

	metadata["total_size"] = totalSize
	metadata["layers"] = layers

	if manifest.Annotations != nil {
		metadata["annotations"] = manifest.Annotations
	}

	return metadata, nil
}

// SupportsMultiArch checks if the manifest is a multi-architecture index
func (h *OCIDistributionHandler) SupportsMultiArch(data []byte) (bool, error) {
	var temp map[string]interface{}
	if err := json.Unmarshal(data, &temp); err != nil {
		return false, err
	}

	mediaType, ok := temp["mediaType"].(string)
	if !ok {
		return false, nil
	}

	return mediaType == "application/vnd.docker.distribution.manifest.list.v2+json" ||
		mediaType == "application/vnd.oci.image.index.v1+json", nil
}

// GetContentAddressableStorage returns CAS path for a digest
func (h *OCIDistributionHandler) GetContentAddressableStorage(digest string) string {
	// Split digest into algorithm and encoded parts
	parts := strings.SplitN(digest, ":", 2)
	if len(parts) != 2 {
		return ""
	}

	algorithm := parts[0]
	encoded := parts[1]

	// Create directory structure: cas/<algorithm>/<first-2-chars>/<digest>
	if len(encoded) < 2 {
		return filepath.Join(h.basePath, "cas", algorithm, encoded)
	}

	return filepath.Join(h.basePath, "cas", algorithm, encoded[:2], digest)
}

// Helper function to check if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// RegistryAPI provides Docker Registry v2 API operations
type RegistryAPI struct {
	ociHandler *OCIDistributionHandler
}

func NewRegistryAPI(basePath string) *RegistryAPI {
	return &RegistryAPI{
		ociHandler: NewOCIDistributionHandler(basePath),
	}
}

// PutManifest stores a manifest with reference
func (r *RegistryAPI) PutManifest(name, reference string, manifest []byte) error {
	// Validate manifest
	if err := r.ociHandler.ValidateManifest(manifest); err != nil {
		return fmt.Errorf("manifest validation failed: %v", err)
	}

	// Validate reference
	if err := r.ociHandler.ValidateReference(reference); err != nil {
		return fmt.Errorf("reference validation failed: %v", err)
	}

	// Get storage path
	manifestPath := r.ociHandler.GetManifestPath(name, reference)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0755); err != nil {
		return fmt.Errorf("failed to create manifest directory: %v", err)
	}

	// Write manifest to file
	if err := os.WriteFile(manifestPath, manifest, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %v", err)
	}

	// Update tag reference if this is a tag (not a digest)
	if !strings.Contains(reference, ":") {
		if err := r.updateTagReference(name, reference, manifest); err != nil {
			return fmt.Errorf("failed to update tag reference: %v", err)
		}
	}

	return nil
}

// GetManifest retrieves a manifest by reference
func (r *RegistryAPI) GetManifest(name, reference string) ([]byte, string, error) {
	manifestPath := r.ociHandler.GetManifestPath(name, reference)

	// Check if file exists
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return nil, "", fmt.Errorf("manifest not found: %s", reference)
	}

	// Read manifest from file
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read manifest: %v", err)
	}

	// Determine content type from manifest data
	var temp map[string]interface{}
	if err := json.Unmarshal(data, &temp); err != nil {
		return nil, "", fmt.Errorf("invalid manifest JSON: %v", err)
	}

	mediaType, ok := temp["mediaType"].(string)
	if !ok {
		mediaType = "application/vnd.docker.distribution.manifest.v2+json"
	}

	return data, mediaType, nil
}

// PutBlob stores a blob (layer or config)
func (r *RegistryAPI) PutBlob(name, digest string, blob io.Reader) error {
	blobPath := r.ociHandler.GetBlobPath(name, digest)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(blobPath), 0755); err != nil {
		return fmt.Errorf("failed to create blob directory: %v", err)
	}

	// Create the blob file
	file, err := os.Create(blobPath)
	if err != nil {
		return fmt.Errorf("failed to create blob file: %v", err)
	}
	defer file.Close()

	// Calculate hash while writing
	hasher := sha256.New()
	writer := io.MultiWriter(file, hasher)

	// Copy blob data
	written, err := io.Copy(writer, blob)
	if err != nil {
		// Clean up on error
		os.Remove(blobPath)
		return fmt.Errorf("failed to write blob: %v", err)
	}

	// Verify digest
	calculatedDigest := "sha256:" + hex.EncodeToString(hasher.Sum(nil))
	if calculatedDigest != digest {
		// Clean up on digest mismatch
		os.Remove(blobPath)
		return fmt.Errorf("digest mismatch: expected %s, got %s", digest, calculatedDigest)
	}

	// Also store in content-addressable storage
	casPath := r.ociHandler.GetContentAddressableStorage(digest)
	if casPath != "" && casPath != blobPath {
		if err := os.MkdirAll(filepath.Dir(casPath), 0755); err == nil {
			// Create a hard link to save space
			if err := os.Link(blobPath, casPath); err != nil {
				// If link fails, copy the file
				if err := r.copyFile(blobPath, casPath); err != nil {
					// Log error but don't fail the operation
					fmt.Printf("Warning: failed to create CAS link: %v\n", err)
				}
			}
		}
	}

	fmt.Printf("Stored blob %s (%d bytes)\n", digest, written)
	return nil
}

// GetBlob retrieves a blob by digest
func (r *RegistryAPI) GetBlob(name, digest string) (io.ReadCloser, error) {
	blobPath := r.ociHandler.GetBlobPath(name, digest)

	// Check if blob exists
	if _, err := os.Stat(blobPath); os.IsNotExist(err) {
		// Try content-addressable storage
		casPath := r.ociHandler.GetContentAddressableStorage(digest)
		if casPath != "" {
			if _, err := os.Stat(casPath); err == nil {
				blobPath = casPath
			} else {
				return nil, fmt.Errorf("blob not found: %s", digest)
			}
		} else {
			return nil, fmt.Errorf("blob not found: %s", digest)
		}
	}

	// Open and return the blob file
	file, err := os.Open(blobPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open blob: %v", err)
	}

	return file, nil
}

// ListTags returns all tags for a repository
func (r *RegistryAPI) ListTags(name string) ([]string, error) {
	manifestsPath := filepath.Join(r.ociHandler.basePath, "v2", name, "manifests")

	// Check if manifests directory exists
	if _, err := os.Stat(manifestsPath); os.IsNotExist(err) {
		return []string{}, nil
	}

	// Read directory contents
	entries, err := os.ReadDir(manifestsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifests directory: %v", err)
	}

	var tags []string
	digestRegex := regexp.MustCompile(`^sha256:`)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		// Skip digest references, only return tags
		if !digestRegex.MatchString(name) {
			tags = append(tags, name)
		}
	}

	return tags, nil
}

// DeleteManifest removes a manifest
func (r *RegistryAPI) DeleteManifest(name, reference string) error {
	manifestPath := r.ociHandler.GetManifestPath(name, reference)

	// Check if manifest exists
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return fmt.Errorf("manifest not found: %s", reference)
	}

	// Remove the manifest file
	if err := os.Remove(manifestPath); err != nil {
		return fmt.Errorf("failed to delete manifest: %v", err)
	}

	// If this was a tag, also try to remove the digest reference
	if !strings.Contains(reference, ":") {
		// This was a tag, calculate digest and remove digest reference
		manifestData, err := os.ReadFile(manifestPath)
		if err == nil {
			hasher := sha256.New()
			hasher.Write(manifestData)
			digest := "sha256:" + hex.EncodeToString(hasher.Sum(nil))

			digestPath := r.ociHandler.GetManifestPath(name, digest)
			os.Remove(digestPath) // Ignore errors
		}
	}

	return nil
}

// CheckBlobExists verifies if a blob exists
func (r *RegistryAPI) CheckBlobExists(name, digest string) (bool, int64, error) {
	blobPath := r.ociHandler.GetBlobPath(name, digest)

	// Check repository-specific blob storage
	if info, err := os.Stat(blobPath); err == nil {
		return true, info.Size(), nil
	}

	// Check content-addressable storage
	casPath := r.ociHandler.GetContentAddressableStorage(digest)
	if casPath != "" {
		if info, err := os.Stat(casPath); err == nil {
			return true, info.Size(), nil
		}
	}

	return false, 0, nil
}

// Helper methods

// updateTagReference updates tag to digest mapping
func (r *RegistryAPI) updateTagReference(name, tag string, manifest []byte) error {
	// Calculate manifest digest
	hasher := sha256.New()
	hasher.Write(manifest)
	digest := "sha256:" + hex.EncodeToString(hasher.Sum(nil))

	// Store manifest by digest as well
	digestPath := r.ociHandler.GetManifestPath(name, digest)
	if err := os.MkdirAll(filepath.Dir(digestPath), 0755); err != nil {
		return err
	}

	// Create hard link or copy
	tagPath := r.ociHandler.GetManifestPath(name, tag)
	if err := os.Link(tagPath, digestPath); err != nil {
		// If link fails, copy the manifest
		return r.copyFile(tagPath, digestPath)
	}

	return nil
}

// copyFile copies a file from src to dst
func (r *RegistryAPI) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}
