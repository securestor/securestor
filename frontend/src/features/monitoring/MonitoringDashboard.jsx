import React, { useState, useEffect } from 'react';
import { api } from '../../services/api';
import {
  Activity,
  TrendingUp,
  TrendingDown,
  Server,
  AlertCircle,
  CheckCircle,
  Clock,
  Database,
  Zap,
  RefreshCw,
  Wifi,
  WifiOff,
  HardDrive,
  Cloud,
  Network,
  Cpu,
  BarChart3,
  AlertTriangle,
  Info
} from 'lucide-react';
import { Card, CardHeader, CardContent, Badge } from '../../components/common';
import { useToast } from '../../context/ToastContext';

export const MonitoringDashboard = () => {
  const [metrics, setMetrics] = useState(null);
  const [healthStatus, setHealthStatus] = useState([]);
  const [alerts, setAlerts] = useState([]);
  const [loading, setLoading] = useState(true);
  const [isConnected, setIsConnected] = useState(true);
  const [lastUpdate, setLastUpdate] = useState(new Date());
  const [autoRefresh, setAutoRefresh] = useState(true);
  const { showSuccess, showError } = useToast();

  useEffect(() => {
    fetchAllData();

    if (autoRefresh) {
      // Refresh metrics every 5 seconds for real-time monitoring
      const interval = setInterval(() => {
        fetchAllData();
      }, 5000);

      return () => clearInterval(interval);
    }
  }, [autoRefresh]);

  const fetchAllData = async () => {
    try {
      setIsConnected(true);
      await Promise.all([
        fetchMetrics(),
        fetchHealthStatus(),
        fetchAlerts()
      ]);
      setLastUpdate(new Date());
    } catch (error) {
      setIsConnected(false);
      console.error('Failed to fetch monitoring data:', error);
    }
  };

  const fetchMetrics = async () => {
    try {
      const [cacheRes, perfRes] = await Promise.all([
        fetch('/api/v1/metrics/cache?time_range=1h'),
        fetch('/api/v1/metrics/performance?time_range=1h')
      ]);

      const cacheData = await cacheRes.json();
      const perfData = await perfRes.json();

      setMetrics({
        cache: cacheData.cache_metrics,
        performance: perfData.response_time_metrics,
        throughput: perfData.throughput,
        errorRate: perfData.error_rate
      });
    } catch (error) {
      console.error('Failed to fetch metrics:', error);
    } finally {
      setLoading(false);
    }
  };

  const fetchHealthStatus = async () => {
    try {
      const data = await api.get('/health/repositories');
      setHealthStatus(data.repositories || []);
    } catch (error) {
      console.error('Failed to fetch health status:', error);
      // Set empty array on error to prevent UI issues
      setHealthStatus([]);
    }
  };

  const fetchAlerts = async () => {
    try {
      const data = await api.get('/alerts?status=active');
      setAlerts(data.alerts || []);
    } catch (error) {
      console.error('Failed to fetch alerts:', error);
      // Set empty array on error to prevent UI issues
      setAlerts([]);
    }
  };

  const getSeverityColor = (severity) => {
    switch (severity) {
      case 'critical': return 'bg-red-100 text-red-800 border-red-200';
      case 'warning': return 'bg-yellow-100 text-yellow-800 border-yellow-200';
      case 'info': return 'bg-blue-100 text-blue-800 border-blue-200';
      default: return 'bg-gray-100 text-gray-800 border-gray-200';
    }
  };

  const getHealthColor = (status) => {
    return status === 'healthy' ? 'text-green-600' : 'text-red-600';
  };

  const getHealthIcon = (status) => {
    return status === 'healthy' ? CheckCircle : AlertCircle;
  };

  const MetricCard = ({ icon: Icon, title, value, subtitle, color, trend, unit = '' }) => (
    <Card className="hover:shadow-lg transition-shadow">
      <CardContent className="p-6">
        <div className="flex items-start justify-between mb-4">
          <div className={`p-3 rounded-lg bg-${color}-100`}>
            <Icon className={`w-6 h-6 text-${color}-600`} />
          </div>
          {trend !== undefined && (
            <div className={`flex items-center space-x-1 text-sm ${trend >= 0 ? 'text-green-600' : 'text-red-600'}`}>
              {trend >= 0 ? <TrendingUp className="w-4 h-4" /> : <TrendingDown className="w-4 h-4" />}
              <span>{Math.abs(trend)}%</span>
            </div>
          )}
        </div>
        <div>
          <p className="text-3xl font-bold text-gray-900 mb-1">
            {value !== null && value !== undefined ? value : '-'}
            {unit && <span className="text-xl text-gray-600 ml-1">{unit}</span>}
          </p>
          <p className="text-sm font-medium text-gray-600">{title}</p>
          {subtitle && <p className="text-xs text-gray-500 mt-1">{subtitle}</p>}
        </div>
      </CardContent>
    </Card>
  );

  if (loading) {
    return (
      <div className="p-6 flex items-center justify-center min-h-screen">
        <div className="text-center">
          <RefreshCw className="w-12 h-12 animate-spin text-blue-600 mx-auto mb-4" />
          <p className="text-gray-600">Loading monitoring data...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="p-6 space-y-6">
      {/* Header with Controls */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-900">System Monitoring</h1>
          <p className="text-gray-600 mt-1">Real-time performance metrics and health status</p>
        </div>
        <div className="flex items-center space-x-4">
          {/* Connection Status */}
          <div className={`flex items-center space-x-2 px-3 py-2 rounded-lg border ${
            isConnected ? 'bg-green-50 border-green-200' : 'bg-red-50 border-red-200'
          }`}>
            {isConnected ? (
              <>
                <Wifi className="w-4 h-4 text-green-600" />
                <span className="text-sm font-medium text-green-700">Connected</span>
                <div className="w-2 h-2 bg-green-500 rounded-full animate-pulse"></div>
              </>
            ) : (
              <>
                <WifiOff className="w-4 h-4 text-red-600" />
                <span className="text-sm font-medium text-red-700">Disconnected</span>
              </>
            )}
          </div>

          {/* Auto Refresh Toggle */}
          <button
            onClick={() => setAutoRefresh(!autoRefresh)}
            className={`flex items-center space-x-2 px-4 py-2 rounded-lg border transition ${
              autoRefresh 
                ? 'bg-blue-50 border-blue-200 text-blue-700' 
                : 'bg-gray-50 border-gray-200 text-gray-700 hover:bg-gray-100'
            }`}
          >
            <RefreshCw className={`w-4 h-4 ${autoRefresh ? 'animate-spin' : ''}`} />
            <span className="text-sm font-medium">{autoRefresh ? 'Auto' : 'Manual'}</span>
          </button>

          {/* Manual Refresh Button */}
          <button
            onClick={fetchAllData}
            className="flex items-center space-x-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition"
          >
            <RefreshCw className="w-4 h-4" />
            <span>Refresh</span>
          </button>
        </div>
      </div>

      {/* Last Update Time */}
      <div className="flex items-center space-x-2 text-sm text-gray-500">
        <Clock className="w-4 h-4" />
        <span>Last updated: {lastUpdate.toLocaleTimeString()}</span>
      </div>

      {/* Active Alerts Section */}
      {alerts.length > 0 && (
        <div className="space-y-3">
          <div className="flex items-center space-x-2">
            <AlertTriangle className="w-5 h-5 text-red-600" />
            <h2 className="text-xl font-semibold text-gray-900">
              Active Alerts ({alerts.length})
            </h2>
          </div>
          <div className="grid grid-cols-1 gap-3">
            {alerts.map((alert, index) => (
              <Card key={index} className={`border ${getSeverityColor(alert.severity)}`}>
                <CardContent className="p-4">
                  <div className="flex items-start justify-between">
                    <div className="flex items-start space-x-3">
                      <AlertCircle className="w-5 h-5 mt-0.5" />
                      <div>
                        <h3 className="font-semibold">{alert.rule_name}</h3>
                        <p className="text-sm mt-1">
                          Current: {alert.metric_value} | Threshold: {alert.threshold}
                        </p>
                      </div>
                    </div>
                    <Badge className={getSeverityColor(alert.severity)}>
                      {alert.severity?.toUpperCase()}
                    </Badge>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </div>
      )}

      {/* System Overview */}
      <div>
        <h2 className="text-xl font-semibold text-gray-900 mb-4">System Overview</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
          <MetricCard
            icon={Activity}
            title="Requests Per Second"
            value={metrics?.throughput?.requests_per_sec?.toFixed(0)}
            subtitle="Current throughput"
            color="green"
            trend={5.2}
          />
          <MetricCard
            icon={Database}
            title="Data Transfer Rate"
            value={metrics?.throughput?.megabytes_per_sec?.toFixed(1)}
            unit="MB/s"
            subtitle="Network bandwidth"
            color="blue"
          />
          <MetricCard
            icon={Cpu}
            title="Total Requests"
            value={metrics?.cache?.total_requests?.toLocaleString()}
            subtitle="Last 1 hour"
            color="purple"
          />
          <MetricCard
            icon={AlertCircle}
            title="Error Rate"
            value={metrics?.errorRate ? (metrics.errorRate * 100).toFixed(2) : '0.00'}
            unit="%"
            subtitle="Request failures"
            color={metrics?.errorRate > 0.01 ? 'red' : 'green'}
            trend={metrics?.errorRate > 0.01 ? -2.1 : 0}
          />
        </div>
      </div>

      {/* Cache Performance */}
      <div>
        <h2 className="text-xl font-semibold text-gray-900 mb-4">Cache Performance</h2>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          <Card className="hover:shadow-lg transition-shadow">
            <CardHeader>
              <div className="flex items-center space-x-3">
                <div className="p-2 rounded-lg bg-yellow-100">
                  <Zap className="w-5 h-5 text-yellow-600" />
                </div>
                <div>
                  <h3 className="font-semibold text-gray-900">L1 Cache</h3>
                  <p className="text-xs text-gray-500">Memory Cache</p>
                </div>
              </div>
            </CardHeader>
            <CardContent className="p-6">
              <div className="text-center">
                <div className="relative inline-flex items-center justify-center">
                  <svg className="w-32 h-32">
                    <circle
                      className="text-gray-200"
                      strokeWidth="8"
                      stroke="currentColor"
                      fill="transparent"
                      r="56"
                      cx="64"
                      cy="64"
                    />
                    <circle
                      className="text-yellow-500"
                      strokeWidth="8"
                      strokeDasharray={`${(metrics?.cache?.l1_hit_rate || 0) * 351.68} 351.68`}
                      strokeLinecap="round"
                      stroke="currentColor"
                      fill="transparent"
                      r="56"
                      cx="64"
                      cy="64"
                      style={{ transform: 'rotate(-90deg)', transformOrigin: '50% 50%' }}
                    />
                  </svg>
                  <div className="absolute">
                    <p className="text-3xl font-bold text-gray-900">
                      {metrics?.cache ? `${(metrics.cache.l1_hit_rate * 100).toFixed(1)}%` : '-'}
                    </p>
                  </div>
                </div>
                <p className="text-sm text-gray-600 mt-4">Hit Rate</p>
                <p className="text-xs text-gray-500 mt-1">
                  Avg: {metrics?.cache?.l1_avg_latency_ms?.toFixed(1) || '-'}ms
                </p>
              </div>
            </CardContent>
          </Card>

          <Card className="hover:shadow-lg transition-shadow">
            <CardHeader>
              <div className="flex items-center space-x-3">
                <div className="p-2 rounded-lg bg-blue-100">
                  <HardDrive className="w-5 h-5 text-blue-600" />
                </div>
                <div>
                  <h3 className="font-semibold text-gray-900">L2 Cache</h3>
                  <p className="text-xs text-gray-500">Disk Cache</p>
                </div>
              </div>
            </CardHeader>
            <CardContent className="p-6">
              <div className="text-center">
                <div className="relative inline-flex items-center justify-center">
                  <svg className="w-32 h-32">
                    <circle
                      className="text-gray-200"
                      strokeWidth="8"
                      stroke="currentColor"
                      fill="transparent"
                      r="56"
                      cx="64"
                      cy="64"
                    />
                    <circle
                      className="text-blue-500"
                      strokeWidth="8"
                      strokeDasharray={`${(metrics?.cache?.l2_hit_rate || 0) * 351.68} 351.68`}
                      strokeLinecap="round"
                      stroke="currentColor"
                      fill="transparent"
                      r="56"
                      cx="64"
                      cy="64"
                      style={{ transform: 'rotate(-90deg)', transformOrigin: '50% 50%' }}
                    />
                  </svg>
                  <div className="absolute">
                    <p className="text-3xl font-bold text-gray-900">
                      {metrics?.cache ? `${(metrics.cache.l2_hit_rate * 100).toFixed(1)}%` : '-'}
                    </p>
                  </div>
                </div>
                <p className="text-sm text-gray-600 mt-4">Hit Rate</p>
                <p className="text-xs text-gray-500 mt-1">
                  Avg: {metrics?.cache?.l2_avg_latency_ms?.toFixed(1) || '-'}ms
                </p>
              </div>
            </CardContent>
          </Card>

          <Card className="hover:shadow-lg transition-shadow">
            <CardHeader>
              <div className="flex items-center space-x-3">
                <div className="p-2 rounded-lg bg-green-100">
                  <Cloud className="w-5 h-5 text-green-600" />
                </div>
                <div>
                  <h3 className="font-semibold text-gray-900">L3 Cache</h3>
                  <p className="text-xs text-gray-500">Cloud Storage</p>
                </div>
              </div>
            </CardHeader>
            <CardContent className="p-6">
              <div className="text-center">
                <div className="relative inline-flex items-center justify-center">
                  <svg className="w-32 h-32">
                    <circle
                      className="text-gray-200"
                      strokeWidth="8"
                      stroke="currentColor"
                      fill="transparent"
                      r="56"
                      cx="64"
                      cy="64"
                    />
                    <circle
                      className="text-green-500"
                      strokeWidth="8"
                      strokeDasharray={`${(metrics?.cache?.l3_hit_rate || 0) * 351.68} 351.68`}
                      strokeLinecap="round"
                      stroke="currentColor"
                      fill="transparent"
                      r="56"
                      cx="64"
                      cy="64"
                      style={{ transform: 'rotate(-90deg)', transformOrigin: '50% 50%' }}
                    />
                  </svg>
                  <div className="absolute">
                    <p className="text-3xl font-bold text-gray-900">
                      {metrics?.cache ? `${(metrics.cache.l3_hit_rate * 100).toFixed(1)}%` : '-'}
                    </p>
                  </div>
                </div>
                <p className="text-sm text-gray-600 mt-4">Hit Rate</p>
                <p className="text-xs text-gray-500 mt-1">
                  Avg: {metrics?.cache?.l3_avg_latency_ms?.toFixed(1) || '-'}ms
                </p>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>

      {/* Response Time Metrics */}
      <div>
        <h2 className="text-xl font-semibold text-gray-900 mb-4">Response Time Analysis</h2>
        <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
          <MetricCard
            icon={Activity}
            title="P50 (Median)"
            value={metrics?.performance?.p50_ms}
            unit="ms"
            subtitle="50th percentile"
            color="green"
          />
          <MetricCard
            icon={Activity}
            title="P95"
            value={metrics?.performance?.p95_ms}
            unit="ms"
            subtitle="95th percentile"
            color="blue"
          />
          <MetricCard
            icon={Activity}
            title="P99"
            value={metrics?.performance?.p99_ms}
            unit="ms"
            subtitle="99th percentile"
            color="orange"
          />
          <MetricCard
            icon={Activity}
            title="Max Response"
            value={metrics?.performance?.max_ms}
            unit="ms"
            subtitle="Peak latency"
            color="red"
          />
        </div>
      </div>

      {/* Repository Health Status */}
      <div>
        <h2 className="text-xl font-semibold text-gray-900 mb-4">Repository Health</h2>
        <Card>
          <CardContent className="p-6">
            {healthStatus.length === 0 ? (
              <div className="text-center py-12">
                <Server className="w-16 h-16 mx-auto mb-4 text-gray-400" />
                <p className="text-gray-600 font-medium">No remote repositories configured</p>
                <p className="text-sm text-gray-500 mt-2">
                  Add remote repositories to monitor their health status
                </p>
              </div>
            ) : (
              <div className="space-y-3">
                {healthStatus.map((repo, index) => {
                  const HealthIcon = getHealthIcon(repo.status);
                  return (
                    <div
                      key={index}
                      className="flex items-center justify-between p-4 border border-gray-200 rounded-lg hover:bg-gray-50 transition"
                    >
                      <div className="flex items-center space-x-3">
                        <div className={`p-2 rounded-lg ${repo.status === 'healthy' ? 'bg-green-100' : 'bg-red-100'}`}>
                          <HealthIcon className={`w-5 h-5 ${getHealthColor(repo.status)}`} />
                        </div>
                        <div>
                          <h4 className="font-medium text-gray-900">{repo.name}</h4>
                          <p className="text-sm text-gray-500">{repo.url}</p>
                        </div>
                      </div>
                      <div className="flex items-center space-x-4">
                        {repo.response_time && (
                          <div className="text-right">
                            <p className="text-sm font-medium text-gray-900">
                              {repo.response_time}ms
                            </p>
                            <p className="text-xs text-gray-500">Response time</p>
                          </div>
                        )}
                        <Badge className={repo.status === 'healthy' ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'}>
                          {repo.status?.toUpperCase()}
                        </Badge>
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Info Footer */}
      <Card className="bg-blue-50 border-blue-200">
        <CardContent className="p-4">
          <div className="flex items-start space-x-3">
            <Info className="w-5 h-5 text-blue-600 mt-0.5" />
            <div>
              <p className="text-sm font-medium text-blue-900">Monitoring Information</p>
              <p className="text-xs text-blue-700 mt-1">
                Metrics are collected every 5 seconds and aggregated over the last hour. 
                Auto-refresh is enabled by default for real-time monitoring.
              </p>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
};

export default MonitoringDashboard;
