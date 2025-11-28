import React from 'react';
import { Database, Eye, Copy, Trash2, HardDrive, Cloud, Network, Globe, Lock } from 'lucide-react';
import { Badge, StatusIndicator } from '../../../../components/common';
import { useTranslation } from '../../../../hooks/useTranslation';

export const RepositoryRow = ({ repo, onClick }) => {
  const { t } = useTranslation('repositories');
  
  // Format the date
  const formatDate = (dateString) => {
    if (!dateString) return 'N/A';
    const date = new Date(dateString);
    return date.toLocaleDateString('en-US', { 
      year: 'numeric', 
      month: 'short', 
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  };

  // Determine repository storage type
  const getRepositoryInfo = () => {
    if (repo.cloud_provider || repo.cloud_region) {
      return {
        type: t('storageTypes.cloud'),
        icon: Cloud,
        color: 'text-purple-600 bg-purple-50',
        badge: `${repo.cloud_provider?.toUpperCase() || 'Cloud'} ${repo.cloud_region ? `• ${repo.cloud_region}` : ''}`,
        badgeColor: 'bg-purple-100 text-purple-700'
      };
    } else if (repo.remote_url) {
      return {
        type: t('storageTypes.remote'),
        icon: Network,
        color: 'text-blue-600 bg-blue-50',
        badge: new URL(repo.remote_url).hostname,
        badgeColor: 'bg-blue-100 text-blue-700'
      };
    } else {
      return {
        type: t('storageTypes.local'),
        icon: HardDrive,
        color: 'text-green-600 bg-green-50',
        badge: t('storageTypes.local'),
        badgeColor: 'bg-green-100 text-green-700'
      };
    }
  };

  const repoInfo = getRepositoryInfo();
  const Icon = repoInfo.icon;

  return (
    <tr className="hover:bg-gray-50 transition cursor-pointer" onClick={onClick}>
      <td className="px-6 py-4 whitespace-nowrap">
        <div className="flex items-center space-x-3">
          <div className={`relative p-2 rounded-lg ${repoInfo.color}`}>
            <Icon className="w-4 h-4" />
            {/* Public access badge overlay */}
            {repo.public_access && (
              <div className="absolute -top-1 -right-1 p-0.5 bg-amber-500 rounded-full" title="Public Repository">
                <Globe className="w-2.5 h-2.5 text-white" />
              </div>
            )}
          </div>
          <div>
            <div className="flex items-center space-x-2">
              <span className="font-medium text-gray-900">{repo.name}</span>
              {repo.public_access && (
                <Lock className="w-3 h-3 text-amber-600" style={{ transform: 'rotate(180deg)' }} title="Public - Anyone can read" />
              )}
            </div>
            <div className="text-xs text-gray-500">{repoInfo.type}</div>
          </div>
        </div>
      </td>
      <td className="px-6 py-4 whitespace-nowrap">
        <div className="flex flex-col space-y-1">
          <Badge>{repo.type}</Badge>
          <span className={`inline-flex items-center px-2 py-0.5 text-xs font-medium rounded ${repoInfo.badgeColor}`}>
            {repoInfo.badge}
          </span>
          {/* Public Access Indicator */}
          {repo.public_access && (
            <span className="inline-flex items-center space-x-1 px-2 py-0.5 text-xs font-medium rounded bg-amber-100 text-amber-700">
              <Globe className="w-3 h-3" />
              <span>{t('storageTypes.public')}</span>
            </span>
          )}
        </div>
      </td>
      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">{repo.total_size || '0 B'}</td>
      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">{(repo.artifact_count || 0).toLocaleString()}</td>
      <td className="px-6 py-4 whitespace-nowrap">
        <StatusIndicator status={repo.status} />
      </td>
      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">{formatDate(repo.updated_at)}</td>
      <td className="px-6 py-4 whitespace-nowrap text-sm">
        <div className="flex space-x-2">
          <button className="p-1 hover:bg-gray-100 rounded transition" onClick={(e) => e.stopPropagation()}>
            <Eye className="w-4 h-4 text-gray-600" />
          </button>
          <button className="p-1 hover:bg-gray-100 rounded transition" onClick={(e) => e.stopPropagation()}>
            <Copy className="w-4 h-4 text-gray-600" />
          </button>
          <button className="p-1 hover:bg-gray-100 rounded transition" onClick={(e) => e.stopPropagation()}>
            <Trash2 className="w-4 h-4 text-red-600" />
          </button>
        </div>
      </td>
    </tr>
  );
};
