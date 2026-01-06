import React, { useState, useEffect, useMemo, useCallback, memo } from 'react';
import { 
  TrendingUp, 
  TrendingDown, 
  Database, 
  Package, 
  Download, 
  Users,
  HardDrive,
  Cloud,
  Network,
  Shield,
  AlertTriangle,
  Activity,
  Clock,
  Server,
  GitBranch,
  Zap
} from 'lucide-react';
import { Card, CardHeader, CardContent } from '../../../components/common';
import { useDashboard } from '../../../context/DashboardContext';
import repositoryAPI from '../../../services/api/repositoryAPI';
import { useTranslation } from '../../../hooks/useTranslation';
import DefaultPasswordWarning from '../../../components/DefaultPasswordWarning';
import SetupChecklist from '../../../components/SetupChecklist';

// Constants moved outside component for better performance
const HEALTHY_STATUSES = ['active', 'healthy', 'online'];
const SIZE_MULTIPLIERS = { B: 1, KB: 1024, MB: 1024**2, GB: 1024**3, TB: 1024**4 };
const SIZE_UNITS = ['B', 'KB', 'MB', 'GB', 'TB'];

// Utility function for parsing size strings (memoized outside component)
const parseSizeString = (sizeStr) => {
  const match = sizeStr.match(/([0-9.]+)\s*([A-Z]+)/);
  if (match) {
    const [, value, unit] = match;
    return parseFloat(value) * (SIZE_MULTIPLIERS[unit] || 1);
  }
  return 0;
};

// Utility function for formatting bytes to human readable
const formatBytes = (bytes) => {
  let unitIndex = 0;
  let size = bytes;
  while (size >= 1024 && unitIndex < SIZE_UNITS.length - 1) {
    size /= 1024;
    unitIndex++;
  }
  return `${size.toFixed(1)} ${SIZE_UNITS[unitIndex]}`;
};

