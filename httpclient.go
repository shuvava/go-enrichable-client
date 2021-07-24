package retryablehttp

import (
	"net/http"
)

// HTTPClient is a wrapper for the Go http.Client
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
	CloseIdleConnections()
}
