#!/bin/bash

# Script para testing rápido del workflow de aprobación de compras

set -e

echo "🛒 Purchase Approval Workflow Test Script"
echo "========================================"

# Check if Temporal is running
if ! nc -z localhost 7233; then
    echo "❌ Temporal server not running on localhost:7233"
    echo "Run: make temporal-up"
    exit 1
fi

echo "✅ Temporal server is running"

# Start worker in background if not running
if ! pgrep -f "cmd/worker/main.go" > /dev/null; then
    echo "🚀 Starting worker in background..."
    go run cmd/worker/main.go &
    WORKER_PID=$!
    echo "Worker PID: $WORKER_PID"
    sleep 3
else
    echo "✅ Worker is already running"
fi

# Start web server if not running
if ! nc -z localhost 8081; then
    echo "🌐 Starting web server in background..."
    go run cmd/web/main.go &
    WEB_PID=$!
    echo "Web server PID: $WEB_PID"
    sleep 3
else
    echo "✅ Web server is already running"
fi

echo ""
echo "🎯 System Ready!"
echo "==============="
echo "📝 Web Interface: http://localhost:8081"
echo "👁️  Temporal UI: http://localhost:8080"
echo ""
echo "🧪 Test Instructions:"
echo "1. Open http://localhost:8081 in your browser"
echo "2. Fill out the purchase request form"
echo "3. Submit and note the Request ID"
echo "4. Watch the worker logs for approval URL"
echo "5. Use approval URL to approve/reject"
echo ""
echo "📊 Monitor progress:"
echo "- Worker logs: tail -f worker output"
echo "- Temporal UI: http://localhost:8080"
echo "- Status page: auto-refreshes every 10s"

# Trap to cleanup on exit
cleanup() {
    if [ -n "$WORKER_PID" ]; then
        echo "🛑 Stopping worker (PID: $WORKER_PID)..."
        kill $WORKER_PID 2>/dev/null || true
    fi
    if [ -n "$WEB_PID" ]; then
        echo "🛑 Stopping web server (PID: $WEB_PID)..."
        kill $WEB_PID 2>/dev/null || true
    fi
}

trap cleanup EXIT

# Wait for user input
echo ""
echo "Press Ctrl+C to stop all services..."
wait