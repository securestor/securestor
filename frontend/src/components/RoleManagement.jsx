import React, { useState, useEffect, useCallback } from 'react';
import { Shield, Users, Plus, Edit, Trash2, Check, X, Search, Settings, Lock, Eye } from 'lucide-react';
import roleManagementAPI from '../services/roleManagementAPI';

const RoleManagement = () => {
  const [roles, setRoles] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [showEditModal, setShowEditModal] = useState(false);
  const [selectedRole, setSelectedRole] = useState(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [includeSystemRoles, setIncludeSystemRoles] = useState(true);

  // Form state
  const [roleForm, setRoleForm] = useState({
    name: '',
    display_name: '',
    description: '',
    is_system_role: false
  });

  // Permission assignment state
  const [showPermissionModal, setShowPermissionModal] = useState(false);
  const [selectedPermissions, setSelectedPermissions] = useState([]);
  const [availablePermissions, setAvailablePermissions] = useState([]);

  // User assignment state
  const [showUserModal, setShowUserModal] = useState(false);
  const [availableUsers, setAvailableUsers] = useState([]);
  const [roleUsers, setRoleUsers] = useState([]);
  const [selectedUsers, setSelectedUsers] = useState([]);

  // Fetch roles
  const fetchRoles = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const data = await roleManagementAPI.getRoles({
        includeSystem: includeSystemRoles,
        search: searchQuery
      });
      setRoles(data.roles || []);
    } catch (err) {
      setError(`Failed to fetch roles: ${err.message}`);
    } finally {
      setLoading(false);
    }
  }, [includeSystemRoles, searchQuery]);

  // Fetch permissions
  const fetchPermissions = useCallback(async () => {
    try {
      const data = await roleManagementAPI.getPermissions();
      setAvailablePermissions(data.permissions || []);
    } catch (err) {
      setError(`Failed to fetch permissions: ${err.message}`);
    }
  }, []);

  // Fetch users
  const fetchUsers = useCallback(async () => {
    try {
      const data = await roleManagementAPI.getUsers();
      setAvailableUsers(data.users || []);
    } catch (err) {
      setError(`Failed to fetch users: ${err.message}`);
    }
  }, []);

  useEffect(() => {
    const loadData = async () => {
      await Promise.all([fetchRoles(), fetchPermissions(), fetchUsers()]);
    };
    loadData();
  }, [fetchRoles, fetchPermissions, fetchUsers]);

  // Create role
  const createRole = async () => {
    setError('');
    try {
      await roleManagementAPI.createRole(roleForm);
      setSuccess('Role created successfully!');
      setShowCreateModal(false);
      setRoleForm({ name: '', display_name: '', description: '', is_system_role: false });
      fetchRoles();
      setTimeout(() => setSuccess(''), 3000);
    } catch (err) {
      setError(`Failed to create role: ${err.message}`);
    }
  };

  // Update role
  const updateRole = async () => {
    setError('');
    try {
      await roleManagementAPI.updateRole(selectedRole.id, roleForm);
      setSuccess('Role updated successfully!');
      setShowEditModal(false);
      setSelectedRole(null);
      setRoleForm({ name: '', display_name: '', description: '', is_system_role: false });
      fetchRoles();
      setTimeout(() => setSuccess(''), 3000);
    } catch (err) {
      setError(`Failed to update role: ${err.message}`);
    }
  };

  // Delete role
  const deleteRole = async (roleId) => {
    if (!window.confirm('Are you sure you want to delete this role?')) return;

    setError('');
    try {
      await roleManagementAPI.deleteRole(roleId);
      setSuccess('Role deleted successfully!');
      fetchRoles();
      setTimeout(() => setSuccess(''), 3000);
    } catch (err) {
      setError(`Failed to delete role: ${err.message}`);
    }
  };

  // Assign permissions to role
  const assignPermissions = async () => {
    setError('');
    try {
      await roleManagementAPI.assignRolePermissions(
        selectedRole.id,
        selectedPermissions.map(p => p.id)
      );
      setSuccess('Permissions assigned successfully!');
      setShowPermissionModal(false);
      setSelectedRole(null);
      setSelectedPermissions([]);
      fetchRoles();
      setTimeout(() => setSuccess(''), 3000);
    } catch (err) {
      setError(`Failed to assign permissions: ${err.message}`);
    }
  };

  const openEditModal = (role) => {
    setSelectedRole(role);
    setRoleForm({
      name: role.name,
      display_name: role.display_name,
      description: role.description,
      is_system_role: role.is_system_role
    });
    setShowEditModal(true);
  };

  const openPermissionModal = (role) => {
    setSelectedRole(role);
    setSelectedPermissions(role.permissions || []);
    setShowPermissionModal(true);
  };

  const openUserModal = async (role) => {
    setSelectedRole(role);
    setLoading(true);
    try {
      // Get users assigned to this role
      const roleUserData = await roleManagementAPI.getRoleUsers(role.id);
      setRoleUsers(roleUserData.users || []);
      setSelectedUsers(roleUserData.users || []);
      setShowUserModal(true);
    } catch (err) {
      setError(`Failed to fetch role users: ${err.message}`);
    } finally {
      setLoading(false);
    }
  };

  const togglePermission = (permission) => {
    setSelectedPermissions(prev => {
      const exists = prev.find(p => p.id === permission.id);
      if (exists) {
        return prev.filter(p => p.id !== permission.id);
      } else {
        return [...prev, permission];
      }
    });
  };

  const toggleUser = (user) => {
    setSelectedUsers(prev => {
      const exists = prev.find(u => u.id === user.id);
      if (exists) {
        return prev.filter(u => u.id !== user.id);
      } else {
        return [...prev, user];
      }
    });
  };

  // Assign users to role
  const assignUsers = async () => {
    setError('');
    setLoading(true);
    
    try {
      // Determine which users to add and which to remove
      const currentUserIds = roleUsers.map(u => u.id);
      const selectedUserIds = selectedUsers.map(u => u.id);
      
      const usersToAdd = selectedUserIds.filter(id => !currentUserIds.includes(id));
      const usersToRemove = currentUserIds.filter(id => !selectedUserIds.includes(id));

      let successCount = 0;
      let errorCount = 0;

      // Add new users to the role
      for (const userId of usersToAdd) {
        try {
          await roleManagementAPI.assignUserRole(userId, selectedRole.id);
          successCount++;
        } catch (err) {
          console.error(`Failed to assign user ${userId}:`, err);
          errorCount++;
        }
      }

      // Remove users from the role
      for (const userId of usersToRemove) {
        try {
          await roleManagementAPI.removeUserRole(userId, selectedRole.id);
          successCount++;
        } catch (err) {
          console.error(`Failed to remove user ${userId}:`, err);
          errorCount++;
        }
      }

      if (errorCount === 0) {
        setSuccess('User assignments updated successfully!');
      } else if (successCount > 0) {
        setSuccess(`Partially completed: ${successCount} successful, ${errorCount} failed`);
      } else {
        throw new Error(`All operations failed (${errorCount} errors)`);
      }

      setShowUserModal(false);
      setSelectedRole(null);
      setSelectedUsers([]);
      setRoleUsers([]);
      fetchRoles(); // Refresh to update user counts
      fetchUsers(); // Refresh to update user roles
      setTimeout(() => setSuccess(''), 5000);
    } catch (err) {
      setError(`Failed to update user assignments: ${err.message}`);
    } finally {
      setLoading(false);
    }
  };

  const getResourceIcon = (resource) => {
    const icons = {
      'artifacts': '📦',
      'repositories': '📚',
      'users': '👥',
      'roles': '🛡️',
      'compliance': '✅',
      'scans': '🔍',
      'policies': '📋',
      'system': '⚙️',
      'tenants': '🏢'
    };
    return icons[resource] || '⚡';
  };

  const getPermissionBadgeColor = (action) => {
    const colors = {
      'read': 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200',
      'write': 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200',
      'delete': 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200',
      'admin': 'bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-200',
      'create': 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900 dark:text-emerald-200',
      'update': 'bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200'
    };
    return colors[action] || 'bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-200';
  };

  const getRoleBadgeColor = (roleName) => {
    const name = roleName?.toLowerCase() || '';
    if (name === 'admin' || name === 'super_admin') {
      return 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200';
    }
    if (name === 'developer' || name === 'user_manager') {
      return 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200';
    }
    if (name === 'auditor' || name === 'viewer') {
      return 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200';
    }
    return 'bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-200';
  };

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Header */}
        <div className="mb-8">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-3">
              <Shield className="w-8 h-8 text-blue-600 dark:text-blue-400" />
              <h1 className="text-3xl font-bold text-gray-900 dark:text-white">Role Management</h1>
            </div>
            <button
              onClick={() => setShowCreateModal(true)}
              className="inline-flex items-center px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 dark:bg-blue-700 dark:hover:bg-blue-800 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
            >
              <Plus className="w-4 h-4 mr-2" />
              Create Role
            </button>
          </div>
          
          {/* Success/Error Messages */}
          {success && (
            <div className="mt-4 p-4 bg-green-50 border border-green-200 rounded-md">
              <div className="flex">
                <Check className="w-5 h-5 text-green-400" />
                <p className="ml-3 text-sm text-green-700">{success}</p>
              </div>
            </div>
          )}
          
          {error && (
            <div className="mt-4 p-4 bg-red-50 border border-red-200 rounded-md">
              <div className="flex">
                <X className="w-5 h-5 text-red-400" />
                <p className="ml-3 text-sm text-red-700">{error}</p>
              </div>
            </div>
          )}
        </div>

        {/* Search and Filters */}
        <div className="bg-white shadow rounded-lg p-6 mb-6">
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between space-y-4 sm:space-y-0">
            <div className="flex-1 max-w-lg">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 w-5 h-5" />
                <input
                  type="text"
                  placeholder="Search roles..."
                  className="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                />
              </div>
            </div>
            
            <div className="flex items-center space-x-4">
              <label className="flex items-center">
                <input
                  type="checkbox"
                  checked={includeSystemRoles}
                  onChange={(e) => setIncludeSystemRoles(e.target.checked)}
                  className="rounded border-gray-300 text-blue-600 dark:border-gray-600 dark:bg-gray-700 focus:ring-blue-500"
                />
                <span className="ml-2 text-sm text-gray-700 dark:text-gray-300">Include System Roles</span>
              </label>
            </div>
          </div>
        </div>

        {/* Roles Table */}
        <div className="bg-white dark:bg-gray-800 shadow rounded-lg">
          <div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
            <h3 className="text-lg font-medium text-gray-900 dark:text-white">
              Roles ({roles.length})
            </h3>
          </div>

          {loading ? (
            <div className="flex justify-center py-12">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
                <thead className="bg-gray-50 dark:bg-gray-700">
                  <tr>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                      Role
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                      Permissions
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                      Users
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                      Type
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                      Actions
                    </th>
                  </tr>
                </thead>
                <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
                  {roles.map((role) => (
                    <tr key={role.id} className="hover:bg-gray-50 dark:hover:bg-gray-700">
                      <td className="px-6 py-4 whitespace-nowrap">
                        <div className="flex items-center">
                          <div className="flex-shrink-0">
                            <Shield className="w-8 h-8 text-blue-600 dark:text-blue-400" />
                          </div>
                          <div className="ml-4">
                            <div className="text-sm font-medium text-gray-900 dark:text-white">
                              {role.display_name}
                            </div>
                            <div className="text-sm text-gray-500 dark:text-gray-300">
                              {role.name}
                            </div>
                            <div className="text-xs text-gray-400 dark:text-gray-500 mt-1">
                              {role.description}
                            </div>
                          </div>
                        </div>
                      </td>
                      <td className="px-6 py-4">
                        <div className="flex flex-wrap gap-1">
                          {(role.permissions || []).slice(0, 3).map((perm) => (
                            <span
                              key={perm.id}
                              className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${getPermissionBadgeColor(perm.action)}`}
                            >
                              {getResourceIcon(perm.resource)} {perm.action}
                            </span>
                          ))}
                          {(role.permissions || []).length > 3 && (
                            <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-200">
                              +{(role.permissions || []).length - 3} more
                            </span>
                          )}
                        </div>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <div className="flex items-center">
                          <Users className="w-4 h-4 text-gray-400 dark:text-gray-500 mr-2" />
                          <span className="text-sm text-gray-900 dark:text-white">{role.user_count || 0}</span>
                        </div>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <div className="flex items-center space-x-2">
                          <span className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${getRoleBadgeColor(role.name)}`}>
                            {role.name}
                          </span>
                          <span className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${
                            role.is_system_role 
                              ? 'bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-200' 
                              : 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900 dark:text-emerald-200'
                          }`}>
                            {role.is_system_role ? (
                              <>
                                <Lock className="w-3 h-3 mr-1" />
                                System
                              </>
                            ) : (
                              <>
                                <Eye className="w-3 h-3 mr-1" />
                                Custom
                              </>
                            )}
                          </span>
                        </div>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm font-medium">
                        <div className="flex items-center space-x-2">
                          <button
                            onClick={() => openPermissionModal(role)}
                            className="text-blue-600 hover:text-blue-900 dark:text-blue-400 dark:hover:text-blue-300"
                            title="Manage Permissions"
                          >
                            <Settings className="w-4 h-4" />
                          </button>
                          <button
                            onClick={() => openUserModal(role)}
                            className="text-green-600 hover:text-green-900 dark:text-green-400 dark:hover:text-green-300"
                            title="Manage Users"
                          >
                            <Users className="w-4 h-4" />
                          </button>
                          {!role.is_system_role && (
                            <>
                              <button
                                onClick={() => openEditModal(role)}
                                className="text-amber-600 hover:text-amber-900 dark:text-amber-400 dark:hover:text-amber-300"
                                title="Edit Role"
                              >
                                <Edit className="w-4 h-4" />
                              </button>
                              <button
                                onClick={() => deleteRole(role.id)}
                                className="text-red-600 hover:text-red-900 dark:text-red-400 dark:hover:text-red-300"
                                title="Delete Role"
                              >
                                <Trash2 className="w-4 h-4" />
                              </button>
                            </>
                          )}
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}

          {roles.length === 0 && !loading && (
            <div className="text-center py-12">
              <Shield className="mx-auto h-12 w-12 text-gray-400" />
              <h3 className="mt-2 text-sm font-medium text-gray-900">No roles found</h3>
              <p className="mt-1 text-sm text-gray-500">
                Get started by creating a new role.
              </p>
            </div>
          )}
        </div>

        {/* Create Role Modal */}
        {showCreateModal && (
          <div className="fixed inset-0 bg-gray-600 bg-opacity-50 overflow-y-auto h-full w-full z-50">
            <div className="relative top-20 mx-auto p-5 border w-96 shadow-lg rounded-md bg-white">
              <div className="mt-3">
                <div className="flex items-center justify-between mb-4">
                  <h3 className="text-lg font-medium text-gray-900">Create New Role</h3>
                  <button
                    onClick={() => setShowCreateModal(false)}
                    className="text-gray-400 hover:text-gray-600"
                  >
                    <X className="w-5 h-5" />
                  </button>
                </div>
                
                <div className="space-y-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-700">Role Name</label>
                    <input
                      type="text"
                      className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500"
                      value={roleForm.name}
                      onChange={(e) => setRoleForm({...roleForm, name: e.target.value})}
                      placeholder="e.g., custom_editor"
                    />
                  </div>
                  
                  <div>
                    <label className="block text-sm font-medium text-gray-700">Display Name</label>
                    <input
                      type="text"
                      className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500"
                      value={roleForm.display_name}
                      onChange={(e) => setRoleForm({...roleForm, display_name: e.target.value})}
                      placeholder="e.g., Content Editor"
                    />
                  </div>
                  
                  <div>
                    <label className="block text-sm font-medium text-gray-700">Description</label>
                    <textarea
                      className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500"
                      rows={3}
                      value={roleForm.description}
                      onChange={(e) => setRoleForm({...roleForm, description: e.target.value})}
                      placeholder="Describe the role's purpose..."
                    />
                  </div>
                  
                  <div>
                    <label className="flex items-center">
                      <input
                        type="checkbox"
                        checked={roleForm.is_system_role}
                        onChange={(e) => setRoleForm({...roleForm, is_system_role: e.target.checked})}
                        className="rounded border-gray-300 text-indigo-600 focus:ring-indigo-500"
                      />
                      <span className="ml-2 text-sm text-gray-700">System Role</span>
                    </label>
                  </div>
                </div>
                
                <div className="flex justify-end space-x-3 mt-6">
                  <button
                    onClick={() => setShowCreateModal(false)}
                    className="px-4 py-2 border border-gray-300 rounded-md text-sm font-medium text-gray-700 hover:bg-gray-50"
                  >
                    Cancel
                  </button>
                  <button
                    onClick={createRole}
                    className="px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700"
                  >
                    Create Role
                  </button>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Edit Role Modal */}
        {showEditModal && (
          <div className="fixed inset-0 bg-gray-600 bg-opacity-50 overflow-y-auto h-full w-full z-50">
            <div className="relative top-20 mx-auto p-5 border w-96 shadow-lg rounded-md bg-white">
              <div className="mt-3">
                <div className="flex items-center justify-between mb-4">
                  <h3 className="text-lg font-medium text-gray-900">Edit Role</h3>
                  <button
                    onClick={() => setShowEditModal(false)}
                    className="text-gray-400 hover:text-gray-600"
                  >
                    <X className="w-5 h-5" />
                  </button>
                </div>
                
                <div className="space-y-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-700">Role Name</label>
                    <input
                      type="text"
                      className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500"
                      value={roleForm.name}
                      onChange={(e) => setRoleForm({...roleForm, name: e.target.value})}
                    />
                  </div>
                  
                  <div>
                    <label className="block text-sm font-medium text-gray-700">Display Name</label>
                    <input
                      type="text"
                      className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500"
                      value={roleForm.display_name}
                      onChange={(e) => setRoleForm({...roleForm, display_name: e.target.value})}
                    />
                  </div>
                  
                  <div>
                    <label className="block text-sm font-medium text-gray-700">Description</label>
                    <textarea
                      className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500"
                      rows={3}
                      value={roleForm.description}
                      onChange={(e) => setRoleForm({...roleForm, description: e.target.value})}
                    />
                  </div>
                  
                  <div>
                    <label className="flex items-center">
                      <input
                        type="checkbox"
                        checked={roleForm.is_system_role}
                        onChange={(e) => setRoleForm({...roleForm, is_system_role: e.target.checked})}
                        className="rounded border-gray-300 text-indigo-600 focus:ring-indigo-500"
                      />
                      <span className="ml-2 text-sm text-gray-700">System Role</span>
                    </label>
                  </div>
                </div>
                
                <div className="flex justify-end space-x-3 mt-6">
                  <button
                    onClick={() => setShowEditModal(false)}
                    className="px-4 py-2 border border-gray-300 rounded-md text-sm font-medium text-gray-700 hover:bg-gray-50"
                  >
                    Cancel
                  </button>
                  <button
                    onClick={updateRole}
                    className="px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700"
                  >
                    Update Role
                  </button>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Permission Assignment Modal */}
        {showPermissionModal && selectedRole && (
          <div className="fixed inset-0 bg-gray-600 bg-opacity-50 overflow-y-auto h-full w-full z-50">
            <div className="relative top-10 mx-auto p-5 border w-4/5 max-w-4xl shadow-lg rounded-md bg-white">
              <div className="mt-3">
                <div className="flex items-center justify-between mb-4">
                  <h3 className="text-lg font-medium text-gray-900">
                    Manage Permissions for "{selectedRole.display_name}"
                  </h3>
                  <button
                    onClick={() => setShowPermissionModal(false)}
                    className="text-gray-400 hover:text-gray-600"
                  >
                    <X className="w-5 h-5" />
                  </button>
                </div>
                
                <div className="max-h-96 overflow-y-auto">
                  <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                    {availablePermissions.map((permission) => {
                      const isSelected = selectedPermissions.find(p => p.id === permission.id);
                      return (
                        <div
                          key={permission.id}
                          className={`border rounded-lg p-3 cursor-pointer transition-colors ${
                            isSelected 
                              ? 'border-indigo-500 bg-indigo-50' 
                              : 'border-gray-200 hover:border-gray-300'
                          }`}
                          onClick={() => togglePermission(permission)}
                        >
                          <div className="flex items-center justify-between">
                            <div className="flex items-center space-x-2">
                              <span className="text-lg">
                                {getResourceIcon(permission.resource)}
                              </span>
                              <div>
                                <div className="text-sm font-medium text-gray-900">
                                  {permission.name}
                                </div>
                                <div className="text-xs text-gray-500">
                                  {permission.description}
                                </div>
                              </div>
                            </div>
                            <input
                              type="checkbox"
                              checked={!!isSelected}
                              onChange={() => togglePermission(permission)}
                              className="rounded border-gray-300 text-indigo-600 focus:ring-indigo-500"
                            />
                          </div>
                        </div>
                      );
                    })}
                  </div>
                </div>
                
                <div className="flex justify-between items-center mt-6">
                  <div className="text-sm text-gray-600">
                    {selectedPermissions.length} of {availablePermissions.length} permissions selected
                  </div>
                  <div className="flex space-x-3">
                    <button
                      onClick={() => setShowPermissionModal(false)}
                      className="px-4 py-2 border border-gray-300 rounded-md text-sm font-medium text-gray-700 hover:bg-gray-50"
                    >
                      Cancel
                    </button>
                    <button
                      onClick={assignPermissions}
                      className="px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700"
                    >
                      Update Permissions
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* User Assignment Modal */}
        {showUserModal && selectedRole && (
          <div className="fixed inset-0 bg-gray-600 bg-opacity-50 overflow-y-auto h-full w-full z-50">
            <div className="relative top-10 mx-auto p-5 border w-4/5 max-w-4xl shadow-lg rounded-md bg-white">
              <div className="mt-3">
                <div className="flex items-center justify-between mb-4">
                  <h3 className="text-lg font-medium text-gray-900">
                    Manage Users for "{selectedRole.display_name}"
                  </h3>
                  <button
                    onClick={() => setShowUserModal(false)}
                    className="text-gray-400 hover:text-gray-600"
                  >
                    <X className="w-5 h-5" />
                  </button>
                </div>
                
                <div className="mb-4">
                  <p className="text-sm text-gray-600">
                    Select users to assign to this role. Users with this role will inherit all associated permissions.
                  </p>
                </div>

                <div className="max-h-96 overflow-y-auto">
                  <div className="space-y-2">
                    {availableUsers.map((user) => {
                      const isSelected = selectedUsers.find(u => u.id === user.id);
                      const hasRole = user.roles?.find(r => r.id === selectedRole.id);
                      
                      return (
                        <div
                          key={user.id}
                          className={`border rounded-lg p-3 cursor-pointer transition-colors ${
                            isSelected 
                              ? 'border-indigo-500 bg-indigo-50' 
                              : hasRole
                              ? 'border-green-500 bg-green-50'
                              : 'border-gray-200 hover:border-gray-300'
                          }`}
                          onClick={() => toggleUser(user)}
                        >
                          <div className="flex items-center justify-between">
                            <div className="flex items-center space-x-3">
                              <div className="flex-shrink-0">
                                <div className="w-8 h-8 bg-gray-300 rounded-full flex items-center justify-center">
                                  <span className="text-sm font-medium text-gray-700">
                                    {user.first_name?.[0]}{user.last_name?.[0]}
                                  </span>
                                </div>
                              </div>
                              <div>
                                <div className="text-sm font-medium text-gray-900">
                                  {user.first_name} {user.last_name}
                                </div>
                                <div className="text-sm text-gray-500">
                                  @{user.username} • {user.email}
                                </div>
                                <div className="flex flex-wrap gap-1 mt-1">
                                  {user.roles?.map((role) => (
                                    <span
                                      key={role.id}
                                      className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-gray-100 text-gray-800"
                                    >
                                      {role.display_name}
                                    </span>
                                  ))}
                                </div>
                              </div>
                            </div>
                            <div className="flex items-center space-x-2">
                              {hasRole && (
                                <span className="text-xs text-green-600 font-medium">
                                  Current Role
                                </span>
                              )}
                              <input
                                type="checkbox"
                                checked={!!isSelected}
                                onChange={() => toggleUser(user)}
                                className="rounded border-gray-300 text-indigo-600 focus:ring-indigo-500"
                              />
                            </div>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                </div>
                
                <div className="flex justify-between items-center mt-6">
                  <div className="text-sm text-gray-600">
                    {selectedUsers.length} of {availableUsers.length} users selected
                  </div>
                  <div className="flex space-x-3">
                    <button
                      onClick={() => setShowUserModal(false)}
                      className="px-4 py-2 border border-gray-300 rounded-md text-sm font-medium text-gray-700 hover:bg-gray-50"
                    >
                      Cancel
                    </button>
                    <button
                      onClick={assignUsers}
                      className="px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700"
                    >
                      Update User Assignments
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default RoleManagement;