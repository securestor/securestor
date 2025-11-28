import { useState, useEffect, useCallback } from 'react';
import { SecurityAPI } from '../services';

export const useVulnerableArtifacts = (initialSeverity = '') => {
  const [artifacts, setArtifacts] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [severity, setSeverity] = useState(initialSeverity);
  const [searchTerm, setSearchTerm] = useState('');

  const fetchArtifacts = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await SecurityAPI.getVulnerableArtifacts(severity);
      setArtifacts(data.artifacts || []);
    } catch (err) {
      setError(err.message);
      console.error('Failed to fetch artifacts:', err);
    } finally {
      setLoading(false);
    }
  }, [severity]);

  useEffect(() => {
    fetchArtifacts();
  }, [fetchArtifacts]);

  const filteredArtifacts = artifacts.filter(artifact =>
    artifact.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
    artifact.version.toLowerCase().includes(searchTerm.toLowerCase())
  );

  return {
    artifacts: filteredArtifacts,
    loading,
    error,
    severity,
    setSeverity,
    searchTerm,
    setSearchTerm,
    refresh: fetchArtifacts
  };
};