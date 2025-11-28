/**
 * Repository Replication Service
 * Handles API communication for per-repository replication settings
 */

import { api } from './api';

const BASE_URL = '/settings/replication/repositories';

export const repositoryReplicationService = {
  // Repository Replication Settings
  async getRepositoryReplicationSettings(repositoryId) {
    try {
      const response = await api.get(`${BASE_URL}/${repositoryId}`);
      return response;
    } catch (error) {
      console.error(`Error fetching replication settings for repo ${repositoryId}:`, error);
      throw error;
    }
  },

  async updateRepositoryReplicationSettings(repositoryId, settings) {
    try {
      const response = await api.put(`${BASE_URL}/${repositoryId}`, settings);
      return response;
    } catch (error) {
      console.error(`Error updating replication settings for repo ${repositoryId}:`, error);
      throw error;
    }
  },

  async getEffectiveReplicationConfig(repositoryId) {
    try {
      const response = await api.get(`${BASE_URL}/${repositoryId}/effective`);
      return response;
    } catch (error) {
      console.error(`Error fetching effective replication config for repo ${repositoryId}:`, error);
      throw error;
    }
  },
};

export default repositoryReplicationService;
