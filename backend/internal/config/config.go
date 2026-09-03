package config

import (
	"os"
	"strings"
)

type Config struct {
	ListenAddr       string
	TailscaleAPIKey  string
	TailscaleTailnet string
	VaultwardenURL   string
	DockerSocket     string
	ContainerFilters []string
	SessionSecret    string
	WebAuthnRPID     string
	WebAuthnOrigin   string
	DBPath           string
}

func Load() *Config {
	return &Config{
		ListenAddr:       getEnv("LISTEN_ADDR", ":8081"),
		TailscaleAPIKey:  os.Getenv("TAILSCALE_API_KEY"),
		TailscaleTailnet: os.Getenv("TAILSCALE_TAILNET"),
		VaultwardenURL:   os.Getenv("VAULTWARDEN_URL"),
		DockerSocket:     getEnv("DOCKER_SOCKET", "/var/run/docker.sock"),
		ContainerFilters: getEnvSlice("CONTAINER_FILTERS", "amnezia,simplex"),
		SessionSecret:    os.Getenv("SESSION_SECRET"),
		WebAuthnRPID:     getEnv("WEBAUTHN_RP_ID", "localhost"),
		WebAuthnOrigin:   getEnv("WEBAUTHN_ORIGIN", "http://localhost:5173"),
		DBPath:           getEnv("DB_PATH", "dashboard.db"),
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

func getEnvSlice(key, fallback string) []string {
	val := getEnv(key, fallback)
	parts := strings.Split(val, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
