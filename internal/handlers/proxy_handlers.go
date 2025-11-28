package handlers

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ======================= PROXY ARTIFACT HANDLER =======================

type ProxyArtifactHandler struct {
	requestRouter   interface{} // ProxyRequestRouter
	virtualResolver interface{} // VirtualRepositoryResolver
	logger          interface{}
	mu              sync.RWMutex
}

func NewProxyArtifactHandler() *ProxyArtifactHandler {
	return &ProxyArtifactHandler{}
}

// GetArtifact handles GET /api/proxy/:repoID/*path
func (pah *ProxyArtifactHandler) GetArtifact(c *gin.Context) {
	repoID := c.Param("repoID")
	path := c.Param("path")

	if repoID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "repository ID required"})
		return
	}

	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "artifact path required"})
		return
	}

	// Extract protocol from request header or detect from path
	protocol := c.GetString("protocol")
	if protocol == "" {
		protocol = pah.detectProtocol(c, path)
	}

	// Create proxy request
	proxyReq := &ProxyRequest{
		RepositoryID: repoID,
		Path:         path,
		Protocol:     protocol,
		Context:      c.Request.Context(),
		ClientIP:     c.ClientIP(),
		RequestID:    c.GetString("requestID"),
	}

	// Route the request
	resp, err := pah.routeProxyRequest(proxyReq)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	// Set response headers
	c.Header("X-Cache-Source", resp.CacheSource)
	c.Header("X-Cache-Hit", strconv.FormatBool(resp.CacheHit))
	c.Header("X-Response-Time", fmt.Sprintf("%dms", resp.Duration.Milliseconds()))

	if resp.Checksum != "" {
		c.Header("X-Checksum", resp.Checksum)
	}

	if resp.ContentType != "" {
		c.Header("Content-Type", resp.ContentType)
	}

	if resp.Size > 0 {
		c.Header("Content-Length", strconv.FormatInt(resp.Size, 10))
	}

	// Stream the response
	if resp.Stream != nil {
		defer resp.Stream.Close()
		c.Stream(func(w io.Writer) bool {
			_, err := io.Copy(w, resp.Stream)
			return err == nil
		})
	}

	c.Status(http.StatusOK)
}

// HeadArtifact handles HEAD /api/proxy/:repoID/*path
func (pah *ProxyArtifactHandler) HeadArtifact(c *gin.Context) {
	repoID := c.Param("repoID")
	path := c.Param("path")

	if repoID == "" || path == "" {
		c.Status(http.StatusBadRequest)
		return
	}

	// Check if artifact exists without downloading
	exists, metadata := pah.checkArtifactExists(repoID, path)
	if !exists {
		c.Status(http.StatusNotFound)
		return
	}

	// Set metadata headers
	if metadata != nil {
		c.Header("Content-Length", strconv.FormatInt(metadata.Size, 10))
		c.Header("X-Checksum", metadata.Checksum)
		c.Header("Last-Modified", metadata.Modified.Format(http.TimeFormat))
		if metadata.ContentType != "" {
			c.Header("Content-Type", metadata.ContentType)
		}
	}

	c.Status(http.StatusOK)
}

func (pah *ProxyArtifactHandler) routeProxyRequest(req *ProxyRequest) (*ProxyResponse, error) {
	// Implementation: Use the request router to handle the request
	return nil, fmt.Errorf("not implemented")
}

func (pah *ProxyArtifactHandler) detectProtocol(c *gin.Context, path string) string {
	// Detect protocol from path patterns
	if strings.Contains(path, ".jar") {
		return "maven"
	}
	if strings.Contains(path, ".tgz") || strings.Contains(path, ".tar.gz") {
		return "npm"
	}
	if strings.Contains(path, ".tar") {
		return "helm"
	}
	if strings.HasPrefix(path, "v2") || strings.HasPrefix(path, "v1") {
		return "docker"
	}
	if strings.HasSuffix(path, ".whl") || strings.HasSuffix(path, ".tar.gz") {
		return "pypi"
	}
	return "unknown"
}

func (pah *ProxyArtifactHandler) checkArtifactExists(repoID string, path string) (bool, *ArtifactMetadata) {
	// Implementation: Check if artifact exists
	return false, nil
}

// ======================= REPOSITORY CONFIGURATION HANDLER =======================

type RepositoryConfigHandler struct {
	db                *sql.DB
	repositoryService interface{}
	logger            interface{}
	mu                sync.RWMutex
}

