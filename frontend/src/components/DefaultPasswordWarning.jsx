import React, { useState, useEffect } from 'react';
import { AlertTriangle, X } from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import { ChangePasswordDialog } from './ChangePasswordDialog';

/**
 * DefaultPasswordWarning - Shows a prominent warning banner when the user is using default credentials
 * Appears at the top of the dashboard until password is changed
 */
export const DefaultPasswordWarning = () => {
  const [isVisible, setIsVisible] = useState(false);
  const [isDismissed, setIsDismissed] = useState(
    sessionStorage.getItem('default-password-warning-dismissed') === 'true'
  );
  const [showPasswordDialog, setShowPasswordDialog] = useState(false);
  const { user, getApiUrl } = useAuth();

  useEffect(() => {
    // Check if user is using default credentials
    const checkDefaultPassword = async () => {
      try {
        const token = localStorage.getItem('auth_token');
        if (!token) {
          return;
        }

        const url = getApiUrl('/api/v1/auth/check-default-password');
        
        const response = await fetch(url, {
          headers: {
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json'
          }
        });
        
        if (response.ok) {
          const data = await response.json();
          setIsVisible(data.is_default_password && !isDismissed);
        }
      } catch (error) {
        console.error('Failed to check default password status:', error);
      }
    };

    if (user) {
      checkDefaultPassword();
    }
  }, [user, isDismissed]);

  const handleDismiss = () => {
    setIsDismissed(true);
    setIsVisible(false);
    // Store dismissal in localStorage (will reappear on page refresh until password changed)
    sessionStorage.setItem('default-password-warning-dismissed', 'true');
  };

  const handlePasswordChange = () => {
    setShowPasswordDialog(true);
  };

  const handlePasswordChangeSuccess = () => {
    setIsVisible(false);
    setIsDismissed(true);
    setShowPasswordDialog(false);
    // Refresh the page to update UI state
    setTimeout(() => window.location.reload(), 1000);
  };

  if (!isVisible) {
    return null;
  }

  return (
    <>
      <div className="bg-red-50 border-l-4 border-red-500 p-4 mb-4 rounded-r-lg shadow-md">
        <div className="flex items-start">
          <div className="flex-shrink-0">
            <AlertTriangle className="h-6 w-6 text-red-600 animate-pulse" />
          </div>
          <div className="ml-3 flex-1">
            <h3 className="text-sm font-bold text-red-800">
              ⚠️ SECURITY WARNING: Default Password Detected
            </h3>
            <div className="mt-2 text-sm text-red-700">
              <p className="mb-2">
                You are currently using the default password (<code className="bg-red-100 px-1 rounded">admin123</code>). 
                This is a critical security risk!
              </p>
              <p className="mb-3">
                <strong>Please change your password immediately to secure your account.</strong>
              </p>
              <div className="flex gap-2">
                <button
                  onClick={handlePasswordChange}
                  className="inline-flex items-center px-4 py-2 bg-red-600 text-white text-sm font-medium rounded-md hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500"
                >
                  Change Password Now
                </button>
                <button
                  onClick={handleDismiss}
                  className="text-red-600 hover:text-red-800 text-sm font-medium"
                >
                  Dismiss (Temporarily)
                </button>
              </div>
            </div>
          </div>
          <div className="ml-auto pl-3">
            <button
              onClick={handleDismiss}
              className="inline-flex text-red-400 hover:text-red-600 focus:outline-none"
            >
              <X className="h-5 w-5" />
            </button>
          </div>
        </div>
      </div>
      
      {/* Change Password Dialog */}
      <ChangePasswordDialog
        isOpen={showPasswordDialog}
        onClose={() => setShowPasswordDialog(false)}
        onSuccess={handlePasswordChangeSuccess}
      />
    </>
  );
};

export default DefaultPasswordWarning;
