package middleware

import (
	"context"
	"net/http"

	"MyDashBoard/internal/auth"
)

type contextKey string

const authKey contextKey = "authenticated"

func Auth(sessions *auth.SessionManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, ok := sessions.GetSession(r)
			ctx := context.WithValue(r.Context(), authKey, ok)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func IsAuthenticated(r *http.Request) bool {
	val, _ := r.Context().Value(authKey).(bool)
	return val
}

func AuthRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !IsAuthenticated(r) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"authentication required"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}
