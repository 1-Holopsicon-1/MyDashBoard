package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"MyDashBoard/internal/auth"
	"MyDashBoard/internal/middleware"
)

func NewRouter(
	tailscale *TailscaleHandler,
	services *ServicesHandler,
	containers *ContainersHandler,
	simplex *SimplexHandler,
	authHandler *AuthHandler,
	sessions *auth.SessionManager,
	allowedOrigin string,
) chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.CORS(allowedOrigin))
	r.Use(middleware.Auth(sessions))

	r.Get("/health", HandleHealth)

	r.Route("/api/auth", func(r chi.Router) {
		r.Get("/status", authHandler.Status)
		r.Post("/register-begin", authHandler.RegisterBegin)
		r.Post("/register-finish", authHandler.RegisterFinish)
		r.Post("/login-begin", authHandler.LoginBegin)
		r.Post("/login-finish", authHandler.LoginFinish)
		r.Post("/logout", authHandler.Logout)
		r.With(middleware.AuthRequired).Post("/add-key-begin", authHandler.AddKeyBegin)
		r.With(middleware.AuthRequired).Post("/add-key-finish", authHandler.AddKeyFinish)
	})

	r.Route("/api", func(r chi.Router) {
		r.Use(middleware.AuthRequired)
		r.Get("/tailscale/devices", tailscale.GetDevices)
		r.Get("/services", services.GetStatus)
		r.Get("/containers", containers.GetStatus)
		r.Get("/simplex/links", simplex.GetLinks)
	})

	return r
}

func IsAuthenticated(r *http.Request) bool {
	return middleware.IsAuthenticated(r)
}
