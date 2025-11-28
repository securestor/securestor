export const getSeverityColor = (severity) => {
  const severityMap = {
    critical: 'text-red-600 bg-red-100 border-red-200',
    high: 'text-orange-600 bg-orange-100 border-orange-200',
    medium: 'text-yellow-600 bg-yellow-100 border-yellow-200',
    low: 'text-blue-600 bg-blue-100 border-blue-200',
    unknown: 'text-gray-600 bg-gray-100 border-gray-200'
  };

  return severityMap[severity?.toLowerCase()] || severityMap.unknown;
};

export const getSeverityBadgeColor = (severity) => {
  const colorMap = {
    critical: 'bg-red-500',
    high: 'bg-orange-500',
    medium: 'bg-yellow-500',
    low: 'bg-blue-500'
  };

  return colorMap[severity?.toLowerCase()] || 'bg-gray-500';
};

export const sortBySeverity = (vulnerabilities) => {
  const severityOrder = { critical: 0, high: 1, medium: 2, low: 3, unknown: 4 };
  
  return [...vulnerabilities].sort((a, b) => {
    const aSeverity = severityOrder[a.severity?.toLowerCase()] ?? 4;
    const bSeverity = severityOrder[b.severity?.toLowerCase()] ?? 4;
    return aSeverity - bSeverity;
  });
};