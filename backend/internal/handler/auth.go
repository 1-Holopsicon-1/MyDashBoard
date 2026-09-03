package handler

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

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
	registered, err := h.store.HasUser()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]bool{
		"registered":    registered,
		"authenticated": IsAuthenticated(r),
	})
}

// POST /api/auth/register-begin — first-run, no token needed
func (h *AuthHandler) RegisterBegin(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	user, err := h.store.CreateUserIfAbsent()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	if len(user.Credential) > 0 {
		respondError(w, http.StatusConflict, errors.New("user already registered"))
		return
	}

	options, session, err := h.webauthn.BeginRegistration(user)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	if err := h.setTempSession(w, session); err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	respondJSON(w, http.StatusOK, options)
}

// POST /api/auth/register-finish
func (h *AuthHandler) RegisterFinish(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	user := h.store.GetUser()
	if user == nil {
		respondError(w, http.StatusBadRequest, errors.New("no registration in progress"))
		return
	}

	if len(user.Credential) > 0 {
		respondError(w, http.StatusConflict, errors.New("user already registered"))
		return
	}

	session := h.getTempSession(r)
	if session == nil {
		respondError(w, http.StatusBadRequest, errors.New("no registration session"))
		return
	}

	credential, err := h.webauthn.FinishRegistration(user, *session, r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}

	if err := h.store.AddCredential(user.ID, credential); err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	h.clearTempSession(w)
	if err := h.sessions.SetSession(w, user.Name); err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// POST /api/auth/login-begin
func (h *AuthHandler) LoginBegin(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	user := h.store.GetUser()
	if user == nil {
		respondError(w, http.StatusNotFound, errors.New("no user registered"))
		return
	}

	options, session, err := h.webauthn.BeginLogin(user)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	if err := h.setTempSession(w, session); err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	respondJSON(w, http.StatusOK, options)
}

// POST /api/auth/login-finish
func (h *AuthHandler) LoginFinish(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	user := h.store.GetUser()
	if user == nil {
		respondError(w, http.StatusNotFound, errors.New("no user registered"))
		return
	}

	session := h.getTempSession(r)
	if session == nil {
		respondError(w, http.StatusBadRequest, errors.New("no login session"))
		return
	}

	_, err := h.webauthn.FinishLogin(user, *session, r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}

	if err := h.store.UpdateCredential(user); err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	h.clearTempSession(w)
	if err := h.sessions.SetSession(w, user.Name); err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// POST /api/auth/logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	h.sessions.ClearSession(w, r)
	respondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// POST /api/auth/add-key-begin — requires authentication
func (h *AuthHandler) AddKeyBegin(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	if !IsAuthenticated(r) {
		respondError(w, http.StatusUnauthorized, errors.New("not authenticated"))
		return
	}

	user := h.store.GetUser()
	if user == nil {
		respondError(w, http.StatusNotFound, errors.New("no user registered"))
		return
	}

	options, session, err := h.webauthn.BeginRegistration(user)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	if err := h.setTempSession(w, session); err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	respondJSON(w, http.StatusOK, options)
}

// POST /api/auth/add-key-finish — requires authentication
func (h *AuthHandler) AddKeyFinish(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	if !IsAuthenticated(r) {
		respondError(w, http.StatusUnauthorized, errors.New("not authenticated"))
		return
	}

	user := h.store.GetUser()
	if user == nil {
		respondError(w, http.StatusNotFound, errors.New("no user registered"))
		return
	}

	session := h.getTempSession(r)
	if session == nil {
		respondError(w, http.StatusBadRequest, errors.New("no registration session"))
		return
	}

	credential, err := h.webauthn.FinishRegistration(user, *session, r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}

	if err := h.store.AddCredential(user.ID, credential); err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	h.clearTempSession(w)
	respondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (h *AuthHandler) setTempSession(w http.ResponseWriter, session *webauthn.SessionData) error {
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("marshal temp session: %w", err)
	}
	encoded := base64.URLEncoding.EncodeToString(data)
	sig := h.sessions.SignTemp(data)
	value := encoded + "." + sig
	http.SetCookie(w, &http.Cookie{
		Name:     "webauthn_session",
		Value:    value,
		Path:     "/",
		MaxAge:   300,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
	return nil
}

func (h *AuthHandler) getTempSession(r *http.Request) *webauthn.SessionData {
	cookie, err := r.Cookie("webauthn_session")
	if err != nil {
		return nil
	}

	parts := strings.SplitN(cookie.Value, ".", 2)
	if len(parts) != 2 {
		return nil
	}

	encoded, sig := parts[0], parts[1]

	data, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return nil
	}

	if !h.sessions.VerifyTemp(data, sig) {
		return nil
	}

	var session webauthn.SessionData
	if err := json.Unmarshal(data, &session); err != nil {
		return nil
	}

	return &session
}

func (h *AuthHandler) clearTempSession(w http.ResponseWriter) {
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
