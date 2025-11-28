import React, { useEffect, Suspense } from 'react';
import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import { AuthProvider } from './context/AuthContext';
import { TenantProvider } from './context/TenantContext';
import { DashboardProvider } from './context/DashboardContext';
import { ToastProvider } from './context/ToastContext';
import { DashboardLayout } from './layouts/DashboardLayout';
import ProtectedRoute from './components/ProtectedRoute';
import InviteAcceptance from './components/InviteAcceptance';
import TenantSwitcher from './components/TenantSwitcher';
import TenantGuard from './components/TenantGuard';
import './i18n'; // Initialize i18n

// Loading component for Suspense
const LoadingFallback = () => (
  <div className="flex items-center justify-center min-h-screen bg-gray-50 dark:bg-gray-900">
    <div className="text-center">
      <div className="inline-block animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
      <p className="mt-4 text-gray-600 dark:text-gray-400">Loading...</p>
    </div>
  </div>
);

export default function App() {
  // Close dropdowns when clicking outside
  useEffect(() => {
    const handleClickOutside = (event) => {
      if (!event.target.closest('.dropdown-container')) {
        // Close any open dropdowns
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  return (
    <Suspense fallback={<LoadingFallback />}>
      <Router>
        <TenantGuard>
          <AuthProvider>
            <TenantProvider>
              <ToastProvider>
                <Routes>
                  {/* Public route for invitation acceptance */}
                  <Route path="/invite/accept" element={<InviteAcceptance />} />
                  
                  {/* Protected routes */}
                  <Route 
                    path="/*" 
                    element={
                      <DashboardProvider>
                        <ProtectedRoute>
                          <DashboardLayout />
                        </ProtectedRoute>
                      </DashboardProvider>
                    } 
                  />
                </Routes>
                
                {/* Tenant Switcher - Only visible in dev mode */}
                <TenantSwitcher />
              </ToastProvider>
            </TenantProvider>
          </AuthProvider>
        </TenantGuard>
      </Router>
    </Suspense>
  );
}

