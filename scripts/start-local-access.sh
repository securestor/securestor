#!/bin/bash
set -e

echo "=== SecureStor Local Access Manager ==="
echo "This script maintains stable port-forwards for local development"
echo ""

# Configuration
NAMESPACE="securestor"
FRONTEND_PORT=3000
API_PORT=8080
LOG_DIR="/tmp/securestor-pf"

# Create log directory
mkdir -p "$LOG_DIR"

# Function to check if port-forward is healthy
check_port_forward() {
    local port=$1
    nc -z localhost "$port" 2>/dev/null
    return $?
}

# Function to start port-forward
start_port_forward() {
    local service=$1
    local port=$2
    local name=$3
    
    echo "Starting port-forward for $name on port $port..."
    kubectl port-forward -n "$NAMESPACE" "svc/$service" "$port:$port" > "$LOG_DIR/$name.log" 2>&1 &
    local pid=$!
    echo $pid > "$LOG_DIR/$name.pid"
    echo "  PID: $pid"
    sleep 2
    
    if check_port_forward "$port"; then
        echo "  ✓ $name is accessible on http://localhost:$port"
        return 0
    else
        echo "  ✗ Failed to start $name port-forward"
        return 1
    fi
}

# Function to stop existing port-forwards
stop_port_forwards() {
    echo "Stopping existing port-forwards..."
    pkill -f "port-forward.*securestor" || true
    sleep 1
}

# Function to monitor and restart port-forwards
monitor_port_forwards() {
    echo ""
    echo "=== Monitoring port-forwards (Ctrl+C to stop) ==="
    echo "Frontend: http://localhost:$FRONTEND_PORT"
    echo "API:      http://localhost:$API_PORT"
    echo ""
    
    while true; do
        # Check frontend
        if ! check_port_forward "$FRONTEND_PORT"; then
            echo "[$(date '+%H:%M:%S')] Frontend port-forward died, restarting..."
            start_port_forward "frontend-service" "$FRONTEND_PORT" "frontend" || true
        fi
        
        # Check API
        if ! check_port_forward "$API_PORT"; then
            echo "[$(date '+%H:%M:%S')] API port-forward died, restarting..."
            start_port_forward "api-service" "$API_PORT" "api" || true
        fi
        
        sleep 5
    done
}

# Cleanup function
cleanup() {
    echo ""
    echo "Cleaning up..."
    stop_port_forwards
    rm -f "$LOG_DIR"/*.pid
    echo "Stopped all port-forwards"
    exit 0
}

# Register cleanup on exit
trap cleanup EXIT INT TERM

# Main execution
stop_port_forwards

echo ""
start_port_forward "frontend-service" "$FRONTEND_PORT" "frontend"
start_port_forward "api-service" "$API_PORT" "api"

monitor_port_forwards
