import React, { useState, useEffect } from 'react';
import './ReplicationNodeForm.css';

const ReplicationNodeForm = ({ node, onSubmit, onCancel, isSubmitting }) => {
  const [formData, setFormData] = useState({
    node_name: '',
    node_path: '',
    priority: 1,
    is_active: true,
  });

  const [errors, setErrors] = useState({});

  useEffect(() => {
    if (node) {
      setFormData({
        node_name: node.node_name,
        node_path: node.node_path,
        priority: node.priority,
        is_active: node.is_active,
      });
    }
  }, [node]);

  const validateForm = () => {
    const newErrors = {};

    if (!formData.node_name.trim()) {
      newErrors.node_name = 'Node name is required';
    } else if (formData.node_name.length < 1 || formData.node_name.length > 255) {
      newErrors.node_name = 'Node name must be between 1 and 255 characters';
    }

    if (!formData.node_path.trim()) {
      newErrors.node_path = 'Node path is required';
    } else if (formData.node_path.length < 1 || formData.node_path.length > 1024) {
      newErrors.node_path = 'Node path must be between 1 and 1024 characters';
    }

    if (formData.priority < 1 || formData.priority > 100) {
      newErrors.priority = 'Priority must be between 1 and 100';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleChange = (e) => {
    const { name, value, type, checked } = e.target;
    setFormData({
      ...formData,
      [name]: type === 'checkbox' ? checked : value,
    });
    // Clear error for this field when user starts typing
    if (errors[name]) {
      setErrors({ ...errors, [name]: '' });
    }
  };

  const handleSubmit = (e) => {
    e.preventDefault();

    if (!validateForm()) {
      return;
    }

    onSubmit(formData);
  };

  return (
    <div className="replication-node-form-overlay">
      <div className="replication-node-form">
        <div className="form-header">
          <h4>{node ? 'Edit Storage Node' : 'Add New Storage Node'}</h4>
          <button
            type="button"
            className="close-btn"
            onClick={onCancel}
            disabled={isSubmitting}
          >
            ✕
          </button>
        </div>

        <form onSubmit={handleSubmit}>
          {/* Node Name */}
          <div className="form-group">
            <label htmlFor="node_name">
              <strong>Node Name *</strong>
              <span className="help-text">Unique identifier for this storage node (e.g., node1, node2)</span>
            </label>
            <input
              type="text"
              id="node_name"
              name="node_name"
              value={formData.node_name}
              onChange={handleChange}
              placeholder="e.g., node1"
              className={errors.node_name ? 'input-error' : ''}
              disabled={node !== null} // Disable if editing
            />
            {errors.node_name && <span className="error-text">{errors.node_name}</span>}
          </div>

          {/* Node Path */}
          <div className="form-group">
            <label htmlFor="node_path">
              <strong>Node Path *</strong>
              <span className="help-text">File system path where data will be stored (e.g., /storage/ssd1/securestor)</span>
            </label>
            <input
              type="text"
              id="node_path"
              name="node_path"
              value={formData.node_path}
              onChange={handleChange}
              placeholder="e.g., /storage/ssd1/securestor"
              className={errors.node_path ? 'input-error' : ''}
            />
            {errors.node_path && <span className="error-text">{errors.node_path}</span>}
          </div>

          {/* Priority */}
          <div className="form-group">
            <label htmlFor="priority">
              <strong>Priority</strong>
              <span className="help-text">Lower number = higher priority for read operations (1-100)</span>
            </label>
            <div className="priority-input-group">
              <input
                type="range"
                id="priority"
                name="priority"
                min="1"
                max="100"
                value={formData.priority}
                onChange={handleChange}
              />
              <input
                type="number"
                name="priority"
                min="1"
                max="100"
                value={formData.priority}
                onChange={handleChange}
                className="number-input"
              />
            </div>
            {errors.priority && <span className="error-text">{errors.priority}</span>}
          </div>

          {/* Active Status */}
          <div className="form-group">
            <div className="checkbox-group">
              <input
                type="checkbox"
                id="is_active"
                name="is_active"
                checked={formData.is_active}
                onChange={handleChange}
              />
              <label htmlFor="is_active">
                <strong>Active</strong>
                <span className="help-text">Only active nodes participate in replication</span>
              </label>
            </div>
          </div>

          {/* Form Actions */}
          <div className="form-actions">
            <button
              type="submit"
              className="btn btn-primary"
              disabled={isSubmitting}
            >
              {isSubmitting ? 'Saving...' : 'Save Node'}
            </button>
            <button
              type="button"
              className="btn btn-secondary"
              onClick={onCancel}
              disabled={isSubmitting}
            >
              Cancel
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

export default ReplicationNodeForm;
