package middleware

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ======================= PROXY AUTHENTICATION MIDDLEWARE =======================

type ProxyAuthMiddleware struct {
	apiKeyStore  map[string]*ApiKeyInfo
	oauthClient  interface{}
	jwtVerifier  interface{}
	allowedRepos map[string]bool
	logger       interface{}
	mu           sync.RWMutex
}

type ApiKeyInfo struct {
	ID          string
	Secret      string
	Permissions []string
	ExpiresAt   time.Time
	LastUsed    time.Time
	RateLimit   int // requests per minute
}

func NewProxyAuthMiddleware() *ProxyAuthMiddleware {
	return &ProxyAuthMiddleware{
		apiKeyStore:  make(map[string]*ApiKeyInfo),
		allowedRepos: make(map[string]bool),
	}
}

// AuthorizeProxyAccess implements authorization check
func (pam *ProxyAuthMiddleware) AuthorizeProxyAccess() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract credentials from request
		authHeader := c.GetHeader("Authorization")
		apiKey := c.GetHeader("X-API-Key")

		if authHeader == "" && apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing credentials"})
			c.Abort()
			return
		}

		var principal *Principal
		var err error

		// Try API Key auth first
		if apiKey != "" {
			principal, err = pam.validateApiKey(apiKey)
		} else if strings.HasPrefix(authHeader, "Bearer ") {
			// Try JWT auth
			token := strings.TrimPrefix(authHeader, "Bearer ")
			principal, err = pam.validateJWT(token)
		} else if strings.HasPrefix(authHeader, "Basic ") {
			// Try Basic auth
			principal, err = pam.validateBasicAuth(authHeader)
		}

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication failed"})
			c.Abort()
			return
		}

		// Check if user has permission for the requested repository
		repoID := c.Param("repoID")
		if !pam.hasRepositoryAccess(principal, repoID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied to repository"})
			c.Abort()
			return
		}

		// Store principal in context
		c.Set("principal", principal)
		c.Set("userID", principal.UserID)
		c.Set("requestID", pam.generateRequestID())

		c.Next()
	}
}

type Principal struct {
	UserID      string
	Email       string
	Permissions []string
	ExpiresAt   time.Time
}

func (pam *ProxyAuthMiddleware) validateApiKey(apiKey string) (*Principal, error) {
	pam.mu.RLock()
	keyInfo, found := pam.apiKeyStore[apiKey]
	pam.mu.RUnlock()

	if !found {
		return nil, fmt.Errorf("invalid API key")
	}

	if time.Now().After(keyInfo.ExpiresAt) {
		return nil, fmt.Errorf("API key expired")
	}

	return &Principal{
		UserID:      keyInfo.ID,
		Permissions: keyInfo.Permissions,
		ExpiresAt:   keyInfo.ExpiresAt,
	}, nil
}

func (pam *ProxyAuthMiddleware) validateJWT(token string) (*Principal, error) {
	// Implementation: Validate JWT token using jwtVerifier
	return nil, fmt.Errorf("JWT validation not implemented")
}

func (pam *ProxyAuthMiddleware) validateBasicAuth(authHeader string) (*Principal, error) {
	// Implementation: Validate Basic auth credentials
	return nil, fmt.Errorf("Basic auth validation not implemented")
}

func (pam *ProxyAuthMiddleware) hasRepositoryAccess(principal *Principal, repoID string) bool {
	// Check if principal has read permission for repository
	for _, perm := range principal.Permissions {
		if strings.HasPrefix(perm, "repo:"+repoID) {
			return true
		}
		if perm == "repo:*" {
			return true
		}
	}
	return false
}

func (pam *ProxyAuthMiddleware) generateRequestID() string {
	return fmt.Sprintf("req-%d", time.Now().UnixNano())
}

// ======================= PROXY RATE LIMITING MIDDLEWARE =======================

type ProxyRateLimitMiddleware struct {
	limiters map[string]*ProxyRateLimiter
	mu       sync.RWMutex
}

type ProxyRateLimiter struct {
	maxRequests int
	windowSize  time.Duration
	requests    []time.Time
	mu          sync.Mutex
}

func NewProxyRateLimitMiddleware() *ProxyRateLimitMiddleware {
	return &ProxyRateLimitMiddleware{
		limiters: make(map[string]*ProxyRateLimiter),
	}
}

// ApplyRateLimit implements rate limiting per user/API key
func (rlm *ProxyRateLimitMiddleware) ApplyRateLimit(maxRequests int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("userID")
		if userID == "" {
			userID = c.ClientIP()
		}

		limiter := rlm.getOrCreateLimiter(userID, maxRequests, window)

		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded",
				"retry_after": window.Seconds(),
			})
			c.Abort()
			return
		}

		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", maxRequests))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", maxRequests-len(limiter.requests)))
		c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(window).Unix()))

		c.Next()
	}
}

