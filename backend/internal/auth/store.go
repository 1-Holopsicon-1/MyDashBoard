package auth

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/go-webauthn/webauthn/webauthn"
	_ "modernc.org/sqlite"
)

type User struct {
	ID         []byte `json:"id"`
	Name       string `json:"name"`
	Credential []byte `json:"credential"`
}

func (u *User) WebAuthnID() []byte          { return u.ID }
func (u *User) WebAuthnName() string        { return u.Name }
func (u *User) WebAuthnDisplayName() string { return u.Name }
func (u *User) WebAuthnCredentials() []webauthn.Credential {
	if len(u.Credential) == 0 {
		return nil
	}
	var creds []webauthn.Credential
	if err := json.Unmarshal(u.Credential, &creds); err != nil {
		log.Printf("warning: failed to unmarshal credentials for user %s: %v", u.Name, err)
		return nil
	}
	return creds
}

func (u *User) AddCredential(cred *webauthn.Credential) error {
	var creds []webauthn.Credential
	if len(u.Credential) > 0 {
		if err := json.Unmarshal(u.Credential, &creds); err != nil {
			return fmt.Errorf("unmarshal credentials: %w", err)
		}
	}
	creds = append(creds, *cred)
	data, err := json.Marshal(creds)
	if err != nil {
		return fmt.Errorf("marshal credentials: %w", err)
	}
	u.Credential = data
	return nil
}

func (s *Store) AddCredential(userID []byte, cred *webauthn.Credential) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var credentialBlob []byte
	if err := s.db.QueryRow("SELECT credential FROM users WHERE id = ?", userID).
		Scan(&credentialBlob); err != nil {
		return fmt.Errorf("read credentials: %w", err)
	}

	var creds []webauthn.Credential
	if len(credentialBlob) > 0 {
		if err := json.Unmarshal(credentialBlob, &creds); err != nil {
			return fmt.Errorf("unmarshal credentials: %w", err)
		}
	}
	creds = append(creds, *cred)
	data, err := json.Marshal(creds)
	if err != nil {
		return fmt.Errorf("marshal credentials: %w", err)
	}

	if _, err := s.db.Exec(
		"UPDATE users SET credential = ? WHERE id = ?",
		data, userID,
	); err != nil {
		return fmt.Errorf("update credentials: %w", err)
	}
	return nil
}

type Store struct {
	db *sql.DB
	mu sync.RWMutex
}

func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	db.SetMaxOpenConns(1)

	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		return nil, fmt.Errorf("set busy_timeout: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("set journal_mode: %w", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id         BLOB PRIMARY KEY,
			name       TEXT NOT NULL,
			credential BLOB NOT NULL
		);
		CREATE TABLE IF NOT EXISTS sessions (
			id       TEXT PRIMARY KEY,
			username TEXT NOT NULL,
			exp      INTEGER NOT NULL
		);
	`)
	if err != nil {
		return nil, fmt.Errorf("creating tables: %w", err)
	}

	if err := migrateDropSignCount(db); err != nil {
		return nil, fmt.Errorf("migrate drop sign_count: %w", err)
	}

	return &Store{db: db}, nil
}

func migrateDropSignCount(db *sql.DB) error {
	rows, err := db.Query("PRAGMA table_info(users)")
	if err != nil {
		return fmt.Errorf("query table_info: %w", err)
	}
	defer rows.Close()

	hasSignCount := false
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return fmt.Errorf("scan table_info: %w", err)
		}
		if name == "sign_count" {
			hasSignCount = true
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate table_info: %w", err)
	}

	if !hasSignCount {
		return nil
	}

	stmts := []string{
		`CREATE TABLE users_new (id BLOB PRIMARY KEY, name TEXT NOT NULL, credential BLOB NOT NULL)`,
		`INSERT OR IGNORE INTO users_new (id, name, credential) SELECT id, name, credential FROM users`,
		`DROP TABLE users`,
		`ALTER TABLE users_new RENAME TO users`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("exec migration step %q: %w", s, err)
		}
	}
	return nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) HasUser() (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var count int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		return false, fmt.Errorf("check existing users: %w", err)
	}
	return count > 0, nil
}

func (s *Store) GetUser() *User {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.getUserLocked()
}

func (s *Store) getUserLocked() *User {
	u := &User{}
	var id []byte
	err := s.db.QueryRow("SELECT id, name, credential FROM users LIMIT 1").
		Scan(&id, &u.Name, &u.Credential)
	if err != nil {
		return nil
	}
	u.ID = id
	return u
}

func (s *Store) CreateUserIfAbsent() (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var count int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		return nil, fmt.Errorf("check existing users: %w", err)
	}
	if count > 0 {
		u := s.getUserLocked()
		if u == nil {
			return nil, fmt.Errorf("get existing user: no rows")
		}
		return u, nil
	}

	id := make([]byte, 64)
	if _, err := rand.Read(id); err != nil {
		return nil, fmt.Errorf("generate user id: %w", err)
	}

	u := &User{
		ID:   id,
		Name: "admin",
	}

	_, err := s.db.Exec(
		"INSERT INTO users (id, name, credential) VALUES (?, ?, ?)",
		u.ID, u.Name, []byte{},
	)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return u, nil
}

func (s *Store) CreateUser(name string, id []byte) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	u := &User{
		ID:   id,
		Name: name,
	}

	_, err := s.db.Exec(
		"INSERT INTO users (id, name, credential) VALUES (?, ?, ?)",
		u.ID, u.Name, []byte{},
	)
	if err != nil {
		return nil, fmt.Errorf("inserting user: %w", err)
	}

	return u, nil
}

func (s *Store) UpdateCredential(user *User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(
		"UPDATE users SET credential = ? WHERE id = ?",
		user.Credential, user.ID,
	)
	return err
}

func (s *Store) CreateSession(id, username string, exp int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(
		"INSERT INTO sessions (id, username, exp) VALUES (?, ?, ?)",
		id, username, exp,
	)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

func (s *Store) DeleteSession(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec("DELETE FROM sessions WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func (s *Store) SessionExists(id string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var count int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM sessions WHERE id = ?", id).Scan(&count); err != nil {
		return false, fmt.Errorf("check session: %w", err)
	}
	return count > 0, nil
}
