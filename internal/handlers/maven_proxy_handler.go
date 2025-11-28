package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/securestor/securestor/internal/cache"
	"github.com/securestor/securestor/internal/scanner"
	"github.com/securestor/securestor/internal/tenant"
)

// MavenProxyHandler handles Maven Central proxy requests
type MavenProxyHandler struct {
	cacheManager *cache.MultiTierCacheManager
	scanManager  *scanner.ScannerManager
	db           *sql.DB
	client       *http.Client
}

// NewMavenProxyHandler creates a new Maven proxy handler
func NewMavenProxyHandler(cacheManager *cache.MultiTierCacheManager, scanManager *scanner.ScannerManager, db *sql.DB) *MavenProxyHandler {
	return &MavenProxyHandler{
		cacheManager: cacheManager,
		scanManager:  scanManager,
		db:           db,
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

// GetArtifact handles GET /api/proxy/maven/*path
func (h *MavenProxyHandler) GetArtifact(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	ctx := r.Context()

	// Extract path from URL
	vars := mux.Vars(r)
	artifactPath := vars["path"]
	if artifactPath == "" {
		artifactPath = strings.TrimPrefix(r.URL.Path, "/api/proxy/maven/")
	}

	// Create cache key
	cacheKey := fmt.Sprintf("maven:%s", artifactPath)

	// Step 1: Check cache (L1 -> L2 -> L3)
	if reader, source, found := h.cacheManager.GetReader(ctx, cacheKey); found {
		h.serveCachedArtifact(w, reader, source, startTime)
		h.recordDownload(ctx, "maven", artifactPath, true, string(source), time.Since(startTime))
		return
	}

	// Step 2: Fetch from Maven Central
	remoteURL := fmt.Sprintf("https://repo1.maven.org/maven2/%s", artifactPath)
	data, contentType, err := h.fetchFromRemote(ctx, remoteURL)
	if err != nil {
		http.Error(w, fmt.Sprintf("Artifact not found: %v", err), http.StatusNotFound)
		h.recordDownload(ctx, "maven", artifactPath, false, "remote", time.Since(startTime))
		return
	}

	// Step 3: Cache the artifact (async)
	go h.cacheManager.Set(ctx, cacheKey, data, contentType, "", 24*time.Hour)

	// Step 4: Stream to client
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	w.Header().Set("X-Cache-Source", "remote")
	w.Header().Set("X-Cache-Hit", "false")
	w.Header().Set("X-Response-Time", fmt.Sprintf("%dms", time.Since(startTime).Milliseconds()))
	w.WriteHeader(http.StatusOK)
	w.Write(data)

	// Step 5: Queue security scan for JAR files (async)
	if strings.HasSuffix(artifactPath, ".jar") {
		go h.queueSecurityScan(ctx, artifactPath, data)
	}

	h.recordDownload(ctx, "maven", artifactPath, false, "remote", time.Since(startTime))
}

// GetMetadata handles HEAD /api/proxy/maven/*path
func (h *MavenProxyHandler) GetMetadata(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vars := mux.Vars(r)
	artifactPath := vars["path"]
	if artifactPath == "" {
		artifactPath = strings.TrimPrefix(r.URL.Path, "/api/proxy/maven/")
	}

	cacheKey := fmt.Sprintf("maven:meta:%s", artifactPath)

	// Check cache for metadata
	if artifact, _, found := h.cacheManager.Get(ctx, cacheKey); found {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", artifact.Size))
		w.Header().Set("Content-Type", artifact.ContentType)
		w.Header().Set("X-Cache-Hit", "true")
		w.WriteHeader(http.StatusOK)
		return
	}

	// Fetch metadata from remote
	remoteURL := fmt.Sprintf("https://repo1.maven.org/maven2/%s", artifactPath)
	resp, err := h.client.Head(remoteURL)
	if err != nil {
		http.Error(w, "Artifact not found", http.StatusNotFound)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, "Artifact not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Length", resp.Header.Get("Content-Length"))
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.Header().Set("Last-Modified", resp.Header.Get("Last-Modified"))
	w.Header().Set("X-Cache-Hit", "false")
	w.WriteHeader(http.StatusOK)
}

func (h *MavenProxyHandler) fetchFromRemote(ctx context.Context, url string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, "", err
	}

	req.Header.Set("User-Agent", "SecureStore/1.0")

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("remote returned %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	return data, contentType, nil
}

func (h *MavenProxyHandler) serveCachedArtifact(w http.ResponseWriter, reader io.ReadCloser, source cache.CacheSource, startTime time.Time) {
	defer reader.Close()

	// Read all data
	data, err := io.ReadAll(reader)
	if err != nil {
		http.Error(w, "Failed to read cached artifact", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	w.Header().Set("X-Cache-Source", string(source))
	w.Header().Set("X-Cache-Hit", "true")
	w.Header().Set("X-Response-Time", fmt.Sprintf("%dms", time.Since(startTime).Milliseconds()))
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (h *MavenProxyHandler) queueSecurityScan(ctx context.Context, artifactPath string, data []byte) {
	// Create temporary file for scanning
	tempFile := fmt.Sprintf("/tmp/maven-scan-%s.jar", uuid.New().String())
	defer func() {
		// Clean up temp file after scan
		time.AfterFunc(5*time.Minute, func() {
			os.Remove(tempFile)
		})
	}()

	// Write artifact to temp file
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		fmt.Printf("Failed to write temp file for scanning: %v\n", err)
		return
	}

	// Get scanners for maven/jar type
	scanners := h.scanManager.GetScannersForType("maven")
	if len(scanners) == 0 {
		fmt.Println("No scanners available for maven artifacts")
		return
	}

	// Run scan with first available scanner
	for _, scanner := range scanners {
		result, err := scanner.Scan(ctx, tempFile, "maven")
		if err != nil {
			fmt.Printf("Scanner %s failed: %v\n", scanner.Name(), err)
			continue
		}

		// Log scan results
		fmt.Printf("Maven artifact %s scanned by %s: %d vulnerabilities found\n",
			artifactPath, scanner.Name(), result.Summary.Total)

		// Store scan results in database (optional)
		h.storeScanResults(ctx, artifactPath, result)
		break
	}
}

func (h *MavenProxyHandler) storeScanResults(ctx context.Context, artifactPath string, result *scanner.ScanResult) {
	// Store in security_scans table
	query := `
		INSERT INTO security_scans (artifact_id, scanner_name, status, vulnerabilities_found, scanned_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (artifact_id, scanner_name) 
		DO UPDATE SET vulnerabilities_found = $4, scanned_at = $5, status = $3
	`

	artifactID := uuid.NewSHA1(uuid.NameSpaceURL, []byte(artifactPath))

	_, err := h.db.ExecContext(ctx, query,
		artifactID,
		result.ScannerName,
		"completed",
		result.Summary.Total,
		time.Now(),
	)

	if err != nil {
		fmt.Printf("Failed to store scan results: %v\n", err)
	}
}

func (h *MavenProxyHandler) recordDownload(ctx context.Context, repoType, artifactPath string, cacheHit bool, source string, responseTime time.Duration) {
	// Record download in remote_artifact_downloads table
	query := `
		INSERT INTO remote_artifact_downloads (
			repository_id, tenant_id, artifact_path, artifact_type, 
			cache_hit, cache_source, response_time_ms, downloaded_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, NOW()
		)
	`

	// Use default repository ID for Maven Central
	// In production, fetch from remote_repositories table
	repoID := uuid.NewSHA1(uuid.NameSpaceURL, []byte("maven-central"))

	// Get tenant ID from context (injected by tenant middleware)
	tenantID, err := tenant.GetTenantID(ctx)
	if err != nil {
		// Log warning and skip recording if no tenant context
		fmt.Printf("Warning: No tenant context for Maven download tracking: %v\n", err)
		return
	}

	_, err = h.db.ExecContext(ctx, query,
		repoID,
		tenantID,
		artifactPath,
		repoType,
		cacheHit,
		source,
		responseTime.Milliseconds(),
	)

	if err != nil {
		fmt.Printf("Failed to record download: %v\n", err)
	}
}
