package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/securestor/securestor/internal/models"
	"github.com/securestor/securestor/internal/service"
)

// AdminTokenHandler handles admin token management operations
type AdminTokenHandler struct {
	introspectionSvc *service.TokenIntrospectionService
	db               *sql.DB
}

// NewAdminTokenHandler creates a new admin token handler
func NewAdminTokenHandler(introspectionSvc *service.TokenIntrospectionService, db *sql.DB) *AdminTokenHandler {
	return &AdminTokenHandler{
		introspectionSvc: introspectionSvc,
		db:               db,
	}
}

// TokenInfo represents token information for the admin UI
type TokenInfo struct {
	ID         uuid.UUID `json:"id"`
	Type       string    `json:"type"`
	Name       string    `json:"name,omitempty"`
	ClientID   string    `json:"client_id,omitempty"`
	UserID     uuid.UUID `json:"user_id,omitempty"`
	IsActive   bool      `json:"is_active"`
	Scopes     []string  `json:"scopes,omitempty"`
	CreatedAt  string    `json:"created_at"`
	ExpiresAt  *string   `json:"expires_at,omitempty"`
	LastUsedAt *string   `json:"last_used_at,omitempty"`
	KeyID      string    `json:"key_id,omitempty"`
}

// ListTokens handles GET /api/admin/tokens
func (h *AdminTokenHandler) ListTokens(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET method is allowed")
		return
	}

	// TODO: Add admin permission check

	// Get pagination parameters
	page := 1
	limit := 50

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offset := (page - 1) * limit

	// Fetch API keys
	apiKeys, err := h.getAPIKeys(offset, limit)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "server_error", "Failed to fetch API keys")
		return
	}

	// Fetch OAuth tokens (if you have them)
	// oauthTokens, err := h.getOAuthTokens(offset, limit)
	// For now, we'll just return API keys

	tokens := make([]TokenInfo, 0, len(apiKeys))
	for _, apiKey := range apiKeys {
		token := TokenInfo{
			ID:        apiKey.ID,
			Type:      "api_key",
			Name:      apiKey.Name,
			UserID:    apiKey.UserID,
			IsActive:  apiKey.IsActive,
			Scopes:    apiKey.Scopes,
			CreatedAt: apiKey.CreatedAt.Format("2006-01-02T15:04:05Z"),
			KeyID:     apiKey.KeyID,
		}

		if apiKey.ExpiresAt != nil {
			expiresAt := apiKey.ExpiresAt.Format("2006-01-02T15:04:05Z")
			token.ExpiresAt = &expiresAt
		}

		if apiKey.LastUsedAt != nil {
			lastUsedAt := apiKey.LastUsedAt.Format("2006-01-02T15:04:05Z")
			token.LastUsedAt = &lastUsedAt
		}

		tokens = append(tokens, token)
	}

	// Get total count for pagination
	totalCount, err := h.getTotalTokenCount()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "server_error", "Failed to get total count")
		return
	}

	response := map[string]interface{}{
		"tokens": tokens,
		"pagination": map[string]interface{}{
			"page":        page,
			"limit":       limit,
			"total":       totalCount,
			"total_pages": (totalCount + limit - 1) / limit,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetTokenDetails handles GET /api/admin/tokens/{id}
func (h *AdminTokenHandler) GetTokenDetails(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET method is allowed")
		return
	}

	// TODO: Add admin permission check

	// Extract token ID from URL path
	tokenIDStr := r.URL.Path[len("/api/admin/tokens/"):]
	tokenID, err := uuid.Parse(tokenIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "Invalid token ID")
		return
	}

	// First try to find as API key
	apiKey, err := h.getAPIKeyByID(tokenID)
	if err == nil && apiKey != nil {
		token := TokenInfo{
			ID:        apiKey.ID,
			Type:      "api_key",
			Name:      apiKey.Name,
			UserID:    apiKey.UserID,
			IsActive:  apiKey.IsActive,
			Scopes:    apiKey.Scopes,
			CreatedAt: apiKey.CreatedAt.Format("2006-01-02T15:04:05Z"),
			KeyID:     apiKey.KeyID,
		}

		if apiKey.ExpiresAt != nil {
			expiresAt := apiKey.ExpiresAt.Format("2006-01-02T15:04:05Z")
			token.ExpiresAt = &expiresAt
		}

		if apiKey.LastUsedAt != nil {
			lastUsedAt := apiKey.LastUsedAt.Format("2006-01-02T15:04:05Z")
			token.LastUsedAt = &lastUsedAt
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"token": token,
		})
		return
	}

	// If not found as API key, try OAuth token
	// TODO: Implement OAuth token lookup

	writeJSONError(w, http.StatusNotFound, "not_found", "Token not found")
}

