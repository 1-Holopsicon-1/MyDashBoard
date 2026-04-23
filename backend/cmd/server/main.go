package main

import (
	"log"
	"net/http"
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
	defer store.Close()

	sessions := auth.NewSessionManager(cfg.SessionSecret)

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
		"vaultwarden": cfg.VaultwardenURL,
	})
	containerSvc := service.NewContainer(dockerClient, cfg.ContainerFilters)
	simplexSvc := service.NewSimplex(dockerClient, []string{"simplex"})

	// Handlers
	authHandler := handler.NewAuth(wn, store, sessions)
	tailscaleHandler := handler.NewTailscale(tailscaleSvc)
	servicesHandler := handler.NewServices(healthSvc)
	containersHandler := handler.NewContainers(containerSvc)
	simplexHandler := handler.NewSimplex(simplexSvc)

	router := handler.NewRouter(tailscaleHandler, servicesHandler, containersHandler, simplexHandler, authHandler, sessions)

	log.Printf("Starting server on %s", cfg.ListenAddr)
	if err := http.ListenAndServe(cfg.ListenAddr, router); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
