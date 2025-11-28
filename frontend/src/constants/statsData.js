import { Database, Package, Download, Users } from 'lucide-react';

export const STATS_DATA = [
  { label: 'Total Storage', value: '335.5 GB', icon: Database, trend: '+12%', color: 'blue' },
  { label: 'Total Artifacts', value: '27,504', icon: Package, trend: '+234', color: 'green' },
  { label: 'Downloads Today', value: '15,847', icon: Download, trend: '+8%', color: 'purple' },
  { label: 'Active Users', value: '142', icon: Users, trend: '+5', color: 'orange' }
];

// Export icons for use in services
export { Database, Package, Download, Users };
