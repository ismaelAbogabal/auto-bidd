package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/ismaelfi/auto-bidd/internal/config"
	"github.com/ismaelfi/auto-bidd/internal/database"
	"github.com/ismaelfi/auto-bidd/internal/router"
	"github.com/ismaelfi/auto-bidd/internal/services"
	"github.com/ismaelfi/auto-bidd/internal/views"
)

func main() {
	migrate := flag.Bool("migrate", false, "Run database migrations")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if *migrate {
		if err := database.Migrate(db); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
		log.Println("Migrations completed successfully")
		return
	}

	if cfg.IsDev() {
		if err := database.Migrate(db); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
	}

	renderer := views.NewRenderer("templates", cfg.IsDev())

	// Create LLM provider based on config
	var provider services.LLMProvider
	switch cfg.LLMProvider {
	case "openai":
		baseURL := cfg.LLMBaseURL
		if baseURL == "" {
			baseURL = "https://api.openai.com/v1/chat/completions"
		}
		provider = services.NewOpenAIProvider(cfg.LLMAPIKey, cfg.LLMModel, baseURL)
		log.Printf("LLM provider: openai-compatible (model: %s, url: %s)", cfg.LLMModel, baseURL)
	default: // "anthropic"
		provider = services.NewAnthropicProvider(cfg.LLMAPIKey, cfg.LLMModel, cfg.LLMBaseURL)
		log.Printf("LLM provider: anthropic (model: %s)", cfg.LLMModel)
	}

	aiService := services.NewAIService(provider)

	r := router.New(db, renderer, aiService)

	log.Printf("Server starting on :%s (env: %s)", cfg.Port, cfg.Env)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
