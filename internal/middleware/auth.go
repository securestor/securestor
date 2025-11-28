package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/securestor/securestor/internal/config"
)

// AuthClaims represents the JWT claims
type AuthClaims struct {
	UserID   string `json:"user_id"`   // UUID as string
	TenantID string `json:"tenant_id"` // UUID as string
	Username string `json:"username"`
	Email    string `json:"email"`
	IsAdmin  bool   `json:"is_admin"`
	jwt.RegisteredClaims
}

// AuthContext contains authentication information
type AuthContext struct {
	UserID   uuid.UUID `json:"user_id"`   // UUID type
	TenantID uuid.UUID `json:"tenant_id"` // UUID type for multi-tenancy
	Username string    `json:"username"`
	Email    string    `json:"email"`
	IsAdmin  bool      `json:"is_admin"`
}

// JWTAuth provides JWT-based authentication
type JWTAuth struct {
	jwtSecret []byte
}

// NewJWTAuth creates a new JWT authentication provider
func NewJWTAuth() *JWTAuth {
	// Use centralized environment loading
	config.LoadEnvOnce()
	secret := config.GetEnvWithFallback("JWT_SECRET", "your-secret-key-change-in-production")

	return &JWTAuth{
		jwtSecret: []byte(secret),
	}
}

// RequireAuth is middleware that requires valid JWT authentication
func (a *JWTAuth) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			a.unauthorizedResponse(w, "Authorization header required")
			return
		}

		// Check if header starts with "Bearer "
		tokenString := ""
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenString = authHeader[7:]
		} else {
			a.unauthorizedResponse(w, "Invalid authorization header format")
			return
		}

		// Parse and validate token
		claims, err := a.validateToken(tokenString)
		if err != nil {
			a.unauthorizedResponse(w, "Invalid token: "+err.Error())
			return
		}

		// Parse UserID from string to UUID
		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			a.unauthorizedResponse(w, "Invalid user ID in token")
			return
		}

		// Parse TenantID from string to UUID
		tenantID, err := uuid.Parse(claims.TenantID)
		if err != nil {
			a.unauthorizedResponse(w, "Invalid tenant ID in token")
			return
		}

		// Add auth context to request
		ctx := context.WithValue(r.Context(), "auth", &AuthContext{
			UserID:   userID,
			TenantID: tenantID,
			Username: claims.Username,
			Email:    claims.Email,
			IsAdmin:  claims.IsAdmin,
		})

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAdmin is middleware that requires admin privileges
func (a *JWTAuth) RequireAdmin(next http.Handler) http.Handler {
	return a.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := GetAuthContext(r)
		if auth == nil || !auth.IsAdmin {
			a.forbiddenResponse(w, "Admin privileges required")
			return
		}

		next.ServeHTTP(w, r)
	}))
}

// GenerateToken generates a JWT token for a user
func (a *JWTAuth) GenerateToken(userID uuid.UUID, tenantID uuid.UUID, username, email string, isAdmin bool) (string, error) {
	claims := AuthClaims{
		UserID:   userID.String(),
		TenantID: tenantID.String(),
		Username: username,
		Email:    email,
		IsAdmin:  isAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "securestor",
			Subject:   userID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.jwtSecret)
}

// validateToken parses and validates a JWT token
func (a *JWTAuth) validateToken(tokenString string) (*AuthClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AuthClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return a.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*AuthClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

// GetAuthContext extracts authentication context from request
func GetAuthContext(r *http.Request) *AuthContext {
	if auth := r.Context().Value("auth"); auth != nil {
		if authCtx, ok := auth.(*AuthContext); ok {
			return authCtx
		}
	}
	return nil
}

// unauthorizedResponse sends a 401 Unauthorized response
func (a *JWTAuth) unauthorizedResponse(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":   "Unauthorized",
		"message": message,
		"code":    http.StatusUnauthorized,
	})
}

// forbiddenResponse sends a 403 Forbidden response
func (a *JWTAuth) forbiddenResponse(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":   "Forbidden",
		"message": message,
		"code":    http.StatusForbidden,
	})
}

// Legacy AuthMiddleware function for backward compatibility
func AuthMiddleware(secret string) func(http.Handler) http.Handler {
	auth := &JWTAuth{jwtSecret: []byte(secret)}
	return auth.RequireAuth
}
