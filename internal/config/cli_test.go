package config

import (
	"testing"
	"time"
)

func TestLoadWithCLI(t *testing.T) {
	testCases := []struct {
		name        string
		cli         *CLIConfig
		expectedNS  string
		expectedAll bool
	}{
		{
			name: "CLI namespace overrides default",
			cli: &CLIConfig{
				Namespace: "production",
			},
			expectedNS:  "production",
			expectedAll: false,
		},
		{
			name: "CLI all-namespaces flag",
			cli: &CLIConfig{
				AllNamespaces: true,
			},
			expectedNS:  "",
			expectedAll: true,
		},
		{
			name:        "nil CLI config defaults to all namespaces",
			cli:         nil,
			expectedNS:  "",
			expectedAll: true,
		},
		{
			name: "CLI check interval override",
			cli: &CLIConfig{
				CheckInterval: 2 * time.Minute,
			},
			expectedNS:  "",
			expectedAll: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := LoadWithCLI(tc.cli)
			if err != nil {
				t.Fatalf("LoadWithCLI() failed: %v", err)
			}

			if cfg.Namespace != tc.expectedNS {
				t.Errorf("Expected namespace '%s', got '%s'", tc.expectedNS, cfg.Namespace)
			}

			if cfg.AllNamespaces != tc.expectedAll {
				t.Errorf("Expected AllNamespaces %t, got %t", tc.expectedAll, cfg.AllNamespaces)
			}

			// Check that CLI overrides work
			if tc.cli != nil && tc.cli.CheckInterval != 0 {
				if cfg.CheckInterval != tc.cli.CheckInterval {
					t.Errorf("Expected check interval %v, got %v", tc.cli.CheckInterval, cfg.CheckInterval)
				}
			}
		})
	}
}

func TestLoadWithCLI_Precedence(t *testing.T) {
	// Set environment variable
	t.Setenv("NAMESPACE", "env-namespace")
	t.Setenv("CHECK_INTERVAL", "45s")

	cli := &CLIConfig{
		Namespace:     "cli-namespace",
		CheckInterval: 2 * time.Minute,
	}

	cfg, err := LoadWithCLI(cli)
	if err != nil {
		t.Fatalf("LoadWithCLI() failed: %v", err)
	}

	// CLI should override environment variable
	if cfg.Namespace != "cli-namespace" {
		t.Errorf("Expected CLI to override env var: got namespace '%s', want 'cli-namespace'", cfg.Namespace)
	}

	if cfg.CheckInterval != 2*time.Minute {
		t.Errorf("Expected CLI to override env var: got interval %v, want %v", cfg.CheckInterval, 2*time.Minute)
	}
}

func TestLoadWithCLI_DefaultBehavior(t *testing.T) {
	// Test default behavior when no namespace is specified
	cfg, err := LoadWithCLI(&CLIConfig{})
	if err != nil {
		t.Fatalf("LoadWithCLI() failed: %v", err)
	}

	// Should default to all namespaces when no specific namespace is provided
	if !cfg.AllNamespaces {
		t.Error("Expected AllNamespaces to be true when no namespace is specified")
	}

	if cfg.Namespace != "" {
		t.Errorf("Expected empty namespace, got '%s'", cfg.Namespace)
	}
}

func TestLoadWithCLI_ValidationIntegrity(t *testing.T) {
	// Test that validation still works with CLI config
	cli := &CLIConfig{
		CheckInterval: -1 * time.Second, // Invalid
	}

	_, err := LoadWithCLI(cli)
	if err == nil {
		t.Error("Expected validation error for negative check interval")
	}
}

func TestNamespaceLogic(t *testing.T) {
	testCases := []struct {
		name        string
		cli         *CLIConfig
		expectedNS  string
		expectedAll bool
		description string
	}{
		{
			name:        "specific namespace wins",
			cli:         &CLIConfig{Namespace: "prod"},
			expectedNS:  "prod",
			expectedAll: false,
			description: "When namespace is specified, AllNamespaces should be false",
		},
		{
			name:        "explicit all namespaces",
			cli:         &CLIConfig{AllNamespaces: true},
			expectedNS:  "",
			expectedAll: true,
			description: "When AllNamespaces is true, Namespace should be empty",
		},
		{
			name:        "empty config defaults to all",
			cli:         &CLIConfig{},
			expectedNS:  "",
			expectedAll: true,
			description: "Default behavior should be all namespaces",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := LoadWithCLI(tc.cli)
			if err != nil {
				t.Fatalf("LoadWithCLI() failed: %v", err)
			}

			if cfg.Namespace != tc.expectedNS {
				t.Errorf("%s: Expected namespace '%s', got '%s'", tc.description, tc.expectedNS, cfg.Namespace)
			}

			if cfg.AllNamespaces != tc.expectedAll {
				t.Errorf("%s: Expected AllNamespaces %t, got %t", tc.description, tc.expectedAll, cfg.AllNamespaces)
			}
		})
	}
}
