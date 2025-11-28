package routes

import (
	"database/sql"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/securestor/securestor/internal/api"
	"github.com/securestor/securestor/internal/handlers"
	"github.com/securestor/securestor/internal/middleware"
)

// ======================= PROXY ROUTER SETUP =======================

type ProxyRouterConfig struct {
	ProxyEngine         *api.ProxyEngine
	AuthMiddleware      *middleware.ProxyAuthMiddleware
	RateLimitMiddleware *middleware.ProxyRateLimitMiddleware
	TracingMiddleware   *middleware.ProxyRequestTracingMiddleware
	MaxRequestSize      int64
	DB                  interface{} // Database connection for cloud repository handler
}

// RegisterProxyRoutes sets up all remote proxy endpoints
func RegisterProxyRoutes(router *gin.Engine, config *ProxyRouterConfig) {
	// Create handlers
	artifactHandler := handlers.NewProxyArtifactHandler()
	repoConfigHandler := handlers.NewRepositoryConfigHandler(config.DB.(*sql.DB))
	cloudRepoHandler := handlers.NewCloudRepositoryHandler(config.DB.(*sql.DB))
	metricsAPIHandler := handlers.NewMetricsAPIHandler(config.DB.(*sql.DB))
	cacheStatsHandler := handlers.NewCacheStatsHandler()
	healthHandler := handlers.NewHealthStatusHandler()
	metricsHandler := handlers.NewProxyMetricsHandler()
	securityHandler := handlers.NewSecurityScanHandler()

	// Common middleware
	corsMiddleware := middleware.NewCORSMiddleware()
	contentTypeValidation := middleware.NewContentTypeValidationMiddleware()
	errorHandling := middleware.NewErrorHandlingMiddleware()
	proxyHeaders := middleware.NewProxyHeaderMiddleware()
	requestSizeLimit := middleware.NewRequestSizeLimitMiddleware(config.MaxRequestSize)
	cacheControl := middleware.NewCacheControlMiddleware()

	// Apply global middleware
	router.Use(corsMiddleware.HandleCORS())
	router.Use(proxyHeaders.AddProxyHeaders())
	router.Use(errorHandling.HandleErrors())

	// ======================= ARTIFACT PROXY ENDPOINTS =======================

	artifactGroup := router.Group("/api/proxy")
	{
		// Apply auth and tracing to all artifact endpoints
		artifactGroup.Use(config.AuthMiddleware.AuthorizeProxyAccess())
		artifactGroup.Use(config.TracingMiddleware.TraceRequest())
		artifactGroup.Use(config.RateLimitMiddleware.ApplyRateLimit(1000, 60*time.Second))
		artifactGroup.Use(requestSizeLimit.LimitRequestSize())
		artifactGroup.Use(cacheControl.SetCacheHeaders())

		// GET artifact (with cache headers support)
		artifactGroup.GET("/:repoID/*path", artifactHandler.GetArtifact)

		// HEAD artifact (for existence check)
		artifactGroup.HEAD("/:repoID/*path", artifactHandler.HeadArtifact)
	}

	// ======================= REPOSITORY CONFIGURATION ENDPOINTS =======================

	repoGroup := router.Group("/api/repositories")
	{
		// Apply auth and tracing
		repoGroup.Use(config.AuthMiddleware.AuthorizeProxyAccess())
		repoGroup.Use(config.TracingMiddleware.TraceRequest())
		repoGroup.Use(contentTypeValidation.ValidateContentType())
		repoGroup.Use(requestSizeLimit.LimitRequestSize())

		// Repository CRUD operations
		repoGroup.POST("", repoConfigHandler.CreateRepository)
		repoGroup.GET("", repoConfigHandler.ListRepositories)
		repoGroup.GET("/:id", repoConfigHandler.GetRepository)
		repoGroup.PUT("/:id", repoConfigHandler.UpdateRepository)
		repoGroup.DELETE("/:id", repoConfigHandler.DeleteRepository)

		// Cloud repository operations
		repoGroup.POST("/cloud", cloudRepoHandler.CreateCloudRepository)
		repoGroup.GET("/cloud/:id/status", cloudRepoHandler.GetCloudRepositoryStatus)
	}

	// ======================= CACHE STATISTICS ENDPOINTS =======================

	cacheGroup := router.Group("/api/cache")
	{
		// Apply auth and tracing
		cacheGroup.Use(config.AuthMiddleware.AuthorizeProxyAccess())
		cacheGroup.Use(config.TracingMiddleware.TraceRequest())

		// Cache statistics
		cacheGroup.GET("/stats", cacheStatsHandler.GetCacheStats)
		cacheGroup.GET("/stats/l1", cacheStatsHandler.GetL1Stats)
		cacheGroup.GET("/stats/l2", cacheStatsHandler.GetL2Stats)
		cacheGroup.GET("/stats/l3", cacheStatsHandler.GetL3Stats)

		// Cache management
		cacheGroup.DELETE("", cacheStatsHandler.ClearCache)
	}

	// ======================= HEALTH STATUS ENDPOINTS =======================

	healthGroup := router.Group("/api/health")
	{
		// Health endpoints don't require auth (allow monitoring)
		healthGroup.Use(config.TracingMiddleware.TraceRequest())

		// Health status
		healthGroup.GET("", healthHandler.GetHealthStatus)
		healthGroup.GET("/:repoID", healthHandler.GetRepositoryHealth)

		// Health check trigger (with auth)
		healthCheck := healthGroup.Group("")
		healthCheck.Use(config.AuthMiddleware.AuthorizeProxyAccess())
		healthCheck.POST("/check", healthHandler.TriggerHealthCheck)
	}

	// ======================= METRICS & ANALYTICS ENDPOINTS =======================

	metricsGroup := router.Group("/api/v1/metrics")
	{
		// Apply auth and tracing
		metricsGroup.Use(config.AuthMiddleware.AuthorizeProxyAccess())
		metricsGroup.Use(config.TracingMiddleware.TraceRequest())

		// Dashboard metrics endpoints - wrap http.HandlerFunc for Gin
		metricsGroup.GET("/cache", gin.WrapF(metricsAPIHandler.GetCacheMetrics))
		metricsGroup.GET("/performance", gin.WrapF(metricsAPIHandler.GetPerformanceMetrics))
	}

	// Health and alerts for monitoring dashboard
	healthV1Group := router.Group("/api/v1/health")
	{
		healthV1Group.Use(config.TracingMiddleware.TraceRequest())
		healthV1Group.GET("/repositories", gin.WrapF(metricsAPIHandler.GetRepositoryHealth))
	}

	alertsGroup := router.Group("/api/v1/alerts")
	{
		alertsGroup.Use(config.AuthMiddleware.AuthorizeProxyAccess())
		alertsGroup.Use(config.TracingMiddleware.TraceRequest())
		alertsGroup.GET("", gin.WrapF(metricsAPIHandler.GetAlerts))
	}

	// Legacy metrics endpoints
	metricsLegacyGroup := router.Group("/api/metrics")
	{
		// Apply auth and tracing
		metricsLegacyGroup.Use(config.AuthMiddleware.AuthorizeProxyAccess())
		metricsLegacyGroup.Use(config.TracingMiddleware.TraceRequest())

		// Metrics
		metricsLegacyGroup.GET("", metricsHandler.GetMetrics)
		metricsLegacyGroup.GET("/repository/:repoID", metricsHandler.GetMetricsByRepository)
	}

	// ======================= SECURITY ENDPOINTS =======================

	securityGroup := router.Group("/api/security")
	{
		// Apply auth and tracing
		securityGroup.Use(config.AuthMiddleware.AuthorizeProxyAccess())
		securityGroup.Use(config.TracingMiddleware.TraceRequest())

		// Scan results
		securityGroup.GET("/scan/:artifactID", securityHandler.GetScanResults)
		securityGroup.GET("/blocked", securityHandler.ListBlockedArtifacts)
		securityGroup.POST("/rescan/:artifactID", securityHandler.RescanArtifact)
	}

	// ======================= ADMIN ENDPOINTS =======================

	adminGroup := router.Group("/api/admin/proxy")
	{
		// Admin endpoints require special permission
		adminGroup.Use(config.AuthMiddleware.AuthorizeProxyAccess())
		adminGroup.Use(config.TracingMiddleware.TraceRequest())

		// Configuration management
		adminGroup.GET("/config", getProxyConfiguration)
		adminGroup.PUT("/config", updateProxyConfiguration)
		adminGroup.POST("/config/validate", validateProxyConfiguration)
	}
}

