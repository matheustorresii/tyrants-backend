package tyrant

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
    CreateTyrant(t models.Tyrant) error
    GetTyrant(id string) (models.Tyrant, error)
    ListTyrants() ([]models.Tyrant, error)
    UpdateTyrant(id string, t models.Tyrant) (models.Tyrant, error)
    DeleteTyrant(id string) error
}

// Handler provides HTTP handlers for tyrant flows.
type Handler struct {
    svc Service
}

// NewHandler creates a new tyrant Handler.
func NewHandler(svc Service) *Handler {
    return &Handler{svc: svc}
}

type attackPayload struct {
    Name       string   `json:"name"`
    Power      int      `json:"power"`
    PP         int      `json:"pp"`
    Attributes []string `json:"attributes"`
}

type createTyrantRequest struct {
    ID         string           `json:"id"`
    Asset      string           `json:"asset"`
    Nickname   *string          `json:"nickname,omitempty"`
    Evolutions *[]string        `json:"evolutions,omitempty"`
    Attacks    []attackPayload  `json:"attacks"`
    HP         int              `json:"hp"`
    Attack     int              `json:"attack"`
    Defense    int              `json:"defense"`
    Speed      int              `json:"speed"`
}

type updateTyrantRequest struct {
    Asset      string           `json:"asset"`
    Nickname   *string          `json:"nickname,omitempty"`
    Evolutions *[]string        `json:"evolutions,omitempty"`
    Attacks    *[]attackPayload `json:"attacks,omitempty"`
    HP         int              `json:"hp"`
    Attack     int              `json:"attack"`
    Defense    int              `json:"defense"`
    Speed      int              `json:"speed"`
}

// TyrantsCollection handles /tyrants for GET (list) and POST (create)
func (h *Handler) TyrantsCollection(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case http.MethodGet:
        items, err := h.svc.ListTyrants()
        if err != nil {
            http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
            return
        }
        w.Header().Set("Content-Type", "application/json")
        _ = json.NewEncoder(w).Encode(items)
        return

    case http.MethodPost:
        var req createTyrantRequest
        dec := json.NewDecoder(r.Body)
        dec.DisallowUnknownFields()
        if err := dec.Decode(&req); err != nil {
            http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
            return
        }
        if req.ID == "" || req.Asset == "" || req.HP == 0 && req.Attack == 0 && req.Defense == 0 && req.Speed == 0 {
            // require id, asset, and some stats provided (>= could be zero legitimately but we assume >0 typical; rely on client to pass intended values)
            http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
            return
        }
        // Build model; if evolutions omitted, keep empty slice (optional)
        var evolutions []string
        if req.Evolutions != nil {
            evolutions = *req.Evolutions
        }
        t := models.Tyrant{
            ID:         req.ID,
            Asset:      req.Asset,
            Nickname:   req.Nickname,
            Evolutions: evolutions,
            HP:         req.HP,
            Attack:     req.Attack,
            Defense:    req.Defense,
            Speed:      req.Speed,
        }
        // Map attacks (optional list; empty accepted)
        for _, a := range req.Attacks {
            t.Attacks = append(t.Attacks, models.Attack{
                Name:       a.Name,
                Power:      a.Power,
                PP:         a.PP,
                Attributes: a.Attributes,
            })
        }
        if err := h.svc.CreateTyrant(t); err != nil {
            if errors.Is(err, db.ErrTyrantExists) {
                http.Error(w, http.StatusText(http.StatusConflict), http.StatusConflict)
                return
            }
            http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
            return
        }
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusCreated)
        _ = json.NewEncoder(w).Encode(t)
        return
    default:
        http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
        return
    }
}

// TyrantsItem handles /tyrants/{id} for GET, PUT, DELETE
func (h *Handler) TyrantsItem(w http.ResponseWriter, r *http.Request) {
    if !strings.HasPrefix(r.URL.Path, "/tyrants/") {
        http.NotFound(w, r)
        return
    }
    id := strings.TrimPrefix(r.URL.Path, "/tyrants/")
    if id == "" || strings.Contains(id, "/") {
        http.NotFound(w, r)
        return
    }

    switch r.Method {
    case http.MethodGet:
        item, err := h.svc.GetTyrant(id)
        if err != nil {
            if errors.Is(err, db.ErrTyrantNotFound) {
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
        var req updateTyrantRequest
        dec := json.NewDecoder(r.Body)
        dec.DisallowUnknownFields()
        if err := dec.Decode(&req); err != nil {
            http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
            return
        }
        if req.Asset == "" {
            http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
            return
        }
        // Load existing to support optional fields (keep if omitted)
        current, err := h.svc.GetTyrant(id)
        if err != nil {
            if errors.Is(err, db.ErrTyrantNotFound) {
                http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
                return
            }
            http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
            return
        }
        t := current
        t.Asset = req.Asset
        t.Nickname = req.Nickname
        if req.Nickname == nil {
            t.Nickname = current.Nickname
        }
        // mark optional collections as omitted by default
        t.Evolutions = nil
        t.Attacks = nil
        if req.Evolutions != nil {
            t.Evolutions = *req.Evolutions
        }
        t.HP = req.HP
        t.Attack = req.Attack
        t.Defense = req.Defense
        t.Speed = req.Speed
        if req.Attacks != nil {
            for _, a := range *req.Attacks {
                t.Attacks = append(t.Attacks, models.Attack{
                    Name:       a.Name,
                    Power:      a.Power,
                    PP:         a.PP,
                    Attributes: a.Attributes,
                })
            }
        }
        item, err := h.svc.UpdateTyrant(id, t)
        if err != nil {
            if errors.Is(err, db.ErrTyrantNotFound) {
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
        if err := h.svc.DeleteTyrant(id); err != nil {
            if errors.Is(err, db.ErrTyrantNotFound) {
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


