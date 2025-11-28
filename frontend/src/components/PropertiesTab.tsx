import React, { useState, useEffect } from 'react';
import {
  Box,
  Button,
  Chip,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  IconButton,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
  Typography,
  Paper,
  FormControlLabel,
  Switch,
  Tooltip,
  CircularProgress,
  Alert,
  Snackbar,
  InputAdornment,
  MenuItem,
  Select,
  FormControl,
  InputLabel,
  Tabs,
  Tab,
  Grid,
} from '@mui/material';
import {
  Add as AddIcon,
  Edit as EditIcon,
  Delete as DeleteIcon,
  Visibility as ViewIcon,
  VisibilityOff as HideIcon,
  Search as SearchIcon,
  Lock as LockIcon,
  FilterList as FilterIcon,
  FileDownload as ExportIcon,
  Refresh as RefreshIcon,
} from '@mui/icons-material';

interface ArtifactProperty {
  id: string;
  key: string;
  value: string;
  value_type: string;
  is_sensitive: boolean;
  is_system: boolean;
  is_multi_value: boolean;
  tags: string[];
  description?: string;
  created_at: string;
  updated_at: string;
  version: number;
  masked?: boolean;
}

interface PropertiesTabProps {
  artifactId: string;
  repositoryId: string;
  tenantId: string;
  canEdit: boolean;
  canDelete: boolean;
  canReadSensitive: boolean;
}