type RepositoryDTO struct {
	ID           string           `json:"id"`
	Name         string           `json:"name"`
	Type         string           `json:"type"` // "maven", "npm", "docker", "pypi", "helm"
	RemoteURL    string           `json:"remote_url"`
	Local        bool             `json:"local"`
	CacheConfig  *CacheConfigDTO  `json:"cache_config"`
	HealthStatus *HealthStatusDTO `json:"health_status"`
	CreatedAt    time.Time        `json:"created_at"`
}

type CacheConfigDTO struct {
	Enabled        bool   `json:"enabled"`
	TTL            string `json:"ttl"`             // "1h", "24h", "7d"
	MaxSize        int64  `json:"max_size"`        // bytes
	EvictionPolicy string `json:"eviction_policy"` // "LRU", "LFU", "FIFO"
}

type HealthStatusDTO struct {
	Status       string    `json:"status"` // "healthy", "degraded", "offline"
	LastCheck    time.Time `json:"last_check"`
	ResponseTime int64     `json:"response_time_ms"`
	ErrorCount   int       `json:"error_count"`
}

func NewRepositoryConfigHandler(db *sql.DB) *RepositoryConfigHandler {
	return &RepositoryConfigHandler{
		db: db,
	}
}

// CreateRepository handles POST /api/repositories
func (rch *RepositoryConfigHandler) CreateRepository(c *gin.Context) {
	var req RepositoryDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate repository type
	if !rch.isValidRepositoryType(req.Type) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository type"})
		return
	}

	// Create repository in service
	created, err := rch.createRepositoryInService(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, created)
}

// GetRepository handles GET /api/repositories/:id
func (rch *RepositoryConfigHandler) GetRepository(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "repository ID required"})
		return
	}

	repo, err := rch.getRepositoryFromService(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	c.JSON(http.StatusOK, repo)
}

// ListRepositories handles GET /api/repositories
func (rch *RepositoryConfigHandler) ListRepositories(c *gin.Context) {
	// Parse query params
	repoType := c.Query("type")
	onlyLocal := c.Query("local") == "true"

	repos, err := rch.listRepositoriesFromService(repoType, onlyLocal)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count":        len(repos),
		"repositories": repos,
	})
}

// UpdateRepository handles PUT /api/repositories/:id
func (rch *RepositoryConfigHandler) UpdateRepository(c *gin.Context) {
	id := c.Param("id")
	var req RepositoryDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updated, err := rch.updateRepositoryInService(id, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, updated)
}

