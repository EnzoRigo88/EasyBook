-- name: GetAvailableSlots :many
-- Retorna los slots disponibles para una cancha en una fecha dada.
-- sqlc genera la función Go correspondiente a partir de este SQL.
SELECT
    ts AS slot_time,
    c.id AS court_id,
    c.name AS court_name,
    p.price_per_hour
FROM
    generate_series(
        $2::timestamptz,                          -- inicio del día
        $2::timestamptz + interval '23 hours',    -- fin del día
        s.slot_duration::interval
    ) AS ts
    JOIN courts c ON c.id = $1
    JOIN schedules s ON s.court_id = c.id AND s.day_of_week = EXTRACT(DOW FROM $2::date)
    JOIN pricing p ON p.court_id = c.id
WHERE
    -- Excluir slots que ya tienen una reserva activa
    NOT EXISTS (
        SELECT 1 FROM bookings b
        WHERE b.court_id = $1
          AND b.starts_at = ts
          AND b.status NOT IN ('cancelled')
    )
    -- Solo dentro del horario habilitado
    AND ts::time >= s.open_time
    AND ts::time < s.close_time
ORDER BY ts;

-- name: CreateBooking :one
INSERT INTO bookings (
    club_id,
    court_id,
    user_phone,
    user_name,
    starts_at,
    ends_at,
    status,
    is_sandbox
) VALUES (
    $1, $2, $3, $4, $5, $6, 'confirmed', $7
)
RETURNING *;

-- name: GetBookingByID :one
SELECT * FROM bookings
WHERE id = $1 AND club_id = $2
LIMIT 1;

-- name: GetActiveBookingByPhone :one
-- Busca la próxima reserva activa de un usuario por su número de WA.
-- Usada cuando el usuario dice "cancelar mi turno".
SELECT * FROM bookings
WHERE user_phone = $1
  AND club_id = $2
  AND status = 'confirmed'
  AND starts_at > NOW()
ORDER BY starts_at ASC
LIMIT 1;

-- name: CancelBooking :one
UPDATE bookings
SET status = 'cancelled', updated_at = NOW()
WHERE id = $1 AND club_id = $2
RETURNING *;

-- name: GetFirstWaitlistEntry :one
SELECT * FROM waitlist
WHERE court_id = $1
  AND slot_time = $2
  AND club_id = $3
  AND notified_at IS NULL
ORDER BY created_at ASC
LIMIT 1;

-- name: AddToWaitlist :one
INSERT INTO waitlist (club_id, court_id, user_phone, user_name, slot_time)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetClubByWANumber :one
-- El Control Plane usa esta query para identificar a qué club
-- pertenece un número de WhatsApp entrante.
SELECT * FROM clubs
WHERE wa_phone_number = $1 AND is_active = true
LIMIT 1;

-- name: GetCourtsByClub :many
SELECT * FROM courts
WHERE club_id = $1 AND is_active = true
ORDER BY name;
