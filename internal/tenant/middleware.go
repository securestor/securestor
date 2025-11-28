package tenant

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// ======================= TENANT MIDDLEWARE =======================

// Middleware handles tenant resolution from subdomain
type Middleware struct {
	resolver      *TenantResolver
	baseDomain    string // e.g., "securestor.io"
	headerName    string // Alternative: X-Tenant-ID header
	allowNoTenant bool   // Allow requests without tenant (use default)
	errorHandler  func(w http.ResponseWriter, r *http.Request, err error)
}

// MiddlewareConfig configures the tenant middleware
type MiddlewareConfig struct {
	Resolver      *TenantResolver
	BaseDomain    string // Base domain for subdomain extraction (e.g., "securestor.io")
	HeaderName    string // Optional: header name for tenant override (e.g., "X-Tenant-ID")
	AllowNoTenant bool   // If true, use default tenant when no subdomain present
	ErrorHandler  func(w http.ResponseWriter, r *http.Request, err error)
}

// NewMiddleware creates a new tenant middleware
func NewMiddleware(config MiddlewareConfig) *Middleware {
	if config.Resolver == nil {
		panic("tenant resolver is required")
	}

	// Set defaults
	if config.BaseDomain == "" {
		config.BaseDomain = "localhost" // Default for development
	}
	if config.HeaderName == "" {
		config.HeaderName = "X-Tenant-Slug"
	}
	if config.ErrorHandler == nil {
		config.ErrorHandler = defaultErrorHandler
	}

	return &Middleware{
		resolver:      config.Resolver,
		baseDomain:    config.BaseDomain,
		headerName:    config.HeaderName,
		allowNoTenant: config.AllowNoTenant,
		errorHandler:  config.ErrorHandler,
	}
}

// isPublicEndpoint checks if the request path is a public endpoint that doesn't require tenant context
func isPublicEndpoint(path string) bool {
	publicPaths := []string{
		"/api/v1/tenants/validate/",
		"/api/v1/auth/login",
		"/api/v1/auth/register",
		"/api/v1/auth/forgot-password",
		"/api/v1/auth/reset-password",
		"/health",
		"/metrics",
	}

	for _, publicPath := range publicPaths {
		if strings.HasPrefix(path, publicPath) {
			return true
		}
	}
	return false
}

// Handler returns the HTTP middleware handler
func (m *Middleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip tenant resolution for public endpoints
		if isPublicEndpoint(r.URL.Path) {
			log.Printf("[TENANT] Skipping tenant middleware for public endpoint: %s", r.URL.Path)
			next.ServeHTTP(w, r)
			return
		}

		// Extract tenant slug from request
		slug := m.extractTenantSlug(r)

		// If no tenant found and not allowed, use default
		if slug == "" {
			if m.allowNoTenant {
				info, err := m.resolver.GetDefaultTenant(r.Context())
				if err != nil {
					log.Printf("[TENANT] Failed to get default tenant: %v", err)
					m.errorHandler(w, r, err)
					return
				}
				slug = info.Slug
				log.Printf("[TENANT] Using default tenant: %s", slug)
			} else {
				m.errorHandler(w, r, ErrNoTenantContext)
				return
			}
		}

		// Resolve tenant information
		info, err := m.resolver.ResolveBySlug(r.Context(), slug)
		if err != nil {
			log.Printf("[TENANT] Failed to resolve tenant '%s': %v", slug, err)
			m.errorHandler(w, r, err)
			return
		}

		// Inject tenant context
		ctx := WithTenant(r.Context(), info.ID, info.Slug, info.Name)
		r = r.WithContext(ctx)

		// Add tenant headers to response for debugging
		w.Header().Set("X-Tenant-ID", info.ID.String())
		w.Header().Set("X-Tenant-Slug", info.Slug)

		log.Printf("[TENANT] Request authenticated for tenant: %s (ID: %s)", info.Name, info.ID)

		next.ServeHTTP(w, r)
	})
}

