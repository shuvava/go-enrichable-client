package client

import (
	"bytes"
	"context"
	"net/http"
	"testing"
)

func TestRequest(t *testing.T) {
	createRequest := func(t testing.TB) *Request {
		t.Helper()
		body := bytes.NewReader([]byte("yo"))
		req, err := NewRequest(context.Background(), "GET", "/", body)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		return req
	}
	t.Run("Fails on invalid request", func(t *testing.T) {
		_, err := NewRequest(context.Background(), "GET", "://foo", nil)
		if err == nil {
			t.Fatalf("should error")
		}
	})
	t.Run("Works with no request body", func(t *testing.T) {
		_, err := NewRequest(context.Background(), "GET", "https://foo", nil)
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
