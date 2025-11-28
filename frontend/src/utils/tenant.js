/**
 * Tenant Utilities
 * Provides helper functions for multi-tenant subdomain-based routing
 * Used across the application for tenant detection and management
 */

/**
 * Configuration
 */
const CONFIG = {
  // Base domain for the application (e.g., securestor.io)
  BASE_DOMAIN: process.env.REACT_APP_BASE_DOMAIN || 'localhost',
  
  // Default tenant slug when no subdomain is detected
  DEFAULT_TENANT: process.env.REACT_APP_DEFAULT_TENANT || 'default',
  
  // Enable development mode features (manual tenant switching)
  DEV_MODE: process.env.NODE_ENV === 'development' || process.env.REACT_APP_DEV_MODE === 'true',
  
  // Backend port (empty string means port 80)
  BACKEND_PORT: process.env.REACT_APP_BACKEND_PORT || '',
  
  // Storage key for manually selected tenant (dev mode)
  STORAGE_KEY: 'securestor_dev_tenant',
};

/**
 * Extract tenant slug from subdomain
 * Examples:
 *   alpha.securestor.io -> 'alpha'
 *   beta.localhost:3000 -> 'beta'
 *   securestor.io -> null (no subdomain)
 *   localhost -> null (no subdomain)
 * 
 * @param {string} hostname - The hostname to parse (defaults to window.location.hostname)
 * @returns {string|null} The tenant slug or null if no subdomain
 */
export const extractTenantFromSubdomain = (hostname = window.location.hostname) => {
  // Remove port if present
  const host = hostname.split(':')[0];
  
  // Split hostname into parts
  const parts = host.split('.');
  
  // Check if we have a subdomain
  // localhost -> 1 part (no subdomain)
  // securestor.io -> 2 parts (no subdomain)
  // alpha.securestor.io -> 3 parts (subdomain exists)
  // alpha.localhost -> 2 parts but localhost special case
  
  if (parts.length === 1) {
    // Just 'localhost' or single word hostname
    return null;
  }
  
  if (parts.length === 2 && parts[1] === 'localhost') {
    // Special case: alpha.localhost
    return parts[0];
  }
  
  if (parts.length >= 3) {
    // Normal subdomain case: alpha.securestor.io
    return parts[0];
  }
  
  return null;
};

/**
 * Get current tenant slug
 * Priority:
 * 1. Manual override from localStorage (dev mode only)
 * 2. Subdomain extraction
 * 3. Default tenant
 * 
 * @returns {string} The current tenant slug
 */
export const getCurrentTenant = () => {
  // Check for dev mode override
  if (CONFIG.DEV_MODE) {
    const override = localStorage.getItem(CONFIG.STORAGE_KEY);
    if (override) {
      return override;
    }
  }
  
  // Extract from subdomain
  const subdomain = extractTenantFromSubdomain();
  if (subdomain) {
    return subdomain;
  }
  
  // Return default
  return CONFIG.DEFAULT_TENANT;
};

/**
 * Set tenant manually (development mode only)
 * Stores tenant in localStorage to override subdomain detection
 * 
 * @param {string} tenantSlug - The tenant slug to set
 */
export const setDevTenant = (tenantSlug) => {
  if (!CONFIG.DEV_MODE) {
    return;
  }
  
  localStorage.setItem(CONFIG.STORAGE_KEY, tenantSlug);
  
  // Reload the page to apply changes
  window.location.reload();
};

/**
 * Clear dev tenant override
 */
export const clearDevTenant = () => {
  localStorage.removeItem(CONFIG.STORAGE_KEY);
};

/**
 * Get tenant info object
 * Returns comprehensive tenant context information
 * 
 * @returns {Object} Tenant information
 */
export const getTenantInfo = () => {
  const hostname = window.location.hostname;
  const subdomain = extractTenantFromSubdomain(hostname);
  const currentTenant = getCurrentTenant();
  const devOverride = CONFIG.DEV_MODE ? localStorage.getItem(CONFIG.STORAGE_KEY) : null;
  
  return {
    slug: currentTenant,
    subdomain: subdomain,
    hostname: hostname,
    baseDomain: CONFIG.BASE_DOMAIN,
    defaultTenant: CONFIG.DEFAULT_TENANT,
    isDefault: currentTenant === CONFIG.DEFAULT_TENANT,
    hasSubdomain: subdomain !== null,
    devMode: CONFIG.DEV_MODE,
    devOverride: devOverride,
  };
};

