import React, { useState, useEffect } from 'react';
import {
  Card,
  CardContent,
  CardHeader,
  Typography,
  Grid,
  Box,
  Button,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  TextField,
  Alert,
  Snackbar,
  LinearProgress,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  Chip,
  IconButton,
  Tooltip,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions
} from '@mui/material';
import {
  Download as DownloadIcon,
  Refresh as RefreshIcon,
  Assessment as AssessmentIcon,
  Security as SecurityIcon,
  Warning as WarningIcon,
  CheckCircle as CheckCircleIcon,
  TrendingUp as TrendingUpIcon,
  TrendingDown as TrendingDownIcon,
  Visibility as VisibilityIcon
} from '@mui/icons-material';
import {
  PieChart,
  Pie,
  Cell,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip as RechartsTooltip,
  Legend,
  LineChart,
  Line,
  ResponsiveContainer
} from 'recharts';

const ComplianceDashboard = () => {
  const [loading, setLoading] = useState(true);
  const [complianceScore, setComplianceScore] = useState(0);
  const [complianceData, setComplianceData] = useState({});
  const [timeRange, setTimeRange] = useState('7d');
  const [reports, setReports] = useState([]);
  const [generatingReport, setGeneratingReport] = useState(false);
  const [selectedReport, setSelectedReport] = useState(null);
  const [openReportDialog, setOpenReportDialog] = useState(false);
  const [snackbar, setSnackbar] = useState({ open: false, message: '', severity: 'info' });

  // Colors for charts
  const COLORS = {
    primary: '#1976d2',
    success: '#4caf50',
    warning: '#ff9800',
    error: '#f44336',
    info: '#2196f3'
  };

  const CHART_COLORS = ['#1976d2', '#4caf50', '#ff9800', '#f44336', '#9c27b0', '#00bcd4'];

  useEffect(() => {
    fetchComplianceData();
    fetchReports();
  }, [timeRange]);

  const fetchComplianceData = async () => {
    setLoading(true);
    try {
      // Fetch compliance metrics
      const response = await fetch(`/api/admin/compliance/dashboard?timeRange=${timeRange}`);
      if (response.ok) {
        const data = await response.json();
        setComplianceData(data);
        setComplianceScore(data.compliance_score || 0);
      } else {
        showSnackbar('Failed to fetch compliance data', 'error');
      }
    } catch (error) {
      showSnackbar('Error fetching compliance data', 'error');
    } finally {
      setLoading(false);
    }
  };

  const fetchReports = async () => {
    try {
      const response = await fetch('/api/admin/compliance/reports');
      if (response.ok) {
        const data = await response.json();
        setReports(data.reports || []);
      }
    } catch (error) {
      showSnackbar('Failed to fetch reports', 'error');
    }
  };

  const generateComplianceReport = async () => {
    setGeneratingReport(true);
    try {
      const endDate = new Date();
      const startDate = new Date();
      startDate.setDate(endDate.getDate() - parseInt(timeRange.replace('d', '')));

      const response = await fetch('/api/admin/compliance/generate-report', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          start_date: startDate.toISOString(),
          end_date: endDate.toISOString(),
          generated_by: 1, // TODO: Get actual user ID
        }),
      });

      if (response.ok) {
        const result = await response.json();
        showSnackbar('Compliance report generated successfully', 'success');
        fetchReports();
      } else {
        showSnackbar('Failed to generate report', 'error');
      }
    } catch (error) {
      showSnackbar('Error generating report', 'error');
    } finally {
      setGeneratingReport(false);
    }
  };

  const downloadReport = async (reportId) => {
    try {
      const response = await fetch(`/api/admin/compliance/reports/${reportId}/download`);
      if (response.ok) {
        const blob = await response.blob();
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `compliance_report_${reportId}.json`;
        document.body.appendChild(a);
        a.click();
        window.URL.revokeObjectURL(url);
        document.body.removeChild(a);
      }
    } catch (error) {
      showSnackbar('Failed to download report', 'error');
    }
  };

  const showSnackbar = (message, severity = 'info') => {
    setSnackbar({ open: true, message, severity });
  };

  const getComplianceStatus = (score) => {
    if (score >= 95) return { label: 'Excellent', color: 'success', icon: <CheckCircleIcon /> };
    if (score >= 85) return { label: 'Good', color: 'success', icon: <CheckCircleIcon /> };
    if (score >= 70) return { label: 'Fair', color: 'warning', icon: <WarningIcon /> };
    return { label: 'Poor', color: 'error', icon: <WarningIcon /> };
  };

  const formatDate = (dateString) => {
    return new Date(dateString).toLocaleString();
  };

  // Mock data for charts - in a real app, this would come from the API
  const mockPolicyDecisionData = [
    { name: 'Allowed', value: complianceData.policy_decisions?.allowed || 0, color: COLORS.success },
    { name: 'Denied', value: complianceData.policy_decisions?.denied || 0, color: COLORS.error },
  ];

  const mockViolationTrendData = [
    { date: '2024-01-01', critical: 2, high: 5, medium: 12, low: 8 },
    { date: '2024-01-02', critical: 1, high: 3, medium: 15, low: 10 },
    { date: '2024-01-03', critical: 0, high: 4, medium: 18, low: 12 },
    { date: '2024-01-04', critical: 1, high: 2, medium: 14, low: 9 },
    { date: '2024-01-05', critical: 0, high: 1, medium: 16, low: 11 },
  ];

  const mockComplianceMetrics = [
    { metric: 'Access Control', score: 92, trend: 'up' },
    { metric: 'Authentication', score: 88, trend: 'up' },
    { metric: 'Data Protection', score: 95, trend: 'stable' },
    { metric: 'Audit Logging', score: 97, trend: 'up' },
    { metric: 'Incident Response', score: 85, trend: 'down' },
  ];

  const complianceStatus = getComplianceStatus(complianceScore);

  if (loading) {
    return (
      <Box display="flex" flexDirection="column" gap={2} p={3}>
        <Typography variant="h4">Compliance Dashboard</Typography>
        <LinearProgress />
        <Typography>Loading compliance data...</Typography>
      </Box>
    );
  }

  return (
    <Box p={3}>
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={3}>
        <Typography variant="h4">Security Compliance Dashboard</Typography>
        
        <Box display="flex" gap={2} alignItems="center">
          <FormControl size="small" style={{ minWidth: 120 }}>
            <InputLabel>Time Range</InputLabel>
            <Select
              value={timeRange}
              label="Time Range"
              onChange={(e) => setTimeRange(e.target.value)}
            >
              <MenuItem value="1d">Last 24 Hours</MenuItem>
              <MenuItem value="7d">Last 7 Days</MenuItem>
              <MenuItem value="30d">Last 30 Days</MenuItem>
              <MenuItem value="90d">Last 90 Days</MenuItem>
            </Select>
          </FormControl>
          
          <Button
            variant="outlined"
            startIcon={<RefreshIcon />}
            onClick={fetchComplianceData}
          >
            Refresh
          </Button>
          
          <Button
            variant="contained"
            startIcon={<AssessmentIcon />}
            onClick={generateComplianceReport}
            disabled={generatingReport}
          >
            {generatingReport ? 'Generating...' : 'Generate Report'}
          </Button>
        </Box>
      </Box>

      {/* Compliance Score Overview */}
      <Grid container spacing={3} mb={3}>
        <Grid item xs={12} md={4}>
          <Card>
            <CardHeader 
              title="Overall Compliance Score"
              avatar={<SecurityIcon color="primary" />}
            />
            <CardContent>
              <Box display="flex" alignItems="center" gap={2}>
                <Typography variant="h2" color={`${complianceStatus.color}.main`}>
                  {complianceScore}%
                </Typography>
                <Box>
                  <Chip 
                    label={complianceStatus.label} 
                    color={complianceStatus.color}
                    icon={complianceStatus.icon}
                  />
                </Box>
              </Box>
              <LinearProgress 
                variant="determinate" 
                value={complianceScore} 
                color={complianceStatus.color}
                style={{ marginTop: 16, height: 8, borderRadius: 4 }}
              />
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} md={4}>
          <Card>
            <CardHeader title="Policy Decisions (24h)" />
            <CardContent>
              <Box display="flex" justifyContent="space-between" mb={2}>
                <Typography color="success.main">
                  ✓ Allowed: {complianceData.policy_decisions?.allowed || 0}
                </Typography>
                <Typography color="error.main">
                  ✗ Denied: {complianceData.policy_decisions?.denied || 0}
                </Typography>
              </Box>
              <ResponsiveContainer width="100%" height={120}>
                <PieChart>
                  <Pie
                    data={mockPolicyDecisionData}
                    cx="50%"
                    cy="50%"
                    innerRadius={30}
                    outerRadius={50}
                    dataKey="value"
                  >
                    {mockPolicyDecisionData.map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={entry.color} />
                    ))}
                  </Pie>
                  <RechartsTooltip />
                </PieChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} md={4}>
          <Card>
            <CardHeader title="Security Violations" />
            <CardContent>
              <Box display="flex" flexDirection="column" gap={1}>
                <Box display="flex" justifyContent="space-between">
                  <Typography variant="body2">Critical</Typography>
                  <Typography color="error.main" fontWeight="bold">
                    {complianceData.security_violations?.critical || 0}
                  </Typography>
                </Box>
                <Box display="flex" justifyContent="space-between">
                  <Typography variant="body2">High</Typography>
                  <Typography color="warning.main" fontWeight="bold">
                    {complianceData.security_violations?.high || 0}
                  </Typography>
                </Box>
                <Box display="flex" justifyContent="space-between">
                  <Typography variant="body2">Medium</Typography>
                  <Typography color="info.main">
                    {complianceData.security_violations?.medium || 0}
                  </Typography>
                </Box>
                <Box display="flex" justifyContent="space-between">
                  <Typography variant="body2">Low</Typography>
                  <Typography color="text.secondary">
                    {complianceData.security_violations?.low || 0}
                  </Typography>
                </Box>
              </Box>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Compliance Metrics Table */}
      <Grid container spacing={3} mb={3}>
        <Grid item xs={12} md={6}>
          <Card>
            <CardHeader title="Compliance Metrics" />
            <CardContent>
              <TableContainer>
                <Table size="small">
                  <TableHead>
                    <TableRow>
                      <TableCell>Metric</TableCell>
                      <TableCell align="right">Score</TableCell>
                      <TableCell align="right">Trend</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {mockComplianceMetrics.map((metric) => (
                      <TableRow key={metric.metric}>
                        <TableCell>{metric.metric}</TableCell>
                        <TableCell align="right">
                          <Chip 
                            label={`${metric.score}%`}
                            size="small"
                            color={metric.score >= 90 ? 'success' : metric.score >= 80 ? 'warning' : 'error'}
                          />
                        </TableCell>
                        <TableCell align="right">
                          {metric.trend === 'up' && <TrendingUpIcon color="success" />}
                          {metric.trend === 'down' && <TrendingDownIcon color="error" />}
                          {metric.trend === 'stable' && <span>—</span>}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </TableContainer>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} md={6}>
          <Card>
            <CardHeader title="Violation Trends" />
            <CardContent>
              <ResponsiveContainer width="100%" height={250}>
                <LineChart data={mockViolationTrendData}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="date" />
                  <YAxis />
                  <RechartsTooltip />
                  <Legend />
                  <Line type="monotone" dataKey="critical" stroke={COLORS.error} strokeWidth={2} />
                  <Line type="monotone" dataKey="high" stroke={COLORS.warning} strokeWidth={2} />
                  <Line type="monotone" dataKey="medium" stroke={COLORS.info} strokeWidth={2} />
                  <Line type="monotone" dataKey="low" stroke={COLORS.success} strokeWidth={2} />
                </LineChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Recent Reports */}
      <Card>
        <CardHeader title="Recent Compliance Reports" />
        <CardContent>
          <TableContainer component={Paper}>
            <Table>
              <TableHead>
                <TableRow>
                  <TableCell>Report ID</TableCell>
                  <TableCell>Type</TableCell>
                  <TableCell>Generated</TableCell>
                  <TableCell>Status</TableCell>
                  <TableCell>Actions</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {reports.map((report) => (
                  <TableRow key={report.id}>
                    <TableCell>{report.id}</TableCell>
                    <TableCell>{report.report_type}</TableCell>
                    <TableCell>{formatDate(report.generated_at)}</TableCell>
                    <TableCell>
                      <Chip 
                        label="Completed" 
                        color="success" 
                        size="small" 
                      />
                    </TableCell>
                    <TableCell>
                      <Tooltip title="View Report">
                        <IconButton 
                          size="small"
                          onClick={() => {
                            setSelectedReport(report);
                            setOpenReportDialog(true);
                          }}
                        >
                          <VisibilityIcon />
                        </IconButton>
                      </Tooltip>
                      
                      <Tooltip title="Download Report">
                        <IconButton 
                          size="small"
                          onClick={() => downloadReport(report.id)}
                        >
                          <DownloadIcon />
                        </IconButton>
                      </Tooltip>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        </CardContent>
      </Card>

      {/* Report Details Dialog */}
      <Dialog 
        open={openReportDialog} 
        onClose={() => setOpenReportDialog(false)}
        maxWidth="md"
        fullWidth
      >
        <DialogTitle>Compliance Report Details</DialogTitle>
        <DialogContent>
          {selectedReport && (
            <Box>
              <Typography variant="h6" gutterBottom>
                Report: {selectedReport.id}
              </Typography>
              
              <Table size="small">
                <TableBody>
                  <TableRow>
                    <TableCell><strong>Type</strong></TableCell>
                    <TableCell>{selectedReport.report_type}</TableCell>
                  </TableRow>
                  <TableRow>
                    <TableCell><strong>Generated</strong></TableCell>
                    <TableCell>{formatDate(selectedReport.generated_at)}</TableCell>
                  </TableRow>
                  <TableRow>
                    <TableCell><strong>Format</strong></TableCell>
                    <TableCell>{selectedReport.report_format || 'JSON'}</TableCell>
                  </TableRow>
                </TableBody>
              </Table>

              {selectedReport.parameters && (
                <Box mt={2}>
                  <Typography variant="h6" gutterBottom>
                    Report Data Summary:
                  </Typography>
                  <Paper elevation={1} style={{ padding: '16px' }}>
                    <pre style={{ 
                      whiteSpace: 'pre-wrap', 
                      fontSize: '12px',
                      margin: 0,
                      maxHeight: '300px',
                      overflow: 'auto'
                    }}>
                      {JSON.stringify(selectedReport.parameters, null, 2)}
                    </pre>
                  </Paper>
                </Box>
              )}
            </Box>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setOpenReportDialog(false)}>Close</Button>
          <Button 
            onClick={() => downloadReport(selectedReport?.id)}
            variant="contained"
            startIcon={<DownloadIcon />}
          >
            Download
          </Button>
        </DialogActions>
      </Dialog>

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
    </Box>
  );
};

export default ComplianceDashboard;