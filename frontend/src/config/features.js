/**
 * Feature Flags Configuration
 * 
 * Controls which features are enabled in different editions:
 * - Community (Open Source): Basic features
 * - Enterprise: All features including advanced management
 * 
 * Set REACT_APP_EDITION=enterprise in .env for enterprise features
 */

const EDITION = process.env.REACT_APP_EDITION || 'community';

export const FEATURES = {
  // Core features (always enabled)
  REPOSITORIES: true,
  ARTIFACTS: true,
  SECURITY_SCANNING: true,
  OVERVIEW_DASHBOARD: true,
  USER_PROFILE: true,
  API_KEYS: true,
  MFA: true,
  AUDIT_LOGS: true,

  // Enterprise-only features
  REPLICATION: EDITION === 'enterprise',
  COMPLIANCE_MANAGEMENT: EDITION === 'enterprise',
  USER_MANAGEMENT: EDITION === 'enterprise',
  ROLE_MANAGEMENT: EDITION === 'enterprise',
  TENANT_MANAGEMENT: EDITION === 'enterprise',
  TENANT_SETTINGS: EDITION === 'enterprise',
  CACHE_MANAGEMENT: EDITION === 'enterprise',
};

export const isEnterpriseEdition = () => EDITION === 'enterprise';
export const isCommunityEdition = () => EDITION === 'community';
export const getEdition = () => EDITION;

// Helper function to check if a feature is enabled
export const isFeatureEnabled = (featureName) => {
  return FEATURES[featureName] === true;
};

export default FEATURES;
