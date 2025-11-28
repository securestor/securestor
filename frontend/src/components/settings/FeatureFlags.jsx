import React from 'react';

const FeatureFlags = ({ 
  settings, 
  updateSettings, 
  onSave, 
  onReset, 
  saving, 
  validationErrors 
}) => {
  const featureSettings = settings?.features || {};

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-medium text-gray-900 mb-4">Feature Flags</h3>
        <p className="text-sm text-gray-500 mb-6">
          Enable or disable specific features for your tenant.
        </p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* Advanced Scanning */}
        <div className="space-y-2">
          <label className="flex items-center">
            <input
              type="checkbox"
              checked={featureSettings.enable_advanced_scanning || false}
              onChange={(e) => updateSettings('features', {
                ...featureSettings,
                enable_advanced_scanning: e.target.checked
              })}
              className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
            />
            <span className="ml-2 text-sm font-medium text-gray-700">
              Advanced Vulnerability Scanning
            </span>
          </label>
          <p className="text-xs text-gray-500 ml-6">
            Enable deep vulnerability scanning with enhanced detection capabilities.
          </p>
        </div>

        {/* ML Analysis */}
        <div className="space-y-2">
          <label className="flex items-center">
            <input
              type="checkbox"
              checked={featureSettings.enable_ml_analysis || false}
              onChange={(e) => updateSettings('features', {
                ...featureSettings,
                enable_ml_analysis: e.target.checked
              })}
              className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
            />
            <span className="ml-2 text-sm font-medium text-gray-700">
              Machine Learning Analysis
            </span>
          </label>
          <p className="text-xs text-gray-500 ml-6">
            Use AI/ML algorithms for intelligent threat detection and analysis.
          </p>
        </div>

        {/* Custom Reporting */}
        <div className="space-y-2">
          <label className="flex items-center">
            <input
              type="checkbox"
              checked={featureSettings.enable_custom_reporting || false}
              onChange={(e) => updateSettings('features', {
                ...featureSettings,
                enable_custom_reporting: e.target.checked
              })}
              className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
            />
            <span className="ml-2 text-sm font-medium text-gray-700">
              Custom Reporting
            </span>
          </label>
          <p className="text-xs text-gray-500 ml-6">
            Create and customize security reports with advanced filtering.
          </p>
        </div>

        {/* API v2 */}
        <div className="space-y-2">
          <label className="flex items-center">
            <input
              type="checkbox"
              checked={featureSettings.enable_api_v2 || false}
              onChange={(e) => updateSettings('features', {
                ...featureSettings,
                enable_api_v2: e.target.checked
              })}
              className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
            />
            <span className="ml-2 text-sm font-medium text-gray-700">
              API v2 Access
            </span>
          </label>
          <p className="text-xs text-gray-500 ml-6">
            Access to the next-generation API with enhanced capabilities.
          </p>
        </div>

        {/* Beta Features */}
        <div className="space-y-2">
          <label className="flex items-center">
            <input
              type="checkbox"
              checked={featureSettings.enable_beta_features || false}
              onChange={(e) => updateSettings('features', {
                ...featureSettings,
                enable_beta_features: e.target.checked
              })}
              className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
            />
            <span className="ml-2 text-sm font-medium text-gray-700">
              Beta Features
            </span>
          </label>
          <p className="text-xs text-gray-500 ml-6">
            Enable access to experimental features in development.
          </p>
        </div>

        {/* Webhook Support */}
        <div className="space-y-2">
          <label className="flex items-center">
            <input
              type="checkbox"
              checked={featureSettings.webhook_support || false}
              onChange={(e) => updateSettings('features', {
                ...featureSettings,
                webhook_support: e.target.checked
              })}
              className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
            />
            <span className="ml-2 text-sm font-medium text-gray-700">
              Webhook Integration
            </span>
          </label>
          <p className="text-xs text-gray-500 ml-6">
            Send real-time notifications to external systems via webhooks.
          </p>
        </div>
      </div>

      {/* Feature Limits Section */}
      <div className="border-t pt-6">
        <h4 className="text-md font-medium text-gray-900 mb-4">Feature Limits</h4>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div className="space-y-2">
            <label className="block text-sm font-medium text-gray-700">Max Integrations</label>
            <input
              type="number"
              value={featureSettings.feature_limits?.max_integrations || 5}
              onChange={(e) => updateSettings('features', {
                ...featureSettings,
                feature_limits: {
                  ...featureSettings.feature_limits,
                  max_integrations: parseInt(e.target.value)
                }
              })}
              className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
            />
          </div>
          <div className="space-y-2">
            <label className="block text-sm font-medium text-gray-700">Max Projects</label>
            <input
              type="number"
              value={featureSettings.feature_limits?.max_projects || 10}
              onChange={(e) => updateSettings('features', {
                ...featureSettings,
                feature_limits: {
                  ...featureSettings.feature_limits,
                  max_projects: parseInt(e.target.value)
                }
              })}
              className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
            />
          </div>
          <div className="space-y-2">
            <label className="block text-sm font-medium text-gray-700">Max Scan Jobs/Month</label>
            <input
              type="number"
              value={featureSettings.feature_limits?.max_scan_jobs || 100}
              onChange={(e) => updateSettings('features', {
                ...featureSettings,
                feature_limits: {
                  ...featureSettings.feature_limits,
                  max_scan_jobs: parseInt(e.target.value)
                }
              })}
              className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
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

export default FeatureFlags;