import React, { useState, useEffect, useCallback } from 'react';
import { 
  ArrowLeft, 
  Search, 
  Filter, 
  Download, 
  Shield, 
  AlertTriangle, 
  CheckCircle,
  XCircle,
  Clock,
  Eye,
  MoreVertical,
  SortAsc,
  SortDesc,
  Wifi,
  WifiOff,
  HardDrive,
  Cloud,
  Network
} from 'lucide-react';
import { Card, CardHeader, CardContent, Badge } from '../../../components/common';
import { useToast } from '../../../context/ToastContext';
import repositoryArtifactsAPI from '../../../services/api/repositoryArtifactsAPI';
import { useArtifactUpdates, useScanUpdates } from '../../../hooks/useRealtime';
import { useTranslation } from '../../../hooks/useTranslation';

export const RepositoryArtifactsView = ({ repository, onBack }) => {
  const { t } = useTranslation('repositories');
  const [artifacts, setArtifacts] = useState([]);
  const [stats, setStats] = useState(null);
  const [pagination, setPagination] = useState({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [filters, setFilters] = useState({
    page: 1,
    limit: 10,
    search: '',
    sortBy: 'uploaded_at',
    sortOrder: 'desc',
    type: '',
    complianceStatus: ''
  });
  const { showSuccess, showError } = useToast();

  // Fetch repository artifacts
  const fetchArtifacts = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const response = await repositoryArtifactsAPI.getRepositoryArtifacts(repository.id, filters);
      setArtifacts(response.artifacts || []);
      setStats(response.stats || {});
      setPagination(response.pagination || {});
    } catch (err) {
      console.error('Failed to fetch repository artifacts:', err);
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }, [repository.id, filters]);

  useEffect(() => {
    fetchArtifacts();
  }, [fetchArtifacts]);

  // Real-time artifact updates
  const artifactConnection = useArtifactUpdates(useCallback((data, event) => {

    // Check if the update is for this repository
    if (data?.repository_id !== repository.id && data?.data?.repository_id !== repository.id) {
      return;
    }

    switch (event) {
      case 'artifact.uploaded':
        showSuccess(t('artifactsView.messages.uploaded', { name: data.name || 'Unknown' }));
        fetchArtifacts();
        break;
      
      case 'artifact.updated':
        // Update specific artifact in the list
        setArtifacts(prev => prev.map(artifact => 
          artifact.id === data.artifact_id || artifact.id === data.id
            ? { ...artifact, ...data }
            : artifact
        ));
        break;
      
      case 'artifact.deleted':
        setArtifacts(prev => prev.filter(artifact => 
          artifact.id !== data.artifact_id && artifact.id !== data.id
        ));
        showSuccess(t('artifactsView.messages.deleted', { name: data.name || 'Unknown' }));
        break;
      
      case 'artifact.scan.started':
        setArtifacts(prev => prev.map(artifact => 
          artifact.id === data.artifact_id || artifact.id === data.id
            ? { ...artifact, scan_status: 'running' }
            : artifact
        ));
        break;
      
      case 'artifact.scan.progress':
        setArtifacts(prev => prev.map(artifact => 
          artifact.id === data.artifact_id || artifact.id === data.id
            ? { ...artifact, scan_progress: data.progress }
            : artifact
        ));
        break;
      
      case 'artifact.scan.completed':
        setArtifacts(prev => prev.map(artifact => 
          artifact.id === data.artifact_id || artifact.id === data.id
            ? { 
                ...artifact, 
                scan_status: 'completed',
                vulnerabilities: data.vulnerabilities,
                compliance: data.compliance,
                security_score: data.security_score
              }
            : artifact
        ));
        showSuccess(`Scan completed for artifact: ${data.artifact_name || 'Unknown'}`);
        break;
      
      case 'artifact.scan.failed':
        setArtifacts(prev => prev.map(artifact => 
          artifact.id === data.artifact_id || artifact.id === data.id
            ? { ...artifact, scan_status: 'failed' }
            : artifact
        ));
        showError(`Scan failed for artifact: ${data.artifact_name || 'Unknown'}`);
        break;
      
      default:
        // Refresh for unknown events
        fetchArtifacts();
    }
  }, [repository.id, fetchArtifacts, showSuccess, showError]));

  // Real-time scan updates
  useScanUpdates(useCallback((data, event) => {
    
    // Update artifact scan status in real-time
    if (data?.artifact_id) {
      setArtifacts(prev => prev.map(artifact => {
        if (artifact.id === data.artifact_id) {
          switch (event) {
            case 'scan.started':
              return { ...artifact, scan_status: 'running', scan_progress: 0 };
            case 'scan.progress':
              return { ...artifact, scan_progress: data.progress };
            case 'scan.completed':
              return { 
                ...artifact, 
                scan_status: 'completed',
                scan_progress: 100,
                vulnerabilities: data.results?.vulnerabilities,
                compliance: data.results?.compliance
              };
            case 'scan.failed':
              return { ...artifact, scan_status: 'failed', scan_progress: 0 };
            default:
              return artifact;
          }
        }
        return artifact;
      }));
    }
  }, []));

  const handleSearchChange = (e) => {
    const search = e.target.value;
    setFilters(prev => ({ ...prev, search, page: 1 }));
  };

  const handleFilterChange = (key, value) => {
    setFilters(prev => ({ ...prev, [key]: value, page: 1 }));
  };

  const handleSort = (sortBy) => {
    const sortOrder = filters.sortBy === sortBy && filters.sortOrder === 'asc' ? 'desc' : 'asc';
    setFilters(prev => ({ ...prev, sortBy, sortOrder }));
  };

  const handlePageChange = (page) => {
    setFilters(prev => ({ ...prev, page }));
  };

  const handleDownload = async (artifact) => {
    try {
      showSuccess(`Downloading ${artifact.name}...`);
      const response = await repositoryArtifactsAPI.downloadArtifact(artifact.id);
      
      // Create download link
      const blob = await response.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `${artifact.name}-${artifact.version}`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
      
      showSuccess(t('artifactsView.messages.downloadSuccess'));
    } catch (err) {
      showError(t('artifactsView.messages.downloadFailed'));
    }
  };

  const handleScan = async (artifact) => {
    try {
      await repositoryArtifactsAPI.scanArtifact(artifact.id);
      showSuccess(`Security scan initiated for ${artifact.name}`);
      // Refresh the artifacts to get updated scan status
      fetchArtifacts();
    } catch (err) {
      showError(`Failed to start scan: ${err.message}`);
    }
  };

  const formatDate = (dateString) => {
    if (!dateString) return 'N/A';
    const date = new Date(dateString);
    return date.toLocaleDateString('en-US', { 
      year: 'numeric', 
      month: 'short', 
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  };

  const getComplianceStatusIcon = (status) => {
    switch (status) {
      case 'compliant':
        return <CheckCircle className="w-4 h-4 text-green-500" />;
      case 'non-compliant':
        return <XCircle className="w-4 h-4 text-red-500" />;
      case 'review':
        return <Clock className="w-4 h-4 text-yellow-500" />;
      default:
        return <Clock className="w-4 h-4 text-gray-400" />;
    }
  };

  const getComplianceStatusColor = (status) => {
    switch (status) {
      case 'compliant':
        return 'bg-green-100 text-green-800';
      case 'non-compliant':
        return 'bg-red-100 text-red-800';
      case 'review':
        return 'bg-yellow-100 text-yellow-800';
      default:
        return 'bg-gray-100 text-gray-800';
    }
  };

  const getVulnerabilityBadge = (vulnerabilities) => {
    if (!vulnerabilities) return null;
    
    const { critical, high, medium, low } = vulnerabilities;
    const total = critical + high + medium + low;
    
    if (critical > 0) {
      return <Badge className="bg-red-600 text-white">Critical: {critical}</Badge>;
    } else if (high > 0) {
      return <Badge className="bg-orange-500 text-white">High: {high}</Badge>;
    } else if (medium > 0) {
      return <Badge className="bg-yellow-500 text-white">Medium: {medium}</Badge>;
    } else if (low > 0) {
      return <Badge className="bg-blue-500 text-white">Low: {low}</Badge>;
    } else if (total === 0) {
      return <Badge className="bg-green-500 text-white">Clean</Badge>;
    }
    
    return null;
  };

  if (loading) {
    return (
      <div className="text-center py-12">
        <p className="text-gray-500">{t('artifactsView.loading')}</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="text-center py-12">
        <p className="text-red-500 mb-4">{t('artifactsView.error')}: {error}</p>
        <button
          onClick={fetchArtifacts}
          className="inline-flex items-center space-x-2 px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 transition"
        >
          <span>{t('retry')}</span>
        </button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header with back button and repository info */}
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-4">
          <button
            onClick={onBack}
            className="flex items-center space-x-2 px-3 py-2 text-gray-600 hover:text-gray-900 hover:bg-gray-100 rounded-lg transition"
          >
            <ArrowLeft className="w-4 h-4" />
            <span>{t('artifactsView.backToRepositories')}</span>
          </button>
          <div>
            <div className="flex items-center space-x-3">
              <h1 className="text-2xl font-bold text-gray-900">{repository.name}</h1>
              {repository.cloud_provider && (
                <span className="inline-flex items-center px-2.5 py-1 text-sm font-medium rounded-lg bg-purple-100 text-purple-700">
                  <Cloud className="w-4 h-4 mr-1.5" />
                  {repository.cloud_provider.toUpperCase()}
                  {repository.cloud_region && ` • ${repository.cloud_region}`}
                </span>
              )}
              {repository.remote_url && !repository.cloud_provider && (
                <span className="inline-flex items-center px-2.5 py-1 text-sm font-medium rounded-lg bg-blue-100 text-blue-700">
                  <Network className="w-4 h-4 mr-1.5" />
                  Proxy: {new URL(repository.remote_url).hostname}
                </span>
              )}
              {!repository.cloud_provider && !repository.remote_url && (
                <span className="inline-flex items-center px-2.5 py-1 text-sm font-medium rounded-lg bg-green-100 text-green-700">
                  <HardDrive className="w-4 h-4 mr-1.5" />
                  {t('storageTypes.local')}
                </span>
              )}
            </div>
            <p className="text-sm text-gray-600 mt-1">{repository.description || 'No description'}</p>
          </div>
        </div>
        <div className="flex items-center space-x-3">
          {/* Real-time connection indicator */}
          <div className="flex items-center space-x-2 px-3 py-1.5 rounded-lg bg-gray-50 border border-gray-200">
            {artifactConnection.isConnected ? (
              <>
                <Wifi className="w-4 h-4 text-green-500" />
                <span className="text-xs text-green-700 font-medium">Live Updates</span>
                <div className="w-2 h-2 bg-green-500 rounded-full animate-pulse"></div>
              </>
            ) : (
              <>
                <WifiOff className="w-4 h-4 text-gray-400" />
                <span className="text-xs text-gray-500">Offline</span>
              </>
            )}
          </div>
          <Badge className={`${repository.type === 'docker' ? 'bg-blue-100 text-blue-800' : 
            repository.type === 'maven' ? 'bg-orange-100 text-orange-800' :
            repository.type === 'npm' ? 'bg-red-100 text-red-800' : 
            'bg-gray-100 text-gray-800'}`}>
            {repository.type.toUpperCase()}
          </Badge>
        </div>
      </div>

      {/* Repository Statistics */}
      {stats && (
        <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
          <Card>
            <CardContent className="p-6">
              <div className="flex items-center">
                <div className="flex-1">
                  <p className="text-sm font-medium text-gray-600">{t('artifactsView.stats.total')}</p>
                  <p className="text-2xl font-bold text-gray-900">{stats.totalArtifacts || 0}</p>
                </div>
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="p-6">
              <div className="flex items-center">
                <div className="flex-1">
                  <p className="text-sm font-medium text-gray-600">{t('artifactsView.stats.size')}</p>
                  <p className="text-2xl font-bold text-gray-900">{stats.totalStorageHuman || '0 B'}</p>
                </div>
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="p-6">
              <div className="flex items-center">
                <div className="flex-1">
                  <p className="text-sm font-medium text-gray-600">Recent Uploads</p>
                  <p className="text-2xl font-bold text-gray-900">{stats.recentUploads || 0}</p>
                  <p className="text-xs text-gray-500">Last 24 hours</p>
                </div>
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="p-6">
              <div className="flex items-center">
                <div className="flex-1">
                  <p className="text-sm font-medium text-gray-600">Compliance</p>
                  {stats.complianceStats && (
                    <div className="mt-2 space-y-1">
                      <div className="flex justify-between text-xs">
                        <span className="text-green-600">Compliant: {stats.complianceStats.compliant || 0}</span>
                        <span className="text-red-600">Issues: {stats.complianceStats.nonCompliant || 0}</span>
                      </div>
                    </div>
                  )}
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      )}

      {/* Filters and Search */}
      <Card>
        <CardContent className="p-6">
          <div className="flex flex-col lg:flex-row lg:items-center lg:justify-between space-y-4 lg:space-y-0 lg:space-x-4">
            {/* Search */}
            <div className="flex-1 max-w-md">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 w-4 h-4" />
                <input
                  type="text"
                  placeholder="Search artifacts..."
                  value={filters.search}
                  onChange={handleSearchChange}
                  className="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                />
              </div>
            </div>
            
            {/* Filters */}
            <div className="flex items-center space-x-4">
              <select
                value={filters.type}
                onChange={(e) => handleFilterChange('type', e.target.value)}
                className="px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              >
                <option value="">All Types</option>
                <option value="docker">Docker</option>
                <option value="maven">Maven</option>
                <option value="npm">NPM</option>
                <option value="pypi">PyPI</option>
                <option value="helm">Helm</option>
                <option value="generic">Generic</option>
              </select>
              
              <select
                value={filters.complianceStatus}
                onChange={(e) => handleFilterChange('complianceStatus', e.target.value)}
                className="px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              >
                <option value="">All Compliance</option>
                <option value="compliant">Compliant</option>
                <option value="non-compliant">Non-Compliant</option>
                <option value="review">Under Review</option>
                <option value="pending">Pending</option>
              </select>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Artifacts Table */}
      <Card>
        <CardContent className="p-0">
          {artifacts.length === 0 ? (
            <div className="text-center py-12">
              <p className="text-gray-500">No artifacts found in this repository</p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      <button
                        onClick={() => handleSort('name')}
                        className="flex items-center space-x-1 hover:text-gray-700"
                      >
                        <span>Artifact</span>
                        {filters.sortBy === 'name' && (
                          filters.sortOrder === 'asc' ? <SortAsc className="w-3 h-3" /> : <SortDesc className="w-3 h-3" />
                        )}
                      </button>
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Version</th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      <button
                        onClick={() => handleSort('size')}
                        className="flex items-center space-x-1 hover:text-gray-700"
                      >
                        <span>Size</span>
                        {filters.sortBy === 'size' && (
                          filters.sortOrder === 'asc' ? <SortAsc className="w-3 h-3" /> : <SortDesc className="w-3 h-3" />
                        )}
                      </button>
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Compliance</th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Security</th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      <button
                        onClick={() => handleSort('uploaded_at')}
                        className="flex items-center space-x-1 hover:text-gray-700"
                      >
                        <span>Uploaded</span>
                        {filters.sortBy === 'uploaded_at' && (
                          filters.sortOrder === 'asc' ? <SortAsc className="w-3 h-3" /> : <SortDesc className="w-3 h-3" />
                        )}
                      </button>
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Actions</th>
                  </tr>
                </thead>
                <tbody className="bg-white divide-y divide-gray-200">
                  {artifacts.map((artifact) => (
                    <tr key={artifact.id} className="hover:bg-gray-50 transition">
                      <td className="px-6 py-4 whitespace-nowrap">
                        <div>
                          <div className="font-medium text-gray-900">{artifact.name}</div>
                          <div className="text-sm text-gray-500">
                            <Badge className="mr-2">{artifact.type}</Badge>
                            {artifact.downloads > 0 && (
                              <span>{artifact.downloads} downloads</span>
                            )}
                          </div>
                        </div>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                        {artifact.version}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">
                        {artifact.size_formatted}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        {artifact.compliance && (
                          <div className="flex items-center space-x-2">
                            {getComplianceStatusIcon(artifact.compliance.status)}
                            <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${getComplianceStatusColor(artifact.compliance.status)}`}>
                              {artifact.compliance.status}
                            </span>
                            {artifact.compliance.score && (
                              <span className="text-sm text-gray-500">({artifact.compliance.score}%)</span>
                            )}
                          </div>
                        )}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        {getVulnerabilityBadge(artifact.vulnerabilities)}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">
                        <div>
                          <div>{formatDate(artifact.uploaded_at)}</div>
                          <div className="text-xs text-gray-500">by {artifact.uploaded_by}</div>
                        </div>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm">
                        <div className="flex items-center space-x-2">
                          <button
                            onClick={() => handleDownload(artifact)}
                            className="p-1 hover:bg-gray-100 rounded transition"
                            title="Download"
                          >
                            <Download className="w-4 h-4 text-gray-600" />
                          </button>
                          <button
                            onClick={() => handleScan(artifact)}
                            className="p-1 hover:bg-gray-100 rounded transition"
                            title="Security Scan"
                          >
                            <Shield className="w-4 h-4 text-gray-600" />
                          </button>
                          <button
                            className="p-1 hover:bg-gray-100 rounded transition"
                            title="More Options"
                          >
                            <MoreVertical className="w-4 h-4 text-gray-600" />
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Pagination */}
      {pagination.totalPages > 1 && (
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div className="text-sm text-gray-600">
                Showing {((pagination.page - 1) * pagination.limit) + 1} to {Math.min(pagination.page * pagination.limit, pagination.total)} of {pagination.total} artifacts
              </div>
              <div className="flex items-center space-x-2">
                <button
                  onClick={() => handlePageChange(pagination.page - 1)}
                  disabled={!pagination.hasPrev}
                  className="px-3 py-1 text-sm border border-gray-300 rounded hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  Previous
                </button>
                
                {/* Page numbers */}
                {Array.from({ length: Math.min(5, pagination.totalPages) }, (_, i) => {
                  const pageNum = Math.max(1, pagination.page - 2) + i;
                  if (pageNum > pagination.totalPages) return null;
                  
                  return (
                    <button
                      key={pageNum}
                      onClick={() => handlePageChange(pageNum)}
                      className={`px-3 py-1 text-sm border rounded ${
                        pageNum === pagination.page
                          ? 'bg-blue-500 text-white border-blue-500'
                          : 'border-gray-300 hover:bg-gray-50'
                      }`}
                    >
                      {pageNum}
                    </button>
                  );
                })}
                
                <button
                  onClick={() => handlePageChange(pagination.page + 1)}
                  disabled={!pagination.hasNext}
                  className="px-3 py-1 text-sm border border-gray-300 rounded hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  Next
                </button>
              </div>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
};