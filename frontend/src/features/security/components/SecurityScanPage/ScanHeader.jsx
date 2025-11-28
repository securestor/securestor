import React from 'react';
import { 
  Shield, 
  Plus, 
  RefreshCw,
  BarChart3 
} from 'lucide-react';
import { useTranslation } from '../../../../hooks/useTranslation';

export const ScanHeader = ({ onCreateScan, onRefresh, scanCount }) => {
  const { t } = useTranslation('security');
  
  return (
    <div className="mb-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-3">
          <div className="bg-blue-100 p-2 rounded-lg">
            <Shield className="h-6 w-6 text-blue-600" />
          </div>
          <div>
            <h1 className="text-2xl font-bold text-gray-900">{t('header.title')}</h1>
            <p className="text-gray-600">{t('header.description')}</p>
          </div>
        </div>
        
        <div className="flex items-center space-x-3">
          <div className="bg-white rounded-lg p-3 shadow-sm border">
            <div className="flex items-center space-x-2">
              <BarChart3 className="h-5 w-5 text-gray-500" />
              <span className="text-sm text-gray-600">{t('header.totalScans')}:</span>
              <span className="font-semibold text-gray-900">{scanCount}</span>
            </div>
          </div>
          
          <button
            onClick={onRefresh}
            className="bg-white hover:bg-gray-50 border border-gray-300 px-4 py-2 rounded-lg flex items-center space-x-2 transition-colors"
          >
            <RefreshCw className="h-4 w-4" />
            <span>{t('header.refresh')}</span>
          </button>
          
          <button
            onClick={onCreateScan}
            className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg flex items-center space-x-2 transition-colors"
          >
            <Plus className="h-4 w-4" />
            <span>{t('header.newScan')}</span>
          </button>
        </div>
      </div>
    </div>
  );
};