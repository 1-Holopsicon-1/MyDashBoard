package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

const sessionCookieName = "dashboard_session"

type SessionManager struct {
	secret []byte
	store  *Store
}

func NewSessionManager(secret string, store *Store) (*SessionManager, error) {
	if secret == "" {
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			return nil, fmt.Errorf("generate session secret: %w", err)
		}
		secret = base64.StdEncoding.EncodeToString(b)
		log.Println("warning: SESSION_SECRET not set, generated random secret (sessions will not survive restart)")
	}
	return &SessionManager{secret: []byte(secret), store: store}, nil
}

func (sm *SessionManager) SetSession(w http.ResponseWriter, username string) error {
	expires := time.Now().Add(7 * 24 * time.Hour)

	sessionIDBytes := make([]byte, 16)
	if _, err := rand.Read(sessionIDBytes); err != nil {
		return fmt.Errorf("generate session id: %w", err)
	}
	sessionID := hex.EncodeToString(sessionIDBytes)

	if err := sm.store.CreateSession(sessionID, username, expires.Unix()); err != nil {
		return fmt.Errorf("persist session: %w", err)
	}

	token := sm.sign(sessionID, username, expires.Unix())
	value := sessionID + "." + base64.URLEncoding.EncodeToString([]byte(username)) + "." + fmt.Sprintf("%d", expires.Unix()) + "." + token

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    value,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
	return nil
}

func (sm *SessionManager) GetSession(r *http.Request) (string, bool) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return "", false
	}

	parts := strings.Split(cookie.Value, ".")
	if len(parts) != 4 {
		return "", false
	}

	sessionID, userB64, expStr, token := parts[0], parts[1], parts[2], parts[3]

	usernameBytes, err := base64.URLEncoding.DecodeString(userB64)
	if err != nil {
		return "", false
	}
	username := string(usernameBytes)

	var exp int64
	if _, err := fmt.Sscanf(expStr, "%d", &exp); err != nil {
		return "", false
	}
	if time.Now().Unix() > exp {
		return "", false
	}

	expected := sm.sign(sessionID, username, exp)
	if !hmac.Equal([]byte(token), []byte(expected)) {
		return "", false
	}

	exists, err := sm.store.SessionExists(sessionID)
	if err != nil || !exists {
		return "", false
	}

	return username, true
}

func (sm *SessionManager) ClearSession(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		parts := strings.Split(cookie.Value, ".")
		if len(parts) == 4 {
			_ = sm.store.DeleteSession(parts[0])
		}
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
}

func (sm *SessionManager) sign(sessionID, username string, exp int64) string {
	mac := hmac.New(sha256.New, sm.secret)
	mac.Write([]byte(fmt.Sprintf("%d:%s%d", len(username), username, exp)))
	mac.Write([]byte(sessionID))
	return base64.URLEncoding.EncodeToString(mac.Sum(nil))
}

func (sm *SessionManager) SignTemp(data []byte) string {
	mac := hmac.New(sha256.New, sm.secret)
	mac.Write(data)
	return base64.URLEncoding.EncodeToString(mac.Sum(nil))
}

func (sm *SessionManager) VerifyTemp(data []byte, sig string) bool {
	expected := sm.SignTemp(data)
	return hmac.Equal([]byte(expected), []byte(sig))
}