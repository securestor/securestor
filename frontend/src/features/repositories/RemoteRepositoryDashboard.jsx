import React, { useState, useEffect } from 'react';
import {
  Cloud,
  Server,
  Activity,
  AlertCircle,
  CheckCircle,
  RefreshCw,
  Upload,
  Download,
  Shield,
  TrendingUp,
  Database
} from 'lucide-react';
import { Card, CardHeader, CardContent } from '../../components/common';
import { useToast } from '../../context/ToastContext';
import { useTranslation } from '../../hooks/useTranslation';

export const RemoteRepositoryDashboard = () => {
  const { t } = useTranslation('repositories');
  const [repositories, setRepositories] = useState([]);
  const [metrics, setMetrics] = useState(null);
  const [loading, setLoading] = useState(true);
  const { showSuccess, showError } = useToast();

  useEffect(() => {
    fetchRepositories();
    fetchMetrics();
  }, []);

  const fetchRepositories = async () => {
    try {
      // This will call your backend API
      const response = await fetch('/api/v1/remote-repositories');
      const data = await response.json();
      setRepositories(data.repositories || []);
    } catch (error) {
      console.error('Failed to fetch repositories:', error);
      showError(t('remote.loadFailed'));
    } finally {
      setLoading(false);
    }
  };

  const fetchMetrics = async () => {
    try {
      const response = await fetch('/api/v1/metrics/cache?time_range=1h');
      const data = await response.json();
      setMetrics(data);
      
      // Also fetch proxy-specific cache metrics
      try {
        const proxyResponse = await fetch('/api/proxy/cache/stats');
        const proxyData = await proxyResponse.json();
        setMetrics(prev => ({ ...prev, proxy: proxyData }));
      } catch (proxyError) {
      }
    } catch (error) {
      console.error('Failed to fetch metrics:', error);
    }
  };

  const testConnection = async (repoId) => {
    try {
      const response = await fetch(`/api/v1/remote-repositories/${repoId}/test`, {
        method: 'POST'
      });
      const data = await response.json();
      
      if (response.ok) {
        showSuccess(t('remote.connectionSuccess', { message: data.message }));
        fetchRepositories();
      } else {
        showError(t('remote.connectionFailed', { error: data.error }));
      }
    } catch (error) {
      showError(t('remote.testFailed'));
    }
  };

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">{t('remote.title')}</h1>
          <p className="text-gray-600">{t('remote.subtitle')}</p>
        </div>
        <button
          onClick={() => window.location.reload()}
          className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
        >
          <RefreshCw className="w-4 h-4" />
          {t('remote.refresh')}
        </button>
      </div>

      {/* Metrics Overview */}
      {metrics && (
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <Card>
            <CardContent className="p-4">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-gray-600">{t('remote.metrics.cacheHitRate')}</p>
                  <p className="text-2xl font-bold text-gray-900">
                    {((metrics.cache_metrics?.l3_hit_rate || 0) * 100).toFixed(1)}%
                  </p>
                </div>
                <TrendingUp className="w-8 h-8 text-green-500" />
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardContent className="p-4">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-gray-600">{t('remote.metrics.totalRequests')}</p>
                  <p className="text-2xl font-bold text-gray-900">
                    {metrics.cache_metrics?.total_requests?.toLocaleString() || 0}
                  </p>
                </div>
                <Activity className="w-8 h-8 text-blue-500" />
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardContent className="p-4">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-gray-600">{t('remote.metrics.cacheHits')}</p>
                  <p className="text-2xl font-bold text-gray-900">
                    {metrics.cache_metrics?.cache_hits?.toLocaleString() || 0}
                  </p>
                </div>
                <CheckCircle className="w-8 h-8 text-green-500" />
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardContent className="p-4">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-gray-600">{t('remote.metrics.cacheMisses')}</p>
                  <p className="text-2xl font-bold text-gray-900">
                    {metrics.cache_metrics?.cache_misses?.toLocaleString() || 0}
                  </p>
                </div>
                <Database className="w-8 h-8 text-orange-500" />
              </div>
            </CardContent>
          </Card>
        </div>
      )}

      {/* Remote Proxy Cache Statistics */}
      {metrics?.proxy && (
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <div>
                <h2 className="text-lg font-semibold text-gray-900">{t('remote.proxyCache.title')}</h2>
                <p className="text-sm text-gray-600">{t('remote.proxyCache.subtitle')}</p>
              </div>
              <Shield className="w-6 h-6 text-indigo-600" />
            </div>
          </CardHeader>
          <CardContent className="space-y-6">
            {/* Overall Proxy Stats */}
            {metrics.proxy.overall && (
              <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
                <div className="p-4 bg-indigo-50 rounded-lg">
                  <p className="text-sm text-gray-600 mb-1">{t('remote.metrics.cacheHitRate')}</p>
                  <p className="text-2xl font-bold text-indigo-600">
                    {metrics.proxy.overall.hit_rate?.toFixed(1)}%
                  </p>
                </div>
                <div className="p-4 bg-blue-50 rounded-lg">
                  <p className="text-sm text-gray-600 mb-1">{t('remote.metrics.totalRequests')}</p>
                  <p className="text-2xl font-bold text-blue-600">
                    {metrics.proxy.overall.total_requests?.toLocaleString()}
                  </p>
                </div>
                <div className="p-4 bg-green-50 rounded-lg">
                  <p className="text-sm text-gray-600 mb-1">{t('remote.proxyCache.bandwidthSaved')}</p>
                  <p className="text-2xl font-bold text-green-600">
                    {metrics.proxy.overall.bandwidth_saved_gb?.toFixed(2)} GB
                  </p>
                </div>
                <div className="p-4 bg-purple-50 rounded-lg">
                  <p className="text-sm text-gray-600 mb-1">{t('remote.proxyCache.avgResponse')}</p>
                  <p className="text-2xl font-bold text-purple-600">
                    {metrics.proxy.overall.avg_response_ms?.toFixed(0)} ms
                  </p>
                </div>
              </div>
            )}

            {/* Cache Tier Breakdown */}
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              {/* L1 Cache (Redis) */}
              {metrics.proxy.l1_cache && (
                <div className="p-4 border border-green-200 rounded-lg bg-green-50">
                  <div className="flex items-center justify-between mb-3">
                    <h3 className="font-semibold text-gray-900">L1 Cache (Redis)</h3>
                    <span className="text-xs bg-green-200 text-green-800 px-2 py-1 rounded-full">
                      {metrics.proxy.l1_cache.hit_rate?.toFixed(1)}% hits
                    </span>
                  </div>
                  <div className="space-y-2 text-sm">
                    <div className="flex justify-between">
                      <span className="text-gray-600">Hits</span>
                      <span className="font-medium text-gray-900">
                        {metrics.proxy.l1_cache.hits?.toLocaleString()}
                      </span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-gray-600">Misses</span>
                      <span className="font-medium text-gray-900">
                        {metrics.proxy.l1_cache.misses?.toLocaleString()}
                      </span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-gray-600">Size</span>
                      <span className="font-medium text-gray-900">
                        {metrics.proxy.l1_cache.size_gb?.toFixed(2)} GB
                      </span>
                    </div>
                  </div>
                </div>
              )}

              {/* L2 Cache (Disk) */}
              {metrics.proxy.l2_cache && (
                <div className="p-4 border border-blue-200 rounded-lg bg-blue-50">
                  <div className="flex items-center justify-between mb-3">
                    <h3 className="font-semibold text-gray-900">L2 Cache (Disk)</h3>
                    <span className="text-xs bg-blue-200 text-blue-800 px-2 py-1 rounded-full">
                      {metrics.proxy.l2_cache.hit_rate?.toFixed(1)}% hits
                    </span>
                  </div>
                  <div className="space-y-2 text-sm">
                    <div className="flex justify-between">
                      <span className="text-gray-600">Hits</span>
                      <span className="font-medium text-gray-900">
                        {metrics.proxy.l2_cache.hits?.toLocaleString()}
                      </span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-gray-600">Misses</span>
                      <span className="font-medium text-gray-900">
                        {metrics.proxy.l2_cache.misses?.toLocaleString()}
                      </span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-gray-600">Size</span>
                      <span className="font-medium text-gray-900">
                        {metrics.proxy.l2_cache.size_gb?.toFixed(2)} GB
                      </span>
                    </div>
                  </div>
                </div>
              )}

              {/* L3 Cache (Cloud) */}
              {metrics.proxy.l3_cache && (
                <div className="p-4 border border-purple-200 rounded-lg bg-purple-50">
                  <div className="flex items-center justify-between mb-3">
                    <h3 className="font-semibold text-gray-900">L3 Cache (Cloud)</h3>
                    <span className="text-xs bg-purple-200 text-purple-800 px-2 py-1 rounded-full">
                      {metrics.proxy.l3_cache.hit_rate?.toFixed(1)}% hits
                    </span>
                  </div>
                  <div className="space-y-2 text-sm">
                    <div className="flex justify-between">
                      <span className="text-gray-600">Hits</span>
                      <span className="font-medium text-gray-900">
                        {metrics.proxy.l3_cache.hits?.toLocaleString()}
                      </span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-gray-600">Misses</span>
                      <span className="font-medium text-gray-900">
                        {metrics.proxy.l3_cache.misses?.toLocaleString()}
                      </span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-gray-600">Size</span>
                      <span className="font-medium text-gray-900">
                        {metrics.proxy.l3_cache.size_tb?.toFixed(2)} TB
                      </span>
                    </div>
                  </div>
                </div>
              )}
            </div>

            {/* Quick Start Guide */}
            <div className="mt-6 p-4 bg-gradient-to-r from-indigo-50 to-purple-50 rounded-lg">
              <h3 className="font-semibold text-gray-900 mb-3">Remote Proxy Usage Examples</h3>
              <div className="space-y-2 text-sm">
                <div className="font-mono text-xs bg-white p-2 rounded border">
                  <span className="text-gray-600">Maven:</span> <span className="text-indigo-600">curl http://localhost/api/v1/proxy/maven/org/springframework/.../spring-boot.jar</span>
                </div>
                <div className="font-mono text-xs bg-white p-2 rounded border">
                  <span className="text-gray-600">PyPI:</span> <span className="text-indigo-600">curl http://localhost/api/v1/proxy/pypi/simple/requests/</span>
                </div>
                <div className="font-mono text-xs bg-white p-2 rounded border">
                  <span className="text-gray-600">Helm:</span> <span className="text-indigo-600">curl http://localhost/api/v1/proxy/helm/index.yaml</span>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Cloud Provider Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {/* AWS S3 */}
        <Card>
          <CardHeader>
            <div className="flex items-center gap-3">
              <div className="p-2 bg-orange-100 rounded-lg">
                <Cloud className="w-6 h-6 text-orange-600" />
              </div>
              <div>
                <h3 className="font-semibold text-gray-900">AWS S3</h3>
                <p className="text-sm text-gray-600">Amazon Simple Storage</p>
              </div>
            </div>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <div className="flex justify-between text-sm">
                <span className="text-gray-600">Status</span>
                <span className="flex items-center gap-1 text-green-600 font-medium">
                  <CheckCircle className="w-4 h-4" />
                  Connected
                </span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-gray-600">Region</span>
                <span className="text-gray-900">us-east-1</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-gray-600">Encryption</span>
                <span className="text-gray-900">AES256</span>
              </div>
            </div>
            
            <button
              onClick={() => testConnection('aws-s3')}
              className="w-full px-4 py-2 bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200 transition-colors"
            >
              Test Connection
            </button>

            <div className="pt-3 border-t space-y-2">
              <div className="flex justify-between text-sm">
                <span className="text-gray-600">Uploads (24h)</span>
                <span className="text-gray-900 font-medium">1,234</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-gray-600">Downloads (24h)</span>
                <span className="text-gray-900 font-medium">5,678</span>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Google Cloud Storage */}
        <Card>
          <CardHeader>
            <div className="flex items-center gap-3">
              <div className="p-2 bg-blue-100 rounded-lg">
                <Cloud className="w-6 h-6 text-blue-600" />
              </div>
              <div>
                <h3 className="font-semibold text-gray-900">Google Cloud Storage</h3>
                <p className="text-sm text-gray-600">GCS Buckets</p>
              </div>
            </div>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <div className="flex justify-between text-sm">
                <span className="text-gray-600">Status</span>
                <span className="flex items-center gap-1 text-green-600 font-medium">
                  <CheckCircle className="w-4 h-4" />
                  Connected
                </span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-gray-600">Project ID</span>
                <span className="text-gray-900 truncate">my-project-123</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-gray-600">Storage Class</span>
                <span className="text-gray-900">STANDARD</span>
              </div>
            </div>
            
            <button
              onClick={() => testConnection('gcs')}
              className="w-full px-4 py-2 bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200 transition-colors"
            >
              Test Connection
            </button>

            <div className="pt-3 border-t space-y-2">
              <div className="flex justify-between text-sm">
                <span className="text-gray-600">Uploads (24h)</span>
                <span className="text-gray-900 font-medium">987</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-gray-600">Downloads (24h)</span>
                <span className="text-gray-900 font-medium">4,321</span>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Azure Blob Storage */}
        <Card>
          <CardHeader>
            <div className="flex items-center gap-3">
              <div className="p-2 bg-indigo-100 rounded-lg">
                <Cloud className="w-6 h-6 text-indigo-600" />
              </div>
              <div>
                <h3 className="font-semibold text-gray-900">Azure Blob Storage</h3>
                <p className="text-sm text-gray-600">Microsoft Azure</p>
              </div>
            </div>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <div className="flex justify-between text-sm">
                <span className="text-gray-600">Status</span>
                <span className="flex items-center gap-1 text-green-600 font-medium">
                  <CheckCircle className="w-4 h-4" />
                  Connected
                </span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-gray-600">Account</span>
                <span className="text-gray-900 truncate">mystorageacct</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-gray-600">Container</span>
                <span className="text-gray-900">artifacts</span>
              </div>
            </div>
            
            <button
              onClick={() => testConnection('azure')}
              className="w-full px-4 py-2 bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200 transition-colors"
            >
              Test Connection
            </button>

            <div className="pt-3 border-t space-y-2">
              <div className="flex justify-between text-sm">
                <span className="text-gray-600">Uploads (24h)</span>
                <span className="text-gray-900 font-medium">765</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-gray-600">Downloads (24h)</span>
                <span className="text-gray-900 font-medium">3,210</span>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Upload Test Section */}
      <Card>
        <CardHeader>
          <h3 className="text-lg font-semibold text-gray-900">Test File Upload</h3>
          <p className="text-sm text-gray-600">Upload a file to test cloud provider integration</p>
        </CardHeader>
        <CardContent>
          <FileUploadTest onSuccess={fetchMetrics} />
        </CardContent>
      </Card>
    </div>
  );
};