// extractTenantSlug extracts tenant slug from subdomain or header
func (m *Middleware) extractTenantSlug(r *http.Request) string {
	// Priority 1: Check custom header (useful for API clients, testing)
	if headerSlug := r.Header.Get(m.headerName); headerSlug != "" {
		log.Printf("[TENANT] Tenant from header '%s': %s", m.headerName, headerSlug)
		return headerSlug
	}

	// Priority 2: Extract from subdomain
	host := r.Host
	if host == "" {
		host = r.URL.Host
	}

	// Remove port if present
	if colonIdx := strings.Index(host, ":"); colonIdx != -1 {
		host = host[:colonIdx]
	}

	// Check if it's a subdomain
	// Example: alpha.securestor.io -> returns "alpha"
	// Example: securestor.io -> returns ""
	// Example: localhost -> returns ""

	if host == m.baseDomain || host == "localhost" || host == "127.0.0.1" {
		// No subdomain
		return ""
	}

	// Check if host ends with base domain
	if !strings.HasSuffix(host, "."+m.baseDomain) && host != m.baseDomain {
		// Not our domain, might be localhost or IP
		if host == "localhost" || strings.HasPrefix(host, "127.0.0.1") || strings.HasPrefix(host, "192.168.") {
			return "" // Local development without subdomain
		}
		log.Printf("[TENANT] Warning: Host '%s' doesn't match base domain '%s'", host, m.baseDomain)
		return ""
	}

	// Extract subdomain
	subdomain := strings.TrimSuffix(host, "."+m.baseDomain)

	// Handle multi-level subdomains (use first level only)
	// Example: api.alpha.securestor.io -> returns "api.alpha" -> we want "alpha"
	parts := strings.Split(subdomain, ".")
	if len(parts) > 1 {
		// Use the last subdomain part (closest to base domain)
		subdomain = parts[len(parts)-1]
	}

	log.Printf("[TENANT] Tenant from subdomain: %s (host: %s)", subdomain, host)
	return subdomain
}

// defaultErrorHandler is the default error handler for tenant resolution failures
func defaultErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	w.Header().Set("Content-Type", "application/json")

	if err == ErrNoTenantContext {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"tenant_required","message":"No tenant specified. Use subdomain (e.g., tenant.securestor.io) or X-Tenant-Slug header"}`))
		return
	}

	if strings.Contains(err.Error(), "not found") {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"tenant_not_found","message":"Tenant not found or inactive"}`))
		return
	}

	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(`{"error":"tenant_resolution_failed","message":"Failed to resolve tenant information"}`))
}

// ======================= GIN ADAPTER =======================

// GinMiddleware returns a Gin-compatible middleware
func (m *Middleware) GinMiddleware() func(*gin.Context) {
	return func(c *gin.Context) {
		// Extract tenant slug
		slug := m.extractTenantSlug(c.Request)

		// Handle missing tenant
		if slug == "" {
			if m.allowNoTenant {
				info, err := m.resolver.GetDefaultTenant(c.Request.Context())
				if err != nil {
					log.Printf("[TENANT] Failed to get default tenant: %v", err)
					c.JSON(http.StatusInternalServerError, gin.H{
						"error":   "tenant_resolution_failed",
						"message": "Failed to resolve default tenant",
					})
					c.Abort()
					return
				}
				slug = info.Slug
			} else {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "tenant_required",
					"message": "No tenant specified. Use subdomain or X-Tenant-Slug header",
				})
				c.Abort()
				return
			}
		}

		// Resolve tenant
		info, err := m.resolver.ResolveBySlug(c.Request.Context(), slug)
		if err != nil {
			log.Printf("[TENANT] Failed to resolve tenant '%s': %v", slug, err)
			if strings.Contains(err.Error(), "not found") {
				c.JSON(http.StatusNotFound, gin.H{
					"error":   "tenant_not_found",
					"message": "Tenant not found or inactive",
				})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":   "tenant_resolution_failed",
					"message": "Failed to resolve tenant",
				})
			}
			c.Abort()
			return
		}

		// Inject tenant context
		ctx := WithTenant(c.Request.Context(), info.ID, info.Slug, info.Name)
		c.Request = c.Request.WithContext(ctx)

		// Add headers
		c.Header("X-Tenant-ID", info.ID.String())
		c.Header("X-Tenant-Slug", info.Slug)

		log.Printf("[TENANT] Gin request for tenant: %s (ID: %s)", info.Name, info.ID)

		c.Next()
	}
}
