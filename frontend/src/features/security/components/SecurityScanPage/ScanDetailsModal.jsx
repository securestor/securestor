import React from 'react';
import { X, Clock, CheckCircle } from 'lucide-react';

export const ScanDetailsModal = ({ scan, onClose }) => {
  const formatDate = (dateString) => {
    return new Date(dateString).toLocaleString();
  };

  const formatDuration = (seconds) => {
    if (!seconds) return 'N/A';
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins}m ${secs}s`;
  };

  const getRiskLevelColor = (level) => {
    switch (level) {
      case 'critical': return 'text-red-600 bg-red-100';
      case 'high': return 'text-orange-600 bg-orange-100';
      case 'medium': return 'text-yellow-600 bg-yellow-100';
      case 'low': return 'text-green-600 bg-green-100';
      default: return 'text-gray-600 bg-gray-100';
    }
  };

  const getSeverityColor = (severity) => {
    switch (severity?.toLowerCase()) {
      case 'critical': return 'text-red-600 bg-red-100';
      case 'high': return 'text-orange-600 bg-orange-100';
      case 'medium': return 'text-yellow-600 bg-yellow-100';
      case 'low': return 'text-green-600 bg-green-100';
      default: return 'text-gray-600 bg-gray-100';
    }
  };

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl max-w-4xl w-full max-h-[90vh] overflow-y-auto mx-4">
        <div className="flex items-center justify-between p-6 border-b sticky top-0 bg-white">
          <h2 className="text-xl font-semibold text-gray-900">
            Scan Details - #{scan.id}
          </h2>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600"
          >
            <X className="h-6 w-6" />
          </button>
        </div>

        <div className="p-6 space-y-6">
          {/* Scan Overview */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="bg-gray-50 p-4 rounded-lg">
              <h3 className="font-medium text-gray-900 mb-2">Scan Information</h3>
              <div className="space-y-1 text-sm">
                <div><span className="text-gray-600">Type:</span> <span className="capitalize">{scan.scan_type}</span></div>
                <div><span className="text-gray-600">Priority:</span> <span className="capitalize">{scan.priority}</span></div>
                <div><span className="text-gray-600">Status:</span> <span className="capitalize">{scan.status}</span></div>
                <div><span className="text-gray-600">Started:</span> {formatDate(scan.started_at)}</div>
                {scan.completed_at && (
                  <div><span className="text-gray-600">Completed:</span> {formatDate(scan.completed_at)}</div>
                )}
                <div><span className="text-gray-600">Duration:</span> {formatDuration(scan.duration)}</div>
              </div>
            </div>

            <div className="bg-gray-50 p-4 rounded-lg">
              <h3 className="font-medium text-gray-900 mb-2">Scan Types</h3>
              <div className="space-y-1 text-sm">
                <div className="flex items-center">
                  {scan.vulnerability_scan ? (
                    <CheckCircle className="h-4 w-4 text-green-500 mr-2" />
                  ) : (
                    <Clock className="h-4 w-4 text-gray-400 mr-2" />
                  )}
                  Vulnerability Scan
                </div>
                <div className="flex items-center">
                  {scan.malware_scan ? (
                    <CheckCircle className="h-4 w-4 text-green-500 mr-2" />
                  ) : (
                    <Clock className="h-4 w-4 text-gray-400 mr-2" />
                  )}
                  Malware Scan
                </div>
                <div className="flex items-center">
                  {scan.license_scan ? (
                    <CheckCircle className="h-4 w-4 text-green-500 mr-2" />
                  ) : (
                    <Clock className="h-4 w-4 text-gray-400 mr-2" />
                  )}
                  License Scan
                </div>
                <div className="flex items-center">
                  {scan.dependency_scan ? (
                    <CheckCircle className="h-4 w-4 text-green-500 mr-2" />
                  ) : (
                    <Clock className="h-4 w-4 text-gray-400 mr-2" />
                  )}
                  Dependency Scan
                </div>
              </div>
            </div>

            {scan.results && (
              <div className="bg-gray-50 p-4 rounded-lg">
                <h3 className="font-medium text-gray-900 mb-2">Overall Results</h3>
                <div className="space-y-2">
                  <div className="text-2xl font-bold text-gray-900">
                    {scan.results.overall_score}/100
                  </div>
                  <div className={`inline-block px-3 py-1 rounded-full text-sm font-medium ${getRiskLevelColor(scan.results.risk_level)}`}>
                    {scan.results.risk_level?.toUpperCase()} RISK
                  </div>
                  <div className="text-sm text-gray-600">
                    {scan.results.summary}
                  </div>
                </div>
              </div>
            )}
          </div>

          {/* Vulnerability Results */}
          {scan.results?.vulnerability_results && (
            <div className="bg-white border rounded-lg p-4">
              <h3 className="text-lg font-semibold text-gray-900 mb-4">Vulnerability Scan Results</h3>
              
              <div className="grid grid-cols-2 md:grid-cols-5 gap-4 mb-6">
                <div className="text-center">
                  <div className="text-2xl font-bold text-gray-900">
                    {scan.results.vulnerability_results.total_found}
                  </div>
                  <div className="text-sm text-gray-600">Total Found</div>
                </div>
                <div className="text-center">
                  <div className="text-2xl font-bold text-red-600">
                    {scan.results.vulnerability_results.critical}
                  </div>
                  <div className="text-sm text-gray-600">Critical</div>
                </div>
                <div className="text-center">
                  <div className="text-2xl font-bold text-orange-600">
                    {scan.results.vulnerability_results.high}
                  </div>
                  <div className="text-sm text-gray-600">High</div>
                </div>
                <div className="text-center">
                  <div className="text-2xl font-bold text-yellow-600">
                    {scan.results.vulnerability_results.medium}
                  </div>
                  <div className="text-sm text-gray-600">Medium</div>
                </div>
                <div className="text-center">
                  <div className="text-2xl font-bold text-green-600">
                    {scan.results.vulnerability_results.low}
                  </div>
                  <div className="text-sm text-gray-600">Low</div>
                </div>
              </div>

              {scan.results.vulnerability_results.vulnerabilities?.length > 0 && (
                <div>
                  <h4 className="font-medium text-gray-900 mb-3">Vulnerabilities Found</h4>
                  <div className="space-y-3 max-h-64 overflow-y-auto">
                    {scan.results.vulnerability_results.vulnerabilities.map((vuln, index) => (
                      <div key={index} className="border border-gray-200 rounded-lg p-3">
                        <div className="flex items-start justify-between">
                          <div className="flex-1">
                            <div className="flex items-center space-x-2 mb-1">
                              <span className="font-medium text-gray-900">{vuln.cve || vuln.id}</span>
                              {vuln.cve && vuln.cve !== vuln.id && (
                                <span className="text-xs text-gray-500">({vuln.id})</span>
                              )}
                              <span className={`px-2 py-1 rounded text-xs font-medium ${getSeverityColor(vuln.severity)}`}>
                                {vuln.severity?.toUpperCase()}
                              </span>
                            </div>
                            <p className="text-sm text-gray-600 mb-2">{vuln.description}</p>
                            <div className="text-xs text-gray-500">
                              <span>Component: {vuln.component}</span>
                              {vuln.version && <span className="ml-2">Version: {vuln.version}</span>}
                            </div>
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}

          {/* Malware Results */}
          {scan.results?.malware_results && (
            <div className="bg-white border rounded-lg p-4">
              <h3 className="text-lg font-semibold text-gray-900 mb-4">Malware Scan Results</h3>
              
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-4">
                <div className="text-center">
                  <div className="text-2xl font-bold text-gray-900">
                    {scan.results.malware_results.total_scanned}
                  </div>
                  <div className="text-sm text-gray-600">Files Scanned</div>
                </div>
                <div className="text-center">
                  <div className="text-2xl font-bold text-red-600">
                    {scan.results.malware_results.threats_found}
                  </div>
                  <div className="text-sm text-gray-600">Threats Found</div>
                </div>
                <div className="text-center">
                  <div className="text-2xl font-bold text-green-600">
                    {scan.results.malware_results.clean_files}
                  </div>
                  <div className="text-sm text-gray-600">Clean Files</div>
                </div>
                <div className="text-center">
                  <div className="text-2xl font-bold text-yellow-600">
                    {scan.results.malware_results.suspicious_files}
                  </div>
                  <div className="text-sm text-gray-600">Suspicious</div>
                </div>
              </div>
            </div>
          )}

          {/* Recommendations */}
          {scan.results?.recommendations?.length > 0 && (
            <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
              <h3 className="text-lg font-semibold text-blue-900 mb-3">Recommendations</h3>
              <ul className="space-y-2">
                {scan.results.recommendations.map((recommendation, index) => (
                  <li key={index} className="flex items-start text-sm text-blue-800">
                    <span className="text-blue-600 mr-2">•</span>
                    {recommendation}
                  </li>
                ))}
              </ul>
            </div>
          )}

          {/* Error Message */}
          {scan.error_message && (
            <div className="bg-red-50 border border-red-200 rounded-lg p-4">
              <h3 className="text-lg font-semibold text-red-900 mb-2">Error</h3>
              <p className="text-sm text-red-800">{scan.error_message}</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};