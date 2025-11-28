import { getAPIBaseURL } from "../../constants/apiConfig";

class RepositoryArtifactsAPI {
  constructor() {
    // Repository and artifacts endpoints are versioned (/api/v1/repositories/*, /api/v1/artifacts/*, etc.)
    const baseOrigin = process.env.REACT_APP_API_URL || getAPIBaseURL();
    this.baseURL = baseOrigin + "/api/v1";
  }

  // Get authentication token
  getToken() {
    return localStorage.getItem('auth_token') || sessionStorage.getItem('auth_token') || '';
  }

  /**
   * Get artifacts for a specific repository
   * @param {number} repositoryId - The repository ID
   * @param {object} options - Query parameters for filtering, sorting, and pagination
   * @returns {Promise} Repository artifacts response
   */
  async getRepositoryArtifacts(repositoryId, options = {}) {
    try {
      const queryParams = new URLSearchParams();

      // Pagination
      if (options.page) queryParams.append('page', options.page);
      if (options.limit) queryParams.append('limit', options.limit);

      // Sorting
      if (options.sortBy) queryParams.append('sortBy', options.sortBy);
      if (options.sortOrder) queryParams.append('sortOrder', options.sortOrder);

      // Filtering
      if (options.search) queryParams.append('search', options.search);
      if (options.type) queryParams.append('type', options.type);
      if (options.dateFrom) queryParams.append('dateFrom', options.dateFrom);
      if (options.dateTo) queryParams.append('dateTo', options.dateTo);
      if (options.minSize) queryParams.append('minSize', options.minSize);
      if (options.maxSize) queryParams.append('maxSize', options.maxSize);
      if (options.complianceStatus) queryParams.append('complianceStatus', options.complianceStatus);
      if (options.tags && options.tags.length > 0) {
        queryParams.append('tags', options.tags.join(','));
      }

      const queryString = queryParams.toString();
      const url = `${this.baseURL}/repositories/${repositoryId}/artifacts${queryString ? `?${queryString}` : ''}`;

      const response = await fetch(url, {
        method: 'GET',
        headers: {
          'Authorization': `Bearer ${this.getToken()}`,
          'Content-Type': 'application/json',
        },
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || `HTTP ${response.status}: ${response.statusText}`);
      }

      return await response.json();
    } catch (error) {
      console.error('Error fetching repository artifacts:', error);
      throw error;
    }
  }

  /**
   * Download an artifact file
   * @param {number} artifactId - The artifact ID
   * @returns {Promise} Download response
   */
  async downloadArtifact(artifactId) {
    try {
      const response = await fetch(`${this.baseURL}/artifacts/${artifactId}/download`, {
        method: 'GET',
        headers: {
          'Authorization': `Bearer ${this.getToken()}`,
        },
      });

      if (!response.ok) {
        throw new Error(`Download failed: ${response.statusText}`);
      }

      return response;
    } catch (error) {
      console.error('Error downloading artifact:', error);
      throw error;
    }
  }

  /**
   * Get artifact scan results
   * @param {number} artifactId - The artifact ID
   * @returns {Promise} Scan results
   */
  async getArtifactScanResults(artifactId) {
    try {
      const response = await fetch(`${this.baseURL}/artifacts/${artifactId}/scan/results`, {
        method: 'GET',
        headers: {
          'Authorization': `Bearer ${this.getToken()}`,
          'Content-Type': 'application/json',
        },
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || `HTTP ${response.status}: ${response.statusText}`);
      }

      return await response.json();
    } catch (error) {
      console.error('Error fetching scan results:', error);
      throw error;
    }
  }

  /**
   * Trigger a new scan for an artifact
   * @param {number} artifactId - The artifact ID
   * @returns {Promise} Scan response
   */
  async scanArtifact(artifactId) {
    try {
      const response = await fetch(`${this.baseURL}/artifacts/${artifactId}/scan`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${this.getToken()}`,
          'Content-Type': 'application/json',
        },
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || `HTTP ${response.status}: ${response.statusText}`);
      }

      return await response.json();
    } catch (error) {
      console.error('Error triggering scan:', error);
      throw error;
    }
  }

  /**
   * Delete an artifact
   * @param {number} artifactId - The artifact ID
   * @returns {Promise} Delete response
   */
  async deleteArtifact(artifactId) {
    try {
      const response = await fetch(`${this.baseURL}/artifacts/${artifactId}`, {
        method: 'DELETE',
        headers: {
          'Authorization': `Bearer ${this.getToken()}`,
          'Content-Type': 'application/json',
        },
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || `HTTP ${response.status}: ${response.statusText}`);
      }

      return response.status === 204 ? {} : await response.json();
    } catch (error) {
      console.error('Error deleting artifact:', error);
      throw error;
    }
  }
}

export default new RepositoryArtifactsAPI();