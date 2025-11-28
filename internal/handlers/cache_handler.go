package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/securestor/securestor/internal/cache"
	"github.com/securestor/securestor/internal/scanner"
)

// CacheHandler handles cache management endpoints
type CacheHandler struct {
	cacheManager   *cache.MultiTierCacheManager
	scannerManager *scanner.ScannerManager
	db             *sql.DB
}

// NewCacheHandler creates a new cache handler
func NewCacheHandler(cacheManager *cache.MultiTierCacheManager, scannerManager *scanner.ScannerManager, db *sql.DB) *CacheHandler {
	return &CacheHandler{
		cacheManager:   cacheManager,
		scannerManager: scannerManager,
		db:             db,
	}
}

// RegisterRoutes registers cache management routes
func (h *CacheHandler) RegisterRoutes(r *gin.RouterGroup) {
	cache := r.Group("/cache")
	{
		cache.GET("/stats", h.GetCacheStats)
		cache.GET("/items", h.GetCacheItems)
		cache.DELETE("/items/:id", h.DeleteCacheItem)
		cache.POST("/items/:id/scan", h.ScanCacheItem)
		cache.GET("/items/:id/download", h.DownloadCacheItem)
		cache.POST("/flush", h.FlushCache)
		cache.POST("/evict", h.EvictUnusedCache)
		cache.POST("/cleanup", h.CleanupExpiredCache)
	}
}

// GetCacheStats returns cache statistics from both Redis (L1) and Database tracking
func (h *CacheHandler) GetCacheStats(c *gin.Context) {
	// Get tenant ID from Gin context
	tenantIDVal, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tenantID, ok := tenantIDVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid tenant context"})
		return
	}

	// Get Redis/Disk cache stats from multi-tier cache manager
	l1Stats := h.cacheManager.GetCacheStats(c.Request.Context())
	if l1Stats == nil {
		l1Stats = make(map[string]interface{})
	}

	// Get database-tracked cache statistics
	var dbStats struct {
		TotalItems       int
		TotalSizeBytes   int64
		L1Items          int
		L2Items          int
		L3Items          int
		PendingScans     int
		QuarantinedItems int
		AvgHitCount      float64
	}

	query := `
		SELECT 
			COUNT(*) as total_items,
			COALESCE(SUM(size_bytes), 0) as total_size_bytes,
			COUNT(*) FILTER (WHERE cache_level = 'L1') as l1_items,
			COUNT(*) FILTER (WHERE cache_level = 'L2') as l2_items,
			COUNT(*) FILTER (WHERE cache_level = 'L3') as l3_items,
			COUNT(*) FILTER (WHERE scan_status = 'pending' OR scan_status = 'queued') as pending_scans,
			COUNT(*) FILTER (WHERE is_quarantined = true) as quarantined_items,
			COALESCE(AVG(hit_count), 0) as avg_hit_count
		FROM cached_artifacts
		WHERE tenant_id = $1
	`

	err := h.db.QueryRowContext(c.Request.Context(), query, tenantID).Scan(
		&dbStats.TotalItems,
		&dbStats.TotalSizeBytes,
		&dbStats.L1Items,
		&dbStats.L2Items,
		&dbStats.L3Items,
		&dbStats.PendingScans,
		&dbStats.QuarantinedItems,
		&dbStats.AvgHitCount,
	)

	if err != nil {
		log.Printf("[CACHE] Failed to get database cache stats: %v", err)
		// Continue with L1 stats only
	}

	// Combine stats from both sources
	response := gin.H{
		"l1_cache": gin.H{
			"redis":       l1Stats,
			"description": "L1 Cache (Redis) - stores binaries temporarily for fast access",
		},
		"database_tracking": gin.H{
			"total_items":       dbStats.TotalItems,
			"total_size_bytes":  dbStats.TotalSizeBytes,
			"l1_items":          dbStats.L1Items,
			"l2_items":          dbStats.L2Items,
			"l3_items":          dbStats.L3Items,
			"pending_scans":     dbStats.PendingScans,
			"quarantined_items": dbStats.QuarantinedItems,
			"avg_hit_count":     dbStats.AvgHitCount,
			"description":       "Database tracks metadata, checksums, and pointers to cached binaries",
		},
		"storage_model": gin.H{
			"l1":       "Redis - In-memory cache for hot data (deduplicated by checksum)",
			"l2":       "Disk - Local filesystem cache for warm data",
			"l3":       "Cloud/S3 - Optional cold storage",
			"database": "PostgreSQL - Metadata, pointers, upstream URLs, and tracking",
		},
	}

	c.JSON(http.StatusOK, response)
}

