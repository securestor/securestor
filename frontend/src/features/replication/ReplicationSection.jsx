import React, { useState, useEffect } from 'react';
import useReplicationSettings from '../../hooks/useReplicationSettings';
import './ReplicationSection.css';

const ReplicationSection = ({ repository, onUpdate }) => {
  const { config: globalConfig, nodes: globalNodes } = useReplicationSettings();
  const [useGlobalDefaults, setUseGlobalDefaults] = useState(!repository?.override_global_replication);
  const [selectedNodes, setSelectedNodes] = useState(repository?.replication_node_ids || []);
  const [customQuorum, setCustomQuorum] = useState(repository?.custom_quorum_size || null);
  const [syncFrequency, setSyncFrequency] = useState(repository?.sync_frequency || 'realtime');
  const [enableReplication, setEnableReplication] = useState(repository?.enable_replication !== false);

  useEffect(() => {
    if (repository) {
      setUseGlobalDefaults(!repository.override_global_replication);
      setSelectedNodes(repository.replication_node_ids || []);
      setCustomQuorum(repository.custom_quorum_size || null);
      setSyncFrequency(repository.sync_frequency || 'realtime');
      setEnableReplication(repository.enable_replication !== false);
    }
  }, [repository]);

  const availableNodes = globalNodes || [];
  const effectiveQuorum = customQuorum || globalConfig?.default_quorum_size || 2;

  const handleNodeToggle = (nodeId) => {
    setSelectedNodes((prev) =>
      prev.includes(nodeId) ? prev.filter((id) => id !== nodeId) : [...prev, nodeId]
    );
  };

  const handleSave = () => {
    const settings = {
      enable_replication: enableReplication,
      replication_node_ids: selectedNodes,
      sync_frequency: syncFrequency,
      override_global_replication: !useGlobalDefaults,
      custom_quorum_size: !useGlobalDefaults && customQuorum ? customQuorum : null,
    };

    onUpdate(settings);
  };

  return (
    <div className="replication-section">
      <div className="section-header">
        <h3>🔄 Replication Settings</h3>
        <p>Configure how this repository should be replicated across storage nodes</p>
      </div>

      <div className="form-group">
        <div className="checkbox-group">
          <input
            type="checkbox"
            id="enable_replication"
            checked={enableReplication}
            onChange={(e) => setEnableReplication(e.target.checked)}
          />
          <label htmlFor="enable_replication">
            <strong>Enable Replication</strong>
            <span className="help-text">Data will be replicated across storage nodes for high availability</span>
          </label>
        </div>
      </div>

      {enableReplication && (
        <>
          {/* Use Global Defaults vs Custom */}
          <div className="form-group">
            <div className="radio-group">
              <label className="radio-label">
                <input
                  type="radio"
                  checked={useGlobalDefaults}
                  onChange={() => setUseGlobalDefaults(true)}
                />
                <strong>Use Global Defaults</strong>
                <span className="help-text">Inherit settings from global replication configuration</span>
              </label>
              <label className="radio-label">
                <input
                  type="radio"
                  checked={!useGlobalDefaults}
                  onChange={() => setUseGlobalDefaults(false)}
                />
                <strong>Custom Settings</strong>
                <span className="help-text">Override with repository-specific settings</span>
              </label>
            </div>
          </div>

          {/* Custom Settings */}
          {!useGlobalDefaults && (
            <>
              {/* Node Selection */}
              <div className="form-group">
                <label>
                  <strong>Select Replication Nodes</strong>
                  <span className="help-text">Choose which nodes should store replicas of this repository</span>
                </label>
                <div className="node-selector">
                  {availableNodes.length > 0 ? (
                    <div className="node-checkboxes">
                      {availableNodes.map((node) => (
                        <div key={node.id} className="node-checkbox-item">
                          <input
                            type="checkbox"
                            id={`node-${node.id}`}
                            checked={selectedNodes.includes(node.id)}
                            onChange={() => handleNodeToggle(node.id)}
                            disabled={!node.is_active}
                          />
                          <label htmlFor={`node-${node.id}`}>
                            <span className="node-name">{node.node_name}</span>
                            <span className="node-path">{node.node_path}</span>
                            {!node.is_active && <span className="badge-inactive">Inactive</span>}
                            <span className={`badge-health ${node.is_healthy ? 'healthy' : 'unhealthy'}`}>
                              {node.is_healthy ? '✓' : '✗'}
                            </span>
                          </label>
                        </div>
                      ))}
                    </div>
                  ) : (
                    <p className="no-nodes-message">No storage nodes configured. Add nodes in settings first.</p>
                  )}
                </div>
                <div className="selection-info">
                  Selected: {selectedNodes.length} node{selectedNodes.length !== 1 ? 's' : ''}
                </div>
              </div>

              {/* Custom Quorum */}
              <div className="form-group">
                <label htmlFor="custom_quorum">
                  <strong>Quorum Size</strong>
                  <span className="help-text">Minimum replicas needed before confirming write success</span>
                </label>
                <div className="quorum-input-group">
                  <input
                    type="range"
                    id="custom_quorum"
                    min="1"
                    max={Math.min(5, selectedNodes.length || 3)}
                    value={customQuorum || 2}
                    onChange={(e) => setCustomQuorum(parseInt(e.target.value))}
                  />
                  <input
                    type="number"
                    min="1"
                    max={Math.min(5, selectedNodes.length || 3)}
                    value={customQuorum || 2}
                    onChange={(e) => setCustomQuorum(parseInt(e.target.value))}
                    className="number-input"
                  />
                  <span className="value-display">
                    {effectiveQuorum} of {selectedNodes.length || 3}
                  </span>
                </div>
              </div>

              {/* Sync Frequency */}
              <div className="form-group">
                <label htmlFor="sync_frequency">
                  <strong>Sync Frequency</strong>
                  <span className="help-text">How often replicas should be synchronized</span>
                </label>
                <select
                  id="sync_frequency"
                  value={syncFrequency}
                  onChange={(e) => setSyncFrequency(e.target.value)}
                  className="form-select"
                >
                  <option value="realtime">Real-time (Synchronous)</option>
                  <option value="hourly">Hourly</option>
                  <option value="daily">Daily</option>
                  <option value="weekly">Weekly</option>
                </select>
              </div>
            </>
          )}

          {/* Summary */}
          <div className="summary-box">
            <h4>📊 Configuration Summary</h4>
            {useGlobalDefaults && globalConfig ? (
              <ul>
                <li><strong>Mode:</strong> Using Global Defaults</li>
                <li><strong>Nodes:</strong> {globalNodes?.length || 0} nodes configured</li>
                <li><strong>Quorum:</strong> {globalConfig.default_quorum_size} replicas</li>
                <li><strong>Sync:</strong> {globalConfig.sync_frequency_default}</li>
              </ul>
            ) : (
              <ul>
                <li><strong>Mode:</strong> Custom Configuration</li>
                <li><strong>Nodes:</strong> {selectedNodes.length} nodes selected</li>
                <li><strong>Quorum:</strong> {effectiveQuorum} replicas</li>
                <li><strong>Sync:</strong> {syncFrequency}</li>
              </ul>
            )}
          </div>
        </>
      )}

      {/* Save Button */}
      <div className="form-actions">
        <button className="btn btn-primary" onClick={handleSave}>
          Save Replication Settings
        </button>
      </div>
    </div>
  );
};

export default ReplicationSection;
