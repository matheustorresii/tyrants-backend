package db

import (
    "context"
    "database/sql"
    "errors"
    "fmt"
    "strings"

    _ "modernc.org/sqlite"

    "github.com/matheustorresii/tyrants-back/internal/models"
)

// SQLiteDB is a persistent database backed by SQLite.
type SQLiteDB struct {
    db *sql.DB
}

// NewSQLiteDB opens (or creates) an SQLite database at the given DSN and runs migrations.
// Example DSN: "file:tyrants.db?cache=shared&mode=rwc&_journal=WAL"
func NewSQLiteDB(dsn string) (*SQLiteDB, error) {
    sqldb, err := sql.Open("sqlite", dsn)
    if err != nil {
        return nil, fmt.Errorf("open sqlite: %w", err)
    }
    if err := sqldb.Ping(); err != nil {
        return nil, fmt.Errorf("ping sqlite: %w", err)
    }

    s := &SQLiteDB{db: sqldb}
    if err := s.migrate(context.Background()); err != nil {
        return nil, err
    }
    return s, nil
}

func (s *SQLiteDB) migrate(ctx context.Context) error {
    stmts := []string{
        `CREATE TABLE IF NOT EXISTS users (
            id TEXT PRIMARY KEY,
            name TEXT NOT NULL
        );`,
        `CREATE TABLE IF NOT EXISTS news (
            id TEXT PRIMARY KEY,
            image TEXT NOT NULL,
            title TEXT NOT NULL,
            content TEXT NOT NULL,
            date TEXT NOT NULL,
            category TEXT NULL
        );`,
        `CREATE INDEX IF NOT EXISTS idx_news_date ON news(date);`,
    }
    for _, stmt := range stmts {
        if _, err := s.db.ExecContext(ctx, stmt); err != nil {
            return fmt.Errorf("migrate: %w", err)
        }
    }
    return nil
}

// Users

func (s *SQLiteDB) CreateUser(user models.User) error {
    if user.ID == "" {
        return errors.New("user id cannot be empty")
    }
    _, err := s.db.Exec(`INSERT INTO users(id, name) VALUES(?, ?)`, user.ID, user.Name)
    if err != nil {
        if isUniqueConstraintError(err) {
            return ErrUserExists
        }
        return err
    }
    return nil
}

func (s *SQLiteDB) GetUser(id string) (models.User, error) {
    row := s.db.QueryRow(`SELECT id, name FROM users WHERE id = ?`, id)
    var u models.User
    if err := row.Scan(&u.ID, &u.Name); err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return models.User{}, ErrUserNotFound
        }
        return models.User{}, err
    }
    return u, nil
}

// News

func (s *SQLiteDB) CreateNews(n models.News) error {
    if n.ID == "" {
        return errors.New("news id cannot be empty")
    }
    _, err := s.db.Exec(`INSERT INTO news(id, image, title, content, date, category) VALUES(?, ?, ?, ?, ?, ?)`,
        n.ID, n.Image, n.Title, n.Content, n.Date, n.Category,
    )
    if err != nil {
        if isUniqueConstraintError(err) {
            return ErrNewsExists
        }
        return err
    }
    return nil
}

func (s *SQLiteDB) GetNews(id string) (models.News, error) {
    row := s.db.QueryRow(`SELECT id, image, title, content, date, category FROM news WHERE id = ?`, id)
    var out models.News
    var category sql.NullString
    if err := row.Scan(&out.ID, &out.Image, &out.Title, &out.Content, &out.Date, &category); err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return models.News{}, ErrNewsNotFound
        }
        return models.News{}, err
    }
    if category.Valid {
        out.Category = &category.String
    }
    return out, nil
}

func (s *SQLiteDB) ListNews() ([]models.News, error) {
    rows, err := s.db.Query(`SELECT id, image, title, content, date, category FROM news ORDER BY date DESC, id ASC`)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    var list []models.News
    for rows.Next() {
        var n models.News
        var category sql.NullString
        if err := rows.Scan(&n.ID, &n.Image, &n.Title, &n.Content, &n.Date, &category); err != nil {
            return nil, err
        }
        if category.Valid {
            n.Category = &category.String
        }
        list = append(list, n)
    }
    return list, rows.Err()
}

func (s *SQLiteDB) UpdateNews(id string, n models.News) (models.News, error) {
    res, err := s.db.Exec(`UPDATE news SET image = ?, title = ?, content = ?, date = ?, category = ? WHERE id = ?`,
        n.Image, n.Title, n.Content, n.Date, n.Category, id,
    )
    if err != nil {
        return models.News{}, err
    }
    affected, _ := res.RowsAffected()
    if affected == 0 {
        return models.News{}, ErrNewsNotFound
    }
    return s.GetNews(id)
}

func (s *SQLiteDB) DeleteNews(id string) error {
    res, err := s.db.Exec(`DELETE FROM news WHERE id = ?`, id)
    if err != nil {
        return err
    }
    affected, _ := res.RowsAffected()
    if affected == 0 {
        return ErrNewsNotFound
    }
    return nil
}

func isUniqueConstraintError(err error) bool {
    if err == nil {
        return false
    }
    // Driver-agnostic best effort check
    msg := strings.ToLower(err.Error())
    return strings.Contains(msg, "unique") || strings.Contains(msg, "constraint failed")
}


