import React, { useState, useEffect } from 'react';
import {
  Settings,
  Shield,
  Users,
  Database,
  Bell,
  Link as LinkIcon,
  FileText,
  Save,
  RefreshCw,
  Building2,
  Edit,
  X,
  Check,
  AlertTriangle,
  Info,
  Eye,
  EyeOff,
  Plus,
  Trash2,
  Globe,
  Key,
  Zap,
  Activity,
  BarChart3,
  Lock,
  Unlock
} from 'lucide-react';
import { tenantApi } from '../services/tenantApi';
import { useToast } from '../context/ToastContext';
import { useTranslation } from '../hooks/useTranslation';

const ModernTenantSettings = () => {
  const { t } = useTranslation('settings');
  const [currentTenant, setCurrentTenant] = useState(null);
  const [settings, setSettings] = useState(null);
  const [usage, setUsage] = useState(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState(null);
  const [success, setSuccess] = useState('');
  const [activeTab, setActiveTab] = useState('general');
  const [editMode, setEditMode] = useState({});

  const { showSuccess, showError } = useToast();

  // For now, using tenant ID 1. In real app, this would come from context/routing
  const tenantId = 1;

  useEffect(() => {
    fetchTenantData();
  }, []);

  const fetchTenantData = async () => {
    try {
      setLoading(true);
      setError(null);
      
      const [tenantData, settingsData, usageData] = await Promise.all([
        tenantApi.getTenant(tenantId).catch(() => ({ id: tenantId, name: 'Default Tenant' })),
        tenantApi.getTenantSettings(tenantId).catch(() => ({})),
        tenantApi.getTenantUsage(tenantId).catch(() => ({}))
      ]);
      
      setCurrentTenant(tenantData);
      setSettings(settingsData);
      setUsage(usageData);
    } catch (err) {
      setError('Failed to fetch tenant data: ' + err.message);
      console.error('Error fetching tenant data:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleSaveSettings = async (section = null) => {
    try {
      setSaving(true);
      setError(null);
      
      if (section) {
        await tenantApi.patchTenantSettings(tenantId, { [section]: settings[section] });
        showSuccess(t('saveSuccess', { section }));
      } else {
        await tenantApi.updateTenantSettings(tenantId, settings);
        showSuccess(t('saveAllSuccess'));
      }
      
      await fetchTenantData();
    } catch (err) {
      const errorMsg = t('errors:saveError', { message: err.message });
      setError(errorMsg);
      showError(errorMsg);
    } finally {
      setSaving(false);
    }
  };

  const updateSettingSection = (section, updates) => {
    setSettings(prev => ({
      ...prev,
      [section]: {
        ...prev?.[section],
        ...updates
      }
    }));
  };

  const updateTenantInfo = (updates) => {
    setCurrentTenant(prev => ({
      ...prev,
      ...updates
    }));
  };

  const handleSaveTenantInfo = async () => {
    try {
      setSaving(true);
      await tenantApi.updateTenant(tenantId, currentTenant);
      showSuccess(t('tenant.updateSuccess'));
      await fetchTenantData();
    } catch (err) {
      showError(t('tenant.updateError', { message: err.message }));
    } finally {
      setSaving(false);
    }
  };

  const tabs = [
    { id: 'general', label: t('tabs.general'), icon: Settings, color: 'blue' },
    { id: 'security', label: t('tabs.security'), icon: Shield, color: 'red' },
    { id: 'users', label: t('tabs.users'), icon: Users, color: 'green' },
    { id: 'storage', label: t('tabs.storage'), icon: Database, color: 'purple' },
    { id: 'notifications', label: t('tabs.notifications'), icon: Bell, color: 'yellow' },
    { id: 'integrations', label: t('tabs.integrations'), icon: LinkIcon, color: 'indigo' },
    { id: 'compliance', label: t('tabs.compliance'), icon: FileText, color: 'gray' },
    { id: 'billing', label: t('tabs.billing'), icon: BarChart3, color: 'orange' }
  ];

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="flex flex-col items-center space-y-4">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
          <p className="text-gray-600">{t('loadingSettings')}</p>
        </div>
      </div>
    );
  }

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-4">
          <div className="p-3 bg-blue-100 rounded-lg">
            <Building2 className="h-8 w-8 text-blue-600" />
          </div>
          <div>
            <h1 className="text-2xl font-bold text-gray-900">
              {currentTenant?.name || t('title')}
            </h1>
            <p className="text-gray-600">
              {t('subtitle')}
            </p>
          </div>
        </div>
        <div className="flex items-center space-x-2">
          <button
            onClick={fetchTenantData}
            disabled={loading}
            className="flex items-center space-x-2 px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors disabled:opacity-50"
          >
            <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
            <span>{t('common:buttons.refresh')}</span>
          </button>
          <button
            onClick={() => handleSaveSettings()}
            disabled={saving}
            className="flex items-center space-x-2 bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50"
          >
            <Save className="h-4 w-4" />
            <span>{saving ? t('common:status.saving') : t('saveAll')}</span>
          </button>
        </div>
      </div>

      {/* Status Messages */}
      {error && (
        <div className="bg-red-50 border border-red-200 rounded-lg p-4 flex items-start space-x-3">
          <AlertTriangle className="h-5 w-5 text-red-600 flex-shrink-0 mt-0.5" />
          <div className="flex-1">
            <h3 className="text-red-800 font-medium">{t('common:status.error')}</h3>
            <p className="text-red-700 text-sm mt-1">{error}</p>
          </div>
          <button onClick={() => setError(null)} className="text-red-400 hover:text-red-600">
            <X className="h-5 w-5" />
          </button>
        </div>
      )}

      {success && (
        <div className="bg-green-50 border border-green-200 rounded-lg p-4 flex items-start space-x-3">
          <Check className="h-5 w-5 text-green-600 flex-shrink-0 mt-0.5" />
          <div className="flex-1">
            <p className="text-green-700">{success}</p>
          </div>
          <button onClick={() => setSuccess('')} className="text-green-400 hover:text-green-600">
            <X className="h-5 w-5" />
          </button>
        </div>
      )}

      <div className="flex gap-6">
        {/* Sidebar Navigation */}
        <div className="w-64 flex-shrink-0">
          <nav className="space-y-1 bg-white rounded-lg border border-gray-200 p-2">
            {tabs.map((tab) => {
              const Icon = tab.icon;
              const isActive = activeTab === tab.id;
              return (
                <button
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id)}
                  className={`w-full flex items-center space-x-3 px-3 py-2.5 text-left rounded-lg transition-colors ${
                    isActive
                      ? `bg-${tab.color}-50 text-${tab.color}-700 border border-${tab.color}-200`
                      : 'text-gray-600 hover:bg-gray-50 hover:text-gray-900'
                  }`}
                >
                  <Icon className="h-4 w-4" />
                  <span className="font-medium">{tab.label}</span>
                </button>
              );
            })}
          </nav>
        </div>

        {/* Main Content */}
        <div className="flex-1">
          <div className="bg-white rounded-lg border border-gray-200">
            {activeTab === 'general' && (
              <GeneralSettings
                tenant={currentTenant}
                onChange={updateTenantInfo}
                onSave={handleSaveTenantInfo}
                saving={saving}
              />
            )}
            {activeTab === 'security' && (
              <SecuritySettings
                settings={settings?.security_settings || {}}
                onChange={(updates) => updateSettingSection('security_settings', updates)}
                onSave={() => handleSaveSettings('security_settings')}
                saving={saving}
              />
            )}
            {activeTab === 'users' && (
              <UserAccessSettings
                settings={settings?.user_settings || {}}
                onChange={(updates) => updateSettingSection('user_settings', updates)}
                onSave={() => handleSaveSettings('user_settings')}
                saving={saving}
              />
            )}
            {activeTab === 'storage' && (
              <StorageLimitSettings
                settings={settings?.storage_settings || {}}
                tenant={currentTenant}
                usage={usage}
                onChange={(updates) => updateSettingSection('storage_settings', updates)}
                onSave={() => handleSaveSettings('storage_settings')}
                saving={saving}
              />
            )}
            {activeTab === 'notifications' && (
              <NotificationSettings
                settings={settings?.notification_settings || {}}
                onChange={(updates) => updateSettingSection('notification_settings', updates)}
                onSave={() => handleSaveSettings('notification_settings')}
                saving={saving}
              />
            )}
            {activeTab === 'integrations' && (
              <IntegrationSettings
                settings={settings?.integration_settings || {}}
                onChange={(updates) => updateSettingSection('integration_settings', updates)}
                onSave={() => handleSaveSettings('integration_settings')}
                saving={saving}
              />
            )}
            {activeTab === 'compliance' && (
              <ComplianceSettings
                settings={settings?.compliance_settings || {}}
                onChange={(updates) => updateSettingSection('compliance_settings', updates)}
                onSave={() => handleSaveSettings('compliance_settings')}
                saving={saving}
              />
            )}
            {activeTab === 'billing' && (
              <BillingUsageSettings
                tenant={currentTenant}
                usage={usage}
                settings={settings?.billing_settings || {}}
                onChange={(updates) => updateSettingSection('billing_settings', updates)}
                onSave={() => handleSaveSettings('billing_settings')}
                saving={saving}
              />
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

// General Settings Component
const GeneralSettings = ({ tenant, onChange, onSave, saving }) => {
  return (
    <div className="p-6 space-y-6">
      <div className="border-b border-gray-200 pb-4">
        <h2 className="text-xl font-semibold text-gray-900 flex items-center space-x-2">
          <Settings className="h-5 w-5" />
          <span>General Settings</span>
        </h2>
        <p className="text-gray-600 text-sm mt-1">
          Basic tenant information and configuration
        </p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Organization Name
            </label>
            <input
              type="text"
              value={tenant?.name || ''}
              onChange={(e) => onChange({ name: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              placeholder="Enter organization name"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Subdomain
            </label>
            <div className="flex">
              <input
                type="text"
                value={tenant?.subdomain || ''}
                onChange={(e) => onChange({ subdomain: e.target.value })}
                className="flex-1 px-3 py-2 border border-gray-300 rounded-l-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                placeholder="company"
              />
              <span className="px-3 py-2 bg-gray-100 border border-l-0 border-gray-300 rounded-r-lg text-gray-500 text-sm">
                .securestor.com
              </span>
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Description
            </label>
            <textarea
              value={tenant?.description || ''}
              onChange={(e) => onChange({ description: e.target.value })}
              rows={3}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              placeholder="Brief description of the organization"
            />
          </div>
        </div>

        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Plan Type
            </label>
            <select
              value={tenant?.plan || 'basic'}
              onChange={(e) => onChange({ plan: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            >
              <option value="free">Free</option>
              <option value="basic">Basic</option>
              <option value="premium">Premium</option>
              <option value="enterprise">Enterprise</option>
            </select>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Contact Email
            </label>
            <input
              type="email"
              value={tenant?.contact_email || ''}
              onChange={(e) => onChange({ contact_email: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              placeholder="admin@company.com"
            />
          </div>

          <div className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
            <div>
              <h4 className="text-sm font-medium text-gray-900">Tenant Status</h4>
              <p className="text-xs text-gray-500">Enable or disable tenant access</p>
            </div>
            <label className="relative inline-flex items-center cursor-pointer">
              <input
                type="checkbox"
                checked={tenant?.is_active ?? true}
                onChange={(e) => onChange({ is_active: e.target.checked })}
                className="sr-only peer"
              />
              <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
            </label>
          </div>
        </div>
      </div>

      <div className="flex justify-end pt-4 border-t border-gray-200">
        <button
          onClick={onSave}
          disabled={saving}
          className="flex items-center space-x-2 bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50"
        >
          <Save className="h-4 w-4" />
          <span>{saving ? 'Saving...' : 'Save Changes'}</span>
        </button>
      </div>
    </div>
  );
};

// Security Settings Component
const SecuritySettings = ({ settings, onChange, onSave, saving }) => {
  return (
    <div className="p-6 space-y-6">
      <div className="border-b border-gray-200 pb-4">
        <h2 className="text-xl font-semibold text-gray-900 flex items-center space-x-2">
          <Shield className="h-5 w-5" />
          <span>Security Settings</span>
        </h2>
        <p className="text-gray-600 text-sm mt-1">
          Configure authentication and security policies
        </p>
      </div>

      <div className="space-y-6">
        {/* Authentication Settings */}
        <div className="bg-gray-50 p-4 rounded-lg space-y-4">
          <h3 className="font-medium text-gray-900">Authentication</h3>
          
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
              <p className="text-sm text-gray-500">Force users to use SSO authentication</p>
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

          <div className="grid grid-cols-2 gap-4">
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
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Max Login Attempts
              </label>
              <input
                type="number"
                min="3"
                max="10"
                value={settings.max_login_attempts || 5}
                onChange={(e) => onChange({ max_login_attempts: parseInt(e.target.value) })}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>
          </div>
        </div>

        {/* Password Policy */}
        <div className="bg-gray-50 p-4 rounded-lg space-y-4">
          <h3 className="font-medium text-gray-900">Password Policy</h3>
          
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Minimum Length
              </label>
              <input
                type="number"
                min="6"
                max="32"
                value={settings.password_min_length || 8}
                onChange={(e) => onChange({ password_min_length: parseInt(e.target.value) })}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Password Expiry (days)
              </label>
              <input
                type="number"
                min="0"
                max="365"
                value={settings.password_expiry_days || 90}
                onChange={(e) => onChange({ password_expiry_days: parseInt(e.target.value) })}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>
          </div>

          <div className="space-y-3">
            {[
              { key: 'require_uppercase', label: 'Require uppercase letters' },
              { key: 'require_lowercase', label: 'Require lowercase letters' },
              { key: 'require_numbers', label: 'Require numbers' },
              { key: 'require_symbols', label: 'Require special characters' }
            ].map((policy) => (
              <div key={policy.key} className="flex items-center justify-between">
                <span className="text-sm text-gray-700">{policy.label}</span>
                <label className="relative inline-flex items-center cursor-pointer">
                  <input
                    type="checkbox"
                    checked={settings[policy.key] || false}
                    onChange={(e) => onChange({ [policy.key]: e.target.checked })}
                    className="sr-only peer"
                  />
                  <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
                </label>
              </div>
            ))}
          </div>
        </div>
      </div>

      <div className="flex justify-end pt-4 border-t border-gray-200">
        <button
          onClick={onSave}
          disabled={saving}
          className="flex items-center space-x-2 bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50"
        >
          <Shield className="h-4 w-4" />
          <span>{saving ? 'Saving...' : 'Save Security Settings'}</span>
        </button>
      </div>
    </div>
  );
};

// Simplified placeholder components for other tabs
const UserAccessSettings = ({ settings, onChange, onSave, saving }) => (
  <div className="p-6">
    <div className="text-center py-12">
      <Users className="h-12 w-12 text-gray-400 mx-auto mb-4" />
      <h3 className="text-lg font-medium text-gray-900">User & Access Settings</h3>
      <p className="text-gray-500">Configure user management and access controls</p>
    </div>
  </div>
);

const StorageLimitSettings = ({ settings, tenant, usage, onChange, onSave, saving }) => (
  <div className="p-6">
    <div className="text-center py-12">
      <Database className="h-12 w-12 text-gray-400 mx-auto mb-4" />
      <h3 className="text-lg font-medium text-gray-900">Storage & Limits</h3>
      <p className="text-gray-500">Manage storage quotas and resource limits</p>
    </div>
  </div>
);

const NotificationSettings = ({ settings, onChange, onSave, saving }) => (
  <div className="p-6">
    <div className="text-center py-12">
      <Bell className="h-12 w-12 text-gray-400 mx-auto mb-4" />
      <h3 className="text-lg font-medium text-gray-900">Notification Settings</h3>
      <p className="text-gray-500">Configure email and webhook notifications</p>
    </div>
  </div>
);

const IntegrationSettings = ({ settings, onChange, onSave, saving }) => (
  <div className="p-6">
    <div className="text-center py-12">
      <LinkIcon className="h-12 w-12 text-gray-400 mx-auto mb-4" />
      <h3 className="text-lg font-medium text-gray-900">Integration Settings</h3>
      <p className="text-gray-500">Manage third-party integrations and APIs</p>
    </div>
  </div>
);

const ComplianceSettings = ({ settings, onChange, onSave, saving }) => (
  <div className="p-6">
    <div className="text-center py-12">
      <FileText className="h-12 w-12 text-gray-400 mx-auto mb-4" />
      <h3 className="text-lg font-medium text-gray-900">Compliance Settings</h3>
      <p className="text-gray-500">Configure compliance policies and audit settings</p>
    </div>
  </div>
);

const BillingUsageSettings = ({ tenant, usage, settings, onChange, onSave, saving }) => (
  <div className="p-6">
    <div className="text-center py-12">
      <BarChart3 className="h-12 w-12 text-gray-400 mx-auto mb-4" />
      <h3 className="text-lg font-medium text-gray-900">Billing & Usage</h3>
      <p className="text-gray-500">View usage statistics and billing information</p>
    </div>
  </div>
);

export default ModernTenantSettings;