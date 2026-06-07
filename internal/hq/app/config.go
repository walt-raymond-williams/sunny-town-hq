package app

import (
	"os"
	"strings"
)

type Config struct {
	Host                   string
	Port                   string
	DatabaseURL            string
	MigrationsDir          string
	KeycloakIssuer         string
	KeycloakAudience       string
	KeycloakJWKSURL        string
	SunnyTownJoinSecret    string
	SunnyTownServiceSecret string
	SunnyTownWebSocketURL  string
	AIServiceURL           string
	HQToAIServiceSecret    string
	AIToHQServiceSecret    string
	AIGradingEnabled       bool
	AIAutoApplyGrades      bool
	AIPromptVersionGrader  string
}

func LoadConfig() Config {
	return Config{
		Host:                   envOrDefault("HQ_HOST", "0.0.0.0"),
		Port:                   envOrDefault("HQ_PORT", "8080"),
		DatabaseURL:            strings.TrimSpace(os.Getenv("DATABASE_URL")),
		MigrationsDir:          envOrDefault("HQ_MIGRATIONS_DIR", "deploy/postgres/migrations"),
		KeycloakIssuer:         envOrDefault("KEYCLOAK_ISSUER", "http://localhost:18081/realms/hq"),
		KeycloakAudience:       envOrDefault("KEYCLOAK_AUDIENCE", "hq-web"),
		KeycloakJWKSURL:        strings.TrimSpace(os.Getenv("KEYCLOAK_JWKS_URL")),
		SunnyTownJoinSecret:    envOrDefault("SUNNY_TOWN_JOIN_SECRET", "local-dev-secret"),
		SunnyTownServiceSecret: envOrDefault("SUNNY_TOWN_SERVICE_SECRET", "local-dev-service-secret"),
		SunnyTownWebSocketURL:  envOrDefault("SUNNY_TOWN_WS_URL", "ws://127.0.0.1:18082/sunny-town/ws"),
		AIServiceURL:           strings.TrimRight(strings.TrimSpace(os.Getenv("AI_SERVICE_URL")), "/"),
		HQToAIServiceSecret:    strings.TrimSpace(os.Getenv("HQ_TO_AI_SERVICE_SECRET")),
		AIToHQServiceSecret:    envOrDefault("AI_TO_HQ_SERVICE_SECRET", "local-dev-ai-service-secret"),
		AIGradingEnabled:       envBool("AI_GRADING_ENABLED", false),
		AIAutoApplyGrades:      envBool("AI_AUTO_APPLY_GRADES", false),
		AIPromptVersionGrader:  envOrDefault("AI_PROMPT_VERSION_GRADER", "assignment-grader-v1"),
	}
}

func (cfg Config) Address() string {
	return cfg.Host + ":" + cfg.Port
}

func envOrDefault(name string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}

func envBool(name string, fallback bool) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(name)))
	if value == "" {
		return fallback
	}
	return value == "1" || value == "true" || value == "yes" || value == "on"
}
