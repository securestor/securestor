import React from 'react';

export const CardContent = ({ children, className = '' }) => (
  <div className={className}>
    {children}
  </div>
);
