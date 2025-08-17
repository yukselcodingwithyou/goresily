package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

// Server configuration
type Config struct {
	Port             int     `json:"port"`
	FailureRate      float64 `json:"failure_rate"`       // 0.0 to 1.0
	SlowResponseRate float64 `json:"slow_response_rate"` // 0.0 to 1.0
	SlowDelay        int     `json:"slow_delay_ms"`      // milliseconds
}

type Server struct {
	config *Config
}

type Response struct {
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Endpoint  string    `json:"endpoint"`
}

func NewServer(config *Config) *Server {
	return &Server{config: config}
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[SERVER] Health check requested from %s", r.RemoteAddr)
	
	response := Response{
		Message:   "Server is healthy",
		Timestamp: time.Now(),
		Endpoint:  "/health",
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) dataHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[SERVER] Data request from %s", r.RemoteAddr)
	
	// Simulate slow response
	if rand.Float64() < s.config.SlowResponseRate {
		delay := time.Duration(s.config.SlowDelay) * time.Millisecond
		log.Printf("[SERVER] Simulating slow response: %v delay", delay)
		time.Sleep(delay)
	}
	
	// Simulate failure
	if rand.Float64() < s.config.FailureRate {
		log.Printf("[SERVER] Simulating failure response")
		http.Error(w, "Internal Server Error - Simulated Failure", http.StatusInternalServerError)
		return
	}
	
	response := Response{
		Message:   fmt.Sprintf("Data retrieved successfully at %s", time.Now().Format("15:04:05")),
		Timestamp: time.Now(),
		Endpoint:  "/data",
	}
	
	log.Printf("[SERVER] Returning successful response")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) configHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		log.Printf("[SERVER] Config requested from %s", r.RemoteAddr)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(s.config)
	case http.MethodPost:
		log.Printf("[SERVER] Config update requested from %s", r.RemoteAddr)
		var newConfig Config
		if err := json.NewDecoder(r.Body).Decode(&newConfig); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		
		s.config.FailureRate = newConfig.FailureRate
		s.config.SlowResponseRate = newConfig.SlowResponseRate
		s.config.SlowDelay = newConfig.SlowDelay
		
		log.Printf("[SERVER] Config updated: failure_rate=%.2f, slow_response_rate=%.2f, slow_delay=%dms", 
			s.config.FailureRate, s.config.SlowResponseRate, s.config.SlowDelay)
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(s.config)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) simulateLoad(w http.ResponseWriter, r *http.Request) {
	// Get duration from query parameter (default 10 seconds)
	durationStr := r.URL.Query().Get("duration")
	duration := 10 * time.Second
	if durationStr != "" {
		if d, err := time.ParseDuration(durationStr + "s"); err == nil {
			duration = d
		}
	}
	
	log.Printf("[SERVER] Load simulation requested for %v", duration)
	
	// Simulate CPU-intensive work
	end := time.Now().Add(duration)
	count := 0
	for time.Now().Before(end) {
		count++
		// Simulate some work
		for i := 0; i < 1000; i++ {
			_ = i * i
		}
		if count%100000 == 0 {
			time.Sleep(1 * time.Millisecond) // Give other goroutines a chance
		}
	}
	
	response := Response{
		Message:   fmt.Sprintf("Load simulation completed after %v (%d iterations)", duration, count),
		Timestamp: time.Now(),
		Endpoint:  "/load",
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	var (
		port             = flag.Int("port", 8080, "Server port")
		failureRate      = flag.Float64("failure-rate", 0.3, "Failure rate (0.0 to 1.0)")
		slowResponseRate = flag.Float64("slow-rate", 0.2, "Slow response rate (0.0 to 1.0)")
		slowDelay        = flag.Int("slow-delay", 2000, "Slow response delay in milliseconds")
	)
	flag.Parse()
	
	config := &Config{
		Port:             *port,
		FailureRate:      *failureRate,
		SlowResponseRate: *slowResponseRate,
		SlowDelay:        *slowDelay,
	}
	
	server := NewServer(config)
	
	// Seed random number generator
	rand.Seed(time.Now().UnixNano())
	
	// Setup routes
	http.HandleFunc("/health", server.healthHandler)
	http.HandleFunc("/data", server.dataHandler)
	http.HandleFunc("/config", server.configHandler)
	http.HandleFunc("/load", server.simulateLoad)
	
	// Root handler
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		
		fmt.Fprintf(w, `
Server Microservice - Resilience Testing

Available endpoints:
- GET  /health        - Health check
- GET  /data          - Get data (may fail or be slow based on config)
- GET  /config        - Get current configuration
- POST /config        - Update configuration
- GET  /load?duration=10 - Simulate CPU load for specified seconds

Current Configuration:
- Port: %d
- Failure Rate: %.1f%% 
- Slow Response Rate: %.1f%%
- Slow Delay: %dms

Examples:
curl http://localhost:%d/data
curl http://localhost:%d/config
curl -X POST http://localhost:%d/config -H "Content-Type: application/json" -d '{"failure_rate":0.5,"slow_response_rate":0.3,"slow_delay":3000}'
`, 
			config.Port, 
			config.FailureRate*100, 
			config.SlowResponseRate*100, 
			config.SlowDelay,
			config.Port,
			config.Port,
			config.Port)
	})
	
	addr := ":" + strconv.Itoa(config.Port)
	log.Printf("[SERVER] Starting server on http://localhost%s", addr)
	log.Printf("[SERVER] Configuration: failure_rate=%.1f%%, slow_response_rate=%.1f%%, slow_delay=%dms", 
		config.FailureRate*100, config.SlowResponseRate*100, config.SlowDelay)
	
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal("[SERVER] Server failed to start:", err)
	}
}