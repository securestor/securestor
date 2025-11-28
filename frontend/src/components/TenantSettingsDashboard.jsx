import React, { useState, useEffect } from 'react';
import {
  Box,
  Container,
  Typography,
  Paper,
  Tab,
  Tabs,
  Grid,
  Card,
  CardContent,
  CardHeader,
  Switch,
  FormControl,
  FormControlLabel,
  InputLabel,
  Select,
  MenuItem,
  TextField,
  Button,
  Chip,
  Alert,
  Divider,
  List,
  ListItem,
  ListItemText,
  ListItemSecondaryAction,
  IconButton,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  LinearProgress,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  Tooltip,
  Badge
} from '@mui/material';
import {
  ChevronDown,
  Shield,
  Users,
  Database,
  Bell,
  Zap,
  FileCheck,
  Edit,
  Save,
  X,
  Info,
  AlertTriangle,
  CheckCircle,
  AlertCircle,
  Plus,
  Trash2
} from 'lucide-react';
import { Line, Doughnut, Bar } from 'react-chartjs-2';

const TenantSettingsDashboard = () => {
  const [tenants, setTenants] = useState([]);
  const [selectedTenant, setSelectedTenant] = useState(null);
  const [settings, setSettings] = useState(null);
  const [usage, setUsage] = useState(null);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [activeTab, setActiveTab] = useState(0);
  const [editMode, setEditMode] = useState({});
  const [errors, setErrors] = useState({});
  const [success, setSuccess] = useState('');
  const [createTenantOpen, setCreateTenantOpen] = useState(false);
  const [newTenant, setNewTenant] = useState({
    name: '',
    subdomain: '',
    plan: 'basic',
    max_users: 100,
    features: []
  });

  const [filter, setFilter] = useState({
    search: '',
    is_active: null,
    plan: '',
    sort_by: 'created_at',
    sort_order: 'desc'
  });

  // Available features for tenants
  const availableFeatures = [
    'api_access',
    'vulnerability_scanning',
    'compliance_reports',
    'sso_integration',
    'audit_logs',
    'custom_domains',
    'webhooks',
    'advanced_analytics',
    'backup_restore',
    'white_labeling'
  ];

  // Available plans
  const plans = [
    { value: 'basic', label: 'Basic', maxUsers: 10, features: ['api_access', 'vulnerability_scanning'] },
    { value: 'professional', label: 'Professional', maxUsers: 100, features: ['api_access', 'vulnerability_scanning', 'compliance_reports', 'audit_logs'] },
    { value: 'enterprise', label: 'Enterprise', maxUsers: 1000, features: availableFeatures },
    { value: 'custom', label: 'Custom', maxUsers: 10000, features: availableFeatures }
  ];

  useEffect(() => {
    fetchTenants();
  }, [filter]);

  useEffect(() => {
    if (selectedTenant) {
      fetchTenantSettings();
      fetchTenantUsage();
    }
  }, [selectedTenant]);

  const fetchTenants = async () => {
    setLoading(true);
    try {
      const queryParams = new URLSearchParams();
      Object.entries(filter).forEach(([key, value]) => {
        if (value !== null && value !== '') {
          queryParams.append(key, value);
        }
      });

      const response = await fetch(`/api/v1/tenants?${queryParams}`);
      if (!response.ok) throw new Error('Failed to fetch tenants');
      
      const data = await response.json();
      setTenants(data.tenants || []);
      
      if (!selectedTenant && data.tenants?.length > 0) {
        setSelectedTenant(data.tenants[0]);
      }
    } catch (error) {
      setErrors({ general: 'Failed to fetch tenants: ' + error.message });
    } finally {
      setLoading(false);
    }
  };

  const fetchTenantSettings = async () => {
    if (!selectedTenant) return;
    
    try {
      const response = await fetch(`/api/v1/tenants/${selectedTenant.id}/settings`);
      if (!response.ok) throw new Error('Failed to fetch settings');
      
      const data = await response.json();
      setSettings(data);
    } catch (error) {
      setErrors({ settings: 'Failed to fetch settings: ' + error.message });
    }
  };

  const fetchTenantUsage = async () => {
    if (!selectedTenant) return;
    
    try {
      const response = await fetch(`/api/v1/tenants/${selectedTenant.id}/usage`);
      if (!response.ok) throw new Error('Failed to fetch usage');
      
      const data = await response.json();
      setUsage(data);
    } catch (error) {
      setErrors({ usage: 'Failed to fetch usage: ' + error.message });
    }
  };

  const createTenant = async () => {
    setSaving(true);
    try {
      const response = await fetch('/api/v1/tenants', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(newTenant)
      });

      if (!response.ok) throw new Error('Failed to create tenant');
      
      const createdTenant = await response.json();
      setTenants(prev => [createdTenant, ...prev]);
      setSelectedTenant(createdTenant);
      setCreateTenantOpen(false);
      setNewTenant({ name: '', subdomain: '', plan: 'basic', max_users: 100, features: [] });
      setSuccess('Tenant created successfully');
    } catch (error) {
      setErrors({ create: 'Failed to create tenant: ' + error.message });
    } finally {
      setSaving(false);
    }
  };

  const updateTenantSettings = async (category, updatedSettings) => {
    setSaving(true);
    try {
      const response = await fetch(`/api/v1/tenants/${selectedTenant.id}/settings`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ [category]: updatedSettings })
      });

      if (!response.ok) throw new Error('Failed to update settings');
      
      const data = await response.json();
      setSettings(data);
      setEditMode(prev => ({ ...prev, [category]: false }));
      setSuccess('Settings updated successfully');
    } catch (error) {
      setErrors({ [category]: 'Failed to update settings: ' + error.message });
    } finally {
      setSaving(false);
    }
  };

  const updateTenant = async (tenantId, updates) => {
    setSaving(true);
    try {
      const response = await fetch(`/api/v1/tenants/${tenantId}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(updates)
      });

      if (!response.ok) throw new Error('Failed to update tenant');
      
      const updatedTenant = await response.json();
      setTenants(prev => prev.map(t => t.id === tenantId ? updatedTenant : t));
      if (selectedTenant?.id === tenantId) {
        setSelectedTenant(updatedTenant);
      }
      setSuccess('Tenant updated successfully');
    } catch (error) {
      setErrors({ tenant: 'Failed to update tenant: ' + error.message });
    } finally {
      setSaving(false);
    }
  };

  const getUsageColor = (percentage) => {
    if (percentage >= 90) return 'error';
    if (percentage >= 75) return 'warning';
    return 'success';
  };

  const renderTenantList = () => (
    <Card sx={{ mb: 3 }}>
      <CardHeader 
        title="Tenants" 
        action={
          <Button 
            variant="contained" 
            startIcon={<Plus />}
            onClick={() => setCreateTenantOpen(true)}
          >
            Create Tenant
          </Button>
        }
      />
      <CardContent>
        <Grid container spacing={2} sx={{ mb: 2 }}>
          <Grid item xs={12} md={4}>
            <TextField
              fullWidth
              label="Search"
              value={filter.search}
              onChange={(e) => setFilter(prev => ({ ...prev, search: e.target.value }))}
            />
          </Grid>
          <Grid item xs={12} md={2}>
            <FormControl fullWidth>
              <InputLabel>Status</InputLabel>
              <Select
                value={filter.is_active || ''}
                onChange={(e) => setFilter(prev => ({ ...prev, is_active: e.target.value || null }))}
              >
                <MenuItem value="">All</MenuItem>
                <MenuItem value={true}>Active</MenuItem>
                <MenuItem value={false}>Inactive</MenuItem>
              </Select>
            </FormControl>
          </Grid>
          <Grid item xs={12} md={2}>
            <FormControl fullWidth>
              <InputLabel>Plan</InputLabel>
              <Select
                value={filter.plan}
                onChange={(e) => setFilter(prev => ({ ...prev, plan: e.target.value }))}
              >
                <MenuItem value="">All Plans</MenuItem>
                {plans.map(plan => (
                  <MenuItem key={plan.value} value={plan.value}>{plan.label}</MenuItem>
                ))}
              </Select>
            </FormControl>
          </Grid>
        </Grid>

        <List>
          {tenants.map((tenant) => (
            <ListItem
              key={tenant.id}
              selected={selectedTenant?.id === tenant.id}
              onClick={() => setSelectedTenant(tenant)}
              sx={{ cursor: 'pointer', borderRadius: 1, mb: 1 }}
            >
              <ListItemText
                primary={
                  <Box display="flex" alignItems="center" gap={1}>
                    <Typography variant="h6">{tenant.name}</Typography>
                    <Chip 
                      label={tenant.plan} 
                      size="small" 
                      color={tenant.is_active ? 'success' : 'default'}
                    />
                    {!tenant.is_active && <Chip label="Inactive" size="small" color="error" />}
                  </Box>
                }
                secondary={
                  <Box>
                    <Typography variant="body2" color="text.secondary">
                      {tenant.current_users}/{tenant.max_users} users ({tenant.usage_percent.toFixed(1)}%)
                    </Typography>
                    {tenant.subdomain && (
                      <Typography variant="body2" color="text.secondary">
                        {tenant.subdomain}.securestor.com
                      </Typography>
                    )}
                  </Box>
                }
              />
              <ListItemSecondaryAction>
                <LinearProgress
                  variant="determinate"
                  value={Math.min(tenant.usage_percent, 100)}
                  color={getUsageColor(tenant.usage_percent)}
                  sx={{ width: 100, mr: 2 }}
                />
              </ListItemSecondaryAction>
            </ListItem>
          ))}
        </List>
      </CardContent>
    </Card>
  );

  const renderUsageOverview = () => {
    if (!usage) return null;

    const usageData = {
      labels: ['Users', 'API Keys', 'Storage', 'API Calls'],
      datasets: [{
        data: [usage.user_count, usage.api_key_count, usage.storage_used_gb, usage.requests_last_30_days],
        backgroundColor: ['#2196F3', '#4CAF50', '#FF9800', '#9C27B0'],
        borderWidth: 0
      }]
    };

    return (
      <Card sx={{ mb: 3 }}>
        <CardHeader title="Usage Overview" />
        <CardContent>
          <Grid container spacing={3}>
            <Grid item xs={12} md={6}>
              <Box height={200}>
                <Doughnut data={usageData} options={{ maintainAspectRatio: false }} />
              </Box>
            </Grid>
            <Grid item xs={12} md={6}>
              <Box display="flex" flexDirection="column" gap={2}>
                <Box display="flex" justifyContent="space-between">
                  <Typography>Users:</Typography>
                  <Typography fontWeight="bold">{usage.user_count}</Typography>
                </Box>
                <Box display="flex" justifyContent="space-between">
                  <Typography>API Keys:</Typography>
                  <Typography fontWeight="bold">{usage.api_key_count}</Typography>
                </Box>
                <Box display="flex" justifyContent="space-between">
                  <Typography>Storage Used:</Typography>
                  <Typography fontWeight="bold">{usage.storage_used_gb.toFixed(2)} GB</Typography>
                </Box>
                <Box display="flex" justifyContent="space-between">
                  <Typography>API Calls (30d):</Typography>
                  <Typography fontWeight="bold">{usage.requests_last_30_days.toLocaleString()}</Typography>
                </Box>
                <Box display="flex" justifyContent="space-between">
                  <Typography>Active Sessions:</Typography>
                  <Typography fontWeight="bold">{usage.active_sessions}</Typography>
                </Box>
              </Box>
            </Grid>
          </Grid>
        </CardContent>
      </Card>
    );
  };

  const renderSecuritySettings = () => {
    if (!settings) return null;

    const isEditing = editMode.security;
    const securitySettings = settings.security;

    return (
      <Accordion expanded={true}>
        <AccordionSummary expandIcon={<ChevronDown />}>
          <Shield sx={{ mr: 2 }} />
          <Typography variant="h6">Security Settings</Typography>
        </AccordionSummary>
        <AccordionDetails>
          <Grid container spacing={3}>
            <Grid item xs={12} md={6}>
              <FormControlLabel
                control={
                  <Switch
                    checked={securitySettings.mfa_required}
                    disabled={!isEditing}
                    onChange={(e) => setSettings(prev => ({
                      ...prev,
                      security: { ...prev.security, mfa_required: e.target.checked }
                    }))}
                  />
                }
                label="Require MFA for all users"
              />
            </Grid>
            <Grid item xs={12} md={6}>
              <FormControlLabel
                control={
                  <Switch
                    checked={securitySettings.require_sso}
                    disabled={!isEditing}
                    onChange={(e) => setSettings(prev => ({
                      ...prev,
                      security: { ...prev.security, require_sso: e.target.checked }
                    }))}
                  />
                }
                label="Require SSO authentication"
              />
            </Grid>
            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                label="Session Timeout (minutes)"
                type="number"
                value={securitySettings.session_timeout_minutes}
                disabled={!isEditing}
                onChange={(e) => setSettings(prev => ({
                  ...prev,
                  security: { ...prev.security, session_timeout_minutes: parseInt(e.target.value) }
                }))}
              />
            </Grid>
            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                label="Max Login Attempts"
                type="number"
                value={securitySettings.max_login_attempts}
                disabled={!isEditing}
                onChange={(e) => setSettings(prev => ({
                  ...prev,
                  security: { ...prev.security, max_login_attempts: parseInt(e.target.value) }
                }))}
              />
            </Grid>
            <Grid item xs={12}>
              <Typography variant="subtitle2" gutterBottom>Password Policy</Typography>
              <Grid container spacing={2}>
                <Grid item xs={12} md={3}>
                  <TextField
                    fullWidth
                    label="Min Length"
                    type="number"
                    value={securitySettings.password_policy.min_length}
                    disabled={!isEditing}
                    onChange={(e) => setSettings(prev => ({
                      ...prev,
                      security: {
                        ...prev.security,
                        password_policy: {
                          ...prev.security.password_policy,
                          min_length: parseInt(e.target.value)
                        }
                      }
                    }))}
                  />
                </Grid>
                <Grid item xs={12} md={3}>
                  <FormControlLabel
                    control={
                      <Switch
                        checked={securitySettings.password_policy.require_uppercase}
                        disabled={!isEditing}
                        onChange={(e) => setSettings(prev => ({
                          ...prev,
                          security: {
                            ...prev.security,
                            password_policy: {
                              ...prev.security.password_policy,
                              require_uppercase: e.target.checked
                            }
                          }
                        }))}
                      />
                    }
                    label="Uppercase"
                  />
                </Grid>
                <Grid item xs={12} md={3}>
                  <FormControlLabel
                    control={
                      <Switch
                        checked={securitySettings.password_policy.require_numbers}
                        disabled={!isEditing}
                        onChange={(e) => setSettings(prev => ({
                          ...prev,
                          security: {
                            ...prev.security,
                            password_policy: {
                              ...prev.security.password_policy,
                              require_numbers: e.target.checked
                            }
                          }
                        }))}
                      />
                    }
                    label="Numbers"
                  />
                </Grid>
                <Grid item xs={12} md={3}>
                  <FormControlLabel
                    control={
                      <Switch
                        checked={securitySettings.password_policy.require_symbols}
                        disabled={!isEditing}
                        onChange={(e) => setSettings(prev => ({
                          ...prev,
                          security: {
                            ...prev.security,
                            password_policy: {
                              ...prev.security.password_policy,
                              require_symbols: e.target.checked
                            }
                          }
                        }))}
                      />
                    }
                    label="Symbols"
                  />
                </Grid>
              </Grid>
            </Grid>
            <Grid item xs={12}>
              <Box display="flex" gap={2} justifyContent="flex-end">
                {isEditing ? (
                  <>
                    <Button
                      variant="outlined"
                      startIcon={<X />}
                      onClick={() => {
                        setEditMode(prev => ({ ...prev, security: false }));
                        fetchTenantSettings(); // Reset changes
                      }}
                    >
                      Cancel
                    </Button>
                    <Button
                      variant="contained"
                      startIcon={<Save />}
                      onClick={() => updateTenantSettings('security', settings.security)}
                      disabled={saving}
                    >
                      Save Changes
                    </Button>
                  </>
                ) : (
                  <Button
                    variant="outlined"
                    startIcon={<Edit />}
                    onClick={() => setEditMode(prev => ({ ...prev, security: true }))}
                  >
                    Edit Security Settings
                  </Button>
                )}
              </Box>
            </Grid>
          </Grid>
        </AccordionDetails>
      </Accordion>
    );
  };

  const renderUserManagementSettings = () => {
    if (!settings) return null;

    const isEditing = editMode.user_management;
    const userSettings = settings.user_management;

    return (
      <Accordion>
        <AccordionSummary expandIcon={<ChevronDown />}>
          <Users sx={{ mr: 2 }} />
          <Typography variant="h6">User Management</Typography>
        </AccordionSummary>
        <AccordionDetails>
          <Grid container spacing={3}>
            <Grid item xs={12} md={6}>
              <FormControlLabel
                control={
                  <Switch
                    checked={userSettings.allow_self_registration}
                    disabled={!isEditing}
                    onChange={(e) => setSettings(prev => ({
                      ...prev,
                      user_management: { ...prev.user_management, allow_self_registration: e.target.checked }
                    }))}
                  />
                }
                label="Allow self-registration"
              />
            </Grid>
            <Grid item xs={12} md={6}>
              <FormControlLabel
                control={
                  <Switch
                    checked={userSettings.email_verification_required}
                    disabled={!isEditing}
                    onChange={(e) => setSettings(prev => ({
                      ...prev,
                      user_management: { ...prev.user_management, email_verification_required: e.target.checked }
                    }))}
                  />
                }
                label="Require email verification"
              />
            </Grid>
            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                label="Max Users"
                type="number"
                value={userSettings.max_users}
                disabled={!isEditing}
                onChange={(e) => setSettings(prev => ({
                  ...prev,
                  user_management: { ...prev.user_management, max_users: parseInt(e.target.value) }
                }))}
              />
            </Grid>
            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                label="Invitation Expiry (days)"
                type="number"
                value={userSettings.invitation_expiry_days}
                disabled={!isEditing}
                onChange={(e) => setSettings(prev => ({
                  ...prev,
                  user_management: { ...prev.user_management, invitation_expiry_days: parseInt(e.target.value) }
                }))}
              />
            </Grid>
            <Grid item xs={12}>
              <Box display="flex" gap={2} justifyContent="flex-end">
                {isEditing ? (
                  <>
                    <Button
                      variant="outlined"
                      startIcon={<X />}
                      onClick={() => {
                        setEditMode(prev => ({ ...prev, user_management: false }));
                        fetchTenantSettings();
                      }}
                    >
                      Cancel
                    </Button>
                    <Button
                      variant="contained"
                      startIcon={<Save />}
                      onClick={() => updateTenantSettings('user_management', settings.user_management)}
                      disabled={saving}
                    >
                      Save Changes
                    </Button>
                  </>
                ) : (
                  <Button
                    variant="outlined"
                    startIcon={<Edit />}
                    onClick={() => setEditMode(prev => ({ ...prev, user_management: true }))}
                  >
                    Edit User Settings
                  </Button>
                )}
              </Box>
            </Grid>
          </Grid>
        </AccordionDetails>
      </Accordion>
    );
  };

  const renderStorageSettings = () => {
    if (!settings) return null;

    const isEditing = editMode.storage;
    const storageSettings = settings.storage;

    return (
      <Accordion>
        <AccordionSummary expandIcon={<ChevronDown />}>
          <Database sx={{ mr: 2 }} />
          <Typography variant="h6">Storage & Data</Typography>
        </AccordionSummary>
        <AccordionDetails>
          <Grid container spacing={3}>
            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                label="Max Storage (GB)"
                type="number"
                value={storageSettings.max_storage_gb}
                disabled={!isEditing}
                onChange={(e) => setSettings(prev => ({
                  ...prev,
                  storage: { ...prev.storage, max_storage_gb: parseInt(e.target.value) }
                }))}
              />
            </Grid>
            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                label="Max File Size (MB)"
                type="number"
                value={storageSettings.max_file_size}
                disabled={!isEditing}
                onChange={(e) => setSettings(prev => ({
                  ...prev,
                  storage: { ...prev.storage, max_file_size: parseInt(e.target.value) }
                }))}
              />
            </Grid>
            <Grid item xs={12} md={6}>
              <FormControlLabel
                control={
                  <Switch
                    checked={storageSettings.backup_enabled}
                    disabled={!isEditing}
                    onChange={(e) => setSettings(prev => ({
                      ...prev,
                      storage: { ...prev.storage, backup_enabled: e.target.checked }
                    }))}
                  />
                }
                label="Enable backups"
              />
            </Grid>
            <Grid item xs={12}>
              <Box 
                sx={{ 
                  p: 2, 
                  bgcolor: 'success.50', 
                  border: '1px solid',
                  borderColor: 'success.200',
                  borderRadius: 1,
                  display: 'flex',
                  alignItems: 'center',
                  gap: 1
                }}
              >
                <svg className="w-5 h-5" style={{ color: '#16a34a' }} fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
                </svg>
                <Box flex={1}>
                  <Box display="flex" alignItems="center" gap={1}>
                    <span style={{ fontSize: '0.875rem', fontWeight: 600 }}>Enterprise Encryption Enabled</span>
                    <span style={{ 
                      fontSize: '0.75rem', 
                      fontWeight: 600,
                      padding: '2px 8px',
                      borderRadius: '12px',
                      backgroundColor: '#d1fae5',
                      color: '#065f46'
                    }}>
                      ACTIVE & ENFORCED
                    </span>
                  </Box>
                  <span style={{ fontSize: '0.75rem', color: '#065f46' }}>
                    AES-256-GCM encryption is system-managed and mandatory for all artifacts
                  </span>
                </Box>
              </Box>
            </Grid>
            <Grid item xs={12}>
              <Box display="flex" gap={2} justifyContent="flex-end">
                {isEditing ? (
                  <>
                    <Button
                      variant="outlined"
                      startIcon={<X />}
                      onClick={() => {
                        setEditMode(prev => ({ ...prev, storage: false }));
                        fetchTenantSettings();
                      }}
                    >
                      Cancel
                    </Button>
                    <Button
                      variant="contained"
                      startIcon={<Save />}
                      onClick={() => updateTenantSettings('storage', settings.storage)}
                      disabled={saving}
                    >
                      Save Changes
                    </Button>
                  </>
                ) : (
                  <Button
                    variant="outlined"
                    startIcon={<Edit />}
                    onClick={() => setEditMode(prev => ({ ...prev, storage: true }))}
                  >
                    Edit Storage Settings
                  </Button>
                )}
              </Box>
            </Grid>
          </Grid>
        </AccordionDetails>
      </Accordion>
    );
  };

  const renderComplianceSettings = () => {
    if (!settings) return null;

    const isEditing = editMode.compliance;
    const complianceSettings = settings.compliance;

    return (
      <Accordion>
        <AccordionSummary expandIcon={<ChevronDown />}>
          <FileCheck sx={{ mr: 2 }} />
          <Typography variant="h6">Compliance & Audit</Typography>
        </AccordionSummary>
        <AccordionDetails>
          <Grid container spacing={3}>
            <Grid item xs={12} md={6}>
              <FormControl fullWidth disabled={!isEditing}>
                <InputLabel>Compliance Mode</InputLabel>
                <Select
                  value={complianceSettings.compliance_mode}
                  onChange={(e) => setSettings(prev => ({
                    ...prev,
                    compliance: { ...prev.compliance, compliance_mode: e.target.value }
                  }))}
                >
                  <MenuItem value="none">None</MenuItem>
                  <MenuItem value="basic">Basic</MenuItem>
                  <MenuItem value="strict">Strict</MenuItem>
                </Select>
              </FormControl>
            </Grid>
            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                label="Audit Retention (days)"
                type="number"
                value={complianceSettings.audit_retention_days}
                disabled={!isEditing}
                onChange={(e) => setSettings(prev => ({
                  ...prev,
                  compliance: { ...prev.compliance, audit_retention_days: parseInt(e.target.value) }
                }))}
              />
            </Grid>
            <Grid item xs={12} md={4}>
              <FormControlLabel
                control={
                  <Switch
                    checked={complianceSettings.gdpr_compliance}
                    disabled={!isEditing}
                    onChange={(e) => setSettings(prev => ({
                      ...prev,
                      compliance: { ...prev.compliance, gdpr_compliance: e.target.checked }
                    }))}
                  />
                }
                label="GDPR Compliance"
              />
            </Grid>
            <Grid item xs={12} md={4}>
              <FormControlLabel
                control={
                  <Switch
                    checked={complianceSettings.soc2_compliance}
                    disabled={!isEditing}
                    onChange={(e) => setSettings(prev => ({
                      ...prev,
                      compliance: { ...prev.compliance, soc2_compliance: e.target.checked }
                    }))}
                  />
                }
                label="SOC2 Compliance"
              />
            </Grid>
            <Grid item xs={12} md={4}>
              <FormControlLabel
                control={
                  <Switch
                    checked={complianceSettings.hipaa_compliance}
                    disabled={!isEditing}
                    onChange={(e) => setSettings(prev => ({
                      ...prev,
                      compliance: { ...prev.compliance, hipaa_compliance: e.target.checked }
                    }))}
                  />
                }
                label="HIPAA Compliance"
              />
            </Grid>
            <Grid item xs={12}>
              <Box display="flex" gap={2} justifyContent="flex-end">
                {isEditing ? (
                  <>
                    <Button
                      variant="outlined"
                      startIcon={<X />}
                      onClick={() => {
                        setEditMode(prev => ({ ...prev, compliance: false }));
                        fetchTenantSettings();
                      }}
                    >
                      Cancel
                    </Button>
                    <Button
                      variant="contained"
                      startIcon={<Save />}
                      onClick={() => updateTenantSettings('compliance', settings.compliance)}
                      disabled={saving}
                    >
                      Save Changes
                    </Button>
                  </>
                ) : (
                  <Button
                    variant="outlined"
                    startIcon={<Edit />}
                    onClick={() => setEditMode(prev => ({ ...prev, compliance: true }))}
                  >
                    Edit Compliance Settings
                  </Button>
                )}
              </Box>
            </Grid>
          </Grid>
        </AccordionDetails>
      </Accordion>
    );
  };

  const renderCreateTenantDialog = () => (
    <Dialog open={createTenantOpen} onClose={() => setCreateTenantOpen(false)} maxWidth="md" fullWidth>
      <DialogTitle>Create New Tenant</DialogTitle>
      <DialogContent>
        <Grid container spacing={3} sx={{ mt: 1 }}>
          <Grid item xs={12} md={6}>
            <TextField
              fullWidth
              label="Tenant Name"
              value={newTenant.name}
              onChange={(e) => setNewTenant(prev => ({ ...prev, name: e.target.value }))}
              required
            />
          </Grid>
          <Grid item xs={12} md={6}>
            <TextField
              fullWidth
              label="Subdomain"
              value={newTenant.subdomain}
              onChange={(e) => setNewTenant(prev => ({ ...prev, subdomain: e.target.value }))}
              helperText="Will be accessible at subdomain.securestor.com"
            />
          </Grid>
          <Grid item xs={12} md={6}>
            <FormControl fullWidth>
              <InputLabel>Plan</InputLabel>
              <Select
                value={newTenant.plan}
                onChange={(e) => {
                  const selectedPlan = plans.find(p => p.value === e.target.value);
                  setNewTenant(prev => ({ 
                    ...prev, 
                    plan: e.target.value,
                    max_users: selectedPlan?.maxUsers || 100,
                    features: selectedPlan?.features || []
                  }));
                }}
              >
                {plans.map(plan => (
                  <MenuItem key={plan.value} value={plan.value}>
                    {plan.label} (Max {plan.maxUsers} users)
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
          </Grid>
          <Grid item xs={12} md={6}>
            <TextField
              fullWidth
              label="Max Users"
              type="number"
              value={newTenant.max_users}
              onChange={(e) => setNewTenant(prev => ({ ...prev, max_users: parseInt(e.target.value) }))}
            />
          </Grid>
          <Grid item xs={12}>
            <Typography variant="subtitle2" gutterBottom>Features</Typography>
            <Box display="flex" flexWrap="wrap" gap={1}>
              {newTenant.features.map(feature => (
                <Chip key={feature} label={feature} onDelete={() => {
                  setNewTenant(prev => ({
                    ...prev,
                    features: prev.features.filter(f => f !== feature)
                  }));
                }} />
              ))}
            </Box>
          </Grid>
        </Grid>
      </DialogContent>
      <DialogActions>
        <Button onClick={() => setCreateTenantOpen(false)}>Cancel</Button>
        <Button 
          onClick={createTenant} 
          variant="contained" 
          disabled={!newTenant.name || saving}
        >
          Create Tenant
        </Button>
      </DialogActions>
    </Dialog>
  );

  if (loading) {
    return (
      <Container maxWidth="xl" sx={{ py: 4 }}>
        <LinearProgress />
        <Typography sx={{ mt: 2 }}>Loading tenant settings...</Typography>
      </Container>
    );
  }

  return (
    <Container maxWidth="xl" sx={{ py: 4 }}>
      <Typography variant="h4" gutterBottom>
        Tenant Settings Management
      </Typography>

      {/* Error Messages */}
      {Object.entries(errors).map(([key, error]) => (
        <Alert key={key} severity="error" sx={{ mb: 2 }} onClose={() => setErrors(prev => ({ ...prev, [key]: undefined }))}>
          {error}
        </Alert>
      ))}

      {/* Success Message */}
      {success && (
        <Alert severity="success" sx={{ mb: 2 }} onClose={() => setSuccess('')}>
          {success}
        </Alert>
      )}

      <Grid container spacing={3}>
        <Grid item xs={12} md={4}>
          {renderTenantList()}
        </Grid>
        <Grid item xs={12} md={8}>
          {selectedTenant && (
            <>
              {renderUsageOverview()}
              
              <Paper sx={{ p: 3 }}>
                <Typography variant="h5" gutterBottom>
                  {selectedTenant.name} Settings
                </Typography>
                
                <Box sx={{ mb: 3 }}>
                  {renderSecuritySettings()}
                  {renderUserManagementSettings()}
                  {renderStorageSettings()}
                  {renderComplianceSettings()}
                </Box>
              </Paper>
            </>
          )}
        </Grid>
      </Grid>

      {renderCreateTenantDialog()}
    </Container>
  );
};

export default TenantSettingsDashboard;