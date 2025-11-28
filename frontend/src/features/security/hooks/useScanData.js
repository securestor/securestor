import { useState, useCallback } from 'react';
import { securityScanService } from '../services/securityScanService';

export const useScanData = () => {
  const [scans, setScans] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  const refreshScans = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const scanData = await securityScanService.getAllScans();
      setScans(scanData);
    } catch (err) {
      setError(err.message || 'Failed to load scans');
      console.error('Error loading scans:', err);
    } finally {
      setLoading(false);
    }
  }, []);

  const createScan = useCallback(async (scanData) => {
    setError(null);
    try {
      const newScan = await securityScanService.createScan(scanData);
      setScans(prev => [newScan, ...prev]);
      return newScan;
    } catch (err) {
      setError(err.message || 'Failed to create scan');
      throw err;
    }
  }, []);

  const getScanDetails = useCallback(async (scanId) => {
    setError(null);
    try {
      const scanDetails = await securityScanService.getScanDetails(scanId);
      return scanDetails;
    } catch (err) {
      setError(err.message || 'Failed to load scan details');
      throw err;
    }
  }, []);

  const cancelScan = useCallback(async (scanId) => {
    setError(null);
    try {
      await securityScanService.cancelScan(scanId);
      setScans(prev => prev.map(scan => 
        scan.id === scanId ? { ...scan, status: 'cancelled' } : scan
      ));
    } catch (err) {
      setError(err.message || 'Failed to cancel scan');
      throw err;
    }
  }, []);

  return {
    scans,
    loading,
    error,
    refreshScans,
    createScan,
    getScanDetails,
    cancelScan
  };
};