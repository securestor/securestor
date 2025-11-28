package cache

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

const (
	// Cache tier size thresholds
	L1MaxSize = 10 * 1024 * 1024       // 10MB - Redis
	L2MaxSize = 1 * 1024 * 1024 * 1024 // 1GB - Disk
	// L3 is unlimited (Cloud/S3)
)

// CacheSource indicates where the artifact was retrieved from
type CacheSource string

const (
	CacheSourceRedis  CacheSource = "redis"
	CacheSourceDisk   CacheSource = "disk"
	CacheSourceCloud  CacheSource = "cloud"
	CacheSourceRemote CacheSource = "remote"
)

// CachedArtifact represents an artifact in cache with metadata
type CachedArtifact struct {
	Data        []byte
	Size        int64
	ContentType string
	Checksum    string
	CachedAt    time.Time
	Source      CacheSource
}

// CacheItem represents a cache item with metadata for listing
type CacheItem struct {
	Key          string      `json:"key"`
	ArtifactPath string      `json:"artifact_path"`
	ArtifactType string      `json:"artifact_type"`
	CacheLevel   CacheSource `json:"cache_level"`
	SizeBytes    int64       `json:"size_bytes"`
	ContentType  string      `json:"content_type"`
	Checksum     string      `json:"checksum"`
	HitCount     int         `json:"hit_count"`
	LastAccessed time.Time   `json:"last_accessed"`
	CachedAt     time.Time   `json:"cached_at"`
	TTLRemaining int64       `json:"ttl_remaining"` // seconds
}

// MultiTierCacheManager manages L1 (Redis), L2 (Disk), and L3 (Cloud) caches
type MultiTierCacheManager struct {
	redis      *RedisClient
	diskCache  *DiskCache
	cloudCache *CloudCache
	logger     *log.Logger
}

// DiskCache handles local filesystem caching
type DiskCache struct {
	basePath string
	maxSize  int64
	logger   *log.Logger
}

// CloudCache handles S3/Cloud storage caching (placeholder for future implementation)
type CloudCache struct {
	enabled bool
	logger  *log.Logger
}

// NewMultiTierCacheManager creates a new multi-tier cache manager
func NewMultiTierCacheManager(redis *RedisClient, diskPath string, logger *log.Logger) (*MultiTierCacheManager, error) {
	if logger == nil {
		logger = log.New(os.Stdout, "[CACHE] ", log.LstdFlags)
	}

	// Initialize disk cache
	diskCache := &DiskCache{
		basePath: filepath.Join(diskPath, "remote-cache"),
		maxSize:  100 * 1024 * 1024 * 1024, // 100GB total disk cache
		logger:   logger,
	}

	// Create disk cache directory
	if err := os.MkdirAll(diskCache.basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create disk cache directory: %w", err)
	}

	// Initialize cloud cache (placeholder)
	cloudCache := &CloudCache{
		enabled: false, // Enable when S3 is configured
		logger:  logger,
	}

	return &MultiTierCacheManager{
		redis:      redis,
		diskCache:  diskCache,
		cloudCache: cloudCache,
		logger:     logger,
	}, nil
}

// Get retrieves an artifact from cache, checking all tiers
func (m *MultiTierCacheManager) Get(ctx context.Context, key string) (*CachedArtifact, CacheSource, bool) {
	// Check L1 (Redis) first
	if artifact, found := m.getFromL1(ctx, key); found {
		m.logger.Printf("Cache HIT: L1 (Redis) for key: %s", key)
		return artifact, CacheSourceRedis, true
	}

	// Check L2 (Disk)
	if artifact, found := m.getFromL2(key); found {
		m.logger.Printf("Cache HIT: L2 (Disk) for key: %s", key)
		// Async promote to L1 if small enough
		if artifact.Size < L1MaxSize {
			go m.promoteToL1(ctx, key, artifact)
		}
		return artifact, CacheSourceDisk, true
	}

	// Check L3 (Cloud) - if enabled
	if m.cloudCache.enabled {
		if artifact, found := m.getFromL3(key); found {
			m.logger.Printf("Cache HIT: L3 (Cloud) for key: %s", key)
			// Async promote to L2 and potentially L1
			go m.promoteToL2(key, artifact)
			return artifact, CacheSourceCloud, true
		}
	}

	m.logger.Printf("Cache MISS: All tiers for key: %s", key)
	return nil, "", false
}

