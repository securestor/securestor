import React, { useState, useEffect } from 'react';
import { Modal } from '../../modals/Modal';
import { Clock, Shield, CheckCircle, XCircle, AlertTriangle } from 'lucide-react';
import scanAPI from '../../../services/api/scanAPI';

export const ScanHistoryModal = ({ isOpen, onClose, artifactId }) => {
  const [scans, setScans] = useState([]);
  const [loading, setLoading] = useState(true);
  const [total, setTotal] = useState(0);
  const [currentPage, setCurrentPage] = useState(1);
  const [selectedScan, setSelectedScan] = useState(null);

  const scansPerPage = 10;

  useEffect(() => {
    if (isOpen && artifactId) {
      loadScanHistory();
    }
  }, [isOpen, artifactId, currentPage]);

  const loadScanHistory = async () => {
    try {
      setLoading(true);
      const offset = (currentPage - 1) * scansPerPage;
      const response = await scanAPI.getScanHistory(artifactId, scansPerPage, offset);
      setScans(response.scans || []);
      setTotal(response.total_count || 0);
    } catch (error) {
      console.error('Failed to load scan history:', error);
      setScans([]);
    } finally {
      setLoading(false);
    }
  };

  const getStatusIcon = (status) => {
    switch (status) {
      case 'completed':
        return <CheckCircle className="w-5 h-5 text-green-600" />;
      case 'completed_with_errors':
        return <AlertTriangle className="w-5 h-5 text-yellow-600" />;
      case 'failed':
        return <XCircle className="w-5 h-5 text-red-600" />;
      case 'cancelled':
        return <XCircle className="w-5 h-5 text-gray-600" />;
      case 'running':
      case 'initiated':
        return <Clock className="w-5 h-5 text-blue-600 animate-pulse" />;
      default:
        return <Clock className="w-5 h-5 text-gray-600" />;
    }
  };

  const getStatusColor = (status) => {
    const colors = {
      completed: 'text-green-600 bg-green-100',
      completed_with_errors: 'text-yellow-600 bg-yellow-100',
      failed: 'text-red-600 bg-red-100',
      cancelled: 'text-gray-600 bg-gray-100',
      running: 'text-blue-600 bg-blue-100',
      initiated: 'text-blue-600 bg-blue-100'
    };
    return colors[status] || 'text-gray-600 bg-gray-100';
  };

  const formatDate = (dateString) => {
    return new Date(dateString).toLocaleString();
  };

  const formatDuration = (seconds) => {
    if (!seconds) return 'N/A';
    
    if (seconds < 60) {
      return `${seconds}s`;
    } else if (seconds < 3600) {
      return `${Math.floor(seconds / 60)}m ${seconds % 60}s`;
    } else {
      const hours = Math.floor(seconds / 3600);
      const minutes = Math.floor((seconds % 3600) / 60);
      return `${hours}h ${minutes}m`;
    }
  };

  const totalPages = Math.ceil(total / scansPerPage);

  return (
    <Modal isOpen={isOpen} onClose={onClose} title="Scan History" size="xl">
      <div className="space-y-6">
        {/* Summary */}
        <div className="flex items-center justify-between">
          <div className="flex items-center">
            <Shield className="w-5 h-5 text-blue-600 mr-2" />
            <span className="text-lg font-medium text-gray-900">
              Security Scan History ({total} total)
            </span>
          </div>
          {total > scansPerPage && (
            <div className="text-sm text-gray-500">
              Page {currentPage} of {totalPages}
            </div>
          )}
        </div>

        {/* Scan List */}
        {loading ? (
          <div className="flex items-center justify-center p-8">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
            <span className="ml-2 text-gray-600">Loading scan history...</span>
          </div>
        ) : scans.length > 0 ? (
          <div className="space-y-4">
            {scans.map((scan) => (
              <div
                key={scan.id}
                className="border border-gray-200 rounded-lg p-4 hover:shadow-md transition-shadow cursor-pointer"
                onClick={() => setSelectedScan(selectedScan?.id === scan.id ? null : scan)}
              >
                <div className="flex items-center justify-between">
                  <div className="flex items-center space-x-3">
                    {getStatusIcon(scan.status)}
                    <div>
                      <div className="font-medium text-gray-900">
                        Scan #{scan.id}
                      </div>
                      <div className="text-sm text-gray-500">
                        {scan.scan_type} • {scan.priority} priority
                      </div>
                    </div>
                  </div>

                  <div className="flex items-center space-x-4">
                    <div className="text-right">
                      <div className="text-sm font-medium text-gray-900">
                        {formatDate(scan.started_at)}
                      </div>
                      <div className="text-sm text-gray-500">
                        Duration: {formatDuration(scan.duration)}
                      </div>
                    </div>
                    <span className={`px-2 py-1 rounded-full text-xs font-medium ${getStatusColor(scan.status)}`}>
                      {scan.status.replace('_', ' ').toUpperCase()}
                    </span>
                  </div>
                </div>

                {/* Expanded Details */}
                {selectedScan?.id === scan.id && (
                  <div className="mt-4 pt-4 border-t border-gray-200">
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                      <div>
                        <h5 className="font-medium text-gray-900 mb-2">Scan Configuration</h5>
                        <div className="space-y-1 text-sm">
                          <div>Vulnerability Scan: {scan.vulnerability_scan ? 'Yes' : 'No'}</div>
                          <div>Malware Scan: {scan.malware_scan ? 'Yes' : 'No'}</div>
                          <div>License Scan: {scan.license_scan ? 'Yes' : 'No'}</div>
                          <div>Dependency Scan: {scan.dependency_scan ? 'Yes' : 'No'}</div>
                        </div>
                      </div>

                      <div>
                        <h5 className="font-medium text-gray-900 mb-2">Details</h5>
                        <div className="space-y-1 text-sm">
                          <div>Initiated by: {scan.initiated_by}</div>
                          <div>Started: {formatDate(scan.started_at)}</div>
                          {scan.completed_at && (
                            <div>Completed: {formatDate(scan.completed_at)}</div>
                          )}
                          {scan.error_message && (
                            <div className="text-red-600">Error: {scan.error_message}</div>
                          )}
                        </div>
                      </div>
                    </div>

                    {/* Scan Results Summary */}
                    {scan.results && (
                      <div className="mt-4">
                        <h5 className="font-medium text-gray-900 mb-2">Results Summary</h5>
                        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                          <div className="text-center p-3 bg-gray-50 rounded">
                            <div className="text-2xl font-bold text-gray-900">
                              {scan.results.overall_score}
                            </div>
                            <div className="text-sm text-gray-500">Security Score</div>
                          </div>
                          
                          {scan.results.vulnerability_results && (
                            <div className="text-center p-3 bg-gray-50 rounded">
                              <div className="text-2xl font-bold text-red-600">
                                {scan.results.vulnerability_results.total_found}
                              </div>
                              <div className="text-sm text-gray-500">Vulnerabilities</div>
                            </div>
                          )}

                          {scan.results.malware_results && (
                            <div className="text-center p-3 bg-gray-50 rounded">
                              <div className="text-2xl font-bold text-purple-600">
                                {scan.results.malware_results.threats_found}
                              </div>
                              <div className="text-sm text-gray-500">Threats</div>
                            </div>
                          )}

                          <div className="text-center p-3 bg-gray-50 rounded">
                            <div className={`text-2xl font-bold ${
                              scan.results.risk_level === 'low' ? 'text-green-600' :
                              scan.results.risk_level === 'medium' ? 'text-yellow-600' :
                              scan.results.risk_level === 'high' ? 'text-orange-600' : 'text-red-600'
                            }`}>
                              {scan.results.risk_level?.toUpperCase()}
                            </div>
                            <div className="text-sm text-gray-500">Risk Level</div>
                          </div>
                        </div>
                      </div>
                    )}
                  </div>
                )}
              </div>
            ))}
          </div>
        ) : (
          <div className="text-center py-8">
            <Shield className="w-12 h-12 text-gray-400 mx-auto mb-4" />
            <h3 className="text-lg font-medium text-gray-900 mb-2">No Scan History</h3>
            <p className="text-gray-500">No security scans have been performed on this artifact</p>
          </div>
        )}

        {/* Pagination */}
        {totalPages > 1 && (
          <div className="flex items-center justify-between border-t border-gray-200 pt-4">
            <button
              onClick={() => setCurrentPage(prev => Math.max(prev - 1, 1))}
              disabled={currentPage === 1}
              className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              Previous
            </button>

            <div className="flex items-center space-x-2">
              {[...Array(totalPages)].map((_, index) => {
                const page = index + 1;
                return (
                  <button
                    key={page}
                    onClick={() => setCurrentPage(page)}
                    className={`px-3 py-2 text-sm font-medium rounded-md ${
                      currentPage === page
                        ? 'bg-blue-600 text-white'
                        : 'text-gray-700 bg-white border border-gray-300 hover:bg-gray-50'
                    }`}
                  >
                    {page}
                  </button>
                );
              })}
            </div>

            <button
              onClick={() => setCurrentPage(prev => Math.min(prev + 1, totalPages))}
              disabled={currentPage === totalPages}
              className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              Next
            </button>
          </div>
        )}
      </div>
    </Modal>
  );
};