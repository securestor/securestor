import React, { useState, useEffect } from 'react';
import { X, Save, Bell, Mail, MessageSquare, Shield, Lock, Activity, FileText, Package } from 'lucide-react';
import NotificationsAPI from '../../services/api/notificationsAPI';
import { useToast } from '../../context/ToastContext';

/**
 * Notification Preferences Modal
 * Allow users to customize notification settings per channel and category
 */
export const NotificationPreferences = ({ isOpen, onClose }) => {
  const [preferences, setPreferences] = useState({
    channels: {
      in_app: true,
      email: true,
      browser: false
    },
    categories: {
      security: {
        enabled: true,
        email: true,
        priority_threshold: 'medium'
      },
      encryption: {
        enabled: true,
        email: true,
        priority_threshold: 'high'
      },
      compliance: {
        enabled: true,
        email: true,
        priority_threshold: 'medium'
      },
      scan: {
        enabled: true,
        email: false,
        priority_threshold: 'high'
      },
      artifact: {
        enabled: true,
        email: false,
        priority_threshold: 'info'
      },
      system: {
        enabled: true,
        email: true,
        priority_threshold: 'high'
      }
    },
    quiet_hours: {
      enabled: false,
      start: '22:00',
      end: '08:00'
    },
    digest: {
      enabled: false,
      frequency: 'daily', // daily, weekly
      time: '09:00'
    }
  });

  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const { showSuccess, showError } = useToast();

  const categoryIcons = {
    security: Shield,
    encryption: Lock,
    compliance: FileText,
    scan: Activity,
    artifact: Package,
    system: Bell
  };

  const categoryLabels = {
    security: 'Security Alerts',
    encryption: 'Encryption Events',
    compliance: 'Compliance Violations',
    scan: 'Security Scans',
    artifact: 'Artifact Operations',
    system: 'System Notifications'
  };

  const priorityLevels = [
    { value: 'info', label: 'All (Info+)' },
    { value: 'low', label: 'Low+' },
    { value: 'medium', label: 'Medium+' },
    { value: 'high', label: 'High+' },
    { value: 'critical', label: 'Critical Only' }
  ];

  useEffect(() => {
    if (isOpen) {
      fetchPreferences();
    }
  }, [isOpen]);

  const fetchPreferences = async () => {
    setLoading(true);
    try {
      // TODO: Replace with actual API call
      // const prefs = await NotificationsAPI.getPreferences();
      // setPreferences(prefs);
      
      // Mock data loaded
      await new Promise(resolve => setTimeout(resolve, 500));
    } catch (error) {
      console.error('Failed to fetch preferences:', error);
      showError('Failed to load preferences');
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async () => {
    setSaving(true);
    try {
      // await NotificationsAPI.updatePreferences(preferences);
      await new Promise(resolve => setTimeout(resolve, 1000));
      showSuccess('Notification preferences saved successfully');
      onClose();
    } catch (error) {
      console.error('Failed to save preferences:', error);
      showError('Failed to save preferences');
    } finally {
      setSaving(false);
    }
  };

  const updateChannel = (channel, value) => {
    setPreferences(prev => ({
      ...prev,
      channels: {
        ...prev.channels,
        [channel]: value
      }
    }));
  };

  const updateCategory = (category, field, value) => {
    setPreferences(prev => ({
      ...prev,
      categories: {
        ...prev.categories,
        [category]: {
          ...prev.categories[category],
          [field]: value
        }
      }
    }));
  };

  const updateQuietHours = (field, value) => {
    setPreferences(prev => ({
      ...prev,
      quiet_hours: {
        ...prev.quiet_hours,
        [field]: value
      }
    }));
  };

  const updateDigest = (field, value) => {
    setPreferences(prev => ({
      ...prev,
      digest: {
        ...prev.digest,
        [field]: value
      }
    }));
  };

  const requestBrowserPermission = async () => {
    if ('Notification' in window) {
      const permission = await Notification.requestPermission();
      if (permission === 'granted') {
        updateChannel('browser', true);
        showSuccess('Browser notifications enabled');
      } else {
        showError('Browser notification permission denied');
      }
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 overflow-hidden">
      {/* Backdrop */}
      <div 
        className="absolute inset-0 bg-black bg-opacity-50 transition-opacity"
        onClick={onClose}
      />
      
      {/* Modal */}
      <div className="absolute inset-y-0 right-0 max-w-3xl w-full bg-white shadow-2xl flex flex-col">
        {/* Header */}
        <div className="px-6 py-4 border-b border-gray-200 bg-gradient-to-r from-blue-50 to-indigo-50">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-3">
              <div className="p-2 bg-blue-600 rounded-lg">
                <Bell className="w-6 h-6 text-white" />
              </div>
              <div>
                <h2 className="text-xl font-bold text-gray-900">Notification Preferences</h2>
                <p className="text-sm text-gray-600">
                  Customize how you receive notifications
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
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-6 space-y-6">
          {loading ? (
            <div className="flex items-center justify-center h-64">
              <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
            </div>
          ) : (
            <>
              {/* Notification Channels */}
              <section>
                <h3 className="text-lg font-semibold text-gray-900 mb-4">Notification Channels</h3>
                <div className="space-y-3">
                  <label className="flex items-center justify-between p-4 bg-gray-50 rounded-lg hover:bg-gray-100 cursor-pointer">
                    <div className="flex items-center space-x-3">
                      <Bell className="w-5 h-5 text-gray-600" />
                      <div>
                        <p className="font-medium text-gray-900">In-App Notifications</p>
                        <p className="text-sm text-gray-600">Show notifications in the app</p>
                      </div>
                    </div>
                    <input
                      type="checkbox"
                      checked={preferences.channels.in_app}
                      onChange={(e) => updateChannel('in_app', e.target.checked)}
                      className="w-5 h-5 rounded text-blue-600 focus:ring-blue-500"
                    />
                  </label>

                  <label className="flex items-center justify-between p-4 bg-gray-50 rounded-lg hover:bg-gray-100 cursor-pointer">
                    <div className="flex items-center space-x-3">
                      <Mail className="w-5 h-5 text-gray-600" />
                      <div>
                        <p className="font-medium text-gray-900">Email Notifications</p>
                        <p className="text-sm text-gray-600">Receive notifications via email</p>
                      </div>
                    </div>
                    <input
                      type="checkbox"
                      checked={preferences.channels.email}
                      onChange={(e) => updateChannel('email', e.target.checked)}
                      className="w-5 h-5 rounded text-blue-600 focus:ring-blue-500"
                    />
                  </label>

                  <div className="p-4 bg-gray-50 rounded-lg">
                    <div className="flex items-center justify-between mb-2">
                      <div className="flex items-center space-x-3">
                        <MessageSquare className="w-5 h-5 text-gray-600" />
                        <div>
                          <p className="font-medium text-gray-900">Browser Notifications</p>
                          <p className="text-sm text-gray-600">Show desktop notifications</p>
                        </div>
                      </div>
                      <input
                        type="checkbox"
                        checked={preferences.channels.browser}
                        onChange={(e) => {
                          if (e.target.checked && Notification.permission !== 'granted') {
                            requestBrowserPermission();
                          } else {
                            updateChannel('browser', e.target.checked);
                          }
                        }}
                        className="w-5 h-5 rounded text-blue-600 focus:ring-blue-500"
                      />
                    </div>
                    {Notification.permission === 'denied' && (
                      <p className="text-xs text-red-600 ml-8">
                        Browser notifications are blocked. Please enable them in your browser settings.
                      </p>
                    )}
                  </div>
                </div>
              </section>

              {/* Category Preferences */}
              <section>
                <h3 className="text-lg font-semibold text-gray-900 mb-4">Notification Categories</h3>
                <div className="space-y-4">
                  {Object.entries(preferences.categories).map(([category, settings]) => {
                    const Icon = categoryIcons[category];
                    return (
                      <div key={category} className="p-4 bg-gray-50 rounded-lg">
                        <div className="flex items-center justify-between mb-3">
                          <div className="flex items-center space-x-3">
                            <Icon className="w-5 h-5 text-gray-600" />
                            <span className="font-medium text-gray-900">
                              {categoryLabels[category]}
                            </span>
                          </div>
                          <input
                            type="checkbox"
                            checked={settings.enabled}
                            onChange={(e) => updateCategory(category, 'enabled', e.target.checked)}
                            className="w-5 h-5 rounded text-blue-600 focus:ring-blue-500"
                          />
                        </div>
                        
                        {settings.enabled && (
                          <div className="ml-8 space-y-2">
                            <label className="flex items-center space-x-2 text-sm">
                              <input
                                type="checkbox"
                                checked={settings.email}
                                onChange={(e) => updateCategory(category, 'email', e.target.checked)}
                                className="rounded text-blue-600 focus:ring-blue-500"
                              />
                              <span className="text-gray-700">Send email notifications</span>
                            </label>
                            <div className="flex items-center space-x-2 text-sm">
                              <span className="text-gray-700">Priority threshold:</span>
                              <select
                                value={settings.priority_threshold}
                                onChange={(e) => updateCategory(category, 'priority_threshold', e.target.value)}
                                className="px-2 py-1 border border-gray-300 rounded text-sm focus:ring-2 focus:ring-blue-500"
                              >
                                {priorityLevels.map(level => (
                                  <option key={level.value} value={level.value}>
                                    {level.label}
                                  </option>
                                ))}
                              </select>
                            </div>
                          </div>
                        )}
                      </div>
                    );
                  })}
                </div>
              </section>

              {/* Quiet Hours */}
              <section>
                <h3 className="text-lg font-semibold text-gray-900 mb-4">Quiet Hours</h3>
                <div className="p-4 bg-gray-50 rounded-lg space-y-3">
                  <label className="flex items-center space-x-2">
                    <input
                      type="checkbox"
                      checked={preferences.quiet_hours.enabled}
                      onChange={(e) => updateQuietHours('enabled', e.target.checked)}
                      className="rounded text-blue-600 focus:ring-blue-500"
                    />
                    <span className="text-gray-900">Enable quiet hours (no notifications during this time)</span>
                  </label>
                  
                  {preferences.quiet_hours.enabled && (
                    <div className="ml-6 flex items-center space-x-4">
                      <div>
                        <label className="block text-sm text-gray-700 mb-1">Start</label>
                        <input
                          type="time"
                          value={preferences.quiet_hours.start}
                          onChange={(e) => updateQuietHours('start', e.target.value)}
                          className="px-3 py-2 border border-gray-300 rounded focus:ring-2 focus:ring-blue-500"
                        />
                      </div>
                      <div>
                        <label className="block text-sm text-gray-700 mb-1">End</label>
                        <input
                          type="time"
                          value={preferences.quiet_hours.end}
                          onChange={(e) => updateQuietHours('end', e.target.value)}
                          className="px-3 py-2 border border-gray-300 rounded focus:ring-2 focus:ring-blue-500"
                        />
                      </div>
                    </div>
                  )}
                </div>
              </section>

              {/* Digest Settings */}
              <section>
                <h3 className="text-lg font-semibold text-gray-900 mb-4">Digest Summary</h3>
                <div className="p-4 bg-gray-50 rounded-lg space-y-3">
                  <label className="flex items-center space-x-2">
                    <input
                      type="checkbox"
                      checked={preferences.digest.enabled}
                      onChange={(e) => updateDigest('enabled', e.target.checked)}
                      className="rounded text-blue-600 focus:ring-blue-500"
                    />
                    <span className="text-gray-900">Receive periodic digest summaries</span>
                  </label>
                  
                  {preferences.digest.enabled && (
                    <div className="ml-6 space-y-3">
                      <div>
                        <label className="block text-sm text-gray-700 mb-1">Frequency</label>
                        <select
                          value={preferences.digest.frequency}
                          onChange={(e) => updateDigest('frequency', e.target.value)}
                          className="px-3 py-2 border border-gray-300 rounded focus:ring-2 focus:ring-blue-500"
                        >
                          <option value="daily">Daily</option>
                          <option value="weekly">Weekly</option>
                        </select>
                      </div>
                      <div>
                        <label className="block text-sm text-gray-700 mb-1">Time</label>
                        <input
                          type="time"
                          value={preferences.digest.time}
                          onChange={(e) => updateDigest('time', e.target.value)}
                          className="px-3 py-2 border border-gray-300 rounded focus:ring-2 focus:ring-blue-500"
                        />
                      </div>
                    </div>
                  )}
                </div>
              </section>
            </>
          )}
        </div>

        {/* Footer */}
        <div className="px-6 py-4 border-t border-gray-200 bg-gray-50 flex items-center justify-end space-x-3">
          <button
            onClick={onClose}
            className="px-4 py-2 text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition"
          >
            Cancel
          </button>
          <button
            onClick={handleSave}
            disabled={saving}
            className="flex items-center space-x-2 px-4 py-2 text-white bg-blue-600 rounded-lg hover:bg-blue-700 transition disabled:opacity-50"
          >
            {saving ? (
              <>
                <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white"></div>
                <span>Saving...</span>
              </>
            ) : (
              <>
                <Save className="w-4 h-4" />
                <span>Save Preferences</span>
              </>
            )}
          </button>
        </div>
      </div>
    </div>
  );
};

export default NotificationPreferences;
