package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eduardoferro/mgmt-monitoring/internal/config"
)

func main() {
	// Set up structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	slog.Info("Starting Kubernetes Management Monitoring Application")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	slog.Info("Configuration loaded successfully",
		"namespace", cfg.Namespace,
		"check_interval", cfg.CheckInterval)

	// Set up context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		slog.Info("Received shutdown signal, gracefully shutting down...")
		cancel()
	}()

	// Main application loop placeholder
	slog.Info("Starting monitoring loop...")

	ticker := time.NewTicker(cfg.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Application shutdown complete")
			return
		case <-ticker.C:
			// This is where the monitoring logic will go
			slog.Info("Running memory check cycle...", "timestamp", time.Now().Format(time.RFC3339))
		}
	}
}
