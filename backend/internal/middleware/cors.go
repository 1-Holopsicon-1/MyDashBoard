package middleware

import (
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

// CORS возвращает chi-совместимый middleware для CORS.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Recoverer реэкспортирует chi Recoverer для удобства.
var Recoverer = middleware.Recoverer
