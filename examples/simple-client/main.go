package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

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
	c := client.DefaultHTTPClient()
	resp, err := c.Get(url)
	if err != nil {
		fmt.Print(err.Error())
	}
	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Print(err.Error())
	}
	var responseObject Response
	json.Unmarshal(bodyBytes, &responseObject)
	s := prettyPrint(responseObject)
	fmt.Printf("API Response as struct %s\n", s)
	// fmt.Printf("API Response as struct %+v\n", responseObject)
}
