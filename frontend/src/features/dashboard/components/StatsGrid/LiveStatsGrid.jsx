import React, { useState } from 'react';
import { StatsGrid } from './StatsGrid';
import { Card } from '../../../../components/common';
import { RefreshCw, Zap, ZapOff } from 'lucide-react';

export const LiveStatsGrid = () => {
  const [realtimeEnabled, setRealtimeEnabled] = useState(false);
  const [autoRefresh, setAutoRefresh] = useState(true);

  const toggleRealtime = () => {
    setRealtimeEnabled(prev => !prev);
  };

  const toggleAutoRefresh = () => {
    setAutoRefresh(prev => !prev);
  };

  return (
    <div className="space-y-4">
      {/* Controls Panel */}
      <Card className="p-4">
        <div className="flex items-center justify-between">
          <h3 className="text-lg font-semibold text-gray-900">Dashboard Statistics</h3>
          
          <div className="flex items-center space-x-4">
            {/* Auto Refresh Toggle */}
            <div className="flex items-center space-x-2">
              <RefreshCw 
                className={`w-4 h-4 ${autoRefresh ? 'text-blue-600' : 'text-gray-400'}`} 
              />
              <label className="flex items-center cursor-pointer">
                <input
                  type="checkbox"
                  checked={autoRefresh}
                  onChange={toggleAutoRefresh}
                  className="sr-only"
                />
                <div className="relative">
                  <div className={`block bg-gray-600 w-14 h-8 rounded-full ${autoRefresh ? 'bg-blue-600' : 'bg-gray-300'}`}></div>
                  <div className={`dot absolute left-1 top-1 bg-white w-6 h-6 rounded-full transition ${autoRefresh ? 'transform translate-x-6' : ''}`}></div>
                </div>
              </label>
              <span className="text-sm text-gray-700">Auto Refresh</span>
            </div>

            {/* Real-time Toggle */}
            <div className="flex items-center space-x-2">
              {realtimeEnabled ? (
                <Zap className="w-4 h-4 text-yellow-500" />
              ) : (
                <ZapOff className="w-4 h-4 text-gray-400" />
              )}
              <label className="flex items-center cursor-pointer">
                <input
                  type="checkbox"
                  checked={realtimeEnabled}
                  onChange={toggleRealtime}
                  className="sr-only"
                />
                <div className="relative">
                  <div className={`block w-14 h-8 rounded-full ${realtimeEnabled ? 'bg-yellow-500' : 'bg-gray-300'}`}></div>
                  <div className={`dot absolute left-1 top-1 bg-white w-6 h-6 rounded-full transition ${realtimeEnabled ? 'transform translate-x-6' : ''}`}></div>
                </div>
              </label>
              <span className="text-sm text-gray-700">Real-time</span>
            </div>

            {realtimeEnabled && (
              <div className="flex items-center space-x-1">
                <div className="w-2 h-2 bg-green-500 rounded-full animate-pulse"></div>
                <span className="text-xs text-green-600">Live</span>
              </div>
            )}
          </div>
        </div>

        {realtimeEnabled && (
          <div className="mt-3 p-3 bg-yellow-50 border border-yellow-200 rounded-md">
            <div className="flex">
              <Zap className="w-4 h-4 text-yellow-600 mt-0.5 mr-2" />
              <div>
                <p className="text-sm text-yellow-800 font-medium">Real-time Updates Enabled</p>
                <p className="text-xs text-yellow-700">Statistics will update automatically as new data arrives.</p>
              </div>
            </div>
          </div>
        )}
      </Card>

      {/* Stats Grid */}
      <StatsGrid 
        enableRealtime={realtimeEnabled}
        autoRefresh={autoRefresh}
      />
    </div>
  );
};