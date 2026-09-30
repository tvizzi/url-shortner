package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"url-shortener/internal/config"
	"url-shortener/internal/lib/logger/handlers/slogpretty"
	"url-shortener/internal/lib/logger/sl"
	"url-shortener/internal/service/urlservice"
	"url-shortener/internal/storage/postgres"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"url-shortener/internal/http-server/handlers/redirect"
	"url-shortener/internal/http-server/handlers/url/delete"
	"url-shortener/internal/http-server/handlers/url/save"
	"url-shortener/internal/http-server/handlers/url/update"
	mwLogger "url-shortener/internal/http-server/middleware/logger"
)

const (
	envLocal = "local"
	envProd  = "prod"
)

func main() {
	if err := run(); err != nil {
		slog.Error("service stopped with an error", sl.Err(err))
		os.Exit(1)
	}
}

func run() error {
	// Get environment
	cfg := config.MustLoad()

	// Settings logger
	log := setupLogger(cfg.Env)
	log.Info("starting the project...", slog.String("env", cfg.Env), slog.String("version", "v1"))
	log.Debug("debug messages are enabled")
	log.Error("error messages are enabled")

	// Settings and started database
	storage, err := postgres.New(cfg.DatabaseURL)
	if err != nil {
		log.Error("failed to init storage", sl.Err(err))
		return err
	}
	defer storage.Close()

	urlService := urlservice.New(storage, cfg.AliasLength)

	// Init router
	router := chi.NewRouter()

	// Middlewares
	router.Use(middleware.RequestID) // Хороший middleware для логирования
	router.Use(mwLogger.New(log))    // Хороший middleware для логирования (custom)
	router.Use(middleware.Logger)    // Логирует все входящие запросы
	router.Use(middleware.Recoverer) // Перехватывает паники и возвращает 500
	router.Use(middleware.URLFormat) // Для красивых URL при подключении к обработчикам

	// Handlers with Auth
	router.Route("/url", func(r chi.Router) {
		r.Use(middleware.BasicAuth("url-shortener", map[string]string{
			cfg.Auth.User: cfg.Auth.Password,
		}))

		r.Post("/", save.New(log, urlService))
		r.Put("/", update.New(log, urlService))
		r.Delete("/{alias}", delete.New(log, urlService))
	})

	// Handlers without Auth
	router.Get("/{alias}", redirect.New(log, urlService))

	log.Info("starting server", slog.String("address", cfg.Address))

	// Settings and started server
	srv := &http.Server{
		Addr:         cfg.Address,
		Handler:      router,
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)
	serverErrors := make(chan error, 1)
	go func() { serverErrors <- srv.ListenAndServe() }()
	select {
	case sig := <-signals:
		log.Info("shutdown signal received", slog.String("signal", sig.String()))
	case err := <-serverErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("server failed: %w", err)
		}
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("graceful shutdown failed", sl.Err(err))
		_ = srv.Close()
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	if err := <-serverErrors; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server stopped with error: %w", err)
	}

	log.Info("server stopped")
	return nil
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = setupPrettySlog() // Для локальный разработки - самописный logger
	case envProd:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}

	return log
}

func setupPrettySlog() *slog.Logger {
	opts := slogpretty.PrettyHandlerOptions{
		SlogOpts: &slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	}

	handler := opts.NewPrettyHandler(os.Stdout)

	return slog.New(handler)
}
