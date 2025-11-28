import React from 'react';

const ComplianceSettings = ({ 
  settings, 
  updateSettings, 
  onSave, 
  onReset, 
  saving, 
  validationErrors 
}) => {
  const complianceSettings = settings?.compliance || {};

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-medium text-gray-900 mb-4">Compliance Settings</h3>
        <p className="text-sm text-gray-500 mb-6">
          Configure compliance and regulatory requirements for your tenant.
        </p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* Audit Logs */}
        <div className="space-y-2">
          <label className="flex items-center">
            <input
              type="checkbox"
              checked={complianceSettings.audit_logs_enabled || false}
              onChange={(e) => updateSettings('compliance', {
                ...complianceSettings,
                audit_logs_enabled: e.target.checked
              })}
              className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
            />
            <span className="ml-2 text-sm font-medium text-gray-700">
              Enable Audit Logging
            </span>
          </label>
          <p className="text-xs text-gray-500 ml-6">
            Log all user actions and system events for compliance tracking.
          </p>
        </div>

        {/* Compliance Mode */}
        <div className="space-y-2">
          <label className="block text-sm font-medium text-gray-700">
            Compliance Mode
          </label>
          <select
            value={complianceSettings.compliance_mode || 'none'}
            onChange={(e) => updateSettings('compliance', {
              ...complianceSettings,
              compliance_mode: e.target.value
            })}
            className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
          >
            <option value="none">None</option>
            <option value="basic">Basic</option>
            <option value="strict">Strict</option>
          </select>
        </div>

        {/* Data Retention */}
        <div className="space-y-2">
          <label className="block text-sm font-medium text-gray-700">
            Audit Retention (days)
          </label>
          <input
            type="number"
            value={complianceSettings.audit_retention_days || 90}
            onChange={(e) => updateSettings('compliance', {
              ...complianceSettings,
              audit_retention_days: parseInt(e.target.value)
            })}
            className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
            min="1"
            max="2555"
          />
        </div>

        {/* GDPR Compliance */}
        <div className="space-y-2">
          <label className="flex items-center">
            <input
              type="checkbox"
              checked={complianceSettings.gdpr_compliance || false}
              onChange={(e) => updateSettings('compliance', {
                ...complianceSettings,
                gdpr_compliance: e.target.checked
              })}
              className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
            />
            <span className="ml-2 text-sm font-medium text-gray-700">
              GDPR Compliance
            </span>
          </label>
        </div>

        {/* SOC2 Compliance */}
        <div className="space-y-2">
          <label className="flex items-center">
            <input
              type="checkbox"
              checked={complianceSettings.soc2_compliance || false}
              onChange={(e) => updateSettings('compliance', {
                ...complianceSettings,
                soc2_compliance: e.target.checked
              })}
              className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
            />
            <span className="ml-2 text-sm font-medium text-gray-700">
              SOC2 Compliance
            </span>
          </label>
        </div>

        {/* HIPAA Compliance */}
        <div className="space-y-2">
          <label className="flex items-center">
            <input
              type="checkbox"
              checked={complianceSettings.hipaa_compliance || false}
              onChange={(e) => updateSettings('compliance', {
                ...complianceSettings,
                hipaa_compliance: e.target.checked
              })}
              className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
            />
            <span className="ml-2 text-sm font-medium text-gray-700">
              HIPAA Compliance
            </span>
          </label>
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

export default ComplianceSettings;