import React, { useState, useEffect } from 'react';
import { useParams } from 'react-router-dom';
import {
  Settings,
  Shield,
  Users,
  Database,
  Bell,
  Link as LinkIcon,
  FileText,
  Save,
  RefreshCw
} from 'lucide-react';
import { tenantApi } from '../../services/tenantApi';

const TenantSettings = () => {
  const { tenantId } = useParams();
  const [tenant, setTenant] = useState(null);
  const [settings, setSettings] = useState(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState(null);
  const [activeTab, setActiveTab] = useState('security');

  useEffect(() => {
    fetchTenantData();
  }, [tenantId]);

  const fetchTenantData = async () => {
    try {
      setLoading(true);
      const [tenantData, settingsData] = await Promise.all([
        tenantApi.getTenant(tenantId),
        tenantApi.getTenantSettings(tenantId)
      ]);
      setTenant(tenantData);
      setSettings(settingsData);
    } catch (err) {
      setError('Failed to fetch tenant data');
      console.error('Error fetching tenant data:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleSaveSettings = async () => {
    try {
      setSaving(true);
      await tenantApi.updateTenantSettings(tenantId, settings);
      setError(null);
      // You might want to show a success message here
    } catch (err) {
      setError('Failed to save settings');
      console.error('Error saving settings:', err);
    } finally {
      setSaving(false);
    }
  };

  const updateSettingSection = (section, updates) => {
    setSettings(prev => ({
      ...prev,
      [section]: {
        ...prev[section],
        ...updates
      }
    }));
  };

  const tabs = [
    { id: 'security', label: 'Security', icon: Shield },
    { id: 'users', label: 'User Management', icon: Users },
    { id: 'storage', label: 'Storage', icon: Database },
    { id: 'notifications', label: 'Notifications', icon: Bell },
    { id: 'integrations', label: 'Integrations', icon: LinkIcon },
    { id: 'compliance', label: 'Compliance', icon: FileText }
  ];

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-64">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
      </div>
    );
  }

  if (error && !settings) {
    return (
      <div className="bg-red-50 border border-red-200 rounded-lg p-4">
        <p className="text-red-600">{error}</p>
        <button
          onClick={fetchTenantData}
          className="mt-2 text-red-600 hover:text-red-800 underline"
        >
          Try again
        </button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 flex items-center gap-2">
            <Settings className="h-6 w-6" />
            Tenant Settings
          </h1>
          <p className="text-gray-600 mt-1">
            Configure settings for {tenant?.name}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={fetchTenantData}
            className="bg-gray-100 hover:bg-gray-200 text-gray-700 px-4 py-2 rounded-lg font-medium inline-flex items-center gap-2 transition-colors"
          >
            <RefreshCw className="h-4 w-4" />
            Refresh
          </button>
          <button
            onClick={handleSaveSettings}
            disabled={saving}
            className="bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white px-4 py-2 rounded-lg font-medium inline-flex items-center gap-2 transition-colors"
          >
            <Save className="h-4 w-4" />
            {saving ? 'Saving...' : 'Save Changes'}
          </button>
        </div>
      </div>

      {/* Error Message */}
      {error && (
        <div className="bg-red-50 border border-red-200 rounded-lg p-4">
          <p className="text-red-600">{error}</p>
        </div>
      )}

      <div className="flex gap-6">
        {/* Sidebar */}
        <div className="w-64 flex-shrink-0">
          <nav className="space-y-1">
            {tabs.map((tab) => {
              const Icon = tab.icon;
              return (
                <button
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id)}
                  className={`w-full flex items-center gap-3 px-3 py-2 text-left rounded-lg transition-colors ${
                    activeTab === tab.id
                      ? 'bg-blue-50 text-blue-700 border border-blue-200'
                      : 'text-gray-600 hover:bg-gray-50 hover:text-gray-900'
                  }`}
                >
                  <Icon className="h-4 w-4" />
                  {tab.label}
                </button>
              );
            })}
          </nav>
        </div>

        {/* Content */}
        <div className="flex-1">
          <div className="bg-white rounded-lg border border-gray-200 p-6">
            {activeTab === 'security' && (
              <SecuritySettings
                settings={settings?.security_settings || {}}
                onChange={(updates) => updateSettingSection('security_settings', updates)}
              />
            )}
            {activeTab === 'users' && (
              <UserSettings
                settings={settings?.user_settings || {}}
                onChange={(updates) => updateSettingSection('user_settings', updates)}
              />
            )}
            {activeTab === 'storage' && (
              <StorageSettings
                settings={settings?.storage_settings || {}}
                onChange={(updates) => updateSettingSection('storage_settings', updates)}
              />
            )}
            {activeTab === 'notifications' && (
              <NotificationSettings
                settings={settings?.notification_settings || {}}
                onChange={(updates) => updateSettingSection('notification_settings', updates)}
              />
            )}
            {activeTab === 'integrations' && (
              <IntegrationSettings
                settings={settings?.integration_settings || {}}
                onChange={(updates) => updateSettingSection('integration_settings', updates)}
              />
            )}
            {activeTab === 'compliance' && (
              <ComplianceSettings
                settings={settings?.compliance_settings || {}}
                onChange={(updates) => updateSettingSection('compliance_settings', updates)}
              />
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

// Security Settings Component
const SecuritySettings = ({ settings, onChange }) => {
  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-medium text-gray-900 mb-4">Security Settings</h3>
        
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <h4 className="text-sm font-medium text-gray-900">Multi-Factor Authentication</h4>
              <p className="text-sm text-gray-500">Require MFA for all users</p>
            </div>
            <label className="relative inline-flex items-center cursor-pointer">
              <input
                type="checkbox"
                checked={settings.mfa_required || false}
                onChange={(e) => onChange({ mfa_required: e.target.checked })}
                className="sr-only peer"
              />
              <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
            </label>
          </div>

          <div className="flex items-center justify-between">
            <div>
              <h4 className="text-sm font-medium text-gray-900">Single Sign-On Required</h4>
              <p className="text-sm text-gray-500">Force users to use SSO for authentication</p>
            </div>
            <label className="relative inline-flex items-center cursor-pointer">
              <input
                type="checkbox"
                checked={settings.require_sso || false}
                onChange={(e) => onChange({ require_sso: e.target.checked })}
                className="sr-only peer"
              />
              <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
            </label>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Session Timeout (minutes)
            </label>
            <input
              type="number"
              min="5"
              max="480"
              value={settings.session_timeout_minutes || 30}
              onChange={(e) => onChange({ session_timeout_minutes: parseInt(e.target.value) })}
              className="w-32 px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Maximum Login Attempts
            </label>
            <input
              type="number"
              min="3"
              max="10"
              value={settings.max_login_attempts || 5}
              onChange={(e) => onChange({ max_login_attempts: parseInt(e.target.value) })}
              className="w-32 px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>
        </div>
      </div>
    </div>
  );
};

// User Settings Component
const UserSettings = ({ settings, onChange }) => {
  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-medium text-gray-900 mb-4">User Management Settings</h3>
        
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <h4 className="text-sm font-medium text-gray-900">Allow Self Registration</h4>
              <p className="text-sm text-gray-500">Let users create accounts without admin approval</p>
            </div>
            <label className="relative inline-flex items-center cursor-pointer">
              <input
                type="checkbox"
                checked={settings.allow_self_registration || false}
                onChange={(e) => onChange({ allow_self_registration: e.target.checked })}
                className="sr-only peer"
              />
              <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
            </label>
          </div>

          <div className="flex items-center justify-between">
            <div>
              <h4 className="text-sm font-medium text-gray-900">Email Verification Required</h4>
              <p className="text-sm text-gray-500">Require email verification for new users</p>
            </div>
            <label className="relative inline-flex items-center cursor-pointer">
              <input
                type="checkbox"
                checked={settings.email_verification_required || false}
                onChange={(e) => onChange({ email_verification_required: e.target.checked })}
                className="sr-only peer"
              />
              <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
            </label>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Maximum Users
            </label>
            <input
              type="number"
              min="1"
              max="10000"
              value={settings.max_users || 10}
              onChange={(e) => onChange({ max_users: parseInt(e.target.value) })}
              className="w-32 px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Invitation Expiry (days)
            </label>
            <input
              type="number"
              min="1"
              max="30"
              value={settings.invitation_expiry_days || 7}
              onChange={(e) => onChange({ invitation_expiry_days: parseInt(e.target.value) })}
              className="w-32 px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>
        </div>
      </div>
    </div>
  );
};

// Storage Settings Component
const StorageSettings = ({ settings, onChange }) => {
  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-medium text-gray-900 mb-4">Storage Settings</h3>
        
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Maximum Storage (GB)
            </label>
            <input
              type="number"
              min="1"
              max="10000"
              value={settings.max_storage_gb || 5}
              onChange={(e) => onChange({ max_storage_gb: parseInt(e.target.value) })}
              className="w-32 px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Maximum File Size (MB)
            </label>
            <input
              type="number"
              min="1"
              max="1000"
              value={settings.max_file_size || 100}
              onChange={(e) => onChange({ max_file_size: parseInt(e.target.value) })}
              className="w-32 px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>

          <div className="flex items-center justify-between">
            <div>
              <h4 className="text-sm font-medium text-gray-900">Encryption Enabled</h4>
              <p className="text-sm text-gray-500">Encrypt stored artifacts</p>
            </div>
            <label className="relative inline-flex items-center cursor-pointer">
              <input
                type="checkbox"
                checked={settings.encryption_enabled || false}
                onChange={(e) => onChange({ encryption_enabled: e.target.checked })}
                className="sr-only peer"
              />
              <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
            </label>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Retention Policy (days)
            </label>
            <input
              type="number"
              min="1"
              max="3650"
              value={settings.retention_policy_days || 365}
              onChange={(e) => onChange({ retention_policy_days: parseInt(e.target.value) })}
              className="w-32 px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>
        </div>
      </div>
    </div>
  );
};

// Notification Settings Component
const NotificationSettings = ({ settings, onChange }) => {
  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-medium text-gray-900 mb-4">Notification Settings</h3>
        
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <h4 className="text-sm font-medium text-gray-900">Email Notifications</h4>
              <p className="text-sm text-gray-500">Send notifications via email</p>
            </div>
            <label className="relative inline-flex items-center cursor-pointer">
              <input
                type="checkbox"
                checked={settings.email_notifications || false}
                onChange={(e) => onChange({ email_notifications: e.target.checked })}
                className="sr-only peer"
              />
              <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
            </label>
          </div>

          <div className="flex items-center justify-between">
            <div>
              <h4 className="text-sm font-medium text-gray-900">Security Alerts</h4>
              <p className="text-sm text-gray-500">Notify about security events</p>
            </div>
            <label className="relative inline-flex items-center cursor-pointer">
              <input
                type="checkbox"
                checked={settings.security_alerts || false}
                onChange={(e) => onChange({ security_alerts: e.target.checked })}
                className="sr-only peer"
              />
              <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
            </label>
          </div>

          <div className="flex items-center justify-between">
            <div>
              <h4 className="text-sm font-medium text-gray-900">System Alerts</h4>
              <p className="text-sm text-gray-500">Notify about system events</p>
            </div>
            <label className="relative inline-flex items-center cursor-pointer">
              <input
                type="checkbox"
                checked={settings.system_alerts || false}
                onChange={(e) => onChange({ system_alerts: e.target.checked })}
                className="sr-only peer"
              />
              <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
            </label>
          </div>
        </div>
      </div>
    </div>
  );
};

