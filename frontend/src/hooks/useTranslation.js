import { useTranslation as useI18nTranslation } from 'react-i18next';

/**
 * Custom hook for using translations with type safety and convenience methods
 * 
 * @param {string} ns - The namespace to use (default: 'common')
 * @returns {object} Translation functions and utilities
 */
export const useTranslation = (ns = 'common') => {
  const { t, i18n } = useI18nTranslation(ns);

  /**
   * Translate with multiple namespace support
   * @param {string} key - Translation key in format 'namespace:key' or just 'key'
   * @param {object} options - Translation options (variables, count, etc.)
   */
  const translate = (key, options = {}) => {
    // If key contains namespace, use it directly
    if (key.includes(':')) {
      return t(key, options);
    }
    // Otherwise use the current namespace
    return t(`${ns}:${key}`, options);
  };

  /**
   * Change the current language
   * @param {string} lng - Language code (en, da, ta)
   */
  const changeLanguage = (lng) => {
    return i18n.changeLanguage(lng);
  };

  /**
   * Get current language
   */
  const currentLanguage = i18n.language;

  /**
   * Check if a translation exists
   * @param {string} key - Translation key
   */
  const exists = (key) => {
    return i18n.exists(key, { ns });
  };

  return {
    t: translate,
    i18n,
    changeLanguage,
    currentLanguage,
    exists,
    // Original function for direct access
    tRaw: t
  };
};

/**
 * Hook to get available languages
 */
export const useLanguages = () => {
  return [
    { 
      code: 'en', 
      name: 'English', 
      nativeName: 'English',
      flag: '🇬🇧'
    },
    { 
      code: 'da', 
      name: 'Danish', 
      nativeName: 'Dansk',
      flag: '🇩🇰'
    },
    { 
      code: 'ta', 
      name: 'Tamil', 
      nativeName: 'தமிழ்',
      flag: '🇮🇳'
    }
  ];
};

export default useTranslation;
