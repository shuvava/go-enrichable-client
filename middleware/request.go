package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const jsonContentType = "application/json"

// ReaderFunc is the type of function that can be given natively to NewRequest
type ReaderFunc func() (io.Reader, error)

// Request wraps the metadata needed to create HTTP requests.
type Request struct {
	// body is a seekable reader over the request body payload. This is
	// used to rewind the request data in between retries.
	body ReaderFunc

	// Embed an HTTP request directly. This makes a *Request act exactly
	// like an *http.Request so that all meta methods are supported.
	*http.Request
}

// WithContext returns wrapped Request with a shallow copy of underlying *http.Request
// with its context changed to ctx. The provided ctx must be non-nil.
func (r *Request) WithContext(ctx context.Context) *Request {
	r.Request = r.Request.WithContext(ctx)
	return r
}

func getBodyReaderAndContentLength(body interface{}) (ReaderFunc, int64, error) {
	if body == nil {
		return nil, 0, nil
	}
	buf, err := json.Marshal(body)
	if err != nil {
		return nil, 0, err
	}
	bodyReader := func() (io.Reader, error) {
		return bytes.NewReader(buf), nil
	}
	contentLength := int64(len(buf))

	return bodyReader, contentLength, nil
}

// FromRequest wraps an http.Request in a retryablehttp.Request
func FromRequest(r *http.Request) (*Request, error) {
	bodyReader, _, err := getBodyReaderAndContentLength(r.Body)
	if err != nil {
		return nil, err
	}
	// Could assert contentLength == r.ContentLength
	return &Request{bodyReader, r}, nil
}

// NewRequest creates a new wrapped request.
func NewRequest(method, url string, rawBody interface{}) (*Request, error) {
	bodyReader, contentLength, err := getBodyReaderAndContentLength(rawBody)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	httpReq.ContentLength = contentLength
	if bodyReader != nil {
		httpReq.Header.Add("Content-Type", fmt.Sprintf("%s; charset=utf-8", jsonContentType))
	}
	httpReq.Header.Add("Accept", jsonContentType)

	return &Request{bodyReader, httpReq}, nil
}
