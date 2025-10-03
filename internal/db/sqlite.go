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
            name TEXT NOT NULL,
            tyrant_id TEXT NULL,
            xp INTEGER NOT NULL DEFAULT 0
        );`,
        `CREATE TABLE IF NOT EXISTS user_items (
            user_id TEXT NOT NULL,
            name TEXT NOT NULL,
            asset TEXT NOT NULL,
            PRIMARY KEY (user_id, name),
            FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
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
        // Tyrants tables
        `CREATE TABLE IF NOT EXISTS tyrants (
            id TEXT PRIMARY KEY,
            asset TEXT NOT NULL,
            nickname TEXT NULL,
            hp INTEGER NOT NULL,
            attack INTEGER NOT NULL,
            magic INTEGER NOT NULL,
            defense INTEGER NOT NULL,
            speed INTEGER NOT NULL
        );`,
        `CREATE TABLE IF NOT EXISTS tyrant_evolutions (
            tyrant_id TEXT NOT NULL,
            evolution_id TEXT NOT NULL,
            PRIMARY KEY (tyrant_id, evolution_id),
            FOREIGN KEY (tyrant_id) REFERENCES tyrants(id) ON DELETE CASCADE
        );`,
        `CREATE TABLE IF NOT EXISTS tyrant_attacks (
            tyrant_id TEXT NOT NULL,
            name TEXT NOT NULL,
            power INTEGER NOT NULL,
            pp INTEGER NOT NULL,
            PRIMARY KEY (tyrant_id, name),
            FOREIGN KEY (tyrant_id) REFERENCES tyrants(id) ON DELETE CASCADE
        );`,
        `CREATE TABLE IF NOT EXISTS tyrant_attack_attributes (
            tyrant_id TEXT NOT NULL,
            attack_name TEXT NOT NULL,
            attribute TEXT NOT NULL,
            PRIMARY KEY (tyrant_id, attack_name, attribute),
            FOREIGN KEY (tyrant_id, attack_name) REFERENCES tyrant_attacks(tyrant_id, name) ON DELETE CASCADE
        );`,
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
    _, err := s.db.Exec(`INSERT INTO users(id, name, xp) VALUES(?, ?, 0)`, user.ID, user.Name)
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

// GetUserDetails retrieves a full user view including tyrant and items.
func (s *SQLiteDB) GetUserDetails(id string) (models.UserDetails, error) {
    var out models.UserDetails
    row := s.db.QueryRow(`SELECT id, name, tyrant_id, xp FROM users WHERE id = ?`, id)
    var tyrantID sql.NullString
    if err := row.Scan(&out.ID, &out.Name, &tyrantID, &out.XP); err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return models.UserDetails{}, ErrUserNotFound
        }
        return models.UserDetails{}, err
    }
    if tyrantID.Valid {
        out.TyrantID = &tyrantID.String
    }
    itemsRows, err := s.db.Query(`SELECT name, asset FROM user_items WHERE user_id = ? ORDER BY name ASC`, id)
    if err != nil {
        return models.UserDetails{}, err
    }
    defer itemsRows.Close()
    for itemsRows.Next() {
        var it models.UserItem
        if err := itemsRows.Scan(&it.Name, &it.Asset); err != nil {
            return models.UserDetails{}, err
        }
        out.Items = append(out.Items, it)
    }
    return out, itemsRows.Err()
}

