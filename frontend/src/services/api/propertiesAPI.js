import axios from 'axios';
import { getAPIBaseURL, getTenantAwareHeaders } from '../../constants/apiConfig';

const API_BASE_URL = process.env.SECURESTOR_APP_API_URL || getAPIBaseURL() + "/api/v1";

/**
 * Properties API Service
 * Handles all artifact property operations
 */
class PropertiesAPI {
  /**
   * Get common headers with tenant context and auth
   * @returns {Object} Headers object
   */
  getHeaders() {
    const token = localStorage.getItem('auth_token') || 
                  sessionStorage.getItem('auth_token') || 
                  localStorage.getItem('jwt_token') || '';
    
    const headers = {
      'Content-Type': 'application/json',
      ...getTenantAwareHeaders()
    };
    
    if (token) {
      headers.Authorization = `Bearer ${token}`;
    }
    
    return headers;
  }

  /**
   * Get all properties for an artifact
   * @param {string} artifactId - The artifact ID
   * @param {object} params - Query parameters (limit, offset, mask_sensitive)
   * @returns {Promise<Array>} - List of properties
   */
  async getArtifactProperties(artifactId, params = {}) {
    try {
      const response = await axios.get(
        `${API_BASE_URL}/artifacts/${artifactId}/properties`,
        { 
          params,
          headers: this.getHeaders()
        }
      );
      return response.data.properties || [];
    } catch (error) {
      console.error('Error fetching properties:', error);
      throw error;
    }
  }

  /**
   * Get statistics for artifact properties
   * @param {string} tenantId - The tenant ID
   * @returns {Promise<object>} - Property statistics
   */
  async getStatistics(tenantId) {
    try {
      const response = await axios.get(
        `${API_BASE_URL}/properties/statistics`,
        { 
          params: { tenant_id: tenantId },
          headers: this.getHeaders()
        }
      );
      return response.data;
    } catch (error) {
      console.error('Error fetching statistics:', error);
      throw error;
    }
  }

  /**
   * Create a new property for an artifact
   * @param {string} artifactId - The artifact ID
   * @param {object} propertyData - Property data
   * @returns {Promise<object>} - Created property
   */
  async createProperty(artifactId, propertyData) {
    try {
      const response = await axios.post(
        `${API_BASE_URL}/artifacts/${artifactId}/properties`,
        {
          key: propertyData.key,
          value: propertyData.value,
          value_type: propertyData.value_type || 'string',
          is_sensitive: propertyData.is_sensitive || false,
          is_multi_value: propertyData.is_multi_value || false,
          tags: propertyData.tags || [],
          description: propertyData.description || ''
        },
        {
          headers: this.getHeaders()
        }
      );
      return response.data.property;
    } catch (error) {
      console.error('Error creating property:', error);
      throw error;
    }
  }

  /**
   * Update an existing property
   * @param {string} propertyId - The property ID
   * @param {object} updates - Property updates
   * @returns {Promise<object>} - Updated property
   */
  async updateProperty(propertyId, updates) {
    try {
      const response = await axios.put(
        `${API_BASE_URL}/properties/${propertyId}`,
        updates,
        {
          headers: this.getHeaders()
        }
      );
      return response.data.property;
    } catch (error) {
      console.error('Error updating property:', error);
      throw error;
    }
  }

  /**
   * Delete a property
   * @param {string} propertyId - The property ID
   * @returns {Promise<void>}
   */
  async deleteProperty(propertyId) {
    try {
      await axios.delete(
        `${API_BASE_URL}/properties/${propertyId}`,
        {
          headers: this.getHeaders()
        }
      );
    } catch (error) {
      console.error('Error deleting property:', error);
      throw error;
    }
  }

  /**
   * Search properties
   * @param {object} searchParams - Search parameters
   * @returns {Promise<Array>} - Search results
   */
  async searchProperties(searchParams) {
    try {
      const response = await axios.post(
        `${API_BASE_URL}/properties/search`,
        searchParams,
        {
          headers: this.getHeaders()
        }
      );
      return response.data.properties || [];
    } catch (error) {
      console.error('Error searching properties:', error);
      throw error;
    }
  }

  /**
   * Batch create properties
   * @param {string} repositoryId - The repository ID
   * @param {Array} properties - Array of property creation requests
   * @returns {Promise<object>} - Batch operation result
   */
  async batchCreateProperties(repositoryId, properties) {
    try {
      const response = await axios.post(
        `${API_BASE_URL}/properties/batch`,
        {
          repository_id: repositoryId,
          properties: properties
        },
        {
          headers: this.getHeaders()
        }
      );
      return response.data;
    } catch (error) {
      console.error('Error batch creating properties:', error);
      throw error;
    }
  }

  /**
   * Batch delete properties
   * @param {Array} propertyIds - Array of property IDs to delete
   * @returns {Promise<object>} - Batch operation result
   */
  async batchDeleteProperties(propertyIds) {
    try {
      const response = await axios.delete(
        `${API_BASE_URL}/properties/batch/delete`,
        {
          data: { property_ids: propertyIds },
          headers: this.getHeaders()
        }
      );
      return response.data;
    } catch (error) {
      console.error('Error batch deleting properties:', error);
      throw error;
    }
  }
}

export default new PropertiesAPI();
