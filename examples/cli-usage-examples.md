# CLI Usage Examples

## ✨ **New Command Line Interface**

The Kubernetes Memory Monitoring tool now supports comprehensive command line flags similar to `kubectl`.

## 🎯 **Basic Usage**

### Show Help
```bash
./build/k8s-memory-watch --help
```

### Monitor Specific Namespace (Most Common)
```bash
# Monitor production namespace only
./build/k8s-memory-watch --namespace=production

# Monitor kube-system namespace
./build/k8s-memory-watch --namespace=kube-system

# Monitor default namespace
./build/k8s-memory-watch --namespace=default
```

### Monitor All Namespaces
```bash
# Explicitly monitor all namespaces
./build/k8s-memory-watch --all-namespaces

# Default behavior (when no namespace specified) is also all namespaces
./build/k8s-memory-watch
```

## 🔧 **Advanced Configuration**

### Custom Kubeconfig
```bash
# Use specific kubeconfig file
./build/k8s-memory-watch --kubeconfig=/path/to/your/kubeconfig

# Combine with namespace filtering
./build/k8s-memory-watch --kubeconfig=/path/to/config --namespace=production
```

### In-Cluster Configuration
```bash
# For running inside Kubernetes
./build/k8s-memory-watch --in-cluster --namespace=monitoring
```

### Custom Monitoring Parameters
```bash
# Advanced configuration
./build/k8s-memory-watch \
    --namespace=production \
    --check-interval=1m \
    --memory-threshold=2048 \
    --memory-warning=75.0 \
    --log-level=debug
```

## 📋 **Available Flags**

| Flag | Type | Description | Example |
|------|------|-------------|---------|
| `--namespace` | string | Monitor specific namespace | `--namespace=production` |
| `--all-namespaces` | bool | Monitor all namespaces | `--all-namespaces` |
| `--kubeconfig` | string | Path to kubeconfig | `--kubeconfig=/path/to/config` |
| `--in-cluster` | bool | Use in-cluster config | `--in-cluster` |
| `--check-interval` | duration | Check interval | `--check-interval=1m` |
| `--memory-threshold` | int | Memory threshold (MB) | `--memory-threshold=2048` |
| `--memory-warning` | float | Warning percentage | `--memory-warning=75.0` |
| `--log-level` | string | Logging level | `--log-level=debug` |
| `--help` | bool | Show help | `--help` |

## ⚠️ **Important Notes**

### Precedence Rules
- **CLI flags** override **environment variables**
- Use CLI flags for the best experience

### Mutually Exclusive Flags
```bash
# ❌ This will fail:
./build/k8s-memory-watch --namespace=prod --all-namespaces

# ✅ Use one or the other:
./build/k8s-memory-watch --namespace=prod
./build/k8s-memory-watch --all-namespaces
```

### Default Behavior
- When **no namespace specified**: monitors **all namespaces**
- When **specific namespace specified**: monitors **only that namespace**

## 📊 **Expected Output**

### With Specific Namespace
```bash
./build/k8s-memory-watch --namespace=production
```
Will show only pods from the `production` namespace:
```
=== Detailed Pod Memory Information ===

Namespace: production
--------------------------------------------------------------------------------
  🟢 app-backend-v2-abc123 | Usage: 320.8 MB | Request: 500.0 MB (64.2%) | Limit: 800.0 MB (40.1%)
  🟢 app-frontend-def456 | Usage: 85.3 MB | Request: 100.0 MB (85.3%) | Limit: 256.0 MB (33.3%)
```

### With All Namespaces
```bash
./build/k8s-memory-watch --all-namespaces
```
Will show pods from all namespaces organized by namespace.

## 🔄 **Migration from Environment Variables**

### Before (Environment Variables)
```bash
NAMESPACE=production KUBECONFIG=/path/to/config ./build/k8s-memory-watch
```

### Now (CLI Flags - Recommended)
```bash
./build/k8s-memory-watch --namespace=production --kubeconfig=/path/to/config
```

Both work, but CLI flags take precedence and provide better UX.

## 🚀 **Production Examples**

### Development Environment
```bash
./build/k8s-memory-watch --namespace=dev --log-level=debug
```

### Staging Environment
```bash
./build/k8s-memory-watch --namespace=staging --check-interval=2m
```

### Production Monitoring
```bash
./build/k8s-memory-watch --namespace=production --memory-warning=85.0
```

### Full Cluster Overview
```bash
./build/k8s-memory-watch --all-namespaces --log-level=info
```

### In-Cluster Deployment
```bash
./build/k8s-memory-watch --in-cluster --namespace=monitoring --check-interval=30s
```

## 📊 **Output Symbols**

The tool uses visual indicators to show pod status at a glance:

- 🟢 Pod Running and Ready
- 🔴 Pod with issues (high memory usage or not Ready)
- 🟡 Pod Pending 
- ⚪ No memory metrics available (pod starting up or metrics not ready)

This provides a **professional CLI experience** matching Kubernetes tooling standards! 🎯
