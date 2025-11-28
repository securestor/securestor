import React, { useState, useEffect } from 'react';
import { useTenant } from '../context/TenantContext';
import { 
  setDevTenant, 
  clearDevTenant, 
  debugTenantInfo,
  tenantConfig 
} from '../utils/tenant';

/**
 * TenantSwitcher Component
 * Development tool for testing multi-tenant functionality
 * Only visible in development mode
 * Allows switching between tenants without changing URL
 */
export const TenantSwitcher = () => {
  const { tenantSlug, tenantInfo, switchTenant } = useTenant();
  const [isOpen, setIsOpen] = useState(false);
  const [customTenant, setCustomTenant] = useState('');
  const [availableTenants] = useState([
    { slug: 'default', name: 'Default Organization' },
    { slug: 'admin', name: 'Admin Organization' },
    { slug: 'alpha', name: 'Alpha Corp' },
    { slug: 'beta', name: 'Beta Tech' },
  ]);

  // Only show in development mode
  if (!tenantConfig.DEV_MODE) {
    return null;
  }

  const handleQuickSwitch = (slug) => {
    // Use URL redirect for real tenant switching
    switchTenant(slug, true);
  };

  const handleDevOverride = (slug) => {
    if (!slug) {
      clearDevTenant();
    } else {
      setDevTenant(slug);
    }
  };

  const handleCustomTenant = () => {
    if (customTenant) {
      handleDevOverride(customTenant);
      setCustomTenant('');
    }
  };

  return (
    <div className="tenant-switcher-container" style={styles.container}>
      {/* Toggle Button */}
      <button
        onClick={() => setIsOpen(!isOpen)}
        style={styles.toggleButton}
        title="Tenant Switcher (Dev Mode)"
      >
        🏢 {tenantSlug}
      </button>

      {/* Switcher Panel */}
      {isOpen && (
        <div style={styles.panel}>
          <div style={styles.header}>
            <h3 style={styles.title}>Tenant Switcher</h3>
            <span style={styles.devBadge}>DEV MODE</span>
          </div>

          {/* Current Tenant Info */}
          <div style={styles.section}>
            <div style={styles.label}>Current Tenant</div>
            <div style={styles.currentTenant}>
              <span style={styles.tenantSlug}>{tenantSlug}</span>
              {tenantInfo.devOverride && (
                <span style={styles.overrideBadge}>Override Active</span>
              )}
            </div>
            <div style={styles.infoGrid}>
              <div style={styles.infoItem}>
                <span style={styles.infoLabel}>Subdomain:</span>
                <span style={styles.infoValue}>
                  {tenantInfo.subdomain || 'none'}
                </span>
              </div>
              <div style={styles.infoItem}>
                <span style={styles.infoLabel}>Hostname:</span>
                <span style={styles.infoValue}>{tenantInfo.hostname}</span>
              </div>
              <div style={styles.infoItem}>
                <span style={styles.infoLabel}>Is Default:</span>
                <span style={styles.infoValue}>
                  {tenantInfo.isDefault ? 'Yes' : 'No'}
                </span>
              </div>
            </div>
          </div>

          {/* Quick Switch (URL Redirect) */}
          <div style={styles.section}>
            <div style={styles.label}>Quick Switch (URL Redirect)</div>
            <div style={styles.buttonGrid}>
              {availableTenants.map((tenant) => (
                <button
                  key={tenant.slug}
                  onClick={() => handleQuickSwitch(tenant.slug)}
                  style={{
                    ...styles.tenantButton,
                    ...(tenant.slug === tenantSlug ? styles.tenantButtonActive : {}),
                  }}
                  disabled={tenant.slug === tenantSlug}
                >
                  {tenant.name}
                  <div style={styles.tenantSlugSmall}>{tenant.slug}</div>
                </button>
              ))}
            </div>
          </div>

          {/* Dev Override (localStorage) */}
          <div style={styles.section}>
            <div style={styles.label}>Dev Override (localStorage)</div>
            <div style={styles.overrideContainer}>
              <input
                type="text"
                value={customTenant}
                onChange={(e) => setCustomTenant(e.target.value)}
                placeholder="Enter tenant slug"
                style={styles.input}
              />
              <button onClick={handleCustomTenant} style={styles.overrideButton}>
                Override
              </button>
              <button
                onClick={() => handleDevOverride(null)}
                style={styles.clearButton}
              >
                Clear
              </button>
            </div>
            <div style={styles.hint}>
              ⚠️ Override bypasses URL detection. Use for testing only.
            </div>
          </div>

          {/* Debug */}
          <div style={styles.section}>
            <button onClick={debugTenantInfo} style={styles.debugButton}>
              🐛 Log Tenant Info to Console
            </button>
          </div>

          {/* Close Button */}
          <button onClick={() => setIsOpen(false)} style={styles.closeButton}>
            Close
          </button>
        </div>
      )}
    </div>
  );
};

