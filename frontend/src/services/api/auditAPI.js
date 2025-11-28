import { getAPIBaseURL } from "../../constants/apiConfig";

class AuditAPI {
  constructor() {
    // Audit endpoints are versioned (/api/v1/audit/*)
    const baseOrigin = process.env.REACT_APP_API_URL || getAPIBaseURL();
    this.baseURL = baseOrigin + '/api/v1';
  }

  // Get authentication headers
  getAuthHeaders() {
    const token = localStorage.getItem('auth_token') || sessionStorage.getItem('auth_token');
    return {
      'Content-Type': 'application/json',
      ...(token && { Authorization: `Bearer ${token}` })
    };
  }

  // Get audit logs with pagination and filtering
  async getAuditLogs(params = {}) {
    try {
      const queryParams = new URLSearchParams();
      
      // Add pagination parameters
      if (params.page) queryParams.append('page', params.page);
      if (params.limit) queryParams.append('limit', params.limit);
      
      // Add filtering parameters
      if (params.user_id) queryParams.append('user_id', params.user_id);
      if (params.event_type) queryParams.append('event_type', params.event_type);
      if (params.resource_type) queryParams.append('resource_type', params.resource_type);
      if (params.action) queryParams.append('action', params.action);
      if (params.success !== undefined) queryParams.append('success', params.success);
      if (params.start_time) queryParams.append('start_time', params.start_time);
      if (params.end_time) queryParams.append('end_time', params.end_time);

      const url = `${this.baseURL}/audit/logs${queryParams.toString() ? `?${queryParams.toString()}` : ''}`;
      
      const response = await fetch(url, {
        method: 'GET',
        headers: this.getAuthHeaders()
      });

      if (!response.ok) {
        throw new Error(`Failed to fetch audit logs: ${response.statusText}`);
      }

      return await response.json();
    } catch (error) {
      console.warn('Audit logs endpoint not available, using defaults:', error.message);
      // Return empty audit logs when endpoint is not available
      return {
        logs: [],
        total: 0,
        page: params.page || 1,
        limit: params.limit || 20,
        total_pages: 0
      };
    }
  }

  // Get audit log statistics
  async getAuditStats(params = {}) {
    try {
      const queryParams = new URLSearchParams();
      
      if (params.start_time) queryParams.append('start_time', params.start_time);
      if (params.end_time) queryParams.append('end_time', params.end_time);

      const url = `${this.baseURL}/audit/stats${queryParams.toString() ? `?${queryParams.toString()}` : ''}`;
      
      const response = await fetch(url, {
        method: 'GET',
        headers: this.getAuthHeaders()
      });

      if (!response.ok) {
        throw new Error(`Failed to fetch audit stats: ${response.statusText}`);
      }

      return await response.json();
    } catch (error) {
      console.warn('Audit stats endpoint not available, using defaults:', error.message);
      // Return default stats when endpoint is not available
      return {
        total_logs: 0,
        by_action: {},
        by_resource: {},
        by_user: {},
        recent_activity: []
      };
    }
  }

  // Get single audit log by ID
  async getAuditLogById(id) {
    try {
      const response = await fetch(`${this.baseURL}/audit/logs/${id}`, {
        method: 'GET',
        headers: this.getAuthHeaders()
      });

      if (!response.ok) {
        throw new Error(`Failed to fetch audit log: ${response.statusText}`);
      }

      return await response.json();
    } catch (error) {
      console.error('Failed to get audit log:', error);
      throw error;
    }
  }

  // Create new audit log entry
  async createAuditLog(logEntry) {
    try {
      const response = await fetch(`${this.baseURL}/audit/logs`, {
        method: 'POST',
        headers: this.getAuthHeaders(),
        body: JSON.stringify(logEntry)
      });

      if (!response.ok) {
        throw new Error(`Failed to create audit log: ${response.statusText}`);
      }

      return await response.json();
    } catch (error) {
      console.error('Failed to create audit log:', error);
      throw error;
    }
  }
}

export const auditAPI = new AuditAPI();