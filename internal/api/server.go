package api

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/securestor/securestor/internal/cache"
	"github.com/securestor/securestor/internal/config"
	"github.com/securestor/securestor/internal/encrypt"
	"github.com/securestor/securestor/internal/handlers"
	"github.com/securestor/securestor/internal/health"
	securelogger "github.com/securestor/securestor/internal/logger"
	"github.com/securestor/securestor/internal/middleware"
	"github.com/securestor/securestor/internal/models"
	"github.com/securestor/securestor/internal/replicate"
	"github.com/securestor/securestor/internal/repository"
	"github.com/securestor/securestor/internal/scanner"
	"github.com/securestor/securestor/internal/scheduler"
	"github.com/securestor/securestor/internal/service"
	"github.com/securestor/securestor/internal/storage"
	"github.com/securestor/securestor/internal/tenant"
)

var (
	// Global unified replicator instance for enterprise-grade replication
	globalReplicator *replicate.UnifiedReplicator
)

// GetUnifiedReplicator returns the global unified replicator instance
func GetUnifiedReplicator() *replicate.UnifiedReplicator {
	return globalReplicator
}

// CreateReplicationMixin creates a replication mixin with the global replicator
func CreateReplicationMixin(logger *securelogger.Logger) *handlers.ReplicationMixin {
	if globalReplicator == nil {
		return handlers.NewReplicationMixin(nil, logger)
	}
	return handlers.NewReplicationMixin(globalReplicator, logger)
}

type Server struct {
	config                     *config.Config
	db                         *sql.DB
	ginRouter                  *gin.Engine
	artifactService            *service.ArtifactService
	repositoryService          *service.RepositoryService
	complianceService          *service.ComplianceService
	compliancePolicyService    *service.CompliancePolicyService
	scanService                *service.ScanService
	workflowService            *service.WorkflowService
	cacheService               *cache.CacheService
	blobStorage                *storage.BlobStorage
	artifactRepo               *repository.ArtifactRepository
	policyService              *PolicyService
	propertyHandler            *handlers.PropertyHandler
	authBasicHandler           *handlers.AuthBasicHandler
	artifactHandler            *handlers.ArtifactHandler
	metricsHandler             *handlers.MetricsHandler
	repositoryHandler          *handlers.RepositoryHandler
	uploadDownloadHandler      *handlers.UploadDownloadHandler
	userManagementHandler      *handlers.UserManagementHandler
	apiKeyHandler              *handlers.APIKeyManagementHandler
	roleManagementHandler      *handlers.RoleManagementHandler
	tenantManagementHandler    *handlers.TenantManagementHandler
	complianceHandler          *handlers.ComplianceHandler
	scanningHandler            *handlers.ScanningHandler
	auditLogHandler            *handlers.AuditLogHandler
	cacheHandler               *handlers.CacheHandler
	remoteProxyHandler         *handlers.RemoteProxyHandler
	encryptionAdminHandler     *handlers.EncryptionAdminHandler
	signatureHandler           *handlers.SignatureHandler
	helmHandler                *HelmHandler
	replicationSettingsHandler *ReplicationSettingsHandler
	metricsAPIHandler          *handlers.MetricsAPIHandler
	jwtAuth                    *middleware.JWTAuth
	ginJWTAuth                 *middleware.GinJWTAuth
	ginTenantMiddleware        *middleware.GinTenantMiddleware
	complianceScheduler        *scheduler.ComplianceScheduler
	auditLogService            *service.AuditLogService
	userSessionService         *service.UserSessionService
	wsHub                      *Hub
	scannerManager             *scanner.ScannerManager
	multiTierCache             *cache.MultiTierCacheManager
	tenantMiddleware           *tenant.Middleware
	tenantResolver             *tenant.TenantResolver
	tmkService                 *encrypt.TMKService
	encryptionService          *encrypt.EncryptionService
	rewrapService              *encrypt.RewrapService
	encryptedBackupService     *service.EncryptedBackupService

	logger *log.Logger
}

// policyServiceAdapter adapts PolicyService to the handlers.PolicyEvaluator interface
type policyServiceAdapter struct {
	policyService *PolicyService
}

