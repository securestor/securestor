import React from 'react';
import { Shield, RefreshCw, Download } from 'lucide-react';
import StatisticsCards from '../StatisticsCards';
import VulnerabilityDistribution from '../VulnerabilityDistribution';
import ScannerStatus from '../ScannerStatus';
import VulnerableArtifactsList from '../VulnerableArtifactsList';
import { useSecurityDashboard } from '../../hooks/useSecurityDashboard';

const SecurityDashboard = () => {
  const {
    dashboard,
    loading,
    error,
    refresh,
    exportReport
  } = useSecurityDashboard();

  if (loading) {
    return <LoadingState />;
  }

  if (error) {
    return <ErrorState error={error} onRetry={refresh} />;
  }

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <DashboardHeader onRefresh={refresh} onExport={exportReport} />
      
      <div className="p-6">
        {/* Statistics */}
        <StatisticsCards stats={dashboard?.stats} />
        
        {/* Main Content Grid */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 mb-6">
          <VulnerabilityDistribution 
            vulnerabilities={dashboard?.vulnerabilities} 
          />
          <ScannerStatus scanners={dashboard?.scanners} />
        </div>
        
        {/* Vulnerable Artifacts */}
        <VulnerableArtifactsList />
      </div>
    </div>
  );
};

const DashboardHeader = ({ onRefresh, onExport }) => (
  <div className="bg-white border-b border-gray-200 px-6 py-4">
    <div className="flex items-center justify-between">
      <div>
        <h1 className="text-2xl font-bold text-gray-900 flex items-center">
          <Shield className="w-8 h-8 text-blue-600 mr-3" />
          Security Scan Dashboard
        </h1>
        <p className="text-sm text-gray-500 mt-1">
          Comprehensive vulnerability analysis and security monitoring
        </p>
      </div>
      <div className="flex space-x-3">
        <button
          onClick={onRefresh}
          className="flex items-center space-x-2 px-4 py-2 text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition"
        >
          <RefreshCw className="w-4 h-4" />
          <span>Refresh</span>
        </button>
        <button
          onClick={onExport}
          className="flex items-center space-x-2 px-4 py-2 text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition"
        >
          <Download className="w-4 h-4" />
          <span>Export Report</span>
        </button>
      </div>
    </div>
  </div>
);

const LoadingState = () => (
  <div className="flex items-center justify-center min-h-screen">
    <div className="text-center">
      <RefreshCw className="w-12 h-12 text-blue-600 animate-spin mx-auto mb-4" />
      <p className="text-gray-600">Loading security dashboard...</p>
    </div>
  </div>
);

const ErrorState = ({ error, onRetry }) => (
  <div className="flex items-center justify-center min-h-screen">
    <div className="text-center">
      <p className="text-red-600 mb-4">{error}</p>
      <button onClick={onRetry} className="px-4 py-2 bg-blue-600 text-white rounded-lg">
        Retry
      </button>
    </div>
  </div>
);

export default SecurityDashboard;