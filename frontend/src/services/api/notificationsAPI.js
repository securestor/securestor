/**
 * Enterprise Notifications API Client
 * Handles all notification-related operations with comprehensive features
 */
import { getAPIBaseURL } from '../../constants/apiConfig';

class NotificationsAPI {
  constructor() {
    const baseOrigin = process.env.REACT_APP_API_URL || getAPIBaseURL();
    this.baseURL = baseOrigin + '/api/v1/notifications';
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

  /**
   * Get all notifications for current user
   * @param {Object} params - Query parameters
   * @param {number} params.page - Page number
   * @param {number} params.limit - Items per page
   * @param {string} params.type - Filter by type (security, system, compliance, encryption, scan, artifact)
   * @param {string} params.priority - Filter by priority (critical, high, medium, low, info)
   * @param {boolean} params.unread - Show only unread
   * @returns {Promise} Notifications response
   */
  async getNotifications(params = {}) {
    const queryParams = new URLSearchParams({
      page: params.page || 1,
      limit: params.limit || 20,
      ...(params.type && { type: params.type }),
      ...(params.priority && { priority: params.priority }),
      ...(params.unread !== undefined && { unread: params.unread })
    });

    const response = await fetch(`${this.baseURL}?${queryParams}`, {
      method: 'GET',
      headers: this.getHeaders()
    });
    return response.json();
  }

  /**
   * Get unread notification count
   * @returns {Promise} Count response
   */
  async getUnreadCount() {
    const response = await fetch(`${this.baseURL}/unread/count`, {
      method: 'GET',
      headers: this.getHeaders()
    });
    const data = await response.json();
    return data.count || 0;
  }

  /**
   * Mark notification as read
   * @param {string} notificationId - Notification ID
   * @returns {Promise} Update response
   */
  async markAsRead(notificationId) {
    const response = await fetch(`${this.baseURL}/${notificationId}/read`, {
      method: 'PATCH',
      headers: this.getHeaders()
    });
    return response.json();
  }

  /**
   * Mark multiple notifications as read
   * @param {Array<string>} notificationIds - Array of notification IDs
   * @returns {Promise} Update response
   */
  async markMultipleAsRead(notificationIds) {
    const response = await fetch(`${this.baseURL}/bulk/read`, {
      method: 'POST',
      headers: this.getHeaders(),
      body: JSON.stringify({ notification_ids: notificationIds })
    });
    return response.json();
  }

  /**
   * Mark all notifications as read
   * @returns {Promise} Update response
   */
  async markAllAsRead() {
    const response = await fetch(`${this.baseURL}/all/read`, {
      method: 'POST',
      headers: this.getHeaders()
    });
    return response.json();
  }

  /**
   * Delete notification
   * @param {string} notificationId - Notification ID
   * @returns {Promise} Delete response
   */
  async deleteNotification(notificationId) {
    const response = await fetch(`${this.baseURL}/${notificationId}`, {
      method: 'DELETE',
      headers: this.getHeaders()
    });
    return response.json();
  }

  /**
   * Delete multiple notifications
   * @param {Array<string>} notificationIds - Array of notification IDs
   * @returns {Promise} Delete response
   */
  async deleteMultiple(notificationIds) {
    const response = await fetch(`${this.baseURL}/bulk/delete`, {
      method: 'POST',
      headers: this.getHeaders(),
      body: JSON.stringify({ notification_ids: notificationIds })
    });
    return response.json();
  }

  /**
   * Get notification preferences
   * @returns {Promise} Preferences response
   */
  async getPreferences() {
    const response = await fetch(`${this.baseURL}/preferences`, {
      method: 'GET',
      headers: this.getHeaders()
    });
    return response.json();
  }

  /**
   * Update notification preferences
   * @param {Object} preferences - Notification preferences
   * @returns {Promise} Update response
   */
  async updatePreferences(preferences) {
    const response = await fetch(`${this.baseURL}/preferences`, {
      method: 'PUT',
      headers: this.getHeaders(),
      body: JSON.stringify(preferences)
    });
    return response.json();
  }

  /**
   * Get notification statistics
   * @param {string} period - Time period (today, week, month)
   * @returns {Promise} Statistics response
   */
  async getStatistics(period = 'week') {
    const response = await fetch(`${this.baseURL}/statistics?period=${period}`, {
      method: 'GET',
      headers: this.getHeaders()
    });
    return response.json();
  }

  /**
   * Subscribe to real-time notifications (WebSocket)
   * @param {Function} onMessage - Callback for new notifications
   * @returns {WebSocket} WebSocket connection
   */
  subscribeToNotifications(onMessage) {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const ws = new WebSocket(`${protocol}//${window.location.host}/api/v1/notifications/ws`);
    
    ws.onmessage = (event) => {
      try {
        const notification = JSON.parse(event.data);
        onMessage(notification);
      } catch (error) {
        console.error('Failed to parse notification:', error);
      }
    };

    ws.onerror = (error) => {
      console.error('WebSocket error:', error);
    };

    return ws;
  }
}

export default new NotificationsAPI();
