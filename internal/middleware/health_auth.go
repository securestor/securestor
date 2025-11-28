package middleware

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/securestor/securestor/internal/auth"
)

// HealthAuthMiddleware provides authentication for health check endpoints
type HealthAuthMiddleware struct {
	jwtAuth       *JWTAuth
	apiKeyService *auth.APIKeyService
}

// NewHealthAuthMiddleware creates a new health authentication middleware
func NewHealthAuthMiddleware(db *sql.DB, jwtAuth *JWTAuth, logger *log.Logger) *HealthAuthMiddleware {
	apiKeyService := auth.NewAPIKeyService(db, logger)
	return &HealthAuthMiddleware{
		jwtAuth:       jwtAuth,
		apiKeyService: apiKeyService,
	}
}

// RequireAuth is middleware that requires valid authentication for health endpoints
func (h *HealthAuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			h.unauthorizedResponse(w, "Authorization header required")
			return
		}

		// Try to authenticate using Bearer token (JWT or API Key)
		if strings.HasPrefix(authHeader, "Bearer ") {
			token := authHeader[7:]

			// First try JWT authentication
			claims, err := h.jwtAuth.validateToken(token)
			if err == nil {
				// Valid JWT token - add auth context and proceed
				ctx := context.WithValue(r.Context(), "auth", &AuthContext{
					UserID:   mustParseUUID(claims.UserID),
					TenantID: mustParseUUID(claims.TenantID),
					Username: claims.Username,
					Email:    claims.Email,
					IsAdmin:  claims.IsAdmin,
				})
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Try API key authentication (format: key_id:key_secret)
			apiKey, err := h.apiKeyService.ValidateAPIKey(token)
			if err == nil && apiKey != nil {
				// Valid API key - add minimal auth context and proceed
				ctx := context.WithValue(r.Context(), "auth", &AuthContext{
					UserID: apiKey.UserID,
					// TenantID will be set by tenant middleware
				})
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Both JWT and API key validation failed
			h.unauthorizedResponse(w, "Invalid authentication token")
			return
		}

		// Invalid authorization header format
		h.unauthorizedResponse(w, "Invalid authorization header format. Use 'Bearer <token>'")
	})
}

// unauthorizedResponse sends a 401 Unauthorized response
func (h *HealthAuthMiddleware) unauthorizedResponse(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":   "Unauthorized",
		"message": message,
		"code":    http.StatusUnauthorized,
	})
}

// mustParseUUID is a helper to parse UUID strings, panics on error
func mustParseUUID(s string) uuid.UUID {
	id, err := uuid.Parse(s)
	if err != nil {
		// Return zero UUID on error instead of panicking
		return uuid.UUID{}
	}
	return id
}