// EvaluateArtifactPolicyFromMap implements handlers.PolicyEvaluator
func (a *policyServiceAdapter) EvaluateArtifactPolicyFromMap(ctx context.Context, artifact *models.Artifact, repo *models.Repository, userID string) (*handlers.PolicyDecisionResult, error) {
	// Use the existing CreateArtifactPolicyInput from policy_service.go
	input := CreateArtifactPolicyInput(artifact, repo, userID)

	// Call the actual policy service
	decision, err := a.policyService.EvaluateArtifactPolicy(ctx, input)
	if err != nil {
		return nil, err
	}

	// Convert to the result type expected by handlers
	return &handlers.PolicyDecisionResult{
		Allow:     decision.Allow,
		Action:    decision.Action,
		RiskScore: decision.RiskScore,
		RiskLevel: decision.RiskLevel,
		Reason:    decision.Reason,
		Timestamp: decision.Timestamp,
	}, nil
}

func NewServer(cfg *config.Config, db *sql.DB) *Server {
	// Initialize logger
	logger := log.New(log.Writer(), "[SECURESTOR] ", log.LstdFlags|log.Lshortfile)

	// Initialize unified replicator for enterprise-grade artifact replication
	logger.Printf("Initializing unified replicator...")
	replicationLogger := securelogger.NewLogger("replication")
	globalReplicator = replicate.NewUnifiedReplicator(db, replicationLogger)
	logger.Printf("✅ Unified replicator initialized successfully")

	// Initialize replication service for enterprise HA (legacy support)
	replicate.InitReplicationService(replicationLogger)

	// Initialize health checker
	logger.Printf("Initializing health checker...")
	healthLogger := &securelogger.Logger{Logger: logger}
	health.InitHealthChecker(db, healthLogger)

	// Initialize backup service for enterprise HA
	logger.Printf("Initializing backup service...")
	backupConfig := service.BackupConfig{
		ScheduleInterval: 24 * time.Hour,
		RetentionDays:    30,
		VerifyAfter:      true,
		CompressionLevel: 0,
		StoragePath:      cfg.StoragePath,
	}
	backupService := service.NewBackupService(db, backupConfig, logger)
	if err := backupService.Initialize(); err != nil {
		logger.Printf("ERROR: Failed to initialize backup service: %v", err)
	} else {
		SetBackupServiceInstance(backupService)
		// Start backup scheduler
		backupService.StartScheduler(24 * time.Hour)
	}

	// Initialize repositories
	artifactRepo := repository.NewArtifactRepository(db)
	repositoryRepo := repository.NewRepositoryRepository(db)
	complianceRepo := repository.NewComplianceRepository(db)
	vulnerabilityRepo := repository.NewVulnerabilityRepository(db)
	scanRepo := repository.NewScanRepository(db)

	// Initialize services
	artifactService := service.NewArtifactService(artifactRepo)
	repositoryService := service.NewRepositoryService(repositoryRepo)
	complianceService := service.NewComplianceService(complianceRepo, vulnerabilityRepo)

	// Initialize blob storage with erasure coding (6+3 configuration)
	blobStorage, err := storage.NewBlobStorage(storage.StorageConfig{
		BasePath:     cfg.StoragePath,
		DataShards:   6,
		ParityShards: 3,
	})
	if err != nil {
		log.Fatalf("Failed to initialize blob storage: %v", err)
	}

	// Set database for replication (create logger wrapper)
	replLogger := securelogger.NewLogger("replication")
	blobStorage.SetDatabaseAndLogger(db, replLogger)

	// Initialize TMK (Tenant Master Key) Service for encryption
	logger.Printf("Initializing TMK service with mode: %s", cfg.EncryptionMode)

	// Create appropriate KMS client based on encryption mode
	var kmsClient encrypt.KMSClient
	ctx := context.Background()

	switch cfg.EncryptionMode {
	case "mock":
		kmsClient = encrypt.NewMockKMSClient()
		logger.Printf("Using MockKMS client (development mode)")
	case "aws":
		awsClient, err := encrypt.NewAWSKMSClient(ctx, cfg.AWSRegion)
		if err != nil {
			log.Fatalf("Failed to initialize AWS KMS client: %v", err)
		}
		kmsClient = awsClient
		logger.Printf("Using AWS KMS client in region: %s", cfg.AWSRegion)
	default:
		log.Fatalf("Unsupported encryption mode: %s (supported: mock, aws)", cfg.EncryptionMode)
	}

	tmkService := encrypt.NewTMKService(db, kmsClient)
	logger.Printf("TMK service initialized successfully")

	// Initialize EncryptionService for artifact encryption/decryption
	encryptionService := encrypt.NewEncryptionService(kmsClient)
	logger.Printf("Encryption service initialized successfully")

	// Initialize RewrapService for DEK re-wrap operations
	rewrapService := encrypt.NewRewrapService(db, tmkService, kmsClient, cfg.StoragePath, encrypt.RewrapConfig{
		BatchSize:      100,
		DelayBetween:   1 * time.Second,
		MaxRetries:     3,
		RetryDelay:     5 * time.Second,
		MaxConcurrency: 5,
	})
	logger.Printf("Rewrap service initialized successfully")

	// Initialize EncryptedBackupService for encrypted backups
	encryptedBackupService := service.NewEncryptedBackupService(
		db,
		tmkService,
		kmsClient,
		service.EncryptedBackupConfig{
			StoragePath:       cfg.StoragePath + "/backups",
			RetentionDays:     90,
			CrossRegionCopy:   false,
			BackupKEKID:       "backup-kek-v1",
			VerifyAfterBackup: true,
			Logger:            logger,
		},
	)
	logger.Printf("Encrypted backup service initialized successfully")

	// Initialize scanner manager
	scannerManager := scanner.NewScannerManager()

	// Initialize scan service
	scanService := service.NewScanService(
		vulnerabilityRepo,
		complianceRepo,
		scanRepo,
		artifactRepo,
		blobStorage,
		scannerManager,
		logger,
		cfg.StoragePath+"/temp",
	)

	// Configure encryption services for secure scanning of encrypted artifacts
	// This enables ephemeral in-memory decryption during security scans
	scanService.SetEncryptionServices(encryptionService, tmkService, nil)
	logger.Printf("Scan service configured with encryption support for secure artifact scanning")

	// Initialize Redis cache
	var cacheService *cache.CacheService
	if cfg.RedisURL != "" {
		redisClient, err := cache.NewRedisClient(cache.RedisConfig{
			URL:        cfg.RedisURL,
			MaxRetries: 3,
			PoolSize:   10,
		}, logger)
		if err != nil {
			logger.Printf("Failed to initialize Redis: %v", err)
		} else {
			cacheConfig := cache.CacheConfig{
				DefaultTTL:    time.Duration(cfg.RedisCacheTTL) * time.Second,
				ArtifactTTL:   24 * time.Hour,
				ScanResultTTL: 12 * time.Hour,
				WorkflowTTL:   6 * time.Hour,
				SessionTTL:    time.Duration(cfg.RedisSessionTTL) * time.Second,
			}
			cacheService = cache.NewCacheService(redisClient, cacheConfig, logger)
			logger.Printf("Redis cache service initialized successfully")
		}
	}

	// Initialize multi-tier cache manager and remote proxy handler
	var multiTierCache *cache.MultiTierCacheManager
	logger.Printf("DEBUG: About to initialize multi-tier cache, cacheService nil? %v", cacheService == nil)
	if cacheService != nil {
		logger.Printf("DEBUG: Calling NewMultiTierCacheManager with path: %s", cfg.StoragePath)
		multiTierCache, err = cache.NewMultiTierCacheManager(
			cacheService.GetRedisClient(),
			cfg.StoragePath,
			logger,
		)
		logger.Printf("DEBUG: NewMultiTierCacheManager returned, err: %v, multiTierCache nil? %v", err, multiTierCache == nil)
		if err != nil {
			logger.Printf("Failed to initialize multi-tier cache: %v", err)
		} else {
			logger.Printf("Multi-tier cache manager initialized successfully")
		}
	} else {
		logger.Printf("DEBUG: cacheService is nil, skipping multi-tier cache initialization")
	}

	// Initialize generic workflow service
	var workflowService *service.WorkflowService
	workflowConfig, err := config.LoadWorkflowConfig("")
	if err != nil {
		logger.Printf("Failed to load workflow config: %v", err)
	} else {
		workflowService, err = service.NewWorkflowService(
			workflowConfig,
			artifactRepo,
			scanRepo,
			vulnerabilityRepo,
			blobStorage,
			logger,
			cfg.StoragePath+"/temp",
		)
		if err != nil {
			logger.Printf("Failed to initialize workflow service: %v", err)
			workflowService = nil
		} else {
			logger.Printf("Workflow service initialized successfully with %d workflows", len(workflowConfig.Workflows))
		}
	}

	// PyPI functionality is now handled through the unified artifact system

	// Initialize compliance policy service
	compliancePolicyService := service.NewCompliancePolicyService(db, logger)

	// Initialize audit service for compliance handler
	auditService := service.NewAuditService(db)

	// Initialize compliance handler
	complianceHandler := handlers.NewComplianceHandler(auditService, compliancePolicyService, db)

	// Initialize OPA policy service
	policyService := NewPolicyService(logger)

	// Create a policy evaluator adapter for the scanning handler
	policyEvaluator := &policyServiceAdapter{policyService: policyService}

	// Initialize scanning handler
	scanningHandler := handlers.NewScanningHandler(
		scanService,
		artifactService,
		repositoryService,
		policyEvaluator,
		db,
		logger,
	)

	// Initialize user management service and handler
	userManagementService := service.NewUserManagementService(db)
	passwordService := service.NewPasswordService()
	emailConfig := service.LoadEmailConfigFromEnv()
	emailService := service.NewEmailService(emailConfig)
	userManagementHandler := handlers.NewUserManagementHandler(userManagementService, passwordService, emailService)

	// Initialize role management service and handler
	roleManagementService := service.NewRoleManagementService(db)
	roleManagementHandler := handlers.NewRoleManagementHandler(roleManagementService)

	// Initialize API key management service and handler
	apiKeyService := service.NewAPIKeyManagementService(db)
	apiKeyHandler := handlers.NewAPIKeyManagementHandler(apiKeyService)

	// Initialize tenant management service and handler
	tenantManagementService := service.NewTenantManagementService(db)
	tenantManagementHandler := handlers.NewTenantManagementHandler(tenantManagementService)

	// Initialize property management service and handler
	propertyRepo := repository.NewPropertyRepository(db)
	propertyService := service.NewPropertyService(propertyRepo, nil, &securelogger.Logger{Logger: logger})
	propertyHandler := handlers.NewPropertyHandler(propertyService, &securelogger.Logger{Logger: logger})

	// Initialize JWT auth middleware
	jwtAuth := middleware.NewJWTAuth()
	ginJWTAuth := middleware.NewGinJWTAuth(jwtAuth)

	// Initialize audit logging services
	auditLogService := service.NewAuditLogService(db, logger)
	userSessionService := service.NewUserSessionService(db, logger, auditLogService)

	// Initialize audit log handler for API endpoints
	auditLogHandler := handlers.NewAuditLogHandler(auditLogService)

	// Initialize cache handler for cache management
	var cacheHandler *handlers.CacheHandler
	if multiTierCache != nil && scannerManager != nil {
		cacheHandler = handlers.NewCacheHandler(multiTierCache, scannerManager, db)
	}

	// Create replication mixin for enterprise replication across all handlers
	replicationMixin := CreateReplicationMixin(replicationLogger)
	logger.Printf("✅ Replication mixin created for unified artifact replication")

	// Initialize remote proxy handler for npm, maven, pypi, docker, helm proxying
	var remoteProxyHandler *handlers.RemoteProxyHandler
	if multiTierCache != nil && scannerManager != nil {
		remoteProxyHandler = handlers.NewRemoteProxyHandler(multiTierCache, scannerManager, db)
		remoteProxyHandler.SetReplicationMixin(replicationMixin)
		logger.Printf("✅ Remote proxy handler initialized with replication support")
	} else {
		logger.Printf("WARNING: Remote proxy handler not initialized (multiTierCache or scannerManager is nil)")
	}

	// Initialize encryption admin handler for TMK rotation, DEK rewrap, encrypted backups
	encryptionAdminHandler := handlers.NewEncryptionAdminHandler(
		tmkService,
		rewrapService,
		encryptedBackupService,
	)
	logger.Printf("Encryption admin handler initialized successfully")

	// Initialize signature handler for Cosign, PGP, Sigstore verification
	signatureHandler := handlers.NewSignatureHandler(db, &securelogger.Logger{Logger: logger})
	logger.Printf("Signature handler initialized successfully")

	// Initialize Helm handler for Helm chart repository with enterprise replication
	helmHandler := NewHelmHandler(
		filepath.Join(cfg.StoragePath, "helm"),
		artifactService,
		repositoryService,
		artifactRepo,
		replicationMixin,
	)
	logger.Printf("✅ Helm handler initialized with replication support")

	// Initialize replication settings handler for replication configuration
	replicationConfigService := service.NewReplicationConfigService(db)
	replicationSettingsHandler := NewReplicationSettingsHandler(replicationConfigService, tenantManagementService, logger)
	logger.Printf("Replication settings handler initialized successfully")

	// Initialize basic auth handler with audit logging
	authBasicHandler := handlers.NewAuthBasicHandler(userManagementService, passwordService, jwtAuth, auditLogService, userSessionService)

	// Initialize artifact handler
	artifactHandler := handlers.NewArtifactHandler(artifactService, scanService)

	// Get health checker instance for metrics
	healthChecker := health.GetInstance()

	// Initialize metrics handler
	metricsHandler := handlers.NewMetricsHandler(db, healthChecker)

	// Initialize repository handler
	repositoryHandler := handlers.NewRepositoryHandler(repositoryService, artifactService)

	// Initialize upload/download handler
	uploadDownloadConfig := &handlers.UploadDownloadConfig{
		EncryptionEnabled:   cfg.EncryptionEnabled,
		EncryptionEnforced:  cfg.EncryptionEnforced,
		EncryptionMode:      cfg.EncryptionMode,
		AWSKMSKeyID:         cfg.AWSKMSKeyID,
		ErasureDataShards:   cfg.ErasureDataShards,
		ErasureParityShards: cfg.ErasureParityShards,
	}
	uploadDownloadHandler := handlers.NewUploadDownloadHandler(
		artifactService,
		repositoryService,
		encryptionService,
		tmkService,
		policyService,
		auditLogService,
		scanService,
		blobStorage,
		uploadDownloadConfig,
	)

	// Initialize metrics API handler
	metricsAPIHandler := handlers.NewMetricsAPIHandler(db)

	// Initialize tenant resolver and middleware
	logger.Printf("Creating tenant resolver and middleware...")
	tenantResolver, err := tenant.NewTenantResolver(tenant.TenantResolverConfig{
		DB:                db,
		DefaultTenantSlug: "default", // Use 'default' tenant for localhost (no subdomain)
		CacheTTL:          5 * time.Minute,
		EnableCache:       true,
	})
	if err != nil {
		logger.Fatalf("Failed to create tenant resolver: %v", err)
	}

	tenantMiddleware := tenant.NewMiddleware(tenant.MiddlewareConfig{
		Resolver:      tenantResolver,
		BaseDomain:    "localhost", // Change to production domain in production
		HeaderName:    "X-Tenant-Slug",
		AllowNoTenant: true, // Allow requests without tenant (uses default)
	})
	logger.Printf("Tenant middleware initialized successfully")

	// Initialize compliance scheduler
	schedulerConfig := scheduler.DefaultSchedulerConfig()
	complianceScheduler := scheduler.NewComplianceScheduler(compliancePolicyService, schedulerConfig, logger)

	// Initialize WebSocket hub
	wsHub := NewHub()
	go wsHub.Run()
	logger.Printf("WebSocket hub initialized and running")

	server := &Server{
		config:                     cfg,
		db:                         db,
		ginRouter:                  gin.New(),
		artifactService:            artifactService,
		repositoryService:          repositoryService,
		complianceService:          complianceService,
		compliancePolicyService:    compliancePolicyService,
		scanService:                scanService,
		workflowService:            workflowService,
		cacheService:               cacheService,
		blobStorage:                blobStorage,
		artifactRepo:               artifactRepo,
		policyService:              policyService,
		propertyHandler:            propertyHandler,
		authBasicHandler:           authBasicHandler,
		artifactHandler:            artifactHandler,
		metricsHandler:             metricsHandler,
		repositoryHandler:          repositoryHandler,
		uploadDownloadHandler:      uploadDownloadHandler,
		userManagementHandler:      userManagementHandler,
		apiKeyHandler:              apiKeyHandler,
		roleManagementHandler:      roleManagementHandler,
		tenantManagementHandler:    tenantManagementHandler,
		complianceHandler:          complianceHandler,
		scanningHandler:            scanningHandler,
		auditLogHandler:            auditLogHandler,
		cacheHandler:               cacheHandler,
		remoteProxyHandler:         remoteProxyHandler,
		encryptionAdminHandler:     encryptionAdminHandler,
		signatureHandler:           signatureHandler,
		helmHandler:                helmHandler,
		replicationSettingsHandler: replicationSettingsHandler,
		metricsAPIHandler:          metricsAPIHandler,
		jwtAuth:                    jwtAuth,
		ginJWTAuth:                 ginJWTAuth,
		ginTenantMiddleware:        middleware.NewGinTenantMiddleware(tenantMiddleware),
		complianceScheduler:        complianceScheduler,
		auditLogService:            auditLogService,
		userSessionService:         userSessionService,
		wsHub:                      wsHub,
		scannerManager:             scannerManager,
		multiTierCache:             multiTierCache,
		tenantMiddleware:           tenantMiddleware,
		tenantResolver:             tenantResolver,
		tmkService:                 tmkService,
		encryptionService:          encryptionService,
		rewrapService:              rewrapService,
		encryptedBackupService:     encryptedBackupService,

		logger: logger,
	}

	server.setupRoutes()
	return server
}

