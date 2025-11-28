import { useState, useEffect } from 'react';
import { artifactsService } from '../services/artifactsService';

// Hook for fetching and managing artifacts data
export const useArtifacts = ({ limit = 10, offset = 0, autoRefresh = false, refreshInterval = 30000 } = {}) => {
  const [artifacts, setArtifacts] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [totalArtifacts, setTotalArtifacts] = useState(0);
  const [lastUpdated, setLastUpdated] = useState(null);

  const fetchArtifacts = async () => {
    try {
      setLoading(true);
      setError(null);
      
      const response = await artifactsService.getArtifacts(limit, offset);
      const formattedArtifacts = (response.artifacts || []).map(artifact => 
        artifactsService.formatArtifactForDisplay(artifact)
      );
      
      setArtifacts(formattedArtifacts);
      setTotalArtifacts(response.total || 0);
      setLastUpdated(new Date());
    } catch (err) {
      console.error('Error fetching artifacts:', err);
      setError(err.message || 'Failed to fetch artifacts');
    } finally {
      setLoading(false);
    }
  };

  // Initial fetch
  useEffect(() => {
    fetchArtifacts();
  }, [limit, offset]);

  // Auto-refresh setup
  useEffect(() => {
    if (!autoRefresh || refreshInterval <= 0) return;

    const interval = setInterval(fetchArtifacts, refreshInterval);
    return () => clearInterval(interval);
  }, [autoRefresh, refreshInterval, limit, offset]);

  const refetch = () => {
    fetchArtifacts();
  };

  return {
    artifacts,
    loading,
    error,
    totalArtifacts,
    lastUpdated,
    refetch
  };
};

// Hook specifically for recent artifacts (last 5)
export const useRecentArtifacts = ({ autoRefresh = true, refreshInterval = 30000 } = {}) => {
  const [artifacts, setArtifacts] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [lastUpdated, setLastUpdated] = useState(null);

  const fetchRecentArtifacts = async () => {
    try {
      setLoading(true);
      setError(null);
      
      const response = await artifactsService.getRecentArtifacts();
      const formattedArtifacts = response.map(artifact => 
        artifactsService.formatArtifactForDisplay(artifact)
      );
      
      setArtifacts(formattedArtifacts);
      setLastUpdated(new Date());
    } catch (err) {
      console.error('Error fetching recent artifacts:', err);
      setError(err.message || 'Failed to fetch recent artifacts');
    } finally {
      setLoading(false);
    }
  };

  // Initial fetch
  useEffect(() => {
    fetchRecentArtifacts();
  }, []);

  // Auto-refresh setup
  useEffect(() => {
    if (!autoRefresh || refreshInterval <= 0) return;

    const interval = setInterval(fetchRecentArtifacts, refreshInterval);
    return () => clearInterval(interval);
  }, [autoRefresh, refreshInterval]);

  const refetch = () => {
    fetchRecentArtifacts();
  };

  return {
    artifacts,
    loading,
    error,
    lastUpdated,
    refetch
  };
};

// Hook for artifact search
export const useArtifactSearch = () => {
  const [results, setResults] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  const searchArtifacts = async (query) => {
    try {
      setLoading(true);
      setError(null);
      
      const response = await artifactsService.searchArtifacts(query);
      const formattedResults = (response.artifacts || []).map(artifact => 
        artifactsService.formatArtifactForDisplay(artifact)
      );
      
      setResults(formattedResults);
    } catch (err) {
      console.error('Error searching artifacts:', err);
      setError(err.message || 'Failed to search artifacts');
      setResults([]);
    } finally {
      setLoading(false);
    }
  };

  const clearResults = () => {
    setResults([]);
    setError(null);
  };

  return {
    results,
    loading,
    error,
    searchArtifacts,
    clearResults
  };
};