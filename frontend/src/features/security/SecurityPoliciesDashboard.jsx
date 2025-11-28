import React, { useState, useEffect } from 'react';
import {
  Shield,
  AlertTriangle,
  CheckCircle,
  XCircle,
  Plus,
  Edit,
  Trash2,
  Eye,
  Search,
  Filter
} from 'lucide-react';
import { Card, CardHeader, CardContent, Badge } from '../../components/common';
import { useToast } from '../../context/ToastContext';

export const SecurityPoliciesDashboard = () => {
  const [policies, setPolicies] = useState([]);
  const [vulnerabilities, setVulnerabilities] = useState([]);
  const [loading, setLoading] = useState(true);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const { showSuccess, showError } = useToast();

  useEffect(() => {
    fetchPolicies();
    fetchVulnerabilities();
  }, []);

  const fetchPolicies = async () => {
    try {
      const response = await fetch('/api/v1/security/policies');
      const data = await response.json();
      setPolicies(data.policies || []);
    } catch (error) {
      console.error('Failed to fetch policies:', error);
    } finally {
      setLoading(false);
    }
  };

  const fetchVulnerabilities = async () => {
    try {
      const response = await fetch('/api/v1/security/vulnerabilities?status=UNRESOLVED');
      const data = await response.json();
      setVulnerabilities(data.vulnerabilities || []);
    } catch (error) {
      console.error('Failed to fetch vulnerabilities:', error);
    }
  };

  const getSeverityColor = (severity) => {
    const colors = {
      CRITICAL: 'bg-red-100 text-red-800',
      HIGH: 'bg-orange-100 text-orange-800',
      MEDIUM: 'bg-yellow-100 text-yellow-800',
      LOW: 'bg-blue-100 text-blue-800'
    };
    return colors[severity] || 'bg-gray-100 text-gray-800';
  };

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Security Policies</h1>
          <p className="text-gray-600">Manage vulnerability scanning and compliance policies</p>
        </div>
        <button
          onClick={() => setShowCreateModal(true)}
          className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
        >
          <Plus className="w-4 h-4" />
          Create Policy
        </button>
      </div>

      {/* Statistics */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-gray-600">Active Policies</p>
                <p className="text-2xl font-bold text-gray-900">{policies.filter(p => p.enabled).length}</p>
              </div>
              <Shield className="w-8 h-8 text-blue-500" />
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-gray-600">Critical Issues</p>
                <p className="text-2xl font-bold text-red-600">
                  {vulnerabilities.filter(v => v.severity === 'CRITICAL').length}
                </p>
              </div>
              <AlertTriangle className="w-8 h-8 text-red-500" />
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-gray-600">High Issues</p>
                <p className="text-2xl font-bold text-orange-600">
                  {vulnerabilities.filter(v => v.severity === 'HIGH').length}
                </p>
              </div>
              <AlertTriangle className="w-8 h-8 text-orange-500" />
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-gray-600">Total Vulnerabilities</p>
                <p className="text-2xl font-bold text-gray-900">{vulnerabilities.length}</p>
              </div>
              <Shield className="w-8 h-8 text-gray-500" />
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Policies Table */}
      <Card>
        <CardHeader>
          <h3 className="text-lg font-semibold text-gray-900">Security Policies</h3>
        </CardHeader>
        <CardContent>
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Name</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Threshold</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Action</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Created</th>
                  <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200">
                {policies.length === 0 ? (
                  <tr>
                    <td colSpan="6" className="px-6 py-12 text-center text-gray-500">
                      <Shield className="w-12 h-12 mx-auto mb-3 text-gray-400" />
                      <p>No security policies configured yet</p>
                      <button
                        onClick={() => setShowCreateModal(true)}
                        className="mt-4 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
                      >
                        Create Your First Policy
                      </button>
                    </td>
                  </tr>
                ) : (
                  policies.map((policy) => (
                    <tr key={policy.id} className="hover:bg-gray-50">
                      <td className="px-6 py-4">
                        <div>
                          <p className="font-medium text-gray-900">{policy.name}</p>
                          <p className="text-sm text-gray-500">{policy.description}</p>
                        </div>
                      </td>
                      <td className="px-6 py-4">
                        <Badge className={getSeverityColor(policy.severity_threshold)}>
                          {policy.severity_threshold}
                        </Badge>
                      </td>
                      <td className="px-6 py-4">
                        <span className="text-sm text-gray-900">{policy.action}</span>
                      </td>
                      <td className="px-6 py-4">
                        {policy.enabled ? (
                          <Badge className="bg-green-100 text-green-800">Active</Badge>
                        ) : (
                          <Badge className="bg-gray-100 text-gray-800">Inactive</Badge>
                        )}
                      </td>
                      <td className="px-6 py-4 text-sm text-gray-500">
                        {new Date(policy.created_at).toLocaleDateString()}
                      </td>
                      <td className="px-6 py-4 text-right">
                        <button className="text-gray-600 hover:text-gray-900 mx-1">
                          <Eye className="w-4 h-4" />
                        </button>
                        <button className="text-blue-600 hover:text-blue-900 mx-1">
                          <Edit className="w-4 h-4" />
                        </button>
                        <button className="text-red-600 hover:text-red-900 mx-1">
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>

      {/* Vulnerabilities Table */}
      <Card>
        <CardHeader>
          <div className="flex justify-between items-center">
            <h3 className="text-lg font-semibold text-gray-900">Recent Vulnerabilities</h3>
            <button className="text-blue-600 hover:text-blue-800 text-sm font-medium">
              View All
            </button>
          </div>
        </CardHeader>
        <CardContent>
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">CVE ID</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Component</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Severity</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">CVSS Score</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Found</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200">
                {vulnerabilities.slice(0, 10).map((vuln, index) => (
                  <tr key={index} className="hover:bg-gray-50">
                    <td className="px-6 py-4">
                      <code className="text-sm font-mono text-blue-600">{vuln.cve_id}</code>
                    </td>
                    <td className="px-6 py-4">
                      <div>
                        <p className="font-medium text-gray-900">{vuln.component}</p>
                        <p className="text-sm text-gray-500">v{vuln.version}</p>
                      </div>
                    </td>
                    <td className="px-6 py-4">
                      <Badge className={getSeverityColor(vuln.severity)}>
                        {vuln.severity}
                      </Badge>
                    </td>
                    <td className="px-6 py-4">
                      <span className="text-sm font-medium text-gray-900">{vuln.cvss_score}</span>
                    </td>
                    <td className="px-6 py-4">
                      <Badge className="bg-yellow-100 text-yellow-800">{vuln.status}</Badge>
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-500">
                      {new Date(vuln.discovered_at).toLocaleDateString()}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>

      {/* Create Policy Modal */}
      {showCreateModal && (
        <CreatePolicyModal
          onClose={() => setShowCreateModal(false)}
          onSuccess={() => {
            setShowCreateModal(false);
            fetchPolicies();
          }}
        />
      )}
    </div>
  );
};

// Create Policy Modal Component
const CreatePolicyModal = ({ onClose, onSuccess }) => {
  const [formData, setFormData] = useState({
    name: '',
    description: '',
    severity_threshold: 'CRITICAL',
    action: 'BLOCK',
    enabled: true,
    compliance_framework: 'PCI-DSS',
    auto_quarantine: true,
    notify_on_breach: true
  });
  const [creating, setCreating] = useState(false);
  const { showSuccess, showError } = useToast();

  const handleSubmit = async (e) => {
    e.preventDefault();
    setCreating(true);

    try {
      const response = await fetch('/api/v1/security/policies', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(formData)
      });

      if (response.ok) {
        showSuccess('Security policy created successfully');
        onSuccess();
      } else {
        const error = await response.json();
        showError(`Failed to create policy: ${error.message}`);
      }
    } catch (error) {
      showError('Failed to create policy');
    } finally {
      setCreating(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg p-6 w-full max-w-2xl max-h-[90vh] overflow-y-auto">
        <h2 className="text-xl font-bold text-gray-900 mb-4">Create Security Policy</h2>
        
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Policy Name *</label>
            <input
              type="text"
              required
              value={formData.name}
              onChange={(e) => setFormData({ ...formData, name: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
              placeholder="e.g., Block CRITICAL & HIGH Vulnerabilities"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Description</label>
            <textarea
              value={formData.description}
              onChange={(e) => setFormData({ ...formData, description: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
              rows="3"
              placeholder="Describe the policy purpose..."
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Severity Threshold *</label>
              <select
                value={formData.severity_threshold}
                onChange={(e) => setFormData({ ...formData, severity_threshold: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
              >
                <option value="CRITICAL">CRITICAL</option>
                <option value="HIGH">HIGH</option>
                <option value="MEDIUM">MEDIUM</option>
                <option value="LOW">LOW</option>
              </select>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Action *</label>
              <select
                value={formData.action}
                onChange={(e) => setFormData({ ...formData, action: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
              >
                <option value="BLOCK">BLOCK</option>
                <option value="WARN">WARN</option>
                <option value="QUARANTINE">QUARANTINE</option>
              </select>
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Compliance Framework</label>
            <select
              value={formData.compliance_framework}
              onChange={(e) => setFormData({ ...formData, compliance_framework: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
            >
              <option value="PCI-DSS">PCI-DSS</option>
              <option value="HIPAA">HIPAA</option>
              <option value="SOC2">SOC 2</option>
              <option value="GDPR">GDPR</option>
              <option value="ISO27001">ISO 27001</option>
            </select>
          </div>

          <div className="space-y-2">
            <label className="flex items-center gap-2">
              <input
                type="checkbox"
                checked={formData.auto_quarantine}
                onChange={(e) => setFormData({ ...formData, auto_quarantine: e.target.checked })}
                className="rounded"
              />
              <span className="text-sm text-gray-700">Auto-quarantine affected artifacts</span>
            </label>

            <label className="flex items-center gap-2">
              <input
                type="checkbox"
                checked={formData.notify_on_breach}
                onChange={(e) => setFormData({ ...formData, notify_on_breach: e.target.checked })}
                className="rounded"
              />
              <span className="text-sm text-gray-700">Send email notifications on policy breach</span>
            </label>

            <label className="flex items-center gap-2">
              <input
                type="checkbox"
                checked={formData.enabled}
                onChange={(e) => setFormData({ ...formData, enabled: e.target.checked })}
                className="rounded"
              />
              <span className="text-sm text-gray-700">Enable policy immediately</span>
            </label>
          </div>

          <div className="flex gap-3 pt-4 border-t">
            <button
              type="button"
              onClick={onClose}
              className="flex-1 px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={creating}
              className="flex-1 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:bg-gray-300"
            >
              {creating ? 'Creating...' : 'Create Policy'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

export default SecurityPoliciesDashboard;