func (s *Server) setupRoutes() {
	s.logger.Printf("Setting up Gin routes...")

	// Configure Gin mode based on environment
	if s.config.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// Apply global middleware
	s.ginRouter.Use(gin.Recovery()) // Panic recovery
	s.ginRouter.Use(gin.Logger())   // Request logging

	// Setup CORS middleware - Allow subdomain-based multi-tenant origins
	corsConfig := cors.Config{
		AllowOriginFunc: func(origin string) bool {
			s.logger.Printf("DEBUG: CORS checking origin: %s", origin)

			// Allow empty origin (same-origin requests or direct API calls)
			if origin == "" {
				s.logger.Printf("DEBUG: Allowed - empty origin (same-origin or direct)")
				return true
			}

			// Allow all localhost origins with port 3000 (React dev server)
			// This includes both localhost:3000 AND *.localhost:3000 (tenant subdomains)
			if strings.Contains(origin, "localhost:3000") {
				s.logger.Printf("DEBUG: Allowed - localhost:3000 (subdomain or direct)")
				return true
			}

			// Allow all localhost origins with port 8080
			if strings.Contains(origin, "localhost:8080") {
				s.logger.Printf("DEBUG: Allowed - localhost:8080")
				return true
			}

			// Allow localhost without port (backend)
			if origin == "http://localhost" || origin == "http://localhost:80" {
				s.logger.Printf("DEBUG: Allowed - localhost")
				return true
			}

			// Allow production subdomains (*.securestor.io)
			if strings.Contains(origin, ".securestor.io") {
				s.logger.Printf("DEBUG: Allowed - securestor.io subdomain")
				return true
			}

			s.logger.Printf("DEBUG: REJECTED origin: %s", origin)
			return false
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Content-Type", "Authorization", "Accept", "Origin", "X-Requested-With", "X-Tenant-ID", "X-Tenant-Slug"},
		ExposeHeaders:    []string{"Content-Disposition", "X-Checksum", "X-Artifact-Type"},
		AllowCredentials: true, // Enable credentials for auth
		MaxAge:           12 * time.Hour,
	}
	s.ginRouter.Use(cors.New(corsConfig))

	// Apply tenant middleware to all routes
	s.ginRouter.Use(s.ginTenantMiddleware.Handler())

	// Create auth group (public, no JWT required)
	authGroup := s.ginRouter.Group("/api/auth")
	s.logger.Printf("Registering authentication routes (public)...")
	s.authBasicHandler.RegisterRoutes(authGroup)

	// Create auth group for protected routes (with JWT)
	authProtectedGroup := s.ginRouter.Group("/api/auth")
	authProtectedGroup.Use(s.ginJWTAuth.RequireAuth())
	s.authBasicHandler.RegisterProtectedRoutes(authProtectedGroup)

	// Create API v1 group
	apiV1 := s.ginRouter.Group("/api/v1")

	// Apply JWT authentication to API routes
	apiV1.Use(s.ginJWTAuth.RequireAuth())

	// Register Property Management routes
	s.logger.Printf("Registering property management routes...")
	s.propertyHandler.RegisterRoutes(apiV1)

	// Register Upload/Download routes BEFORE Artifact Management to ensure /artifacts/:id/download takes precedence
	s.logger.Printf("Registering upload/download routes...")
	s.uploadDownloadHandler.RegisterRoutes(apiV1)

	// Register Artifact Management routes
	s.logger.Printf("Registering artifact management routes...")
	s.artifactHandler.RegisterRoutes(apiV1)

	// Register Repository Management routes
	s.logger.Printf("Registering repository management routes...")
	s.repositoryHandler.RegisterRoutes(apiV1)

	// Register User Management routes
	s.logger.Printf("Registering user management routes...")
	s.userManagementHandler.RegisterRoutes(apiV1)

	// Register API Key Management routes
	s.logger.Printf("Registering API key management routes...")
	s.apiKeyHandler.RegisterRoutes(apiV1)

	// Register Role Management routes
	s.logger.Printf("Registering role management routes...")
	s.roleManagementHandler.RegisterRoutes(apiV1)

	// Register Tenant Management routes
	s.logger.Printf("Registering tenant management routes...")
	s.tenantManagementHandler.RegisterRoutes(apiV1)

	// Register Compliance Management routes
	s.logger.Printf("Registering compliance management routes...")
	s.complianceHandler.RegisterRoutes(apiV1)

	// Register Scanning Management routes
	s.logger.Printf("Registering scanning management routes...")
	s.scanningHandler.RegisterRoutes(apiV1)

	// Register Audit Log routes
	s.logger.Printf("Registering audit log routes...")
	s.auditLogHandler.RegisterRoutes(apiV1)

	// Register Cache Management routes (under /proxy/cache)
	if s.cacheHandler != nil {
		s.logger.Printf("Registering cache management routes...")
		proxyGroup := apiV1.Group("/proxy")
		s.cacheHandler.RegisterRoutes(proxyGroup)

		// Also register directly under /cache for frontend compatibility
		s.cacheHandler.RegisterRoutes(apiV1)
	}

	// Register Remote Proxy routes (npm, maven, pypi, docker, helm)
	if s.remoteProxyHandler != nil {
		s.logger.Printf("Registering remote proxy routes (npm, maven, pypi, docker, helm)...")
		proxyGroup := apiV1.Group("/proxy")
		s.remoteProxyHandler.RegisterRoutes(proxyGroup)
		s.logger.Printf("✅ Remote proxy routes registered successfully")
	} else {
		s.logger.Printf("⚠️  WARNING: Remote proxy handler is nil, skipping route registration")
	}

	// Register Encryption Admin routes (TMK, DEK rewrap, encrypted backups)
	if s.encryptionAdminHandler != nil {
		s.logger.Printf("Registering encryption admin routes...")
		encryptionGroup := apiV1.Group("/encryption")
		s.encryptionAdminHandler.RegisterRoutes(encryptionGroup)
		s.logger.Printf("✅ Encryption admin routes registered successfully")
	}

	// Register Signature routes (Cosign, PGP, Sigstore verification)
	if s.signatureHandler != nil {
		s.logger.Printf("Registering signature verification routes...")
		signatureGroup := apiV1.Group("/signatures")
		s.signatureHandler.RegisterRoutes(signatureGroup)
		s.logger.Printf("✅ Signature verification routes registered successfully")
	}

	// Register Helm routes for Helm chart repository
	if s.helmHandler != nil {
		s.logger.Printf("Registering Helm chart repository routes...")
		helmGroup := apiV1.Group("/helm")
		s.helmHandler.RegisterRoutes(helmGroup)
		s.logger.Printf("✅ Helm chart repository routes registered successfully")
	}

	// Register Replication Settings routes
	if s.replicationSettingsHandler != nil {
		s.logger.Printf("Registering replication settings routes...")
		replicationGroup := apiV1.Group("/settings/replication")
		s.replicationSettingsHandler.RegisterRoutes(replicationGroup)
		s.logger.Printf("✅ Replication settings routes registered successfully")
	}

	// Register Metrics and Health routes
	s.logger.Printf("Registering metrics and health routes...")

	// Public health endpoints (no JWT required - for K8s/Docker health checks)
	publicGroup := s.ginRouter.Group("/api/v1")
	publicGroup.GET("/health/live", s.metricsHandler.GetLiveness)
	publicGroup.GET("/health/ready", s.metricsHandler.GetReadiness)

	// Public tenant validation endpoints (no JWT required - for frontend tenant resolution)
	// Note: These must be registered separately from protected tenant management routes
	publicGroup.GET("/tenants/validate/:slug", s.tenantManagementHandler.ValidateTenant)
	publicGroup.GET("/tenants/resolve/:slug", s.tenantManagementHandler.GetTenantBySlug)

	// Protected metrics endpoints (require JWT)
	apiV1.GET("/metrics/cache", s.metricsHandler.GetCacheMetrics)
	apiV1.GET("/metrics/performance", s.metricsHandler.GetPerformanceMetrics)
	apiV1.GET("/health/repositories", s.metricsHandler.GetRepositoryHealth)
	apiV1.GET("/alerts", s.metricsHandler.GetAlerts)
	apiV1.GET("/health", s.metricsHandler.GetHealth)

	// WebSocket route for real-time updates
	s.logger.Printf("Registering WebSocket routes...")
	s.ginRouter.GET("/ws/updates", s.handleWebSocket)
	s.logger.Printf("✅ WebSocket routes registered successfully")

	s.logger.Printf("✅ All routes registered successfully on Gin framework")
}

func (s *Server) Start() error {
	// Start compliance scheduler
	if err := s.complianceScheduler.Start(); err != nil {
		s.logger.Printf("⚠️  Failed to start compliance scheduler: %v", err)
	}
	s.logger.Printf("✅ Compliance scheduler started")

	s.logger.Printf("🚀 Starting Gin server on port %s", s.config.Port)
	s.logger.Printf("   - Auth:      /api/auth/*")
	s.logger.Printf("   - API:       /api/v1/*")
	s.logger.Printf("   - Health:    /api/v1/health/*")
	s.logger.Printf("✅ Enterprise-grade Gin migration complete")

	// Start server with Gin router directly (CORS is now middleware)
	return http.ListenAndServe(":"+s.config.Port, s.ginRouter)
}

// Stop gracefully shuts down the server
func (s *Server) Stop() error {
	// Stop compliance scheduler
	if s.complianceScheduler != nil {
		if err := s.complianceScheduler.Stop(); err != nil {
			s.logger.Printf("Error stopping compliance scheduler: %v", err)
		}
	}
	return nil
}

// initializeArtifactScans triggers automatic security scans for a newly uploaded artifact
func (s *Server) initializeArtifactScans(artifactID string) {
	s.logger.Printf("🔍 Initializing security scans for artifact: %s", artifactID)

	// Parse artifact ID
	id, err := uuid.Parse(artifactID)
	if err != nil {
		s.logger.Printf("❌ Invalid artifact ID format: %v", err)
		return
	}

	// Get artifact details
	artifact, err := s.artifactService.GetByID(id)
	if err != nil {
		s.logger.Printf("❌ Failed to get artifact: %v", err)
		return
	}

	// Check if scan is already in progress
	if existingScan, err := s.scanService.GetActiveScan(id); err == nil && existingScan != nil {
		s.logger.Printf("⏭️  Scan already in progress for artifact %s (scan_id: %d)", artifactID, existingScan.ID)
		return
	}

	// Create scan record with automatic configuration
	scan := &models.SecurityScan{
		ArtifactID:        id,
		Status:            "initiated",
		ScanType:          "full",
		Priority:          "normal",
		VulnerabilityScan: true,
		MalwareScan:       true,
		LicenseScan:       true,
		DependencyScan:    true,
		InitiatedBy:       nil, // Automatic scan
		StartedAt:         time.Now(),
	}

	if err := s.scanService.CreateScan(scan); err != nil {
		s.logger.Printf("❌ Failed to create scan record: %v", err)
		return
	}

	s.logger.Printf("✅ Scan initiated for artifact %s (scan_id: %d)", artifactID, scan.ID)

	// Start scan asynchronously
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		s.logger.Printf("🔍 Starting scan execution for artifact %s", artifactID)
		if err := s.scanService.ScanArtifact(ctx, artifact, scan); err != nil {
			s.logger.Printf("❌ Scan failed for artifact %s: %v", artifactID, err)
			scan.Status = "failed"
			errMsg := err.Error()
			scan.ErrorMessage = &errMsg
			now := time.Now()
			scan.CompletedAt = &now
			s.scanService.UpdateScan(scan)
		} else {
			s.logger.Printf("✅ Scan completed successfully for artifact %s", artifactID)
		}
	}()
}

