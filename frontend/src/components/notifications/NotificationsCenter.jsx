import React, { useState, useEffect } from 'react';
import { 
  Bell, 
  X, 
  Check, 
  CheckCheck, 
  Trash2, 
  Filter, 
  AlertTriangle,
  Info,
  Shield,
  Activity,
  Lock,
  FileText,
  Package,
  Search,
  ChevronDown,
  Settings
} from 'lucide-react';
import NotificationsAPI from '../../services/api/notificationsAPI';
import { useToast } from '../../context/ToastContext';
import { NotificationPreferences } from './NotificationPreferences';

/**
 * Enterprise-Grade Notifications Center
 * Features: Categorization, filtering, bulk actions, real-time updates, search
 */
export const NotificationsCenter = ({ isOpen, onClose }) => {
  const [notifications, setNotifications] = useState([]);
  const [loading, setLoading] = useState(false);
  const [selectedType, setSelectedType] = useState('all');
  const [selectedPriority, setSelectedPriority] = useState('all');
  const [showUnreadOnly, setShowUnreadOnly] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedNotifications, setSelectedNotifications] = useState(new Set());
  const [stats, setStats] = useState(null);
  const [page, setPage] = useState(1);
  const [showPreferences, setShowPreferences] = useState(false);
  const { showSuccess, showError } = useToast();

  // Notification types with icons and colors
  const notificationTypes = {
    security: { icon: Shield, color: 'red', label: 'Security' },
    encryption: { icon: Lock, color: 'blue', label: 'Encryption' },
    compliance: { icon: FileText, color: 'yellow', label: 'Compliance' },
    scan: { icon: Activity, color: 'purple', label: 'Security Scan' },
    artifact: { icon: Package, color: 'green', label: 'Artifact' },
    system: { icon: Info, color: 'gray', label: 'System' },
    alert: { icon: AlertTriangle, color: 'orange', label: 'Alert' }
  };

  // Priority levels
  const priorityLevels = {
    critical: { color: 'red', label: 'Critical', bgColor: 'bg-red-100', textColor: 'text-red-800' },
    high: { color: 'orange', label: 'High', bgColor: 'bg-orange-100', textColor: 'text-orange-800' },
    medium: { color: 'yellow', label: 'Medium', bgColor: 'bg-yellow-100', textColor: 'text-yellow-800' },
    low: { color: 'blue', label: 'Low', bgColor: 'bg-blue-100', textColor: 'text-blue-800' },
    info: { color: 'gray', label: 'Info', bgColor: 'bg-gray-100', textColor: 'text-gray-800' }
  };

  // Mock notifications - Replace with API call
  const mockNotifications = [
    {
      id: '1',
      type: 'encryption',
      priority: 'high',
      title: 'Encryption Key Rotation Required',
      message: 'Tenant master key approaching 90-day rotation schedule. Rotation recommended within 5 days.',
      timestamp: new Date(Date.now() - 1000 * 60 * 30).toISOString(),
      read: false,
      metadata: {
        tenant_id: 'tenant-123',
        days_until_rotation: 5,
        current_key_version: 1
      }
    },
    {
      id: '2',
      type: 'security',
      priority: 'critical',
      title: 'Security Scan: Critical Vulnerability Detected',
      message: 'Artifact "test-package-1.0.0" contains 3 critical vulnerabilities (CVE-2024-1234, CVE-2024-5678, CVE-2024-9012)',
      timestamp: new Date(Date.now() - 1000 * 60 * 45).toISOString(),
      read: false,
      metadata: {
        artifact_id: 'artifact-456',
        vulnerability_count: 3,
        scan_id: 'scan-789'
      }
    },
    {
      id: '3',
      type: 'artifact',
      priority: 'info',
      title: 'New Artifact Deployed',
      message: 'Successfully deployed "my-app:v2.1.0" to repository "docker-local" (encrypted, 245 MB)',
      timestamp: new Date(Date.now() - 1000 * 60 * 60).toISOString(),
      read: false,
      metadata: {
        artifact_name: 'my-app:v2.1.0',
        repository: 'docker-local',
        size: 245000000
      }
    },
    {
      id: '4',
      type: 'compliance',
      priority: 'medium',
      title: 'Compliance Policy Violation',
      message: 'Artifact "legacy-lib-0.9.0" violates license policy: GPL-3.0 license detected',
      timestamp: new Date(Date.now() - 1000 * 60 * 90).toISOString(),
      read: true,
      metadata: {
        artifact_id: 'artifact-012',
        policy_id: 'policy-345',
        violation_type: 'license'
      }
    },
    {
      id: '5',
      type: 'scan',
      priority: 'low',
      title: 'Security Scan Completed',
      message: 'Scan completed for "npm-package-3.2.1" - No vulnerabilities found',
      timestamp: new Date(Date.now() - 1000 * 60 * 120).toISOString(),
      read: true,
      metadata: {
        artifact_id: 'artifact-678',
        scan_duration: '45s',
        vulnerabilities_found: 0
      }
    },
    {
      id: '6',
      type: 'system',
      priority: 'info',
      title: 'Storage Quota Warning',
      message: 'Storage usage at 75% for tenant "acme-corp" (7.5 GB of 10 GB used)',
      timestamp: new Date(Date.now() - 1000 * 60 * 180).toISOString(),
      read: true,
      metadata: {
        tenant_id: 'tenant-123',
        usage_percent: 75,
        used_gb: 7.5,
        total_gb: 10
      }
    }
  ];

  useEffect(() => {
    if (isOpen) {
      fetchNotifications();
      fetchStats();
    }
  }, [isOpen, selectedType, selectedPriority, showUnreadOnly, page]);

  const fetchNotifications = async () => {
    setLoading(true);
    try {
      // TODO: Replace with actual API call
      // const response = await NotificationsAPI.getNotifications({
      //   page,
      //   type: selectedType !== 'all' ? selectedType : undefined,
      //   priority: selectedPriority !== 'all' ? selectedPriority : undefined,
      //   unread: showUnreadOnly
      // });
      
      // Simulate API delay
      await new Promise(resolve => setTimeout(resolve, 500));
      
      let filtered = [...mockNotifications];
      if (selectedType !== 'all') {
        filtered = filtered.filter(n => n.type === selectedType);
      }
      if (selectedPriority !== 'all') {
        filtered = filtered.filter(n => n.priority === selectedPriority);
      }
      if (showUnreadOnly) {
        filtered = filtered.filter(n => !n.read);
      }
      if (searchQuery) {
        filtered = filtered.filter(n => 
          n.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
          n.message.toLowerCase().includes(searchQuery.toLowerCase())
        );
      }
      
      setNotifications(filtered);
    } catch (error) {
      console.error('Failed to fetch notifications:', error);
      showError('Failed to load notifications');
    } finally {
      setLoading(false);
    }
  };

  const fetchStats = async () => {
    try {
      // TODO: Replace with actual API call
      setStats({
        total: mockNotifications.length,
        unread: mockNotifications.filter(n => !n.read).length,
        by_type: {
          security: 1,
          encryption: 1,
          compliance: 1,
          scan: 1,
          artifact: 1,
          system: 1
        }
      });
    } catch (error) {
      console.error('Failed to fetch stats:', error);
    }
  };

  const handleMarkAsRead = async (notificationId) => {
    try {
      // await NotificationsAPI.markAsRead(notificationId);
      setNotifications(prev => 
        prev.map(n => n.id === notificationId ? { ...n, read: true } : n)
      );
      showSuccess('Marked as read');
    } catch (error) {
      showError('Failed to mark as read');
    }
  };

  const handleMarkAllAsRead = async () => {
    try {
      // await NotificationsAPI.markAllAsRead();
      setNotifications(prev => prev.map(n => ({ ...n, read: true })));
      showSuccess('All notifications marked as read');
    } catch (error) {
      showError('Failed to mark all as read');
    }
  };

  const handleDelete = async (notificationId) => {
    try {
      // await NotificationsAPI.deleteNotification(notificationId);
      setNotifications(prev => prev.filter(n => n.id !== notificationId));
      showSuccess('Notification deleted');
    } catch (error) {
      showError('Failed to delete notification');
    }
  };

  const handleBulkAction = async (action) => {
    const ids = Array.from(selectedNotifications);
    try {
      if (action === 'read') {
        // await NotificationsAPI.markMultipleAsRead(ids);
        setNotifications(prev => 
          prev.map(n => ids.includes(n.id) ? { ...n, read: true } : n)
        );
        showSuccess(`Marked ${ids.length} notifications as read`);
      } else if (action === 'delete') {
        // await NotificationsAPI.deleteMultiple(ids);
        setNotifications(prev => prev.filter(n => !ids.includes(n.id)));
        showSuccess(`Deleted ${ids.length} notifications`);
      }
      setSelectedNotifications(new Set());
    } catch (error) {
      showError(`Failed to ${action} notifications`);
    }
  };

  const toggleSelection = (notificationId) => {
    setSelectedNotifications(prev => {
      const newSet = new Set(prev);
      if (newSet.has(notificationId)) {
        newSet.delete(notificationId);
      } else {
        newSet.add(notificationId);
      }
      return newSet;
    });
  };

  const selectAll = () => {
    setSelectedNotifications(new Set(notifications.map(n => n.id)));
  };

  const deselectAll = () => {
    setSelectedNotifications(new Set());
  };

  const formatTimestamp = (timestamp) => {
    const date = new Date(timestamp);
    const now = new Date();
    const diff = now - date;
    
    const minutes = Math.floor(diff / 60000);
    const hours = Math.floor(diff / 3600000);
    const days = Math.floor(diff / 86400000);
    
    if (minutes < 1) return 'Just now';
    if (minutes < 60) return `${minutes}m ago`;
    if (hours < 24) return `${hours}h ago`;
    if (days < 7) return `${days}d ago`;
    return date.toLocaleDateString();
  };

  const NotificationIcon = ({ type }) => {
    const config = notificationTypes[type] || notificationTypes.system;
    const Icon = config.icon;
    return <Icon className="w-5 h-5" />;
  };

  const PriorityBadge = ({ priority }) => {
    const config = priorityLevels[priority] || priorityLevels.info;
    return (
      <span className={`px-2 py-0.5 rounded-full text-xs font-semibold ${config.bgColor} ${config.textColor}`}>
        {config.label}
      </span>
    );
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 overflow-hidden">
      {/* Backdrop */}
      <div 
        className="absolute inset-0 bg-black bg-opacity-50 transition-opacity"
        onClick={onClose}
      />
      
      {/* Slide-over panel */}
      <div className="absolute inset-y-0 right-0 max-w-2xl w-full bg-white shadow-2xl flex flex-col">
        {/* Header */}
        <div className="px-6 py-4 border-b border-gray-200 bg-gradient-to-r from-blue-50 to-indigo-50">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-3">
              <div className="p-2 bg-blue-600 rounded-lg">
                <Bell className="w-6 h-6 text-white" />
              </div>
              <div>
                <h2 className="text-xl font-bold text-gray-900">Notifications Center</h2>
                <p className="text-sm text-gray-600">
                  {stats && (
                    <span>
                      {stats.unread} unread of {stats.total} total
                    </span>
                  )}
                </p>
              </div>
            </div>
            <button
              onClick={onClose}
              className="p-2 hover:bg-white rounded-lg transition"
            >
              <X className="w-5 h-5 text-gray-600" />
            </button>
          </div>

          {/* Quick Stats */}
          {stats && (
            <div className="mt-4 grid grid-cols-3 gap-3">
              <div className="bg-white rounded-lg p-3 border border-gray-200">
                <div className="text-2xl font-bold text-red-600">
                  {stats.by_type.security || 0}
                </div>
                <div className="text-xs text-gray-600">Security</div>
              </div>
              <div className="bg-white rounded-lg p-3 border border-gray-200">
                <div className="text-2xl font-bold text-blue-600">
                  {stats.by_type.encryption || 0}
                </div>
                <div className="text-xs text-gray-600">Encryption</div>
              </div>
              <div className="bg-white rounded-lg p-3 border border-gray-200">
                <div className="text-2xl font-bold text-yellow-600">
                  {stats.by_type.compliance || 0}
                </div>
                <div className="text-xs text-gray-600">Compliance</div>
              </div>
            </div>
          )}
        </div>

        {/* Filters and Actions */}
        <div className="px-6 py-4 border-b border-gray-200 bg-gray-50">
          {/* Search */}
          <div className="mb-3">
            <div className="relative">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-gray-400" />
              <input
                type="text"
                placeholder="Search notifications..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              />
            </div>
          </div>

          <div className="flex items-center justify-between space-x-3">
            {/* Type Filter */}
            <select
              value={selectedType}
              onChange={(e) => setSelectedType(e.target.value)}
              className="px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500"
            >
              <option value="all">All Types</option>
              {Object.entries(notificationTypes).map(([key, value]) => (
                <option key={key} value={key}>{value.label}</option>
              ))}
            </select>

            {/* Priority Filter */}
            <select
              value={selectedPriority}
              onChange={(e) => setSelectedPriority(e.target.value)}
              className="px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500"
            >
              <option value="all">All Priorities</option>
              {Object.entries(priorityLevels).map(([key, value]) => (
                <option key={key} value={key}>{value.label}</option>
              ))}
            </select>

            {/* Unread Filter */}
            <label className="flex items-center space-x-2 text-sm">
              <input
                type="checkbox"
                checked={showUnreadOnly}
                onChange={(e) => setShowUnreadOnly(e.target.checked)}
                className="rounded text-blue-600 focus:ring-blue-500"
              />
              <span className="text-gray-700">Unread only</span>
            </label>
          </div>

          {/* Bulk Actions */}
          {selectedNotifications.size > 0 && (
            <div className="mt-3 flex items-center justify-between p-3 bg-blue-50 border border-blue-200 rounded-lg">
              <span className="text-sm font-medium text-blue-900">
                {selectedNotifications.size} selected
              </span>
              <div className="flex items-center space-x-2">
                <button
                  onClick={() => handleBulkAction('read')}
                  className="px-3 py-1 text-sm text-blue-700 hover:text-blue-900 font-medium"
                >
                  Mark as Read
                </button>
                <button
                  onClick={() => handleBulkAction('delete')}
                  className="px-3 py-1 text-sm text-red-700 hover:text-red-900 font-medium"
                >
                  Delete
                </button>
                <button
                  onClick={deselectAll}
                  className="px-3 py-1 text-sm text-gray-700 hover:text-gray-900 font-medium"
                >
                  Deselect
                </button>
              </div>
            </div>
          )}

          {/* Quick Actions */}
          <div className="mt-3 flex items-center justify-between">
            <button
              onClick={selectAll}
              className="text-sm text-blue-600 hover:text-blue-800 font-medium"
            >
              Select All
            </button>
            <button
              onClick={handleMarkAllAsRead}
              className="flex items-center space-x-1 text-sm text-blue-600 hover:text-blue-800 font-medium"
            >
              <CheckCheck className="w-4 h-4" />
              <span>Mark All as Read</span>
            </button>
          </div>
        </div>

        {/* Notifications List */}
        <div className="flex-1 overflow-y-auto">
          {loading ? (
            <div className="flex items-center justify-center h-64">
              <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
            </div>
          ) : notifications.length === 0 ? (
            <div className="flex flex-col items-center justify-center h-64 text-gray-500">
              <Bell className="w-16 h-16 mb-4 opacity-50" />
              <p className="text-lg font-medium">No notifications</p>
              <p className="text-sm">You're all caught up!</p>
            </div>
          ) : (
            <div className="divide-y divide-gray-200">
              {notifications.map((notification) => (
                <div
                  key={notification.id}
                  className={`px-6 py-4 hover:bg-gray-50 transition ${
                    !notification.read ? 'bg-blue-50 border-l-4 border-blue-500' : ''
                  }`}
                >
                  <div className="flex items-start space-x-3">
                    {/* Checkbox */}
                    <input
                      type="checkbox"
                      checked={selectedNotifications.has(notification.id)}
                      onChange={() => toggleSelection(notification.id)}
                      className="mt-1 rounded text-blue-600 focus:ring-blue-500"
                    />

                    {/* Icon */}
                    <div className={`p-2 rounded-lg ${
                      notification.read ? 'bg-gray-100' : 'bg-blue-100'
                    }`}>
                      <NotificationIcon type={notification.type} />
                    </div>

                    {/* Content */}
                    <div className="flex-1 min-w-0">
                      <div className="flex items-start justify-between mb-1">
                        <h4 className="text-sm font-semibold text-gray-900">
                          {notification.title}
                        </h4>
                        <PriorityBadge priority={notification.priority} />
                      </div>
                      <p className="text-sm text-gray-700 mb-2">
                        {notification.message}
                      </p>
                      <div className="flex items-center justify-between">
                        <span className="text-xs text-gray-500">
                          {formatTimestamp(notification.timestamp)}
                        </span>
                        <div className="flex items-center space-x-2">
                          {!notification.read && (
                            <button
                              onClick={() => handleMarkAsRead(notification.id)}
                              className="text-xs text-blue-600 hover:text-blue-800 font-medium"
                            >
                              Mark as read
                            </button>
                          )}
                          <button
                            onClick={() => handleDelete(notification.id)}
                            className="text-xs text-red-600 hover:text-red-800 font-medium"
                          >
                            Delete
                          </button>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="px-6 py-4 border-t border-gray-200 bg-gray-50">
          <div className="flex items-center justify-between">
            <button 
              onClick={() => setShowPreferences(true)}
              className="flex items-center space-x-2 text-sm text-gray-600 hover:text-gray-900 transition"
            >
              <Settings className="w-4 h-4" />
              <span>Notification Preferences</span>
            </button>
            <div className="flex items-center space-x-2">
              <button
                onClick={() => setPage(Math.max(1, page - 1))}
                disabled={page === 1}
                className="px-3 py-1 text-sm border border-gray-300 rounded hover:bg-gray-100 disabled:opacity-50"
              >
                Previous
              </button>
              <span className="text-sm text-gray-600">Page {page}</span>
              <button
                onClick={() => setPage(page + 1)}
                className="px-3 py-1 text-sm border border-gray-300 rounded hover:bg-gray-100"
              >
                Next
              </button>
            </div>
          </div>
        </div>
      </div>

      {/* Notification Preferences Modal */}
      <NotificationPreferences
        isOpen={showPreferences}
        onClose={() => setShowPreferences(false)}
      />
    </div>
  );
};

export default NotificationsCenter;
