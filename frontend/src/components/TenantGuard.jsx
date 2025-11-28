import React, { useEffect, useState } from 'react';
import { getCurrentTenant, getTenantInfo, isValidTenantSlug } from '../utils/tenant';
import { api } from '../services/api';

/**
 * TenantGuard - Validates tenant existence before rendering children
 * Prevents access to non-existent tenants by checking with backend
 * Shows loading state during validation and error page for invalid tenants
 */
export const TenantGuard = ({ children }) => {
  const [validating, setValidating] = useState(true);
  const [tenantExists, setTenantExists] = useState(null);
  const [tenantSlug, setTenantSlug] = useState(null);
  const [error, setError] = useState(null);

  useEffect(() => {
    const validateTenant = async () => {
      try {
        setValidating(true);
        
        // Get current tenant from subdomain
        const slug = getCurrentTenant();
        const info = getTenantInfo();
        
        setTenantSlug(slug);

        // Validate slug format
        if (!isValidTenantSlug(slug)) {
          console.error('[TenantGuard] Invalid tenant slug format:', slug);
          setTenantExists(false);
          setError(`Invalid tenant name format: "${slug}"`);
          setValidating(false);
          return;
        }

        // Call backend to verify tenant exists
        // This endpoint should be public (no auth required)
        try {
          const response = await api.get(`/tenants/validate/${slug}`);
          
          if (response && response.exists === true) {
            setTenantExists(true);
            setError(null);
          } else {
            console.warn('[TenantGuard] Tenant does not exist:', slug);
            setTenantExists(false);
            setError(`Tenant "${slug}" not found`);
          }
        } catch (err) {
          // If endpoint doesn't exist yet, log and allow through
          // TODO: Remove this fallback once backend endpoint is implemented
          if (err.response?.status === 404) {
            console.warn('[TenantGuard] Validation endpoint not found, allowing through');
            setTenantExists(true);
          } else {
            console.error('[TenantGuard] Validation failed:', err);
            setTenantExists(false);
            setError(`Failed to validate tenant: ${err.message}`);
          }
        }

      } catch (err) {
        console.error('[TenantGuard] Unexpected error during validation:', err);
        setError(err.message);
        setTenantExists(false);
      } finally {
        setValidating(false);
      }
    };

    validateTenant();
  }, []);

  // Loading state
  if (validating) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto mb-4"></div>
          <p className="text-gray-600">Validating tenant...</p>
        </div>
      </div>
    );
  }

  // Invalid tenant error page
  if (!tenantExists) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <div className="max-w-md w-full bg-white shadow-lg rounded-lg p-8 text-center">
          <div className="mb-6">
            <svg 
              className="mx-auto h-16 w-16 text-red-500" 
              fill="none" 
              stroke="currentColor" 
              viewBox="0 0 24 24"
            >
              <path 
                strokeLinecap="round" 
                strokeLinejoin="round" 
                strokeWidth={2} 
                d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" 
              />
            </svg>
          </div>
          
          <h1 className="text-2xl font-bold text-gray-900 mb-2">
            Tenant Not Found
          </h1>
          
          <p className="text-gray-600 mb-6">
            {error || `The tenant "${tenantSlug}" does not exist or has been deactivated.`}
          </p>
          
          <div className="space-y-3">
            <button
              onClick={() => window.location.href = '/'}
              className="w-full bg-blue-600 hover:bg-blue-700 text-white font-medium py-2 px-4 rounded transition-colors"
            >
              Go to Default Tenant
            </button>
            
            <button
              onClick={() => window.location.reload()}
              className="w-full bg-gray-200 hover:bg-gray-300 text-gray-700 font-medium py-2 px-4 rounded transition-colors"
            >
              Retry
            </button>
          </div>
          
          <div className="mt-6 text-sm text-gray-500">
            <p>If you believe this is an error, please contact your administrator.</p>
          </div>
        </div>
      </div>
    );
  }

  // Tenant is valid, render children
  return <>{children}</>;
};

export default TenantGuard;
