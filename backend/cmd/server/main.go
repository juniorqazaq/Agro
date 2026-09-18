package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/juniorqazaq/Agro/backend/internal/analyzer"
	"github.com/juniorqazaq/Agro/backend/internal/config"
	"github.com/juniorqazaq/Agro/backend/internal/httpapi"
	"github.com/juniorqazaq/Agro/backend/internal/knowledgebase"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	store, err := knowledgebase.Load(cfg.KnowledgeBase)
	if err != nil {
		logger.Error("failed to load knowledge base", "error", err)
		os.Exit(1)
	}

	var imageAnalyzer analyzer.Analyzer
	if cfg.MockDiseaseID != "" {
		imageAnalyzer = analyzer.Mock{DiseaseID: cfg.MockDiseaseID}
		logger.Warn("AI mock mode is enabled", "disease_id", cfg.MockDiseaseID)
	} else {
		imageAnalyzer = analyzer.NewGemini(
			cfg.GeminiAPIKey,
			cfg.GeminiModel,
			cfg.AITimeout,
			cfg.AIRetries,
		)
	}

	server := &http.Server{
		Addr:              cfg.Address,
		Handler:           httpapi.New(imageAnalyzer, store, logger),
		ReadHeaderTimeout: cfg.AITimeout,
	}

	logger.Info("Plantix backend started", "address", cfg.Address, "model", cfg.GeminiModel)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
