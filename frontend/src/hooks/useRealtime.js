/**
 * React hook for real-time updates
 */
import { useEffect, useCallback, useRef, useState } from 'react';
import realtimeService from '../services/realtimeService';

/**
 * Hook to subscribe to real-time events
 * @param {string|string[]} events - Event name(s) to subscribe to
 * @param {function} callback - Callback function when event is received
 * @param {object} options - Configuration options
 */
export const useRealtime = (events, callback, options = {}) => {
  const { enabled = true, autoConnect = true } = options;
  const callbackRef = useRef(callback);
  const [connectionStatus, setConnectionStatus] = useState({
    isConnected: false,
    type: 'none'
  });

  // Update callback ref when it changes
  useEffect(() => {
    callbackRef.current = callback;
  }, [callback]);

  useEffect(() => {
    if (!enabled) return;

    const eventList = Array.isArray(events) ? events : [events];
    const unsubscribeFns = [];

    // Subscribe to connection status
    const unsubscribeConnection = realtimeService.subscribe('connection', (data) => {
      setConnectionStatus({
        isConnected: data.status === 'connected',
        type: data.type || 'none'
      });
    });
    unsubscribeFns.push(unsubscribeConnection);

    // Subscribe to each event
    eventList.forEach(event => {
      const unsubscribe = realtimeService.subscribe(event, (data) => {
        if (callbackRef.current) {
          callbackRef.current(data, event);
        }
      });
      unsubscribeFns.push(unsubscribe);
    });

    // Auto-connect if enabled
    if (autoConnect && !realtimeService.isConnected) {
      realtimeService.connect();
    }

    // Cleanup on unmount
    return () => {
      unsubscribeFns.forEach(fn => fn());
    };
  }, [events, enabled, autoConnect]);

  return connectionStatus;
};

/**
 * Hook for repository updates
 */
export const useRepositoryUpdates = (callback, options = {}) => {
  return useRealtime([
    'repository.created',
    'repository.updated',
    'repository.deleted'
  ], callback, options);
};

/**
 * Hook for artifact updates
 */
export const useArtifactUpdates = (callback, options = {}) => {
  return useRealtime([
    'artifact.uploaded',
    'artifact.updated',
    'artifact.deleted',
    'artifact.scan.started',
    'artifact.scan.progress',
    'artifact.scan.completed',
    'artifact.scan.failed'
  ], callback, options);
};

/**
 * Hook for scan updates
 */
export const useScanUpdates = (callback, options = {}) => {
  return useRealtime([
    'scan.initiated',
    'scan.started',
    'scan.progress',
    'scan.completed',
    'scan.failed'
  ], callback, options);
};

/**
 * Hook for compliance updates
 */
export const useComplianceUpdates = (callback, options = {}) => {
  return useRealtime([
    'compliance.updated',
    'compliance.violation',
    'compliance.resolved'
  ], callback, options);
};

/**
 * Hook for user updates
 */
export const useUserUpdates = (callback, options = {}) => {
  return useRealtime([
    'user.created',
    'user.updated',
    'user.deleted',
    'user.invited'
  ], callback, options);
};

/**
 * Hook for notification updates
 */
export const useNotificationUpdates = (callback, options = {}) => {
  return useRealtime([
    'notification.new',
    'notification.read',
    'notification.deleted'
  ], callback, options);
};

/**
 * Hook to send messages via WebSocket
 */
export const useRealtimeSend = () => {
  return useCallback((message) => {
    return realtimeService.send(message);
  }, []);
};

/**
 * Hook to get connection status
 */
export const useConnectionStatus = () => {
  const [status, setStatus] = useState(realtimeService.getStatus());

  useEffect(() => {
    const unsubscribe = realtimeService.subscribe('connection', () => {
      setStatus(realtimeService.getStatus());
    });

    return () => unsubscribe();
  }, []);

  return status;
};
