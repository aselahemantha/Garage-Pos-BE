package config

import (
	"os"
	"time"
)

type Config struct {
	AppEnv            string
	HTTPAddr          string
	DatabaseURL       string
	JWTSecret         string
	AccessTokenTTL    time.Duration
	RefreshTokenTTL   time.Duration
	SeedOwnerUsername string
	SeedOwnerPassword string
}

func Load() *Config {
	return &Config{
		AppEnv:            getEnv("APP_ENV", "development"),
		HTTPAddr:          getEnv("HTTP_ADDR", ":8080"),
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://garage:garagepassword@localhost:5433/garage?sslmode=disable"),
		JWTSecret:         getEnv("JWT_SECRET", "garage-super-secret-jwt-key-minimum-32-chars-long"),
		AccessTokenTTL:    getEnvDuration("ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTokenTTL:   getEnvDuration("REFRESH_TOKEN_TTL", 7*24*time.Hour),
		SeedOwnerUsername: getEnv("SEED_OWNER_USERNAME", "owner"),
		SeedOwnerPassword: getEnv("SEED_OWNER_PASSWORD", "admin123"),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		d, err := time.ParseDuration(val)
		if err == nil {
			return d
		}
	}
	return defaultVal
}
