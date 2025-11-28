import React, { createContext, useContext, useState, useEffect } from 'react';
import { getTenantApiUrl, getCurrentTenant, getTenantHeaders } from '../utils/tenant';

const AuthContext = createContext();

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};

/**
 * Enterprise SaaS Multi-Tenant Auth Provider
 * Handles authentication with tenant context awareness
 * Uses dynamic API URLs based on current subdomain
 */
export const AuthProvider = ({ children }) => {
  const [user, setUser] = useState(null);
  const [token, setToken] = useState(localStorage.getItem('auth_token'));
  const [loading, setLoading] = useState(true);

  /**
   * Get tenant-aware API URL
   * Dynamically constructs API endpoint based on current subdomain
   */
  const getApiUrl = (endpoint) => {
    const baseUrl = getTenantApiUrl();
    return `${baseUrl}${endpoint}`;
  };

  /**
   * Get headers with both auth and tenant context
   */
  const getRequestHeaders = (includeAuth = true) => {
    const headers = {
      'Content-Type': 'application/json',
      ...getTenantHeaders(),
    };

    if (includeAuth && token) {
      headers['Authorization'] = `Bearer ${token}`;
    }

    return headers;
  };

  // Initialize auth state from localStorage
  useEffect(() => {
    const initializeAuth = async () => {
      const storedToken = localStorage.getItem('auth_token');
      const storedUser = localStorage.getItem('auth_user');
      
      if (storedToken && storedUser) {
        setToken(storedToken);
        setUser(JSON.parse(storedUser));
        
        // Verify token is still valid
        try {
          const response = await fetch(getApiUrl('/api/auth/me'), {
            headers: getRequestHeaders(true),
          });
          
          if (!response.ok) {
            // Token is invalid, clear auth
            logout();
          } else {
            const userData = await response.json();
            setUser(userData);
            localStorage.setItem('auth_user', JSON.stringify(userData));
          }
        } catch (error) {
          console.error('[Auth] Verification failed:', error);
          logout();
        }
      }
      
      setLoading(false);
    };

    initializeAuth();
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  const login = async (username, password) => {
    try {
      const currentTenant = getCurrentTenant();

      const response = await fetch(getApiUrl('/api/auth/login'), {
        method: 'POST',
        headers: getRequestHeaders(false),
        body: JSON.stringify({ username, password }),
      });

      if (!response.ok) {
        const errorText = await response.text();
        throw new Error(errorText || 'Login failed');
      }

      const data = await response.json();
      
      setToken(data.token);
      setUser(data.user);
      
      // Store in localStorage
      localStorage.setItem('auth_token', data.token);
      localStorage.setItem('auth_user', JSON.stringify(data.user));
      
      
      return { success: true, user: data.user };
    } catch (error) {
      console.error('[Auth] Login error:', error);
      return { success: false, error: error.message };
    }
  };

  const logout = () => {
    setToken(null);
    setUser(null);
    localStorage.removeItem('auth_token');
    localStorage.removeItem('auth_user');
  };

  const changePassword = async (currentPassword, newPassword) => {
    try {
      const response = await fetch(getApiUrl('/api/auth/change-password'), {
        method: 'POST',
        headers: getRequestHeaders(true),
        body: JSON.stringify({
          current_password: currentPassword,
          new_password: newPassword,
        }),
      });

      if (!response.ok) {
        const errorText = await response.text();
        throw new Error(errorText || 'Password change failed');
      }

      return { success: true };
    } catch (error) {
      console.error('[Auth] Password change error:', error);
      return { success: false, error: error.message };
    }
  };

  const isAuthenticated = () => {
    return !!token && !!user;
  };

  const getAuthHeaders = () => {
    return getRequestHeaders(true);
  };

  const value = {
    user,
    token,
    loading,
    login,
    logout,
    changePassword,
    isAuthenticated,
    getAuthHeaders,
    // Expose tenant-aware utilities
    currentTenant: getCurrentTenant(),
    getApiUrl,
  };

  return (
    <AuthContext.Provider value={value}>
      {children}
    </AuthContext.Provider>
  );
};