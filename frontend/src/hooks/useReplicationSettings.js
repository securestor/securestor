/**
 * useReplicationSettings Hook
 * Manages global replication settings state
 */

import { useState, useEffect } from 'react';
import { useTenant } from '../context/TenantContext';
import replicationSettingsService from '../services/replicationSettingsService';

export const useReplicationSettings = () => {
  const { tenantId, isReady } = useTenant();
  const [config, setConfig] = useState(null);
  const [nodes, setNodes] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  const fetchSettings = async () => {
    try {
      setLoading(true);
      const data = await replicationSettingsService.getGlobalConfig();
      
      // The API returns both config and nodes in one response
      if (data && data.nodes) {
        const { nodes: nodesData, ...configData } = data;
        setConfig(configData);
        setNodes(nodesData);
      } else {
        setConfig(data);
        setNodes([]);
      }
      
      setError(null);
    } catch (err) {
      setError(err.message || 'Failed to fetch replication settings');
      console.error('Error fetching replication settings:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    // Only fetch when tenant is ready
    if (isReady && tenantId) {
      fetchSettings();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isReady, tenantId]);

  const updateConfig = async (configData) => {
    try {
      const data = await replicationSettingsService.updateGlobalConfig(configData);
      
      // The API returns both config and nodes in one response
      if (data && data.nodes) {
        const { nodes: nodesData, ...updatedConfig } = data;
        setConfig(updatedConfig);
        setNodes(nodesData);
        return updatedConfig;
      } else {
        setConfig(data);
        return data;
      }
    } catch (err) {
      setError(err.message || 'Failed to update replication settings');
      throw err;
    }
  };

  const createNode = async (nodeData) => {
    try {
      const newNode = await replicationSettingsService.createNode(nodeData);
      setNodes([...nodes, newNode]);
      return newNode;
    } catch (err) {
      setError(err.message || 'Failed to create replication node');
      throw err;
    }
  };

  const updateNode = async (nodeId, nodeData) => {
    try {
      const updated = await replicationSettingsService.updateNode(nodeId, nodeData);
      setNodes(nodes.map(n => n.id === nodeId ? updated : n));
      return updated;
    } catch (err) {
      setError(err.message || 'Failed to update replication node');
      throw err;
    }
  };

  const deleteNode = async (nodeId) => {
    try {
      await replicationSettingsService.deleteNode(nodeId);
      setNodes(nodes.filter(n => n.id !== nodeId));
    } catch (err) {
      setError(err.message || 'Failed to delete replication node');
      throw err;
    }
  };

  const getNodeHealth = async (nodeId) => {
    try {
      return await replicationSettingsService.getNodeHealth(nodeId);
    } catch (err) {
      setError(err.message || 'Failed to fetch node health');
      throw err;
    }
  };

  return {
    config,
    nodes,
    loading,
    error,
    fetchSettings,
    updateConfig,
    createNode,
    updateNode,
    deleteNode,
    getNodeHealth,
  };
};

export default useReplicationSettings;
