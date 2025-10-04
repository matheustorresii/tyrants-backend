package models

// Attack represents a Tyrant's attack.
type Attack struct {
    Name       string   `json:"name"`
    Power      int      `json:"power"`
    PP         int      `json:"pp"`
    Attributes []string `json:"attributes"`
}

// Tyrant represents a user's monster.
// ID is also the Tyrant's canonical name.
type Tyrant struct {
    ID         string   `json:"id"`
    Asset      string   `json:"asset"`
    Nickname   *string  `json:"nickname,omitempty"`
    Evolutions []string `json:"evolutions"`
    Attacks    []Attack `json:"attacks"`
    HP         int      `json:"hp"`
    Attack     int      `json:"attack"`
    Defense    int      `json:"defense"`
    Speed      int      `json:"speed"`
}


