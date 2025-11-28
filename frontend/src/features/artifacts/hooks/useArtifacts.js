import { useState, useCallback } from 'react';
import ArtifactAPI from '../../../services/api/artifactAPI';

export const useArtifacts = () => {
  const [artifacts, setArtifacts] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  const fetchArtifacts = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await ArtifactAPI.listArtifacts();
      setArtifacts(data.artifacts || []);
    } catch (err) {
      setError(err.message || 'Failed to fetch artifacts');
      console.error('Error fetching artifacts:', err);
    } finally {
      setLoading(false);
    }
  }, []);

  return {
    artifacts,
    loading,
    error,
    fetchArtifacts
  };
};