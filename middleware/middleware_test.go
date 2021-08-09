package middleware_test

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/shuvava/go-enrichable-client/client"
)

type httpMock struct {
	calls int // number of retries
	mock  *client.MockTransport
}

type responseMock struct {
	StatusCode int
	Body       string
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

func createGetMock(url string, statusCode int, body string, errCnt, errCode int) *httpMock {
	return createMock(http.MethodGet, url, statusCode, body, errCnt, errCode)
}

func createPostMock(url string, statusCode int, body string, errCnt, errCode int) *httpMock {
	return createMock(http.MethodPost, url, statusCode, body, errCnt, errCode)
}

func createMock(method, url string, statusCode int, body string, errCnt, errCode int) *httpMock {
	m := &httpMock{
		mock: client.NewMockTransport(true),
	}
	m.mock.RegisterResponder(method, url,
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

func createMockMultiResponse(method, url string, responses []responseMock) *httpMock {
	m := &httpMock{
		mock: client.NewMockTransport(true),
	}
	m.mock.RegisterResponder(method, url,
		func(request *http.Request) (*http.Response, error) {
			l := len(responses)
			i := l - 1
			if l > m.calls {
				i = m.calls
			}
			code := responses[i].StatusCode
			body := responses[i].Body
			m.calls++
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
