package models

// News represents a news item visible in the iOS app.
// Image is a string identifier managed by the client (not an enum on the backend).
type News struct {
    ID       string  `json:"id"`
    Image    string  `json:"image"`
    Title    string  `json:"title"`
    Content  string  `json:"content"`
    Date     string  `json:"date"`
    Category *string `json:"category,omitempty"`
}


