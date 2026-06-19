package service

import (
	"context"
	"encoding/json"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
	"github.com/rs/zerolog/log"
)

// AgentService maneja la conversación con el modelo de IA.
// Usamos GPT-4o-mini (OpenAI) por costo: ~95% más barato que Claude Sonnet
// para este volumen de conversaciones, con soporte maduro de function calling.
// Ver docs/ADR/002-modelo-ia.md para el detalle de la decisión.
type AgentService struct {
	client         ChatCompleter
	bookingService *BookingService
}

func NewAgentService(client ChatCompleter, bookingSvc *BookingService) *AgentService {
	return &AgentService{
		client:         client,
		bookingService: bookingSvc,
	}
}

// ConversationMessage representa un mensaje del historial de conversación.
// Lo guardamos en Postgres para que el bot recuerde el contexto entre mensajes.
type ConversationMessage struct {
	Role    string `json:"role"` // "user" | "assistant"
	Content string `json:"content"`
}

// ProcessMessage es el punto de entrada principal del agente.
// Recibe el mensaje del usuario y el historial, devuelve la respuesta del bot.
func (s *AgentService) ProcessMessage(
	ctx context.Context,
	clubID string,
	userPhone string,
	userName string,
	userMessage string,
	history []ConversationMessage,
) (string, error) {

	messages := s.buildMessages(clubID, history, userMessage)
	tools := s.defineTools()

	log.Debug().
		Str("clubId", clubID).
		Str("userPhone", userPhone).
		Int("historyLen", len(history)).
		Msg("procesando mensaje con GPT-4o-mini")

	// Agentic loop: el modelo puede pedir usar herramientas en secuencia
	// (ej: primero check_availability, después create_booking) antes
	// de devolver la respuesta final en texto.
	for range 5 { // máximo 5 iteraciones para evitar loops infinitos
		resp, err := s.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
			Model:       openai.GPT4oMini,
			Messages:    messages,
			Tools:       tools,
			MaxTokens:   1024,
			Temperature: 0.3, // baja temperatura: queremos consistencia, no creatividad, en un bot de reservas
		})
		if err != nil {
			return "", fmt.Errorf("error llamando a OpenAI API: %w", err)
		}

		if len(resp.Choices) == 0 {
			return "", fmt.Errorf("OpenAI no devolvió ninguna respuesta")
		}

		choice := resp.Choices[0]

		// Si el modelo no pidió usar ninguna herramienta, esa es la respuesta final
		if len(choice.Message.ToolCalls) == 0 {
			return choice.Message.Content, nil
		}

		// El modelo quiere usar una o más herramientas.
		// Agregamos su mensaje (con los tool_calls) al historial...
		messages = append(messages, choice.Message)

		// ...ejecutamos cada herramienta solicitada...
		for _, toolCall := range choice.Message.ToolCalls {
			result, err := s.executeTool(ctx, clubID, userPhone, userName, toolCall)
			if err != nil {
				log.Error().Err(err).Str("tool", toolCall.Function.Name).Msg("error ejecutando herramienta")
				result = fmt.Sprintf(`{"error": "%s"}`, err.Error())
			}

			// ...y agregamos el resultado como mensaje de rol "tool",
			// referenciado por ToolCallID para que el modelo sepa a qué llamada corresponde.
			messages = append(messages, openai.ChatCompletionMessage{
				Role:       openai.ChatMessageRoleTool,
				Content:    result,
				ToolCallID: toolCall.ID,
			})
		}
		// Continúa el loop: el modelo ve los resultados y decide el próximo paso
	}

	return "Lo siento, no pude procesar tu solicitud. Por favor intentá de nuevo.", nil
}

// buildMessages construye el array de mensajes en el formato de OpenAI,
// con el system prompt primero, después el historial, después el mensaje nuevo.
func (s *AgentService) buildMessages(clubID string, history []ConversationMessage, newMessage string) []openai.ChatCompletionMessage {
	messages := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: s.buildSystemPrompt(clubID)},
	}

	for _, h := range history {
		role := openai.ChatMessageRoleUser
		if h.Role == "assistant" {
			role = openai.ChatMessageRoleAssistant
		}
		messages = append(messages, openai.ChatCompletionMessage{Role: role, Content: h.Content})
	}

	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: newMessage,
	})

	return messages
}

// buildSystemPrompt genera el system prompt con el contexto del club.
// En producción esto vendría de la DB (configuración del club).
func (s *AgentService) buildSystemPrompt(clubID string) string {
	return fmt.Sprintf(`Sos el asistente de reservas del club de pádel.
Tu trabajo es ayudar a los clientes a reservar, cancelar y consultar canchas de manera amigable y eficiente.

Comportamiento:
- Respondé siempre en español, de forma amigable y concisa
- Si el cliente quiere reservar, necesitás: cancha (o preguntás cuál prefiere), fecha y hora
- Si hay ambigüedad en la fecha ("mañana", "el viernes"), confirmala antes de reservar
- Ante una cancelación, identificá la reserva y pedí confirmación antes de cancelar
- Si no podés ayudar con algo, escalá a un humano del club

Club ID (para queries internas, no lo menciones al usuario): %s`, clubID)
}

