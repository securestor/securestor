import React, { useState } from 'react';
import ReplicationNodeForm from './ReplicationNodeForm';
import ReplicationNodeTable from './ReplicationNodeTable';
import './ReplicationNodeManagement.css';

const ReplicationNodeManagement = ({ nodes, onCreateNode, onUpdateNode, onDeleteNode, onSuccess }) => {
  const [showForm, setShowForm] = useState(false);
  const [editingNode, setEditingNode] = useState(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleAddClick = () => {
    setEditingNode(null);
    setShowForm(true);
  };

  const handleEditClick = (node) => {
    setEditingNode(node);
    setShowForm(true);
  };

  const handleFormSubmit = async (formData) => {
    setIsSubmitting(true);
    try {
      if (editingNode) {
        await onUpdateNode(editingNode.id, formData);
        onSuccess(`Node "${editingNode.node_name}" updated successfully`);
      } else {
        await onCreateNode(formData);
        onSuccess('New replication node created successfully');
      }
      setShowForm(false);
      setEditingNode(null);
    } catch (error) {
      console.error('Error submitting form:', error);
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleDeleteClick = async (node) => {
    if (window.confirm(`Are you sure you want to delete node "${node.node_name}"?`)) {
      try {
        await onDeleteNode(node.id);
        onSuccess(`Node "${node.node_name}" deleted successfully`);
      } catch (error) {
        console.error('Error deleting node:', error);
      }
    }
  };

  const handleFormCancel = () => {
    setShowForm(false);
    setEditingNode(null);
  };

  return (
    <div className="replication-node-management">
      <div className="section-header">
        <h3>Storage Nodes Management</h3>
        <p>Manage storage nodes for data replication. Each repository will replicate data across these nodes.</p>
      </div>

      <div className="node-actions">
        <button
          className="btn btn-primary"
          onClick={handleAddClick}
          disabled={showForm}
        >
          + Add Storage Node
        </button>
      </div>

      {showForm && (
        <ReplicationNodeForm
          node={editingNode}
          onSubmit={handleFormSubmit}
          onCancel={handleFormCancel}
          isSubmitting={isSubmitting}
        />
      )}

      {nodes && nodes.length > 0 ? (
        <ReplicationNodeTable
          nodes={nodes}
          onEdit={handleEditClick}
          onDelete={handleDeleteClick}
        />
      ) : (
        <div className="empty-state">
          <p>No storage nodes configured yet. Add your first node to enable replication.</p>
        </div>
      )}

      <div className="node-info-box">
        <h4>💡 About Storage Nodes</h4>
        <ul>
          <li><strong>Node Name:</strong> Unique identifier for the storage node</li>
          <li><strong>Node Path:</strong> File system path where replicated data is stored</li>
          <li><strong>Priority:</strong> Lower number = higher priority for read operations</li>
          <li><strong>Active:</strong> Only active nodes participate in replication</li>
        </ul>
      </div>
    </div>
  );
};

export default ReplicationNodeManagement;
