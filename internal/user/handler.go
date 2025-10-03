package user

import (
    "encoding/json"
    "errors"
    "net/http"

    "github.com/matheustorresii/tyrants-back/internal/db"
    "github.com/matheustorresii/tyrants-back/internal/models"
)

// Service defines the behaviors the handler requires from the persistence layer.
type Service interface {
    CreateUser(user models.User) error
    GetUser(id string) (models.User, error)
}

// Handler provides HTTP handlers for user flows.
type Handler struct {
    svc Service
}

// NewHandler creates a new Handler.
func NewHandler(svc Service) *Handler {
    return &Handler{svc: svc}
}

// createUserRequest represents the payload for POST /users.
type createUserRequest struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}

// loginRequest represents the payload for POST /login.
type loginRequest struct {
    ID string `json:"id"`
}

// PostUsers handles POST /users
func (h *Handler) PostUsers(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
        return
    }

    var req createUserRequest
    dec := json.NewDecoder(r.Body)
    dec.DisallowUnknownFields()
    if err := dec.Decode(&req); err != nil {
        http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
        return
    }
    if req.ID == "" || req.Name == "" {
        http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
        return
    }

    user := models.User{ID: req.ID, Name: req.Name}
    if err := h.svc.CreateUser(user); err != nil {
        if errors.Is(err, db.ErrUserExists) {
            http.Error(w, http.StatusText(http.StatusConflict), http.StatusConflict)
            return
        }
        http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    _ = json.NewEncoder(w).Encode(user)
}

// PostLogin handles POST /login
func (h *Handler) PostLogin(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
        return
    }

    var req loginRequest
    dec := json.NewDecoder(r.Body)
    dec.DisallowUnknownFields()
    if err := dec.Decode(&req); err != nil {
        http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
        return
    }
    if req.ID == "" {
        http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
        return
    }

    user, err := h.svc.GetUser(req.ID)
    if err != nil {
        if errors.Is(err, db.ErrUserNotFound) {
            http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
            return
        }
        http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(user)
}


