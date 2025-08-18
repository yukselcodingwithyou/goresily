# goresily

`goresily` provides lightweight implementations of the **circuit breaker** and **bulkhead** patterns and a convenient HTTP client that composes them. It is designed to make building resilient Go services straightforward.

## Features
- Circuit breaker with configurable failure thresholds, time windows, open timeout and half‑open trial requests.
- Bulkhead concurrency limiter to cap the number of simultaneous executions.
- HTTP client powered by `fasthttp` with optional breaker and bulkhead, configuration helpers and Prometheus metrics.
- Simple metric structs for introspection or exporting to monitoring systems.
- Example programs including a microservices tutorial.

## Installation

```bash
go get goresily
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "net/http"
    "time"

    "goresily/httpclient"
)

func main() {
    client := httpclient.New(&httpclient.Config{
        HTTP: &httpclient.HTTPClientConfig{
            Timeout: 5 * time.Second,
        },
        Breaker: &httpclient.BreakerConfig{
            MaxFailures:   3,
            Window:        30 * time.Second,
            Timeout:       10 * time.Second,
            TrialRequests: 2,
        },
        Bulkhead: &httpclient.BulkheadConfig{
            Limit: 5,
        },
    })

    req := httpclient.NewBasicRequestBuilder().
        Method(http.MethodGet).
        URL("https://example.com/data").
        Build()

    resp, err := client.Call(context.Background(), req)
    if err != nil {
        fmt.Println("call failed:", err)
        return
    }
    fmt.Println("status:", resp.StatusCode())
}
```

## Metrics

Each component exposes counters for basic insight into behaviour:

```go
m := client.Metrics()
fmt.Printf("requests: %d errors: %d\n", m.Requests, m.Errors)
```

Use `client.RegisterPrometheus(nil)` to publish these counters to the default Prometheus registerer.

## Examples

- `examples/` contains a small standalone demo that can be run with:
  ```bash
  go run ./examples
  ```
- `examples/microservices` provides a full tutorial showcasing the patterns between two services. See its README for step‑by‑step instructions.
