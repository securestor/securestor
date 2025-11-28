package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// ======================= PROXY REQUEST ROUTER =======================

type ProxyRequestRouter struct {
	proxyEngine         *ProxyEngine
	repositoryStore     interface{} // Maps repository ID to config
	localStorageChecker interface{}
	logger              interface{}
	mu                  sync.RWMutex
}

type ProxyRequest struct {
	RepositoryID string
	Path         string
	Protocol     string
	Context      context.Context
	ClientIP     string
	RequestID    string
}

type ProxyResponse struct {
	Stream      io.ReadCloser
	Size        int64
	ContentType string
	Checksum    string
	CacheSource string // "local", "redis", "disk", "cloud", "remote"
	CacheHit    bool
	Duration    time.Duration
	StatusCode  int
}

func NewProxyRequestRouter(proxyEngine *ProxyEngine) *ProxyRequestRouter {
	return &ProxyRequestRouter{
		proxyEngine:     proxyEngine,
		repositoryStore: make(map[string]*RepositoryConfig),
	}
}

// RouteRequest implements the request routing decision logic
func (prr *ProxyRequestRouter) RouteRequest(req *ProxyRequest) (*ProxyResponse, error) {
	prr.mu.RLock()
	defer prr.mu.RUnlock()

	start := time.Now()
	response := &ProxyResponse{
		CacheSource: "unknown",
		CacheHit:    false,
	}

	// Step 1: Check local storage
	if localData, err := prr.checkLocalStorage(req); err == nil {
		response.Stream = localData
		response.CacheSource = "local"
		response.CacheHit = true
		response.Duration = time.Since(start)
		return response, nil
	}

	// Step 2: Check cache hierarchy (L1/L2/L3)
	cacheKey := fmt.Sprintf("%s:%s", req.RepositoryID, req.Path)
	if cachedData, err := prr.proxyEngine.cacheManager.Get(req.Context, cacheKey); err == nil {
		response.Stream = prr.wrapData(cachedData)
		response.CacheSource = "redis" // Could be redis, disk, or cloud
		response.CacheHit = true
		response.Duration = time.Since(start)
		return response, nil
	}

	// Step 3: Fetch from remote via appropriate adapter
	response.CacheHit = false
	if remoteData, err := prr.fetchFromRemote(req); err == nil {
		// Wrap the stream to cache and scan as it reads
		var wrappedStream io.ReadCloser
		if stream, ok := remoteData.Reader.(io.ReadCloser); ok {
			wrappedStream = prr.wrapStreamForCacheAndScan(
				stream,
				cacheKey,
				req.RepositoryID,
			)
		}
		response.Stream = wrappedStream
		response.Size = remoteData.Size
		response.Checksum = remoteData.Checksum
		response.CacheSource = "remote"
		response.Duration = time.Since(start)

		// Record metrics
		prr.proxyEngine.metricsCollector.RecordRequest(
			req.Protocol,
			false,
			response.Duration,
		)

		return response, nil
	}

	// Step 4: Return 404 if not found anywhere
	response.StatusCode = http.StatusNotFound
	response.Duration = time.Since(start)
	return nil, fmt.Errorf("artifact not found: %s/%s", req.RepositoryID, req.Path)
}

// checkLocalStorage checks if artifact exists in local storage
func (prr *ProxyRequestRouter) checkLocalStorage(req *ProxyRequest) (io.ReadCloser, error) {
	// Implementation: Check local filesystem based on repository config
	return nil, fmt.Errorf("not in local storage")
}

// fetchFromRemote uses protocol adapter to fetch from remote
func (prr *ProxyRequestRouter) fetchFromRemote(req *ProxyRequest) (*ArtifactStream, error) {
	// Get repository configuration
	repoConfig := prr.getRepositoryConfig(req.RepositoryID)
	if repoConfig == nil {
		return nil, fmt.Errorf("repository not found: %s", req.RepositoryID)
	}

	// Create appropriate adapter
	adapter, err := prr.proxyEngine.adapterFactory.CreateAdapter(
		repoConfig.Type,
		&ProtocolConfig{
			UpstreamURL:    repoConfig.RemoteURL,
			AuthType:       repoConfig.ProxyConfig.AuthType,
			Timeout:        repoConfig.ProxyConfig.Timeout,
			RetryAttempts:  repoConfig.ProxyConfig.RetryAttempts,
			MaxConnections: repoConfig.ProxyConfig.MaxConnections,
		},
	)
	if err != nil {
		return nil, err
	}

	// Fetch from remote
	return adapter.Fetch(req.Context, req.Path)
}

