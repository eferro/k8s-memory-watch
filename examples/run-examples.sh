#!/bin/bash

# Examples of how to run the Kubernetes Memory Monitoring application

echo "🚀 Kubernetes Memory Monitoring - Usage Examples"
echo "================================================"

# Example 1: Default configuration (looks for ~/.kube/config)
echo "
1️⃣  Default configuration (uses ~/.kube/config):
    ./build/mgmt-monitoring
"

# Example 2: Custom kubeconfig
echo "
2️⃣  Custom kubeconfig file:
    KUBECONFIG=/path/to/your/kubeconfig ./build/mgmt-monitoring
"

# Example 3: In-cluster configuration (when running inside Kubernetes)
echo "
3️⃣  In-cluster configuration (for running inside K8s):
    IN_CLUSTER=true ./build/mgmt-monitoring
"

# Example 4: Custom monitoring configuration
echo "
4️⃣  Custom monitoring settings:
    CHECK_INTERVAL=1m \\
    MEMORY_THRESHOLD_MB=2048 \\
    MEMORY_WARNING_PERCENT=75.0 \\
    LOG_LEVEL=debug \\
    ./build/mgmt-monitoring
"

# Example 5: Monitor specific namespace
echo "
5️⃣  Monitor specific namespace:
    NAMESPACE=kube-system ./build/mgmt-monitoring
"

# Example 6: All configuration options
echo "
6️⃣  All configuration options:
    NAMESPACE=default \\
    KUBECONFIG=~/.kube/config \\
    IN_CLUSTER=false \\
    CHECK_INTERVAL=30s \\
    MEMORY_THRESHOLD_MB=1024 \\
    MEMORY_WARNING_PERCENT=80.0 \\
    LOG_LEVEL=info \\
    LOG_FORMAT=json \\
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
