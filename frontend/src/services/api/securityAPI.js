import { getAPIBaseURL } from "../../constants/apiConfig";

const API_BASE_URL = (() => {
  // Security endpoints are versioned (/api/v1/security/*, etc.)
  const baseOrigin = process.env.REACT_APP_API_URL || getAPIBaseURL();
  return baseOrigin + "/api/v1";
})();

export const SecurityAPI = {
  async getDashboard() {
    const response = await fetch(`${API_BASE_URL}/security/dashboard`);
    if (!response.ok) throw new Error('Failed to fetch dashboard');
    return response.json();
  },

  async getVulnerableArtifacts(severity = '') {
    const url = new URL(`${API_BASE_URL}/security/vulnerable-artifacts`);
    if (severity) url.searchParams.append('severity', severity);
    
    const response = await fetch(url);
    if (!response.ok) throw new Error('Failed to fetch artifacts');
    return response.json();
  },

  async getVulnerabilities(artifactId) {
    const response = await fetch(`${API_BASE_URL}/artifacts/${artifactId}/vulnerabilities`);
    if (!response.ok) throw new Error('Failed to fetch vulnerabilities');
    return response.json();
  },

  async scanArtifact(artifactId) {
    const response = await fetch(`${API_BASE_URL}/artifacts/${artifactId}/scan`, {
      method: 'POST'
    });
    if (!response.ok) throw new Error('Failed to initiate scan');
    return response.json();
  },

  async bulkScan(artifactIds) {
    const response = await fetch(`${API_BASE_URL}/security/bulk-scan`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ artifact_ids: artifactIds })
    });
    if (!response.ok) throw new Error('Failed to initiate bulk scan');
    return response.json();
  },

  async exportReport(format = 'json') {
    const response = await fetch(`${API_BASE_URL}/security/report?format=${format}`);
    if (!response.ok) throw new Error('Failed to export report');
    const blob = await response.blob();
    
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `security-report.${format}`;
    document.body.appendChild(a);
    a.click();
    window.URL.revokeObjectURL(url);
    document.body.removeChild(a);
  },

  async getAvailableScanners() {
    const response = await fetch(`${API_BASE_URL}/security/scanners`);
    if (!response.ok) throw new Error('Failed to fetch scanners');
    return response.json();
  }
};