package httpclient

import (
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
)

// RegisterPrometheus registers the client's counters with the provided
// Prometheus registerer. If reg is nil, the default registerer is used.
func (c *Client) RegisterPrometheus(reg prometheus.Registerer) {
	if reg == nil {
		reg = prometheus.DefaultRegisterer
	}
	reg.MustRegister(
		prometheus.NewCounterFunc(prometheus.CounterOpts{
			Name: "goresily_httpclient_requests_total",
			Help: "Total number of HTTP requests sent by the client",
		}, func() float64 {
			return float64(atomic.LoadUint64(&c.reqCount))
		}),
		prometheus.NewCounterFunc(prometheus.CounterOpts{
			Name: "goresily_httpclient_successes_total",
			Help: "Total number of successful client calls",
		}, func() float64 {
			return float64(atomic.LoadUint64(&c.successCount))
		}),
		prometheus.NewCounterFunc(prometheus.CounterOpts{
			Name: "goresily_httpclient_errors_total",
			Help: "Total number of errored client calls",
		}, func() float64 {
			return float64(atomic.LoadUint64(&c.errorCount))
		}),
		prometheus.NewCounterFunc(prometheus.CounterOpts{
			Name:        "goresily_httpclient_responses_total",
			Help:        "HTTP responses by status class",
			ConstLabels: prometheus.Labels{"class": "2xx"},
		}, func() float64 {
			return float64(atomic.LoadUint64(&c.resp2xxCount))
		}),
		prometheus.NewCounterFunc(prometheus.CounterOpts{
			Name:        "goresily_httpclient_responses_total",
			Help:        "HTTP responses by status class",
			ConstLabels: prometheus.Labels{"class": "3xx"},
		}, func() float64 {
			return float64(atomic.LoadUint64(&c.resp3xxCount))
		}),
		prometheus.NewCounterFunc(prometheus.CounterOpts{
			Name:        "goresily_httpclient_responses_total",
			Help:        "HTTP responses by status class",
			ConstLabels: prometheus.Labels{"class": "4xx"},
		}, func() float64 {
			return float64(atomic.LoadUint64(&c.resp4xxCount))
		}),
		prometheus.NewCounterFunc(prometheus.CounterOpts{
			Name:        "goresily_httpclient_responses_total",
			Help:        "HTTP responses by status class",
			ConstLabels: prometheus.Labels{"class": "5xx"},
		}, func() float64 {
			return float64(atomic.LoadUint64(&c.resp5xxCount))
		}),
	)

	if c.CB != nil {
		reg.MustRegister(
			prometheus.NewCounterFunc(prometheus.CounterOpts{
				Name:        "goresily_httpclient_breaker_transitions_total",
				Help:        "Circuit breaker state transitions",
				ConstLabels: prometheus.Labels{"state": "open"},
			}, func() float64 {
				return float64(c.CB.Metrics().Opened)
			}),
			prometheus.NewCounterFunc(prometheus.CounterOpts{
				Name:        "goresily_httpclient_breaker_transitions_total",
				Help:        "Circuit breaker state transitions",
				ConstLabels: prometheus.Labels{"state": "half_open"},
			}, func() float64 {
				return float64(c.CB.Metrics().HalfOpened)
			}),
			prometheus.NewCounterFunc(prometheus.CounterOpts{
				Name:        "goresily_httpclient_breaker_transitions_total",
				Help:        "Circuit breaker state transitions",
				ConstLabels: prometheus.Labels{"state": "closed"},
			}, func() float64 {
				return float64(c.CB.Metrics().Closed)
			}),
		)
	}

	if c.BH != nil {
		reg.MustRegister(
			prometheus.NewCounterFunc(prometheus.CounterOpts{
				Name: "goresily_httpclient_bulkhead_full_total",
				Help: "Times the bulkhead rejected executions",
			}, func() float64 {
				return float64(c.BH.Metrics().Rejected)
			}),
		)
	}
}
