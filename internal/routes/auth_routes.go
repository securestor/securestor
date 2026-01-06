package routes

import (
	"database/sql"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/securestor/securestor/internal/auth"
	"github.com/securestor/securestor/internal/handlers"
	internallogger "github.com/securestor/securestor/internal/logger"
	"github.com/securestor/securestor/internal/middleware"
	"github.com/securestor/securestor/internal/service"
)

// AuthRoutesConfig holds configuration for authentication routes
type AuthRoutesConfig struct {
	Router        *gin.Engine
	DB            *sql.DB
	RedisClient   *redis.Client
	Logger        *log.Logger
	OIDCService   *auth.OIDCService
	JWTMiddleware *middleware.JWTMiddleware
	RBACService   *service.RBACService
	AuthHandler   *handlers.AuthHandler
}

// SetupAuthRoutes configures all authentication and authorization routes
func SetupAuthRoutes(config *AuthRoutesConfig) {
	// Public authentication routes (no auth required)
	authGroup := config.Router.Group("/api/v1/auth")
	{
		authGroup.POST("/login", config.AuthHandler.Login)
		authGroup.GET("/callback", config.AuthHandler.Callback)
		authGroup.POST("/logout", config.AuthHandler.Logout)
		authGroup.POST("/refresh", config.AuthHandler.RefreshToken)
	}

	// Protected routes that require authentication
	protectedGroup := config.Router.Group("/api/v1")
	protectedGroup.Use(config.JWTMiddleware.AuthRequired())
	{
		// User info endpoint
		protectedGroup.GET("/me", config.AuthHandler.Me)

		// Setup status endpoints (for onboarding wizard)
		setupHandler := handlers.NewSetupHandler(config.DB)
		protectedGroup.GET("/auth/check-default-password", setupHandler.CheckDefaultPassword)
		protectedGroup.GET("/setup/status", setupHandler.GetSetupStatus)

		// Admin-only routes
		adminGroup := protectedGroup.Group("/admin")
		adminGroup.Use(config.JWTMiddleware.RequireRole("admin"))
		{
			adminGroup.GET("/users", handleGetUsers)
			adminGroup.POST("/users/:id/roles", handleAssignUserRole)
			adminGroup.DELETE("/users/:id/roles/:roleId", handleRemoveUserRole)
			adminGroup.GET("/roles", handleGetRoles)
			adminGroup.POST("/roles", handleCreateRole)
			adminGroup.GET("/permissions", handleGetPermissions)
			adminGroup.POST("/permissions", handleCreatePermission)
		}

		// Developer routes (requires developer or admin role)
		devGroup := protectedGroup.Group("/dev")
		devGroup.Use(config.JWTMiddleware.RequireRole("developer", "admin"))
		{
			devGroup.GET("/artifacts", handleGetArtifacts)
			devGroup.POST("/artifacts", handleUploadArtifact)
			devGroup.DELETE("/artifacts/:id", handleDeleteArtifact)
		}

		// Auditor routes (requires auditor, developer, or admin role)
		auditGroup := protectedGroup.Group("/audit")
		auditGroup.Use(config.JWTMiddleware.RequireRole("auditor", "developer", "admin"))
		{
			auditGroup.GET("/scans", handleGetScans)
			auditGroup.GET("/compliance", handleGetCompliance)
			auditGroup.POST("/compliance/reports", handleCreateComplianceReport)
		}

		// Permission-based routes
		permissionGroup := protectedGroup.Group("/secure")
		{
			permissionGroup.GET("/artifacts/read",
				config.JWTMiddleware.RequirePermission("artifacts:read"),
				handleSecureArtifactsRead)
			permissionGroup.POST("/artifacts/write",
				config.JWTMiddleware.RequirePermission("artifacts:write"),
				handleSecureArtifactsWrite)
			permissionGroup.DELETE("/artifacts/delete",
				config.JWTMiddleware.RequirePermission("artifacts:delete"),
				handleSecureArtifactsDelete)
			permissionGroup.GET("/scans/read",
				config.JWTMiddleware.RequirePermission("scans:read"),
				handleSecureScansRead)
			permissionGroup.POST("/scans/write",
				config.JWTMiddleware.RequirePermission("scans:write"),
				handleSecureScansWrite)
			permissionGroup.GET("/compliance/read",
				config.JWTMiddleware.RequirePermission("compliance:read"),
				handleSecureComplianceRead)
			permissionGroup.POST("/compliance/write",
				config.JWTMiddleware.RequirePermission("compliance:write"),
				handleSecureComplianceWrite)
		}
	}
}

