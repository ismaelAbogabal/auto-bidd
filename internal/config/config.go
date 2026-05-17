package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL   string
	SessionSecret string
	Port          string
	Env           string

	// LLM provider settings
	LLMProvider string // "anthropic" or "openai" (openai-compatible: deepseek, ollama, etc.)
	LLMAPIKey   string
	LLMModel    string
	LLMBaseURL  string // optional: override the default API endpoint
}

func Load() (*Config, error) {
	godotenv.Load()

	cfg := &Config{
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://autobidd:autobidd@localhost:5432/autobidd?sslmode=disable"),
		SessionSecret: getEnv("SESSION_SECRET", "default-secret-change-me"),
		Port:          getEnv("PORT", "8080"),
		Env:           getEnv("ENV", "development"),

		LLMProvider: getEnv("LLM_PROVIDER", "anthropic"),
		LLMAPIKey:   getEnvAny([]string{"LLM_API_KEY", "CLAUDE_API_KEY"}, ""),
		LLMModel:    getEnvAny([]string{"LLM_MODEL", "CLAUDE_MODEL"}, ""),
		LLMBaseURL:  getEnv("LLM_BASE_URL", ""),
	}

	// Set default model based on provider if not specified
	if cfg.LLMModel == "" {
		switch cfg.LLMProvider {
		case "anthropic":
			cfg.LLMModel = "claude-sonnet-4-20250514"
		case "openai":
			cfg.LLMModel = "gpt-4o"
		}
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

// getEnvAny tries multiple env var names, returns first non-empty value
func getEnvAny(keys []string, fallback string) string {
	for _, key := range keys {
		if v := os.Getenv(key); v != "" {
			return v
		}
	}
	return fallback
}
