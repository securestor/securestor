package handlers

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/securestor/securestor/internal/cache"
	"github.com/securestor/securestor/internal/replicate"
	"github.com/securestor/securestor/internal/scanner"
	"github.com/securestor/securestor/internal/tenant"
)

// RemoteProxyHandler handles proxy requests for all remote registries (Gin version)
type RemoteProxyHandler struct {
	cacheManager     *cache.MultiTierCacheManager
	scanManager      *scanner.ScannerManager
	db               *sql.DB
	client           *http.Client
	replicationMixin *ReplicationMixin // Enterprise replication support
}

// NewRemoteProxyHandler creates a new unified remote proxy handler
func NewRemoteProxyHandler(cacheManager *cache.MultiTierCacheManager, scanManager *scanner.ScannerManager, db *sql.DB) *RemoteProxyHandler {
	return &RemoteProxyHandler{
		cacheManager: cacheManager,
		scanManager:  scanManager,
		db:           db,
		client: &http.Client{
			Timeout: 60 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		replicationMixin: nil, // Will be set via SetReplicationMixin
	}
}

// SetReplicationMixin enables replication support for proxied artifacts
func (h *RemoteProxyHandler) SetReplicationMixin(mixin *ReplicationMixin) {
	h.replicationMixin = mixin
}

// ProxyMaven handles Maven Central proxy - GET /api/v1/proxy/maven/*path
func (h *RemoteProxyHandler) ProxyMaven(c *gin.Context) {
	artifactPath := c.Param("path")
	if after, ok := strings.CutPrefix(artifactPath, "/"); ok {
		artifactPath = after
	}
	remoteURL := fmt.Sprintf("https://repo1.maven.org/maven2/%s", artifactPath)
	h.proxyArtifact(c, "maven", artifactPath, remoteURL)
}

// ProxyPyPI handles PyPI proxy - GET /api/v1/proxy/pypi/*path
func (h *RemoteProxyHandler) ProxyPyPI(c *gin.Context) {
	artifactPath := c.Param("path")
	if after, ok := strings.CutPrefix(artifactPath, "/"); ok {
		artifactPath = after
	}

	// PyPI simple API
	if strings.HasPrefix(artifactPath, "simple/") {
		packageName := strings.TrimPrefix(artifactPath, "simple/")
		packageName = strings.TrimSuffix(packageName, "/")
		remoteURL := fmt.Sprintf("https://pypi.org/simple/%s/", packageName)
		h.proxyArtifact(c, "pypi", artifactPath, remoteURL)
		return
	}

	// PyPI package files
	if strings.HasPrefix(artifactPath, "packages/") {
		remoteURL := fmt.Sprintf("https://files.pythonhosted.org/%s", artifactPath)
		h.proxyArtifact(c, "pypi", artifactPath, remoteURL)
		return
	}

	c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid PyPI path"})
}

// ProxyHelm handles Helm Charts proxy - GET /api/v1/proxy/helm/*path
func (h *RemoteProxyHandler) ProxyHelm(c *gin.Context) {
	artifactPath := c.Param("path")
	if after, ok := strings.CutPrefix(artifactPath, "/"); ok {
		artifactPath = after
	}
	remoteURL := fmt.Sprintf("https://charts.helm.sh/stable/%s", artifactPath)
	h.proxyArtifact(c, "helm", artifactPath, remoteURL)
}

// ProxyDocker handles Docker Hub proxy - GET /api/v1/proxy/docker/v2/*path
func (h *RemoteProxyHandler) ProxyDocker(c *gin.Context) {
	artifactPath := c.Param("path")
	if after, ok := strings.CutPrefix(artifactPath, "/"); ok {
		artifactPath = after
	}
	remoteURL := fmt.Sprintf("https://registry-1.docker.io/%s", artifactPath)
	h.proxyArtifact(c, "docker", artifactPath, remoteURL)
}

// ProxyNpm handles npm registry proxy - GET /api/v1/proxy/npm/*path
func (h *RemoteProxyHandler) ProxyNpm(c *gin.Context) {
	artifactPath := c.Param("path")
	if after, ok := strings.CutPrefix(artifactPath, "/"); ok {
		artifactPath = after
	}

	var remoteURL string
	if strings.HasPrefix(artifactPath, "@") {
		remoteURL = fmt.Sprintf("https://registry.npmjs.org/%s", artifactPath)
	} else if strings.Contains(artifactPath, "/-/") {
		remoteURL = fmt.Sprintf("https://registry.npmjs.org/%s", artifactPath)
	} else {
		remoteURL = fmt.Sprintf("https://registry.npmjs.org/%s", artifactPath)
	}

	h.proxyArtifact(c, "npm", artifactPath, remoteURL)
}

// proxyArtifact is the core proxy logic shared by all artifact types
func (h *RemoteProxyHandler) proxyArtifact(c *gin.Context, artifactType, artifactPath, remoteURL string) {
	startTime := time.Now()
	ctx := c.Request.Context()

	cacheKey := fmt.Sprintf("%s:%s", artifactType, artifactPath)

	// Step 1: Check multi-tier cache
	if reader, source, found := h.cacheManager.GetReader(ctx, cacheKey); found {
		h.serveCached(c, reader, source, startTime)
		h.recordDownload(ctx, artifactType, artifactPath, true, string(source), time.Since(startTime))
		return
	}

	// Step 2: Fetch from remote
	data, contentType, err := h.fetchFromRemote(ctx, remoteURL, artifactType)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Artifact not found: %v", err)})
		h.recordDownload(ctx, artifactType, artifactPath, false, "remote", time.Since(startTime))
		return
	}

	log.Printf("[PROXY] Fetched %s (%d bytes), starting cache + persist", artifactPath, len(data))

	// Step 3: Get tenant_id from Gin context (set by auth middleware)
	var tenantID uuid.UUID
	if tenantIDVal, exists := c.Get("tenant_id"); exists {
		if tid, ok := tenantIDVal.(uuid.UUID); ok {
			tenantID = tid
			log.Printf("[PROXY] Got tenant_id from context: %s", tenantID)
		} else {
			log.Printf("[PROXY] tenant_id exists in context but wrong type: %T", tenantIDVal)
		}
	} else {
		log.Printf("[PROXY] tenant_id not found in Gin context")
	}

	// If tenant_id not in context, try to get from user
	if tenantID == uuid.Nil {
		log.Printf("[PROXY] tenant_id is nil, trying to get from user_id")
		if userIDVal, exists := c.Get("user_id"); exists {
			log.Printf("[PROXY] Found user_id in context: %v (type: %T)", userIDVal, userIDVal)
			if uid, ok := userIDVal.(uuid.UUID); ok {
				// Query user's tenant_id from database
				err := h.db.QueryRow("SELECT tenant_id FROM users WHERE user_id = $1", uid).Scan(&tenantID)
				if err != nil {
					log.Printf("[PROXY] Failed to get tenant_id for user %s: %v", uid, err)
				} else {
					log.Printf("[PROXY] Got tenant_id from database: %s", tenantID)
				}
			} else {
				log.Printf("[PROXY] user_id has wrong type: %T", userIDVal)
			}
		} else {
			log.Printf("[PROXY] user_id not found in Gin context")
		}
	}

	log.Printf("[PROXY] Final tenant_id for caching: %s", tenantID)

	// Step 4: Cache the binary data in multi-tier cache (Redis/Disk/Cloud)
	checksum := fmt.Sprintf("%x", sha256.Sum256(data))
	cacheTTL := 24 * time.Hour
	if artifactType == "docker" {
		cacheTTL = 168 * time.Hour // 7 days for docker
	}

	if err := h.cacheManager.Set(ctx, cacheKey, data, contentType, checksum, cacheTTL); err != nil {
		log.Printf("[PROXY] Failed to cache binary in multi-tier cache: %v", err)
	} else {
		log.Printf("[PROXY] Successfully cached binary: %s (%d bytes) in multi-tier cache", cacheKey, len(data))
	}

	// Step 5: Persist metadata to database asynchronously
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[CACHE-PERSIST] Recovered from panic: %v", r)
			}
		}()

		log.Printf("[CACHE-PERSIST] Starting for: %s (%s) with tenant: %s", artifactPath, artifactType, tenantID)
		h.persistCachedArtifact(tenantID, artifactType, artifactPath, cacheKey, data, contentType, remoteURL)
		log.Printf("[CACHE-PERSIST] Completed database persistence for: %s", artifactPath)
	}()

	// Step 6: Trigger enterprise replication (async)
	if h.replicationMixin != nil && h.replicationMixin.IsEnabled() && tenantID != uuid.Nil {
		go h.replicationMixin.ReplicateAsync(&replicate.ReplicationRequest{
			TenantID:     tenantID.String(),
			RepositoryID: "proxy-cache", // Logical repository for proxied artifacts
			ArtifactID:   fmt.Sprintf("%s-%s", artifactType, artifactPath),
			ArtifactType: artifactType,
			Data:         data,
			Metadata: map[string]string{
				"remote_url":   remoteURL,
				"content_type": contentType,
				"checksum":     checksum,
			},
			BucketName: fmt.Sprintf("proxy/%s", artifactType),
			FileName:   artifactPath,
		})
	}

	// Step 7: Stream to client
	c.Header("Content-Type", contentType)
	c.Header("Content-Length", fmt.Sprintf("%d", len(data)))
	c.Header("X-Cache-Source", "remote")
	c.Header("X-Cache-Hit", "false")
	c.Header("X-Response-Time", fmt.Sprintf("%dms", time.Since(startTime).Milliseconds()))
	c.Data(http.StatusOK, contentType, data)

	// Step 8: Queue security scan if applicable
	if h.shouldScan(artifactPath) {
		go h.queueScan(ctx, artifactType, artifactPath, data)
	}

	h.recordDownload(ctx, artifactType, artifactPath, false, "remote", time.Since(startTime))
}

