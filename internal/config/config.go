package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port               string
	DBUrl              string
	JWTSecret          string
	SupabaseURL        string
	SupabaseBucket     string
	SupabaseServiceKey string
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
		Port:               getEnv("PORT", "8080"),
		DBUrl:              getEnv("DB_URL", ""),
		JWTSecret:          jwtSecret,
		SupabaseURL:        getEnv("SUPABASE_URL", ""),
		SupabaseBucket:     getEnv("SUPABASE_BUCKET", ""),
		SupabaseServiceKey: getEnv("SUPABASE_SERVICE_KEY", ""),
	}, nil
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
