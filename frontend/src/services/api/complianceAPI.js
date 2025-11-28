import { getAPIBaseURL } from '../../constants/apiConfig';

class ComplianceAPI {
  constructor() {
    // Compliance endpoints are versioned (/api/v1/compliance/*, /api/v1/artifacts/*, etc.)
    const baseOrigin = process.env.REACT_APP_API_URL || getAPIBaseURL();
    this.baseURL = baseOrigin + "/api/v1";
  }

  // Get authentication token
  getToken() {
    return localStorage.getItem('auth_token') || sessionStorage.getItem('auth_token') || '';
  }
  
  // Get compliance information for an artifact
  async getCompliance(artifactId) {
    try {
      const response = await fetch(`${this.baseURL}/artifacts/${artifactId}/compliance`, {
        method: 'GET',
        headers: {
          'Authorization': `Bearer ${this.getToken()}`,
          'Content-Type': 'application/json'
        }
      });
      
      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('Failed to get compliance:', error);
      throw error;
    }
  }

  // Create compliance audit (with optional scan trigger)
  async createCompliance(artifactId, complianceData, triggerScan = false) {
    try {
      const payload = {
        ...complianceData,
        trigger_scan: triggerScan
      };

      const response = await fetch(`${this.baseURL}/artifacts/${artifactId}/compliance`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${this.getToken()}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(payload),
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('Failed to create compliance:', error);
      throw error;
    }
  }

  // Get vulnerabilities with enhanced support for both legacy and new systems
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
      
      const response = await fetch(url, {
        method: 'GET',
        headers: {
          'Authorization': `Bearer ${this.getToken()}`,
          'Content-Type': 'application/json'
        }
      });
      
      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
      }

      const data = await response.json();
      
