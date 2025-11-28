import React, { useState, useEffect } from 'react';
import {
  Key,
  Plus,
  Edit,
  Trash2,
  Eye,
  EyeOff,
  Copy,
  CheckCircle,
  XCircle,
  AlertCircle,
  Calendar,
  Activity,
  Shield,
  Search,
  MoreVertical,
  X
} from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import { API_BASE_URL } from '../constants';

const APIKeyManagement = () => {
  const { getAuthHeaders } = useAuth();
  
  // State management
  const [apiKeys, setApiKeys] = useState([]);
  const [scopes, setScopes] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [success, setSuccess] = useState(null);
  
  // UI state
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [selectedKey, setSelectedKey] = useState(null);
  const [showKeyValue, setShowKeyValue] = useState({});
  const [searchTerm, setSearchTerm] = useState('');
  const [copiedKey, setCopiedKey] = useState(null);

  // Form state
  const [createForm, setCreateForm] = useState({
    name: '',
    description: '',
    scopes: [],
    expires_at: '',
    rate_limit_per_hour: 1000,
    rate_limit_per_day: 10000
  });

  // Fetch API keys
  const fetchAPIKeys = async () => {
    try {
      setLoading(true);
      const response = await fetch(`${API_BASE_URL}/api/v1/keys`, {
        headers: {
          'Content-Type': 'application/json',
          ...getAuthHeaders()
        }
      });
      
      if (response.ok) {
        const data = await response.json();
        setApiKeys(data.api_keys || []);
      } else {
        throw new Error('Failed to fetch API keys');
      }
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  // Fetch available scopes
  const fetchScopes = async () => {
    try {
      const response = await fetch(`${API_BASE_URL}/api/v1/scopes`, {
        headers: {
          'Content-Type': 'application/json',
          ...getAuthHeaders()
        }
      });
      
      if (response.ok) {
        const data = await response.json();
        setScopes(data.scopes || []);
      }
    } catch (err) {
      console.error('Failed to fetch scopes:', err);
    }
  };

  // Create API key
  const handleCreateAPIKey = async () => {
    try {
      // Prepare the request data with proper date formatting
      const requestData = {
        ...createForm,
        // Convert datetime-local to RFC3339 format or null if empty
        expires_at: createForm.expires_at ? new Date(createForm.expires_at).toISOString() : null
      };
      
      // Remove expires_at if it's null to avoid sending empty string
      if (!requestData.expires_at) {
        delete requestData.expires_at;
      }

      const response = await fetch(`${API_BASE_URL}/api/v1/keys`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...getAuthHeaders()
        },
        body: JSON.stringify(requestData)
      });

      if (response.ok) {
        const data = await response.json();
        setSuccess('API key created successfully!');
        setShowCreateModal(false);
        setCreateForm({
          name: '',
          description: '',
          scopes: [],
          expires_at: '',
          rate_limit_per_hour: 1000,
          rate_limit_per_day: 10000
        });
        fetchAPIKeys();
        
        // Show the generated key
        setSelectedKey(data);
        setShowKeyValue({ [data.id]: true });
      } else {
        const errorData = await response.json();
        throw new Error(errorData.message || 'Failed to create API key');
      }
    } catch (err) {
      setError(err.message);
    }
  };

  // Revoke API key
  const handleRevokeKey = async (keyId) => {
    if (!window.confirm('Are you sure you want to revoke this API key? This action cannot be undone.')) {
      return;
    }

    try {
      const response = await fetch(`${API_BASE_URL}/api/v1/keys/${keyId}`, {
        method: 'DELETE',
        headers: {
          'Content-Type': 'application/json',
          ...getAuthHeaders()
        }
      });

      if (response.ok) {
        setSuccess('API key revoked successfully!');
        fetchAPIKeys();
      } else {
        throw new Error('Failed to revoke API key');
      }
    } catch (err) {
      setError(err.message);
    }
  };

  // Copy API key to clipboard
  const handleCopyKey = async (key) => {
    try {
      await navigator.clipboard.writeText(key);
      setCopiedKey(key);
      setTimeout(() => setCopiedKey(null), 2000);
    } catch (err) {
      console.error('Failed to copy API key:', err);
    }
  };

  // Toggle key visibility
  const toggleKeyVisibility = (keyId) => {
    setShowKeyValue(prev => ({
      ...prev,
      [keyId]: !prev[keyId]
    }));
  };

  // Truncate API key for display
  const truncateKey = (key, maxLength = 40) => {
    if (!key) return '';
    if (key.length <= maxLength) return key;
    
    const start = key.substring(0, 20);
    const end = key.substring(key.length - 16);
    return `${start}...${end}`;
  };

  // Filter API keys based on search
  const filteredKeys = apiKeys.filter(key =>
    key.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
    key.description?.toLowerCase().includes(searchTerm.toLowerCase())
  );

  // Get scope color
  const getScopeColor = (scope) => {
    const colors = {
      'read': 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200',
      'write': 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200',
      'delete': 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200',
      'admin': 'bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-200',
      '*': 'bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200'
    };
    return colors[scope] || 'bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-200';
  };

  // Format date
  const formatDate = (dateString) => {
    if (!dateString) return 'Never expires';
    return new Date(dateString).toLocaleDateString();
  };

  useEffect(() => {
    fetchAPIKeys();
    fetchScopes();
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

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Header */}
        <div className="mb-8">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-3">
              <Key className="w-8 h-8 text-blue-600 dark:text-blue-400" />
              <h1 className="text-3xl font-bold text-gray-900 dark:text-white">API Key Management</h1>
            </div>
            <button
              onClick={() => setShowCreateModal(true)}
              className="inline-flex items-center px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 dark:bg-blue-700 dark:hover:bg-blue-800 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
            >
              <Plus className="w-4 h-4 mr-2" />
              Create API Key
            </button>
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

          {/* New API Key Created - Prominent Display */}
          {selectedKey && selectedKey.key_secret && (
            <div className="mt-4 p-6 bg-green-50 dark:bg-green-900 border-2 border-green-500 dark:border-green-600 rounded-lg shadow-lg">
              <div className="flex items-start justify-between mb-3">
                <div className="flex items-center space-x-2">
                  <CheckCircle className="w-6 h-6 text-green-600 dark:text-green-400" />
                  <h3 className="text-lg font-semibold text-green-900 dark:text-green-100">
                    🔑 API Key Created Successfully: {selectedKey.name}
                  </h3>
                </div>
                <button
                  onClick={() => setSelectedKey(null)}
                  className="text-green-600 hover:text-green-800 dark:text-green-400 dark:hover:text-green-300"
                >
                  <X className="w-5 h-5" />
                </button>
              </div>
              
              <div className="mb-3 p-3 bg-yellow-50 dark:bg-yellow-900 border border-yellow-300 dark:border-yellow-700 rounded">
                <div className="flex items-start">
                  <AlertCircle className="w-5 h-5 text-yellow-600 dark:text-yellow-400 mt-0.5 mr-2 flex-shrink-0" />
                  <div>
                    <p className="text-sm font-semibold text-yellow-800 dark:text-yellow-200">⚠️ IMPORTANT: Save this API key now!</p>
                    <p className="text-xs text-yellow-700 dark:text-yellow-300 mt-1">
                      This is the only time you'll be able to see the full API key. Copy it now and store it securely.
                    </p>
                  </div>
                </div>
              </div>

              <div className="space-y-3">
                {/* API Key Display */}
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    API Key:
                  </label>
                  <div className="flex items-center space-x-2">
                    <code className="flex-1 px-3 py-2 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded font-mono text-sm break-all">
                      {showKeyValue[selectedKey.id] ? selectedKey.key_secret : '•'.repeat(60)}
                    </code>
                    <button
                      onClick={() => toggleKeyVisibility(selectedKey.id)}
                      className="px-3 py-2 bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 border border-gray-300 dark:border-gray-600 rounded transition"
                      title={showKeyValue[selectedKey.id] ? "Hide key" : "Show key"}
                    >
                      {showKeyValue[selectedKey.id] ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                    </button>
                    <button
                      onClick={() => handleCopyKey(selectedKey.key_secret)}
                      className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white font-medium rounded transition flex items-center space-x-2"
                      title="Copy API key"
                    >
                      {copiedKey === selectedKey.key_secret ? (
                        <>
                          <CheckCircle className="w-4 h-4" />
                          <span>Copied!</span>
                        </>
                      ) : (
                        <>
                          <Copy className="w-4 h-4" />
                          <span>Copy Key</span>
                        </>
                      )}
                    </button>
                  </div>
                </div>

                {/* Quick Copy Options */}
                <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                  {/* Authorization Header */}
                  <div className="p-3 bg-gray-50 dark:bg-gray-800 rounded border border-gray-200 dark:border-gray-700">
                    <p className="text-xs font-semibold text-gray-700 dark:text-gray-300 mb-2">
                      📋 Authorization Header:
                    </p>
                    <code className="block text-xs bg-white dark:bg-gray-900 p-2 rounded border border-gray-200 dark:border-gray-600 font-mono break-all mb-2">
                      Authorization: Bearer {selectedKey.key_secret}
                    </code>
                    <button
                      onClick={() => handleCopyKey(`Authorization: Bearer ${selectedKey.key_secret}`)}
                      className="w-full px-3 py-1.5 bg-white dark:bg-gray-700 hover:bg-gray-100 dark:hover:bg-gray-600 border border-gray-300 dark:border-gray-600 rounded text-xs flex items-center justify-center space-x-1"
                    >
                      <Copy className="w-3 h-3" />
                      <span>Copy Header</span>
                    </button>
                  </div>

                  {/* cURL Example */}
                  <div className="p-3 bg-gray-50 dark:bg-gray-800 rounded border border-gray-200 dark:border-gray-700">
                    <p className="text-xs font-semibold text-gray-700 dark:text-gray-300 mb-2">
                      🚀 cURL Example:
                    </p>
                    <code className="block text-xs bg-white dark:bg-gray-900 p-2 rounded border border-gray-200 dark:border-gray-600 font-mono break-all mb-2">
                      curl -H "Authorization: Bearer {selectedKey.key_secret}" https://api.example.com/endpoint
                    </code>
                    <button
                      onClick={() => handleCopyKey(`curl -H "Authorization: Bearer ${selectedKey.key_secret}" https://api.example.com/endpoint`)}
                      className="w-full px-3 py-1.5 bg-white dark:bg-gray-700 hover:bg-gray-100 dark:hover:bg-gray-600 border border-gray-300 dark:border-gray-600 rounded text-xs flex items-center justify-center space-x-1"
                    >
                      <Copy className="w-3 h-3" />
                      <span>Copy cURL</span>
                    </button>
                  </div>
                </div>

                {/* Key Details */}
                <div className="p-3 bg-blue-50 dark:bg-blue-900 rounded border border-blue-200 dark:border-blue-700">
                  <p className="text-xs font-semibold text-blue-900 dark:text-blue-100 mb-2">📊 Key Details:</p>
                  <div className="grid grid-cols-2 gap-2 text-xs text-blue-800 dark:text-blue-200">
                    <div><strong>Key ID:</strong> {selectedKey.key_id}</div>
                    <div><strong>Rate Limit:</strong> {selectedKey.rate_limit_per_hour}/hour</div>
                    {selectedKey.expires_at && (
                      <div className="col-span-2"><strong>Expires:</strong> {new Date(selectedKey.expires_at).toLocaleString()}</div>
                    )}
                  </div>
                </div>
              </div>
            </div>
          )}

          {/* Search */}
          <div className="mt-6">
            <div className="relative">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-gray-400" />
              <input
                type="text"
                placeholder="Search API keys..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="w-full pl-10 pr-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-white"
              />
            </div>
          </div>
        </div>

        {/* API Keys Table */}
        <div className="bg-white dark:bg-gray-800 shadow rounded-lg">
          <div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
            <h3 className="text-lg font-medium text-gray-900 dark:text-white">
              API Keys ({filteredKeys.length})
            </h3>
          </div>

          {loading ? (
            <div className="flex justify-center py-12">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
            </div>
          ) : filteredKeys.length === 0 ? (
            <div className="text-center py-12">
              <Key className="mx-auto h-12 w-12 text-gray-400" />
              <h3 className="mt-2 text-sm font-medium text-gray-900 dark:text-white">No API keys</h3>
              <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {searchTerm ? 'No API keys match your search.' : 'Get started by creating a new API key.'}
              </p>
              {!searchTerm && (
                <div className="mt-6">
                  <button
                    onClick={() => setShowCreateModal(true)}
                    className="inline-flex items-center px-4 py-2 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700"
                  >
                    <Plus className="w-4 h-4 mr-2" />
                    Create API Key
                  </button>
                </div>
              )}
            </div>
          ) : (
            <div className="overflow-x-auto shadow ring-1 ring-black ring-opacity-5 md:rounded-lg">
              <table className="min-w-full table-fixed divide-y divide-gray-200 dark:divide-gray-700">
                <thead className="bg-gray-50 dark:bg-gray-700">
                  <tr>
                    <th className="w-2/5 px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                      Name & Key
                    </th>
                    <th className="w-1/6 px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                      Scopes
                    </th>
                    <th className="w-1/6 px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                      Usage
                    </th>
                    <th className="w-1/12 px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                      Status
                    </th>
                    <th className="w-1/6 px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                      Actions
                    </th>
                  </tr>
                </thead>
                <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
                  {filteredKeys.map((apiKey) => (
                    <tr key={apiKey.id} className="hover:bg-gray-50 dark:hover:bg-gray-700">
                      <td className="w-2/5 px-6 py-4 break-words">
                        <div className="flex items-center">
                          <div className="flex-shrink-0">
                            <Key className="w-8 h-8 text-blue-600 dark:text-blue-400" />
                          </div>
                          <div className="ml-4">
                            <div className="text-sm font-medium text-gray-900 dark:text-white">
                              {apiKey.name}
                            </div>
                            <div className="text-sm text-gray-500 dark:text-gray-300">
                              {apiKey.description}
                            </div>
                            {/* API Key display */}
                            <div className="mt-2 flex items-start space-x-2">
                              <div className="flex-1 min-w-0 max-w-md">
                                <code className="text-xs font-mono bg-gray-100 dark:bg-gray-700 px-2 py-1 rounded block overflow-hidden">
                                  {showKeyValue[apiKey.id] && selectedKey?.id === apiKey.id 
                                    ? (
                                        <span className="break-all word-break-break-all" title="Click copy button to copy full key">
                                          {truncateKey(selectedKey.key_secret, 40)}
                                        </span>
                                      )
                                    : `${apiKey.key_prefix}••••••••••••••••`
                                  }
                                </code>
                              </div>
                              <div className="flex items-center space-x-1 flex-shrink-0">
                                {selectedKey?.id === apiKey.id && (
                                  <>
                                    <button
                                      onClick={() => toggleKeyVisibility(apiKey.id)}
                                      className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 p-1 rounded"
                                      title={showKeyValue[apiKey.id] ? "Hide API key" : "Show API key"}
                                    >
                                      {showKeyValue[apiKey.id] ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                                    </button>
                                    {showKeyValue[apiKey.id] && (
                                      <button
                                        onClick={() => handleCopyKey(selectedKey.key_secret)}
                                        className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 p-1 rounded"
                                        title={copiedKey === selectedKey.key_secret ? "Copied!" : "Copy full API key"}
                                      >
                                        {copiedKey === selectedKey.key_secret ? 
                                          <CheckCircle className="w-4 h-4 text-green-500" /> : 
                                          <Copy className="w-4 h-4" />
                                        }
                                      </button>
                                    )}
                                  </>
                                )}
                              </div>
                            </div>
                          </div>
                        </div>
                      </td>
                      <td className="w-1/6 px-6 py-4">
                        <div className="flex flex-wrap gap-1">
                          {apiKey.scopes.slice(0, 3).map((scope) => (
                            <span
                              key={scope}
                              className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${getScopeColor(scope)}`}
                            >
                              {scope === '*' ? 'All' : scope}
                            </span>
                          ))}
                          {apiKey.scopes.length > 3 && (
                            <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-200">
                              +{apiKey.scopes.length - 3} more
                            </span>
                          )}
                        </div>
                      </td>
                      <td className="w-1/6 px-6 py-4">
                        <div className="text-sm text-gray-900 dark:text-white">
                          <div className="flex items-center">
                            <Activity className="w-4 h-4 text-gray-400 mr-1" />
                            {apiKey.usage_count} requests
                          </div>
                          <div className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                            {apiKey.last_used_at ? 
                              `Last used: ${formatDate(apiKey.last_used_at)}` : 
                              'Never used'
                            }
                          </div>
                        </div>
                      </td>
                      <td className="w-1/12 px-6 py-4">
                        <div className="flex flex-col space-y-1">
                          <span className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${
                            apiKey.is_active 
                              ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200' 
                              : 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200'
                          }`}>
                            {apiKey.is_active ? 'Active' : 'Revoked'}
                          </span>
                          <div className="text-xs text-gray-500 dark:text-gray-400">
                            <Calendar className="w-3 h-3 inline mr-1" />
                            {formatDate(apiKey.expires_at)}
                          </div>
                        </div>
                      </td>
                      <td className="w-1/6 px-6 py-4 text-sm font-medium">
                        <div className="flex items-center space-x-2">
                          <button
                            onClick={() => handleRevokeKey(apiKey.id)}
                            disabled={!apiKey.is_active}
                            className="text-red-600 hover:text-red-900 dark:text-red-400 dark:hover:text-red-300 disabled:opacity-50 disabled:cursor-not-allowed"
                            title="Revoke API Key"
                          >
                            <Trash2 className="w-4 h-4" />
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

        {/* Create API Key Modal */}
        {showCreateModal && (
          <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
            <div className="bg-white dark:bg-gray-800 rounded-lg p-6 w-full max-w-lg">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold text-gray-900 dark:text-white">Create New API Key</h3>
                <button
                  onClick={() => setShowCreateModal(false)}
                  className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                >
                  <X className="w-5 h-5" />
                </button>
              </div>
              
              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    Name <span className="text-red-500">*</span>
                  </label>
                  <input
                    type="text"
                    value={createForm.name}
                    onChange={(e) => setCreateForm({ ...createForm, name: e.target.value })}
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-white"
                    placeholder="My API Key"
                  />
                </div>
                
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    Description
                  </label>
                  <input
                    type="text"
                    value={createForm.description}
                    onChange={(e) => setCreateForm({ ...createForm, description: e.target.value })}
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-white"
                    placeholder="API key for external integrations"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                    Scopes <span className="text-red-500">*</span>
                  </label>
                  <div className="space-y-2 max-h-32 overflow-y-auto border border-gray-300 dark:border-gray-600 rounded-lg p-3">
                    {scopes.map((scope) => (
                      <label
                        key={scope.name}
                        className="flex items-center space-x-2 cursor-pointer"
                      >
                        <input
                          type="checkbox"
                          checked={createForm.scopes.includes(scope.name)}
                          onChange={(e) => {
                            const isChecked = e.target.checked;
                            setCreateForm(prev => ({
                              ...prev,
                              scopes: isChecked
                                ? [...prev.scopes, scope.name]
                                : prev.scopes.filter(s => s !== scope.name)
                            }));
                          }}
                          className="h-4 w-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
                        />
                        <span className="text-sm text-gray-900 dark:text-white">{scope.name}</span>
                        <span className="text-xs text-gray-500 dark:text-gray-400">
                          {scope.description}
                        </span>
                      </label>
                    ))}
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                      Hourly Rate Limit
                    </label>
                    <input
                      type="number"
                      value={createForm.rate_limit_per_hour}
                      onChange={(e) => setCreateForm({ ...createForm, rate_limit_per_hour: parseInt(e.target.value) })}
                      className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-white"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                      Daily Rate Limit
                    </label>
                    <input
                      type="number"
                      value={createForm.rate_limit_per_day}
                      onChange={(e) => setCreateForm({ ...createForm, rate_limit_per_day: parseInt(e.target.value) })}
                      className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-white"
                    />
                  </div>
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    Expiration Date (Optional)
                  </label>
                  <input
                    type="datetime-local"
                    value={createForm.expires_at}
                    onChange={(e) => setCreateForm({ ...createForm, expires_at: e.target.value })}
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-white"
                  />
                </div>
              </div>
              
              <div className="mt-6 flex justify-end space-x-3">
                <button
                  onClick={() => setShowCreateModal(false)}
                  className="px-4 py-2 text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-600"
                >
                  Cancel
                </button>
                <button
                  onClick={handleCreateAPIKey}
                  disabled={!createForm.name || createForm.scopes.length === 0}
                  className="px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-400 disabled:cursor-not-allowed text-white rounded-lg"
                >
                  Create API Key
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default APIKeyManagement;