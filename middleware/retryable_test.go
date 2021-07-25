package middleware

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/shuvava/go-enrichable-client/client"
)

func TestRetryableMiddleware(t *testing.T) {
	t.Run("Should not do any retry on successful response code", func(t *testing.T) {
		var (
			url            = "https://www.example.com"
			wantStatusCode = http.StatusOK
			wantBody       = `error`
		)
		m := createMock(url, wantStatusCode, wantBody, -1, 0)
		richClient := client.NewClient(m.mock)
		richClient.Use(Retry())
		c := richClient.Client

		response, err := c.Get(url)
		assertResponse(t, response, err, wantStatusCode, wantBody)

		if m.calls != 1 {
			t.Errorf("retry got %d, expected %d", m.calls, 1)
		}
	})
	t.Run("Should not do any retry on not retryable response code", func(t *testing.T) {
		var (
			url            = "https://www.example.com"
			wantStatusCode = http.StatusBadRequest
			wantBody       = `error`
		)
		m := createMock(url, wantStatusCode, wantBody, -1, 0)
		richClient := client.NewClient(m.mock)
		richClient.Use(RetryWithConfig(newRetryConfig()))
		c := richClient.Client

		response, err := c.Get(url)
		assertResponse(t, response, err, wantStatusCode, wantBody)

		if m.calls != 1 {
			t.Errorf("retry got %d, expected %d", m.calls, 1)
		}
	})
	t.Run("Should stop retry after defaultRetryMax", func(t *testing.T) {
		var (
			url            = "https://www.example.com"
			wantStatusCode = http.StatusInternalServerError
			wantBody       = `error`
		)
		m := createMock(url, wantStatusCode, wantBody, -1, 0)
		richClient := client.NewClient(m.mock)
		richClient.Use(RetryWithConfig(newRetryConfig()))
		c := richClient.Client

		response, err := c.Get(url)
		assertResponse(t, response, err, wantStatusCode, wantBody)

		if m.calls != defaultRetryMax+1 {
			t.Errorf("retry got %d, expected %d", m.calls, defaultRetryMax+1)
		}
	})
	t.Run("Should stop retry on successful response code", func(t *testing.T) {
		var (
			url            = "https://www.example.com"
			wantStatusCode = http.StatusOK
			wantBody       = `error`
		)
		m := createMock(url, wantStatusCode, wantBody, 2, http.StatusInternalServerError)
		richClient := client.NewClient(m.mock)
		richClient.Use(RetryWithConfig(newRetryConfig()))
		c := richClient.Client

		response, err := c.Get(url)
		assertResponse(t, response, err, wantStatusCode, wantBody)

		if m.calls != 3 {
			t.Errorf("retry got %d, expected %d", m.calls, 3)
		}
	})
}

func assertResponse(t testing.TB, response *http.Response, err error, wantStatusCode int, wantBody string) {
	if err != nil {
		t.Fatalf("did not expect an error but got one %v", err)
	}
	if response.StatusCode != wantStatusCode {
		t.Errorf("got %q, wantStatusCode %q", response.StatusCode, wantStatusCode)
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("did not expect an error but got one %v", err)
	}
	bodyString := string(body)
	if bodyString != wantBody {
		t.Errorf("got %d, wantStatusCode %d", response.StatusCode, wantStatusCode)
	}
}

type RetryMock struct {
	calls int // number of retries
	mock  *client.MockTransport
}

func createMock(url string, statusCode int, body string, errCnt, errCode int) *RetryMock {
	m := &RetryMock{
		mock: client.NewMockTransport(true),
	}
	m.mock.RegisterResponder(http.MethodGet, url,
		func(request *http.Request) (*http.Response, error) {
			m.calls++
			code := errCode
			if m.calls-1 >= errCnt {
				code = statusCode
			}
			return &http.Response{
				StatusCode: code,
				// Send response to be tested
				Body: ioutil.NopCloser(bytes.NewBufferString(body)),
				// Must be set to non-nil value or it panics
				Header: make(http.Header),
			}, nil
		})
	return m
}

func newRetryConfig() RetryConfig {
	return RetryConfig{
		RetryWaitMin: 0,
		RetryWaitMax: 0,
		RetryMax:     defaultRetryMax,
		CheckRetry:   DefaultRetryPolicy,
		Backoff:      DefaultBackoff,
	}
}
