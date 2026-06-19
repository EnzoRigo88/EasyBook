package api

import (
	"net/http"
	"time"

	"github.com/EnzoRigo88/EasyBook/control-plane/internal/api/handlers"
	"github.com/EnzoRigo88/EasyBook/control-plane/internal/api/middleware"
	"github.com/EnzoRigo88/EasyBook/control-plane/internal/config"
	"github.com/EnzoRigo88/EasyBook/control-plane/internal/service"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	openai "github.com/sashabaranov/go-openai"
)

// newLLMClient builds the correct ChatCompleter based on LLM_PROVIDER.
// "mock"   → deterministic scripted responses, no network (default for local dev)
// "ollama" → OpenAI-compatible client pointed at a local Ollama container
// "openai" → real OpenAI API (requires OPENAI_API_KEY)
func newLLMClient(cfg *config.Config) service.ChatCompleter {
	switch cfg.LLMProvider {
	case "mock":
		log.Info().Msg("LLM: using mock (no network calls)")
		return service.NewMockChatCompleter()
	case "ollama":
		log.Info().Str("base_url", cfg.OllamaBaseURL).Msg("LLM: using Ollama")
		clientCfg := openai.DefaultConfig("ollama")
		clientCfg.BaseURL = cfg.OllamaBaseURL
		return openai.NewClientWithConfig(clientCfg)
	default:
		log.Info().Msg("LLM: using OpenAI GPT-4o-mini")
		return openai.NewClient(cfg.OpenAIAPIKey)
	}
}

// NewRouter configura todas las rutas de la aplicación.
// En Go, preferimos funciones de construcción explícitas sobre frameworks mágicos.
func NewRouter(cfg *config.Config, pool *pgxpool.Pool) http.Handler {
	// ── Services ──────────────────────────────────────────────────────────────
	bookingSvc := service.NewBookingService(pool)
	agentSvc   := service.NewAgentService(newLLMClient(cfg), bookingSvc)

	// ── Handlers ──────────────────────────────────────────────────────────────
	webhookHandler := handlers.NewWebhookHandler(agentSvc, bookingSvc)

	// ── Router ────────────────────────────────────────────────────────────────
	r := chi.NewRouter()

	// Middleware global — aplica a todos los endpoints
	r.Use(chimiddleware.RequestID)           // ID único por request para tracing
	r.Use(middleware.Logger)                 // logging estructurado con zerolog
	r.Use(chimiddleware.Recoverer)           // recover de panics — no quiero que un bug derribe el server
	r.Use(chimiddleware.Timeout(45 * time.Second)) // timeout global (Claude puede tardar ~10-15s)

	// Rate limiting global: 200 req/min por IP
	r.Use(httprate.LimitByIP(200, time.Minute))

	// ── Rutas públicas ────────────────────────────────────────────────────────

	// Health check — sin auth, lo usa el load balancer y el monitoring
	r.Get("/health", handlers.HealthCheck)

	// Webhooks de WhatsApp — sin JWT (la auth es via signature validation)
	r.Route("/api/v1/webhooks", func(r chi.Router) {
		// Rate limit más estricto para webhooks: 60 req/min por IP
		r.Use(httprate.LimitByIP(60, time.Minute))

		// Twilio
		r.Post("/whatsapp/twilio", webhookHandler.HandleTwilio)

		// Meta Cloud API
		r.Get("/whatsapp/meta", webhookHandler.HandleMetaVerification(cfg.MetaWAWebhookVerifyToken))
		r.Post("/whatsapp/meta", webhookHandler.HandleMeta)
	})

	// ── Rutas protegidas (JWT requerido) ──────────────────────────────────────
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.RequireAuth(cfg.JWTSecret))
		r.Use(middleware.InjectTenant(pool))   // inyecta club_id en el contexto

		// Clubs
		r.Get("/clubs/{clubId}", handlers.NotImplemented)
		r.Put("/clubs/{clubId}", handlers.NotImplemented)

		// Bookings
		r.Route("/clubs/{clubId}/bookings", func(r chi.Router) {
			r.Get("/", handlers.NotImplemented)          // listar reservas
			r.Post("/", handlers.NotImplemented)         // crear reserva desde dashboard
			r.Get("/{bookingId}", handlers.NotImplemented)
			r.Delete("/{bookingId}", handlers.NotImplemented) // cancelar desde dashboard
		})

		// Courts
		r.Get("/clubs/{clubId}/courts", handlers.NotImplemented)

		// Provisioning (solo admin interno)
		r.Route("/admin", func(r chi.Router) {
			r.Use(middleware.RequireAdminRole)
			r.Post("/clubs", handlers.NotImplemented) // provisionar nuevo club
		})
	})

	return r
}
