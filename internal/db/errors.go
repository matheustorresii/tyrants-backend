package db

import "errors"

// Shared DB errors used across implementations
var (
    ErrUserExists   = errors.New("user already exists")
    ErrUserNotFound = errors.New("user not found")

    ErrNewsExists   = errors.New("news already exists")
    ErrNewsNotFound = errors.New("news not found")

    ErrTyrantExists   = errors.New("tyrant already exists")
    ErrTyrantNotFound = errors.New("tyrant not found")
)


