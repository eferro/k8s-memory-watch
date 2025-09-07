package k8s

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/metrics/pkg/client/clientset/versioned"
)

// Client wraps Kubernetes clients
type Client struct {
	clientset     kubernetes.Interface
	metricsClient versioned.Interface
	config        *rest.Config
}

// NewClient creates a new Kubernetes client
func NewClient(kubeconfig string, inCluster bool) (*Client, error) {
	var config *rest.Config
	var err error

	if inCluster {
		// Use in-cluster configuration
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to create in-cluster config: %w", err)
		}
	} else {
		// Use kubeconfig file
		if kubeconfig == "" {
			// Default to ~/.kube/config
			home, homeErr := os.UserHomeDir()
			if homeErr != nil {
				return nil, fmt.Errorf("failed to get home directory: %w", homeErr)
			}
			kubeconfig = filepath.Join(home, ".kube", "config")
		}

		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to build config from kubeconfig: %w", err)
		}
	}

	// Create standard Kubernetes clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	// Create metrics clientset
	metricsClient, err := versioned.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create metrics client: %w", err)
	}

	return &Client{
		clientset:     clientset,
		metricsClient: metricsClient,
		config:        config,
	}, nil
}

// HealthCheck verifies the client can connect to the cluster
func (c *Client) HealthCheck(_ context.Context) error {
	_, err := c.clientset.Discovery().ServerVersion()
	if err != nil {
		return fmt.Errorf("failed to connect to Kubernetes cluster: %w", err)
	}
	return nil
}