// GetCacheItems returns paginated list of cached items from database
func (h *CacheHandler) GetCacheItems(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	cacheType := c.Query("type")
	search := c.Query("search")
	sort := c.DefaultQuery("sort", "last_accessed")
	order := c.DefaultQuery("order", "desc")

	// Get tenant ID from Gin context (set by auth middleware)
	tenantIDVal, exists := c.Get("tenant_id")
	if !exists {
		log.Printf("[CACHE] tenant_id not found in Gin context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tenantID, ok := tenantIDVal.(uuid.UUID)
	if !ok {
		log.Printf("[CACHE] tenant_id has wrong type: %T", tenantIDVal)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid tenant context"})
		return
	}

	// Calculate offset from page number
	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	// Build query based on filters
	queryBase := `
		SELECT 
			id, artifact_path, artifact_type, artifact_name, artifact_version,
			cache_level, cache_key, size_bytes, checksum, hit_count,
			last_accessed, created_at, expiry_at, scan_status, metadata
		FROM cached_artifacts
		WHERE tenant_id = $1
	`
	countQueryBase := `SELECT COUNT(*) FROM cached_artifacts WHERE tenant_id = $1`

	args := []interface{}{tenantID}
	argIndex := 2

	// Add type filter
	if cacheType != "" && cacheType != "all" {
		queryBase += fmt.Sprintf(" AND artifact_type = $%d", argIndex)
		countQueryBase += fmt.Sprintf(" AND artifact_type = $%d", argIndex)
		args = append(args, cacheType)
		argIndex++
	}

	// Add search filter
	if search != "" {
		queryBase += fmt.Sprintf(" AND (artifact_path ILIKE $%d OR artifact_name ILIKE $%d)", argIndex, argIndex)
		countQueryBase += fmt.Sprintf(" AND (artifact_path ILIKE $%d OR artifact_name ILIKE $%d)", argIndex, argIndex)
		args = append(args, "%"+search+"%")
		argIndex++
	}

	// Add sorting
	validSorts := map[string]bool{
		"last_accessed": true,
		"created_at":    true,
		"size_bytes":    true,
		"hit_count":     true,
		"artifact_name": true,
	}
	if !validSorts[sort] {
		sort = "last_accessed"
	}

	validOrders := map[string]bool{"asc": true, "desc": true}
	if !validOrders[order] {
		order = "desc"
	}

	queryBase += fmt.Sprintf(" ORDER BY %s %s LIMIT $%d OFFSET $%d", sort, order, argIndex, argIndex+1)

	// Get total count
	var total int
	var err error
	err = h.db.QueryRowContext(c.Request.Context(), countQueryBase, args...).Scan(&total)
	if err != nil {
		log.Printf("[CACHE] Failed to count cached items: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count cache items"})
		return
	}

	// Get paginated items
	args = append(args, limit, offset)
	rows, err := h.db.QueryContext(c.Request.Context(), queryBase, args...)
	if err != nil {
		log.Printf("[CACHE] Failed to query cached items: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list cache items"})
		return
	}
	defer rows.Close()

	items := []map[string]interface{}{}
	for rows.Next() {
		var (
			id              string
			artifactPath    string
			artifactType    string
			artifactName    sql.NullString
			artifactVersion sql.NullString
			cacheLevel      string
			cacheKey        string
			sizeBytes       int64
			checksum        string
			hitCount        int
			lastAccessed    sql.NullTime
			createdAt       sql.NullTime
			expiryAt        sql.NullTime
			scanStatus      string
			metadata        sql.NullString
		)

		err := rows.Scan(
			&id, &artifactPath, &artifactType, &artifactName, &artifactVersion,
			&cacheLevel, &cacheKey, &sizeBytes, &checksum, &hitCount,
			&lastAccessed, &createdAt, &expiryAt, &scanStatus, &metadata,
		)
		if err != nil {
			log.Printf("[CACHE] Failed to scan row: %v", err)
			continue
		}

		item := map[string]interface{}{
			"id":               id,
			"key":              cacheKey,
			"artifact_path":    artifactPath,
			"artifact_type":    artifactType,
			"artifact_name":    artifactName.String,
			"artifact_version": artifactVersion.String,
			"cache_level":      cacheLevel,
			"size_bytes":       sizeBytes,
			"checksum":         checksum,
			"hit_count":        hitCount,
			"scan_status":      scanStatus,
		}

		if lastAccessed.Valid {
			item["last_accessed"] = lastAccessed.Time
		}
		if createdAt.Valid {
			item["cached_at"] = createdAt.Time
		}
		if expiryAt.Valid {
			item["expiry_at"] = expiryAt.Time
		}
		if metadata.Valid {
			item["metadata"] = metadata.String
		}

		items = append(items, item)
	}

	// Calculate total pages
	totalPages := (total + limit - 1) / limit
	if totalPages < 0 {
		totalPages = 0
	}

	response := gin.H{
		"items": items,
		"total": total,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
		},
		"filters": gin.H{
			"type":   cacheType,
			"search": search,
			"sort":   sort,
			"order":  order,
		},
	}

	c.JSON(http.StatusOK, response)
}

