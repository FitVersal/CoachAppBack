package config

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                 string
	DBUrl                string
	JWTSecret            string
	SupabaseURL          string
	SupabaseBucket       string
	SupabaseServiceKey   string
	AppEnv               string
	EnableDocs           bool
	DefaultUserEmail     string
	DefaultUserPassword  string
	DefaultUserRole      string
	DefaultCoachEmail    string
	DefaultCoachPassword string
}

func LoadConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	jwtSecret, exists := os.LookupEnv("JWT_SECRET")
	if !exists || jwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	return &Config{
		Port:                 getEnv("PORT", "8080"),
		DBUrl:                getEnv("DB_URL", ""),
		JWTSecret:            jwtSecret,
		SupabaseURL:          getEnv("SUPABASE_URL", ""),
		SupabaseBucket:       getEnv("SUPABASE_BUCKET", ""),
		SupabaseServiceKey:   getEnv("SUPABASE_SERVICE_KEY", ""),
		AppEnv:               normalizeEnv(getEnv("APP_ENV", "production")),
		EnableDocs:           getEnvBool("ENABLE_API_DOCS", false),
		DefaultUserEmail:     getEnv("DEFAULT_USER_EMAIL", ""),
		DefaultUserPassword:  getEnv("DEFAULT_USER_PASSWORD", ""),
		DefaultUserRole:      getEnv("DEFAULT_USER_ROLE", ""),
		DefaultCoachEmail:    getEnv("DEFAULT_COACH_EMAIL", ""),
		DefaultCoachPassword: getEnv("DEFAULT_COACH_PASSWORD", ""),
	}, nil
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	value, exists := os.LookupEnv(key)
	if !exists || value == "" {
		return fallback
	}

	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func normalizeEnv(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "dev", "develop", "development", "local":
		return "development"
	case "prod", "production":
		return "production"
	case "stage", "staging":
		return "staging"
	case "test", "testing":
		return "test"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func (c *Config) DocsEnabled() bool {
	return c != nil && c.EnableDocs && c.AppEnv == "development"
}
