package db

import (
    "errors"
    "sync"

    "github.com/matheustorresii/tyrants-back/internal/models"
)

// ErrUserExists is returned when attempting to create a user that already exists.
var ErrUserExists = errors.New("user already exists")

// ErrUserNotFound is returned when a user could not be found.
var ErrUserNotFound = errors.New("user not found")

// MockDB is an in-memory, concurrency-safe database implementation.
type MockDB struct {
    mu    sync.RWMutex
    users map[string]models.User
}

// NewMockDB constructs a new MockDB instance.
func NewMockDB() *MockDB {
    return &MockDB{
        users: make(map[string]models.User),
    }
}

// CreateUser stores a new user if the ID is not already taken.
func (m *MockDB) CreateUser(user models.User) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    if user.ID == "" {
        return errors.New("user id cannot be empty")
    }
    if _, exists := m.users[user.ID]; exists {
        return ErrUserExists
    }
    m.users[user.ID] = user
    return nil
}

// GetUser retrieves a user by ID.
func (m *MockDB) GetUser(id string) (models.User, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()

    user, ok := m.users[id]
    if !ok {
        return models.User{}, ErrUserNotFound
    }
    return user, nil
}