func (rlm *ProxyRateLimitMiddleware) getOrCreateLimiter(
	key string,
	maxRequests int,
	window time.Duration,
) *ProxyRateLimiter {
	rlm.mu.RLock()
	limiter, found := rlm.limiters[key]
	rlm.mu.RUnlock()

	if found {
		return limiter
	}

	rlm.mu.Lock()
	limiter = &ProxyRateLimiter{
		maxRequests: maxRequests,
		windowSize:  window,
		requests:    make([]time.Time, 0),
	}
	rlm.limiters[key] = limiter
	rlm.mu.Unlock()

	return limiter
}

func (rl *ProxyRateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.windowSize)

	// Remove old requests outside the window
	validRequests := make([]time.Time, 0)
	for _, reqTime := range rl.requests {
		if reqTime.After(windowStart) {
			validRequests = append(validRequests, reqTime)
		}
	}
	rl.requests = validRequests

	// Check if under limit
	if len(rl.requests) < rl.maxRequests {
		rl.requests = append(rl.requests, now)
		return true
	}

	return false
}

// ======================= REQUEST LOGGING & TRACING MIDDLEWARE =======================

type ProxyRequestTracingMiddleware struct {
	logger interface{}
}

type RequestTrace struct {
	RequestID    string
	Method       string
	Path         string
	Status       int
	StartTime    time.Time
	Duration     time.Duration
	BytesRead    int64
	BytesWritten int64
	CacheHit     bool
	Error        string
}

func NewProxyRequestTracingMiddleware() *ProxyRequestTracingMiddleware {
	return &ProxyRequestTracingMiddleware{}
}

// TraceRequest implements request tracing
func (prtm *ProxyRequestTracingMiddleware) TraceRequest() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetString("requestID")
		if requestID == "" {
			requestID = prtm.generateTraceID()
		}

		trace := &RequestTrace{
			RequestID: requestID,
			Method:    c.Request.Method,
			Path:      c.Request.URL.Path,
			StartTime: time.Now(),
		}

		// Inject trace ID into response headers
		c.Header("X-Trace-ID", requestID)
		c.Header("X-Request-ID", requestID)

		defer func() {
			trace.Duration = time.Since(trace.StartTime)
			prtm.logTrace(trace)
		}()

		c.Next()
	}
}

func (prtm *ProxyRequestTracingMiddleware) logTrace(trace *RequestTrace) {
	// Implementation: Log trace information
	fmt.Printf("[TRACE] %s %s %s %d %dms\n",
		trace.RequestID,
		trace.Method,
		trace.Path,
		trace.Status,
		trace.Duration.Milliseconds(),
	)
}

func (prtm *ProxyRequestTracingMiddleware) generateTraceID() string {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().UnixNano())))
	return fmt.Sprintf("%x", hash[:8])
}

// ======================= CACHE CONTROL MIDDLEWARE =======================

type CacheControlMiddleware struct{}

func NewCacheControlMiddleware() *CacheControlMiddleware {
	return &CacheControlMiddleware{}
}

// SetCacheHeaders sets appropriate cache control headers
func (ccm *CacheControlMiddleware) SetCacheHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Don't cache if user doesn't have cache preference
		if c.Query("no-cache") == "true" {
			c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
			c.Header("Pragma", "no-cache")
			c.Header("Expires", "0")
		} else {
			// Default: cache for 1 hour
			c.Header("Cache-Control", "public, max-age=3600")
			c.Header("ETag", c.GetString("requestID"))
		}

		c.Next()
	}
}

// HandleConditionalRequests handles If-Modified-Since, If-None-Match
func (ccm *CacheControlMiddleware) HandleConditionalRequests() gin.HandlerFunc {
	return func(c *gin.Context) {
		ifNoneMatch := c.GetHeader("If-None-Match")
		etag := c.GetString("etag")

		if ifNoneMatch != "" && ifNoneMatch == etag {
			c.Status(http.StatusNotModified)
			c.Abort()
			return
		}

		c.Next()
	}
}

// ======================= COMPRESSION MIDDLEWARE =======================

type CompressionMiddleware struct {
	minSize int // Minimum bytes to compress
}

func NewCompressionMiddleware() *CompressionMiddleware {
	return &CompressionMiddleware{
		minSize: 1024, // 1KB minimum
	}
}

// ApplyCompression implements response compression
func (cm *CompressionMiddleware) ApplyCompression() gin.HandlerFunc {
	return func(c *gin.Context) {
		acceptEncoding := c.GetHeader("Accept-Encoding")

		// Check if client supports gzip
		if strings.Contains(acceptEncoding, "gzip") {
			c.Header("Content-Encoding", "gzip")
			// Compression handled by Gin framework
		}

		c.Next()
	}
}

// ======================= ERROR HANDLING MIDDLEWARE =======================

type ErrorHandlingMiddleware struct {
	logger interface{}
}