// Set stores an artifact in the appropriate cache tier based on size
func (m *MultiTierCacheManager) Set(ctx context.Context, key string, data []byte, contentType, checksum string, ttl time.Duration) error {
	size := int64(len(data))

	artifact := &CachedArtifact{
		Data:        data,
		Size:        size,
		ContentType: contentType,
		Checksum:    checksum,
		CachedAt:    time.Now(),
	}

	// Store based on size
	if size < L1MaxSize {
		// Small artifacts go to L1 (Redis) and L2 (Disk)
		m.logger.Printf("Caching to L1 (Redis) and L2 (Disk): %s (%d bytes)", key, size)
		if err := m.setInL1(ctx, key, artifact, ttl); err != nil {
			m.logger.Printf("Failed to cache in L1: %v", err)
		}
		if err := m.setInL2(key, artifact); err != nil {
			m.logger.Printf("Failed to cache in L2: %v", err)
		}
	} else if size < L2MaxSize {
		// Medium artifacts go to L2 (Disk) only
		m.logger.Printf("Caching to L2 (Disk): %s (%d bytes)", key, size)
		if err := m.setInL2(key, artifact); err != nil {
			m.logger.Printf("Failed to cache in L2: %v", err)
			return err
		}
	} else {
		// Large artifacts go to L3 (Cloud) if enabled
		if m.cloudCache.enabled {
			m.logger.Printf("Caching to L3 (Cloud): %s (%d bytes)", key, size)
			if err := m.setInL3(key, artifact); err != nil {
				m.logger.Printf("Failed to cache in L3: %v", err)
				return err
			}
		} else {
			m.logger.Printf("Artifact too large for available cache tiers: %s (%d bytes)", key, size)
			// Still cache to L2 even if it's large
			if err := m.setInL2(key, artifact); err != nil {
				return err
			}
		}
	}

	return nil
}

// L1 (Redis) operations
func (m *MultiTierCacheManager) getFromL1(ctx context.Context, key string) (*CachedArtifact, bool) {
	cacheKey := fmt.Sprintf("remote:artifact:%s", key)
	var artifact CachedArtifact

	err := m.redis.Get(ctx, cacheKey, &artifact)
	if err != nil {
		if err != ErrKeyNotFound {
			m.logger.Printf("Redis get error: %v", err)
		}
		return nil, false
	}

	artifact.Source = CacheSourceRedis
	return &artifact, true
}

func (m *MultiTierCacheManager) setInL1(ctx context.Context, key string, artifact *CachedArtifact, ttl time.Duration) error {
	cacheKey := fmt.Sprintf("remote:artifact:%s", key)
	return m.redis.Set(ctx, cacheKey, artifact, ttl)
}

func (m *MultiTierCacheManager) promoteToL1(ctx context.Context, key string, artifact *CachedArtifact) {
	if err := m.setInL1(ctx, key, artifact, 24*time.Hour); err != nil {
		m.logger.Printf("Failed to promote to L1: %v", err)
	} else {
		m.logger.Printf("Promoted to L1 (Redis): %s", key)
	}
}

// L2 (Disk) operations
func (m *MultiTierCacheManager) getFromL2(key string) (*CachedArtifact, bool) {
	return m.diskCache.Get(key)
}

func (m *MultiTierCacheManager) setInL2(key string, artifact *CachedArtifact) error {
	return m.diskCache.Set(key, artifact)
}

func (m *MultiTierCacheManager) promoteToL2(key string, artifact *CachedArtifact) {
	if err := m.setInL2(key, artifact); err != nil {
		m.logger.Printf("Failed to promote to L2: %v", err)
	} else {
		m.logger.Printf("Promoted to L2 (Disk): %s", key)
	}
}

// L3 (Cloud) operations - placeholder
func (m *MultiTierCacheManager) getFromL3(key string) (*CachedArtifact, bool) {
	// TODO: Implement S3/Cloud storage retrieval
	return nil, false
}

func (m *MultiTierCacheManager) setInL3(key string, artifact *CachedArtifact) error {
	// TODO: Implement S3/Cloud storage
	return nil
}

