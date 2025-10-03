package models

// User represents a Tyrants user.
// ID is a user-chosen identifier (not random).
type User struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}

// UserItem represents an item owned by a user.
type UserItem struct {
    Name  string `json:"name"`
    Asset string `json:"asset"`
}

// UserUpdate contains optional fields that can be updated for a user.
type UserUpdate struct {
    TyrantID *string    `json:"tyrant,omitempty"`
    XP       *int       `json:"xp,omitempty"`
    Items    *[]UserItem `json:"items,omitempty"`
}

// UserDetails represents the full view of a user, for login and reads.
type UserDetails struct {
    ID       string     `json:"id"`
    Name     string     `json:"name"`
    TyrantID *string    `json:"tyrant,omitempty"`
    XP       int        `json:"xp"`
    Items    []UserItem `json:"items"`
}