// Integration Settings Component
const IntegrationSettings = ({ settings, onChange }) => {
  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-medium text-gray-900 mb-4">Integration Settings</h3>
        
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              API Rate Limit (hourly)
            </label>
            <input
              type="number"
              min="100"
              max="100000"
              value={settings.api_rate_limits?.hourly || 1000}
              onChange={(e) => onChange({ 
                api_rate_limits: { 
                  ...settings.api_rate_limits, 
                  hourly: parseInt(e.target.value) 
                } 
              })}
              className="w-32 px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Webhook Endpoints
            </label>
            <textarea
              rows="3"
              placeholder="Enter webhook URLs, one per line"
              value={(settings.webhook_endpoints || []).join('\n')}
              onChange={(e) => onChange({ 
                webhook_endpoints: e.target.value.split('\n').filter(url => url.trim()) 
              })}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>
        </div>
      </div>
    </div>
  );
};

// Compliance Settings Component
const ComplianceSettings = ({ settings, onChange }) => {
  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-medium text-gray-900 mb-4">Compliance Settings</h3>
        
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <h4 className="text-sm font-medium text-gray-900">Audit Logs Enabled</h4>
              <p className="text-sm text-gray-500">Keep detailed audit logs</p>
            </div>
            <label className="relative inline-flex items-center cursor-pointer">
              <input
                type="checkbox"
                checked={settings.audit_logs_enabled || false}
                onChange={(e) => onChange({ audit_logs_enabled: e.target.checked })}
                className="sr-only peer"
              />
              <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
            </label>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Audit Retention (days)
            </label>
            <input
              type="number"
              min="30"
              max="2555"
              value={settings.audit_retention_days || 90}
              onChange={(e) => onChange({ audit_retention_days: parseInt(e.target.value) })}
              className="w-32 px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Compliance Mode
            </label>
            <select
              value={settings.compliance_mode || 'basic'}
              onChange={(e) => onChange({ compliance_mode: e.target.value })}
              className="w-48 px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            >
              <option value="basic">Basic</option>
              <option value="standard">Standard</option>
              <option value="strict">Strict</option>
              <option value="custom">Custom</option>
            </select>
          </div>

          <div className="flex items-center justify-between">
            <div>
              <h4 className="text-sm font-medium text-gray-900">GDPR Compliance</h4>
              <p className="text-sm text-gray-500">Enable GDPR compliance features</p>
            </div>
            <label className="relative inline-flex items-center cursor-pointer">
              <input
                type="checkbox"
                checked={settings.gdpr_compliance || false}
                onChange={(e) => onChange({ gdpr_compliance: e.target.checked })}
                className="sr-only peer"
              />
              <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
            </label>
          </div>
        </div>
      </div>
    </div>
  );
};

export default TenantSettings;