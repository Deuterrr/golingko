package api

import (
	"log/slog"
	"net/http"
	"time"

	"golingko/internal/config"
	"golingko/pkg/middleware"
)

func wrapMiddlewares(raw http.Handler, logger *slog.Logger) http.Handler {
	idempotency := middleware.NewIdempotencyStore()

	var wrapped http.Handler = raw
	wrapped = middleware.LoggingMiddleware(logger)(wrapped)
	wrapped = idempotency.Middleware(5 * time.Minute)(wrapped)
	wrapped = middleware.RecoveryMiddleware(logger)(wrapped)
	wrapped = middleware.CORSMiddleware(wrapped)

	return wrapped
}

func NewServer(cfg *config.Config, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              cfg.ServerAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}
