package middleware

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/shuvava/go-enrichable-client/client"
)

type (
	// OAuthConfig is OAuth middleware configuration
	OAuthConfig struct {
		AuthServerURL string // URI of oatuh server
		ClientID      string // application's Client ID
		ClientSecret  string // application's Client Secret
		Scope         string // audience for the token, which is your AP
	}

	// BearerResponse is response from OAuth server
	BearerResponse struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
	}

	// BearerToken is a bearer token model
	BearerToken struct {
		AccessToken         string
		ExpirationTokenTime time.Time
	}

	// OAuthService is oauth related logic implementation
	OAuthService struct {
		client *http.Client
		config OAuthConfig
		lock   sync.RWMutex
		token  *BearerToken
	}
)

// getBearerToken makes http call to oauth server
func getBearerToken(cl *http.Client, c OAuthConfig) (*BearerToken, error) {
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", c.ClientID)
	if c.ClientSecret != "" {
		data.Set("client_secret", c.ClientSecret)
	}
	if c.Scope != "" {
		data.Set("scope", c.Scope)
	}
	encodedData := data.Encode()
	payload := strings.NewReader(encodedData)

	req, err := http.NewRequest("POST", c.AuthServerURL, payload)
	if err != nil {
		return nil, err
	}
	req.Header.Add("content-type", "application/x-www-form-urlencoded")
	res, err := cl.Do(req)
	if err != nil {
		return nil, err
	}
	var tokenObj BearerResponse
	err = client.ReadResponse(res, &tokenObj)
	if err != nil {
		return nil, err
	}
	exptime := time.Now().Add(time.Second * time.Duration(tokenObj.ExpiresIn))
	return &BearerToken{
		AccessToken:         tokenObj.AccessToken,
		ExpirationTokenTime: exptime,
	}, nil
}

// GetToken checks if token is expired the request it in thread safe mode
func (s *OAuthService) GetToken() (string, error) {
	s.lock.RLock()
	token := ""
	if s.token != nil && time.Now().Before(s.token.ExpirationTokenTime) {
		token = s.token.AccessToken
	}
	s.lock.RUnlock()
	if token == "" {
		s.lock.Lock()
		t, err := getBearerToken(s.client, s.config)
		if err != nil {
			return "", err
		}
		s.token = t
		token = t.AccessToken
		s.lock.Unlock()
	}
	return token, nil
}

// AddAuthorizationHeader adds authorization header to http.Request
func (s *OAuthService) AddAuthorizationHeader(request *http.Request) error {
	t, err := s.GetToken()
	if err != nil {
		return err
	}
	request.Header.Add("authorization", fmt.Sprintf("Bearer %s", t))

	return nil
}

// NewOAuthService creates OAuthService instance
func NewOAuthService(c OAuthConfig, cl *http.Client) OAuthService {
	if cl == nil {
		// create enriched http client
		clnt := client.DefaultClient()
		// add retry middleware
		clnt.Use(Retry())
		cl = clnt.Client
	}
	return OAuthService{
		client: cl,
		config: c,
	}
}

// OAuth adds Bearer token authentication to requests
func OAuth(c OAuthConfig) client.MiddlewareFunc {
	return OAuthWithClient(c, nil)
}

// OAuthWithClient adds Bearer token authentication to requests
func OAuthWithClient(c OAuthConfig, cl *http.Client) client.MiddlewareFunc {
	s := NewOAuthService(c, cl)
	return func(c *http.Client, next client.Responder) client.Responder {
		return func(request *http.Request) (*http.Response, error) {
			if err := s.AddAuthorizationHeader(request); err != nil {
				return nil, err
			}
			return next(request)
		}
	}
}
