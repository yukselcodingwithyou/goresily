package fasthttp

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"
)

const StatusInternalServerError = http.StatusInternalServerError

type Header struct {
	method  string
	headers http.Header
}

func (h *Header) ensure() {
	if h.headers == nil {
		h.headers = make(http.Header)
	}
}

func (h *Header) SetMethod(m string) {
	h.method = m
}

func (h *Header) Add(k, v string) {
	h.ensure()
	h.headers.Add(k, v)
}

func (h *Header) VisitAll(fn func(k, v []byte)) {
	if h.headers == nil {
		return
	}
	for k, vs := range h.headers {
		for _, v := range vs {
			fn([]byte(k), []byte(v))
		}
	}
}

type Request struct {
	Header Header
	uri    string
	body   []byte
}

func (r *Request) SetRequestURI(u string) {
	r.uri = u
}

func (r *Request) SetBody(b []byte) {
	r.body = b
}

type Response struct {
	Header     Header
	statusCode int
	body       []byte
}

func (r *Response) StatusCode() int { return r.statusCode }
func (r *Response) Body() []byte    { return r.body }

// Client performs HTTP requests. It wraps net/http.Client for simplicity.
type Client struct {
	HTTP *http.Client
}

func (c *Client) ensure() {
	if c.HTTP == nil {
		c.HTTP = &http.Client{}
	}
}

func (c *Client) Do(req *Request, resp *Response) error {
	c.ensure()
	return c.do(req, resp, func(r *http.Request) (*http.Response, error) {
		return c.HTTP.Do(r)
	})
}

func (c *Client) DoDeadline(req *Request, resp *Response, deadline time.Time) error {
	c.ensure()
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()
	return c.do(req, resp, func(r *http.Request) (*http.Response, error) {
		return c.HTTP.Do(r.WithContext(ctx))
	})
}

func (c *Client) do(req *Request, resp *Response, fn func(*http.Request) (*http.Response, error)) error {
	httpReq, err := http.NewRequest(req.Header.method, req.uri, bytes.NewReader(req.body))
	if err != nil {
		return err
	}
	if req.Header.headers != nil {
		httpReq.Header = req.Header.headers.Clone()
	}
	httpResp, err := fn(httpReq)
	if err != nil {
		return err
	}
	defer httpResp.Body.Close()
	resp.statusCode = httpResp.StatusCode
	resp.Header.headers = httpResp.Header.Clone()
	resp.Header.method = req.Header.method
	resp.body, err = io.ReadAll(httpResp.Body)
	return err
}
