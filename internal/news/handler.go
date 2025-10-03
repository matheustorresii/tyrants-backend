package news

import (
    "encoding/json"
    "errors"
    "net/http"
    "strings"

    "github.com/matheustorresii/tyrants-back/internal/db"
    "github.com/matheustorresii/tyrants-back/internal/models"
)

// Service defines the behaviors the handler requires from the persistence layer.
type Service interface {
    CreateNews(n models.News) error
    GetNews(id string) (models.News, error)
    ListNews() ([]models.News, error)
    UpdateNews(id string, n models.News) (models.News, error)
    DeleteNews(id string) error
}

// Handler provides HTTP handlers for news flows.
type Handler struct {
    svc Service
}

// NewHandler creates a new news Handler.
func NewHandler(svc Service) *Handler {
    return &Handler{svc: svc}
}

type createNewsRequest struct {
    ID       string  `json:"id"`
    Image    string  `json:"image"`
    Title    string  `json:"title"`
    Content  string  `json:"content"`
    Date     string  `json:"date"`
    Category *string `json:"category,omitempty"`
}

type updateNewsRequest struct {
    Image    string  `json:"image"`
    Title    string  `json:"title"`
    Content  string  `json:"content"`
    Date     string  `json:"date"`
    Category *string `json:"category,omitempty"`
}

// NewsCollection handles /news for GET (list) and POST (create)
func (h *Handler) NewsCollection(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case http.MethodGet:
        items, err := h.svc.ListNews()
        if err != nil {
            http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
            return
        }
        w.Header().Set("Content-Type", "application/json")
        _ = json.NewEncoder(w).Encode(items)
        return

    case http.MethodPost:
        var req createNewsRequest
        dec := json.NewDecoder(r.Body)
        dec.DisallowUnknownFields()
        if err := dec.Decode(&req); err != nil {
            http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
            return
        }
        if req.ID == "" || req.Image == "" || req.Title == "" || req.Content == "" || req.Date == "" {
            http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
            return
        }
        item := models.News{
            ID:       req.ID,
            Image:    req.Image,
            Title:    req.Title,
            Content:  req.Content,
            Date:     req.Date,
            Category: req.Category,
        }
        if err := h.svc.CreateNews(item); err != nil {
            if errors.Is(err, db.ErrNewsExists) {
                http.Error(w, http.StatusText(http.StatusConflict), http.StatusConflict)
                return
            }
            http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
            return
        }
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusCreated)
        _ = json.NewEncoder(w).Encode(item)
        return
    default:
        http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
        return
    }
}

// NewsItem handles /news/{id} for GET, PUT, DELETE
func (h *Handler) NewsItem(w http.ResponseWriter, r *http.Request) {
    if !strings.HasPrefix(r.URL.Path, "/news/") {
        http.NotFound(w, r)
        return
    }
    id := strings.TrimPrefix(r.URL.Path, "/news/")
    if id == "" || strings.Contains(id, "/") {
        http.NotFound(w, r)
        return
    }

    switch r.Method {
    case http.MethodGet:
        item, err := h.svc.GetNews(id)
        if err != nil {
            if errors.Is(err, db.ErrNewsNotFound) {
                http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
                return
            }
            http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
            return
        }
        w.Header().Set("Content-Type", "application/json")
        _ = json.NewEncoder(w).Encode(item)
        return

    case http.MethodPut:
        var req updateNewsRequest
        dec := json.NewDecoder(r.Body)
        dec.DisallowUnknownFields()
        if err := dec.Decode(&req); err != nil {
            http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
            return
        }
        if req.Image == "" || req.Title == "" || req.Content == "" || req.Date == "" {
            http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
            return
        }
        update := models.News{
            ID:       id,
            Image:    req.Image,
            Title:    req.Title,
            Content:  req.Content,
            Date:     req.Date,
            Category: req.Category,
        }
        item, err := h.svc.UpdateNews(id, update)
        if err != nil {
            if errors.Is(err, db.ErrNewsNotFound) {
                http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
                return
            }
            http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
            return
        }
        w.Header().Set("Content-Type", "application/json")
        _ = json.NewEncoder(w).Encode(item)
        return

    case http.MethodDelete:
        if err := h.svc.DeleteNews(id); err != nil {
            if errors.Is(err, db.ErrNewsNotFound) {
                http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
                return
            }
            http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
            return
        }
        w.WriteHeader(http.StatusNoContent)
        return

    default:
        http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
        return
    }
}


