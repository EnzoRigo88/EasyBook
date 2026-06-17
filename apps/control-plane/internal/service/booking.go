package service

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// BookingService maneja toda la lógica de negocio de reservas.
// Separar la lógica del servicio del handler HTTP hace el código testeable.
type BookingService struct {
	pool *pgxpool.Pool
}

func NewBookingService(pool *pgxpool.Pool) *BookingService {
	return &BookingService{pool: pool}
}

// AvailabilityResult es lo que devolvemos cuando Claude pregunta por disponibilidad.
type AvailabilityResult struct {
	Available bool          `json:"available"`
	SlotTime  time.Time     `json:"slot_time,omitempty"`
	CourtID   string        `json:"court_id,omitempty"`
	CourtName string        `json:"court_name,omitempty"`
	Price     float64       `json:"price_per_hour,omitempty"`
	Slots     []SlotOption  `json:"slots,omitempty"` // alternativas si el pedido no está disponible
	Message   string        `json:"message"`
}

type SlotOption struct {
	CourtName string    `json:"court_name"`
	CourtID   string    `json:"court_id"`
	SlotTime  time.Time `json:"slot_time"`
	Price     float64   `json:"price"`
}

// BookingResult es lo que devolvemos después de crear una reserva.
type BookingResult struct {
	ID        string    `json:"id"`
	CourtName string    `json:"court_name"`
	StartsAt  time.Time `json:"starts_at"`
	EndsAt    time.Time `json:"ends_at"`
	UserName  string    `json:"user_name"`
	Status    string    `json:"status"`
}

// CheckAvailability verifica si una cancha está disponible en una fecha/hora.
// Si no está disponible, devuelve slots alternativos cercanos.
func (s *BookingService) CheckAvailability(
	ctx context.Context,
	clubID, date, timeStr, courtName string,
) (*AvailabilityResult, error) {

	log.Debug().
		Str("clubId", clubID).
		Str("date", date).
		Str("time", timeStr).
		Str("court", courtName).
		Msg("verificando disponibilidad")

	// Parsear la fecha y hora
	slotTime, err := parseDateTime(date, timeStr)
	if err != nil {
		return nil, fmt.Errorf("fecha/hora inválida: %w", err)
	}

	// TODO: implementar la query real a Postgres usando sqlc
	// Por ahora, simular disponibilidad para el club piloto
	_ = slotTime

	// Simulación para Fase 1 (reemplazar con query real)
	return &AvailabilityResult{
		Available: true,
		CourtID:   "court-1-uuid",
		CourtName: "Cancha 1",
		Price:     3500,
		Message:   fmt.Sprintf("La %s está disponible el %s a las %s", courtName, date, timeStr),
	}, nil
}

// CreateBooking crea una reserva confirmada en la base de datos.
func (s *BookingService) CreateBooking(
	ctx context.Context,
	clubID, courtID, userPhone, userName, date, timeStr string,
) (*BookingResult, error) {

	startsAt, err := parseDateTime(date, timeStr)
	if err != nil {
		return nil, fmt.Errorf("fecha/hora inválida: %w", err)
	}

	endsAt := startsAt.Add(90 * time.Minute) // duración default: 90 minutos

	log.Info().
		Str("clubId", clubID).
		Str("courtId", courtID).
		Str("userPhone", userPhone).
		Time("startsAt", startsAt).
		Msg("creando reserva")

	// TODO: usar sqlc para insertar en la DB
	// query: INSERT INTO bookings (club_id, court_id, user_phone, ...) VALUES (...) RETURNING *
	_ = endsAt

	// Simulación para Fase 1
	return &BookingResult{
		ID:        "booking-uuid-generado",
		CourtName: "Cancha 1",
		StartsAt:  startsAt,
		EndsAt:    endsAt,
		UserName:  userName,
		Status:    "confirmed",
	}, nil
}

// CancelActiveBooking cancela la próxima reserva activa de un usuario.
func (s *BookingService) CancelActiveBooking(
	ctx context.Context,
	clubID, userPhone string,
) (*BookingResult, error) {

	log.Info().
		Str("clubId", clubID).
		Str("userPhone", userPhone).
		Msg("cancelando reserva activa")

	// TODO: query real:
	// 1. GetActiveBookingByPhone (sqlc query)
	// 2. CancelBooking (sqlc query)
	// 3. Notificar al primero en waitlist si existe

	// Simulación para Fase 1
	return &BookingResult{
		ID:     "booking-cancelada-uuid",
		Status: "cancelled",
	}, nil
}

// GetConversationHistory recupera el historial de conversación de un usuario.
// Guardamos el historial en Postgres para que el agente tenga contexto entre mensajes.
func (s *BookingService) GetConversationHistory(
	ctx context.Context,
	clubID, userPhone string,
) ([]ConversationMessage, error) {

	// TODO: query real a Postgres
	// SELECT role, content FROM conversation_messages
	// WHERE club_id = $1 AND user_phone = $2
	// ORDER BY created_at DESC LIMIT 20

	// Retornar historial vacío por ahora (Fase 1)
	return []ConversationMessage{}, nil
}

// SaveConversationMessage guarda el último intercambio en el historial.
func (s *BookingService) SaveConversationMessage(
	ctx context.Context,
	clubID, userPhone, userMessage, botResponse string,
) error {

	// TODO: query real a Postgres
	// INSERT INTO conversation_messages (club_id, user_phone, role, content)
	// VALUES ($1, $2, 'user', $3), ($1, $2, 'assistant', $4)

	log.Debug().
		Str("clubId", clubID).
		Str("userPhone", userPhone).
		Msg("historial de conversación guardado")

	return nil
}

// parseDateTime convierte strings de fecha y hora a time.Time.
// Maneja el timezone de Argentina (UTC-3).
func parseDateTime(date, timeStr string) (time.Time, error) {
	// Argentina: UTC-3 (sin cambio de horario desde 2008)
	loc, err := time.LoadLocation("America/Argentina/Buenos_Aires")
	if err != nil {
		loc = time.FixedZone("ART", -3*60*60)
	}

	combined := date + " " + timeStr
	t, err := time.ParseInLocation("2006-01-02 15:04", combined, loc)
	if err != nil {
		return time.Time{}, fmt.Errorf("formato de fecha/hora inválido (esperado YYYY-MM-DD HH:MM): %w", err)
	}

	return t, nil
}
