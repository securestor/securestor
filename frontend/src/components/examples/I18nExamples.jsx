/**
 * Example: Dashboard Header Component with i18n
 * 
 * This example demonstrates how to use i18n in a React component
 */

import React from 'react';
import { useTranslation } from '../hooks/useTranslation';
import LanguageSwitcher from './LanguageSwitcher';
import { formatDate, formatFileSize } from '../utils/i18n';

function DashboardHeaderExample() {
  // Import translations from 'common' namespace
  const { t, currentLanguage } = useTranslation('common');
  
  // Sample data
  const lastBackup = new Date('2025-01-15T10:30:00');
  const storageUsed = 5368709120; // 5 GB in bytes
  const artifactCount = 42;

  return (
    <header className="bg-white shadow-sm dark:bg-gray-800">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
        <div className="flex items-center justify-between">
          {/* Left section */}
          <div>
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
              {t('navigation.dashboard')}
            </h1>
            <p className="text-sm text-gray-600 dark:text-gray-400">
              {t('messages.welcome')}
            </p>
          </div>

          {/* Right section */}
          <div className="flex items-center space-x-4">
            {/* Storage info */}
            <div className="text-right">
              <p className="text-sm font-medium text-gray-900 dark:text-white">
                {t('common.size')}: {formatFileSize(storageUsed)}
              </p>
              <p className="text-xs text-gray-500 dark:text-gray-400">
                {t('artifacts:list.count', { count: artifactCount })}
              </p>
            </div>

            {/* Last backup time */}
            <div className="text-right">
              <p className="text-xs text-gray-500 dark:text-gray-400">
                {t('common.updatedAt')}
              </p>
              <p className="text-sm font-medium text-gray-900 dark:text-white">
                {formatDate(lastBackup)}
              </p>
            </div>

            {/* Language switcher */}
            <LanguageSwitcher />
          </div>
        </div>

        {/* Action buttons */}
        <div className="mt-4 flex space-x-2">
          <button className="btn-primary">
            {t('buttons.refresh')}
          </button>
          <button className="btn-secondary">
            {t('buttons.export')}
          </button>
          <button className="btn-secondary">
            {t('navigation.settings')}
          </button>
        </div>

        {/* Info message */}
        <div className="mt-4 p-3 bg-blue-50 dark:bg-blue-900 rounded-md">
          <p className="text-sm text-blue-800 dark:text-blue-200">
            {t('status.info')}: {t('messages.loadingData')}
          </p>
        </div>

        {/* Current language indicator (for demo) */}
        <div className="mt-2 text-xs text-gray-500 dark:text-gray-400">
          Current language: {currentLanguage}
        </div>
      </div>
    </header>
  );
}

/**
 * Example: Login Form with i18n
 */
