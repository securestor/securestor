// API service for statistics
import { getAPIBaseURL } from '../constants/apiConfig';

class StatsService {
  constructor() {
    // Stats endpoints are versioned (/api/v1/stats/*)
    const baseOrigin = process.env.REACT_APP_API_URL || getAPIBaseURL();
    this.baseURL = baseOrigin + '/api/v1';
    this.eventSource = null;
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

  // Get basic dashboard stats
  async getDashboardStats() {
    try {
      const response = await fetch(`${this.baseURL}/stats/dashboard`, {
        headers: this.getHeaders()
      });
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      const data = await response.json();
      return data;
    } catch (error) {
      console.error('Error fetching dashboard stats:', error);
      throw error;
    }
  }

  // Get detailed stats
  async getDetailedStats() {
    try {
      const response = await fetch(`${this.baseURL}/stats/detailed`, {
        headers: this.getHeaders()
      });
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      return await response.json();
    } catch (error) {
      console.error('Error fetching detailed stats:', error);
      throw error;
    }
  }

  // Get specific metric stats
  async getMetricStats(metricType) {
    try {
      const response = await fetch(`${this.baseURL}/stats/metrics?type=${metricType}`);
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      return await response.json();
    } catch (error) {
      console.error(`Error fetching ${metricType} stats:`, error);
      throw error;
    }
  }

  // Subscribe to real-time stats updates
  subscribeToRealtimeStats(callback, errorCallback) {
    if (this.eventSource) {
      this.eventSource.close();
    }

    try {
      this.eventSource = new EventSource(`${this.baseURL}/stats/realtime`);
      
      this.eventSource.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          callback(data);
        } catch (error) {
          console.error('Error parsing realtime stats:', error);
          if (errorCallback) errorCallback(error);
        }
      };

      this.eventSource.onerror = (error) => {
        console.error('EventSource failed:', error);
        if (errorCallback) errorCallback(error);
      };

      return () => this.unsubscribeFromRealtimeStats();
    } catch (error) {
      console.error('Error setting up EventSource:', error);
      if (errorCallback) errorCallback(error);
      return null;
    }
  }

  // Unsubscribe from real-time updates
  unsubscribeFromRealtimeStats() {
    if (this.eventSource) {
      this.eventSource.close();
      this.eventSource = null;
    }
  }

  // Format bytes for display
  formatBytes(bytes, decimals = 2) {
    if (bytes === 0) return '0 Bytes';

    const k = 1024;
    const dm = decimals < 0 ? 0 : decimals;
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB'];

    const i = Math.floor(Math.log(bytes) / Math.log(k));

    return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
  }

  // Format numbers with commas
  formatNumber(num) {
    return num.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ',');
  }

  // Convert trend string to display format
  formatTrend(trend) {
    if (!trend) return { value: '+0', isPositive: true, isNeutral: true };
    
    const isPositive = trend.startsWith('+');
    const isNegative = trend.startsWith('-');
    const isNeutral = trend === '+0' || trend === '0' || trend === '+0%' || trend === '0%';
    
    return {
      value: trend,
      isPositive: isPositive && !isNeutral,
      isNegative: isNegative,
      isNeutral: isNeutral
    };
  }

  // Transform backend stats to frontend format
  transformStatsForDisplay(backendStats) {
    if (!backendStats) return null;

    // Import icons dynamically
    const { Database, Package, Download, Users } = require('../constants/statsData');

    return {
      totalStorage: {
        label: 'Total Storage',
        value: backendStats.totalStorage || '0 B',
        icon: Database,
        trend: backendStats.trends?.storageTrend || '+0%',
        color: 'blue'
      },
      totalArtifacts: {
        label: 'Total Artifacts',
        value: this.formatNumber(backendStats.totalArtifacts || 0),
        icon: Package,
        trend: backendStats.trends?.artifactsTrend || '+0',
        color: 'green'
      },
      downloadsToday: {
        label: 'Downloads Today',
        value: this.formatNumber(backendStats.downloadsToday || 0),
        icon: Download,
        trend: backendStats.trends?.downloadsTrend || '+0%',
        color: 'purple'
      },
      activeUsers: {
        label: 'Active Users',
        value: this.formatNumber(backendStats.activeUsers || 0),
        icon: Users,
        trend: backendStats.trends?.activeUsersTrend || '+0',
        color: 'orange'
      },
      lastUpdated: backendStats.lastUpdated
    };
  }
}

// Create a singleton instance
const statsService = new StatsService();

export default statsService;
export { StatsService };