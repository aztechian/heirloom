package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aztechian/heirloom/internal/config"
	"github.com/aztechian/heirloom/internal/server"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const shutdownTimeout = 30 * time.Second

func main() {
	// Configure zerolog
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).Level(zerolog.InfoLevel)
	serverCtx := log.Logger.WithContext(context.Background()) // add logger to top-level context

	// Load configuration
	config, err := config.LoadConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Check if help was requested
	if config.ShowHelp() {
		config.PrintHelp()
		return
	}

	srv, cancel := server.NewServer(&serverCtx, *config)
	defer cancel()

	// Set up channel to listen for signals
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// Create server, and start in a go routine
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error().Err(err).Msg("Server failed to start")
		}
	}()

	// Wait for signal
	sig := <-signalChan
	log.Info().Str("signal", sig.String()).Msg("Received shutdown signal")

	// Create a timeout context for shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(serverCtx, shutdownTimeout)
	defer shutdownCancel()

	// Shutdown the server
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("Server shutdown error")
	}

	log.Info().Msg("Server shutdown complete")
}
