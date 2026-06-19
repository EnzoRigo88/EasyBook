package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config holds all environment-based configuration.
// In Go, we load env vars once at startup and pass the Config struct around.
// No scattered os.Getenv() calls across the codebase.
type Config struct {
	Port string
	Env  string // "development" | "production"

	DatabaseURL string

	JWTSecret        string
	JWTRefreshSecret string

	// LLMProvider selects the LLM backend: "openai" | "ollama" | "mock"
	LLMProvider  string
	OpenAIAPIKey string
	OllamaBaseURL string

	// WhatsApp — Meta Cloud API (producción)
	MetaWAPhoneNumberID      string
	MetaWAAccessToken        string
	MetaWAWebhookVerifyToken string

	// Twilio (Fase 1 / sandbox)
	TwilioAccountSID string
	TwilioAuthToken  string
	TwilioWANumber   string

	N8NWebhookURL    string
	N8NBasicAuthUser string
	N8NBasicAuthPass string
}

func Load() (*Config, error) {
	// En development, carga el archivo .env si existe.
	// En producción, las variables vienen del sistema operativo (Docker, Hetzner).
	if os.Getenv("ENV") != "production" {
		_ = godotenv.Load() // ignorar error: .env es opcional en CI
	}

	provider := getEnv("LLM_PROVIDER", "openai")

	cfg := &Config{
		Port:                     getEnv("PORT", "3000"),
		Env:                      getEnv("ENV", "development"),
		DatabaseURL:              mustGetEnv("DATABASE_URL"),
		JWTSecret:                mustGetEnv("JWT_SECRET"),
		JWTRefreshSecret:         getEnv("JWT_REFRESH_SECRET", ""),
		LLMProvider:              provider,
		OllamaBaseURL:            getEnv("OLLAMA_BASE_URL", "http://ollama:11434/v1"),
		OpenAIAPIKey:             conditionalMustGetEnv("OPENAI_API_KEY", provider == "openai"),
		MetaWAPhoneNumberID:      getEnv("META_WA_PHONE_NUMBER_ID", ""),
		MetaWAAccessToken:        getEnv("META_WA_ACCESS_TOKEN", ""),
		MetaWAWebhookVerifyToken: getEnv("META_WA_WEBHOOK_VERIFY_TOKEN", "dev-verify-token"),
		TwilioAccountSID:         getEnv("TWILIO_ACCOUNT_SID", ""),
		TwilioAuthToken:          getEnv("TWILIO_AUTH_TOKEN", ""),
		TwilioWANumber:           getEnv("TWILIO_WA_NUMBER", ""),
		N8NWebhookURL:            getEnv("N8N_WEBHOOK_URL", "http://localhost:5678"),
		N8NBasicAuthUser:         getEnv("N8N_BASIC_AUTH_USER", "admin"),
		N8NBasicAuthPass:         getEnv("N8N_BASIC_AUTH_PASSWORD", "admin123"),
	}

	return cfg, nil
}

func (c *Config) IsDevelopment() bool { return c.Env == "development" }
func (c *Config) IsProduction() bool  { return c.Env == "production" }

// getEnv devuelve el valor de la env var o el fallback. Usar para vars opcionales.
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// mustGetEnv panics at startup if a required env var is missing.
func mustGetEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("variable de entorno requerida %q no está configurada", key))
	}
	return v
}

// conditionalMustGetEnv behaves like mustGetEnv when required is true,
// otherwise it returns the value or empty string without panicking.
func conditionalMustGetEnv(key string, required bool) string {
	if required {
		return mustGetEnv(key)
	}
	return os.Getenv(key)
}