func NewErrorHandlingMiddleware() *ErrorHandlingMiddleware {
	return &ErrorHandlingMiddleware{}
}

// HandleErrors catches and formats errors
func (ehm *ErrorHandlingMiddleware) HandleErrors() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				ehm.handlePanic(c, err)
			}
		}()

		c.Next()

		// Check if errors occurred during processing
		if len(c.Errors) > 0 {
			lastErr := c.Errors.Last()
			ehm.handleError(c, lastErr.Err)
		}
	}
}

func (ehm *ErrorHandlingMiddleware) handleError(c *gin.Context, err error) {
	errMsg := err.Error()

	if strings.Contains(errMsg, "not found") {
		c.JSON(http.StatusNotFound, gin.H{"error": errMsg})
	} else if strings.Contains(errMsg, "unauthorized") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": errMsg})
	} else if strings.Contains(errMsg, "forbidden") {
		c.JSON(http.StatusForbidden, gin.H{"error": errMsg})
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}

func (ehm *ErrorHandlingMiddleware) handlePanic(c *gin.Context, err interface{}) {
	c.JSON(http.StatusInternalServerError, gin.H{
		"error": "internal server error",
		"panic": fmt.Sprintf("%v", err),
	})
}

// ======================= CONTENT TYPE VALIDATION MIDDLEWARE =======================

type ContentTypeValidationMiddleware struct{}

func NewContentTypeValidationMiddleware() *ContentTypeValidationMiddleware {
	return &ContentTypeValidationMiddleware{}
}

// ValidateContentType validates Content-Type for POST/PUT requests
func (ctvm *ContentTypeValidationMiddleware) ValidateContentType() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "POST" || c.Request.Method == "PUT" {
			contentType := c.GetHeader("Content-Type")

			if !strings.Contains(contentType, "application/json") {
				c.JSON(http.StatusUnsupportedMediaType, gin.H{
					"error": "Content-Type must be application/json",
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// ======================= PROXY HEADER HANDLING MIDDLEWARE =======================

type ProxyHeaderMiddleware struct{}

func NewProxyHeaderMiddleware() *ProxyHeaderMiddleware {
	return &ProxyHeaderMiddleware{}
}

// AddProxyHeaders adds security and proxy headers
func (phm *ProxyHeaderMiddleware) AddProxyHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Security headers
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// Proxy headers
		c.Header("Via", "SecureStore/1.0")
		c.Header("X-Powered-By", "SecureStore-Proxy")

		// Remove sensitive headers
		c.Header("Server", "")

		c.Next()
	}
}

// ======================= REQUEST SIZE LIMIT MIDDLEWARE =======================

type RequestSizeLimitMiddleware struct {
	maxSize int64
}

func NewRequestSizeLimitMiddleware(maxSize int64) *RequestSizeLimitMiddleware {
	return &RequestSizeLimitMiddleware{
		maxSize: maxSize,
	}
}

// LimitRequestSize limits request body size
func (rslm *RequestSizeLimitMiddleware) LimitRequestSize() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > rslm.maxSize {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": fmt.Sprintf("request body exceeds %d bytes", rslm.maxSize),
			})
			c.Abort()
			return
		}

		// Set request body limit
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, rslm.maxSize)

		c.Next()
	}
}

// ======================= CORS MIDDLEWARE =======================

type CORSMiddleware struct {
	allowedOrigins []string
	allowedMethods []string
}

func NewCORSMiddleware() *CORSMiddleware {
	return &CORSMiddleware{
		allowedOrigins: []string{"*"},
		allowedMethods: []string{"GET", "HEAD", "POST", "PUT", "DELETE", "OPTIONS"},
	}
}

// HandleCORS handles CORS requests
func (cm *CORSMiddleware) HandleCORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// Check if origin is allowed
		allowed := false
		for _, allowedOrigin := range cm.allowedOrigins {
			if allowedOrigin == "*" || allowedOrigin == origin {
				allowed = true
				break
			}
		}

		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", strings.Join(cm.allowedMethods, ", "))
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key, X-Tenant-ID, X-Tenant-Slug")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Max-Age", "3600")
		}

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.Status(http.StatusOK)
			return
		}

		c.Next()
	}
}

// ======================= COPY REQUEST BODY MIDDLEWARE =======================

type CopyRequestBodyMiddleware struct{}

func NewCopyRequestBodyMiddleware() *CopyRequestBodyMiddleware {
	return &CopyRequestBodyMiddleware{}
}

// CopyBody enables reading request body multiple times
func (crbm *CopyRequestBodyMiddleware) CopyBody() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Copy body for later use if needed
		var bodyBytes []byte
		if c.Request.Body != nil {
			bodyBytes, _ = io.ReadAll(c.Request.Body)
		}

		// Restore body for processing
		c.Request.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

		// Store body in context
		c.Set("requestBody", string(bodyBytes))

		c.Next()
	}
}