// RevokeTokenByID handles DELETE /api/admin/tokens/{id}
func (h *AdminTokenHandler) RevokeTokenByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only DELETE method is allowed")
		return
	}

	// TODO: Add admin permission check

	// Extract token ID from URL path
	tokenIDStr := r.URL.Path[len("/api/admin/tokens/"):]
	tokenID, err := uuid.Parse(tokenIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "Invalid token ID")
		return
	}

	// Revoke API key if it exists
	err = h.revokeAPIKeyByID(tokenID)
	if err == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "success",
			"message": "Token revoked successfully",
		})
		return
	}

	// If not found as API key, try OAuth token
	// TODO: Implement OAuth token revocation

	writeJSONError(w, http.StatusNotFound, "not_found", "Token not found")
}

// Helper methods for database operations

func (h *AdminTokenHandler) getAPIKeys(offset, limit int) ([]*models.APIKey, error) {
	query := `
		SELECT key_id, name, description, scopes, user_id, 
		       is_active, expires_at, last_used_at, created_at, updated_at
		FROM api_keys 
		ORDER BY created_at DESC 
		LIMIT $1 OFFSET $2
	`

	rows, err := h.db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apiKeys []*models.APIKey
	for rows.Next() {
		apiKey := &models.APIKey{}
		err := rows.Scan(
			&apiKey.KeyID, &apiKey.Name, &apiKey.Description,
			&apiKey.Scopes, &apiKey.UserID, &apiKey.IsActive,
			&apiKey.ExpiresAt, &apiKey.LastUsedAt, &apiKey.CreatedAt, &apiKey.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		apiKeys = append(apiKeys, apiKey)
	}

	return apiKeys, nil
}

func (h *AdminTokenHandler) getAPIKeyByID(id uuid.UUID) (*models.APIKey, error) {
	query := `
		SELECT key_id, name, description, scopes, user_id, 
		       is_active, expires_at, last_used_at, created_at, updated_at
		FROM api_keys 
		WHERE key_id = $1
	`

	apiKey := &models.APIKey{}
	err := h.db.QueryRow(query, id).Scan(
		&apiKey.KeyID, &apiKey.Name, &apiKey.Description,
		&apiKey.Scopes, &apiKey.UserID, &apiKey.IsActive,
		&apiKey.ExpiresAt, &apiKey.LastUsedAt, &apiKey.CreatedAt, &apiKey.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return apiKey, nil
}

func (h *AdminTokenHandler) revokeAPIKeyByID(id uuid.UUID) error {
	query := `UPDATE api_keys SET is_active = false WHERE key_id = $1`
	result, err := h.db.Exec(query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (h *AdminTokenHandler) getTotalTokenCount() (int, error) {
	var count int

	// Count API keys
	err := h.db.QueryRow("SELECT COUNT(*) FROM api_keys").Scan(&count)
	if err != nil {
		return 0, err
	}

	// TODO: Add OAuth token count when implemented

	return count, nil
}

// writeJSONError writes a JSON error response
func writeJSONError(w http.ResponseWriter, statusCode int, errorCode, errorDescription string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResponse := map[string]interface{}{
		"error":             errorCode,
		"error_description": errorDescription,
	}

	json.NewEncoder(w).Encode(errorResponse)
}
