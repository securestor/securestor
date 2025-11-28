import { useMemo } from 'react';
import FEATURES, { isFeatureEnabled, isEnterpriseEdition, isCommunityEdition, getEdition } from '../config/features';

/**
 * Hook to access feature flags throughout the application
 * 
 * Usage:
 * const { isEnabled, edition, features } = useFeatureFlags();
 * 
 * if (isEnabled('COMPLIANCE_MANAGEMENT')) {
 *   // Show compliance features
 * }
 */
export const useFeatureFlags = () => {
  const features = useMemo(() => FEATURES, []);
  
  return {
    features,
    isEnabled: isFeatureEnabled,
    isEnterprise: isEnterpriseEdition(),
    isCommunity: isCommunityEdition(),
    edition: getEdition(),
  };
};

export default useFeatureFlags;