      // Normalize response format between legacy and new systems
      if (data.source === 'scan_system' && data.has_details) {
        // New detailed format
        return {
          vulnerabilities: data.vulnerabilities,
          total_count: data.total_count,
          severity_count: data.severity_count,
          source: 'enhanced',
          has_details: true
        };
      } else {
        // Legacy format - convert to compatible structure
        const legacyVuln = data.vulnerabilities || data;
        return {
          vulnerabilities: this.convertLegacyToDetailed(legacyVuln),
          total_count: legacyVuln.total || 0,
          severity_count: {
            critical: legacyVuln.critical || 0,
            high: legacyVuln.high || 0,
            medium: legacyVuln.medium || 0,
            low: legacyVuln.low || 0
          },
          source: 'legacy',
          has_details: false
        };
      }
    } catch (error) {
      console.error('Failed to get vulnerabilities:', error);
      throw error;
    }
  }

  // Get compliance report
  async getComplianceReport(filters = {}) {
    try {
      const params = new URLSearchParams();
      
      Object.keys(filters).forEach(key => {
        if (filters[key]) {
          params.append(key, filters[key]);
        }
      });

      const queryString = params.toString();
      const url = `${this.baseURL}/compliance/report${queryString ? `?${queryString}` : ''}`;
      
      const response = await fetch(url, {
        method: 'GET',
        headers: {
          'Authorization': `Bearer ${this.getToken()}`,
          'Content-Type': 'application/json'
        }
      });
      
      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('Failed to get compliance report:', error);
      throw error;
    }
  }

  // Helper method to convert legacy vulnerability format to detailed format
  convertLegacyToDetailed(legacyVuln) {
    if (!legacyVuln) return [];

    const vulnerabilities = [];
    const severities = ['critical', 'high', 'medium', 'low'];
    
    severities.forEach(severity => {
      const count = legacyVuln[severity] || 0;
      for (let i = 0; i < count; i++) {
        vulnerabilities.push({
          id: `legacy-${severity}-${i}`,
          title: `${severity.charAt(0).toUpperCase() + severity.slice(1)} Vulnerability`,
          description: `Legacy vulnerability scan detected a ${severity} severity issue`,
          severity: severity,
          component: 'Unknown',
          status: 'open',
          cve: null,
          score: this.getSeverityScore(severity),
          references: []
        });
      }
    });

    return vulnerabilities;
  }

  // Helper method to get numeric score for severity
  getSeverityScore(severity) {
    const scores = {
      critical: 9.0,
      high: 7.0,
      medium: 5.0,
      low: 3.0,
      info: 1.0
    };
    return scores[severity] || 0;
  }

  // Helper method to check if enhanced scanning is available
  async isEnhancedScanningAvailable() {
    try {
      const response = await fetch(`${this.baseURL}/scanners/health`, {
        method: 'GET',
        headers: {
          'Authorization': `Bearer ${this.getToken()}`,
          'Content-Type': 'application/json'
        }
      });
      return response.ok;
    } catch (error) {
      return false;
    }
  }

  // Trigger a compliance scan
  async triggerComplianceScan(artifactId) {
    try {
      const scanAPI = await import('./scanAPI');
      return await scanAPI.default.startScan(artifactId, {
        scan_type: 'compliance',
        vulnerability_scan: true,
        malware_scan: true,
        license_scan: true,
        dependency_scan: true,
        priority: 'normal'
      });
    } catch (error) {
      console.error('Failed to trigger compliance scan:', error);
      throw error;
    }
  }

  // =============================================
  // COMPLIANCE MANAGEMENT API METHODS
  // =============================================

  // Get compliance dashboard data
  async getDashboardData(timeRange = '7d') {
    try {
      const response = await fetch(`${this.baseURL}/admin/compliance/dashboard?timeRange=${timeRange}`, {
        method: 'GET',
        headers: {
          'Authorization': `Bearer ${this.getToken()}`,
          'Content-Type': 'application/json',
        },
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('Failed to get dashboard data:', error);
      throw error;
    }
  }

  // Get compliance policies
  async getCompliancePolicies() {
    try {
      const response = await fetch(`${this.baseURL}/compliance/policies`, {
        method: 'GET',
        headers: {
          'Authorization': `Bearer ${this.getToken()}`,
          'Content-Type': 'application/json',
        },
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('Failed to get compliance policies:', error);
      throw error;
    }
  }

  // Create compliance policy
  async createCompliancePolicy(policyData) {
    try {
      const response = await fetch(`${this.baseURL}/compliance/policies`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${this.getToken()}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(policyData),
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('Failed to create compliance policy:', error);
      throw error;
    }
  }

  // Update compliance policy
  async updateCompliancePolicy(policyId, policyData) {
    try {
      const response = await fetch(`${this.baseURL}/compliance/policies/${policyId}`, {
        method: 'PUT',
        headers: {
          'Authorization': `Bearer ${this.getToken()}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(policyData),
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('Failed to update compliance policy:', error);
      throw error;
    }
  }

  // Delete compliance policy
  async deleteCompliancePolicy(policyId) {
    try {
      const response = await fetch(`${this.baseURL}/compliance/policies/${policyId}`, {
        method: 'DELETE',
        headers: {
          'Authorization': `Bearer ${this.getToken()}`,
          'Content-Type': 'application/json',
        },
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('Failed to delete compliance policy:', error);
      throw error;
    }
  }

  // Apply retention policies
  async applyRetentionPolicies(options = {}) {
    try {
      const response = await fetch(`${this.baseURL}/compliance/retention/apply`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${this.getToken()}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(options),
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('Failed to apply retention policies:', error);
      throw error;
    }
  }

  // Get audit logs
  async getComplianceAuditLogs(filters = {}) {
    try {
      const params = new URLSearchParams();
      
      if (filters.startDate) params.append('startDate', filters.startDate);
      if (filters.endDate) params.append('endDate', filters.endDate);
      if (filters.action) params.append('action', filters.action);
      if (filters.limit) params.append('limit', filters.limit.toString());
      if (filters.offset) params.append('offset', filters.offset.toString());

      const queryString = params.toString();
      const url = `${this.baseURL}/compliance/audit-logs${queryString ? `?${queryString}` : ''}`;

      const response = await fetch(url, {
        method: 'GET',
        headers: {
          'Authorization': `Bearer ${this.getToken()}`,
          'Content-Type': 'application/json',
        },
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('Failed to get audit logs:', error);
      throw error;
    }
  }

  // Create data erasure request
  async createDataErasureRequest(requestData) {
    try {
      const response = await fetch(`${this.baseURL}/compliance/erase-request`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${this.getToken()}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(requestData),
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('Failed to create data erasure request:', error);
      throw error;
    }
  }

  // Get policies by artifact type
  async getPoliciesByArtifactType(artifactType) {
    try {
      const response = await fetch(`${this.baseURL}/compliance/policies/${artifactType}`, {
        method: 'GET',
        headers: {
          'Authorization': `Bearer ${this.getToken()}`,
          'Content-Type': 'application/json',
        },
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('Failed to get policies by artifact type:', error);
      throw error;
    }
  }

  // Get compliance statistics
  async getComplianceStats() {
    try {
      const response = await fetch(`${this.baseURL}/compliance/stats`, {
        method: 'GET',
        headers: {
          'Authorization': `Bearer ${this.getToken()}`,
          'Content-Type': 'application/json',
        },
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('Failed to get compliance statistics:', error);
      throw error;
    }
  }

  // Legal Holds Management
  async getLegalHolds() {
    try {
      const response = await fetch(`${this.baseURL}/compliance/legal-holds`, {
        method: 'GET',
        headers: {
          'Authorization': `Bearer ${this.getToken()}`,
          'Content-Type': 'application/json',
        },
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.warn('Legal holds endpoint not available, using defaults:', error.message);
      // Return empty legal holds when endpoint is not available
      return {
        legal_holds: [],
        total: 0
      };
    }
  }

  async createLegalHold(legalHoldData) {
    try {
      const response = await fetch(`${this.baseURL}/compliance/legal-holds`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${this.getToken()}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(legalHoldData),
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('Failed to create legal hold:', error);
      throw error;
    }
  }

  async updateLegalHold(holdId, legalHoldData) {
    try {
      const response = await fetch(`${this.baseURL}/compliance/legal-holds/${holdId}`, {
        method: 'PUT',
        headers: {
          'Authorization': `Bearer ${this.getToken()}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(legalHoldData),
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('Failed to update legal hold:', error);
      throw error;
    }
  }

  async deleteLegalHold(holdId) {
    try {
      const response = await fetch(`${this.baseURL}/compliance/legal-holds/${holdId}`, {
        method: 'DELETE',
        headers: {
          'Authorization': `Bearer ${this.getToken()}`,
          'Content-Type': 'application/json',
        },
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('Failed to delete legal hold:', error);
      throw error;
    }
  }

  // Data Erasure Management
  async getDataErasureRequests(filters = {}) {
    try {
      const params = new URLSearchParams();
      
      if (filters.status) params.append('status', filters.status);
      if (filters.limit) params.append('limit', filters.limit.toString());
      if (filters.offset) params.append('offset', filters.offset.toString());

      const queryString = params.toString();
      const url = `${this.baseURL}/compliance/erasure-requests${queryString ? `?${queryString}` : ''}`;

      const response = await fetch(url, {
        method: 'GET',
        headers: {
          'Authorization': `Bearer ${this.getToken()}`,
          'Content-Type': 'application/json',
        },
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('Failed to get data erasure requests:', error);
      throw error;
    }
  }

  async approveDataErasureRequest(requestId) {
    try {
      const response = await fetch(`${this.baseURL}/compliance/erasure-requests/${requestId}/approve`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${this.getToken()}`,
          'Content-Type': 'application/json',
        },
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('Failed to approve data erasure request:', error);
      throw error;
    }
  }

  async rejectDataErasureRequest(requestId, reason) {
    try {
      const response = await fetch(`${this.baseURL}/compliance/erasure-requests/${requestId}/reject`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${this.getToken()}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ reason }),
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('Failed to reject data erasure request:', error);
      throw error;
    }
  }

  // Get scheduler status
  async getSchedulerStatus() {
    try {
      const response = await fetch(`${this.baseURL}/compliance/scheduler/status`, {
        method: 'GET',
        headers: {
          'Authorization': `Bearer ${this.getToken()}`,
          'Content-Type': 'application/json'
        }
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('Failed to get scheduler status:', error);
      throw error;
    }
  }

  // Trigger scheduler job
  async triggerSchedulerJob(jobType) {
    try {
      const response = await fetch(`${this.baseURL}/compliance/scheduler/trigger`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${this.getToken()}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          job_type: jobType
        })
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error(`Failed to trigger scheduler job ${jobType}:`, error);
      throw error;
    }
  }
}

const complianceAPI = new ComplianceAPI();
export default complianceAPI;