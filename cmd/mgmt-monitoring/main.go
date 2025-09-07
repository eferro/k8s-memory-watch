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
	"github.com/eduardoferro/mgmt-monitoring/internal/monitor"
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

	// Create memory monitor
	memMonitor, err := monitor.New(cfg)
	if err != nil {
		log.Fatal("Failed to create memory monitor:", err)
	}

	// Set up context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Perform initial health check
	slog.Info("Performing initial health check...")
	if err := memMonitor.HealthCheck(ctx); err != nil {
		slog.Error("Health check failed", "error", err)
		cancel()
		return
	}

	// Set up graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		slog.Info("Received shutdown signal, gracefully shutting down...")
		cancel()
	}()

	// Main application loop
	slog.Info("Starting monitoring loop...")

	// Run initial collection and analysis
	if err := runMemoryCheck(ctx, memMonitor); err != nil {
		slog.Error("Initial memory check failed", "error", err)
	}

	ticker := time.NewTicker(cfg.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Application shutdown complete")
			return
		case <-ticker.C:
			if err := runMemoryCheck(ctx, memMonitor); err != nil {
				slog.Error("Memory check cycle failed", "error", err)
			}
		}
	}
}

// runMemoryCheck executes a single cycle of memory monitoring and analysis
func runMemoryCheck(ctx context.Context, memMonitor *monitor.MemoryMonitor) error {
	slog.Info("Starting memory check cycle...", "timestamp", time.Now().Format(time.RFC3339))

	// Perform memory analysis
	analysis, err := memMonitor.AnalyzeMemoryUsage(ctx)
	if err != nil {
		return err
	}

	// Print the complete detailed report showing all pods
	analysis.Report.PrintDetailedReport()
	
	// Always print analysis (warnings, recommendations)
	analysis.PrintAnalysis()

	// Log summary information structured
	slog.Info("Memory check completed",
		"total_pods", analysis.Report.Summary.TotalPods,
		"running_pods", analysis.Report.Summary.RunningPods,
		"problems_found", len(analysis.ProblemsFound),
		"high_usage_pods", len(analysis.HighUsagePods),
		"warning_pods", len(analysis.WarningPods),
		"total_memory_usage", analysis.Report.Summary.TotalMemoryUsage.String(),
	)

	return nil
}