const PropertiesTab: React.FC<PropertiesTabProps> = ({
  artifactId,
  repositoryId,
  tenantId,
  canEdit,
  canDelete,
  canReadSensitive,
}) => {
  // State
  const [properties, setProperties] = useState<ArtifactProperty[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [selectedProperty, setSelectedProperty] = useState<ArtifactProperty | null>(null);
  const [openDialog, setOpenDialog] = useState(false);
  const [dialogMode, setDialogMode] = useState<'create' | 'edit' | 'view'>('create');
  const [searchTerm, setSearchTerm] = useState('');
  const [filterType, setFilterType] = useState<string>('all');
  const [showSensitive, setShowSensitive] = useState(false);
  const [tabValue, setTabValue] = useState(0);
  
  // Form state
  const [formData, setFormData] = useState({
    key: '',
    value: '',
    value_type: 'string',
    is_sensitive: false,
    is_multi_value: false,
    tags: [] as string[],
    description: '',
  });

  // Statistics
  const [stats, setStats] = useState({
    total: 0,
    sensitive: 0,
    system: 0,
    custom: 0,
  });

  useEffect(() => {
    loadProperties();
  }, [artifactId, showSensitive]);

  const loadProperties = async () => {
    setLoading(true);
    try {
      const response = await fetch(
        `/api/v1/artifacts/${artifactId}/properties?mask_sensitive=${!showSensitive}`,
        {
          headers: {
            Authorization: `Bearer ${localStorage.getItem('token')}`,
            'X-Tenant-ID': tenantId,
          },
        }
      );

      if (!response.ok) {
        throw new Error('Failed to load properties');
      }

      const data = await response.json();
      setProperties(data.properties || []);
      
      // Calculate statistics
      const total = data.properties?.length || 0;
      const sensitive = data.properties?.filter((p: ArtifactProperty) => p.is_sensitive).length || 0;
      const system = data.properties?.filter((p: ArtifactProperty) => p.is_system).length || 0;
      const custom = total - system;
      
      setStats({ total, sensitive, system, custom });
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load properties');
    } finally {
      setLoading(false);
    }
  };

  const handleCreateProperty = async () => {
    try {
      const response = await fetch(`/api/v1/artifacts/${artifactId}/properties`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${localStorage.getItem('token')}`,
          'X-Tenant-ID': tenantId,
        },
        body: JSON.stringify({
          repository_id: repositoryId,
          property: formData,
        }),
      });

      if (!response.ok) {
        throw new Error('Failed to create property');
      }

      setSuccess('Property created successfully');
      setOpenDialog(false);
      resetForm();
      loadProperties();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create property');
    }
  };

  const handleUpdateProperty = async () => {
    if (!selectedProperty) return;

    try {
      const response = await fetch(`/api/v1/properties/${selectedProperty.id}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${localStorage.getItem('token')}`,
          'X-Tenant-ID': tenantId,
        },
        body: JSON.stringify({
          value: formData.value || undefined,
          value_type: formData.value_type || undefined,
          is_sensitive: formData.is_sensitive,
          tags: formData.tags,
          description: formData.description || undefined,
        }),
      });

      if (!response.ok) {
        throw new Error('Failed to update property');
      }

      setSuccess('Property updated successfully');
      setOpenDialog(false);
      resetForm();
      loadProperties();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update property');
    }
  };

  const handleDeleteProperty = async (propertyId: string) => {
    if (!confirm('Are you sure you want to delete this property?')) {
      return;
    }

    try {
      const response = await fetch(`/api/v1/properties/${propertyId}`, {
        method: 'DELETE',
        headers: {
          Authorization: `Bearer ${localStorage.getItem('token')}`,
          'X-Tenant-ID': tenantId,
        },
      });

      if (!response.ok) {
        throw new Error('Failed to delete property');
      }

      setSuccess('Property deleted successfully');
      loadProperties();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete property');
    }
  };

  const openCreateDialog = () => {
    resetForm();
    setDialogMode('create');
    setOpenDialog(true);
  };

  const openEditDialog = (property: ArtifactProperty) => {
    setSelectedProperty(property);
    setFormData({
      key: property.key,
      value: property.masked ? '' : property.value,
      value_type: property.value_type,
      is_sensitive: property.is_sensitive,
      is_multi_value: property.is_multi_value,
      tags: property.tags || [],
      description: property.description || '',
    });
    setDialogMode('edit');
    setOpenDialog(true);
  };

  const openViewDialog = (property: ArtifactProperty) => {
    setSelectedProperty(property);
    setFormData({
      key: property.key,
      value: property.value,
      value_type: property.value_type,
      is_sensitive: property.is_sensitive,
      is_multi_value: property.is_multi_value,
      tags: property.tags || [],
      description: property.description || '',
    });
    setDialogMode('view');
    setOpenDialog(true);
  };

  const resetForm = () => {
    setFormData({
      key: '',
      value: '',
      value_type: 'string',
      is_sensitive: false,
      is_multi_value: false,
      tags: [],
      description: '',
    });
    setSelectedProperty(null);
  };

  const handleExport = () => {
    const csv = [
      ['Key', 'Value', 'Type', 'Sensitive', 'System', 'Tags', 'Description'],
      ...filteredProperties.map((p) => [
        p.key,
        p.masked ? '***MASKED***' : p.value,
        p.value_type,
        p.is_sensitive ? 'Yes' : 'No',
        p.is_system ? 'Yes' : 'No',
        p.tags?.join('; ') || '',
        p.description || '',
      ]),
    ]
      .map((row) => row.map((cell) => `"${cell}"`).join(','))
      .join('\n');

    const blob = new Blob([csv], { type: 'text/csv' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `artifact-${artifactId}-properties.csv`;
    a.click();
  };

  // Filtering
  const filteredProperties = properties.filter((prop) => {
    const matchesSearch =
      prop.key.toLowerCase().includes(searchTerm.toLowerCase()) ||
      prop.value.toLowerCase().includes(searchTerm.toLowerCase());

    const matchesFilter =
      filterType === 'all' ||
      (filterType === 'sensitive' && prop.is_sensitive) ||
      (filterType === 'system' && prop.is_system) ||
      (filterType === 'custom' && !prop.is_system);

    return matchesSearch && matchesFilter;
  });

  // Group properties by prefix
  const groupedProperties = filteredProperties.reduce((acc, prop) => {
    const prefix = prop.key.includes('.') ? prop.key.split('.')[0] : 'custom';
    if (!acc[prefix]) {
      acc[prefix] = [];
    }
    acc[prefix].push(prop);
    return acc;
  }, {} as Record<string, ArtifactProperty[]>);

  return (
    <Box sx={{ p: 3 }}>
      {/* Header */}
      <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 3 }}>
        <Typography variant="h5" fontWeight="bold">
          Artifact Properties
        </Typography>
        <Box sx={{ display: 'flex', gap: 1 }}>
          <Tooltip title="Refresh">
            <IconButton onClick={loadProperties} color="primary">
              <RefreshIcon />
            </IconButton>
          </Tooltip>
          <Tooltip title="Export to CSV">
            <IconButton onClick={handleExport} color="primary">
              <ExportIcon />
            </IconButton>
          </Tooltip>
          {canEdit && (
            <Button
              variant="contained"
              startIcon={<AddIcon />}
              onClick={openCreateDialog}
            >
              Add Property
            </Button>
          )}
        </Box>
      </Box>

      {/* Statistics Cards */}
      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid item xs={3}>
          <Paper sx={{ p: 2, textAlign: 'center' }}>
            <Typography variant="h4" color="primary">
              {stats.total}
            </Typography>
            <Typography variant="body2" color="textSecondary">
              Total Properties
            </Typography>
          </Paper>
        </Grid>
        <Grid item xs={3}>
          <Paper sx={{ p: 2, textAlign: 'center' }}>
            <Typography variant="h4" color="warning.main">
              {stats.sensitive}
            </Typography>
            <Typography variant="body2" color="textSecondary">
              Sensitive
            </Typography>
          </Paper>
        </Grid>
        <Grid item xs={3}>
          <Paper sx={{ p: 2, textAlign: 'center' }}>
            <Typography variant="h4" color="info.main">
              {stats.system}
            </Typography>
            <Typography variant="body2" color="textSecondary">
              System
            </Typography>
          </Paper>
        </Grid>
        <Grid item xs={3}>
          <Paper sx={{ p: 2, textAlign: 'center' }}>
            <Typography variant="h4" color="success.main">
              {stats.custom}
            </Typography>
            <Typography variant="body2" color="textSecondary">
              Custom
            </Typography>
          </Paper>
        </Grid>
      </Grid>

      {/* Filters */}
      <Paper sx={{ p: 2, mb: 2 }}>
        <Box sx={{ display: 'flex', gap: 2, alignItems: 'center' }}>
          <TextField
            placeholder="Search properties..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            size="small"
            sx={{ flexGrow: 1 }}
            InputProps={{
              startAdornment: (
                <InputAdornment position="start">
                  <SearchIcon />
                </InputAdornment>
              ),
            }}
          />
          <FormControl size="small" sx={{ minWidth: 150 }}>
            <InputLabel>Filter</InputLabel>
            <Select
              value={filterType}
              label="Filter"
              onChange={(e) => setFilterType(e.target.value)}
            >
              <MenuItem value="all">All Properties</MenuItem>
              <MenuItem value="sensitive">Sensitive Only</MenuItem>
              <MenuItem value="system">System Only</MenuItem>
              <MenuItem value="custom">Custom Only</MenuItem>
            </Select>
          </FormControl>
          {canReadSensitive && (
            <FormControlLabel
              control={
                <Switch
                  checked={showSensitive}
                  onChange={(e) => setShowSensitive(e.target.checked)}
                />
              }
              label="Show Sensitive Values"
            />
          )}
        </Box>
      </Paper>

      {/* Properties Tabs (Grouped by prefix) */}
      <Paper>
        <Tabs value={tabValue} onChange={(_, v) => setTabValue(v)} sx={{ borderBottom: 1, borderColor: 'divider' }}>
          <Tab label="All" />
          {Object.keys(groupedProperties).map((prefix) => (
            <Tab key={prefix} label={prefix} />
          ))}
        </Tabs>

        {/* Properties Table */}
        {loading ? (
          <Box sx={{ display: 'flex', justifyContent: 'center', p: 4 }}>
            <CircularProgress />
          </Box>
        ) : filteredProperties.length === 0 ? (
          <Box sx={{ p: 4, textAlign: 'center' }}>
            <Typography color="textSecondary">No properties found</Typography>
          </Box>
        ) : (
          <TableContainer>
            <Table>
              <TableHead>
                <TableRow>
                  <TableCell>Key</TableCell>
                  <TableCell>Value</TableCell>
                  <TableCell>Type</TableCell>
                  <TableCell>Flags</TableCell>
                  <TableCell>Tags</TableCell>
                  <TableCell>Updated</TableCell>
                  <TableCell align="right">Actions</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {(tabValue === 0
                  ? filteredProperties
                  : groupedProperties[Object.keys(groupedProperties)[tabValue - 1]] || []
                ).map((property) => (
                  <TableRow key={property.id} hover>
                    <TableCell>
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                        {property.is_sensitive && (
                          <Tooltip title="Sensitive">
                            <LockIcon fontSize="small" color="warning" />
                          </Tooltip>
                        )}
                        <Typography variant="body2" fontFamily="monospace">
                          {property.key}
                        </Typography>
                      </Box>
                    </TableCell>
                    <TableCell>
                      <Typography
                        variant="body2"
                        fontFamily="monospace"
                        sx={{
                          maxWidth: 300,
                          overflow: 'hidden',
                          textOverflow: 'ellipsis',
                          whiteSpace: 'nowrap',
                        }}
                      >
                        {property.masked ? '***MASKED***' : property.value}
                      </Typography>
                    </TableCell>
                    <TableCell>
                      <Chip label={property.value_type} size="small" variant="outlined" />
                    </TableCell>
                    <TableCell>
                      <Box sx={{ display: 'flex', gap: 0.5 }}>
                        {property.is_system && <Chip label="System" size="small" color="info" />}
                        {property.is_multi_value && (
                          <Chip label="Multi" size="small" color="secondary" />
                        )}
                      </Box>
                    </TableCell>
                    <TableCell>
                      <Box sx={{ display: 'flex', gap: 0.5, flexWrap: 'wrap' }}>
                        {property.tags?.slice(0, 2).map((tag) => (
                          <Chip key={tag} label={tag} size="small" />
                        ))}
                        {property.tags?.length > 2 && (
                          <Chip label={`+${property.tags.length - 2}`} size="small" />
                        )}
                      </Box>
                    </TableCell>
                    <TableCell>
                      <Typography variant="body2" color="textSecondary">
                        {new Date(property.updated_at).toLocaleDateString()}
                      </Typography>
                    </TableCell>
                    <TableCell align="right">
                      <Box sx={{ display: 'flex', justifyContent: 'flex-end', gap: 0.5 }}>
                        <Tooltip title="View">
                          <IconButton size="small" onClick={() => openViewDialog(property)}>
                            <ViewIcon fontSize="small" />
                          </IconButton>
                        </Tooltip>
                        {canEdit && !property.is_system && (
                          <Tooltip title="Edit">
                            <IconButton
                              size="small"
                              color="primary"
                              onClick={() => openEditDialog(property)}
                            >
                              <EditIcon fontSize="small" />
                            </IconButton>
                          </Tooltip>
                        )}
                        {canDelete && !property.is_system && (
                          <Tooltip title="Delete">
                            <IconButton
                              size="small"
                              color="error"
                              onClick={() => handleDeleteProperty(property.id)}
                            >
                              <DeleteIcon fontSize="small" />
                            </IconButton>
                          </Tooltip>
                        )}
                      </Box>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        )}
      </Paper>

      {/* Create/Edit Dialog */}
      <Dialog open={openDialog} onClose={() => setOpenDialog(false)} maxWidth="md" fullWidth>
        <DialogTitle>
          {dialogMode === 'create' ? 'Create Property' : dialogMode === 'edit' ? 'Edit Property' : 'View Property'}
        </DialogTitle>
        <DialogContent>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, mt: 2 }}>
            <TextField
              label="Key"
              value={formData.key}
              onChange={(e) => setFormData({ ...formData, key: e.target.value })}
              disabled={dialogMode !== 'create'}
              fullWidth
              required
              helperText="Alphanumeric with dots, dashes, underscores (e.g., custom.my-property)"
            />
            <TextField
              label="Value"
              value={formData.value}
              onChange={(e) => setFormData({ ...formData, value: e.target.value })}
              disabled={dialogMode === 'view'}
              fullWidth
              required
              multiline
              rows={3}
              helperText={
                formData.is_sensitive
                  ? 'This value will be encrypted at rest'
                  : 'Maximum 64KB'
              }
            />
            <FormControl fullWidth>
              <InputLabel>Value Type</InputLabel>
              <Select
                value={formData.value_type}
                label="Value Type"
                onChange={(e) => setFormData({ ...formData, value_type: e.target.value })}
                disabled={dialogMode === 'view'}
              >
                <MenuItem value="string">String</MenuItem>
                <MenuItem value="number">Number</MenuItem>
                <MenuItem value="boolean">Boolean</MenuItem>
                <MenuItem value="json">JSON</MenuItem>
                <MenuItem value="array">Array</MenuItem>
              </Select>
            </FormControl>
            <TextField
              label="Description"
              value={formData.description}
              onChange={(e) => setFormData({ ...formData, description: e.target.value })}
              disabled={dialogMode === 'view'}
              fullWidth
              multiline
              rows={2}
            />
            <TextField
              label="Tags (comma-separated)"
              value={formData.tags.join(', ')}
              onChange={(e) =>
                setFormData({ ...formData, tags: e.target.value.split(',').map((t) => t.trim()) })
              }
              disabled={dialogMode === 'view'}
              fullWidth
              helperText="Maximum 20 tags"
            />
            {dialogMode === 'create' && (
              <>
                <FormControlLabel
                  control={
                    <Switch
                      checked={formData.is_sensitive}
                      onChange={(e) =>
                        setFormData({ ...formData, is_sensitive: e.target.checked })
                      }
                    />
                  }
                  label="Sensitive (encrypt value)"
                />
                <FormControlLabel
                  control={
                    <Switch
                      checked={formData.is_multi_value}
                      onChange={(e) =>
                        setFormData({ ...formData, is_multi_value: e.target.checked })
                      }
                    />
                  }
                  label="Allow multiple values"
                />
              </>
            )}
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setOpenDialog(false)}>Cancel</Button>
          {dialogMode !== 'view' && (
            <Button
              variant="contained"
              onClick={dialogMode === 'create' ? handleCreateProperty : handleUpdateProperty}
            >
              {dialogMode === 'create' ? 'Create' : 'Update'}
            </Button>
          )}
        </DialogActions>
      </Dialog>

      {/* Snackbars */}
      <Snackbar
        open={!!error}
        autoHideDuration={6000}
        onClose={() => setError(null)}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
      >
        <Alert severity="error" onClose={() => setError(null)}>
          {error}
        </Alert>
      </Snackbar>
      <Snackbar
        open={!!success}
        autoHideDuration={3000}
        onClose={() => setSuccess(null)}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
      >
        <Alert severity="success" onClose={() => setSuccess(null)}>
          {success}
        </Alert>
      </Snackbar>
    </Box>
  );
};

export default PropertiesTab;
