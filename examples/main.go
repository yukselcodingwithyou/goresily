package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"

	"goresily/circuitbreaker"
	"goresily/httpclient"
)

func main() {
	// test server with success and failure endpoints
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/fail":
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("fail"))
		default:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		}
	}))
	defer srv.Close()

	cfg := &httpclient.Config{
		HTTP: &httpclient.HTTPClientConfig{Timeout: 500 * time.Millisecond},
		Breaker: &httpclient.BreakerConfig{
			MaxFailures:   2,
			Window:        500 * time.Millisecond,
			Timeout:       2 * time.Second,
			TrialRequests: 2,
			TrialDuration: 2 * time.Second,
			OnStateChange: func(s circuitbreaker.State) {
				fmt.Println("circuit breaker state:", s)
			},
		},
		Bulkhead: &httpclient.BulkheadConfig{Limit: 1},
	}

	client := httpclient.New(cfg)

	// failing calls to open the circuit breaker
	for i := 0; i < 3; i++ {
		req := httpclient.NewBasicRequestBuilder().
			Method(http.MethodGet).
			URL(srv.URL + "/fail").
			Build()
		_, err := client.Call(context.Background(), req)
		if err != nil {
			fmt.Println("call error:", err)
		}
	}

	// wait for breaker to close again
	time.Sleep(3 * time.Second)

	// demonstrate bulkhead with concurrent calls
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			req := httpclient.NewBasicRequestBuilder().
				Method(http.MethodGet).
				URL(srv.URL + "/success").
				Build()
			resp, err := client.Call(context.Background(), req)
			if err != nil {
				fmt.Printf("worker %d error: %v\n", id, err)
				return
			}
			fmt.Printf("worker %d status: %d\n", id, resp.StatusCode())
		}(i)
	}
	wg.Wait()
}
