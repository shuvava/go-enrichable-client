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
	// add user-agent middleware
	c.Use(middleware.UserAgent(middleware.UserAgentConfig{App: "client-with-useragent", Version: "1.0.0"}))

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
