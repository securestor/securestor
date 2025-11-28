import React, { useState, useEffect } from 'react';
import { Modal } from './Modal';
import { Upload, File, AlertCircle } from 'lucide-react';
import repositoryAPI from '../../services/api/repositoryAPI';
import artifactAPI from '../../services/api/artifactAPI';
import { useToast } from '../../context/ToastContext';
import realtimeService from '../../services/realtimeService';

export const DeployArtifactModal = ({ isOpen, onClose, onSubmit }) => {
  const { showSuccess, showError } = useToast();
  const [formData, setFormData] = useState({
    repository: '',
    artifactName: '',
    version: '',
    file: null,
    description: '',
    tags: '',
    license: '',
    artifactType: 'regular', // New field: 'regular', 'manifest', 'blob'
    digest: '' // For blob uploads
  });
  const [dragActive, setDragActive] = useState(false);
  const [repositories, setRepositories] = useState([]);
  const [loading, setLoading] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [uploadProgress, setUploadProgress] = useState(0);

  // Fetch repositories when modal opens
  useEffect(() => {
    if (isOpen) {
      fetchRepositories();
    }
  }, [isOpen]);

  const fetchRepositories = async () => {
    try {
      setLoading(true);
      const data = await repositoryAPI.listRepositories();
      setRepositories(data.repositories || []);
    } catch (error) {
      console.error('Failed to fetch repositories:', error);
      setRepositories([]);
    } finally {
      setLoading(false);
    }
  };

  const handleChange = (e) => {
    const { name, value } = e.target;
    setFormData(prev => ({ ...prev, [name]: value }));
  };

  const handleFileChange = (e) => {
    const file = e.target.files[0];
    if (file) {
      setFormData(prev => ({ ...prev, file }));
      
      // Auto-detect Docker artifact type based on file content
      if (selectedRepository?.type === 'docker') {
        detectDockerArtifactType(file);
      }
    }
  };

  const detectDockerArtifactType = async (file) => {
    try {
      const text = await file.text();
      
      // Check if it's a Docker manifest
      if (text.includes('schemaVersion') && text.includes('mediaType')) {
        const manifest = JSON.parse(text);
        if (manifest.mediaType && 
            (manifest.mediaType.includes('manifest') || 
             manifest.mediaType.includes('index'))) {
          setFormData(prev => ({ ...prev, artifactType: 'manifest' }));
          return;
        }
      }
      
      // Default to regular file for Docker repositories
      setFormData(prev => ({ ...prev, artifactType: 'regular' }));
    } catch (error) {
      // If we can't parse it, treat as regular file
      setFormData(prev => ({ ...prev, artifactType: 'regular' }));
    }
  };

  const selectedRepository = repositories.find(repo => 
    repo.id.toString() === formData.repository.toString()
  );

  const handleDrag = (e) => {
    e.preventDefault();
    e.stopPropagation();
    if (e.type === "dragenter" || e.type === "dragover") {
      setDragActive(true);
    } else if (e.type === "dragleave") {
      setDragActive(false);
    }
  };

  const handleDrop = (e) => {
    e.preventDefault();
    e.stopPropagation();
    setDragActive(false);
    
    if (e.dataTransfer.files && e.dataTransfer.files[0]) {
      setFormData(prev => ({ ...prev, file: e.dataTransfer.files[0] }));
    }
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    
    if (!formData.file) {
      showError('Please select a file to upload');
      return;
    }

    if (!formData.repository) {
      showError('Please select a repository');
      return;
    }

    if (!formData.artifactName || formData.artifactName.trim() === '') {
      showError('Please enter an artifact name');
      return;
    }

    if (!formData.version || formData.version.trim() === '') {
      showError('Please enter a version');
      return;
    }

    setUploading(true);
    setUploadProgress(0);

    // Store artifact name and version before making the API call
    const artifactName = formData.artifactName ? formData.artifactName.trim() : '';
    const version = formData.version ? formData.version.trim() : '';
    
    // Additional validation after trimming
    if (!artifactName) {
      showError('Please enter a valid artifact name');
      setUploading(false);
      return;
    }
    
    if (!version) {
      showError('Please enter a valid version');
      setUploading(false);
      return;
    }
    

    try {
      const response = await artifactAPI.deployArtifact(
        formData,
        formData.file,
        (progress) => {
          // Cap at 95% during upload, save 5% for processing
          const displayProgress = Math.min(progress * 0.95, 95);
          setUploadProgress(displayProgress);
        }
      );

      // Smooth transition to 100% after upload completes
      setUploadProgress(100);
      
      // Brief delay to show completion before closing
      await new Promise(resolve => setTimeout(resolve, 500));

      
      
      // Use the stored values instead of formData to avoid potential undefined issues
      const displayName = artifactName || 'Artifact';
      const displayVersion = version || 'Unknown';
      showSuccess(`Artifact "${displayName}:${displayVersion}" deployed successfully!`);
      
      // Emit real-time event for artifact upload
      realtimeService.send({
        type: 'artifact',
        event: 'artifact.uploaded',
        data: {
          ...response,
          repository_id: formData.repository,
          name: artifactName,
          version: version
        }
      });
      
      // Call parent's onSubmit if provided with artifact info
      if (onSubmit) {
        // Find the repository name from the repositories list
        const selectedRepo = repositories.find(repo => repo.id.toString() === formData.repository.toString());
        const repositoryName = selectedRepo ? selectedRepo.name : formData.repository;
        
        onSubmit({
          ...response,
          artifactName: artifactName,
          version: version,
          repository: repositoryName
        });
      }

      // Close modal and reset form
      onClose();
      resetForm();
    } catch (error) {
      console.error('Failed to deploy artifact:', error);
      console.error('Error details:', {
        message: error.message,
        stack: error.stack
      });
      
      // Handle specific error types with user-friendly messages
      let errorMessage = error.message;
      
      if (error.message.includes('duplicate key value violates unique constraint') || 
          error.message.includes('artifacts_name_version_repository_id_key')) {
        errorMessage = `Artifact "${artifactName}:${version}" already exists in this repository. Please use a different version number.`;
      } else if (error.message.includes('Network error') || error.message.includes('Unable to connect')) {
        errorMessage = 'Unable to connect to server. Please check if the backend is running.';
      } else if (error.message.includes('File too large')) {
        errorMessage = 'File is too large. Maximum upload size is 10GB.';
      } else if (error.message.includes('Invalid server response')) {
        errorMessage = 'Server returned an invalid response. Please try again.';
      }
      
      showError(`Failed to deploy artifact: ${errorMessage}`);
    } finally {
      setUploading(false);
      setUploadProgress(0);
    }
  };

  const resetForm = () => {
    setFormData({
      repository: '',
      artifactName: '',
      version: '',
      file: null,
      description: '',
      tags: '',
      license: '',
      artifactType: 'regular',
      digest: ''
    });
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} title="Deploy Artifact" size="lg">
      <form onSubmit={handleSubmit} className="space-y-6">
        {/* Repository Selection */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            Target Repository *
          </label>
          <select
            name="repository"
            required
            value={formData.repository}
            onChange={handleChange}
            disabled={loading}
            className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent disabled:bg-gray-100 disabled:cursor-not-allowed"
          >
            <option value="">
              {loading ? 'Loading repositories...' : 'Select a repository...'}
            </option>
            {repositories?.map(repo => (
              <option key={repo.id} value={repo.id}>
                {repo.name} ({repo.type})
              </option>
            ))}
          </select>
        </div>

        {/* Artifact Details */}
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Artifact Name *
            </label>
            <input
              type="text"
              name="artifactName"
              required
              value={formData.artifactName}
              onChange={handleChange}
              placeholder={selectedRepository?.type === 'docker' ? 'e.g., myapp/nginx' : 'e.g., myapp'}
              className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
            {selectedRepository?.type === 'docker' && (
              <p className="text-xs text-gray-500 mt-1">
                Use namespace/repository format for Docker images
              </p>
            )}
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Version *
            </label>
            <input
              type="text"
              name="version"
              required
              value={formData.version}
              onChange={handleChange}
              placeholder={selectedRepository?.type === 'docker' ? 'e.g., latest, v1.0.0' : 'e.g., 1.0.0'}
              className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
            {selectedRepository?.type === 'docker' && (
              <p className="text-xs text-gray-500 mt-1">
                Docker tag for the image
              </p>
            )}
          </div>
        </div>

        {/* Docker-specific artifact type selection */}
        {selectedRepository?.type === 'docker' && (
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Artifact Type
            </label>
            <div className="grid grid-cols-3 gap-3">
              <label className={`flex items-center space-x-2 p-3 border rounded-lg cursor-pointer transition ${
                formData.artifactType === 'regular'
                  ? "border-blue-500 bg-blue-50"
                  : "border-gray-300 hover:border-gray-400"
              }`}>
                <input
                  type="radio"
                  name="artifactType"
                  value="regular"
                  checked={formData.artifactType === 'regular'}
                  onChange={handleChange}
                  className="text-blue-600 focus:ring-blue-500"
                />
                <span className="text-sm font-medium text-gray-900">
                  Regular File
                </span>
              </label>
              
              <label className={`flex items-center space-x-2 p-3 border rounded-lg cursor-pointer transition ${
                formData.artifactType === 'manifest'
                  ? "border-blue-500 bg-blue-50"
                  : "border-gray-300 hover:border-gray-400"
              }`}>
                <input
                  type="radio"
                  name="artifactType"
                  value="manifest"
                  checked={formData.artifactType === 'manifest'}
                  onChange={handleChange}
                  className="text-blue-600 focus:ring-blue-500"
                />
                <span className="text-sm font-medium text-gray-900">
                  OCI Manifest
                </span>
              </label>
              
              <label className={`flex items-center space-x-2 p-3 border rounded-lg cursor-pointer transition ${
                formData.artifactType === 'blob'
                  ? "border-blue-500 bg-blue-50"
                  : "border-gray-300 hover:border-gray-400"
              }`}>
                <input
                  type="radio"
                  name="artifactType"
                  value="blob"
                  checked={formData.artifactType === 'blob'}
                  onChange={handleChange}
                  className="text-blue-600 focus:ring-blue-500"
                />
                <span className="text-sm font-medium text-gray-900">
                  OCI Blob
                </span>
              </label>
            </div>
            
            {formData.artifactType === 'manifest' && (
              <div className="mt-2 p-3 bg-blue-50 border border-blue-200 rounded-lg">
                <p className="text-sm text-blue-800">
                  <strong>OCI Manifest:</strong> Upload a Docker/OCI image manifest JSON file. 
                  This defines the image configuration and layers.
                </p>
              </div>
            )}
            
            {formData.artifactType === 'blob' && (
              <div className="mt-2 space-y-3">
                <div className="p-3 bg-blue-50 border border-blue-200 rounded-lg">
                  <p className="text-sm text-blue-800">
                    <strong>OCI Blob:</strong> Upload image layers, configs, or other blob content.
                    You must provide the expected digest.
                  </p>
                </div>
                
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Expected Digest *
                  </label>
                  <input
                    type="text"
                    name="digest"
                    required={formData.artifactType === 'blob'}
                    value={formData.digest}
                    onChange={handleChange}
                    placeholder="sha256:abc123..."
                    className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  />
                  <p className="text-xs text-gray-500 mt-1">
                    The SHA256 digest of the blob content
                  </p>
                </div>
              </div>
            )}
          </div>
        )}

        {/* File Upload */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            Artifact File *
          </label>
          <div
            className={`relative border-2 border-dashed rounded-lg p-8 text-center transition ${
              dragActive 
                ? 'border-blue-500 bg-blue-50' 
                : 'border-gray-300 hover:border-gray-400'
            }`}
            onDragEnter={handleDrag}
            onDragLeave={handleDrag}
            onDragOver={handleDrag}
            onDrop={handleDrop}
          >
            <input
              type="file"
              required
              onChange={handleFileChange}
              className="absolute inset-0 w-full h-full opacity-0 cursor-pointer"
            />
            <div className="space-y-3">
              {formData.file ? (
                <>
                  <File className="w-12 h-12 text-green-600 mx-auto" />
                  <div>
                    <p className="font-medium text-gray-900">{formData.file.name}</p>
                    <p className="text-sm text-gray-500">
                      {(formData.file.size / 1024 / 1024).toFixed(2)} MB
                    </p>
                  </div>
                  <button
                    type="button"
                    onClick={() => setFormData(prev => ({ ...prev, file: null }))}
                    className="text-sm text-red-600 hover:text-red-700"
                  >
                    Remove file
                  </button>
                </>
              ) : (
                <>
                  <Upload className="w-12 h-12 text-gray-400 mx-auto" />
                  <div>
                    <p className="text-gray-700">
                      <span className="font-medium text-blue-600">Click to upload</span> or drag and drop
                    </p>
                    <p className="text-sm text-gray-500 mt-1">
                      {selectedRepository?.type === 'docker' 
                        ? formData.artifactType === 'manifest' 
                          ? 'Docker/OCI manifest JSON file'
                          : formData.artifactType === 'blob'
                          ? 'Docker layer, config, or blob file'
                          : 'Docker image archive, Dockerfile, or any file'
                        : selectedRepository?.type === 'npm'
                        ? 'NPM package (.tgz)'
                        : selectedRepository?.type === 'maven'
                        ? 'JAR, WAR, POM files'
                        : selectedRepository?.type === 'pypi'
                        ? 'Python wheel (.whl) or source distribution'
                        : selectedRepository?.type === 'helm'
                        ? 'Helm chart (.tgz)'
                        : 'JAR, WAR, ZIP, TAR.GZ, or any file type'
                      }
                    </p>
                  </div>
                </>
              )}
            </div>
          </div>
        </div>

        {/* Description */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            Description
          </label>
          <textarea
            name="description"
            value={formData.description}
            onChange={handleChange}
            rows={3}
            placeholder="What's new in this version..."
            className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
        </div>

        {/* Tags */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            Tags
          </label>
          <input
            type="text"
            name="tags"
            value={formData.tags}
            onChange={handleChange}
            placeholder="production, stable, feature-x (comma-separated)"
            className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
        </div>

        {/* License */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            License
          </label>
          <input
            type="text"
            name="license"
            value={formData.license}
            onChange={handleChange}
            placeholder="e.g., MIT, Apache-2.0, GPL-3.0"
            className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
        </div>

        {/* Upload Progress */}
        {uploading && (
          <div className="space-y-2">
            <div className="flex justify-between text-sm">
              <span className="text-gray-700">
                {uploadProgress >= 95 && uploadProgress < 100 ? 'Processing...' : 'Uploading...'}
              </span>
              <span className="text-gray-900 font-medium">{Math.round(uploadProgress)}%</span>
            </div>
            <div className="w-full bg-gray-200 rounded-full h-2.5">
              <div 
                className={`h-2.5 rounded-full transition-all duration-500 ease-out ${
                  uploadProgress >= 95 && uploadProgress < 100 
                    ? 'bg-gradient-to-r from-blue-500 to-purple-500 animate-pulse' 
                    : uploadProgress === 100
                    ? 'bg-green-500'
                    : 'bg-blue-600'
                }`}
                style={{ width: `${uploadProgress}%` }}
              ></div>
            </div>
            {uploadProgress >= 95 && uploadProgress < 100 && (
              <p className="text-xs text-gray-500 text-center">
                Finalizing upload and processing artifact...
              </p>
            )}
            {uploadProgress === 100 && (
              <p className="text-xs text-green-600 text-center font-medium">
                ✓ Upload complete!
              </p>
            )}
          </div>
        )}

        {/* Warning */}
        <div className="flex items-start space-x-2 p-4 bg-yellow-50 border border-yellow-200 rounded-lg">
          <AlertCircle className="w-5 h-5 text-yellow-600 mt-0.5 flex-shrink-0" />
          <div className="text-sm text-yellow-900">
            <p className="font-medium mb-1">Important:</p>
            <ul className="list-disc list-inside space-y-1 text-yellow-800">
              <li>Deploying will overwrite existing artifact with same name and version</li>
              <li>Ensure artifact is properly tested before deployment</li>
              <li>Deployment cannot be undone</li>
            </ul>
          </div>
        </div>

        {/* Actions */}
        <div className="flex justify-end space-x-3 pt-4 border-t border-gray-200">
          <button
            type="button"
            onClick={onClose}
            disabled={uploading}
            className="px-4 py-2 text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition disabled:opacity-50 disabled:cursor-not-allowed"
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={uploading || loading}
            className="px-4 py-2 text-white bg-blue-600 rounded-lg hover:bg-blue-700 transition flex items-center space-x-2 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <Upload className="w-4 h-4" />
            <span>{uploading ? 'Deploying...' : 'Deploy Artifact'}</span>
          </button>
        </div>
      </form>
    </Modal>
  );
};