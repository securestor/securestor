import React, { useState, useEffect } from 'react';
import { useTranslation } from '../hooks/useTranslation';
import { auditAPI } from '../services/api/auditAPI';
import { 
  Clock, 
  User, 
  Shield, 
  CheckCircle, 
  XCircle, 
  Filter,
  RefreshCw,
  Eye,
  ChevronLeft,
  ChevronRight,
  Activity,
  AlertTriangle,
  FileText,
  Database
} from 'lucide-react';

const AuditLogs = () => {
  const { t } = useTranslation('audit');
  const [logs, setLogs] = useState([]);
  const [stats, setStats] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [selectedLog, setSelectedLog] = useState(null);
  const [showFilters, setShowFilters] = useState(false);
  
  // Pagination state
  const [currentPage, setCurrentPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [totalItems, setTotalItems] = useState(0);
  const [itemsPerPage] = useState(20);

  // Filter state
  const [filters, setFilters] = useState({
    user_id: '',
    event_type: '',
    resource_type: '',
    action: '',
    success: '',
    start_time: '',
    end_time: ''
  });

  const [appliedFilters, setAppliedFilters] = useState({});

  // Load audit logs
  const loadAuditLogs = async (page = 1, filterParams = {}) => {
    try {
      setLoading(true);
      setError(null);

      const params = {
        page,
        limit: itemsPerPage,
        ...filterParams
      };

      const response = await auditAPI.getAuditLogs(params);
      
      setLogs(response.logs || []);
      setCurrentPage(response.pagination?.current_page || 1);
      setTotalPages(response.pagination?.total_pages || 1);
      setTotalItems(response.pagination?.total_items || 0);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  // Load audit statistics
  const loadAuditStats = async (filterParams = {}) => {
    try {
      const statsData = await auditAPI.getAuditStats(filterParams);
      setStats(statsData);
    } catch (err) {
      console.error('Failed to load audit stats:', err);
    }
  };

  // Initial load
  useEffect(() => {
    const loadInitialData = async () => {
      await loadAuditLogs(1);
      await loadAuditStats();
    };
    loadInitialData();
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  // Handle filter changes
  const handleFilterChange = (field, value) => {
    setFilters(prev => ({ ...prev, [field]: value }));
  };

  // Apply filters
  const applyFilters = () => {
    const activeFilters = Object.entries(filters).reduce((acc, [key, value]) => {
      if (value && value !== '') {
        acc[key] = value;
      }
      return acc;
    }, {});

    setAppliedFilters(activeFilters);
    setCurrentPage(1);
    loadAuditLogs(1, activeFilters);
    loadAuditStats(activeFilters);
    setShowFilters(false);
  };

  // Clear filters
  const clearFilters = () => {
    setFilters({
      user_id: '',
      event_type: '',
      resource_type: '',
      action: '',
      success: '',
      start_time: '',
      end_time: ''
    });
    setAppliedFilters({});
    setCurrentPage(1);
    loadAuditLogs(1);
    loadAuditStats();
    setShowFilters(false);
  };

  // Handle pagination
  const handlePageChange = (page) => {
    setCurrentPage(page);
    loadAuditLogs(page, appliedFilters);
  };

  // Refresh data
  const handleRefresh = () => {
    loadAuditLogs(currentPage, appliedFilters);
    loadAuditStats(appliedFilters);
  };

  // Format timestamp
  const formatTimestamp = (timestamp) => {
    return new Date(timestamp).toLocaleString('en-US', {
      year: 'numeric',
      month: 'short',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit'
    });
  };

  // Get event type icon
  const getEventTypeIcon = (eventType) => {
    switch (eventType) {
      case 'user_session': return User;
      case 'security_scan': return Shield;
      case 'compliance': return CheckCircle;
      case 'artifact': return FileText;
      case 'repository': return Database;
      default: return Activity;
    }
  };

  // Get action color
  const getActionColor = (action, success) => {
    if (!success) return 'text-red-600 bg-red-50';
    
    switch (action) {
      case 'login': return 'text-green-600 bg-green-50';
      case 'logout': return 'text-blue-600 bg-blue-50';
      case 'create': return 'text-green-600 bg-green-50';
      case 'update': return 'text-yellow-600 bg-yellow-50';
      case 'delete': return 'text-red-600 bg-red-50';
      case 'scan': return 'text-purple-600 bg-purple-50';
      default: return 'text-gray-600 bg-gray-50';
    }
  };

  // Statistics cards
  const StatCard = ({ icon: Icon, title, value, subtitle, color = 'blue' }) => (
    <div className={`bg-white rounded-lg p-6 shadow-sm border border-gray-200`}>
      <div className="flex items-center">
        <div className={`flex-shrink-0 p-3 bg-${color}-100 rounded-lg`}>
          <Icon className={`w-6 h-6 text-${color}-600`} />
        </div>
        <div className="ml-4">
          <p className="text-sm font-medium text-gray-500">{title}</p>
          <p className="text-2xl font-bold text-gray-900">{value}</p>
          {subtitle && <p className="text-xs text-gray-400">{subtitle}</p>}
        </div>
      </div>
    </div>
  );

  // Log detail modal
  const LogDetailModal = ({ log, onClose }) => {
    if (!log) return null;

    return (
      <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
        <div className="bg-white rounded-lg max-w-2xl w-full max-h-[90vh] overflow-y-auto">
          <div className="p-6 border-b border-gray-200">
            <div className="flex items-center justify-between">
              <h3 className="text-lg font-semibold text-gray-900">{t('modal.title')}</h3>
              <button
                onClick={onClose}
                className="text-gray-400 hover:text-gray-600"
              >
                <XCircle className="w-6 h-6" />
              </button>
            </div>
          </div>
          
          <div className="p-6 space-y-4">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700">{t('modal.id')}</label>
                <p className="mt-1 text-sm text-gray-900 font-mono">{log.id}</p>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700">{t('modal.timestamp')}</label>
                <p className="mt-1 text-sm text-gray-900">{formatTimestamp(log.timestamp)}</p>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700">{t('modal.user')}</label>
                <p className="mt-1 text-sm text-gray-900">{log.user_id}</p>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700">{t('modal.eventType')}</label>
                <p className="mt-1 text-sm text-gray-900">{log.event_type}</p>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700">{t('modal.action')}</label>
                <p className="mt-1 text-sm text-gray-900">{log.action}</p>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700">{t('modal.resourceType')}</label>
                <p className="mt-1 text-sm text-gray-900">{log.resource_type}</p>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700">{t('modal.resourceId')}</label>
                <p className="mt-1 text-sm text-gray-900">{log.resource_id || 'N/A'}</p>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700">{t('modal.status')}</label>
                <div className="mt-1 flex items-center">
                  {log.success ? (
                    <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
                      <CheckCircle className="w-3 h-3 mr-1" />
                      {t('status.success')}
                    </span>
                  ) : (
                    <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-800">
                      <XCircle className="w-3 h-3 mr-1" />
                      {t('status.failed')}
                    </span>
                  )}
                </div>
              </div>
            </div>

            {log.description && (
              <div>
                <label className="block text-sm font-medium text-gray-700">Description</label>
                <p className="mt-1 text-sm text-gray-900">{log.description}</p>
              </div>
            )}

            {log.error_message && (
              <div>
                <label className="block text-sm font-medium text-gray-700">{t('modal.errorMessage')}</label>
                <p className="mt-1 text-sm text-red-600 bg-red-50 p-3 rounded-md font-mono">{log.error_message}</p>
              </div>
            )}

            {log.metadata && Object.keys(log.metadata).length > 0 && (
              <div>
                <label className="block text-sm font-medium text-gray-700">{t('modal.metadata')}</label>
                <pre className="mt-1 text-xs text-gray-900 bg-gray-50 p-3 rounded-md overflow-x-auto">
                  {JSON.stringify(log.metadata, null, 2)}
                </pre>
              </div>
            )}
          </div>
        </div>
      </div>
    );
  };

  if (loading && !logs.length) {
    return (
      <div className="p-6">
        <div className="flex items-center justify-center h-64">
          <RefreshCw className="w-8 h-8 text-gray-400 animate-spin" />
          <span className="ml-2 text-gray-600">{t('messages.loading')}</span>
        </div>
      </div>
    );
  }

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">{t('page.title')}</h1>
          <p className="text-gray-600 mt-1">{t('page.description')}</p>
        </div>
        <div className="flex items-center space-x-3">
          <button
            onClick={() => setShowFilters(!showFilters)}
            className={`inline-flex items-center px-4 py-2 border rounded-md shadow-sm text-sm font-medium ${
              showFilters || Object.keys(appliedFilters).length > 0
                ? 'border-blue-500 text-blue-700 bg-blue-50'
                : 'border-gray-300 text-gray-700 bg-white hover:bg-gray-50'
            }`}
          >
            <Filter className="w-4 h-4 mr-2" />
            {t('buttons.filters')}
            {Object.keys(appliedFilters).length > 0 && (
              <span className="ml-2 bg-blue-600 text-white text-xs rounded-full px-2 py-0.5">
                {Object.keys(appliedFilters).length}
              </span>
            )}
          </button>
          <button
            onClick={handleRefresh}
            disabled={loading}
            className="inline-flex items-center px-4 py-2 border border-gray-300 rounded-md shadow-sm text-sm font-medium text-gray-700 bg-white hover:bg-gray-50 disabled:opacity-50"
          >
            <RefreshCw className={`w-4 h-4 mr-2 ${loading ? 'animate-spin' : ''}`} />
            {t('buttons.refresh')}
          </button>
        </div>
      </div>

      {/* Statistics */}
      {stats && (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
          <StatCard
            icon={FileText}
            title={t('stats.totalLogs')}
            value={stats.total_logs?.toLocaleString() || '0'}
            color="blue"
          />
          <StatCard
            icon={CheckCircle}
            title={t('stats.successfulEvents')}
            value={stats.successful_logs?.toLocaleString() || '0'}
            subtitle={`${(stats.success_rate || 0).toFixed(1)}% ${t('stats.successRate')}`}
            color="green"
          />
          <StatCard
            icon={XCircle}
            title={t('stats.failedEvents')}
            value={stats.failed_logs?.toLocaleString() || '0'}
            color="red"
          />
          <StatCard
            icon={Activity}
            title={t('stats.eventTypes')}
            value={Object.keys(stats.event_types || {}).length}
            color="purple"
          />
        </div>
      )}

      {/* Filters Panel */}
      {showFilters && (
        <div className="bg-white border border-gray-200 rounded-lg p-6">
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">{t('filters.userId')}</label>
              <input
                type="text"
                value={filters.user_id}
                onChange={(e) => handleFilterChange('user_id', e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                placeholder={t('filters.placeholder.userId')}
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">{t('filters.eventType')}</label>
              <select
                value={filters.event_type}
                onChange={(e) => handleFilterChange('event_type', e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              >
                <option value="">{t('filters.allTypes')}</option>
                {stats?.event_types && Object.keys(stats.event_types).map(type => (
                  <option key={type} value={type}>{type}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">{t('filters.action')}</label>
              <select
                value={filters.action}
                onChange={(e) => handleFilterChange('action', e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              >
                <option value="">{t('filters.allActions')}</option>
                {stats?.actions && Object.keys(stats.actions).map(action => (
                  <option key={action} value={action}>{action}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">{t('filters.status')}</label>
              <select
                value={filters.success}
                onChange={(e) => handleFilterChange('success', e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              >
                <option value="">{t('filters.allStatus')}</option>
                <option value="true">{t('filters.success')}</option>
                <option value="false">{t('filters.failed')}</option>
              </select>
            </div>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">{t('filters.startDate')}</label>
              <input
                type="datetime-local"
                value={filters.start_time}
                onChange={(e) => handleFilterChange('start_time', e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">{t('filters.endDate')}</label>
              <input
                type="datetime-local"
                value={filters.end_time}
                onChange={(e) => handleFilterChange('end_time', e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>
          </div>
          <div className="flex items-center space-x-3">
            <button
              onClick={applyFilters}
              className="inline-flex items-center px-4 py-2 bg-blue-600 border border-transparent rounded-md shadow-sm text-sm font-medium text-white hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
            >
              {t('buttons.applyFilters')}
            </button>
            <button
              onClick={clearFilters}
              className="inline-flex items-center px-4 py-2 border border-gray-300 rounded-md shadow-sm text-sm font-medium text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
            >
              {t('buttons.clearAll')}
            </button>
          </div>
        </div>
      )}

      {/* Error State */}
      {error && (
        <div className="bg-red-50 border border-red-200 rounded-md p-4">
          <div className="flex">
            <AlertTriangle className="h-5 w-5 text-red-400" />
            <div className="ml-3">
              <h3 className="text-sm font-medium text-red-800">{t('messages.errorLoading')}</h3>
              <div className="mt-2 text-sm text-red-700">
                <p>{error}</p>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Audit Logs Table */}
      <div className="bg-white border border-gray-200 rounded-lg overflow-hidden">
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  {t('table.timestamp')}
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  {t('table.event')}
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  {t('table.user')}
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  {t('table.action')}
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  {t('table.status')}
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  {t('table.details')}
                </th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {logs.map((log) => {
                const EventIcon = getEventTypeIcon(log.event_type);
                return (
                  <tr key={log.id} className="hover:bg-gray-50">
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                      <div className="flex items-center">
                        <Clock className="w-4 h-4 text-gray-400 mr-2" />
                        {formatTimestamp(log.timestamp)}
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="flex items-center">
                        <EventIcon className="w-4 h-4 text-gray-500 mr-2" />
                        <span className="text-sm text-gray-900">{log.event_type}</span>
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                      {log.user_id}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${getActionColor(log.action, log.success)}`}>
                        {log.action}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      {log.success ? (
                        <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
                          <CheckCircle className="w-3 h-3 mr-1" />
                          {t('status.success')}
                        </span>
                      ) : (
                        <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-800">
                          <XCircle className="w-3 h-3 mr-1" />
                          {t('status.failed')}
                        </span>
                      )}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm font-medium">
                      <button
                        onClick={() => setSelectedLog(log)}
                        className="text-blue-600 hover:text-blue-900 flex items-center"
                      >
                        <Eye className="w-4 h-4 mr-1" />
                        {t('buttons.view')}
                      </button>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>

        {/* Pagination */}
        {totalPages > 1 && (
          <div className="bg-white px-4 py-3 border-t border-gray-200 sm:px-6">
            <div className="flex items-center justify-between">
              <div className="flex-1 flex justify-between sm:hidden">
                <button
                  onClick={() => handlePageChange(currentPage - 1)}
                  disabled={currentPage === 1}
                  className="relative inline-flex items-center px-4 py-2 border border-gray-300 text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {t('buttons.previous')}
                </button>
                <button
                  onClick={() => handlePageChange(currentPage + 1)}
                  disabled={currentPage === totalPages}
                  className="ml-3 relative inline-flex items-center px-4 py-2 border border-gray-300 text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {t('buttons.next')}
                </button>
              </div>
              <div className="hidden sm:flex-1 sm:flex sm:items-center sm:justify-between">
                <div>
                  <p className="text-sm text-gray-700">
                    {t('pagination.showing')}{' '}
                    <span className="font-medium">{(currentPage - 1) * itemsPerPage + 1}</span>
                    {' '}{t('pagination.to')}{' '}
                    <span className="font-medium">
                      {Math.min(currentPage * itemsPerPage, totalItems)}
                    </span>
                    {' '}{t('pagination.of')}{' '}
                    <span className="font-medium">{totalItems}</span>
                    {' '}{t('pagination.results')}
                  </p>
                </div>
                <div>
                  <nav className="relative z-0 inline-flex rounded-md shadow-sm -space-x-px" aria-label="Pagination">
                    <button
                      onClick={() => handlePageChange(currentPage - 1)}
                      disabled={currentPage === 1}
                      className="relative inline-flex items-center px-2 py-2 rounded-l-md border border-gray-300 bg-white text-sm font-medium text-gray-500 hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      <ChevronLeft className="h-5 w-5" />
                    </button>
                    
                    {[...Array(Math.min(5, totalPages))].map((_, i) => {
                      const pageNum = Math.max(1, Math.min(currentPage - 2 + i, totalPages - 4 + i));
                      if (pageNum > totalPages) return null;
                      
                      return (
                        <button
                          key={pageNum}
                          onClick={() => handlePageChange(pageNum)}
                          className={`relative inline-flex items-center px-4 py-2 border text-sm font-medium ${
                            currentPage === pageNum
                              ? 'z-10 bg-blue-50 border-blue-500 text-blue-600'
                              : 'bg-white border-gray-300 text-gray-500 hover:bg-gray-50'
                          }`}
                        >
                          {pageNum}
                        </button>
                      );
                    })}
                    
                    <button
                      onClick={() => handlePageChange(currentPage + 1)}
                      disabled={currentPage === totalPages}
                      className="relative inline-flex items-center px-2 py-2 rounded-r-md border border-gray-300 bg-white text-sm font-medium text-gray-500 hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      <ChevronRight className="h-5 w-5" />
                    </button>
                  </nav>
                </div>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Empty State */}
      {!loading && logs.length === 0 && (
        <div className="text-center py-12">
          <FileText className="mx-auto h-12 w-12 text-gray-400" />
          <h3 className="mt-2 text-sm font-medium text-gray-900">{t('messages.noLogsFound')}</h3>
          <p className="mt-1 text-sm text-gray-500">
            {Object.keys(appliedFilters).length > 0 
              ? t('messages.noLogsFiltered')
              : t('messages.noLogsDefault')
            }
          </p>
          {Object.keys(appliedFilters).length > 0 && (
            <div className="mt-6">
              <button
                onClick={clearFilters}
                className="inline-flex items-center px-4 py-2 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
              >
                {t('buttons.clearFilters')}
              </button>
            </div>
          )}
        </div>
      )}

      {/* Log Detail Modal */}
      <LogDetailModal
        log={selectedLog}
        onClose={() => setSelectedLog(null)}
      />
    </div>
  );
};

export default AuditLogs;