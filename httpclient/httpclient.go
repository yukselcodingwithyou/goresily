package httpclient

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/valyala/fasthttp"

	"goresily/bulkhead"
	"goresily/circuitbreaker"
)

// Request represents an HTTP request abstraction.
type Request interface {
	Method() string
	URL() string
	Query() url.Values
	Body() io.Reader
	Headers() http.Header
}

// Response represents an HTTP response abstraction.
type Response interface {
	StatusCode() int
	Body() []byte
	Headers() http.Header
}

// BasicRequest is a simple Request implementation.
type BasicRequest struct {
	MethodVal  string
	URLVal     string
	QueryVal   url.Values
	BodyBytes  []byte
	HeaderVals http.Header
}

func (r *BasicRequest) Method() string { return r.MethodVal }
func (r *BasicRequest) URL() string    { return r.URLVal }
func (r *BasicRequest) Query() url.Values {
	if r.QueryVal == nil {
		return url.Values{}
	}
	return r.QueryVal
}
func (r *BasicRequest) Body() io.Reader { return bytes.NewReader(r.BodyBytes) }
func (r *BasicRequest) Headers() http.Header {
	if r.HeaderVals == nil {
		return http.Header{}
	}
	return r.HeaderVals
}

// BasicResponse is a simple Response implementation.
type BasicResponse struct {
	StatusCodeVal int
	BodyBytes     []byte
	HeaderVals    http.Header
}

func (r *BasicResponse) StatusCode() int      { return r.StatusCodeVal }
func (r *BasicResponse) Body() []byte         { return r.BodyBytes }
func (r *BasicResponse) Headers() http.Header { return r.HeaderVals }

// BasicRequestBuilder helps construct a BasicRequest.
type BasicRequestBuilder struct {
	req *BasicRequest
}

// NewBasicRequestBuilder returns a builder with empty values.
func NewBasicRequestBuilder() *BasicRequestBuilder {
	return &BasicRequestBuilder{req: &BasicRequest{}}
}

// Method sets the HTTP method.
func (b *BasicRequestBuilder) Method(m string) *BasicRequestBuilder {
	b.req.MethodVal = m
	return b
}

// URL sets the request URL.
func (b *BasicRequestBuilder) URL(u string) *BasicRequestBuilder {
	b.req.URLVal = u
	return b
}

// Query sets query parameters.
func (b *BasicRequestBuilder) Query(v url.Values) *BasicRequestBuilder {
	if v != nil {
		if b.req.QueryVal == nil {
			b.req.QueryVal = url.Values{}
		}
		for k, vs := range v {
			for _, val := range vs {
				b.req.QueryVal.Add(k, val)
			}
		}
	}
	return b
}

// Body sets the request body bytes.
func (b *BasicRequestBuilder) Body(body []byte) *BasicRequestBuilder {
	b.req.BodyBytes = body
	return b
}

// Header adds a single header value.
func (b *BasicRequestBuilder) Header(k, v string) *BasicRequestBuilder {
	if b.req.HeaderVals == nil {
		b.req.HeaderVals = http.Header{}
	}
	b.req.HeaderVals.Add(k, v)
	return b
}

// Headers adds multiple headers.
func (b *BasicRequestBuilder) Headers(h http.Header) *BasicRequestBuilder {
	if h != nil {
		if b.req.HeaderVals == nil {
			b.req.HeaderVals = http.Header{}
		}
		for k, vs := range h {
			for _, v := range vs {
				b.req.HeaderVals.Add(k, v)
			}
		}
	}
	return b
}

// Build returns the constructed BasicRequest.
func (b *BasicRequestBuilder) Build() *BasicRequest {
	if b.req.QueryVal == nil {
		b.req.QueryVal = url.Values{}
	}
	if b.req.HeaderVals == nil {
		b.req.HeaderVals = http.Header{}
	}
	return b.req
}

