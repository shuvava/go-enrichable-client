package client

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"testing"
)

var (
	url            = "https://www.example.com"
	wantStatusCode = http.StatusOK
	wantBody       = `OK`
)

func TestWithOutMiddleware(t *testing.T) {
	mock := createMock(url, wantStatusCode, wantBody)

	t.Run("Should successfully process request without middleware", func(t *testing.T) {
		richClient := NewHTTPClient(mock)
		client := richClient.Client
		response, err := client.Get(url)
		assertResponse(t, response, err)
	})
}

func TestMiddleware(t *testing.T) {
	mock := createMock(url, wantStatusCode, wantBody)
	richClient := NewHTTPClient(mock)
	richClient.Use(createMiddleware(http.MethodHead, http.StatusConflict))
	client := richClient.Client

	t.Run("Should use default responder", func(t *testing.T) {
		response, err := client.Get(url)
		assertResponse(t, response, err)
	})
	t.Run("Should use middleware responder", func(t *testing.T) {
		response, err := client.Head(url)
		if err != nil {
			t.Fatalf("did not expect an error but got one %v", err)
		}
		if response.StatusCode != http.StatusConflict {
			t.Errorf("got %d, wantStatusCode %d", response.StatusCode, wantStatusCode)
		}
	})
}

func TestMultipleMiddleware(t *testing.T) {
	mock := createMock(url, wantStatusCode, wantBody)
	richClient := NewHTTPClient(mock)
	richClient.Use(createMiddleware(http.MethodHead, http.StatusBadGateway))
	richClient.Use(createMiddleware(http.MethodHead, http.StatusConflict))
	client := richClient.Client

	t.Run("Should apply middleware from first to last", func(t *testing.T) {
		response, err := client.Head(url)
		if err != nil {
			t.Fatalf("did not expect an error but got one %v", err)
		}
		if response.StatusCode != http.StatusBadGateway {
			t.Errorf("got %d, wantStatusCode %d", response.StatusCode, wantStatusCode)
		}
	})
}

func assertResponse(t testing.TB, response *http.Response, err error) {
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

func createMiddleware(method string, statusCode int) MiddlewareFunc {
	return func(c *http.Client, next Responder) Responder {
		return func(request *http.Request) (*http.Response, error) {
			if request.Method == method {
				return &http.Response{
					StatusCode: statusCode,
					// Send response to be tested
					Body: nil,
					// Must be set to non-nil value or it panics
					Header: make(http.Header),
				}, nil
			}
			return next(request)
		}
	}
}

func createMock(url string, statusCode int, body string) *MockTransport {
	mock := NewMockTransport(true)
	mock.RegisterResponder(http.MethodGet, url,
		func(request *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: statusCode,
				// Send response to be tested
				Body: ioutil.NopCloser(bytes.NewBufferString(body)),
				// Must be set to non-nil value or it panics
				Header: make(http.Header),
			}, nil
		})
	return mock
}
