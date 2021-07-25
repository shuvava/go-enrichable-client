package client

import (
	"net/http"
)

// MiddlewareFunc defines a function to process middleware.
type MiddlewareFunc func(*http.Client, Responder) Responder

// Client is a wrapper on the top of http.Client allowing add rich functions
type Client struct {
	defaultResponder Responder
	middleware       []MiddlewareFunc
	Client           *http.Client
}

// NewClient creates http.Client with provided transport
func NewClient(transport http.RoundTripper) *Client {
	if transport == nil {
		transport = http.DefaultClient.Transport
	}
	client := &Client{
		defaultResponder: transport.RoundTrip,
	}
	client.Client = NewHTTPClient(client)

	return client
}

// DefaultClient returns a new Client with similar default values to
// http.Client, but with a non-shared Transport, idle connections disabled, and
// keepalives disabled.
func DefaultClient() *Client {
	return NewClient(DefaultTransport())
}

// DefaultPooledClient returns a new Client with similar default values to
// http.Client, but with a shared Transport. Do not use this function for
// transient clients as it can leak file descriptors over time. Only use this
// for clients that will be re-used for the same host(s).
func DefaultPooledClient() *Client {
	return NewClient(DefaultPooledTransport())
}

// Use adds middleware to the chain which is run on processing request.
func (c *Client) Use(middleware ...MiddlewareFunc) {
	c.middleware = append(c.middleware, middleware...)
}

// RoundTrip executes a single HTTP transaction, returning a Response for the provided Request
func (c *Client) RoundTrip(req *http.Request) (*http.Response, error) {
	h := applyMiddleware(c.Client, c.defaultResponder, c.middleware...)
	return h(req)
}

func applyMiddleware(c *http.Client, h Responder, middleware ...MiddlewareFunc) Responder {
	for i := len(middleware) - 1; i >= 0; i-- {
		h = middleware[i](c, h)
	}
	return h
}
