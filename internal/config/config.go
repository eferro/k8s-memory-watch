package config

import (
	"fmt"
	"os"
	"strconv"
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
}

// Load loads configuration from environment variables with sensible defaults
func Load() (*Config, error) {
	return LoadWithCLI(nil)
}

// LoadWithCLI loads configuration from environment variables and CLI flags
// CLI flags take precedence over environment variables
func LoadWithCLI(cli *CLIConfig) (*Config, error) {
	cfg := &Config{
		// Default values from environment variables
		Namespace:            getEnv("NAMESPACE", ""),
		AllNamespaces:        getEnvBool("ALL_NAMESPACES", false),
		KubeConfig:           getEnv("KUBECONFIG", ""),
		InCluster:            getEnvBool("IN_CLUSTER", false),
		CheckInterval:        getEnvDuration("CHECK_INTERVAL", "30s"),
		MemoryThresholdMB:    getEnvInt64("MEMORY_THRESHOLD_MB", 1024), // 1GB default
		MemoryWarningPercent: getEnvFloat("MEMORY_WARNING_PERCENT", 80.0),
		LogLevel:             getEnv("LOG_LEVEL", "info"),
		LogFormat:            getEnv("LOG_FORMAT", "json"),
	}

	// Apply CLI flags (they override environment variables)
	if cli != nil {
		if cli.Namespace != "" {
			cfg.Namespace = cli.Namespace
		}
		if cli.AllNamespaces {
			cfg.AllNamespaces = cli.AllNamespaces
		}
		if cli.KubeConfig != "" {
			cfg.KubeConfig = cli.KubeConfig
		}
		if cli.InCluster {
			cfg.InCluster = cli.InCluster
		}
		if cli.CheckInterval != 0 {
			cfg.CheckInterval = cli.CheckInterval
		}
		if cli.MemoryThresholdMB != 0 {
			cfg.MemoryThresholdMB = cli.MemoryThresholdMB
		}
		if cli.MemoryWarningPercent != 0 {
			cfg.MemoryWarningPercent = cli.MemoryWarningPercent
		}
		if cli.LogLevel != "" {
			cfg.LogLevel = cli.LogLevel
		}
	}

	// Apply default behavior for namespace selection
	if cfg.Namespace == "" && !cfg.AllNamespaces {
		// If no specific namespace and not explicitly all-namespaces,
		// default to all namespaces (Kubernetes-style behavior)
		cfg.AllNamespaces = true
	}

	// Validate configuration
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cfg, nil
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
