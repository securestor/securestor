/**
 * useRepositoryReplication Hook
 * Manages per-repository replication settings
 */

import { useState, useEffect } from 'react';
import repositoryReplicationService from '../services/repositoryReplicationService';

export const useRepositoryReplication = (repositoryId) => {
  const [settings, setSettings] = useState(null);
  const [effectiveConfig, setEffectiveConfig] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  const fetchSettings = async () => {
    try {
      setLoading(true);
      const [settingsData, effectiveData] = await Promise.all([
        repositoryReplicationService.getRepositoryReplicationSettings(repositoryId),
        repositoryReplicationService.getEffectiveReplicationConfig(repositoryId),
      ]);
      setSettings(settingsData);
      setEffectiveConfig(effectiveData);
      setError(null);
    } catch (err) {
      setError(err.message || 'Failed to fetch repository replication settings');
      console.error('Error fetching repository replication settings:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (repositoryId) {
      fetchSettings();
    }
  }, [repositoryId]);

  const updateSettings = async (newSettings) => {
    try {
      const updated = await repositoryReplicationService.updateRepositoryReplicationSettings(repositoryId, newSettings);
      setSettings(updated);
      // Update effective config after successful update
      const effective = await repositoryReplicationService.getEffectiveReplicationConfig(repositoryId);
      setEffectiveConfig(effective);
      return updated;
    } catch (err) {
      setError(err.message || 'Failed to update repository replication settings');
      throw err;
    }
  };

  return {
    settings,
    effectiveConfig,
    loading,
    error,
    fetchSettings,
    updateSettings,
  };
};

export default useRepositoryReplication;
