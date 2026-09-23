package server

import (
	"context"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/aztechian/heirloom/internal/config"
	"github.com/aztechian/heirloom/internal/storage"
	"github.com/rs/zerolog"
)

const requestTimeout = 30 * time.Second

// NewServer builds the HTTP handler and returns it along with a cancel func
// for the derived context. The caller owns the context's lifetime and must
// call the returned cancel func (e.g. via defer) once the server is done.
func NewServer(ctx *context.Context, cfg config.Config, store storage.Storage) (*http.Server, context.CancelFunc) {
	_, cancel := context.WithCancel(*ctx)

	mux := http.NewServeMux()
	applyRoutes(mux, store)
	handler := applyGlobalMiddleware(mux)

	// get the zerolog logger from the context, then wrap it with a log.New() to use it as the http Server's ErrorLog
	logger := zerolog.Ctx(*ctx)
	httpErrLog := log.New(logger.Level(zerolog.ErrorLevel), "", 0)

	return &http.Server{
		BaseContext:  func(net.Listener) context.Context { return *ctx },
		Addr:         ":" + cfg.Server.Port,
		Handler:      handler,
		ReadTimeout:  requestTimeout,
		WriteTimeout: requestTimeout,
		ErrorLog:     httpErrLog,
	}, cancel
}
