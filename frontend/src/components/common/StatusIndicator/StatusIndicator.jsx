import React from 'react';
import { CheckCircle, Clock } from 'lucide-react';

export const StatusIndicator = ({ status }) => {
  const statusConfig = {
    healthy: { icon: CheckCircle, color: 'text-green-600', label: 'Healthy' },
    syncing: { icon: Clock, color: 'text-yellow-600', label: 'Syncing' }
  };

  const { icon: Icon, color, label } = statusConfig[status] || statusConfig.healthy;

  return (
    <span className={`flex items-center text-sm ${color}`}>
      <Icon className="w-4 h-4 mr-1" />
      {label}
    </span>
  );
};
