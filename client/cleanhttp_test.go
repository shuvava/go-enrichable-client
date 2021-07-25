package client

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"testing"
)

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
		client := NewHTTPClient(mock)
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
