package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/securestor/securestor/internal/service"
)

// AuditMiddleware logs API requests and responses
type AuditMiddleware struct {
	auditService   *service.AuditLogService
	sessionService *service.UserSessionService
	excludePaths   []string
}

// NewAuditMiddleware creates a new audit middleware
func NewAuditMiddleware(auditService *service.AuditLogService, sessionService *service.UserSessionService) *AuditMiddleware {
	return &AuditMiddleware{
		auditService:   auditService,
		sessionService: sessionService,
		excludePaths: []string{
			"/health",
			"/metrics",
			"/static",
			"/favicon.ico",
		},
	}
}

// AuditContext holds audit information for the request
type AuditContext struct {
	UserID    string
	TenantID  string
	SessionID string
	IPAddress string
	UserAgent string
	StartTime time.Time
}

// ContextKey for storing audit context
type ContextKey string

const AuditContextKey ContextKey = "audit_context"

// Handler returns the middleware handler
func (m *AuditMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip excluded paths
		for _, path := range m.excludePaths {
			if strings.HasPrefix(r.URL.Path, path) {
				next.ServeHTTP(w, r)
				return
			}
		}

		// Extract audit context
		auditCtx := m.extractAuditContext(r)

		// Add audit context to request
		ctx := context.WithValue(r.Context(), AuditContextKey, auditCtx)
		r = r.WithContext(ctx)

		// Handle session tracking
		if auditCtx.UserID != "" {
			m.handleSessionTracking(ctx, auditCtx)
		}

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: 200}

		// Call next handler
		next.ServeHTTP(wrapped, r)

		// Log API access (only if we have tenant context)
		if m.auditService != nil && auditCtx.UserID != "" && auditCtx.TenantID != "" {
			go m.logAPIAccess(context.Background(), auditCtx, r, wrapped.statusCode)
		}

		// Log page views for non-API requests
		if !strings.HasPrefix(r.URL.Path, "/api/") && auditCtx.UserID != "" && auditCtx.TenantID != "" {
			go m.logPageView(context.Background(), auditCtx, r)
		}
	})
}

// extractAuditContext extracts audit information from the request
func (m *AuditMiddleware) extractAuditContext(r *http.Request) *AuditContext {
	// Extract user ID and tenant ID from auth context
	userID := "anonymous"
	tenantID := ""

	if authCtx := GetAuthContext(r); authCtx != nil {
		userID = authCtx.UserID.String()
		tenantID = authCtx.TenantID.String()
	}

	// Fallback to headers if auth context not available
	if userID == "anonymous" {
		if headerUserID := r.Header.Get("X-User-ID"); headerUserID != "" {
			userID = headerUserID
		}
	}
	if tenantID == "" {
		if headerTenantID := r.Header.Get("X-Tenant-ID"); headerTenantID != "" {
			tenantID = headerTenantID
		}
	}

	// Extract session ID
	sessionID := r.Header.Get("X-Session-ID")
	if sessionID == "" {
		// Try to get from cookie
		if cookie, err := r.Cookie("session_id"); err == nil {
			sessionID = cookie.Value
		}
	}

	// Extract IP address
	ipAddress := service.ExtractIPFromRequest(
		r.RemoteAddr,
		r.Header.Get("X-Forwarded-For"),
		r.Header.Get("X-Real-IP"),
	)

	// Extract user agent
	userAgent := r.Header.Get("User-Agent")

	return &AuditContext{
		UserID:    userID,
		TenantID:  tenantID,
		SessionID: sessionID,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		StartTime: time.Now(),
	}
}

// handleSessionTracking manages user session tracking
func (m *AuditMiddleware) handleSessionTracking(ctx context.Context, auditCtx *AuditContext) {
	if m.sessionService == nil || auditCtx.UserID == "anonymous" {
		return
	}

	// Get or create session
	session, err := m.sessionService.GetOrCreateSession(
		ctx, auditCtx.SessionID, auditCtx.UserID,
		auditCtx.IPAddress, auditCtx.UserAgent,
	)

	if err == nil && session != nil {
		auditCtx.SessionID = session.ID
	}
}

