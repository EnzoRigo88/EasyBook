package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool crea un connection pool de Postgres usando pgx.
// pgx es el driver de Postgres más completo para Go — soporta
// pgxpool para concurrencia, tipos nativos de PG, y copy protocol.
func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("error parseando DATABASE_URL: %w", err)
	}

	// Configuración del pool de conexiones.
	// Con n8n corriendo múltiples workflows en paralelo, necesitamos
	// suficientes conexiones pero sin exceder el límite de Postgres.
	config.MaxConns = 25
	config.MinConns = 5

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("error creando pool de Postgres: %w", err)
	}

	// Verificar conectividad
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("no se pudo hacer ping a Postgres: %w", err)
	}

	return pool, nil
}
