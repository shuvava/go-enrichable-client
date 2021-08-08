package middleware_test

import (
	"net/http"
	"testing"

	"github.com/shuvava/go-enrichable-client/client"
	"github.com/shuvava/go-enrichable-client/middleware"
)

const defaultRetryMax = 3

func TestRetryableMiddleware(t *testing.T) {
	t.Run("Should not do any retry on successful response code", func(t *testing.T) {
		var (
			url            = "https://www.example.com"
			wantStatusCode = http.StatusOK
			wantBody       = `error`
		)
		m := createGetMock(url, wantStatusCode, wantBody, -1, 0)
		richClient := client.NewClient(m.mock)
		richClient.Use(middleware.Retry())
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
		m := createGetMock(url, wantStatusCode, wantBody, -1, 0)
		richClient := client.NewClient(m.mock)
		richClient.Use(middleware.RetryWithConfig(newRetryConfig()))
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
		m := createGetMock(url, wantStatusCode, wantBody, -1, 0)
		richClient := client.NewClient(m.mock)
		richClient.Use(middleware.RetryWithConfig(newRetryConfig()))
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
		m := createGetMock(url, wantStatusCode, wantBody, 2, http.StatusInternalServerError)
		richClient := client.NewClient(m.mock)
		richClient.Use(middleware.RetryWithConfig(newRetryConfig()))
		c := richClient.Client

		response, err := c.Get(url)
		assertResponse(t, response, err, wantStatusCode, wantBody)

		if m.calls != 3 {
			t.Errorf("retry got %d, expected %d", m.calls, 3)
		}
	})
}

func newRetryConfig() middleware.RetryConfig {
	return middleware.RetryConfig{
		RetryWaitMin: 0,
		RetryWaitMax: 0,
		RetryMax:     defaultRetryMax,
		CheckRetry:   middleware.DefaultRetryPolicy,
		Backoff:      middleware.DefaultBackoff,
	}
}