// InitializeAuth initializes the complete authentication system
func InitializeAuth(router *gin.Engine, db *sql.DB, redisClient *redis.Client, logger *log.Logger) (*middleware.JWTMiddleware, *handlers.AuthHandler, error) {
	// Create OIDC configuration
	oidcConfig := auth.CreateOIDCConfig()

	// Initialize OIDC service
	oidcService, err := auth.NewOIDCService(oidcConfig, logger)
	if err != nil {
		return nil, nil, err
	}

	// Initialize JWT middleware
	jwtMiddleware := middleware.NewJWTMiddleware(oidcService, redisClient, logger)

	// Initialize RBAC service
	rbacService := service.NewRBACService(db, logger)

	// Initialize SCIM service (use internal logger)
	scimLogger := internallogger.New()
	scimService := auth.NewSCIMService(db, scimLogger)

	// Initialize auth handler
	authHandler := handlers.NewAuthHandler(oidcService, jwtMiddleware, scimService, redisClient, db)

	// Setup auth routes
	config := &AuthRoutesConfig{
		Router:        router,
		DB:            db,
		RedisClient:   redisClient,
		Logger:        logger,
		OIDCService:   oidcService,
		JWTMiddleware: jwtMiddleware,
		RBACService:   rbacService,
		AuthHandler:   authHandler,
	}

	SetupAuthRoutes(config)

	return jwtMiddleware, authHandler, nil
}

// Placeholder handlers - these would be implemented based on your existing handlers
func handleGetUsers(c *gin.Context) {
	c.JSON(200, gin.H{"message": "Get users - admin only", "endpoint": "GET /api/v1/admin/users"})
}

func handleAssignUserRole(c *gin.Context) {
	userID := c.Param("id")
	c.JSON(200, gin.H{"message": "Assign role to user - admin only", "user_id": userID, "endpoint": "POST /api/v1/admin/users/:id/roles"})
}

func handleRemoveUserRole(c *gin.Context) {
	userID := c.Param("id")
	roleID := c.Param("roleId")
	c.JSON(200, gin.H{"message": "Remove role from user - admin only", "user_id": userID, "role_id": roleID, "endpoint": "DELETE /api/v1/admin/users/:id/roles/:roleId"})
}

func handleGetRoles(c *gin.Context) {
	c.JSON(200, gin.H{"message": "Get roles - admin only", "endpoint": "GET /api/v1/admin/roles"})
}

func handleCreateRole(c *gin.Context) {
	c.JSON(200, gin.H{"message": "Create role - admin only", "endpoint": "POST /api/v1/admin/roles"})
}

func handleGetPermissions(c *gin.Context) {
	c.JSON(200, gin.H{"message": "Get permissions - admin only", "endpoint": "GET /api/v1/admin/permissions"})
}

func handleCreatePermission(c *gin.Context) {
	c.JSON(200, gin.H{"message": "Create permission - admin only", "endpoint": "POST /api/v1/admin/permissions"})
}

func handleGetArtifacts(c *gin.Context) {
	c.JSON(200, gin.H{"message": "Get artifacts - developer/admin only", "endpoint": "GET /api/v1/dev/artifacts"})
}

func handleUploadArtifact(c *gin.Context) {
	c.JSON(200, gin.H{"message": "Upload artifact - developer/admin only", "endpoint": "POST /api/v1/dev/artifacts"})
}

func handleDeleteArtifact(c *gin.Context) {
	artifactID := c.Param("id")
	c.JSON(200, gin.H{"message": "Delete artifact - developer/admin only", "artifact_id": artifactID, "endpoint": "DELETE /api/v1/dev/artifacts/:id"})
}

func handleGetScans(c *gin.Context) {
	c.JSON(200, gin.H{"message": "Get scans - auditor/developer/admin", "endpoint": "GET /api/v1/audit/scans"})
}

func handleGetCompliance(c *gin.Context) {
	c.JSON(200, gin.H{"message": "Get compliance - auditor/developer/admin", "endpoint": "GET /api/v1/audit/compliance"})
}

func handleCreateComplianceReport(c *gin.Context) {
	c.JSON(200, gin.H{"message": "Create compliance report - auditor/developer/admin", "endpoint": "POST /api/v1/audit/compliance/reports"})
}

func handleSecureArtifactsRead(c *gin.Context) {
	c.JSON(200, gin.H{"message": "Secure artifacts read - requires artifacts:read permission", "endpoint": "GET /api/v1/secure/artifacts/read"})
}

func handleSecureArtifactsWrite(c *gin.Context) {
	c.JSON(200, gin.H{"message": "Secure artifacts write - requires artifacts:write permission", "endpoint": "POST /api/v1/secure/artifacts/write"})
}

func handleSecureArtifactsDelete(c *gin.Context) {
	c.JSON(200, gin.H{"message": "Secure artifacts delete - requires artifacts:delete permission", "endpoint": "DELETE /api/v1/secure/artifacts/delete"})
}

func handleSecureScansRead(c *gin.Context) {
	c.JSON(200, gin.H{"message": "Secure scans read - requires scans:read permission", "endpoint": "GET /api/v1/secure/scans/read"})
}

func handleSecureScansWrite(c *gin.Context) {
	c.JSON(200, gin.H{"message": "Secure scans write - requires scans:write permission", "endpoint": "POST /api/v1/secure/scans/write"})
}

func handleSecureComplianceRead(c *gin.Context) {
	c.JSON(200, gin.H{"message": "Secure compliance read - requires compliance:read permission", "endpoint": "GET /api/v1/secure/compliance/read"})
}

func handleSecureComplianceWrite(c *gin.Context) {
	c.JSON(200, gin.H{"message": "Secure compliance write - requires compliance:write permission", "endpoint": "POST /api/v1/secure/compliance/write"})
}
