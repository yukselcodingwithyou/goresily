# Microservices Resilience Tutorial

This tutorial demonstrates real-world usage of the **goresily** library with two microservices that showcase circuit breaker and bulkhead patterns in action.

## Overview

We have created two microservices:

1. **Server Microservice** (`server/main.go`) - A resilience testing server that can simulate failures and slow responses
2. **Client Microservice** (`client/main.go`) - A client that uses the goresily library to make resilient requests to the server

## Architecture

```
┌─────────────────┐    HTTP Requests    ┌─────────────────┐
│                 │ ───────────────────▶ │                 │
│ Client Service  │                     │ Server Service  │
│  (goresily)     │ ◀─────────────────── │ (test endpoints)│
│                 │   Responses/Errors   │                 │
└─────────────────┘                     └─────────────────┘
       │
       ├─ Circuit Breaker (prevents cascade failures)
       ├─ Bulkhead (limits concurrent requests)
       └─ HTTP Client (with timeouts)
```

## Features Demonstrated

### Circuit Breaker
- **Closed State**: Requests flow normally
- **Open State**: Requests are blocked after failures exceed threshold
- **Half-Open State**: Limited requests allowed to test if service recovered

### Bulkhead Pattern
- Limits concurrent requests to prevent resource exhaustion
- Rejects excess requests when limit is reached
- Protects downstream services from overload

## Getting Started

### 1. Start the Server Microservice

```bash
# In terminal 1 - Start server with 30% failure rate and 20% slow responses
cd examples/microservices/server
go run main.go -failure-rate=0.3 -slow-rate=0.2 -slow-delay=2000

# Server will start on http://localhost:8080
```

Server configuration options:
- `-port=8080`: Server port
- `-failure-rate=0.3`: Probability of returning 500 errors (0.0 to 1.0)
- `-slow-rate=0.2`: Probability of slow responses (0.0 to 1.0) 
- `-slow-delay=2000`: Delay in milliseconds for slow responses

### 2. Test Circuit Breaker Pattern

```bash
# In terminal 2 - Run circuit breaker demo
cd examples/microservices/client
go run main.go -mode=cb -requests=15 -interval=1s

# Watch the logs to see:
# 🟢 CLOSED → 🔴 OPEN → 🟡 HALF-OPEN → 🟢 CLOSED state transitions
```

### 3. Test Bulkhead Pattern

```bash
# In terminal 2 - Run bulkhead demo with concurrent workers
cd examples/microservices/client
go run main.go -mode=bh -requests=5 -workers=4 -interval=500ms

# Watch the logs to see:
# - Only 2 concurrent requests allowed through
# - Additional requests rejected with "bulkhead full" error
```

### 4. Test Combined Patterns

```bash
# In terminal 2 - Run both circuit breaker and bulkhead
cd examples/microservices/client
go run main.go -mode=both -requests=10 -workers=3 -interval=1s

# Watch the logs to see both patterns working together
```

## Demo Scenarios

### Scenario 1: Circuit Breaker Demo

**Goal**: Show circuit breaker opening, half-opening, and closing

**Setup**:
```bash
# Terminal 1: Start server with high failure rate
go run server/main.go -failure-rate=0.6

# Terminal 2: Run circuit breaker client
go run client/main.go -mode=cb -requests=15 -interval=800ms
```

**Expected Behavior**:
1. First few requests succeed (Closed state)
2. After 3 failures, circuit breaker opens (Open state)
3. Requests blocked for 10 seconds
4. Circuit breaker enters Half-Open state
5. If trial requests succeed, circuit closes again

### Scenario 2: Bulkhead Demo

**Goal**: Show request limiting in action

**Setup**:
```bash
# Terminal 1: Start server with slow responses
go run server/main.go -failure-rate=0.1 -slow-rate=0.8 -slow-delay=3000

# Terminal 2: Run bulkhead client with many concurrent workers
go run client/main.go -mode=bh -requests=3 -workers=5 -interval=100ms
```

