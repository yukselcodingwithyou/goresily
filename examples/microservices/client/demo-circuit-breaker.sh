#!/bin/bash

# Circuit Breaker Demo Script
# This script demonstrates the circuit breaker pattern

echo "=========================================="
echo "🔄 CIRCUIT BREAKER DEMONSTRATION"
echo "=========================================="
echo
echo "This demo will show:"
echo "- 🟢 Closed state (normal operation)"  
echo "- 🔴 Open state (blocking requests after failures)"
echo "- 🟡 Half-Open state (testing recovery)"
echo

echo "1. Starting server with high failure rate..."
cd ../server
./server -failure-rate=0.7 -slow-rate=0.1 &
SERVER_PID=$!
sleep 2

echo "2. Running circuit breaker client..."
cd ../client
./client -mode=cb -requests=12 -interval=600ms

echo
echo "3. Waiting for circuit breaker to transition to half-open..."
sleep 8

echo "4. Fixing server (reducing failure rate)..."
curl -s -X POST http://localhost:8080/config \
  -H "Content-Type: application/json" \
  -d '{"failure_rate":0.1,"slow_response_rate":0.1,"slow_delay":500}' > /dev/null

echo "5. Running more requests to see recovery..."
./client -mode=cb -requests=8 -interval=500ms

echo
echo "✅ Circuit breaker demo completed!"
echo "🛑 Stopping server..."
kill $SERVER_PID 2>/dev/null
wait $SERVER_PID 2>/dev/null