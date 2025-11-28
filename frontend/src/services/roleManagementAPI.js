// API service for role management
import { getAPIBaseURL } from '../constants/apiConfig';

class RoleManagementAPI {
  constructor() {
    // baseURL should be '/api/v1' for versioned endpoints
    this.baseURL = (process.env.REACT_APP_API_URL || getAPIBaseURL()) + '/api/v1';
  }

  // Get JWT token from localStorage
  getAuthToken() {
    return localStorage.getItem('auth_token') || sessionStorage.getItem('auth_token') || '';
  }

  // Get common headers for API requests
  getHeaders() {
    return {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${this.getAuthToken()}`
    };
  }

  // Handle API response
  async handleResponse(response) {
    if (!response.ok) {
      const error = await response.json().catch(() => ({ message: 'Network error' }));
      throw new Error(error.message || `HTTP ${response.status}`);
    }
    
    const contentType = response.headers.get('content-type');
    if (contentType && contentType.includes('application/json')) {
      return await response.json();
    }
    return null;
  }

  // Fetch all roles
  async getRoles(params = {}) {
    const searchParams = new URLSearchParams({
      include_system: params.includeSystem ?? true,
      limit: params.limit || 50,
      offset: params.offset || 0,
      ...params.search && { search: params.search }
    });

    const response = await fetch(`${this.baseURL}/roles?${searchParams}`, {
      headers: this.getHeaders()
    });

    return this.handleResponse(response);
  }

  // Get a specific role by ID
  async getRole(roleId) {
    const response = await fetch(`${this.baseURL}/roles/${roleId}`, {
      headers: this.getHeaders()
    });

    return this.handleResponse(response);
  }

  // Create a new role
  async createRole(roleData) {
    const response = await fetch(`${this.baseURL}/roles`, {
      method: 'POST',
      headers: this.getHeaders(),
      body: JSON.stringify(roleData)
    });

    return this.handleResponse(response);
  }

  // Update an existing role
  async updateRole(roleId, roleData) {
    const response = await fetch(`${this.baseURL}/roles/${roleId}`, {
      method: 'PUT',
      headers: this.getHeaders(),
      body: JSON.stringify(roleData)
    });

    return this.handleResponse(response);
  }

  // Delete a role
  async deleteRole(roleId) {
    const response = await fetch(`${this.baseURL}/roles/${roleId}`, {
      method: 'DELETE',
      headers: this.getHeaders()
    });

    return this.handleResponse(response);
  }

  // Fetch all permissions
  async getPermissions(params = {}) {
    const searchParams = new URLSearchParams({
      limit: params.limit || 100,
      offset: params.offset || 0,
      ...params.resource && { resource: params.resource },
      ...params.action && { action: params.action }
    });

    const response = await fetch(`${this.baseURL}/permissions?${searchParams}`, {
      headers: this.getHeaders()
    });

    return this.handleResponse(response);
  }

  // Assign permissions to a role
  async assignRolePermissions(roleId, permissionIds) {
    const response = await fetch(`${this.baseURL}/roles/${roleId}/permissions`, {
      method: 'PUT',
      headers: this.getHeaders(),
      body: JSON.stringify({ permission_ids: permissionIds })
    });

    return this.handleResponse(response);
  }

  // Remove permissions from a role
  async removeRolePermissions(roleId, permissionIds) {
    const response = await fetch(`${this.baseURL}/roles/${roleId}/permissions`, {
      method: 'DELETE',
      headers: this.getHeaders(),
      body: JSON.stringify({ permission_ids: permissionIds })
    });

    return this.handleResponse(response);
  }

  // Get users assigned to a role
  async getRoleUsers(roleId) {
    const response = await fetch(`${this.baseURL}/roles/${roleId}/users`, {
      headers: this.getHeaders()
    });

    return this.handleResponse(response);
  }

  // Assign a role to a user
  async assignUserRole(userId, roleId) {
    const response = await fetch(`${this.baseURL}/users/${userId}/roles`, {
      method: 'POST',
      headers: this.getHeaders(),
      body: JSON.stringify({ role_id: roleId })
    });

    return this.handleResponse(response);
  }

  // Remove a role from a user
  async removeUserRole(userId, roleId) {
    const response = await fetch(`${this.baseURL}/users/${userId}/roles/${roleId}`, {
      method: 'DELETE',
      headers: this.getHeaders()
    });

    return this.handleResponse(response);
  }

  // Assign users to a role (batch operation)
  async assignRoleUsers(roleId, userIds) {
    // Get current users with this role
    const currentUsers = await this.getRoleUsers(roleId);
    const currentUserIds = new Set((currentUsers.users || []).map(u => u.id));
    const targetUserIds = new Set(userIds);

    // Users to add (in target but not in current)
    const usersToAdd = userIds.filter(id => !currentUserIds.has(id));
    
    // Users to remove (in current but not in target)
    const usersToRemove = (currentUsers.users || [])
      .map(u => u.id)
      .filter(id => !targetUserIds.has(id));

    // Execute assignments and removals
    const promises = [];
    
    usersToAdd.forEach(userId => {
      promises.push(this.assignUserRole(userId, roleId));
    });
    
    usersToRemove.forEach(userId => {
      promises.push(this.removeUserRole(userId, roleId));
    });

    await Promise.all(promises);
  }

  // Remove users from a role
  async removeRoleUsers(roleId, userIds) {
    const promises = userIds.map(userId => this.removeUserRole(userId, roleId));
    await Promise.all(promises);
  }

  // Get all users (for role assignment)
  async getUsers(params = {}) {
    const searchParams = new URLSearchParams({
      limit: params.limit || 50,
      offset: params.offset || 0,
      ...params.search && { search: params.search }
    });

    const response = await fetch(`${this.baseURL}/users?${searchParams}`, {
      headers: this.getHeaders()
    });

    return this.handleResponse(response);
  }

  // Login (for testing - you may already have this elsewhere)
  async login(credentials) {
    const response = await fetch(`${this.baseURL}/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(credentials)
    });

    const data = await this.handleResponse(response);
    
    if (data.token) {
      localStorage.setItem('auth_token', data.token);
    }
    
    return data;
  }

  // Logout
  logout() {
    localStorage.removeItem('auth_token');
    sessionStorage.removeItem('auth_token');
  }

  // Check if user is authenticated
  isAuthenticated() {
    return !!this.getAuthToken();
  }
}

// Create a singleton instance
const roleManagementAPI = new RoleManagementAPI();

export default roleManagementAPI;
export { RoleManagementAPI };