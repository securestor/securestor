import React, { useState } from 'react';
import { Settings, Shield, Building2 } from 'lucide-react';
import TenantSettingsDashboard from './TenantSettingsDashboard';
import TenantSettingsEnterprise from './TenantSettingsEnterprise';

const TenantSettingsVerification = () => {
  const [currentView, setCurrentView] = useState('dashboard');
  const [selectedTenantId, setSelectedTenantId] = useState(1);

  const views = [
    { 
      id: 'dashboard', 
      label: 'Multi-Tenant Dashboard', 
      icon: Building2,
      description: 'Overview and management of multiple tenants with basic settings'
    },
    { 
      id: 'enterprise', 
      label: 'Enterprise Settings', 
      icon: Shield,
      description: 'Advanced enterprise-grade configuration for a single tenant'
    }
  ];

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header Navigation */}
      <div className="bg-white shadow-sm border-b">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center py-4">
            <div className="flex items-center gap-2">
              <Settings className="h-6 w-6 text-blue-600" />
              <h1 className="text-xl font-semibold text-gray-900">
                Tenant Settings - UI Flow Verification
              </h1>
            </div>
            
            {/* View Selector */}
            <div className="flex gap-2">
              {views.map((view) => {
                const Icon = view.icon;
                return (
                  <button
                    key={view.id}
                    onClick={() => setCurrentView(view.id)}
                    className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                      currentView === view.id
                        ? 'bg-blue-100 text-blue-700 border-2 border-blue-200'
                        : 'bg-gray-100 text-gray-700 hover:bg-gray-200 border-2 border-transparent'
                    }`}
                  >
                    <Icon className="h-4 w-4" />
                    {view.label}
                  </button>
                );
              })}
            </div>
          </div>
          
          {/* Description */}
          <div className="pb-4">
            <p className="text-sm text-gray-600">
              {views.find(v => v.id === currentView)?.description}
            </p>
          </div>
        </div>
      </div>

      {/* Component Display */}
      <div className="max-w-7xl mx-auto">
        {currentView === 'dashboard' && (
          <div>
            <div className="p-4 bg-blue-50 border-l-4 border-blue-400 m-4">
              <div className="flex">
                <div className="ml-3">
                  <p className="text-sm text-blue-700">
                    <strong>Multi-Tenant Dashboard:</strong> This component provides an overview of all tenants 
                    with the ability to switch between tenants and manage basic settings like security, 
                    user management, and storage.
                  </p>
                </div>
              </div>
            </div>
            <TenantSettingsDashboard />
          </div>
        )}

        {currentView === 'enterprise' && (
          <div>
            <div className="p-4 bg-green-50 border-l-4 border-green-400 m-4">
              <div className="flex">
                <div className="ml-3">
                  <p className="text-sm text-green-700">
                    <strong>Enterprise Settings:</strong> Advanced enterprise-grade tenant configuration 
                    with 7 comprehensive categories: General, Security, Compliance, Billing, Features, 
                    Monitoring, and Advanced Security.
                  </p>
                </div>
              </div>
            </div>
            
            <div className="p-4">
              <div className="mb-4">
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Select Tenant ID for Enterprise Configuration:
                </label>
                <select
                  value={selectedTenantId}
                  onChange={(e) => setSelectedTenantId(parseInt(e.target.value))}
                  className="border border-gray-300 rounded-md px-3 py-2 text-sm"
                >
                  <option value={1}>Tenant 1</option>
                  <option value={42}>Tenant 42 (Test Enterprise)</option>
                  <option value={100}>Tenant 100</option>
                </select>
              </div>
              
              <TenantSettingsEnterprise 
                tenantId={selectedTenantId} 
                onClose={() => {}} 
              />
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default TenantSettingsVerification;