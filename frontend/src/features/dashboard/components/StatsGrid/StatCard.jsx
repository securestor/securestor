import React from 'react';
import { Card } from '../../../../components/common';
import statsService from '../../../../services/statsService';

export const StatCard = ({ stat, loading = false, lastUpdated }) => {
  const colorClasses = {
    blue: 'bg-blue-100 text-blue-600',
    green: 'bg-green-100 text-green-600',
    purple: 'bg-purple-100 text-purple-600',
    orange: 'bg-orange-100 text-orange-600'
  };

  const trendClasses = {
    positive: 'text-green-600',
    negative: 'text-red-600',
    neutral: 'text-gray-500'
  };

  if (!stat) {
    return (
      <Card className="p-6 hover:shadow-lg transition">
        <div className="animate-pulse">
          <div className="flex items-center justify-between mb-4">
            <div className="w-12 h-12 bg-gray-200 rounded-lg"></div>
            <div className="w-16 h-4 bg-gray-200 rounded"></div>
          </div>
          <div className="w-20 h-8 bg-gray-200 rounded mb-1"></div>
          <div className="w-24 h-4 bg-gray-200 rounded"></div>
        </div>
      </Card>
    );
  }

  const trend = statsService.formatTrend(stat.trend);
  const trendClass = trend.isPositive ? trendClasses.positive : 
                     trend.isNegative ? trendClasses.negative : 
                     trendClasses.neutral;

  return (
    <Card className={`p-6 hover:shadow-lg transition ${loading ? 'opacity-75' : ''}`}>
      <div className="flex items-center justify-between mb-4">
        <div className={`p-3 rounded-lg ${colorClasses[stat.color]}`}>
          <stat.icon className="w-6 h-6" />
        </div>
        <div className="text-right">
          <span className={`text-sm font-medium ${trendClass}`}>{trend.value}</span>
          {loading && (
            <div className="w-2 h-2 bg-blue-600 rounded-full animate-pulse mt-1 ml-auto"></div>
          )}
        </div>
      </div>
      <h3 className="text-2xl font-bold text-gray-900 mb-1">{stat.value}</h3>
      <p className="text-sm text-gray-600">{stat.label}</p>
      
      {lastUpdated && (
        <div className="mt-2 text-xs text-gray-400">
          Updated: {lastUpdated.toLocaleTimeString()}
        </div>
      )}
    </Card>
  );
};
