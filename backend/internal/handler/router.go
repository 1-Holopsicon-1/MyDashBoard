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
) chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.CORS)
	r.Use(middleware.Auth(sessions))

	// Public
	r.Get("/health", HandleHealth)
	r.Get("/api/auth/status", authHandler.Status)
	r.Post("/api/auth/register-begin", authHandler.RegisterBegin)
	r.Post("/api/auth/register-finish", authHandler.RegisterFinish)
	r.Post("/api/auth/login-begin", authHandler.LoginBegin)
	r.Post("/api/auth/login-finish", authHandler.LoginFinish)
	r.Post("/api/auth/logout", authHandler.Logout)
	r.Post("/api/auth/add-key-begin", authHandler.AddKeyBegin)
	r.Post("/api/auth/add-key-finish", authHandler.AddKeyFinish)

	// Data endpoints
	r.Get("/api/tailscale/devices", tailscale.GetDevices)
	r.Get("/api/services", services.GetStatus)
	r.Get("/api/containers", containers.GetStatus)
	r.Get("/api/simplex/links", simplex.GetLinks)

	return r
}

func IsAuthenticated(r *http.Request) bool {
	return middleware.IsAuthenticated(r)
}
