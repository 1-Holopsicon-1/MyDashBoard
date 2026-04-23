package handler

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/go-webauthn/webauthn/webauthn"

	"MyDashBoard/internal/auth"
)

type AuthHandler struct {
	webauthn *webauthn.WebAuthn
	store    *auth.Store
	sessions *auth.SessionManager
}

func NewAuth(wn *webauthn.WebAuthn, store *auth.Store, sessions *auth.SessionManager) *AuthHandler {
	return &AuthHandler{
		webauthn: wn,
		store:    store,
		sessions: sessions,
	}
}

// GET /api/auth/status
func (h *AuthHandler) Status(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]bool{
		"registered":    h.store.HasUser(),
		"authenticated": IsAuthenticated(r),
	})
}

// POST /api/auth/register-begin — first-run, no token needed
func (h *AuthHandler) RegisterBegin(w http.ResponseWriter, r *http.Request) {
	if h.store.HasUser() {
		respondError(w, http.StatusConflict, "user already registered")
		return
	}

	id := make([]byte, 64)
	if _, err := rand.Read(id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	user, err := h.store.CreateUser("admin", id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	options, session, err := h.webauthn.BeginRegistration(user)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	setTempSession(w, session)
	respondJSON(w, http.StatusOK, options)
}

// POST /api/auth/register-finish
func (h *AuthHandler) RegisterFinish(w http.ResponseWriter, r *http.Request) {
	user := h.store.GetUser()
	if user == nil {
		respondError(w, http.StatusBadRequest, "no registration in progress")
		return
	}

	if len(user.Credential) > 0 {
		respondError(w, http.StatusConflict, "user already registered")
		return
	}

	session := getTempSession(r)
	if session == nil {
		respondError(w, http.StatusBadRequest, "no registration session")
		return
	}

	credential, err := h.webauthn.FinishRegistration(user, *session, r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	user.AddCredential(credential)
	if err := h.store.UpdateCredential(user); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	clearTempSession(w)
	h.sessions.SetSession(w, user.Name)
	respondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// POST /api/auth/login-begin
func (h *AuthHandler) LoginBegin(w http.ResponseWriter, r *http.Request) {
	user := h.store.GetUser()
	if user == nil {
		respondError(w, http.StatusNotFound, "no user registered")
		return
	}

	options, session, err := h.webauthn.BeginLogin(user)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	setTempSession(w, session)
	respondJSON(w, http.StatusOK, options)
}

// POST /api/auth/login-finish
func (h *AuthHandler) LoginFinish(w http.ResponseWriter, r *http.Request) {
	user := h.store.GetUser()
	if user == nil {
		respondError(w, http.StatusNotFound, "no user registered")
		return
	}

	session := getTempSession(r)
	if session == nil {
		respondError(w, http.StatusBadRequest, "no login session")
		return
	}

	credential, err := h.webauthn.FinishLogin(user, *session, r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	user.SignCount = credential.Authenticator.SignCount
	h.store.UpdateCredential(user)

	clearTempSession(w)
	h.sessions.SetSession(w, user.Name)
	respondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// POST /api/auth/logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	h.sessions.ClearSession(w)
	respondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// POST /api/auth/add-key-begin — requires authentication
func (h *AuthHandler) AddKeyBegin(w http.ResponseWriter, r *http.Request) {
	if !IsAuthenticated(r) {
		respondError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	user := h.store.GetUser()
	if user == nil {
		respondError(w, http.StatusNotFound, "no user registered")
		return
	}

	options, session, err := h.webauthn.BeginRegistration(user)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	setTempSession(w, session)
	respondJSON(w, http.StatusOK, options)
}

// POST /api/auth/add-key-finish — requires authentication
func (h *AuthHandler) AddKeyFinish(w http.ResponseWriter, r *http.Request) {
	if !IsAuthenticated(r) {
		respondError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	user := h.store.GetUser()
	if user == nil {
		respondError(w, http.StatusNotFound, "no user registered")
		return
	}

	session := getTempSession(r)
	if session == nil {
		respondError(w, http.StatusBadRequest, "no registration session")
		return
	}

	credential, err := h.webauthn.FinishRegistration(user, *session, r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	user.AddCredential(credential)
	if err := h.store.UpdateCredential(user); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	clearTempSession(w)
	respondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func setTempSession(w http.ResponseWriter, session *webauthn.SessionData) {
	data, _ := json.Marshal(session)
	encoded := base64.URLEncoding.EncodeToString(data)
	http.SetCookie(w, &http.Cookie{
		Name:     "webauthn_session",
		Value:    encoded,
		Path:     "/",
		MaxAge:   300,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
}

func getTempSession(r *http.Request) *webauthn.SessionData {
	cookie, err := r.Cookie("webauthn_session")
	if err != nil {
		return nil
	}

	data, err := base64.URLEncoding.DecodeString(cookie.Value)
	if err != nil {
		return nil
	}

	var session webauthn.SessionData
	if err := json.Unmarshal(data, &session); err != nil {
		return nil
	}

	return &session
}

func clearTempSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "webauthn_session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
}
