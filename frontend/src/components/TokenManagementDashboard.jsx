import React, { useState, useEffect } from 'react';
import { 
  Card, 
  CardContent, 
  CardHeader, 
  Typography, 
  Table, 
  TableBody, 
  TableCell, 
  TableContainer, 
  TableHead, 
  TableRow, 
  Paper, 
  Button, 
  IconButton, 
  Chip, 
  Dialog, 
  DialogTitle, 
  DialogContent, 
  DialogActions, 
  TextField, 
  Alert,
  Snackbar,
  Box,
  Tooltip,
  CircularProgress
} from '@mui/material';
import {
  Delete as DeleteIcon,
  Visibility as VisibilityIcon,
  Security as SecurityIcon,
  AccessTime as AccessTimeIcon,
  Info as InfoIcon
} from '@mui/icons-material';

const TokenManagementDashboard = () => {
  const [tokens, setTokens] = useState([]);
  const [tokenStats, setTokenStats] = useState({});
  const [loading, setLoading] = useState(true);
  const [selectedToken, setSelectedToken] = useState(null);
  const [introspectionResult, setIntrospectionResult] = useState(null);
  const [openDialog, setOpenDialog] = useState(false);
  const [openIntrospectionDialog, setOpenIntrospectionDialog] = useState(false);
  const [tokenToRevoke, setTokenToRevoke] = useState(null);
  const [testToken, setTestToken] = useState('');
  const [snackbar, setSnackbar] = useState({ open: false, message: '', severity: 'info' });

  useEffect(() => {
    fetchTokens();
    fetchTokenStats();
  }, []);

  const fetchTokens = async () => {
    try {
      const response = await fetch('/api/admin/tokens');
      if (response.ok) {
        const data = await response.json();
        setTokens(data.tokens || []);
      }
    } catch (error) {
      showSnackbar('Failed to fetch tokens', 'error');
    } finally {
      setLoading(false);
    }
  };

  const fetchTokenStats = async () => {
    try {
      const response = await fetch('/oauth/token-stats');
      if (response.ok) {
        const data = await response.json();
        setTokenStats(data.data || {});
      }
    } catch (error) {
      showSnackbar('Failed to fetch token statistics', 'error');
    }
  };

  const handleRevokeToken = async (token) => {
    try {
      const formData = new FormData();
      formData.append('token', token.key_id || token.token);
      
      const response = await fetch('/oauth/revoke', {
        method: 'POST',
        body: formData
      });

      if (response.ok) {
        showSnackbar('Token revoked successfully', 'success');
        fetchTokens();
        fetchTokenStats();
      } else {
        showSnackbar('Failed to revoke token', 'error');
      }
    } catch (error) {
      showSnackbar('Error revoking token', 'error');
    }
    setOpenDialog(false);
    setTokenToRevoke(null);
  };

  const handleIntrospectToken = async () => {
    if (!testToken) {
      showSnackbar('Please enter a token to introspect', 'warning');
      return;
    }

    try {
      const formData = new FormData();
      formData.append('token', testToken);
      formData.append('client_id', 'admin-client');
      formData.append('client_secret', 'admin-secret');
      
      const response = await fetch('/oauth/introspect', {
        method: 'POST',
        body: formData
      });

      if (response.ok) {
        const result = await response.json();
        setIntrospectionResult(result);
      } else {
        showSnackbar('Failed to introspect token', 'error');
      }
    } catch (error) {
      showSnackbar('Error introspecting token', 'error');
    }
  };

  const handleCleanupCache = async () => {
    try {
      const response = await fetch('/oauth/cleanup-cache', {
        method: 'POST'
      });

      if (response.ok) {
        showSnackbar('Cache cleaned up successfully', 'success');
        fetchTokenStats();
      } else {
        showSnackbar('Failed to cleanup cache', 'error');
      }
    } catch (error) {
      showSnackbar('Error cleaning up cache', 'error');
    }
  };

  const showSnackbar = (message, severity = 'info') => {
    setSnackbar({ open: true, message, severity });
  };

  const formatDate = (dateString) => {
    if (!dateString) return 'Never';
    return new Date(dateString).toLocaleString();
  };

  const getTokenStatus = (token) => {
    if (!token.is_active) return { label: 'Inactive', color: 'error' };
    if (token.expires_at && new Date(token.expires_at) < new Date()) {
      return { label: 'Expired', color: 'error' };
    }
    return { label: 'Active', color: 'success' };
  };

  const renderTokenType = (token) => {
    if (token.key_id) return 'API Key';
    if (token.client_id) return 'OAuth Token';
    return 'Unknown';
  };

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight="400px">
        <CircularProgress />
      </Box>
    );
  }

  return (
    <div className="token-management-dashboard">
      {/* Token Statistics */}
      <Box mb={3}>
        <Typography variant="h4" gutterBottom>
          Token Management Dashboard
        </Typography>
        
        <Box display="flex" gap={2} mb={3}>
          <Card>
            <CardContent>
              <Typography variant="h6" color="textSecondary">
                Total Cached Tokens
              </Typography>
              <Typography variant="h4">
                {tokenStats.total_cached || 0}
              </Typography>
            </CardContent>
          </Card>
          
          <Card>
            <CardContent>
              <Typography variant="h6" color="textSecondary">
                Active Tokens
              </Typography>
              <Typography variant="h4" color="success.main">
                {tokenStats.active_tokens || 0}
              </Typography>
            </CardContent>
          </Card>
          
          <Card>
            <CardContent>
              <Typography variant="h6" color="textSecondary">
                API Keys
              </Typography>
              <Typography variant="h4" color="info.main">
                {tokenStats.api_keys || 0}
              </Typography>
            </CardContent>
          </Card>
          
          <Card>
            <CardContent>
              <Typography variant="h6" color="textSecondary">
                Expired Tokens
              </Typography>
              <Typography variant="h4" color="error.main">
                {tokenStats.expired_tokens || 0}
              </Typography>
            </CardContent>
          </Card>
        </Box>

        <Box display="flex" gap={2} mb={3}>
          <Button 
            variant="outlined" 
            onClick={handleCleanupCache}
            startIcon={<DeleteIcon />}
          >
            Cleanup Expired Cache
          </Button>
          
          <Button 
            variant="outlined" 
            onClick={() => setOpenIntrospectionDialog(true)}
            startIcon={<SecurityIcon />}
          >
            Test Token Introspection
          </Button>
        </Box>
      </Box>

      {/* Token Table */}
      <Card>
        <CardHeader title="Active Tokens" />
        <CardContent>
          <TableContainer component={Paper}>
            <Table>
              <TableHead>
                <TableRow>
                  <TableCell>ID</TableCell>
                  <TableCell>Type</TableCell>
                  <TableCell>Name/Client</TableCell>
                  <TableCell>Status</TableCell>
                  <TableCell>Created</TableCell>
                  <TableCell>Expires</TableCell>
                  <TableCell>Last Used</TableCell>
                  <TableCell>Actions</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {tokens.map((token) => {
                  const status = getTokenStatus(token);
                  return (
                    <TableRow key={token.id}>
                      <TableCell>{token.id}</TableCell>
                      <TableCell>
                        <Chip 
                          label={renderTokenType(token)} 
                          size="small" 
                          color="primary"
                        />
                      </TableCell>
                      <TableCell>
                        {token.name || token.client_id || 'N/A'}
                      </TableCell>
                      <TableCell>
                        <Chip 
                          label={status.label} 
                          color={status.color} 
                          size="small" 
                        />
                      </TableCell>
                      <TableCell>{formatDate(token.created_at)}</TableCell>
                      <TableCell>{formatDate(token.expires_at)}</TableCell>
                      <TableCell>{formatDate(token.last_used_at)}</TableCell>
                      <TableCell>
                        <Tooltip title="View Details">
                          <IconButton 
                            size="small" 
                            onClick={() => setSelectedToken(token)}
                          >
                            <VisibilityIcon />
                          </IconButton>
                        </Tooltip>
                        
                        <Tooltip title="Revoke Token">
                          <IconButton 
                            size="small" 
                            color="error"
                            onClick={() => {
                              setTokenToRevoke(token);
                              setOpenDialog(true);
                            }}
                          >
                            <DeleteIcon />
                          </IconButton>
                        </Tooltip>
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          </TableContainer>
        </CardContent>
      </Card>

      {/* Revocation Confirmation Dialog */}
      <Dialog open={openDialog} onClose={() => setOpenDialog(false)}>
        <DialogTitle>Confirm Token Revocation</DialogTitle>
        <DialogContent>
          <Typography>
            Are you sure you want to revoke this token? This action cannot be undone.
          </Typography>
          {tokenToRevoke && (
            <Box mt={2}>
              <Typography variant="body2" color="textSecondary">
                Token: {tokenToRevoke.name || tokenToRevoke.client_id || tokenToRevoke.id}
              </Typography>
              <Typography variant="body2" color="textSecondary">
                Type: {renderTokenType(tokenToRevoke)}
              </Typography>
            </Box>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setOpenDialog(false)}>Cancel</Button>
          <Button 
            onClick={() => handleRevokeToken(tokenToRevoke)} 
            color="error" 
            variant="contained"
          >
            Revoke Token
          </Button>
        </DialogActions>
      </Dialog>

      {/* Token Introspection Dialog */}
      <Dialog 
        open={openIntrospectionDialog} 
        onClose={() => setOpenIntrospectionDialog(false)}
        maxWidth="md"
        fullWidth
      >
        <DialogTitle>Test Token Introspection</DialogTitle>
        <DialogContent>
          <TextField
            fullWidth
            label="Token to Introspect"
            value={testToken}
            onChange={(e) => setTestToken(e.target.value)}
            multiline
            rows={3}
            margin="normal"
            placeholder="Enter a token (API key or OAuth token)"
          />
          
          {introspectionResult && (
            <Box mt={2}>
              <Typography variant="h6" gutterBottom>
                Introspection Result:
              </Typography>
              <Paper elevation={1} style={{ padding: '16px' }}>
                <pre style={{ 
                  whiteSpace: 'pre-wrap', 
                  fontSize: '12px',
                  margin: 0 
                }}>
                  {JSON.stringify(introspectionResult, null, 2)}
                </pre>
              </Paper>
            </Box>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setOpenIntrospectionDialog(false)}>Close</Button>
          <Button 
            onClick={handleIntrospectToken} 
            variant="contained"
            startIcon={<SecurityIcon />}
          >
            Introspect Token
          </Button>
        </DialogActions>
      </Dialog>

      {/* Token Details Dialog */}
      {selectedToken && (
        <Dialog 
          open={!!selectedToken} 
          onClose={() => setSelectedToken(null)}
          maxWidth="md"
          fullWidth
        >
          <DialogTitle>Token Details</DialogTitle>
          <DialogContent>
            <Box>
              <Typography variant="h6" gutterBottom>
                {selectedToken.name || 'Token Information'}
              </Typography>
              
              <Table size="small">
                <TableBody>
                  <TableRow>
                    <TableCell><strong>ID</strong></TableCell>
                    <TableCell>{selectedToken.id}</TableCell>
                  </TableRow>
                  <TableRow>
                    <TableCell><strong>Type</strong></TableCell>
                    <TableCell>{renderTokenType(selectedToken)}</TableCell>
                  </TableRow>
                  <TableRow>
                    <TableCell><strong>Status</strong></TableCell>
                    <TableCell>
                      <Chip 
                        label={getTokenStatus(selectedToken).label} 
                        color={getTokenStatus(selectedToken).color} 
                        size="small" 
                      />
                    </TableCell>
                  </TableRow>
                  <TableRow>
                    <TableCell><strong>Created</strong></TableCell>
                    <TableCell>{formatDate(selectedToken.created_at)}</TableCell>
                  </TableRow>
                  <TableRow>
                    <TableCell><strong>Expires</strong></TableCell>
                    <TableCell>{formatDate(selectedToken.expires_at)}</TableCell>
                  </TableRow>
                  <TableRow>
                    <TableCell><strong>Last Used</strong></TableCell>
                    <TableCell>{formatDate(selectedToken.last_used_at)}</TableCell>
                  </TableRow>
                  {selectedToken.scopes && (
                    <TableRow>
                      <TableCell><strong>Scopes</strong></TableCell>
                      <TableCell>
                        {selectedToken.scopes.map((scope, index) => (
                          <Chip key={index} label={scope} size="small" style={{ margin: '2px' }} />
                        ))}
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </Box>
          </DialogContent>
          <DialogActions>
            <Button onClick={() => setSelectedToken(null)}>Close</Button>
          </DialogActions>
        </Dialog>
      )}

      {/* Snackbar for notifications */}
      <Snackbar
        open={snackbar.open}
        autoHideDuration={6000}
        onClose={() => setSnackbar({ ...snackbar, open: false })}
      >
        <Alert 
          onClose={() => setSnackbar({ ...snackbar, open: false })} 
          severity={snackbar.severity}
        >
          {snackbar.message}
        </Alert>
      </Snackbar>
    </div>
  );
};

export default TokenManagementDashboard;