export const DashboardOverview = () => {
  const { t } = useTranslation('dashboard');
  const { repositories, fetchRepositories } = useDashboard();

  // Fetch repositories on mount to ensure data is available
  useEffect(() => {
    fetchRepositories();
  }, [fetchRepositories]);

  // Memoize expensive metrics calculation
  const metrics = useMemo(() => {
    let local = 0, remote = 0, cloud = 0;
    let totalArtifacts = 0;
    let activeCount = 0;
    let healthyCount = 0;
    let totalSizeBytes = 0;

    repositories.forEach(repo => {
      // Categorize by storage type
      if (repo.cloud_provider) {
        cloud++;
      } else if (repo.remote_url) {
        remote++;
      } else {
        local++;
      }

      // Aggregate stats
      totalArtifacts += repo.artifact_count || 0;
      
      // Count active repositories
      if (repo.status === 'active') activeCount++;
      
      // Count healthy repositories (active, healthy, or online states)
      if (HEALTHY_STATUSES.includes(repo.status?.toLowerCase())) {
        healthyCount++;
      }

      // Calculate total size
      const size = repo.total_size || '0 B';
      totalSizeBytes += parseSizeString(size);
    });

    return {
      total: repositories.length,
      local,
      remote,
      cloud,
      totalArtifacts,
      totalSize: formatBytes(totalSizeBytes),
      activeRepositories: activeCount,
      healthyRepositories: healthyCount
    };
  }, [repositories]);

  // Memoized sub-components for better performance
  const StatCard = memo(({ icon: Icon, title, value, subtitle, color = 'blue', trend }) => (
    <Card className="hover:shadow-lg transition-shadow">
      <CardContent className="p-6">
        <div className="flex items-center justify-between mb-4">
          <div className={`p-3 rounded-lg bg-${color}-100`}>
            <Icon className={`w-6 h-6 text-${color}-600`} />
          </div>
          {trend && (
            <div className={`flex items-center space-x-1 text-sm ${trend > 0 ? 'text-green-600' : 'text-red-600'}`}>
              {trend > 0 ? <TrendingUp className="w-4 h-4" /> : <TrendingDown className="w-4 h-4" />}
              <span>{Math.abs(trend)}%</span>
            </div>
          )}
        </div>
        <h3 className="text-3xl font-bold text-gray-900 mb-1">{value}</h3>
        <p className="text-sm font-medium text-gray-600">{title}</p>
        {subtitle && <p className="text-xs text-gray-500 mt-1">{subtitle}</p>}
      </CardContent>
    </Card>
  ));

  // Memoize percentage calculations
  const localPercentage = useMemo(() => 
    metrics.total > 0 ? ((metrics.local / metrics.total) * 100).toFixed(0) : 0,
    [metrics.local, metrics.total]
  );

  const remotePercentage = useMemo(() => 
    metrics.total > 0 ? ((metrics.remote / metrics.total) * 100).toFixed(0) : 0,
    [metrics.remote, metrics.total]
  );

  const cloudPercentage = useMemo(() => 
    metrics.total > 0 ? ((metrics.cloud / metrics.total) * 100).toFixed(0) : 0,
    [metrics.cloud, metrics.total]
  );

  const healthPercentage = useMemo(() => 
    metrics.total > 0 ? ((metrics.healthyRepositories / metrics.total) * 100).toFixed(0) : 100,
    [metrics.healthyRepositories, metrics.total]
  );

  const systemHealthColor = useMemo(() => {
    if (metrics.total === 0) return 'gray';
    const ratio = metrics.healthyRepositories / metrics.total;
    if (ratio >= 0.9) return 'green';
    if (ratio >= 0.7) return 'yellow';
    return 'red';
  }, [metrics.healthyRepositories, metrics.total]);

  const TypeBreakdownCard = memo(() => (
    <Card className="hover:shadow-lg transition-shadow">
      <CardHeader>
        <div className="flex items-center space-x-2">
          <GitBranch className="w-5 h-5 text-gray-600" />
          <h3 className="text-lg font-semibold text-gray-900">{t('storageDistribution.title')}</h3>
        </div>
      </CardHeader>
      <CardContent className="p-6">
        <div className="space-y-4">
          {/* Local Storage */}
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-3">
              <div className="p-2 rounded-lg bg-green-100">
                <HardDrive className="w-4 h-4 text-green-600" />
              </div>
              <div>
                <p className="text-sm font-medium text-gray-900">{t('storageDistribution.localStorage')}</p>
                <p className="text-xs text-gray-500">{t('storageDistribution.localStorageDesc')}</p>
              </div>
            </div>
            <div className="text-right">
              <p className="text-2xl font-bold text-gray-900">{metrics.local}</p>
              <p className="text-xs text-gray-500">{localPercentage}%</p>
            </div>
          </div>

          {/* Remote Proxy */}
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-3">
              <div className="p-2 rounded-lg bg-blue-100">
                <Network className="w-4 h-4 text-blue-600" />
              </div>
              <div>
                <p className="text-sm font-medium text-gray-900">{t('storageDistribution.remoteProxy')}</p>
                <p className="text-xs text-gray-500">{t('storageDistribution.remoteProxyDesc')}</p>
              </div>
            </div>
            <div className="text-right">
              <p className="text-2xl font-bold text-gray-900">{metrics.remote}</p>
              <p className="text-xs text-gray-500">{remotePercentage}%</p>
            </div>
          </div>

          {/* Cloud Storage */}
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-3">
              <div className="p-2 rounded-lg bg-purple-100">
                <Cloud className="w-4 h-4 text-purple-600" />
              </div>
              <div>
                <p className="text-sm font-medium text-gray-900">{t('storageDistribution.cloudStorage')}</p>
                <p className="text-xs text-gray-500">{t('storageDistribution.cloudStorageDesc')}</p>
              </div>
            </div>
            <div className="text-right">
              <p className="text-2xl font-bold text-gray-900">{metrics.cloud}</p>
              <p className="text-xs text-gray-500">{cloudPercentage}%</p>
            </div>
          </div>

          {/* Visual Bar */}
          <div className="pt-4 border-t border-gray-200">
            <div className="flex h-3 rounded-full overflow-hidden">
              {metrics.local > 0 && (
                <div 
                  className="bg-green-500" 
                  style={{ width: `${(metrics.local / metrics.total) * 100}%` }}
                  title={`Local: ${metrics.local}`}
                />
              )}
              {metrics.remote > 0 && (
                <div 
                  className="bg-blue-500" 
                  style={{ width: `${(metrics.remote / metrics.total) * 100}%` }}
                  title={`Remote: ${metrics.remote}`}
                />
              )}
              {metrics.cloud > 0 && (
                <div 
                  className="bg-purple-500" 
                  style={{ width: `${(metrics.cloud / metrics.total) * 100}%` }}
                  title={`Cloud: ${metrics.cloud}`}
                />
              )}
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  ));

  const HealthStatusCard = memo(() => (
      <Card className="hover:shadow-lg transition-shadow">
        <CardHeader>
          <div className="flex items-center space-x-2">
            <Activity className="w-5 h-5 text-gray-600" />
            <h3 className="text-lg font-semibold text-gray-900">{t('repositoryHealth.title')}</h3>
          </div>
        </CardHeader>
        <CardContent className="p-6">
          <div className="flex items-center justify-center mb-6">
            <div className="relative">
              <svg className="w-32 h-32">
                <circle
                  className="text-gray-200"
                  strokeWidth="8"
                  stroke="currentColor"
                  fill="transparent"
                  r="58"
                  cx="64"
                  cy="64"
                />
                <circle
                  className="text-green-500"
                  strokeWidth="8"
                  strokeDasharray={`${(healthPercentage / 100) * 364} 364`}
                  strokeLinecap="round"
                  stroke="currentColor"
                  fill="transparent"
                  r="58"
                  cx="64"
                  cy="64"
                  style={{ transform: 'rotate(-90deg)', transformOrigin: '50% 50%' }}
                />
              </svg>
              <div className="absolute inset-0 flex items-center justify-center">
                <div className="text-center">
                  <p className="text-3xl font-bold text-gray-900">{healthPercentage}%</p>
                  <p className="text-xs text-gray-500">{t('repositoryHealth.healthy')}</p>
                </div>
              </div>
            </div>
          </div>
          <div className="space-y-2 text-sm">
            <div className="flex items-center justify-between">
              <span className="text-gray-600">{t('repositoryHealth.activeRepositories')}</span>
              <span className="font-semibold text-green-600">{metrics.activeRepositories}</span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-gray-600">{t('repositoryHealth.totalRepositories')}</span>
              <span className="font-semibold text-gray-900">{metrics.total}</span>
            </div>
          </div>
        </CardContent>
      </Card>
  ));

  return (
    <div className="space-y-6">
      {/* Security Warning Banner */}
      <DefaultPasswordWarning />

      {/* Setup Checklist */}
      <SetupChecklist />

      {/* Page Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-900">{t('title')}</h1>
          <p className="text-gray-600 mt-1">{t('subtitle')}</p>
        </div>
        <div className="flex items-center space-x-2 text-sm text-gray-500">
          <Clock className="w-4 h-4" />
          <span>{t('lastUpdated', { time: new Date().toLocaleTimeString() })}</span>
        </div>
      </div>

      {/* Key Metrics Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <StatCard
          icon={Database}
          title={t('stats.totalRepositories')}
          value={metrics.total}
          subtitle={t('stats.totalRepositoriesSubtitle')}
          color="blue"
        />
        <StatCard
          icon={Package}
          title={t('stats.totalArtifacts')}
          value={metrics.totalArtifacts.toLocaleString()}
          subtitle={t('stats.totalArtifactsSubtitle')}
          color="green"
        />
        <StatCard
          icon={Server}
          title={t('stats.storageUsed')}
          value={metrics.totalSize}
          subtitle={t('stats.storageUsedSubtitle')}
          color="purple"
        />
        <StatCard
          icon={Shield}
          title={t('stats.systemHealth')}
          value={`${healthPercentage}%`}
          subtitle={metrics.total > 0 ? t('stats.systemHealthSubtitle', { healthy: metrics.healthyRepositories, total: metrics.total }) : t('stats.noRepositories')}
          color={systemHealthColor}
        />
      </div>

      {/* Detailed Breakdown */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <TypeBreakdownCard />
        <HealthStatusCard />
      </div>

      {/* Quick Actions */}
      <Card>
        <CardHeader>
          <div className="flex items-center space-x-2">
            <Zap className="w-5 h-5 text-gray-600" />
            <h3 className="text-lg font-semibold text-gray-900">{t('quickInsights.title')}</h3>
          </div>
        </CardHeader>
        <CardContent className="p-6">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="flex items-center space-x-3 p-4 bg-blue-50 rounded-lg">
              <Network className="w-8 h-8 text-blue-600" />
              <div>
                <p className="text-sm font-medium text-blue-900">{t('quickInsights.remoteProxies')}</p>
                <p className="text-xs text-blue-700">
                  {t('quickInsights.remoteProxiesDesc', { count: metrics.remote })}
                </p>
              </div>
            </div>
            <div className="flex items-center space-x-3 p-4 bg-purple-50 rounded-lg">
              <Cloud className="w-8 h-8 text-purple-600" />
              <div>
                <p className="text-sm font-medium text-purple-900">{t('quickInsights.cloudIntegration')}</p>
                <p className="text-xs text-purple-700">
                  {t('quickInsights.cloudIntegrationDesc', { count: metrics.cloud })}
                </p>
              </div>
            </div>
            <div className="flex items-center space-x-3 p-4 bg-green-50 rounded-lg">
              <HardDrive className="w-8 h-8 text-green-600" />
              <div>
                <p className="text-sm font-medium text-green-900">{t('quickInsights.localStorage')}</p>
                <p className="text-xs text-green-700">
                  {t('quickInsights.localStorageDesc', { count: metrics.local })}
                </p>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
};
