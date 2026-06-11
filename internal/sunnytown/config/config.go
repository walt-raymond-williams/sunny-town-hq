package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Config contains Sunny Town runtime settings loaded from the environment.
type Config struct {
	Host           string
	Port           string
	JoinSecret     string
	ServiceSecret  string
	HQInternalURL  string
	AllowedOrigins map[string]bool
	MapsDir        string
	NPCDayLength   time.Duration
}

func Load() Config {
	return Config{
		Host:           envOrDefault("SUNNY_TOWN_HOST", "0.0.0.0"),
		Port:           envOrDefault("SUNNY_TOWN_PORT", "18082"),
		JoinSecret:     envOrDefault("SUNNY_TOWN_JOIN_SECRET", "local-dev-secret"),
		ServiceSecret:  envOrDefault("SUNNY_TOWN_SERVICE_SECRET", "local-dev-service-secret"),
		HQInternalURL:  strings.TrimRight(envOrDefault("HQ_INTERNAL_BASE_URL", "http://127.0.0.1:8080"), "/"),
		AllowedOrigins: allowedOrigins(envOrDefault("SUNNY_TOWN_ALLOWED_ORIGINS", "http://localhost:5173,http://localhost:18080,http://127.0.0.1:5173,http://127.0.0.1:18080")),
		MapsDir:        envOrDefault("SUNNY_TOWN_MAPS_DIR", filepath.Join("sunny-town", "maps")),
		NPCDayLength:   envMinutesOrDefault("SUNNY_TOWN_NPC_DAY_LENGTH_MINUTES", 24),
	}
}

func allowedOrigins(value string) map[string]bool {
	origins := map[string]bool{}
	for _, origin := range strings.Split(value, ",") {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			origins[origin] = true
		}
	}
	return origins
}

func envOrDefault(name string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}

func envMinutesOrDefault(name string, fallback int) time.Duration {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return time.Duration(fallback) * time.Minute
	}
	minutes, err := strconv.Atoi(value)
	if err != nil || minutes <= 0 {
		return time.Duration(fallback) * time.Minute
	}
	return time.Duration(minutes) * time.Minute
}
