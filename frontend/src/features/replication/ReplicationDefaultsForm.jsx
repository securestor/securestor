import React, { useState } from 'react';
import './ReplicationDefaultsForm.css';

const ReplicationDefaultsForm = ({ config, onUpdate, onSuccess }) => {
  const [formData, setFormData] = useState({
    enable_replication_default: config.enable_replication_default,
    default_quorum_size: config.default_quorum_size,
    sync_frequency_default: config.sync_frequency_default,
    node_health_check_interval: config.node_health_check_interval,
    failover_timeout: config.failover_timeout,
  });

  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleChange = (e) => {
    const { name, value, type, checked } = e.target;
    setFormData({
      ...formData,
      [name]: type === 'checkbox' ? checked : value,
    });
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setIsSubmitting(true);

    try {
      await onUpdate(formData);
      onSuccess('Replication settings updated successfully');
    } catch (error) {
      console.error('Error updating settings:', error);
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="replication-defaults-form">
      <div className="form-section">
        <h3>Global Replication Defaults</h3>
        <p className="section-description">
          Configure default replication settings that will be applied to new repositories.
          Existing repositories can override these defaults.
        </p>

        <form onSubmit={handleSubmit}>
          {/* Enable Replication */}
          <div className="form-group">
            <div className="checkbox-group">
              <input
                type="checkbox"
                id="enable_replication"
                name="enable_replication_default"
                checked={formData.enable_replication_default}
                onChange={handleChange}
              />
              <label htmlFor="enable_replication">
                <strong>Enable Replication by Default</strong>
                <span className="help-text">New repositories will have replication enabled automatically</span>
              </label>
            </div>
          </div>

          {/* Quorum Size */}
          <div className="form-group">
            <label htmlFor="quorum_size">
              <strong>Default Quorum Size</strong>
              <span className="help-text">Minimum number of replicas required for successful writes (2-5)</span>
            </label>
            <div className="input-range-group">
              <input
                type="range"
                id="quorum_size"
                name="default_quorum_size"
                min="2"
                max="5"
                value={formData.default_quorum_size}
                onChange={handleChange}
              />
              <input
                type="number"
                name="default_quorum_size"
                min="2"
                max="5"
                value={formData.default_quorum_size}
                onChange={handleChange}
                className="number-input"
              />
              <span className="value-display">
                {formData.default_quorum_size} replica{formData.default_quorum_size !== 1 ? 's' : ''}
              </span>
            </div>
          </div>

          {/* Sync Frequency */}
          <div className="form-group">
            <label htmlFor="sync_frequency">
              <strong>Default Sync Frequency</strong>
              <span className="help-text">How often to synchronize replicas</span>
            </label>
            <select
              id="sync_frequency"
              name="sync_frequency_default"
              value={formData.sync_frequency_default}
              onChange={handleChange}
              className="form-select"
            >
              <option value="realtime">Real-time (synchronous)</option>
              <option value="hourly">Hourly</option>
              <option value="daily">Daily</option>
              <option value="weekly">Weekly</option>
            </select>
          </div>

          {/* Health Check Interval */}
          <div className="form-group">
            <label htmlFor="health_check">
              <strong>Node Health Check Interval (seconds)</strong>
              <span className="help-text">How often to check if nodes are healthy (10-300 seconds)</span>
            </label>
            <div className="input-range-group">
              <input
                type="range"
                id="health_check"
                name="node_health_check_interval"
                min="10"
                max="300"
                step="5"
                value={formData.node_health_check_interval}
                onChange={handleChange}
              />
              <input
                type="number"
                name="node_health_check_interval"
                min="10"
                max="300"
                value={formData.node_health_check_interval}
                onChange={handleChange}
                className="number-input"
              />
              <span className="value-display">{formData.node_health_check_interval}s</span>
            </div>
          </div>

          {/* Failover Timeout */}
          <div className="form-group">
            <label htmlFor="failover_timeout">
              <strong>Failover Timeout (seconds)</strong>
              <span className="help-text">Time to wait before promoting standby on primary failure (5-300 seconds)</span>
            </label>
            <div className="input-range-group">
              <input
                type="range"
                id="failover_timeout"
                name="failover_timeout"
                min="5"
                max="300"
                step="5"
                value={formData.failover_timeout}
                onChange={handleChange}
              />
              <input
                type="number"
                name="failover_timeout"
                min="5"
                max="300"
                value={formData.failover_timeout}
                onChange={handleChange}
                className="number-input"
              />
              <span className="value-display">{formData.failover_timeout}s</span>
            </div>
          </div>

          {/* Submit Button */}
          <div className="form-actions">
            <button
              type="submit"
              className="btn btn-primary"
              disabled={isSubmitting}
            >
              {isSubmitting ? 'Saving...' : 'Save Replication Defaults'}
            </button>
          </div>
        </form>
      </div>

      {/* Info Box */}
      <div className="info-box">
        <h4>📋 Current Settings</h4>
        <ul>
          <li><strong>Replication:</strong> {formData.enable_replication_default ? 'Enabled' : 'Disabled'}</li>
          <li><strong>Quorum Size:</strong> {formData.default_quorum_size} replicas</li>
          <li><strong>Sync:</strong> {formData.sync_frequency_default}</li>
          <li><strong>Health Check:</strong> Every {formData.node_health_check_interval} seconds</li>
          <li><strong>Failover Timeout:</strong> {formData.failover_timeout} seconds</li>
        </ul>
      </div>
    </div>
  );
};

export default ReplicationDefaultsForm;
