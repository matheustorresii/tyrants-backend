package main

import (
    "log"
    "net/http"

    "github.com/matheustorresii/tyrants-back/internal/db"
    newshandler "github.com/matheustorresii/tyrants-back/internal/news"
    userhandler "github.com/matheustorresii/tyrants-back/internal/user"
)

func main() {
    // Initialize persistent SQLite DB (file: tyrants.db in project root)
    storage, err := db.NewSQLiteDB("file:tyrants.db?cache=shared&mode=rwc&_journal=WAL")
    if err != nil {
        log.Fatalf("db init error: %v", err)
    }

    // Wire HTTP handlers
    h := userhandler.NewHandler(storage)
    nh := newshandler.NewHandler(storage)

    mux := http.NewServeMux()
    mux.HandleFunc("/users", h.PostUsers)
    mux.HandleFunc("/login", h.PostLogin)
    mux.HandleFunc("/news", nh.NewsCollection)
    mux.HandleFunc("/news/", nh.NewsItem)

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


