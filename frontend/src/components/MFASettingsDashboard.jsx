import React, { useState, useEffect } from 'react';
import {
  Box,
  Container,
  Typography,
  Paper,
  Card,
  CardContent,
  CardHeader,
  Switch,
  FormControlLabel,
  Button,
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
  TextField,
  Chip,
  Grid,
  LinearProgress,
  Avatar,
  Step,
  Stepper,
  StepLabel,
  StepContent,
  Badge,
  Tooltip,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow
} from '@mui/material';
import {
  Shield,
  Smartphone,
  Key,
  Plus,
  Trash2,
  Edit,
  QrCode,
  Archive,
  AlertTriangle,
  CheckCircle,
  AlertCircle,
  History,
  ShieldCheck,
  ChevronDown,
  Eye,
  EyeOff,
  Download,
  Printer
} from 'lucide-react';
import { QRCodeSVG } from 'qrcode.react';

const MFASettingsDashboard = ({ userId = 1 }) => {
  const [mfaStatus, setMfaStatus] = useState(null);
  const [mfaMethods, setMfaMethods] = useState([]);
  const [webauthnCredentials, setWebauthnCredentials] = useState([]);
  const [mfaAttempts, setMfaAttempts] = useState([]);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [errors, setErrors] = useState({});
  const [success, setSuccess] = useState('');

  // Setup states
  const [setupOpen, setSetupOpen] = useState(false);
  const [setupMethod, setSetupMethod] = useState('');
  const [setupData, setSetupData] = useState(null);
  const [setupStep, setSetupStep] = useState(0);
  const [verificationCode, setVerificationCode] = useState('');
  const [showBackupCodes, setShowBackupCodes] = useState(false);
  const [backupCodes, setBackupCodes] = useState([]);

  // Dialog states
  const [disableConfirmOpen, setDisableConfirmOpen] = useState(false);
  const [attemptsOpen, setAttemptsOpen] = useState(false);
  const [deviceNameDialog, setDeviceNameDialog] = useState({ open: false, deviceId: null });

  useEffect(() => {
    fetchMFAStatus();
    fetchMFAMethods();
    fetchWebAuthnCredentials();
    fetchMFAAttempts();
  }, [userId]);

  const fetchMFAStatus = async () => {
    setLoading(true);
    try {
      const response = await fetch(`/api/users/${userId}/mfa`);
      if (!response.ok) throw new Error('Failed to fetch MFA status');
      
      const data = await response.json();
      setMfaStatus(data);
    } catch (error) {
      setErrors({ general: 'Failed to fetch MFA status: ' + error.message });
    } finally {
      setLoading(false);
    }
  };

  const fetchMFAMethods = async () => {
    try {
      const response = await fetch('/api/mfa/methods');
      if (!response.ok) throw new Error('Failed to fetch MFA methods');
      
      const data = await response.json();
      setMfaMethods(data.methods || []);
    } catch (error) {
      setErrors({ methods: 'Failed to fetch MFA methods: ' + error.message });
    }
  };

  const fetchWebAuthnCredentials = async () => {
    try {
      const response = await fetch(`/api/users/${userId}/mfa/webauthn/credentials`);
      if (!response.ok) throw new Error('Failed to fetch WebAuthn credentials');
      
      const data = await response.json();
      setWebauthnCredentials(data.credentials || []);
    } catch (error) {
      setErrors({ webauthn: 'Failed to fetch WebAuthn credentials: ' + error.message });
    }
  };

  const fetchMFAAttempts = async () => {
    try {
      const response = await fetch(`/api/users/${userId}/mfa/attempts?limit=10`);
      if (!response.ok) throw new Error('Failed to fetch MFA attempts');
      
      const data = await response.json();
      setMfaAttempts(data.attempts || []);
    } catch (error) {
      setErrors({ attempts: 'Failed to fetch MFA attempts: ' + error.message });
    }
  };

  const setupMFA = async (method) => {
    setSaving(true);
    try {
      const response = await fetch(`/api/users/${userId}/mfa/setup`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ method })
      });

      if (!response.ok) throw new Error('Failed to setup MFA');
      
      const data = await response.json();
      setSetupData(data);
      setSetupMethod(method);
      setSetupOpen(true);
      setSetupStep(1);

      if (method === 'backup_codes') {
        setBackupCodes(data.backup_codes || []);
        setShowBackupCodes(true);
      }
    } catch (error) {
      setErrors({ setup: 'Failed to setup MFA: ' + error.message });
    } finally {
      setSaving(false);
    }
  };

  const verifyMFASetup = async () => {
    setSaving(true);
    try {
      const response = await fetch(`/api/users/${userId}/mfa/verify-setup`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ 
          method: setupMethod, 
          verification_code: verificationCode 
        })
      });

      if (!response.ok) throw new Error('Failed to verify MFA setup');
      
      const data = await response.json();
      setSuccess('MFA setup completed successfully!');
      setSetupOpen(false);
      setVerificationCode('');
      setSetupStep(0);
      fetchMFAStatus();
    } catch (error) {
      setErrors({ verify: 'Failed to verify MFA setup: ' + error.message });
    } finally {
      setSaving(false);
    }
  };

  const disableMFA = async () => {
    setSaving(true);
    try {
      const response = await fetch(`/api/users/${userId}/mfa/disable`, {
        method: 'DELETE'
      });

      if (!response.ok) throw new Error('Failed to disable MFA');
      
      setSuccess('MFA disabled successfully');
      setDisableConfirmOpen(false);
      fetchMFAStatus();
    } catch (error) {
      setErrors({ disable: 'Failed to disable MFA: ' + error.message });
    } finally {
      setSaving(false);
    }
  };

  const downloadBackupCodes = () => {
    const codes = backupCodes.join('\n');
    const blob = new Blob([codes], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'mfa-backup-codes.txt';
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  const printBackupCodes = () => {
    const printWindow = window.open('', '_blank');
    printWindow.document.write(`
      <html>
        <head>
          <title>MFA Backup Codes</title>
          <style>
            body { font-family: Arial, sans-serif; margin: 20px; }
            h1 { color: #333; }
            .code { font-family: monospace; font-size: 14px; margin: 5px 0; }
            .warning { color: #f44336; margin: 20px 0; }
          </style>
        </head>
        <body>
          <h1>MFA Backup Codes</h1>
          <p class="warning">⚠️ Store these codes in a safe place. Each code can only be used once.</p>
          ${backupCodes.map(code => `<div class="code">${code}</div>`).join('')}
        </body>
      </html>
    `);
    printWindow.document.close();
    printWindow.print();
  };

  const getMethodIcon = (method) => {
    switch (method) {
      case 'totp': return <Smartphone />;
      case 'webauthn': return <Key />;
      case 'backup_codes': return <Archive />;
      default: return <Shield />;
    }
  };

  const getStatusIcon = (enabled) => {
    return enabled ? 
      <CheckCircle color="success" /> : 
      <AlertCircle color="error" />;
  };

  const renderMFAOverview = () => {
    if (!mfaStatus) return null;

    return (
      <Card sx={{ mb: 3 }}>
        <CardHeader 
          title="Multi-Factor Authentication Status"
          avatar={<Shield color={mfaStatus.mfa_enabled ? 'success' : 'disabled'} />}
        />
        <CardContent>
          <Grid container spacing={3}>
            <Grid item xs={12} md={6}>
              <Box display="flex" alignItems="center" gap={2} mb={2}>
                {getStatusIcon(mfaStatus.mfa_enabled)}
                <Typography variant="h6" color={mfaStatus.mfa_enabled ? 'success.main' : 'error.main'}>
                  MFA {mfaStatus.mfa_enabled ? 'Enabled' : 'Disabled'}
                </Typography>
              </Box>
              
              {mfaStatus.mfa_enabled && (
                <Box>
                  <Typography variant="body2" color="text.secondary" gutterBottom>
                    Preferred Method: {mfaStatus.preferred_method || 'None'}
                  </Typography>
                  {mfaStatus.last_mfa_setup && (
                    <Typography variant="body2" color="text.secondary">
                      Setup: {new Date(mfaStatus.last_mfa_setup).toLocaleDateString()}
                    </Typography>
                  )}
                  {mfaStatus.last_mfa_used && (
                    <Typography variant="body2" color="text.secondary">
                      Last Used: {new Date(mfaStatus.last_mfa_used).toLocaleDateString()}
                    </Typography>
                  )}
                </Box>
              )}
            </Grid>
            
            <Grid item xs={12} md={6}>
              <Box display="flex" flexDirection="column" gap={1}>
                <Box display="flex" justifyContent="space-between">
                  <Typography variant="body2">TOTP Configured:</Typography>
                  {getStatusIcon(mfaStatus.totp_configured)}
                </Box>
                <Box display="flex" justifyContent="space-between">
                  <Typography variant="body2">WebAuthn Devices:</Typography>
                  <Chip size="small" label={mfaStatus.webauthn_devices} />
                </Box>
                <Box display="flex" justifyContent="space-between">
                  <Typography variant="body2">Backup Codes:</Typography>
                  <Chip size="small" label={mfaStatus.unused_backup_codes} />
                </Box>
                <Box display="flex" justifyContent="space-between">
                  <Typography variant="body2">Used Codes:</Typography>
                  <Chip size="small" label={mfaStatus.backup_codes_used_count} />
                </Box>
              </Box>
            </Grid>
          </Grid>

          <Divider sx={{ my: 2 }} />

          <Box display="flex" gap={2} flexWrap="wrap">
            {!mfaStatus.mfa_enabled ? (
              <Button
                variant="contained"
                startIcon={<Shield />}
                onClick={() => setSetupOpen(true)}
              >
                Enable MFA
              </Button>
            ) : (
              <>
                <Button
                  variant="outlined"
                  startIcon={<Plus />}
                  onClick={() => setSetupOpen(true)}
                >
                  Add Method
                </Button>
                <Button
                  variant="outlined"
                  color="error"
                  startIcon={<Trash2 />}
                  onClick={() => setDisableConfirmOpen(true)}
                >
                  Disable MFA
                </Button>
                <Button
                  variant="outlined"
                  startIcon={<History />}
                  onClick={() => setAttemptsOpen(true)}
                >
                  View History
                </Button>
              </>
            )}
          </Box>
        </CardContent>
      </Card>
    );
  };

  const renderSetupStepper = () => {
    const steps = [
      'Choose Method',
      'Configure',
      'Verify'
    ];

    return (
      <Stepper activeStep={setupStep} orientation="vertical">
        <Step>
          <StepLabel>Choose MFA Method</StepLabel>
          <StepContent>
            <Grid container spacing={2}>
              {mfaMethods.map((method) => (
                <Grid item xs={12} sm={6} key={method.name}>
                  <Card 
                    sx={{ cursor: 'pointer', '&:hover': { boxShadow: 3 } }}
                    onClick={() => {
                      setupMFA(method.name);
                    }}
                  >
                    <CardContent>
                      <Box display="flex" alignItems="center" gap={2}>
                        {getMethodIcon(method.name)}
                        <Box>
                          <Typography variant="h6">{method.display_name}</Typography>
                          <Typography variant="body2" color="text.secondary">
                            {method.name === 'totp' && 'Use an authenticator app'}
                            {method.name === 'webauthn' && 'Use a security key or biometric'}
                            {method.name === 'backup_codes' && 'Generate backup codes'}
                          </Typography>
                        </Box>
                      </Box>
                    </CardContent>
                  </Card>
                </Grid>
              ))}
            </Grid>
          </StepContent>
        </Step>

        <Step>
          <StepLabel>Configure Method</StepLabel>
          <StepContent>
            {setupMethod === 'totp' && setupData && (
              <Box>
                <Typography variant="h6" gutterBottom>Scan QR Code</Typography>
                <Typography variant="body2" gutterBottom>
                  Scan this QR code with your authenticator app:
                </Typography>
                <Box display="flex" justifyContent="center" my={3}>
                  <Paper elevation={3} sx={{ p: 2 }}>
                    <QRCodeSVG value={setupData.qr_code_url} size={200} />
                  </Paper>
                </Box>
                <Typography variant="body2" gutterBottom>
                  Or manually enter this secret: 
                </Typography>
                <TextField
                  fullWidth
                  value={setupData.secret}
                  InputProps={{ readOnly: true }}
                  sx={{ mb: 2 }}
                />
                <Button onClick={() => setSetupStep(2)} variant="contained">
                  Continue to Verification
                </Button>
              </Box>
            )}

            {setupMethod === 'backup_codes' && backupCodes.length > 0 && (
              <Box>
                <Typography variant="h6" gutterBottom>Backup Codes Generated</Typography>
                <Alert severity="warning" sx={{ mb: 2 }}>
                  Save these backup codes in a safe place. Each code can only be used once.
                </Alert>
                
                <Paper sx={{ p: 2, mb: 2, maxHeight: 200, overflow: 'auto' }}>
                  {backupCodes.map((code, index) => (
                    <Typography key={index} variant="body2" sx={{ fontFamily: 'monospace', mb: 1 }}>
                      {code}
                    </Typography>
                  ))}
                </Paper>

                <Box display="flex" gap={2} mb={2}>
                  <Button
                    variant="outlined"
                    startIcon={<Download />}
                    onClick={downloadBackupCodes}
                  >
                    Download
                  </Button>
                  <Button
                    variant="outlined"
                    startIcon={<Printer />}
                    onClick={printBackupCodes}
                  >
                    Print
                  </Button>
                </Box>

                <Button onClick={() => { setSetupOpen(false); fetchMFAStatus(); }} variant="contained">
                  Complete Setup
                </Button>
              </Box>
            )}
          </StepContent>
        </Step>

        <Step>
          <StepLabel>Verify Setup</StepLabel>
          <StepContent>
            {setupMethod === 'totp' && (
              <Box>
                <Typography variant="h6" gutterBottom>Enter Verification Code</Typography>
                <Typography variant="body2" gutterBottom>
                  Enter the 6-digit code from your authenticator app:
                </Typography>
                <TextField
                  fullWidth
                  label="Verification Code"
                  value={verificationCode}
                  onChange={(e) => setVerificationCode(e.target.value)}
                  inputProps={{ maxLength: 6, pattern: '[0-9]*' }}
                  sx={{ mb: 2 }}
                />
                <Box display="flex" gap={2}>
                  <Button
                    variant="contained"
                    onClick={verifyMFASetup}
                    disabled={verificationCode.length !== 6 || saving}
                  >
                    Verify & Enable
                  </Button>
                  <Button onClick={() => setSetupStep(1)}>
                    Back
                  </Button>
                </Box>
              </Box>
            )}
          </StepContent>
        </Step>
      </Stepper>
    );
  };

  const renderWebAuthnDevices = () => (
    <Accordion>
      <AccordionSummary expandIcon={<ChevronDown />}>
        <Box display="flex" alignItems="center" gap={2}>
          <Key />
          <Typography variant="h6">Security Keys & Devices</Typography>
          <Chip size="small" label={webauthnCredentials.length} />
        </Box>
      </AccordionSummary>
      <AccordionDetails>
        {webauthnCredentials.length === 0 ? (
          <Typography color="text.secondary">No security keys registered</Typography>
        ) : (
          <List>
            {webauthnCredentials.map((credential) => (
              <ListItem key={credential.id}>
                <ListItemText
                  primary={credential.device_name || `Device ${credential.id}`}
                  secondary={
                    <Box>
                      <Typography variant="body2" color="text.secondary">
                        Added: {new Date(credential.created_at).toLocaleDateString()}
                      </Typography>
                      {credential.last_used && (
                        <Typography variant="body2" color="text.secondary">
                          Last used: {new Date(credential.last_used).toLocaleDateString()}
                        </Typography>
                      )}
                    </Box>
                  }
                />
                <ListItemSecondaryAction>
                  <IconButton
                    onClick={() => setDeviceNameDialog({ open: true, deviceId: credential.id })}
                  >
                    <Edit />
                  </IconButton>
                  <IconButton color="error">
                    <Trash2 />
                  </IconButton>
                </ListItemSecondaryAction>
              </ListItem>
            ))}
          </List>
        )}
        
        <Button
          variant="outlined"
          startIcon={<Plus />}
          sx={{ mt: 2 }}
          onClick={() => setupMFA('webauthn')}
        >
          Add Security Key
        </Button>
      </AccordionDetails>
    </Accordion>
  );

  const renderMFAAttempts = () => (
    <Dialog open={attemptsOpen} onClose={() => setAttemptsOpen(false)} maxWidth="md" fullWidth>
      <DialogTitle>Recent MFA Attempts</DialogTitle>
      <DialogContent>
        <TableContainer>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>Date</TableCell>
                <TableCell>Method</TableCell>
                <TableCell>Status</TableCell>
                <TableCell>IP Address</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {mfaAttempts.map((attempt) => (
                <TableRow key={attempt.id}>
                  <TableCell>
                    {new Date(attempt.created_at).toLocaleString()}
                  </TableCell>
                  <TableCell>
                    <Chip
                      icon={getMethodIcon(attempt.method_used)}
                      label={attempt.method_used}
                      size="small"
                    />
                  </TableCell>
                  <TableCell>
                    <Chip
                      label={attempt.success ? 'Success' : 'Failed'}
                      color={attempt.success ? 'success' : 'error'}
                      size="small"
                    />
                  </TableCell>
                  <TableCell>{attempt.ip_address}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      </DialogContent>
      <DialogActions>
        <Button onClick={() => setAttemptsOpen(false)}>Close</Button>
      </DialogActions>
    </Dialog>
  );

  if (loading) {
    return (
      <Container maxWidth="md" sx={{ py: 4 }}>
        <LinearProgress />
        <Typography sx={{ mt: 2 }}>Loading MFA settings...</Typography>
      </Container>
    );
  }

  return (
    <Container maxWidth="md" sx={{ py: 4 }}>
      <Typography variant="h4" gutterBottom>
        Multi-Factor Authentication
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

      {renderMFAOverview()}
      {renderWebAuthnDevices()}

      {/* Setup Dialog */}
      <Dialog open={setupOpen} onClose={() => setSetupOpen(false)} maxWidth="md" fullWidth>
        <DialogTitle>Setup Multi-Factor Authentication</DialogTitle>
        <DialogContent>
          {renderSetupStepper()}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setSetupOpen(false)}>Cancel</Button>
        </DialogActions>
      </Dialog>

      {/* Disable Confirmation Dialog */}
      <Dialog open={disableConfirmOpen} onClose={() => setDisableConfirmOpen(false)}>
        <DialogTitle>Disable Multi-Factor Authentication</DialogTitle>
        <DialogContent>
          <Alert severity="warning" sx={{ mb: 2 }}>
            Disabling MFA will make your account less secure. Are you sure you want to continue?
          </Alert>
          <Typography>
            This will remove all configured MFA methods including TOTP, WebAuthn devices, and backup codes.
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDisableConfirmOpen(false)}>Cancel</Button>
          <Button onClick={disableMFA} color="error" disabled={saving}>
            Disable MFA
          </Button>
        </DialogActions>
      </Dialog>

      {renderMFAAttempts()}
    </Container>
  );
};

export default MFASettingsDashboard;