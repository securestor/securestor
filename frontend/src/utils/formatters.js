// Utility functions for formatting data in the UI

/**
 * Format bytes to human-readable format
 * @param {number} bytes - Number of bytes
 * @param {number} decimals - Number of decimal places
 * @returns {string} Formatted string (e.g., "1.5 MB")
 */
export const formatBytes = (bytes, decimals = 2) => {
  if (bytes === 0) return '0 Bytes';
  if (bytes === null || bytes === undefined) return 'N/A';
  
  const k = 1024;
  const dm = decimals < 0 ? 0 : decimals;
  const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB', 'PB'];
  
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  
  return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
};

/**
 * Format timestamp to relative time (e.g., "2 hours ago")
 * @param {string|Date} timestamp - ISO timestamp or Date object
 * @returns {string} Relative time string
 */
export const formatRelativeTime = (timestamp) => {
  if (!timestamp) return 'Never';
  
  const now = new Date();
  const then = new Date(timestamp);
  const seconds = Math.floor((now - then) / 1000);
  
  if (seconds < 0) return 'In the future';
  if (seconds < 60) return `${seconds}s ago`;
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`;
  if (seconds < 2592000) return `${Math.floor(seconds / 86400)}d ago`;
  if (seconds < 31536000) return `${Math.floor(seconds / 2592000)}mo ago`;
  
  return `${Math.floor(seconds / 31536000)}y ago`;
};

/**
 * Format absolute timestamp
 * @param {string|Date} timestamp - ISO timestamp or Date object
 * @param {boolean} includeTime - Whether to include time
 * @returns {string} Formatted date string
 */
export const formatTimestamp = (timestamp, includeTime = true) => {
  if (!timestamp) return 'N/A';
  
  const date = new Date(timestamp);
  
  if (includeTime) {
    return date.toLocaleString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  }
  
  return date.toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric'
  });
};

/**
 * Format duration in milliseconds
 * @param {number} ms - Duration in milliseconds
 * @returns {string} Formatted duration
 */
export const formatDuration = (ms) => {
  if (ms === null || ms === undefined) return 'N/A';
  if (ms < 0) return 'Invalid';
  
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  if (ms < 3600000) return `${(ms / 60000).toFixed(1)}m`;
  
  return `${(ms / 3600000).toFixed(1)}h`;
};

/**
 * Format percentage
 * @param {number} value - Percentage value (0-100)
 * @param {number} decimals - Number of decimal places
 * @returns {string} Formatted percentage
 */
export const formatPercentage = (value, decimals = 1) => {
  if (value === null || value === undefined) return 'N/A';
  
  return `${value.toFixed(decimals)}%`;
};

/**
 * Format number with commas
 * @param {number} num - Number to format
 * @returns {string} Formatted number
 */
export const formatNumber = (num) => {
  if (num === null || num === undefined) return 'N/A';
  
  return num.toLocaleString('en-US');
};

/**
 * Format vulnerability severity
 * @param {string} severity - Severity level
 * @returns {object} Color and label info
 */
export const formatSeverity = (severity) => {
  const severityMap = {
    critical: { color: 'red', label: 'Critical', bgColor: 'bg-red-100', textColor: 'text-red-800' },
    high: { color: 'orange', label: 'High', bgColor: 'bg-orange-100', textColor: 'text-orange-800' },
    medium: { color: 'yellow', label: 'Medium', bgColor: 'bg-yellow-100', textColor: 'text-yellow-800' },
    low: { color: 'blue', label: 'Low', bgColor: 'bg-blue-100', textColor: 'text-blue-800' },
    info: { color: 'gray', label: 'Info', bgColor: 'bg-gray-100', textColor: 'text-gray-800' }
  };
  
  return severityMap[severity?.toLowerCase()] || severityMap.info;
};

/**
 * Format scan status
 * @param {string} status - Scan status
 * @returns {object} Color and label info
 */
export const formatScanStatus = (status) => {
  const statusMap = {
    pending: { color: 'gray', label: 'Pending', bgColor: 'bg-gray-100', textColor: 'text-gray-800' },
    queued: { color: 'blue', label: 'Queued', bgColor: 'bg-blue-100', textColor: 'text-blue-800' },
    scanning: { color: 'yellow', label: 'Scanning', bgColor: 'bg-yellow-100', textColor: 'text-yellow-800' },
    completed: { color: 'green', label: 'Completed', bgColor: 'bg-green-100', textColor: 'text-green-800' },
    failed: { color: 'red', label: 'Failed', bgColor: 'bg-red-100', textColor: 'text-red-800' },
    quarantined: { color: 'purple', label: 'Quarantined', bgColor: 'bg-purple-100', textColor: 'text-purple-800' }
  };
  
  return statusMap[status?.toLowerCase()] || statusMap.pending;
};

/**
 * Format cache level
 * @param {string} level - Cache level (L1, L2, L3)
 * @returns {object} Color and label info
 */
export const formatCacheLevel = (level) => {
  const levelMap = {
    L1: { color: 'green', label: 'L1 (Redis)', bgColor: 'bg-green-100', textColor: 'text-green-800' },
    L2: { color: 'yellow', label: 'L2 (Disk)', bgColor: 'bg-yellow-100', textColor: 'text-yellow-800' },
    L3: { color: 'blue', label: 'L3 (Cloud)', bgColor: 'bg-blue-100', textColor: 'text-blue-800' }
  };
  
  return levelMap[level] || { color: 'gray', label: level, bgColor: 'bg-gray-100', textColor: 'text-gray-800' };
};

/**
 * Format artifact type
 * @param {string} type - Artifact type
 * @returns {object} Color and label info
 */
export const formatArtifactType = (type) => {
  const typeMap = {
    maven: { color: 'orange', label: 'Maven', bgColor: 'bg-orange-100', textColor: 'text-orange-800' },
    npm: { color: 'red', label: 'npm', bgColor: 'bg-red-100', textColor: 'text-red-800' },
    pypi: { color: 'blue', label: 'PyPI', bgColor: 'bg-blue-100', textColor: 'text-blue-800' },
    helm: { color: 'purple', label: 'Helm', bgColor: 'bg-purple-100', textColor: 'text-purple-800' },
    docker: { color: 'cyan', label: 'Docker', bgColor: 'bg-cyan-100', textColor: 'text-cyan-800' }
  };
  
  return typeMap[type?.toLowerCase()] || { color: 'gray', label: type, bgColor: 'bg-gray-100', textColor: 'text-gray-800' };
};

/**
 * Truncate string with ellipsis
 * @param {string} str - String to truncate
 * @param {number} length - Max length
 * @returns {string} Truncated string
 */
export const truncate = (str, length = 50) => {
  if (!str) return '';
  if (str.length <= length) return str;
  
  return str.substring(0, length) + '...';
};

/**
 * Get risk score color
 * @param {number} score - Risk score (0-100)
 * @returns {string} Tailwind color class
 */
export const getRiskScoreColor = (score) => {
  if (score === null || score === undefined) return 'gray';
  if (score >= 80) return 'red';
  if (score >= 60) return 'orange';
  if (score >= 40) return 'yellow';
  if (score >= 20) return 'blue';
  return 'green';
};

/**
 * Format checksum for display
 * @param {string} checksum - Full checksum string
 * @param {number} length - Number of characters to show from start and end
 * @returns {string} Formatted checksum
 */
export const formatChecksum = (checksum, length = 8) => {
  if (!checksum) return 'N/A';
  if (checksum.length <= length * 2) return checksum;
  
  return `${checksum.substring(0, length)}...${checksum.substring(checksum.length - length)}`;
};

/**
 * Format CVE ID with link
 * @param {string} cveId - CVE identifier
 * @returns {object} CVE info with link
 */
export const formatCVE = (cveId) => {
  if (!cveId || !cveId.startsWith('CVE-')) {
    return { id: cveId, url: null };
  }
  
  return {
    id: cveId,
    url: `https://nvd.nist.gov/vuln/detail/${cveId}`
  };
};

