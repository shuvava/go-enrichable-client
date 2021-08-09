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
