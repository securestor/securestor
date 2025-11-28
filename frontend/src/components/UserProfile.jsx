import React, { useState, useEffect } from 'react';
import {
  User,
  Mail,
  Shield,
  Key,
  Bell,
  Lock,
  Settings,
  Edit,
  Save,
  X,
  CheckCircle,
  XCircle,
  Calendar,
  MapPin,
  Phone,
  Building,
  Globe
} from 'lucide-react';
import { API_BASE_URL } from '../constants';
import { useAuth } from '../context/AuthContext';
import APIKeyManagement from './APIKeyManagement';

const UserProfile = () => {
  const { user, getAuthHeaders } = useAuth();
  
  // State management
  const [activeTab, setActiveTab] = useState('profile');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [success, setSuccess] = useState(null);
  const [isEditing, setIsEditing] = useState(false);
  const [userId, setUserId] = useState(null);
  
  // Profile form state
  const [profileForm, setProfileForm] = useState({
    first_name: '',
    last_name: '',
    display_name: '',
    email: '',
    username: ''
  });

  // Password change state
  const [passwordForm, setPasswordForm] = useState({
    current_password: '',
    new_password: '',
    confirm_password: ''
  });

  // Notification preferences state
  const [notificationPreferences, setNotificationPreferences] = useState({
    email_notifications: true,
    security_alerts: true,
    workflow_updates: false,
    digest_frequency: 'weekly'
  });

  // Load user profile data
  const loadProfile = async () => {
    try {
      setLoading(true);
      const response = await fetch(`${API_BASE_URL}/api/auth/me`, {
        headers: {
          'Content-Type': 'application/json',
          ...getAuthHeaders()
        }
      });
      
      if (response.ok) {
        const data = await response.json();
        // Store user ID for updates
        setUserId(data.id);
        // Set profile form with user data (only editable fields)
        setProfileForm({
          first_name: data.first_name || '',
          last_name: data.last_name || '',
          display_name: data.display_name || '',
          email: data.email || '',
          username: data.username || ''
        });
      }
    } catch (err) {
      console.error('Failed to load profile:', err);
      setError('Failed to load profile data');
    } finally {
      setLoading(false);
    }
  };

  // Update profile
  const handleUpdateProfile = async () => {
    try {
      setLoading(true);
      setError(null);
      setSuccess(null);

      if (!userId) {
        throw new Error('User ID not available');
      }

      // Only send fields that can be updated
      const updateData = {
        first_name: profileForm.first_name || null,
        last_name: profileForm.last_name || null,
        display_name: profileForm.display_name || null
      };

      const response = await fetch(`${API_BASE_URL}/api/v1/users/${userId}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          ...getAuthHeaders()
        },
        body: JSON.stringify(updateData)
      });
      
      if (response.ok) {
        setSuccess('Profile updated successfully!');
        setIsEditing(false);
        await loadProfile(); // Reload to get updated data
      } else {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || 'Failed to update profile');
      }
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  // Change password
  const handleChangePassword = async () => {
    if (passwordForm.new_password !== passwordForm.confirm_password) {
      setError('New passwords do not match');
      return;
    }

    try {
      setLoading(true);
      const response = await fetch(`${API_BASE_URL}/api/v1/change-password`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...getAuthHeaders()
        },
        body: JSON.stringify({
          current_password: passwordForm.current_password,
          new_password: passwordForm.new_password
        })
      });
      
      if (response.ok) {
        setSuccess('Password changed successfully!');
        setPasswordForm({ current_password: '', new_password: '', confirm_password: '' });
      } else {
        const errorData = await response.json();
        throw new Error(errorData.message || 'Failed to change password');
      }
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  // Update notification preferences
  const handleUpdateNotifications = async () => {
    try {
      setLoading(true);
      const response = await fetch(`${API_BASE_URL}/api/v1/preferences`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          ...getAuthHeaders()
        },
        body: JSON.stringify(notificationPreferences)
      });
      
      if (response.ok) {
        setSuccess('Notification preferences updated!');
      } else {
        throw new Error('Failed to update preferences');
      }
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  // Get role color
  const getRoleColor = (role) => {
    const colors = {
      'super_admin': 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200',
      'admin': 'bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200',
      'manager': 'bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-200',
      'security_analyst': 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200',
      'compliance_officer': 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200',
      'developer': 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200',
      'auditor': 'bg-indigo-100 text-indigo-800 dark:bg-indigo-900 dark:text-indigo-200',
      'viewer': 'bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-200'
    };
    return colors[role] || 'bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-200';
  };

  useEffect(() => {
    loadProfile();
  }, []);

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

  const tabs = [
    { id: 'profile', label: 'Profile', icon: User },
    { id: 'security', label: 'Security', icon: Lock },
    { id: 'api-keys', label: 'API Keys', icon: Key },
    { id: 'notifications', label: 'Notifications', icon: Bell }
  ];

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Header */}
        <div className="mb-8">
          <div className="flex items-center space-x-3">
            <Settings className="w-8 h-8 text-blue-600 dark:text-blue-400" />
            <h1 className="text-3xl font-bold text-gray-900 dark:text-white">User Profile</h1>
          </div>
          
          {/* Success/Error Messages */}
          {success && (
            <div className="mt-4 p-4 bg-green-50 dark:bg-green-900 border border-green-200 dark:border-green-700 rounded-md">
              <div className="flex">
                <CheckCircle className="w-5 h-5 text-green-400" />
                <p className="ml-3 text-sm text-green-700 dark:text-green-200">{success}</p>
              </div>
            </div>
          )}
          
          {error && (
            <div className="mt-4 p-4 bg-red-50 dark:bg-red-900 border border-red-200 dark:border-red-700 rounded-md">
              <div className="flex">
                <XCircle className="w-5 h-5 text-red-400" />
                <p className="ml-3 text-sm text-red-700 dark:text-red-200">{error}</p>
              </div>
            </div>
          )}
        </div>

        <div className="flex flex-col lg:flex-row gap-8">
          {/* Sidebar */}
          <div className="lg:w-1/4">
            <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
              {/* User Info */}
              <div className="text-center mb-6">
                <div className="w-20 h-20 bg-blue-600 rounded-full flex items-center justify-center mx-auto mb-4">
                  <User className="w-10 h-10 text-white" />
                </div>
                <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
                  {user?.first_name || 'N/A'} {user?.last_name || ''}
                </h3>
                <p className="text-sm text-gray-500 dark:text-gray-400">{user?.email}</p>
                <div className="mt-2">
                  <span className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${getRoleColor(user?.role)}`}>
                    <Shield className="w-3 h-3 mr-1" />
                    {user?.role?.replace('_', ' ')}
                  </span>
                </div>
              </div>

              {/* Navigation */}
              <nav className="space-y-2">
                {tabs.map((tab) => {
                  const Icon = tab.icon;
                  return (
                    <button
                      key={tab.id}
                      onClick={() => setActiveTab(tab.id)}
                      className={`w-full flex items-center px-3 py-2 text-sm font-medium rounded-md ${
                        activeTab === tab.id
                          ? 'bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-200'
                          : 'text-gray-600 hover:bg-gray-50 dark:text-gray-300 dark:hover:bg-gray-700'
                      }`}
                    >
                      <Icon className="w-4 h-4 mr-3" />
                      {tab.label}
                    </button>
                  );
                })}
              </nav>
            </div>
          </div>

          {/* Main Content */}
          <div className="lg:w-3/4">
            <div className="bg-white dark:bg-gray-800 rounded-lg shadow">
              {/* Profile Tab */}
              {activeTab === 'profile' && (
                <div className="p-6">
                  <div className="flex items-center justify-between mb-6">
                    <h2 className="text-xl font-semibold text-gray-900 dark:text-white">Profile Information</h2>
                    <button
                      onClick={() => isEditing ? setIsEditing(false) : setIsEditing(true)}
                      className="inline-flex items-center px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-700 hover:bg-gray-50 dark:hover:bg-gray-600"
                    >
                      {isEditing ? (
                        <>
                          <X className="w-4 h-4 mr-2" />
                          Cancel
                        </>
                      ) : (
                        <>
                          <Edit className="w-4 h-4 mr-2" />
                          Edit
                        </>
                      )}
                    </button>
                  </div>

                  <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                    <div>
                      <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                        First Name
                      </label>
                      {isEditing ? (
                        <input
                          type="text"
                          value={profileForm.first_name}
                          onChange={(e) => setProfileForm({ ...profileForm, first_name: e.target.value })}
                          className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-white"
                        />
                      ) : (
                        <p className="text-gray-900 dark:text-white">{profileForm.first_name || 'Not set'}</p>
                      )}
                    </div>

                    <div>
                      <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                        Last Name
                      </label>
                      {isEditing ? (
                        <input
                          type="text"
                          value={profileForm.last_name}
                          onChange={(e) => setProfileForm({ ...profileForm, last_name: e.target.value })}
                          className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-white"
                        />
                      ) : (
                        <p className="text-gray-900 dark:text-white">{profileForm.last_name || 'Not set'}</p>
                      )}
                    </div>

                    <div>
                      <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                        <Mail className="w-4 h-4 inline mr-1" />
                        Email
                      </label>
                      <p className="text-gray-900 dark:text-white">{user?.email}</p>
                      <p className="text-xs text-gray-500 dark:text-gray-400">Email cannot be changed</p>
                    </div>

                    <div>
                      <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                        <User className="w-4 h-4 inline mr-1" />
                        Username
                      </label>
                      <p className="text-gray-900 dark:text-white">{user?.username}</p>
                      <p className="text-xs text-gray-500 dark:text-gray-400">Username cannot be changed</p>
                    </div>

                    <div className="md:col-span-2">
                      <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                        Display Name
                      </label>
                      {isEditing ? (
                        <input
                          type="text"
                          value={profileForm.display_name}
                          onChange={(e) => setProfileForm({ ...profileForm, display_name: e.target.value })}
                          className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-white"
                          placeholder="Optional display name"
                        />
                      ) : (
                        <p className="text-gray-900 dark:text-white">{profileForm.display_name || 'Not set'}</p>
                      )}
                    </div>
                  </div>

                  {isEditing && (
                    <div className="mt-6 flex justify-end">
                      <button
                        onClick={handleUpdateProfile}
                        disabled={loading}
                        className="inline-flex items-center px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50"
                      >
                        <Save className="w-4 h-4 mr-2" />
                        Save Changes
                      </button>
                    </div>
                  )}
                </div>
              )}

              {/* Security Tab */}
              {activeTab === 'security' && (
                <div className="p-6">
                  <h2 className="text-xl font-semibold text-gray-900 dark:text-white mb-6">Security Settings</h2>
                  
                  <div className="space-y-6">
                    <div>
                      <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-4">Change Password</h3>
                      <div className="space-y-4">
                        <div>
                          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                            Current Password
                          </label>
                          <input
                            type="password"
                            value={passwordForm.current_password}
                            onChange={(e) => setPasswordForm({ ...passwordForm, current_password: e.target.value })}
                            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-white"
                          />
                        </div>
                        <div>
                          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                            New Password
                          </label>
                          <input
                            type="password"
                            value={passwordForm.new_password}
                            onChange={(e) => setPasswordForm({ ...passwordForm, new_password: e.target.value })}
                            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-white"
                          />
                        </div>
                        <div>
                          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                            Confirm New Password
                          </label>
                          <input
                            type="password"
                            value={passwordForm.confirm_password}
                            onChange={(e) => setPasswordForm({ ...passwordForm, confirm_password: e.target.value })}
                            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-white"
                          />
                        </div>
                        <button
                          onClick={handleChangePassword}
                          disabled={loading || !passwordForm.current_password || !passwordForm.new_password || !passwordForm.confirm_password}
                          className="inline-flex items-center px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50"
                        >
                          <Lock className="w-4 h-4 mr-2" />
                          Change Password
                        </button>
                      </div>
                    </div>

                    <div className="border-t border-gray-200 dark:border-gray-700 pt-6">
                      <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-4">Account Information</h3>
                      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                        <div>
                          <p className="text-sm font-medium text-gray-700 dark:text-gray-300">Account Created</p>
                          <p className="text-gray-900 dark:text-white">
                            <Calendar className="w-4 h-4 inline mr-1" />
                            {user?.created_at ? new Date(user.created_at).toLocaleDateString() : 'Unknown'}
                          </p>
                        </div>
                        <div>
                          <p className="text-sm font-medium text-gray-700 dark:text-gray-300">Last Login</p>
                          <p className="text-gray-900 dark:text-white">
                            <Calendar className="w-4 h-4 inline mr-1" />
                            {user?.last_login ? new Date(user.last_login).toLocaleDateString() : 'Unknown'}
                          </p>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              )}

              {/* API Keys Tab */}
              {activeTab === 'api-keys' && (
                <div className="p-0">
                  <APIKeyManagement />
                </div>
              )}

              {/* Notifications Tab */}
              {activeTab === 'notifications' && (
                <div className="p-6">
                  <h2 className="text-xl font-semibold text-gray-900 dark:text-white mb-6">Notification Preferences</h2>
                  
                  <div className="space-y-6">
                    <div>
                      <label className="flex items-center">
                        <input
                          type="checkbox"
                          checked={notificationPreferences.email_notifications}
                          onChange={(e) => setNotificationPreferences({ 
                            ...notificationPreferences, 
                            email_notifications: e.target.checked 
                          })}
                          className="h-4 w-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
                        />
                        <span className="ml-3 text-sm font-medium text-gray-700 dark:text-gray-300">
                          Email Notifications
                        </span>
                      </label>
                      <p className="ml-7 text-sm text-gray-500 dark:text-gray-400">
                        Receive general notifications via email
                      </p>
                    </div>

                    <div>
                      <label className="flex items-center">
                        <input
                          type="checkbox"
                          checked={notificationPreferences.security_alerts}
                          onChange={(e) => setNotificationPreferences({ 
                            ...notificationPreferences, 
                            security_alerts: e.target.checked 
                          })}
                          className="h-4 w-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
                        />
                        <span className="ml-3 text-sm font-medium text-gray-700 dark:text-gray-300">
                          Security Alerts
                        </span>
                      </label>
                      <p className="ml-7 text-sm text-gray-500 dark:text-gray-400">
                        Receive security-related notifications and alerts
                      </p>
                    </div>

                    <div>
                      <label className="flex items-center">
                        <input
                          type="checkbox"
                          checked={notificationPreferences.workflow_updates}
                          onChange={(e) => setNotificationPreferences({ 
                            ...notificationPreferences, 
                            workflow_updates: e.target.checked 
                          })}
                          className="h-4 w-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
                        />
                        <span className="ml-3 text-sm font-medium text-gray-700 dark:text-gray-300">
                          Workflow Updates
                        </span>
                      </label>
                      <p className="ml-7 text-sm text-gray-500 dark:text-gray-400">
                        Receive updates about workflow executions and status changes
                      </p>
                    </div>

                    <div>
                      <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                        Email Digest Frequency
                      </label>
                      <select
                        value={notificationPreferences.digest_frequency}
                        onChange={(e) => setNotificationPreferences({ 
                          ...notificationPreferences, 
                          digest_frequency: e.target.value 
                        })}
                        className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-white"
                      >
                        <option value="never">Never</option>
                        <option value="daily">Daily</option>
                        <option value="weekly">Weekly</option>
                        <option value="monthly">Monthly</option>
                      </select>
                    </div>

                    <div className="pt-4">
                      <button
                        onClick={handleUpdateNotifications}
                        disabled={loading}
                        className="inline-flex items-center px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50"
                      >
                        <Save className="w-4 h-4 mr-2" />
                        Save Preferences
                      </button>
                    </div>
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default UserProfile;