import React, { useState } from 'react';
import { Search, Download, Upload, Plus } from 'lucide-react';
import { useDashboard } from '../../../../context/DashboardContext';
import { Card, Button } from '../../../../components/common';
import { CreateRepositoryModal, DeployArtifactModal, ImportArtifactModal } from '../../../../components/modals';
import { useToast } from '../../../../context/ToastContext';
import { useTranslation } from '../../../../hooks/useTranslation';

export const SearchBar = () => {
  const { t } = useTranslation('repositories');
  const { searchTerm, setSearchTerm, addRepository, refreshRepositories, repositories } = useDashboard();
  const [showCreateRepo, setShowCreateRepo] = useState(false);
  const [showDeployModal, setShowDeployModal] = useState(false);
  const [showImportModal, setShowImportModal] = useState(false);
  const { showSuccess } = useToast();

  const handleCreateRepository = async (response) => {
    
    // Add the new repository to global state for immediate update
    if (response && response.repository) {
      addRepository(response.repository);
    }
    
    // Refresh from server to ensure consistency
    await refreshRepositories();
    
    const repoName = response?.repository?.name || 'Repository';
    showSuccess(t('createSuccess', { name: repoName }));
  };

  const handleDeployArtifact = (artifactData) => {
    
    // Extract artifact name and version from the response data
    // The response can have either 'name'/'version' (from backend) or 'artifactName'/'version' (from frontend)
    const name = artifactData.name || artifactData.Name || artifactData.artifactName || 'Unknown';
    const version = artifactData.version || artifactData.Version || 'Unknown';
    
    // Extract repository name and clean it up
    let repository = artifactData.repository || artifactData.Repository || 'Unknown';
    
    // Clean up repository name if it has extra characters
    if (typeof repository === 'string') {
      repository = repository.replace(/[!@#$%^&*()]+/g, '').trim();
      if (!repository) {
        repository = 'Unknown';
      }
    }
    
    
    // Alert removed - success message is already shown by the modal
  };

  const handleImportArtifact = (importData) => {
    // TODO: Implement API call to import artifact
    alert(`Artifact import started for repository: ${importData.repository}`);
  };

  return (
    <>
      <Card className="p-6 mb-6">
        <div className="flex items-center justify-between flex-wrap gap-4">
          <div className="flex-1 max-w-md relative">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-5 h-5 text-gray-400" />
            <input
              type="text"
              placeholder={t('searchPlaceholder')}
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full pl-10 pr-4 py-2.5 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>
          <div className="flex space-x-3">
            <Button variant="secondary" icon={Plus} onClick={() => setShowCreateRepo(true)}>
              {t('newRepository')}
            </Button>
            <Button variant="secondary" icon={Download} onClick={() => setShowImportModal(true)}>
              {t('import')}
            </Button>
            <Button variant="primary" icon={Upload} onClick={() => setShowDeployModal(true)}>
              {t('deployArtifact')}
            </Button>
          </div>
        </div>
      </Card>

      {/* Modals */}
      <CreateRepositoryModal
        isOpen={showCreateRepo}
        onClose={() => setShowCreateRepo(false)}
        onSubmit={handleCreateRepository}
      />

      <DeployArtifactModal
        isOpen={showDeployModal}
        onClose={() => setShowDeployModal(false)}
        onSubmit={handleDeployArtifact}
      />

      <ImportArtifactModal
        isOpen={showImportModal}
        onClose={() => setShowImportModal(false)}
        onSubmit={handleImportArtifact}
        repositories={repositories}
      />
    </>
  );
};