// DiskCache implementation
func (dc *DiskCache) Get(key string) (*CachedArtifact, bool) {
	filePath := dc.getFilePath(key)

	data, err := os.ReadFile(filePath)
	if err != nil {
		if !os.IsNotExist(err) {
			dc.logger.Printf("Disk cache read error: %v", err)
		}
		return nil, false
	}

	// Read metadata
	metaPath := filePath + ".meta"
	metaData, err := os.ReadFile(metaPath)
	if err != nil {
		dc.logger.Printf("Disk cache metadata read error: %v", err)
	}

	artifact := &CachedArtifact{
		Data:        data,
		Size:        int64(len(data)),
		ContentType: "application/octet-stream",
		Source:      CacheSourceDisk,
		CachedAt:    time.Now(),
	}

	// Parse metadata if available
	if len(metaData) > 0 {
		// Simple format: ContentType|Checksum
		// In production, use JSON or protobuf
		artifact.ContentType = string(metaData)
	}

	return artifact, true
}

func (dc *DiskCache) Set(key string, artifact *CachedArtifact) error {
	filePath := dc.getFilePath(key)

	// Create directory if needed
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Write data
	if err := os.WriteFile(filePath, artifact.Data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	// Write metadata
	metaPath := filePath + ".meta"
	metaData := []byte(artifact.ContentType)
	if err := os.WriteFile(metaPath, metaData, 0644); err != nil {
		dc.logger.Printf("Failed to write metadata: %v", err)
	}

	return nil
}

func (dc *DiskCache) getFilePath(key string) string {
	// Use first 2 chars as subdirectory for better distribution
	if len(key) < 2 {
		return filepath.Join(dc.basePath, "default", key)
	}
	return filepath.Join(dc.basePath, key[:2], key)
}

// GetReader returns a streaming reader for the cached artifact
func (m *MultiTierCacheManager) GetReader(ctx context.Context, key string) (io.ReadCloser, CacheSource, bool) {
	artifact, source, found := m.Get(ctx, key)
	if !found {
		return nil, "", false
	}

	// Convert byte slice to io.ReadCloser
	reader := io.NopCloser(io.NewSectionReader(
		&bytesReaderAt{artifact.Data},
		0,
		artifact.Size,
	))

	return reader, source, true
}

// bytesReaderAt wraps a byte slice to implement io.ReaderAt
type bytesReaderAt struct {
	data []byte
}

func (b *bytesReaderAt) ReadAt(p []byte, off int64) (n int, err error) {
	if off >= int64(len(b.data)) {
		return 0, io.EOF
	}
	n = copy(p, b.data[off:])
	if n < len(p) {
		err = io.EOF
	}
	return
}

// Invalidate removes an artifact from all cache tiers
func (m *MultiTierCacheManager) Invalidate(ctx context.Context, key string) error {
	// Remove from L1
	cacheKey := fmt.Sprintf("remote:artifact:%s", key)
	if err := m.redis.Delete(ctx, cacheKey); err != nil {
		m.logger.Printf("Failed to invalidate L1 cache: %v", err)
	}

	// Remove from L2
	filePath := m.diskCache.getFilePath(key)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		m.logger.Printf("Failed to invalidate L2 cache: %v", err)
	}
	os.Remove(filePath + ".meta") // Best effort

	// Remove from L3 if enabled
	// TODO: Implement S3 deletion

	m.logger.Printf("Invalidated cache for key: %s", key)
	return nil
}

