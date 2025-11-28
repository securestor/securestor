import React, { useState, useEffect, useCallback } from 'react';
import { Search, Filter, Download, Trash2, Shield, Tag, Calendar, User, FileText, Database, ChevronDown, ChevronUp, AlertTriangle, Clock, Eye, Copy, Archive, RefreshCw } from 'lucide-react';
import ArtifactAPI from '../../../services/api/artifactAPI';
import VulnerabilityReportModal from './VulnerabilityReportModal';
import PropertiesDialog from './PropertiesDialog';
import { useToast } from '../../../context/ToastContext';
import { getAPIBaseURL } from '../../../constants/apiConfig';
import { securityScanService } from '../../security/services/securityScanService';
import { useTranslation } from '../../../hooks/useTranslation';

const API_BASE_URL =
  process.env.SECURESTOR_APP_API_URL || getAPIBaseURL() + "/api/v1";


const ArtifactsManagement = () => {
  const { t } = useTranslation('artifacts');
  const [artifacts, setArtifacts] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [filter, setFilter] = useState({
    limit: 50,
    offset: 0,
    search: ''
  });
  const [selectedArtifact, setSelectedArtifact] = useState(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [showFilters, setShowFilters] = useState(false);
  const [activeTab, setActiveTab] = useState('all');
  const [vulnerabilityModalOpen, setVulnerabilityModalOpen] = useState(false);
  const [selectedScanId, setSelectedScanId] = useState(null);
  const [propertiesDialogOpen, setPropertiesDialogOpen] = useState(false);
  const [propertiesArtifact, setPropertiesArtifact] = useState(null);
  
  // Toast notifications
  const { showError, showWarning, showSuccess } = useToast();

  // Utility function to format dates safely
  const formatDate = (dateString) => {
    if (!dateString) return 'N/A';
    try {
      const date = new Date(dateString);
      if (isNaN(date.getTime())) return 'Invalid Date';
      return date.toLocaleDateString();
    } catch (error) {
      return 'Invalid Date';
    }
  };

  const loadArtifacts = useCallback(async () => {
    try {
      setLoading(true);
      const data = await ArtifactAPI.fetchArtifacts(filter);
      
      // Parse vulnerability counts from compliance security_scan field
      const parseVulnerabilities = (securityScan) => {
        if (!securityScan) return { critical: 0, high: 0, medium: 0, low: 0 };
        
        // Parse format: "Critical: 13, High: 48, Medium: 9, Low: 0"
        const critical = (securityScan.match(/Critical:\s*(\d+)/i) || [0, 0])[1];
        const high = (securityScan.match(/High:\s*(\d+)/i) || [0, 0])[1];
        const medium = (securityScan.match(/Medium:\s*(\d+)/i) || [0, 0])[1];
        const low = (securityScan.match(/Low:\s*(\d+)/i) || [0, 0])[1];
        
        return {
          critical: parseInt(critical) || 0,
          high: parseInt(high) || 0,
          medium: parseInt(medium) || 0,
          low: parseInt(low) || 0
        };
      };
      
      // Enhanced compliance data with policy integration
      const enrichComplianceData = (compliance) => {
        if (!compliance) {
          return { 
            status: 'pending', 
            score: 0, 
            policies: [],
            gdpr_status: 'pending',
            retention_status: 'pending',
            legal_hold: false,
            data_locality: 'unknown',
            encryption_status: 'pending'
          };
        }

        return {
          ...compliance,
          // Map legacy status to new comprehensive status
          status: compliance.status || 'pending',
          score: compliance.score || 0,
          policies: compliance.policies || [],
          gdpr_status: compliance.gdpr_status || 'pending',
          retention_status: compliance.retention_status || 'pending',
          legal_hold: compliance.legal_hold || false,
          data_locality: compliance.data_locality || 'unknown',
          encryption_status: compliance.encryption_status || 'pending',
          // Enhanced compliance details
          policy_violations: compliance.policy_violations || [],
          audit_trail: compliance.audit_trail || [],
          last_policy_check: compliance.last_policy_check || null
        };
      };

      // Add default structure for missing properties with enhanced compliance
      const artifactsWithDefaults = (data.artifacts || []).map(artifact => ({
        ...artifact,
        compliance: enrichComplianceData(artifact.compliance),
        vulnerabilities: parseVulnerabilities(artifact.compliance?.security_scan),
        tags: artifact.tags || []
      }));
      
      setArtifacts(artifactsWithDefaults);
      setError(null);
    } catch (err) {
      setError(err.message);
      console.error('Failed to load artifacts:', err);
    } finally {
      setLoading(false);
    }
  }, [filter]);

  useEffect(() => {
    loadArtifacts();
  }, [loadArtifacts]);

  const handleDelete = async (artifactId) => {
    if (!window.confirm(t('messages.deleteConfirm'))) {
      return;
    }
    try {
      await ArtifactAPI.deleteArtifact(artifactId);
      loadArtifacts();
      showSuccess(t('messages.deletedSuccess'));
    } catch (err) {
      showError(t('messages.deleteFailed') + ': ' + err.message);
    }
  };

  const handleDownload = async (artifactId, artifactName, artifactVersion) => {
    try {
      // Pass a fallback filename in case backend doesn't have original_filename
      const fallbackFilename = `${artifactName}-${artifactVersion}`;
      await ArtifactAPI.downloadArtifact(artifactId, fallbackFilename);
      
      // Reload artifacts to get updated download count
      loadArtifacts();
      showSuccess(t('messages.downloadedSuccess'));
    } catch (err) {
      showError(t('messages.downloadFailed') + ': ' + err.message);
    }
  };

  // Helper function for copying checksum
  const handleCopyChecksum = async (checksum) => {
    try {
      if (navigator.clipboard && navigator.clipboard.writeText) {
        await navigator.clipboard.writeText(checksum);
        showSuccess(t('messages.checksumCopied'));
      } else {
        // Fallback for older browsers
        const textArea = document.createElement('textarea');
        textArea.value = checksum;
        document.body.appendChild(textArea);
        textArea.select();
        document.execCommand('copy');
        document.body.removeChild(textArea);
        showSuccess(t('messages.checksumCopied'));
      }
    } catch (err) {
      showError(t('messages.checksumCopyFailed'));
    }
  };

  // Comprehensive compliance action handlers
  const handleGDPRErasure = async (artifactId) => {
    if (!window.confirm(t('messages.gdprErasureConfirm'))) {
      return;
    }
    try {
      const response = await fetch('/api/v1/compliance/data-erasure', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-User-ID': 'admin'
        },
        body: JSON.stringify({
          artifact_ids: [artifactId],
          reason: 'GDPR Right to Erasure Request from UI'
        })
      });

      if (response.ok) {
        showSuccess(t('messages.gdprErasureSuccess'));
        loadArtifacts(); // Refresh to show updated status
      } else {
        showWarning(t('messages.gdprErasureConfiguring'));
      }
    } catch (error) {
      showWarning(t('messages.gdprErasureConfiguring'));
    }
  };

  const handleLegalHold = async (artifactId) => {
    if (!window.confirm(t('messages.legalHoldConfirm'))) {
      return;
    }
    try {
      const response = await fetch('/api/v1/compliance/legal-holds', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-User-ID': 'admin'
        },
        body: JSON.stringify({
          artifact_ids: [artifactId],
          reason: 'Legal hold placed from artifact management UI',
          case_name: 'UI Legal Hold'
        })
      });

      if (response.ok) {
        showSuccess(t('messages.legalHoldSuccess'));
        loadArtifacts(); // Refresh to show updated status
      } else {
        showWarning(t('messages.legalHoldConfiguring'));
      }
    } catch (error) {
      showWarning(t('messages.legalHoldConfiguring'));
    }
  };

  const handlePolicyCheck = async (artifactId) => {
    showWarning(t('messages.policyCheckSoon'));
  };

  const handleAuditReport = async (artifactId) => {
    try {
      const response = await fetch(`/api/v1/compliance/artifacts/${artifactId}/report`, {
        headers: {
          'X-User-ID': 'admin'
        }
      });

      if (response.ok) {
        const report = await response.json();
        // This would typically open a detailed audit report modal
        showSuccess(t('messages.auditReportSuccess', { count: report.events_count }));
      } else {
        showError(t('messages.auditReportFailed'));
      }
    } catch (error) {
      showError(t('messages.auditReportError'));
    }
  };

  if (loading) {
    return <div className="flex items-center justify-center min-h-screen">
      <div className="text-gray-600">{t('common:status.loading')}</div>
    </div>;
  }

  if (error) {
    return <div className="flex items-center justify-center min-h-screen">
      <div className="text-red-600">{t('common:status.error')}: {error}</div>
    </div>;
  }

    // Filter artifacts based on search and filters
  const filteredArtifacts = artifacts.filter(artifact => {
    const matchesSearch = artifact.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
                         artifact.version.toLowerCase().includes(searchTerm.toLowerCase()) ||
                         (artifact.tags && artifact.tags.some(tag => tag.toLowerCase().includes(searchTerm.toLowerCase())));
    
    return matchesSearch;
  });


  const getComplianceColor = (status) => {
    switch(status) {
      case 'compliant': return 'text-green-600 bg-green-100';
      case 'review': return 'text-yellow-600 bg-yellow-100';
      case 'non-compliant': return 'text-red-600 bg-red-100';
      case 'policy-violation': return 'text-red-700 bg-red-200 border border-red-300';
      case 'pending-review': return 'text-orange-600 bg-orange-100';
      case 'under-legal-hold': return 'text-purple-600 bg-purple-100 border border-purple-300';
      case 'gdpr-erasure-requested': return 'text-indigo-600 bg-indigo-100';
      case 'retention-expired': return 'text-red-500 bg-red-50 border border-red-200';
      default: return 'text-gray-600 bg-gray-100';
    }
  };

  // Enhanced compliance status check with policy validation
  const getComplianceStatus = (compliance) => {
    if (!compliance) return 'pending';

    // Check for critical compliance issues first
    if (compliance.legal_hold) return 'under-legal-hold';
    if (compliance.policy_violations && compliance.policy_violations.length > 0) return 'policy-violation';
    if (compliance.gdpr_status === 'erasure-requested') return 'gdpr-erasure-requested';
    if (compliance.retention_status === 'expired') return 'retention-expired';
    
    // Return the main compliance status
    return compliance.status || 'pending';
  };

  // Comprehensive compliance indicator component
  const ComplianceIndicator = ({ compliance }) => {
    const status = getComplianceStatus(compliance);
    const colorClass = getComplianceColor(status);
    
    const getStatusIcon = () => {
      if (compliance.legal_hold) return '🔒';
      if (compliance.policy_violations && compliance.policy_violations.length > 0) return '⚠️';
      if (compliance.gdpr_status === 'erasure-requested') return '🗑️';
      if (compliance.retention_status === 'expired') return '⏰';
      if (status === 'compliant') return '✅';
      if (status === 'non-compliant') return '❌';
      return '⏳';
    };

    return (
      <div className="flex items-center space-x-2">
        <span className={`px-2 py-1 text-xs font-medium rounded ${colorClass}`}>
          <span className="mr-1">{getStatusIcon()}</span>
          {status.replace('-', ' ')}
        </span>
        {compliance.policies && compliance.policies.length > 0 && (
          <span className="text-xs text-gray-500" title={`${compliance.policies.length} policies applied`}>
            📋 {compliance.policies.length}
          </span>
        )}
      </div>
    );
  };

  const getVulnerabilityColor = (count, severity) => {
    if (count === 0) return 'text-gray-400';
    switch(severity) {
      case 'critical': return 'text-red-600';
      case 'high': return 'text-orange-600';
      case 'medium': return 'text-yellow-600';
      case 'low': return 'text-blue-600';
      default: return 'text-gray-600';
    }
  };

  const handleViewVulnerabilityReport = async (artifact) => {
    try {
      // Get all scans using the authenticated service (returns array directly)
      const scans = await securityScanService.getAllScans();
      const artifactScans = scans?.filter(scan => scan.artifact_id === artifact.id) || [];
      
      if (artifactScans.length > 0) {
        // Get the most recent completed scan
        const completedScans = artifactScans.filter(scan => scan.status === 'completed');
        
        if (completedScans.length > 0) {
          // Sort by completed_at or started_at and get the most recent
          completedScans.sort((a, b) => {
            const aTime = a.completed_at || a.started_at;
            const bTime = b.completed_at || b.started_at;
            return new Date(bTime) - new Date(aTime);
          });
          
          setSelectedScanId(completedScans[0].id);
          setVulnerabilityModalOpen(true);
          return;
        } else {
          // Check if there are any scans in progress
          const runningScan = artifactScans.find(scan => scan.status === 'running' || scan.status === 'initiated');
          if (runningScan) {
            showWarning(t('messages.scanInProgress'));
            return;
          }
        }
      }
      
      showWarning(t('messages.noScanResults'));
    } catch (error) {
      console.error('Error finding scan for artifact:', error);
      showError(t('messages.scanLoadFailed'));
    }
  };

  return (
    <div className="h-full w-full bg-gray-50 flex flex-col">
      {/* Header */}
      <div className="bg-white border-b border-gray-200 px-6 py-4 flex-shrink-0 w-full">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h1 className="text-2xl font-bold text-gray-900">{t('title')}</h1>
            <p className="text-sm text-gray-500 mt-1">
              {t('list.count', { count: filteredArtifacts.length })}
            </p>
          </div>
          <div className="flex space-x-3">
            <button className="flex items-center space-x-2 px-4 py-2 text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition">
              <RefreshCw className="w-4 h-4" />
              <span>{t('common:buttons.refresh')}</span>
            </button>
            <button className="flex items-center space-x-2 px-4 py-2 text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition">
              <Download className="w-4 h-4" />
              <span>{t('common:buttons.export')}</span>
            </button>
          </div>
        </div>

        {/* Advanced Search */}
        <div className="flex items-center space-x-3">
          <div className="flex-1 relative">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-5 h-5 text-gray-400" />
            <input
              type="text"
              placeholder={t('list.searchPlaceholder')}
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full pl-10 pr-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>
          <button
            onClick={() => setShowFilters(!showFilters)}
            className={`flex items-center space-x-2 px-4 py-3 border rounded-lg transition ${
              showFilters ? 'bg-blue-50 border-blue-500 text-blue-700' : 'bg-white border-gray-300 text-gray-700 hover:bg-gray-50'
            }`}
          >
            <Filter className="w-5 h-5" />
            <span>{t('filters.typeLabel')}</span>
            {showFilters ? <ChevronUp className="w-4 h-4" /> : <ChevronDown className="w-4 h-4" />}
          </button>
        </div>

        {/* Filter Panel */}
        {showFilters && (
          <div className="mt-4 p-4 bg-gray-50 rounded-lg border border-gray-200">
            <div className="grid grid-cols-4 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">{t('filterOptions.type')}</label>
                <div className="space-y-2">
                  {['docker', 'npm', 'maven', 'pypi'].map(type => (
                    <label key={type} className="flex items-center">
                      <input type="checkbox" className="rounded text-blue-600 focus:ring-blue-500" />
                      <span className="ml-2 text-sm text-gray-700 capitalize">{type}</span>
                    </label>
                  ))}
                </div>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">{t('filterOptions.complianceStatus')}</label>
                <div className="space-y-2">
                  {['compliant', 'review', 'non-compliant'].map(status => (
                    <label key={status} className="flex items-center">
                      <input type="checkbox" className="rounded text-blue-600 focus:ring-blue-500" />
                      <span className="ml-2 text-sm text-gray-700 capitalize">{status.replace('-', ' ')}</span>
                    </label>
                  ))}
                </div>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">{t('filterOptions.dateRange')}</label>
                <select className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500">
                  <option>{t('filterOptions.allTime')}</option>
                  <option>{t('filterOptions.last24Hours')}</option>
                  <option>{t('filterOptions.last7Days')}</option>
                  <option>{t('filterOptions.last30Days')}</option>
                  <option>{t('filterOptions.customRange')}</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">{t('filterOptions.vulnerabilities')}</label>
                <select className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500">
                  <option>{t('filterOptions.all')}</option>
                  <option>{t('filterOptions.hasCritical')}</option>
                  <option>{t('filterOptions.hasHigh')}</option>
                  <option>{t('filterOptions.noVulnerabilities')}</option>
                </select>
              </div>
            </div>
            <div className="flex justify-end space-x-2 mt-4 pt-4 border-t border-gray-200">
              <button className="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition">
                {t('filters.clearFilters')}
              </button>
              <button className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition">
                {t('common:buttons.apply')}
              </button>
            </div>
          </div>
        )}
      </div>

      {/* Tabs */}
      <div className="bg-white border-b border-gray-200 px-6 flex-shrink-0 w-full">
        <div className="flex space-x-8">
          {[
            { id: 'all', label: t('tabs.all'), count: filteredArtifacts.length },
            { id: 'compliant', label: t('tabs.compliant'), count: filteredArtifacts.filter(a => getComplianceStatus(a.compliance) === 'compliant').length },
            { id: 'policy-violation', label: t('tabs.policyViolation'), count: filteredArtifacts.filter(a => getComplianceStatus(a.compliance) === 'policy-violation').length },
            { id: 'legal-hold', label: t('tabs.legalHold'), count: filteredArtifacts.filter(a => a.compliance?.legal_hold).length },
            { id: 'review', label: t('tabs.review'), count: filteredArtifacts.filter(a => ['review', 'pending-review'].includes(getComplianceStatus(a.compliance))).length },
            { id: 'vulnerable', label: t('tabs.vulnerable'), count: filteredArtifacts.filter(a => (a.vulnerabilities?.critical || 0) > 0 || (a.vulnerabilities?.high || 0) > 0).length }
          ].map(tab => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={`py-4 px-2 border-b-2 transition ${
                activeTab === tab.id
                  ? 'border-blue-500 text-blue-600 font-medium'
                  : 'border-transparent text-gray-500 hover:text-gray-700'
              }`}
            >
              {tab.label}
              <span className={`ml-2 px-2 py-0.5 text-xs rounded-full ${
                activeTab === tab.id ? 'bg-blue-100 text-blue-600' : 'bg-gray-100 text-gray-600'
              }`}>
                {tab.count}
              </span>
            </button>
          ))}
        </div>
      </div>

      {/* Content */}
      <div className="p-6 flex-1 overflow-hidden w-full">
        <div className="h-full grid grid-cols-1 lg:grid-cols-3 gap-6 w-full">
          {/* Artifacts List */}
          <div className="lg:col-span-2 space-y-3 overflow-y-auto">
            {filteredArtifacts.map(artifact => (
              <div
                key={artifact.id}
                onClick={() => setSelectedArtifact(artifact)}
                className={`bg-white rounded-lg border p-4 cursor-pointer transition hover:shadow-md ${
                  selectedArtifact?.id === artifact.id ? 'border-blue-500 shadow-md' : 'border-gray-200'
                }`}
              >
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <div className="flex items-center space-x-3 mb-2">
                      <h3 className="font-semibold text-gray-900 text-lg">
                        {artifact.name}:{artifact.version}
                      </h3>
                      <span className="px-2 py-1 text-xs font-medium bg-purple-100 text-purple-800 rounded">
                        {artifact.type}
                      </span>
                      <ComplianceIndicator compliance={artifact.compliance} />
                      <span className="ml-2 px-2 py-1 text-xs bg-blue-50 text-blue-600 rounded" title="Auto-calculated from security scan results">
                        🤖 Auto
                      </span>
                    </div>
                    
                    <div className="flex items-center space-x-4 text-sm text-gray-600 mb-3">
                      <span className="flex items-center">
                        <Database className="w-4 h-4 mr-1" />
                        {String(artifact.repository)}
                      </span>
                      <span className="flex items-center">
                        <User className="w-4 h-4 mr-1" />
                        {String(artifact.uploadedBy)}
                      </span>
                      <span className="flex items-center">
                        <Calendar className="w-4 h-4 mr-1" />
                        {formatDate(artifact.uploaded_at)}
                      </span>
                      <span className="flex items-center">
                        <Download className="w-4 h-4 mr-1" />
                        {String(artifact.downloads)}
                      </span>
                    </div>

                    <div className="flex items-center space-x-6">
                      <div className="flex items-center space-x-2">
                        <span className="text-xs font-medium text-gray-600">Vulnerabilities:</span>
                        <span className={`text-xs font-semibold ${getVulnerabilityColor(artifact.vulnerabilities.critical, 'critical')}`}>
                          C: {artifact.vulnerabilities.critical}
                        </span>
                        <span className={`text-xs font-semibold ${getVulnerabilityColor(artifact.vulnerabilities.high, 'high')}`}>
                          H: {artifact.vulnerabilities.high}
                        </span>
                        <span className={`text-xs font-semibold ${getVulnerabilityColor(artifact.vulnerabilities.medium, 'medium')}`}>
                          M: {artifact.vulnerabilities.medium}
                        </span>
                        <span className={`text-xs font-semibold ${getVulnerabilityColor(artifact.vulnerabilities.low, 'low')}`}>
                          L: {artifact.vulnerabilities.low}
                        </span>
                      </div>
                      <div className="flex items-center space-x-2">
                        <span className="text-xs font-medium text-gray-600">Score:</span>
                        <span className={`text-xs font-semibold ${
                          artifact.compliance.score >= 90 ? 'text-green-600' :
                          artifact.compliance.score >= 70 ? 'text-yellow-600' : 'text-red-600'
                        }`}>
                          {artifact.compliance.score}/100
                        </span>
                      </div>
                    </div>

                    {artifact.tags && artifact.tags.length > 0 && (
                      <div className="flex items-center space-x-2 mt-3">
                        {artifact.tags.map(tag => (
                          <span key={tag} className="px-2 py-1 text-xs bg-gray-100 text-gray-700 rounded">
                            {tag}
                          </span>
                        ))}
                      </div>
                    )}
                  </div>

                  <div className="flex items-center space-x-2 ml-4">
                    <button 
                      className="p-2 hover:bg-gray-100 rounded-lg transition"
                      onClick={(e) => {
                        e.stopPropagation();
                        handleDownload(artifact.id, artifact.name, artifact.version);
                      }}
                      title="Download Artifact"
                    >
                      <Download className="w-4 h-4 text-gray-600" />
                    </button>
                    <button 
                      className="p-2 hover:bg-blue-50 rounded-lg transition"
                      onClick={(e) => {
                        e.stopPropagation();
                        setPropertiesArtifact(artifact);
                        setPropertiesDialogOpen(true);
                      }}
                      title="Manage Properties"
                    >
                      <Tag className="w-4 h-4 text-blue-600" />
                    </button>
                    <button className="p-2 hover:bg-gray-100 rounded-lg transition">
                      <Copy className="w-4 h-4 text-gray-600" />
                    </button>
                    <button 
                      className="p-2 hover:bg-gray-100 rounded-lg transition"
                      onClick={(e) => {
                        e.stopPropagation();
                        handleDelete(artifact.id);
                      }}
                      title="Delete Artifact"
                    >
                      <Trash2 className="w-4 h-4 text-red-600" />
                    </button>
                  </div>
                </div>
              </div>
            ))}
          </div>

          {/* Details Panel */}
          <div className="lg:col-span-1 overflow-y-auto">
            {selectedArtifact ? (
              <div className="bg-white rounded-lg border border-gray-200 min-h-full">
                <div className="px-6 py-4 border-b border-gray-200">
                  <h3 className="font-semibold text-gray-900">{t('details.title')}</h3>
                </div>
                
                <div className="p-6 space-y-6">
                  {/* Basic Info */}
                  <div>
                    <h4 className="text-sm font-medium text-gray-900 mb-3 flex items-center">
                      <FileText className="w-4 h-4 mr-2" />
                      {t('details.basicInfo')}
                    </h4>
                    <div className="space-y-2 text-sm">
                      <div className="flex justify-between">
                        <span className="text-gray-600">{t('metadata.size')}:</span>
                        <span className="font-medium text-gray-900">{String(selectedArtifact.size)}</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-gray-600">{t('metadata.license')}:</span>
                        <span className="font-medium text-gray-900">{String(selectedArtifact.license)}</span>
                      </div>
                      <div className="space-y-1">
                        <span className="text-gray-600 text-sm">{t('metadata.checksum')}:</span>
                        <div className="flex items-center space-x-2 bg-gray-50 p-2 rounded">
                          <div className="flex-1 min-w-0">
                            <div className="font-mono text-xs text-gray-900 truncate" title={String(selectedArtifact.checksum)}>
                              {selectedArtifact.checksum ? 
                                `${String(selectedArtifact.checksum).substring(0, 20)}...${String(selectedArtifact.checksum).slice(-8)}` 
                                : 'N/A'
                              }
                            </div>
                            <div className="text-xs text-gray-500 mt-1">
                              Click to copy full checksum
                            </div>
                          </div>
                          <button 
                            onClick={() => handleCopyChecksum(selectedArtifact.checksum)}
                            className="p-1 text-gray-400 hover:text-blue-600 transition flex-shrink-0"
                            title="Copy full checksum"
                          >
                            <Copy className="w-3 h-3" />
                          </button>
                        </div>
                      </div>
                    </div>
                  </div>

                  {/* Metadata */}
                  <div>
                    <h4 className="text-sm font-medium text-gray-900 mb-3 flex items-center">
                      <Tag className="w-4 h-4 mr-2" />
                      {t('details.metadata')}
                    </h4>
                    <div className="space-y-2 text-sm">
                      {Object.entries(selectedArtifact.metadata).map(([key, value]) => (
                        <div key={key} className="flex justify-between">
                          <span className="text-gray-600 capitalize">{key.replace(/([A-Z])/g, ' $1')}:</span>
                          <span className="font-medium text-gray-900 text-right max-w-[60%] truncate">
                            {typeof value === 'object' ? JSON.stringify(value) : String(value)}
                          </span>
                        </div>
                      ))}
                    </div>
                  </div>

                  {/* Enhanced Compliance */}
                  <div>
                    <h4 className="text-sm font-medium text-gray-900 mb-3 flex items-center justify-between">
                      <span className="flex items-center">
                        <Shield className="w-4 h-4 mr-2" />
                        {t('compliance.details')}
                      </span>
                      <span className="text-xs px-2 py-1 bg-green-50 text-green-600 rounded" title="Policy-based compliance">
                        📋 Policy
                      </span>
                    </h4>
                    
                    <div className="space-y-4">
                      {/* Main Status */}
                      <div className="flex items-center justify-between">
                        <span className="text-sm text-gray-600">{t('compliance.status')}:</span>
                        <ComplianceIndicator compliance={selectedArtifact.compliance} />
                      </div>

                      {/* Compliance Score */}
                      <div className="flex items-center justify-between">
                        <span className="text-sm text-gray-600">{t('compliance.score')}:</span>
                        <div className="flex items-center space-x-2">
                          <div className="w-24 h-2 bg-gray-200 rounded-full overflow-hidden">
                            <div
                              className={`h-full ${
                                selectedArtifact.compliance.score >= 90 ? 'bg-green-500' :
                                selectedArtifact.compliance.score >= 70 ? 'bg-yellow-500' : 'bg-red-500'
                              }`}
                              style={{ width: `${selectedArtifact.compliance.score}%` }}
                            />
                          </div>
                          <span className="text-sm font-semibold text-gray-900">
                            {selectedArtifact.compliance.score}%
                          </span>
                        </div>
                      </div>

                      {/* Policy Status Grid */}
                      <div className="grid grid-cols-2 gap-3 text-xs">
                        <div className="flex items-center justify-between">
                          <span className="text-gray-600">{t('compliance.gdpr')}:</span>
                          <span className={`px-2 py-1 rounded ${
                            selectedArtifact.compliance.gdpr_status === 'compliant' ? 'bg-green-100 text-green-700' :
                            selectedArtifact.compliance.gdpr_status === 'erasure-requested' ? 'bg-red-100 text-red-700' :
                            'bg-yellow-100 text-yellow-700'
                          }`}>
                            {selectedArtifact.compliance.gdpr_status}
                          </span>
                        </div>
                        <div className="flex items-center justify-between">
                          <span className="text-gray-600">{t('compliance.retention')}:</span>
                          <span className={`px-2 py-1 rounded ${
                            selectedArtifact.compliance.retention_status === 'compliant' ? 'bg-green-100 text-green-700' :
                            selectedArtifact.compliance.retention_status === 'expired' ? 'bg-red-100 text-red-700' :
                            'bg-yellow-100 text-yellow-700'
                          }`}>
                            {selectedArtifact.compliance.retention_status}
                          </span>
                        </div>
                        <div className="flex items-center justify-between">
                          <span className="text-gray-600">{t('compliance.legalHold')}:</span>
                          <span className={`px-2 py-1 rounded ${
                            selectedArtifact.compliance.legal_hold ? 'bg-purple-100 text-purple-700' : 'bg-gray-100 text-gray-700'
                          }`}>
                            {selectedArtifact.compliance.legal_hold ? t('compliance.active') : t('compliance.none')}
                          </span>
                        </div>
                        <div className="flex items-center justify-between">
                          <span className="text-gray-600">{t('compliance.locality')}:</span>
                          <span className="px-2 py-1 rounded bg-blue-100 text-blue-700">
                            {selectedArtifact.compliance.data_locality}
                          </span>
                        </div>
                      </div>

                      {/* Policy Violations */}
                      {selectedArtifact.compliance.policy_violations && selectedArtifact.compliance.policy_violations.length > 0 && (
                        <div className="p-3 bg-red-50 border border-red-200 rounded-lg">
                          <p className="text-xs font-medium text-red-800 mb-2">⚠️ {t('compliance.policyViolations')}:</p>
                          <ul className="text-xs text-red-700 space-y-1">
                            {selectedArtifact.compliance.policy_violations.map((violation, index) => (
                              <li key={index}>• {violation}</li>
                            ))}
                          </ul>
                        </div>
                      )}

                      {/* Compliance Actions */}
                      <div className="pt-3 border-t border-gray-200">
                        <p className="text-xs font-medium text-gray-900 mb-2">{t('compliance.actions')}:</p>
                        <div className="grid grid-cols-2 gap-2">
                          <button 
                            onClick={() => handleGDPRErasure(selectedArtifact.id)}
                            className="px-3 py-2 text-xs bg-red-100 text-red-700 rounded hover:bg-red-200 transition"
                            disabled={selectedArtifact.compliance.legal_hold}
                          >
                            🗑️ {t('compliance.gdprErasure')}
                          </button>
                          <button 
                            onClick={() => handleLegalHold(selectedArtifact.id)}
                            className="px-3 py-2 text-xs bg-purple-100 text-purple-700 rounded hover:bg-purple-200 transition"
                          >
                            🔒 {t('compliance.legalHold')}
                          </button>
                          <button 
                            onClick={() => handlePolicyCheck(selectedArtifact.id)}
                            className="px-3 py-2 text-xs bg-blue-100 text-blue-700 rounded hover:bg-blue-200 transition"
                          >
                            📋 {t('compliance.policyCheck')}
                          </button>
                          <button 
                            onClick={() => handleAuditReport(selectedArtifact.id)}
                            className="px-3 py-2 text-xs bg-green-100 text-green-700 rounded hover:bg-green-200 transition"
                          >
                            📊 {t('compliance.auditReport')}
                          </button>
                        </div>
                      </div>

                      {/* Vulnerability Summary */}
                      <div className="pt-3 border-t border-gray-200">
                        <p className="text-xs font-medium text-gray-900 mb-2">{t('details.securityScan')}:</p>
                        <div className="grid grid-cols-4 gap-2 text-xs">
                          <div className="text-center p-2 bg-red-50 rounded">
                            <div className="font-bold text-red-600">{selectedArtifact.vulnerabilities.critical}</div>
                            <div className="text-red-500">{t('vulnerabilities.critical')}</div>
                          </div>
                          <div className="text-center p-2 bg-orange-50 rounded">
                            <div className="font-bold text-orange-600">{selectedArtifact.vulnerabilities.high}</div>
                            <div className="text-orange-500">{t('vulnerabilities.high')}</div>
                          </div>
                          <div className="text-center p-2 bg-yellow-50 rounded">
                            <div className="font-bold text-yellow-600">{selectedArtifact.vulnerabilities.medium}</div>
                            <div className="text-yellow-500">{t('vulnerabilities.medium')}</div>
                          </div>
                          <div className="text-center p-2 bg-blue-50 rounded">
                            <div className="font-bold text-blue-600">{selectedArtifact.vulnerabilities.low}</div>
                            <div className="text-blue-500">{t('vulnerabilities.low')}</div>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>

                  {/* Vulnerabilities */}
                  <div>
                    <h4 className="text-sm font-medium text-gray-900 mb-3 flex items-center">
                      <AlertTriangle className="w-4 h-4 mr-2" />
                      {t('details.securityScan')}
                    </h4>
                    <div className="space-y-2">
                      {[
                        { label: t('vulnerabilities.critical'), count: selectedArtifact.vulnerabilities.critical, color: 'red' },
                        { label: t('vulnerabilities.high'), count: selectedArtifact.vulnerabilities.high, color: 'orange' },
                        { label: t('vulnerabilities.medium'), count: selectedArtifact.vulnerabilities.medium, color: 'yellow' },
                        { label: t('vulnerabilities.low'), count: selectedArtifact.vulnerabilities.low, color: 'blue' }
                      ].map(vuln => (
                        <div key={vuln.label} className="flex items-center justify-between">
                          <span className="text-sm text-gray-600">{vuln.label}:</span>
                          <span className={`px-2 py-1 text-xs font-semibold bg-${vuln.color}-100 text-${vuln.color}-700 rounded`}>
                            {vuln.count} {vuln.count === 1 ? t('issues.issue') : t('issues.issues')}
                          </span>
                        </div>
                      ))}
                    </div>
                    <button 
                      className="w-full mt-3 px-4 py-2 text-sm font-medium text-blue-600 bg-blue-50 rounded-lg hover:bg-blue-100 transition"
                      onClick={() => handleViewVulnerabilityReport(selectedArtifact)}
                    >
                      {t('details.viewFullReport')}
                    </button>
                  </div>

                  {/* Actions */}
                  <div className="pt-4 border-t border-gray-200">
                    <button 
                      className="w-full mb-2 px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 transition"
                      onClick={() => handleDownload(selectedArtifact.id, selectedArtifact.name, selectedArtifact.version)}
                    >
                      {t('actions.download')}
                    </button>
                    <button className="w-full mb-2 px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition">
                      {t('details.copyInstallCommand')}
                    </button>
                    <button 
                      className="w-full px-4 py-2 text-sm font-medium text-red-600 bg-white border border-red-300 rounded-lg hover:bg-red-50 transition"
                      onClick={() => handleDelete(selectedArtifact.id)}
                    >
                      {t('actions.delete')}
                    </button>
                  </div>
                </div>
              </div>
            ) : (
              <div className="bg-white rounded-lg border border-gray-200 min-h-full flex flex-col">
                <div className="px-6 py-4 border-b border-gray-200">
                  <h3 className="font-semibold text-gray-900">{t('details.quickActions')}</h3>
                </div>
                
                <div className="p-6 flex-1 flex flex-col justify-between">
                  <div className="space-y-6">
                    {/* Quick Stats */}
                    <div>
                      <h4 className="text-sm font-medium text-gray-900 mb-3 flex items-center">
                        <Database className="w-4 h-4 mr-2" />
                        {t('details.repositoryOverview')}
                      </h4>
                      <div className="space-y-2 text-sm">
                        <div className="flex justify-between">
                          <span className="text-gray-600">{t('stats.totalArtifacts')}:</span>
                          <span className="font-medium text-gray-900">{artifacts.length}</span>
                        </div>
                        <div className="flex justify-between">
                          <span className="text-gray-600">{t('stats.compliant')}:</span>
                          <span className="font-medium text-green-600">
                            {artifacts.filter(a => a.compliance?.status === 'compliant').length}
                          </span>
                        </div>
                        <div className="flex justify-between">
                          <span className="text-gray-600">{t('stats.underReview')}:</span>
                          <span className="font-medium text-yellow-600">
                            {artifacts.filter(a => a.compliance?.status === 'review').length}
                          </span>
                        </div>
                        <div className="flex justify-between">
                          <span className="text-gray-600">{t('stats.criticalIssues')}:</span>
                          <span className="font-medium text-red-600">
                            {artifacts.reduce((sum, a) => sum + (a.vulnerabilities?.critical || 0), 0)}
                          </span>
                        </div>
                      </div>
                    </div>

                    {/* Recent Activity */}
                    <div>
                      <h4 className="text-sm font-medium text-gray-900 mb-3 flex items-center">
                        <Clock className="w-4 h-4 mr-2" />
                        {t('details.recentActivity')}
                      </h4>
                      <div className="space-y-3">
                        {artifacts.slice(0, 3).map(artifact => (
                          <div key={artifact.id} className="flex items-center space-x-3 p-2 rounded-lg">
                            <div className={`w-2 h-2 rounded-full ${
                              artifact.compliance.status === 'compliant' ? 'bg-green-500' :
                              artifact.compliance.status === 'review' ? 'bg-yellow-500' : 'bg-red-500'
                            }`} />
                            <div className="flex-1 min-w-0">
                              <p className="text-sm font-medium text-gray-900 truncate">
                                {String(artifact.name)}:{String(artifact.version)}
                              </p>
                              <p className="text-xs text-gray-500">
                                {formatDate(artifact.uploaded_at)}
                              </p>
                            </div>
                          </div>
                        ))}
                      </div>
                    </div>

                    {/* Quick Actions */}
                    <div>
                      <h4 className="text-sm font-medium text-gray-900 mb-3 flex items-center">
                        <Archive className="w-4 h-4 mr-2" />
                        {t('details.quickActions')}
                      </h4>
                      <div className="space-y-2">
                        <button className="w-full text-left px-3 py-2 text-sm text-gray-700 hover:bg-gray-50 rounded-lg transition flex items-center space-x-2">
                          <Download className="w-4 h-4" />
                          <span>{t('details.bulkDownload')}</span>
                        </button>
                        <button className="w-full text-left px-3 py-2 text-sm text-gray-700 hover:bg-gray-50 rounded-lg transition flex items-center space-x-2">
                          <Shield className="w-4 h-4" />
                          <span>{t('details.runSecurityScan')}</span>
                        </button>
                        <button className="w-full text-left px-3 py-2 text-sm text-gray-700 hover:bg-gray-50 rounded-lg transition flex items-center space-x-2">
                          <Archive className="w-4 h-4" />
                          <span>{t('details.cleanupOldVersions')}</span>
                        </button>
                      </div>
                    </div>
                  </div>

                  {/* Help Section */}
                  <div className="pt-4 border-t border-gray-200">
                    <div className="text-center">
                      <Eye className="w-8 h-8 text-gray-300 mx-auto mb-2" />
                      <p className="text-sm text-gray-500 mb-3">{t('details.selectArtifact')}</p>
                      <div className="text-xs text-gray-400">
                        <p>💡 {t('details.tip')}</p>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Vulnerability Report Modal */}
      <VulnerabilityReportModal 
        open={vulnerabilityModalOpen}
        onClose={() => setVulnerabilityModalOpen(false)}
        scanId={selectedScanId}
        artifactName={selectedArtifact ? `${selectedArtifact.name}:${selectedArtifact.version}` : ''}
      />

      {/* Properties Dialog */}
      <PropertiesDialog
        isOpen={propertiesDialogOpen}
        onClose={() => {
          setPropertiesDialogOpen(false);
          setPropertiesArtifact(null);
        }}
        artifact={propertiesArtifact}
        onPropertiesUpdated={() => {
          // Optionally refresh artifact list
          loadArtifacts();
        }}
      />
    </div>
  );
};

export default ArtifactsManagement;