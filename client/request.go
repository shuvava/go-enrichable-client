package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

const jsonContentType = "application/json"

// ReaderFunc is the type of function that can be given natively to NewRequest
type ReaderFunc func() (io.Reader, error)

// LenReader is an interface implemented by many in-memory io.Reader's. Used
// for automatically sending the right Content-Length header when possible.
type LenReader interface {
	Len() int
}

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

func getBodyReaderAndContentLength(rawBody interface{}) (ReaderFunc, int64, error) {
	var bodyReader ReaderFunc
	var contentLength int64
	switch body := rawBody.(type) {
	case func() (io.Reader, error):
		bodyReader = body
		tmp, err := body()
		if err != nil {
			return nil, 0, err
		}
		if lr, ok := tmp.(LenReader); ok {
			contentLength = int64(lr.Len())
		}
		if c, ok := tmp.(io.Closer); ok {
			_ = c.Close()
		}

	// If a regular byte slice, we can read it over and over via new
	// readers
	case []byte:
		buf := body
		bodyReader = func() (io.Reader, error) {
			return bytes.NewReader(buf), nil
		}
		contentLength = int64(len(buf))

	// If a bytes.Buffer we can read the underlying byte slice over and
	// over
	case *bytes.Buffer:
		buf := body
		bodyReader = func() (io.Reader, error) {
			return bytes.NewReader(buf.Bytes()), nil
		}
		contentLength = int64(buf.Len())

		// We prioritize *bytes.Reader here because we don't really want to
	// deal with it seeking so want it to match here instead of the
	// io.ReadSeeker case.
	case *bytes.Reader:
		buf, err := ioutil.ReadAll(body)
		if err != nil {
			return nil, 0, err
		}
		bodyReader = func() (io.Reader, error) {
			return bytes.NewReader(buf), nil
		}
		contentLength = int64(len(buf))
	// Compat case
	case io.ReadSeeker:
		raw := body
		bodyReader = func() (io.Reader, error) {
			_, err := raw.Seek(0, 0)
			return ioutil.NopCloser(raw), err
		}
		if lr, ok := raw.(LenReader); ok {
			contentLength = int64(lr.Len())
		}
	// Read all in so we can reset
	case io.Reader:
		buf, err := ioutil.ReadAll(body)
		if err != nil {
			return nil, 0, err
		}
		bodyReader = func() (io.Reader, error) {
			return bytes.NewReader(buf), nil
		}
		contentLength = int64(len(buf))

	// No body provided, nothing to do
	case nil:
		return nil, 0, nil
	// json object
	default:
		buf, err := json.Marshal(rawBody)
		if err != nil {
			return nil, 0, err
		}
		bodyReader = func() (io.Reader, error) {
			return bytes.NewReader(buf), nil
		}
		contentLength = int64(len(buf))
	}

	return bodyReader, contentLength, nil
}

func getBodyReaderAndRequest(method, url string, rawBody interface{}) (*http.Request, ReaderFunc, error) {
	bodyReader, contentLength, err := getBodyReaderAndContentLength(rawBody)
	if err != nil {
		return nil, nil, err
	}

	httpReq, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, nil, err
	}
	httpReq.ContentLength = contentLength
	if bodyReader != nil {
		httpReq.Header.Add("Content-Type", fmt.Sprintf("%s; charset=utf-8", jsonContentType))
	}
	httpReq.Header.Add("Accept", jsonContentType)
	return httpReq, bodyReader, nil
}

// RewindBody rewinds the http body when non-nil.
func RewindBody(r *http.Request, body ReaderFunc) error {
	if body != nil {
		b, err := body()
		if err != nil {
			return err
		}

		if c, ok := b.(io.ReadCloser); ok {
			r.Body = c
		} else {
			r.Body = ioutil.NopCloser(b)
		}
	}
	return nil
}

// RewindBody rewinds the http body when non-nil.
func (r *Request) RewindBody() error {
	return RewindBody(r.Request, r.body)
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

// NewHTTPRequest creates new http.Request with default header
func NewHTTPRequest(method, url string, rawBody interface{}) (*http.Request, error) {
	httpReq, bodyReader, err := getBodyReaderAndRequest(method, url, rawBody)
	if err != nil {
		return nil, err
	}
	if err = RewindBody(httpReq, bodyReader); err != nil {
		return nil, err
	}

	return httpReq, nil
}

// NewRequest creates a new wrapped request.
func NewRequest(method, url string, rawBody interface{}) (*Request, error) {
	httpReq, bodyReader, err := getBodyReaderAndRequest(method, url, rawBody)
	if err != nil {
		return nil, err
	}

	return &Request{bodyReader, httpReq}, nil
}
