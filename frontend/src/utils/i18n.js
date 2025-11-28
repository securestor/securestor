import i18n from 'i18next';

/**
 * Format date according to current locale
 */
export const formatDate = (date, options = {}) => {
  const defaultOptions = {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    ...options
  };
  
  return new Intl.DateTimeFormat(i18n.language, defaultOptions).format(new Date(date));
};

/**
 * Format date and time according to current locale
 */
export const formatDateTime = (date, options = {}) => {
  const defaultOptions = {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    ...options
  };
  
  return new Intl.DateTimeFormat(i18n.language, defaultOptions).format(new Date(date));
};

/**
 * Format relative time (e.g., "2 days ago")
 */
export const formatRelativeTime = (date) => {
  const rtf = new Intl.RelativeTimeFormat(i18n.language, { numeric: 'auto' });
  const now = new Date();
  const past = new Date(date);
  const diffInSeconds = Math.floor((past - now) / 1000);
  
  if (Math.abs(diffInSeconds) < 60) {
    return i18n.t('common:time.justNow');
  } else if (Math.abs(diffInSeconds) < 3600) {
    const minutes = Math.floor(diffInSeconds / 60);
    return rtf.format(minutes, 'minute');
  } else if (Math.abs(diffInSeconds) < 86400) {
    const hours = Math.floor(diffInSeconds / 3600);
    return rtf.format(hours, 'hour');
  } else {
    const days = Math.floor(diffInSeconds / 86400);
    return rtf.format(days, 'day');
  }
};

/**
 * Format number according to current locale
 */
export const formatNumber = (number, options = {}) => {
  return new Intl.NumberFormat(i18n.language, options).format(number);
};

/**
 * Format currency according to current locale
 */
export const formatCurrency = (amount, currency = 'USD', options = {}) => {
  const defaultOptions = {
    style: 'currency',
    currency,
    ...options
  };
  
  return new Intl.NumberFormat(i18n.language, defaultOptions).format(amount);
};

/**
 * Format file size with appropriate unit
 */
export const formatFileSize = (bytes) => {
  if (bytes === 0) return `0 ${i18n.t('common:units.bytes')}`;
  
  const sizes = ['bytes', 'kb', 'mb', 'gb', 'tb', 'pb'];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  const value = bytes / Math.pow(1024, i);
  
  return `${formatNumber(value, { maximumFractionDigits: 2 })} ${i18n.t(`common:units.${sizes[i]}`)}`;
};

/**
 * Format percentage
 */
export const formatPercentage = (value, options = {}) => {
  const defaultOptions = {
    style: 'percent',
    minimumFractionDigits: 0,
    maximumFractionDigits: 2,
    ...options
  };
  
  return new Intl.NumberFormat(i18n.language, defaultOptions).format(value / 100);
};

/**
 * Get available languages
 */
export const getAvailableLanguages = () => {
  return [
    { code: 'en', name: 'English', nativeName: 'English' },
    { code: 'da', name: 'Danish', nativeName: 'Dansk' },
    { code: 'ta', name: 'Tamil', nativeName: 'தமிழ்' }
  ];
};

/**
 * Change language
 */
export const changeLanguage = (lng) => {
  return i18n.changeLanguage(lng);
};

/**
 * Get current language
 */
export const getCurrentLanguage = () => {
  return i18n.language;
};

/**
 * Check if translation key exists
 */
export const translationExists = (key, ns = 'common') => {
  return i18n.exists(key, { ns });
};
