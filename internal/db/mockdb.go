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

// ErrNewsExists is returned when attempting to create a news item that already exists.
var ErrNewsExists = errors.New("news already exists")

// ErrNewsNotFound is returned when a news item could not be found.
var ErrNewsNotFound = errors.New("news not found")

// MockDB is an in-memory, concurrency-safe database implementation.
type MockDB struct {
    mu    sync.RWMutex
    users map[string]models.User
    news  map[string]models.News
}

// NewMockDB constructs a new MockDB instance.
func NewMockDB() *MockDB {
    return &MockDB{
        users: make(map[string]models.User),
        news:  make(map[string]models.News),
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

// CreateNews stores a new news item if the ID is not already taken.
func (m *MockDB) CreateNews(n models.News) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    if n.ID == "" {
        return errors.New("news id cannot be empty")
    }
    if _, exists := m.news[n.ID]; exists {
        return ErrNewsExists
    }
    m.news[n.ID] = n
    return nil
}

// GetNews retrieves a news item by ID.
func (m *MockDB) GetNews(id string) (models.News, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()

    item, ok := m.news[id]
    if !ok {
        return models.News{}, ErrNewsNotFound
    }
    return item, nil
}

// ListNews retrieves all news items.
func (m *MockDB) ListNews() ([]models.News, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()

    items := make([]models.News, 0, len(m.news))
    for _, n := range m.news {
        items = append(items, n)
    }
    return items, nil
}

// UpdateNews updates an existing news item by ID.
func (m *MockDB) UpdateNews(id string, n models.News) (models.News, error) {
    m.mu.Lock()
    defer m.mu.Unlock()

    existing, ok := m.news[id]
    if !ok {
        return models.News{}, ErrNewsNotFound
    }
    // Keep ID from path authoritative
    existing.ID = id
    existing.Image = n.Image
    existing.Title = n.Title
    existing.Content = n.Content
    existing.Date = n.Date
    existing.Category = n.Category
    m.news[id] = existing
    return existing, nil
}

// DeleteNews removes a news item by ID.
func (m *MockDB) DeleteNews(id string) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    if _, ok := m.news[id]; !ok {
        return ErrNewsNotFound
    }
    delete(m.news, id)
    return nil
}


