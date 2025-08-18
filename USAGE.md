# Usage Guide

This guide shows how to use the individual components provided by `goresily`.

## Circuit Breaker

```go
cb := circuitbreaker.NewBuilder().
    MaxFailures(5).
    Timeout(30 * time.Second).
    Window(1 * time.Minute).
    Build()

err := cb.Execute(func() error {
    // call protected resource
    return nil
})
if err == circuitbreaker.ErrOpen {
    // breaker is open
}
```

## Bulkhead

```go
bh := bulkhead.NewBuilder().Limit(10).Build()

err := bh.Execute(func() error {
    // work to be limited
    return nil
})
if err == bulkhead.ErrFull {
    // rejected because limit reached
}
```

## HTTP Client

```go
client := httpclient.NewWithBreakerAndBulkhead(
    &httpclient.HTTPClientConfig{Timeout: 2 * time.Second},
    &httpclient.BreakerConfig{MaxFailures: 3, Timeout: 5 * time.Second},
    &httpclient.BulkheadConfig{Limit: 2},
)

req := httpclient.NewBasicRequestBuilder().
    Method(http.MethodGet).
    URL("https://example.com").
    Build()

resp, err := client.Call(context.Background(), req)
if err != nil {
    // handle error
}
_ = resp
```

## Running the examples

Try the demo programs to see the components in action:

```bash
go run ./examples                                   # simple demonstration
go run ./examples/microservices/server &           # start test server
go run ./examples/microservices/client             # run client against server
```