// GetCacheStats returns current cache statistics with actual counts
func (m *MultiTierCacheManager) GetCacheStats(ctx context.Context) map[string]interface{} {
	stats := map[string]interface{}{
		"l1_type":    "redis",
		"l2_type":    "disk",
		"l3_type":    "cloud",
		"l3_enabled": m.cloudCache.enabled,
	}

	// Count L1 (Redis) items and size
	l1Items := 0
	var l1Size int64 = 0
	redisKeys, err := m.redis.Keys(ctx, "remote:artifact:*")
	if err == nil {
		l1Items = len(redisKeys)
		// Estimate size from Redis (approximate)
		for _, key := range redisKeys {
			var artifact CachedArtifact
			if err := m.redis.Get(ctx, key, &artifact); err == nil {
				l1Size += artifact.Size
			}
		}
	}

	// Count L2 (Disk) items and size
	l2Items := 0
	var l2Size int64 = 0
	diskPattern := filepath.Join(m.diskCache.basePath, "*", "*")
	diskFiles, err := filepath.Glob(diskPattern)
	if err == nil {
		for _, diskFile := range diskFiles {
			// Skip metadata files
			if filepath.Ext(diskFile) == ".meta" {
				continue
			}
			fileInfo, err := os.Stat(diskFile)
			if err == nil && !fileInfo.IsDir() {
				l2Items++
				l2Size += fileInfo.Size()
			}
		}
	}

	// Add root level files too
	rootFiles, _ := filepath.Glob(filepath.Join(m.diskCache.basePath, "*"))
	for _, diskFile := range rootFiles {
		if filepath.Ext(diskFile) == ".meta" {
			continue
		}
		fileInfo, err := os.Stat(diskFile)
		if err == nil && !fileInfo.IsDir() {
			l2Items++
			l2Size += fileInfo.Size()
		}
	}

	stats["l1_items"] = l1Items
	stats["l1_size_bytes"] = l1Size
	stats["l2_items"] = l2Items
	stats["l2_size_bytes"] = l2Size
	stats["l3_items"] = 0
	stats["l3_size_bytes"] = 0
	stats["total_items"] = l1Items + l2Items
	stats["total_size_bytes"] = l1Size + l2Size

	return stats
}

// ListItems returns a list of cached items with metadata
func (m *MultiTierCacheManager) ListItems(ctx context.Context, offset, limit int, artifactType, search string) ([]CacheItem, int, error) {
	items := []CacheItem{}

	// Get items from Redis (L1)
	redisKeys, err := m.redis.Keys(ctx, "cache:*")
	if err != nil {
		m.logger.Printf("Warning: Failed to list Redis keys: %v", err)
	} else {
		for _, key := range redisKeys {
			// Get metadata from Redis
			metadata, err := m.redis.GetJSON(ctx, key+":meta")
			if err != nil {
				continue
			}

			// Parse artifact path from key (format: cache:type:path)
			artifactPath := key
			if len(key) > 6 { // Remove "cache:" prefix
				artifactPath = key[6:]
			}

			// Extract artifact type from path
			detectedType := "unknown"
			if len(artifactPath) > 0 {
				if filepath.Ext(artifactPath) == ".jar" || filepath.Ext(artifactPath) == ".pom" {
					detectedType = "maven"
				} else if filepath.Ext(artifactPath) == ".tgz" || filepath.Ext(artifactPath) == ".tar.gz" {
					detectedType = "npm"
				} else if filepath.Ext(artifactPath) == ".whl" || filepath.Ext(artifactPath) == ".tar.gz" {
					detectedType = "pypi"
				}
			}

			// Apply type filter
			if artifactType != "" && detectedType != artifactType {
				continue
			}

			// Apply search filter
			if search != "" && !contains(artifactPath, search) {
				continue
			}

			// Get TTL
			ttl, _ := m.redis.TTL(ctx, key)

			item := CacheItem{
				Key:          key,
				ArtifactPath: artifactPath,
				ArtifactType: detectedType,
				CacheLevel:   CacheSourceRedis,
				TTLRemaining: int64(ttl.Seconds()),
				LastAccessed: time.Now(), // Redis doesn't track last access by default
				CachedAt:     time.Now(), // Would need to store this in metadata
			}

			// Parse metadata if available
			if metaMap, ok := metadata.(map[string]interface{}); ok {
				if size, ok := metaMap["size"].(float64); ok {
					item.SizeBytes = int64(size)
				}
				if ct, ok := metaMap["content_type"].(string); ok {
					item.ContentType = ct
				}
				if cs, ok := metaMap["checksum"].(string); ok {
					item.Checksum = cs
				}
				if hc, ok := metaMap["hit_count"].(float64); ok {
					item.HitCount = int(hc)
				}
			}

			items = append(items, item)
		}
	}

	// Get items from Disk (L2)
	diskFiles, err := filepath.Glob(filepath.Join(m.diskCache.basePath, "*"))
	if err != nil {
		m.logger.Printf("Warning: Failed to list disk cache: %v", err)
	} else {
		for _, diskFile := range diskFiles {
			fileInfo, err := os.Stat(diskFile)
			if err != nil {
				continue
			}

			// Skip metadata files
			if filepath.Ext(diskFile) == ".meta" {
				continue
			}

			baseName := filepath.Base(diskFile)

			// Extract artifact type
			detectedType := "unknown"
			if filepath.Ext(baseName) == ".jar" || filepath.Ext(baseName) == ".pom" {
				detectedType = "maven"
			} else if filepath.Ext(baseName) == ".tgz" || filepath.Ext(baseName) == ".tar.gz" {
				detectedType = "npm"
			} else if filepath.Ext(baseName) == ".whl" {
				detectedType = "pypi"
			}

			// Apply filters
			if artifactType != "" && detectedType != artifactType {
				continue
			}

			if search != "" && !contains(baseName, search) {
				continue
			}

			item := CacheItem{
				Key:          baseName,
				ArtifactPath: baseName,
				ArtifactType: detectedType,
				CacheLevel:   CacheSourceDisk,
				SizeBytes:    fileInfo.Size(),
				LastAccessed: fileInfo.ModTime(),
				CachedAt:     fileInfo.ModTime(),
			}

			items = append(items, item)
		}
	}

	// Calculate total before pagination
	total := len(items)

	// Apply pagination
	if offset >= len(items) {
		return []CacheItem{}, total, nil
	}

	end := offset + limit
	if end > len(items) {
		end = len(items)
	}

	return items[offset:end], total, nil
}

