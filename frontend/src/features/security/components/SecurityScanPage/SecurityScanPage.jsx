import React, { useState, useEffect } from 'react';
import { ScanHeader } from './ScanHeader';
import { ScanFilters } from './ScanFilters';
import { ScanTable } from './ScanTable';
import { ScanModal } from './ScanModal';
import { ScanDetailsModal } from './ScanDetailsModal';
import { useScanData } from '../../hooks/useScanData';
import { useToast } from '../../../../context/ToastContext';
import { useTranslation } from '../../../../hooks/useTranslation';

export const SecurityScanPage = () => {
  const { t } = useTranslation('security');
  const [isCreateScanModalOpen, setIsCreateScanModalOpen] = useState(false);
  const [selectedScan, setSelectedScan] = useState(null);
  const [filters, setFilters] = useState({
    status: 'all',
    type: 'all',
    priority: 'all',
    dateRange: 'all'
  });

  const { 
    scans, 
    loading, 
    error, 
    createScan, 
    refreshScans,
    getScanDetails 
  } = useScanData();
  const { showSuccess, showError } = useToast();

  useEffect(() => {
    refreshScans();
  }, []);

  const handleCreateScan = async (scanData) => {
    try {
      await createScan(scanData);
      setIsCreateScanModalOpen(false);
      showSuccess(t('messages.scanInitiated'));
      refreshScans();
    } catch (error) {
      showError(t('messages.scanFailed') + ': ' + error.message);
    }
  };

  const handleViewScanDetails = async (scan) => {
    try {
      const details = await getScanDetails(scan.id);
      setSelectedScan(details);
    } catch (error) {
      showError(t('messages.detailsFailed') + ': ' + error.message);
    }
  };

  const filteredScans = scans.filter(scan => {
    if (filters.status !== 'all' && scan.status !== filters.status) return false;
    if (filters.type !== 'all' && scan.scan_type !== filters.type) return false;
    if (filters.priority !== 'all' && scan.priority !== filters.priority) return false;
    return true;
  });

  if (error) {
    return (
      <div className="flex-1 p-6">
        <div className="bg-red-50 border border-red-200 rounded-lg p-4">
          <h3 className="text-red-800 font-medium">{t('messages.errorLoading')}</h3>
          <p className="text-red-600 mt-1">{error}</p>
          <button 
            onClick={refreshScans}
            className="mt-3 bg-red-600 text-white px-4 py-2 rounded-md hover:bg-red-700"
          >
            {t('messages.retry')}
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="flex-1 p-6">
      <ScanHeader 
        onCreateScan={() => setIsCreateScanModalOpen(true)}
        onRefresh={refreshScans}
        scanCount={scans.length}
      />
      
      <ScanFilters 
        filters={filters}
        onFiltersChange={setFilters}
      />
      
      <ScanTable 
        scans={filteredScans}
        loading={loading}
        onViewDetails={handleViewScanDetails}
        onRefresh={refreshScans}
      />

      {isCreateScanModalOpen && (
        <ScanModal
          onClose={() => setIsCreateScanModalOpen(false)}
          onSubmit={handleCreateScan}
        />
      )}

      {selectedScan && (
        <ScanDetailsModal
          scan={selectedScan}
          onClose={() => setSelectedScan(null)}
        />
      )}
    </div>
  );
};