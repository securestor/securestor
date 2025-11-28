import { getAPIBaseURL } from "../../constants/apiConfig";

class ScanAPI {
  constructor() {
    // Scan endpoints are versioned (/api/v1/artifacts/*/scan, etc.)
    const baseOrigin = process.env.REACT_APP_API_URL || getAPIBaseURL();
    this.baseURL = baseOrigin + "/api/v1";
  }
  
  // Start a security scan for an artifact
  async startScan(artifactId, config = {}) {
    const defaultConfig = {
      vulnerability_scan: true,
      malware_scan: true,
      license_scan: true,
      dependency_scan: true,
      priority: "normal"
    };

    const scanConfig = { ...defaultConfig, ...config };

    try {
      const response = await fetch(`${this.baseURL}/artifacts/${artifactId}/scan`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(scanConfig),
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('Failed to start scan:', error);
      throw error;
    }
  }

  // Get scan results for an artifact
  async getScanResults(artifactId) {
    try {
      const response = await fetch(`${this.baseURL}/artifacts/${artifactId}/scan/results`);
      
      if (!response.ok) {
        if (response.status === 404) {
          return null; // No scan results available
        }
        const errorData = await response.json();
        throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('Failed to get scan results:', error);
      throw error;
    }
  }

  // Get scan history for an artifact
  async getScanHistory(artifactId, limit = 10, offset = 0) {
    try {
      const params = new URLSearchParams({
        limit: limit.toString(),
        offset: offset.toString(),
      });

      const response = await fetch(`${this.baseURL}/artifacts/${artifactId}/scan/history?${params}`);
      
      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('Failed to get scan history:', error);
      throw error;
    }
  }

  // Get details of a specific scan
  async getScanDetails(scanId) {
    try {
      const response = await fetch(`${this.baseURL}/scans/${scanId}`);
      
      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('Failed to get scan details:', error);
      throw error;
    }
  }

  // Cancel an active scan
  async cancelScan(scanId) {
    try {
      const response = await fetch(`${this.baseURL}/scans/${scanId}/cancel`, {
        method: 'POST',
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('Failed to cancel scan:', error);
      throw error;
    }
  }

  // Get vulnerabilities for an artifact with optional filtering
  async getVulnerabilities(artifactId, filters = {}) {
    try {
      const params = new URLSearchParams();
      
      if (filters.severity) {
        params.append('severity', filters.severity);
      }
      
      if (filters.status) {
        params.append('status', filters.status);
      }

      const queryString = params.toString();
      const url = `${this.baseURL}/artifacts/${artifactId}/vulnerabilities${queryString ? `?${queryString}` : ''}`;
      
      const response = await fetch(url);
      
      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('Failed to get vulnerabilities:', error);
      throw error;
    }
  }

  // Get available security scanners
  async getAvailableScanners() {
    try {
      const response = await fetch(`${this.baseURL}/scanners`);
      
      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('Failed to get available scanners:', error);
      throw error;
    }
  }

  // Check scanner health status
  async getScannerHealth() {
    try {
      const response = await fetch(`${this.baseURL}/scanners/health`);
      
      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('Failed to get scanner health:', error);
      throw error;
    }
  }

  // Start bulk scan for multiple artifacts
  async startBulkScan(artifactIds, config = {}) {
    const defaultConfig = {
      vulnerability_scan: true,
      malware_scan: true,
      license_scan: true,
      dependency_scan: true,
      priority: "normal"
    };

    const scanConfig = { ...defaultConfig, ...config };

    try {
      const response = await fetch(`${this.baseURL}/scans/bulk`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          artifact_ids: artifactIds,
          config: scanConfig,
        }),
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('Failed to start bulk scan:', error);
      throw error;
    }
  }

  // Poll scan status until completion
  async pollScanStatus(scanId, onProgress = null, maxAttempts = 60, interval = 5000) {
    let attempts = 0;
    
    const poll = async () => {
      try {
        const scan = await this.getScanDetails(scanId);
        
        if (onProgress) {
          onProgress(scan);
        }

        // Check if scan is complete
        if (scan.status === 'completed' || scan.status === 'completed_with_errors') {
          return scan;
        }
        
        if (scan.status === 'failed' || scan.status === 'cancelled') {
          throw new Error(`Scan ${scan.status}: ${scan.error_message || 'Unknown error'}`);
        }

        attempts++;
        if (attempts >= maxAttempts) {
          throw new Error('Scan polling timeout - scan is taking too long');
        }

        // Continue polling
        return new Promise((resolve, reject) => {
          setTimeout(async () => {
            try {
              const result = await poll();
              resolve(result);
            } catch (error) {
              reject(error);
            }
          }, interval);
        });

      } catch (error) {
        throw error;
      }
    };

    return poll();
  }

  // Get scan statistics and metrics
  async getScanStatistics() {
    try {
      const response = await fetch(`${this.baseURL}/scans/statistics`);
      
      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('Failed to get scan statistics:', error);
      throw error;
    }
  }

  // Helper method to format scan duration
  formatScanDuration(durationSeconds) {
    if (!durationSeconds) return 'N/A';
    
    if (durationSeconds < 60) {
      return `${durationSeconds}s`;
    } else if (durationSeconds < 3600) {
      return `${Math.floor(durationSeconds / 60)}m ${durationSeconds % 60}s`;
    } else {
      const hours = Math.floor(durationSeconds / 3600);
      const minutes = Math.floor((durationSeconds % 3600) / 60);
      return `${hours}h ${minutes}m`;
    }
  }

  // Helper method to get risk level color/style
  getRiskLevelStyle(riskLevel) {
    const styles = {
      low: { color: 'green', bg: 'bg-green-100', text: 'text-green-800' },
      medium: { color: 'yellow', bg: 'bg-yellow-100', text: 'text-yellow-800' },
      high: { color: 'orange', bg: 'bg-orange-100', text: 'text-orange-800' },
      critical: { color: 'red', bg: 'bg-red-100', text: 'text-red-800' }
    };
    
    return styles[riskLevel] || styles.low;
  }

  // Helper method to get vulnerability severity style
  getVulnerabilitySeverityStyle(severity) {
    const styles = {
      info: { color: 'blue', bg: 'bg-blue-100', text: 'text-blue-800' },
      low: { color: 'green', bg: 'bg-green-100', text: 'text-green-800' },
      medium: { color: 'yellow', bg: 'bg-yellow-100', text: 'text-yellow-800' },
      high: { color: 'orange', bg: 'bg-orange-100', text: 'text-orange-800' },
      critical: { color: 'red', bg: 'bg-red-100', text: 'text-red-800' }
    };
    
    return styles[severity] || styles.info;
  }
}

const scanAPI = new ScanAPI();
export default scanAPI;