// BasicResponseBuilder helps construct a BasicResponse.
type BasicResponseBuilder struct {
	resp *BasicResponse
}

// NewBasicResponseBuilder returns a builder with empty values.
func NewBasicResponseBuilder() *BasicResponseBuilder {
	return &BasicResponseBuilder{resp: &BasicResponse{}}
}

// StatusCode sets the status code.
func (b *BasicResponseBuilder) StatusCode(code int) *BasicResponseBuilder {
	b.resp.StatusCodeVal = code
	return b
}

// Body sets the body bytes.
func (b *BasicResponseBuilder) Body(body []byte) *BasicResponseBuilder {
	b.resp.BodyBytes = body
	return b
}

// Header adds a header value.
func (b *BasicResponseBuilder) Header(k, v string) *BasicResponseBuilder {
	if b.resp.HeaderVals == nil {
		b.resp.HeaderVals = http.Header{}
	}
	b.resp.HeaderVals.Add(k, v)
	return b
}

// Headers adds multiple headers.
func (b *BasicResponseBuilder) Headers(h http.Header) *BasicResponseBuilder {
	if h != nil {
		if b.resp.HeaderVals == nil {
			b.resp.HeaderVals = http.Header{}
		}
		for k, vs := range h {
			for _, v := range vs {
				b.resp.HeaderVals.Add(k, v)
			}
		}
	}
	return b
}

// Build returns the constructed BasicResponse.
func (b *BasicResponseBuilder) Build() *BasicResponse {
	if b.resp.HeaderVals == nil {
		b.resp.HeaderVals = http.Header{}
	}
	return b.resp
}

// HTTPClientConfig provides options for constructing a fasthttp.Client.
type HTTPClientConfig struct {
	// Timeout sets the underlying http.Client timeout.
	Timeout time.Duration
}

// BreakerConfig defines CircuitBreaker settings.
type BreakerConfig struct {
	MaxFailures   int
	Window        time.Duration
	Timeout       time.Duration
	TrialRequests int
	TrialDuration time.Duration
	OnStateChange func(circuitbreaker.State)
}

// BulkheadConfig defines Bulkhead settings.
type BulkheadConfig struct {
	Limit int
}

// Config groups the optional pieces used to build a Client.
type Config struct {
	HTTP     *HTTPClientConfig
	Breaker  *BreakerConfig
	Bulkhead *BulkheadConfig
}

func buildHTTP(cfg *HTTPClientConfig) *fasthttp.Client {
	if cfg == nil {
		return &fasthttp.Client{HTTP: &http.Client{}}
	}
	return &fasthttp.Client{HTTP: &http.Client{Timeout: cfg.Timeout}}
}

func buildBreaker(cfg *BreakerConfig) *circuitbreaker.CircuitBreaker {
	if cfg == nil {
		return nil
	}
	b := circuitbreaker.NewBuilder()
	if cfg.MaxFailures > 0 {
		b.MaxFailures(cfg.MaxFailures)
	}
	if cfg.Window > 0 {
		b.Window(cfg.Window)
	}
	if cfg.Timeout > 0 {
		b.Timeout(cfg.Timeout)
	}
	if cfg.TrialRequests > 0 {
		b.TrialRequests(cfg.TrialRequests)
	}
	if cfg.TrialDuration > 0 {
		b.TrialDuration(cfg.TrialDuration)
	}
	if cfg.OnStateChange != nil {
		b.OnStateChange(cfg.OnStateChange)
	}
	return b.Build()
}

func buildBulkhead(cfg *BulkheadConfig) *bulkhead.Bulkhead {
	if cfg == nil {
		return nil
	}
	b := bulkhead.NewBuilder()
	if cfg.Limit > 0 {
		b.Limit(cfg.Limit)
	}
	return b.Build()
}

// Client wraps a fasthttp.Client with optional circuit breaker and bulkhead support.
type Client struct {
	HTTP *fasthttp.Client
	CB   *circuitbreaker.CircuitBreaker
	BH   *bulkhead.Bulkhead
}

