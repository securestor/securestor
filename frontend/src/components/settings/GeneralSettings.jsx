import React from 'react';
import { Save, RotateCcw, Users, Database, Bell, Globe } from 'lucide-react';

// General Settings Component
const GeneralSettings = ({ settings, updateSettings, onSave, onReset, saving, validationErrors }) => {
  if (!settings) return <div>Loading...</div>;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-xl font-semibold text-gray-900">General Settings</h3>
          <p className="text-gray-600 mt-1">Basic tenant configuration and user management</p>
        </div>
        <div className="flex items-center space-x-3">
          <button
            onClick={onReset}
            disabled={saving}
            className="flex items-center px-4 py-2 text-gray-600 hover:text-gray-800 border border-gray-300 rounded-lg disabled:opacity-50"
          >
            <RotateCcw className="w-4 h-4 mr-2" />
            Reset
          </button>
          <button
            onClick={onSave}
            disabled={saving}
            className="flex items-center px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg disabled:opacity-50"
          >
            <Save className="w-4 h-4 mr-2" />
            Save
          </button>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* User Management */}
        <div className="bg-white border border-gray-200 rounded-lg p-6">
          <div className="flex items-center mb-4">
            <Users className="w-5 h-5 text-blue-600 mr-2" />
            <h4 className="text-lg font-medium text-gray-900">User Management</h4>
          </div>
          
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Maximum Users
              </label>
              <input
                type="number"
                value={settings.user_management?.max_users || 10}
                onChange={(e) => updateSettings('user_management', 'max_users', parseInt(e.target.value))}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                min="1"
              />
            </div>

            <div className="flex items-center">
              <input
                type="checkbox"
                id="allowSelfRegistration"
                checked={settings.user_management?.allow_self_registration || false}
                onChange={(e) => updateSettings('user_management', 'allow_self_registration', e.target.checked)}
                className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
              />
              <label htmlFor="allowSelfRegistration" className="ml-2 text-sm text-gray-700">
                Allow self-registration
              </label>
            </div>

            <div className="flex items-center">
              <input
                type="checkbox"
                id="emailVerificationRequired"
                checked={settings.user_management?.email_verification_required || false}
                onChange={(e) => updateSettings('user_management', 'email_verification_required', e.target.checked)}
                className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
              />
              <label htmlFor="emailVerificationRequired" className="ml-2 text-sm text-gray-700">
                Require email verification
              </label>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Invitation Expiry (days)
              </label>
              <input
                type="number"
                value={settings.user_management?.invitation_expiry_days || 7}
                onChange={(e) => updateSettings('user_management', 'invitation_expiry_days', parseInt(e.target.value))}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                min="1"
                max="30"
              />
            </div>
          </div>
        </div>

        {/* Storage Settings */}
        <div className="bg-white border border-gray-200 rounded-lg p-6">
          <div className="flex items-center mb-4">
            <Database className="w-5 h-5 text-green-600 mr-2" />
            <h4 className="text-lg font-medium text-gray-900">Storage Settings</h4>
          </div>
          
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Maximum Storage (GB)
              </label>
              <input
                type="number"
                value={settings.storage?.max_storage_gb || 5}
                onChange={(e) => updateSettings('storage', 'max_storage_gb', parseInt(e.target.value))}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                min="1"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Maximum File Size (MB)
              </label>
              <input
                type="number"
                value={settings.storage?.max_file_size || 100}
                onChange={(e) => updateSettings('storage', 'max_file_size', parseInt(e.target.value))}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                min="1"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Retention Policy (days)
              </label>
              <input
                type="number"
                value={settings.storage?.retention_policy_days || 365}
                onChange={(e) => updateSettings('storage', 'retention_policy_days', parseInt(e.target.value))}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                min="30"
              />
            </div>

            <div className="flex items-center">
              <input
                type="checkbox"
                id="backupEnabled"
                checked={settings.storage?.backup_enabled || false}
                onChange={(e) => updateSettings('storage', 'backup_enabled', e.target.checked)}
                className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
              />
              <label htmlFor="backupEnabled" className="ml-2 text-sm text-gray-700">
                Enable automatic backups
              </label>
            </div>

            {/* Encryption Status - Enterprise Mode (Read-Only) */}
            <div className="mt-4 pt-4 border-t border-gray-200">
              <div className="flex items-center justify-between">
                <div className="flex items-center">
                  <div className="w-2 h-2 bg-green-500 rounded-full mr-2 animate-pulse"></div>
                  <span className="text-sm font-medium text-gray-900">Encryption at Rest</span>
                </div>
                <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
                  ACTIVE & ENFORCED
                </span>
              </div>
              <p className="text-xs text-gray-600 mt-1 ml-4">
                Enterprise-grade AES-256-GCM encryption is enabled system-wide. All artifacts are automatically encrypted.
              </p>
            </div>
          </div>
        </div>

        {/* Notification Settings */}
        <div className="bg-white border border-gray-200 rounded-lg p-6">
          <div className="flex items-center mb-4">
            <Bell className="w-5 h-5 text-yellow-600 mr-2" />
            <h4 className="text-lg font-medium text-gray-900">Notification Settings</h4>
          </div>
          
          <div className="space-y-4">
            <div className="flex items-center">
              <input
                type="checkbox"
                id="emailNotifications"
                checked={settings.notifications?.email_notifications || false}
                onChange={(e) => updateSettings('notifications', 'email_notifications', e.target.checked)}
                className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
              />
              <label htmlFor="emailNotifications" className="ml-2 text-sm text-gray-700">
                Email notifications
              </label>
            </div>

            <div className="flex items-center">
              <input
                type="checkbox"
                id="securityAlerts"
                checked={settings.notifications?.security_alerts || false}
                onChange={(e) => updateSettings('notifications', 'security_alerts', e.target.checked)}
                className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
              />
              <label htmlFor="securityAlerts" className="ml-2 text-sm text-gray-700">
                Security alerts
              </label>
            </div>

            <div className="flex items-center">
              <input
                type="checkbox"
                id="systemAlerts"
                checked={settings.notifications?.system_alerts || false}
                onChange={(e) => updateSettings('notifications', 'system_alerts', e.target.checked)}
                className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
              />
              <label htmlFor="systemAlerts" className="ml-2 text-sm text-gray-700">
                System alerts
              </label>
            </div>

            <div className="flex items-center">
              <input
                type="checkbox"
                id="complianceAlerts"
                checked={settings.notifications?.compliance_alerts || false}
                onChange={(e) => updateSettings('notifications', 'compliance_alerts', e.target.checked)}
                className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
              />
              <label htmlFor="complianceAlerts" className="ml-2 text-sm text-gray-700">
                Compliance alerts
              </label>
            </div>
          </div>
        </div>

        {/* Integration Settings */}
        <div className="bg-white border border-gray-200 rounded-lg p-6">
          <div className="flex items-center mb-4">
            <Globe className="w-5 h-5 text-purple-600 mr-2" />
            <h4 className="text-lg font-medium text-gray-900">Integration Settings</h4>
          </div>
          
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Webhook Endpoints
              </label>
              <textarea
                value={(settings.integrations?.webhook_endpoints || []).join('\n')}
                onChange={(e) => updateSettings('integrations', 'webhook_endpoints', e.target.value.split('\n').filter(url => url.trim()))}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                rows="3"
                placeholder="Enter webhook URLs, one per line"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                API Rate Limits (requests/hour)
              </label>
              <div className="grid grid-cols-2 gap-2">
                <div>
                  <label className="block text-xs text-gray-500 mb-1">Default</label>
                  <input
                    type="number"
                    value={settings.integrations?.api_rate_limits?.default || 1000}
                    onChange={(e) => updateSettings('integrations', 'api_rate_limits', {
                      ...settings.integrations?.api_rate_limits,
                      default: parseInt(e.target.value)
                    })}
                    className="w-full px-2 py-1 border border-gray-300 rounded text-sm"
                    min="1"
                  />
                </div>
                <div>
                  <label className="block text-xs text-gray-500 mb-1">Premium</label>
                  <input
                    type="number"
                    value={settings.integrations?.api_rate_limits?.premium || 5000}
                    onChange={(e) => updateSettings('integrations', 'api_rate_limits', {
                      ...settings.integrations?.api_rate_limits,
                      premium: parseInt(e.target.value)
                    })}
                    className="w-full px-2 py-1 border border-gray-300 rounded text-sm"
                    min="1"
                  />
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Validation Errors */}
      {validationErrors.general && (
        <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg">
          <ul className="list-disc list-inside space-y-1">
            {validationErrors.general.map((error, index) => (
              <li key={index}>{error}</li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
};

export default GeneralSettings;