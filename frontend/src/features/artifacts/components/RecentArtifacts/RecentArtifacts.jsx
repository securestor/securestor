import React from 'react';
import { ArtifactItem } from './ArtifactItem';
import { Card, CardHeader, CardContent } from '../../../../components/common';
import { useRecentArtifacts } from '../../../../hooks/useArtifacts';

export const RecentArtifacts = () => {
  const { artifacts, loading, error, lastUpdated } = useRecentArtifacts({
    autoRefresh: true,
    refreshInterval: 30000 // Refresh every 30 seconds
  });

  return (
    <Card className="mt-6 overflow-hidden">
      <CardHeader>
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold text-gray-900">Recent Artifacts</h2>
          {lastUpdated && (
            <span className="text-xs text-gray-500">
              Last updated: {lastUpdated.toLocaleTimeString()}
            </span>
          )}
        </div>
      </CardHeader>
      <CardContent className="divide-y divide-gray-200">
        {loading && artifacts.length === 0 ? (
          <div className="py-4 text-center">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto"></div>
            <p className="mt-2 text-sm text-gray-500">Loading artifacts...</p>
          </div>
        ) : error ? (
          <div className="py-4 text-center">
            <div className="bg-yellow-50 border border-yellow-200 rounded-md p-3">
              <p className="text-sm text-yellow-800">
                Using cached data: {error}
              </p>
            </div>
          </div>
        ) : artifacts.length === 0 ? (
          <div className="py-8 text-center">
            <p className="text-sm text-gray-500">No artifacts found</p>
            <p className="text-xs text-gray-400 mt-1">Upload some artifacts to see them here</p>
          </div>
        ) : (
          artifacts.map(artifact => (
            <ArtifactItem key={artifact.id} artifact={artifact} />
          ))
        )}
      </CardContent>
    </Card>
  );
};
