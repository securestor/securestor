import React from 'react';
import './ReplicationNodeTable.css';

const ReplicationNodeTable = ({ nodes, onEdit, onDelete }) => {
  return (
    <div className="replication-node-table-container">
      <table className="replication-node-table">
        <thead>
          <tr>
            <th>Node Name</th>
            <th>Path</th>
            <th>Priority</th>
            <th>Status</th>
            <th>Health</th>
            <th>Last Check</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          {nodes.map((node) => (
            <tr key={node.id} className={!node.is_active ? 'inactive' : ''}>
              <td>
                <strong>{node.node_name}</strong>
              </td>
              <td className="path-cell">
                <code>{node.node_path}</code>
              </td>
              <td>
                <span className="priority-badge">{node.priority}</span>
              </td>
              <td>
                <span className={`status-badge ${node.is_active ? 'active' : 'inactive'}`}>
                  {node.is_active ? 'Active' : 'Inactive'}
                </span>
              </td>
              <td>
                <span className={`health-badge ${node.is_healthy ? 'healthy' : 'unhealthy'}`}>
                  {node.is_healthy ? '✓ Healthy' : '✗ Unhealthy'}
                </span>
                {node.health_status && (
                  <span className="health-status-text"> ({node.health_status})</span>
                )}
              </td>
              <td>
                {node.last_health_check ? (
                  <span className="timestamp">{new Date(node.last_health_check).toLocaleString()}</span>
                ) : (
                  <span className="timestamp">Never</span>
                )}
              </td>
              <td className="actions-cell">
                <button
                  className="btn-small btn-edit"
                  onClick={() => onEdit(node)}
                  title="Edit node"
                >
                  Edit
                </button>
                <button
                  className="btn-small btn-delete"
                  onClick={() => onDelete(node)}
                  title="Delete node"
                >
                  Delete
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      <div className="table-footer">
        <p><strong>Total Nodes:</strong> {nodes.length}</p>
        <p><strong>Active Nodes:</strong> {nodes.filter(n => n.is_active).length}</p>
        <p><strong>Healthy Nodes:</strong> {nodes.filter(n => n.is_healthy).length}</p>
      </div>
    </div>
  );
};

export default ReplicationNodeTable;
