/**
 * Replication Settings Service
 * Handles API communication for global replication configuration
 */

import { api } from './api';

const BASE_URL = '/settings/replication';

export const replicationSettingsService = {
  // Global Configuration
  async getGlobalConfig() {
    try {
      const response = await api.get(`${BASE_URL}/config`);
      return response;
    } catch (error) {
      console.warn('Replication settings endpoint not available, using defaults:', error.message);
      // Return default configuration when endpoint is not available
      return {
        enabled: false,
        mode: 'async',
        strategy: 'primary-secondary',
        consistency_level: 'eventual',
        retry_attempts: 3,
        retry_delay_ms: 1000,
        health_check_interval_ms: 30000,
        conflict_resolution: 'last-write-wins'
      };
    }
  },

  async updateGlobalConfig(configData) {
    try {
      const response = await api.put(`${BASE_URL}/config`, configData);
      return response;
    } catch (error) {
      console.error('Error updating global replication config:', error);
      throw error;
    }
  },

  // Replication Nodes
  async listNodes() {
    try {
      const response = await api.get(`${BASE_URL}/nodes`);
      return response;
    } catch (error) {
      console.warn('Replication nodes endpoint not available, using defaults:', error.message);
      // Return empty list when endpoint is not available
      return [];
    }
  },

  async createNode(nodeData) {
    try {
      const response = await api.post(`${BASE_URL}/nodes`, nodeData);
      return response;
    } catch (error) {
      console.error('Error creating replication node:', error);
      throw error;
    }
  },

  async getNode(nodeId) {
    try {
      const response = await api.get(`${BASE_URL}/nodes/${nodeId}`);
      return response;
    } catch (error) {
      console.error(`Error fetching node ${nodeId}:`, error);
      throw error;
    }
  },

  async updateNode(nodeId, nodeData) {
    try {
      const response = await api.put(`${BASE_URL}/nodes/${nodeId}`, nodeData);
      return response;
    } catch (error) {
      console.error(`Error updating node ${nodeId}:`, error);
      throw error;
    }
  },

  async deleteNode(nodeId) {
    try {
      const response = await api.delete(`${BASE_URL}/nodes/${nodeId}`);
      return response;
    } catch (error) {
      console.error(`Error deleting node ${nodeId}:`, error);
      throw error;
    }
  },

  async getNodeHealth(nodeId) {
    try {
      const response = await api.get(`${BASE_URL}/nodes/${nodeId}/health`);
      return response;
    } catch (error) {
      console.error(`Error fetching node ${nodeId} health:`, error);
      throw error;
    }
  },

  // Configuration History
  async getConfigHistory(limit = 50, offset = 0) {
    try {
      const response = await api.get(`${BASE_URL}/config/history?limit=${limit}&offset=${offset}`);
      return response;
    } catch (error) {
      console.error('Error fetching configuration history:', error);
      throw error;
    }
  },
};

export default replicationSettingsService;
