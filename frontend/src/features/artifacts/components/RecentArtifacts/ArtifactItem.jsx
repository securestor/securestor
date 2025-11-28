import React, { useState } from 'react';
import { Package, ChevronRight, Shield } from 'lucide-react';
import scanAPI from '../../../../services/api/scanAPI';
import { useToast } from '../../../../context/ToastContext';

export const ArtifactItem = ({ artifact }) => {
  const { showSuccess, showError } = useToast();
  const [scanning, setScanning] = useState(false);

  const handleQuickScan = async (e) => {
    e.stopPropagation(); // Prevent item click
    
    try {
      setScanning(true);
      await scanAPI.startScan(artifact.id, {
        vulnerability_scan: true,
        malware_scan: true,
        priority: 'high'
      });
      showSuccess(`Security scan started for ${artifact.name}`);
    } catch (error) {
      showError(`Failed to start scan: ${error.message}`);
    } finally {
      setScanning(false);
    }
  };

  return (
    <div className="px-6 py-4 hover:bg-gray-50 transition cursor-pointer flex items-center justify-between">
      <div className="flex items-center space-x-4">
        <Package className="w-8 h-8 text-blue-600" />
        <div>
          <h3 className="font-medium text-gray-900">{artifact.name}</h3>
          <p className="text-sm text-gray-600">{artifact.repo} • {artifact.size}</p>
        </div>
      </div>
      <div className="flex items-center space-x-4">
        <div className="text-right">
          <p className="text-sm text-gray-600">by {artifact.user}</p>
          <p className="text-xs text-gray-500">{artifact.uploaded}</p>
        </div>
        
        <button
          onClick={handleQuickScan}
          disabled={scanning}
          className="p-2 text-gray-600 hover:text-blue-600 hover:bg-blue-50 rounded-lg transition disabled:opacity-50"
          title="Quick Security Scan"
        >
          <Shield className={`w-5 h-5 ${scanning ? 'animate-pulse' : ''}`} />
        </button>
        
        <ChevronRight className="w-5 h-5 text-gray-400" />
      </div>
    </div>
  );
};
