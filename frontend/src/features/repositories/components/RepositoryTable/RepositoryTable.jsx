import React, { useState, useEffect } from 'react';
import { Plus, HardDrive, Cloud, Network } from 'lucide-react';
import { RepositoryRow } from './RepositoryRow';
import { useDashboard } from '../../../../context/DashboardContext';
import { Card, CardHeader, CardContent } from '../../../../components/common';
import { CreateRepositoryModal } from '../../../../components/modals';
import { filterRepositories } from '../../utils';
import { useToast } from '../../../../context/ToastContext';
import { useTranslation } from '../../../../hooks/useTranslation';

export const RepositoryTable = () => {
  const { t } = useTranslation('repositories');
  const { 
    searchTerm, 
    setSelectedRepo, 
    repositories,
    repositoriesLoading: loading,
    repositoriesError: error,
    fetchRepositories,
    addRepository,
    refreshRepositories
  } = useDashboard();
  
  const [showCreateRepo, setShowCreateRepo] = useState(false);
  const filteredRepos = filterRepositories(repositories, searchTerm);
  const { showSuccess } = useToast();

  // Fetch repositories on mount
  useEffect(() => {
    fetchRepositories();
  }, [fetchRepositories]);

  const handleCreateRepository = async (response) => {
    
    // Add the new repository to state for immediate update
    if (response && response.repository) {
      addRepository(response.repository);
    }
    
    // Also refresh from server to ensure consistency
    await refreshRepositories();
    
    // Then show success message
    const repoName = response?.repository?.name || 'Repository';
    showSuccess(t('createSuccess', { name: repoName }));
  };

  return (
    <>
      <Card className="overflow-hidden">
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-lg font-semibold text-gray-900">{t('title')}</h2>
              <div className="flex items-center space-x-4 mt-2">
                <div className="flex items-center space-x-1.5 text-xs text-gray-600">
                  <div className="p-1 rounded bg-green-50">
                    <HardDrive className="w-3 h-3 text-green-600" />
                  </div>
                  <span>{t('legend.local')}</span>
                </div>
                <div className="flex items-center space-x-1.5 text-xs text-gray-600">
                  <div className="p-1 rounded bg-blue-50">
                    <Network className="w-3 h-3 text-blue-600" />
                  </div>
                  <span>{t('legend.remoteProxy')}</span>
                </div>
                <div className="flex items-center space-x-1.5 text-xs text-gray-600">
                  <div className="p-1 rounded bg-purple-50">
                    <Cloud className="w-3 h-3 text-purple-600" />
                  </div>
                  <span>{t('legend.cloudStorage')}</span>
                </div>
              </div>
            </div>
            <button
              onClick={() => setShowCreateRepo(true)}
              className="flex items-center space-x-2 px-3 py-2 text-sm font-medium text-blue-600 hover:bg-blue-50 rounded-lg transition"
            >
              <Plus className="w-4 h-4" />
              <span>{t('addRepository')}</span>
            </button>
          </div>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="text-center py-12">
              <p className="text-gray-500">{t('loading')}</p>
            </div>
          ) : error ? (
            <div className="text-center py-12">
              <p className="text-red-500 mb-4">{t('error')}: {error}</p>
              <button
                onClick={refreshRepositories}
                className="inline-flex items-center space-x-2 px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 transition"
              >
                <span>{t('retry')}</span>
              </button>
            </div>
          ) : filteredRepos.length === 0 ? (
            <div className="text-center py-12">
              <p className="text-gray-500 mb-4">{t('noRepositories')}</p>
              <button
                onClick={() => setShowCreateRepo(true)}
                className="inline-flex items-center space-x-2 px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 transition"
              >
                <Plus className="w-4 h-4" />
                <span>Create First Repository</span>
              </button>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Repository</th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Type</th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Size</th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Artifacts</th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Last Modified</th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Actions</th>
                  </tr>
                </thead>
                <tbody className="bg-white divide-y divide-gray-200">
                  {filteredRepos.map(repo => (
                    <RepositoryRow key={repo.id} repo={repo} onClick={() => setSelectedRepo(repo)} />
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>

      <CreateRepositoryModal
        isOpen={showCreateRepo}
        onClose={() => setShowCreateRepo(false)}
        onSubmit={handleCreateRepository}
      />
    </>
  );
};