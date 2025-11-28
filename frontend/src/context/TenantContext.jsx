import React, { createContext, useContext, useState, useEffect, useCallback } from 'react';
import { useAuth } from './AuthContext';
import { api } from '../services/api';
import { 
  getCurrentTenant, 
  getTenantInfo, 
  switchToTenant as switchToTenantUrl,
  isValidTenantSlug 
} from '../utils/tenant';

const TenantContext = createContext();

/**
 * Hook to access the current tenant context
 * Provides tenant slug, tenant data, and loading/error states
 * Now uses subdomain-based tenant detection (Enterprise SaaS pattern)
 */
export const useTenant = () => {
  const context = useContext(TenantContext);
  if (!context) {
    throw new Error('useTenant must be used within a TenantProvider');
  }
  return context;
};

/**
 * Enterprise SaaS Multi-Tenant Provider
 * Manages tenant state with subdomain-based routing
 * Automatically detects tenant from URL subdomain
 * Caches tenant data in memory for performance
 */
export const TenantProvider = ({ children }) => {
  const { user, token } = useAuth();
  
  // Tenant state
  const [tenant, setTenant] = useState(null);
  const [tenantSlug, setTenantSlug] = useState(null);
  const [tenantId, setTenantId] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  
  // Tenant info from subdomain
  const [tenantInfo, setTenantInfo] = useState(getTenantInfo());

  /**
   * Fetch tenant data from API based on slug
   * Caches result in state to avoid repeated API calls
   */
  const fetchTenantData = useCallback(async (slug) => {
    if (!slug || !token) return null;

    try {
      
      // Call backend API to validate/resolve tenant by slug
      const response = await api.get(`/tenants/validate/${slug}`, {
        headers: {
          'Authorization': `Bearer ${token}`,
        },
      });

      // The validate endpoint returns: { exists, slug, tenant_id, tenant_name }
      // Transform it to match expected format
      if (response && response.exists) {
        return {
          id: response.tenant_id,
          tenant_id: response.tenant_id,
          slug: response.slug,
          name: response.tenant_name
        };
      }
      
      return null;
    } catch (err) {
      console.error('[Tenant] Failed to fetch tenant data:', err);
      throw err;
    }
  }, [token]);

  /**
   * Initialize tenant from subdomain
   * Runs once on mount and when subdomain changes
   */
  useEffect(() => {
    const initializeTenant = async () => {
      try {
        setLoading(true);
        setError(null);

        // Get current tenant slug from subdomain
        const currentSlug = getCurrentTenant();
        const info = getTenantInfo();
        
        
        setTenantSlug(currentSlug);
        setTenantInfo(info);

        // Validate slug format
        if (!isValidTenantSlug(currentSlug)) {
          console.warn('[Tenant] Invalid tenant slug format:', currentSlug);
        }

        // If user is authenticated, fetch full tenant data
        if (token) {
          try {
            const tenantData = await fetchTenantData(currentSlug);
            
            if (tenantData) {
              setTenant(tenantData);
              setTenantId(tenantData.tenant_id || tenantData.id);
            }
          } catch (err) {
            // Non-fatal: we can still work with slug-only
            console.warn('[Tenant] Could not load full tenant data, using slug only');
            setError('Could not load tenant details');
          }
        } else {
          // Not authenticated yet - just use slug
        }

      } catch (err) {
        console.error('[Tenant] Initialization failed:', err);
        setError(err.message);
      } finally {
        setLoading(false);
      }
    };

    initializeTenant();
  }, [token, fetchTenantData]);

  /**
   * Update API client with tenant slug whenever it changes
   */
  useEffect(() => {
    if (tenantSlug) {
      api.setTenantSlug(tenantSlug);
    }
  }, [tenantSlug]);

  /**
   * Update API client with tenant ID callback
   * This allows API client to dynamically get the current tenant ID
   * Also store in localStorage for non-api client usage (like artifactAPI)
   */
  useEffect(() => {
    api.setTenantIdCallback(() => tenantId);
    
    // Store in localStorage for other API clients to access
    if (tenantId) {
      localStorage.setItem('tenant_id', tenantId);
    } else {
      localStorage.removeItem('tenant_id');
    }
  }, [tenantId]);

  /**
   * Switch to a different tenant (via URL redirect)
   * Redirects browser to new tenant's subdomain
   */
  const switchTenant = useCallback((newTenantSlug, preservePath = true) => {
    if (!isValidTenantSlug(newTenantSlug)) {
      console.error('[Tenant] Invalid tenant slug:', newTenantSlug);
      return;
    }
    
    switchToTenantUrl(newTenantSlug, preservePath);
  }, []);

  /**
   * Refetch current tenant data from API
   * Useful after settings updates
   */
  const refetchTenant = useCallback(async () => {
    if (!tenantSlug || !token) {
      console.warn('[Tenant] Cannot refetch: missing slug or token');
      return;
    }

    try {
      setLoading(true);
      const tenantData = await fetchTenantData(tenantSlug);
      
      if (tenantData) {
        setTenant(tenantData);
        setTenantId(tenantData.tenant_id || tenantData.id);
      }
    } catch (err) {
      console.error('[Tenant] Refetch failed:', err);
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }, [tenantSlug, token, fetchTenantData]);

  /**
   * Check if tenant data is ready for use
   */
  const isReady = tenantSlug !== null && !loading;

  const value = {
    // Core tenant data
    tenant,
    tenantId,
    tenantSlug,
    tenantInfo,
    
    // State
    loading,
    error,
    isReady,
    
    // Actions
    switchTenant,
    refetchTenant,
    
    // Helpers
    isOnSubdomain: tenantInfo.hasSubdomain,
    isDefaultTenant: tenantInfo.isDefault,
  };

  return (
    <TenantContext.Provider value={value}>
      {children}
    </TenantContext.Provider>
  );
};

export default TenantContext;