// defineTools declara las herramientas disponibles para el modelo.
// El formato sigue la especificación de "function calling" de OpenAI:
// cada tool tiene un nombre, descripción, y JSON Schema de parámetros.
func (s *AgentService) defineTools() []openai.Tool {
	return []openai.Tool{
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "check_availability",
				Description: "Verifica la disponibilidad de una cancha en una fecha y hora específica",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"court_name": map[string]any{
							"type":        "string",
							"description": "Nombre o número de la cancha (ej: 'Cancha 1', 'Cancha Principal')",
						},
						"date": map[string]any{
							"type":        "string",
							"description": "Fecha en formato YYYY-MM-DD",
						},
						"time": map[string]any{
							"type":        "string",
							"description": "Hora en formato HH:MM (ej: '19:00')",
						},
					},
					"required": []string{"date", "time"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "create_booking",
				Description: "Crea una reserva confirmada para el usuario",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"court_id":  map[string]any{"type": "string"},
						"date":      map[string]any{"type": "string", "description": "YYYY-MM-DD"},
						"time":      map[string]any{"type": "string", "description": "HH:MM"},
						"confirmed": map[string]any{"type": "boolean", "description": "true solo si el usuario confirmó explícitamente"},
					},
					"required": []string{"court_id", "date", "time", "confirmed"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "cancel_booking",
				Description: "Cancela la próxima reserva activa del usuario",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"confirmed": map[string]any{
							"type":        "boolean",
							"description": "true solo si el usuario confirmó que quiere cancelar",
						},
					},
					"required": []string{"confirmed"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "escalate_to_human",
				Description: "Escala la conversación a un humano del club cuando no podés resolver el problema",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"reason":   map[string]any{"type": "string", "description": "Motivo del escalado"},
						"severity": map[string]any{"type": "string", "enum": []string{"low", "medium", "high"}},
					},
					"required": []string{"reason", "severity"},
				},
			},
		},
	}
}

// executeTool ejecuta la herramienta solicitada por el modelo y devuelve
// el resultado como un string JSON (que se reinyecta en la conversación).
func (s *AgentService) executeTool(
	ctx context.Context,
	clubID, userPhone, userName string,
	toolCall openai.ToolCall,
) (string, error) {

	log.Debug().
		Str("tool", toolCall.Function.Name).
		Str("args", toolCall.Function.Arguments).
		Msg("ejecutando herramienta")

	switch toolCall.Function.Name {
	case "check_availability":
		return s.handleCheckAvailability(ctx, clubID, toolCall.Function.Arguments)
	case "create_booking":
		return s.handleCreateBooking(ctx, clubID, userPhone, userName, toolCall.Function.Arguments)
	case "cancel_booking":
		return s.handleCancelBooking(ctx, clubID, userPhone, toolCall.Function.Arguments)
	case "escalate_to_human":
		return s.handleEscalation(ctx, clubID, userPhone, userName, toolCall.Function.Arguments)
	default:
		return fmt.Sprintf(`{"error": "herramienta desconocida: %s"}`, toolCall.Function.Name), nil
	}
}

func (s *AgentService) handleCheckAvailability(ctx context.Context, clubID, args string) (string, error) {
	var params struct {
		CourtName string `json:"court_name"`
		Date      string `json:"date"`
		Time      string `json:"time"`
	}
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", fmt.Errorf("argumentos inválidos: %w", err)
	}

	slots, err := s.bookingService.CheckAvailability(ctx, clubID, params.Date, params.Time, params.CourtName)
	if err != nil {
		return "", err
	}

	result, _ := json.Marshal(slots)
	return string(result), nil
}

func (s *AgentService) handleCreateBooking(ctx context.Context, clubID, userPhone, userName, args string) (string, error) {
	var params struct {
		CourtID   string `json:"court_id"`
		Date      string `json:"date"`
		Time      string `json:"time"`
		Confirmed bool   `json:"confirmed"`
	}
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", fmt.Errorf("argumentos inválidos: %w", err)
	}

	if !params.Confirmed {
		return `{"status": "pending_confirmation", "message": "esperando confirmación del usuario"}`, nil
	}

	booking, err := s.bookingService.CreateBooking(ctx, clubID, params.CourtID, userPhone, userName, params.Date, params.Time)
	if err != nil {
		return "", err
	}

	result, _ := json.Marshal(booking)
	return string(result), nil
}

func (s *AgentService) handleCancelBooking(ctx context.Context, clubID, userPhone, args string) (string, error) {
	var params struct {
		Confirmed bool `json:"confirmed"`
	}
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", fmt.Errorf("argumentos inválidos: %w", err)
	}

	if !params.Confirmed {
		return `{"status": "pending_confirmation", "message": "esperando confirmación de cancelación"}`, nil
	}

	booking, err := s.bookingService.CancelActiveBooking(ctx, clubID, userPhone)
	if err != nil {
		return "", err
	}

	result, _ := json.Marshal(booking)
	return string(result), nil
}

func (s *AgentService) handleEscalation(ctx context.Context, clubID, userPhone, userName, args string) (string, error) {
	var params struct {
		Reason   string `json:"reason"`
		Severity string `json:"severity"`
	}
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", fmt.Errorf("argumentos inválidos: %w", err)
	}

	log.Warn().
		Str("clubId", clubID).
		Str("userPhone", userPhone).
		Str("severity", params.Severity).
		Str("reason", params.Reason).
		Msg("escalando conversación a humano")

	// TODO: enviar notificación al club vía Slack/email
	return `{"status": "escalated", "message": "un agente humano se comunicará pronto"}`, nil
}
