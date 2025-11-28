import React, { useState, useEffect } from 'react';
import { X } from 'lucide-react';
import { useArtifacts } from '../../../artifacts/hooks/useArtifacts';
import { useTranslation } from '../../../../hooks/useTranslation';

export const ScanModal = ({ onClose, onSubmit }) => {
  const { t } = useTranslation('security');
  const [formData, setFormData] = useState({
    artifact_id: '',
    scan_type: 'full',
    priority: 'normal',
    vulnerability_scan: true,
    malware_scan: true,
    license_scan: false,
    dependency_scan: true
  });

  const { artifacts, loading: artifactsLoading, fetchArtifacts } = useArtifacts();

  useEffect(() => {
    fetchArtifacts();
  }, []);

  const handleSubmit = (e) => {
    e.preventDefault();
    if (!formData.artifact_id) {
      alert(t('modal.selectArtifactError'));
      return;
    }
    onSubmit(formData);
  };

  const handleChange = (field, value) => {
    setFormData(prev => ({
      ...prev,
      [field]: value
    }));
  };

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl max-w-md w-full mx-4">
        <div className="flex items-center justify-between p-6 border-b">
          <h2 className="text-xl font-semibold text-gray-900">{t('modal.createTitle')}</h2>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600"
          >
            <X className="h-6 w-6" />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="p-6 space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              {t('modal.artifactLabel')} *
            </label>
            <select
              value={formData.artifact_id}
              onChange={(e) => handleChange('artifact_id', e.target.value)}
              className="w-full border border-gray-300 rounded-md px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
              required
            >
              <option value="">{t('modal.artifactPlaceholder')}</option>
              {artifacts.map(artifact => (
                <option key={artifact.id} value={artifact.id}>
                  {artifact.name} (v{artifact.version})
                </option>
              ))}
            </select>
            {artifactsLoading && (
              <p className="text-sm text-gray-500 mt-1">{t('modal.loadingArtifacts')}</p>
            )}
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              {t('modal.scanTypeLabel')}
            </label>
            <select
              value={formData.scan_type}
              onChange={(e) => handleChange('scan_type', e.target.value)}
              className="w-full border border-gray-300 rounded-md px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="full">{t('scanTypes.full')}</option>
              <option value="quick">{t('scanTypes.quick')}</option>
              <option value="custom">{t('scanTypes.custom')}</option>
            </select>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              {t('modal.priorityLabel')}
            </label>
            <select
              value={formData.priority}
              onChange={(e) => handleChange('priority', e.target.value)}
              className="w-full border border-gray-300 rounded-md px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="low">{t('priorities.low')}</option>
              <option value="normal">{t('priorities.normal')}</option>
              <option value="high">{t('priorities.high')}</option>
              <option value="critical">{t('priorities.critical')}</option>
            </select>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-3">
              {t('modal.scanOptionsLabel')}
            </label>
            <div className="space-y-2">
              <label className="flex items-center">
                <input
                  type="checkbox"
                  checked={formData.vulnerability_scan}
                  onChange={(e) => handleChange('vulnerability_scan', e.target.checked)}
                  className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                />
                <span className="ml-2 text-sm text-gray-700">{t('modal.vulnerabilityScan')}</span>
              </label>
              <label className="flex items-center">
                <input
                  type="checkbox"
                  checked={formData.malware_scan}
                  onChange={(e) => handleChange('malware_scan', e.target.checked)}
                  className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                />
                <span className="ml-2 text-sm text-gray-700">{t('modal.malwareScan')}</span>
              </label>
              <label className="flex items-center">
                <input
                  type="checkbox"
                  checked={formData.license_scan}
                  onChange={(e) => handleChange('license_scan', e.target.checked)}
                  className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                />
                <span className="ml-2 text-sm text-gray-700">{t('modal.licenseScan')}</span>
              </label>
              <label className="flex items-center">
                <input
                  type="checkbox"
                  checked={formData.dependency_scan}
                  onChange={(e) => handleChange('dependency_scan', e.target.checked)}
                  className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                />
                <span className="ml-2 text-sm text-gray-700">{t('modal.dependencyScan')}</span>
              </label>
            </div>
          </div>

          <div className="flex justify-end space-x-3 pt-4">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 border border-gray-300 rounded-md text-gray-700 hover:bg-gray-50"
            >
              {t('modal.cancel')}
            </button>
            <button
              type="submit"
              className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700"
            >
              {t('modal.startScan')}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};