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

|  Name  | Description                                   |
|:------:|:----------------------------------------------|
| Retry  | add retry functionality                       |
| OAuth  | add bearer authorization token to all request |

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

## Links 

* [AWS error handling](https://docs.aws.amazon.com/apigateway/api-reference/handling-errors/)
* [hashicorp http client](https://github.com/hashicorp/go-retryablehttp.git)
* [Circuit Breaker pattern](https://msdn.microsoft.com/en-us/library/dn589784.aspx)
