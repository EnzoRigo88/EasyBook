package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// contextKey es un tipo privado para evitar colisiones de claves en context.
// En Go, nunca uses strings como claves de context en producción.
type contextKey string

const (
	ClubIDKey   contextKey = "club_id"
	UserRoleKey contextKey = "user_role"
)

// Logger es un middleware de logging estructurado con zerolog.
// Loguea cada request con: método, path, status, duración y club_id.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		defer func() {
			log.Info().
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Int("status", ww.Status()).
				Dur("duration", time.Since(start)).
				Str("requestId", middleware.GetReqID(r.Context())).
				Msg("request")
		}()

		next.ServeHTTP(ww, r)
	})
}

// RequireAuth valida el JWT en el header Authorization.
// En Go, los middleware son simplemente funciones que retornan http.Handler.
func RequireAuth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extraer token del header "Authorization: Bearer <token>"
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"error":"unauthorized","code":"MISSING_TOKEN"}`, http.StatusUnauthorized)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, `{"error":"unauthorized","code":"INVALID_TOKEN_FORMAT"}`, http.StatusUnauthorized)
				return
			}

			tokenStr := parts[1]
			token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
				// Verificar que el algoritmo sea HMAC (HS256), no "none"
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(jwtSecret), nil
			})

			if err != nil || !token.Valid {
				http.Error(w, `{"error":"unauthorized","code":"INVALID_TOKEN"}`, http.StatusUnauthorized)
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				http.Error(w, `{"error":"unauthorized","code":"INVALID_CLAIMS"}`, http.StatusUnauthorized)
				return
			}

			// Inyectar club_id en el context para que los handlers lo usen
			clubID, _ := claims["club_id"].(string)
			role, _ := claims["role"].(string)

			ctx := context.WithValue(r.Context(), ClubIDKey, clubID)
			ctx = context.WithValue(ctx, UserRoleKey, role)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// InjectTenant configura el schema de Postgres para el club del request.
// Implementa el aislamiento multi-tenant: cada query usa el schema correcto.
func InjectTenant(pool *pgxpool.Pool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clubID, ok := r.Context().Value(ClubIDKey).(string)
			if !ok || clubID == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Adquirir una conexión del pool y setear el search_path al schema del club.
			// Esto garantiza que todas las queries de este request lean solo datos del club.
			conn, err := pool.Acquire(r.Context())
			if err != nil {
				log.Error().Err(err).Msg("error adquiriendo conexión del pool")
				http.Error(w, `{"error":"internal_error"}`, http.StatusInternalServerError)
				return
			}
			defer conn.Release()

			// SET app.club_id activa el Row Level Security en Postgres
			_, err = conn.Exec(r.Context(),
				"SET app.club_id = $1", clubID,
			)
			if err != nil {
				log.Error().Err(err).Str("clubId", clubID).Msg("error seteando club_id en Postgres")
				http.Error(w, `{"error":"internal_error"}`, http.StatusInternalServerError)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAdminRole verifica que el usuario tenga rol de admin.
func RequireAdminRole(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, _ := r.Context().Value(UserRoleKey).(string)
		if role != "admin" {
			http.Error(w, `{"error":"forbidden","code":"INSUFFICIENT_ROLE"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// GetClubID es un helper para extraer el club_id del context en los handlers.
func GetClubID(ctx context.Context) string {
	v, _ := ctx.Value(ClubIDKey).(string)
	return v
}
