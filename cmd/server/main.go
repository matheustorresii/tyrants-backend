package main

import (
    "log"
    "net/http"

    "github.com/matheustorresii/tyrants-back/internal/db"
    userhandler "github.com/matheustorresii/tyrants-back/internal/user"
)

func main() {
    // Initialize in-memory DB
    storage := db.NewMockDB()

    // Wire HTTP handler
    h := userhandler.NewHandler(storage)

    mux := http.NewServeMux()
    mux.HandleFunc("/users", h.PostUsers)
    mux.HandleFunc("/login", h.PostLogin)

    addr := "localhost:8080"
    log.Printf("Tyrants server listening on http://%s", addr)
    if err := http.ListenAndServe(addr, loggingMiddleware(mux)); err != nil {
        log.Fatalf("server error: %v", err)
    }
}

// loggingMiddleware is a simple request logger.
func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        log.Printf("%s %s", r.Method, r.URL.Path)
        next.ServeHTTP(w, r)
    })
}


