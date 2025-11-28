/**
 * Maven Protocol API Helpers
 * 
 * This file provides helper functions for Maven-specific protocol operations.
 * These are used when working with Maven artifacts within the unified artifact system.
 * 
 * Note: Maven artifacts are NOT displayed in a separate UI.
 * They appear in the main Artifacts tab alongside Docker and NPM artifacts.
 * 
 * Use artifactAPI.js for main artifact operations.
 * Use this only for Maven-specific protocol operations like:
 * - Getting POM files
 * - Maven metadata
 * - Maven coordinate parsing
 */

import { getAPIBaseURL } from "../../constants/apiConfig";

// Maven endpoints are versioned (/api/v1/maven/*, etc.)
const API_BASE_URL = (() => {
  const baseOrigin = process.env.REACT_APP_API_URL || getAPIBaseURL();
  return baseOrigin + "/api/v1";
})();

class MavenAPI {
  // Search for Maven artifacts
  async searchArtifacts(query) {
    const response = await fetch(`${API_BASE_URL}/maven/search?q=${encodeURIComponent(query)}`);
    if (!response.ok) {
      throw new Error('Failed to search artifacts');
    }
    return response.json();
  }

  // Get versions for an artifact
  async getVersions(groupId, artifactId) {
    const response = await fetch(`${API_BASE_URL}/maven/${groupId}/${artifactId}/versions`);
    if (!response.ok) {
      throw new Error('Failed to fetch versions');
    }
    return response.json();
  }

  // Get artifact information
  async getArtifactInfo(groupId, artifactId, version) {
    const response = await fetch(`${API_BASE_URL}/maven/${groupId}/${artifactId}/${version}/info`);
    if (!response.ok) {
      throw new Error('Failed to fetch artifact info');
    }
    return response.json();
  }

  // Deploy artifact
  async deployArtifact(coordinates, file) {
    const { groupId, artifactId, version, packaging } = coordinates;
    
    // Construct the Maven path
    const groupPath = groupId.replace(/\./g, '/');
    const filename = `${artifactId}-${version}.${packaging}`;
    const path = `${groupPath}/${artifactId}/${version}/${filename}`;
    
    const response = await fetch(`${API_BASE_URL}/maven2/${path}`, {
      method: 'PUT',
      body: file,
      headers: {
        'Content-Type': 'application/octet-stream'
      }
    });
    
    if (!response.ok) {
      const error = await response.text();
      throw new Error(error || 'Failed to deploy artifact');
    }
    
    return response.json();
  }

  // Delete artifact version
  async deleteArtifact(groupId, artifactId, version) {
    const response = await fetch(
      `${API_BASE_URL}/maven/${groupId}/${artifactId}/${version}`,
      { method: 'DELETE' }
    );
    
    if (!response.ok) {
      throw new Error('Failed to delete artifact');
    }
    
    // 204 No Content has no response body
    return response.status === 204 ? {} : await response.json();
  }

  // Download artifact
  async downloadArtifact(groupId, artifactId, version, packaging = 'jar', classifier = '') {
    const groupPath = groupId.replace(/\./g, '/');
    let filename = `${artifactId}-${version}`;
    if (classifier) {
      filename += `-${classifier}`;
    }
    filename += `.${packaging}`;
    
    const path = `${groupPath}/${artifactId}/${version}/${filename}`;
    const url = `${API_BASE_URL}/maven2/${path}`;
    
    // Trigger download
    const link = document.createElement('a');
    link.href = url;
    link.download = filename;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  }

  // Get POM file
  async getPOM(groupId, artifactId, version) {
    const groupPath = groupId.replace(/\./g, '/');
    const filename = `${artifactId}-${version}.pom`;
    const path = `${groupPath}/${artifactId}/${version}/${filename}`;
    
    const response = await fetch(`${API_BASE_URL}/maven2/${path}`);
    if (!response.ok) {
      throw new Error('Failed to fetch POM');
    }
    
    return response.text();
  }

  // Get maven-metadata.xml
  async getMetadata(groupId, artifactId) {
    const groupPath = groupId.replace(/\./g, '/');
    const path = `${groupPath}/${artifactId}/maven-metadata.xml`;
    
    const response = await fetch(`${API_BASE_URL}/maven2/${path}`);
    if (!response.ok) {
      throw new Error('Failed to fetch metadata');
    }
    
    return response.text();
  }
}

export default new MavenAPI();