// New creates a Client from the provided configuration.
func New(cfg *Config) *Client {
	httpClient := buildHTTP(nil)
	var cb *circuitbreaker.CircuitBreaker
	var bh *bulkhead.Bulkhead
	if cfg != nil {
		httpClient = buildHTTP(cfg.HTTP)
		cb = buildBreaker(cfg.Breaker)
		bh = buildBulkhead(cfg.Bulkhead)
	}
	return &Client{HTTP: httpClient, CB: cb, BH: bh}
}

// NewWithBreaker creates a Client using only a circuit breaker.
func NewWithBreaker(httpCfg *HTTPClientConfig, cbCfg *BreakerConfig) *Client {
	return New(&Config{HTTP: httpCfg, Breaker: cbCfg})
}

// NewWithBulkhead creates a Client using only a bulkhead.
func NewWithBulkhead(httpCfg *HTTPClientConfig, bhCfg *BulkheadConfig) *Client {
	return New(&Config{HTTP: httpCfg, Bulkhead: bhCfg})
}

// NewWithBreakerAndBulkhead creates a Client using both patterns.
func NewWithBreakerAndBulkhead(httpCfg *HTTPClientConfig, cbCfg *BreakerConfig, bhCfg *BulkheadConfig) *Client {
	return New(&Config{HTTP: httpCfg, Breaker: cbCfg, Bulkhead: bhCfg})
}

// NewPlain creates a Client with no circuit breaker or bulkhead.
func NewPlain(httpCfg *HTTPClientConfig) *Client {
	return New(&Config{HTTP: httpCfg})
}

// Call sends the request through the breaker and bulkhead and returns a response.
func (c *Client) Call(ctx context.Context, req Request) (Response, error) {
	u, err := url.Parse(req.URL())
	if err != nil {
		return nil, err
	}
	q := u.Query()
	for k, vs := range req.Query() {
		for _, v := range vs {
			q.Add(k, v)
		}
	}
	u.RawQuery = q.Encode()

	var fr fasthttp.Request
	fr.Header.SetMethod(req.Method())
	fr.SetRequestURI(u.String())
	for k, vs := range req.Headers() {
		for _, v := range vs {
			fr.Header.Add(k, v)
		}
	}
	if b, err := io.ReadAll(req.Body()); err == nil {
		fr.SetBody(b)
	} else {
		return nil, err
	}

	var resp fasthttp.Response
	callFn := func() error {
		if deadline, ok := ctx.Deadline(); ok {
			err = c.HTTP.DoDeadline(&fr, &resp, deadline)
		} else {
			err = c.HTTP.Do(&fr, &resp)
		}
		if err != nil {
			return err
		}
		if resp.StatusCode() >= fasthttp.StatusInternalServerError {
			return fmt.Errorf("server error: %d", resp.StatusCode())
		}
		return nil
	}

	if err := c.execute(callFn); err != nil {
		return &BasicResponse{StatusCodeVal: resp.StatusCode(), BodyBytes: append([]byte(nil), resp.Body()...), HeaderVals: convertHeaders(&resp)}, err
	}

	return &BasicResponse{StatusCodeVal: resp.StatusCode(), BodyBytes: append([]byte(nil), resp.Body()...), HeaderVals: convertHeaders(&resp)}, nil
}

func (c *Client) execute(fn func() error) error {
	if c.CB != nil && c.BH != nil {
		return c.CB.Execute(func() error { return c.BH.Execute(fn) })
	}
	if c.CB != nil {
		return c.CB.Execute(fn)
	}
	if c.BH != nil {
		return c.BH.Execute(fn)
	}
	return fn()
}

func convertHeaders(resp *fasthttp.Response) http.Header {
	h := http.Header{}
	resp.Header.VisitAll(func(k, v []byte) {
		h.Add(string(k), string(v))
	})
	return h
}
