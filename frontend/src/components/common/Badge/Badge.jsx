import React from 'react';

export const Badge = ({ children, variant = 'default' }) => {
  const variants = {
    default: 'bg-purple-100 text-purple-800',
    success: 'bg-green-100 text-green-800',
    warning: 'bg-yellow-100 text-yellow-800',
    info: 'bg-blue-100 text-blue-800'
  };

  return (
    <span className={`px-2 py-1 text-xs font-medium rounded ${variants[variant]}`}>
      {children}
    </span>
  );
};
