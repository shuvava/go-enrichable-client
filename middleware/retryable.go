package middleware

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/shuvava/go-enrichable-client/client"
)

const (
	// Default retry configuration
	defaultRetryWaitMin = 1 * time.Second
	defaultRetryWaitMax = 30 * time.Second
	defaultRetryMax     = 3

	// We need to consume response bodies to maintain http connections, but
	// limit the size we consume to respBodyReadLimit.
	respBodyReadLimit = 1024
)

type (
	// Backoff specifies a policy for how long to wait between retries.
	// It is called after a failing request to determine the amount of time
	// that should pass before trying again.
	Backoff func(min, max time.Duration, attemptNum int, resp *http.Response) time.Duration

	// CheckRetry specifies a policy for handling retries. It is called
	// following each request with the response and error values returned by
	// the http.Client. If CheckRetry returns false, the RetryableClient stops retrying
	// and returns the response to the caller. If CheckRetry returns an error,
	// that error value is returned in lieu of the error from the request. The
	// RetryableClient will close any response body when retrying, but if the retry is
	// aborted it is up to the CheckRetry callback to properly close any
	// response body before returning.
	CheckRetry func(ctx context.Context, resp *http.Response, err error) (bool, error)

	// RequestHook allows a function to run before each HTTP request.
	RequestHook func(*http.Request)

	// RetryConfig middleware config
	RetryConfig struct {
		RetryWaitMin time.Duration // Minimum time to wait
		RetryWaitMax time.Duration // Maximum time to wait
		RetryMax     int           // Maximum number of retries

		// CheckRetry specifies the policy for handling retries, and is called
		// after each request. The default policy is DefaultRetryPolicy.
		CheckRetry CheckRetry

		// RequestHook allows a user-supplied function to be called
		// with each HTTP request executed.
		RequestHook RequestHook

		// Backoff specifies the policy for how long to wait between retries
		Backoff Backoff
	}
)

var (
	// DefaultRetryConfig is default Retry middleware config
	DefaultRetryConfig = RetryConfig{
		RetryWaitMin: defaultRetryWaitMin,
		RetryWaitMax: defaultRetryWaitMax,
		RetryMax:     defaultRetryMax,
		CheckRetry:   DefaultRetryPolicy,
		Backoff:      DefaultBackoff,
	}
)

// SetRequestHook set a user-supplied function to be called
// with each HTTP request executed.
func (c *RetryConfig) SetRequestHook(hook RequestHook) {
	c.RequestHook = hook
}

// SetRetryMax set a maximum number of retries
func (c *RetryConfig) SetRetryMax(max int) error {
	if max <= 0 {
		return fmt.Errorf("retry count cannot be less 0")
	}

	c.RetryMax = max

	return nil
}

// DefaultRetryPolicy provides a default callback for ClientOld.CheckRetry, which
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
	//if resp.StatusCode == 0 || (resp.StatusCode >= 500 && resp.StatusCode != 501) {
	if resp.StatusCode >= 500 && resp.StatusCode != 501 {
		return true, nil
	}
	if resp.StatusCode == 0 {
		return true, fmt.Errorf("unexpected HTTP status %s", resp.Status)
	}

	return false, nil
}

// DefaultBackoff provides a default callback for Client.Backoff which
// will perform exponential backoff based on the attempt number and limited
// by the provided minimum and maximum durations.
//
// It also tries to parse Retry-After response header when a http.StatusTooManyRequests
// (HTTP Code 429) is found in the resp parameter. Hence it will return the number of
// seconds the server states it may be ready to process more requests from this client.
func DefaultBackoff(min, max time.Duration, attemptNum int, resp *http.Response) time.Duration {
	if resp != nil {
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable {
			if s, ok := resp.Header["Retry-After"]; ok {
				if sleep, err := strconv.ParseInt(s[0], 10, 64); err == nil {
					return time.Second * time.Duration(sleep)
				}
			}
		}
	}

	mult := math.Pow(2, float64(attemptNum)) * float64(min)
	sleep := time.Duration(mult)
	if float64(sleep) != mult || sleep > max {
		sleep = max
	}
	return sleep
}

