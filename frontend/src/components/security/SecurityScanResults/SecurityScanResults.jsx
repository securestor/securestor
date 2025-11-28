import React, { useState, useEffect } from 'react';
import { Shield, AlertTriangle, CheckCircle, Clock, Download, RefreshCw } from 'lucide-react';
import scanAPI from '../../../services/api/scanAPI';
import { useToast } from '../../../context/ToastContext';

export const SecurityScanResults = ({ artifactId, onScanStart }) => {
  const { showSuccess, showError } = useToast();
  const [scanResults, setScanResults] = useState(null);
  const [loading, setLoading] = useState(true);
  const [scanning, setScanning] = useState(false);
  const [activeScan, setActiveScan] = useState(null);

  useEffect(() => {
    loadScanResults();
    checkActiveScan();
  }, [artifactId]);

  const loadScanResults = async () => {
    try {
      setLoading(true);
      const results = await scanAPI.getScanResults(artifactId);
      setScanResults(results);
    } catch (error) {
      console.error('Failed to load scan results:', error);
      setScanResults(null);
    } finally {
      setLoading(false);
    }
  };

  const checkActiveScan = async () => {
    try {
      const history = await scanAPI.getScanHistory(artifactId, 1, 0);
      if (history.scans && history.scans.length > 0) {
        const latestScan = history.scans[0];
        if (latestScan.status === 'initiated' || latestScan.status === 'running') {
          setActiveScan(latestScan);
          setScanning(true);
          // Start polling for scan completion
          pollScanProgress(latestScan.id);
        }
      }
    } catch (error) {
      console.error('Failed to check active scan:', error);
    }
  };

  const startScan = async (config = {}) => {
    try {
      setScanning(true);
      const response = await scanAPI.startScan(artifactId, config);
      setActiveScan(response);
      showSuccess('Security scan started successfully');
      
      if (onScanStart) {
        onScanStart(response);
      }

      // Start polling for completion
      pollScanProgress(response.scan_id);
    } catch (error) {
      setScanning(false);
      showError(`Failed to start scan: ${error.message}`);
    }
  };

  const pollScanProgress = async (scanId) => {
    try {
      await scanAPI.pollScanStatus(
        scanId,
        (scan) => {
          setActiveScan(scan);
        },
        30, // max attempts
        3000 // 3 second interval
      );
      
      // Scan completed, reload results
      setScanning(false);
      setActiveScan(null);
      await loadScanResults();
      showSuccess('Security scan completed successfully');
    } catch (error) {
      setScanning(false);
      setActiveScan(null);
      showError(`Scan failed: ${error.message}`);
    }
  };

  const cancelScan = async () => {
    if (!activeScan) return;
    
    try {
      await scanAPI.cancelScan(activeScan.id);
      setScanning(false);
      setActiveScan(null);
      showSuccess('Scan cancelled successfully');
    } catch (error) {
      showError(`Failed to cancel scan: ${error.message}`);
    }
  };

  const getRiskLevelColor = (riskLevel) => {
    const colors = {
      low: 'text-green-600 bg-green-100',
      medium: 'text-yellow-600 bg-yellow-100',
      high: 'text-orange-600 bg-orange-100',
      critical: 'text-red-600 bg-red-100'
    };
    return colors[riskLevel] || colors.low;
  };

  const getSeverityColor = (severity) => {
    const colors = {
      info: 'text-blue-600 bg-blue-100',
      low: 'text-green-600 bg-green-100',
      medium: 'text-yellow-600 bg-yellow-100',
      high: 'text-orange-600 bg-orange-100',
      critical: 'text-red-600 bg-red-100'
    };
    return colors[severity] || colors.info;
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center p-8">
        <RefreshCw className="w-6 h-6 animate-spin text-blue-600" />
        <span className="ml-2 text-gray-600">Loading scan results...</span>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Scan Controls */}
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-medium text-gray-900 flex items-center">
          <Shield className="w-5 h-5 mr-2 text-blue-600" />
          Security Scan Results
        </h3>
        
        <div className="flex space-x-2">
          {scanning && activeScan && (
            <button
              onClick={cancelScan}
              className="px-3 py-2 text-sm bg-red-600 text-white rounded-lg hover:bg-red-700 transition"
            >
              Cancel Scan
            </button>
          )}
          
          <button
            onClick={() => startScan()}
            disabled={scanning}
            className="px-4 py-2 text-sm bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition disabled:opacity-50 disabled:cursor-not-allowed flex items-center"
          >
            {scanning ? (
              <>
                <RefreshCw className="w-4 h-4 mr-2 animate-spin" />
                Scanning...
              </>
            ) : (
              <>
                <Shield className="w-4 h-4 mr-2" />
                Run Scan
              </>
            )}
          </button>
        </div>
      </div>

      {/* Active Scan Progress */}
      {scanning && activeScan && (
        <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center">
              <Clock className="w-5 h-5 text-blue-600 mr-2" />
              <div>
                <p className="font-medium text-blue-900">Scan in Progress</p>
                <p className="text-sm text-blue-700">
                  Status: {activeScan.status} • 
                  Started: {new Date(activeScan.started_at).toLocaleTimeString()}
                </p>
              </div>
            </div>
            <div className="flex items-center space-x-2">
              <RefreshCw className="w-4 h-4 animate-spin text-blue-600" />
            </div>
          </div>
        </div>
      )}

      {/* Scan Results */}
      {scanResults ? (
        <div className="space-y-6">
          {/* Overall Score */}
          <div className="bg-white border border-gray-200 rounded-lg p-6">
            <div className="flex items-center justify-between mb-4">
              <h4 className="text-lg font-medium text-gray-900">Overall Security Score</h4>
              <span className={`px-3 py-1 rounded-full text-sm font-medium ${getRiskLevelColor(scanResults.risk_level)}`}>
                {scanResults.risk_level.toUpperCase()}
              </span>
            </div>
            
            <div className="flex items-center space-x-4">
              <div className="flex-1">
                <div className="flex items-center justify-between mb-2">
                  <span className="text-3xl font-bold text-gray-900">{scanResults.overall_score}/100</span>
                  <span className="text-sm text-gray-500">Security Score</span>
                </div>
                <div className="w-full bg-gray-200 rounded-full h-2">
                  <div 
                    className={`h-2 rounded-full ${
                      scanResults.overall_score >= 80 ? 'bg-green-500' :
                      scanResults.overall_score >= 60 ? 'bg-yellow-500' :
                      scanResults.overall_score >= 40 ? 'bg-orange-500' : 'bg-red-500'
                    }`}
                    style={{ width: `${scanResults.overall_score}%` }}
                  ></div>
                </div>
              </div>
            </div>

            <p className="mt-4 text-gray-600">{scanResults.summary}</p>
          </div>

          {/* Vulnerability Results */}
          {scanResults.vulnerability_results && (
            <div className="bg-white border border-gray-200 rounded-lg p-6">
              <h4 className="text-lg font-medium text-gray-900 mb-4 flex items-center">
                <AlertTriangle className="w-5 h-5 mr-2 text-orange-600" />
                Vulnerabilities
              </h4>

              <div className="grid grid-cols-2 md:grid-cols-5 gap-4 mb-4">
                <div className="text-center">
                  <div className="text-2xl font-bold text-red-600">
                    {scanResults.vulnerability_results.critical}
                  </div>
                  <div className="text-sm text-gray-500">Critical</div>
                </div>
                <div className="text-center">
                  <div className="text-2xl font-bold text-orange-600">
                    {scanResults.vulnerability_results.high}
                  </div>
                  <div className="text-sm text-gray-500">High</div>
                </div>
                <div className="text-center">
                  <div className="text-2xl font-bold text-yellow-600">
                    {scanResults.vulnerability_results.medium}
                  </div>
                  <div className="text-sm text-gray-500">Medium</div>
                </div>
                <div className="text-center">
                  <div className="text-2xl font-bold text-green-600">
                    {scanResults.vulnerability_results.low}
                  </div>
                  <div className="text-sm text-gray-500">Low</div>
                </div>
                <div className="text-center">
                  <div className="text-2xl font-bold text-gray-600">
                    {scanResults.vulnerability_results.total_found}
                  </div>
                  <div className="text-sm text-gray-500">Total</div>
                </div>
              </div>

              {scanResults.vulnerability_results.vulnerabilities && 
               scanResults.vulnerability_results.vulnerabilities.length > 0 && (
                <div className="space-y-2">
                  <h5 className="font-medium text-gray-900">Recent Vulnerabilities</h5>
                  {scanResults.vulnerability_results.vulnerabilities.slice(0, 5).map((vuln, index) => (
                    <div key={index} className="flex items-center justify-between p-3 bg-gray-50 rounded">
                      <div className="flex-1">
                        <div className="font-medium text-gray-900">{vuln.title}</div>
                        <div className="text-sm text-gray-500">{vuln.component}</div>
                      </div>
                      <span className={`px-2 py-1 rounded text-xs font-medium ${getSeverityColor(vuln.severity)}`}>
                        {vuln.severity.toUpperCase()}
                      </span>
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}

          {/* Malware Results */}
          {scanResults.malware_results && (
            <div className="bg-white border border-gray-200 rounded-lg p-6">
              <h4 className="text-lg font-medium text-gray-900 mb-4 flex items-center">
                <Shield className="w-5 h-5 mr-2 text-purple-600" />
                Malware Scan
              </h4>

              <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                <div className="text-center">
                  <div className="text-2xl font-bold text-green-600">
                    {scanResults.malware_results.clean_files}
                  </div>
                  <div className="text-sm text-gray-500">Clean Files</div>
                </div>
                <div className="text-center">
                  <div className="text-2xl font-bold text-red-600">
                    {scanResults.malware_results.threats_found}
                  </div>
                  <div className="text-sm text-gray-500">Threats</div>
                </div>
                <div className="text-center">
                  <div className="text-2xl font-bold text-yellow-600">
                    {scanResults.malware_results.suspicious_files}
                  </div>
                  <div className="text-sm text-gray-500">Suspicious</div>
                </div>
                <div className="text-center">
                  <div className="text-2xl font-bold text-gray-600">
                    {scanResults.malware_results.total_scanned}
                  </div>
                  <div className="text-sm text-gray-500">Total Scanned</div>
                </div>
              </div>

              {scanResults.malware_results.threats_found === 0 && (
                <div className="mt-4 flex items-center text-green-600">
                  <CheckCircle className="w-5 h-5 mr-2" />
                  <span>No malware threats detected</span>
                </div>
              )}
            </div>
          )}

          {/* Recommendations */}
          {scanResults.recommendations && scanResults.recommendations.length > 0 && (
            <div className="bg-white border border-gray-200 rounded-lg p-6">
              <h4 className="text-lg font-medium text-gray-900 mb-4">Recommendations</h4>
              <ul className="space-y-2">
                {scanResults.recommendations.map((recommendation, index) => (
                  <li key={index} className="flex items-start">
                    <div className="flex-shrink-0 w-2 h-2 bg-blue-600 rounded-full mt-2 mr-3"></div>
                    <span className="text-gray-700">{recommendation}</span>
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>
      ) : (
        !scanning && (
          <div className="text-center py-8">
            <Shield className="w-12 h-12 text-gray-400 mx-auto mb-4" />
            <h3 className="text-lg font-medium text-gray-900 mb-2">No Scan Results Available</h3>
            <p className="text-gray-500 mb-4">Run a security scan to see detailed results</p>
            <button
              onClick={() => startScan()}
              className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition"
            >
              Start Security Scan
            </button>
          </div>
        )
      )}
    </div>
  );
};