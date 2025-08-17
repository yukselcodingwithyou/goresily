package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"goresily/circuitbreaker"
	"goresily/httpclient"
)

// ClientConfig holds configuration for the client
type ClientConfig struct {
	ServerURL              string
	RequestInterval        time.Duration
	MaxRequests           int
	ConcurrentWorkers     int
	CircuitBreakerEnabled bool
	BulkheadEnabled       bool
}

// Client represents our microservice client
type Client struct {
	config     *ClientConfig
	httpClient *httpclient.Client
}

// ServerResponse represents the response from the server
type ServerResponse struct {
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Endpoint  string    `json:"endpoint"`
}

func NewClient(config *ClientConfig) *Client {
	// Configure HTTP client
	httpCfg := &httpclient.HTTPClientConfig{
		Timeout: 3 * time.Second,
	}

	// Configure Circuit Breaker
	var breakerCfg *httpclient.BreakerConfig
	if config.CircuitBreakerEnabled {
		breakerCfg = &httpclient.BreakerConfig{
			MaxFailures:   3,                // Open after 3 failures
			Window:        5 * time.Second,  // Count failures in 5 second window
			Timeout:       10 * time.Second, // Stay open for 10 seconds
			TrialRequests: 2,                // Allow 2 requests in half-open state
			TrialDuration: 5 * time.Second,  // Trial period duration
			OnStateChange: func(s circuitbreaker.State) {
				switch s {
				case circuitbreaker.Closed:
					log.Printf("[CLIENT] 🔄 ✅ Circuit Breaker: CLOSED → Service is healthy, allowing all requests")
				case circuitbreaker.Open:
					log.Printf("[CLIENT] 🔄 🚨 Circuit Breaker: OPEN → Service appears down, blocking requests for 10s")
				case circuitbreaker.HalfOpen:
					log.Printf("[CLIENT] 🔄 🔍 Circuit Breaker: HALF-OPEN → Testing service with limited requests (max 2)")
				}
			},
		}
	}

	// Configure Bulkhead
	var bulkheadCfg *httpclient.BulkheadConfig
	if config.BulkheadEnabled {
		bulkheadCfg = &httpclient.BulkheadConfig{
			Limit: 2, // Allow only 2 concurrent requests
		}
	}

	// Create HTTP client based on configuration
	var client *httpclient.Client
	if config.CircuitBreakerEnabled && config.BulkheadEnabled {
		client = httpclient.NewWithBreakerAndBulkhead(httpCfg, breakerCfg, bulkheadCfg)
		log.Printf("[CLIENT] ⚙️  Initialized with Circuit Breaker + Bulkhead")
	} else if config.CircuitBreakerEnabled {
		client = httpclient.NewWithBreaker(httpCfg, breakerCfg)
		log.Printf("[CLIENT] ⚙️  Initialized with Circuit Breaker only")
	} else if config.BulkheadEnabled {
		client = httpclient.NewWithBulkhead(httpCfg, bulkheadCfg)
		log.Printf("[CLIENT] ⚙️  Initialized with Bulkhead only")
	} else {
		client = httpclient.NewPlain(httpCfg)
		log.Printf("[CLIENT] ⚙️  Initialized with no resilience patterns")
	}

	return &Client{
		config:     config,
		httpClient: client,
	}
}

func formatState(state circuitbreaker.State) string {
	switch state {
	case circuitbreaker.Closed:
		return "🟢 CLOSED (allowing requests)"
	case circuitbreaker.Open:
		return "🔴 OPEN (blocking requests)"
	case circuitbreaker.HalfOpen:
		return "🟡 HALF-OPEN (testing service)"
	default:
		return "❓ UNKNOWN"
	}
}

func (c *Client) makeRequest(ctx context.Context, endpoint string, requestID int) (*ServerResponse, error) {
	url := c.config.ServerURL + endpoint
	
	req := httpclient.NewBasicRequestBuilder().
		Method(http.MethodGet).
		URL(url).
		Build()

	log.Printf("[CLIENT] 📤 Request #%d: %s", requestID, endpoint)
	
	// Show bulkhead status before making request
	if c.config.BulkheadEnabled && c.httpClient.BH != nil {
		current := c.httpClient.BH.CurrentUsage()
		limit := c.httpClient.BH.Limit()
		log.Printf("[CLIENT] 🚧 Bulkhead status: %d/%d slots used", current, limit)
	}
	
	start := time.Now()
	resp, err := c.httpClient.Call(ctx, req)
	duration := time.Since(start)

	if err != nil {
		// Provide more detailed error information
		if err.Error() == "circuit breaker is open" {
			log.Printf("[CLIENT] ❌ Request #%d BLOCKED by Circuit Breaker (duration: %v)", requestID, duration)
		} else if err.Error() == "bulkhead full" {
			current := c.httpClient.BH.CurrentUsage()
			limit := c.httpClient.BH.Limit()
			log.Printf("[CLIENT] ❌ Request #%d REJECTED by Bulkhead - all %d/%d slots occupied (duration: %v)", 
				requestID, current, limit, duration)
		} else {
			log.Printf("[CLIENT] ❌ Request #%d failed after %v: %v", requestID, duration, err)
		}
		return nil, err
	}

	log.Printf("[CLIENT] ✅ Request #%d succeeded after %v (status: %d)", requestID, duration, resp.StatusCode())

	var serverResp ServerResponse
	if err := json.Unmarshal(resp.Body(), &serverResp); err != nil {
		log.Printf("[CLIENT] ⚠️  Request #%d: Failed to parse response: %v", requestID, err)
		return nil, err
	}

	return &serverResp, nil
}