/**
 * Build tenant URL
 * Constructs a URL for a specific tenant
 * 
 * @param {string} tenantSlug - The tenant slug
 * @param {string} path - Optional path to append
 * @returns {string} The full URL
 */
export const buildTenantUrl = (tenantSlug, path = '/') => {
  const protocol = window.location.protocol;
  const port = window.location.port ? `:${window.location.port}` : '';
  
  // Handle localhost specially
  if (CONFIG.BASE_DOMAIN === 'localhost') {
    return `${protocol}//${tenantSlug}.localhost${port}${path}`;
  }
  
  // Production domain
  return `${protocol}//${tenantSlug}.${CONFIG.BASE_DOMAIN}${port}${path}`;
};

/**
 * Switch to another tenant
 * Redirects browser to the specified tenant's URL
 * 
 * @param {string} tenantSlug - The tenant to switch to
 * @param {boolean} preservePath - Whether to keep current path (default: true)
 */
export const switchToTenant = (tenantSlug, preservePath = true) => {
  const path = preservePath ? window.location.pathname : '/';
  const targetUrl = buildTenantUrl(tenantSlug, path);
  
  // Clear any dev overrides
  clearDevTenant();
  
  // Redirect
  window.location.href = targetUrl;
};

/**
 * Validate tenant slug format
 * Ensures tenant slug follows naming conventions
 * 
 * @param {string} slug - The slug to validate
 * @returns {boolean} True if valid
 */
export const isValidTenantSlug = (slug) => {
  if (!slug || typeof slug !== 'string') {
    return false;
  }
  
  // Must be lowercase alphanumeric with hyphens
  // Must start and end with alphanumeric
  // Length: 3-63 characters
  const regex = /^[a-z0-9]([a-z0-9-]{1,61}[a-z0-9])?$/;
  return regex.test(slug);
};

/**
 * Get tenant-specific API base URL
 * Returns the API URL with tenant context preserved
 * Handles development (React on :3000, backend on :80) and production scenarios
 * 
 * @returns {string} The API base URL
 */
export const getTenantApiUrl = () => {
  const protocol = window.location.protocol;
  const hostname = window.location.hostname;
  const currentPort = window.location.port;
  
  // For localhost development, always use base hostname without subdomain
  // The backend uses X-Tenant-Slug header for tenant routing, not subdomains
  let apiHostname = hostname;
  if (hostname.includes('.localhost')) {
    // Remove subdomain from localhost URLs (e.g., default.localhost -> localhost)
    apiHostname = 'localhost';
  } else if (hostname !== 'localhost' && hostname.split('.').length > 2) {
    // For production domains with subdomains, keep the subdomain
    // This is for cases where API is also on subdomain
    apiHostname = hostname;
  }
  
  // Determine target port
  let targetPort = '';
  
  if (currentPort === '3000') {
    // Development: React dev server on 3000, backend on configured port (default 80)
    if (CONFIG.BACKEND_PORT) {
      targetPort = `:${CONFIG.BACKEND_PORT}`;
    }
    // If BACKEND_PORT is empty, use default (port 80, no suffix needed)
  } else {
    // Production: same port as current
    targetPort = currentPort ? `:${currentPort}` : '';
  }
  
  return `${protocol}//${apiHostname}${targetPort}`;
};

/**
 * Check if running on tenant subdomain
 * @returns {boolean}
 */
export const isOnTenantSubdomain = () => {
  return extractTenantFromSubdomain() !== null;
};

/**
 * Get tenant headers for API requests
 * Returns headers object with tenant context
 * 
 * @returns {Object} Headers object
 */
export const getTenantHeaders = () => {
  const tenant = getCurrentTenant();
  
  return {
    'X-Tenant-Slug': tenant,
  };
};

/**
 * Debug utility - logs current tenant state (disabled in production)
 */
export const debugTenantInfo = () => {
  if (process.env.NODE_ENV === 'development') {
    const info = getTenantInfo();
    console.group('[Tenant Debug Info]');
    console.table(info);
    console.groupEnd();
  }
};

// Export configuration for advanced use cases
export const tenantConfig = CONFIG;

export default {
  extractTenantFromSubdomain,
  getCurrentTenant,
  setDevTenant,
  clearDevTenant,
  getTenantInfo,
  buildTenantUrl,
  switchToTenant,
  isValidTenantSlug,
  getTenantApiUrl,
  isOnTenantSubdomain,
  getTenantHeaders,
  debugTenantInfo,
  config: CONFIG,
};
