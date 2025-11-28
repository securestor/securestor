import React from 'react';

export const Button = ({ variant = 'primary', children, onClick, icon: Icon, className = '' }) => {
  const baseStyles = 'flex items-center space-x-2 px-4 py-2.5 rounded-lg transition font-medium';
  const variants = {
    primary: 'bg-blue-600 text-white hover:bg-blue-700',
    secondary: 'bg-white border border-gray-300 hover:bg-gray-50 text-gray-700',
    ghost: 'hover:bg-gray-100 text-gray-600'
  };

  return (
    <button onClick={onClick} className={`${baseStyles} ${variants[variant]} ${className}`}>
      {Icon && <Icon className="w-4 h-4" />}
      <span>{children}</span>
    </button>
  );
};
