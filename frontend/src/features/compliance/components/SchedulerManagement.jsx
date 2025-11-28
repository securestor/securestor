import React, { useState, useEffect } from 'react';
import { 
  Clock, 
  Play, 
  Pause, 
  AlertCircle, 
  CheckCircle, 
  RefreshCw,
  Settings,
  Activity
} from 'lucide-react';
import complianceAPI from '../../../services/api/complianceAPI';

const SchedulerManagement = () => {
  const [schedulerStatus, setSchedulerStatus] = useState(null);
  const [loading, setLoading] = useState(true);
  const [triggering, setTriggering] = useState(null);
  const [error, setError] = useState(null);

  // Fetch scheduler status
  const fetchSchedulerStatus = async () => {
    try {
      setLoading(true);
      setError(null);
      const status = await complianceAPI.getSchedulerStatus();
      setSchedulerStatus(status);
    } catch (err) {
      setError(err.message || 'Failed to fetch scheduler status');
      console.error('Failed to fetch scheduler status:', err);
    } finally {
      setLoading(false);
    }
  };

  // Trigger a specific job
  const triggerJob = async (jobType) => {
    try {
      setTriggering(jobType);
      setError(null);
      
      await complianceAPI.triggerSchedulerJob(jobType);
      
      // Show success message
      setTimeout(() => {
        fetchSchedulerStatus(); // Refresh status
      }, 1000);
      
    } catch (err) {
      setError(`Failed to trigger ${jobType} job: ${err.message}`);
      console.error(`Failed to trigger ${jobType} job:`, err);
    } finally {
      setTriggering(null);
    }
  };

  useEffect(() => {
    fetchSchedulerStatus();
    
    // Auto-refresh every 30 seconds
    const interval = setInterval(fetchSchedulerStatus, 30000);
    return () => clearInterval(interval);
  }, []);

  const jobs = [
    {
      type: 'retention',
      name: 'Data Retention Enforcement',
      description: 'Enforces data retention policies and removes expired artifacts',
      icon: Clock,
      color: 'blue'
    },
    {
      type: 'erasure',
      name: 'Data Erasure Processing',
      description: 'Processes pending data erasure requests',
      icon: RefreshCw,
      color: 'red'
    },
    {
      type: 'integrity',
      name: 'Data Integrity Checks',
      description: 'Verifies data integrity across the storage system',
      icon: CheckCircle,
      color: 'green'
    },
    {
      type: 'audit_cleanup',
      name: 'Audit Log Cleanup',
      description: 'Removes old audit log entries based on retention policy',
      icon: Activity,
      color: 'purple'
    }
  ];

  if (loading && !schedulerStatus) {
    return (
      <div className="flex items-center justify-center py-8">
        <RefreshCw className="h-6 w-6 animate-spin text-indigo-500" />
        <span className="ml-2 text-gray-600">Loading scheduler status...</span>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-lg font-medium text-gray-900">Compliance Scheduler</h3>
          <p className="text-sm text-gray-600">
            Manage automated compliance enforcement jobs
          </p>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={fetchSchedulerStatus}
            disabled={loading}
            className="inline-flex items-center px-3 py-1 text-sm font-medium text-gray-600 bg-white border border-gray-300 rounded-md hover:bg-gray-50 disabled:opacity-50"
          >
            <RefreshCw className={`h-4 w-4 mr-1 ${loading ? 'animate-spin' : ''}`} />
            Refresh
          </button>
        </div>
      </div>

      {/* Error Display */}
      {error && (
        <div className="bg-red-50 border border-red-200 rounded-md p-4">
          <div className="flex items-center">
            <AlertCircle className="h-5 w-5 text-red-400" />
            <div className="ml-3">
              <h3 className="text-sm font-medium text-red-800">Error</h3>
              <div className="mt-2 text-sm text-red-700">{error}</div>
            </div>
          </div>
        </div>
      )}

      {/* Scheduler Status */}
      {schedulerStatus && (
        <div className="bg-white border border-gray-200 rounded-lg p-6">
          <div className="flex items-center justify-between mb-4">
            <h4 className="text-md font-medium text-gray-900 flex items-center">
              <Settings className="h-5 w-5 mr-2 text-gray-400" />
              Scheduler Status
            </h4>
            <div className="flex items-center">
              <div className={`w-2 h-2 rounded-full mr-2 ${
                schedulerStatus.running ? 'bg-green-500' : 'bg-red-500'
              }`} />
              <span className={`text-sm font-medium ${
                schedulerStatus.running ? 'text-green-600' : 'text-red-600'
              }`}>
                {schedulerStatus.running ? 'Running' : 'Stopped'}
              </span>
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4 text-sm">
            <div>
              <span className="text-gray-500">Retention Interval:</span>
              <span className="ml-2 font-medium">{schedulerStatus.retention_interval}</span>
            </div>
            <div>
              <span className="text-gray-500">Erasure Interval:</span>
              <span className="ml-2 font-medium">{schedulerStatus.erasure_interval}</span>
            </div>
            <div>
              <span className="text-gray-500">Integrity Interval:</span>
              <span className="ml-2 font-medium">{schedulerStatus.integrity_interval}</span>
            </div>
            <div>
              <span className="text-gray-500">Audit Cleanup Interval:</span>
              <span className="ml-2 font-medium">{schedulerStatus.audit_cleanup_interval}</span>
            </div>
          </div>
        </div>
      )}

      {/* Manual Job Triggers */}
      <div className="bg-white border border-gray-200 rounded-lg p-6">
        <h4 className="text-md font-medium text-gray-900 mb-4 flex items-center">
          <Play className="h-5 w-5 mr-2 text-gray-400" />
          Manual Job Triggers
        </h4>
        <p className="text-sm text-gray-600 mb-6">
          Manually trigger compliance jobs outside of their scheduled intervals.
        </p>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {jobs.map((job) => (
            <div
              key={job.type}
              className="border border-gray-200 rounded-lg p-4 hover:shadow-sm transition-shadow"
            >
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <div className="flex items-center mb-2">
                    <job.icon className={`h-5 w-5 mr-2 text-${job.color}-500`} />
                    <h5 className="text-sm font-medium text-gray-900">{job.name}</h5>
                  </div>
                  <p className="text-xs text-gray-600 mb-3">{job.description}</p>
                </div>
              </div>
              
              <button
                onClick={() => triggerJob(job.type)}
                disabled={triggering === job.type || !schedulerStatus?.running}
                className={`w-full inline-flex items-center justify-center px-3 py-2 text-sm font-medium rounded-md transition-colors ${
                  triggering === job.type
                    ? 'bg-gray-100 text-gray-400 cursor-not-allowed'
                    : !schedulerStatus?.running
                    ? 'bg-gray-100 text-gray-400 cursor-not-allowed'
                    : `bg-${job.color}-50 text-${job.color}-600 border border-${job.color}-200 hover:bg-${job.color}-100`
                } disabled:opacity-50`}
              >
                {triggering === job.type ? (
                  <>
                    <RefreshCw className="h-4 w-4 animate-spin mr-1" />
                    Triggering...
                  </>
                ) : (
                  <>
                    <Play className="h-4 w-4 mr-1" />
                    Trigger Now
                  </>
                )}
              </button>
            </div>
          ))}
        </div>
      </div>

      {/* Schedule Information */}
      <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
        <div className="flex items-center">
          <AlertCircle className="h-5 w-5 text-blue-400" />
          <div className="ml-3">
            <h3 className="text-sm font-medium text-blue-800">Automated Scheduling</h3>
            <div className="mt-1 text-sm text-blue-700">
              Jobs run automatically based on their configured intervals. Manual triggers 
              execute jobs immediately without affecting the scheduled intervals.
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default SchedulerManagement;