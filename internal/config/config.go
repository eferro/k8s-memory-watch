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
	Namespace  string
	KubeConfig string
	InCluster  bool

	// Monitoring configuration
	CheckInterval        time.Duration
	MemoryThresholdMB    int64
	MemoryWarningPercent float64

	// Logging configuration
	LogLevel  string
	LogFormat string
}

// Load loads configuration from environment variables with sensible defaults
func Load() (*Config, error) {
	cfg := &Config{
		// Default values
		Namespace:            getEnv("NAMESPACE", "default"),
		KubeConfig:           getEnv("KUBECONFIG", ""),
		InCluster:            getEnvBool("IN_CLUSTER", false),
		CheckInterval:        getEnvDuration("CHECK_INTERVAL", "30s"),
		MemoryThresholdMB:    getEnvInt64("MEMORY_THRESHOLD_MB", 1024), // 1GB default
		MemoryWarningPercent: getEnvFloat("MEMORY_WARNING_PERCENT", 80.0),
		LogLevel:             getEnv("LOG_LEVEL", "info"),
		LogFormat:            getEnv("LOG_FORMAT", "json"),
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