// Helper function to check if string contains substring (case-insensitive)
func contains(str, substr string) bool {
	return len(str) >= len(substr) &&
		(str == substr || len(substr) == 0 ||
			matchesIgnoreCase(str, substr))
}

func matchesIgnoreCase(str, substr string) bool {
	strLower := ""
	substrLower := ""

	for _, r := range str {
		if r >= 'A' && r <= 'Z' {
			strLower += string(r + 32)
		} else {
			strLower += string(r)
		}
	}

	for _, r := range substr {
		if r >= 'A' && r <= 'Z' {
			substrLower += string(r + 32)
		} else {
			substrLower += string(r)
		}
	}

	for i := 0; i <= len(strLower)-len(substrLower); i++ {
		if strLower[i:i+len(substrLower)] == substrLower {
			return true
		}
	}

	return false
}

// FlushL1 clears all entries from L1 cache (Redis)
func (m *MultiTierCacheManager) FlushL1(ctx context.Context) error {
	if m.redis == nil {
		return nil // No L1 cache configured
	}

	// Get all cache keys
	keys, err := m.redis.Keys(ctx, "cache:*")
	if err != nil {
		return fmt.Errorf("failed to get cache keys: %w", err)
	}

	// Delete each key
	for _, key := range keys {
		if err := m.redis.Delete(ctx, key); err != nil {
			log.Printf("[WARN] Failed to delete key %s from L1: %v", key, err)
		}
	}

	log.Printf("[INFO] Flushed %d entries from L1 cache (Redis)", len(keys))
	return nil
}

// FlushL2 clears all entries from L2 cache (Disk)
func (m *MultiTierCacheManager) FlushL2(ctx context.Context) error {
	if m.diskCache == nil {
		return nil // No L2 cache configured
	}

	// Remove all files from disk cache directory
	files, err := filepath.Glob(filepath.Join(m.diskCache.basePath, "*"))
	if err != nil {
		return fmt.Errorf("failed to list disk cache files: %w", err)
	}

	deletedCount := 0
	for _, file := range files {
		if err := os.Remove(file); err != nil {
			log.Printf("[WARN] Failed to delete file %s from L2: %v", file, err)
		} else {
			deletedCount++
		}
	}

	log.Printf("[INFO] Flushed %d entries from L2 cache (Disk)", deletedCount)
	return nil
}

// FlushL3 clears all entries from L3 cache (Cloud)
func (m *MultiTierCacheManager) FlushL3(ctx context.Context) error {
	// L3 (cloud) is not yet implemented
	log.Printf("[INFO] L3 cache flush skipped (not implemented)")
	return nil
}
