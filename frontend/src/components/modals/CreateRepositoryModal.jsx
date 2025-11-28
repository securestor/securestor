import React, { useState } from "react";
import { Modal } from "./Modal";
import { Database, Info } from "lucide-react";
import RepositoryAPI from "../../services/api/repositoryAPI";
import { useToast } from "../../context/ToastContext";
import { useTranslation } from "../../hooks/useTranslation";

export const CreateRepositoryModal = ({ isOpen, onClose, onSubmit }) => {
  const { t } = useTranslation('repositories');
  const { showError, showWarning } = useToast();
  const [formData, setFormData] = useState({
    name: "",
    type: "docker",
    repositoryType: "local",
    description: "",
    publicAccess: false,
    enableIndexing: true,
    remoteUrl: "",
    username: "",
    password: "",
    enableEncryption: false,
    encryptionKey: "",
    // Cloud storage specific fields
    cloudProvider: "s3",
    region: "",
    bucketName: "",
    accessKeyId: "",
    secretAccessKey: "",
    endpoint: "",
    githubToken: "",
    githubOrg: "",
    githubRepo: "",
  });

  const [errors, setErrors] = useState({});
  const [loading, setLoading] = useState(false);

  // Organized repository types by category for better scalability
  const repositoryCategories = [
    {
      name: t('create.categories.popular'),
      icon: "⭐",
      types: [
        { value: "docker", label: "Docker / OCI", icon: "🐳", description: "Container images" },
        { value: "npm", label: "npm", icon: "📦", description: "Node.js packages" },
        { value: "maven", label: "Maven", icon: "☕", description: "Java artifacts" },
        { value: "pypi", label: "PyPI", icon: "🐍", description: "Python packages" },
        { value: "generic", label: "Generic", icon: "📁", description: "Universal storage" },
      ]
    },
    {
      name: t('create.categories.container'),
      icon: "☸️",
      types: [
        { value: "docker", label: "Docker / OCI", icon: "🐳", description: "Container images" },
        { value: "helm", label: "Helm", icon: "⎈", description: "Kubernetes packages" },
      ]
    },
    {
      name: t('create.categories.languages'),
      icon: "💻",
      types: [
        { value: "npm", label: "npm", icon: "📦", description: "Node.js packages" },
        { value: "maven", label: "Maven", icon: "☕", description: "Java artifacts" },
        { value: "pypi", label: "PyPI", icon: "🐍", description: "Python packages" },
        { value: "go", label: "Go Modules", icon: "🐹", description: "Go packages" },
        { value: "cargo", label: "Cargo", icon: "🦀", description: "Rust crates" },
        { value: "nuget", label: "NuGet", icon: "🎯", description: ".NET packages" },
        { value: "rubygems", label: "RubyGems", icon: "💎", description: "Ruby gems" },
        { value: "composer", label: "Composer", icon: "�", description: "PHP packages" },
      ]
    },
    {
      name: t('create.categories.aiMl'),
      icon: "🤖",
      types: [
        { value: "model", label: "AI Model", icon: "🧠", description: "ML models & weights" },
        { value: "dataset", label: "Dataset", icon: "📊", description: "Training data" },
        { value: "notebook", label: "Notebook", icon: "📓", description: "Jupyter notebooks" },
      ]
    },
    {
      name: t('create.categories.infrastructure'),
      icon: "🛠️",
      types: [
        { value: "terraform", label: "Terraform", icon: "🌍", description: "IaC modules" },
        { value: "helm", label: "Helm", icon: "⎈", description: "K8s charts" },
        { value: "generic", label: "Generic", icon: "📁", description: "Any artifacts" },
      ]
    },
    {
      name: t('create.categories.embedded'),
      icon: "🔌",
      types: [
        { value: "firmware", label: "Firmware", icon: "�", description: "Device firmware" },
        { value: "sdk", label: "SDK", icon: "🧰", description: "Development kits" },
        { value: "fpga", label: "FPGA", icon: "⚙️", description: "FPGA bitstreams" },
        { value: "ota", label: "OTA Updates", icon: "�", description: "Over-the-air packages" },
      ]
    },
    {
      name: t('create.categories.specialized'),
      icon: "🎯",
      types: [
        { value: "marine-data", label: "Marine Data", icon: "🌊", description: "Oceanographic data" },
        { value: "simulation", label: "Simulation", icon: "💨", description: "Simulation outputs" },
        { value: "media", label: "Media", icon: "🎥", description: "Media repository" },
        { value: "video", label: "Video", icon: "📺", description: "Video files" },
        { value: "audio", label: "Audio", icon: "🎧", description: "Audio files" },
      ]
    },
  ];

  // Flatten all types for searching
  const allRepositoryTypes = repositoryCategories.flatMap(cat => cat.types);
  
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedCategory, setSelectedCategory] = useState(t('create.categories.popular'));

  const repoTypes = [
    { value: "local", label: t('types.local'), description: t('create.repositoryTypes.localDescription') },
    {
      value: "remote",
      label: t('types.remote'),
      description: t('create.repositoryTypes.remoteDescription'),
    },
    {
      value: "cloud",
      label: t('types.cloud'),
      description: t('create.repositoryTypes.cloudDescription'),
    },
  ];

  const cloudProviders = [
    {
      value: "s3",
      label: t('create.cloudProviders.s3'),
      icon: "☁️",
      description: t('create.cloudProviders.s3Description'),
    },
    {
      value: "s3-compatible",
      label: t('create.cloudProviders.s3Compatible'),
      icon: "🔄",
      description: t('create.cloudProviders.s3CompatibleDescription'),
    },
    {
      value: "github",
      label: t('create.cloudProviders.github'),
      icon: "🐙",
      description: t('create.cloudProviders.githubDescription'),
    },
    {
      value: "aws-ecr",
      label: t('create.cloudProviders.awsEcr'),
      icon: "🏗️",
      description: t('create.cloudProviders.awsEcrDescription'),
    },
    {
      value: "azure",
      label: t('create.cloudProviders.azure'),
      icon: "🔵",
      description: t('create.cloudProviders.azureDescription'),
    },
    {
      value: "gcp",
      label: t('create.cloudProviders.gcp'),
      icon: "🌐",
      description: t('create.cloudProviders.gcpDescription'),
    },
  ];

  const validateForm = () => {
    const newErrors = {};

    // Validate name
    if (!formData.name) {
      newErrors.name = t('validation.repositoryNameRequired');
    } else if (formData.name.length < 3) {
      newErrors.name = t('validation.repositoryNameTooShort');
    } else if (!/^[a-z0-9][a-z0-9-]*[a-z0-9]$/.test(formData.name)) {
      newErrors.name = t('validation.repositoryNameInvalid');
    }

    // Validate remote URL for remote repos
    if (formData.repositoryType === "remote" && !formData.remoteUrl) {
      newErrors.remoteUrl = t('validation.remoteUrlRequired');
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleChange = (e) => {
    const { name, value, type, checked } = e.target;
    setFormData((prev) => ({
      ...prev,
      [name]: type === "checkbox" ? checked : value,
    }));

    // Clear error for this field
    if (errors[name]) {
      setErrors((prev) => ({ ...prev, [name]: null }));
    }
  };

  const handleSubmit = async (e) => {
    e.preventDefault();

    // Run form validation
    if (!validateForm()) {
      return;
    }

    // Cloud storage validation
    if (formData.repositoryType === "cloud") {
      if (
        formData.cloudProvider === "s3" ||
        formData.cloudProvider === "s3-compatible"
      ) {
        if (
          !formData.accessKeyId ||
          !formData.secretAccessKey ||
          !formData.bucketName
        ) {
          showWarning(t('validation.fillS3Fields'));
          return;
        }
        if (formData.cloudProvider === "s3" && !formData.region) {
          showWarning(t('validation.awsRegionRequired'));
          return;
        }
        if (formData.cloudProvider === "s3-compatible" && !formData.endpoint) {
          showWarning(t('validation.customEndpointRequired'));
          return;
        }
      } else if (formData.cloudProvider === "github") {
        if (!formData.githubToken || !formData.githubOrg) {
          showWarning(t('validation.fillGithubFields'));
          return;
        }
      } else if (formData.cloudProvider === "aws-ecr") {
        if (
          !formData.accessKeyId ||
          !formData.secretAccessKey ||
          !formData.region
        ) {
          showWarning(t('validation.fillEcrFields'));
          return;
        }
      }
    }

    setLoading(true);
    try {
      // Call the API to create repository
      const response = await RepositoryAPI.createRepository(formData);

      // Call the parent's onSubmit callback and wait for it to complete
      // The parent will handle showing the success message
      if (onSubmit) {
        await onSubmit(response);
      }

      // Close modal and reset form
      onClose();
      resetForm();
    } catch (error) {
      console.error('Failed to create repository:', error);
      showError(t('errors.failedToCreate', { message: error.message }));
    } finally {
      setLoading(false);
    }
  };

  const resetForm = () => {
    setFormData({
      name: "",
      type: "docker",
      repositoryType: "local",
      description: "",
      publicAccess: false,
      enableIndexing: true,
      remoteUrl: "",
      username: "",
      password: "",
      enableEncryption: false,
      encryptionKey: "",
      cloudProvider: "s3",
      region: "",
      bucketName: "",
      accessKeyId: "",
      secretAccessKey: "",
      endpoint: "",
      githubToken: "",
      githubOrg: "",
      githubRepo: "",
    });
    setErrors({});
    setLoading(false);
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={t('create.title')}
      size="lg"
    >
      <form onSubmit={handleSubmit} className="space-y-6">
        {/* Repository Name */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            {t('create.nameLabel')} *
          </label>
          <input
            type="text"
            name="name"
            required
            value={formData.name}
            onChange={handleChange}
            placeholder={t('create.namePlaceholder')}
            className={`w-full px-4 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent ${errors.name ? 'border-red-500' : 'border-gray-300'
              }`}
          />
          {errors.name ? (
            <p className="mt-1 text-xs text-red-600">{errors.name}</p>
          ) : (
            <p className="mt-1 text-xs text-gray-500">
              {t('create.nameHelperText')}
            </p>
          )}
        </div>

        {/* Package Type */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            {t('create.typeLabel')} *
          </label>
          
          {/* Search Box */}
          <div className="mb-3">
            <input
              type="text"
              placeholder={t('create.searchPlaceholder')}
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent text-sm"
            />
          </div>

          {/* Category Tabs - Wrapped grid, no scrollbar */}
          {!searchQuery && (
            <div className="mb-4">
              <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-2">
                {repositoryCategories.map((category) => (
                  <button
                    key={category.name}
                    type="button"
                    onClick={() => setSelectedCategory(category.name)}
                    className={`flex items-center justify-center space-x-1.5 px-3 py-2.5 rounded-lg text-sm font-medium transition-all ${
                      selectedCategory === category.name
                        ? "bg-blue-500 text-white shadow-md ring-2 ring-blue-300"
                        : "bg-white text-gray-700 border border-gray-300 hover:border-blue-400 hover:bg-blue-50"
                    }`}
                  >
                    <span className="text-lg">{category.icon}</span>
                    <span className="truncate">{category.name}</span>
                    <span className={`text-xs ${
                      selectedCategory === category.name
                        ? "text-blue-100"
                        : "text-gray-500"
                    }`}>
                      ({category.types.length})
                    </span>
                  </button>
                ))}
              </div>
            </div>
          )}

          {/* Package Type Grid - Improved without scroll */}
          <div className="border border-gray-200 rounded-lg p-4 bg-gray-50">
            {searchQuery ? (
              // Search Results
              <>
                <div className="mb-3 text-sm text-gray-600">
                  {t('create.searchResults')} "{searchQuery}"
                </div>
                <div className="grid grid-cols-3 gap-2.5">
                  {allRepositoryTypes
                    .filter((type) =>
                      type.label.toLowerCase().includes(searchQuery.toLowerCase()) ||
                      type.description?.toLowerCase().includes(searchQuery.toLowerCase()) ||
                      type.value.toLowerCase().includes(searchQuery.toLowerCase())
                    )
                    .map((type) => (
                      <label
                        key={type.value}
                        className={`group relative flex items-center space-x-2.5 p-2.5 border rounded-lg cursor-pointer transition-all ${
                          formData.type === type.value
                            ? "border-blue-500 bg-blue-50 shadow-sm ring-2 ring-blue-200"
                            : "border-gray-300 bg-white hover:border-blue-300 hover:shadow-sm"
                        }`}
                        title={type.description}
                      >
                        <input
                          type="radio"
                          name="type"
                          value={type.value}
                          checked={formData.type === type.value}
                          onChange={handleChange}
                          className="text-blue-600 focus:ring-blue-500"
                        />
                        <span className="text-xl flex-shrink-0">{type.icon}</span>
                        <span className="text-sm font-medium text-gray-900 truncate">
                          {type.label}
                        </span>
                        {/* Tooltip on hover */}
                        {type.description && (
                          <div className="absolute bottom-full left-1/2 transform -translate-x-1/2 mb-2 px-3 py-1.5 bg-gray-900 text-white text-xs rounded-lg opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none whitespace-nowrap z-10 shadow-lg">
                            {type.description}
                            <div className="absolute top-full left-1/2 transform -translate-x-1/2 -mt-1">
                              <div className="border-4 border-transparent border-t-gray-900"></div>
                            </div>
                          </div>
                        )}
                      </label>
                    ))}
                </div>
                {allRepositoryTypes.filter((type) =>
                  type.label.toLowerCase().includes(searchQuery.toLowerCase()) ||
                  type.description?.toLowerCase().includes(searchQuery.toLowerCase()) ||
                  type.value.toLowerCase().includes(searchQuery.toLowerCase())
                ).length === 0 && (
                  <div className="text-center py-8 text-gray-500">
                    <p className="text-sm">{t('create.noResults')} "{searchQuery}"</p>
                    <button
                      type="button"
                      onClick={() => setSearchQuery("")}
                      className="mt-2 text-sm text-blue-600 hover:text-blue-700 hover:underline"
                    >
                      {t('create.clearSearch')}
                    </button>
                  </div>
                )}
              </>
            ) : (
              // Category View - Compact grid layout
              <>
                {repositoryCategories
                  .filter((category) => category.name === selectedCategory)
                  .map((category) => (
                    <div key={category.name}>
                      <div className="flex items-center space-x-2 mb-3 pb-2 border-b border-gray-300">
                        <span className="text-xl">{category.icon}</span>
                        <h3 className="text-sm font-semibold text-gray-700">
                          {category.name}
                        </h3>
                        <span className="text-xs text-gray-500">
                          ({category.types.length} types)
                        </span>
                      </div>
                      <div className="grid grid-cols-3 gap-2.5">
                        {category.types.map((type) => (
                          <label
                            key={type.value}
                            className={`group relative flex items-center space-x-2.5 p-2.5 border rounded-lg cursor-pointer transition-all ${
                              formData.type === type.value
                                ? "border-blue-500 bg-blue-50 shadow-sm ring-2 ring-blue-200"
                                : "border-gray-300 bg-white hover:border-blue-300 hover:shadow-sm"
                            }`}
                            title={type.description}
                          >
                            <input
                              type="radio"
                              name="type"
                              value={type.value}
                              checked={formData.type === type.value}
                              onChange={handleChange}
                              className="text-blue-600 focus:ring-blue-500"
                            />
                            <span className="text-xl flex-shrink-0">{type.icon}</span>
                            <span className="text-sm font-medium text-gray-900 truncate">
                              {type.label}
                            </span>
                            {/* Tooltip on hover */}
                            {type.description && (
                              <div className="absolute bottom-full left-1/2 transform -translate-x-1/2 mb-2 px-3 py-1.5 bg-gray-900 text-white text-xs rounded-lg opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none whitespace-nowrap z-10 shadow-lg">
                                {type.description}
                                <div className="absolute top-full left-1/2 transform -translate-x-1/2 -mt-1">
                                  <div className="border-4 border-transparent border-t-gray-900"></div>
                                </div>
                              </div>
                            )}
                          </label>
                        ))}
                      </div>
                    </div>
                  ))}
              </>
            )}
          </div>

          {/* Selected Type Display */}
          {formData.type && (
            <div className="mt-2 flex items-center space-x-2 text-sm text-gray-600">
              <span className="font-medium">{t('create.selectedType')}:</span>
              <span className="inline-flex items-center space-x-1 px-2 py-1 bg-blue-100 text-blue-700 rounded-md">
                <span>
                  {allRepositoryTypes.find((t) => t.value === formData.type)?.icon}
                </span>
                <span className="font-medium">
                  {allRepositoryTypes.find((t) => t.value === formData.type)?.label}
                </span>
              </span>
            </div>
          )}
        </div>

        {/* Repository Type */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            {t('create.repositoryTypeLabel')} *
          </label>
          <div className="space-y-2">
            {repoTypes.map((type) => (
              <label
                key={type.value}
                className={`flex items-start space-x-3 p-4 border rounded-lg cursor-pointer transition ${formData.repositoryType === type.value
                  ? "border-blue-500 bg-blue-50"
                  : "border-gray-300 hover:border-gray-400"
                  }`}
              >
                <input
                  type="radio"
                  name="repositoryType"
                  value={type.value}
                  checked={formData.repositoryType === type.value}
                  onChange={handleChange}
                  className="mt-1 text-blue-600 focus:ring-blue-500"
                />
                <div>
                  <div className="font-medium text-gray-900">{type.label}</div>
                  <div className="text-sm text-gray-500">
                    {type.description}
                  </div>
                </div>
              </label>
            ))}
          </div>
        </div>

        {/* Remote URL (only for remote repositories) */}
        {formData.repositoryType === "remote" && (
          <div className="space-y-4 p-4 bg-gray-50 rounded-lg">
            <h4 className="font-medium text-gray-900">
              {t('create.remoteSettings')}
            </h4>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                {t('create.remoteUrlLabel')} *
              </label>
              <input
                type="url"
                name="remoteUrl"
                required={formData.repositoryType === "remote"}
                value={formData.remoteUrl}
                onChange={handleChange}
                placeholder={t('create.remoteUrlPlaceholder')}
                className={`w-full px-4 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent ${errors.remoteUrl ? 'border-red-500' : 'border-gray-300'
                  }`}
              />
              {errors.remoteUrl && (
                <p className="mt-1 text-xs text-red-600">{errors.remoteUrl}</p>
              )}
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  {t('create.usernameLabel')}
                </label>
                <input
                  type="text"
                  name="username"
                  value={formData.username}
                  onChange={handleChange}
                  className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  {t('create.passwordLabel')}
                </label>
                <input
                  type="password"
                  name="password"
                  value={formData.password}
                  onChange={handleChange}
                  className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
              </div>
            </div>
          </div>
        )}

        {/* Cloud Storage Settings (only for cloud repositories) */}
        {formData.repositoryType === "cloud" && (
          <div className="space-y-4 p-4 bg-sky-50 border border-sky-200 rounded-lg">
            <div className="flex items-center space-x-2">
              <h4 className="font-medium text-gray-900">
                {t('create.cloudSettings')}
              </h4>
              <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-sky-100 text-sky-800">
                ☁️ Cloud Provider
              </span>
            </div>

            {/* Cloud Provider Selection */}
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                {t('create.cloudProviderLabel')} *
              </label>
              <div className="grid grid-cols-2 gap-3">
                {cloudProviders.map((provider) => (
                  <label
                    key={provider.value}
                    className={`flex items-center space-x-2 p-3 border rounded-lg cursor-pointer transition ${formData.cloudProvider === provider.value
                      ? "border-sky-500 bg-sky-50"
                      : "border-gray-300 hover:border-gray-400"
                      }`}
                  >
                    <input
                      type="radio"
                      name="cloudProvider"
                      value={provider.value}
                      checked={formData.cloudProvider === provider.value}
                      onChange={handleChange}
                      className="text-sky-600 focus:ring-sky-500"
                    />
                    <span className="text-lg">{provider.icon}</span>
                    <div>
                      <div className="text-sm font-medium text-gray-900">
                        {provider.label}
                      </div>
                      <div className="text-xs text-gray-500">
                        {provider.description}
                      </div>
                    </div>
                  </label>
                ))}
              </div>
            </div>

            {/* S3 / S3-Compatible Configuration */}
            {(formData.cloudProvider === "s3" ||
              formData.cloudProvider === "s3-compatible") && (
                <div className="space-y-4 p-4 bg-white border border-gray-200 rounded-lg">
                  <h5 className="font-medium text-gray-900">
                    {formData.cloudProvider === "s3"
                      ? "Amazon S3"
                      : "S3-Compatible Storage"}{" "}
                    Configuration
                  </h5>

                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-2">
                        {t('create.cloudConfig.accessKeyIdLabel')} *
                      </label>
                      <input
                        type="text"
                        name="accessKeyId"
                        required={
                          formData.repositoryType === "cloud" &&
                          (formData.cloudProvider === "s3" ||
                            formData.cloudProvider === "s3-compatible")
                        }
                        value={formData.accessKeyId}
                        onChange={handleChange}
                        className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-sky-500 focus:border-transparent"
                      />
                    </div>

                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-2">
                        {t('create.cloudConfig.secretAccessKeyLabel')} *
                      </label>
                      <input
                        type="password"
                        name="secretAccessKey"
                        required={
                          formData.repositoryType === "cloud" &&
                          (formData.cloudProvider === "s3" ||
                            formData.cloudProvider === "s3-compatible")
                        }
                        value={formData.secretAccessKey}
                        onChange={handleChange}
                        className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-sky-500 focus:border-transparent"
                      />
                    </div>
                  </div>

                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-2">
                        {t('create.cloudConfig.bucketNameLabel')} *
                      </label>
                      <input
                        type="text"
                        name="bucketName"
                        required={
                          formData.repositoryType === "cloud" &&
                          (formData.cloudProvider === "s3" ||
                            formData.cloudProvider === "s3-compatible")
                        }
                        value={formData.bucketName}
                        onChange={handleChange}
                        placeholder={t('create.cloudConfig.bucketNamePlaceholder')}
                        className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-sky-500 focus:border-transparent"
                      />
                    </div>

                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-2">
                        {t('create.cloudConfig.regionLabel')}
                        {" "}
                        {formData.cloudProvider === "s3" ? "*" : "(optional)"}
                      </label>
                      <input
                        type="text"
                        name="region"
                        required={
                          formData.repositoryType === "cloud" &&
                          formData.cloudProvider === "s3"
                        }
                        value={formData.region}
                        onChange={handleChange}
                        placeholder={
                          formData.cloudProvider === "s3"
                            ? t('create.cloudConfig.regionPlaceholder')
                            : t('create.cloudConfig.regionPlaceholder') + " (optional)"
                        }
                        className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-sky-500 focus:border-transparent"
                      />
                    </div>
                  </div>

                  {formData.cloudProvider === "s3-compatible" && (
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-2">
                        {t('create.cloudConfig.endpointLabel')} *
                      </label>
                      <input
                        type="url"
                        name="endpoint"
                        required={
                          formData.repositoryType === "cloud" &&
                          formData.cloudProvider === "s3-compatible"
                        }
                        value={formData.endpoint}
                        onChange={handleChange}
                        placeholder={t('create.cloudConfig.endpointPlaceholder')}
                        className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-sky-500 focus:border-transparent"
                      />
                    </div>
                  )}
                </div>
              )}

            {/* GitHub Packages Configuration */}
            {formData.cloudProvider === "github" && (
              <div className="space-y-4 p-4 bg-white border border-gray-200 rounded-lg">
                <h5 className="font-medium text-gray-900">
                  GitHub Packages Configuration
                </h5>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    {t('create.cloudConfig.githubTokenLabel')} *
                  </label>
                  <input
                    type="password"
                    name="githubToken"
                    required={
                      formData.repositoryType === "cloud" &&
                      formData.cloudProvider === "github"
                    }
                    value={formData.githubToken}
                    onChange={handleChange}
                    placeholder={t('create.cloudConfig.githubTokenPlaceholder')}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-sky-500 focus:border-transparent"
                  />
                  <p className="mt-1 text-xs text-gray-500">
                    {t('create.cloudConfig.githubTokenHelp')}
                  </p>
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-2">
                      {t('create.cloudConfig.githubOrgLabel')} *
                    </label>
                    <input
                      type="text"
                      name="githubOrg"
                      required={
                        formData.repositoryType === "cloud" &&
                        formData.cloudProvider === "github"
                      }
                      value={formData.githubOrg}
                      onChange={handleChange}
                      placeholder={t('create.cloudConfig.githubOrgPlaceholder')}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-sky-500 focus:border-transparent"
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-2">
                      {t('create.cloudConfig.githubRepoLabel')}
                    </label>
                    <input
                      type="text"
                      name="githubRepo"
                      value={formData.githubRepo}
                      onChange={handleChange}
                      placeholder={t('create.cloudConfig.githubRepoPlaceholder')}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-sky-500 focus:border-transparent"
                    />
                  </div>
                </div>
              </div>
            )}

            {/* AWS ECR Configuration */}
            {formData.cloudProvider === "aws-ecr" && (
              <div className="space-y-4 p-4 bg-white border border-gray-200 rounded-lg">
                <h5 className="font-medium text-gray-900">
                  AWS ECR Configuration
                </h5>

                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-2">
                      {t('create.cloudConfig.accessKeyIdLabel')} *
                    </label>
                    <input
                      type="text"
                      name="accessKeyId"
                      required={
                        formData.repositoryType === "cloud" &&
                        formData.cloudProvider === "aws-ecr"
                      }
                      value={formData.accessKeyId}
                      onChange={handleChange}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-sky-500 focus:border-transparent"
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-2">
                      {t('create.cloudConfig.secretAccessKeyLabel')} *
                    </label>
                    <input
                      type="password"
                      name="secretAccessKey"
                      required={
                        formData.repositoryType === "cloud" &&
                        formData.cloudProvider === "aws-ecr"
                      }
                      value={formData.secretAccessKey}
                      onChange={handleChange}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-sky-500 focus:border-transparent"
                    />
                  </div>
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    {t('create.cloudConfig.regionLabel')} *
                  </label>
                  <input
                    type="text"
                    name="region"
                    required={
                      formData.repositoryType === "cloud" &&
                      formData.cloudProvider === "aws-ecr"
                    }
                    value={formData.region}
                    onChange={handleChange}
                    placeholder={t('create.cloudConfig.regionPlaceholder')}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-sky-500 focus:border-transparent"
                  />
                </div>
              </div>
            )}

            {/* Other cloud providers can be added here with similar patterns */}
            {(formData.cloudProvider === "azure" ||
              formData.cloudProvider === "gcp") && (
                <div className="p-4 bg-yellow-50 border border-yellow-200 rounded-lg">
                  <p className="text-sm text-yellow-800">
                    {t('create.cloudConfig.comingSoon', { 
                      provider: cloudProviders.find(
                        (p) => p.value === formData.cloudProvider
                      )?.label 
                    })}
                  </p>
                </div>
              )}
          </div>
        )}

        {/* Description */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            {t('create.descriptionLabel')}
          </label>
          <textarea
            name="description"
            value={formData.description}
            onChange={handleChange}
            rows={3}
            placeholder={t('create.descriptionPlaceholder')}
            className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
        </div>

        {/* Encryption Settings - Enterprise Mode */}
        {formData.repositoryType === "local" && (
          <div className="space-y-3 p-4 bg-blue-50 border-l-4 border-blue-400 rounded-lg">
            <div className="flex items-start space-x-3">
              <div className="flex-shrink-0">
                <svg className="w-5 h-5 text-blue-600 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
                </svg>
              </div>
              <div className="flex-1">
                <div className="flex items-center justify-between mb-2">
                  <h4 className="font-semibold text-gray-900 text-sm">{t('create.encryption.title')}</h4>
                  <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-semibold bg-green-100 text-green-800">
                    <svg className="w-3 h-3 mr-1" fill="currentColor" viewBox="0 0 20 20">
                      <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
                    </svg>
                    {t('create.encryption.enforced')}
                  </span>
                </div>
                <p className="text-sm text-blue-900 mb-3">
                  {t('create.encryption.description')}
                </p>
                <div className="bg-white bg-opacity-60 rounded-lg p-3 space-y-2">
                  <div className="flex items-center text-xs">
                    <svg className="w-4 h-4 text-green-600 mr-2 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                      <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
                    </svg>
                    <span className="text-gray-700" dangerouslySetInnerHTML={{ __html: t('create.encryption.algorithm') }} />
                  </div>
                  <div className="flex items-center text-xs">
                    <svg className="w-4 h-4 text-green-600 mr-2 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                      <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
                    </svg>
                    <span className="text-gray-700" dangerouslySetInnerHTML={{ __html: t('create.encryption.keyManagement') }} />
                  </div>
                  <div className="flex items-center text-xs">
                    <svg className="w-4 h-4 text-green-600 mr-2 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                      <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
                    </svg>
                    <span className="text-gray-700" dangerouslySetInnerHTML={{ __html: t('create.encryption.protection') }} />
                  </div>
                  <div className="flex items-center text-xs">
                    <svg className="w-4 h-4 text-green-600 mr-2 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                      <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
                    </svg>
                    <span className="text-gray-700" dangerouslySetInnerHTML={{ __html: t('create.encryption.audit') }} />
                  </div>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Options */}
        <div className="space-y-3">
          <label className="flex items-center space-x-2">
            <input
              type="checkbox"
              name="publicAccess"
              checked={formData.publicAccess}
              onChange={handleChange}
              className="rounded text-blue-600 focus:ring-blue-500"
            />
            <span className="text-sm text-gray-700">
              {t('create.publicAccessLabel')}
            </span>
          </label>

          <label className="flex items-center space-x-2">
            <input
              type="checkbox"
              name="enableIndexing"
              checked={formData.enableIndexing}
              onChange={handleChange}
              className="rounded text-blue-600 focus:ring-blue-500"
            />
            <span className="text-sm text-gray-700">
              {t('create.indexingLabel')}
            </span>
          </label>
        </div>

        {/* Info Box */}
        <div className="flex items-start space-x-2 p-4 bg-blue-50 border border-blue-200 rounded-lg">
          <Info className="w-5 h-5 text-blue-600 mt-0.5 flex-shrink-0" />
          <div className="text-sm text-blue-900">
            <p className="font-medium mb-1">{t('create.infoBox.title')}</p>
            <ul className="list-disc list-inside space-y-1 text-blue-800">
              <li>{t('create.infoBox.defaultQuota')}</li>
              <li>{t('create.infoBox.cleanupPolicies')}</li>
              <li>{t('create.infoBox.retentionPeriod')}</li>
              <li dangerouslySetInnerHTML={{ __html: t('create.infoBox.encryption') }} />
              {formData.repositoryType === "cloud" && (
                <li>
                  {t('create.infoBox.cloudStorage', {
                    provider: cloudProviders.find(
                      (p) => p.value === formData.cloudProvider
                    )?.label
                  })}
                  {formData.bucketName && ` (${formData.bucketName})`}
                  {formData.githubOrg && ` (${formData.githubOrg})`}
                </li>
              )}
            </ul>
          </div>
        </div>

        {/* Actions */}
        <div className="flex justify-end space-x-3 pt-4 border-t border-gray-200">
          <button
            type="button"
            onClick={onClose}
            disabled={loading}
            className="px-4 py-2 text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {t('create.cancelButton')}
          </button>
          <button
            type="submit"
            disabled={loading}
            className="px-4 py-2 text-white bg-blue-600 rounded-lg hover:bg-blue-700 transition flex items-center space-x-2 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <Database className="w-4 h-4" />
            <span>{loading ? t('create.creating') : t('create.createButton')}</span>
          </button>
        </div>
      </form>
    </Modal>
  );
};
