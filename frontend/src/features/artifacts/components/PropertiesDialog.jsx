import React, { useState, useEffect } from 'react';
import { X, Plus, Save, Tag, Lock, AlertCircle } from 'lucide-react';
import PropertiesAPI from '../../../services/api/propertiesAPI';
import { useToast } from '../../../context/ToastContext';

const PropertiesDialog = ({ 
  isOpen, 
  onClose, 
  artifact, 
  onPropertiesUpdated 
}) => {
  const [properties, setProperties] = useState([]);
  const [loading, setLoading] = useState(false);
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [newProperty, setNewProperty] = useState({
    key: '',
    value: '',
    value_type: 'string',
    is_sensitive: false,
    is_multi_value: false,
    tags: '',
    description: ''
  });
  
  const { showError, showSuccess } = useToast();

  useEffect(() => {
    if (isOpen && artifact) {
      loadProperties();
    }
  }, [isOpen, artifact]);

  const loadProperties = async () => {
    try {
      setLoading(true);
      const data = await PropertiesAPI.getArtifactProperties(artifact.id);
      setProperties(data);
    } catch (error) {
      showError('Failed to load properties: ' + error.message);
    } finally {
      setLoading(false);
    }
  };

  const handleCreateProperty = async () => {
    // Validate
    if (!newProperty.key || !newProperty.value) {
      showError('Key and value are required');
      return;
    }

    // Validate key format
    const keyRegex = /^[a-zA-Z0-9._-]+$/;
    if (!keyRegex.test(newProperty.key)) {
      showError('Key can only contain letters, numbers, dots, underscores, and hyphens');
      return;
    }

    try {
      setLoading(true);
      
      // Parse tags
      const tags = newProperty.tags 
        ? newProperty.tags.split(',').map(t => t.trim()).filter(t => t)
        : [];

      await PropertiesAPI.createProperty(artifact.id, {
        ...newProperty,
        tags
      });

      showSuccess('Property created successfully');
      
      // Reset form
      setNewProperty({
        key: '',
        value: '',
        value_type: 'string',
        is_sensitive: false,
        is_multi_value: false,
        tags: '',
        description: ''
      });
      setShowCreateForm(false);
      
      // Reload properties
      await loadProperties();
      
      // Notify parent
      if (onPropertiesUpdated) {
        onPropertiesUpdated();
      }
    } catch (error) {
      showError('Failed to create property: ' + error.message);
    } finally {
      setLoading(false);
    }
  };

  const handleDeleteProperty = async (propertyId) => {
    if (!window.confirm('Are you sure you want to delete this property?')) {
      return;
    }

    try {
      setLoading(true);
      await PropertiesAPI.deleteProperty(propertyId);
      showSuccess('Property deleted successfully');
      await loadProperties();
      
      if (onPropertiesUpdated) {
        onPropertiesUpdated();
      }
    } catch (error) {
      showError('Failed to delete property: ' + error.message);
    } finally {
      setLoading(false);
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl w-full max-w-4xl max-h-[90vh] overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b">
          <div>
            <h2 className="text-xl font-semibold">Artifact Properties</h2>
            <p className="text-sm text-gray-600 mt-1">
              {artifact?.name} - {artifact?.version}
            </p>
          </div>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600"
          >
            <X className="w-6 h-6" />
          </button>
        </div>

        {/* Content */}
        <div className="p-6 overflow-y-auto max-h-[calc(90vh-180px)]">
          {/* Create Property Button */}
          {!showCreateForm && (
            <button
              onClick={() => setShowCreateForm(true)}
              className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 mb-4"
            >
              <Plus className="w-4 h-4" />
              Add Property
            </button>
          )}

          {/* Create Form */}
          {showCreateForm && (
            <div className="bg-gray-50 rounded-lg p-4 mb-4">
              <h3 className="text-lg font-medium mb-4">Create New Property</h3>
              
              <div className="space-y-4">
                {/* Key */}
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Key *
                  </label>
                  <input
                    type="text"
                    value={newProperty.key}
                    onChange={(e) => setNewProperty({ ...newProperty, key: e.target.value })}
                    placeholder="custom.version"
                    className="w-full px-3 py-2 border border-gray-300 rounded focus:ring-2 focus:ring-blue-500"
                  />
                  <p className="text-xs text-gray-500 mt-1">
                    Format: letters, numbers, dots, hyphens, underscores
                  </p>
                </div>

                {/* Value */}
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Value *
                  </label>
                  <textarea
                    value={newProperty.value}
                    onChange={(e) => setNewProperty({ ...newProperty, value: e.target.value })}
                    placeholder="1.0.0"
                    rows={3}
                    className="w-full px-3 py-2 border border-gray-300 rounded focus:ring-2 focus:ring-blue-500"
                  />
                </div>

                {/* Type */}
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Type
                  </label>
                  <select
                    value={newProperty.value_type}
                    onChange={(e) => setNewProperty({ ...newProperty, value_type: e.target.value })}
                    className="w-full px-3 py-2 border border-gray-300 rounded focus:ring-2 focus:ring-blue-500"
                  >
                    <option value="string">String</option>
                    <option value="number">Number</option>
                    <option value="boolean">Boolean</option>
                    <option value="json">JSON</option>
                    <option value="array">Array</option>
                  </select>
                </div>

                {/* Flags */}
                <div className="flex gap-4">
                  <label className="flex items-center gap-2">
                    <input
                      type="checkbox"
                      checked={newProperty.is_sensitive}
                      onChange={(e) => setNewProperty({ ...newProperty, is_sensitive: e.target.checked })}
                      className="rounded"
                    />
                    <Lock className="w-4 h-4" />
                    <span className="text-sm">Sensitive (encrypted)</span>
                  </label>

                  <label className="flex items-center gap-2">
                    <input
                      type="checkbox"
                      checked={newProperty.is_multi_value}
                      onChange={(e) => setNewProperty({ ...newProperty, is_multi_value: e.target.checked })}
                      className="rounded"
                    />
                    <span className="text-sm">Multi-value</span>
                  </label>
                </div>

                {/* Tags */}
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Tags (comma-separated)
                  </label>
                  <input
                    type="text"
                    value={newProperty.tags}
                    onChange={(e) => setNewProperty({ ...newProperty, tags: e.target.value })}
                    placeholder="release, production, v1"
                    className="w-full px-3 py-2 border border-gray-300 rounded focus:ring-2 focus:ring-blue-500"
                  />
                </div>

                {/* Description */}
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Description
                  </label>
                  <textarea
                    value={newProperty.description}
                    onChange={(e) => setNewProperty({ ...newProperty, description: e.target.value })}
                    placeholder="Optional description"
                    rows={2}
                    className="w-full px-3 py-2 border border-gray-300 rounded focus:ring-2 focus:ring-blue-500"
                  />
                </div>

                {/* Buttons */}
                <div className="flex gap-2">
                  <button
                    onClick={handleCreateProperty}
                    disabled={loading}
                    className="flex items-center gap-2 px-4 py-2 bg-green-600 text-white rounded hover:bg-green-700 disabled:opacity-50"
                  >
                    <Save className="w-4 h-4" />
                    Create
                  </button>
                  <button
                    onClick={() => setShowCreateForm(false)}
                    className="px-4 py-2 bg-gray-200 text-gray-700 rounded hover:bg-gray-300"
                  >
                    Cancel
                  </button>
                </div>
              </div>
            </div>
          )}

          {/* Properties List */}
          {loading && !properties.length ? (
            <div className="text-center py-8">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto"></div>
              <p className="text-gray-600 mt-2">Loading properties...</p>
            </div>
          ) : properties.length === 0 ? (
            <div className="text-center py-8">
              <AlertCircle className="w-12 h-12 text-gray-400 mx-auto mb-2" />
              <p className="text-gray-600">No properties found</p>
              <p className="text-sm text-gray-500 mt-1">Create one to get started</p>
            </div>
          ) : (
            <div className="space-y-2">
              {properties.map((property) => (
                <div
                  key={property.id}
                  className="flex items-start justify-between p-4 bg-white border rounded hover:shadow-sm"
                >
                  <div className="flex-1">
                    <div className="flex items-center gap-2">
                      <span className="font-medium text-gray-900">{property.key}</span>
                      {property.is_sensitive && (
                        <Lock className="w-4 h-4 text-orange-500" />
                      )}
                      {property.is_system && (
                        <span className="px-2 py-1 text-xs bg-blue-100 text-blue-700 rounded">
                          System
                        </span>
                      )}
                    </div>
                    <div className="mt-1 text-sm text-gray-600">
                      {property.is_sensitive && !property.value ? (
                        <span className="text-orange-600">••••••••</span>
                      ) : (
                        <span>{property.value}</span>
                      )}
                    </div>
                    {property.tags && property.tags.length > 0 && (
                      <div className="mt-2 flex gap-1">
                        {property.tags.map((tag, idx) => (
                          <span
                            key={idx}
                            className="inline-flex items-center gap-1 px-2 py-1 text-xs bg-gray-100 text-gray-700 rounded"
                          >
                            <Tag className="w-3 h-3" />
                            {tag}
                          </span>
                        ))}
                      </div>
                    )}
                    {property.description && (
                      <p className="mt-1 text-xs text-gray-500">{property.description}</p>
                    )}
                    <p className="mt-1 text-xs text-gray-400">
                      Type: {property.value_type} | Updated: {new Date(property.updated_at).toLocaleString()}
                    </p>
                  </div>
                  
                  {!property.is_system && (
                    <button
                      onClick={() => handleDeleteProperty(property.id)}
                      className="ml-4 text-red-600 hover:text-red-800"
                      title="Delete property"
                    >
                      <X className="w-5 h-5" />
                    </button>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="flex justify-end gap-2 p-6 border-t bg-gray-50">
          <button
            onClick={onClose}
            className="px-4 py-2 bg-gray-200 text-gray-700 rounded hover:bg-gray-300"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
};

export default PropertiesDialog;
