import React from 'react';

const AdvancedSecuritySettings = ({ 
  settings, 
  updateSettings, 
  onSave, 
  onReset, 
  saving, 
  validationErrors 
}) => {
  const advancedSecuritySettings = settings?.['advanced-security'] || {};

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-medium text-gray-900 mb-4">Advanced Security Settings</h3>
        <p className="text-sm text-gray-500 mb-6">
          Configure advanced security features including zero-trust, threat detection, and network security.
        </p>
      </div>

      {/* Zero Trust Mode */}
      <div className="border-b pb-6">
        <h4 className="text-md font-medium text-gray-900 mb-4">Zero Trust Security</h4>
        <div className="space-y-2">
          <label className="flex items-center">
            <input
              type="checkbox"
              checked={advancedSecuritySettings.zero_trust_mode || false}
              onChange={(e) => updateSettings('advanced-security', {
                ...advancedSecuritySettings,
                zero_trust_mode: e.target.checked
              })}
              className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
            />
            <span className="ml-2 text-sm font-medium text-gray-700">
              Enable Zero Trust Mode
            </span>
          </label>
          <p className="text-xs text-gray-500 ml-6">
            Implement zero-trust security model with continuous verification.
          </p>
        </div>
      </div>

      {/* Threat Detection */}
      <div className="border-b pb-6">
        <h4 className="text-md font-medium text-gray-900 mb-4">Threat Detection</h4>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className="space-y-2">
            <label className="flex items-center">
              <input
                type="checkbox"
                checked={advancedSecuritySettings.threat_detection?.enabled || false}
                onChange={(e) => updateSettings('advanced-security', {
                  ...advancedSecuritySettings,
                  threat_detection: {
                    ...advancedSecuritySettings.threat_detection,
                    enabled: e.target.checked
                  }
                })}
                className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
              />
              <span className="ml-2 text-sm font-medium text-gray-700">
                Enable Threat Detection
              </span>
            </label>
          </div>
          <div className="space-y-2">
            <label className="flex items-center">
              <input
                type="checkbox"
                checked={advancedSecuritySettings.threat_detection?.anomaly_detection || false}
                onChange={(e) => updateSettings('advanced-security', {
                  ...advancedSecuritySettings,
                  threat_detection: {
                    ...advancedSecuritySettings.threat_detection,
                    anomaly_detection: e.target.checked
                  }
                })}
                className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
              />
              <span className="ml-2 text-sm font-medium text-gray-700">
                Anomaly Detection
              </span>
            </label>
          </div>
          <div className="space-y-2">
            <label className="flex items-center">
              <input
                type="checkbox"
                checked={advancedSecuritySettings.threat_detection?.behavioral_analysis || false}
                onChange={(e) => updateSettings('advanced-security', {
                  ...advancedSecuritySettings,
                  threat_detection: {
                    ...advancedSecuritySettings.threat_detection,
                    behavioral_analysis: e.target.checked
                  }
                })}
                className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
              />
              <span className="ml-2 text-sm font-medium text-gray-700">
                Behavioral Analysis
              </span>
            </label>
          </div>
          <div className="space-y-2">
            <label className="flex items-center">
              <input
                type="checkbox"
                checked={advancedSecuritySettings.threat_detection?.auto_quarantine || false}
                onChange={(e) => updateSettings('advanced-security', {
                  ...advancedSecuritySettings,
                  threat_detection: {
                    ...advancedSecuritySettings.threat_detection,
                    auto_quarantine: e.target.checked
                  }
                })}
                className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
              />
              <span className="ml-2 text-sm font-medium text-gray-700">
                Auto Quarantine
              </span>
            </label>
          </div>
        </div>
      </div>

      {/* Data Loss Prevention */}
      <div className="border-b pb-6">
        <h4 className="text-md font-medium text-gray-900 mb-4">Data Loss Prevention (DLP)</h4>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className="space-y-2">
            <label className="flex items-center">
              <input
                type="checkbox"
                checked={advancedSecuritySettings.data_loss_prevention?.enabled || false}
                onChange={(e) => updateSettings('advanced-security', {
                  ...advancedSecuritySettings,
                  data_loss_prevention: {
                    ...advancedSecuritySettings.data_loss_prevention,
                    enabled: e.target.checked
                  }
                })}
                className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
              />
              <span className="ml-2 text-sm font-medium text-gray-700">
                Enable DLP
              </span>
            </label>
          </div>
          <div className="space-y-2">
            <label className="flex items-center">
              <input
                type="checkbox"
                checked={advancedSecuritySettings.data_loss_prevention?.scan_uploads || false}
                onChange={(e) => updateSettings('advanced-security', {
                  ...advancedSecuritySettings,
                  data_loss_prevention: {
                    ...advancedSecuritySettings.data_loss_prevention,
                    scan_uploads: e.target.checked
                  }
                })}
                className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
              />
              <span className="ml-2 text-sm font-medium text-gray-700">
                Scan File Uploads
              </span>
            </label>
          </div>
          <div className="space-y-2">
            <label className="flex items-center">
              <input
                type="checkbox"
                checked={advancedSecuritySettings.data_loss_prevention?.block_sensitive_data || false}
                onChange={(e) => updateSettings('advanced-security', {
                  ...advancedSecuritySettings,
                  data_loss_prevention: {
                    ...advancedSecuritySettings.data_loss_prevention,
                    block_sensitive_data: e.target.checked
                  }
                })}
                className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
              />
              <span className="ml-2 text-sm font-medium text-gray-700">
                Block Sensitive Data
              </span>
            </label>
          </div>
          <div className="space-y-2">
            <label className="flex items-center">
              <input
                type="checkbox"
                checked={advancedSecuritySettings.data_loss_prevention?.notify_on_violation || false}
                onChange={(e) => updateSettings('advanced-security', {
                  ...advancedSecuritySettings,
                  data_loss_prevention: {
                    ...advancedSecuritySettings.data_loss_prevention,
                    notify_on_violation: e.target.checked
                  }
                })}
                className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
              />
              <span className="ml-2 text-sm font-medium text-gray-700">
                Notify on Violations
              </span>
            </label>
          </div>
        </div>
      </div>

      {/* Network Security */}
      <div className="border-b pb-6">
        <h4 className="text-md font-medium text-gray-900 mb-4">Network Security</h4>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className="space-y-2">
            <label className="flex items-center">
              <input
                type="checkbox"
                checked={advancedSecuritySettings.network_security?.firewall_enabled || false}
                onChange={(e) => updateSettings('advanced-security', {
                  ...advancedSecuritySettings,
                  network_security: {
                    ...advancedSecuritySettings.network_security,
                    firewall_enabled: e.target.checked
                  }
                })}
                className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
              />
              <span className="ml-2 text-sm font-medium text-gray-700">
                Enable Firewall
              </span>
            </label>
          </div>
          <div className="space-y-2">
            <label className="flex items-center">
              <input
                type="checkbox"
                checked={advancedSecuritySettings.network_security?.vpn_required || false}
                onChange={(e) => updateSettings('advanced-security', {
                  ...advancedSecuritySettings,
                  network_security: {
                    ...advancedSecuritySettings.network_security,
                    vpn_required: e.target.checked
                  }
                })}
                className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
              />
              <span className="ml-2 text-sm font-medium text-gray-700">
                Require VPN Access
              </span>
            </label>
          </div>
          <div className="space-y-2">
            <label className="block text-sm font-medium text-gray-700">
              TLS Version
            </label>
            <select
              value={advancedSecuritySettings.network_security?.tls_version || '1.3'}
              onChange={(e) => updateSettings('advanced-security', {
                ...advancedSecuritySettings,
                network_security: {
                  ...advancedSecuritySettings.network_security,
                  tls_version: e.target.value
                }
              })}
              className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
            >
              <option value="1.2">TLS 1.2</option>
              <option value="1.3">TLS 1.3</option>
            </select>
          </div>
          <div className="space-y-2">
            <label className="flex items-center">
              <input
                type="checkbox"
                checked={advancedSecuritySettings.network_security?.certificate_pinning || false}
                onChange={(e) => updateSettings('advanced-security', {
                  ...advancedSecuritySettings,
                  network_security: {
                    ...advancedSecuritySettings.network_security,
                    certificate_pinning: e.target.checked
                  }
                })}
                className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
              />
              <span className="ml-2 text-sm font-medium text-gray-700">
                Certificate Pinning
              </span>
            </label>
          </div>
        </div>
      </div>

      {/* Encryption Settings - ENTERPRISE MODE */}
      <div className="border-b pb-6">
        <div className="flex items-center justify-between mb-4">
          <h4 className="text-md font-medium text-gray-900">Enterprise Encryption</h4>
          <span className="inline-flex items-center px-3 py-1 rounded-full text-xs font-semibold bg-green-100 text-green-800">
            <svg className="w-3 h-3 mr-1.5" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
            </svg>
            ACTIVE & ENFORCED
          </span>
        </div>
        
        {/* Enterprise Encryption Notice */}
        <div className="bg-blue-50 border-l-4 border-blue-400 p-4 mb-6">
          <div className="flex">
            <div className="flex-shrink-0">
              <svg className="h-5 w-5 text-blue-400" viewBox="0 0 20 20" fill="currentColor">
                <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clipRule="evenodd" />
              </svg>
            </div>
            <div className="ml-3">
              <p className="text-sm font-medium text-blue-800">
                Enterprise-Grade Encryption Enabled
              </p>
              <p className="text-sm text-blue-700 mt-1">
                All artifacts are automatically encrypted using AES-256-GCM envelope encryption with tenant-specific master keys. 
                Encryption is system-managed and enforced at the platform level to ensure security and compliance.
              </p>
            </div>
          </div>
        </div>

        {/* Encryption Details Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
          {/* Encryption Configuration */}
          <div className="bg-white border border-gray-200 rounded-lg p-4">
            <div className="flex items-center mb-3">
              <svg className="w-5 h-5 text-blue-600 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
              </svg>
              <h5 className="text-sm font-semibold text-gray-900">Encryption Configuration</h5>
            </div>
            <dl className="space-y-2">
              <div className="flex justify-between items-center py-1">
                <dt className="text-xs text-gray-600">Algorithm:</dt>
                <dd className="text-xs font-medium text-gray-900 bg-gray-100 px-2 py-0.5 rounded">AES-256-GCM</dd>
              </div>
              <div className="flex justify-between items-center py-1">
                <dt className="text-xs text-gray-600">Key Management:</dt>
                <dd className="text-xs font-medium text-gray-900">
                  {advancedSecuritySettings.encryption_settings?.key_management_service || 'System Managed'}
                </dd>
              </div>
              <div className="flex justify-between items-center py-1">
                <dt className="text-xs text-gray-600">At Rest:</dt>
                <dd className="flex items-center text-xs font-medium text-green-600">
                  <svg className="w-3 h-3 mr-1" fill="currentColor" viewBox="0 0 20 20">
                    <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                  </svg>
                  Enabled
                </dd>
              </div>
              <div className="flex justify-between items-center py-1">
                <dt className="text-xs text-gray-600">In Transit:</dt>
                <dd className="flex items-center text-xs font-medium text-green-600">
                  <svg className="w-3 h-3 mr-1" fill="currentColor" viewBox="0 0 20 20">
                    <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                  </svg>
                  TLS 1.3
                </dd>
              </div>
              <div className="flex justify-between items-center py-1">
                <dt className="text-xs text-gray-600">Enforcement:</dt>
                <dd className="flex items-center text-xs font-medium text-green-600">
                  <svg className="w-3 h-3 mr-1" fill="currentColor" viewBox="0 0 20 20">
                    <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                  </svg>
                  Mandatory
                </dd>
              </div>
            </dl>
          </div>

          {/* Key Rotation Status */}
          <div className="bg-white border border-gray-200 rounded-lg p-4">
            <div className="flex items-center mb-3">
              <svg className="w-5 h-5 text-green-600 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z" />
              </svg>
              <h5 className="text-sm font-semibold text-gray-900">Key Rotation</h5>
            </div>
            <dl className="space-y-2">
              <div className="flex justify-between items-center py-1">
                <dt className="text-xs text-gray-600">Auto-Rotation:</dt>
                <dd className="flex items-center text-xs font-medium text-green-600">
                  <svg className="w-3 h-3 mr-1" fill="currentColor" viewBox="0 0 20 20">
                    <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                  </svg>
                  Enabled
                </dd>
              </div>
              <div className="flex justify-between items-center py-1">
                <dt className="text-xs text-gray-600">Schedule:</dt>
                <dd className="text-xs font-medium text-gray-900">
                  Every {advancedSecuritySettings.encryption_settings?.key_rotation_days || 90} days
                </dd>
              </div>
              <div className="flex justify-between items-center py-1">
                <dt className="text-xs text-gray-600">Current Key Version:</dt>
                <dd className="text-xs font-medium text-gray-900 bg-gray-100 px-2 py-0.5 rounded">v1</dd>
              </div>
              <div className="flex justify-between items-center py-1">
                <dt className="text-xs text-gray-600">Last Rotation:</dt>
                <dd className="text-xs font-medium text-gray-900">45 days ago</dd>
              </div>
              <div className="flex justify-between items-center py-1">
                <dt className="text-xs text-gray-600">Next Rotation:</dt>
                <dd className="text-xs font-medium text-blue-600">In 45 days</dd>
              </div>
            </dl>
          </div>
        </div>

        {/* Security Features */}
        <div className="bg-gray-50 border border-gray-200 rounded-lg p-4 mb-4">
          <h5 className="text-sm font-semibold text-gray-900 mb-3">Security Features</h5>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div className="flex items-start">
              <svg className="w-4 h-4 text-green-500 mr-2 mt-0.5 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
              </svg>
              <div>
                <p className="text-xs font-medium text-gray-900">Envelope Encryption</p>
                <p className="text-xs text-gray-600">DEK encrypted with tenant TMK</p>
              </div>
            </div>
            <div className="flex items-start">
              <svg className="w-4 h-4 text-green-500 mr-2 mt-0.5 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
              </svg>
              <div>
                <p className="text-xs font-medium text-gray-900">Ephemeral Scanning</p>
                <p className="text-xs text-gray-600">In-memory decryption for scans</p>
              </div>
            </div>
            <div className="flex items-start">
              <svg className="w-4 h-4 text-green-500 mr-2 mt-0.5 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
              </svg>
              <div>
                <p className="text-xs font-medium text-gray-900">Audit Logging</p>
                <p className="text-xs text-gray-600">All encryption ops logged</p>
              </div>
            </div>
            <div className="flex items-start">
              <svg className="w-4 h-4 text-green-500 mr-2 mt-0.5 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
              </svg>
              <div>
                <p className="text-xs font-medium text-gray-900">Memory Scrubbing</p>
                <p className="text-xs text-gray-600">Secure plaintext cleanup</p>
              </div>
            </div>
          </div>
        </div>

        {/* Action Links */}
        <div className="flex items-center justify-between pt-4 border-t border-gray-200">
          <button 
            type="button"
            className="text-sm text-blue-600 hover:text-blue-800 font-medium flex items-center"
            onClick={() => {}}
          >
            <svg className="w-4 h-4 mr-1.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
            </svg>
            View Encryption Dashboard
          </button>
          <button 
            type="button"
            className="text-sm text-gray-600 hover:text-gray-800 font-medium flex items-center"
            onClick={() => {}}
          >
            <svg className="w-4 h-4 mr-1.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
            </svg>
            Download Audit Log
          </button>
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

export default AdvancedSecuritySettings;