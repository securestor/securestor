import { useState, useEffect } from 'react';
import { SecurityAPI } from '../services';

export const useScanners = () => {
  const [scanners, setScanners] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchScanners = async () => {
      try {
        const data = await SecurityAPI.getAvailableScanners();
        setScanners(data.scanners || []);
      } catch (error) {
        console.error('Failed to fetch scanners:', error);
      } finally {
        setLoading(false);
      }
    };

    fetchScanners();
  }, []);

  return { scanners, loading };
};