// LinearJitterBackoff provides a callback for Client.Backoff which will
// perform linear backoff based on the attempt number and with jitter to
// prevent a thundering herd.
//
// min and max here are *not* absolute values. The number to be multiplied by
// the attempt number will be chosen at random from between them, thus they are
// bounding the jitter.
//
// For instance:
// * To get strictly linear backoff of one second increasing each retry, set
// both to one second (1s, 2s, 3s, 4s, ...)
// * To get a small amount of jitter centered around one second increasing each
// retry, set to around one second, such as a min of 800ms and max of 1200ms
// (892ms, 2102ms, 2945ms, 4312ms, ...)
// * To get extreme jitter, set to a very wide spread, such as a min of 100ms
// and a max of 20s (15382ms, 292ms, 51321ms, 35234ms, ...)
func LinearJitterBackoff(min, max time.Duration, attemptNum int, resp *http.Response) time.Duration {
	// attemptNum always starts at zero but we want to start at 1 for multiplication
	attemptNum++

	if max <= min {
		// Unclear what to do here, or they are the same, so return min *
		// attemptNum
		return min * time.Duration(attemptNum)
	}

	// Seed rand; doing this every time is fine
	rand := rand.New(rand.NewSource(int64(time.Now().Nanosecond())))

	// Pick a random number that lies somewhere between the min and max and
	// multiply by the attemptNum. attemptNum starts at zero so we always
	// increment here. We first get a random percentage, then apply that to the
	// difference between min and max, and add to min.
	jitter := rand.Float64() * float64(max-min)
	jitterMin := int64(jitter) + int64(min)
	return time.Duration(jitterMin * int64(attemptNum))
}

// Try to read the response body so we can reuse this connection.
func drainBody(body io.ReadCloser) {
	defer func() {
		_ = body.Close()
	}()
	_, _ = io.Copy(ioutil.Discard, io.LimitReader(body, respBodyReadLimit))
}

// Retry creates retry middleware with DefaultRetryConfig
func Retry() client.MiddlewareFunc {
	c := DefaultRetryConfig
	return RetryWithConfig(c)
}

// RetryWithConfig creates retry middleware with config
func RetryWithConfig(config RetryConfig) client.MiddlewareFunc {
	return func(c *http.Client, next client.Responder) client.Responder {
		return func(request *http.Request) (*http.Response, error) {
			var resp *http.Response
			var shouldRetry bool
			var attempt int
			var doErr, checkErr error
			req, err := FromRequest(request)
			if err != nil {
				return nil, err
			}
			for i := 0; ; i++ {
				attempt++
				// Always rewind the http body when non-nil.
				if req.body != nil {
					body, err := req.body()
					if err != nil {
						c.CloseIdleConnections()
						return resp, err
					}

					if c, ok := body.(io.ReadCloser); ok {
						req.Body = c
					} else {
						req.Body = ioutil.NopCloser(body)
					}
				}

				if config.RequestHook != nil {
					config.RequestHook(req.Request)
				}

				resp, doErr = next(request)

				// Check if we should continue with retries.
				shouldRetry, checkErr = config.CheckRetry(req.Context(), resp, doErr)
				if !shouldRetry {
					break
				}

				// We do this before drainBody because there's no need for the I/O if
				// we're breaking out
				remain := config.RetryMax - i
				if remain <= 0 {
					break
				}

				if doErr == nil && resp.Body != nil {
					drainBody(resp.Body)
				}
				wait := config.Backoff(config.RetryWaitMin, config.RetryWaitMax, i, resp)
				select {
				case <-req.Context().Done():
					c.CloseIdleConnections()
					return nil, req.Context().Err()
				case <-time.After(wait):
				}
			}

			// this is the closest we have to success criteria
			if doErr == nil && checkErr == nil && !shouldRetry {
				return resp, nil
			}

			defer c.CloseIdleConnections()

			err = doErr
			if checkErr != nil {
				err = checkErr
			}

			if err == nil {
				return resp, nil
			}

			return nil, fmt.Errorf("%s %s giving up after %d attempt(s): %w",
				req.Method, req.URL, attempt, err)
		}
	}
}
