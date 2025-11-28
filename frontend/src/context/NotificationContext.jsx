import React, { createContext, useContext, useState, useEffect, useCallback } from 'react';
import NotificationsAPI from '../services/api/notificationsAPI';

const NotificationContext = createContext();

export const useNotifications = () => {
  const context = useContext(NotificationContext);
  if (!context) {
    throw new Error('useNotifications must be used within NotificationProvider');
  }
  return context;
};

/**
 * Enterprise Notification Provider
 * Global state management for notifications across the application
 * Features: Real-time updates, unread count tracking, WebSocket support
 */
export const NotificationProvider = ({ children }) => {
  const [unreadCount, setUnreadCount] = useState(0);
  const [recentNotifications, setRecentNotifications] = useState([]);
  const [wsConnection, setWsConnection] = useState(null);
  const [isConnected, setIsConnected] = useState(false);

  // Fetch unread count on mount
  useEffect(() => {
    fetchUnreadCount();
    fetchRecentNotifications();
    setupWebSocket();

    return () => {
      if (wsConnection) {
        wsConnection.close();
      }
    };
  }, []);

  const fetchUnreadCount = useCallback(async () => {
    try {
      const count = await NotificationsAPI.getUnreadCount();
      setUnreadCount(count);
    } catch (error) {
      console.error('Failed to fetch unread count:', error);
    }
  }, []);

  const fetchRecentNotifications = useCallback(async () => {
    try {
      const response = await NotificationsAPI.getNotifications({
        page: 1,
        limit: 10,
        unread: false
      });
      setRecentNotifications(response.notifications || []);
    } catch (error) {
      console.error('Failed to fetch recent notifications:', error);
    }
  }, []);

  const setupWebSocket = useCallback(() => {
    try {
      // TODO: Implement WebSocket connection
      // const unsubscribe = NotificationsAPI.subscribeToNotifications((notification) => {
      //   handleNewNotification(notification);
      // });
      // setWsConnection(unsubscribe);
      // setIsConnected(true);
    } catch (error) {
      console.error('Failed to setup WebSocket:', error);
      setIsConnected(false);
    }
  }, []);

  const handleNewNotification = useCallback((notification) => {
    // Add new notification to recent list
    setRecentNotifications(prev => [notification, ...prev].slice(0, 10));
    
    // Increment unread count if notification is unread
    if (!notification.read) {
      setUnreadCount(prev => prev + 1);
    }

    // Optional: Show browser notification
    if (Notification.permission === 'granted') {
      new Notification(notification.title, {
        body: notification.message,
        icon: '/logo.png',
        badge: '/logo.png'
      });
    }
  }, []);

  const markAsRead = useCallback(async (notificationId) => {
    try {
      await NotificationsAPI.markAsRead(notificationId);
      
      // Update local state
      setRecentNotifications(prev =>
        prev.map(n => n.id === notificationId ? { ...n, read: true } : n)
      );
      
      // Decrement unread count
      setUnreadCount(prev => Math.max(0, prev - 1));
    } catch (error) {
      console.error('Failed to mark notification as read:', error);
      throw error;
    }
  }, []);

  const markMultipleAsRead = useCallback(async (notificationIds) => {
    try {
      await NotificationsAPI.markMultipleAsRead(notificationIds);
      
      // Update local state
      setRecentNotifications(prev =>
        prev.map(n => notificationIds.includes(n.id) ? { ...n, read: true } : n)
      );
      
      // Recalculate unread count
      await fetchUnreadCount();
    } catch (error) {
      console.error('Failed to mark notifications as read:', error);
      throw error;
    }
  }, [fetchUnreadCount]);

  const markAllAsRead = useCallback(async () => {
    try {
      await NotificationsAPI.markAllAsRead();
      
      // Update all notifications to read
      setRecentNotifications(prev => prev.map(n => ({ ...n, read: true })));
      setUnreadCount(0);
    } catch (error) {
      console.error('Failed to mark all as read:', error);
      throw error;
    }
  }, []);

  const deleteNotification = useCallback(async (notificationId) => {
    try {
      await NotificationsAPI.deleteNotification(notificationId);
      
      // Remove from local state
      setRecentNotifications(prev => {
        const notification = prev.find(n => n.id === notificationId);
        const newNotifications = prev.filter(n => n.id !== notificationId);
        
        // Decrement unread count if deleted notification was unread
        if (notification && !notification.read) {
          setUnreadCount(prevCount => Math.max(0, prevCount - 1));
        }
        
        return newNotifications;
      });
    } catch (error) {
      console.error('Failed to delete notification:', error);
      throw error;
    }
  }, []);

  const deleteMultiple = useCallback(async (notificationIds) => {
    try {
      await NotificationsAPI.deleteMultiple(notificationIds);
      
      // Remove from local state
      setRecentNotifications(prev => {
        const deletedUnreadCount = prev
          .filter(n => notificationIds.includes(n.id) && !n.read)
          .length;
        
        setUnreadCount(prevCount => Math.max(0, prevCount - deletedUnreadCount));
        
        return prev.filter(n => !notificationIds.includes(n.id));
      });
    } catch (error) {
      console.error('Failed to delete notifications:', error);
      throw error;
    }
  }, []);

  const requestBrowserPermission = useCallback(async () => {
    if ('Notification' in window && Notification.permission === 'default') {
      try {
        const permission = await Notification.requestPermission();
        return permission === 'granted';
      } catch (error) {
        console.error('Failed to request notification permission:', error);
        return false;
      }
    }
    return Notification.permission === 'granted';
  }, []);

  const refresh = useCallback(async () => {
    await Promise.all([
      fetchUnreadCount(),
      fetchRecentNotifications()
    ]);
  }, [fetchUnreadCount, fetchRecentNotifications]);

  const value = {
    // State
    unreadCount,
    recentNotifications,
    isConnected,
    
    // Actions
    markAsRead,
    markMultipleAsRead,
    markAllAsRead,
    deleteNotification,
    deleteMultiple,
    refresh,
    requestBrowserPermission,
    
    // Internal methods (exposed for advanced use)
    fetchUnreadCount,
    fetchRecentNotifications
  };

  return (
    <NotificationContext.Provider value={value}>
      {children}
    </NotificationContext.Provider>
  );
};

export default NotificationContext;
