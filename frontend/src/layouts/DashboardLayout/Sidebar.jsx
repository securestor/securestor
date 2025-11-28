import React from 'react';
import { useDashboard } from '../../context/DashboardContext';
import { NAV_ITEMS } from '../../constants';
import { useTranslation } from '../../hooks/useTranslation';

// Translation map for navigation labels
const NAV_TRANSLATIONS = {
  'overview': 'navigation.dashboard',
  'repositories': 'navigation.repositories',
  'artifacts': 'navigation.artifacts',
  'security': 'security:title',
  'compliance': 'compliance:title',
  'logs': 'navigation.logs',
  'api-keys': 'auth:apiKeys.title',
  'users': 'navigation.users',
  'roles': 'navigation.roles',
  'tenant-management': 'settings:tenant.title',
  'cache': 'repositories:cache.title',
  'tenant-settings': 'settings:title',
  'replication-settings': 'settings:advanced.replication'
};

export const Sidebar = () => {
  const { t } = useTranslation('common');
  const { activeTab, setActiveTab, setSelectedRepo } = useDashboard();

  const handleTabChange = (tabId) => {
    setActiveTab(tabId);
    // Clear selected repository when switching tabs
    setSelectedRepo(null);
  };

  const getTranslatedLabel = (itemId, defaultLabel) => {
    const translationKey = NAV_TRANSLATIONS[itemId];
    if (translationKey) {
      return t(translationKey);
    }
    return defaultLabel;
  };

  return (
    <aside className="w-64 bg-white border-r border-gray-200 min-h-screen p-4">
      <nav className="space-y-1">
        {NAV_ITEMS.map(item => (
          <button
            key={item.id}
            onClick={() => handleTabChange(item.id)}
            className={`w-full flex items-center space-x-3 px-4 py-2.5 rounded-lg transition ${
              activeTab === item.id
                ? 'bg-blue-50 text-blue-700'
                : 'text-gray-700 hover:bg-gray-50'
            }`}
          >
            <item.icon className="w-5 h-5" />
            <span className="font-medium">{getTranslatedLabel(item.id, item.label)}</span>
          </button>
        ))}
      </nav>
    </aside>
  );
};
