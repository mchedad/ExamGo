package main

import (
	"log/slog"
	"net/http"
	"os"
	"strings"

	"moduleGo/urlwatch/internal/api"
	"moduleGo/urlwatch/internal/checker"
	"moduleGo/urlwatch/internal/store"
)

func main() {
	level := parseLogLevel(os.Getenv("LOG_LEVEL"))
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)

	httpChecker := checker.NewHTTPChecker()
	memStore := store.NewMemoryStore()

	addr := ":8080"
	router := api.NewRouter(httpChecker, memStore, logger)

	logger.Info("démarrage URLWatch", "addr", addr, "log_level", level.String())

	if err := http.ListenAndServe(addr, router); err != nil {
		logger.Error("erreur serveur", "error", err)
		os.Exit(1)
	}
}

func parseLogLevel(s string) slog.Level {
	switch strings.ToUpper(s) {
	case "DEBUG":
		return slog.LevelDebug
	case "WARN", "WARNING":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
