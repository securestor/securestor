// API service for artifacts
import { getAPIBaseURL } from '../constants/apiConfig';

class ArtifactsService {
  constructor() {
    // Artifacts endpoints are versioned (/api/v1/artifacts/*)
    const baseOrigin = process.env.REACT_APP_API_URL || getAPIBaseURL();
    this.baseURL = baseOrigin + "/api/v1";
  }

  // Get JWT token from storage
  getAuthToken() {
    return localStorage.getItem('auth_token') || 
           sessionStorage.getItem('auth_token') || 
           localStorage.getItem('jwt_token') || '';
  }

  // Get common headers for API requests
  getHeaders() {
    const token = this.getAuthToken();
    const headers = {
      'Content-Type': 'application/json'
    };
    
    if (token) {
      headers.Authorization = `Bearer ${token}`;
    }
    
    return headers;
  }

  // Get all artifacts with pagination
  async getArtifacts(limit = 10, offset = 0) {
    try {
      const response = await fetch(`${this.baseURL}/artifacts?limit=${limit}&offset=${offset}`, {
        headers: this.getHeaders()
      });
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      const data = await response.json();
      return data;
    } catch (error) {
      console.error('Error fetching artifacts:', error);
      throw error;
    }
  }

  // Get recent artifacts (last 5)
  async getRecentArtifacts() {
    try {
      const data = await this.getArtifacts(5, 0);
      return data.artifacts || [];
    } catch (error) {
      console.error('Error fetching recent artifacts:', error);
      return [];
    }
  }

  // Get artifact by ID
  async getArtifact(id) {
    try {
      const response = await fetch(`${this.baseURL}/artifacts/${id}`, {
        headers: this.getHeaders()
      });
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      const data = await response.json();
      return data;
    } catch (error) {
      console.error('Error fetching artifact:', error);
      throw error;
    }
  }

  // Search artifacts
  async searchArtifacts(query) {
    try {
      const response = await fetch(`${this.baseURL}/artifacts/search`, {
        method: 'POST',
        headers: this.getHeaders(),
        body: JSON.stringify({ query }),
      });
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      const data = await response.json();
      return data;
    } catch (error) {
      console.error('Error searching artifacts:', error);
      throw error;
    }
  }

  // Format artifact data for display
  formatArtifactForDisplay(artifact) {
    return {
      id: artifact.id,
      name: this.formatArtifactName(artifact),
      repo: artifact.repository,
      size: artifact.size_formatted,
      uploaded: this.formatUploadTime(artifact.uploaded_at),
      user: artifact.uploaded_by,
      type: artifact.type,
      version: artifact.version,
      license: artifact.license,
      tags: artifact.tags || [],
      metadata: artifact.metadata || {}
    };
  }

  // Format artifact name based on type
  formatArtifactName(artifact) {
    switch (artifact.type) {
      case 'docker':
        return `${artifact.name}:${artifact.version}`;
      case 'npm':
        return `${artifact.name}@${artifact.version}`;
      case 'maven':
        return `${artifact.name}-${artifact.version}.jar`;
      case 'pypi':
        return `${artifact.name}-${artifact.version}.whl`;
      case 'helm':
        return `${artifact.name}-${artifact.version}.tgz`;
      default:
        return `${artifact.name}-${artifact.version}`;
    }
  }

  // Format upload time to relative time
  formatUploadTime(uploadedAt) {
    const now = new Date();
    const uploaded = new Date(uploadedAt);
    const diffMs = now - uploaded;
    const diffMins = Math.floor(diffMs / (1000 * 60));
    const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
    const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));

    if (diffMins < 1) return 'just now';
    if (diffMins < 60) return `${diffMins} min ago`;
    if (diffHours < 24) return `${diffHours} hour${diffHours !== 1 ? 's' : ''} ago`;
    return `${diffDays} day${diffDays !== 1 ? 's' : ''} ago`;
  }
}

export const artifactsService = new ArtifactsService();
export default artifactsService;