func (c *Client) runSequentialRequests(ctx context.Context) {
	log.Printf("[CLIENT] 🚀 Starting sequential requests (max: %d, interval: %v)", 
		c.config.MaxRequests, c.config.RequestInterval)

	for i := 1; i <= c.config.MaxRequests; i++ {
		if ctx.Err() != nil {
			log.Printf("[CLIENT] 🛑 Context cancelled, stopping requests")
			break
		}

		_, err := c.makeRequest(ctx, "/data", i)
		if err != nil {
			// Log the error (already logged in makeRequest)
		}

		if i < c.config.MaxRequests {
			log.Printf("[CLIENT] ⏳ Waiting %v before next request...", c.config.RequestInterval)
			time.Sleep(c.config.RequestInterval)
		}
	}

	log.Printf("[CLIENT] 🏁 Sequential requests completed")
}

func (c *Client) runConcurrentRequests(ctx context.Context) {
	log.Printf("[CLIENT] 🚀 Starting concurrent requests (%d workers, %d requests each)", 
		c.config.ConcurrentWorkers, c.config.MaxRequests)

	var wg sync.WaitGroup
	
	for workerID := 1; workerID <= c.config.ConcurrentWorkers; workerID++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			log.Printf("[CLIENT] 👷 Worker #%d started", id)
			
			for reqNum := 1; reqNum <= c.config.MaxRequests; reqNum++ {
				if ctx.Err() != nil {
					log.Printf("[CLIENT] 👷 Worker #%d: Context cancelled", id)
					return
				}

				requestID := (id-1)*c.config.MaxRequests + reqNum
				_, err := c.makeRequest(ctx, "/data", requestID)
				if err != nil {
					// Error already logged in makeRequest
				}

				if reqNum < c.config.MaxRequests {
					time.Sleep(c.config.RequestInterval)
				}
			}
			
			log.Printf("[CLIENT] 👷 Worker #%d completed", id)
		}(workerID)
	}

	wg.Wait()
	log.Printf("[CLIENT] 🏁 All concurrent workers completed")
}

func (c *Client) demonstrateCircuitBreaker(ctx context.Context) {
	log.Printf("\n" + strings.Repeat("=", 70))
	log.Printf("[CLIENT] 🔄 CIRCUIT BREAKER DEMONSTRATION")
	log.Printf(strings.Repeat("=", 70))
	
	log.Printf("[CLIENT] 📋 This demo will show circuit breaker states:")
	log.Printf("[CLIENT] 📋 1. Closed → Open (after failures)")
	log.Printf("[CLIENT] 📋 2. Open → Half-Open (after timeout)")
	log.Printf("[CLIENT] 📋 3. Half-Open → Closed (after successful trials)")
	log.Printf("")

	// Phase 1: Trigger failures to open circuit breaker
	log.Printf("[CLIENT] 🔥 Phase 1: Triggering failures to open circuit breaker...")
	c.runSequentialRequests(ctx)
	
	// Phase 2: Wait for circuit breaker to enter half-open state
	log.Printf("\n[CLIENT] ⏰ Phase 2: Waiting for circuit breaker timeout (10s) to enter HALF-OPEN state...")
	log.Printf("[CLIENT] 💤 Sleeping for 11 seconds to allow state transition...")
	time.Sleep(11 * time.Second)
	
	// Phase 3: Make requests in half-open state to show recovery
	log.Printf("\n[CLIENT] 🔄 Phase 3: Making requests in HALF-OPEN state to demonstrate recovery...")
	
	// Reduce the number of requests for half-open demonstration
	originalRequests := c.config.MaxRequests
	c.config.MaxRequests = 5
	defer func() { c.config.MaxRequests = originalRequests }()
	
	c.runSequentialRequests(ctx)
}

