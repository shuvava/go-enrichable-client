package client

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

var (
	// defaultClient is used for performing requests without explicitly making
	// a new client. It is purposely private to avoid modifications.
	defaultClient = DefaultClient()
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

// Get is a convenience helper for doing simple GET requests.
func (c *Client) Get(url string, response interface{}) error {
	resp, err := c.Client.Get(url)
	if err != nil {
		return err
	}

	return ReadResponse(resp, &response)
}

// Get is a shortcut for doing a GET request without making a new client.
func Get(url string, response interface{}) error {
	return defaultClient.Get(url, &response)
}

func (c *Client) sendRestRequest(method, url string, body interface{}, response interface{}) error {
	req, err := NewHTTPRequest(method, url, body)
	if err != nil {
		return err
	}
	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}

	return ReadResponse(resp, &response)
}

// Post is a convenience method for doing simple POST requests.
func (c *Client) Post(url string, body interface{}, response interface{}) error {
	return c.sendRestRequest("POST", url, body, &response)
}

// Post is a shortcut for doing a POST request without making a new client.
func Post(url string, body interface{}, response interface{}) error {
	return defaultClient.Post(url, body, &response)
}

// Put is a convenience method for doing simple PUT requests.
func (c *Client) Put(url string, body interface{}, response interface{}) error {
	return c.sendRestRequest("PUT", url, body, &response)
}

// Put is a shortcut for doing a PUT request without making a new client.
func Put(url string, body interface{}, response interface{}) error {
	return defaultClient.Put(url, body, &response)
}

// Delete is a convenience method for doing simple DELETE requests.
func (c *Client) Delete(url string, body interface{}, response interface{}) error {
	return c.sendRestRequest("DELETE", url, body, &response)
}

// Delete is a shortcut for doing a DELETE request without making a new client.
func Delete(url string, body interface{}, response interface{}) error {
	return defaultClient.Delete(url, body, &response)
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

// AssertStatusCode verify if response status code is successful
func AssertStatusCode(resp *http.Response) error {
	if resp == nil {
		return nil
	}
	if (resp.StatusCode >= http.StatusOK && resp.StatusCode <= 299) ||
		resp.StatusCode == http.StatusNotFound ||
		resp.StatusCode == http.StatusNotModified {
		return nil
	}
	return fmt.Errorf("unexpected HTTP status %s", resp.Status)
}

// ReadResponse read JSON response and return deserialized object
func ReadResponse(resp *http.Response, response interface{}) error {
	if err := AssertStatusCode(resp); err != nil {
		return err
	}
	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(bodyBytes, &response)
	return err
}
