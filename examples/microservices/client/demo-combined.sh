#!/bin/bash

# Combined Patterns Demo Script
# This script demonstrates circuit breaker + bulkhead working together

echo "=========================================="
echo "🎭 COMBINED PATTERNS DEMONSTRATION"
echo "=========================================="
echo
echo "This demo will show:"
echo "- Circuit breaker AND bulkhead working together"
echo "- How bulkhead limits concurrent requests"
echo "- How circuit breaker prevents cascade failures"
echo "- Realistic microservice resilience patterns"
echo

echo "1. Starting server with mixed failure patterns..."
cd ../server
./server -failure-rate=0.4 -slow-rate=0.4 -slow-delay=1500 &
SERVER_PID=$!
sleep 2

echo "2. Running combined patterns client..."
cd ../client
./client -mode=both -requests=6 -workers=4 -interval=700ms

echo
echo "✅ Combined patterns demo completed!"
echo "🛑 Stopping server..."
kill $SERVER_PID 2>/dev/null
wait $SERVER_PID 2>/dev/null