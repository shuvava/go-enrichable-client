package main

import "time"

// User API user data model
type User struct {
	ID        int    `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// UserObject API model for creating new user
type UserObject struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Job       string    `json:"job"`
	CreatedAt time.Time `json:"createdAt"`
}

// Response API response data model
type Response struct {
	Data []User `json:"data"`
}
