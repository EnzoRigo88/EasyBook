package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/EnzoRigo88/EasyBook/control-plane/internal/api"
	"github.com/EnzoRigo88/EasyBook/control-plane/internal/config"
	"github.com/EnzoRigo88/EasyBook/control-plane/internal/db"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// ── Logger ────────────────────────────────────────────────────────────────
	// En development usamos pretty-print. En producción, JSON estructurado.
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error cargando config: %v\n", err)
		os.Exit(1)
	}

	if cfg.IsDevelopment() {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "15:04:05"})
	}

	log.Info().Str("env", cfg.Env).Str("port", cfg.Port).Msg("🎾 EasyBook control-plane iniciando")

	// ── Database ──────────────────────────────────────────────────────────────
	pool, err := db.NewPool(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("no se pudo conectar a Postgres")
	}
	defer pool.Close()
	log.Info().Msg("✅ Postgres conectado")

	// ── Router ────────────────────────────────────────────────────────────────
	router := api.NewRouter(cfg, pool)

	// ── HTTP Server ───────────────────────────────────────────────────────────
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,  // 30s para que Claude tenga tiempo de responder
		IdleTimeout:  60 * time.Second,
	}

	// Arrancar en goroutine separada para poder manejar shutdown graceful
	go func() {
		log.Info().Str("addr", srv.Addr).Msg("🚀 Servidor HTTP escuchando")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("error en servidor HTTP")
		}
	}()

	// ── Graceful shutdown ─────────────────────────────────────────────────────
	// Espera señal SIGINT (Ctrl+C) o SIGTERM (Docker stop / Kubernetes)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("🛑 Apagando servidor...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("error en shutdown del servidor")
	}

	log.Info().Msg("✅ Servidor apagado correctamente")
}