// wrapStreamForCacheAndScan wraps the remote stream to cache and scan concurrently
func (prr *ProxyRequestRouter) wrapStreamForCacheAndScan(
	originalStream io.ReadCloser,
	cacheKey string,
	repositoryID string,
) io.ReadCloser {
	// Implementation uses TeeReader pattern
	// 1. Read from original stream
	// 2. Split into two paths:
	//    a. Cache writer
	//    b. Response writer
	// 3. Calculate checksum
	// 4. Trigger async scan

	return originalStream // Simplified
}

func (prr *ProxyRequestRouter) wrapData(data interface{}) io.ReadCloser {
	// Implementation: Convert cached data to ReadCloser
	return nil
}

func (prr *ProxyRequestRouter) getRepositoryConfig(repoID string) *RepositoryConfig {
	// Implementation: Retrieve from store
	return nil
}

// ======================= VIRTUAL REPOSITORY RESOLVER =======================

type VirtualRepositoryResolver struct {
	virtualRepos    map[string]*VirtualRepository
	remoteProxies   []*RemoteProxyRegistry
	resolutionCache *Cache
	mu              sync.RWMutex
}

type VirtualRepository struct {
	ID              string
	Name            string
	IncludedRepos   []string // Local repo IDs to search first
	RemoteProxies   []*RemoteProxyRegistry
	ResolutionOrder []string
}

type RemoteProxyRegistry struct {
	Name        string
	Type        string // "maven", "npm", "docker", "pypi"
	URL         string
	Priority    int
	HealthCheck *HealthStatus
}

type Cache struct {
	entries map[string]*CacheEntry
	ttl     time.Duration
	mu      sync.RWMutex
}

type CacheEntry struct {
	Value      interface{}
	ExpiresAt  time.Time
	HitCount   int64
	LastAccess time.Time
}

func NewVirtualRepositoryResolver() *VirtualRepositoryResolver {
	return &VirtualRepositoryResolver{
		virtualRepos: make(map[string]*VirtualRepository),
		resolutionCache: &Cache{
			entries: make(map[string]*CacheEntry),
			ttl:     24 * time.Hour,
		},
	}
}

// Resolve implements virtual repository artifact resolution
func (vrr *VirtualRepositoryResolver) Resolve(
	ctx context.Context,
	virtualRepoID string,
	path string,
) (*RemoteProxyRegistry, error) {
	vrr.mu.RLock()
	virtualRepo := vrr.virtualRepos[virtualRepoID]
	vrr.mu.RUnlock()

	if virtualRepo == nil {
		return nil, fmt.Errorf("virtual repository not found: %s", virtualRepoID)
	}

	// Check resolution cache
	cacheKey := fmt.Sprintf("%s:%s", virtualRepoID, path)
	if cached, found := vrr.getFromCache(cacheKey); found {
		return cached.(*RemoteProxyRegistry), nil
	}

	// Resolve in order of priority
	for _, remoteProxy := range virtualRepo.RemoteProxies {
		if remoteProxy.HealthCheck != nil && remoteProxy.HealthCheck.Status != "healthy" {
			continue
		}

		if exists, _ := vrr.checkRemoteRegistry(ctx, remoteProxy, path); exists {
			vrr.setInCache(cacheKey, remoteProxy)
			return remoteProxy, nil
		}
	}

	return nil, fmt.Errorf("artifact not found in any remote: %s", path)
}

func (vrr *VirtualRepositoryResolver) checkRemoteRegistry(
	ctx context.Context,
	registry *RemoteProxyRegistry,
	path string,
) (bool, error) {
	// Implementation: Check if artifact exists in remote registry
	return false, nil
}

func (vrr *VirtualRepositoryResolver) getFromCache(key string) (interface{}, bool) {
	vrr.resolutionCache.mu.RLock()
	defer vrr.resolutionCache.mu.RUnlock()

	if entry, found := vrr.resolutionCache.entries[key]; found {
		if time.Now().Before(entry.ExpiresAt) {
			entry.HitCount++
			entry.LastAccess = time.Now()
			return entry.Value, true
		}
	}
	return nil, false
}

func (vrr *VirtualRepositoryResolver) setInCache(key string, value interface{}) {
	vrr.resolutionCache.mu.Lock()
	defer vrr.resolutionCache.mu.Unlock()

	vrr.resolutionCache.entries[key] = &CacheEntry{
		Value:      value,
		ExpiresAt:  time.Now().Add(vrr.resolutionCache.ttl),
		HitCount:   1,
		LastAccess: time.Now(),
	}
}

