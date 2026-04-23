package auth

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/go-webauthn/webauthn/webauthn"
	_ "modernc.org/sqlite"
)

type User struct {
	ID         []byte `json:"id"`
	Name       string `json:"name"`
	Credential []byte `json:"credential"`
	SignCount  uint32 `json:"sign_count"`
}

func (u *User) WebAuthnID() []byte                          { return u.ID }
func (u *User) WebAuthnName() string                        { return u.Name }
func (u *User) WebAuthnDisplayName() string                 { return u.Name }
func (u *User) WebAuthnCredentials() []webauthn.Credential {
	if len(u.Credential) == 0 {
		return nil
	}
	var creds []webauthn.Credential
	if err := json.Unmarshal(u.Credential, &creds); err != nil {
		return nil
	}
	return creds
}

func (u *User) AddCredential(cred *webauthn.Credential) {
	var creds []webauthn.Credential
	if len(u.Credential) > 0 {
		json.Unmarshal(u.Credential, &creds)
	}
	creds = append(creds, *cred)
	data, _ := json.Marshal(creds)
	u.Credential = data
	u.SignCount = cred.Authenticator.SignCount
}

// Store — SQLite-backed single-user credential store
type Store struct {
	db *sql.DB
	mu sync.RWMutex
}

func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Enable WAL mode for better concurrent reads
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("setting WAL mode: %w", err)
	}

	// Create table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id         BLOB PRIMARY KEY,
			name       TEXT NOT NULL,
			credential BLOB NOT NULL,
			sign_count INTEGER DEFAULT 0
		)
	`)
	if err != nil {
		return nil, fmt.Errorf("creating table: %w", err)
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) HasUser() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var count int
	row := s.db.QueryRow("SELECT COUNT(*) FROM users")
	row.Scan(&count)
	return count > 0
}

func (s *Store) GetUser() *User {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.getUser()
}

func (s *Store) getUser() *User {
	u := &User{}
	var id []byte
	err := s.db.QueryRow("SELECT id, name, credential, sign_count FROM users LIMIT 1").
		Scan(&id, &u.Name, &u.Credential, &u.SignCount)
	if err != nil {
		return nil
	}
	u.ID = id
	return u
}

func (s *Store) CreateUser(name string, id []byte) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	u := &User{
		ID:   id,
		Name: name,
	}

	_, err := s.db.Exec(
		"INSERT INTO users (id, name, credential, sign_count) VALUES (?, ?, ?, 0)",
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
		"UPDATE users SET credential = ?, sign_count = ? WHERE id = ?",
		user.Credential, user.SignCount, user.ID,
	)
	return err
}
