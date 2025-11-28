import React, { useState, useEffect } from 'react';
import { useTranslation } from '../hooks/useTranslation';
import { 
  Shield, 
  Settings, 
  CreditCard, 
  ToggleLeft, 
  BarChart3, 
  Lock, 
  FileText, 
  Users, 
  Database, 
  Bell,
  CheckCircle,
  AlertTriangle,
  Save,
  RotateCcw,
  Eye,
  EyeOff
} from 'lucide-react';
import { tenantApi } from '../services/tenantApi';

// Import settings components
import GeneralSettings from './settings/GeneralSettings';
import SecuritySettings from './settings/SecuritySettings';
import ComplianceSettings from './settings/ComplianceSettings';
import BillingSettings from './settings/BillingSettings';
import FeatureFlags from './settings/FeatureFlags';
import MonitoringSettings from './settings/MonitoringSettings';
import AdvancedSecuritySettings from './settings/AdvancedSecuritySettings';

const TenantSettingsEnterprise = ({ tenantId, onClose = () => {} }) => {
  const { t } = useTranslation('tenant');
  const [activeTab, setActiveTab] = useState('general');
  const [settings, setSettings] = useState(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState(null);
  const [success, setSuccess] = useState(null);
  const [hasChanges, setHasChanges] = useState(false);
  const [validationErrors, setValidationErrors] = useState({});

  // Tabs configuration
  const tabs = [
    { id: 'general', label: t('tabs.general'), icon: Settings },
    { id: 'security', label: t('tabs.security'), icon: Shield },
    { id: 'compliance', label: t('tabs.compliance'), icon: FileText },
    { id: 'billing', label: t('tabs.billing'), icon: CreditCard },
    { id: 'features', label: t('tabs.features'), icon: ToggleLeft },
    { id: 'monitoring', label: t('tabs.monitoring'), icon: BarChart3 },
    { id: 'advanced-security', label: t('tabs.advancedSecurity'), icon: Lock }
  ];

  useEffect(() => {
    fetchTenantSettings();
  }, [tenantId]);

  const fetchTenantSettings = async () => {
    try {
      setLoading(true);
      const response = await tenantApi.getTenantSettings(tenantId);
      setSettings(response);
    } catch (err) {
      setError(t('messages.loadFailed') + ': ' + err.message);
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async (section = null) => {
    try {
      setSaving(true);
      setValidationErrors({});
      
      if (section) {
        // Save specific section
        const sectionEndpoint = {
          security: 'security',
          compliance: 'compliance',
          billing: 'billing',
          features: 'features',
          monitoring: 'monitoring',
          'advanced-security': 'advanced-security'
        }[section];

        if (sectionEndpoint) {
          await tenantApi.updateTenantSettingsSection(tenantId, sectionEndpoint, {
            [section.replace('-', '_')]: settings[section.replace('-', '_')]
          });
        }
      } else {
        // Save all settings
        await tenantApi.updateTenantSettings(tenantId, settings);
      }
      
      setSuccess(section ? t('messages.sectionSaveSuccess', { section: section.charAt(0).toUpperCase() + section.slice(1) }) : t('messages.saveSuccess'));
      setHasChanges(false);
      setTimeout(() => setSuccess(null), 3000);
    } catch (err) {
      const errorMessage = err.message || 'Failed to save settings';
      setError(errorMessage);
      
      // Extract validation errors if present
      if (err.response?.data?.errors) {
        setValidationErrors(err.response.data.errors);
      }
    } finally {
      setSaving(false);
    }
  };

  const handleReset = async (section) => {
    try {
      setSaving(true);
      await tenantApi.resetTenantSettingsSection(tenantId, section);
      await fetchTenantSettings();
      setSuccess(`${section.charAt(0).toUpperCase() + section.slice(1)} settings reset to defaults`);
      setTimeout(() => setSuccess(null), 3000);
    } catch (err) {
      setError('Failed to reset settings: ' + err.message);
    } finally {
      setSaving(false);
    }
  };

  const handleValidate = async () => {
    try {
      setSaving(true);
      setValidationErrors({});
      await tenantApi.validateTenantSettings(tenantId, settings);
      setSuccess('Settings validation passed!');
      setTimeout(() => setSuccess(null), 3000);
    } catch (err) {
      if (err.response?.data?.errors) {
        setValidationErrors(err.response.data.errors);
        setError('Settings validation failed. Please check the errors below.');
      } else {
        setError('Failed to validate settings: ' + err.message);
      }
    } finally {
      setSaving(false);
    }
  };

  const updateSettings = (section, field, value) => {
    setSettings(prev => ({
      ...prev,
      [section]: {
        ...prev[section],
        [field]: value
      }
    }));
    setHasChanges(true);
  };

  const updateNestedSettings = (section, subsection, field, value) => {
    setSettings(prev => ({
      ...prev,
      [section]: {
        ...prev[section],
        [subsection]: {
          ...prev[section][subsection],
          [field]: value
        }
      }
    }));
    setHasChanges(true);
  };

  if (loading) {
    return (
      <div className="p-6">
        <div className="bg-white rounded-xl shadow-lg p-8 max-w-md mx-auto">
          <div className="flex items-center justify-center">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
            <span className="ml-3 text-gray-700">{t('messages.loading')}</span>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="p-6">
      <div className="bg-white rounded-xl shadow-lg w-full flex flex-col min-h-[calc(100vh-8rem)]">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-gray-200">
          <div>
            <h2 className="text-2xl font-bold text-gray-900">{t('page.title')}</h2>
            <p className="text-gray-600 mt-1">{t('page.description')}</p>
          </div>
          <div className="flex items-center space-x-3">
            {hasChanges && (
              <button
                onClick={handleValidate}
                disabled={saving}
                className="flex items-center px-3 py-2 bg-yellow-500 hover:bg-yellow-600 text-white rounded-lg text-sm font-medium disabled:opacity-50"
              >
                <CheckCircle className="w-4 h-4 mr-2" />
                {t('buttons.validate')}
              </button>
            )}
          </div>
        </div>

        {/* Success/Error Messages */}
        {success && (
          <div className="mx-6 mt-4 bg-green-50 border border-green-200 text-green-700 px-4 py-3 rounded-lg flex items-center">
            <CheckCircle className="w-5 h-5 mr-2" />
            {success}
          </div>
        )}
        {error && (
          <div className="mx-6 mt-4 bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg flex items-center">
            <AlertTriangle className="w-5 h-5 mr-2" />
            {error}
          </div>
        )}

        <div className="flex flex-1 overflow-hidden">
          {/* Sidebar */}
          <div className="w-64 bg-gray-50 border-r border-gray-200 overflow-y-auto">
            <nav className="p-4 space-y-2">
              {tabs.map(tab => {
                const Icon = tab.icon;
                return (
                  <button
                    key={tab.id}
                    onClick={() => setActiveTab(tab.id)}
                    className={`w-full flex items-center px-4 py-3 rounded-lg text-left transition-colors ${
                      activeTab === tab.id
                        ? 'bg-blue-100 text-blue-700 border border-blue-200'
                        : 'text-gray-700 hover:bg-gray-100'
                    }`}
                  >
                    <Icon className="w-5 h-5 mr-3" />
                    {tab.label}
                  </button>
                );
              })}
            </nav>
          </div>

          {/* Content Area */}
          <div className="flex-1 overflow-y-auto">
            <div className="p-6">
              {activeTab === 'general' && (
                <GeneralSettings 
                  settings={settings}
                  updateSettings={updateSettings}
                  onSave={() => handleSave('general')}
                  onReset={() => handleReset('general')}
                  saving={saving}
                  validationErrors={validationErrors}
                />
              )}
              {activeTab === 'security' && (
                <SecuritySettings 
                  settings={settings}
                  updateSettings={updateSettings}
                  updateNestedSettings={updateNestedSettings}
                  onSave={() => handleSave('security')}
                  onReset={() => handleReset('security')}
                  saving={saving}
                  validationErrors={validationErrors}
                />
              )}
              {activeTab === 'compliance' && (
                <ComplianceSettings 
                  settings={settings}
                  updateSettings={updateSettings}
                  onSave={() => handleSave('compliance')}
                  onReset={() => handleReset('compliance')}
                  saving={saving}
                  validationErrors={validationErrors}
                />
              )}
              {activeTab === 'billing' && (
                <BillingSettings 
                  settings={settings}
                  updateSettings={updateSettings}
                  updateNestedSettings={updateNestedSettings}
                  onSave={() => handleSave('billing')}
                  onReset={() => handleReset('billing')}
                  saving={saving}
                  validationErrors={validationErrors}
                />
              )}
              {activeTab === 'features' && (
                <FeatureFlags 
                  settings={settings}
                  updateSettings={updateSettings}
                  onSave={() => handleSave('features')}
                  onReset={() => handleReset('features')}
                  saving={saving}
                  validationErrors={validationErrors}
                />
              )}
              {activeTab === 'monitoring' && (
                <MonitoringSettings 
                  settings={settings}
                  updateSettings={updateSettings}
                  updateNestedSettings={updateNestedSettings}
                  onSave={() => handleSave('monitoring')}
                  onReset={() => handleReset('monitoring')}
                  saving={saving}
                  validationErrors={validationErrors}
                />
              )}
              {activeTab === 'advanced-security' && (
                <AdvancedSecuritySettings 
                  settings={settings}
                  updateSettings={updateSettings}
                  updateNestedSettings={updateNestedSettings}
                  onSave={() => handleSave('advanced-security')}
                  onReset={() => handleReset('advanced-security')}
                  saving={saving}
                  validationErrors={validationErrors}
                />
              )}
            </div>
          </div>
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between p-6 border-t border-gray-200 bg-gray-50">
          <div className="flex items-center space-x-4">
            {hasChanges && (
              <span className="text-amber-600 text-sm font-medium flex items-center">
                <AlertTriangle className="w-4 h-4 mr-2" />
                {t('messages.unsavedChanges')}
              </span>
            )}
          </div>
          <div className="flex items-center space-x-3">
            <button
              onClick={onClose}
              className="px-4 py-2 text-gray-600 hover:text-gray-800 font-medium"
            >
              {t('buttons.cancel')}
            </button>
            <button
              onClick={() => handleSave()}
              disabled={saving || !hasChanges}
              className="flex items-center px-6 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-medium disabled:opacity-50"
            >
              {saving ? (
                <>
                  <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2"></div>
                  {t('buttons.saving')}
                </>
              ) : (
                <>
                  <Save className="w-4 h-4 mr-2" />
                  {t('buttons.saveAll')}
                </>
              )}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};

export default TenantSettingsEnterprise;