import React, { useState, useEffect } from 'react';
import {
  UserPlus,
  Edit,
  Trash2,
  Mail,
  CheckCircle,
  XCircle,
  Search,
  MoreVertical,
  RefreshCw,
  UserX,
  X,
  AlertCircle,
  Check
} from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import { API_BASE_URL } from '../constants';

const UserManagementDashboard = () => {
  const { getAuthHeaders, user } = useAuth();
  // State management
  const [users, setUsers] = useState([]);
  const [totalCount, setTotalCount] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [success, setSuccess] = useState(null);

  // Pagination and filtering
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(25);
  const [searchTerm, setSearchTerm] = useState('');
  const [statusFilter, setStatusFilter] = useState('all');
  const [roleFilter, setRoleFilter] = useState('all');

  // Dialog states
  const [inviteDialogOpen, setInviteDialogOpen] = useState(false);
  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const [selectedUser, setSelectedUser] = useState(null);
  const [actionMenu, setActionMenu] = useState({ anchorEl: null, user: null });

  // Invite form state
  const [inviteForm, setInviteForm] = useState({
    email: '',
    firstName: '',
    lastName: '',
    roleIds: []
  });

  // Edit form state
  const [editForm, setEditForm] = useState({
    firstName: '',
    lastName: '',
    displayName: '',
    department: '',
    jobTitle: '',
    phone: '',
    timezone: 'UTC',
    language: 'en'
  });

  // Available roles - will be fetched from backend
  const [roles, setRoles] = useState([]);

  // Fetch users
  const fetchUsers = async () => {
    try {
      setLoading(true);
      const params = new URLSearchParams({
        limit: rowsPerPage.toString(),
        offset: (page * rowsPerPage).toString(),
        ...(searchTerm && { search: searchTerm }),
        ...(statusFilter !== 'all' && { is_active: statusFilter === 'active' }),
        ...(roleFilter !== 'all' && { role_id: roleFilter })
      });

      const response = await fetch(`${API_BASE_URL}/api/v1/users?${params}`, {
        headers: {
          'Content-Type': 'application/json',
          ...getAuthHeaders()
        }
      });
      const data = await response.json();

      if (response.ok) {
        setUsers(data.users || []);
        setTotalCount(data.total_count || 0);
        setError(null);
      } else {
        throw new Error(data.message || 'Failed to fetch users');
      }
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  // Fetch roles
  const fetchRoles = async () => {
    try {
      const response = await fetch(`${API_BASE_URL}/api/v1/roles`, {
        headers: {
          'Content-Type': 'application/json',
          ...getAuthHeaders()
        }
      });
      const data = await response.json();

      if (response.ok) {
        // Map backend response to frontend format
        const mappedRoles = (data.roles || []).map(role => ({
          id: role.role_id || role.id,
          name: role.name,
          displayName: role.display_name || role.name
        }));
        setRoles(mappedRoles);
      } else {
        console.error('Failed to fetch roles:', data.message);
      }
    } catch (err) {
      console.error('Error fetching roles:', err);
    }
  };

  // Invite user
  const handleInviteUser = async () => {
    // Validation
    if (!inviteForm.email.trim()) {
      setError('Email is required');
      return;
    }
    
    if (inviteForm.roleIds.length === 0) {
      setError('Please select at least one role');
      return;
    }

    // Ensure user is authenticated
    if (!user || !user.id || !user.tenant_id) {
      setError('User session invalid. Please log in again.');
      return;
    }

    const requestBody = {
      email: inviteForm.email,
      first_name: inviteForm.firstName || null,
      last_name: inviteForm.lastName || null,
      role_ids: inviteForm.roleIds,
      invited_by: user.id,
      tenant_id: user.tenant_id
    };

    try {
      const response = await fetch(`${API_BASE_URL}/api/v1/users/invite`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...getAuthHeaders()
        },
        body: JSON.stringify(requestBody),
      });
      
      if (response.ok) {
        setSuccess('User invitation sent successfully');
        setInviteDialogOpen(false);
        setInviteForm({ email: '', firstName: '', lastName: '', roleIds: [] });
        fetchUsers();
      } else {
        let errorMessage = 'Failed to invite user';
        try {
          const textResponse = await response.text();
          
          // Try to parse as JSON
          try {
            const data = JSON.parse(textResponse);
            errorMessage = data.message || data.error || errorMessage;
            
            // Add more context for specific status codes
            if (response.status === 409) {
              errorMessage = `${errorMessage}. Please check if the user already exists or has a pending invitation.`;
            }
          } catch (jsonErr) {
            // It's plain text
            errorMessage = textResponse || `Error ${response.status}: ${response.statusText}`;
          }
        } catch (e) {
          errorMessage = `Error ${response.status}: ${response.statusText}`;
        }
        throw new Error(errorMessage);
      }
    } catch (err) {
      setError(err.message);
    }
  };

  // Update user
  const handleUpdateUser = async () => {
    try {
      const response = await fetch(`${API_BASE_URL}/api/v1/users/${selectedUser.id}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          ...getAuthHeaders()
        },
        body: JSON.stringify(editForm),
      });

      if (response.ok) {
        setSuccess('User updated successfully');
        setEditDialogOpen(false);
        setSelectedUser(null);
        fetchUsers();
      } else {
        const data = await response.json();
        throw new Error(data.message || 'Failed to update user');
      }
    } catch (err) {
      setError(err.message);
    }
  };

  // Toggle user status
  const handleToggleUserStatus = async (user) => {
    try {
      const response = await fetch(`${API_BASE_URL}/api/v1/users/${user.id}`, {
        method: 'PATCH',
        headers: {
          'Content-Type': 'application/json',
          ...getAuthHeaders()
        },
        body: JSON.stringify({
          is_active: !user.is_active
        }),
      });

      if (response.ok) {
        setSuccess(`User ${user.is_active ? 'deactivated' : 'activated'} successfully`);
        fetchUsers();
      } else {
        const data = await response.json();
        throw new Error(data.message || 'Failed to update user status');
      }
    } catch (err) {
      setError(err.message);
    }
    setActionMenu({ anchorEl: null, user: null });
  };

  // Delete user
  const handleDeleteUser = async (user) => {
    if (!window.confirm(`Are you sure you want to delete user ${user.username}?`)) {
      return;
    }

    try {
      const response = await fetch(`${API_BASE_URL}/api/v1/users/${user.id}`, {
        method: 'DELETE',
        headers: {
          ...getAuthHeaders()
        }
      });

      if (response.ok) {
        setSuccess('User deleted successfully');
        fetchUsers();
      } else {
        const data = await response.json();
        throw new Error(data.message || 'Failed to delete user');
      }
    } catch (err) {
      setError(err.message);
    }
    setActionMenu({ anchorEl: null, user: null });
  };

  // Open edit dialog
  const handleEditUser = (user) => {
    setSelectedUser(user);
    setEditForm({
      firstName: user.first_name || '',
      lastName: user.last_name || '',
      displayName: user.display_name || '',
      department: user.department || '',
      jobTitle: user.job_title || '',
      phone: user.phone || '',
      timezone: user.timezone || 'UTC',
      language: user.language || 'en'
    });
    setEditDialogOpen(true);
    setActionMenu({ anchorEl: null, user: null });
  };

  // Handle search
  const handleSearch = (event) => {
    setSearchTerm(event.target.value);
    setPage(0);
  };

  // Handle page change
  const handleChangePage = (event, newPage) => {
    setPage(newPage);
  };

  // Handle rows per page change
  const handleChangeRowsPerPage = (event) => {
    setRowsPerPage(parseInt(event.target.value, 10));
    setPage(0);
  };

  // Get user status color
  const getUserStatusColor = (user) => {
    if (!user.is_active) return 'error';
    if (!user.is_email_verified) return 'warning';
    return 'success';
  };

  // Get user status label
  const getUserStatusLabel = (user) => {
    if (!user.is_active) return 'Inactive';
    if (!user.is_email_verified) return 'Unverified';
    return 'Active';
  };

  // Format last login
  const formatLastLogin = (lastLogin) => {
    if (!lastLogin) return 'Never';
    return new Date(lastLogin).toLocaleDateString();
  };

  // Effects
  useEffect(() => {
    fetchRoles(); // Fetch roles once on mount
  }, []);

  useEffect(() => {
    fetchUsers();
  }, [page, rowsPerPage, searchTerm, statusFilter, roleFilter]);

  // Clear messages after 5 seconds
  useEffect(() => {
    if (error || success) {
      const timer = setTimeout(() => {
        setError(null);
        setSuccess(null);
      }, 5000);
      return () => clearTimeout(timer);
    }
  }, [error, success]);

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold text-gray-900 dark:text-white">User Management</h1>
          <p className="text-gray-600 dark:text-gray-400 mt-1">Manage users, roles, and permissions</p>
        </div>
        <button
          onClick={() => setInviteDialogOpen(true)}
          className="inline-flex items-center px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white font-medium rounded-lg transition-colors duration-200"
        >
          <UserPlus className="w-5 h-5 mr-2" />
          Invite User
        </button>
      </div>

      {/* Alerts */}
      {error && (
        <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4 flex items-center justify-between">
          <div className="flex items-center">
            <AlertCircle className="w-5 h-5 text-red-500 mr-3" />
            <span className="text-red-800 dark:text-red-200">{error}</span>
          </div>
          <button onClick={() => setError(null)} className="text-red-500 hover:text-red-700">
            <X className="w-5 h-5" />
          </button>
        </div>
      )}
      {success && (
        <div className="bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg p-4 flex items-center justify-between">
          <div className="flex items-center">
            <Check className="w-5 h-5 text-green-500 mr-3" />
            <span className="text-green-800 dark:text-green-200">{success}</span>
          </div>
          <button onClick={() => setSuccess(null)} className="text-green-500 hover:text-green-700">
            <X className="w-5 h-5" />
          </button>
        </div>
      )}

      {/* Filters */}
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 p-6">
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4 items-end">
          <div className="md:col-span-1">
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Search Users
            </label>
            <div className="relative">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-5 h-5 text-gray-400" />
              <input
                type="text"
                placeholder="Search users..."
                value={searchTerm}
                onChange={handleSearch}
                className="w-full pl-10 pr-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-white"
              />
            </div>
          </div>
          
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Status
            </label>
            <select
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-white"
            >
              <option value="all">All Status</option>
              <option value="active">Active</option>
              <option value="inactive">Inactive</option>
            </select>
          </div>
          
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Role
            </label>
            <select
              value={roleFilter}
              onChange={(e) => setRoleFilter(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-white"
            >
              <option value="all">All Roles</option>
              {roles.map((role) => (
                <option key={role.id} value={role.id}>
                  {role.displayName}
                </option>
              ))}
            </select>
          </div>
          
          <div>
            <button
              onClick={fetchUsers}
              className="w-full inline-flex items-center justify-center px-4 py-2 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-700 hover:bg-gray-50 dark:hover:bg-gray-600 rounded-lg transition-colors duration-200"
            >
              <RefreshCw className="w-4 h-4 mr-2" />
              Refresh
            </button>
          </div>
        </div>
      </div>

      {/* Users Table */}
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 overflow-hidden">
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
            <thead className="bg-gray-50 dark:bg-gray-700">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                  User
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                  Email
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                  Roles
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                  Status
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                  Last Login
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                  Created
                </th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
              {loading ? (
                <tr>
                  <td colSpan={7} className="px-6 py-12 text-center">
                    <div className="flex justify-center">
                      <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
                    </div>
                  </td>
                </tr>
              ) : users.length === 0 ? (
                <tr>
                  <td colSpan={7} className="px-6 py-12 text-center text-gray-500 dark:text-gray-400">
                    No users found
                  </td>
                </tr>
              ) : (
                users.map((user) => (
                  <tr key={user.id} className="hover:bg-gray-50 dark:hover:bg-gray-700">
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="flex items-center">
                        <div className="h-10 w-10 rounded-full bg-blue-100 dark:bg-blue-900 flex items-center justify-center">
                          <span className="text-sm font-medium text-blue-800 dark:text-blue-200">
                            {user.display_name ? user.display_name.charAt(0) : user.username.charAt(0)}
                          </span>
                        </div>
                        <div className="ml-4">
                          <div className="text-sm font-medium text-gray-900 dark:text-white">
                            {user.display_name || `${user.first_name || ''} ${user.last_name || ''}`.trim() || user.username}
                          </div>
                          <div className="text-sm text-gray-500 dark:text-gray-400">
                            @{user.username}
                          </div>
                        </div>
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="flex items-center">
                        <span className="text-sm text-gray-900 dark:text-white">{user.email}</span>
                        {!user.is_email_verified && (
                          <Mail className="w-4 h-4 text-orange-500 ml-2" title="Email not verified" />
                        )}
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="flex flex-wrap gap-1">
                        {user.roles?.map((role) => (
                          <span
                            key={role.id}
                            className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200"
                          >
                            {role.display_name || role.name}
                          </span>
                        )) || (
                          <span className="text-sm text-gray-500 dark:text-gray-400">No roles</span>
                        )}
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span
                        className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                          getUserStatusColor(user) === 'success'
                            ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
                            : getUserStatusColor(user) === 'warning'
                            ? 'bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200'
                            : 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200'
                        }`}
                      >
                        {user.is_active ? (
                          <CheckCircle className="w-3 h-3 mr-1" />
                        ) : (
                          <XCircle className="w-3 h-3 mr-1" />
                        )}
                        {getUserStatusLabel(user)}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-white">
                      {formatLastLogin(user.last_login_at)}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-white">
                      {new Date(user.created_at).toLocaleDateString()}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                      <button
                        onClick={(e) => setActionMenu({ anchorEl: e.currentTarget, user })}
                        className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-200"
                      >
                        <MoreVertical className="w-5 h-5" />
                      </button>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
        
        {/* Pagination */}
        <div className="bg-white dark:bg-gray-800 px-4 py-3 flex items-center justify-between border-t border-gray-200 dark:border-gray-700 sm:px-6">
          <div className="flex-1 flex justify-between items-center">
            <div>
              <p className="text-sm text-gray-700 dark:text-gray-300">
                Showing <span className="font-medium">{page * rowsPerPage + 1}</span> to{' '}
                <span className="font-medium">{Math.min((page + 1) * rowsPerPage, totalCount)}</span> of{' '}
                <span className="font-medium">{totalCount}</span> results
              </p>
            </div>
            <div className="flex items-center space-x-2">
              <select
                value={rowsPerPage}
                onChange={handleChangeRowsPerPage}
                className="px-3 py-1 border border-gray-300 dark:border-gray-600 rounded text-sm dark:bg-gray-700 dark:text-white"
              >
                <option value={10}>10</option>
                <option value={25}>25</option>
                <option value={50}>50</option>
                <option value={100}>100</option>
              </select>
              <button
                onClick={(e) => handleChangePage(e, page - 1)}
                disabled={page === 0}
                className="px-3 py-1 border border-gray-300 dark:border-gray-600 rounded text-sm disabled:opacity-50 disabled:cursor-not-allowed dark:bg-gray-700 dark:text-white"
              >
                Previous
              </button>
              <button
                onClick={(e) => handleChangePage(e, page + 1)}
                disabled={page >= Math.ceil(totalCount / rowsPerPage) - 1}
                className="px-3 py-1 border border-gray-300 dark:border-gray-600 rounded text-sm disabled:opacity-50 disabled:cursor-not-allowed dark:bg-gray-700 dark:text-white"
              >
                Next
              </button>
            </div>
          </div>
        </div>
      </div>

      {/* Action Menu */}
      {actionMenu.anchorEl && (
        <div className="fixed inset-0 z-50" onClick={() => setActionMenu({ anchorEl: null, user: null })}>
          <div 
            className="absolute bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 py-1 min-w-[150px]"
            style={{
              top: actionMenu.anchorEl.getBoundingClientRect().bottom + 5,
              left: actionMenu.anchorEl.getBoundingClientRect().left - 100,
            }}
            onClick={(e) => e.stopPropagation()}
          >
            <button
              onClick={() => handleEditUser(actionMenu.user)}
              className="w-full px-4 py-2 text-left text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center"
            >
              <Edit className="w-4 h-4 mr-2" />
              Edit User
            </button>
            <button
              onClick={() => handleToggleUserStatus(actionMenu.user)}
              className="w-full px-4 py-2 text-left text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center"
            >
              {actionMenu.user?.is_active ? (
                <>
                  <UserX className="w-4 h-4 mr-2" />
                  Deactivate
                </>
              ) : (
                <>
                  <CheckCircle className="w-4 h-4 mr-2" />
                  Activate
                </>
              )}
            </button>
            <hr className="border-gray-200 dark:border-gray-600 my-1" />
            <button
              onClick={() => handleDeleteUser(actionMenu.user)}
              className="w-full px-4 py-2 text-left text-sm text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20 flex items-center"
            >
              <Trash2 className="w-4 h-4 mr-2" />
              Delete User
            </button>
          </div>
        </div>
      )}

      {/* Invite User Dialog */}
      {inviteDialogOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50">
          <div className="bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-md w-full mx-4">
            <div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
              <h3 className="text-lg font-semibold text-gray-900 dark:text-white">Invite New User</h3>
            </div>
            
            <div className="px-6 py-4 space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Email *
                </label>
                <input
                  type="email"
                  required
                  value={inviteForm.email}
                  onChange={(e) => setInviteForm({ ...inviteForm, email: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-white"
                  placeholder="user@example.com"
                />
              </div>
              
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    First Name
                  </label>
                  <input
                    type="text"
                    value={inviteForm.firstName}
                    onChange={(e) => setInviteForm({ ...inviteForm, firstName: e.target.value })}
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-white"
                  />
                </div>
                
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    Last Name
                  </label>
                  <input
                    type="text"
                    value={inviteForm.lastName}
                    onChange={(e) => setInviteForm({ ...inviteForm, lastName: e.target.value })}
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-white"
                  />
                </div>
              </div>
              
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  Roles <span className="text-red-500">*</span>
                </label>
                <div className="space-y-2 max-h-48 overflow-y-auto border border-gray-300 dark:border-gray-600 rounded-lg p-3 bg-gray-50 dark:bg-gray-800">
                  {roles.map((role) => (
                    <label
                      key={role.id}
                      className="flex items-center space-x-3 p-2 rounded-md hover:bg-gray-100 dark:hover:bg-gray-700 cursor-pointer transition-colors duration-200"
                    >
                      <input
                        type="checkbox"
                        checked={inviteForm.roleIds.includes(role.id)}
                        onChange={(e) => {
                          const isChecked = e.target.checked;
                          setInviteForm(prev => ({
                            ...prev,
                            roleIds: isChecked
                              ? [...prev.roleIds, role.id]
                              : prev.roleIds.filter(id => id !== role.id)
                          }));
                        }}
                        className="h-4 w-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500 focus:ring-2"
                      />
                      <div className="flex-1">
                        <div className="flex items-center justify-between">
                          <span className="text-sm font-medium text-gray-900 dark:text-white">
                            {role.displayName}
                          </span>
                          <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${
                            role.name === 'admin' || role.name === 'super_admin' 
                              ? 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200'
                              : role.name === 'developer' || role.name === 'user_manager'
                              ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
                              : role.name === 'auditor' || role.name === 'viewer'
                              ? 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200'
                              : 'bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-200'
                          }`}>
                            {role.name}
                          </span>
                        </div>
                        {role.name === 'admin' && (
                          <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                            Full system access and administration
                          </p>
                        )}
                        {role.name === 'developer' && (
                          <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                            Read/write access to artifacts and repositories
                          </p>
                        )}
                        {role.name === 'auditor' && (
                          <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                            Read-only access for compliance and auditing
                          </p>
                        )}
                        {role.name === 'viewer' && (
                          <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                            Read-only access to resources
                          </p>
                        )}
                        {role.name === 'user_manager' && (
                          <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                            Can manage users and roles within tenant
                          </p>
                        )}
                        {role.name === 'user' && (
                          <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                            Basic user access
                          </p>
                        )}
                      </div>
                    </label>
                  ))}
                </div>
                
                {/* Selected roles summary */}
                {inviteForm.roleIds.length > 0 && (
                  <div className="mt-3">
                    <p className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                      Selected Roles ({inviteForm.roleIds.length})
                    </p>
                    <div className="flex flex-wrap gap-2">
                      {inviteForm.roleIds.map((id) => {
                        const role = roles.find(r => r.id === id);
                        return role ? (
                          <span
                            key={id}
                            className="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200"
                          >
                            {role.displayName}
                            <button
                              type="button"
                              onClick={() => {
                                setInviteForm(prev => ({
                                  ...prev,
                                  roleIds: prev.roleIds.filter(roleId => roleId !== id)
                                }));
                              }}
                              className="ml-2 inline-flex items-center justify-center w-4 h-4 rounded-full hover:bg-blue-200 dark:hover:bg-blue-800 transition-colors duration-200"
                            >
                              <X className="w-3 h-3" />
                            </button>
                          </span>
                        ) : null;
                      })}
                    </div>
                  </div>
                )}
                
                {inviteForm.roleIds.length === 0 && (
                  <p className="mt-2 text-sm text-amber-600 dark:text-amber-400">
                    Please select at least one role for the user
                  </p>
                )}
              </div>
            </div>
            
            <div className="px-6 py-4 border-t border-gray-200 dark:border-gray-700 flex justify-end space-x-3">
              <button
                onClick={() => setInviteDialogOpen(false)}
                className="px-4 py-2 text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-600 transition-colors duration-200"
              >
                Cancel
              </button>
              <button
                onClick={handleInviteUser}
                disabled={!inviteForm.email.trim() || inviteForm.roleIds.length === 0}
                className="px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-400 disabled:cursor-not-allowed text-white rounded-lg transition-colors duration-200"
              >
                Send Invitation
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Edit User Dialog */}
      {editDialogOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50">
          <div className="bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-2xl w-full mx-4">
            <div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
              <h3 className="text-lg font-semibold text-gray-900 dark:text-white">Edit User Profile</h3>
            </div>
            
            <div className="px-6 py-4 space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    First Name
                  </label>
                  <input
                    type="text"
                    value={editForm.firstName}
                    onChange={(e) => setEditForm({ ...editForm, firstName: e.target.value })}
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-white"
                  />
                </div>
                
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    Last Name
                  </label>
                  <input
                    type="text"
                    value={editForm.lastName}
                    onChange={(e) => setEditForm({ ...editForm, lastName: e.target.value })}
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-white"
                  />
                </div>
              </div>
              
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Display Name
                </label>
                <input
                  type="text"
                  value={editForm.displayName}
                  onChange={(e) => setEditForm({ ...editForm, displayName: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-white"
                />
              </div>
              
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    Department
                  </label>
                  <input
                    type="text"
                    value={editForm.department}
                    onChange={(e) => setEditForm({ ...editForm, department: e.target.value })}
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-white"
                  />
                </div>
                
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    Job Title
                  </label>
                  <input
                    type="text"
                    value={editForm.jobTitle}
                    onChange={(e) => setEditForm({ ...editForm, jobTitle: e.target.value })}
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-white"
                  />
                </div>
              </div>
              
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    Phone
                  </label>
                  <input
                    type="tel"
                    value={editForm.phone}
                    onChange={(e) => setEditForm({ ...editForm, phone: e.target.value })}
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-white"
                  />
                </div>
                
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    Timezone
                  </label>
                  <select
                    value={editForm.timezone}
                    onChange={(e) => setEditForm({ ...editForm, timezone: e.target.value })}
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-white"
                  >
                    <option value="UTC">UTC</option>
                    <option value="America/New_York">Eastern Time</option>
                    <option value="America/Chicago">Central Time</option>
                    <option value="America/Denver">Mountain Time</option>
                    <option value="America/Los_Angeles">Pacific Time</option>
                    <option value="Europe/London">London</option>
                    <option value="Europe/Paris">Paris</option>
                    <option value="Asia/Tokyo">Tokyo</option>
                  </select>
                </div>
              </div>
            </div>
            
            <div className="px-6 py-4 border-t border-gray-200 dark:border-gray-700 flex justify-end space-x-3">
              <button
                onClick={() => setEditDialogOpen(false)}
                className="px-4 py-2 text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-600 transition-colors duration-200"
              >
                Cancel
              </button>
              <button
                onClick={handleUpdateUser}
                className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors duration-200"
              >
                Update User
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default UserManagementDashboard;