// UpdateUser updates optional tyrant, xp and items.
func (s *SQLiteDB) UpdateUser(id string, upd models.UserUpdate) (models.UserDetails, error) {
    tx, err := s.db.Begin()
    if err != nil {
        return models.UserDetails{}, err
    }
    defer func() { _ = tx.Rollback() }()

    // Ensure exists
    if _, err := tx.Exec(`SELECT 1 FROM users WHERE id = ?`, id); err != nil {
        return models.UserDetails{}, err
    }

    if upd.TyrantID != nil {
        if _, err := tx.Exec(`UPDATE users SET tyrant_id = ? WHERE id = ?`, upd.TyrantID, id); err != nil {
            return models.UserDetails{}, err
        }
    }
    if upd.XP != nil {
        if _, err := tx.Exec(`UPDATE users SET xp = ? WHERE id = ?`, *upd.XP, id); err != nil {
            return models.UserDetails{}, err
        }
    }
    if upd.Items != nil {
        if _, err := tx.Exec(`DELETE FROM user_items WHERE user_id = ?`, id); err != nil {
            return models.UserDetails{}, err
        }
        for _, it := range *upd.Items {
            if _, err := tx.Exec(`INSERT INTO user_items(user_id, name, asset) VALUES(?, ?, ?)`, id, it.Name, it.Asset); err != nil {
                return models.UserDetails{}, err
            }
        }
    }
    if err := tx.Commit(); err != nil {
        return models.UserDetails{}, err
    }
    return s.GetUserDetails(id)
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

// Tyrants

func (s *SQLiteDB) CreateTyrant(t models.Tyrant) error {
    if t.ID == "" {
        return errors.New("tyrant id cannot be empty")
    }
    tx, err := s.db.Begin()
    if err != nil {
        return err
    }
    defer func() {
        _ = tx.Rollback()
    }()

    if _, err := tx.Exec(`INSERT INTO tyrants(id, asset, nickname, hp, attack, magic, defense, speed) VALUES(?, ?, ?, ?, ?, ?, ?, ?)`,
        t.ID, t.Asset, t.Nickname, t.HP, t.Attack, t.Magic, t.Defense, t.Speed,
    ); err != nil {
        if isUniqueConstraintError(err) {
            return ErrTyrantExists
        }
        return err
    }
    for _, evo := range t.Evolutions {
        if _, err := tx.Exec(`INSERT INTO tyrant_evolutions(tyrant_id, evolution_id) VALUES(?, ?)`, t.ID, evo); err != nil {
            return err
        }
    }
    for _, a := range t.Attacks {
        if _, err := tx.Exec(`INSERT INTO tyrant_attacks(tyrant_id, name, power, pp) VALUES(?, ?, ?, ?)`, t.ID, a.Name, a.Power, a.PP); err != nil {
            return err
        }
        for _, attr := range a.Attributes {
            if _, err := tx.Exec(`INSERT INTO tyrant_attack_attributes(tyrant_id, attack_name, attribute) VALUES(?, ?, ?)`, t.ID, a.Name, attr); err != nil {
                return err
            }
        }
    }
    if err := tx.Commit(); err != nil {
        return err
    }
    return nil
}

func (s *SQLiteDB) GetTyrant(id string) (models.Tyrant, error) {
    var t models.Tyrant
    row := s.db.QueryRow(`SELECT id, asset, nickname, hp, attack, magic, defense, speed FROM tyrants WHERE id = ?`, id)
    var nickname sql.NullString
    if err := row.Scan(&t.ID, &t.Asset, &nickname, &t.HP, &t.Attack, &t.Magic, &t.Defense, &t.Speed); err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return models.Tyrant{}, ErrTyrantNotFound
        }
        return models.Tyrant{}, err
    }
    if nickname.Valid {
        t.Nickname = &nickname.String
    }
    // Evolutions
    evoRows, err := s.db.Query(`SELECT evolution_id FROM tyrant_evolutions WHERE tyrant_id = ? ORDER BY evolution_id ASC`, id)
    if err != nil {
        return models.Tyrant{}, err
    }
    defer evoRows.Close()
    for evoRows.Next() {
        var evo string
        if err := evoRows.Scan(&evo); err != nil {
            return models.Tyrant{}, err
        }
        t.Evolutions = append(t.Evolutions, evo)
    }
    // Attacks
    atkRows, err := s.db.Query(`SELECT name, power, pp FROM tyrant_attacks WHERE tyrant_id = ? ORDER BY name ASC`, id)
    if err != nil {
        return models.Tyrant{}, err
    }
    defer atkRows.Close()
    for atkRows.Next() {
        var a models.Attack
        if err := atkRows.Scan(&a.Name, &a.Power, &a.PP); err != nil {
            return models.Tyrant{}, err
        }
        // attributes per attack
        attrRows, err := s.db.Query(`SELECT attribute FROM tyrant_attack_attributes WHERE tyrant_id = ? AND attack_name = ? ORDER BY attribute ASC`, id, a.Name)
        if err != nil {
            return models.Tyrant{}, err
        }
        for attrRows.Next() {
            var attr string
            if err := attrRows.Scan(&attr); err != nil {
                _ = attrRows.Close()
                return models.Tyrant{}, err
            }
            a.Attributes = append(a.Attributes, attr)
        }
        _ = attrRows.Close()
        t.Attacks = append(t.Attacks, a)
    }
    return t, nil
}