function LoginFormExample() {
  const { t } = useTranslation('auth');
  const [email, setEmail] = React.useState('');
  const [password, setPassword] = React.useState('');

  const handleSubmit = (e) => {
    e.preventDefault();
    // Login logic here
  };

  return (
    <div className="max-w-md mx-auto mt-8 p-6 bg-white rounded-lg shadow-md dark:bg-gray-800">
      <h2 className="text-2xl font-bold text-gray-900 dark:text-white mb-2">
        {t('login.title')}
      </h2>
      <p className="text-gray-600 dark:text-gray-400 mb-6">
        {t('login.subtitle')}
      </p>

      <form onSubmit={handleSubmit}>
        {/* Email field */}
        <div className="mb-4">
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            {t('login.emailLabel')}
          </label>
          <input
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder={t('login.emailPlaceholder')}
            className="w-full px-3 py-2 border border-gray-300 rounded-md dark:bg-gray-700 dark:border-gray-600"
            required
          />
        </div>

        {/* Password field */}
        <div className="mb-4">
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            {t('login.passwordLabel')}
          </label>
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder={t('login.passwordPlaceholder')}
            className="w-full px-3 py-2 border border-gray-300 rounded-md dark:bg-gray-700 dark:border-gray-600"
            required
          />
        </div>

        {/* Remember me */}
        <div className="mb-4 flex items-center">
          <input
            type="checkbox"
            id="remember"
            className="mr-2"
          />
          <label htmlFor="remember" className="text-sm text-gray-700 dark:text-gray-300">
            {t('login.rememberMe')}
          </label>
        </div>

        {/* Submit button */}
        <button
          type="submit"
          className="w-full py-2 px-4 bg-blue-600 hover:bg-blue-700 text-white font-medium rounded-md"
        >
          {t('login.loginButton')}
        </button>

        {/* Forgot password */}
        <div className="mt-4 text-center">
          <a href="/forgot-password" className="text-sm text-blue-600 hover:text-blue-800 dark:text-blue-400">
            {t('login.forgotPassword')}
          </a>
        </div>

        {/* Sign up link */}
        <div className="mt-4 text-center text-sm text-gray-600 dark:text-gray-400">
          {t('login.noAccount')}{' '}
          <a href="/register" className="text-blue-600 hover:text-blue-800 dark:text-blue-400">
            {t('login.signUpLink')}
          </a>
        </div>
      </form>
    </div>
  );
}

/**
 * Example: Artifact List with Pluralization
 */
function ArtifactListExample() {
  const { t } = useTranslation('artifacts');
  const artifacts = [
    { name: 'nginx', version: '1.21.0', type: 'docker' },
    { name: 'react', version: '18.2.0', type: 'npm' },
  ];

  return (
    <div className="p-6">
      <div className="mb-4 flex items-center justify-between">
        <h2 className="text-xl font-bold">
          {t('list.title')}
        </h2>
        <span className="text-sm text-gray-600">
          {t('list.count', { count: artifacts.length })}
        </span>
      </div>

      <div className="space-y-2">
        {artifacts.map((artifact, index) => (
          <div key={index} className="p-4 bg-white rounded-lg shadow dark:bg-gray-800">
            <div className="flex items-center justify-between">
              <div>
                <h3 className="font-medium">{artifact.name}</h3>
                <p className="text-sm text-gray-600 dark:text-gray-400">
                  {t('metadata.version')}: {artifact.version}
                </p>
                <p className="text-sm text-gray-600 dark:text-gray-400">
                  {t('metadata.type')}: {t(`types.${artifact.type}`)}
                </p>
              </div>
              <div className="space-x-2">
                <button className="btn-sm btn-primary">
                  {t('actions.download')}
                </button>
                <button className="btn-sm btn-secondary">
                  {t('actions.scan')}
                </button>
              </div>
            </div>
          </div>
        ))}
      </div>

      {artifacts.length === 0 && (
        <div className="text-center py-8 text-gray-500">
          {t('list.noArtifacts')}
        </div>
      )}
    </div>
  );
}

/**
 * Example: Error Handling with i18n
 */
function ErrorHandlingExample() {
  const { t } = useTranslation('errors');
  const [error, setError] = React.useState(null);

  const validateForm = () => {
    const errors = [];

    // Check required field
    if (!email) {
      errors.push(t('validation.required'));
    }

    // Check email format
    if (email && !email.includes('@')) {
      errors.push(t('validation.invalidEmail'));
    }

    // Check password length
    if (password && password.length < 8) {
      errors.push(t('validation.minLength', { min: 8 }));
    }

    if (errors.length > 0) {
      setError(errors.join(', '));
      return false;
    }

    return true;
  };

  return (
    <div>
      {error && (
        <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-md dark:bg-red-900 dark:border-red-800">
          <p className="text-sm text-red-800 dark:text-red-200">
            {t('general.error')}: {error}
          </p>
        </div>
      )}
    </div>
  );
}

export {
  DashboardHeaderExample,
  LoginFormExample,
  ArtifactListExample,
  ErrorHandlingExample
};