// logAPIAccess logs API access events
func (m *AuditMiddleware) logAPIAccess(ctx context.Context, auditCtx *AuditContext, r *http.Request, statusCode int) {
	if m.auditService == nil {
		return
	}

	endpoint := r.URL.Path
	if r.URL.RawQuery != "" {
		endpoint += "?" + r.URL.RawQuery
	}

	m.auditService.LogAPIAccess(
		ctx, auditCtx.UserID, auditCtx.TenantID, endpoint, r.Method,
		auditCtx.IPAddress, auditCtx.UserAgent, auditCtx.SessionID, statusCode,
	)
}

// logPageView logs page view events
func (m *AuditMiddleware) logPageView(ctx context.Context, auditCtx *AuditContext, r *http.Request) {
	if m.auditService == nil {
		return
	}

	page := r.URL.Path
	if page == "/" {
		page = "dashboard"
	}

	m.auditService.LogPageView(
		ctx, auditCtx.TenantID, auditCtx.UserID, page,
		auditCtx.IPAddress, auditCtx.UserAgent, auditCtx.SessionID,
	)
}

// GetAuditContext retrieves audit context from request context
func GetAuditContext(ctx context.Context) *AuditContext {
	if auditCtx, ok := ctx.Value(AuditContextKey).(*AuditContext); ok {
		return auditCtx
	}
	return &AuditContext{
		UserID:    "anonymous",
		IPAddress: "unknown",
		UserAgent: "unknown",
		StartTime: time.Now(),
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// DownloadTrackingMiddleware specifically tracks artifact downloads
type DownloadTrackingMiddleware struct {
	auditService *service.AuditLogService
}

// NewDownloadTrackingMiddleware creates download tracking middleware
func NewDownloadTrackingMiddleware(auditService *service.AuditLogService) *DownloadTrackingMiddleware {
	return &DownloadTrackingMiddleware{
		auditService: auditService,
	}
}

// Handler tracks artifact downloads
func (m *DownloadTrackingMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only track downloads
		if !strings.Contains(r.URL.Path, "/download") {
			next.ServeHTTP(w, r)
			return
		}

		auditCtx := GetAuditContext(r.Context())

		// Extract artifact ID from URL path
		pathParts := strings.Split(r.URL.Path, "/")
		var artifactID uuid.UUID
		for i, part := range pathParts {
			if part == "artifacts" && i+1 < len(pathParts) {
				if id, err := uuid.Parse(pathParts[i+1]); err == nil {
					artifactID = id
					break
				}
			}
		}

		// Wrap response writer to track success
		wrapped := &downloadResponseWriter{ResponseWriter: w, startTime: time.Now()}

		// Call next handler
		next.ServeHTTP(wrapped, r)

		// Log download event if we have an artifact ID and user
		emptyUUID := uuid.UUID{}
		if artifactID != emptyUUID && auditCtx.UserID != "anonymous" && m.auditService != nil {
			go m.logDownload(context.Background(), artifactID, auditCtx, wrapped)
		}
	})
}

// logDownload logs the download event
func (m *DownloadTrackingMiddleware) logDownload(ctx context.Context, artifactID uuid.UUID, auditCtx *AuditContext, wrapped *downloadResponseWriter) {
	// Log to audit service
	m.auditService.LogDownload(
		ctx, auditCtx.TenantID, artifactID, auditCtx.UserID,
		auditCtx.IPAddress, auditCtx.UserAgent, auditCtx.SessionID,
	)

	// Could also log to artifact_downloads table with additional metrics
	// This would require access to database service
}

// downloadResponseWriter tracks download-specific metrics
type downloadResponseWriter struct {
	http.ResponseWriter
	startTime  time.Time
	statusCode int
	bytes      int64
}

func (rw *downloadResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *downloadResponseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytes += int64(n)
	return n, err
}
