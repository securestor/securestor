import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import LanguageDetector from 'i18next-browser-languagedetector';
import HttpBackend from 'i18next-http-backend';

// Import translation files
// English
import enCommon from './locales/en/common.json';
import enAuth from './locales/en/auth.json';
import enArtifacts from './locales/en/artifacts.json';
import enRepositories from './locales/en/repositories.json';
import enSecurity from './locales/en/security.json';
import enCompliance from './locales/en/compliance.json';
import enSettings from './locales/en/settings.json';
import enNotifications from './locales/en/notifications.json';
import enErrors from './locales/en/errors.json';
import enDashboard from './locales/en/dashboard.json';
import enAudit from './locales/en/audit.json';
import enCache from './locales/en/cache.json';
import enTenant from './locales/en/tenant.json';
import enReplication from './locales/en/replication.json';

// Danish
import daCommon from './locales/da/common.json';
import daAuth from './locales/da/auth.json';
import daArtifacts from './locales/da/artifacts.json';
import daRepositories from './locales/da/repositories.json';
import daSecurity from './locales/da/security.json';
import daCompliance from './locales/da/compliance.json';
import daSettings from './locales/da/settings.json';
import daNotifications from './locales/da/notifications.json';
import daErrors from './locales/da/errors.json';
import daDashboard from './locales/da/dashboard.json';
import daAudit from './locales/da/audit.json';
import daCache from './locales/da/cache.json';
import daTenant from './locales/da/tenant.json';
import daReplication from './locales/da/replication.json';

// Tamil
import taCommon from './locales/ta/common.json';
import taAuth from './locales/ta/auth.json';
import taArtifacts from './locales/ta/artifacts.json';
import taRepositories from './locales/ta/repositories.json';
import taSecurity from './locales/ta/security.json';
import taCompliance from './locales/ta/compliance.json';
import taSettings from './locales/ta/settings.json';
import taNotifications from './locales/ta/notifications.json';
import taErrors from './locales/ta/errors.json';
import taDashboard from './locales/ta/dashboard.json';
import taAudit from './locales/ta/audit.json';
import taCache from './locales/ta/cache.json';
import taTenant from './locales/ta/tenant.json';
import taReplication from './locales/ta/replication.json';

const resources = {
  en: {
    common: enCommon,
    auth: enAuth,
    artifacts: enArtifacts,
    repositories: enRepositories,
    security: enSecurity,
    compliance: enCompliance,
    settings: enSettings,
    notifications: enNotifications,
    errors: enErrors,
    dashboard: enDashboard,
    audit: enAudit,
    cache: enCache,
    tenant: enTenant,
    replication: enReplication
  },
  da: {
    common: daCommon,
    auth: daAuth,
    artifacts: daArtifacts,
    repositories: daRepositories,
    security: daSecurity,
    compliance: daCompliance,
    settings: daSettings,
    notifications: daNotifications,
    errors: daErrors,
    dashboard: daDashboard,
    audit: daAudit,
    cache: daCache,
    tenant: daTenant,
    replication: daReplication
  },
  ta: {
    common: taCommon,
    auth: taAuth,
    artifacts: taArtifacts,
    repositories: taRepositories,
    security: taSecurity,
    compliance: taCompliance,
    settings: taSettings,
    notifications: taNotifications,
    errors: taErrors,
    dashboard: taDashboard,
    audit: taAudit,
    cache: taCache,
    tenant: taTenant,
    replication: taReplication
  }
};

i18n
  // Load translation using http backend
  .use(HttpBackend)
  // Detect user language
  .use(LanguageDetector)
  // Pass the i18n instance to react-i18next
  .use(initReactI18next)
  // Initialize i18next
  .init({
    resources,
    fallbackLng: 'en',
    defaultNS: 'common',
    ns: ['common', 'auth', 'artifacts', 'repositories', 'security', 'compliance', 'settings', 'notifications', 'errors', 'dashboard', 'audit', 'cache', 'tenant', 'replication'],
    
    debug: process.env.NODE_ENV === 'development',
    
    interpolation: {
      escapeValue: false, // React already does escaping
    },
    
    detection: {
      // Order of language detection
      order: ['localStorage', 'navigator', 'htmlTag', 'path', 'subdomain'],
      
      // Keys or params to lookup language from
      lookupLocalStorage: 'securestor_language',
      lookupFromPathIndex: 0,
      lookupFromSubdomainIndex: 0,
      
      // Cache user language
      caches: ['localStorage'],
      excludeCacheFor: ['cimode'],
    },
    
    react: {
      useSuspense: true,
      bindI18n: 'languageChanged loaded',
      bindI18nStore: 'added removed',
      transEmptyNodeValue: '',
      transSupportBasicHtmlNodes: true,
      transKeepBasicHtmlNodesFor: ['br', 'strong', 'i', 'p'],
    },
    
    backend: {
      loadPath: '/locales/{{lng}}/{{ns}}.json',
    },
  });

export default i18n;
