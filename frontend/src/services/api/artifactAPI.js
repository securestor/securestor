import { getAPIBaseURL } from "../../constants/apiConfig";

class ArtifactAPI {
  constructor() {
    // Artifact endpoints are versioned (/api/v1/artifacts/*, etc.)
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
  getHeaders(includeContentType = true) {
    const token = this.getAuthToken();
    const headers = {};
    
    if (includeContentType) {
      headers['Content-Type'] = 'application/json';
    }
    
    if (token) {
      headers.Authorization = `Bearer ${token}`;
    }
    
    // Add tenant headers (imported from api.js or extracted inline)
    const hostname = window.location.hostname;
    const subdomain = hostname.split('.')[0];
    
    // Check if it's a subdomain (not localhost or IP)
    if (subdomain && subdomain !== 'localhost' && subdomain !== hostname) {
      headers['X-Tenant-Slug'] = subdomain;
    } else if (hostname.includes('localhost')) {
      // For localhost with subdomain format like "alpha.localhost"
      const parts = hostname.split('.');
      if (parts.length > 1 && parts[0] !== 'localhost') {
        headers['X-Tenant-Slug'] = parts[0];
      }
    }
    
    // Try to get tenant ID from localStorage/context
    const tenantId = localStorage.getItem('tenant_id');
    if (tenantId) {
      headers['X-Tenant-ID'] = tenantId;
    }
    
    return headers;
  }

  // Deploy artifact with file upload
  async deployArtifact(artifactData, file, onProgress) {
    const formData = new FormData();

    // Append file
    formData.append("file", file);

    // Append form fields
    formData.append("name", artifactData.artifactName);
    formData.append("version", artifactData.version);
    formData.append("repository_id", artifactData.repository);

    if (artifactData.description) {
      formData.append("description", artifactData.description);
    }

    if (artifactData.license) {
      formData.append("license", artifactData.license);
    }

    if (artifactData.tags && artifactData.tags.length > 0) {
      const tags = artifactData.tags
        .split(",")
        .map((t) => t.trim())
        .filter((t) => t);
      formData.append("tags", JSON.stringify(tags));
    }

    if (artifactData.metadata) {
      formData.append("metadata", JSON.stringify(artifactData.metadata));
    }

    // Add Docker-specific fields
    if (artifactData.artifactType) {
      formData.append("artifact_type", artifactData.artifactType);
    }

    if (artifactData.digest) {
      formData.append("digest", artifactData.digest);
    }

    return new Promise((resolve, reject) => {
      const xhr = new XMLHttpRequest();

      // Track upload progress
      xhr.upload.addEventListener("progress", (e) => {
        if (e.lengthComputable && onProgress) {
          const percentComplete = (e.loaded / e.total) * 100;
          onProgress(percentComplete);
        }
      });

      xhr.addEventListener("load", () => {
        if (xhr.status >= 200 && xhr.status < 300) {
          try {
            resolve(JSON.parse(xhr.responseText));
          } catch (e) {
            reject(new Error("Invalid server response"));
          }
        } else {
          try {
            const error = JSON.parse(xhr.responseText);
            reject(new Error(error.error || `Upload failed with status ${xhr.status}`));
          } catch (e) {
            reject(new Error(`Upload failed with status ${xhr.status}: ${xhr.responseText}`));
          }
        }
      });

      xhr.addEventListener("error", (e) => {
        console.error("XHR Error:", e);
        console.error("XHR Status:", xhr.status);
        console.error("XHR Response:", xhr.responseText);
        reject(new Error(`Network error: Unable to connect to ${this.baseURL}/artifacts/deploy. Make sure the backend server is running.`));
      });

      xhr.addEventListener("abort", () => {
        reject(new Error("Upload cancelled"));
      });

      xhr.addEventListener("timeout", () => {
        reject(new Error("Upload timed out"));
      });

      try {
        xhr.open("POST", `${this.baseURL}/artifacts/upload`);
        xhr.timeout = 300000; // 5 minutes timeout
        
        // Set headers (excluding Content-Type as FormData sets it automatically with boundary)
        const headers = this.getHeaders(false);
        Object.entries(headers).forEach(([key, value]) => {
          xhr.setRequestHeader(key, value);
        });
        
        xhr.send(formData);
      } catch (error) {
        reject(new Error(`Failed to initiate upload: ${error.message}`));
      }
    });
  }

  // Download artifact
  async downloadArtifact(id, filename) {
    const downloadUrl = `${this.baseURL}/artifacts/${id}/download`;
    const headers = this.getHeaders(false);
    
    
    const response = await fetch(downloadUrl, {
      headers: headers
    });
    

    if (!response.ok) {
      const errorText = await response.text();
      console.error('[DOWNLOAD] Error response:', errorText);
      throw new Error(`Download failed: ${response.status} ${response.statusText}`);
    }

    // Try to get filename from Content-Disposition header
    const contentDisposition = response.headers.get('Content-Disposition');
    let downloadFilename = filename || `artifact-${id}`;
    
    
    if (contentDisposition) {
      const filenameMatch = contentDisposition.match(/filename[^;=\n]*=((['"]).*?\2|[^;\n]*)/);
      if (filenameMatch && filenameMatch[1]) {
        downloadFilename = filenameMatch[1].replace(/['"]/g, '');
      }
    }
    

    const blob = await response.blob();

    // Create download link
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = downloadFilename;
    document.body.appendChild(a);
    a.click();
    window.URL.revokeObjectURL(url);
    document.body.removeChild(a);
  }

  // Get artifact details
  async getArtifact(id) {
    const response = await fetch(`${this.baseURL}/artifacts/${id}`, {
      headers: this.getHeaders()
    });

    if (!response.ok) {
      if (response.status === 404) {
        throw new Error(`Artifact ${id} not found`);
      }
      throw new Error(`Failed to fetch artifact: ${response.statusText}`);
    }

    return response.json();
  }

  // List artifacts
  async listArtifacts(filter = {}) {
    const queryParams = new URLSearchParams();
    if (filter.search) queryParams.append("search", filter.search);
    if (filter.limit) queryParams.append("limit", filter.limit);
    if (filter.offset) queryParams.append("offset", filter.offset);
    if (filter.type) queryParams.append("type", filter.type);

    const response = await fetch(`${this.baseURL}/artifacts?${queryParams}`, {
      headers: this.getHeaders()
    });
    if (!response.ok) throw new Error("Failed to fetch artifacts");
    return response.json();
  }

  // Fetch artifacts with filters
  async fetchArtifacts(filter = {}) {
    const queryParams = new URLSearchParams();
    if (filter.search) queryParams.append("search", filter.search);
    if (filter.limit) queryParams.append("limit", filter.limit);
    if (filter.offset) queryParams.append("offset", filter.offset);
    if (filter.type) queryParams.append("type", filter.type);

    const response = await fetch(`${this.baseURL}/artifacts?${queryParams}`, {
      headers: this.getHeaders()
    });
    if (!response.ok) throw new Error("Failed to fetch artifacts");
    return response.json();
  }

  // Advanced search
  async searchArtifacts(filter) {
    const response = await fetch(`${this.baseURL}/artifacts/search`, {
      method: "POST",
      headers: this.getHeaders(),
      body: JSON.stringify(filter),
    });
    if (!response.ok) throw new Error("Search failed");
    return response.json();
  }

  // Create artifact
  async createArtifact(artifact) {
    const response = await fetch(`${this.baseURL}/artifacts`, {
      method: "POST",
      headers: this.getHeaders(),
      body: JSON.stringify(artifact),
    });
    if (!response.ok) throw new Error("Failed to create artifact");
    return response.json();
  }

  // Delete artifact
  async deleteArtifact(id) {
    const response = await fetch(`${this.baseURL}/artifacts/${id}`, {
      method: "DELETE",
      headers: this.getHeaders()
    });
    if (!response.ok) throw new Error("Failed to delete artifact");
    // 204 No Content has no response body
    return response.status === 204 ? {} : await response.json();
  }

  // Scan artifact with comprehensive security scanning
  async scanArtifact(id, scanOptions = {}) {
    const defaultOptions = {
      vulnerability_scan: true,
      malware_scan: true,
      license_scan: true,
      dependency_scan: true,
      priority: 'medium'
    };

    const response = await fetch(`${this.baseURL}/artifacts/${id}/scan`, {
      method: "POST",
      headers: this.getHeaders(),
      body: JSON.stringify({ ...defaultOptions, ...scanOptions }),
    });
    if (!response.ok) {
      const error = await response.text();
      throw new Error(`Scan failed: ${error}`);
    }
    return response.json();
  }

  // Fetch repositories
  async fetchRepositories() {
    const response = await fetch(`${this.baseURL}/repositories`, {
      headers: this.getHeaders()
    });
    if (!response.ok) throw new Error("Failed to fetch repositories");
    return response.json();
  }

  // Create repository
  async createRepository(repository) {
    const response = await fetch(`${this.baseURL}/repositories`, {
      method: "POST",
      headers: this.getHeaders(),
      body: JSON.stringify(repository),
    });
    if (!response.ok) throw new Error("Failed to create repository");
    return response.json();
  }

  // Get compliance
  async getCompliance(artifactId) {
    const response = await fetch(
      `${this.baseURL}/artifacts/${artifactId}/compliance`,
      {
        headers: this.getHeaders()
      }
    );
    if (!response.ok) throw new Error("Failed to fetch compliance");
    return response.json();
  }

  // Create compliance audit
  async createComplianceAudit(artifactId, audit) {
    const response = await fetch(
      `${this.baseURL}/artifacts/${artifactId}/compliance`,
      {
        method: "POST",
        headers: this.getHeaders(),
        body: JSON.stringify(audit),
      }
    );
    if (!response.ok) throw new Error("Failed to create audit");
    return response.json();
  }

  // Update compliance status
  async updateComplianceStatus(artifactId, complianceData) {
    const response = await fetch(
      `${this.baseURL}/artifacts/${artifactId}/compliance`,
      {
        method: "PUT",
        headers: this.getHeaders(),
        body: JSON.stringify(complianceData),
      }
    );
    if (!response.ok) throw new Error("Failed to update compliance status");
    return response.json();
  }

  // Get vulnerabilities
  async getVulnerabilities(artifactId) {
    const response = await fetch(
      `${this.baseURL}/artifacts/${artifactId}/vulnerabilities`,
      {
        headers: this.getHeaders()
      }
    );
    if (!response.ok) throw new Error("Failed to fetch vulnerabilities");
    return response.json();
  }

  // Get dashboard stats
  async getDashboardStats() {
    const response = await fetch(`${this.baseURL}/stats/dashboard`, {
      headers: this.getHeaders()
    });
    if (!response.ok) throw new Error("Failed to fetch stats");
    return response.json();
  }
}

const artifactAPI = new ArtifactAPI();
export default artifactAPI;
