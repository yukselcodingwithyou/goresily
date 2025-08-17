# Quick Start Guide

This directory contains two microservices that demonstrate the **goresily** library patterns in real-world scenarios.

## 🚀 Quick Demo

1. **Build the services:**
   ```bash
   # In examples/microservices/server/
   go build -o server main.go
   
   # In examples/microservices/client/
   go build -o client main.go
   ```

2. **Run Circuit Breaker Demo:**
   ```bash
   # Terminal 1: Start server with high failure rate
   cd examples/microservices/server
   ./server -failure-rate=0.7
   
   # Terminal 2: Run circuit breaker test
   cd examples/microservices/client
   ./client -mode=cb -requests=10 -interval=600ms
   ```

3. **Run Bulkhead Demo:**
   ```bash
   # Terminal 1: Start server with slow responses
   cd examples/microservices/server
   ./server -slow-rate=0.9 -slow-delay=2000
   
   # Terminal 2: Run bulkhead test
   cd examples/microservices/client  
   ./client -mode=bh -requests=4 -workers=6 -interval=200ms
   ```

## 📋 What You'll See

### Circuit Breaker
- 🟢 **CLOSED**: Normal operation, requests flowing
- 🔴 **OPEN**: Service failing, requests blocked
- 🟡 **HALF-OPEN**: Testing if service recovered

### Bulkhead
- ✅ Successful requests (within limit)
- ❌ "bulkhead full" errors (exceeding limit)

### Combined Patterns
- Both circuit breaker and bulkhead working together
- Comprehensive resilience protection

## 🎭 Demo Scripts

For automated demonstrations, use the provided scripts:
- `./demo-circuit-breaker.sh` - Shows circuit breaker behavior
- `./demo-bulkhead.sh` - Shows request limiting  
- `./demo-combined.sh` - Shows both patterns together

## 📖 Full Tutorial

See [README.md](README.md) for complete documentation, configuration options, and advanced scenarios.