// DeleteCacheItem deletes a specific cache item
func (h *CacheHandler) DeleteCacheItem(c *gin.Context) {
	itemID := c.Param("id")

	// Invalidate cache for this item
	err := h.cacheManager.Invalidate(c.Request.Context(), itemID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete cache item: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Cache item deleted successfully",
		"id":      itemID,
	})
}

// ScanCacheItem triggers a security scan for a cached item
func (h *CacheHandler) ScanCacheItem(c *gin.Context) {
	itemID := c.Param("id")

	// Extract tenant_id from Gin context (set by auth middleware)
	tenantIDInterface, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Tenant ID not found in context",
		})
		return
	}

	tenantID, ok := tenantIDInterface.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid tenant ID format",
		})
		return
	}

	// Parse item ID as UUID
	artifactUUID, err := uuid.Parse(itemID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid item ID format",
		})
		return
	}

	// Query database for cached artifact details
	var artifact struct {
		ID           uuid.UUID
		ArtifactPath string
		ArtifactType string
		CacheLevel   string
		Checksum     string
		ScanStatus   string
		TenantID     uuid.UUID
	}

	query := `
		SELECT id, artifact_path, artifact_type, cache_level, checksum, scan_status, tenant_id
		FROM cached_artifacts
		WHERE id = $1 AND tenant_id = $2
	`
	err = h.db.QueryRow(query, artifactUUID, tenantID).Scan(
		&artifact.ID,
		&artifact.ArtifactPath,
		&artifact.ArtifactType,
		&artifact.CacheLevel,
		&artifact.Checksum,
		&artifact.ScanStatus,
		&artifact.TenantID,
	)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Cache item not found or access denied",
		})
		return
	}
	if err != nil {
		log.Printf("Error querying cached artifact: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve cache item",
		})
		return
	}

	// Check if scan is already in progress
	if artifact.ScanStatus == "scanning" {
		c.JSON(http.StatusConflict, gin.H{
			"error":   "Scan already in progress",
			"id":      itemID,
			"status":  "scanning",
			"message": "A scan is already running for this artifact",
		})
		return
	}

	// Create scan queue entry
	scanQueueID := uuid.New()
	insertQueueQuery := `
		INSERT INTO scan_queue (
			id, cached_artifact_id, artifact_path, artifact_type, 
			priority, status, tenant_id, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		RETURNING id
	`
	err = h.db.QueryRow(
		insertQueueQuery,
		scanQueueID,
		artifact.ID,
		artifact.ArtifactPath,
		artifact.ArtifactType,
		75, // High priority for manual scans
		"queued",
		tenantID,
	).Scan(&scanQueueID)
	if err != nil {
		log.Printf("Error creating scan queue entry: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to queue scan",
		})
		return
	}

	// Update cached_artifacts scan_status to 'scanning'
	updateStatusQuery := `
		UPDATE cached_artifacts 
		SET scan_status = 'scanning', 
		    metadata = jsonb_set(
		        COALESCE(metadata, '{}'::jsonb), 
		        '{last_scan_triggered}', 
		        to_jsonb(NOW()::text)
		    )
		WHERE id = $1
	`
	_, err = h.db.Exec(updateStatusQuery, artifact.ID)
	if err != nil {
		log.Printf("Error updating scan status: %v", err)
		// Continue anyway - scan queue entry is created
	}

	// Trigger async scan in background
	go h.performScan(context.Background(), scanQueueID, artifact.ID, artifact.ArtifactPath, artifact.ArtifactType, tenantID)

	// Return immediate response
	c.JSON(http.StatusAccepted, gin.H{
		"message":       "Scan triggered successfully",
		"id":            itemID,
		"scan_id":       scanQueueID.String(),
		"status":        "scanning",
		"artifact_type": artifact.ArtifactType,
	})
}

