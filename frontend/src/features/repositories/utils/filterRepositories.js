export const filterRepositories = (repositories, searchTerm) => {
  return repositories.filter(repo =>
    repo.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
    repo.type.toLowerCase().includes(searchTerm.toLowerCase())
  );
};
