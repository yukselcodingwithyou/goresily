#!/bin/bash

# Bulkhead Demo Script
# This script demonstrates the bulkhead pattern

echo "=========================================="
echo "🚧 BULKHEAD DEMONSTRATION"
echo "=========================================="
echo
echo "This demo will show:"
echo "- Concurrent request limiting"
echo "- 'bulkhead full' errors when limit exceeded"
echo "- Protection against resource exhaustion"
echo

echo "1. Starting server with slow responses..."
cd ../server
./server -failure-rate=0.1 -slow-rate=0.9 -slow-delay=2000 &
SERVER_PID=$!
sleep 2

echo "2. Running bulkhead client with many concurrent workers..."
cd ../client
./client -mode=bh -requests=4 -workers=6 -interval=200ms

echo
echo "✅ Bulkhead demo completed!"
echo "🛑 Stopping server..."
kill $SERVER_PID 2>/dev/null
wait $SERVER_PID 2>/dev/null