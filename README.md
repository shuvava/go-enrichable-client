# Enrichable http client

[![GoDoc](https://godoc.org/github.com/shuvava/go-enrichable-client?status.svg)](http://godoc.org/github.com/shuvava/go-enrichable-client)
[![Build Status](https://travis-ci.com/shuvava/go-enrichable-client.svg?branch=master)](https://travis-ci.com/shuvava/go-enrichable-client)
[![Coverage Status](https://coveralls.io/repos/github/shuvava/go-enrichable-client/badge.svg?branch=master)](https://coveralls.io/github/shuvava/go-enrichable-client?branch=master)

the `go-enrichable-client` package is wrapper over the standard `net/http` client library 
allowing to enrich it functionality by using middleware extensions.


## Installation

```shell
go get github.com/shuvava/go-enrichable-client
```

## Usage

**Example of pure http.client usage**

```go
package main

import "github.com/shuvava/go-enrichable-client/client"

func main() {
  var responseObject ResponseObject
  // make GET request and deserialize response body 
  // using default http client with with idle connections 
  // and keepalives disabled.
  err := client.Get(url, &responseObject)
  ...
}
```

## Creating custom middleware

```go
package main

import (
  "net/http"

  "github.com/shuvava/go-enrichable-client/client"
  "github.com/shuvava/go-enrichable-client/middleware"
)

func main() {
  // create enriched client
  c := client.DefaultClient()
  // add custom middleware
  c.Use(
  func(c *http.Client, next client.Responder) client.Responder {
    return func(request *http.Request) (*http.Response, error) {
      // logic before doing request    
      res, err:= next(request)
      // logic after doing request
      return res, err
    }
  })
  ...
}

```

## Middleware 

|  Name           | Description                                   |
|:---------------:|:----------------------------------------------|
| Retry           | add retry functionality                       |
| OAuth           | add bearer authorization token to all request |
| CircuitBreaker  | add Circuit Breaker to all request            |

### Retry middleware

This middleware adds retry functionality with automatic retries and exponential backoff policy. Currently, package supports only json content type

#### Example usage retryable client

```go
package main

import (
	"github.com/shuvava/go-enrichable-client/client"
  "github.com/shuvava/go-enrichable-client/middleware"
)

func main() {
  ...
	// create enriched client
  c := client.DefaultClient()
  // add retry middleware
  c.Use(middleware.Retry())
  // receive reference to http.Client
  err := c.Get(url, &responseObject)
  ...
}
```

### OAuth middleware

With machine-to-machine (M2M) applications, such as CLIs, daemons, or services running on your back-end, the system authenticates and authorizes the app rather than a user. For this scenario, typical authentication schemes like username + password or social logins don't make sense. Instead, M2M apps use the Client Credentials Flow (defined in OAuth 2.0 RFC 6749, section 4.4), in which they pass along their Client ID and Client Secret to authenticate themselves and get a token.

#### Example usage oauth client

```go
package main

import (
  "fmt"

  "github.com/shuvava/go-enrichable-client/client"
  "github.com/shuvava/go-enrichable-client/middleware"
)

func main() {
  ...
	// create enriched client
  c := client.DefaultClient()
  // add oauth middleware
  tenant := "00000000-0000-0000-0000-000000000000"
  uri := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenant)
  oauthConifg := middleware.OAuthConfig{
    AuthServerURL: uri,
    ClientID:      "00000000-0000-0000-0000-000000000000",
    ClientSecret:  "some secret",
    Scope:         "api://00000000-0000-0000-0000-000000000000/.default",
  }
  c.Use(middleware.OAuth(oauthConifg))
  ...
}
```

### CircuitBreaker middleware

The Circuit Breaker pattern can prevent an application repeatedly trying to execute an operation that is likely to fail, allowing it to continue without waiting for the fault to be rectified or wasting CPU cycles while it determines that the fault is long lasting. The Circuit Breaker pattern also enables an application to detect whether the fault has been resolved. If the problem appears to have been rectified, the application can attempt to invoke the operation.

ou can configure `CircuitBreaker` by the struct `Settings`:

```go
type CircuitBreakerSettings struct {
	MaxRequests   uint32
	Interval      time.Duration
	Timeout       time.Duration
	ReadyToTrip   func(counts Counts) bool
	OnStateChange func(name string, from State, to State)
}
```

- `MaxRequests` is the maximum number of requests allowed to pass through
  when the `CircuitBreakerService` is half-open.
  If `MaxRequests` is 0, `CircuitBreakerService` allows only 1 request.
- `Interval` is the cyclic period of the closed state
  for `CircuitBreaker` to clear the internal `Counts`, described later in this section.
  If `Interval` is 0, `CircuitBreakerService` doesn't clear the internal `Counts` during the closed state.
- `Timeout` is the period of the open state,
  after which the state of `CircuitBreakerService` becomes half-open.
  If `Timeout` is 0, the timeout value of `CircuitBreakerService` is set to 60 seconds.
- `ReadyToTrip` is called with a copy of `Counts` whenever a request fails in the closed state.
  If `ReadyToTrip` returns true, `CircuitBreakerService` will be placed into the open state.
  If `ReadyToTrip` is `nil`, default `ReadyToTrip` is used.
  Default `ReadyToTrip` returns true when the number of consecutive failures is more than 5.
- `OnStateChange` is called whenever the state of `CircuitBreakerService` changes.
The struct `CircuitBreakerCounts` holds the numbers of requests and their successes/failures:

```go
type CircuitBreakerCounts struct {
	Requests             uint32
	TotalSuccesses       uint32
	TotalFailures        uint32
	ConsecutiveSuccesses uint32
	ConsecutiveFailures  uint32
}
```

`CircuitBreakerService` clears the internal `CircuitBreakerCounts` either
on the change of the state or at the closed-state intervals.
`CircuitBreakerCounts` ignores the results of the requests sent before clearing.

#### Example usage oauth client

```go
package main

import (
  "fmt"

  "github.com/shuvava/go-enrichable-client/client"
  "github.com/shuvava/go-enrichable-client/middleware"
)

func main() {
  ...
  // create enriched http client
  c := client.DefaultClient()
  // add circuit breaker middleware
  c.Use(middleware.CircuitBreaker(middleware.CircuitBreakerSettings{}))
  ...
}
```

## Links 

* [AWS error handling](https://docs.aws.amazon.com/apigateway/api-reference/handling-errors/)
* [hashicorp http client](https://github.com/hashicorp/go-retryablehttp.git)
* [Circuit Breaker pattern](https://msdn.microsoft.com/en-us/library/dn589784.aspx)
