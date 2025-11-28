import React from 'react';
import { Shield, CheckCircle, XCircle, AlertTriangle, Clock } from 'lucide-react';

export const ComplianceDetails = ({ artifact }) => {
  if (!artifact) return null;

  const getCheckIcon = (status) => {
    switch (status) {
      case 'passed':
        return <CheckCircle className="w-5 h-5 text-green-600" />;
      case 'failed':
        return <XCircle className="w-5 h-5 text-red-600" />;
      case 'warning':
        return <AlertTriangle className="w-5 h-5 text-yellow-600" />;
      case 'review':
        return <Clock className="w-5 h-5 text-blue-600" />;
      default:
        return <Clock className="w-5 h-5 text-gray-400" />;
    }
  };

  const getCheckLabel = (status) => {
    return status.charAt(0).toUpperCase() + status.slice(1);
  };

  return (
    <div className="bg-white rounded-lg border border-gray-200 p-6">
      <div className="flex items-center justify-between mb-6">
        <h3 className="text-lg font-semibold text-gray-900 flex items-center">
          <Shield className="w-5 h-5 mr-2 text-blue-600" />
          Compliance Audit Report
        </h3>
        <span className={`px-3 py-1 text-sm font-medium rounded-full ${
          artifact.compliance.status === 'compliant' ? 'bg-green-100 text-green-800' :
          artifact.compliance.status === 'review' ? 'bg-yellow-100 text-yellow-800' :
          'bg-red-100 text-red-800'
        }`}>
          {artifact.compliance.status.toUpperCase()}
        </span>
      </div>

      {/* Compliance Score */}
      <div className="mb-6">
        <div className="flex items-center justify-between mb-2">
          <span className="text-sm font-medium text-gray-700">Overall Compliance Score</span>
          <span className="text-2xl font-bold text-gray-900">{artifact.compliance.score}%</span>
        </div>
        <div className="w-full h-3 bg-gray-200 rounded-full overflow-hidden">
          <div
            className={`h-full transition-all ${
              artifact.compliance.score >= 90 ? 'bg-green-500' :
              artifact.compliance.score >= 70 ? 'bg-yellow-500' : 'bg-red-500'
            }`}
            style={{ width: `${artifact.compliance.score}%` }}
          />
        </div>
      </div>

      {/* Audit Details */}
      <div className="space-y-3 mb-6">
        <div className="flex justify-between text-sm">
          <span className="text-gray-600">Last Audit:</span>
          <span className="font-medium text-gray-900">
            {new Date(artifact.compliance.lastAudit).toLocaleString()}
          </span>
        </div>
        <div className="flex justify-between text-sm">
          <span className="text-gray-600">Audited By:</span>
          <span className="font-medium text-gray-900">{artifact.compliance.auditor}</span>
        </div>
      </div>

      {/* Compliance Checks */}
      <div className="space-y-3">
        <h4 className="text-sm font-semibold text-gray-900 mb-3">Compliance Checks</h4>
        {Object.entries(artifact.compliance.checks).map(([key, status]) => (
          <div key={key} className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
            <span className="text-sm font-medium text-gray-700 capitalize">
              {key.replace(/([A-Z])/g, ' $1').trim()}
            </span>
            <div className="flex items-center space-x-2">
              {getCheckIcon(status)}
              <span className={`text-sm font-medium ${
                status === 'passed' ? 'text-green-700' :
                status === 'failed' ? 'text-red-700' :
                status === 'warning' ? 'text-yellow-700' :
                'text-blue-700'
              }`}>
                {getCheckLabel(status)}
              </span>
            </div>
          </div>
        ))}
      </div>

      {/* Actions */}
      <div className="mt-6 pt-6 border-t border-gray-200">
        <button className="w-full px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition">
          Download Full Audit Report
        </button>
      </div>
    </div>
  );
};
