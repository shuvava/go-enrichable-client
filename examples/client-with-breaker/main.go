package main

import (
	"fmt"

	"github.com/shuvava/go-enrichable-client/client"
	"github.com/shuvava/go-enrichable-client/middleware"
)

const url = "https://reqres.in/api/users"

func main() {
	// create enriched http client
	c := client.DefaultClient()
	// add circuit breaker middleware
	c.Use(middleware.CircuitBreaker(middleware.CircuitBreakerSettings{}))

	doGet(c)
}

func doGet(c *client.Client) {
	var responseObject Response
	// make GET request and deserialize response body
	err := c.Get(url, &responseObject)
	if err != nil {
		fmt.Print(err.Error())
	}
	s := prettyPrint(responseObject)
	fmt.Printf("API GET Response as struct %s\n", s)
}
