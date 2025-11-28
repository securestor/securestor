import { getAPIBaseURL, getTenantAwareHeaders, getCurrentTenantSlug } from '../constants/apiConfig';

/**
 * Enterprise SaaS Multi-Tenant API Client
 * Automatically handles tenant context via subdomain detection
 * and injects appropriate headers for all requests
 */
class APIClient {
  constructor() {
    // API endpoints are versioned (/api/v1/*)
    const baseOrigin = process.env.REACT_APP_API_URL || getAPIBaseURL();
    this.baseURL = baseOrigin + "/api/v1";
    this.tenantId = null; // Legacy support
    this.tenantSlug = null; // Current tenant slug
    this.getTenantIdCallback = null; // Callback to get current tenant ID
  }

  /**
   * Legacy method - kept for backward compatibility
   * @deprecated Use tenant context from subdomain instead
   */
  setTenantId(tenantId) {
    console.warn('[API] setTenantId is deprecated. Tenant is now detected from subdomain.');
    this.tenantId = tenantId;
  }

  /**
   * Set tenant slug manually (for special cases)
   * Generally not needed as tenant is auto-detected from subdomain
   */
  setTenantSlug(tenantSlug) {
    this.tenantSlug = tenantSlug;
  }

  /**
   * Set callback to get current tenant ID dynamically
   * This allows the API client to get the tenant ID from React context
   */
  setTenantIdCallback(callback) {
    this.getTenantIdCallback = callback;
  }

  /**
   * Get current tenant ID
   * Uses callback if set, otherwise returns manually set ID
   */
  getCurrentTenantId() {
    if (this.getTenantIdCallback) {
      return this.getTenantIdCallback();
    }
    return this.tenantId;
  }

  /**
   * Get current tenant slug
   * Priority: manual override -> subdomain detection
   */
  getCurrentTenantSlug() {
    return this.tenantSlug || getCurrentTenantSlug();
  }

  /**
   * Make HTTP request with automatic tenant context
   */
  async request(endpoint, options = {}) {
    const url = `${this.baseURL}${endpoint}`;
    
    // Start with default headers
    const headers = {
      'Content-Type': 'application/json',
      ...options.headers,
    };

    // Add authentication token
    const token = localStorage.getItem('auth_token') || sessionStorage.getItem('auth_token');
    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }

    // Add tenant headers automatically
    const tenantHeaders = getTenantAwareHeaders();
    Object.assign(headers, tenantHeaders);

    // Add X-Tenant-ID if available
    const currentTenantId = this.getCurrentTenantId();
    if (currentTenantId) {
      headers['X-Tenant-ID'] = currentTenantId;
    }
    
    const config = {
      headers,
      ...options,
    };

    if (config.body && typeof config.body === 'object') {
      config.body = JSON.stringify(config.body);
    }

    try {
      const response = await fetch(url, config);
      
      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.message || `HTTP error! status: ${response.status}`);
      }

      const contentType = response.headers.get('content-type');
      if (contentType && contentType.includes('application/json')) {
        return await response.json();
      }
      
      return await response.text();
    } catch (error) {
      console.error('[API] Request failed:', {
        endpoint,
        tenant: this.getCurrentTenantSlug(),
        error: error.message,
      });
      throw error;
    }
  }

  async get(endpoint, options = {}) {
    return this.request(endpoint, { ...options, method: 'GET' });
  }

  async post(endpoint, data, options = {}) {
    return this.request(endpoint, {
      ...options,
      method: 'POST',
      body: data,
    });
  }

  async put(endpoint, data, options = {}) {
    return this.request(endpoint, {
      ...options,
      method: 'PUT',
      body: data,
    });
  }

  async patch(endpoint, data, options = {}) {
    return this.request(endpoint, {
      ...options,
      method: 'PATCH',
      body: data,
    });
  }

  async delete(endpoint, options = {}) {
    return this.request(endpoint, { ...options, method: 'DELETE' });
  }
}

export const api = new APIClient();