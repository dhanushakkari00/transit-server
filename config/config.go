package config

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application.
type Config struct {
	JWTSecret             string
	JWTAccessTokenExpiry  time.Duration
	JWTRefreshTokenExpiry time.Duration
	DatabaseURL           string
	DBPath                string
	Port                  string
}

// AppConfig is the global configuration instance.
var AppConfig *Config

// Load reads configuration from environment variables and .env file.
func Load() {
	// Load .env file (ignore error if not found — env vars may be set directly)
	_ = godotenv.Load()

	accessExpiry, err := time.ParseDuration(getEnv("JWT_ACCESS_TOKEN_EXPIRY", "15m"))
	if err != nil {
		log.Fatalf("Invalid JWT_ACCESS_TOKEN_EXPIRY: %v", err)
	}

	refreshExpiry, err := time.ParseDuration(getEnv("JWT_REFRESH_TOKEN_EXPIRY", "168h"))
	if err != nil {
		log.Fatalf("Invalid JWT_REFRESH_TOKEN_EXPIRY: %v", err)
	}

	AppConfig = &Config{
		JWTSecret:             getEnv("JWT_SECRET", ""),
		JWTAccessTokenExpiry:  accessExpiry,
		JWTRefreshTokenExpiry: refreshExpiry,
		DatabaseURL:           getEnv("DATABASE_URL", ""),
		DBPath:                getEnv("DB_PATH", "transit.db"),
		Port:                  getEnv("PORT", "8080"),
	}

	if AppConfig.JWTSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}
}

// getEnv returns the value of an environment variable or a fallback default.
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
