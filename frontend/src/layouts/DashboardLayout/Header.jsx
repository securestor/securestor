import React, { useState, useEffect, useRef } from 'react';
import { Package, Activity, Settings, Bell, Plus, LogOut, User } from 'lucide-react';
import { Badge } from '../../components/common';
import { CreateRepositoryModal } from '../../components/modals';
import { NotificationsCenter } from '../../components/notifications/NotificationsCenter';
import { useToast } from '../../context/ToastContext';
import { useAuth } from '../../context/AuthContext';
import { useDashboard } from '../../context/DashboardContext';
import NotificationsAPI from '../../services/api/notificationsAPI';
import { useTranslation } from '../../hooks/useTranslation';
import LanguageSwitcher from '../../components/LanguageSwitcher';

export const Header = () => {
  const { t } = useTranslation('common');
  const [showCreateRepo, setShowCreateRepo] = useState(false);
  const [showNotifications, setShowNotifications] = useState(false);
  const [showNotificationsCenter, setShowNotificationsCenter] = useState(false);
  const [showUserMenu, setShowUserMenu] = useState(false);
  const [recentNotifications, setRecentNotifications] = useState([]);
  const [unreadCount, setUnreadCount] = useState(0);
  const { showSuccess } = useToast();
  const { user, logout } = useAuth();
  const { setActiveTab, setSelectedRepo, addRepository, refreshRepositories } = useDashboard();

  const notificationsRef = useRef(null);
  const userMenuRef = useRef(null);

  // Close dropdowns when clicking outside
  useEffect(() => {
    const handleClickOutside = (event) => {
      if (notificationsRef.current && !notificationsRef.current.contains(event.target)) {
        setShowNotifications(false);
      }
      if (userMenuRef.current && !userMenuRef.current.contains(event.target)) {
        setShowUserMenu(false);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const handleCreateRepository = async (response) => {
    
    // Add the new repository to global state for immediate update
    if (response && response.repository) {
      addRepository(response.repository);
    }
    
    // Refresh from server to ensure consistency
    await refreshRepositories();
    
    const repoName = response?.repository?.name || 'Repository';
    showSuccess(`Repository "${repoName}" created successfully!`);
  };

  const handleProfileClick = () => {
    setActiveTab('profile');
    setSelectedRepo(null); // Clear any selected repository
    setShowUserMenu(false); // Close the user menu
    showSuccess('Opening profile settings...'); // Give user feedback
  };

  // Fetch recent notifications on mount and set up real-time updates
  useEffect(() => {
    fetchRecentNotifications();
    fetchUnreadCount();

    // TODO: Set up WebSocket for real-time updates
    // const unsubscribe = NotificationsAPI.subscribeToNotifications((notification) => {
    //   setRecentNotifications(prev => [notification, ...prev].slice(0, 5));
    //   fetchUnreadCount();
    // });
    // return () => unsubscribe();
  }, []);

  const fetchRecentNotifications = async () => {
    try {
      // TODO: Replace with actual API call
      // const response = await NotificationsAPI.getNotifications({ limit: 5, page: 1 });
      // setRecentNotifications(response.notifications);
      
      // Mock data for now
      setRecentNotifications([
        { 
          id: 1, 
          title: 'New artifact deployed',
          message: 'Successfully deployed to docker-local', 
          timestamp: new Date(Date.now() - 1000 * 60 * 5).toISOString(), 
          read: false,
          type: 'artifact',
          priority: 'info'
        },
        { 
          id: 2, 
          title: 'Security scan completed',
          message: 'Scan completed for maven-central - 2 vulnerabilities found', 
          timestamp: new Date(Date.now() - 1000 * 60 * 60).toISOString(), 
          read: false,
          type: 'scan',
          priority: 'high'
        },
        { 
          id: 3, 
          title: 'Storage quota warning',
          message: 'Storage quota at 75% for npm-remote', 
          timestamp: new Date(Date.now() - 1000 * 60 * 180).toISOString(), 
          read: true,
          type: 'system',
          priority: 'medium'
        }
      ]);
    } catch (error) {
      console.error('Failed to fetch notifications:', error);
    }
  };

  const fetchUnreadCount = async () => {
    try {
      // TODO: Replace with actual API call
      // const count = await NotificationsAPI.getUnreadCount();
      // setUnreadCount(count);
      setUnreadCount(2); // Mock data
    } catch (error) {
      console.error('Failed to fetch unread count:', error);
    }
  };

  const handleNotificationClick = async (notificationId, read) => {
    if (!read) {
      try {
        // await NotificationsAPI.markAsRead(notificationId);
        setRecentNotifications(prev => 
          prev.map(n => n.id === notificationId ? { ...n, read: true } : n)
        );
        setUnreadCount(prev => Math.max(0, prev - 1));
      } catch (error) {
        console.error('Failed to mark as read:', error);
      }
    }
  };

  const formatTimestamp = (timestamp) => {
    const date = new Date(timestamp);
    const now = new Date();
    const diff = now - date;
    
    const minutes = Math.floor(diff / 60000);
    const hours = Math.floor(diff / 3600000);
    const days = Math.floor(diff / 86400000);
    
    if (minutes < 1) return 'Just now';
    if (minutes < 60) return `${minutes}m ago`;
    if (hours < 24) return `${hours}h ago`;
    if (days < 7) return `${days}d ago`;
    return date.toLocaleDateString();
  };

  return (
    <>
      <header className="bg-white border-b border-gray-200 sticky top-0 z-10">
        <div className="px-6 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-4">
              <div className="flex items-center space-x-2">
                <Package className="w-8 h-8 text-blue-600" />
                <h1 className="text-2xl font-bold text-gray-900">SecureStor</h1>
              </div>
              {/* <Badge variant="info">Enterprise</Badge> */}
            </div>
            
            <div className="flex items-center space-x-3">
              {/* Quick Create Button */}
              <button
                onClick={() => setShowCreateRepo(true)}
                className="flex items-center space-x-2 px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 transition"
              >
                <Plus className="w-4 h-4" />
                <span>{t('repositories:repositoryActions.create')}</span>
              </button>

              {/* Notifications */}
              <div className="relative" ref={notificationsRef}>
                <button
                  onClick={() => setShowNotifications(!showNotifications)}
                  className="p-2 hover:bg-gray-100 rounded-lg transition relative"
                >
                  <Bell className="w-5 h-5 text-gray-600" />
                  {unreadCount > 0 && (
                    <span className="absolute top-0.5 right-0.5 min-w-[18px] h-[18px] bg-red-500 text-white text-xs font-semibold rounded-full flex items-center justify-center px-1">
                      {unreadCount > 99 ? '99+' : unreadCount}
                    </span>
                  )}
                </button>

                {/* Notifications Dropdown - Enhanced */}
                {showNotifications && (
                  <div className="absolute right-0 mt-2 w-96 bg-white rounded-lg shadow-xl border border-gray-200 z-50">
                    <div className="px-4 py-3 border-b border-gray-200 bg-gradient-to-r from-blue-50 to-indigo-50">
                      <div className="flex items-center justify-between">
                        <h3 className="font-semibold text-gray-900">{t('notifications:center.title')}</h3>
                        {unreadCount > 0 && (
                          <span className="px-2 py-0.5 bg-red-100 text-red-800 text-xs font-semibold rounded-full">
                            {t('notifications:center.unread', { count: unreadCount })}
                          </span>
                        )}
                      </div>
                    </div>
                    <div className="max-h-[400px] overflow-y-auto">
                      {recentNotifications.length === 0 ? (
                        <div className="flex flex-col items-center justify-center py-8 text-gray-500">
                          <Bell className="w-12 h-12 mb-2 opacity-50" />
                          <p className="text-sm">{t('notifications:center.noNotifications')}</p>
                        </div>
                      ) : (
                        recentNotifications.map(notification => (
                          <div
                            key={notification.id}
                            onClick={() => handleNotificationClick(notification.id, notification.read)}
                            className={`px-4 py-3 hover:bg-gray-50 cursor-pointer border-b border-gray-100 transition ${
                              !notification.read ? 'bg-blue-50 border-l-4 border-l-blue-500' : ''
                            }`}
                          >
                            <div className="flex items-start space-x-3">
                              {!notification.read && (
                                <span className="w-2 h-2 mt-1.5 bg-blue-600 rounded-full"></span>
                              )}
                              <div className="flex-1 min-w-0">
                                <p className="text-sm font-semibold text-gray-900 mb-0.5">
                                  {notification.title}
                                </p>
                                <p className="text-sm text-gray-700 mb-1 line-clamp-2">
                                  {notification.message}
                                </p>
                                <div className="flex items-center justify-between">
                                  <span className="text-xs text-gray-500">
                                    {formatTimestamp(notification.timestamp)}
                                  </span>
                                  {notification.priority && notification.priority !== 'info' && (
                                    <span className={`text-xs font-medium ${
                                      notification.priority === 'critical' ? 'text-red-600' :
                                      notification.priority === 'high' ? 'text-orange-600' :
                                      notification.priority === 'medium' ? 'text-yellow-600' :
                                      'text-blue-600'
                                    }`}>
                                      {notification.priority.toUpperCase()}
                                    </span>
                                  )}
                                </div>
                              </div>
                            </div>
                          </div>
                        ))
                      )}
                    </div>
                    <div className="px-4 py-3 text-center border-t border-gray-200 bg-gray-50">
                      <button 
                        onClick={() => {
                          setShowNotifications(false);
                          setShowNotificationsCenter(true);
                        }}
                        className="text-sm text-blue-600 hover:text-blue-800 font-medium transition"
                      >
                        {t('notifications:center.showAll')} →
                      </button>
                    </div>
                  </div>
                )}
              </div>

              <button className="p-2 hover:bg-gray-100 rounded-lg transition">
                <Activity className="w-5 h-5 text-gray-600" />
              </button>
              
              <button className="p-2 hover:bg-gray-100 rounded-lg transition">
                <Settings className="w-5 h-5 text-gray-600" />
              </button>
              
              {/* Language Switcher */}
              <LanguageSwitcher />
              
              {/* User Menu */}
              <div className="relative" ref={userMenuRef}>
                <button
                  onClick={() => setShowUserMenu(!showUserMenu)}
                  className="flex items-center space-x-2 p-2 hover:bg-gray-100 rounded-lg transition"
                >
                  <div className="w-8 h-8 bg-gradient-to-br from-blue-500 to-purple-600 rounded-full flex items-center justify-center text-white font-semibold">
                    {user?.first_name?.charAt(0) || user?.username?.charAt(0) || 'U'}
                  </div>
                  <span className="text-sm font-medium text-gray-700 hidden md:block">
                    {user?.first_name ? `${user.first_name} ${user.last_name || ''}`.trim() : user?.username}
                  </span>
                </button>

                {/* User Dropdown */}
                {showUserMenu && (
                  <div className="absolute right-0 mt-2 w-56 bg-white rounded-lg shadow-lg border border-gray-200 z-50">
                    <div className="px-4 py-3 border-b border-gray-200">
                      <p className="text-sm font-medium text-gray-900">
                        {user?.first_name ? `${user.first_name} ${user.last_name || ''}`.trim() : user?.username}
                      </p>
                      <p className="text-sm text-gray-500">{user?.email}</p>
                    </div>
                    <div className="py-1">
                      <button 
                        onClick={handleProfileClick}
                        className="flex items-center px-4 py-2 text-sm text-gray-700 hover:bg-gray-100 w-full text-left"
                      >
                        <User className="w-4 h-4 mr-3" />
                        {t('navigation.profile')}
                      </button>
                      <button
                        onClick={() => {
                          logout();
                          setShowUserMenu(false);
                        }}
                        className="flex items-center px-4 py-2 text-sm text-gray-700 hover:bg-gray-100 w-full text-left"
                      >
                        <LogOut className="w-4 h-4 mr-3" />
                        {t('navigation.logout')}
                      </button>
                    </div>
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>
      </header>

      {/* Create Repository Modal */}
      <CreateRepositoryModal
        isOpen={showCreateRepo}
        onClose={() => setShowCreateRepo(false)}
        onSubmit={handleCreateRepository}
      />

      {/* Notifications Center */}
      <NotificationsCenter
        isOpen={showNotificationsCenter}
        onClose={() => {
          setShowNotificationsCenter(false);
          fetchRecentNotifications();
          fetchUnreadCount();
        }}
      />
    </>
  );
};