**Expected Behavior**:
1. Only 2 requests processed concurrently
2. Additional requests rejected immediately
3. "bulkhead full" errors shown in logs

### Scenario 3: Recovery Simulation

**Goal**: Show service recovery handling

**Setup**:
```bash
# Terminal 1: Start server with high failure rate
go run server/main.go -failure-rate=0.8

# Terminal 2: Start client
go run client/main.go -mode=cb -requests=20 -interval=1s

# Terminal 3: Reduce failure rate while client is running
curl -X POST http://localhost:8080/config \
  -H "Content-Type: application/json" \
  -d '{"failure_rate":0.1,"slow_response_rate":0.1,"slow_delay":500}'
```

**Expected Behavior**:
1. Circuit breaker opens due to high failure rate
2. When server configuration improves, circuit breaker detects recovery
3. Circuit breaker transitions back to closed state

## Understanding the Logs

### Circuit Breaker Log Messages

```
🟢 CLOSED (allowing requests)     - Normal operation
🔴 OPEN (blocking requests)       - Service appears down, blocking requests  
🟡 HALF-OPEN (testing service)    - Testing if service recovered
```

### Request Log Messages

```
📤 Request #1: /data              - Outgoing request
✅ Request #1 succeeded after 245ms (status: 200) - Successful response
❌ Request #2 failed after 2.1s: server error: 500 - Failed response
❌ Request #3 failed after 0ms: circuit breaker is open - Blocked by circuit breaker
❌ Request #4 failed after 0ms: bulkhead full - Blocked by bulkhead
```

## Server Endpoints

The server provides several endpoints for testing:

- `GET /health` - Always returns 200 (health check)
- `GET /data` - May fail or be slow based on configuration
- `GET /config` - Get current server configuration
- `POST /config` - Update server configuration
- `GET /load?duration=10` - Simulate CPU load for testing

### Example: Change Server Behavior

```bash
# Make server very unreliable
curl -X POST http://localhost:8080/config \
  -H "Content-Type: application/json" \
  -d '{"failure_rate":0.9,"slow_response_rate":0.5,"slow_delay":5000}'

# Make server reliable again  
curl -X POST http://localhost:8080/config \
  -H "Content-Type: application/json" \
  -d '{"failure_rate":0.1,"slow_response_rate":0.1,"slow_delay":500}'
```

## Configuration Options

### Client Configuration

```bash
-server="http://localhost:8080"  # Server URL
-mode="cb"                       # Mode: cb, bh, or both
-requests=10                     # Requests per worker
-interval=1s                     # Time between requests
-workers=3                       # Concurrent workers (for bulkhead)
-sequential=false                # Run requests sequentially
```

### Server Configuration

```bash
-port=8080                       # Server port
-failure-rate=0.3                # Failure probability (0.0-1.0)
-slow-rate=0.2                   # Slow response probability (0.0-1.0) 
-slow-delay=2000                 # Slow response delay (ms)
```

## Real-World Applications

This tutorial simulates real-world scenarios where:

1. **Circuit Breaker** prevents:
   - Cascade failures when downstream services fail
   - Resource exhaustion from retrying failed services
   - Long wait times when services are unresponsive

2. **Bulkhead** prevents:
   - One slow service from consuming all resources
   - Thread pool exhaustion
   - Complete service unavailability

## Next Steps

After running these examples, you can:

1. Modify the configuration values to see different behaviors
2. Add more complex failure scenarios to the server
3. Integrate these patterns into your own microservices
4. Monitor the patterns in production with metrics and alerting

## Tips for Production

1. **Circuit Breaker**: 
   - Set appropriate failure thresholds based on your SLA
   - Monitor state changes and alert on frequent opens
   - Consider different timeouts for different operations

2. **Bulkhead**:
   - Size bulkheads based on expected load and resource capacity
   - Monitor bulkhead rejections to size appropriately
   - Consider separate bulkheads for different types of operations

3. **Combined Usage**:
   - Circuit breakers protect against downstream failures
   - Bulkheads protect against resource exhaustion
   - Use both for comprehensive resilience