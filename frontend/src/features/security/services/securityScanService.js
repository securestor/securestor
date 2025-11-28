import { api } from '../../../services/api';
import artifactAPI from '../../../services/api/artifactAPI';

export const securityScanService = {
  // Get all scans with artifact details
  async getAllScans() {
    try {
      const response = await api.get('/scanning');
      const scans = response.scans || [];
      
      // Fetch artifact details for each scan
      const scansWithArtifacts = await Promise.all(scans.map(async (scan) => {
        // Check if this is a cache scan (has metadata with artifact info)
        if (scan.metadata && scan.metadata.artifact_name && scan.metadata.source === 'cache') {
          return {
            ...scan,
            artifact_name: scan.metadata.artifact_name,
            artifact_version: scan.metadata.version || 'N/A',
            artifact_type: scan.metadata.artifact_type,
            artifact_exists: true,
            is_cache_scan: true,
            cache_key: scan.metadata.cache_key
          };
        }
        
        // For regular artifact scans, fetch from artifacts API
        try {
          const artifactDetails = await artifactAPI.getArtifact(scan.artifact_id);
          return {
            ...scan,
            artifact_name: artifactDetails.name,
            artifact_version: artifactDetails.version,
            artifact_exists: true,
            is_cache_scan: false
          };
        } catch (error) {
          // Artifact may have been deleted or doesn't exist
          return {
            ...scan,
            artifact_name: 'Unknown Artifact',
            artifact_version: 'N/A',
            artifact_exists: false,
            artifact_deleted: true,
            is_cache_scan: false
          };
        }
      }));
      
      return scansWithArtifacts;
    } catch (error) {
      console.error('Error fetching scans:', error);
      throw new Error(error.message || 'Failed to fetch scans');
    }
  },

  // Get scan by ID with details
  async getScanDetails(scanId) {
    try {
      const response = await api.get(`/scanning/${scanId}`);
      return response;
    } catch (error) {
      console.error('Error fetching scan details:', error);
      throw new Error(error.message || 'Failed to fetch scan details');
    }
  },

  // Create a new scan
  async createScan(scanData) {
    try {
      const response = await api.post('/scanning', scanData);
      return response;
    } catch (error) {
      console.error('Error creating scan:', error);
      throw new Error(error.message || 'Failed to create scan');
    }
  },

  // Cancel a scan
  async cancelScan(scanId) {
    try {
      const response = await api.post(`/scanning/${scanId}/cancel`);
      return response;
    } catch (error) {
      console.error('Error cancelling scan:', error);
      throw new Error(error.message || 'Failed to cancel scan');
    }
  },

  // Get scan results
  async getScanResults(scanId) {
    try {
      const response = await api.get(`/scanning/${scanId}/results`);
      return response;
    } catch (error) {
      console.error('Error fetching scan results:', error);
      throw new Error(error.message || 'Failed to fetch scan results');
    }
  },

  // Get available scanners
  async getAvailableScanners() {
    try {
      const response = await api.get('/scanning/scanners');
      return response.scanners || [];
    } catch (error) {
      console.error('Error fetching available scanners:', error);
      throw new Error(error.message || 'Failed to fetch available scanners');
    }
  },

  // Get scanner health
  async getScannerHealth() {
    try {
      const response = await api.get('/scanning/scanners/health');
      return response;
    } catch (error) {
      console.error('Error fetching scanner health:', error);
      throw new Error(error.message || 'Failed to fetch scanner health');
    }
  }
};