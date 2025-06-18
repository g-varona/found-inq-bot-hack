package config

import (
	"os"
	"strconv"
)

// Config holds all configuration for the application
type Config struct {
	// Slack configuration
	SlackBotToken      string
	SlackSigningSecret string
	SlackAppToken      string
	SlackChannelID     string
	TriggerEmoji       string

	// Confluence configuration
	ConfluenceBaseURL  string
	ConfluenceUsername string
	ConfluenceAPIToken string
	ConfluenceSpaceKey string

	// Server configuration
	Port string
	Env  string

	// Database configuration
	DBPath string

	// AI/Search configuration
	SimilarityThreshold float64
	MaxSearchResults    int
	SearchDaysBack      int

	// LiteLLM configuration
	LiteLLMAPIKey  string
	LiteLLMBaseURL string
	LLMModel       string
	LLMTemperature float64
	LLMMaxTokens   int
}

// Load loads configuration from environment variables
func Load() *Config {
	return &Config{
		SlackBotToken:       getEnv("SLACK_BOT_TOKEN", ""),
		SlackSigningSecret:  getEnv("SLACK_SIGNING_SECRET", ""),
		SlackAppToken:       getEnv("SLACK_APP_TOKEN", ""),
		SlackChannelID:      getEnv("SLACK_CHANNEL_ID", ""),
		TriggerEmoji:        getEnv("TRIGGER_EMOJI", "eyes"),
		ConfluenceBaseURL:   getEnv("CONFLUENCE_BASE_URL", ""),
		ConfluenceUsername:  getEnv("CONFLUENCE_USERNAME", ""),
		ConfluenceAPIToken:  getEnv("CONFLUENCE_API_TOKEN", ""),
		ConfluenceSpaceKey:  getEnv("CONFLUENCE_SPACE_KEY", "DOCS"),
		Port:                getEnv("PORT", "8080"),
		Env:                 getEnv("ENV", "development"),
		DBPath:              getEnv("DB_PATH", "./data/inquiries.db"),
		SimilarityThreshold: getEnvFloat("SIMILARITY_THRESHOLD", 0.7),
		MaxSearchResults:    getEnvInt("MAX_SEARCH_RESULTS", 10),
		SearchDaysBack:      getEnvInt("SEARCH_DAYS_BACK", 90),
		LiteLLMAPIKey:       getEnv("LITELLM_API_KEY", ""),
		LiteLLMBaseURL:      getEnv("LITELLM_BASE_URL", "https://litellm.mercari.in"),
		LLMModel:            getEnv("LLM_MODEL", "gpt-4o-mini"),
		LLMTemperature:      getEnvFloat("LLM_TEMPERATURE", 0.3),
		LLMMaxTokens:        getEnvInt("LLM_MAX_TOKENS", 1000),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}
