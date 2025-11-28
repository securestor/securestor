import React, { useState } from 'react';
import { useTenant } from '../../context/TenantContext';
import useReplicationSettings from '../../hooks/useReplicationSettings';
import ReplicationNodeManagement from './ReplicationNodeManagement';
import ReplicationDefaultsForm from './ReplicationDefaultsForm';
import './ReplicationSettings.css';

const ReplicationSettings = () => {
  const { tenantId, loading: tenantLoading } = useTenant();
  const { config, nodes, loading, error, updateConfig, createNode, updateNode, deleteNode } = useReplicationSettings();
  const [activeTab, setActiveTab] = useState('defaults');
  const [successMessage, setSuccessMessage] = useState('');
  const [showSuccessMessage, setShowSuccessMessage] = useState(false);

  const handleShowSuccess = (message) => {
    setSuccessMessage(message);
    setShowSuccessMessage(true);
    setTimeout(() => setShowSuccessMessage(false), 3000);
  };

  // Wait for tenant to be ready
  if (tenantLoading) {
    return <div className="replication-settings loading">Loading tenant information...</div>;
  }

  if (!tenantId) {
    return (
      <div className="replication-settings error">
        <div className="alert alert-error">
          <span>Tenant information not available. Please refresh the page.</span>
        </div>
      </div>
    );
  }

  if (loading) {
    return <div className="replication-settings loading">Loading replication settings...</div>;
  }

  return (
    <div className="replication-settings">
      <div className="settings-header">
        <h1>🔄 Replication Settings</h1>
        <p>Configure global replication defaults and manage storage nodes</p>
      </div>

      {error && (
        <div className="alert alert-error">
          <span>{error}</span>
        </div>
      )}

      {showSuccessMessage && (
        <div className="alert alert-success">
          <span>✓ {successMessage}</span>
        </div>
      )}

      <div className="tabs">
        <button
          className={`tab-button ${activeTab === 'defaults' ? 'active' : ''}`}
          onClick={() => setActiveTab('defaults')}
        >
          Replication Defaults
        </button>
        <button
          className={`tab-button ${activeTab === 'nodes' ? 'active' : ''}`}
          onClick={() => setActiveTab('nodes')}
        >
          Storage Nodes ({nodes?.length || 0})
        </button>
      </div>

      <div className="tab-content">
        {activeTab === 'defaults' && config && (
          <ReplicationDefaultsForm
            config={config}
            onUpdate={updateConfig}
            onSuccess={handleShowSuccess}
          />
        )}

        {activeTab === 'nodes' && (
          <ReplicationNodeManagement
            nodes={nodes}
            onCreateNode={createNode}
            onUpdateNode={updateNode}
            onDeleteNode={deleteNode}
            onSuccess={handleShowSuccess}
          />
        )}
      </div>
    </div>
  );
};

export default ReplicationSettings;
