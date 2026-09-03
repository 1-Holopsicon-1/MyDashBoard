package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/joho/godotenv"

	"MyDashBoard/internal/auth"
	"MyDashBoard/internal/client"
	"MyDashBoard/internal/config"
	"MyDashBoard/internal/handler"
	"MyDashBoard/internal/service"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment")
	}

	cfg := config.Load()

	httpClient := &http.Client{Timeout: 10 * time.Second}

	// Auth — SQLite store
	store, err := auth.NewStore(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	sessions, err := auth.NewSessionManager(cfg.SessionSecret, store)
	if err != nil {
		log.Fatalf("Failed to create session manager: %v", err)
	}

	if cfg.WebAuthnRPID != "localhost" && cfg.SessionSecret == "" {
		log.Fatal("SESSION_SECRET must be set in production")
	}

	wn, err := webauthn.New(&webauthn.Config{
		RPDisplayName: "holopsicon.ru Dashboard",
		RPID:          cfg.WebAuthnRPID,
		RPOrigins:     []string{cfg.WebAuthnOrigin},
	})
	if err != nil {
		log.Fatalf("Failed to configure WebAuthn: %v", err)
	}

	// Clients
	tailscaleClient := client.NewTailscale(httpClient, cfg.TailscaleAPIKey, cfg.TailscaleTailnet)
	healthChecker := client.NewHealthChecker(httpClient)
	dockerClient := client.NewDocker(cfg.DockerSocket)

	// Services
	tailscaleSvc := service.NewTailscale(tailscaleClient)
	healthSvc := service.NewHealth(healthChecker, map[string]string{
		"vaultwarden": strings.TrimRight(cfg.VaultwardenURL, "/") + "/alive",
	})
	containerSvc := service.NewContainer(dockerClient, cfg.ContainerFilters)
	simplexSvc := service.NewSimplex(dockerClient, []string{"simplex"})

	// Handlers
	authHandler := handler.NewAuth(wn, store, sessions)
	tailscaleHandler := handler.NewTailscale(tailscaleSvc)
	servicesHandler := handler.NewServices(healthSvc)
	containersHandler := handler.NewContainers(containerSvc)
	simplexHandler := handler.NewSimplex(simplexSvc)

	router := handler.NewRouter(tailscaleHandler, servicesHandler, containersHandler, simplexHandler, authHandler, sessions, cfg.WebAuthnOrigin)

	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	log.Printf("Starting server on %s", cfg.ListenAddr)
	<-ctx.Done()
	log.Println("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}

	if err := store.Close(); err != nil {
		log.Printf("store close error: %v", err)
	}
}
