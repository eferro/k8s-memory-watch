#!/bin/bash

# Examples of how to run the Kubernetes Memory Monitoring application

echo "🚀 Kubernetes Memory Monitoring - Usage Examples"
echo "================================================"

echo "📋 Command Line Flags (NEW! - override environment variables):"
echo ""

# Example 1: Show help
echo "1️⃣  Show help and available options:
    ./build/mgmt-monitoring --help
"

# Example 2: Monitor specific namespace (CLI flag)
echo "2️⃣  Monitor specific namespace (recommended):
    ./build/mgmt-monitoring --namespace=production
    ./build/mgmt-monitoring --namespace=kube-system
"

# Example 3: Monitor all namespaces explicitly (CLI flag)
echo "3️⃣  Monitor all namespaces explicitly:
    ./build/mgmt-monitoring --all-namespaces
"

# Example 4: Custom kubeconfig with CLI flag
echo "4️⃣  Custom kubeconfig file:
    ./build/mgmt-monitoring --kubeconfig=/path/to/config
    ./build/mgmt-monitoring --kubeconfig=/path/to/config --namespace=production
"

# Example 5: In-cluster configuration (CLI flag)
echo "5️⃣  In-cluster configuration (for running inside K8s):
    ./build/mgmt-monitoring --in-cluster --namespace=monitoring
"

# Example 6: Custom monitoring settings with CLI flags
echo "6️⃣  Custom monitoring settings (CLI flags override env vars):
    ./build/mgmt-monitoring \\
        --namespace=production \\
        --check-interval=1m \\
        --memory-threshold=2048 \\
        --memory-warning=75.0 \\
        --log-level=debug
"

echo ""
echo "🔧 Environment Variables (legacy support - lower priority):"
echo ""

# Example 7: Default configuration (looks for ~/.kube/config)
echo "7️⃣  Default configuration (uses ~/.kube/config):
    ./build/mgmt-monitoring
"

# Example 8: Custom kubeconfig via env var
echo "8️⃣  Custom kubeconfig file (env var):
    KUBECONFIG=/path/to/your/kubeconfig ./build/mgmt-monitoring
"

# Example 9: Monitor specific namespace via env var
echo "9️⃣  Monitor specific namespace (env var):
    NAMESPACE=kube-system ./build/mgmt-monitoring
"

# Example 10: All configuration options via env vars
echo "🔟 All configuration options (env vars):
    NAMESPACE=production \\
    KUBECONFIG=~/.kube/config \\
    IN_CLUSTER=false \\
    CHECK_INTERVAL=30s \\
    MEMORY_THRESHOLD_MB=1024 \\
    MEMORY_WARNING_PERCENT=80.0 \\
    LOG_LEVEL=info \\
    ./build/mgmt-monitoring
"

echo "
📋 Available Environment Variables:
   NAMESPACE              - Kubernetes namespace to monitor (default: default)
   KUBECONFIG            - Path to kubeconfig file (default: ~/.kube/config)  
   IN_CLUSTER            - Set to true when running inside K8s (default: false)
   CHECK_INTERVAL        - How often to check memory (default: 30s)
   MEMORY_THRESHOLD_MB   - Memory threshold in MB (default: 1024)
   MEMORY_WARNING_PERCENT - Warning threshold percentage (default: 80.0)
   LOG_LEVEL             - Logging level: debug, info, warn, error (default: info)
   LOG_FORMAT            - Log format: json, text (default: json)
"

echo "
🔍 What the application will show:
   • Cluster-wide memory summary statistics
   • Per-pod memory usage, requests, and limits
   • Identification of pods with high memory usage
   • Recommendations for pods without proper limits/requests
   • Proactive alerts for potential memory issues
"

echo "
🐳 Docker Usage:
   docker run --rm -v ~/.kube:/root/.kube:ro \\
     mgmt-monitoring:latest
"

echo "
☸️  Kubernetes Deployment:
   kubectl apply -f examples/kubernetes/
"