/**
 * Calculate hit rate
 * @param {number} hits - Number of cache hits
 * @param {number} misses - Number of cache misses
 * @returns {number} Hit rate percentage
 */
export const calculateHitRate = (hits, misses) => {
  const total = hits + misses;
  if (total === 0) return 0;
  
  return (hits / total) * 100;
};

/**
 * Format priority number to label
 * @param {number} priority - Priority value (0-100)
 * @returns {object} Priority info
 */
export const formatPriority = (priority) => {
  if (priority >= 80) {
    return { label: 'Critical', color: 'red', bgColor: 'bg-red-100', textColor: 'text-red-800' };
  }
  if (priority >= 60) {
    return { label: 'High', color: 'orange', bgColor: 'bg-orange-100', textColor: 'text-orange-800' };
  }
  if (priority >= 40) {
    return { label: 'Medium', color: 'yellow', bgColor: 'bg-yellow-100', textColor: 'text-yellow-800' };
  }
  if (priority >= 20) {
    return { label: 'Low', color: 'blue', bgColor: 'bg-blue-100', textColor: 'text-blue-800' };
  }
  
  return { label: 'Lowest', color: 'gray', bgColor: 'bg-gray-100', textColor: 'text-gray-800' };
};

export default {
  formatBytes,
  formatRelativeTime,
  formatTimestamp,
  formatDuration,
  formatPercentage,
  formatNumber,
  formatSeverity,
  formatScanStatus,
  formatCacheLevel,
  formatArtifactType,
  truncate,
  getRiskScoreColor,
  formatChecksum,
  formatCVE,
  calculateHitRate,
  formatPriority
};
