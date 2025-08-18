module goresily

go 1.20

require (
	github.com/prometheus/client_golang v0.0.0
	github.com/valyala/fasthttp v0.0.0
)

replace github.com/valyala/fasthttp => ./internal/fasthttp

replace github.com/prometheus/client_golang => ./internal/prometheus
