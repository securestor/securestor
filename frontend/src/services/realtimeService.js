/**
 * Enterprise-grade Real-time Service
 * Manages WebSocket and SSE connections for live UI updates
 */

class RealtimeService {
  constructor() {
    // Use the API base URL from environment or default to localhost
    // In Docker HA mode, this will be http://localhost (port 80 through nginx)
    this.baseURL = process.env.REACT_APP_API_URL || 'http://localhost/api/v1';
    // Extract the base WS URL properly, keeping the port
    const httpUrl = this.baseURL.replace('/api/v1', '');
    this.wsURL = httpUrl.replace(/^http/, 'ws');
    this.subscribers = new Map();
    this.eventSource = null;
    this.ws = null;
    this.reconnectAttempts = 0;
    this.maxReconnectAttempts = 5;
    this.reconnectDelay = 1000; // Start with 1 second
    this.heartbeatInterval = null;
    this.isConnected = false;
  }

  /**
   * Initialize real-time connection
   */
  connect() {
    if (this.isConnected) {
      return;
    }

    // Try WebSocket first, fallback to SSE
    this.connectWebSocket();
  }

  /**
   * Connect via WebSocket for bidirectional communication
   */
  connectWebSocket() {
    try {
      this.ws = new WebSocket(`${this.wsURL}/ws/updates`);

      this.ws.onopen = () => {
        this.isConnected = true;
        this.reconnectAttempts = 0;
        this.reconnectDelay = 1000;
        this.startHeartbeat();
        this.notifySubscribers('connection', { status: 'connected', type: 'websocket' });
      };

      this.ws.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data);
          this.handleMessage(message);
        } catch (error) {
          console.error('[RealtimeService] Error parsing WebSocket message:', error);
        }
      };

      this.ws.onerror = (error) => {
        console.error('[RealtimeService] WebSocket error:', error);
        this.notifySubscribers('error', { error: 'WebSocket connection error' });
      };

      this.ws.onclose = () => {
        this.isConnected = false;
        this.stopHeartbeat();
        this.notifySubscribers('connection', { status: 'disconnected' });
        
        // Fallback to SSE if WebSocket fails
        if (this.reconnectAttempts === 0) {
          this.connectSSE();
        } else {
          this.attemptReconnect();
        }
      };
    } catch (error) {
      console.error('[RealtimeService] Error creating WebSocket:', error);
      this.connectSSE();
    }
  }

  /**
   * Connect via Server-Sent Events as fallback
   */
  connectSSE() {
    try {
      this.eventSource = new EventSource(`${this.baseURL}/events/stream`);

      this.eventSource.onopen = () => {
        this.isConnected = true;
        this.reconnectAttempts = 0;
        this.notifySubscribers('connection', { status: 'connected', type: 'sse' });
      };

      this.eventSource.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data);
          this.handleMessage(message);
        } catch (error) {
          console.error('[RealtimeService] Error parsing SSE message:', error);
        }
      };

      this.eventSource.onerror = (error) => {
        console.error('[RealtimeService] SSE error:', error);
        this.isConnected = false;
        this.eventSource?.close();
        this.attemptReconnect();
      };
    } catch (error) {
      console.error('[RealtimeService] Error creating SSE:', error);
      this.attemptReconnect();
    }
  }

  /**
   * Handle incoming messages
   */
  handleMessage(message) {
    const { type, event, data } = message;
    

    // Notify specific event subscribers
    if (event) {
      this.notifySubscribers(event, data);
    }

    // Notify type subscribers (e.g., 'artifact', 'repository', 'scan')
    if (type) {
      this.notifySubscribers(type, { event, data });
    }

    // Notify all subscribers
    this.notifySubscribers('*', message);
  }

  /**
   * Notify all subscribers of an event
   */
  notifySubscribers(event, data) {
    const eventSubscribers = this.subscribers.get(event) || [];
    eventSubscribers.forEach(callback => {
      try {
        callback(data);
      } catch (error) {
        console.error(`[RealtimeService] Error in subscriber callback for ${event}:`, error);
      }
    });
  }

  /**
   * Subscribe to real-time events
   */
  subscribe(event, callback) {
    if (!this.subscribers.has(event)) {
      this.subscribers.set(event, []);
    }
    
    this.subscribers.get(event).push(callback);
    

    // Auto-connect if not connected
    if (!this.isConnected) {
      this.connect();
    }

    // Return unsubscribe function
    return () => {
      const callbacks = this.subscribers.get(event) || [];
      const index = callbacks.indexOf(callback);
      if (index > -1) {
        callbacks.splice(index, 1);
      }
    };
  }

  /**
   * Unsubscribe from an event
   */
  unsubscribe(event, callback) {
    const callbacks = this.subscribers.get(event);
    if (callbacks) {
      const index = callbacks.indexOf(callback);
      if (index > -1) {
        callbacks.splice(index, 1);
      }
    }
  }

  /**
   * Send message via WebSocket
   */
  send(message) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message));
      return true;
    }
    console.warn('[RealtimeService] Cannot send message, WebSocket not connected');
    return false;
  }

  /**
   * Attempt to reconnect
   */
  attemptReconnect() {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error('[RealtimeService] Max reconnection attempts reached');
      this.notifySubscribers('connection', { 
        status: 'failed', 
        message: 'Failed to establish real-time connection' 
      });
      return;
    }

    this.reconnectAttempts++;
    const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1); // Exponential backoff


    setTimeout(() => {
      if (this.ws) {
        this.connectWebSocket();
      } else {
        this.connectSSE();
      }
    }, delay);
  }

  /**
   * Start heartbeat to keep connection alive
   */
  startHeartbeat() {
    this.heartbeatInterval = setInterval(() => {
      if (this.ws && this.ws.readyState === WebSocket.OPEN) {
        this.ws.send(JSON.stringify({ type: 'ping' }));
      }
    }, 30000); // Every 30 seconds
  }

  /**
   * Stop heartbeat
   */
  stopHeartbeat() {
    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval);
      this.heartbeatInterval = null;
    }
  }

  /**
   * Disconnect and cleanup
   */
  disconnect() {
    
    this.stopHeartbeat();
    
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
    
    if (this.eventSource) {
      this.eventSource.close();
      this.eventSource = null;
    }
    
    this.isConnected = false;
    this.subscribers.clear();
  }

  /**
   * Get connection status
   */
  getStatus() {
    return {
      isConnected: this.isConnected,
      type: this.ws ? 'websocket' : this.eventSource ? 'sse' : 'none',
      subscribers: this.subscribers.size
    };
  }
}

// Export singleton instance
const realtimeService = new RealtimeService();
export default realtimeService;
