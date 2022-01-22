package middleware

import (
	"fmt"
	"net/http"

	"github.com/shuvava/go-enrichable-client/client"
)

/*
UserAgent is a middleware that add the user agent string into request http.Header.
*/

// UserAgentConfig defines the config for UserAgent middleware.
type UserAgentConfig struct {
	// App is the name of the application
	App string
	// Version is the version of the application
	Version string
}

// UserAgent is a middleware that parses the user agent string into http.Header.
func UserAgent(cfg UserAgentConfig) client.MiddlewareFunc {
	return UserAgentWithClient(cfg, nil)
}

// UserAgentWithClient is a middleware that parses the user agent string into http.Header.
func UserAgentWithClient(cfg UserAgentConfig, cl *http.Client) client.MiddlewareFunc {
	if cl == nil {
		// create enriched http client
		clnt := client.DefaultClient()
		cl = clnt.Client
	}
	userAgent := fmt.Sprintf("%s/%s", cfg.App, cfg.Version)
	return func(c *http.Client, next client.Responder) client.Responder {
		return func(request *http.Request) (*http.Response, error) {
			request.Header.Set("User-Agent", userAgent)
			return next(request)
		}
	}
}
