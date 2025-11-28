import React from 'react';

const BillingSettings = ({ 
  settings, 
  updateSettings, 
  onSave, 
  onReset, 
  saving, 
  validationErrors 
}) => {
  const billingSettings = settings?.billing || {};

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-medium text-gray-900 mb-4">Billing Settings</h3>
        <p className="text-sm text-gray-500 mb-6">
          Configure billing and subscription settings for your tenant.
        </p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* Plan Type */}
        <div className="space-y-2">
          <label className="block text-sm font-medium text-gray-700">
            Plan Type
          </label>
          <select
            value={billingSettings.plan_type || 'basic'}
            onChange={(e) => updateSettings('billing', {
              ...billingSettings,
              plan_type: e.target.value
            })}
            className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
          >
            <option value="basic">Basic</option>
            <option value="professional">Professional</option>
            <option value="enterprise">Enterprise</option>
            <option value="custom">Custom</option>
          </select>
        </div>

        {/* Billing Cycle */}
        <div className="space-y-2">
          <label className="block text-sm font-medium text-gray-700">
            Billing Cycle
          </label>
          <select
            value={billingSettings.billing_cycle || 'monthly'}
            onChange={(e) => updateSettings('billing', {
              ...billingSettings,
              billing_cycle: e.target.value
            })}
            className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
          >
            <option value="monthly">Monthly</option>
            <option value="annually">Annually</option>
          </select>
        </div>

        {/* Billing Contact */}
        <div className="space-y-2">
          <label className="block text-sm font-medium text-gray-700">
            Billing Contact Email
          </label>
          <input
            type="email"
            value={billingSettings.billing_contact || ''}
            onChange={(e) => updateSettings('billing', {
              ...billingSettings,
              billing_contact: e.target.value
            })}
            className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
            placeholder="billing@company.com"
          />
        </div>

        {/* Auto Renewal */}
        <div className="space-y-2">
          <label className="flex items-center">
            <input
              type="checkbox"
              checked={billingSettings.auto_renewal || false}
              onChange={(e) => updateSettings('billing', {
                ...billingSettings,
                auto_renewal: e.target.checked
              })}
              className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
            />
            <span className="ml-2 text-sm font-medium text-gray-700">
              Auto-renewal enabled
            </span>
          </label>
        </div>

        {/* Overage Charges */}
        <div className="space-y-2">
          <label className="flex items-center">
            <input
              type="checkbox"
              checked={billingSettings.overage_charges_enabled || false}
              onChange={(e) => updateSettings('billing', {
                ...billingSettings,
                overage_charges_enabled: e.target.checked
              })}
              className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
            />
            <span className="ml-2 text-sm font-medium text-gray-700">
              Enable overage charges
            </span>
          </label>
        </div>

        {/* Invoice Delivery */}
        <div className="space-y-2">
          <label className="block text-sm font-medium text-gray-700">
            Invoice Delivery Method
          </label>
          <select
            value={billingSettings.invoice_delivery || 'email'}
            onChange={(e) => updateSettings('billing', {
              ...billingSettings,
              invoice_delivery: e.target.value
            })}
            className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
          >
            <option value="email">Email</option>
            <option value="portal">Portal</option>
            <option value="both">Both</option>
          </select>
        </div>
      </div>

      {/* Usage Limits Section */}
      <div className="border-t pt-6">
        <h4 className="text-md font-medium text-gray-900 mb-4">Usage Limits</h4>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div className="space-y-2">
            <label className="block text-sm font-medium text-gray-700">Max Users</label>
            <input
              type="number"
              value={billingSettings.usage_limits?.max_users || 100}
              onChange={(e) => updateSettings('billing', {
                ...billingSettings,
                usage_limits: {
                  ...billingSettings.usage_limits,
                  max_users: parseInt(e.target.value)
                }
              })}
              className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
            />
          </div>
          <div className="space-y-2">
            <label className="block text-sm font-medium text-gray-700">Max Storage (GB)</label>
            <input
              type="number"
              value={billingSettings.usage_limits?.max_storage_gb || 10}
              onChange={(e) => updateSettings('billing', {
                ...billingSettings,
                usage_limits: {
                  ...billingSettings.usage_limits,
                  max_storage_gb: parseInt(e.target.value)
                }
              })}
              className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
            />
          </div>
          <div className="space-y-2">
            <label className="block text-sm font-medium text-gray-700">Max API Requests</label>
            <input
              type="number"
              value={billingSettings.usage_limits?.max_api_requests || 10000}
              onChange={(e) => updateSettings('billing', {
                ...billingSettings,
                usage_limits: {
                  ...billingSettings.usage_limits,
                  max_api_requests: parseInt(e.target.value)
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

export default BillingSettings;