func (s *SQLiteDB) ListTyrants() ([]models.Tyrant, error) {
    rows, err := s.db.Query(`SELECT id FROM tyrants ORDER BY id ASC`)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    var result []models.Tyrant
    for rows.Next() {
        var id string
        if err := rows.Scan(&id); err != nil {
            return nil, err
        }
        t, err := s.GetTyrant(id)
        if err != nil {
            return nil, err
        }
        result = append(result, t)
    }
    return result, rows.Err()
}

func (s *SQLiteDB) UpdateTyrant(id string, t models.Tyrant) (models.Tyrant, error) {
    tx, err := s.db.Begin()
    if err != nil {
        return models.Tyrant{}, err
    }
    defer func() { _ = tx.Rollback() }()

    res, err := tx.Exec(`UPDATE tyrants SET asset = ?, nickname = ?, hp = ?, attack = ?, magic = ?, defense = ?, speed = ? WHERE id = ?`,
        t.Asset, t.Nickname, t.HP, t.Attack, t.Magic, t.Defense, t.Speed, id,
    )
    if err != nil {
        return models.Tyrant{}, err
    }
    affected, _ := res.RowsAffected()
    if affected == 0 {
        return models.Tyrant{}, ErrTyrantNotFound
    }
    // Replace evolutions only if provided (nil means keep existing)
    if t.Evolutions != nil {
        if _, err := tx.Exec(`DELETE FROM tyrant_evolutions WHERE tyrant_id = ?`, id); err != nil {
            return models.Tyrant{}, err
        }
        for _, evo := range t.Evolutions {
            if _, err := tx.Exec(`INSERT INTO tyrant_evolutions(tyrant_id, evolution_id) VALUES(?, ?)`, id, evo); err != nil {
                return models.Tyrant{}, err
            }
        }
    }
    // Replace attacks and attributes only if provided
    if t.Attacks != nil {
        if _, err := tx.Exec(`DELETE FROM tyrant_attack_attributes WHERE tyrant_id = ?`, id); err != nil {
            return models.Tyrant{}, err
        }
        if _, err := tx.Exec(`DELETE FROM tyrant_attacks WHERE tyrant_id = ?`, id); err != nil {
            return models.Tyrant{}, err
        }
        for _, a := range t.Attacks {
            if _, err := tx.Exec(`INSERT INTO tyrant_attacks(tyrant_id, name, power, pp) VALUES(?, ?, ?, ?)`, id, a.Name, a.Power, a.PP); err != nil {
                return models.Tyrant{}, err
            }
            for _, attr := range a.Attributes {
                if _, err := tx.Exec(`INSERT INTO tyrant_attack_attributes(tyrant_id, attack_name, attribute) VALUES(?, ?, ?)`, id, a.Name, attr); err != nil {
                    return models.Tyrant{}, err
                }
            }
        }
    }
    if err := tx.Commit(); err != nil {
        return models.Tyrant{}, err
    }
    return s.GetTyrant(id)
}

func (s *SQLiteDB) DeleteTyrant(id string) error {
    res, err := s.db.Exec(`DELETE FROM tyrants WHERE id = ?`, id)
    if err != nil {
        return err
    }
    affected, _ := res.RowsAffected()
    if affected == 0 {
        return ErrTyrantNotFound
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


