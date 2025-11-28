/**
 * API Configuration
 * Multi-tenant aware configuration for enterprise SaaS
 * Automatically handles subdomain-based tenant routing
 */

import { getTenantApiUrl, getCurrentTenant, getTenantHeaders } from '../utils/tenant';

/**
 * Get API base URL with tenant context
 * Priority:
 * 1. Environment variable REACT_APP_API_URL (for custom deployments)
 * 2. Current window origin (preserves subdomain)
 * 3. Fallback to localhost
 * 
 * @returns {string} The API base URL
 */
export const getAPIBaseURL = () => {
  // Check for environment variable first (allows override)
  if (process.env.REACT_APP_API_URL) {
    return process.env.REACT_APP_API_URL;
  }

  // If running in browser, use tenant-aware URL
  if (typeof window !== 'undefined') {
    return getTenantApiUrl();
  }

  // Fallback to localhost
  return 'http://localhost';
};

/**
 * Get current tenant slug
 * @returns {string} Current tenant identifier
 */
export const getCurrentTenantSlug = () => {
  return getCurrentTenant();
};

/**
 * Get tenant-aware headers for API requests
 * Automatically includes X-Tenant-Slug header
 * 
 * @param {Object} additionalHeaders - Any additional headers to merge
 * @returns {Object} Headers object with tenant context
 */
export const getTenantAwareHeaders = (additionalHeaders = {}) => {
  return {
    ...getTenantHeaders(),
    ...additionalHeaders,
  };
};

// Export the API base URL
export const API_BASE_URL = getAPIBaseURL();

// Common API endpoints
export const API_ENDPOINTS = {
  // Auth
  AUTH_LOGIN: `${API_BASE_URL}/api/auth/login`,
  AUTH_LOGOUT: `${API_BASE_URL}/api/auth/logout`,
  AUTH_PROFILE: `${API_BASE_URL}/api/auth/me`,
  AUTH_CHANGE_PASSWORD: `${API_BASE_URL}/api/auth/change-password`,
  
  // Users
  USERS: `${API_BASE_URL}/api/v1/users`,
  USER_PROFILE: `${API_BASE_URL}/api/auth/me`,
  USER_PREFERENCES: `${API_BASE_URL}/api/v1/preferences`,
  USER_INVITE: `${API_BASE_URL}/api/v1/users/invite`,
  
  // API Keys
  API_KEYS: `${API_BASE_URL}/api/v1/keys`,
  API_SCOPES: `${API_BASE_URL}/api/v1/scopes`,
  
  // Repositories
  REPOSITORIES: `${API_BASE_URL}/api/v1/repositories`,
  
  // Artifacts
  ARTIFACTS: `${API_BASE_URL}/api/v1/artifacts`,
  
  // Health
  HEALTH: `${API_BASE_URL}/api/v1/health`,
};

export default API_BASE_URL;