// handleValidateTenant validates if a tenant exists by slug (public endpoint)
func (s *Server) handleValidateTenant(c *gin.Context) {
	slug := c.Param("slug")

	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"exists": false,
			"error":  "Tenant slug is required",
		})
		return
	}

	// Resolve tenant by slug
	tenantInfo, err := s.tenantResolver.ResolveBySlug(c.Request.Context(), slug)
	if err != nil {
		s.logger.Printf("[TENANT_VALIDATE] Failed to resolve tenant '%s': %v", slug, err)
		c.JSON(http.StatusOK, gin.H{
			"exists": false,
			"slug":   slug,
		})
		return
	}

	// Tenant exists
	c.JSON(http.StatusOK, gin.H{
		"exists":      true,
		"slug":        tenantInfo.Slug,
		"tenant_id":   tenantInfo.ID,
		"tenant_name": tenantInfo.Name,
	})
}

// handleResolveTenant resolves full tenant data by slug (public endpoint)
func (s *Server) handleResolveTenant(c *gin.Context) {
	slug := c.Param("slug")

	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Tenant slug is required",
		})
		return
	}

	// Resolve tenant by slug
	tenantInfo, err := s.tenantResolver.ResolveBySlug(c.Request.Context(), slug)
	if err != nil {
		s.logger.Printf("[TENANT_RESOLVE] Failed to resolve tenant '%s': %v", slug, err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Tenant not found",
		})
		return
	}

	// Return full tenant data
	c.JSON(http.StatusOK, gin.H{
		"id":        tenantInfo.ID,
		"name":      tenantInfo.Name,
		"slug":      tenantInfo.Slug,
		"is_active": tenantInfo.IsActive,
		"plan":      tenantInfo.Plan,
	})
}
