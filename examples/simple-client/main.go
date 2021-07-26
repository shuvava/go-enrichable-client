package main

import (
	"encoding/json"
	"fmt"

	"github.com/shuvava/go-enrichable-client/client"
)

// User API user data model
type User struct {
	ID        int    `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// Response API response data model
type Response struct {
	Data []User `json:"data"`
}

func prettyPrint(i interface{}) string {
	s, _ := json.MarshalIndent(i, "", "\t")
	return string(s)
}

func main() {
	url := "https://reqres.in/api/users"
	var responseObject Response
	// make GET request and deserialize response body with using default http client
	err := client.Get(url, &responseObject)
	if err != nil {
		fmt.Print(err.Error())
	}
	s := prettyPrint(responseObject)
	fmt.Printf("API Response as struct %s\n", s)
	// fmt.Printf("API Response as struct %+v\n", responseObject)
}
