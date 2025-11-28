package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/securestor/securestor/internal/service"
)

// TokenIntrospectionHandler handles OAuth2 token introspection requests
type TokenIntrospectionHandler struct {
	introspectionSvc *service.TokenIntrospectionService
}

// NewTokenIntrospectionHandler creates a new token introspection handler
func NewTokenIntrospectionHandler(introspectionSvc *service.TokenIntrospectionService) *TokenIntrospectionHandler {
	return &TokenIntrospectionHandler{
		introspectionSvc: introspectionSvc,
	}
}

// IntrospectToken handles POST /oauth/introspect requests
func (h *TokenIntrospectionHandler) IntrospectToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST method is allowed")
		return
	}

	// Parse form data
	err := r.ParseForm()
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "invalid_request", "Failed to parse form data")
		return
	}

	// Get token from form
	token := r.FormValue("token")
	if token == "" {
		writeErrorResponse(w, http.StatusBadRequest, "invalid_request", "Token parameter is required")
		return
	}

	// Optional: Check client authentication
	// In a production system, you'd validate client credentials here
	clientID := r.FormValue("client_id")
	clientSecret := r.FormValue("client_secret")

	// For simplicity, we'll accept any client credentials for now
	if clientID == "" && clientSecret == "" {
		// Check Authorization header for Basic auth
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Basic ") {
			writeErrorResponse(w, http.StatusUnauthorized, "invalid_client", "Client authentication required")
			return
		}
	}

	// Perform token introspection
	response, err := h.introspectionSvc.IntrospectToken(r.Context(), token)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "server_error", "Internal server error")
		return
	}

	// Return introspection response
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")

	json.NewEncoder(w).Encode(response)
}

// RevokeToken handles POST /oauth/revoke requests
func (h *TokenIntrospectionHandler) RevokeToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST method is allowed")
		return
	}

	// Parse form data
	err := r.ParseForm()
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "invalid_request", "Failed to parse form data")
		return
	}

	// Get token from form
	token := r.FormValue("token")
	if token == "" {
		writeErrorResponse(w, http.StatusBadRequest, "invalid_request", "Token parameter is required")
		return
	}

	// Optional token type hint
	tokenTypeHint := r.FormValue("token_type_hint")
	_ = tokenTypeHint // We'll ignore this for now

	// Revoke the token
	err = h.introspectionSvc.RevokeToken(r.Context(), token)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "server_error", "Failed to revoke token")
		return
	}

	// Return success (200 OK with empty body per RFC 7009)
	w.WriteHeader(http.StatusOK)
}

// GetTokenStats handles GET /oauth/token-stats requests (admin endpoint)
func (h *TokenIntrospectionHandler) GetTokenStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET method is allowed")
		return
	}

	// In a production system, you'd check admin permissions here

	stats, err := h.introspectionSvc.GetTokenStats(r.Context())
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "server_error", "Failed to get token stats")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"data":   stats,
	})
}

// CleanupExpiredCache handles POST /oauth/cleanup-cache requests (admin endpoint)
func (h *TokenIntrospectionHandler) CleanupExpiredCache(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST method is allowed")
		return
	}

	// In a production system, you'd check admin permissions here

	err := h.introspectionSvc.CleanupExpiredCache(r.Context())
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "server_error", "Failed to cleanup cache")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Expired cache entries cleaned up successfully",
	})
}

// writeErrorResponse writes an OAuth2 error response
func writeErrorResponse(w http.ResponseWriter, statusCode int, errorCode, errorDescription string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.WriteHeader(statusCode)

	errorResponse := map[string]interface{}{
		"error":             errorCode,
		"error_description": errorDescription,
	}

	json.NewEncoder(w).Encode(errorResponse)
}
