import React, { useState, useEffect, useCallback } from 'react';
import { useTranslation } from '../../hooks/useTranslation';
import { api } from '../../services/api';
import { 
  Download, 
  Trash2, 
  Shield, 
  RefreshCw, 
  AlertTriangle,
  CheckCircle,
  Clock,
  Database,
  HardDrive,
  Cloud,
  Package
} from 'lucide-react';
import { formatBytes, formatRelativeTime } from '../../utils/formatters';

const CacheManagement = () => {
  const { t } = useTranslation('cache');
  const [cacheItems, setCacheItems] = useState([]);
  const [stats, setStats] = useState(null);
  const [selectedType, setSelectedType] = useState('all');
  const [selectedLevel, setSelectedLevel] = useState('all');
  const [sortBy, setSortBy] = useState('last_accessed');
  const [sortOrder, setSortOrder] = useState('desc');
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [loading, setLoading] = useState(false);
  const [scanningItems, setScanningItems] = useState(new Set());
  const pageSize = 50;

  useEffect(() => {
    fetchCacheItems();
    fetchCacheStats();
    
    // Auto-refresh every 30 seconds
    const interval = setInterval(() => {
      fetchCacheStats();
    }, 30000);
    
    return () => clearInterval(interval);
  }, [page, selectedType, selectedLevel, sortBy, sortOrder]);

  const fetchCacheItems = async () => {
    setLoading(true);
    try {
      // Use artifacts endpoint to show cached items
      const params = new URLSearchParams({
        page: page.toString(),
        limit: pageSize.toString()
      });
      
      if (selectedType !== 'all') {
        params.append('type', selectedType);
      }
      if (selectedLevel !== 'all') {
        params.append('level', selectedLevel);
      }
      if (sortBy) {
        params.append('sort', sortBy);
        params.append('order', sortOrder || 'desc');
      }
      
      const data = await api.get(`/cache/items?${params}`);
      
      setCacheItems(data.items || []);
      setTotalPages(data.total_pages || 0);
    } catch (error) {
      console.error('Failed to fetch cache items:', error);
      setCacheItems([]);
    } finally {
      setLoading(false);
    }
  };

  const fetchCacheStats = async () => {
    try {
      const data = await api.get('/cache/stats');
      setStats(data);
    } catch (error) {
      console.error('Failed to fetch cache stats:', error);
      setStats(null);
    }
  };

  const handleScanItem = async (itemId, itemPath) => {
    setScanningItems(prev => new Set(prev).add(itemId));
    
    try {
      // Trigger scan for cached item
      await api.post(`/cache/items/${itemId}/scan`);
      
      // Show success notification
      await fetchCacheItems(); // Refresh to show updated status
    } catch (error) {
      console.error('Failed to trigger scan:', error);
      alert(`${t('messages.scanFailed')} ${itemPath}`);
    } finally {
      setScanningItems(prev => {
        const newSet = new Set(prev);
        newSet.delete(itemId);
        return newSet;
      });
    }
  };

  const handleDeleteItem = async (itemId, itemPath) => {
    if (!window.confirm(`${t('messages.confirmDelete')} "${itemPath}"?`)) {
      return;
    }
    
    try {
      await api.delete(`/cache/items/${itemId}`);
      await fetchCacheItems();
      await fetchCacheStats();
    } catch (error) {
      console.error('Failed to delete cached item:', error);
      alert(`${t('messages.deleteFailed')} ${itemPath}`);
    }
  };

  const handleDownloadItem = async (itemId, itemPath) => {
    try {
      // Use fetch for blob download but include tenant headers
      const headers = {
        'X-Tenant-Slug': api.getCurrentTenantSlug(),
      };
      
      const tenantId = api.getCurrentTenantId();
      if (tenantId) {
        headers['X-Tenant-ID'] = tenantId;
      }
      
      // Download cached item file
      const response = await fetch(`${api.baseURL}/cache/items/${itemId}/download`, {
        headers
      });
      
      if (response.ok) {
        const blob = await response.blob();
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = itemPath.split('/').pop() || 'artifact';
        document.body.appendChild(a);
        a.click();
        window.URL.revokeObjectURL(url);
        document.body.removeChild(a);
      } else {
        throw new Error('Download failed');
      }
    } catch (error) {
      console.error('Failed to download:', error);
      alert(`${t('messages.downloadFailed')} ${itemPath}`);
    }
  };

  const handleFlushCache = async () => {
    if (!window.confirm(t('messages.confirmFlush'))) {
      return;
    }
    
    try {
      await api.post('/cache/flush?level=all');
      await fetchCacheItems();
      await fetchCacheStats();
    } catch (error) {
      console.error('Failed to flush cache:', error);
      alert(t('messages.flushFailed'));
    }
  };

  const handleEvictUnused = async () => {
    const days = prompt(t('messages.evictPrompt'), '30');
    if (!days) return;
    
    try {
      const response = await api.post(`/cache/evict?days=${days}&min_hits=0`);
      alert(t('messages.evictSuccess', { count: response.items_removed }));
      await fetchCacheItems();
      await fetchCacheStats();
    } catch (error) {
      console.error('Failed to evict cache:', error);
      alert(t('messages.evictFailed'));
    }
  };

  const handleCleanupExpired = async () => {
    if (!window.confirm(t('messages.confirmCleanup'))) {
      return;
    }
    
    try {
      const response = await api.post('/cache/cleanup');
      alert(t('messages.cleanupSuccess', { count: response.items_removed }));
      await fetchCacheItems();
      await fetchCacheStats();
    } catch (error) {
      console.error('Failed to cleanup cache:', error);
      alert(t('messages.cleanupFailed'));
    }
  };

  const getTypeColor = (type) => {
    const colors = {
      maven: 'bg-orange-100 text-orange-800',
      npm: 'bg-red-100 text-red-800',
      pypi: 'bg-blue-100 text-blue-800',
      helm: 'bg-purple-100 text-purple-800',
      docker: 'bg-cyan-100 text-cyan-800'
    };
    return colors[type] || 'bg-gray-100 text-gray-800';
  };

  const getCacheLevelColor = (level) => {
    const colors = {
      L1: 'bg-green-100 text-green-800',
      L2: 'bg-yellow-100 text-yellow-800',
      L3: 'bg-blue-100 text-blue-800'
    };
    return colors[level] || 'bg-gray-100 text-gray-800';
  };

  const getCacheLevelIcon = (level) => {
    const icons = {
      L1: Database,
      L2: HardDrive,
      L3: Cloud
    };
    const Icon = icons[level] || Database;
    return <Icon className="w-4 h-4 inline mr-1" />;
  };

  const ScanStatusBadge = ({ status }) => {
    const statusConfig = {
      pending: { color: 'bg-gray-100 text-gray-800', icon: Clock, label: 'Pending' },
      scanning: { color: 'bg-blue-100 text-blue-800', icon: RefreshCw, label: 'Scanning' },
      completed: { color: 'bg-green-100 text-green-800', icon: CheckCircle, label: 'Completed' },
      failed: { color: 'bg-red-100 text-red-800', icon: AlertTriangle, label: 'Failed' }
    };
    
    const config = statusConfig[status] || statusConfig.pending;
    const Icon = config.icon;
    
    return (
      <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${config.color}`}>
        <Icon className="w-3 h-3 mr-1" />
        {config.label}
      </span>
    );
  };

  const StatCard = ({ title, value, subtitle, color, icon: Icon }) => (
    <div className="bg-white rounded-lg shadow p-6">
      <div className="flex items-center justify-between mb-2">
        <h3 className="text-sm font-medium text-gray-600">{title}</h3>
        {Icon && <Icon className="w-5 h-5 text-gray-400" />}
      </div>
      <div className="text-2xl font-bold text-gray-900">{value || 0}</div>
      {subtitle && (
        <div className="mt-2 text-sm text-gray-500">
          {subtitle}
        </div>
      )}
    </div>
  );

  return (
    <div className="cache-management p-6">
      <div className="mb-6">
        <h1 className="text-3xl font-bold text-gray-900">{t('page.title')}</h1>
        <p className="text-gray-600 mt-2">{t('page.description')}</p>
      </div>

      {/* Cache Statistics Dashboard */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
        <StatCard 
          title={t('stats.l1Cache')} 
          value={stats?.l1_cache?.redis?.l1_items || 0}
          subtitle={formatBytes(stats?.l1_cache?.redis?.l1_size_bytes || 0)}
          color="green"
          icon={Database}
        />
        <StatCard 
          title={t('stats.l2Cache')} 
          value={stats?.l1_cache?.redis?.l2_items || 0}
          subtitle={formatBytes(stats?.l1_cache?.redis?.l2_size_bytes || 0)}
          color="yellow"
          icon={HardDrive}
        />
        <StatCard 
          title={t('stats.l3Cache')} 
          value={stats?.l1_cache?.redis?.l3_items || 0}
          subtitle={stats?.l1_cache?.redis?.l3_enabled ? t('stats.enabled') : t('stats.disabled')}
          color="blue"
          icon={Cloud}
        />
        <StatCard 
          title={t('stats.databaseTracking')} 
          value={stats?.database_tracking?.total_items || 0}
          subtitle={formatBytes(stats?.database_tracking?.total_size_bytes || 0)}
          color="purple"
          icon={Package}
        />
      </div>

      {/* Additional Stats Row */}
      {stats?.database_tracking && (
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
          <div className="bg-blue-50 rounded-lg shadow p-4 border border-blue-200">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-blue-600 font-medium">{t('stats.pendingScans')}</p>
                <p className="text-2xl font-bold text-blue-900">{stats.database_tracking.pending_scans || 0}</p>
              </div>
              <Clock className="w-8 h-8 text-blue-400" />
            </div>
          </div>
          <div className="bg-red-50 rounded-lg shadow p-4 border border-red-200">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-red-600 font-medium">{t('stats.quarantinedItems')}</p>
                <p className="text-2xl font-bold text-red-900">{stats.database_tracking.quarantined_items || 0}</p>
              </div>
              <AlertTriangle className="w-8 h-8 text-red-400" />
            </div>
          </div>
          <div className="bg-green-50 rounded-lg shadow p-4 border border-green-200">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-green-600 font-medium">{t('stats.avgHitCount')}</p>
                <p className="text-2xl font-bold text-green-900">{stats.database_tracking.avg_hit_count?.toFixed(1) || 0}</p>
              </div>
              <CheckCircle className="w-8 h-8 text-green-400" />
            </div>
          </div>
        </div>
      )}

      {/* Filters and Actions */}
      <div className="bg-white rounded-lg shadow p-4 mb-6">
        <div className="flex flex-wrap items-center justify-between gap-4">
          <div className="flex gap-4">
            <select 
              value={selectedType} 
              onChange={(e) => {
                setSelectedType(e.target.value);
                setPage(1);
              }}
              className="px-4 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            >
              <option value="all">{t('filters.allTypes')}</option>
              <option value="maven">{t('filters.maven')}</option>
              <option value="npm">{t('filters.npm')}</option>
              <option value="pypi">{t('filters.pypi')}</option>
              <option value="helm">{t('filters.helm')}</option>
              <option value="docker">{t('filters.docker')}</option>
            </select>

            <select 
              value={selectedLevel} 
              onChange={(e) => {
                setSelectedLevel(e.target.value);
                setPage(1);
              }}
              className="px-4 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            >
              <option value="all">{t('filters.allLevels')}</option>
              <option value="L1">{t('filters.l1Redis')}</option>
              <option value="L2">{t('filters.l2Disk')}</option>
              <option value="L3">{t('filters.l3Cloud')}</option>
            </select>
          </div>

          <div className="flex gap-2">
            <button 
              onClick={fetchCacheItems}
              className="inline-flex items-center px-4 py-2 border border-gray-300 rounded-md shadow-sm text-sm font-medium text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <RefreshCw className="w-4 h-4 mr-2" />
              {t('buttons.refresh')}
            </button>
            <button 
              onClick={handleEvictUnused}
              className="inline-flex items-center px-4 py-2 border border-gray-300 rounded-md shadow-sm text-sm font-medium text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-yellow-500"
            >
              <Clock className="w-4 h-4 mr-2" />
              {t('buttons.evictUnused')}
            </button>
            <button 
              onClick={handleCleanupExpired}
              className="inline-flex items-center px-4 py-2 border border-gray-300 rounded-md shadow-sm text-sm font-medium text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-orange-500"
            >
              <AlertTriangle className="w-4 h-4 mr-2" />
              {t('buttons.cleanupExpired')}
            </button>
            <button 
              onClick={handleFlushCache}
              className="inline-flex items-center px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-red-600 hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-red-500"
            >
              <Trash2 className="w-4 h-4 mr-2" />
              {t('buttons.clearAll')}
            </button>
          </div>
        </div>
      </div>

      {/* Cached Items Table */}
      <div className="bg-white rounded-lg shadow overflow-hidden">
        {loading ? (
          <div className="text-center py-12">
            <RefreshCw className="w-8 h-8 animate-spin mx-auto text-gray-400" />
            <p className="mt-2 text-gray-600">{t('messages.loading')}</p>
          </div>
        ) : cacheItems.length === 0 ? (
          <div className="text-center py-12">
            <Database className="w-12 h-12 mx-auto text-gray-400" />
            <p className="mt-2 text-gray-600">{t('messages.noItems')}</p>
          </div>
        ) : (
          <>
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    {t('table.artifactPath')}
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    {t('table.type')}
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    {t('table.cacheLevel')}
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    {t('table.size')}
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    {t('table.hits')}
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    {t('table.lastAccessed')}
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    {t('table.scanStatus')}
                  </th>
                  <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                    {t('table.actions')}
                  </th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {cacheItems.map(item => (
                  <tr key={item.id} className="hover:bg-gray-50">
                    <td className="px-6 py-4 text-sm font-mono text-gray-900 max-w-md truncate" title={item.artifact_path}>
                      {item.artifact_path}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${getTypeColor(item.artifact_type)}`}>
                        {item.artifact_type}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${getCacheLevelColor(item.cache_level)}`}>
                        {getCacheLevelIcon(item.cache_level)}
                        {item.cache_level}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                      {formatBytes(item.size_bytes)}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                      {item.hit_count}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      {formatRelativeTime(item.last_accessed)}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <ScanStatusBadge status={item.scan_status} />
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                      <div className="flex justify-end gap-2">
                        <button 
                          onClick={() => handleScanItem(item.id, item.artifact_path)}
                          disabled={item.scan_status === 'scanning' || scanningItems.has(item.id)}
                          className="text-blue-600 hover:text-blue-900 disabled:text-gray-400 disabled:cursor-not-allowed"
                          title={t('actions.scan')}
                        >
                          <Shield className="w-4 h-4" />
                        </button>
                        <button 
                          onClick={() => handleDownloadItem(item.id, item.artifact_path)}
                          className="text-green-600 hover:text-green-900"
                          title={t('actions.download')}
                        >
                          <Download className="w-4 h-4" />
                        </button>
                        <button 
                          onClick={() => handleDeleteItem(item.id, item.artifact_path)}
                          className="text-red-600 hover:text-red-900"
                          title={t('actions.remove')}
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>

            {/* Pagination */}
            {totalPages > 1 && (
              <div className="bg-white px-4 py-3 flex items-center justify-between border-t border-gray-200 sm:px-6">
                <div className="flex-1 flex justify-between sm:hidden">
                  <button
                    onClick={() => setPage(p => Math.max(1, p - 1))}
                    disabled={page === 1}
                    className="relative inline-flex items-center px-4 py-2 border border-gray-300 text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    {t('buttons.previous')}
                  </button>
                  <button
                    onClick={() => setPage(p => Math.min(totalPages, p + 1))}
                    disabled={page === totalPages}
                    className="ml-3 relative inline-flex items-center px-4 py-2 border border-gray-300 text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    {t('buttons.next')}
                  </button>
                </div>
                <div className="hidden sm:flex-1 sm:flex sm:items-center sm:justify-between">
                  <div>
                    <p className="text-sm text-gray-700">
                      {t('messages.page')} <span className="font-medium">{page}</span> {t('messages.of')} <span className="font-medium">{totalPages}</span>
                    </p>
                  </div>
                  <div>
                    <nav className="relative z-0 inline-flex rounded-md shadow-sm -space-x-px">
                      <button
                        onClick={() => setPage(p => Math.max(1, p - 1))}
                        disabled={page === 1}
                        className="relative inline-flex items-center px-2 py-2 rounded-l-md border border-gray-300 bg-white text-sm font-medium text-gray-500 hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
                      >
                        {t('buttons.previous')}
                      </button>
                      <button
                        onClick={() => setPage(p => Math.min(totalPages, p + 1))}
                        disabled={page === totalPages}
                        className="relative inline-flex items-center px-2 py-2 rounded-r-md border border-gray-300 bg-white text-sm font-medium text-gray-500 hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
                      >
                        {t('buttons.next')}
                      </button>
                    </nav>
                  </div>
                </div>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
};

export default CacheManagement;
