import React, { useState } from 'react';
import { Save, RotateCcw, Shield, Lock, Eye, EyeOff, AlertTriangle } from 'lucide-react';

const SecuritySettings = ({ settings, updateSettings, updateNestedSettings, onSave, onReset, saving, validationErrors }) => {
  const [showAdvanced, setShowAdvanced] = useState(false);

  if (!settings) return <div>Loading...</div>;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-xl font-semibold text-gray-900">Security Settings</h3>
          <p className="text-gray-600 mt-1">Configure authentication, authorization, and security policies</p>
        </div>
        <div className="flex items-center space-x-3">
          <button
            onClick={() => setShowAdvanced(!showAdvanced)}
            className="flex items-center px-4 py-2 text-blue-600 hover:text-blue-800 border border-blue-200 rounded-lg"
          >
            {showAdvanced ? <EyeOff className="w-4 h-4 mr-2" /> : <Eye className="w-4 h-4 mr-2" />}
            {showAdvanced ? 'Hide Advanced' : 'Show Advanced'}
          </button>
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
        {/* Authentication Settings */}
        <div className="bg-white border border-gray-200 rounded-lg p-6">
          <div className="flex items-center mb-4">
            <Shield className="w-5 h-5 text-blue-600 mr-2" />
            <h4 className="text-lg font-medium text-gray-900">Authentication</h4>
          </div>
          
          <div className="space-y-4">
            <div className="flex items-center">
              <input
                type="checkbox"
                id="mfaRequired"
                checked={settings.security?.mfa_required || false}
                onChange={(e) => updateSettings('security', 'mfa_required', e.target.checked)}
                className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
              />
              <label htmlFor="mfaRequired" className="ml-2 text-sm text-gray-700">
                Require Multi-Factor Authentication
              </label>
            </div>

            <div className="flex items-center">
              <input
                type="checkbox"
                id="requireSSO"
                checked={settings.security?.require_sso || false}
                onChange={(e) => updateSettings('security', 'require_sso', e.target.checked)}
                className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
              />
              <label htmlFor="requireSSO" className="ml-2 text-sm text-gray-700">
                Require Single Sign-On (SSO)
              </label>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Session Timeout (minutes)
              </label>
              <input
                type="number"
                value={settings.security?.session_timeout_minutes || 60}
                onChange={(e) => updateSettings('security', 'session_timeout_minutes', parseInt(e.target.value))}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                min="5"
                max="480"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Max Login Attempts
              </label>
              <input
                type="number"
                value={settings.security?.max_login_attempts || 5}
                onChange={(e) => updateSettings('security', 'max_login_attempts', parseInt(e.target.value))}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                min="1"
                max="10"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Lockout Duration (minutes)
              </label>
              <input
                type="number"
                value={settings.security?.lockout_duration_minutes || 15}
                onChange={(e) => updateSettings('security', 'lockout_duration_minutes', parseInt(e.target.value))}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                min="1"
                max="1440"
              />
            </div>
          </div>
        </div>

        {/* Password Policy */}
        <div className="bg-white border border-gray-200 rounded-lg p-6">
          <div className="flex items-center mb-4">
            <Lock className="w-5 h-5 text-red-600 mr-2" />
            <h4 className="text-lg font-medium text-gray-900">Password Policy</h4>
          </div>
          
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Minimum Length
              </label>
              <input
                type="number"
                value={settings.security?.password_policy?.min_length || 8}
                onChange={(e) => updateNestedSettings('security', 'password_policy', 'min_length', parseInt(e.target.value))}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                min="4"
                max="32"
              />
            </div>

            <div className="space-y-2">
              <div className="flex items-center">
                <input
                  type="checkbox"
                  id="requireUppercase"
                  checked={settings.security?.password_policy?.require_uppercase || false}
                  onChange={(e) => updateNestedSettings('security', 'password_policy', 'require_uppercase', e.target.checked)}
                  className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                />
                <label htmlFor="requireUppercase" className="ml-2 text-sm text-gray-700">
                  Require uppercase letters
                </label>
              </div>

              <div className="flex items-center">
                <input
                  type="checkbox"
                  id="requireLowercase"
                  checked={settings.security?.password_policy?.require_lowercase || false}
                  onChange={(e) => updateNestedSettings('security', 'password_policy', 'require_lowercase', e.target.checked)}
                  className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                />
                <label htmlFor="requireLowercase" className="ml-2 text-sm text-gray-700">
                  Require lowercase letters
                </label>
              </div>

              <div className="flex items-center">
                <input
                  type="checkbox"
                  id="requireNumbers"
                  checked={settings.security?.password_policy?.require_numbers || false}
                  onChange={(e) => updateNestedSettings('security', 'password_policy', 'require_numbers', e.target.checked)}
                  className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                />
                <label htmlFor="requireNumbers" className="ml-2 text-sm text-gray-700">
                  Require numbers
                </label>
              </div>

              <div className="flex items-center">
                <input
                  type="checkbox"
                  id="requireSymbols"
                  checked={settings.security?.password_policy?.require_symbols || false}
                  onChange={(e) => updateNestedSettings('security', 'password_policy', 'require_symbols', e.target.checked)}
                  className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                />
                <label htmlFor="requireSymbols" className="ml-2 text-sm text-gray-700">
                  Require special symbols
                </label>
              </div>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Password Max Age (days)
              </label>
              <input
                type="number"
                value={settings.security?.password_policy?.max_age || ''}
                onChange={(e) => updateNestedSettings('security', 'password_policy', 'max_age', e.target.value ? parseInt(e.target.value) : null)}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                placeholder="Never expires"
                min="30"
                max="365"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Prevent Password Reuse (last N passwords)
              </label>
              <input
                type="number"
                value={settings.security?.password_policy?.prevent_reuse || 5}
                onChange={(e) => updateNestedSettings('security', 'password_policy', 'prevent_reuse', parseInt(e.target.value))}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                min="0"
                max="24"
              />
            </div>
          </div>
        </div>

        {/* API Security */}
        <div className="bg-white border border-gray-200 rounded-lg p-6">
          <div className="flex items-center mb-4">
            <Shield className="w-5 h-5 text-green-600 mr-2" />
            <h4 className="text-lg font-medium text-gray-900">API Security</h4>
          </div>
          
          <div className="space-y-4">
            <div className="flex items-center">
              <input
                type="checkbox"
                id="allowAPIKeys"
                checked={settings.security?.allow_api_keys || false}
                onChange={(e) => updateSettings('security', 'allow_api_keys', e.target.checked)}
                className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
              />
              <label htmlFor="allowAPIKeys" className="ml-2 text-sm text-gray-700">
                Allow API Key authentication
              </label>
            </div>

            {settings.security?.allow_api_keys && (
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  API Key Expiration (days)
                </label>
                <input
                  type="number"
                  value={settings.security?.api_key_expiration_days || ''}
                  onChange={(e) => updateSettings('security', 'api_key_expiration_days', e.target.value ? parseInt(e.target.value) : null)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                  placeholder="Never expires"
                  min="1"
                  max="365"
                />
              </div>
            )}

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Allowed IP Ranges
              </label>
              <textarea
                value={(settings.security?.allowed_ip_ranges || []).join('\n')}
                onChange={(e) => updateSettings('security', 'allowed_ip_ranges', 
                  e.target.value.split('\n').filter(ip => ip.trim()))}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                rows="3"
                placeholder="Enter IP addresses or CIDR blocks, one per line&#10;e.g., 192.168.1.0/24"
              />
              <p className="text-xs text-gray-500 mt-1">
                Leave empty to allow all IPs
              </p>
            </div>
          </div>
        </div>

        {/* MFA Methods */}
        <div className="bg-white border border-gray-200 rounded-lg p-6">
          <div className="flex items-center mb-4">
            <Lock className="w-5 h-5 text-purple-600 mr-2" />
            <h4 className="text-lg font-medium text-gray-900">Multi-Factor Authentication</h4>
          </div>
          
          <div className="space-y-4">
            <div className="space-y-2">
              <label className="block text-sm font-medium text-gray-700">Allowed MFA Methods</label>
              
              <div className="flex items-center">
                <input
                  type="checkbox"
                  id="mfaTOTP"
                  checked={(settings.security?.mfa_methods || []).includes('totp')}
                  onChange={(e) => {
                    const methods = settings.security?.mfa_methods || [];
                    const updatedMethods = e.target.checked
                      ? [...methods.filter(m => m !== 'totp'), 'totp']
                      : methods.filter(m => m !== 'totp');
                    updateSettings('security', 'mfa_methods', updatedMethods);
                  }}
                  className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                />
                <label htmlFor="mfaTOTP" className="ml-2 text-sm text-gray-700">
                  Time-based One-Time Password (TOTP)
                </label>
              </div>

              <div className="flex items-center">
                <input
                  type="checkbox"
                  id="mfaSMS"
                  checked={(settings.security?.mfa_methods || []).includes('sms')}
                  onChange={(e) => {
                    const methods = settings.security?.mfa_methods || [];
                    const updatedMethods = e.target.checked
                      ? [...methods.filter(m => m !== 'sms'), 'sms']
                      : methods.filter(m => m !== 'sms');
                    updateSettings('security', 'mfa_methods', updatedMethods);
                  }}
                  className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                />
                <label htmlFor="mfaSMS" className="ml-2 text-sm text-gray-700">
                  SMS Text Message
                </label>
              </div>

              <div className="flex items-center">
                <input
                  type="checkbox"
                  id="mfaEmail"
                  checked={(settings.security?.mfa_methods || []).includes('email')}
                  onChange={(e) => {
                    const methods = settings.security?.mfa_methods || [];
                    const updatedMethods = e.target.checked
                      ? [...methods.filter(m => m !== 'email'), 'email']
                      : methods.filter(m => m !== 'email');
                    updateSettings('security', 'mfa_methods', updatedMethods);
                  }}
                  className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                />
                <label htmlFor="mfaEmail" className="ml-2 text-sm text-gray-700">
                  Email Verification Code
                </label>
              </div>

              <div className="flex items-center">
                <input
                  type="checkbox"
                  id="mfaHardware"
                  checked={(settings.security?.mfa_methods || []).includes('hardware')}
                  onChange={(e) => {
                    const methods = settings.security?.mfa_methods || [];
                    const updatedMethods = e.target.checked
                      ? [...methods.filter(m => m !== 'hardware'), 'hardware']
                      : methods.filter(m => m !== 'hardware');
                    updateSettings('security', 'mfa_methods', updatedMethods);
                  }}
                  className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                />
                <label htmlFor="mfaHardware" className="ml-2 text-sm text-gray-700">
                  Hardware Security Keys (WebAuthn/FIDO2)
                </label>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Advanced Security Settings */}
      {showAdvanced && (
        <div className="bg-gray-50 border border-gray-200 rounded-lg p-6">
          <div className="flex items-center mb-4">
            <AlertTriangle className="w-5 h-5 text-amber-600 mr-2" />
            <h4 className="text-lg font-medium text-gray-900">Advanced Security Settings</h4>
            <span className="ml-2 text-xs bg-amber-100 text-amber-800 px-2 py-1 rounded">ADVANCED</span>
          </div>
          
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <div className="space-y-4">
              <div className="flex items-center">
                <input
                  type="checkbox"
                  id="zeroTrustMode"
                  checked={settings.advanced_security?.zero_trust_mode || false}
                  onChange={(e) => updateSettings('advanced_security', 'zero_trust_mode', e.target.checked)}
                  className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                />
                <label htmlFor="zeroTrustMode" className="ml-2 text-sm text-gray-700">
                  Enable Zero Trust Mode
                </label>
              </div>

              <div className="flex items-center">
                <input
                  type="checkbox"
                  id="threatDetection"
                  checked={settings.advanced_security?.threat_detection?.enabled || false}
                  onChange={(e) => updateNestedSettings('advanced_security', 'threat_detection', 'enabled', e.target.checked)}
                  className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                />
                <label htmlFor="threatDetection" className="ml-2 text-sm text-gray-700">
                  Enable Threat Detection
                </label>
              </div>

              <div className="flex items-center">
                <input
                  type="checkbox"
                  id="anomalyDetection"
                  checked={settings.advanced_security?.threat_detection?.anomaly_detection || false}
                  onChange={(e) => updateNestedSettings('advanced_security', 'threat_detection', 'anomaly_detection', e.target.checked)}
                  className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                />
                <label htmlFor="anomalyDetection" className="ml-2 text-sm text-gray-700">
                  Anomaly Detection
                </label>
              </div>
            </div>

            <div className="space-y-4">
              <div className="flex items-center">
                <input
                  type="checkbox"
                  id="dlpEnabled"
                  checked={settings.advanced_security?.data_loss_prevention?.enabled || false}
                  onChange={(e) => updateNestedSettings('advanced_security', 'data_loss_prevention', 'enabled', e.target.checked)}
                  className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                />
                <label htmlFor="dlpEnabled" className="ml-2 text-sm text-gray-700">
                  Data Loss Prevention (DLP)
                </label>
              </div>

              <div className="flex items-center">
                <input
                  type="checkbox"
                  id="firewallEnabled"
                  checked={settings.advanced_security?.network_security?.firewall_enabled || false}
                  onChange={(e) => updateNestedSettings('advanced_security', 'network_security', 'firewall_enabled', e.target.checked)}
                  className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                />
                <label htmlFor="firewallEnabled" className="ml-2 text-sm text-gray-700">
                  Application Firewall
                </label>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Minimum TLS Version
                </label>
                <select
                  value={settings.advanced_security?.network_security?.tls_version || '1.2'}
                  onChange={(e) => updateNestedSettings('advanced_security', 'network_security', 'tls_version', e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                >
                  <option value="1.0">TLS 1.0</option>
                  <option value="1.1">TLS 1.1</option>
                  <option value="1.2">TLS 1.2</option>
                  <option value="1.3">TLS 1.3</option>
                </select>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Validation Errors */}
      {validationErrors.security && (
        <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg">
          <ul className="list-disc list-inside space-y-1">
            {validationErrors.security.map((error, index) => (
              <li key={index}>{error}</li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
};

export default SecuritySettings;