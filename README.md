# goresily

Circuit breaker and bulkhead implementations in Go.

A small example using them with an HTTP client is available under `examples/` and can be run with:

```
go run ./examples
```

The HTTP client in the example uses `fasthttp` and can be configured with a circuit breaker and/or a bulkhead using simple configuration structs. The client itself is created from a `Config` that builds the breaker and bulkhead for you. The circuit breaker supports a half-open state with configurable trial requests and duration. You can also tune the underlying client via `HTTPClientConfig`.
