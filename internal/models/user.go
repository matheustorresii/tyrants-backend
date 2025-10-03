package models

// User represents a Tyrants user.
// ID is a user-chosen identifier (not random).
type User struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}


