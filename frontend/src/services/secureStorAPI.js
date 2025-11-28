// Comprehensive API configuration for SecureStor
// Handles both versioned (/api/v1/) and non-versioned (/api/) endpoints

import { getAPIBaseURL } from '../constants/apiConfig';

class SecureStorAPI {
  constructor() {
    // Base URLs for different API groups
    this.baseURL = process.env.REACT_APP_API_URL || getAPIBaseURL();
    this.apiV1 = `${this.baseURL}/api/v1`;  // Versioned endpoints
    this.api = `${this.baseURL}/api`;       // Non-versioned endpoints
  }

  // Get JWT token from storage
  getAuthToken() {
    return localStorage.getItem('auth_token') || 
           sessionStorage.getItem('auth_token') || 
           localStorage.getItem('jwt_token') || '';
  }

  // Get common headers for API requests
  getHeaders() {
    const token = this.getAuthToken();
    const headers = {
      'Content-Type': 'application/json'
    };
    
    if (token) {
      headers.Authorization = `Bearer ${token}`;
    }
    
    return headers;
  }

  // Generic request method
  async request(url, options = {}) {
    const config = {
      headers: {
        ...this.getHeaders(),
        ...options.headers,
      },
      ...options,
    };

    if (config.body && typeof config.body === 'object' && !(config.body instanceof FormData)) {
      config.body = JSON.stringify(config.body);
    }

    try {
      const response = await fetch(url, config);
      
      if (!response.ok) {
        let errorData;
        try {
          errorData = await response.json();
        } catch {
          errorData = { message: `HTTP error! status: ${response.status}` };
        }
        throw new Error(errorData.message || errorData.error || `HTTP error! status: ${response.status}`);
      }

      // 204 No Content has no response body
      if (response.status === 204) {
        return {};
      }

      const contentType = response.headers.get('content-type');
      if (contentType && contentType.includes('application/json')) {
        return await response.json();
      }
      
      return await response.text();
    } catch (error) {
      console.error('API request failed:', error);
      throw error;
    }
  }

  // Convenience methods for different HTTP verbs
  async get(endpoint, options = {}) {
    return this.request(endpoint, { ...options, method: 'GET' });
  }

  async post(endpoint, data, options = {}) {
    return this.request(endpoint, {
      ...options,
      method: 'POST',
      body: data
    });
  }

  async put(endpoint, data, options = {}) {
    return this.request(endpoint, {
      ...options,
      method: 'PUT',
      body: data
    });
  }

  async patch(endpoint, data, options = {}) {
    return this.request(endpoint, {
      ...options,
      method: 'PATCH',
      body: data
    });
  }

  async delete(endpoint, options = {}) {
    return this.request(endpoint, { ...options, method: 'DELETE' });
  }

  // Authentication APIs (non-versioned /api/)
  auth = {
    // Get current user
    getCurrentUser: () => this.get(`${this.api}/auth/me`),
    
    // Login
    login: (credentials) => this.post(`${this.api}/auth/login`, credentials),
    
    // Logout
    logout: () => this.post(`${this.api}/auth/logout`),
    
    // Change password
    changePassword: (data) => this.post(`${this.api}/auth/change-password`, data),
  };

  // User Management APIs (non-versioned /api/)
  users = {
    // List users
    list: (params = {}) => {
      const query = new URLSearchParams(params).toString();
      return this.get(`${this.api}/users${query ? `?${query}` : ''}`);
    },
    
    // Get user by ID
    getById: (userId) => this.get(`${this.api}/users/${userId}`),
    
    // Create user
    create: (userData) => this.post(`${this.api}/users`, userData),
    
    // Update user
    update: (userId, userData) => this.put(`${this.api}/users/${userId}`, userData),
    
    // Delete user
    delete: (userId) => this.delete(`${this.api}/users/${userId}`),
    
    // User roles
    getRoles: (userId) => this.get(`${this.api}/users/${userId}/roles`),
    assignRole: (userId, roleData) => this.post(`${this.api}/users/${userId}/roles`, roleData),
    removeRole: (userId, roleId) => this.delete(`${this.api}/users/${userId}/roles/${roleId}`),
    
    // User invitations
    invite: (inviteData) => this.post(`${this.api}/users/invite`, inviteData),
    getInvites: () => this.get(`${this.api}/users/invites`),
    acceptInvite: (token, userData) => this.post(`${this.api}/users/invites/${token}/accept`, userData),
    resendInvite: (inviteId) => this.post(`${this.api}/users/invites/${inviteId}/resend`),
  };

  // Role Management APIs (non-versioned /api/)
  roles = {
    // List all roles
    list: () => this.get(`${this.api}/roles`),
    
    // Get role by ID  
    getById: (roleId) => this.get(`${this.api}/roles/${roleId}`),
    
    // Create role
    create: (roleData) => this.post(`${this.api}/roles`, roleData),
    
    // Update role
    update: (roleId, roleData) => this.put(`${this.api}/roles/${roleId}`, roleData),
    
    // Delete role
    delete: (roleId) => this.delete(`${this.api}/roles/${roleId}`),
    
    // Role permissions
    getPermissions: (roleId) => this.get(`${this.api}/roles/${roleId}/permissions`),
    
    // Get simple roles (for dropdowns)
    getSimple: () => this.get(`${this.api}/users/roles`),
  };

  // Artifact APIs (versioned /api/v1/)
  artifacts = {
    // List artifacts
    list: (params = {}) => {
      const query = new URLSearchParams(params).toString();
      return this.get(`${this.apiV1}/artifacts${query ? `?${query}` : ''}`);
    },
    
    // Get artifact by ID
    getById: (artifactId) => this.get(`${this.apiV1}/artifacts/${artifactId}`),
    
    // Upload artifact
    upload: (repositoryId, formData) => {
      return this.request(`${this.apiV1}/repositories/${repositoryId}/artifacts`, {
        method: 'POST',
        body: formData,
        headers: {
          // Don't set Content-Type, let browser set it with boundary for FormData
          Authorization: this.getHeaders().Authorization
        }
      });
    },
    
    // Delete artifact
    delete: (artifactId) => this.delete(`${this.apiV1}/artifacts/${artifactId}`),
    
    // Search artifacts
    search: (searchParams) => this.post(`${this.apiV1}/artifacts/search`, searchParams),
  };

  // Repository APIs (versioned /api/v1/) 
  repositories = {
    // List repositories
    list: (params = {}) => {
      const query = new URLSearchParams(params).toString();
      return this.get(`${this.apiV1}/repositories${query ? `?${query}` : ''}`);
    },
    
    // Get repository by ID
    getById: (repositoryId) => this.get(`${this.apiV1}/repositories/${repositoryId}`),
    
    // Create repository
    create: (repoData) => this.post(`${this.apiV1}/repositories`, repoData),
    
    // Update repository
    update: (repositoryId, repoData) => this.put(`${this.apiV1}/repositories/${repositoryId}`, repoData),
    
    // Delete repository
    delete: (repositoryId) => this.delete(`${this.apiV1}/repositories/${repositoryId}`),
  };

  // Health check and system APIs
  health = {
    check: () => this.get(`${this.api}/health`),
    detailed: () => this.get(`${this.apiV1}/health/detailed`),
  };
}

// Export singleton instance
export const secureStorAPI = new SecureStorAPI();

// Export class for custom instances
export default SecureStorAPI;