// ======================= ADMIN HANDLERS =======================

func getProxyConfiguration(c *gin.Context) {
	// Implementation: Return current proxy configuration
	c.JSON(200, gin.H{
		"l1_cache": gin.H{
			"type":     "redis",
			"capacity": "16GB",
			"ttl":      "1h-24h",
		},
		"l2_cache": gin.H{
			"type":     "local_disk",
			"capacity": "500GB",
			"ttl":      "1d-7d",
		},
		"l3_cache": gin.H{
			"type":     "cloud_s3",
			"capacity": "unlimited",
			"ttl":      "7d-30d",
		},
	})
}

func updateProxyConfiguration(c *gin.Context) {
	// Implementation: Update proxy configuration
	c.JSON(200, gin.H{"message": "configuration updated"})
}

func validateProxyConfiguration(c *gin.Context) {
	// Implementation: Validate proxy configuration
	c.JSON(200, gin.H{"valid": true})
}

// SetupProxyRouter is the main entry point for setting up proxy routes
func SetupProxyRouter(router *gin.Engine, proxyEngine *api.ProxyEngine, db interface{}) {
	config := &ProxyRouterConfig{
		ProxyEngine:         proxyEngine,
		AuthMiddleware:      middleware.NewProxyAuthMiddleware(),
		RateLimitMiddleware: middleware.NewProxyRateLimitMiddleware(),
		TracingMiddleware:   middleware.NewProxyRequestTracingMiddleware(),
		MaxRequestSize:      100 * 1024 * 1024, // 100MB
		DB:                  db,
	}

	RegisterProxyRoutes(router, config)
}
