import { useState, useEffect, useCallback } from 'react';
import { SecurityAPI } from '../services';

export const useSecurityDashboard = () => {
  const [dashboard, setDashboard] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  const fetchDashboard = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await SecurityAPI.getDashboard();
      setDashboard(data);
    } catch (err) {
      setError(err.message);
      console.error('Failed to fetch dashboard:', err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchDashboard();
  }, [fetchDashboard]);

  const refresh = useCallback(() => {
    fetchDashboard();
  }, [fetchDashboard]);

  const exportReport = useCallback(async (format = 'json') => {
    try {
      await SecurityAPI.exportReport(format);
    } catch (err) {
      console.error('Failed to export report:', err);
      throw err;
    }
  }, []);

  return {
    dashboard,
    loading,
    error,
    refresh,
    exportReport
  };
};