// DeleteRepository handles DELETE /api/repositories/:id
func (rch *RepositoryConfigHandler) DeleteRepository(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "repository ID required"})
		return
	}

	err := rch.deleteRepositoryFromService(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func (rch *RepositoryConfigHandler) isValidRepositoryType(repoType string) bool {
	validTypes := map[string]bool{
		"maven":  true,
		"npm":    true,
		"docker": true,
		"pypi":   true,
		"helm":   true,
	}
	return validTypes[repoType]
}

func (rch *RepositoryConfigHandler) createRepositoryInService(dto *RepositoryDTO) (*RepositoryDTO, error) {
	// Generate UUID for new repository
	repoID := uuid.New().String()

	// Insert into remote_repositories table
	query := `
		INSERT INTO remote_repositories (
			id, name, type, url, cache_enabled, cache_ttl_hours,
			max_cache_size_mb, upstream_health_check_interval_seconds,
			protocol_adapter, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		RETURNING id, created_at
	`

	// Set defaults if not provided
	cacheEnabled := true
	cacheTTL := 24
	maxCacheSize := 10240      // 10GB
	healthCheckInterval := 300 // 5 minutes

	var createdAt time.Time
	err := rch.db.QueryRow(query,
		repoID,
		dto.Name,
		dto.Type,
		dto.RemoteURL,
		cacheEnabled,
		cacheTTL,
		maxCacheSize,
		healthCheckInterval,
		dto.Type, // protocol_adapter same as type
	).Scan(&repoID, &createdAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create repository: %w", err)
	}

	// Return created repository
	return &RepositoryDTO{
		ID:        repoID,
		Name:      dto.Name,
		Type:      dto.Type,
		RemoteURL: dto.RemoteURL,
		Local:     false,
		CreatedAt: createdAt,
	}, nil
}

func (rch *RepositoryConfigHandler) getRepositoryFromService(id string) (*RepositoryDTO, error) {
	query := `
		SELECT id, name, type, url, cache_enabled, created_at
		FROM remote_repositories
		WHERE id = $1
	`

	var repo RepositoryDTO
	var cacheEnabled bool
	var url string

	err := rch.db.QueryRow(query, id).Scan(
		&repo.ID, &repo.Name, &repo.Type, &url, &cacheEnabled, &repo.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("repository not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	repo.RemoteURL = url
	repo.Local = false

	return &repo, nil
}

func (rch *RepositoryConfigHandler) listRepositoriesFromService(repoType string, onlyLocal bool) ([]RepositoryDTO, error) {
	query := `
		SELECT id, name, type, url, cache_enabled, created_at
		FROM remote_repositories
		WHERE 1=1
	`

	args := []interface{}{}
	argPos := 1

	if repoType != "" {
		query += fmt.Sprintf(" AND type = $%d", argPos)
		args = append(args, repoType)
		argPos++
	}

	query += " ORDER BY created_at DESC"

	rows, err := rch.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}
	defer rows.Close()

	var repositories []RepositoryDTO
	for rows.Next() {
		var repo RepositoryDTO
		var cacheEnabled bool
		var url string

		err := rows.Scan(&repo.ID, &repo.Name, &repo.Type, &url, &cacheEnabled, &repo.CreatedAt)
		if err != nil {
			continue
		}

		repo.RemoteURL = url
		repo.Local = false
		repositories = append(repositories, repo)
	}

	return repositories, nil
}

func (rch *RepositoryConfigHandler) updateRepositoryInService(id string, dto *RepositoryDTO) (*RepositoryDTO, error) {
	return nil, fmt.Errorf("not implemented")
}

func (rch *RepositoryConfigHandler) deleteRepositoryFromService(id string) error {
	return fmt.Errorf("not implemented")
}

// ======================= CACHE STATISTICS HANDLER =======================

type CacheStatsHandler struct {
	cacheManager interface{}
	logger       interface{}
}

type CacheStatsDTO struct {
	L1Stats  *TierStatsDTO    `json:"l1_redis"`
	L2Stats  *TierStatsDTO    `json:"l2_local"`
	L3Stats  *TierStatsDTO    `json:"l3_cloud"`
	Overall  *CacheMetricsDTO `json:"overall"`
	TopItems []CacheItemDTO   `json:"top_items"`
}

type TierStatsDTO struct {
	Used          int64   `json:"used_bytes"`
	Capacity      int64   `json:"capacity_bytes"`
	HitRate       float64 `json:"hit_rate"`
	EvictionCount int64   `json:"eviction_count"`
	Items         int64   `json:"item_count"`
}

type CacheMetricsDTO struct {
	TotalHits      int64   `json:"total_hits"`
	TotalMisses    int64   `json:"total_misses"`
	HitRate        float64 `json:"hit_rate"`
	BandwidthSaved int64   `json:"bandwidth_saved_bytes"`
	EvictionPolicy string  `json:"eviction_policy"`
}

type CacheItemDTO struct {
	Key         string    `json:"key"`
	Size        int64     `json:"size_bytes"`
	Tier        string    `json:"tier"`
	AccessTime  time.Time `json:"access_time"`
	AccessCount int64     `json:"access_count"`
}

func NewCacheStatsHandler() *CacheStatsHandler {
	return &CacheStatsHandler{}
}

// GetCacheStats handles GET /api/cache/stats
func (csh *CacheStatsHandler) GetCacheStats(c *gin.Context) {
	stats, err := csh.aggregateCacheStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetL1Stats handles GET /api/cache/stats/l1
func (csh *CacheStatsHandler) GetL1Stats(c *gin.Context) {
	stats, err := csh.getTierStats("l1")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetL2Stats handles GET /api/cache/stats/l2
func (csh *CacheStatsHandler) GetL2Stats(c *gin.Context) {
	stats, err := csh.getTierStats("l2")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetL3Stats handles GET /api/cache/stats/l3
func (csh *CacheStatsHandler) GetL3Stats(c *gin.Context) {
	stats, err := csh.getTierStats("l3")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// ClearCache handles DELETE /api/cache
func (csh *CacheStatsHandler) ClearCache(c *gin.Context) {
	tier := c.Query("tier") // Optional: clear specific tier

	err := csh.clearCacheByTier(tier)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "cache cleared"})
}

func (csh *CacheStatsHandler) aggregateCacheStats() (*CacheStatsDTO, error) {
	return nil, fmt.Errorf("not implemented")
}

func (csh *CacheStatsHandler) getTierStats(tier string) (*TierStatsDTO, error) {
	return nil, fmt.Errorf("not implemented")
}

func (csh *CacheStatsHandler) clearCacheByTier(tier string) error {
	return fmt.Errorf("not implemented")
}

// ======================= HEALTH STATUS HANDLER =======================

type HealthStatusHandler struct {
	healthChecker interface{}
	logger        interface{}
}

type RepositoryHealthDTO struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Status         string    `json:"status"` // "healthy", "degraded", "offline"
	LastCheck      time.Time `json:"last_check"`
	ResponseTime   int64     `json:"response_time_ms"`
	ErrorCount     int       `json:"error_count"`
	SuccessCount   int       `json:"success_count"`
	Uptime         float64   `json:"uptime_percent"`
	DowntimeEvents int       `json:"downtime_events"`
}

type HealthCheckResponseDTO struct {
	Timestamp     time.Time             `json:"timestamp"`
	Repositories  []RepositoryHealthDTO `json:"repositories"`
	OverallStatus string                `json:"overall_status"`
	HealthyCount  int                   `json:"healthy_count"`
	DegradedCount int                   `json:"degraded_count"`
	OfflineCount  int                   `json:"offline_count"`
}

func NewHealthStatusHandler() *HealthStatusHandler {
	return &HealthStatusHandler{}
}

// GetHealthStatus handles GET /api/health
func (hsh *HealthStatusHandler) GetHealthStatus(c *gin.Context) {
	detailed := c.Query("detailed") == "true"

	health, err := hsh.getHealthCheckResults(detailed)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, health)
}

// GetRepositoryHealth handles GET /api/health/:repoID
func (hsh *HealthStatusHandler) GetRepositoryHealth(c *gin.Context) {
	repoID := c.Param("repoID")
	if repoID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "repository ID required"})
		return
	}

	health, err := hsh.getRepositoryHealthStatus(repoID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	c.JSON(http.StatusOK, health)
}