func (c *Client) demonstrateBulkhead(ctx context.Context) {
	log.Printf("\n" + strings.Repeat("=", 70))
	log.Printf("[CLIENT] 🚧 BULKHEAD DEMONSTRATION")
	log.Printf(strings.Repeat("=", 70))
	
	log.Printf("[CLIENT] 📋 This demo will show bulkhead limiting:")
	log.Printf("[CLIENT] 📋 - Only %d concurrent requests allowed", 2)
	log.Printf("[CLIENT] 📋 - Additional requests will be rejected")
	log.Printf("[CLIENT] 📋 - Watch for bulkhead capacity usage logs")
	log.Printf("")

	// Make concurrent requests to trigger bulkhead
	c.runConcurrentRequests(ctx)
}

func (c *Client) demonstrateBoth(ctx context.Context) {
	log.Printf("\n" + strings.Repeat("=", 70))
	log.Printf("[CLIENT] 🎭 COMBINED DEMONSTRATION (Circuit Breaker + Bulkhead)")
	log.Printf(strings.Repeat("=", 70))
	
	log.Printf("[CLIENT] 📋 This demo shows interaction between both patterns:")
	log.Printf("[CLIENT] 📋 1. Bulkhead limits concurrent requests (max 2)")
	log.Printf("[CLIENT] 📋 2. Circuit breaker monitors failure rate")
	log.Printf("[CLIENT] 📋 3. Both patterns work together for resilience")
	log.Printf("")
	
	log.Printf("[CLIENT] 🔥 Phase 1: High concurrency to demonstrate bulkhead...")
	c.runConcurrentRequests(ctx)
	
	log.Printf("\n[CLIENT] ⏰ Phase 2: Sequential requests to trigger circuit breaker...")
	// Temporarily reduce concurrent workers for this phase
	originalWorkers := c.config.ConcurrentWorkers
	c.config.ConcurrentWorkers = 1
	defer func() { c.config.ConcurrentWorkers = originalWorkers }()
	
	c.runSequentialRequests(ctx)
	
	log.Printf("\n[CLIENT] 💤 Phase 3: Waiting for circuit breaker recovery (11s)...")
	time.Sleep(11 * time.Second)
	
	log.Printf("\n[CLIENT] 🔄 Phase 4: Testing recovery with both patterns active...")
	originalRequests := c.config.MaxRequests
	c.config.MaxRequests = 4
	defer func() { c.config.MaxRequests = originalRequests }()
	
	c.runSequentialRequests(ctx)
}

func main() {
	var (
		serverURL         = flag.String("server", "http://localhost:8080", "Server URL")
		mode              = flag.String("mode", "cb", "Mode: 'cb' (circuit breaker), 'bh' (bulkhead), 'both' (both patterns)")
		maxRequests       = flag.Int("requests", 10, "Maximum number of requests per worker")
		interval          = flag.Duration("interval", 1*time.Second, "Interval between requests")
		workers           = flag.Int("workers", 3, "Number of concurrent workers (for bulkhead demo)")
		sequential        = flag.Bool("sequential", false, "Run requests sequentially instead of concurrently")
	)
	flag.Parse()

	config := &ClientConfig{
		ServerURL:             *serverURL,
		RequestInterval:       *interval,
		MaxRequests:          *maxRequests,
		ConcurrentWorkers:    *workers,
		CircuitBreakerEnabled: *mode == "cb" || *mode == "both",
		BulkheadEnabled:      *mode == "bh" || *mode == "both",
	}

	log.Printf("[CLIENT] 🎯 Starting Client Microservice")
	log.Printf("[CLIENT] 🎯 Mode: %s", *mode)
	log.Printf("[CLIENT] 🎯 Server: %s", config.ServerURL)
	log.Printf("[CLIENT] 🎯 Max Requests: %d", config.MaxRequests)
	log.Printf("[CLIENT] 🎯 Request Interval: %v", config.RequestInterval)
	if !*sequential {
		log.Printf("[CLIENT] 🎯 Concurrent Workers: %d", config.ConcurrentWorkers)
	}

	client := NewClient(config)
	ctx := context.Background()

	// Test server connectivity first
	log.Printf("\n[CLIENT] 🔍 Testing server connectivity...")
	if _, err := client.makeRequest(ctx, "/health", 0); err != nil {
		log.Printf("[CLIENT] ❌ Server connectivity test failed: %v", err)
		log.Printf("[CLIENT] 💡 Make sure the server is running: go run examples/microservices/server/main.go")
		return
	}
	log.Printf("[CLIENT] ✅ Server connectivity confirmed")

	switch *mode {
	case "cb":
		client.demonstrateCircuitBreaker(ctx)
	case "bh":
		client.demonstrateBulkhead(ctx)
	case "both":
		client.demonstrateBoth(ctx)
	default:
		log.Printf("[CLIENT] ❌ Unknown mode: %s", *mode)
		log.Printf("[CLIENT] 💡 Use -mode=cb, -mode=bh, or -mode=both")
	}

	log.Printf("\n[CLIENT] 🎉 Client demonstration completed!")
}