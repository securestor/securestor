import React from 'react';
import { useDashboard } from '../../context/DashboardContext';
import { useTenant } from '../../context/TenantContext';
import { useTranslation } from '../../hooks/useTranslation';
import { DashboardOverview } from '../../features/dashboard/components/DashboardOverview';
import { SearchBar } from '../../features/repositories/components/SearchBar';
import { RepositoryTable } from '../../features/repositories/components/RepositoryTable';
import { RepositoryArtifactsView } from '../../features/repositories/components/RepositoryArtifactsView';
import { ArtifactsManagement } from '../../features/artifacts';
import { SecurityScanPage } from '../../features/security/components/SecurityScanPage';
import APIKeyManagement from '../../components/APIKeyManagement';
import UserProfile from '../../components/UserProfile';
import MFASettingsDashboard from '../../components/MFASettingsDashboard';
import AuditLogs from '../../components/AuditLogs';
import ReplicationSettings from '../../features/settings/components/ReplicationSettings';
import FEATURES from '../../config/features';

// Lazy load enterprise components only if needed
const ComplianceManagement = FEATURES.COMPLIANCE_MANAGEMENT 
  ? React.lazy(() => import('../../features/compliance/components/ComplianceManagement'))
  : null;
const UserManagementDashboard = FEATURES.USER_MANAGEMENT
  ? React.lazy(() => import('../../components/UserManagementDashboard'))
  : null;
const RoleManagement = FEATURES.ROLE_MANAGEMENT
  ? React.lazy(() => import('../../components/RoleManagement'))
  : null;
const TenantSettingsEnterprise = FEATURES.TENANT_SETTINGS
  ? React.lazy(() => import('../../components/TenantSettingsEnterprise'))
  : null;
const TenantSettingsVerification = FEATURES.TENANT_SETTINGS
  ? React.lazy(() => import('../../components/TenantSettingsVerification'))
  : null;
const TenantManagementDashboard = FEATURES.TENANT_MANAGEMENT
  ? React.lazy(() => import('../../components/TenantManagementDashboard'))
  : null;
const CacheManagement = FEATURES.CACHE_MANAGEMENT
  ? React.lazy(() => import('../../features/cache/CacheManagement'))
  : null;

