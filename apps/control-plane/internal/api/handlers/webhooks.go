package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/EnzoRigo88/EasyBook/control-plane/internal/service"
	"github.com/rs/zerolog/log"
)

// WebhookHandler maneja los webhooks entrantes de WhatsApp.
// Soporta tanto el formato de Twilio como el de Meta Cloud API.
type WebhookHandler struct {
	agentSvc   *service.AgentService
	bookingSvc *service.BookingService
}

func NewWebhookHandler(agentSvc *service.AgentService, bookingSvc *service.BookingService) *WebhookHandler {
	return &WebhookHandler{agentSvc: agentSvc, bookingSvc: bookingSvc}
}

// HandleTwilio procesa mensajes entrantes del formato de Twilio.
// Twilio envía los datos como application/x-www-form-urlencoded.
// POST /api/v1/webhooks/whatsapp/twilio
func (h *WebhookHandler) HandleTwilio(w http.ResponseWriter, r *http.Request) {
	// En Go, errores se manejan explícitamente — no hay try/catch
	if err := r.ParseForm(); err != nil {
		log.Error().Err(err).Msg("error parseando form de Twilio")
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Extraer los campos relevantes del form de Twilio
	from := r.FormValue("From")        // ej: "whatsapp:+5491112345678"
	body := r.FormValue("Body")        // el mensaje del usuario
	profileName := r.FormValue("ProfileName")

	if from == "" || body == "" {
		http.Error(w, "missing required fields", http.StatusBadRequest)
		return
	}

	// Normalizar el número: quitar el prefijo "whatsapp:"
	userPhone := normalizePhone(from)

	log.Info().
		Str("from", userPhone).
		Str("name", profileName).
		Str("message", body).
		Msg("mensaje de WhatsApp recibido (Twilio)")

	// Identificar a qué club pertenece este número
	// TODO: implementar lookup real en DB
	clubID := "club-piloto-uuid" // hardcoded en Fase 1

	// Obtener historial de conversación del usuario
	history, err := h.bookingSvc.GetConversationHistory(r.Context(), clubID, userPhone)
	if err != nil {
		log.Error().Err(err).Msg("error obteniendo historial")
		// No fallamos — empezamos conversación nueva
		history = nil
	}

	// Procesar el mensaje con el agente de IA
	response, err := h.agentSvc.ProcessMessage(
		r.Context(),
		clubID,
		userPhone,
		profileName,
		body,
		history,
	)
	if err != nil {
		log.Error().Err(err).Msg("error procesando mensaje con Claude")
		// Responder con mensaje de error amigable al usuario
		response = "Lo siento, tuve un problema procesando tu mensaje. Por favor intentá de nuevo en un momento."
	}

	// Guardar la conversación actualizada
	if err := h.bookingSvc.SaveConversationMessage(r.Context(), clubID, userPhone, body, response); err != nil {
		log.Error().Err(err).Msg("error guardando mensaje en historial")
		// No fallamos el request por esto — el usuario ya tiene su respuesta
	}

	// Twilio espera una respuesta en formato TwiML o JSON
	// Usando JSON para mayor flexibilidad
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"to":   from,
		"body": response,
	})
}

// HandleMetaVerification maneja el challenge de verificación del webhook de Meta.
// Meta hace un GET con un challenge que debemos devolver para verificar la URL.
// GET /api/v1/webhooks/whatsapp/meta
func (h *WebhookHandler) HandleMetaVerification(verifyToken string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mode      := r.URL.Query().Get("hub.mode")
		token     := r.URL.Query().Get("hub.verify_token")
		challenge := r.URL.Query().Get("hub.challenge")

		if mode == "subscribe" && token == verifyToken {
			log.Info().Msg("webhook de Meta verificado correctamente")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(challenge))
			return
		}

		log.Warn().Str("token", token).Msg("token de verificación de Meta inválido")
		http.Error(w, "Forbidden", http.StatusForbidden)
	}
}

// HandleMeta procesa mensajes entrantes del formato de Meta Cloud API.
// POST /api/v1/webhooks/whatsapp/meta
func (h *WebhookHandler) HandleMeta(w http.ResponseWriter, r *http.Request) {
	// Meta envía JSON — decodificamos la estructura del payload
	var payload MetaWebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Iterar sobre los mensajes del payload (puede venir más de uno)
	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			if change.Field != "messages" {
				continue
			}
			for _, msg := range change.Value.Messages {
				if msg.Type != "text" {
					// TODO: manejar mensajes de voz, imagen, etc.
					continue
				}

				userPhone := msg.From
				body := msg.Text.Body
				log.Info().Str("from", userPhone).Str("message", body).Msg("mensaje Meta WA recibido")

				// TODO: misma lógica que Twilio — extraer a método compartido
			}
		}
	}

	// Meta espera un 200 rápido para confirmar recepción
	w.WriteHeader(http.StatusOK)
}

// HealthCheck devuelve el estado de todos los sistemas del control-plane.
// GET /health
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"service": "easybook-control-plane",
	})
}

// normalizePhone limpia el número de teléfono, quitando prefijos de Twilio.
func normalizePhone(phone string) string {
	if len(phone) > 10 && phone[:10] == "whatsapp:+" {
		return phone[10:] // quita "whatsapp:+"
	}
	if len(phone) > 1 && phone[0] == '+' {
		return phone[1:] // quita el "+"
	}
	return phone
}

// MetaWebhookPayload estructura del payload de Meta Cloud API
type MetaWebhookPayload struct {
	Object string `json:"object"`
	Entry  []struct {
		ID      string `json:"id"`
		Changes []struct {
			Field string `json:"field"`
			Value struct {
				Messages []struct {
					From      string `json:"from"`
					ID        string `json:"id"`
					Timestamp string `json:"timestamp"`
					Type      string `json:"type"`
					Text      struct {
						Body string `json:"body"`
					} `json:"text"`
				} `json:"messages"`
			} `json:"value"`
		} `json:"changes"`
	} `json:"entry"`
}
