import React, { useState } from 'react';
import { Modal } from './Modal';
import { Download, Link as LinkIcon, AlertCircle } from 'lucide-react';

export const ImportArtifactModal = ({ isOpen, onClose, onSubmit, repositories }) => {
  const [formData, setFormData] = useState({
    repository: '',
    importType: 'url',
    url: '',
    coordinates: '',
    groupId: '',
    artifactId: '',
    version: '',
    includeMetadata: true,
    validateChecksum: true
  });

  const handleChange = (e) => {
    const { name, value, type, checked } = e.target;
    setFormData(prev => ({
      ...prev,
      [name]: type === 'checkbox' ? checked : value
    }));
  };

  const handleSubmit = (e) => {
    e.preventDefault();
    onSubmit(formData);
    onClose();
    setFormData({
      repository: '',
      importType: 'url',
      url: '',
      coordinates: '',
      groupId: '',
      artifactId: '',
      version: '',
      includeMetadata: true,
      validateChecksum: true
    });
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} title="Import Artifact" size="lg">
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
            className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          >
            <option value="">Select a repository...</option>
            {repositories?.map(repo => (
              <option key={repo.id} value={repo.name}>
                {repo.name} ({repo.type})
              </option>
            ))}
          </select>
        </div>

        {/* Import Type */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            Import Method *
          </label>
          <div className="grid grid-cols-2 gap-3">
            <label
              className={`flex items-center justify-center p-4 border rounded-lg cursor-pointer transition ${
                formData.importType === 'url'
                  ? 'border-blue-500 bg-blue-50'
                  : 'border-gray-300 hover:border-gray-400'
              }`}
            >
              <input
                type="radio"
                name="importType"
                value="url"
                checked={formData.importType === 'url'}
                onChange={handleChange}
                className="sr-only"
              />
              <div className="text-center">
                <LinkIcon className="w-6 h-6 mx-auto mb-2 text-gray-700" />
                <div className="font-medium text-gray-900">From URL</div>
                <div className="text-xs text-gray-500">Direct download link</div>
              </div>
            </label>

            <label
              className={`flex items-center justify-center p-4 border rounded-lg cursor-pointer transition ${
                formData.importType === 'maven'
                  ? 'border-blue-500 bg-blue-50'
                  : 'border-gray-300 hover:border-gray-400'
              }`}
            >
              <input
                type="radio"
                name="importType"
                value="maven"
                checked={formData.importType === 'maven'}
                onChange={handleChange}
                className="sr-only"
              />
              <div className="text-center">
                <Download className="w-6 h-6 mx-auto mb-2 text-gray-700" />
                <div className="font-medium text-gray-900">Maven Coordinates</div>
                <div className="text-xs text-gray-500">Group:Artifact:Version</div>
              </div>
            </label>
          </div>
        </div>

        {/* URL Import */}
        {formData.importType === 'url' && (
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Artifact URL *
            </label>
            <input
              type="url"
              name="url"
              required
              value={formData.url}
              onChange={handleChange}
              placeholder="https://example.com/artifact.jar"
              className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
            <p className="mt-1 text-xs text-gray-500">
              Direct download URL to the artifact file
            </p>
          </div>
        )}

        {/* Maven Coordinates Import */}
        {formData.importType === 'maven' && (
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Maven Coordinates
              </label>
              <input
                type="text"
                name="coordinates"
                value={formData.coordinates}
                onChange={handleChange}
                placeholder="org.springframework.boot:spring-boot-starter:3.1.0"
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
              <p className="mt-1 text-xs text-gray-500">
                Format: groupId:artifactId:version
              </p>
            </div>

            <div className="text-center text-gray-500 text-sm">OR</div>

            <div className="grid grid-cols-3 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Group ID *
                </label>
                <input
                  type="text"
                  name="groupId"
                  required={formData.importType === 'maven' && !formData.coordinates}
                  value={formData.groupId}
                  onChange={handleChange}
                  placeholder="org.example"
                  className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Artifact ID *
                </label>
                <input
                  type="text"
                  name="artifactId"
                  required={formData.importType === 'maven' && !formData.coordinates}
                  value={formData.artifactId}
                  onChange={handleChange}
                  placeholder="my-artifact"
                  className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Version *
                </label>
                <input
                  type="text"
                  name="version"
                  required={formData.importType === 'maven' && !formData.coordinates}
                  value={formData.version}
                  onChange={handleChange}
                  placeholder="1.0.0"
                  className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
              </div>
            </div>
          </div>
        )}

        {/* Options */}
        <div className="space-y-3 p-4 bg-gray-50 rounded-lg">
          <h4 className="font-medium text-gray-900">Import Options</h4>
          
          <label className="flex items-center space-x-2">
            <input
              type="checkbox"
              name="includeMetadata"
              checked={formData.includeMetadata}
              onChange={handleChange}
              className="rounded text-blue-600 focus:ring-blue-500"
            />
            <span className="text-sm text-gray-700">Include metadata (POM files, checksums)</span>
          </label>

          <label className="flex items-center space-x-2">
            <input
              type="checkbox"
              name="validateChecksum"
              checked={formData.validateChecksum}
              onChange={handleChange}
              className="rounded text-blue-600 focus:ring-blue-500"
            />
            <span className="text-sm text-gray-700">Validate checksum after import</span>
          </label>
        </div>

        {/* Info */}
        <div className="flex items-start space-x-2 p-4 bg-blue-50 border border-blue-200 rounded-lg">
          <AlertCircle className="w-5 h-5 text-blue-600 mt-0.5 flex-shrink-0" />
          <div className="text-sm text-blue-700">This will import the artifact and all its dependencies.</div>
        </div>

        {/* Action Buttons */}
        <div className="flex justify-end space-x-3">
          <button
            type="button"
            onClick={onClose}
            className="px-4 py-2 text-sm font-medium text-gray-700 bg-gray-100 border border-gray-300 rounded-lg hover:bg-gray-200"
          >
            Cancel
          </button>
          <button
            type="submit"
            className="px-4 py-2 text-sm font-medium text-white bg-blue-600 border border-transparent rounded-lg shadow-sm hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
          >
            Import
          </button>
        </div>
      </form>
    </Modal>
  );
};   