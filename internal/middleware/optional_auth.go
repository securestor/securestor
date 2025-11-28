package middleware

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/securestor/securestor/internal/models"
	"github.com/securestor/securestor/internal/tenant"
)

// OptionalAuthMiddleware provides authentication that allows anonymous access
// for public repositories while still authenticating users when credentials are provided
type OptionalAuthMiddleware struct {
	jwtAuth *JWTAuth
	db      *sql.DB
	logger  *log.Logger
}

// NewOptionalAuthMiddleware creates a new optional auth middleware
func NewOptionalAuthMiddleware(jwtAuth *JWTAuth, db *sql.DB, logger *log.Logger) *OptionalAuthMiddleware {
	return &OptionalAuthMiddleware{
		jwtAuth: jwtAuth,
		db:      db,
		logger:  logger,
	}
}

// Handler wraps an HTTP handler with optional authentication
func (m *OptionalAuthMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to extract authentication token
		authHeader := r.Header.Get("Authorization")

		var authContext *models.AuthContext

		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")

			// Validate token (note: validateToken is lowercase)
			claims, err := m.jwtAuth.validateToken(token)
			if err == nil && claims != nil {
				// Valid authentication - inject auth context
				authContext = &models.AuthContext{
					UserID:   claims.UserID,
					Email:    claims.Email,
					Username: claims.Username,
					IsAdmin:  claims.IsAdmin,
					AuthType: "user",
				}

				// Add auth context to request
				ctx := context.WithValue(r.Context(), "auth", authContext)
				r = r.WithContext(ctx)

				m.logger.Printf("[OPTIONAL_AUTH] Authenticated user: %s (tenant: %s)", claims.Email, claims.TenantID)
			} else {
				m.logger.Printf("[OPTIONAL_AUTH] Invalid token provided: %v", err)
			}
		}

		// If no valid auth, check if this is for a public repository
		if authContext == nil {
			m.logger.Printf("[OPTIONAL_AUTH] No authentication provided - checking for public repository access")
		}

		// Continue to next handler regardless of authentication status
		next.ServeHTTP(w, r)
	})
}

// CheckRepositoryAccess checks if a user has access to a repository
// Returns true if:
// - Repository is public (allows anonymous access)
// - User is authenticated and belongs to the same tenant as the repository
func (m *OptionalAuthMiddleware) CheckRepositoryAccess(ctx context.Context, repositoryID uuid.UUID) (bool, bool, error) {
	// Get repository details including public_access flag
	var publicAccess bool
	var repoTenantID uuid.UUID

	query := `SELECT public_access, tenant_id FROM repositories WHERE id = $1`
	err := m.db.QueryRowContext(ctx, query, repositoryID).Scan(&publicAccess, &repoTenantID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, false, nil // Repository not found
		}
		return false, false, err
	}

	// If repository is public, allow access
	if publicAccess {
		m.logger.Printf("[OPTIONAL_AUTH] Repository %s is public - allowing access", repositoryID)
		return true, true, nil // accessible, public
	}

	// Repository is private - check authentication
	authContext, _ := ctx.Value("auth").(*models.AuthContext)
	if authContext == nil {
		m.logger.Printf("[OPTIONAL_AUTH] Repository %s is private and no authentication provided", repositoryID)
		return false, false, nil // not accessible, not public
	}

	// Get tenant ID from context
	tenantID, err := tenant.GetTenantID(ctx)
	if err != nil {
		m.logger.Printf("[OPTIONAL_AUTH] Failed to get tenant ID from context: %v", err)
		return false, false, nil
	}

	// Check if user's tenant matches repository tenant
	if tenantID != repoTenantID {
		m.logger.Printf("[OPTIONAL_AUTH] Tenant mismatch: user %s, repository %s", tenantID, repoTenantID)
		return false, false, nil
	}

	m.logger.Printf("[OPTIONAL_AUTH] User authenticated and authorized for private repository %s", repositoryID)
	return true, false, nil // accessible, not public
}

// GetOptionalAuthContext retrieves the auth context from the context if available
func GetOptionalAuthContext(ctx context.Context) *models.AuthContext {
	if ctx == nil {
		return nil
	}
	authContext, _ := ctx.Value("auth").(*models.AuthContext)
	return authContext
}
