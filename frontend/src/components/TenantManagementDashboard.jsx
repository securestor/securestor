import React, { useState, useEffect } from 'react';
import {
  Building2,
  Plus,
  Search,
  Filter,
  Eye,
  Edit,
  Settings,
  Trash2,
  Users,
  Activity,
  Database,
  MoreVertical
} from 'lucide-react';
import { tenantApi } from '../services/tenantApi';
import { CreateTenantModal } from './modals';
import { useToast } from '../context/ToastContext';
import TenantSettingsEnterprise from './TenantSettingsEnterprise';

const TenantManagementDashboard = () => {
  const [tenants, setTenants] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [filterStatus, setFilterStatus] = useState('all');
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [selectedTenant, setSelectedTenant] = useState(null);
  const [showTenantSettings, setShowTenantSettings] = useState(false);
  const [settingsTenantId, setSettingsTenantId] = useState(null);
  const [showEditModal, setShowEditModal] = useState(false);
  const [editTenant, setEditTenant] = useState(null);
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [deleteTenant, setDeleteTenant] = useState(null);
  const [stats, setStats] = useState({});
  const [pagination, setPagination] = useState({
    page: 1,
    limit: 10,
    total: 0,
    totalPages: 0
  });

  const { showSuccess, showError } = useToast();

  useEffect(() => {
    fetchData();
  }, [pagination.page, filterStatus, searchTerm]);

  const fetchData = async () => {
    try {
      setLoading(true);
      
      const [tenantsResponse, statsResponse] = await Promise.all([
        tenantApi.getTenants({
          page: pagination.page,
          limit: pagination.limit,
          search: searchTerm,
          status: filterStatus !== 'all' ? filterStatus : undefined
        }),
        tenantApi.getTenantStats().catch(() => ({ total: 0, active: 0, inactive: 0 }))
      ]);


      // Safely extract tenants array
      const tenants = tenantsResponse?.tenants || [];
      const total = tenantsResponse?.total || 0;
      
      setTenants(tenants);
      setStats(statsResponse || {});
      
      setPagination(prev => ({
        ...prev,
        total: total,
        totalPages: Math.ceil(total / prev.limit)
      }));
    } catch (err) {
      setError('Failed to fetch tenant data');
      console.error('Error fetching tenant data:', err);
      showError('Failed to fetch tenant data: ' + err.message);
    } finally {
      setLoading(false);
    }
  };

  const handleCreateTenant = async (tenantData) => {
    try {
      await tenantApi.createTenant(tenantData);
      showSuccess('Tenant created successfully!');
      fetchData(); // Refresh the data
      setShowCreateModal(false);
    } catch (err) {
      console.error('Failed to create tenant:', err);
      showError('Failed to create tenant: ' + err.message);
    }
  };

  const handleStatusChange = async (tenantId, isActive) => {
    try {
      await tenantApi.setTenantStatus(tenantId, isActive);
      showSuccess(`Tenant ${isActive ? 'activated' : 'deactivated'} successfully!`);
      fetchData();
    } catch (err) {
      showError('Failed to update tenant status: ' + err.message);
    }
  };

  const handleDeleteTenant = (tenant) => {
    setDeleteTenant(tenant);
    setShowDeleteModal(true);
  };

  const confirmDeleteTenant = async () => {
    if (!deleteTenant) return;

    try {
      await tenantApi.deleteTenant(deleteTenant.id);
      showSuccess('Tenant deleted successfully!');
      setShowDeleteModal(false);
      setDeleteTenant(null);
      fetchData();
    } catch (err) {
      showError('Failed to delete tenant: ' + err.message);
    }
  };

  const handleViewTenant = (tenant) => {
    setSelectedTenant(tenant);
  };

  const handleTenantSettings = (tenant) => {
    setSettingsTenantId(tenant.id);
    setShowTenantSettings(true);
  };

  const handleEditTenant = (tenant) => {
    setEditTenant(tenant);
    setShowEditModal(true);
  };

  const handleSaveEditTenant = async (updatedData) => {
    if (!editTenant) return;

    try {
      await tenantApi.updateTenant(editTenant.id, updatedData);
      showSuccess('Tenant updated successfully!');
      setShowEditModal(false);
      setEditTenant(null);
      fetchData();
    } catch (err) {
      showError('Failed to update tenant: ' + err.message);
    }
  };

  const getStatusBadge = (isActive) => (
    <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
      isActive ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'
    }`}>
      {isActive ? 'Active' : 'Inactive'}
    </span>
  );

  const getPlanBadge = (plan) => {
    const planColors = {
      free: 'bg-gray-100 text-gray-800',
      basic: 'bg-blue-100 text-blue-800',
      premium: 'bg-purple-100 text-purple-800',
      enterprise: 'bg-amber-100 text-amber-800'
    };

    return (
      <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
        planColors[plan] || 'bg-gray-100 text-gray-800'
      }`}>
        {plan ? plan.charAt(0).toUpperCase() + plan.slice(1) : 'Basic'}
      </span>
    );
  };

  if (loading && tenants.length === 0) {
    return (
      <div className="flex items-center justify-center min-h-64">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
      </div>
    );
  }

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 flex items-center gap-2">
            <Building2 className="h-7 w-7 text-blue-600" />
            Tenant Management
          </h1>
          <p className="text-gray-600 mt-1">
            Manage organization tenants, their settings, and resource allocations
          </p>
        </div>
        <button
          onClick={() => setShowCreateModal(true)}
          className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg font-medium inline-flex items-center gap-2 transition-colors shadow-sm"
        >
          <Plus className="h-4 w-4" />
          Add Tenant
        </button>
      </div>

      {/* Stats Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <div className="bg-white rounded-lg border border-gray-200 p-6">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-gray-600">Total Tenants</p>
              <p className="text-2xl font-bold text-gray-900">{stats.total || tenants.length}</p>
            </div>
            <div className="p-2 bg-blue-50 rounded-lg">
              <Building2 className="h-6 w-6 text-blue-600" />
            </div>
          </div>
        </div>

        <div className="bg-white rounded-lg border border-gray-200 p-6">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-gray-600">Active Tenants</p>
              <p className="text-2xl font-bold text-green-700">{stats.active || tenants.filter(t => t.is_active).length}</p>
            </div>
            <div className="p-2 bg-green-50 rounded-lg">
              <Activity className="h-6 w-6 text-green-600" />
            </div>
          </div>
        </div>

        <div className="bg-white rounded-lg border border-gray-200 p-6">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-gray-600">Total Users</p>
              <p className="text-2xl font-bold text-purple-700">
                {tenants.reduce((sum, t) => sum + (t.current_users || 0), 0)}
              </p>
            </div>
            <div className="p-2 bg-purple-50 rounded-lg">
              <Users className="h-6 w-6 text-purple-600" />
            </div>
          </div>
        </div>

        <div className="bg-white rounded-lg border border-gray-200 p-6">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-gray-600">Storage Used</p>
              <p className="text-2xl font-bold text-orange-700">
                {tenants.reduce((sum, t) => sum + (t.storage_used_gb || 0), 0).toFixed(1)} GB
              </p>
            </div>
            <div className="p-2 bg-orange-50 rounded-lg">
              <Database className="h-6 w-6 text-orange-600" />
            </div>
          </div>
        </div>
      </div>

      {/* Filters */}
      <div className="bg-white rounded-lg border border-gray-200 p-4">
        <div className="flex flex-col sm:flex-row gap-4">
          <div className="flex-1">
            <div className="relative">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 h-4 w-4" />
              <input
                type="text"
                placeholder="Search tenants by name, subdomain, or email..."
                className="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
              />
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Filter className="h-4 w-4 text-gray-500" />
            <select
              className="border border-gray-300 rounded-lg px-3 py-2 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              value={filterStatus}
              onChange={(e) => setFilterStatus(e.target.value)}
            >
              <option value="all">All Status</option>
              <option value="active">Active</option>
              <option value="inactive">Inactive</option>
            </select>
          </div>
        </div>
      </div>

      {/* Error Message */}
      {error && (
        <div className="bg-red-50 border border-red-200 rounded-lg p-4">
          <p className="text-red-600">{error}</p>
          <button
            onClick={fetchData}
            className="mt-2 text-red-600 hover:text-red-800 underline text-sm"
          >
            Try again
          </button>
        </div>
      )}

      {/* Tenant Table */}
      <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
        {tenants.length === 0 ? (
          <div className="text-center py-12">
            <Building2 className="h-12 w-12 text-gray-400 mx-auto mb-4" />
            <h3 className="text-lg font-medium text-gray-900 mb-2">No tenants found</h3>
            <p className="text-gray-600 mb-4">
              {searchTerm ? 'No tenants match your search criteria.' : 'Get started by creating your first tenant.'}
            </p>
            <button
              onClick={() => setShowCreateModal(true)}
              className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg font-medium inline-flex items-center gap-2 transition-colors"
            >
              <Plus className="h-4 w-4" />
              Add Tenant
            </button>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Tenant
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Status
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Plan
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Users
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Storage
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Created
                  </th>
                  <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {tenants.map((tenant) => (
                  <tr key={tenant.id} className="hover:bg-gray-50 transition-colors">
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="flex items-center">
                        <div className="flex-shrink-0 h-10 w-10">
                          <div className="h-10 w-10 rounded-lg bg-gradient-to-br from-blue-400 to-blue-600 flex items-center justify-center">
                            <Building2 className="h-5 w-5 text-white" />
                          </div>
                        </div>
                        <div className="ml-4">
                          <div className="text-sm font-medium text-gray-900">
                            {tenant.name}
                          </div>
                          {tenant.subdomain && (
                            <div className="text-sm text-gray-500">
                              {tenant.subdomain}.securestor.com
                            </div>
                          )}
                          {tenant.contact_email && (
                            <div className="text-xs text-gray-400">
                              {tenant.contact_email}
                            </div>
                          )}
                        </div>
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      {getStatusBadge(tenant.is_active)}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      {getPlanBadge(tenant.plan)}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="flex items-center text-sm text-gray-900">
                        <Users className="h-4 w-4 text-gray-400 mr-2" />
                        <span className="font-medium">{tenant.current_users || 0}</span>
                        <span className="text-gray-500 ml-1">/ {tenant.max_users}</span>
                      </div>
                      {tenant.usage_percent && (
                        <div className="w-full bg-gray-200 rounded-full h-1.5 mt-1">
                          <div 
                            className={`h-1.5 rounded-full transition-all ${
                              tenant.usage_percent > 90 ? 'bg-red-500' : 
                              tenant.usage_percent > 75 ? 'bg-yellow-500' : 'bg-green-500'
                            }`}
                            style={{ width: `${Math.min(tenant.usage_percent, 100)}%` }}
                          ></div>
                        </div>
                      )}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="text-sm font-medium text-gray-900">
                        {tenant.storage_used_gb ? `${tenant.storage_used_gb.toFixed(1)} GB` : '0 GB'}
                      </div>
                      <div className="text-sm text-gray-500">
                        {tenant.artifacts_count || 0} artifacts
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      {tenant.created_at ? new Date(tenant.created_at).toLocaleDateString() : 'N/A'}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                      <div className="flex items-center justify-end gap-1">
                        <button
                          onClick={() => handleViewTenant(tenant)}
                          className="text-blue-600 hover:text-blue-900 p-2 rounded-lg hover:bg-blue-50 transition-colors"
                          title="View Details"
                        >
                          <Eye className="h-4 w-4" />
                        </button>
                        <button
                          onClick={() => handleTenantSettings(tenant)}
                          className="text-gray-600 hover:text-gray-900 p-2 rounded-lg hover:bg-gray-50 transition-colors"
                          title="Tenant Settings"
                        >
                          <Settings className="h-4 w-4" />
                        </button>
                        <button
                          onClick={() => handleEditTenant(tenant)}
                          className="text-green-600 hover:text-green-900 p-2 rounded-lg hover:bg-green-50 transition-colors"
                          title="Edit Tenant"
                        >
                          <Edit className="h-4 w-4" />
                        </button>
                        <button
                          onClick={() => handleDeleteTenant(tenant)}
                          className="text-red-600 hover:text-red-900 p-2 rounded-lg hover:bg-red-50 transition-colors"
                          title="Delete Tenant"
                        >
                          <Trash2 className="h-4 w-4" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Pagination */}
      {pagination.totalPages > 1 && (
        <div className="bg-white px-4 py-3 flex items-center justify-between border border-gray-200 rounded-lg">
          <div className="flex-1 flex justify-between sm:hidden">
            <button
              onClick={() => setPagination(prev => ({ ...prev, page: Math.max(1, prev.page - 1) }))}
              disabled={pagination.page === 1}
              className="relative inline-flex items-center px-4 py-2 border border-gray-300 text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              Previous
            </button>
            <button
              onClick={() => setPagination(prev => ({ ...prev, page: Math.min(prev.totalPages, prev.page + 1) }))}
              disabled={pagination.page === pagination.totalPages}
              className="ml-3 relative inline-flex items-center px-4 py-2 border border-gray-300 text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              Next
            </button>
          </div>
          <div className="hidden sm:flex-1 sm:flex sm:items-center sm:justify-between">
            <div>
              <p className="text-sm text-gray-700">
                Showing{' '}
                <span className="font-medium">
                  {(pagination.page - 1) * pagination.limit + 1}
                </span>{' '}
                to{' '}
                <span className="font-medium">
                  {Math.min(pagination.page * pagination.limit, pagination.total)}
                </span>{' '}
                of{' '}
                <span className="font-medium">{pagination.total}</span> results
              </p>
            </div>
            <div>
              <nav className="relative z-0 inline-flex rounded-md shadow-sm -space-x-px">
                <button
                  onClick={() => setPagination(prev => ({ ...prev, page: Math.max(1, prev.page - 1) }))}
                  disabled={pagination.page === 1}
                  className="relative inline-flex items-center px-2 py-2 rounded-l-md border border-gray-300 bg-white text-sm font-medium text-gray-500 hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  Previous
                </button>
                {[...Array(pagination.totalPages)].map((_, i) => {
                  const pageNum = i + 1;
                  const isCurrentPage = pageNum === pagination.page;
                  
                  if (pagination.totalPages <= 7 || 
                      pageNum === 1 || 
                      pageNum === pagination.totalPages ||
                      Math.abs(pageNum - pagination.page) <= 2) {
                    return (
                      <button
                        key={pageNum}
                        onClick={() => setPagination(prev => ({ ...prev, page: pageNum }))}
                        className={`relative inline-flex items-center px-4 py-2 border text-sm font-medium ${
                          isCurrentPage
                            ? 'z-10 bg-blue-50 border-blue-500 text-blue-600'
                            : 'bg-white border-gray-300 text-gray-500 hover:bg-gray-50'
                        }`}
                      >
                        {pageNum}
                      </button>
                    );
                  } else if (pageNum === pagination.page - 3 || pageNum === pagination.page + 3) {
                    return (
                      <span
                        key={pageNum}
                        className="relative inline-flex items-center px-4 py-2 border border-gray-300 bg-white text-sm font-medium text-gray-700"
                      >
                        ...
                      </span>
                    );
                  }
                  return null;
                })}
                <button
                  onClick={() => setPagination(prev => ({ ...prev, page: Math.min(prev.totalPages, prev.page + 1) }))}
                  disabled={pagination.page === pagination.totalPages}
                  className="relative inline-flex items-center px-2 py-2 rounded-r-md border border-gray-300 bg-white text-sm font-medium text-gray-500 hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  Next
                </button>
              </nav>
            </div>
          </div>
        </div>
      )}

      {/* Create Tenant Modal */}
      <CreateTenantModal
        isOpen={showCreateModal}
        onClose={() => setShowCreateModal(false)}
        onSubmit={handleCreateTenant}
      />

      {/* Tenant Settings Modal/View */}
      {showTenantSettings && (
        <div className="fixed inset-0 bg-black bg-opacity-50 z-50 flex items-center justify-center p-4">
          <div className="bg-white rounded-lg shadow-xl w-full max-w-6xl max-h-[90vh] overflow-hidden">
            <div className="flex items-center justify-between p-4 border-b border-gray-200">
              <h2 className="text-lg font-semibold text-gray-900">Tenant Settings</h2>
              <button
                onClick={() => setShowTenantSettings(false)}
                className="text-gray-400 hover:text-gray-600 transition-colors"
              >
                <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
            <div className="overflow-y-auto max-h-[calc(90vh-4rem)]">
              {settingsTenantId && (
                <TenantSettingsEnterprise 
                  tenantId={settingsTenantId} 
                  onClose={() => setShowTenantSettings(false)}
                />
              )}
            </div>
          </div>
        </div>
      )}

      {/* Tenant Details Modal */}
      {selectedTenant && (
        <div className="fixed inset-0 bg-black bg-opacity-50 z-50 flex items-center justify-center p-4">
          <div className="bg-white rounded-lg shadow-xl w-full max-w-2xl max-h-[90vh] overflow-hidden">
            <div className="flex items-center justify-between p-6 border-b border-gray-200">
              <h2 className="text-xl font-semibold text-gray-900">Tenant Details</h2>
              <button
                onClick={() => setSelectedTenant(null)}
                className="text-gray-400 hover:text-gray-600 transition-colors"
              >
                <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
            <div className="p-6 space-y-6 overflow-y-auto max-h-[calc(90vh-8rem)]">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="text-sm font-medium text-gray-600">Tenant Name</label>
                  <p className="text-gray-900">{selectedTenant.name}</p>
                </div>
                <div>
                  <label className="text-sm font-medium text-gray-600">Subdomain</label>
                  <p className="text-gray-900">{selectedTenant.subdomain || 'N/A'}</p>
                </div>
                <div>
                  <label className="text-sm font-medium text-gray-600">Plan</label>
                  <p>{getPlanBadge(selectedTenant.plan)}</p>
                </div>
                <div>
                  <label className="text-sm font-medium text-gray-600">Status</label>
                  <p>{getStatusBadge(selectedTenant.is_active)}</p>
                </div>
                <div>
                  <label className="text-sm font-medium text-gray-600">Users</label>
                  <p className="text-gray-900">{selectedTenant.current_users || 0} / {selectedTenant.max_users}</p>
                </div>
                <div>
                  <label className="text-sm font-medium text-gray-600">Storage Used</label>
                  <p className="text-gray-900">{selectedTenant.storage_used_gb ? `${selectedTenant.storage_used_gb.toFixed(1)} GB` : '0 GB'}</p>
                </div>
                <div>
                  <label className="text-sm font-medium text-gray-600">Created</label>
                  <p className="text-gray-900">{selectedTenant.created_at ? new Date(selectedTenant.created_at).toLocaleDateString() : 'N/A'}</p>
                </div>
                <div>
                  <label className="text-sm font-medium text-gray-600">Updated</label>
                  <p className="text-gray-900">{selectedTenant.updated_at ? new Date(selectedTenant.updated_at).toLocaleDateString() : 'N/A'}</p>
                </div>
              </div>
              {selectedTenant.features && selectedTenant.features.length > 0 && (
                <div>
                  <label className="text-sm font-medium text-gray-600">Features</label>
                  <div className="mt-1 flex flex-wrap gap-2">
                    {selectedTenant.features.map((feature, index) => (
                      <span key={index} className="px-2 py-1 bg-blue-100 text-blue-800 text-xs font-medium rounded-full">
                        {feature}
                      </span>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>
      )}

      {/* Edit Tenant Modal */}
      {showEditModal && editTenant && (
        <div className="fixed inset-0 bg-black bg-opacity-50 z-50 flex items-center justify-center p-4">
          <div className="bg-white rounded-lg shadow-xl w-full max-w-md">
            <div className="flex items-center justify-between p-6 border-b border-gray-200">
              <h2 className="text-xl font-semibold text-gray-900">Edit Tenant</h2>
              <button
                onClick={() => setShowEditModal(false)}
                className="text-gray-400 hover:text-gray-600 transition-colors"
              >
                <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
            <form onSubmit={(e) => {
              e.preventDefault();
              const formData = new FormData(e.target);
              handleSaveEditTenant({
                name: formData.get('name'),
                subdomain: formData.get('subdomain'),
                plan: formData.get('plan'),
                max_users: parseInt(formData.get('max_users')),
                is_active: formData.get('is_active') === 'true'
              });
            }}>
              <div className="p-6 space-y-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Tenant Name *
                  </label>
                  <input
                    type="text"
                    name="name"
                    defaultValue={editTenant.name}
                    required
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Subdomain
                  </label>
                  <input
                    type="text"
                    name="subdomain"
                    defaultValue={editTenant.subdomain || ''}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Plan *
                  </label>
                  <select
                    name="plan"
                    defaultValue={editTenant.plan}
                    required
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  >
                    <option value="basic">Basic</option>
                    <option value="premium">Premium</option>
                    <option value="enterprise">Enterprise</option>
                  </select>
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Max Users *
                  </label>
                  <input
                    type="number"
                    name="max_users"
                    defaultValue={editTenant.max_users}
                    min="1"
                    required
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Status
                  </label>
                  <select
                    name="is_active"
                    defaultValue={editTenant.is_active.toString()}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  >
                    <option value="true">Active</option>
                    <option value="false">Inactive</option>
                  </select>
                </div>
              </div>
              <div className="px-6 py-4 border-t border-gray-200 flex justify-end gap-3">
                <button
                  type="button"
                  onClick={() => setShowEditModal(false)}
                  className="px-4 py-2 text-gray-600 hover:text-gray-800 transition-colors"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
                >
                  Save Changes
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Delete Confirmation Modal */}
      {showDeleteModal && deleteTenant && (
        <div className="fixed inset-0 bg-black bg-opacity-50 z-50 flex items-center justify-center p-4">
          <div className="bg-white rounded-lg shadow-xl w-full max-w-md">
            <div className="p-6">
              <div className="flex items-center mb-4">
                <div className="flex-shrink-0 w-10 h-10 bg-red-100 rounded-full flex items-center justify-center">
                  <Trash2 className="w-5 h-5 text-red-600" />
                </div>
                <div className="ml-4">
                  <h3 className="text-lg font-medium text-gray-900">Delete Tenant</h3>
                  <p className="text-sm text-gray-600">This action cannot be undone.</p>
                </div>
              </div>
              <p className="text-gray-700 mb-6">
                Are you sure you want to delete <strong>{deleteTenant.name}</strong>? 
                All associated data, users, and settings will be permanently removed.
              </p>
              <div className="flex justify-end gap-3">
                <button
                  onClick={() => setShowDeleteModal(false)}
                  className="px-4 py-2 text-gray-600 hover:text-gray-800 transition-colors"
                >
                  Cancel
                </button>
                <button
                  onClick={confirmDeleteTenant}
                  className="px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors"
                >
                  Delete Tenant
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default TenantManagementDashboard;