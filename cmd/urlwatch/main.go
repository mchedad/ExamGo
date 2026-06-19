// Package main est le point d'entrée du service URLWatch.
// Il assemble les dépendances et démarre le serveur HTTP.
package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"moduleGo/urlwatch/internal/api"
	"moduleGo/urlwatch/internal/checker"
	"moduleGo/urlwatch/internal/store"
)

func main() {
	// Initialisation du logger structuré
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Câblage des dépendances (inversion de dépendance)
	httpChecker := checker.NewHTTPChecker()
	memStore := store.NewMemoryStore()

	// Configuration et démarrage du serveur HTTP
	addr := ":8080"
	router := api.NewRouter(httpChecker, memStore, logger)

	logger.Info("démarrage du serveur URLWatch", "addr", addr)
	fmt.Printf("URLWatch en écoute sur %s\n", addr)

	if err := http.ListenAndServe(addr, router); err != nil {
		logger.Error("erreur serveur", "error", err)
		os.Exit(1)
	}
}