export const MainContent = () => {
  const { t } = useTranslation('repositories');
  const { activeTab, selectedRepo, setSelectedRepo } = useDashboard();
  const { tenantId, loading: tenantLoading } = useTenant();

  // Show overview dashboard when overview tab is active
  if (activeTab === 'overview') {
    return (
      <main className="flex-1 p-6">
        <DashboardOverview />
      </main>
    );
  }

  // Show repository artifacts view when a repository is selected (on repositories tab)
  if (activeTab === 'repositories' && selectedRepo) {
    return (
      <main className="flex-1 p-6">
        <RepositoryArtifactsView 
          repository={selectedRepo} 
          onBack={() => setSelectedRepo(null)} 
        />
      </main>
    );
  }

  // Show artifacts screen when artifacts tab is active
  if (activeTab === 'artifacts') {
    return (
      <main className="flex-1">
        <ArtifactsManagement />
      </main>
    );
  }

  // Show security scan page when security tab is active
  if (activeTab === 'security') {
    return (
      <main className="flex-1">
        <SecurityScanPage />
      </main>
    );
  }

  // Show cache management when cache tab is active (Enterprise only)
  if (activeTab === 'cache') {
    if (!FEATURES.CACHE_MANAGEMENT) {
      return (
        <main className="flex-1 p-6">
          <div className="text-center py-12">
            <h2 className="text-2xl font-bold text-gray-900 mb-4">Enterprise Feature</h2>
            <p className="text-gray-600">Cache management is available in the Enterprise edition.</p>
          </div>
        </main>
      );
    }
    return (
      <main className="flex-1">
        <React.Suspense fallback={<div className="flex items-center justify-center h-full"><div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div></div>}>
          <CacheManagement />
        </React.Suspense>
      </main>
    );
  }

  // Show compliance management when compliance tab is active (Enterprise only)
  if (activeTab === 'compliance') {
    if (!FEATURES.COMPLIANCE_MANAGEMENT) {
      return (
        <main className="flex-1 p-6">
          <div className="text-center py-12">
            <h2 className="text-2xl font-bold text-gray-900 mb-4">Enterprise Feature</h2>
            <p className="text-gray-600">Compliance management is available in the Enterprise edition.</p>
          </div>
        </main>
      );
    }
    return (
      <main className="flex-1">
        <React.Suspense fallback={<div className="flex items-center justify-center h-full"><div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div></div>}>
          <ComplianceManagement />
        </React.Suspense>
      </main>
    );
  }

  // Show user management when users tab is active (Enterprise only)
  if (activeTab === 'users') {
    if (!FEATURES.USER_MANAGEMENT) {
      return (
        <main className="flex-1 p-6">
          <div className="text-center py-12">
            <h2 className="text-2xl font-bold text-gray-900 mb-4">Enterprise Feature</h2>
            <p className="text-gray-600">User management is available in the Enterprise edition.</p>
          </div>
        </main>
      );
    }
    return (
      <main className="flex-1">
        <React.Suspense fallback={<div className="flex items-center justify-center h-full"><div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div></div>}>
          <UserManagementDashboard />
        </React.Suspense>
      </main>
    );
  }

  // Show role management when roles tab is active (Enterprise only)
  if (activeTab === 'roles') {
    if (!FEATURES.ROLE_MANAGEMENT) {
      return (
        <main className="flex-1 p-6">
          <div className="text-center py-12">
            <h2 className="text-2xl font-bold text-gray-900 mb-4">Enterprise Feature</h2>
            <p className="text-gray-600">Role management is available in the Enterprise edition.</p>
          </div>
        </main>
      );
    }
    return (
      <main className="flex-1">
        <React.Suspense fallback={<div className="flex items-center justify-center h-full"><div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div></div>}>
          <RoleManagement />
        </React.Suspense>
      </main>
    );
  }

  // Show API key management when api-keys tab is active
  // Show API key management when api-keys tab is active
  if (activeTab === 'api-keys') {
    return (
      <main className="flex-1">
        <APIKeyManagement />
      </main>
    );
  }

  // Show profile when profile tab is active
  if (activeTab === 'profile') {
    return (
      <main className="flex-1">
        <UserProfile />
      </main>
    );
  }

  // Show tenant management when tenant-management tab is active (Enterprise only)
  if (activeTab === 'tenant-management') {
    if (!FEATURES.TENANT_MANAGEMENT) {
      return (
        <main className="flex-1 p-6">
          <div className="text-center py-12">
            <h2 className="text-2xl font-bold text-gray-900 mb-4">Enterprise Feature</h2>
            <p className="text-gray-600">Tenant management is available in the Enterprise edition.</p>
          </div>
        </main>
      );
    }
    return (
      <main className="flex-1">
        <React.Suspense fallback={<div className="flex items-center justify-center h-full"><div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div></div>}>
          <TenantManagementDashboard />
        </React.Suspense>
      </main>
    );
  }

  // Show tenant settings when tenant-settings tab is active (Enterprise Configuration)
  if (activeTab === 'tenant-settings') {
    if (!FEATURES.TENANT_SETTINGS) {
      return (
        <main className="flex-1 p-6">
          <div className="text-center py-12">
            <h2 className="text-2xl font-bold text-gray-900 mb-4">Enterprise Feature</h2>
            <p className="text-gray-600">Advanced tenant settings are available in the Enterprise edition.</p>
          </div>
        </main>
      );
    }
    return (
      <main className="flex-1">
        {tenantLoading ? (
          <div className="flex items-center justify-center h-full">
            <div className="text-center">
              <div className="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
              <p className="mt-4 text-gray-600">Loading tenant settings...</p>
            </div>
          </div>
        ) : tenantId ? (
          <React.Suspense fallback={<div className="flex items-center justify-center h-full"><div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div></div>}>
            <TenantSettingsEnterprise tenantId={tenantId} />
          </React.Suspense>
        ) : (
          <div className="flex items-center justify-center h-full">
            <div className="text-center">
              <p className="text-red-600">Unable to load tenant settings. No tenant ID found.</p>
            </div>
          </div>
        )}
      </main>
    );
  }

  // Show tenant settings verification page when tenant-verification tab is active (Enterprise only)
  if (activeTab === 'tenant-verification') {
    if (!FEATURES.TENANT_SETTINGS) {
      return (
        <main className="flex-1 p-6">
          <div className="text-center py-12">
            <h2 className="text-2xl font-bold text-gray-900 mb-4">Enterprise Feature</h2>
            <p className="text-gray-600">Tenant settings verification is available in the Enterprise edition.</p>
          </div>
        </main>
      );
    }
    return (
      <main className="flex-1">
        <React.Suspense fallback={<div className="flex items-center justify-center h-full"><div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div></div>}>
          <TenantSettingsVerification />
        </React.Suspense>
      </main>
    );
  }

  // Show MFA settings when mfa-settings tab is active
  if (activeTab === 'mfa-settings') {
    return (
      <main className="flex-1">
        <MFASettingsDashboard />
      </main>
    );
  }

  // Show Replication settings when replication-settings tab is active (Enterprise only)
  if (activeTab === 'replication-settings') {
    if (!FEATURES.REPLICATION) {
      return (
        <main className="flex-1 p-6">
          <div className="text-center py-12">
            <h2 className="text-2xl font-bold text-gray-900 mb-4">Enterprise Feature</h2>
            <p className="text-gray-600">Replication settings are available in the Enterprise edition.</p>
          </div>
        </main>
      );
    }
    return (
      <main className="flex-1">
        <ReplicationSettings />
      </main>
    );
  }

  // Show audit logs when logs tab is active
  if (activeTab === 'logs') {
    return (
      <main className="flex-1">
        <AuditLogs />
      </main>
    );
  }

  // Default dashboard view (repositories tab)
  return (
    <main className="flex-1 p-6">
      <div className="mb-6">
        <h1 className="text-3xl font-bold text-gray-900">{t('title')}</h1>
        <p className="text-gray-600 mt-1">{t('subtitle')}</p>
      </div>
      <SearchBar />
      <RepositoryTable />
    </main>
  );
};
