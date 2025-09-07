package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eduardoferro/k8s-memory-watch/internal/config"
	"github.com/eduardoferro/k8s-memory-watch/internal/monitor"
)

// Global variable to track if CSV header has been printed
var csvHeaderPrinted = false

func main() {
	// Parse command line flags
	var (
		namespace       = flag.String("namespace", "", "Monitor specific namespace (default: all namespaces)")
		allNamespaces   = flag.Bool("all-namespaces", false, "Monitor all namespaces explicitly")
		kubeconfig      = flag.String("kubeconfig", "", "Path to kubeconfig file")
		inCluster       = flag.Bool("in-cluster", false, "Use in-cluster configuration")
		checkInterval   = flag.Duration("check-interval", 0, "Check interval (e.g., 30s, 1m)")
		memoryThreshold = flag.Int64("memory-threshold", 0, "Memory threshold in MB")
		memoryWarning   = flag.Float64("memory-warning", 0, "Memory warning percentage")
		logLevel        = flag.String("log-level", "", "Log level (debug, info, warn, error)")
		labels          = flag.String("labels", "", "Comma-separated list of labels to display (e.g., dag_id,task_id,run_id)")
		annotations     = flag.String("annotations", "", "Comma-separated list of annotations to display")
		output          = flag.String("output", "table", "Output format (table, csv)")
		help            = flag.Bool("help", false, "Show help message")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Kubernetes Memory Monitoring Tool\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "OPTIONS:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s --namespace=production\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --all-namespaces\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --kubeconfig=/path/to/config --check-interval=1m\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --labels=dag_id,task_id,run_id\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --annotations=owner,team --labels=app\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --output=csv --labels=app,version > pods.csv\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --output=csv --all-namespaces > cluster-memory.csv\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables (lower priority than CLI flags):\n")
		fmt.Fprintf(os.Stderr, "  NAMESPACE, KUBECONFIG, IN_CLUSTER, CHECK_INTERVAL,\n")
		fmt.Fprintf(os.Stderr, "  MEMORY_THRESHOLD_MB, MEMORY_WARNING_PERCENT, LOG_LEVEL\n")
	}

	flag.Parse()

	if *help {
		flag.Usage()
		os.Exit(0)
	}

	// Validate mutually exclusive flags
	if *namespace != "" && *allNamespaces {
		fmt.Fprintf(os.Stderr, "Error: --namespace and --all-namespaces are mutually exclusive\n")
		os.Exit(1)
	}

	// Create CLI config
	cliConfig := &config.CLIConfig{
		Namespace:            *namespace,
		AllNamespaces:        *allNamespaces,
		KubeConfig:           *kubeconfig,
		InCluster:            *inCluster,
		CheckInterval:        *checkInterval,
		MemoryThresholdMB:    *memoryThreshold,
		MemoryWarningPercent: *memoryWarning,
		LogLevel:             *logLevel,
		Labels:               *labels,
		Annotations:          *annotations,
		Output:               *output,
	}

	// Load configuration (combines env vars with CLI flags)
	cfg, err := config.LoadWithCLI(cliConfig)
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	// Set up structured logging (suppressed in CSV mode)
	if cfg.Output != "csv" {
		logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
		slog.SetDefault(logger)
		slog.Info("Starting Kubernetes Management Monitoring Application")
		slog.Info("Configuration loaded successfully",
			"namespace", cfg.Namespace,
			"all_namespaces", cfg.AllNamespaces,
			"check_interval", cfg.CheckInterval)
	}

	// Create memory monitor
	memMonitor, err := monitor.New(cfg)
	if err != nil {
		log.Fatal("Failed to create memory monitor:", err)
	}

	// Set up context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Perform initial health check
	if cfg.Output != "csv" {
		slog.Info("Performing initial health check...")
	}
	if err := memMonitor.HealthCheck(ctx); err != nil {
		if cfg.Output != "csv" {
			slog.Error("Health check failed", "error", err)
		}
		cancel()
		return
	}

	// Set up graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		if cfg.Output != "csv" {
			slog.Info("Received shutdown signal, gracefully shutting down...")
		}
		cancel()
	}()

	// Main application loop
	if cfg.Output != "csv" {
		slog.Info("Starting monitoring loop...")
	}

	// Run initial collection and analysis
	if err := runMemoryCheck(ctx, memMonitor, cfg); err != nil {
		if cfg.Output != "csv" {
			slog.Error("Initial memory check failed", "error", err)
		}
	}

	ticker := time.NewTicker(cfg.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			if cfg.Output != "csv" {
				slog.Info("Application shutdown complete")
			}
			return
		case <-ticker.C:
			if err := runMemoryCheck(ctx, memMonitor, cfg); err != nil {
				if cfg.Output != "csv" {
					slog.Error("Memory check cycle failed", "error", err)
				}
			}
		}
	}
}

// runMemoryCheck executes a single cycle of memory monitoring and analysis
func runMemoryCheck(ctx context.Context, memMonitor *monitor.MemoryMonitor, cfg *config.Config) error {
	if cfg.Output != "csv" {
		slog.Info("Starting memory check cycle...", "timestamp", time.Now().Format(time.RFC3339))
	}

	// Perform memory analysis
	analysis, err := memMonitor.AnalyzeMemoryUsage(ctx)
	if err != nil {
		return err
	}

	// Print output according to format
	if cfg.Output == "csv" {
		// Show header only on first run
		analysis.Report.PrintCSV(cfg, !csvHeaderPrinted)
		csvHeaderPrinted = true
	} else {
		// Print the complete detailed report showing all pods
		analysis.Report.PrintDetailedReport(cfg)
		// Always print analysis (warnings, recommendations)
		analysis.PrintAnalysis(cfg)
	}

	// Log summary information structured (only in table mode)
	if cfg.Output != "csv" {
		slog.Info("Memory check completed",
			"total_pods", analysis.Report.Summary.TotalPods,
			"running_pods", analysis.Report.Summary.RunningPods,
			"problems_found", len(analysis.ProblemsFound),
			"high_usage_pods", len(analysis.HighUsagePods),
			"warning_pods", len(analysis.WarningPods),
			"total_memory_usage", analysis.Report.Summary.TotalMemoryUsage.String(),
		)
	}

	return nil
}
