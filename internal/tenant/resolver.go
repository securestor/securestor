package tenant

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ======================= TENANT RESOLVER =======================

// TenantResolver resolves tenant information from slug/subdomain
type TenantResolver struct {
	db          *sql.DB
	cache       *tenantCache
	defaultSlug string
	defaultID   uuid.UUID
	cacheTTL    time.Duration
	enableCache bool
}

// TenantResolverConfig configures the tenant resolver
type TenantResolverConfig struct {
	DB                *sql.DB
	DefaultTenantSlug string        // Default tenant slug for requests without subdomain
	DefaultTenantID   uuid.UUID     // Default tenant ID (fallback)
	CacheTTL          time.Duration // TTL for tenant cache (default: 5 minutes)
	EnableCache       bool          // Enable in-memory caching
}

// NewTenantResolver creates a new tenant resolver
func NewTenantResolver(config TenantResolverConfig) (*TenantResolver, error) {
	if config.DB == nil {
		return nil, fmt.Errorf("database connection is required")
	}

	// Set defaults
	if config.CacheTTL == 0 {
		config.CacheTTL = 5 * time.Minute
	}

	resolver := &TenantResolver{
		db:          config.DB,
		defaultSlug: config.DefaultTenantSlug,
		defaultID:   config.DefaultTenantID,
		cacheTTL:    config.CacheTTL,
		enableCache: config.EnableCache,
	}

	// Initialize cache if enabled
	if config.EnableCache {
		resolver.cache = newTenantCache(config.CacheTTL)
		log.Printf("[TENANT] Cache enabled with TTL: %v", config.CacheTTL)
	}

	// Verify default tenant exists
	if config.DefaultTenantSlug != "" {
		_, err := resolver.ResolveBySlug(context.Background(), config.DefaultTenantSlug)
		if err != nil {
			log.Printf("[TENANT] Warning: Default tenant '%s' not found: %v", config.DefaultTenantSlug, err)
		} else {
			log.Printf("[TENANT] Default tenant validated: %s", config.DefaultTenantSlug)
		}
	}

	return resolver, nil
}

// ResolveBySlug resolves tenant by slug (subdomain)
func (r *TenantResolver) ResolveBySlug(ctx context.Context, slug string) (*TenantInfo, error) {
	// Check cache first
	if r.enableCache && r.cache != nil {
		if info := r.cache.Get(slug); info != nil {
			return info, nil
		}
	}

	// Query database
	query := `
		SELECT tenant_id, slug, name, is_active, plan
		FROM tenants
		WHERE slug = $1 AND is_active = true
		LIMIT 1
	`

	info := &TenantInfo{}
	err := r.db.QueryRowContext(ctx, query, slug).Scan(
		&info.ID,
		&info.Slug,
		&info.Name,
		&info.IsActive,
		&info.Plan,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("tenant not found: %s", slug)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to resolve tenant: %w", err)
	}

	// Cache the result
	if r.enableCache && r.cache != nil {
		r.cache.Set(slug, info)
	}

	return info, nil
}

// ResolveByID resolves tenant by ID
func (r *TenantResolver) ResolveByID(ctx context.Context, tenantID uuid.UUID) (*TenantInfo, error) {
	// Check cache by ID
	if r.enableCache && r.cache != nil {
		if info := r.cache.GetByID(tenantID); info != nil {
			return info, nil
		}
	}

	// Query database
	query := `
		SELECT tenant_id, slug, name, is_active, plan
		FROM tenants
		WHERE tenant_id = $1 AND is_active = true
		LIMIT 1
	`

	info := &TenantInfo{}
	err := r.db.QueryRowContext(ctx, query, tenantID).Scan(
		&info.ID,
		&info.Slug,
		&info.Name,
		&info.IsActive,
		&info.Plan,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("tenant not found: %s", tenantID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to resolve tenant: %w", err)
	}

	// Cache the result
	if r.enableCache && r.cache != nil {
		r.cache.Set(info.Slug, info)
	}

	return info, nil
}

// GetDefaultTenant returns the default tenant info
func (r *TenantResolver) GetDefaultTenant(ctx context.Context) (*TenantInfo, error) {
	if r.defaultSlug != "" {
		return r.ResolveBySlug(ctx, r.defaultSlug)
	}
	if r.defaultID != uuid.Nil {
		return r.ResolveByID(ctx, r.defaultID)
	}
	return nil, fmt.Errorf("no default tenant configured")
}

// InvalidateCache clears tenant from cache
func (r *TenantResolver) InvalidateCache(slug string) {
	if r.enableCache && r.cache != nil {
		r.cache.Delete(slug)
	}
}

// ClearCache clears entire tenant cache
func (r *TenantResolver) ClearCache() {
	if r.enableCache && r.cache != nil {
		r.cache.Clear()
	}
}

// ======================= TENANT CACHE =======================

type tenantCache struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry
	ttl     time.Duration
}

type cacheEntry struct {
	info      *TenantInfo
	expiresAt time.Time
}

func newTenantCache(ttl time.Duration) *tenantCache {
	cache := &tenantCache{
		entries: make(map[string]*cacheEntry),
		ttl:     ttl,
	}

	// Start cleanup goroutine
	go cache.cleanupExpired()

	return cache
}

func (c *tenantCache) Get(slug string) *TenantInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[slug]
	if !exists {
		return nil
	}

	// Check expiration
	if time.Now().After(entry.expiresAt) {
		return nil
	}

	return entry.info
}

func (c *tenantCache) GetByID(tenantID uuid.UUID) *TenantInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Linear search by ID (acceptable for small cache)
	for _, entry := range c.entries {
		if entry.info.ID == tenantID && time.Now().Before(entry.expiresAt) {
			return entry.info
		}
	}

	return nil
}

func (c *tenantCache) Set(slug string, info *TenantInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[slug] = &cacheEntry{
		info:      info,
		expiresAt: time.Now().Add(c.ttl),
	}
}

func (c *tenantCache) Delete(slug string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, slug)
}

func (c *tenantCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*cacheEntry)
}

func (c *tenantCache) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for slug, entry := range c.entries {
			if now.After(entry.expiresAt) {
				delete(c.entries, slug)
			}
		}
		c.mu.Unlock()
	}
}