// Inline styles for self-contained component
const styles = {
  container: {
    position: 'fixed',
    bottom: '20px',
    right: '20px',
    zIndex: 10000,
    fontFamily: 'system-ui, -apple-system, sans-serif',
  },
  toggleButton: {
    backgroundColor: '#6366f1',
    color: 'white',
    border: 'none',
    borderRadius: '50px',
    padding: '12px 20px',
    fontSize: '14px',
    fontWeight: '600',
    cursor: 'pointer',
    boxShadow: '0 4px 6px rgba(0, 0, 0, 0.1)',
    transition: 'all 0.2s',
  },
  panel: {
    position: 'absolute',
    bottom: '60px',
    right: '0',
    backgroundColor: 'white',
    borderRadius: '12px',
    boxShadow: '0 10px 40px rgba(0, 0, 0, 0.15)',
    padding: '20px',
    width: '400px',
    maxHeight: '600px',
    overflowY: 'auto',
  },
  header: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '20px',
    paddingBottom: '15px',
    borderBottom: '2px solid #e5e7eb',
  },
  title: {
    margin: 0,
    fontSize: '18px',
    fontWeight: '700',
    color: '#1f2937',
  },
  devBadge: {
    backgroundColor: '#fbbf24',
    color: '#78350f',
    padding: '4px 8px',
    borderRadius: '4px',
    fontSize: '11px',
    fontWeight: '700',
  },
  section: {
    marginBottom: '20px',
    paddingBottom: '20px',
    borderBottom: '1px solid #e5e7eb',
  },
  label: {
    fontSize: '12px',
    fontWeight: '600',
    color: '#6b7280',
    textTransform: 'uppercase',
    marginBottom: '10px',
    letterSpacing: '0.5px',
  },
  currentTenant: {
    display: 'flex',
    alignItems: 'center',
    gap: '10px',
    marginBottom: '15px',
  },
  tenantSlug: {
    fontSize: '20px',
    fontWeight: '700',
    color: '#1f2937',
  },
  overrideBadge: {
    backgroundColor: '#ef4444',
    color: 'white',
    padding: '3px 8px',
    borderRadius: '4px',
    fontSize: '10px',
    fontWeight: '600',
  },
  infoGrid: {
    display: 'grid',
    gap: '8px',
  },
  infoItem: {
    display: 'flex',
    justifyContent: 'space-between',
    fontSize: '13px',
  },
  infoLabel: {
    color: '#6b7280',
    fontWeight: '500',
  },
  infoValue: {
    color: '#1f2937',
    fontWeight: '600',
  },
  buttonGrid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(2, 1fr)',
    gap: '10px',
  },
  tenantButton: {
    backgroundColor: '#f3f4f6',
    border: '2px solid #e5e7eb',
    borderRadius: '8px',
    padding: '12px',
    cursor: 'pointer',
    fontSize: '13px',
    fontWeight: '600',
    color: '#1f2937',
    transition: 'all 0.2s',
  },
  tenantButtonActive: {
    backgroundColor: '#dbeafe',
    borderColor: '#3b82f6',
    color: '#1e40af',
  },
  tenantSlugSmall: {
    fontSize: '11px',
    color: '#6b7280',
    marginTop: '4px',
  },
  overrideContainer: {
    display: 'flex',
    gap: '8px',
    marginBottom: '10px',
  },
  input: {
    flex: 1,
    padding: '8px 12px',
    border: '2px solid #e5e7eb',
    borderRadius: '6px',
    fontSize: '13px',
  },
  overrideButton: {
    backgroundColor: '#10b981',
    color: 'white',
    border: 'none',
    borderRadius: '6px',
    padding: '8px 16px',
    fontSize: '13px',
    fontWeight: '600',
    cursor: 'pointer',
  },
  clearButton: {
    backgroundColor: '#ef4444',
    color: 'white',
    border: 'none',
    borderRadius: '6px',
    padding: '8px 16px',
    fontSize: '13px',
    fontWeight: '600',
    cursor: 'pointer',
  },
  hint: {
    fontSize: '11px',
    color: '#ef4444',
    marginTop: '8px',
  },
  debugButton: {
    width: '100%',
    backgroundColor: '#8b5cf6',
    color: 'white',
    border: 'none',
    borderRadius: '6px',
    padding: '10px',
    fontSize: '13px',
    fontWeight: '600',
    cursor: 'pointer',
  },
  closeButton: {
    width: '100%',
    backgroundColor: '#6b7280',
    color: 'white',
    border: 'none',
    borderRadius: '6px',
    padding: '10px',
    fontSize: '13px',
    fontWeight: '600',
    cursor: 'pointer',
    marginTop: '10px',
  },
};

export default TenantSwitcher;
