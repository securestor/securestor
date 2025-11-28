package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/securestor/securestor/internal/tenant"
)

// GinTenantMiddleware is a Gin middleware adapter for tenant resolution
type GinTenantMiddleware struct {
	tenantMiddleware *tenant.Middleware
}

// NewGinTenantMiddleware creates a new Gin tenant middleware
func NewGinTenantMiddleware(tenantMiddleware *tenant.Middleware) *GinTenantMiddleware {
	return &GinTenantMiddleware{
		tenantMiddleware: tenantMiddleware,
	}
}

// Handler is Gin middleware that resolves and injects tenant context
func (m *GinTenantMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Run the chain first to let JWT auth set tenant_id if present
		c.Next()

		// After JWT runs, check if tenant_id was set by JWT auth
		if _, exists := c.Get("tenant_id"); exists {
			// Tenant already set by JWT, nothing to do
			return
		}

		// JWT didn't set tenant (unauthenticated request), so resolve from header/domain
		// Try to resolve tenant from header or subdomain
		slug := c.GetHeader("X-Tenant-Slug")
		if slug == "" {
			// Try to extract from subdomain
			host := c.Request.Host
			slug = extractTenantSlug(host)
		}

		// If still no slug, use default
		if slug == "" {
			slug = "default"
		}

		// Resolve tenant ID from slug
		// For now, we'll use a simple lookup - in production this should query the database
		// This is a simplified version - the full implementation would use the tenant service
		tenantID := resolveTenantIDFromSlug(slug)

		c.Set("tenant_id", tenantID)
		c.Set("tenant_slug", slug)
	}
}

// extractTenantSlug extracts tenant slug from subdomain
func extractTenantSlug(host string) string {
	// Simple subdomain extraction
	// Format: tenant.domain.com -> tenant
	// Skip for localhost
	if host == "localhost" || host == "localhost:8080" {
		return ""
	}

	parts := splitHost(host)
	if len(parts) > 2 {
		return parts[0]
	}

	return ""
}

// splitHost splits host into parts
func splitHost(host string) []string {
	// Remove port if present
	if idx := findChar(host, ':'); idx != -1 {
		host = host[:idx]
	}

	// Split by dots
	var parts []string
	start := 0
	for i := 0; i < len(host); i++ {
		if host[i] == '.' {
			parts = append(parts, host[start:i])
			start = i + 1
		}
	}
	if start < len(host) {
		parts = append(parts, host[start:])
	}

	return parts
}

// findChar finds the index of a character in a string
func findChar(s string, c rune) int {
	for i, ch := range s {
		if ch == c {
			return i
		}
	}
	return -1
}

// resolveTenantIDFromSlug resolves tenant ID from slug
// This is a simplified version - should query database in production
func resolveTenantIDFromSlug(slug string) uuid.UUID {
	// Default tenant ID for "default" slug
	// In production, this should query the tenants table
	if slug == "default" {
		// Return the default tenant ID from your database
		defaultID, _ := uuid.Parse("bfbde431-c627-49fe-9295-9e51643dc190")
		return defaultID
	}

	// For other slugs, generate a zero UUID as fallback
	// In production, query the database
	return uuid.UUID{}
}
