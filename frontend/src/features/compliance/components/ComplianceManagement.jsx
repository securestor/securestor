import React, { useState, useEffect, useCallback } from 'react';
import { 
  Shield, 
  Lock, 
  Globe, 
  Clock, 
  Trash2, 
  Eye, 
  Plus, 
  AlertTriangle,
  CheckCircle,
  XCircle,
  FileText,
  Download,
  Edit3,
  Save,
  Play,
  Code,
  Filter,
  Search,
  Calendar,
  User,
  Settings,
  BarChart3,
  Activity,
  Database,
  AlertCircle,
  BookOpen,
  Gavel,
  Fingerprint,
  FileX,
  RefreshCw,
  ArrowRight,
  TrendingUp,
  Users,
  Server,
  ShieldCheck,
  ClipboardList
} from 'lucide-react';
import { useToast } from '../../../context/ToastContext';
import { useTranslation } from '../../../hooks/useTranslation';
import complianceAPI from '../../../services/api/complianceAPI';
import SchedulerManagement from './SchedulerManagement';

const ComplianceManagement = () => {
  const { t } = useTranslation('compliance');
  // Toast functions
  const { showSuccess, showError, showWarning, showInfo } = useToast();
  
  const [policies, setPolicies] = useState([]);
  const [legalHolds, setLegalHolds] = useState([]);
  const [auditLogs, setAuditLogs] = useState([]);
  const [dataErasureRequests, setDataErasureRequests] = useState([]);
  const [complianceStats, setComplianceStats] = useState({});
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState('dashboard');
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [showPolicyEditor, setShowPolicyEditor] = useState(false);
  const [showDataErasureModal, setShowDataErasureModal] = useState(false);
  const [selectedPolicy, setSelectedPolicy] = useState(null);
  const [filterType, setFilterType] = useState('all');
  const [searchTerm, setSearchTerm] = useState('');

  const tabs = [
    { id: 'dashboard', label: t('tabs.dashboard'), icon: BarChart3 },
    { id: 'policies', label: t('tabs.policies'), icon: Clock },
    { id: 'legal-holds', label: t('tabs.legalHolds'), icon: Gavel },
    { id: 'erasure', label: t('tabs.erasure'), icon: FileX },
    { id: 'audit', label: t('tabs.audit'), icon: FileText },
    { id: 'policy-editor', label: t('tabs.policyEditor'), icon: Code },
    { id: 'scheduler', label: t('tabs.scheduler'), icon: Settings }
  ];

  useEffect(() => {
    loadData();
  }, [activeTab]);

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      switch (activeTab) {
        case 'dashboard':
          await Promise.all([loadComplianceStats(), loadPolicies()]);
          break;
        case 'policies':
          await loadPolicies();
          break;
        case 'legal-holds':
          await loadLegalHolds();
          break;
        case 'erasure':
          await loadDataErasureRequests();
          break;
        case 'audit':
          await loadAuditLogs();
          break;
        default:
          break;
      }
    } catch (error) {
      showError(t('messages.loadDataFailed'));
    } finally {
      setLoading(false);
    }
  }, [activeTab, showError]);

  const loadComplianceStats = async () => {
    try {
      const data = await complianceAPI.getComplianceStats();
      setComplianceStats(data);
    } catch (error) {
      console.error('Failed to load compliance stats:', error);
      showError(t('messages.loadStatsFailed'));
    }
  };

  const loadPolicies = async () => {
    try {
      const data = await complianceAPI.getCompliancePolicies();
      setPolicies(data.policies || []);
    } catch (error) {
      console.error('Failed to load policies:', error);
      setPolicies([]);
      showError(t('messages.loadPoliciesFailed'));
    }
  };

  const loadLegalHolds = async () => {
    try {
      const data = await complianceAPI.getLegalHolds();
      setLegalHolds(data.legal_holds || []);
    } catch (error) {
      console.error('Failed to load legal holds:', error);
      setLegalHolds([]);
      showError(t('messages.loadLegalHoldsFailed'));
    }
  };

  const loadDataErasureRequests = async () => {
    try {
      const data = await complianceAPI.getDataErasureRequests();
      setDataErasureRequests(data.erasure_requests || []);
    } catch (error) {
      console.error('Failed to load data erasure requests:', error);
      setDataErasureRequests([]);
      showError(t('messages.loadErasureRequestsFailed'));
    }
  };

  const loadAuditLogs = async () => {
    try {
      const data = await complianceAPI.getComplianceAuditLogs({ limit: 100 });
      setAuditLogs(data.logs || []);
    } catch (error) {
      console.error('Failed to load audit logs:', error);
      setAuditLogs([]);
      showError(t('messages.loadAuditLogsFailed'));
    }
  };

  // Policy Management Handlers
  const handleCreatePolicy = async (policyData) => {
    try {
      await complianceAPI.createCompliancePolicy(policyData);
      showSuccess(t('messages.policyCreated'));
      setShowCreateModal(false);
      loadPolicies();
    } catch (error) {
      console.error('Failed to create policy:', error);
      showInfo(error.message || 'Failed to create compliance policy', 'error');
    }
  };

  const handleUpdatePolicy = async (policyId, policyData) => {
    try {
      await complianceAPI.updateCompliancePolicy(policyId, policyData);
      showSuccess(t('messages.policyUpdated'));
      loadPolicies();
    } catch (error) {
      console.error('Failed to update policy:', error);
      showInfo(error.message || 'Failed to update policy', 'error');
    }
  };

  const handleDeletePolicy = async (policyId) => {
    if (!window.confirm(t('messages.confirmDeletePolicy'))) return;
    
    try {
      await complianceAPI.deleteCompliancePolicy(policyId);
      showSuccess(t('messages.policyDeleted'));
      loadPolicies();
    } catch (error) {
      console.error('Failed to delete policy:', error);
      showInfo(error.message || 'Failed to delete policy', 'error');
    }
  };

  const handleEnforceRetention = async () => {
    try {
      const result = await complianceAPI.applyRetentionPolicies();
      showInfo(`Retention enforcement completed. ${result.processed || 0} artifacts processed.`, 'success');
      loadComplianceStats();
    } catch (error) {
      console.error('Failed to enforce retention policies:', error);
      showInfo(error.message || 'Failed to enforce retention policies', 'error');
    }
  };

  // Legal Hold Handlers
  const handleCreateLegalHold = async (holdData) => {
    try {
      await complianceAPI.createLegalHold(holdData);
      showSuccess(t('messages.legalHoldCreated'));
      loadLegalHolds();
    } catch (error) {
      console.error('Failed to create legal hold:', error);
      showInfo(error.message || 'Failed to create legal hold', 'error');
    }
  };

  const handleReleaseLegalHold = async (holdId) => {
    if (!window.confirm(t('messages.confirmReleaseLegalHold'))) return;
    
    try {
      const response = await fetch(`/api/compliance/legal-holds/${holdId}/release`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        }
      });

      if (response.ok) {
        showSuccess(t('messages.legalHoldReleased'));
        loadLegalHolds();
      } else {
        const error = await response.json();
        showError(error.message || t('messages.releaseLegalHoldFailed'));
      }
    } catch (error) {
      showError(t('messages.releaseLegalHoldFailed'));
    }
  };

  // Data Erasure Handlers  
  const handleCreateErasureRequest = async (requestData) => {
    try {
      await complianceAPI.createDataErasureRequest(requestData);
      showSuccess(t('messages.erasureRequestCreated'));
      setShowDataErasureModal(false);
      loadDataErasureRequests();
    } catch (error) {
      console.error('Failed to create erasure request:', error);
      showInfo(error.message || 'Failed to create data erasure request', 'error');
    }
  };

  const handleApproveErasureRequest = async (requestId) => {
    try {
      await complianceAPI.approveDataErasureRequest(requestId);
      showSuccess(t('messages.erasureRequestApproved'));
      loadDataErasureRequests();
    } catch (error) {
      console.error('Failed to approve erasure request:', error);
      showInfo(error.message || 'Failed to approve erasure request', 'error');
    }
  };

  const handleRejectErasureRequest = async (requestId, reason) => {
    try {
      await complianceAPI.rejectDataErasureRequest(requestId, reason);
      showSuccess(t('messages.erasureRequestRejected'));
      loadDataErasureRequests();
    } catch (error) {
      console.error('Failed to reject erasure request:', error);
      showInfo(error.message || 'Failed to reject erasure request', 'error');
    }
  };

  // Utility functions
  const formatDate = (dateString) => {
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  };

  const getStatusColor = (status) => {
    const colors = {
      active: 'bg-green-100 text-green-800',
      inactive: 'bg-red-100 text-red-800',
      draft: 'bg-yellow-100 text-yellow-800',
      pending: 'bg-blue-100 text-blue-800',
      approved: 'bg-green-100 text-green-800',
      rejected: 'bg-red-100 text-red-800',
      completed: 'bg-gray-100 text-gray-800'
    };
    return colors[status] || 'bg-gray-100 text-gray-800';
  };

  // Dashboard Components
  const ComplianceDashboard = () => (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold text-gray-900">Compliance Status Dashboard</h2>
        <button
          onClick={loadData}
          className="flex items-center space-x-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition"
        >
          <RefreshCw className="w-4 h-4" />
          <span>Refresh</span>
        </button>
      </div>

      {/* Key Metrics */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <div className="bg-white rounded-lg border border-gray-200 p-6">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-3">
              <div className="p-2 bg-green-100 rounded-lg">
                <CheckCircle className="w-6 h-6 text-green-600" />
              </div>
              <div>
                <p className="text-sm font-medium text-gray-500">Active Policies</p>
                <p className="text-2xl font-bold text-gray-900">{complianceStats.activePolicies || 0}</p>
              </div>
            </div>
          </div>
        </div>

        <div className="bg-white rounded-lg border border-gray-200 p-6">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-3">
              <div className="p-2 bg-yellow-100 rounded-lg">
                <AlertTriangle className="w-6 h-6 text-yellow-600" />
              </div>
              <div>
                <p className="text-sm font-medium text-gray-500">Expiring Soon</p>
                <p className="text-2xl font-bold text-gray-900">{complianceStats.expiringSoon || 0}</p>
              </div>
            </div>
          </div>
        </div>

        <div className="bg-white rounded-lg border border-gray-200 p-6">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-3">
              <div className="p-2 bg-purple-100 rounded-lg">
                <Gavel className="w-6 h-6 text-purple-600" />
              </div>
              <div>
                <p className="text-sm font-medium text-gray-500">Legal Holds</p>
                <p className="text-2xl font-bold text-gray-900">{complianceStats.activeLegalHolds || 0}</p>
              </div>
            </div>
          </div>
        </div>

        <div className="bg-white rounded-lg border border-gray-200 p-6">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-3">
              <div className="p-2 bg-red-100 rounded-lg">
                <FileX className="w-6 h-6 text-red-600" />
              </div>
              <div>
                <p className="text-sm font-medium text-gray-500">Erasure Requests</p>
                <p className="text-2xl font-bold text-gray-900">{complianceStats.pendingErasureRequests || 0}</p>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Recent Activities */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-white rounded-lg border border-gray-200 p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">Recent Policy Changes</h3>
          <div className="space-y-3">
            {policies.slice(0, 5).map((policy) => (
              <div key={policy.id} className="flex items-center justify-between py-2 border-b border-gray-100 last:border-0">
                <div className="flex items-center space-x-3">
                  <Clock className="w-4 h-4 text-gray-400" />
                  <div>
                    <p className="text-sm font-medium text-gray-900">{policy.name}</p>
                    <p className="text-xs text-gray-500">Updated {formatDate(policy.updated_at)}</p>
                  </div>
                </div>
                <span className={`px-2 py-1 text-xs font-medium rounded ${getStatusColor(policy.status)}`}>
                  {policy.status}
                </span>
              </div>
            ))}
          </div>
        </div>

        <div className="bg-white rounded-lg border border-gray-200 p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">Compliance Trends</h3>
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <span className="text-sm text-gray-500">Retention Compliance</span>
              <div className="flex items-center space-x-2">
                <div className="w-20 bg-gray-200 rounded-full h-2">
                  <div className="bg-green-600 h-2 rounded-full" style={{width: '85%'}}></div>
                </div>
                <span className="text-sm font-medium text-gray-900">85%</span>
              </div>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-sm text-gray-500">Data Locality</span>
              <div className="flex items-center space-x-2">
                <div className="w-20 bg-gray-200 rounded-full h-2">
                  <div className="bg-blue-600 h-2 rounded-full" style={{width: '92%'}}></div>
                </div>
                <span className="text-sm font-medium text-gray-900">92%</span>
              </div>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-sm text-gray-500">Audit Coverage</span>
              <div className="flex items-center space-x-2">
                <div className="w-20 bg-gray-200 rounded-full h-2">
                  <div className="bg-purple-600 h-2 rounded-full" style={{width: '78%'}}></div>
                </div>
                <span className="text-sm font-medium text-gray-900">78%</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );

  const PolicyCard = ({ policy }) => (
    <div className="bg-white rounded-lg border border-gray-200 p-6 hover:shadow-md transition-shadow">
      <div className="flex items-start justify-between mb-4">
        <div>
          <h3 className="text-lg font-semibold text-gray-900">{policy.name}</h3>
          <p className="text-sm text-gray-500 capitalize">{policy.type?.replace('_', ' ')}</p>
        </div>
        <span className={`px-2 py-1 text-xs font-medium rounded ${getStatusColor(policy.status)}`}>
          {policy.status}
        </span>
      </div>
      
      <p className="text-sm text-gray-600 mb-4">{policy.description}</p>
      
      <div className="flex items-center justify-between">
        <div className="text-xs text-gray-500">
          Region: <span className="font-medium">{policy.region || 'Global'}</span>
        </div>
        <div className="flex space-x-2">
          <button 
            onClick={() => setSelectedPolicy(policy)}
            className="p-1 text-gray-400 hover:text-blue-600 transition"
            title="Edit Policy"
          >
            <Edit3 className="w-4 h-4" />
          </button>
          <button 
            onClick={() => handleDeletePolicy(policy.id)}
            className="p-1 text-gray-400 hover:text-red-600 transition"
            title="Delete Policy"
          >
            <Trash2 className="w-4 h-4" />
          </button>
        </div>
      </div>
    </div>
  );

  // Data Retention Policies Tab
  const DataRetentionTab = () => (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-semibold text-gray-900">Data Retention Policies</h2>
          <p className="text-gray-500">Manage automatic data retention and deletion policies per tenant</p>
        </div>
        <div className="flex space-x-3">
          <button
            onClick={handleEnforceRetention}
            className="flex items-center space-x-2 px-4 py-2 bg-orange-600 text-white rounded-lg hover:bg-orange-700 transition"
          >
            <Activity className="w-4 h-4" />
            <span>Apply Retention</span>
          </button>
          <button
            onClick={() => setShowCreateModal(true)}
            className="flex items-center space-x-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition"
          >
            <Plus className="w-4 h-4" />
            <span>New Policy</span>
          </button>
        </div>
      </div>

      {/* Filter and Search */}
      <div className="flex items-center space-x-4 bg-gray-50 p-4 rounded-lg">
        <div className="flex items-center space-x-2">
          <Filter className="w-4 h-4 text-gray-400" />
          <select 
            value={filterType} 
            onChange={(e) => setFilterType(e.target.value)}
            className="border-0 bg-transparent focus:ring-0 text-sm"
          >
            <option value="all">All Policies</option>
            <option value="active">Active</option>
            <option value="draft">Draft</option>
            <option value="inactive">Inactive</option>
          </select>
        </div>
        <div className="flex items-center space-x-2 flex-1">
          <Search className="w-4 h-4 text-gray-400" />
          <input
            type="text"
            placeholder="Search policies..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="border-0 bg-transparent focus:ring-0 text-sm flex-1"
          />
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {policies
          .filter(policy => {
            const matchesFilter = filterType === 'all' || policy.status === filterType;
            const matchesSearch = policy.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
                                 policy.description?.toLowerCase().includes(searchTerm.toLowerCase());
            return matchesFilter && matchesSearch;
          })
          .map(policy => (
            <PolicyCard key={policy.id} policy={policy} />
          ))}
      </div>

      {policies.length === 0 && !loading && (
        <div className="text-center py-12">
          <Clock className="w-12 h-12 text-gray-400 mx-auto mb-4" />
          <h3 className="text-lg font-medium text-gray-900 mb-2">No Retention Policies</h3>
          <p className="text-gray-500 mb-4">Create your first data retention policy to get started.</p>
          <button
            onClick={() => setShowCreateModal(true)}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition"
          >
            Create Policy
          </button>
        </div>
      )}
    </div>
  );

  // Legal Holds Tab
  const LegalHoldsTab = () => (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-semibold text-gray-900">Legal Holds Management</h2>
          <p className="text-gray-500">Prevent deletion of artifacts under legal investigation</p>
        </div>
        <button
          onClick={() => setShowCreateModal(true)}
          className="flex items-center space-x-2 px-4 py-2 bg-purple-600 text-white rounded-lg hover:bg-purple-700 transition"
        >
          <Plus className="w-4 h-4" />
          <span>New Legal Hold</span>
        </button>
      </div>

      <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
        <div className="px-6 py-4 border-b border-gray-200">
          <h3 className="text-lg font-medium text-gray-900">Active Legal Holds</h3>
        </div>
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Case Number</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Artifact</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Reason</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Created</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Actions</th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {legalHolds.map((hold) => (
                <tr key={hold.id}>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div className="font-medium text-gray-900">{hold.case_number}</div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div className="text-sm text-gray-500">Artifact #{hold.artifact_id}</div>
                  </td>
                  <td className="px-6 py-4">
                    <div className="text-sm text-gray-900 max-w-xs truncate">{hold.reason}</div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {formatDate(hold.created_at)}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className={`px-2 py-1 text-xs font-medium rounded ${getStatusColor(hold.status)}`}>
                      {hold.status}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-medium">
                    {hold.status === 'active' && (
                      <button
                        onClick={() => handleReleaseLegalHold(hold.id)}
                        className="text-red-600 hover:text-red-900"
                      >
                        Release Hold
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {legalHolds.length === 0 && (
          <div className="text-center py-8">
            <Gavel className="w-8 h-8 text-gray-400 mx-auto mb-3" />
            <p className="text-gray-500">No active legal holds</p>
          </div>
        )}
      </div>
    </div>
  );

  // Data Erasure Requests Tab
  const DataErasureTab = () => (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-semibold text-gray-900">Data Erasure Requests</h2>
          <p className="text-gray-500">GDPR-compliant data erasure request management</p>
        </div>
        <button
          onClick={() => setShowDataErasureModal(true)}
          className="flex items-center space-x-2 px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition"
        >
          <FileX className="w-4 h-4" />
          <span>New Erasure Request</span>
        </button>
      </div>

      <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
        <div className="px-6 py-4 border-b border-gray-200">
          <h3 className="text-lg font-medium text-gray-900">Erasure Requests</h3>
        </div>
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Request ID</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Data Subject</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Artifacts</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Requested</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Actions</th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {dataErasureRequests.map((request) => (
                <tr key={request.id}>
                  <td className="px-6 py-4 whitespace-nowrap font-medium text-gray-900">
                    #{request.id}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {request.data_subject}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {request.artifact_count} artifacts
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {formatDate(request.created_at)}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className={`px-2 py-1 text-xs font-medium rounded ${getStatusColor(request.status)}`}>
                      {request.status}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-medium">
                    {request.status === 'pending' && (
                      <button
                        onClick={() => handleApproveErasureRequest(request.id)}
                        className="text-green-600 hover:text-green-900 mr-4"
                      >
                        Approve
                      </button>
                    )}
                    <button className="text-blue-600 hover:text-blue-900">
                      View Details
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {dataErasureRequests.length === 0 && (
          <div className="text-center py-8">
            <FileX className="w-8 h-8 text-gray-400 mx-auto mb-3" />
            <p className="text-gray-500">No data erasure requests</p>
          </div>
        )}
      </div>
    </div>
  );

  // Audit Logs Tab
  const AuditLogsTab = () => (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-semibold text-gray-900">Compliance Audit Logs</h2>
          <p className="text-gray-500">Complete audit trail of all compliance activities</p>
        </div>
        <button
          onClick={() => loadAuditLogs()}
          className="flex items-center space-x-2 px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 transition"
        >
          <RefreshCw className="w-4 h-4" />
          <span>Refresh</span>
        </button>
      </div>

      <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
        <div className="px-6 py-4 border-b border-gray-200">
          <h3 className="text-lg font-medium text-gray-900">Recent Audit Events</h3>
        </div>
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Timestamp</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Event Type</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Resource</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">User</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Action</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Result</th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {auditLogs.map((log) => (
                <tr key={log.id}>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {formatDate(log.timestamp)}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className="px-2 py-1 text-xs font-medium bg-blue-100 text-blue-800 rounded">
                      {log.event_type}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                    {log.resource_type} #{log.resource_id}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div className="flex items-center">
                      <User className="w-4 h-4 text-gray-400 mr-2" />
                      <span className="text-sm text-gray-900">{log.user_id}</span>
                    </div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {log.action}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className={`px-2 py-1 text-xs font-medium rounded ${
                      log.success ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'
                    }`}>
                      {log.success ? 'Success' : 'Failed'}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {auditLogs.length === 0 && (
          <div className="text-center py-8">
            <FileText className="w-8 h-8 text-gray-400 mx-auto mb-3" />
            <p className="text-gray-500">No audit logs available</p>
          </div>
        )}
      </div>
    </div>
  );

  // Policy Editor Tab (OPA Rego Editor)
  const PolicyEditorTab = () => (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-semibold text-gray-900">OPA Policy Editor</h2>
          <p className="text-gray-500">Create and test Open Policy Agent (Rego) policies</p>
        </div>
        <div className="flex space-x-3">
          <button
            onClick={() => setShowPolicyEditor(true)}
            className="flex items-center space-x-2 px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition"
          >
            <Play className="w-4 h-4" />
            <span>Test Policy</span>
          </button>
          <button className="flex items-center space-x-2 px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 transition">
            <Save className="w-4 h-4" />
            <span>Save Policy</span>
          </button>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-white rounded-lg border border-gray-200 p-6">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-medium text-gray-900">Policy Editor</h3>
            <Code className="w-5 h-5 text-gray-400" />
          </div>
          <div className="relative">
            <textarea
              className="w-full h-96 p-4 border border-gray-300 rounded-lg font-mono text-sm focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
              placeholder="# Enter your OPA Rego policy here
package compliance.retention

default allow = false

allow {
    input.artifact.type in [&quot;docker&quot;, &quot;maven&quot;, &quot;npm&quot;]
    input.artifact.age_days < 365
    not has_legal_hold
}

has_legal_hold {
    input.artifact.legal_holds[_].status == &quot;active&quot;
}"
            />
          </div>
        </div>

        <div className="bg-white rounded-lg border border-gray-200 p-6">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-medium text-gray-900">Test Input & Results</h3>
            <BookOpen className="w-5 h-5 text-gray-400" />
          </div>
          
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">Test Input JSON</label>
              <textarea
                className="w-full h-32 p-3 border border-gray-300 rounded-lg font-mono text-xs focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
                placeholder='{
  "artifact": {
    "id": 123,
    "type": "docker",
    "age_days": 400,
    "legal_holds": []
  }
}'
              />
            </div>
            
            <div className="border-t pt-4">
              <label className="block text-sm font-medium text-gray-700 mb-2">Policy Result</label>
              <div className="p-3 bg-gray-50 rounded-lg">
                <div className="flex items-center space-x-2">
                  <div className="w-3 h-3 bg-red-500 rounded-full"></div>
                  <span className="text-sm font-medium text-gray-900">Policy Evaluation: DENY</span>
                </div>
                <p className="text-xs text-gray-500 mt-1">Artifact exceeds retention period (400 days &gt; 365 days)</p>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div className="bg-white rounded-lg border border-gray-200 p-6">
        <h3 className="text-lg font-medium text-gray-900 mb-4">Policy Templates</h3>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div className="p-4 border border-gray-200 rounded-lg hover:border-indigo-300 cursor-pointer transition">
            <h4 className="font-medium text-gray-900">Data Retention Policy</h4>
            <p className="text-sm text-gray-500 mt-1">Basic retention policy template</p>
          </div>
          <div className="p-4 border border-gray-200 rounded-lg hover:border-indigo-300 cursor-pointer transition">
            <h4 className="font-medium text-gray-900">GDPR Compliance</h4>
            <p className="text-sm text-gray-500 mt-1">GDPR compliance policy template</p>
          </div>
          <div className="p-4 border border-gray-200 rounded-lg hover:border-indigo-300 cursor-pointer transition">
            <h4 className="font-medium text-gray-900">Legal Hold Policy</h4>
            <p className="text-sm text-gray-500 mt-1">Legal hold enforcement template</p>
          </div>
        </div>
      </div>
    </div>
  );

  // Main Render
  return (
    <div className="min-h-screen bg-gray-50">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="mb-8">
          <h1 className="text-3xl font-bold text-gray-900">{t('page.title')}</h1>
          <p className="mt-2 text-gray-600">{t('page.description')}</p>
        </div>

        {/* Tab Navigation */}
        <div className="border-b border-gray-200 mb-8">
          <nav className="-mb-px flex space-x-8">
            {tabs.map((tab) => {
              const Icon = tab.icon;
              return (
                <button
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id)}
                  className={`flex items-center space-x-2 py-2 px-1 border-b-2 font-medium text-sm whitespace-nowrap ${
                    activeTab === tab.id
                      ? 'border-blue-500 text-blue-600'
                      : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                  }`}
                >
                  <Icon className="w-5 h-5" />
                  <span>{tab.label}</span>
                </button>
              );
            })}
          </nav>
        </div>

        {/* Tab Content */}
        {loading ? (
          <div className="flex justify-center items-center py-12">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
          </div>
        ) : (
          <div className="tab-content">
            {activeTab === 'dashboard' && <ComplianceDashboard />}
            {activeTab === 'policies' && <DataRetentionTab />}
            {activeTab === 'legal-holds' && <LegalHoldsTab />}
            {activeTab === 'erasure' && <DataErasureTab />}
            {activeTab === 'audit' && <AuditLogsTab />}
            {activeTab === 'policy-editor' && <PolicyEditorTab />}
            {activeTab === 'scheduler' && <SchedulerManagement />}
          </div>
        )}

        {/* Modal placeholder - TODO: Implement CreatePolicyModal */}
      </div>
    </div>
  );
};

export default ComplianceManagement;