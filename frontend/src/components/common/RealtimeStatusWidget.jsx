/**
 * Real-time Connection Status Component
 * Displays connection status and recent events for debugging/demo
 */
import React, { useState, useEffect } from 'react';
import { Wifi, WifiOff, Activity, X } from 'lucide-react';
import { useConnectionStatus, useRealtime } from '../../hooks/useRealtime';

export const RealtimeStatusWidget = ({ position = 'bottom-right' }) => {
  const [events, setEvents] = useState([]);
  const [isExpanded, setIsExpanded] = useState(false);
  const [isVisible, setIsVisible] = useState(true);
  const status = useConnectionStatus();

  // Subscribe to all events for debugging
  useRealtime('*', (message) => {
    const newEvent = {
      id: Date.now(),
      type: message.type || 'unknown',
      event: message.event || 'unknown',
      timestamp: new Date().toLocaleTimeString(),
      data: message.data || message
    };
    
    setEvents(prev => [newEvent, ...prev].slice(0, 20)); // Keep last 20 events
  }, { enabled: isExpanded });

  const positionClasses = {
    'bottom-right': 'bottom-4 right-4',
    'bottom-left': 'bottom-4 left-4',
    'top-right': 'top-4 right-4',
    'top-left': 'top-4 left-4'
  };

  if (!isVisible) return null;

  return (
    <div className={`fixed ${positionClasses[position]} z-50`}>
      {/* Collapsed State - Connection Indicator */}
      {!isExpanded && (
        <button
          onClick={() => setIsExpanded(true)}
          className={`flex items-center space-x-2 px-4 py-2 rounded-full shadow-lg transition-all ${
            status.isConnected
              ? 'bg-green-500 hover:bg-green-600 text-white'
              : 'bg-gray-400 hover:bg-gray-500 text-white'
          }`}
        >
          {status.isConnected ? (
            <>
              <Wifi className="w-4 h-4" />
              <span className="text-sm font-medium">Live</span>
              <div className="w-2 h-2 bg-white rounded-full animate-pulse" />
            </>
          ) : (
            <>
              <WifiOff className="w-4 h-4" />
              <span className="text-sm font-medium">Offline</span>
            </>
          )}
        </button>
      )}

      {/* Expanded State - Event Monitor */}
      {isExpanded && (
        <div className="bg-white rounded-lg shadow-2xl border border-gray-200 w-96 max-h-96 flex flex-col">
          {/* Header */}
          <div className={`px-4 py-3 rounded-t-lg flex items-center justify-between ${
            status.isConnected ? 'bg-green-500' : 'bg-gray-400'
          }`}>
            <div className="flex items-center space-x-2 text-white">
              {status.isConnected ? (
                <Wifi className="w-5 h-5" />
              ) : (
                <WifiOff className="w-5 h-5" />
              )}
              <div>
                <h3 className="font-semibold text-sm">Real-Time Monitor</h3>
                <p className="text-xs opacity-90">
                  {status.isConnected 
                    ? `Connected via ${status.type}` 
                    : 'Disconnected'}
                </p>
              </div>
            </div>
            <div className="flex items-center space-x-2">
              <button
                onClick={() => setIsVisible(false)}
                className="text-white hover:bg-white hover:bg-opacity-20 p-1 rounded"
                title="Hide widget"
              >
                <X className="w-4 h-4" />
              </button>
              <button
                onClick={() => setIsExpanded(false)}
                className="text-white hover:bg-white hover:bg-opacity-20 px-2 py-1 rounded text-sm"
              >
                Minimize
              </button>
            </div>
          </div>

          {/* Events List */}
          <div className="flex-1 overflow-y-auto p-3 space-y-2 bg-gray-50 max-h-80">
            {events.length === 0 ? (
              <div className="text-center py-8 text-gray-500">
                <Activity className="w-8 h-8 mx-auto mb-2 opacity-50" />
                <p className="text-sm">Waiting for events...</p>
                <p className="text-xs mt-1">
                  Try uploading an artifact or starting a scan
                </p>
              </div>
            ) : (
              events.map(event => (
                <div
                  key={event.id}
                  className="bg-white border border-gray-200 rounded p-2 text-xs hover:shadow-md transition-shadow"
                >
                  <div className="flex items-center justify-between mb-1">
                    <span className={`font-semibold px-2 py-0.5 rounded text-white ${
                      event.type === 'artifact' ? 'bg-blue-500' :
                      event.type === 'scan' ? 'bg-purple-500' :
                      event.type === 'repository' ? 'bg-green-500' :
                      event.type === 'compliance' ? 'bg-yellow-500' :
                      'bg-gray-500'
                    }`}>
                      {event.type}
                    </span>
                    <span className="text-gray-500">{event.timestamp}</span>
                  </div>
                  <div className="text-gray-700 font-medium">{event.event}</div>
                  <div className="mt-1 text-gray-600 break-all">
                    {JSON.stringify(event.data, null, 2).slice(0, 100)}
                    {JSON.stringify(event.data).length > 100 && '...'}
                  </div>
                </div>
              ))
            )}
          </div>

          {/* Footer */}
          <div className="px-4 py-2 bg-gray-100 border-t border-gray-200 rounded-b-lg flex items-center justify-between">
            <span className="text-xs text-gray-600">
              {events.length} event{events.length !== 1 ? 's' : ''} received
            </span>
            <button
              onClick={() => setEvents([])}
              className="text-xs text-blue-600 hover:text-blue-700 font-medium"
            >
              Clear
            </button>
          </div>
        </div>
      )}
    </div>
  );
};

export default RealtimeStatusWidget;
