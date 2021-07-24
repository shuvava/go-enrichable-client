package retryablehttp

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"
)

const defaultRetryMax = 3

// We need to consume response bodies to maintain http connections, but
// limit the size we consume to respBodyReadLimit.
const respBodyReadLimit = 1024

// Backoff specifies a policy for how long to wait between retries.
// It is called after a failing request to determine the amount of time
// that should pass before trying again.
type Backoff func(attemptNum int, resp *http.Response) time.Duration

//type EventHandler

// CheckRetry specifies a policy for handling retries. It is called
// following each request with the response and error values returned by
// the http.Client. If CheckRetry returns false, the RetryableClient stops retrying
// and returns the response to the caller. If CheckRetry returns an error,
// that error value is returned in lieu of the error from the request. The
// RetryableClient will close any response body when retrying, but if the retry is
// aborted it is up to the CheckRetry callback to properly close any
// response body before returning.
type CheckRetry func(ctx context.Context, resp *http.Response, err error) (bool, error)

// RequestHook allows a function to run before each HTTP request.
type RequestHook func(*http.Request)

// Client implements client for HTTP REST endpoints
type Client struct {
	RetryableClient
	httpClient HTTPClient // Internal HTTP client.
	RetryMax   int        // Maximum number of retries
	// CheckRetry specifies the policy for handling retries, and is called
	// after each request. The default policy is DefaultRetryPolicy.
	CheckRetry CheckRetry
	// RequestHook allows a user-supplied function to be called
	// with each HTTP request executed.
	RequestHook RequestHook
}

// RetryableClient defines the interface for a HTTP client
type RetryableClient interface {
	Do(*Request) (*http.Response, error)
	SetRequestHook(RequestHook)
	SetRetryMax(int) error
}

// DefaultRetryPolicy provides a default callback for Client.CheckRetry, which
// will retry on connection errors and server errors.
func DefaultRetryPolicy(ctx context.Context, resp *http.Response, err error) (bool, error) {
	// do not retry on context.Canceled or context.DeadlineExceeded
	if ctx.Err() != nil {
		return false, ctx.Err()
	}

	if err != nil {
		return true, nil
	}

	// 429 Too Many Requests is recoverable.
	if resp.StatusCode == http.StatusTooManyRequests {
		return true, nil
	}

	// Check the response code. We retry on 500-range responses to allow
	// the server time to recover, as 500's are typically not permanent
	// errors and may relate to outages on the server side. This will catch
	// invalid response codes as well, like 0 and 999.
	if resp.StatusCode == 0 || (resp.StatusCode >= 500 && resp.StatusCode != 501) {
		return true, fmt.Errorf("unexpected HTTP status %s", resp.Status)
	}

	return false, nil
}

// NewRetryableClient creates a retryable http client
func NewRetryableClient(transport http.RoundTripper) *Client {
	if transport == nil {
		transport = DefaultPooledTransport()
	}
	httpClient := NewClient(transport)
	return &Client{
		httpClient: httpClient,
		RetryMax:   defaultRetryMax,
		CheckRetry: DefaultRetryPolicy,
	}
}

// Try to read the response body so we can reuse this connection.
func (client *Client) drainBody(body io.ReadCloser) {
	defer func() {
		_ = body.Close()
	}()
	_, _ = io.Copy(ioutil.Discard, io.LimitReader(body, respBodyReadLimit))
}

// Do wraps calling an HTTP method with retries.
func (client *Client) Do(req *Request) (*http.Response, error) {
	var resp *http.Response
	var shouldRetry bool
	var attempt int
	var doErr, checkErr error
	for i := 0; ; i++ {
		attempt++
		// Always rewind the http body when non-nil.
		if req.body != nil {
			body, err := req.body()
			if err != nil {
				client.httpClient.CloseIdleConnections()
				return resp, err
			}

			if c, ok := body.(io.ReadCloser); ok {
				req.Body = c
			} else {
				req.Body = ioutil.NopCloser(body)
			}

			if client.RequestHook != nil {
				client.RequestHook(req.Request)
			}

			resp, doErr = client.httpClient.Do(req.Request)

			// Check if we should continue with retries.
			shouldRetry, checkErr = client.CheckRetry(req.Context(), resp, doErr)
			if !shouldRetry {
				break
			}

			// We do this before drainBody because there's no need for the I/O if
			// we're breaking out
			remain := client.RetryMax - i
			if remain <= 0 {
				break
			}

			if doErr == nil && resp.Body != nil {
				client.drainBody(resp.Body)
			}
		}
	}
	// this is the closest we have to success criteria
	if doErr == nil && checkErr == nil && !shouldRetry {
		return resp, nil
	}

	defer client.httpClient.CloseIdleConnections()

	err := doErr
	if checkErr != nil {
		err = checkErr
	}

	if err == nil {
		return nil, fmt.Errorf("%s %s giving up after %d attempt(s)",
			req.Method, req.URL, attempt)
	}

	return nil, fmt.Errorf("%s %s giving up after %d attempt(s): %w",
		req.Method, req.URL, attempt, err)
}

// SetRequestHook set a user-supplied function to be called
// with each HTTP request executed.
func (client *Client) SetRequestHook(hook RequestHook) {
	client.RequestHook = hook
}

// SetRetryMax set a maximum number of retries
func (client *Client) SetRetryMax(max int) error {
	if max <= 0 {
		return fmt.Errorf("retry count cannot be less 0")
	}

	client.RetryMax = max

	return nil
}

// Do is short hand for processing HTTP request object
func (req *Request) Do(client *Client) (*http.Response, error) {
	return client.Do(req)
}