// TriggerHealthCheck handles POST /api/health/check
func (hsh *HealthStatusHandler) TriggerHealthCheck(c *gin.Context) {
	repoID := c.Query("repository")

	err := hsh.triggerHealthCheckForRepository(repoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "health check triggered"})
}

func (hsh *HealthStatusHandler) getHealthCheckResults(detailed bool) (*HealthCheckResponseDTO, error) {
	return nil, fmt.Errorf("not implemented")
}

func (hsh *HealthStatusHandler) getRepositoryHealthStatus(repoID string) (*RepositoryHealthDTO, error) {
	return nil, fmt.Errorf("not implemented")
}

func (hsh *HealthStatusHandler) triggerHealthCheckForRepository(repoID string) error {
	return fmt.Errorf("not implemented")
}

// ======================= METRICS & ANALYTICS HANDLER (DEPRECATED - Use metrics_handler.go) =======================

type ProxyMetricsHandler struct {
	metricsCollector interface{}
	logger           interface{}
}

type MetricsDTO struct {
	Period           string               `json:"period"`
	StartTime        time.Time            `json:"start_time"`
	EndTime          time.Time            `json:"end_time"`
	RequestStats     *RequestStatsDTO     `json:"request_stats"`
	PerformanceStats *PerformanceStatsDTO `json:"performance_stats"`
	SecurityStats    *SecurityStatsDTO    `json:"security_stats"`
	TopArtifacts     []ArtifactStatsDTO   `json:"top_artifacts"`
}

type RequestStatsDTO struct {
	TotalRequests   int64            `json:"total_requests"`
	SuccessfulCount int64            `json:"successful_count"`
	FailedCount     int64            `json:"failed_count"`
	ByProtocol      map[string]int64 `json:"by_protocol"`
	ByRepository    map[string]int64 `json:"by_repository"`
}

type PerformanceStatsDTO struct {
	AvgResponseTime int64 `json:"avg_response_time_ms"`
	MaxResponseTime int64 `json:"max_response_time_ms"`
	MinResponseTime int64 `json:"min_response_time_ms"`
	P95ResponseTime int64 `json:"p95_response_time_ms"`
	P99ResponseTime int64 `json:"p99_response_time_ms"`
	BandwidthUsed   int64 `json:"bandwidth_used_bytes"`
	BandwidthSaved  int64 `json:"bandwidth_saved_bytes"`
}

type SecurityStatsDTO struct {
	VulnerabilitiesDetected int64 `json:"vulnerabilities_detected"`
	ArtifactsScanned        int64 `json:"artifacts_scanned"`
	ArtifactsBlocked        int64 `json:"artifacts_blocked"`
	LicenseIssues           int64 `json:"license_issues"`
	PolicyViolations        int64 `json:"policy_violations"`
}

type ArtifactStatsDTO struct {
	Name             string    `json:"name"`
	Downloads        int64     `json:"downloads"`
	AverageSize      int64     `json:"average_size_bytes"`
	LastAccessedTime time.Time `json:"last_accessed"`
}

func NewProxyMetricsHandler() *ProxyMetricsHandler {
	return &ProxyMetricsHandler{}
}

