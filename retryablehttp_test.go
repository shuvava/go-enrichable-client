package retryablehttp

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

// equals fails the test if exp is not equal to act.
func equals(tb testing.TB, exp, act interface{}) {
	if !reflect.DeepEqual(exp, act) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, exp, act)
		tb.FailNow()
	}
}

func TestRequest(t *testing.T) {
	createRequest := func(t testing.TB) *Request {
		t.Helper()
		body := bytes.NewReader([]byte("yo"))
		req, err := NewRequest("GET", "/", body)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		return req
	}
	t.Run("Fails on invalid request", func(t *testing.T) {
		_, err := NewRequest("GET", "://foo", nil)
		if err == nil {
			t.Fatalf("should error")
		}
	})
	t.Run("Works with no request body", func(t *testing.T) {
		_, err := NewRequest("GET", "https://foo", nil)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
	})

	t.Run("Works with request body", func(t *testing.T) {
		_ = createRequest(t)
	})
	t.Run("Request allows typical HTTP request forming methods", func(t *testing.T) {
		req := createRequest(t)
		req.Header.Set("X-Test", "foo")
		if v, ok := req.Header["X-Test"]; !ok || len(v) != 1 || v[0] != "foo" {
			t.Fatalf("bad headers: %v", req.Header)
		}
	})
	t.Run("Sets the Content-Length automatically", func(t *testing.T) {
		req := createRequest(t)
		if req.ContentLength != 2 {
			t.Fatalf("bad ContentLength: %d", req.ContentLength)
		}
	})
}

func TestFromRequest(t *testing.T) {
	createRequest := func(t testing.TB) (*http.Request, *Request) {
		t.Helper()
		body := bytes.NewReader([]byte("yo"))
		httpReq, err := http.NewRequest("GET", "/", body)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		req, err := FromRequest(httpReq)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		return httpReq, req
	}
	t.Run("Works with no request body", func(t *testing.T) {
		httpReq, err := http.NewRequest("GET", "https://foo", nil)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		_, err = FromRequest(httpReq)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
	})
	t.Run("Works with request body", func(t *testing.T) {
		_, _ = createRequest(t)
	})
	t.Run("Preserves headers", func(t *testing.T) {
		httpReq, req := createRequest(t)
		httpReq.Header.Set("X-Test", "foo")
		if v, ok := req.Header["X-Test"]; !ok || len(v) != 1 || v[0] != "foo" {
			t.Fatalf("bad headers: %v", req.Header)
		}
	})
	t.Run("Preserves the Content-Length automatically", func(t *testing.T) {
		_, req := createRequest(t)
		if req.ContentLength != 2 {
			t.Fatalf("bad ContentLength: %d", req.ContentLength)
		}
	})
}

func TestSuccessfulResponse(t *testing.T) {
	t.Run("Successful GET of mocked request", func(t *testing.T) {
		mock := NewMockTransport(true)
		url := "https://www.example.com"
		wantStatusCode := http.StatusOK
		wantBody := `OK`
		mock.RegisterResponder(http.MethodGet, url,
			func(request *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: wantStatusCode,
					// Send response to be tested
					Body: ioutil.NopCloser(bytes.NewBufferString(wantBody)),
					// Must be set to non-nil value or it panics
					Header: make(http.Header),
				}, nil
			})
		client := NewClient(mock)
		response, err := client.Get(url)
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
			t.Errorf("got %q, wantStatusCode %q", response.StatusCode, wantStatusCode)
		}
	})
}
