import React, { useState } from 'react';
import { Modal } from './Modal';
import { Building2, AlertCircle, Check } from 'lucide-react';
import { tenantApi } from '../../services/tenantApi';
import { useToast } from '../../context/ToastContext';

export const CreateTenantModal = ({ isOpen, onClose, onSubmit }) => {
  const { showSuccess, showError } = useToast();
  const [formData, setFormData] = useState({
    name: '',
    slug: '',  // Changed from subdomain to slug to match backend
    description: '',
    plan: 'basic',
    max_users: 100,
    max_repositories: 50,
    max_storage_gb: 100,
    features: [],
    contact_email: '',
    billing_email: ''
  });
  const [loading, setLoading] = useState(false);
  const [errors, setErrors] = useState({});

  const availableFeatures = [
    { id: 'security_scanning', label: 'Security Scanning', description: 'Vulnerability detection and reporting' },
    { id: 'compliance_reporting', label: 'Compliance Reporting', description: 'Regulatory compliance tracking' },
    { id: 'api_access', label: 'API Access', description: 'Full REST API access' },
    { id: 'custom_branding', label: 'Custom Branding', description: 'White-label customization' },
    { id: 'advanced_audit_logs', label: 'Advanced Audit Logs', description: 'Detailed activity logging' },
    { id: 'sso_integration', label: 'SSO Integration', description: 'Single sign-on support' },
    { id: 'backup_restore', label: 'Backup & Restore', description: 'Automated data backup' }
  ];

  const planLimits = {
    free: { max_users: 5, max_repositories: 10, max_storage_gb: 10 },
    basic: { max_users: 100, max_repositories: 50, max_storage_gb: 100 },
    premium: { max_users: 500, max_repositories: 200, max_storage_gb: 500 },
    enterprise: { max_users: 10000, max_repositories: 1000, max_storage_gb: 10000 }
  };

  const handleChange = (e) => {
    const { name, value, type, checked } = e.target;
    
    if (name === 'features') {
      const currentFeatures = formData.features || [];
      const updatedFeatures = checked
        ? [...currentFeatures, value]
        : currentFeatures.filter(f => f !== value);
      
      setFormData(prev => ({ ...prev, features: updatedFeatures }));
    } else if (name === 'plan') {
      // Update limits based on plan
      const limits = planLimits[value] || planLimits.basic;
      setFormData(prev => ({ 
        ...prev, 
        [name]: value,
        ...limits
      }));
    } else {
      setFormData(prev => ({ 
        ...prev, 
        [name]: type === 'number' ? parseInt(value) || 0 : value 
      }));
    }
    
    // Clear error when user starts typing
    if (errors[name]) {
      setErrors(prev => ({ ...prev, [name]: null }));
    }
  };

  const validateForm = () => {
    const newErrors = {};
    
    if (!formData.name.trim()) {
      newErrors.name = 'Tenant name is required';
    }
    
    if (!formData.slug.trim()) {
      newErrors.slug = 'Slug is required';
    } else if (!/^[a-z0-9]([a-z0-9-]*[a-z0-9])?$/.test(formData.slug)) {
      newErrors.slug = 'Slug must be lowercase alphanumeric with hyphens, and start/end with alphanumeric';
    } else if (formData.slug.length > 63) {
      newErrors.slug = 'Slug must be 63 characters or less';
    }
    
    if (!formData.contact_email.trim()) {
      newErrors.contact_email = 'Contact email is required';
    } else if (!/\S+@\S+\.\S+/.test(formData.contact_email)) {
      newErrors.contact_email = 'Please enter a valid email address';
    }
    
    if (formData.max_users < 1) {
      newErrors.max_users = 'Max users must be at least 1';
    }
    
    if (formData.max_repositories < 1) {
      newErrors.max_repositories = 'Max repositories must be at least 1';
    }
    
    if (formData.max_storage_gb < 1) {
      newErrors.max_storage_gb = 'Max storage must be at least 1 GB';
    }
    
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    
    if (!validateForm()) {
      return;
    }
    
    setLoading(true);
    
    try {
      const response = await tenantApi.createTenant(formData);
      
      showSuccess(`Tenant "${formData.name}" created successfully!`);
      
      // Call parent callback if provided
      if (onSubmit) {
        onSubmit(response);
      }
      
      // Close modal and reset form
      onClose();
      resetForm();
    } catch (error) {
      console.error('Failed to create tenant:', error);
      showError(`Failed to create tenant: ${error.message}`);
    } finally {
      setLoading(false);
    }
  };

  const resetForm = () => {
    setFormData({
      name: '',
      slug: '',  // Changed from subdomain to slug
      description: '',
      plan: 'basic',
      max_users: 100,
      max_repositories: 50,
      max_storage_gb: 100,
      features: [],
      contact_email: '',
      billing_email: ''
    });
    setErrors({});
  };

  const handleClose = () => {
    if (!loading) {
      onClose();
      resetForm();
    }
  };

  return (
    <Modal isOpen={isOpen} onClose={handleClose} title="Create New Tenant" size="lg">
      <form onSubmit={handleSubmit} className="space-y-6">
        {/* Basic Information */}
        <div className="space-y-4">
          <h3 className="text-lg font-medium text-gray-900 flex items-center gap-2">
            <Building2 className="h-5 w-5" />
            Basic Information
          </h3>
          
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Tenant Name *
              </label>
              <input
                type="text"
                name="name"
                required
                value={formData.name}
                onChange={handleChange}
                placeholder="e.g., Acme Corporation"
                className={`w-full px-4 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent ${
                  errors.name ? 'border-red-500' : 'border-gray-300'
                }`}
              />
              {errors.name && (
                <p className="text-red-600 text-sm mt-1 flex items-center gap-1">
                  <AlertCircle className="h-4 w-4" />
                  {errors.name}
                </p>
              )}
            </div>
            
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Tenant Slug * <span className="text-gray-500 text-xs">(used for subdomain)</span>
              </label>
              <div className="flex">
                <input
                  type="text"
                  name="slug"
                  required
                  value={formData.slug}
                  onChange={handleChange}
                  placeholder="acme"
                  className={`flex-1 px-4 py-2 border rounded-l-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent ${
                    errors.slug ? 'border-red-500' : 'border-gray-300'
                  }`}
                />
                <span className="px-3 py-2 bg-gray-100 border border-l-0 border-gray-300 rounded-r-lg text-gray-500 text-sm">
                  .localhost
                </span>
              </div>
              {errors.slug && (
                <p className="text-red-600 text-sm mt-1 flex items-center gap-1">
                  <AlertCircle className="h-4 w-4" />
                  {errors.slug}
                </p>
              )}
              <p className="text-gray-500 text-xs mt-1">
                Must be lowercase, alphanumeric with hyphens only. Will be used as: {formData.slug || 'your-slug'}.localhost:3000
              </p>
            </div>
          </div>
          
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Description
            </label>
            <textarea
              name="description"
              value={formData.description}
              onChange={handleChange}
              rows={3}
              placeholder="Brief description of the organization..."
              className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>
        </div>

        {/* Plan & Limits */}
        <div className="space-y-4">
          <h3 className="text-lg font-medium text-gray-900">Plan & Resource Limits</h3>
          
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Subscription Plan
              </label>
              <select
                name="plan"
                value={formData.plan}
                onChange={handleChange}
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              >
                <option value="free">Free</option>
                <option value="basic">Basic</option>
                <option value="premium">Premium</option>
                <option value="enterprise">Enterprise</option>
              </select>
            </div>
          </div>
          
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Max Users
              </label>
              <input
                type="number"
                name="max_users"
                min="1"
                value={formData.max_users}
                onChange={handleChange}
                className={`w-full px-4 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent ${
                  errors.max_users ? 'border-red-500' : 'border-gray-300'
                }`}
              />
              {errors.max_users && (
                <p className="text-red-600 text-sm mt-1">{errors.max_users}</p>
              )}
            </div>
            
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Max Repositories
              </label>
              <input
                type="number"
                name="max_repositories"
                min="1"
                value={formData.max_repositories}
                onChange={handleChange}
                className={`w-full px-4 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent ${
                  errors.max_repositories ? 'border-red-500' : 'border-gray-300'
                }`}
              />
              {errors.max_repositories && (
                <p className="text-red-600 text-sm mt-1">{errors.max_repositories}</p>
              )}
            </div>
            
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Max Storage (GB)
              </label>
              <input
                type="number"
                name="max_storage_gb"
                min="1"
                value={formData.max_storage_gb}
                onChange={handleChange}
                className={`w-full px-4 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent ${
                  errors.max_storage_gb ? 'border-red-500' : 'border-gray-300'
                }`}
              />
              {errors.max_storage_gb && (
                <p className="text-red-600 text-sm mt-1">{errors.max_storage_gb}</p>
              )}
            </div>
          </div>
        </div>

        {/* Contact Information */}
        <div className="space-y-4">
          <h3 className="text-lg font-medium text-gray-900">Contact Information</h3>
          
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Contact Email *
              </label>
              <input
                type="email"
                name="contact_email"
                required
                value={formData.contact_email}
                onChange={handleChange}
                placeholder="admin@acme.com"
                className={`w-full px-4 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent ${
                  errors.contact_email ? 'border-red-500' : 'border-gray-300'
                }`}
              />
              {errors.contact_email && (
                <p className="text-red-600 text-sm mt-1 flex items-center gap-1">
                  <AlertCircle className="h-4 w-4" />
                  {errors.contact_email}
                </p>
              )}
            </div>
            
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Billing Email
              </label>
              <input
                type="email"
                name="billing_email"
                value={formData.billing_email}
                onChange={handleChange}
                placeholder="billing@acme.com"
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
              <p className="text-xs text-gray-500 mt-1">
                Leave empty to use contact email
              </p>
            </div>
          </div>
        </div>

        {/* Features */}
        <div className="space-y-4">
          <h3 className="text-lg font-medium text-gray-900">Features</h3>
          
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            {availableFeatures.map(feature => (
              <label key={feature.id} className="flex items-start space-x-3 p-3 border border-gray-200 rounded-lg hover:bg-gray-50 cursor-pointer">
                <input
                  type="checkbox"
                  name="features"
                  value={feature.id}
                  checked={formData.features.includes(feature.id)}
                  onChange={handleChange}
                  className="mt-0.5 h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                />
                <div className="flex-1">
                  <div className="text-sm font-medium text-gray-900">{feature.label}</div>
                  <div className="text-xs text-gray-600">{feature.description}</div>
                </div>
              </label>
            ))}
          </div>
        </div>

        {/* Form Actions */}
        <div className="flex justify-end space-x-4 pt-4 border-t border-gray-200">
          <button
            type="button"
            onClick={handleClose}
            disabled={loading}
            className="px-4 py-2 text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 focus:ring-2 focus:ring-blue-500 focus:border-transparent disabled:opacity-50 disabled:cursor-not-allowed"
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={loading}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
          >
            {loading ? (
              <>
                <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white"></div>
                Creating...
              </>
            ) : (
              <>
                <Check className="h-4 w-4" />
                Create Tenant
              </>
            )}
          </button>
        </div>
      </form>
    </Modal>
  );
};