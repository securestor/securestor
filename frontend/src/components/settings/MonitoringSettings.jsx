import React from 'react';

const MonitoringSettings = ({ 
  settings, 
  updateSettings, 
  onSave, 
  onReset, 
  saving, 
  validationErrors 
}) => {
  const monitoringSettings = settings?.monitoring || {};

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-medium text-gray-900 mb-4">Monitoring Settings</h3>
        <p className="text-sm text-gray-500 mb-6">
          Configure monitoring, analytics, and alerting for your tenant.
        </p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* Metrics Enabled */}
        <div className="space-y-2">
          <label className="flex items-center">
            <input
              type="checkbox"
              checked={monitoringSettings.metrics_enabled || false}
              onChange={(e) => updateSettings('monitoring', {
                ...monitoringSettings,
                metrics_enabled: e.target.checked
              })}
              className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
            />
            <span className="ml-2 text-sm font-medium text-gray-700">
              Enable Metrics Collection
            </span>
          </label>
          <p className="text-xs text-gray-500 ml-6">
            Collect and store performance metrics for analysis.
          </p>
        </div>

        {/* Performance Monitoring */}
        <div className="space-y-2">
          <label className="flex items-center">
            <input
              type="checkbox"
              checked={monitoringSettings.performance_monitoring || false}
              onChange={(e) => updateSettings('monitoring', {
                ...monitoringSettings,
                performance_monitoring: e.target.checked
              })}
              className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
            />
            <span className="ml-2 text-sm font-medium text-gray-700">
              Performance Monitoring
            </span>
          </label>
          <p className="text-xs text-gray-500 ml-6">
            Monitor system performance and resource usage.
          </p>
        </div>

        {/* Error Tracking */}
        <div className="space-y-2">
          <label className="flex items-center">
            <input
              type="checkbox"
              checked={monitoringSettings.error_tracking || false}
              onChange={(e) => updateSettings('monitoring', {
                ...monitoringSettings,
                error_tracking: e.target.checked
              })}
              className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
            />
            <span className="ml-2 text-sm font-medium text-gray-700">
              Error Tracking
            </span>
          </label>
          <p className="text-xs text-gray-500 ml-6">
            Track and analyze application errors and exceptions.
          </p>
        </div>

        {/* Usage Analytics */}
        <div className="space-y-2">
          <label className="flex items-center">
            <input
              type="checkbox"
              checked={monitoringSettings.usage_analytics || false}
              onChange={(e) => updateSettings('monitoring', {
                ...monitoringSettings,
                usage_analytics: e.target.checked
              })}
              className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
            />
            <span className="ml-2 text-sm font-medium text-gray-700">
              Usage Analytics
            </span>
          </label>
          <p className="text-xs text-gray-500 ml-6">
            Collect usage patterns and user behavior analytics.
          </p>
        </div>

        {/* Real-time Alerts */}
        <div className="space-y-2">
          <label className="flex items-center">
            <input
              type="checkbox"
              checked={monitoringSettings.real_time_alerts || false}
              onChange={(e) => updateSettings('monitoring', {
                ...monitoringSettings,
                real_time_alerts: e.target.checked
              })}
              className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
            />
            <span className="ml-2 text-sm font-medium text-gray-700">
              Real-time Alerts
            </span>
          </label>
          <p className="text-xs text-gray-500 ml-6">
            Send immediate notifications for critical events.
          </p>
        </div>

        {/* Log Level */}
        <div className="space-y-2">
          <label className="block text-sm font-medium text-gray-700">
            Log Level
          </label>
          <select
            value={monitoringSettings.log_level || 'info'}
            onChange={(e) => updateSettings('monitoring', {
              ...monitoringSettings,
              log_level: e.target.value
            })}
            className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
          >
            <option value="debug">Debug</option>
            <option value="info">Info</option>
            <option value="warn">Warning</option>
            <option value="error">Error</option>
          </select>
        </div>
      </div>

      {/* Alert Thresholds Section */}
      <div className="border-t pt-6">
        <h4 className="text-md font-medium text-gray-900 mb-4">Alert Thresholds</h4>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div className="space-y-2">
            <label className="block text-sm font-medium text-gray-700">CPU Usage (%)</label>
            <input
              type="number"
              value={monitoringSettings.alert_thresholds?.cpu_usage_percent || 80}
              onChange={(e) => updateSettings('monitoring', {
                ...monitoringSettings,
                alert_thresholds: {
                  ...monitoringSettings.alert_thresholds,
                  cpu_usage_percent: parseInt(e.target.value)
                }
              })}
              className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
              min="1"
              max="100"
            />
          </div>
          <div className="space-y-2">
            <label className="block text-sm font-medium text-gray-700">Memory Usage (%)</label>
            <input
              type="number"
              value={monitoringSettings.alert_thresholds?.memory_usage_percent || 85}
              onChange={(e) => updateSettings('monitoring', {
                ...monitoringSettings,
                alert_thresholds: {
                  ...monitoringSettings.alert_thresholds,
                  memory_usage_percent: parseInt(e.target.value)
                }
              })}
              className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
              min="1"
              max="100"
            />
          </div>
          <div className="space-y-2">
            <label className="block text-sm font-medium text-gray-700">Error Rate (%)</label>
            <input
              type="number"
              value={monitoringSettings.alert_thresholds?.error_rate_percent || 5}
              onChange={(e) => updateSettings('monitoring', {
                ...monitoringSettings,
                alert_thresholds: {
                  ...monitoringSettings.alert_thresholds,
                  error_rate_percent: parseInt(e.target.value)
                }
              })}
              className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
              min="1"
              max="100"
            />
          </div>
        </div>
      </div>

      {/* Retention Settings */}
      <div className="border-t pt-6">
        <h4 className="text-md font-medium text-gray-900 mb-4">Data Retention</h4>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className="space-y-2">
            <label className="block text-sm font-medium text-gray-700">Metrics Retention (days)</label>
            <input
              type="number"
              value={monitoringSettings.metrics_retention_days || 90}
              onChange={(e) => updateSettings('monitoring', {
                ...monitoringSettings,
                metrics_retention_days: parseInt(e.target.value)
              })}
              className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
              min="1"
              max="365"
            />
          </div>
          <div className="space-y-2">
            <label className="block text-sm font-medium text-gray-700">Log Retention (days)</label>
            <input
              type="number"
              value={monitoringSettings.log_retention_days || 30}
              onChange={(e) => updateSettings('monitoring', {
                ...monitoringSettings,
                log_retention_days: parseInt(e.target.value)
              })}
              className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
              min="1"
              max="365"
            />
          </div>
        </div>
      </div>

      {/* Action Buttons */}
      <div className="flex justify-end space-x-3 pt-6 border-t">
        <button
          type="button"
          onClick={onReset}
          className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md shadow-sm hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
        >
          Reset
        </button>
        <button
          type="button"
          onClick={onSave}
          disabled={saving}
          className="px-4 py-2 text-sm font-medium text-white bg-blue-600 border border-transparent rounded-md shadow-sm hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50"
        >
          {saving ? 'Saving...' : 'Save Changes'}
        </button>
      </div>
    </div>
  );
};

export default MonitoringSettings;