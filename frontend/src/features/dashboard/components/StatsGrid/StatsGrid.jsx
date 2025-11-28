import React from 'react';
import { StatCard } from './StatCard';
import { useStats } from '../../../../hooks/useStats';
import { STATS_DATA } from '../../../../constants';

export const StatsGrid = ({ enableRealtime = false, autoRefresh = true }) => {
  const { stats, loading, error, lastUpdated } = useStats({
    autoRefresh,
    refreshInterval: 30000, // 30 seconds
    enableRealtime
  });

  // Fallback to static data if there's an error or no stats
  const displayStats = stats || {
    totalStorage: STATS_DATA[0],
    totalArtifacts: STATS_DATA[1], 
    downloadsToday: STATS_DATA[2],
    activeUsers: STATS_DATA[3]
  };

  const statsArray = [
    displayStats.totalStorage,
    displayStats.totalArtifacts,
    displayStats.downloadsToday,
    displayStats.activeUsers
  ];

  return (
    <div className="space-y-4">
      {error && (
        <div className="bg-yellow-50 border border-yellow-200 rounded-md p-3">
          <div className="flex">
            <div className="ml-3">
              <p className="text-sm text-yellow-800">
                Using cached data: {error}
              </p>
            </div>
          </div>
        </div>
      )}
      
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-6">
        {statsArray.map((stat, idx) => (
          <StatCard 
            key={idx} 
            stat={stat} 
            loading={loading && !stats}
            lastUpdated={lastUpdated}
          />
        ))}
      </div>
      
      {lastUpdated && (
        <div className="text-xs text-gray-500 text-right">
          Last updated: {lastUpdated.toLocaleTimeString()}
        </div>
      )}
    </div>
  );
};
