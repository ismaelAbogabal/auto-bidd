package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL   string
	ClaudeAPIKey  string
	SessionSecret string
	Port          string
	Env           string
	ClaudeModel   string
}

func Load() (*Config, error) {
	godotenv.Load()

	cfg := &Config{
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://autobidd:autobidd@localhost:5432/autobidd?sslmode=disable"),
		ClaudeAPIKey:  getEnv("CLAUDE_API_KEY", ""),
		SessionSecret: getEnv("SESSION_SECRET", "default-secret-change-me"),
		Port:          getEnv("PORT", "8080"),
		Env:           getEnv("ENV", "development"),
		ClaudeModel:   getEnv("CLAUDE_MODEL", "claude-sonnet-4-20250514"),
	}

	return cfg, nil
}

func (c *Config) IsDev() bool {
	return c.Env == "development"
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
