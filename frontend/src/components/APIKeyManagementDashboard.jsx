import React, { useState, useEffect } from 'react';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Button,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TablePagination,
  TextField,
  InputAdornment,
  Chip,
  IconButton,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Grid,
  Tooltip,
  Menu,
  Alert,
  CircularProgress,
  Divider,
  FormControlLabel,
  Checkbox,
  FormGroup,
  Paper,
  List,
  ListItem,
  ListItemText,
  ListItemIcon,
  Collapse,
  LinearProgress,
  Tab,
  Tabs,
  TabPanel,
  Snackbar
} from '@mui/material';
import {
  Search,
  Plus,
  Trash2,
  MoreVertical,
  Key,
  Eye,
  EyeOff,
  Copy,
  BarChart3,
  TrendingUp,
  Shield,
  AlertTriangle,
  CheckCircle,
  X,
  ChevronDown,
  ChevronUp,
  TrendingUp as TrendingUpArrow,
  Gauge,
  Database,
  AlertCircle
} from 'lucide-react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip as RechartsTooltip, ResponsiveContainer, BarChart, Bar, PieChart, Pie, Cell } from 'recharts';

const APIKeyManagementDashboard = () => {
  // State management
  const [apiKeys, setApiKeys] = useState([]);
  const [scopes, setScopes] = useState([]);
  const [groupedScopes, setGroupedScopes] = useState({});
  const [totalCount, setTotalCount] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [success, setSuccess] = useState(null);

  // Pagination and filtering
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(25);
  const [searchTerm, setSearchTerm] = useState('');
  const [statusFilter, setStatusFilter] = useState('all');
  const [userFilter, setUserFilter] = useState('all');

  // Dialog states
  const [createKeyDialogOpen, setCreateKeyDialogOpen] = useState(false);
  const [analyticsDialogOpen, setAnalyticsDialogOpen] = useState(false);
  const [selectedApiKey, setSelectedApiKey] = useState(null);
  const [actionMenu, setActionMenu] = useState({ anchorEl: null, apiKey: null });

  // Form states
  const [keyForm, setKeyForm] = useState({
    name: '',
    description: '',
    scopes: [],
    expiresAt: '',
    rateLimitPerHour: 1000,
    rateLimitPerDay: 10000
  });

  // Analytics state
  const [analytics, setAnalytics] = useState(null);
  const [analyticsLoading, setAnalyticsLoading] = useState(false);
  const [analyticsDays, setAnalyticsDays] = useState(30);

  // Key creation response
  const [createdKey, setCreatedKey] = useState(null);
  const [showKeySecret, setShowKeySecret] = useState(false);
  const [copiedItems, setCopiedItems] = useState(new Set());

  // Scope expansion state
  const [expandedScopes, setExpandedScopes] = useState({});

  // Tab state for analytics
  const [analyticsTab, setAnalyticsTab] = useState(0);

  // Toast notification state
  const [snackbar, setSnackbar] = useState({ open: false, message: '', severity: 'success' });

  // Demo mode for testing
  const [demoMode, setDemoMode] = useState(false);

  // Fetch API keys
  const fetchApiKeys = async () => {
    try {
      setLoading(true);
      const params = new URLSearchParams({
        limit: rowsPerPage.toString(),
        offset: (page * rowsPerPage).toString(),
        ...(searchTerm && { search: searchTerm }),
        ...(statusFilter !== 'all' && { is_active: statusFilter === 'active' }),
        ...(userFilter !== 'all' && { user_id: userFilter })
      });

      const response = await fetch(`/api/keys?${params}`);
      const data = await response.json();

      if (response.ok) {
        setApiKeys(data.api_keys || []);
        setTotalCount(data.total_count || 0);
        setError(null);
      } else {
        throw new Error(data.message || 'Failed to fetch API keys');
      }
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  // Fetch scopes
  const fetchScopes = async () => {
    try {
      const response = await fetch('/api/scopes');
      const data = await response.json();

      if (response.ok) {
        setScopes(data.scopes || []);
        setGroupedScopes(data.grouped_scopes || {});
      } else {
        throw new Error(data.message || 'Failed to fetch scopes');
      }
    } catch (err) {
      setError(err.message);
    }
  };

  // Create API key
  const handleCreateApiKey = async () => {
    try {
      // Demo mode for testing
      if (demoMode) {
        const demoKey = {
          id: `demo-${Date.now()}`,
          name: keyForm.name,
          description: keyForm.description,
          key_secret: `sk_test_${Math.random().toString(36).substr(2, 32)}${Math.random().toString(36).substr(2, 32)}`,
          key_id: `key_${Math.random().toString(36).substr(2, 16)}`,
          rate_limit_per_hour: keyForm.rateLimitPerHour,
          rate_limit_per_day: keyForm.rateLimitPerDay,
          expires_at: keyForm.expiresAt || null,
          scopes: keyForm.scopes
        };
        
        setCreatedKey(demoKey);
        setSuccess('API key created successfully (Demo Mode)');
        setCreateKeyDialogOpen(false);
        
        // Reset form
        setKeyForm({
          name: '',
          description: '',
          scopes: [],
          expiresAt: '',
          rateLimitPerHour: 1000,
          rateLimitPerDay: 10000
        });
        return;
      }

      const response = await fetch('/api/keys', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          name: keyForm.name,
          description: keyForm.description,
          scopes: keyForm.scopes,
          expires_at: keyForm.expiresAt || null,
          rate_limit_per_hour: keyForm.rateLimitPerHour,
          rate_limit_per_day: keyForm.rateLimitPerDay
        }),
      });

      if (response.ok) {
        const data = await response.json();
        
        // Ensure we always set the created key
        setCreatedKey(data);
        setSuccess('API key created successfully');
        setCreateKeyDialogOpen(false);
        
        // Show a prominent alert if the key_secret is missing
        if (!data.key_secret && !data.key) {
          console.warn('⚠️ Warning: API response does not contain key_secret or key field');
          alert('API key created, but the secret key was not returned by the server. This might be a backend configuration issue.');
        }
        
        // Reset form
        setKeyForm({
          name: '',
          description: '',
          scopes: [],
          expiresAt: '',
          rateLimitPerHour: 1000,
          rateLimitPerDay: 10000
        });
        
        fetchApiKeys();
      } else {
        const data = await response.json();
        throw new Error(data.message || 'Failed to create API key');
      }
    } catch (err) {
      setError(err.message);
    }
  };

  // Revoke API key
  const handleRevokeApiKey = async (apiKey) => {
    const reason = prompt(`Why are you revoking the API key "${apiKey.name}"?`);
    if (reason === null) return; // User cancelled

    try {
      const response = await fetch(`/api/keys/${apiKey.id}`, {
        method: 'DELETE',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ reason }),
      });

      if (response.ok) {
        setSuccess('API key revoked successfully');
        fetchApiKeys();
      } else {
        const data = await response.json();
        throw new Error(data.message || 'Failed to revoke API key');
      }
    } catch (err) {
      setError(err.message);
    }
    setActionMenu({ anchorEl: null, apiKey: null });
  };

  // View analytics
  const handleViewAnalytics = async (apiKey) => {
    setSelectedApiKey(apiKey);
    setAnalyticsDialogOpen(true);
    setActionMenu({ anchorEl: null, apiKey: null });
    await fetchAnalytics(apiKey.id);
  };

  // Fetch analytics
  const fetchAnalytics = async (keyId) => {
    try {
      setAnalyticsLoading(true);
      const response = await fetch(`/api/keys/${keyId}/analytics?days=${analyticsDays}`);
      const data = await response.json();

      if (response.ok) {
        setAnalytics(data);
      } else {
        throw new Error(data.message || 'Failed to fetch analytics');
      }
    } catch (err) {
      setError(err.message);
    } finally {
      setAnalyticsLoading(false);
    }
  };

  // Copy to clipboard with enhanced feedback
  const copyToClipboard = async (text, label = 'text', itemId = null) => {
    try {
      await navigator.clipboard.writeText(text);
      
      // Show both success alert and snackbar for immediate feedback
      setSuccess(`${label} copied to clipboard!`);
      setSnackbar({
        open: true,
        message: `✅ ${label} copied successfully!`,
        severity: 'success'
      });
      
      // Add visual feedback for the specific action
      if (createdKey && text === createdKey.key_secret) {
        // Trigger a brief visual effect for the created key
        setShowKeySecret(true);
        setTimeout(() => setShowKeySecret(false), 2000);
      }
      
      // Add visual feedback for copied items in the table
      if (itemId) {
        setCopiedItems(prev => new Set([...prev, itemId]));
        setTimeout(() => {
          setCopiedItems(prev => {
            const newSet = new Set(prev);
            newSet.delete(itemId);
            return newSet;
          });
        }, 2000);
      }
    } catch (err) {
      setError('Failed to copy to clipboard');
      setSnackbar({
        open: true,
        message: '❌ Failed to copy to clipboard',
        severity: 'error'
      });
    }
  };

  // Handle search
  const handleSearch = (event) => {
    setSearchTerm(event.target.value);
    setPage(0);
  };

  // Handle page change
  const handleChangePage = (event, newPage) => {
    setPage(newPage);
  };

  // Handle rows per page change
  const handleChangeRowsPerPage = (event) => {
    setRowsPerPage(parseInt(event.target.value, 10));
    setPage(0);
  };

  // Toggle scope selection
  const handleToggleScope = (scopeName) => {
    setKeyForm(prev => ({
      ...prev,
      scopes: prev.scopes.includes(scopeName)
        ? prev.scopes.filter(s => s !== scopeName)
        : [...prev.scopes, scopeName]
    }));
  };

  // Toggle scope group expansion
  const toggleScopeExpansion = (resource) => {
    setExpandedScopes(prev => ({
      ...prev,
      [resource]: !prev[resource]
    }));
  };

  // Get status color
  const getStatusColor = (apiKey) => {
    if (!apiKey.is_active) return 'error';
    if (apiKey.expires_at && new Date(apiKey.expires_at) < new Date()) return 'warning';
    return 'success';
  };

  // Get status label
  const getStatusLabel = (apiKey) => {
    if (!apiKey.is_active) return 'Revoked';
    if (apiKey.expires_at && new Date(apiKey.expires_at) < new Date()) return 'Expired';
    return 'Active';
  };

  // Format usage count
  const formatUsageCount = (count) => {
    if (count >= 1000000) return `${(count / 1000000).toFixed(1)}M`;
    if (count >= 1000) return `${(count / 1000).toFixed(1)}K`;
    return count.toString();
  };

  // Format data size
  const formatDataSize = (bytes) => {
    if (bytes >= 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`;
    if (bytes >= 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
    if (bytes >= 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${bytes} B`;
  };

  // Chart colors
  const CHART_COLORS = ['#8884d8', '#82ca9d', '#ffc658', '#ff7300', '#00ff00'];

  // Effects
  useEffect(() => {
    fetchApiKeys();
  }, [page, rowsPerPage, searchTerm, statusFilter, userFilter]);

  useEffect(() => {
    fetchScopes();
  }, []);

  // Clear messages after 5 seconds
  useEffect(() => {
    if (error || success) {
      const timer = setTimeout(() => {
        setError(null);
        setSuccess(null);
      }, 5000);
      return () => clearTimeout(timer);
    }
  }, [error, success]);

  // Update analytics when days change
  useEffect(() => {
    if (selectedApiKey && analyticsDialogOpen) {
      fetchAnalytics(selectedApiKey.id);
    }
  }, [analyticsDays]);

  // Debug: Log when createdKey changes
  useEffect(() => {
    if (createdKey) {
    }
  }, [createdKey]);

  return (
    <Box sx={{ p: 3 }}>
      {/* Header */}
      <Box sx={{ mb: 3, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Box>
          <Typography variant="h4" component="h1">
            API Key Management
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
            Create and manage API keys for secure access to your resources
          </Typography>
        </Box>
        <Box sx={{ display: 'flex', gap: 2 }}>
          <Button
            variant={demoMode ? "contained" : "outlined"}
            color={demoMode ? "secondary" : "default"}
            onClick={() => setDemoMode(!demoMode)}
            sx={{ minHeight: 48 }}
          >
            {demoMode ? 'Demo Mode ON' : 'Enable Demo Mode'}
          </Button>
          <Button
            variant="contained"
            size="large"
            startIcon={<Key />}
            onClick={() => setCreateKeyDialogOpen(true)}
            sx={{
              minHeight: 48,
              px: 3,
              '&:hover': {
                transform: 'translateY(-2px)',
                boxShadow: '0 8px 25px rgba(0, 0, 0, 0.15)',
              },
              transition: 'all 0.3s ease'
            }}
          >
            Create New API Key
          </Button>
        </Box>
      </Box>

      {/* Demo Mode Alert */}
      {demoMode && (
        <Alert severity="info" sx={{ mb: 2 }}>
          <Typography variant="subtitle2" gutterBottom>
            🧪 Demo Mode Active
          </Typography>
          <Typography variant="body2">
            Creating API keys will generate demo keys for testing the copy functionality. 
            No real API calls will be made.
          </Typography>
        </Alert>
      )}

      {/* Alerts */}
      {error && (
        <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>
          {error}
        </Alert>
      )}
      {success && (
        <Alert severity="success" sx={{ mb: 2 }} onClose={() => setSuccess(null)}>
          {success}
        </Alert>
      )}

      {/* Floating API Key Display */}
      {createdKey ? (
        <Card sx={{ mb: 3, border: '3px solid', borderColor: 'success.main', backgroundColor: 'success.light' }}>
          <CardContent>
            <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2 }}>
              <Typography variant="h6" sx={{ color: 'success.contrastText', fontWeight: 'bold' }}>
                🔑 New API Key Created: {createdKey.name}
              </Typography>
              <IconButton onClick={() => setCreatedKey(null)} sx={{ color: 'success.contrastText' }}>
                <X />
              </IconButton>
            </Box>
            
            <Alert severity="warning" sx={{ mb: 2 }}>
              ⚠️ Save this key now! This is the only time you'll see it.
            </Alert>
            
            <Box sx={{ display: 'flex', gap: 2, alignItems: 'center', flexWrap: 'wrap' }}>
              <TextField
                value={createdKey?.key_secret || createdKey?.key || 'API-KEY-NOT-AVAILABLE'}
                type={showKeySecret ? 'text' : 'password'}
                variant="outlined"
                size="small"
                sx={{ 
                  flexGrow: 1, 
                  minWidth: 300,
                  '& .MuiInputBase-input': {
                    fontFamily: 'monospace',
                    fontSize: '14px'
                  }
                }}
                InputProps={{
                  readOnly: true,
                  endAdornment: (
                    <InputAdornment position="end">
                      <IconButton onClick={() => setShowKeySecret(!showKeySecret)} size="small">
                        {showKeySecret ? <EyeOff /> : <Eye />}
                      </IconButton>
                    </InputAdornment>
                  ),
                }}
              />
              <Button
                variant="contained"
                color="primary"
                size="large"
                startIcon={<Copy />}
                onClick={() => copyToClipboard(
                  createdKey?.key_secret || createdKey?.key || 'API-KEY-NOT-AVAILABLE', 
                  'API key',
                  'floating-key'
                )}
                sx={{ 
                  minWidth: 150,
                  '&:hover': {
                    transform: 'scale(1.05)',
                  },
                  transition: 'transform 0.2s'
                }}
              >
                Copy Key
              </Button>
              <Button
                variant="outlined"
                startIcon={<Copy />}
                onClick={() => copyToClipboard(
                  `Authorization: Bearer ${createdKey?.key_secret || createdKey?.key || 'API-KEY-NOT-AVAILABLE'}`,
                  'Authorization header',
                  'floating-auth'
                )}
              >
                Copy Auth Header
              </Button>
            </Box>
          </CardContent>
        </Card>
      ) : null}

      {/* Filters */}
      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Grid container spacing={2} alignItems="center">
            <Grid item xs={12} md={5}>
              <TextField
                fullWidth
                placeholder="Search API keys..."
                value={searchTerm}
                onChange={handleSearch}
                InputProps={{
                  startAdornment: (
                    <InputAdornment position="start">
                      <Search />
                    </InputAdornment>
                  ),
                }}
              />
            </Grid>
            <Grid item xs={12} md={3}>
              <FormControl fullWidth>
                <InputLabel>Status</InputLabel>
                <Select
                  value={statusFilter}
                  label="Status"
                  onChange={(e) => setStatusFilter(e.target.value)}
                >
                  <MenuItem value="all">All</MenuItem>
                  <MenuItem value="active">Active</MenuItem>
                  <MenuItem value="inactive">Revoked</MenuItem>
                </Select>
              </FormControl>
            </Grid>
            <Grid item xs={12} md={4}>
              <Typography variant="body2" color="text.secondary">
                Total: {totalCount} API keys
              </Typography>
            </Grid>
          </Grid>
        </CardContent>
      </Card>

      {/* API Keys Table */}
      <Card>
        <TableContainer>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>Name</TableCell>
                <TableCell>Key Prefix</TableCell>
                <TableCell>Scopes</TableCell>
                <TableCell>Usage</TableCell>
                <TableCell>Rate Limits</TableCell>
                <TableCell>Status</TableCell>
                <TableCell>Last Used</TableCell>
                <TableCell align="right">Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {loading ? (
                <TableRow>
                  <TableCell colSpan={8} align="center" sx={{ py: 4 }}>
                    <CircularProgress />
                  </TableCell>
                </TableRow>
              ) : apiKeys.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={8} align="center" sx={{ py: 4 }}>
                    No API keys found
                  </TableCell>
                </TableRow>
              ) : (
                apiKeys.map((apiKey) => (
                  <TableRow key={apiKey.id} hover>
                    <TableCell>
                      <Box>
                        <Typography variant="subtitle2">{apiKey.name}</Typography>
                        {apiKey.description && (
                          <Typography variant="body2" color="text.secondary">
                            {apiKey.description}
                          </Typography>
                        )}
                        <Typography variant="caption" color="text.secondary">
                          by {apiKey.username}
                        </Typography>
                      </Box>
                    </TableCell>
                    <TableCell>
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                        <Typography variant="body2" fontFamily="monospace">
                          {apiKey.key_prefix}...
                        </Typography>
                        <Tooltip title="Copy Key ID">
                          <IconButton 
                            size="small" 
                            onClick={() => copyToClipboard(apiKey.key_id, 'Key ID', `key-${apiKey.id}`)}
                            sx={{
                              color: copiedItems.has(`key-${apiKey.id}`) ? 'success.main' : 'inherit',
                              '& .MuiSvgIcon-root': {
                                transition: 'color 0.3s ease'
                              }
                            }}
                          >
                            <Copy fontSize="small" />
                          </IconButton>
                        </Tooltip>
                      </Box>
                    </TableCell>
                    <TableCell>
                      <Box sx={{ display: 'flex', gap: 0.5, flexWrap: 'wrap', maxWidth: 200 }}>
                        {apiKey.scopes.slice(0, 3).map((scope) => (
                          <Chip key={scope} label={scope} size="small" variant="outlined" />
                        ))}
                        {apiKey.scopes.length > 3 && (
                          <Chip label={`+${apiKey.scopes.length - 3} more`} size="small" />
                        )}
                      </Box>
                    </TableCell>
                    <TableCell>
                      <Typography variant="body2">
                        {formatUsageCount(apiKey.usage_count)} requests
                      </Typography>
                    </TableCell>
                    <TableCell>
                      <Typography variant="body2">
                        {apiKey.rate_limit_per_hour}/hour
                      </Typography>
                      <Typography variant="caption" color="text.secondary">
                        {apiKey.rate_limit_per_day}/day
                      </Typography>
                    </TableCell>
                    <TableCell>
                      <Chip
                        label={getStatusLabel(apiKey)}
                        color={getStatusColor(apiKey)}
                        size="small"
                        icon={apiKey.is_active ? <CheckCircle /> : <X />}
                      />
                    </TableCell>
                    <TableCell>
                      <Typography variant="body2">
                        {apiKey.last_used_at 
                          ? new Date(apiKey.last_used_at).toLocaleDateString()
                          : 'Never'
                        }
                      </Typography>
                      {apiKey.last_used_ip && (
                        <Typography variant="caption" color="text.secondary">
                          {apiKey.last_used_ip}
                        </Typography>
                      )}
                    </TableCell>
                    <TableCell align="right">
                      <IconButton
                        onClick={(e) => setActionMenu({ anchorEl: e.currentTarget, apiKey })}
                      >
                        <MoreVertical />
                      </IconButton>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </TableContainer>
        
        <TablePagination
          rowsPerPageOptions={[10, 25, 50, 100]}
          component="div"
          count={totalCount}
          rowsPerPage={rowsPerPage}
          page={page}
          onPageChange={handleChangePage}
          onRowsPerPageChange={handleChangeRowsPerPage}
        />
      </Card>

      {/* Action Menu */}
      <Menu
        anchorEl={actionMenu.anchorEl}
        open={Boolean(actionMenu.anchorEl)}
        onClose={() => setActionMenu({ anchorEl: null, apiKey: null })}
      >
        <MenuItem onClick={() => handleViewAnalytics(actionMenu.apiKey)}>
          <BarChart3 sx={{ mr: 1 }} fontSize="small" />
          View Analytics
        </MenuItem>
        <Divider />
        <MenuItem 
          onClick={() => handleRevokeApiKey(actionMenu.apiKey)}
          sx={{ color: 'error.main' }}
          disabled={!actionMenu.apiKey?.is_active}
        >
          <Trash2 sx={{ mr: 1 }} fontSize="small" />
          Revoke Key
        </MenuItem>
      </Menu>

      {/* Create API Key Dialog */}
      <Dialog 
        open={createKeyDialogOpen} 
        onClose={() => setCreateKeyDialogOpen(false)}
        maxWidth="md"
        fullWidth
      >
        <DialogTitle>Create New API Key</DialogTitle>
        <DialogContent>
          <Grid container spacing={2} sx={{ mt: 1 }}>
            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                label="Name"
                value={keyForm.name}
                onChange={(e) => setKeyForm({ ...keyForm, name: e.target.value })}
                required
                helperText="Unique name for this API key"
              />
            </Grid>
            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                label="Expires At"
                type="datetime-local"
                value={keyForm.expiresAt}
                onChange={(e) => setKeyForm({ ...keyForm, expiresAt: e.target.value })}
                InputLabelProps={{ shrink: true }}
                helperText="Leave empty for no expiration"
              />
            </Grid>
            <Grid item xs={12}>
              <TextField
                fullWidth
                label="Description"
                multiline
                rows={2}
                value={keyForm.description}
                onChange={(e) => setKeyForm({ ...keyForm, description: e.target.value })}
                helperText="Optional description of the key's purpose"
              />
            </Grid>
            <Grid item xs={6}>
              <TextField
                fullWidth
                label="Rate Limit (per hour)"
                type="number"
                value={keyForm.rateLimitPerHour}
                onChange={(e) => setKeyForm({ ...keyForm, rateLimitPerHour: parseInt(e.target.value) })}
                inputProps={{ min: 1, max: 100000 }}
              />
            </Grid>
            <Grid item xs={6}>
              <TextField
                fullWidth
                label="Rate Limit (per day)"
                type="number"
                value={keyForm.rateLimitPerDay}
                onChange={(e) => setKeyForm({ ...keyForm, rateLimitPerDay: parseInt(e.target.value) })}
                inputProps={{ min: 1, max: 1000000 }}
              />
            </Grid>
            <Grid item xs={12}>
              <Typography variant="h6" gutterBottom>
                API Scopes
              </Typography>
              <Box sx={{ maxHeight: 300, overflow: 'auto', border: 1, borderColor: 'divider', borderRadius: 1, p: 1 }}>
                {Object.entries(groupedScopes).map(([resource, resourceScopes]) => (
                  <Box key={resource} sx={{ mb: 1 }}>
                    <Box 
                      sx={{ display: 'flex', alignItems: 'center', cursor: 'pointer', py: 0.5 }}
                      onClick={() => toggleScopeExpansion(resource)}
                    >
                      {expandedScopes[resource] ? <ChevronUp /> : <ChevronDown />}
                      <Typography variant="subtitle2" sx={{ ml: 1 }}>
                        {resource} ({resourceScopes.filter(s => keyForm.scopes.includes(s.name)).length}/{resourceScopes.length})
                      </Typography>
                    </Box>
                    <Collapse in={expandedScopes[resource]}>
                      <FormGroup sx={{ ml: 3 }}>
                        {resourceScopes.map((scope) => (
                          <FormControlLabel
                            key={scope.name}
                            control={
                              <Checkbox
                                checked={keyForm.scopes.includes(scope.name)}
                                onChange={() => handleToggleScope(scope.name)}
                                size="small"
                              />
                            }
                            label={
                              <Box>
                                <Typography variant="body2">
                                  {scope.display_name}
                                  {scope.is_sensitive && <Shield color="warning" sx={{ ml: 1, fontSize: 16 }} />}
                                </Typography>
                                <Typography variant="caption" color="text.secondary">
                                  {scope.description}
                                </Typography>
                              </Box>
                            }
                          />
                        ))}
                      </FormGroup>
                    </Collapse>
                  </Box>
                ))}
              </Box>
            </Grid>
          </Grid>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setCreateKeyDialogOpen(false)}>Cancel</Button>
          <Button 
            onClick={handleCreateApiKey}
            variant="contained"
            disabled={!keyForm.name || keyForm.scopes.length === 0}
          >
            Create API Key
          </Button>
        </DialogActions>
      </Dialog>

      {/* Analytics Dialog */}
      <Dialog 
        open={analyticsDialogOpen} 
        onClose={() => setAnalyticsDialogOpen(false)}
        maxWidth="lg"
        fullWidth
      >
        <DialogTitle>
          API Key Analytics - {selectedApiKey?.name}
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, mt: 1 }}>
            <FormControl size="small">
              <InputLabel>Time Period</InputLabel>
              <Select
                value={analyticsDays}
                label="Time Period"
                onChange={(e) => setAnalyticsDays(e.target.value)}
              >
                <MenuItem value={7}>Last 7 days</MenuItem>
                <MenuItem value={30}>Last 30 days</MenuItem>
                <MenuItem value={90}>Last 90 days</MenuItem>
              </Select>
            </FormControl>
          </Box>
        </DialogTitle>
        <DialogContent>
          {analyticsLoading ? (
            <Box sx={{ display: 'flex', justifyContent: 'center', py: 4 }}>
              <CircularProgress />
            </Box>
          ) : analytics ? (
            <Box>
              {/* Overview Cards */}
              <Grid container spacing={2} sx={{ mb: 3 }}>
                <Grid item xs={12} md={3}>
                  <Card>
                    <CardContent>
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                        <TrendingUp color="primary" />
                        <Box>
                          <Typography variant="h6">{analytics.total_requests}</Typography>
                          <Typography variant="body2" color="text.secondary">Total Requests</Typography>
                        </Box>
                      </Box>
                    </CardContent>
                  </Card>
                </Grid>
                <Grid item xs={12} md={3}>
                  <Card>
                    <CardContent>
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                        <CheckCircle color="success" />
                        <Box>
                          <Typography variant="h6">
                            {((analytics.successful_requests / analytics.total_requests) * 100).toFixed(1)}%
                          </Typography>
                          <Typography variant="body2" color="text.secondary">Success Rate</Typography>
                        </Box>
                      </Box>
                    </CardContent>
                  </Card>
                </Grid>
                <Grid item xs={12} md={3}>
                  <Card>
                    <CardContent>
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                        <Gauge color="info" />
                        <Box>
                          <Typography variant="h6">{analytics.avg_response_time.toFixed(0)}ms</Typography>
                          <Typography variant="body2" color="text.secondary">Avg Response Time</Typography>
                        </Box>
                      </Box>
                    </CardContent>
                  </Card>
                </Grid>
                <Grid item xs={12} md={3}>
                  <Card>
                    <CardContent>
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                        <Database color="secondary" />
                        <Box>
                          <Typography variant="h6">{formatDataSize(analytics.total_data_transfer)}</Typography>
                          <Typography variant="body2" color="text.secondary">Data Transfer</Typography>
                        </Box>
                      </Box>
                    </CardContent>
                  </Card>
                </Grid>
              </Grid>

              {/* Charts */}
              <Tabs value={analyticsTab} onChange={(e, v) => setAnalyticsTab(v)} sx={{ mb: 2 }}>
                <Tab label="Usage Timeline" />
                <Tab label="Top Endpoints" />
                <Tab label="Status Codes" />
              </Tabs>

              {analyticsTab === 0 && (
                <Card>
                  <CardContent>
                    <Typography variant="h6" gutterBottom>Hourly Usage</Typography>
                    <ResponsiveContainer width="100%" height={300}>
                      <LineChart data={analytics.hourly_usage}>
                        <CartesianGrid strokeDasharray="3 3" />
                        <XAxis dataKey="hour" />
                        <YAxis />
                        <RechartsTooltip />
                        <Line type="monotone" dataKey="request_count" stroke="#8884d8" />
                        <Line type="monotone" dataKey="error_count" stroke="#ff7300" />
                      </LineChart>
                    </ResponsiveContainer>
                  </CardContent>
                </Card>
              )}

              {analyticsTab === 1 && (
                <Card>
                  <CardContent>
                    <Typography variant="h6" gutterBottom>Top Endpoints</Typography>
                    <ResponsiveContainer width="100%" height={300}>
                      <BarChart data={analytics.top_endpoints}>
                        <CartesianGrid strokeDasharray="3 3" />
                        <XAxis dataKey="endpoint" />
                        <YAxis />
                        <RechartsTooltip />
                        <Bar dataKey="request_count" fill="#8884d8" />
                      </BarChart>
                    </ResponsiveContainer>
                  </CardContent>
                </Card>
              )}

              {analyticsTab === 2 && (
                <Card>
                  <CardContent>
                    <Typography variant="h6" gutterBottom>Response Status Distribution</Typography>
                    <Grid container spacing={2}>
                      <Grid item xs={12} md={6}>
                        <List>
                          <ListItem>
                            <ListItemIcon><CheckCircle color="success" /></ListItemIcon>
                            <ListItemText 
                              primary={`Successful (2xx): ${analytics.successful_requests}`}
                              secondary={`${((analytics.successful_requests / analytics.total_requests) * 100).toFixed(1)}%`}
                            />
                          </ListItem>
                          <ListItem>
                            <ListItemIcon><AlertCircle color="error" /></ListItemIcon>
                            <ListItemText 
                              primary={`Errors (4xx/5xx): ${analytics.error_requests}`}
                              secondary={`${((analytics.error_requests / analytics.total_requests) * 100).toFixed(1)}%`}
                            />
                          </ListItem>
                        </List>
                      </Grid>
                      <Grid item xs={12} md={6}>
                        <LinearProgress 
                          variant="determinate" 
                          value={(analytics.successful_requests / analytics.total_requests) * 100}
                          sx={{ height: 10, borderRadius: 5 }}
                        />
                        <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
                          Success Rate: {((analytics.successful_requests / analytics.total_requests) * 100).toFixed(1)}%
                        </Typography>
                      </Grid>
                    </Grid>
                  </CardContent>
                </Card>
              )}
            </Box>
          ) : (
            <Typography>No analytics data available</Typography>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setAnalyticsDialogOpen(false)}>Close</Button>
        </DialogActions>
      </Dialog>

      {/* Created Key Success Dialog */}
      <Dialog 
        open={Boolean(createdKey)} 
        onClose={() => setCreatedKey(null)}
        maxWidth="md"
        fullWidth
        disableEscapeKeyDown
        PaperProps={{
          sx: {
            borderRadius: 2,
            boxShadow: '0 10px 40px rgba(0, 0, 0, 0.15)',
          }
        }}
      >
        <DialogTitle>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, color: 'success.main' }}>
            <CheckCircle fontSize="large" />
            <Typography variant="h5" component="div">
              API Key Created Successfully!
            </Typography>
          </Box>
        </DialogTitle>
        <DialogContent>
          <Alert 
            severity="warning" 
            icon={<AlertTriangle />}
            sx={{ 
              mb: 3, 
              border: '2px solid',
              borderColor: 'warning.main',
              backgroundColor: 'warning.light',
              color: 'warning.contrastText'
            }}
          >
            <Typography variant="subtitle1" sx={{ fontWeight: 'bold', mb: 1 }}>
              ⚠️ IMPORTANT: Save this API key immediately!
            </Typography>
            <Typography variant="body2">
              This is the only time you'll be able to see the full API key. 
              Copy it now and store it securely.
            </Typography>
          </Alert>
          
          <Card sx={{ mb: 3, border: '2px solid', borderColor: 'primary.main' }}>
            <CardContent>
              <Typography variant="h6" gutterBottom sx={{ color: 'primary.main' }}>
                🔑 Your New API Key
              </Typography>
              
              <Box sx={{ mb: 2 }}>
                <Typography variant="subtitle2" gutterBottom>
                  Key Name: <strong>{createdKey?.name}</strong>
                </Typography>
                {createdKey?.description && (
                  <Typography variant="body2" color="text.secondary" gutterBottom>
                    Description: {createdKey.description}
                  </Typography>
                )}
              </Box>

              <Box sx={{ mb: 2 }}>
                <Typography variant="subtitle2" gutterBottom>
                  API Key:
                </Typography>
                <Box sx={{ 
                  display: 'flex', 
                  alignItems: 'center', 
                  gap: 1,
                  p: 2,
                  backgroundColor: 'grey.50',
                  borderRadius: 1,
                  border: '1px solid',
                  borderColor: 'grey.300'
                }}>
                  <TextField
                    fullWidth
                    value={createdKey?.key_secret || ''}
                    type={showKeySecret ? 'text' : 'password'}
                    variant="outlined"
                    size="small"
                    InputProps={{
                      readOnly: true,
                      style: { 
                        fontFamily: 'monospace', 
                        fontSize: '14px',
                        backgroundColor: 'white'
                      },
                      endAdornment: (
                        <InputAdornment position="end">
                          <Tooltip title={showKeySecret ? "Hide key" : "Show key"}>
                            <IconButton onClick={() => setShowKeySecret(!showKeySecret)} size="small">
                              {showKeySecret ? <EyeOff /> : <Eye />}
                            </IconButton>
                          </Tooltip>
                        </InputAdornment>
                      ),
                    }}
                  />
                  <Tooltip title="Copy API key">
                    <Button
                      variant="contained"
                      color="primary"
                      startIcon={<Copy />}
                      onClick={() => copyToClipboard(createdKey?.key_secret || '', 'API key', 'created-key')}
                      sx={{ 
                        minWidth: 'auto',
                        px: 2,
                        py: 1,
                        '&:hover': {
                          transform: 'scale(1.05)',
                        },
                        transition: 'transform 0.2s'
                      }}
                    >
                      Copy
                    </Button>
                  </Tooltip>
                </Box>
              </Box>

              {/* Quick Copy Buttons */}
              <Grid container spacing={2} sx={{ mt: 2 }}>
                <Grid item xs={12} md={6}>
                  <Paper sx={{ p: 2, backgroundColor: 'grey.50' }}>
                    <Typography variant="subtitle2" gutterBottom>
                      🚀 Quick Copy - cURL Example:
                    </Typography>
                    <Typography 
                      variant="body2" 
                      fontFamily="monospace" 
                      sx={{ 
                        wordBreak: 'break-all',
                        backgroundColor: 'white',
                        p: 1,
                        borderRadius: 1,
                        border: '1px solid',
                        borderColor: 'grey.300',
                        mb: 1
                      }}
                    >
                      curl -H "Authorization: Bearer {createdKey?.key_secret || ''}" https://your-api.com/api/endpoint
                    </Typography>
                    <Button
                      size="small"
                      variant="outlined"
                      startIcon={<Copy />}
                      onClick={() => copyToClipboard(
                        `curl -H "Authorization: Bearer ${createdKey?.key_secret || ''}" https://your-api.com/api/endpoint`,
                        'cURL example',
                        'curl-example'
                      )}
                      fullWidth
                    >
                      Copy cURL Example
                    </Button>
                  </Paper>
                </Grid>
                <Grid item xs={12} md={6}>
                  <Paper sx={{ p: 2, backgroundColor: 'grey.50' }}>
                    <Typography variant="subtitle2" gutterBottom>
                      📋 Quick Copy - Authorization Header:
                    </Typography>
                    <Typography 
                      variant="body2" 
                      fontFamily="monospace"
                      sx={{ 
                        wordBreak: 'break-all',
                        backgroundColor: 'white',
                        p: 1,
                        borderRadius: 1,
                        border: '1px solid',
                        borderColor: 'grey.300',
                        mb: 1
                      }}
                    >
                      Authorization: Bearer {createdKey?.key_secret || ''}
                    </Typography>
                    <Button
                      size="small"
                      variant="outlined"
                      startIcon={<Copy />}
                      onClick={() => copyToClipboard(
                        `Authorization: Bearer ${createdKey?.key_secret || ''}`,
                        'Authorization header',
                        'auth-header'
                      )}
                      fullWidth
                    >
                      Copy Auth Header
                    </Button>
                  </Paper>
                </Grid>
              </Grid>

              {/* Key Details */}
              <Box sx={{ mt: 3, p: 2, backgroundColor: 'info.light', borderRadius: 1 }}>
                <Typography variant="subtitle2" gutterBottom sx={{ color: 'info.contrastText' }}>
                  📊 Key Details:
                </Typography>
                <Grid container spacing={2}>
                  <Grid item xs={6}>
                    <Typography variant="body2" sx={{ color: 'info.contrastText' }}>
                      <strong>Key ID:</strong> {createdKey?.key_id}
                    </Typography>
                  </Grid>
                  <Grid item xs={6}>
                    <Typography variant="body2" sx={{ color: 'info.contrastText' }}>
                      <strong>Created:</strong> {new Date().toLocaleString()}
                    </Typography>
                  </Grid>
                  {createdKey?.expires_at && (
                    <Grid item xs={12}>
                      <Typography variant="body2" sx={{ color: 'info.contrastText' }}>
                        <strong>Expires:</strong> {new Date(createdKey.expires_at).toLocaleString()}
                      </Typography>
                    </Grid>
                  )}
                  <Grid item xs={6}>
                    <Typography variant="body2" sx={{ color: 'info.contrastText' }}>
                      <strong>Rate Limit:</strong> {createdKey?.rate_limit_per_hour}/hour
                    </Typography>
                  </Grid>
                  <Grid item xs={6}>
                    <Typography variant="body2" sx={{ color: 'info.contrastText' }}>
                      <strong>Daily Limit:</strong> {createdKey?.rate_limit_per_day}/day
                    </Typography>
                  </Grid>
                </Grid>
              </Box>
            </CardContent>
          </Card>

          <Alert severity="info" sx={{ mb: 2 }}>
            <Typography variant="body2">
              💡 <strong>Tip:</strong> Store this key in a secure password manager or environment variable. 
              Never commit it to version control or share it in plain text.
            </Typography>
          </Alert>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 3 }}>
          <Button 
            onClick={() => copyToClipboard(createdKey?.key_secret || '', 'API key', 'final-copy')} 
            variant="outlined"
            size="large"
            startIcon={<Copy />}
            sx={{ mr: 'auto' }}
          >
            Copy Key Again
          </Button>
          <Button 
            onClick={() => setCreatedKey(null)} 
            variant="contained"
            size="large"
            color="success"
            sx={{ 
              minWidth: 200,
              '&:hover': {
                transform: 'scale(1.02)',
              },
              transition: 'transform 0.2s'
            }}
          >
            ✅ I've Saved the Key Securely
          </Button>
        </DialogActions>
      </Dialog>

      {/* Toast Notification */}
      <Snackbar
        open={snackbar.open}
        autoHideDuration={3000}
        onClose={() => setSnackbar({ ...snackbar, open: false })}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
      >
        <Alert 
          onClose={() => setSnackbar({ ...snackbar, open: false })} 
          severity={snackbar.severity}
          variant="filled"
          sx={{ 
            '& .MuiAlert-message': { 
              fontWeight: 'medium' 
            }
          }}
        >
          {snackbar.message}
        </Alert>
      </Snackbar>
    </Box>
  );
};

export default APIKeyManagementDashboard;