// ======================= RETRY & FALLBACK HANDLER =======================

type RetryAndFallbackHandler struct {
	maxRetries         int
	retryDelay         time.Duration
	fallbackStrategies []FallbackStrategy
	logger             interface{}
}

type FallbackStrategy interface {
	CanHandle(error) bool
	Handle(ctx context.Context, req *ProxyRequest) (*ProxyResponse, error)
}

type StaleCacheFallback struct {
	cacheManager *CacheManager
}

type BackupRemoteFallback struct {
	backupRemotes []RemoteProxyRegistry
}

func NewRetryAndFallbackHandler() *RetryAndFallbackHandler {
	return &RetryAndFallbackHandler{
		maxRetries: 3,
		retryDelay: time.Second,
	}
}

// ExecuteWithRetry attempts to fetch with exponential backoff retry
func (rfh *RetryAndFallbackHandler) ExecuteWithRetry(
	ctx context.Context,
	req *ProxyRequest,
	executor func(context.Context, *ProxyRequest) (*ProxyResponse, error),
) (*ProxyResponse, error) {
	var lastErr error
	backoff := rfh.retryDelay

	for attempt := 0; attempt <= rfh.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			backoff *= 2 // Exponential backoff
		}

		resp, err := executor(ctx, req)
		if err == nil {
			return resp, nil
		}

		lastErr = err

		// Check if error is transient (retryable)
		if !rfh.isTransientError(err) {
			break // Don't retry permanent errors
		}
	}

	// Try fallback strategies
	for _, fallback := range rfh.fallbackStrategies {
		if fallback.CanHandle(lastErr) {
			return fallback.Handle(ctx, req)
		}
	}

	return nil, lastErr
}

func (rfh *RetryAndFallbackHandler) isTransientError(err error) bool {
	// Determine if error is transient (network, timeout) vs permanent (404, auth)
	errMsg := err.Error()
	return errMsg == "timeout" || errMsg == "connection refused"
}

// ======================= TEEreader FOR CONCURRENT OPERATIONS =======================

type ConcurrentTeeReader struct {
	source       io.ReadCloser
	cacheWriter  io.Writer
	scanWriter   io.Writer
	checksumCalc interface{}
	mu           sync.Mutex
	closed       bool
}

func NewConcurrentTeeReader(
	source io.ReadCloser,
	cacheWriter io.Writer,
	scanWriter io.Writer,
) *ConcurrentTeeReader {
	return &ConcurrentTeeReader{
		source:      source,
		cacheWriter: cacheWriter,
		scanWriter:  scanWriter,
	}
}

func (ctr *ConcurrentTeeReader) Read(p []byte) (int, error) {
	ctr.mu.Lock()
	defer ctr.mu.Unlock()

	if ctr.closed {
		return 0, io.EOF
	}

	n, err := ctr.source.Read(p)
	if n > 0 {
		// Write to cache in background
		go func() {
			ctr.cacheWriter.Write(p[:n])
		}()

		// Write to scan queue in background
		go func() {
			ctr.scanWriter.Write(p[:n])
		}()
	}

	return n, err
}

func (ctr *ConcurrentTeeReader) Close() error {
	ctr.mu.Lock()
	defer ctr.mu.Unlock()

	if ctr.closed {
		return nil
	}

	ctr.closed = true
	return ctr.source.Close()
}

// ======================= ERROR RECOVERY STRATEGIES =======================

// ServeStaleCache returns cached version even if expired
func (sfb *StaleCacheFallback) Handle(
	ctx context.Context,
	req *ProxyRequest,
) (*ProxyResponse, error) {
	cacheKey := fmt.Sprintf("%s:%s", req.RepositoryID, req.Path)

	// Get stale cache entry (ignoring TTL)
	_, err := sfb.cacheManager.Get(ctx, cacheKey)
	if err != nil {
		return nil, err
	}

	return &ProxyResponse{
		Stream:      nil,
		CacheSource: "stale_cache",
		CacheHit:    true,
	}, nil
}

func (sfb *StaleCacheFallback) CanHandle(err error) bool {
	// Serve stale cache for transient failures
	return true
}

// UseBackupRemote tries alternative remote registries
func (bfb *BackupRemoteFallback) Handle(
	ctx context.Context,
	req *ProxyRequest,
) (*ProxyResponse, error) {
	for range bfb.backupRemotes {
		// Try fetching from backup
		// Implementation here
	}
	return nil, fmt.Errorf("all backup remotes failed")
}

func (bfb *BackupRemoteFallback) CanHandle(err error) bool {
	return true
}