func (h *RemoteProxyHandler) fetchFromRemote(ctx context.Context, url, artifactType string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, "", err
	}

	req.Header.Set("User-Agent", "SecureStore/1.0")

	switch artifactType {
	case "docker":
		req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	case "pypi":
		req.Header.Set("Accept", "application/vnd.pypi.simple.v1+json")
	case "helm":
		req.Header.Set("Accept", "application/yaml")
	}

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

func (h *RemoteProxyHandler) serveCached(c *gin.Context, reader io.ReadCloser, source cache.CacheSource, startTime time.Time) {
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read cache"})
		return
	}

	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Length", fmt.Sprintf("%d", len(data)))
	c.Header("X-Cache-Source", string(source))
	c.Header("X-Cache-Hit", "true")
	c.Header("X-Response-Time", fmt.Sprintf("%dms", time.Since(startTime).Milliseconds()))
	c.Data(http.StatusOK, "application/octet-stream", data)
}

func (h *RemoteProxyHandler) shouldScan(path string) bool {
	scannable := []string{".jar", ".war", ".whl", ".tar.gz", ".tgz", ".zip"}
	for _, ext := range scannable {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}

func (h *RemoteProxyHandler) queueScan(ctx context.Context, artifactType, artifactPath string, data []byte) {
	tempFile := fmt.Sprintf("/tmp/scan-%s-%s", artifactType, uuid.New().String())
	defer func() {
		time.AfterFunc(5*time.Minute, func() {
			os.Remove(tempFile)
		})
	}()

	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return
	}

	scanners := h.scanManager.GetScannersForType(artifactType)
	if len(scanners) == 0 {
		return
	}

	for _, scanner := range scanners {
		result, err := scanner.Scan(ctx, tempFile, artifactType)
		if err != nil {
			continue
		}

		h.storeScanResults(ctx, artifactPath, result)
		break
	}
}

