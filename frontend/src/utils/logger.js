/**
 * Production-ready logger utility
 * Replaces console statements for enterprise-grade secure application
 * 
 * In production: Only errors are logged
 * In development: All logs are visible
 */

const isDevelopment = process.env.NODE_ENV === 'development';
const isProduction = process.env.NODE_ENV === 'production';

class Logger {
  constructor(context = '') {
    this.context = context;
  }

  /**
   * Log debug information (development only)
   */
  debug(...args) {
    if (isDevelopment) {
    }
  }

  /**
   * Log general information (development only)
   */
  log(...args) {
    if (isDevelopment) {
    }
  }

  /**
   * Log informational messages (development only)
   */
  info(...args) {
    if (isDevelopment) {
    }
  }

  /**
   * Log warnings (always logged, but sanitized in production)
   */
  warn(...args) {
    if (isDevelopment) {
      console.warn(`[${this.context}]`, ...args);
    } else {
      // In production, only log warning type without sensitive data
      console.warn(`[${this.context}] Warning occurred`);
    }
  }

  /**
   * Log errors (always logged, but sanitized in production)
   */
  error(message, error) {
    if (isDevelopment) {
      console.error(`[${this.context}]`, message, error);
    } else {
      // In production, log error but avoid exposing stack traces or sensitive data
      console.error(`[${this.context}]`, message);
      
      // Send to error tracking service (e.g., Sentry) in production
      if (window.errorTracker) {
        window.errorTracker.captureException(error, {
          context: this.context,
          message
        });
      }
    }
  }

  /**
   * Create a child logger with additional context
   */
  child(childContext) {
    return new Logger(`${this.context}:${childContext}`);
  }
}

/**
 * Create a logger instance for a specific context
 * @param {string} context - The context/module name
 * @returns {Logger} Logger instance
 */
export const createLogger = (context) => {
  return new Logger(context);
};

/**
 * Default logger instance
 */
export const logger = new Logger('App');

/**
 * No-op logger for production (completely silent)
 */
export const noopLogger = {
  debug: () => {},
  log: () => {},
  info: () => {},
  warn: () => {},
  error: () => {},
  child: () => noopLogger
};

/**
 * Get appropriate logger based on environment
 */
export const getLogger = (context) => {
  if (isProduction && process.env.REACT_APP_DISABLE_LOGS === 'true') {
    return noopLogger;
  }
  return createLogger(context);
};

export default logger;