// GetMetrics handles GET /api/metrics
func (mh *ProxyMetricsHandler) GetMetrics(c *gin.Context) {
	period := c.DefaultQuery("period", "24h")

	metrics, err := mh.getMetricsForPeriod(period)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

// GetMetricsByRepository handles GET /api/metrics/repository/:repoID
func (mh *ProxyMetricsHandler) GetMetricsByRepository(c *gin.Context) {
	repoID := c.Param("repoID")
	period := c.DefaultQuery("period", "24h")

	metrics, err := mh.getMetricsForRepository(repoID, period)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

func (mh *ProxyMetricsHandler) getMetricsForPeriod(period string) (*MetricsDTO, error) {
	return nil, fmt.Errorf("not implemented")
}

func (mh *ProxyMetricsHandler) getMetricsForRepository(repoID string, period string) (*MetricsDTO, error) {
	return nil, fmt.Errorf("not implemented")
}

// ======================= SECURITY SCAN RESULTS HANDLER =======================

type SecurityScanHandler struct {
	scanPipeline interface{}
	logger       interface{}
}

type ScanResultDTO struct {
	ArtifactID            string             `json:"artifact_id"`
	RepositoryID          string             `json:"repository_id"`
	Path                  string             `json:"path"`
	ScanTime              time.Time          `json:"scan_time"`
	Vulnerabilities       []VulnerabilityDTO `json:"vulnerabilities"`
	LicenseIssues         []LicenseDTO       `json:"license_issues"`
	PolicyRecommendations []string           `json:"policy_recommendations"`
	OverallRisk           string             `json:"overall_risk"` // "CRITICAL", "HIGH", "MEDIUM", "LOW"
	Blocked               bool               `json:"blocked"`
	BlockReason           string             `json:"block_reason"`
}

type VulnerabilityDTO struct {
	ID          string  `json:"id"`
	Severity    string  `json:"severity"`
	Description string  `json:"description"`
	CVSS        float64 `json:"cvss"`
	Reference   string  `json:"reference"`
	Source      string  `json:"source"` // "owasp", "grype", "nvd"
}

type LicenseDTO struct {
	Package       string `json:"package"`
	License       string `json:"license"`
	Compatibility string `json:"compatibility"` // "compatible", "incompatible", "unknown"
	Issue         string `json:"issue"`
}

func NewSecurityScanHandler() *SecurityScanHandler {
	return &SecurityScanHandler{}
}

// GetScanResults handles GET /api/security/scan/:artifactID
func (ssh *SecurityScanHandler) GetScanResults(c *gin.Context) {
	artifactID := c.Param("artifactID")
	if artifactID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "artifact ID required"})
		return
	}

	results, err := ssh.getScanResultsForArtifact(artifactID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "scan results not found"})
		return
	}

	c.JSON(http.StatusOK, results)
}

// ListBlockedArtifacts handles GET /api/security/blocked
func (ssh *SecurityScanHandler) ListBlockedArtifacts(c *gin.Context) {
	repoID := c.Query("repository")
	severity := c.Query("severity") // Optional filter

	blocked, err := ssh.getBlockedArtifacts(repoID, severity)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count":     len(blocked),
		"artifacts": blocked,
	})
}

// RescanArtifact handles POST /api/security/rescan/:artifactID
func (ssh *SecurityScanHandler) RescanArtifact(c *gin.Context) {
	artifactID := c.Param("artifactID")
	if artifactID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "artifact ID required"})
		return
	}

	err := ssh.triggerRescan(artifactID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"message": "rescan triggered"})
}

func (ssh *SecurityScanHandler) getScanResultsForArtifact(artifactID string) (*ScanResultDTO, error) {
	return nil, fmt.Errorf("not implemented")
}

func (ssh *SecurityScanHandler) getBlockedArtifacts(repoID string, severity string) ([]ScanResultDTO, error) {
	return nil, fmt.Errorf("not implemented")
}

func (ssh *SecurityScanHandler) triggerRescan(artifactID string) error {
	return fmt.Errorf("not implemented")
}

// ======================= TYPE DEFINITIONS =======================

type ProxyRequest struct {
	RepositoryID string
	Path         string
	Protocol     string
	Context      interface{}
	ClientIP     string
	RequestID    string
}

type ProxyResponse struct {
	Stream      io.ReadCloser
	Size        int64
	ContentType string
	Checksum    string
	CacheSource string
	CacheHit    bool
	Duration    time.Duration
	StatusCode  int
}

type ArtifactMetadata struct {
	Size        int64
	Checksum    string
	ContentType string
	Modified    time.Time
}
