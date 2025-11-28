import { api } from './api';

// Helper function to get auth headers
const getAuthHeaders = () => {
  const token = localStorage.getItem('auth_token');
  return token ? { 'Authorization': `Bearer ${token}` } : {};
};

export const tenantApi = {
  // Get list of tenants with filtering and pagination
  getTenants: async (params = {}) => {
    try {
      const queryParams = new URLSearchParams();
      
      if (params.page) queryParams.append('page', params.page);
      if (params.limit) queryParams.append('limit', params.limit);
      if (params.search) queryParams.append('search', params.search);
      if (params.status) queryParams.append('status', params.status);
      if (params.plan) queryParams.append('plan', params.plan);

      const response = await api.get(`/tenants?${queryParams.toString()}`, { headers: getAuthHeaders() });
      return response;
    } catch (error) {
      console.warn('Tenants list endpoint not available, using defaults:', error.message);
      // Return empty list when endpoint is not available
      return {
        tenants: [],
        total: 0,
        page: params.page || 1,
        limit: params.limit || 10
      };
    }
  },

  // Get single tenant by ID
  getTenant: async (tenantId) => {
    try {
      const response = await api.get(`/tenants/${tenantId}`, { headers: getAuthHeaders() });
      return response;
    } catch (error) {
      console.warn('Tenant detail endpoint not available:', error.message);
      // Return minimal tenant info
      return {
        id: tenantId,
        name: 'Unknown Tenant',
        slug: 'unknown',
        is_active: false
      };
    }
  },

  // Create new tenant
  createTenant: async (tenantData) => {
    const response = await api.post('/tenants', tenantData, { headers: getAuthHeaders() });
    return response;
  },

  // Update tenant
  updateTenant: async (tenantId, tenantData) => {
    const response = await api.put(`/tenants/${tenantId}`, tenantData, { headers: getAuthHeaders() });
    return response;
  },

  // Delete tenant (soft delete)
  deleteTenant: async (tenantId) => {
    const response = await api.delete(`/tenants/${tenantId}`, { headers: getAuthHeaders() });
    return response;
  },

  // Set tenant status (activate/deactivate)
  setTenantStatus: async (tenantId, isActive) => {
    const response = await api.patch(`/tenants/${tenantId}/status`, { is_active: isActive }, { headers: getAuthHeaders() });
    return response;
  },

  // Get tenant settings
  getTenantSettings: async (tenantId) => {
    try {
      const response = await api.get(`/tenants/${tenantId}/settings`, { headers: getAuthHeaders() });
      return response;
    } catch (error) {
      console.warn('Tenant settings endpoint not available, using defaults:', error.message);
      // Return default settings when endpoint is not available
      return {
        general: {
          name: '',
          description: '',
          contact_email: '',
          timezone: 'UTC',
          language: 'en'
        },
        security: {
          mfa_required: false,
          password_policy: {
            min_length: 8,
            require_uppercase: true,
            require_lowercase: true,
            require_numbers: true,
            require_special: false
          },
          session_timeout_minutes: 30,
          allowed_ip_ranges: []
        },
        storage: {
          max_storage_gb: 100,
          retention_days: 90,
          versioning_enabled: true
        },
        notifications: {
          email_enabled: true,
          webhook_enabled: false,
          webhook_url: ''
        },
        billing: {
          plan: 'free',
          billing_email: ''
        }
      };
    }
  },

  // Update tenant settings
  updateTenantSettings: async (tenantId, settings) => {
    try {
      const response = await api.put(`/tenants/${tenantId}/settings`, settings, { headers: getAuthHeaders() });
      return response;
    } catch (error) {
      console.warn('Tenant settings update endpoint not available:', error.message);
      // Return the settings that were sent (optimistic update)
      return settings;
    }
  },

  // Partially update tenant settings
  patchTenantSettings: async (tenantId, settingsPatch) => {
    try {
      const response = await api.patch(`/tenants/${tenantId}/settings`, settingsPatch, { headers: getAuthHeaders() });
      return response;
    } catch (error) {
      console.warn('Tenant settings patch endpoint not available:', error.message);
      // Return the patch that was sent (optimistic update)
      return settingsPatch;
    }
  },

  // Get tenant usage statistics
  getTenantUsage: async (tenantId) => {
    try {
      const response = await api.get(`/tenants/${tenantId}/usage`, { headers: getAuthHeaders() });
      return response;
    } catch (error) {
      console.warn('Tenant usage endpoint not available:', error.message);
      return {
        storage_used_gb: 0,
        artifacts_count: 0,
        repositories_count: 0
      };
    }
  },

  // Check if tenant can perform action (usage limits)
  checkUsageLimits: async (tenantId, action) => {
    try {
      const response = await api.post(`/tenants/${tenantId}/check-limits`, { action }, { headers: getAuthHeaders() });
      return response;
    } catch (error) {
      console.warn('Tenant usage limits endpoint not available:', error.message);
      return { allowed: true };
    }
  },

  // Get system-wide tenant statistics (admin only)
  getTenantStats: async () => {
    try {
      const response = await api.get('/admin/tenants/stats', { headers: getAuthHeaders() });
      return response;
    } catch (error) {
      console.warn('Tenant stats endpoint not available, using defaults:', error.message);
      return {
        total_tenants: 0,
        active_tenants: 0,
        inactive_tenants: 0,
        total_storage_gb: 0,
        total_artifacts: 0
      };
    }
  },

  // Enterprise Settings API Endpoints
  
  // Get specific settings sections
  getSecuritySettings: async (tenantId) => {
    const response = await api.get(`/tenants/${tenantId}/settings/security`, { headers: getAuthHeaders() });
    return response;
  },

  updateSecuritySettings: async (tenantId, securitySettings) => {
    const response = await api.put(`/tenants/${tenantId}/settings/security`, {
      security: securitySettings.security,
      advanced_security: securitySettings.advanced_security
    }, { headers: getAuthHeaders() });
    return response;
  },

  getComplianceSettings: async (tenantId) => {
    const response = await api.get(`/tenants/${tenantId}/settings/compliance`, { headers: getAuthHeaders() });
    return response;
  },

  updateComplianceSettings: async (tenantId, complianceSettings) => {
    const response = await api.put(`/tenants/${tenantId}/settings/compliance`, {
      compliance: complianceSettings
    }, { headers: getAuthHeaders() });
    return response;
  },

  getBillingSettings: async (tenantId) => {
    const response = await api.get(`/tenants/${tenantId}/settings/billing`, { headers: getAuthHeaders() });
    return response;
  },

  updateBillingSettings: async (tenantId, billingSettings) => {
    const response = await api.put(`/tenants/${tenantId}/settings/billing`, {
      billing: billingSettings
    }, { headers: getAuthHeaders() });
    return response;
  },

  getFeatureFlags: async (tenantId) => {
    const response = await api.get(`/tenants/${tenantId}/settings/features`, { headers: getAuthHeaders() });
    return response;
  },

  updateFeatureFlags: async (tenantId, featureFlags) => {
    const response = await api.put(`/tenants/${tenantId}/settings/features`, {
      feature_flags: featureFlags
    }, { headers: getAuthHeaders() });
    return response;
  },

  getMonitoringSettings: async (tenantId) => {
    const response = await api.get(`/tenants/${tenantId}/settings/monitoring`, { headers: getAuthHeaders() });
    return response;
  },

  updateMonitoringSettings: async (tenantId, monitoringSettings) => {
    const response = await api.put(`/tenants/${tenantId}/settings/monitoring`, {
      monitoring: monitoringSettings
    }, { headers: getAuthHeaders() });
    return response;
  },

  getAdvancedSecuritySettings: async (tenantId) => {
    const response = await api.get(`/tenants/${tenantId}/settings/advanced-security`, { headers: getAuthHeaders() });
    return response;
  },

  updateAdvancedSecuritySettings: async (tenantId, advancedSecuritySettings) => {
    const response = await api.put(`/tenants/${tenantId}/settings/advanced-security`, {
      advanced_security: advancedSecuritySettings
    }, { headers: getAuthHeaders() });
    return response;
  },

  // Settings validation and utilities
  validateTenantSettings: async (tenantId, settings) => {
    const response = await api.post(`/tenants/${tenantId}/settings/validate`, settings, { headers: getAuthHeaders() });
    return response;
  },

  resetTenantSettingsSection: async (tenantId, section) => {
    const response = await api.post(`/tenants/${tenantId}/settings/reset/${section}`, {}, { headers: getAuthHeaders() });
    return response;
  },

  updateTenantSettingsSection: async (tenantId, section, data) => {
    const response = await api.put(`/tenants/${tenantId}/settings/${section}`, data, { headers: getAuthHeaders() });
    return response;
  },

  // User management settings
  updateUserSettings: async (tenantId, userSettings) => {
    const response = await api.patch(`/tenants/${tenantId}/settings`, {
      user_management: userSettings
    }, { headers: getAuthHeaders() });
    return response;
  },

  // Storage settings
  updateStorageSettings: async (tenantId, storageSettings) => {
    const response = await api.patch(`/tenants/${tenantId}/settings`, {
      storage: storageSettings
    }, { headers: getAuthHeaders() });
    return response;
  },

  // Notification settings
  updateNotificationSettings: async (tenantId, notificationSettings) => {
    const response = await api.patch(`/tenants/${tenantId}/settings`, {
      notifications: notificationSettings
    }, { headers: getAuthHeaders() });
    return response;
  },

  // Integration settings
  updateIntegrationSettings: async (tenantId, integrationSettings) => {
    const response = await api.patch(`/tenants/${tenantId}/settings`, {
      integrations: integrationSettings
    }, { headers: getAuthHeaders() });
    return response;
  },

  // Tenant plan and feature management
  upgradePlan: async (tenantId, newPlan) => {
    const response = await api.patch(`/tenants/${tenantId}`, { plan: newPlan }, { headers: getAuthHeaders() });
    return response;
  },

  updateFeatures: async (tenantId, features) => {
    const response = await api.patch(`/tenants/${tenantId}`, { features }, { headers: getAuthHeaders() });
    return response;
  },

  // Billing and usage
  getBillingInfo: async (tenantId) => {
    const response = await api.get(`/tenants/${tenantId}/billing`, { headers: getAuthHeaders() });
    return response;
  },

  getUsageHistory: async (tenantId, period = '30d') => {
    const response = await api.get(`/tenants/${tenantId}/usage/history?period=${period}`, { headers: getAuthHeaders() });
    return response;
  },

  // Tenant domain and subdomain management
  updateDomain: async (tenantId, domain, subdomain) => {
    const response = await api.patch(`/tenants/${tenantId}`, { 
      domain: domain,
      subdomain: subdomain 
    }, { headers: getAuthHeaders() });
    return response;
  },

  // Tenant backup and export
  exportTenantData: async (tenantId, format = 'json') => {
    const response = await api.get(`/tenants/${tenantId}/export?format=${format}`, {
      responseType: 'blob',
      headers: getAuthHeaders()
    });
    return response;
  },

  // Tenant onboarding
  initializeTenant: async (tenantId, initData) => {
    const response = await api.post(`/tenants/${tenantId}/initialize`, initData, { headers: getAuthHeaders() });
    return response;
  },

  // Tenant health check
  getTenantHealth: async (tenantId) => {
    const response = await api.get(`/tenants/${tenantId}/health`, { headers: getAuthHeaders() });
    return response;
  }
};

export default tenantApi;