import React, { useState, useCallback } from 'react';
import { DashboardContext } from './DashboardContext';
import repositoryAPI from '../../services/api/repositoryAPI';

export const DashboardProvider = ({ children }) => {
  const [activeTab, setActiveTab] = useState('overview');
  const [selectedRepo, setSelectedRepo] = useState(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [repositories, setRepositories] = useState([]);
  const [repositoriesLoading, setRepositoriesLoading] = useState(false);
  const [repositoriesError, setRepositoriesError] = useState(null);

  // Fetch repositories from API
  const fetchRepositories = useCallback(async () => {
    try {
      setRepositoriesLoading(true);
      setRepositoriesError(null);
      const data = await repositoryAPI.listRepositories();
      setRepositories(data.repositories || []);
      return data.repositories || [];
    } catch (err) {
      console.error('Failed to fetch repositories:', err);
      setRepositoriesError(err.message);
      return [];
    } finally {
      setRepositoriesLoading(false);
    }
  }, []);

  // Add a new repository to the list (optimistic update)
  const addRepository = useCallback((repository) => {
    setRepositories(prevRepos => [...prevRepos, repository]);
  }, []);

  // Update a repository in the list
  const updateRepository = useCallback((updatedRepo) => {
    setRepositories(prevRepos =>
      prevRepos.map(repo =>
        repo.id === updatedRepo.id ? { ...repo, ...updatedRepo } : repo
      )
    );
  }, []);

  // Remove a repository from the list
  const removeRepository = useCallback((repositoryId) => {
    setRepositories(prevRepos =>
      prevRepos.filter(repo => repo.id !== repositoryId)
    );
  }, []);

  // Refresh repositories (can be called from anywhere)
  const refreshRepositories = useCallback(async () => {
    await fetchRepositories();
  }, [fetchRepositories]);

  return (
    <DashboardContext.Provider value={{
      activeTab,
      setActiveTab,
      selectedRepo,
      setSelectedRepo,
      searchTerm,
      setSearchTerm,
      // Repository management
      repositories,
      repositoriesLoading,
      repositoriesError,
      fetchRepositories,
      addRepository,
      updateRepository,
      removeRepository,
      refreshRepositories,
    }}>
      {children}
    </DashboardContext.Provider>
  );
};