// performScan executes the actual security scan asynchronously
func (h *CacheHandler) performScan(ctx context.Context, scanQueueID, cachedArtifactID uuid.UUID, artifactPath, artifactType string, tenantID uuid.UUID) {
	log.Printf("[SCAN] Starting scan for cached artifact: %s (type: %s, tenant: %s)", cachedArtifactID, artifactType, tenantID)

	// Get artifact_id from cached_artifacts (may be null)
	var artifactID *uuid.UUID
	err := h.db.QueryRow(`SELECT artifact_id FROM cached_artifacts WHERE id = $1`, cachedArtifactID).Scan(&artifactID)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("[SCAN] Error getting artifact_id: %v", err)
	}

	// Update scan queue to 'processing'
	updateQueueQuery := `
		UPDATE scan_queue 
		SET status = 'processing', started_at = NOW()
		WHERE id = $1
	`
	_, err = h.db.Exec(updateQueueQuery, scanQueueID)
	if err != nil {
		log.Printf("[SCAN] Error updating scan queue status: %v", err)
	}

	startTime := time.Now()

	// Execute scan using ScannerManager
	scanResults, err := h.scannerManager.ScanWithAll(ctx, artifactPath, artifactType)
	if err != nil {
		log.Printf("[SCAN] Scan failed for cached artifact %s: %v", cachedArtifactID, err)

		// Mark scan as failed in queue
		h.db.Exec(`
			UPDATE scan_queue 
			SET status = 'failed', completed_at = NOW()
			WHERE id = $1
		`, scanQueueID)

		// Update cached_artifacts status
		h.db.Exec(`
			UPDATE cached_artifacts 
			SET scan_status = 'failed'
			WHERE id = $1
		`, cachedArtifactID)
		return
	}

	duration := int(time.Since(startTime).Seconds())

	// Aggregate scan results
	aggregated := h.scannerManager.AggregateScanResults(scanResults)
	if aggregated == nil {
		log.Printf("[SCAN] No scan results for cached artifact %s", cachedArtifactID)

		// Mark as completed with no vulnerabilities
		h.db.Exec(`
			UPDATE scan_queue 
			SET status = 'completed', completed_at = NOW()
			WHERE id = $1
		`, scanQueueID)

		h.db.Exec(`
			UPDATE cached_artifacts 
			SET scan_status = 'completed', 
			    last_scan_at = NOW(),
			    next_scan_at = NOW() + INTERVAL '7 days',
			    vulnerabilities_count = 0,
			    critical_vulnerabilities = 0,
			    high_vulnerabilities = 0,
			    medium_vulnerabilities = 0,
			    low_vulnerabilities = 0,
			    is_quarantined = false
			WHERE id = $1
		`, cachedArtifactID)
		return
	}

	// Count vulnerabilities by severity
	vulnCounts := make(map[string]int)
	var hasCritical bool
	for _, vuln := range aggregated.Vulnerabilities {
		vulnCounts[vuln.Severity]++
		if vuln.Severity == "CRITICAL" {
			hasCritical = true
		}
	}

	// Prepare metadata with artifact information for display
	metadata := map[string]interface{}{
		"artifact_path": artifactPath,
		"artifact_name": artifactPath, // Use path as name for cache items
		"artifact_type": artifactType,
		"source":        "cache",
		"cache_key":     fmt.Sprintf("%s:%s", artifactType, artifactPath),
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		log.Printf("[SCAN] Error marshaling metadata: %v", err)
		metadataJSON = []byte("{}")
	}

	// Create security_scans entry with metadata
	securityScanID := uuid.New()
	insertSecurityScanQuery := `
		INSERT INTO security_scans (
			scan_id, tenant_id, artifact_id, status, scan_type, 
			priority, vulnerability_scan, started_at, completed_at, duration, metadata
		) VALUES ($1, $2, $3, 'completed', $4, 'high', true, $5, NOW(), $6, $7)
	`
	_, err = h.db.Exec(
		insertSecurityScanQuery,
		securityScanID,
		tenantID,
		artifactID, // May be NULL
		artifactType,
		startTime,
		duration,
		string(metadataJSON),
	)
	if err != nil {
		log.Printf("[SCAN] Error creating security_scans entry: %v", err)
		// Continue anyway - try to save results
	}

	// Determine risk level
	riskLevel := "low"
	if hasCritical {
		riskLevel = "critical"
	} else if vulnCounts["HIGH"] > 0 {
		riskLevel = "high"
	} else if vulnCounts["MEDIUM"] > 0 {
		riskLevel = "medium"
	}

	// Prepare vulnerability results JSONB
	vulnerabilityResults := map[string]interface{}{
		"total":    len(aggregated.Vulnerabilities),
		"critical": vulnCounts["CRITICAL"],
		"high":     vulnCounts["HIGH"],
		"medium":   vulnCounts["MEDIUM"],
		"low":      vulnCounts["LOW"],
		"scanners": aggregated.Metadata,
		"details":  aggregated.Vulnerabilities,
	}

	// Marshal to JSON
	vulnerabilityResultsJSON, err := json.Marshal(vulnerabilityResults)
	if err != nil {
		log.Printf("[SCAN] Error marshaling vulnerability results: %v", err)
		vulnerabilityResultsJSON = []byte("{}")
	}

	// Insert scan_results entry
	scanResultID := uuid.New()
	insertResultQuery := `
		INSERT INTO scan_results (
			result_id, tenant_id, scan_id, overall_score, risk_level,
			summary, vulnerability_results
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	// Calculate overall score (0-100, lower is worse)
	overallScore := 100
	if hasCritical {
		overallScore = 20
	} else if vulnCounts["HIGH"] > 0 {
		overallScore = 50
	} else if vulnCounts["MEDIUM"] > 0 {
		overallScore = 70
	} else if vulnCounts["LOW"] > 0 {
		overallScore = 85
	}

	summary := fmt.Sprintf("Found %d vulnerabilities: %d critical, %d high, %d medium, %d low",
		len(aggregated.Vulnerabilities),
		vulnCounts["CRITICAL"],
		vulnCounts["HIGH"],
		vulnCounts["MEDIUM"],
		vulnCounts["LOW"])

	_, err = h.db.Exec(
		insertResultQuery,
		scanResultID,
		tenantID,
		securityScanID,
		overallScore,
		riskLevel,
		summary,
		string(vulnerabilityResultsJSON),
	)
	if err != nil {
		log.Printf("[SCAN] Error saving scan results: %v", err)
	} else {
		log.Printf("[SCAN] Scan results saved: result_id=%s, risk=%s, score=%d", scanResultID, riskLevel, overallScore)
	}

	// Update cached_artifacts with scan results
	updateArtifactQuery := `
		UPDATE cached_artifacts 
		SET scan_status = 'completed',
		    scan_results_id = $2,
		    last_scan_at = NOW(),
		    next_scan_at = NOW() + INTERVAL '7 days',
		    vulnerabilities_count = $3,
		    critical_vulnerabilities = $4,
		    high_vulnerabilities = $5,
		    medium_vulnerabilities = $6,
		    low_vulnerabilities = $7,
		    is_quarantined = $8
		WHERE id = $1
	`
	_, err = h.db.Exec(
		updateArtifactQuery,
		cachedArtifactID,
		scanResultID,
		len(aggregated.Vulnerabilities),
		vulnCounts["CRITICAL"],
		vulnCounts["HIGH"],
		vulnCounts["MEDIUM"],
		vulnCounts["LOW"],
		hasCritical,
	)
	if err != nil {
		log.Printf("[SCAN] Error updating cached artifact with results: %v", err)
	}

	// Mark scan queue as completed
	h.db.Exec(`
		UPDATE scan_queue 
		SET status = 'completed', completed_at = NOW()
		WHERE id = $1
	`, scanQueueID)

	log.Printf("[SCAN] Scan completed for cached artifact %s: %d vulnerabilities found (quarantined: %v, risk: %s)",
		cachedArtifactID, len(aggregated.Vulnerabilities), hasCritical, riskLevel)
}

// DownloadCacheItem downloads a cached artifact
func (h *CacheHandler) DownloadCacheItem(c *gin.Context) {
	itemID := c.Param("id")

	// Get artifact from cache
	artifact, source, found := h.cacheManager.Get(c.Request.Context(), itemID)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Cache item not found",
		})
		return
	}

	// Set response headers
	c.Header("Content-Type", artifact.ContentType)
	c.Header("Content-Length", strconv.FormatInt(artifact.Size, 10))
	c.Header("X-Cache-Source", string(source))
	c.Header("X-Cache-Checksum", artifact.Checksum)
	c.Header("Content-Disposition", "attachment; filename="+itemID)

	// Stream the data
	c.Data(http.StatusOK, artifact.ContentType, artifact.Data)
}

// FlushCache flushes all cache entries
func (h *CacheHandler) FlushCache(c *gin.Context) {
	ctx := c.Request.Context()

	// Get optional cache level parameter (l1, l2, l3, or all)
	level := c.Query("level")
	if level == "" {
		level = "all"
	}

	// Flush based on level
	var err error
	switch level {
	case "l1":
		err = h.cacheManager.FlushL1(ctx)
	case "l2":
		err = h.cacheManager.FlushL2(ctx)
	case "l3":
		err = h.cacheManager.FlushL3(ctx)
	case "all":
		// Flush all levels
		if err := h.cacheManager.FlushL1(ctx); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to flush L1 cache: " + err.Error(),
			})
			return
		}
		if err := h.cacheManager.FlushL2(ctx); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to flush L2 cache: " + err.Error(),
			})
			return
		}
		// L3 flush is optional, don't fail if not available
		_ = h.cacheManager.FlushL3(ctx)
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid cache level. Use 'l1', 'l2', 'l3', or 'all'",
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to flush cache: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Cache flushed successfully",
		"level":   level,
	})
}

// EvictUnusedCache removes cached items that haven't been accessed in X days
func (h *CacheHandler) EvictUnusedCache(c *gin.Context) {
	// Get tenant ID from Gin context
	tenantIDVal, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tenantID, ok := tenantIDVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid tenant context"})
		return
	}

	// Get days parameter (default: 30 days)
	days := 30
	if daysParam := c.Query("days"); daysParam != "" {
		if d, err := strconv.Atoi(daysParam); err == nil && d > 0 {
			days = d
		}
	}

	// Get minimum hit count (default: 0 - evict everything unused)
	minHits := 0
	if minHitsParam := c.Query("min_hits"); minHitsParam != "" {
		if h, err := strconv.Atoi(minHitsParam); err == nil && h >= 0 {
			minHits = h
		}
	}

	// Delete from database (which will cascade to related tables)
	query := `
		DELETE FROM cached_artifacts
		WHERE tenant_id = $1
		AND (
			last_accessed < NOW() - INTERVAL '1 day' * $2
			OR (last_accessed IS NULL AND created_at < NOW() - INTERVAL '1 day' * $2)
		)
		AND hit_count <= $3
		AND is_quarantined = false
	`

	result, err := h.db.ExecContext(c.Request.Context(), query, tenantID, days, minHits)
	if err != nil {
		log.Printf("[CACHE] Failed to evict unused cache: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to evict cache"})
		return
	}

	rowsAffected, _ := result.RowsAffected()

	c.JSON(http.StatusOK, gin.H{
		"message":       "Cache eviction completed",
		"items_removed": rowsAffected,
		"criteria": gin.H{
			"unused_days": days,
			"max_hits":    minHits,
		},
	})
}

// CleanupExpiredCache removes expired cached items based on expiry_at timestamp
func (h *CacheHandler) CleanupExpiredCache(c *gin.Context) {
	// Get tenant ID from Gin context
	tenantIDVal, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tenantID, ok := tenantIDVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid tenant context"})
		return
	}

	// Delete expired items from database
	query := `
		DELETE FROM cached_artifacts
		WHERE tenant_id = $1
		AND expiry_at < NOW()
		AND is_quarantined = false
	`

	result, err := h.db.ExecContext(c.Request.Context(), query, tenantID)
	if err != nil {
		log.Printf("[CACHE] Failed to cleanup expired cache: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cleanup cache"})
		return
	}

	rowsAffected, _ := result.RowsAffected()

	c.JSON(http.StatusOK, gin.H{
		"message":       "Expired cache cleanup completed",
		"items_removed": rowsAffected,
	})
}