func (h *RemoteProxyHandler) storeScanResults(ctx context.Context, artifactPath string, result *scanner.ScanResult) {
	query := `
		INSERT INTO security_scans (artifact_id, scanner_name, status, vulnerabilities_found, scanned_at)
		VALUES ($1, $2, $3, $4, $5)
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
		log.Printf("Failed to store scan results: %v\n", err)
	}
}

func (h *RemoteProxyHandler) recordDownload(ctx context.Context, repoType, artifactPath string, cacheHit bool, source string, responseTime time.Duration) {
	query := `
		INSERT INTO remote_artifact_downloads (
			repository_id, tenant_id, artifact_path, artifact_type, 
			cache_hit, cache_source, response_time_ms, downloaded_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, NOW()
		)
	`

	repoID := uuid.NewSHA1(uuid.NameSpaceURL, []byte(repoType+"-proxy"))

	tenantID, err := tenant.GetTenantID(ctx)
	if err != nil {
		log.Printf("Warning: No tenant context for download tracking: %v\n", err)
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
		log.Printf("Failed to record download: %v\n", err)
	}
}

// persistCachedArtifact records cached artifact to database for enterprise tracking
func (h *RemoteProxyHandler) persistCachedArtifact(tenantID uuid.UUID, artifactType, artifactPath, cacheKey string, data []byte, contentType, originURL string) {
	log.Printf("[CACHE-PERSIST] Function entered for %s, parsing artifact info...", artifactPath)

	artifactName, artifactVersion := h.parseArtifactInfo(artifactPath, artifactType)
	log.Printf("[CACHE-PERSIST] Parsed: name=%s, version=%s", artifactName, artifactVersion)

	checksum := fmt.Sprintf("%x", sha256.Sum256(data))
	log.Printf("[CACHE-PERSIST] Computed checksum: %s", checksum[:16]+"...")

	expiryHours := 720
	if artifactType == "docker" {
		expiryHours = 168
	}
	expiryAt := time.Now().Add(time.Duration(expiryHours) * time.Hour)

	query := `
		INSERT INTO cached_artifacts (
			tenant_id, artifact_path, artifact_type, artifact_name, artifact_version,
			cache_level, cache_key, size_bytes, checksum, checksum_algorithm,
			scan_status, origin_url, origin_repository, last_accessed, 
			next_scan_at, expiry_at, created_at, updated_at, metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW(), 
			NOW() + INTERVAL '24 hours', $14, NOW(), NOW(), $15
		)
		ON CONFLICT (cache_key, cache_level, tenant_id) 
		DO UPDATE SET 
			size_bytes = EXCLUDED.size_bytes,
			checksum = EXCLUDED.checksum,
			last_accessed = NOW(),
			updated_at = NOW(),
			hit_count = cached_artifacts.hit_count + 1
		RETURNING id
	`

	metadata := map[string]interface{}{
		"content_type": contentType,
		"cached_at":    time.Now().Format(time.RFC3339),
		"cache_source": "multi-tier",
	}
	metadataJSON, _ := json.Marshal(metadata)

	var cachedID uuid.UUID
	log.Printf("[CACHE-PERSIST] About to execute database query for %s (size: %d bytes)", artifactPath, len(data))

	row := h.db.QueryRowContext(context.Background(), query,
		tenantID,
		artifactPath,
		artifactType,
		artifactName,
		artifactVersion,
		"L1",
		cacheKey,
		len(data),
		checksum,
		"sha256",
		"pending",
		originURL,
		fmt.Sprintf("%s-proxy", artifactType),
		expiryAt,
		string(metadataJSON),
	)

	log.Printf("[CACHE-PERSIST] Database query executed, scanning result...")
	dbErr := row.Scan(&cachedID)

	if dbErr != nil {
		log.Printf("[CACHE-PERSIST] Failed to persist cached artifact %s: %v", artifactPath, dbErr)
		return
	}

	log.Printf("[CACHE-PERSIST] Successfully persisted cached artifact: %s (ID: %s, Size: %d bytes)",
		artifactPath, cachedID, len(data))

	logQuery := `
		INSERT INTO cache_access_logs (
			tenant_id, cached_artifact_id, access_type, success, accessed_at
		) VALUES ($1, $2, $3, $4, NOW())
	`
	_, logErr := h.db.Exec(logQuery, tenantID, cachedID, "cache_write", true)
	if logErr != nil {
		log.Printf("[CACHE-ACCESS-LOG] Failed to create access log: %v", logErr)
	}
}

func (h *RemoteProxyHandler) parseArtifactInfo(artifactPath, artifactType string) (name, version string) {
	parts := strings.Split(artifactPath, "/")

	switch artifactType {
	case "npm":
		if len(parts) > 0 {
			if strings.HasPrefix(parts[0], "@") && len(parts) > 1 {
				name = parts[0] + "/" + parts[1]
			} else {
				name = parts[0]
			}
		}
	case "maven":
		if len(parts) >= 3 {
			name = strings.Join(parts[0:len(parts)-1], "/")
			version = parts[len(parts)-1]
		}
	case "pypi":
		if strings.HasPrefix(artifactPath, "simple/") && len(parts) > 1 {
			name = parts[1]
		}
	case "docker":
		if len(parts) >= 2 {
			name = parts[len(parts)-2]
			if len(parts) >= 4 {
				version = parts[len(parts)-1]
			}
		}
	default:
		if len(parts) > 0 {
			name = parts[len(parts)-1]
		}
	}

	if name == "" {
		name = artifactPath
	}

	return name, version
}

// GetRepositories returns configured remote repositories - GET /api/v1/proxy/repositories
func (h *RemoteProxyHandler) GetRepositories(c *gin.Context) {
	query := `
		SELECT id, name, type, remote_url, cache_enabled, health_status, 
		       last_health_check, created_at
		FROM remote_repositories
		WHERE is_active = true
		ORDER BY created_at DESC
	`

	rows, err := h.db.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var repos []map[string]interface{}
	for rows.Next() {
		var id, name, repoType, remoteURL, healthStatus string
		var cacheEnabled bool
		var lastHealthCheck, createdAt time.Time

		err := rows.Scan(&id, &name, &repoType, &remoteURL, &cacheEnabled,
			&healthStatus, &lastHealthCheck, &createdAt)
		if err != nil {
			continue
		}

		repos = append(repos, map[string]interface{}{
			"id":                id,
			"name":              name,
			"type":              repoType,
			"remote_url":        remoteURL,
			"cache_enabled":     cacheEnabled,
			"health_status":     healthStatus,
			"last_health_check": lastHealthCheck,
			"created_at":        createdAt,
		})
	}

	c.JSON(http.StatusOK, repos)
}

// GetCacheStats returns cache statistics - GET /api/v1/proxy/cache/stats
func (h *RemoteProxyHandler) GetCacheStats(c *gin.Context) {
	tenantID, err := tenant.GetTenantID(c.Request.Context())
	if err != nil {
		log.Printf("[CACHE-MGMT] No tenant context found in GetCacheStats request: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No tenant context found"})
		return
	}
	log.Printf("[CACHE-MGMT] Getting cache stats for tenant: %s", tenantID.String())

	query := `
		SELECT 
			COALESCE(SUM(l1_hits), 0) as l1_hits,
			COALESCE(SUM(l1_misses), 0) as l1_misses,
			COALESCE(SUM(l2_hits), 0) as l2_hits,
			COALESCE(SUM(l2_misses), 0) as l2_misses,
			COALESCE(SUM(l3_hits), 0) as l3_hits,
			COALESCE(SUM(l3_misses), 0) as l3_misses,
			COALESCE(SUM(remote_fetches), 0) as remote_fetches,
			COALESCE(AVG(avg_response_time_ms), 0) as avg_response_time,
			COALESCE(SUM(total_bandwidth_bytes), 0) as total_bandwidth,
			COALESCE(SUM(bandwidth_saved_bytes), 0) as bandwidth_saved
		FROM remote_cache_stats
		WHERE time_bucket >= NOW() - INTERVAL '24 hours'
	`

	var l1Hits, l1Misses, l2Hits, l2Misses, l3Hits, l3Misses, remoteFetches int64
	var avgResponseTime float64
	var totalBandwidth, bandwidthSaved int64

	err = h.db.QueryRow(query).Scan(
		&l1Hits, &l1Misses, &l2Hits, &l2Misses, &l3Hits, &l3Misses,
		&remoteFetches, &avgResponseTime, &totalBandwidth, &bandwidthSaved,
	)

	if err != nil {
		log.Printf("[CACHE-MGMT] Failed to query cache stats: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	totalHits := l1Hits + l2Hits + l3Hits
	totalRequests := totalHits + remoteFetches
	hitRate := 0.0
	if totalRequests > 0 {
		hitRate = float64(totalHits) / float64(totalRequests) * 100
	}

	var l1Size, l2Size, l3Size int64
	sizeQuery := `
		SELECT 
			COALESCE(SUM(CASE WHEN cache_level = 'L1' THEN size_bytes ELSE 0 END), 0) as l1_size,
			COALESCE(SUM(CASE WHEN cache_level = 'L2' THEN size_bytes ELSE 0 END), 0) as l2_size,
			COALESCE(SUM(CASE WHEN cache_level = 'L3' THEN size_bytes ELSE 0 END), 0) as l3_size
		FROM cached_artifacts
		WHERE tenant_id = $1 AND (expiry_at IS NULL OR expiry_at > NOW())
	`
	h.db.QueryRow(sizeQuery, tenantID).Scan(&l1Size, &l2Size, &l3Size)

	stats := gin.H{
		"timestamp": time.Now().Format(time.RFC3339),
		"period":    "24 hours",
		"tiers": gin.H{
			"l1_cache": gin.H{
				"name":        "Redis (L1)",
				"type":        "in-memory",
				"hits":        l1Hits,
				"misses":      l1Misses,
				"hit_rate":    calculateHitRate(l1Hits, l1Misses),
				"size_bytes":  l1Size,
				"size_mb":     float64(l1Size) / 1048576,
				"performance": "fastest",
				"latency_ms":  "<5",
			},
			"l2_cache": gin.H{
				"name":        "Disk (L2)",
				"type":        "local-disk",
				"hits":        l2Hits,
				"misses":      l2Misses,
				"hit_rate":    calculateHitRate(l2Hits, l2Misses),
				"size_bytes":  l2Size,
				"size_mb":     float64(l2Size) / 1048576,
				"performance": "fast",
				"latency_ms":  "5-50",
			},
			"l3_cache": gin.H{
				"name":        "Cloud (L3)",
				"type":        "cloud-storage",
				"hits":        l3Hits,
				"misses":      l3Misses,
				"hit_rate":    calculateHitRate(l3Hits, l3Misses),
				"size_bytes":  l3Size,
				"size_mb":     float64(l3Size) / 1048576,
				"performance": "moderate",
				"latency_ms":  "50-200",
			},
		},
		"overall": gin.H{
			"total_requests":         totalRequests,
			"total_hits":             totalHits,
			"remote_fetches":         remoteFetches,
			"hit_rate":               fmt.Sprintf("%.2f%%", hitRate),
			"avg_response_ms":        avgResponseTime,
			"total_bandwidth_gb":     float64(totalBandwidth) / 1073741824,
			"bandwidth_saved_gb":     float64(bandwidthSaved) / 1073741824,
			"total_cache_size_bytes": l1Size + l2Size + l3Size,
			"total_cache_size_gb":    float64(l1Size+l2Size+l3Size) / 1073741824,
		},
		"efficiency": gin.H{
			"cache_efficiency":     fmt.Sprintf("%.2f%%", hitRate),
			"bandwidth_saved_pct":  calculateBandwidthSavedPercent(totalBandwidth, bandwidthSaved),
			"avg_response_time_ms": avgResponseTime,
		},
	}

	c.JSON(http.StatusOK, stats)
}

func calculateHitRate(hits, misses int64) float64 {
	total := hits + misses
	if total == 0 {
		return 0.0
	}
	return float64(hits) / float64(total) * 100
}

func calculateBandwidthSavedPercent(totalBandwidth, bandwidthSaved int64) string {
	if totalBandwidth == 0 {
		return "0.00%"
	}
	pct := float64(bandwidthSaved) / float64(totalBandwidth) * 100
	return fmt.Sprintf("%.2f%%", pct)
}

// ListCachedItems - GET /api/v1/proxy/cache/items
func (h *RemoteProxyHandler) ListCachedItems(c *gin.Context) {
	log.Printf("[CACHE-MGMT] ListCachedItems handler called - Method: %s, Path: %s", c.Request.Method, c.Request.URL.Path)

	tenantID, err := tenant.GetTenantID(c.Request.Context())
	if err != nil {
		log.Printf("[CACHE-MGMT] No tenant context found: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No tenant context found"})
		return
	}
	log.Printf("[CACHE-MGMT] Filtering cached items for tenant: %s", tenantID.String())

	page := getQueryIntGin(c, "page", 1)
	if page < 1 {
		page = 1
	}

	pageSize := getQueryIntGin(c, "pageSize", 50)
	if pageSize < 1 || pageSize > 1000 {
		pageSize = 50
	}

	artifactType := c.Query("type")
	cacheLevel := c.Query("level")
	sortBy := c.Query("sort")
	order := c.Query("order")
	searchTerm := c.Query("search")

	if order != "asc" && order != "desc" {
		order = "desc"
	}

	validSortColumns := map[string]bool{
		"last_accessed": true, "created_at": true, "size_bytes": true,
		"hit_count": true, "vulnerabilities_count": true,
		"artifact_type": true, "cache_level": true,
	}
	if !validSortColumns[sortBy] {
		sortBy = "last_accessed"
	}

	whereClause := "tenant_id = $1"
	args := []interface{}{tenantID}
	argIndex := 2

	if artifactType != "" && artifactType != "all" {
		whereClause += fmt.Sprintf(" AND artifact_type = $%d", argIndex)
		args = append(args, artifactType)
		argIndex++
	}

	if cacheLevel != "" && cacheLevel != "all" {
		whereClause += fmt.Sprintf(" AND cache_level = $%d", argIndex)
		args = append(args, cacheLevel)
		argIndex++
	}

	if searchTerm != "" {
		whereClause += fmt.Sprintf(" AND (artifact_path ILIKE $%d OR artifact_name ILIKE $%d)", argIndex, argIndex)
		args = append(args, "%"+searchTerm+"%")
		argIndex++
	}

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM cached_artifacts WHERE %s", whereClause)
	err = h.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count items"})
		return
	}

	offset := (page - 1) * pageSize

	query := fmt.Sprintf(`
		SELECT 
			id, artifact_path, artifact_type, artifact_name, artifact_version,
			cache_level, size_bytes, hit_count, miss_count, last_accessed,
			scan_status, vulnerabilities_count, critical_vulnerabilities,
			high_vulnerabilities, medium_vulnerabilities, low_vulnerabilities,
			is_quarantined, checksum, created_at, updated_at
		FROM cached_artifacts
		WHERE %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, sortBy, order, argIndex, argIndex+1)

	args = append(args, pageSize, offset)

	rows, err := h.db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to fetch items: %v", err)})
		return
	}
	defer rows.Close()

	items := []gin.H{}
	for rows.Next() {
		var id, artifactPath, artifactType sql.NullString
		var artifactName, artifactVersion sql.NullString
		var cacheLevel string
		var sizeBytes, hitCount, missCount int64
		var vulnerabilitiesCount, criticalVulns, highVulns, mediumVulns, lowVulns int64
		var lastAccessed, createdAt, updatedAt sql.NullTime
		var scanStatus, checksum sql.NullString
		var isQuarantined bool

		err := rows.Scan(
			&id, &artifactPath, &artifactType, &artifactName, &artifactVersion,
			&cacheLevel, &sizeBytes, &hitCount, &missCount, &lastAccessed,
			&scanStatus, &vulnerabilitiesCount, &criticalVulns, &highVulns, &mediumVulns, &lowVulns,
			&isQuarantined, &checksum, &createdAt, &updatedAt,
		)
		if err != nil {
			continue
		}

		items = append(items, gin.H{
			"id":                       id.String,
			"artifact_path":            artifactPath.String,
			"artifact_type":            artifactType.String,
			"artifact_name":            artifactName.String,
			"artifact_version":         artifactVersion.String,
			"cache_level":              cacheLevel,
			"size_bytes":               sizeBytes,
			"hit_count":                hitCount,
			"miss_count":               missCount,
			"last_accessed":            lastAccessed.Time,
			"scan_status":              scanStatus.String,
			"vulnerabilities_count":    vulnerabilitiesCount,
			"critical_vulnerabilities": criticalVulns,
			"high_vulnerabilities":     highVulns,
			"medium_vulnerabilities":   mediumVulns,
			"low_vulnerabilities":      lowVulns,
			"is_quarantined":           isQuarantined,
			"checksum":                 checksum.String,
			"created_at":               createdAt.Time,
			"updated_at":               updatedAt.Time,
		})
	}

	totalPages := (total + pageSize - 1) / pageSize

	c.JSON(http.StatusOK, gin.H{
		"items":       items,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": totalPages,
		"has_next":    page < totalPages,
		"has_prev":    page > 1,
	})
}

func getQueryIntGin(c *gin.Context, key string, defaultValue int) int {
	valueStr := c.Query(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}

// RegisterRoutes registers all remote proxy routes
func (h *RemoteProxyHandler) RegisterRoutes(router *gin.RouterGroup) {
	log.Printf("[PROXY] Registering remote proxy routes...")

	// Repository management
	router.GET("/repositories", h.GetRepositories)

	// Artifact proxy routes (wildcard routes)
	router.GET("/maven/*path", h.ProxyMaven)
	router.GET("/pypi/*path", h.ProxyPyPI)
	router.GET("/helm/*path", h.ProxyHelm)
	router.GET("/docker/*path", h.ProxyDocker)
	router.GET("/npm/*path", h.ProxyNpm)

	log.Printf("[PROXY] ✅ Remote proxy routes registered successfully")
}
