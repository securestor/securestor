import { useState, useEffect, useCallback, useRef } from 'react';
import statsService from '../services/statsService';

export const useStats = (options = {}) => {
  const {
    autoRefresh = false,
    refreshInterval = 30000, // 30 seconds
    enableRealtime = false
  } = options;

  const [stats, setStats] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [lastUpdated, setLastUpdated] = useState(null);
  
  const intervalRef = useRef(null);
  const unsubscribeRef = useRef(null);

  const fetchStats = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      
      const backendStats = await statsService.getDashboardStats();
      const transformedStats = statsService.transformStatsForDisplay(backendStats);
      
      setStats(transformedStats);
      setLastUpdated(new Date());
    } catch (err) {
      setError(err.message || 'Failed to fetch stats');
      console.error('Error fetching stats:', err);
    } finally {
      setLoading(false);
    }
  }, []);

  const handleRealtimeUpdate = useCallback((newStats) => {
    const transformedStats = statsService.transformStatsForDisplay(newStats);
    setStats(transformedStats);
    setLastUpdated(new Date());
    setError(null);
  }, []);

  const handleRealtimeError = useCallback((err) => {
    console.error('Realtime stats error:', err);
    setError('Real-time connection failed, falling back to periodic updates');
    
    // Fall back to periodic refresh if realtime fails
    if (!intervalRef.current && autoRefresh) {
      intervalRef.current = setInterval(fetchStats, refreshInterval);
    }
  }, [fetchStats, autoRefresh, refreshInterval]);

  // Initial load and setup
  useEffect(() => {
    fetchStats();

    if (enableRealtime) {
      // Set up real-time updates
      unsubscribeRef.current = statsService.subscribeToRealtimeStats(
        handleRealtimeUpdate,
        handleRealtimeError
      );
    } else if (autoRefresh) {
      // Set up periodic refresh
      intervalRef.current = setInterval(fetchStats, refreshInterval);
    }

    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
        intervalRef.current = null;
      }
      if (unsubscribeRef.current) {
        unsubscribeRef.current();
        unsubscribeRef.current = null;
      }
    };
  }, [fetchStats, autoRefresh, refreshInterval, enableRealtime, handleRealtimeUpdate, handleRealtimeError]);

  const refresh = useCallback(() => {
    fetchStats();
  }, [fetchStats]);

  return {
    stats,
    loading,
    error,
    lastUpdated,
    refresh
  };
};

export const useDetailedStats = () => {
  const [detailedStats, setDetailedStats] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  const fetchDetailedStats = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      
      const stats = await statsService.getDetailedStats();
      setDetailedStats(stats);
    } catch (err) {
      setError(err.message || 'Failed to fetch detailed stats');
      console.error('Error fetching detailed stats:', err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchDetailedStats();
  }, [fetchDetailedStats]);

  return {
    detailedStats,
    loading,
    error,
    refresh: fetchDetailedStats
  };
};

export const useMetricStats = (metricType) => {
  const [metricStats, setMetricStats] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  const fetchMetricStats = useCallback(async () => {
    if (!metricType) return;
    
    try {
      setLoading(true);
      setError(null);
      
      const stats = await statsService.getMetricStats(metricType);
      setMetricStats(stats);
    } catch (err) {
      setError(err.message || `Failed to fetch ${metricType} stats`);
      console.error(`Error fetching ${metricType} stats:`, err);
    } finally {
      setLoading(false);
    }
  }, [metricType]);

  useEffect(() => {
    fetchMetricStats();
  }, [fetchMetricStats]);

  return {
    metricStats,
    loading,
    error,
    refresh: fetchMetricStats
  };
};