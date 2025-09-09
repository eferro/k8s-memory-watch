package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all configuration for the application
type Config struct {
	// Kubernetes configuration
	Namespace     string
	AllNamespaces bool // true if monitoring all namespaces explicitly
	KubeConfig    string
	InCluster     bool

	// Monitoring configuration
	CheckInterval        time.Duration
	MemoryThresholdMB    int64
	MemoryWarningPercent float64

	// Logging configuration
	LogLevel  string
	LogFormat string

	// Display configuration
	Labels      []string // Labels to display for each pod
	Annotations []string // Annotations to display for each pod
	Output      string   // Output format (table, csv)
}

// CLIConfig holds command line argument values
type CLIConfig struct {
	Namespace            string
	AllNamespaces        bool
	KubeConfig           string
	InCluster            bool
	CheckInterval        time.Duration
	MemoryThresholdMB    int64
	MemoryWarningPercent float64
	LogLevel             string
	Labels               string // Comma-separated list of labels to display
	Annotations          string // Comma-separated list of annotations to display
	Output               string // Output format (table, csv)
}

// Load loads configuration from environment variables with sensible defaults
func Load() (*Config, error) {
	return LoadWithCLI(nil)
}

// LoadWithCLI loads configuration from environment variables and CLI flags
// CLI flags take precedence over environment variables
func LoadWithCLI(cli *CLIConfig) (*Config, error) {
	cfg := defaultConfigFromEnv()
	applyCLIOverrides(cfg, cli)
	applyDefaultNamespace(cfg)
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}
	return cfg, nil
}

func defaultConfigFromEnv() *Config {
	return &Config{
		Namespace:            getEnv("NAMESPACE", ""),
		AllNamespaces:        getEnvBool("ALL_NAMESPACES", false),
		KubeConfig:           getEnv("KUBECONFIG", ""),
		InCluster:            getEnvBool("IN_CLUSTER", false),
		CheckInterval:        getEnvDuration("CHECK_INTERVAL", "30s"),
		MemoryThresholdMB:    getEnvInt64("MEMORY_THRESHOLD_MB", 1024),
		MemoryWarningPercent: getEnvFloat("MEMORY_WARNING_PERCENT", 80.0),
		LogLevel:             getEnv("LOG_LEVEL", "info"),
		LogFormat:            getEnv("LOG_FORMAT", "json"),
		Labels:               parseCommaSeparated(getEnv("LABELS", "")),
		Annotations:          parseCommaSeparated(getEnv("ANNOTATIONS", "")),
		Output:               getEnv("OUTPUT", "table"),
	}
}

func applyCLIOverrides(cfg *Config, cli *CLIConfig) {
	if cli == nil {
		return
	}
	overrideNamespace(cfg, cli)
	overrideKubeConfig(cfg, cli)
	overrideIntervals(cfg, cli)
	overrideLogging(cfg, cli)
	overrideDisplay(cfg, cli)
}

func overrideNamespace(cfg *Config, cli *CLIConfig) {
	if cli.Namespace != "" {
		cfg.Namespace = cli.Namespace
	}
	if cli.AllNamespaces {
		cfg.AllNamespaces = true
	}
}

func overrideKubeConfig(cfg *Config, cli *CLIConfig) {
	if cli.KubeConfig != "" {
		cfg.KubeConfig = cli.KubeConfig
	}
	if cli.InCluster {
		cfg.InCluster = true
	}
}

func overrideIntervals(cfg *Config, cli *CLIConfig) {
	if cli.CheckInterval != 0 {
		cfg.CheckInterval = cli.CheckInterval
	}
	if cli.MemoryThresholdMB != 0 {
		cfg.MemoryThresholdMB = cli.MemoryThresholdMB
	}
	if cli.MemoryWarningPercent != 0 {
		cfg.MemoryWarningPercent = cli.MemoryWarningPercent
	}
}

func overrideLogging(cfg *Config, cli *CLIConfig) {
	if cli.LogLevel != "" {
		cfg.LogLevel = cli.LogLevel
	}
	if cli.Output != "" {
		cfg.Output = cli.Output
	}
}

func overrideDisplay(cfg *Config, cli *CLIConfig) {
	if cli.Labels != "" {
		cfg.Labels = parseCommaSeparated(cli.Labels)
	}
	if cli.Annotations != "" {
		cfg.Annotations = parseCommaSeparated(cli.Annotations)
	}
}

func applyDefaultNamespace(cfg *Config) {
	if cfg.Namespace == "" && !cfg.AllNamespaces {
		cfg.AllNamespaces = true
	}
}

// validate checks that the configuration is valid
func (c *Config) validate() error {
	if c.CheckInterval <= 0 {
		return fmt.Errorf("check_interval must be positive")
	}

	if c.MemoryThresholdMB <= 0 {
		return fmt.Errorf("memory_threshold_mb must be positive")
	}

	if c.MemoryWarningPercent <= 0 || c.MemoryWarningPercent > 100 {
		return fmt.Errorf("memory_warning_percent must be between 0 and 100")
	}

	if c.Output != "table" && c.Output != "csv" {
		return fmt.Errorf("output must be either 'table' or 'csv'")
	}

	return nil
}

// Helper functions for environment variable parsing

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvDuration(key, defaultValue string) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			return parsed
		}
	}

	// Parse default value
	parsed, err := time.ParseDuration(defaultValue)
	if err != nil {
		panic(fmt.Sprintf("invalid default duration %s: %v", defaultValue, err))
	}
	return parsed
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			return parsed
		}
	}
	return defaultValue
}

// parseCommaSeparated parses a comma-separated string into a slice of trimmed, non-empty strings
func parseCommaSeparated(value string) []string {
	if value == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
