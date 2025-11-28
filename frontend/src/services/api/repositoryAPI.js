import { getAPIBaseURL } from "../../constants/apiConfig";

class RepositoryAPI {
  constructor() {
    // Repository endpoints are versioned (/api/v1/repositories/*, etc.)
    const baseOrigin = process.env.REACT_APP_API_URL || getAPIBaseURL();
    this.baseURL = baseOrigin + '/api/v1';
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
    
    // Add tenant context headers
    const hostname = window.location.hostname;
    const subdomain = hostname.split('.')[0];
    if (subdomain && subdomain !== 'localhost' && subdomain !== hostname) {
      headers['X-Tenant-Slug'] = subdomain;
    }
    
    // Add tenant ID if available in localStorage
    const tenantId = localStorage.getItem('tenant_id');
    if (tenantId) {
      headers['X-Tenant-ID'] = tenantId;
    }
    
    return headers;
  }

  // Create repository
  async createRepository(repositoryData) {
    // Transform the data to match the backend API expectations
    const payload = {
      name: repositoryData.name,
      type: repositoryData.type,
      repository_type: repositoryData.repositoryType,
      description: repositoryData.description,
      public_access: repositoryData.publicAccess,
      enable_indexing: repositoryData.enableIndexing,
      
      // Remote repository settings
      remote_url: repositoryData.remoteUrl || '',
      username: repositoryData.username || '',
      password: repositoryData.password || '',
      
      // Encryption settings
      enable_encryption: repositoryData.enableEncryption || false,
      encryption_key: repositoryData.encryptionKey || '',
      
      // Replication settings
      enable_replication: repositoryData.enableReplication || false,
      replication_buckets: (repositoryData.enableReplication && repositoryData.replicationBuckets?.length > 0) 
        ? repositoryData.replicationBuckets 
        : null,
      sync_frequency: repositoryData.syncFrequency || 'daily',
      
      // Cloud storage settings
      cloud_provider: repositoryData.cloudProvider || '',
      region: repositoryData.region || '',
      bucket_name: repositoryData.bucketName || '',
      access_key_id: repositoryData.accessKeyId || '',
      secret_access_key: repositoryData.secretAccessKey || '',
      endpoint: repositoryData.endpoint || '',
      github_token: repositoryData.githubToken || '',
      github_org: repositoryData.githubOrg || '',
      github_repo: repositoryData.githubRepo || '',
      
      // Storage limits
      max_storage_gb: 100,
      retention_days: 30,
    };

    // Determine if this is a cloud repository (S3, GCS, Azure)
    // Remote repositories (npmjs.org, Maven Central, etc.) use the regular endpoint
    const isCloudStorage = repositoryData.repositoryType === 'cloud' && 
                           repositoryData.cloudProvider && 
                           ['s3', 's3-compatible', 'gcs', 'azure', 'aws-ecr', 'azure-acr', 'gcp-gcr'].includes(repositoryData.cloudProvider);
    
    // Use the cloud endpoint only for cloud storage providers
    const endpoint = isCloudStorage 
      ? `${this.baseURL}/repositories/cloud`
      : `${this.baseURL}/repositories`;

    const response = await fetch(endpoint, {
      method: 'POST',
      headers: this.getHeaders(),
      body: JSON.stringify(payload)
    });
    
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to create repository');
    }
    
    return response.json();
  }

  // List repositories with stats
  async listRepositories() {
    // Add timestamp to prevent caching
    const timestamp = new Date().getTime();
    const response = await fetch(`${this.baseURL}/repositories/stats?_t=${timestamp}`, {
      headers: this.getHeaders()
    });
    
    if (!response.ok) {
      throw new Error('Failed to fetch repositories');
    }
    
    return response.json();
  }

  // Get repository with stats
  async getRepository(id) {
    const response = await fetch(`${this.baseURL}/repositories/${id}/stats`);
    
    if (!response.ok) {
      throw new Error('Repository not found');
    }
    
    return response.json();
  }

  // Test repository connection
  async testConnection(repositoryData) {
    const response = await fetch(`${this.baseURL}/repositories/test-connection`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(repositoryData)
    });
    
    if (!response.ok) {
      throw new Error('Connection test failed');
    }
    
    return response.json();
  }

  // Delete repository
  async deleteRepository(id) {
    const response = await fetch(`${this.baseURL}/repositories/${id}`, {
      method: 'DELETE'
    });
    
    if (!response.ok) {
      throw new Error('Failed to delete repository');
    }
    
    return response.json();
  }

  // Get repository stats
  async getStats() {
    const response = await fetch(`${this.baseURL}/repositories/stats`);
    
    if (!response.ok) {
      throw new Error('Failed to fetch stats');
    }
    
    const data = await response.json();
    return data.stats;
  }
}

const repositoryAPI = new RepositoryAPI();
export default repositoryAPI;