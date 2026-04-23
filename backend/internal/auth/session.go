package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const sessionCookieName = "dashboard_session"

type SessionManager struct {
	secret []byte
}

func NewSessionManager(secret string) *SessionManager {
	if secret == "" {
		b := make([]byte, 32)
		rand.Read(b)
		secret = base64.StdEncoding.EncodeToString(b)
	}
	return &SessionManager{secret: []byte(secret)}
}

func (sm *SessionManager) SetSession(w http.ResponseWriter, username string) {
	expires := time.Now().Add(7 * 24 * time.Hour)
	token := sm.sign(username, expires.Unix())
	value := username + "|" + token + "|" + fmt.Sprintf("%d", expires.Unix())

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    value,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
}

func (sm *SessionManager) GetSession(r *http.Request) (string, bool) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return "", false
	}

	parts := strings.SplitN(cookie.Value, "|", 3)
	if len(parts) != 3 {
		return "", false
	}

	username, token, expStr := parts[0], parts[1], parts[2]

	var exp int64
	fmt.Sscanf(expStr, "%d", &exp)
	if time.Now().Unix() > exp {
		return "", false
	}

	expected := sm.sign(username, exp)
	if !hmac.Equal([]byte(token), []byte(expected)) {
		return "", false
	}

	return username, true
}

func (sm *SessionManager) ClearSession(w http.ResponseWriter) {
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

func (sm *SessionManager) sign(username string, exp int64) string {
	mac := hmac.New(sha256.New, sm.secret)
	mac.Write([]byte(username))
	mac.Write([]byte(fmt.Sprintf("%d", exp)))
	return base64.URLEncoding.EncodeToString(mac.Sum(nil))
}
