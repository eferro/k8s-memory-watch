package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	// Test with default values
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify defaults (now defaults to all namespaces)
	if cfg.Namespace != "" {
		t.Errorf("Expected empty namespace (all namespaces), got '%s'", cfg.Namespace)
	}

	if !cfg.AllNamespaces {
		t.Error("Expected AllNamespaces to be true by default")
	}

	if cfg.CheckInterval != 30*time.Second {
		t.Errorf("Expected check interval '30s', got '%v'", cfg.CheckInterval)
	}

	if cfg.MemoryThresholdMB != 1024 {
		t.Errorf("Expected memory threshold '1024', got '%d'", cfg.MemoryThresholdMB)
	}
}

func TestLoadWithEnvironmentVariables(t *testing.T) {
	// Set up test environment variables
	testCases := map[string]string{
		"NAMESPACE":              "test-namespace",
		"CHECK_INTERVAL":         "1m",
		"MEMORY_THRESHOLD_MB":    "2048",
		"MEMORY_WARNING_PERCENT": "90.0",
		"IN_CLUSTER":             "true",
	}

	// Set environment variables
	for key, value := range testCases {
		if err := os.Setenv(key, value); err != nil {
			t.Fatalf("Failed to set env var %s: %v", key, err)
		}
	}

	// Clean up after test
	defer func() {
		for key := range testCases {
			os.Unsetenv(key)
		}
	}()

	// Load configuration
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify environment variables were applied
	if cfg.Namespace != "test-namespace" {
		t.Errorf("Expected namespace 'test-namespace', got '%s'", cfg.Namespace)
	}

	if cfg.CheckInterval != time.Minute {
		t.Errorf("Expected check interval '1m', got '%v'", cfg.CheckInterval)
	}

	if cfg.MemoryThresholdMB != 2048 {
		t.Errorf("Expected memory threshold '2048', got '%d'", cfg.MemoryThresholdMB)
	}

	if cfg.MemoryWarningPercent != 90.0 {
		t.Errorf("Expected warning percent '90.0', got '%f'", cfg.MemoryWarningPercent)
	}

	if !cfg.InCluster {
		t.Error("Expected InCluster to be true")
	}
}

func TestValidate(t *testing.T) {
	testCases := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				CheckInterval:        30 * time.Second,
				MemoryThresholdMB:    1024,
				MemoryWarningPercent: 80.0,
				Output:               "table",
			},
			wantErr: false,
		},
		{
			name: "invalid check interval",
			config: Config{
				CheckInterval:        0,
				MemoryThresholdMB:    1024,
				MemoryWarningPercent: 80.0,
				Output:               "table",
			},
			wantErr: true,
		},
		{
			name: "invalid memory threshold",
			config: Config{
				CheckInterval:        30 * time.Second,
				MemoryThresholdMB:    0,
				MemoryWarningPercent: 80.0,
				Output:               "table",
			},
			wantErr: true,
		},
		{
			name: "invalid warning percent - too low",
			config: Config{
				CheckInterval:        30 * time.Second,
				MemoryThresholdMB:    1024,
				MemoryWarningPercent: 0,
				Output:               "table",
			},
			wantErr: true,
		},
		{
			name: "invalid warning percent - too high",
			config: Config{
				CheckInterval:        30 * time.Second,
				MemoryThresholdMB:    1024,
				MemoryWarningPercent: 101.0,
				Output:               "table",
			},
			wantErr: true,
		},
		{
			name: "valid output - csv",
			config: Config{
				CheckInterval:        30 * time.Second,
				MemoryThresholdMB:    1024,
				MemoryWarningPercent: 80.0,
				Output:               "csv",
			},
			wantErr: false,
		},
		{
			name: "invalid output format",
			config: Config{
				CheckInterval:        30 * time.Second,
				MemoryThresholdMB:    1024,
				MemoryWarningPercent: 80.0,
				Output:               "json",
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.config.validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}
