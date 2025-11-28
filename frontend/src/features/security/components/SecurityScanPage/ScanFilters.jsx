import React from 'react';
import { Filter } from 'lucide-react';
import { useTranslation } from '../../../../hooks/useTranslation';

export const ScanFilters = ({ filters, onFiltersChange }) => {
  const { t } = useTranslation('security');
  
  const handleFilterChange = (key, value) => {
    onFiltersChange({
      ...filters,
      [key]: value
    });
  };

  return (
    <div className="mb-6 bg-white rounded-lg shadow-sm border p-4">
      <div className="flex items-center space-x-4">
        <div className="flex items-center space-x-2">
          <Filter className="h-5 w-5 text-gray-500" />
          <span className="text-sm font-medium text-gray-700">{t('filters.title')}:</span>
        </div>

        <div className="flex items-center space-x-2">
          <label className="text-sm text-gray-600">{t('filters.status')}:</label>
          <select
            value={filters.status}
            onChange={(e) => handleFilterChange('status', e.target.value)}
            className="border border-gray-300 rounded-md px-3 py-1 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="all">{t('filters.allStatus')}</option>
            <option value="initiated">{t('statuses.initiated')}</option>
            <option value="running">{t('statuses.running')}</option>
            <option value="completed">{t('statuses.completed')}</option>
            <option value="failed">{t('statuses.failed')}</option>
            <option value="cancelled">{t('statuses.cancelled')}</option>
          </select>
        </div>

        <div className="flex items-center space-x-2">
          <label className="text-sm text-gray-600">{t('filters.type')}:</label>
          <select
            value={filters.type}
            onChange={(e) => handleFilterChange('type', e.target.value)}
            className="border border-gray-300 rounded-md px-3 py-1 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="all">{t('filters.allTypes')}</option>
            <option value="full">{t('scanTypes.full')}</option>
            <option value="quick">{t('scanTypes.quick')}</option>
            <option value="custom">{t('scanTypes.custom')}</option>
            <option value="bulk">{t('scanTypes.bulk')}</option>
          </select>
        </div>

        <div className="flex items-center space-x-2">
          <label className="text-sm text-gray-600">{t('filters.priority')}:</label>
          <select
            value={filters.priority}
            onChange={(e) => handleFilterChange('priority', e.target.value)}
            className="border border-gray-300 rounded-md px-3 py-1 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="all">{t('filters.allPriorities')}</option>
            <option value="low">{t('priorities.low')}</option>
            <option value="normal">{t('priorities.normal')}</option>
            <option value="high">{t('priorities.high')}</option>
            <option value="critical">{t('priorities.critical')}</option>
          </select>
        </div>

        <div className="flex items-center space-x-2">
          <label className="text-sm text-gray-600">{t('filters.date')}:</label>
          <select
            value={filters.dateRange}
            onChange={(e) => handleFilterChange('dateRange', e.target.value)}
            className="border border-gray-300 rounded-md px-3 py-1 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="all">{t('filters.allTime')}</option>
            <option value="today">{t('filters.today')}</option>
            <option value="week">{t('filters.lastWeek')}</option>
            <option value="month">{t('filters.lastMonth')}</option>
          </select>
        </div>

        {(filters.status !== 'all' || filters.type !== 'all' || filters.priority !== 'all' || filters.dateRange !== 'all') && (
          <button
            onClick={() => onFiltersChange({
              status: 'all',
              type: 'all',
              priority: 'all',
              dateRange: 'all'
            })}
            className="text-sm text-blue-600 hover:text-blue-800 underline"
          >
            {t('filters.clearAll')}
          </button>
        )}
      </div>
    </div>
  );
};