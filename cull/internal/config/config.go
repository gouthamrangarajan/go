package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	IMAPHost   string
	IMAPPort   string
	IMAPUser   string
	IMAPPass   string
	StatePath  string
	UseAI      bool
	AIModelId  string
	AIEndpoint string
	AIAPIKey   string
}

func Load() (Config, error) {
	_ = godotenv.Load()

	cfg := Config{
		IMAPHost:   os.Getenv("YAHOO_IMAP_HOST"),
		IMAPPort:   os.Getenv("YAHOO_IMAP_PORT"),
		IMAPUser:   os.Getenv("YAHOO_IMAP_USER"),
		IMAPPass:   os.Getenv("YAHOO_IMAP_PASS"),
		StatePath:  os.Getenv("STATE_PATH"),
		UseAI:      os.Getenv("CULL_AI") == "true",
		AIModelId:  os.Getenv("OPENROUTER_API_DECISION_MODEL"),
		AIEndpoint: os.Getenv("OPENROUTER_API_URL_FOR_DECISION"),
		AIAPIKey:   os.Getenv("OPENROUTER_API_KEY"),
	}

	if cfg.IMAPHost == "" || cfg.IMAPPort == "" ||
		cfg.IMAPUser == "" || cfg.IMAPPass == "" {
		return Config{}, fmt.Errorf("missing required IMAP environment variable")
	}
	if cfg.UseAI && (cfg.AIModelId == "" || cfg.AIEndpoint == "" || cfg.AIAPIKey == "") {
		return Config{}, fmt.Errorf("missing required AI environment variable")
	}
	return cfg, nil

}
