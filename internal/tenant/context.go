package tenant

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// ======================= TENANT CONTEXT =======================

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// TenantIDKey is the context key for tenant ID
	TenantIDKey contextKey = "tenant_id"
	// TenantSlugKey is the context key for tenant slug (subdomain)
	TenantSlugKey contextKey = "tenant_slug"
	// TenantNameKey is the context key for tenant name
	TenantNameKey contextKey = "tenant_name"
)

var (
	// ErrNoTenantContext is returned when tenant context is missing
	ErrNoTenantContext = errors.New("no tenant context found in request")
	// ErrInvalidTenantID is returned when tenant ID is invalid
	ErrInvalidTenantID = errors.New("invalid tenant ID")
)

// ======================= CONTEXT HELPERS =======================

// WithTenantID adds tenant ID to context
func WithTenantID(ctx context.Context, tenantID uuid.UUID) context.Context {
	return context.WithValue(ctx, TenantIDKey, tenantID)
}

// WithTenantSlug adds tenant slug to context
func WithTenantSlug(ctx context.Context, slug string) context.Context {
	return context.WithValue(ctx, TenantSlugKey, slug)
}

// WithTenantName adds tenant name to context
func WithTenantName(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, TenantNameKey, name)
}

// WithTenant adds all tenant information to context
func WithTenant(ctx context.Context, tenantID uuid.UUID, slug, name string) context.Context {
	ctx = WithTenantID(ctx, tenantID)
	ctx = WithTenantSlug(ctx, slug)
	ctx = WithTenantName(ctx, name)
	return ctx
}

// GetTenantID extracts tenant ID from context
func GetTenantID(ctx context.Context) (uuid.UUID, error) {
	tenantID, ok := ctx.Value(TenantIDKey).(uuid.UUID)
	if !ok {
		return uuid.Nil, ErrNoTenantContext
	}
	return tenantID, nil
}

// GetTenantSlug extracts tenant slug from context
func GetTenantSlug(ctx context.Context) (string, error) {
	slug, ok := ctx.Value(TenantSlugKey).(string)
	if !ok {
		return "", ErrNoTenantContext
	}
	return slug, nil
}

// GetTenantName extracts tenant name from context
func GetTenantName(ctx context.Context) (string, error) {
	name, ok := ctx.Value(TenantNameKey).(string)
	if !ok {
		return "", ErrNoTenantContext
	}
	return name, nil
}

// MustGetTenantID extracts tenant ID from context or panics
// Use this only in handlers where tenant middleware guarantees context exists
func MustGetTenantID(ctx context.Context) uuid.UUID {
	tenantID, err := GetTenantID(ctx)
	if err != nil {
		panic("tenant context not found: ensure tenant middleware is applied")
	}
	return tenantID
}

// ======================= TENANT INFO STRUCT =======================

// TenantInfo contains complete tenant information
type TenantInfo struct {
	ID       uuid.UUID `json:"id"`
	Slug     string    `json:"slug"`
	Name     string    `json:"name"`
	IsActive bool      `json:"is_active"`
	Plan     string    `json:"plan"`
}

// GetTenantInfo extracts complete tenant information from context
func GetTenantInfo(ctx context.Context) (*TenantInfo, error) {
	tenantID, err := GetTenantID(ctx)
	if err != nil {
		return nil, err
	}

	slug, _ := GetTenantSlug(ctx)
	name, _ := GetTenantName(ctx)

	return &TenantInfo{
		ID:   tenantID,
		Slug: slug,
		Name: name,
	}, nil
}
