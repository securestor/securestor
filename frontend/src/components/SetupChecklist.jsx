import React, { useState, useEffect } from 'react';
import { CheckCircle2, Circle, AlertCircle, ChevronRight, ExternalLink } from 'lucide-react';
import { useAuth } from '../context/AuthContext';

/**
 * SetupChecklist - Shows initial setup tasks that need to be completed
 * Displays on the dashboard until all critical tasks are done
 */
export const SetupChecklist = () => {
  const [setupStatus, setSetupStatus] = useState(null);
  const [isExpanded, setIsExpanded] = useState(true);
  const [isHidden, setIsHidden] = useState(false);
  const { getApiUrl } = useAuth();

  useEffect(() => {
    const checkSetupStatus = async () => {
      try {
        const response = await fetch(getApiUrl('/api/v1/setup/status'), {
          headers: {
            'Authorization': `Bearer ${localStorage.getItem('auth_token')}`
          }
        });
        
        if (response.ok) {
          const data = await response.json();
          setSetupStatus(data);
          
          // Auto-hide if all critical tasks are completed
          if (data.all_critical_complete) {
            const dismissed = localStorage.getItem('setup-checklist-dismissed');
            if (dismissed) {
              setIsHidden(true);
            }
          }
        }
      } catch (error) {
        console.error('Failed to fetch setup status:', error);
      }
    };

    checkSetupStatus();
    // Refresh every 30 seconds
    const interval = setInterval(checkSetupStatus, 30000);
    return () => clearInterval(interval);
  }, []);

  const handleDismiss = () => {
    setIsHidden(true);
    localStorage.setItem('setup-checklist-dismissed', 'true');
  };

  if (!setupStatus || isHidden) {
    return null;
  }

  const criticalIncomplete = setupStatus.tasks.filter(t => t.priority === 'critical' && !t.completed).length;
  const recommendedIncomplete = setupStatus.tasks.filter(t => t.priority === 'recommended' && !t.completed).length;

  return (
    <div className="bg-white border border-gray-200 rounded-lg shadow-sm overflow-hidden">
      <div
        className="flex items-center justify-between p-4 cursor-pointer hover:bg-gray-50"
        onClick={() => setIsExpanded(!isExpanded)}
      >
        <div className="flex items-center gap-3">
          {criticalIncomplete > 0 ? (
            <AlertCircle className="h-6 w-6 text-orange-500" />
          ) : (
            <CheckCircle2 className="h-6 w-6 text-green-500" />
          )}
          <div>
            <h3 className="text-lg font-semibold text-gray-900">
              Initial Setup Checklist
            </h3>
            <p className="text-sm text-gray-600">
              {criticalIncomplete > 0 ? (
                <span className="text-orange-600 font-medium">
                  {criticalIncomplete} critical task{criticalIncomplete !== 1 ? 's' : ''} remaining
                </span>
              ) : (
                <span className="text-green-600">All critical tasks completed! 🎉</span>
              )}
              {recommendedIncomplete > 0 && (
                <span className="text-gray-500 ml-2">
                  · {recommendedIncomplete} recommended task{recommendedIncomplete !== 1 ? 's' : ''} remaining
                </span>
              )}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          {setupStatus.all_critical_complete && (
            <button
              onClick={(e) => {
                e.stopPropagation();
                handleDismiss();
              }}
              className="text-sm text-gray-500 hover:text-gray-700 px-3 py-1 rounded hover:bg-gray-100"
            >
              Dismiss
            </button>
          )}
          <ChevronRight
            className={`h-5 w-5 text-gray-400 transition-transform ${
              isExpanded ? 'rotate-90' : ''
            }`}
          />
        </div>
      </div>

      {isExpanded && (
        <div className="border-t border-gray-200 p-4 space-y-3">
          {setupStatus.tasks.map((task, index) => (
            <SetupTask key={index} task={task} />
          ))}

          <div className="mt-4 pt-4 border-t border-gray-100">
            <a
              href="/docs/getting-started"
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-2 text-sm text-blue-600 hover:text-blue-700 font-medium"
            >
              View Complete Setup Guide
              <ExternalLink className="h-4 w-4" />
            </a>
          </div>
        </div>
      )}
    </div>
  );
};

const SetupTask = ({ task }) => {
  const Icon = task.completed ? CheckCircle2 : Circle;
  const priorityColors = {
    critical: 'text-red-600',
    recommended: 'text-blue-600',
    optional: 'text-gray-500'
  };

  return (
    <div className={`flex items-start gap-3 p-3 rounded-lg ${
      task.completed ? 'bg-gray-50' : 'bg-white border border-gray-200'
    }`}>
      <Icon className={`h-5 w-5 flex-shrink-0 mt-0.5 ${
        task.completed ? 'text-green-500' : 'text-gray-300'
      }`} />
      <div className="flex-1 min-w-0">
        <div className="flex items-start justify-between gap-2">
          <div>
            <h4 className={`text-sm font-medium ${
              task.completed ? 'text-gray-500 line-through' : 'text-gray-900'
            }`}>
              {task.title}
            </h4>
            <p className="text-sm text-gray-600 mt-1">{task.description}</p>
          </div>
          {!task.completed && (
            <span className={`text-xs font-medium px-2 py-1 rounded-full whitespace-nowrap ${
              task.priority === 'critical' 
                ? 'bg-red-100 text-red-700'
                : task.priority === 'recommended'
                ? 'bg-blue-100 text-blue-700'
                : 'bg-gray-100 text-gray-600'
            }`}>
              {task.priority}
            </span>
          )}
        </div>
        {!task.completed && task.action_url && (
          <a
            href={task.action_url}
            className="inline-flex items-center gap-1 text-sm text-blue-600 hover:text-blue-700 font-medium mt-2"
          >
            {task.action_label || 'Take Action'}
            <ChevronRight className="h-4 w-4" />
          </a>
        )}
      </div>
    </div>
  );
};

export default SetupChecklist;