// File Upload Test Component
const FileUploadTest = ({ onSuccess }) => {
  const [file, setFile] = useState(null);
  const [provider, setProvider] = useState('aws-s3');
  const [uploading, setUploading] = useState(false);
  const { showSuccess, showError } = useToast();

  const handleUpload = async () => {
    if (!file) {
      showError('Please select a file');
      return;
    }

    setUploading(true);
    const formData = new FormData();
    formData.append('file', file);
    formData.append('cloud_provider', provider);

    try {
      const response = await fetch('/api/v1/artifacts/upload', {
        method: 'POST',
        body: formData
      });

      if (response.ok) {
        const data = await response.json();
        showSuccess(`File uploaded successfully to ${provider}: ${data.location}`);
        setFile(null);
        onSuccess();
      } else {
        const error = await response.json();
        showError(`Upload failed: ${error.message}`);
      }
    } catch (error) {
      showError('Upload failed: ' + error.message);
    } finally {
      setUploading(false);
    }
  };

  return (
    <div className="space-y-4">
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-2">
          Select Cloud Provider
        </label>
        <select
          value={provider}
          onChange={(e) => setProvider(e.target.value)}
          className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
        >
          <option value="aws-s3">AWS S3</option>
          <option value="gcs">Google Cloud Storage</option>
          <option value="azure">Azure Blob Storage</option>
        </select>
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-700 mb-2">
          Select File
        </label>
        <input
          type="file"
          onChange={(e) => setFile(e.target.files[0])}
          className="w-full px-3 py-2 border border-gray-300 rounded-lg"
        />
      </div>

      {file && (
        <div className="p-3 bg-gray-50 rounded-lg">
          <p className="text-sm text-gray-600">
            <strong>File:</strong> {file.name} ({(file.size / 1024 / 1024).toFixed(2)} MB)
          </p>
        </div>
      )}

      <button
        onClick={handleUpload}
        disabled={!file || uploading}
        className="w-full flex items-center justify-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:bg-gray-300 disabled:cursor-not-allowed"
      >
        {uploading ? (
          <>
            <RefreshCw className="w-4 h-4 animate-spin" />
            Uploading...
          </>
        ) : (
          <>
            <Upload className="w-4 h-4" />
            Upload to {provider}
          </>
        )}
      </button>
    </div>
  );
};

export default RemoteRepositoryDashboard;
