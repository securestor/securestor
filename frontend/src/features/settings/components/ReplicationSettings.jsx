import React, { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Save,
  RefreshCw,
  HardDrive,
  Trash2,
  Edit,
  Plus,
  CheckCircle,
  AlertCircle,
  AlertTriangle,
  Info,
  X,
} from 'lucide-react';
import { api } from '../../../services/api';

const ReplicationSettings = () => {
  const { t } = useTranslation('replication');
  const [config, setConfig] = useState({
    enabled: true,
    defaultQuorumSize: 2,
    syncFrequency: 'daily',
    healthCheckInterval: 60,
    failoverTimeout: 20,
  });

  const [storageNodes, setStorageNodes] = useState([]);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState(null);
  const [success, setSuccess] = useState(null);
  const [openDialog, setOpenDialog] = useState(false);
  const [editingNode, setEditingNode] = useState(null);
  const [nodeForm, setNodeForm] = useState({
    name: '',
    path: '',
    nodeType: 'local',
    priority: 50,
    isActive: true,
    // S3/MinIO fields
    s3Endpoint: '',
    s3Region: 'us-east-1',
    s3Bucket: '',
    s3AccessKey: '',
    s3SecretKey: '',
    s3UseSSL: false,
    s3PathPrefix: '',
  });

  useEffect(() => {
    loadSettings();
    loadStorageNodes();
  }, []);

  const loadSettings = async () => {
    try {
      setLoading(true);
      const response = await api.get('/settings/replication/config');
      
      if (response) {
        setConfig({
          enabled: response.enable_replication_default ?? true,
          defaultQuorumSize: response.default_quorum_size ?? 2,
          syncFrequency: response.sync_frequency_default ?? 'daily',
          healthCheckInterval: response.node_health_check_interval ?? 60,
          failoverTimeout: response.failover_timeout ?? 20,
        });
      }
    } catch (err) {
      console.error('Error loading replication settings:', err);
      setError(t('messages.loadError'));
    } finally {
      setLoading(false);
    }
  };

  const loadStorageNodes = async () => {
    try {
      const response = await api.get('/settings/replication/nodes');
      
      if (response && Array.isArray(response)) {
        const mappedNodes = response.map((node) => {
          // Only calculate capacity/used if backend provides the data
          const totalGB = node.storage_total_gb || 0;
          const availableGB = node.storage_available_gb || 0;
          const usedGB = totalGB > 0 ? totalGB - availableGB : 0;
          
          // Map health status to UI status
          let status = 'healthy';
          let healthMessage = 'Node is healthy and accessible';
          
          if (!node.is_healthy) {
            status = 'error';
            // Map backend health_status to user-friendly messages
            switch (node.health_status) {
              case 'path_not_found':
                healthMessage = t('healthStatus.pathNotFound');
                break;
              case 'path_inaccessible':
                healthMessage = t('healthStatus.pathInaccessible');
                break;
              case 'path_not_directory':
                healthMessage = 'Storage path is not a directory';
                break;
              case 'filesystem_error':
                healthMessage = 'Unable to read filesystem information';
                break;
              default:
                healthMessage = node.health_status || 'Node is unhealthy';
            }
          } else if (totalGB === 0) {
            status = 'warning';
            healthMessage = 'Storage metrics not yet collected - click "Check now"';
          }
          
          return {
            id: node.node_id || node.id,
            name: node.node_name || node.name,
            path: node.node_path || node.path,
            nodeType: node.node_type || 'local',
            status,
            healthMessage,
            healthStatus: node.health_status,
            capacity: totalGB * 1024 * 1024 * 1024, // Convert GB to bytes
            used: usedGB * 1024 * 1024 * 1024, // Convert GB to bytes
            priority: node.priority || 50,
            filesCount: node.artifact_count || 0,
            lastSync: node.last_health_check || node.updated_at || new Date().toISOString(),
            isActive: node.is_active !== false,
            // S3 fields
            s3Endpoint: node.s3_endpoint || '',
            s3Region: node.s3_region || '',
            s3Bucket: node.s3_bucket || '',
            s3AccessKey: node.s3_access_key || '',
            s3SecretKey: node.s3_secret_key || '',
            s3UseSSL: node.s3_use_ssl || false,
            s3PathPrefix: node.s3_path_prefix || '',
          };
        });
        setStorageNodes(mappedNodes);
      } else {
        // Empty response - no nodes configured yet
        setStorageNodes([]);
      }
    } catch (err) {
      console.error('Error loading storage nodes:', err);
      // Don't show error for empty list - just set to empty
      if (err.response?.status === 404 || err.message.includes('404')) {
        setStorageNodes([]);
      } else {
        setError(`Failed to load storage nodes: ${err.response?.data?.error || err.message}`);
      }
    }
  };

  const handleRefreshAll = async () => {
    if (storageNodes.length === 0) return;
    
    setLoading(true);
    setError(null);
    
    try {
      // Trigger health check for all nodes in parallel
      const healthChecks = storageNodes.map(node => 
        api.get(`/settings/replication/nodes/${node.id}/health`)
          .catch(err => console.error(`Failed to check node ${node.name}:`, err))
      );
      
      await Promise.all(healthChecks);
      
      // Reload nodes to get updated storage metrics
      await loadStorageNodes();
      
      setSuccess(t('messages.syncSuccess'));
      setTimeout(() => setSuccess(null), 3000);
    } catch (err) {
      console.error('Error refreshing nodes:', err);
      setError(`Failed to refresh nodes: ${err.response?.data?.error || err.message}`);
    } finally {
      setLoading(false);
    }
  };

  const handleSaveConfig = async () => {
    try {
      setSaving(true);
      setError(null);
      
      const payload = {
        enable_replication_default: config.enabled,
        default_quorum_size: config.defaultQuorumSize,
        sync_frequency_default: config.syncFrequency,
        node_health_check_interval: config.healthCheckInterval,
        failover_timeout: config.failoverTimeout,
      };
      
      await api.put('/settings/replication/config', payload);
      setSuccess(t('messages.saveSuccess'));
      setTimeout(() => setSuccess(null), 3000);
    } catch (err) {
      console.error('Error saving replication settings:', err);
      setError(err.response?.data?.error || t('messages.saveFailed'));
    } finally {
      setSaving(false);
    }
  };

  const handleConfigChange = (field, value) => {
    setConfig((prev) => ({
      ...prev,
      [field]: value,
    }));
  };

  const handleOpenDialog = (node = null) => {
    if (node) {
      setEditingNode(node);
      setNodeForm({
        name: node.name,
        path: node.path,
        nodeType: node.nodeType || 'local',
        priority: node.priority || 50,
        isActive: node.status === 'healthy',
        s3Endpoint: node.s3Endpoint || '',
        s3Region: node.s3Region || 'us-east-1',
        s3Bucket: node.s3Bucket || '',
        s3AccessKey: node.s3AccessKey || '',
        s3SecretKey: node.s3SecretKey || '',
        s3UseSSL: node.s3UseSSL || false,
        s3PathPrefix: node.s3PathPrefix || '',
      });
    } else {
      setEditingNode(null);
      setNodeForm({
        name: '',
        path: '',
        nodeType: 'local',
        priority: 50,
        isActive: true,
        s3Endpoint: '',
        s3Region: 'us-east-1',
        s3Bucket: '',
        s3AccessKey: '',
        s3SecretKey: '',
        s3UseSSL: false,
        s3PathPrefix: '',
      });
    }
    setOpenDialog(true);
  };

  const handleCloseDialog = () => {
    setOpenDialog(false);
    setEditingNode(null);
    setNodeForm({
      name: '',
      path: '',
      nodeType: 'local',
      priority: 50,
      isActive: true,
      s3Endpoint: '',
      s3Region: 'us-east-1',
      s3Bucket: '',
      s3AccessKey: '',
      s3SecretKey: '',
      s3UseSSL: false,
      s3PathPrefix: '',
    });
  };

  const handleSaveNode = async () => {
    try {
      setSaving(true);
      setError(null);

      // Validate inputs
      if (!nodeForm.name || nodeForm.name.trim().length === 0) {
        setError('Node name is required');
        return;
      }
      
      // For local nodes, path is required
      if ((nodeForm.nodeType === 'local' || !nodeForm.nodeType) && (!nodeForm.path || nodeForm.path.trim().length === 0)) {
        setError('Storage path is required for local nodes');
        return;
      }
      
      if (nodeForm.priority < 1 || nodeForm.priority > 100) {
        setError('Priority must be between 1 and 100');
        return;
      }

      // Validate S3 fields if node type is s3/minio
      if (nodeForm.nodeType === 's3' || nodeForm.nodeType === 'minio') {
        if (!nodeForm.s3Endpoint.trim()) {
          setError('S3 Endpoint is required for S3/MinIO nodes');
          return;
        }
        if (!nodeForm.s3Bucket.trim()) {
          setError('S3 Bucket is required for S3/MinIO nodes');
          return;
        }
        if (!nodeForm.s3Region.trim()) {
          setError('S3 Region is required for S3/MinIO nodes');
          return;
        }
      }
      
      // Set path for S3 nodes (backend uses s3_path_prefix instead)
      const nodePath = (nodeForm.nodeType === 's3' || nodeForm.nodeType === 'minio') 
        ? (nodeForm.s3PathPrefix || '/') 
        : nodeForm.path.trim();

      if (editingNode) {
        // Update existing node
        const nodeData = {
          node_path: nodePath,
          priority: parseInt(nodeForm.priority),
          is_active: nodeForm.isActive,
          s3_endpoint: nodeForm.s3Endpoint.trim(),
          s3_region: nodeForm.s3Region.trim(),
          s3_bucket: nodeForm.s3Bucket.trim(),
          s3_access_key: nodeForm.s3AccessKey.trim(),
          s3_secret_key: nodeForm.s3SecretKey.trim(),
          s3_use_ssl: nodeForm.s3UseSSL,
          s3_path_prefix: nodeForm.s3PathPrefix.trim(),
        };
        await api.put(`/settings/replication/nodes/${editingNode.id}`, nodeData);
        setSuccess(t('messages.nodeUpdateSuccess'));
      } else {
        // Create new node
        const nodeData = {
          node_name: nodeForm.name.trim(),
          node_path: nodePath,
          node_type: nodeForm.nodeType,
          priority: parseInt(nodeForm.priority),
          is_active: true,
          s3_endpoint: nodeForm.s3Endpoint.trim(),
          s3_region: nodeForm.s3Region.trim(),
          s3_bucket: nodeForm.s3Bucket.trim(),
          s3_access_key: nodeForm.s3AccessKey.trim(),
          s3_secret_key: nodeForm.s3SecretKey.trim(),
          s3_use_ssl: nodeForm.s3UseSSL,
          s3_path_prefix: nodeForm.s3PathPrefix.trim(),
        };
        const response = await api.post('/settings/replication/nodes', nodeData);
        setSuccess(t('messages.nodeAddSuccess'));
        
        // Automatically trigger health check for newly created nodes
        if (response && response.id) {
          setTimeout(async () => {
            try {
              await api.get(`/settings/replication/nodes/${response.id}/health`);
              await loadStorageNodes();
              setSuccess(t('messages.nodeAddSuccess'));
            } catch (healthErr) {
              setSuccess(t('messages.nodeAddSuccess'));
            }
          }, 1000);
        }
      }

      handleCloseDialog();
      await loadStorageNodes();
      
      setTimeout(() => setSuccess(null), 3000);
    } catch (err) {
      console.error('Error saving storage node:', err);
      const errorMessage = err.response?.data?.error || err.response?.data?.message || err.message || t('messages.nodeSaveFailed');
      setError(errorMessage);
    } finally {
      setSaving(false);
    }
  };

  const handleDeleteNode = async (nodeId) => {
    if (!window.confirm('Are you sure you want to delete this storage node?')) {
      return;
    }

    try {
      setSaving(true);
      await api.delete(`/settings/replication/nodes/${nodeId}`);
      setSuccess(t('messages.nodeDeleteSuccess'));
      loadStorageNodes();
      setTimeout(() => setSuccess(null), 3000);
    } catch (err) {
      console.error('Error deleting storage node:', err);
      setError(err.response?.data?.error || t('messages.nodeDeleteFailed'));
    } finally {
      setSaving(false);
    }
  };

  const handleSyncNode = async (nodeId) => {
    try {
      setSaving(true);
      await api.get(`/settings/replication/nodes/${nodeId}/health`);
      setSuccess(t('messages.syncSuccess'));
      loadStorageNodes();
      setTimeout(() => setSuccess(null), 3000);
    } catch (err) {
      console.error('Error syncing node:', err);
      setError(err.response?.data?.error || t('messages.syncFailed'));
    } finally {
      setSaving(false);
    }
  };

  const formatBytes = (bytes) => {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i];
  };

  const getStatusIcon = (status) => {
    switch (status) {
      case 'healthy':
        return <CheckCircle className="w-5 h-5 text-green-500" />;
      case 'warning':
        return <AlertTriangle className="w-5 h-5 text-yellow-500" />;
      case 'error':
        return <AlertCircle className="w-5 h-5 text-red-500" />;
      default:
        return <Info className="w-5 h-5 text-blue-500" />;
    }
  };

  const getUsagePercentage = (used, capacity) => {
    return Math.round((used / capacity) * 100);
  };

  const getUsageColor = (percentage) => {
    if (percentage >= 90) return 'bg-red-500';
    if (percentage >= 75) return 'bg-yellow-500';
    return 'bg-green-500';
  };

  if (loading) {
    return (
      <div className="flex justify-center items-center min-h-screen">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
      </div>
    );
  }

  return (
    <div className="p-6 max-w-7xl mx-auto">
      {/* Header */}
      <div className="flex justify-between items-center mb-6">
        <div>
          <h1 className="text-3xl font-bold text-gray-900 dark:text-white">{t('page.title')}</h1>
          <p className="text-gray-600 dark:text-gray-400 mt-1">
            {t('page.description')}
          </p>
        </div>
        <div className="flex gap-2">
          <button
            onClick={loadStorageNodes}
            className="flex items-center gap-2 px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-800"
          >
            <RefreshCw className="w-4 h-4" />
            {t('buttons.refresh')}
          </button>
          <button
            onClick={handleSaveConfig}
            disabled={saving}
            className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <Save className="w-4 h-4" />
            {t('buttons.saveConfiguration')}
          </button>
        </div>
      </div>

      {/* Alerts */}
      {error && (
        <div className="mb-4 p-4 bg-red-50 border border-red-200 rounded-lg flex items-start justify-between">
          <div className="flex items-start gap-2">
            <AlertCircle className="w-5 h-5 text-red-600 mt-0.5" />
            <span className="text-red-800">{error}</span>
          </div>
          <button onClick={() => setError(null)} className="text-red-600 hover:text-red-800">
            <X className="w-4 h-4" />
          </button>
        </div>
      )}

      {success && (
        <div className="mb-4 p-4 bg-green-50 border border-green-200 rounded-lg flex items-start justify-between">
          <div className="flex items-start gap-2">
            <CheckCircle className="w-5 h-5 text-green-600 mt-0.5" />
            <span className="text-green-800">{success}</span>
          </div>
          <button onClick={() => setSuccess(null)} className="text-green-600 hover:text-green-800">
            <X className="w-4 h-4" />
          </button>
        </div>
      )}

      {/* Erasure Coding Configuration */}
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 p-6 mb-6">
        <h2 className="text-xl font-semibold text-gray-900 dark:text-white mb-2">
          {t('config.title')}
        </h2>
        <p className="text-sm text-gray-600 dark:text-gray-400 mb-6">
          {t('config.description', { quorumSize: config.defaultQuorumSize })}
        </p>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          {/* Default Quorum Size */}
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              {t('config.defaultQuorumSize.label')}
            </label>
            <input
              type="number"
              value={config.defaultQuorumSize}
              onChange={(e) => handleConfigChange('defaultQuorumSize', parseInt(e.target.value))}
              min="1"
              max="5"
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:text-white"
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              {t('config.defaultQuorumSize.help')}
            </p>
          </div>

          {/* Sync Frequency */}
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              {t('config.syncFrequency.label')}
            </label>
            <select
              value={config.syncFrequency}
              onChange={(e) => handleConfigChange('syncFrequency', e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:text-white"
            >
              <option value="realtime">{t('config.syncFrequency.options.realtime')}</option>
              <option value="hourly">{t('config.syncFrequency.options.hourly')}</option>
              <option value="daily">{t('config.syncFrequency.options.daily')}</option>
              <option value="weekly">{t('config.syncFrequency.options.weekly')}</option>
            </select>
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              {t('config.syncFrequency.help')}
            </p>
          </div>

          {/* Health Check Interval */}
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              {t('config.healthCheckInterval.label')}
            </label>
            <input
              type="number"
              value={config.healthCheckInterval}
              onChange={(e) => handleConfigChange('healthCheckInterval', parseInt(e.target.value))}
              min="10"
              max="300"
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:text-white"
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              {t('config.healthCheckInterval.help')}
            </p>
          </div>

          {/* Failover Timeout */}
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              {t('config.failoverTimeout.label')}
            </label>
            <input
              type="number"
              value={config.failoverTimeout}
              onChange={(e) => handleConfigChange('failoverTimeout', parseInt(e.target.value))}
              min="5"
              max="300"
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:text-white"
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              {t('config.failoverTimeout.help')}
            </p>
          </div>
        </div>

        {/* Toggles */}
        <div className="grid grid-cols-1 md:grid-cols-1 gap-4 mt-6">
          <label className="flex items-center gap-3 cursor-pointer">
            <input
              type="checkbox"
              checked={config.enabled}
              onChange={(e) => handleConfigChange('enabled', e.target.checked)}
              className="w-5 h-5 text-blue-600 rounded focus:ring-2 focus:ring-blue-500"
            />
            <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
              {t('config.enabled')}
            </span>
          </label>
        </div>
      </div>

      {/* Storage Nodes */}
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 p-6">
        <div className="flex justify-between items-center mb-4">
          <h2 className="text-xl font-semibold text-gray-900 dark:text-white">{t('nodes.title')}</h2>
          <div className="flex items-center gap-2">
            {storageNodes.length > 0 && (
              <button
                onClick={handleRefreshAll}
                disabled={loading}
                className="flex items-center gap-2 px-4 py-2 bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-200 dark:hover:bg-gray-600 disabled:opacity-50 disabled:cursor-not-allowed"
                title={t('nodes.tooltips.refreshAll')}
              >
                <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
                {t('buttons.refreshAll')}
              </button>
            )}
            <button
              onClick={() => handleOpenDialog()}
              className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
            >
              <Plus className="w-4 h-4" />
              {t('buttons.addNode')}
            </button>
          </div>
        </div>

        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-gray-50 dark:bg-gray-700">
              <tr>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-700 dark:text-gray-300 uppercase tracking-wider">
                  {t('nodes.table.status')}
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-700 dark:text-gray-300 uppercase tracking-wider">
                  {t('nodes.table.name')}
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-700 dark:text-gray-300 uppercase tracking-wider">
                  {t('nodes.table.path')}
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-700 dark:text-gray-300 uppercase tracking-wider">
                  {t('nodes.table.storageUsage')}
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-700 dark:text-gray-300 uppercase tracking-wider">
                  {t('nodes.table.files')}
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-700 dark:text-gray-300 uppercase tracking-wider">
                  {t('nodes.table.priority')}
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-700 dark:text-gray-300 uppercase tracking-wider">
                  {t('nodes.table.lastSync')}
                </th>
                <th className="px-4 py-3 text-right text-xs font-medium text-gray-700 dark:text-gray-300 uppercase tracking-wider">
                  {t('nodes.table.actions')}
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
              {storageNodes.length === 0 ? (
                <tr>
                  <td colSpan="8" className="px-4 py-12 text-center">
                    <HardDrive className="w-12 h-12 text-gray-400 mx-auto mb-3" />
                    <p className="text-gray-500 dark:text-gray-400 font-medium mb-1">
                      {t('nodes.emptyTitle')}
                    </p>
                    <p className="text-sm text-gray-400 dark:text-gray-500 mb-4">
                      {t('nodes.emptyDescription')}
                    </p>
                    <button
                      onClick={() => handleOpenDialog()}
                      className="inline-flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 text-sm"
                    >
                      <Plus className="w-4 h-4" />
                      {t('buttons.addFirstNode')}
                    </button>
                  </td>
                </tr>
              ) : (
                storageNodes.map((node) => {
                  const usagePercentage = getUsagePercentage(node.used, node.capacity);
                  return (
                    <tr key={node.id} className="hover:bg-gray-50 dark:hover:bg-gray-700">
                      <td className="px-4 py-4 whitespace-nowrap">
                        <div title={node.healthMessage} className="cursor-help">
                          {getStatusIcon(node.status)}
                        </div>
                      </td>
                      <td className="px-4 py-4 whitespace-nowrap">
                        <div className="flex items-center gap-2">
                          <HardDrive className="w-4 h-4 text-blue-600" />
                          <span className="font-medium text-gray-900 dark:text-white">
                            {node.name}
                          </span>
                        </div>
                      </td>
                      <td className="px-4 py-4 whitespace-nowrap">
                        <code className="text-sm text-gray-600 dark:text-gray-400">{node.path}</code>
                      </td>
                      <td className="px-4 py-4">
                        {node.capacity > 0 ? (
                          <div className="w-48">
                            <div className="flex justify-between text-xs text-gray-600 dark:text-gray-400 mb-1">
                              <span>
                                {formatBytes(node.used)} / {formatBytes(node.capacity)}
                              </span>
                              <span className="font-semibold">{usagePercentage}%</span>
                            </div>
                            <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
                              <div
                                className={`h-2 rounded-full ${getUsageColor(usagePercentage)}`}
                                style={{ width: `${usagePercentage}%` }}
                              ></div>
                            </div>
                          </div>
                        ) : (
                          <div className="flex items-center gap-2">
                            <span className="text-sm text-gray-500 dark:text-gray-400 italic">
                              {t('nodes.noData')}
                            </span>
                            <button
                              onClick={(e) => {
                                e.stopPropagation();
                                handleSyncNode(node.id);
                              }}
                              className="text-xs text-blue-600 hover:text-blue-700 underline"
                              title={t('nodes.tooltips.runHealthCheck')}
                            >
                              {t('buttons.checkNow')}
                            </button>
                          </div>
                        )}
                      </td>
                      <td className="px-4 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-white">
                        {node.filesCount.toLocaleString()}
                      </td>
                      <td className="px-4 py-4 whitespace-nowrap">
                        <div className="flex items-center gap-2">
                          <div
                            className={`w-2 h-2 rounded-full ${
                              node.priority >= 80
                                ? 'bg-red-500'
                                : node.priority >= 50
                                ? 'bg-blue-500'
                                : 'bg-gray-500'
                            }`}
                          ></div>
                          <span className="text-sm font-medium text-gray-900 dark:text-white">
                            {node.priority}
                          </span>
                          <span className="text-xs text-gray-500">
                            {node.priority >= 80 ? t('nodes.priority.high') : node.priority >= 50 ? t('nodes.priority.normal') : t('nodes.priority.low')}
                          </span>
                        </div>
                      </td>
                      <td className="px-4 py-4 whitespace-nowrap text-xs text-gray-500 dark:text-gray-400">
                        {new Date(node.lastSync).toLocaleString()}
                      </td>
                      <td className="px-4 py-4 whitespace-nowrap text-right text-sm font-medium">
                        <div className="flex justify-end gap-1">
                          <button
                            onClick={() => handleSyncNode(node.id)}
                            disabled={saving}
                            className="p-2 text-gray-600 hover:text-blue-600 dark:text-gray-400 dark:hover:text-blue-400"
                            title={t('nodes.tooltips.checkHealth')}
                          >
                            <RefreshCw className="w-4 h-4" />
                          </button>
                          <button
                            onClick={() => handleOpenDialog(node)}
                            className="p-2 text-gray-600 hover:text-blue-600 dark:text-gray-400 dark:hover:text-blue-400"
                            title={t('nodes.tooltips.edit')}
                          >
                            <Edit className="w-4 h-4" />
                          </button>
                          <button
                            onClick={() => handleDeleteNode(node.id)}
                            className="p-2 text-gray-600 hover:text-red-600 dark:text-gray-400 dark:hover:text-red-400"
                            title={t('nodes.tooltips.delete')}
                          >
                            <Trash2 className="w-4 h-4" />
                          </button>
                        </div>
                      </td>
                    </tr>
                  );
                })
              )}
            </tbody>
          </table>
        </div>

        {/* Summary */}
        {storageNodes.length > 0 && (
          <div className="mt-6 grid grid-cols-1 md:grid-cols-4 gap-4">
            <div className="bg-blue-50 dark:bg-blue-900/20 p-4 rounded-lg">
              <div className="text-sm text-blue-600 dark:text-blue-400">{t('summary.totalNodes')}</div>
              <div className="text-2xl font-bold text-blue-900 dark:text-blue-300">
                {storageNodes.length}
              </div>
            </div>
            <div className="bg-green-50 dark:bg-green-900/20 p-4 rounded-lg">
              <div className="text-sm text-green-600 dark:text-green-400">{t('summary.healthyNodes')}</div>
              <div className="text-2xl font-bold text-green-900 dark:text-green-300">
                {storageNodes.filter((n) => n.status === 'healthy').length}
              </div>
            </div>
            <div className="bg-purple-50 dark:bg-purple-900/20 p-4 rounded-lg">
              <div className="text-sm text-purple-600 dark:text-purple-400">{t('summary.totalCapacity')}</div>
              <div className="text-2xl font-bold text-purple-900 dark:text-purple-300">
                {formatBytes(storageNodes.reduce((sum, n) => sum + n.capacity, 0))}
              </div>
            </div>
            <div className="bg-orange-50 dark:bg-orange-900/20 p-4 rounded-lg">
              <div className="text-sm text-orange-600 dark:text-orange-400">{t('summary.usedSpace')}</div>
              <div className="text-2xl font-bold text-orange-900 dark:text-orange-300">
                {formatBytes(storageNodes.reduce((sum, n) => sum + n.used, 0))}
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Add/Edit Node Dialog */}
      {openDialog && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-md w-full mx-4">
            <div className="flex justify-between items-center p-6 border-b border-gray-200 dark:border-gray-700">
              <h3 className="text-xl font-semibold text-gray-900 dark:text-white">
                {editingNode ? t('dialog.editTitle') : t('dialog.addTitle')}
              </h3>
              <button
                onClick={handleCloseDialog}
                className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            <div className="p-6 space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  {t('dialog.nodeName.label')} <span className="text-red-500">{t('dialog.required')}</span>
                </label>
                <input
                  type="text"
                  value={nodeForm.name}
                  onChange={(e) => setNodeForm({ ...nodeForm, name: e.target.value })}
                  disabled={editingNode !== null}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:text-white disabled:bg-gray-100 disabled:cursor-not-allowed"
                  placeholder={t('dialog.nodeName.placeholder')}
                  maxLength={255}
                />
                {editingNode && (
                  <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    {t('dialog.nodeName.help')}
                  </p>
                )}
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  {t('dialog.nodeType.label')} <span className="text-red-500">{t('dialog.required')}</span>
                </label>
                <select
                  value={nodeForm.nodeType}
                  onChange={(e) => setNodeForm({ ...nodeForm, nodeType: e.target.value })}
                  disabled={editingNode !== null}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:text-white disabled:bg-gray-100 disabled:cursor-not-allowed"
                >
                  <option value="local">{t('dialog.nodeType.options.local')}</option>
                  <option value="s3">{t('dialog.nodeType.options.s3')}</option>
                  <option value="minio">{t('dialog.nodeType.options.minio')}</option>
                  <option value="gcs">{t('dialog.nodeType.options.gcs')}</option>
                  <option value="azure">{t('dialog.nodeType.options.azure')}</option>
                </select>
                {editingNode && (
                  <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    {t('dialog.nodeType.help')}
                  </p>
                )}
              </div>

              {(nodeForm.nodeType === 'local' || !nodeForm.nodeType) && (
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                    {t('dialog.storagePath.label')} <span className="text-red-500">{t('dialog.required')}</span>
                  </label>
                  <input
                    type="text"
                    value={nodeForm.path}
                    onChange={(e) => setNodeForm({ ...nodeForm, path: e.target.value })}
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:text-white"
                    placeholder={t('dialog.storagePath.placeholder')}
                    maxLength={1024}
                  />
                  <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    {t('dialog.storagePath.help')}
                  </p>
                </div>
              )}

              {(nodeForm.nodeType === 's3' || nodeForm.nodeType === 'minio') && (
                <>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                      {t('dialog.s3Endpoint.label')} <span className="text-red-500">{t('dialog.required')}</span>
                    </label>
                    <input
                      type="text"
                      value={nodeForm.s3Endpoint}
                      onChange={(e) => setNodeForm({ ...nodeForm, s3Endpoint: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:text-white"
                      placeholder={t('dialog.s3Endpoint.placeholder')}
                    />
                    <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                      {t('dialog.s3Endpoint.help')}
                    </p>
                  </div>

                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                        {t('dialog.region.label')} <span className="text-red-500">{t('dialog.required')}</span>
                      </label>
                      <input
                        type="text"
                        value={nodeForm.s3Region}
                        onChange={(e) => setNodeForm({ ...nodeForm, s3Region: e.target.value })}
                        className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:text-white"
                        placeholder={t('dialog.region.placeholder')}
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                        {t('dialog.bucketName.label')} <span className="text-red-500">{t('dialog.required')}</span>
                      </label>
                      <input
                        type="text"
                        value={nodeForm.s3Bucket}
                        onChange={(e) => setNodeForm({ ...nodeForm, s3Bucket: e.target.value })}
                        className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:text-white"
                        placeholder={t('dialog.bucketName.placeholder')}
                      />
                    </div>
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                      {t('dialog.accessKey.label')}
                    </label>
                    <input
                      type="text"
                      value={nodeForm.s3AccessKey}
                      onChange={(e) => setNodeForm({ ...nodeForm, s3AccessKey: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:text-white"
                      placeholder={t('dialog.accessKey.placeholder')}
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                      {t('dialog.secretKey.label')}
                    </label>
                    <input
                      type="password"
                      value={nodeForm.s3SecretKey}
                      onChange={(e) => setNodeForm({ ...nodeForm, s3SecretKey: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:text-white"
                      placeholder={t('dialog.secretKey.placeholder')}
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                      {t('dialog.pathPrefix.label')}
                    </label>
                    <input
                      type="text"
                      value={nodeForm.s3PathPrefix}
                      onChange={(e) => setNodeForm({ ...nodeForm, s3PathPrefix: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:text-white"
                      placeholder={t('dialog.pathPrefix.placeholder')}
                    />
                    <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                      {t('dialog.pathPrefix.help')}
                    </p>
                  </div>

                  <div>
                    <label className="flex items-center gap-3 cursor-pointer">
                      <input
                        type="checkbox"
                        checked={nodeForm.s3UseSSL}
                        onChange={(e) => setNodeForm({ ...nodeForm, s3UseSSL: e.target.checked })}
                        className="w-5 h-5 text-blue-600 rounded focus:ring-2 focus:ring-blue-500"
                      />
                      <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
                        {t('dialog.useSsl')}
                      </span>
                    </label>
                  </div>

                  <div className="hidden">
                    <input
                      type="text"
                      value={nodeForm.path}
                      onChange={(e) => setNodeForm({ ...nodeForm, path: e.target.value })}
                    />
                  </div>
                </>
              )}

              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  {t('dialog.priority.label')} <span className="text-red-500">{t('dialog.required')}</span>
                </label>
                <input
                  type="number"
                  value={nodeForm.priority}
                  onChange={(e) => setNodeForm({ ...nodeForm, priority: parseInt(e.target.value) || 1 })}
                  min="1"
                  max="100"
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:text-white"
                />
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                  {t('dialog.priority.help')}
                </p>
              </div>

              {editingNode && (
                <div>
                  <label className="flex items-center gap-3 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={nodeForm.isActive}
                      onChange={(e) => setNodeForm({ ...nodeForm, isActive: e.target.checked })}
                      className="w-5 h-5 text-blue-600 rounded focus:ring-2 focus:ring-blue-500"
                    />
                    <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
                      {t('dialog.isActive.label')}
                    </span>
                  </label>
                  <p className="text-xs text-gray-500 dark:text-gray-400 mt-1 ml-8">
                    {t('dialog.isActive.help')}
                  </p>
                </div>
              )}
            </div>

            <div className="flex justify-end gap-2 p-6 border-t border-gray-200 dark:border-gray-700">
              <button
                onClick={handleCloseDialog}
                disabled={saving}
                className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 text-gray-700 dark:text-gray-300 disabled:opacity-50"
              >
                {t('buttons.cancel')}
              </button>
              <button
                onClick={handleSaveNode}
                disabled={
                  saving || 
                  !nodeForm.name || 
                  ((nodeForm.nodeType === 'local' || !nodeForm.nodeType) && !nodeForm.path) ||
                  ((nodeForm.nodeType === 's3' || nodeForm.nodeType === 'minio') && (!nodeForm.s3Endpoint || !nodeForm.s3Bucket || !nodeForm.s3Region)) ||
                  nodeForm.priority < 1 || 
                  nodeForm.priority > 100
                }
                className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {saving ? t('buttons.saving') : editingNode ? t('buttons.updateNode') : t('buttons.addNode')}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default ReplicationSettings;
