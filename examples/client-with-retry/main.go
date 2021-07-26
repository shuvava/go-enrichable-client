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
	// add retry middleware
	c.Use(middleware.Retry())

	doGet(c)
	doPost(c)
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

func doPost(c *client.Client) {
	var responseObject UserObject
	user := UserObject{
		Name: "morpheus",
		Job:  "leader",
	}
	err := c.Post(url, user, &responseObject)
	if err != nil {
		fmt.Print(err.Error())
	}
	s := prettyPrint(responseObject)
	fmt.Printf("API POST Response as struct %s\n", s)
}
