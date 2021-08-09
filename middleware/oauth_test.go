package middleware_test

import (
	"net/http"
	"sync"
	"testing"

	"github.com/shuvava/go-enrichable-client/client"
	"github.com/shuvava/go-enrichable-client/middleware"
)

func TestOAuthMiddleware(t *testing.T) {
	t.Run("Should request token if it not exist", func(t *testing.T) {
		var (
			url            = "https://YOUR_DOMAIN/oauth/token"
			wantStatusCode = http.StatusOK
			wantToken      = "123"
			wantBody       = `{"token_type":"Bearer","expires_in":3599,"access_token": "123"}`
		)

		m := createPostMock(url, wantStatusCode, wantBody, -1, 0)
		richClient := client.NewClient(m.mock)
		c := richClient.Client
		svc := middleware.NewOAuthService(middleware.OAuthConfig{
			AuthServerURL: url,
			ClientID:      "1",
			ClientSecret:  "2",
		}, c)

		token, _ := svc.GetToken()
		if token != wantToken {
			t.Errorf("token got '%s', want '%s'", token, wantToken)
		}
		if m.calls != 1 {
			t.Errorf("retry got %d, expected %d", m.calls, 1)
		}
	})
	t.Run("Should return error on bad response", func(t *testing.T) {
		var (
			url            = "https://YOUR_DOMAIN/oauth/token"
			wantStatusCode = http.StatusInternalServerError
			wantToken      = ""
			wantBody       = `error`
			wantCalls      = 1
		)

		m := createPostMock(url, wantStatusCode, wantBody, -1, 0)
		richClient := client.NewClient(m.mock)
		c := richClient.Client
		svc := middleware.NewOAuthService(middleware.OAuthConfig{
			AuthServerURL: url,
			ClientID:      "1",
			ClientSecret:  "2",
		}, c)

		token, err := svc.GetToken()
		if token != wantToken {
			t.Errorf("token got '%s', want '%s'", token, wantToken)
		}
		if err == nil {
			t.Errorf("error should be returned")
		}
		if m.calls != wantCalls {
			t.Errorf("retry got %d, expected %d", m.calls, wantCalls)
		}
	})
	t.Run("Should NOT request token if it exist", func(t *testing.T) {
		var (
			url            = "https://YOUR_DOMAIN/oauth/token"
			wantStatusCode = http.StatusOK
			wantToken      = "123"
			wantBody       = `{"token_type":"Bearer","expires_in":3599,"access_token": "123"}`
			wg             sync.WaitGroup
		)

		m := createPostMock(url, wantStatusCode, wantBody, -1, 0)
		richClient := client.NewClient(m.mock)
		c := richClient.Client
		svc := middleware.NewOAuthService(middleware.OAuthConfig{
			AuthServerURL: url,
			ClientID:      "1",
			ClientSecret:  "2",
		}, c)
		start := make(chan struct{})
		token := ""
		wg.Add(10)
		for i := 0; i < 10; i++ {
			go func() {
				<-start
				defer wg.Done()
				token, _ = svc.GetToken()
			}()
		}
		close(start)
		wg.Wait()

		if token != wantToken {
			t.Errorf("token got '%s', want '%s'", token, wantToken)
		}
		if m.calls != 1 {
			t.Errorf("retry got %d, expected %d", m.calls, 1)
		}
	})

	t.Run("Should refresh token once when it expired", func(t *testing.T) {
		var (
			url       = "https://YOUR_DOMAIN/oauth/token"
			wantToken = "456"
			wantCalls = 2
			wg        sync.WaitGroup
		)

		m := createMockMultiResponse(http.MethodPost, url, []responseMock{
			{
				StatusCode: http.StatusOK,
				Body:       `{"token_type":"Bearer","expires_in":0,"access_token": "123"}`,
			},
			{
				StatusCode: http.StatusOK,
				Body:       `{"token_type":"Bearer","expires_in":3599,"access_token": "456"}`,
			},
		})
		richClient := client.NewClient(m.mock)
		c := richClient.Client
		svc := middleware.NewOAuthService(middleware.OAuthConfig{
			AuthServerURL: url,
			ClientID:      "1",
			ClientSecret:  "2",
		}, c)
		token, _ := svc.GetToken()
		start := make(chan struct{})
		wg.Add(10)
		for i := 0; i < 10; i++ {
			go func() {
				<-start
				defer wg.Done()
				token, _ = svc.GetToken()
			}()
		}
		close(start)
		wg.Wait()

		if token != wantToken {
			t.Errorf("token got '%s', want '%s'", token, wantToken)
		}
		if m.calls != wantCalls {
			t.Errorf("retry got %d, expected %d", m.calls, wantCalls)
		}
	})
}
