# Kubernetes Integration Guide

This guide covers deploying and integrating LnMonja with Kubernetes clusters for comprehensive container orchestration monitoring.

## 📋 Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Deployment Options](#deployment-options)
- [Installation](#installation)
- [Configuration](#configuration)
- [Monitoring Capabilities](#monitoring-capabilities)
- [Auto-Discovery](#auto-discovery)
- [Multi-Cluster Support](#multi-cluster-support)
- [Troubleshooting](#troubleshooting)

---

## Overview

LnMonja provides native Kubernetes monitoring through:
- **DaemonSet agents** on every node for system-level metrics
- **kube-state-metrics** integration for cluster state
- **Pod-level monitoring** with automatic discovery
- **Custom Resource** monitoring (CRDs)
- **Event collection** for cluster diagnostics

### What Gets Monitored

#### Node Level
- CPU, memory, disk, network usage per node
- Node conditions (Ready, DiskPressure, MemoryPressure, etc.)
- Kubelet metrics
- Node capacity and allocatable resources

#### Pod Level
- Container CPU and memory usage
- Restart counts and failure reasons
- Pod phase transitions
- Resource requests vs actual usage
- Volume usage

#### Cluster Level
- Deployment health and replicas
- StatefulSet status
- DaemonSet coverage
- Service endpoints
- Ingress status
- Persistent Volume Claims
- ConfigMaps and Secrets (metadata only)

#### Application Level
- Custom application metrics via annotations
- Service health checks
- Endpoint availability
- Request rates and latencies (if instrumented)

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Kubernetes Cluster                        │
│                                                              │
│  ┌────────────────────────────────────────────────────┐    │
│  │            LnMonja Server (Deployment)             │    │
│  │  - Receives metrics from agents                    │    │
│  │  - Stores in time-series DB                        │    │
│  │  - Evaluates alerts                                │    │
│  │  - Serves dashboard                                │    │
│  │  Exposed via: Service (ClusterIP) + Ingress        │    │
│  └────────────────────────────────────────────────────┘    │
│                                                              │
│  ┌────────────────────────────────────────────────────┐    │
│  │        LnMonja Agents (DaemonSet)                  │    │
│  │  Runs on every node:                               │    │
│  │  - Collects node metrics                           │    │
│  │  - Monitors pods on the node                       │    │
│  │  - Tracks containers                               │    │
│  │  - Sends to server via gRPC                        │    │
│  └────────────────────────────────────────────────────┘    │
│                                                              │
│  ┌────────────────────────────────────────────────────┐    │
│  │    LnMonja K8s Collector (Deployment)              │    │
│  │  - Queries Kubernetes API                          │    │
│  │  - Collects cluster-wide state                     │    │
│  │  - Watches for events                              │    │
│  │  - Auto-discovers resources                        │    │
│  └────────────────────────────────────────────────────┘    │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

---

## Deployment Options

### Option 1: Full Stack (Recommended)

Deploy all components within Kubernetes:
- LnMonja server as a Deployment
- Agents as a DaemonSet
- Dashboard as a Deployment with Ingress

**Pros:**
- Everything in one place
- Easy to manage with Kubernetes tools
- Scales with your cluster

**Cons:**
- Server monitoring itself (may miss cluster-wide failures)
- Resource overhead within cluster

### Option 2: External Server

Run server outside Kubernetes, agents inside:
- LnMonja server on dedicated VMs/cloud instances
- Agents as DaemonSet in K8s

**Pros:**
- Server survives cluster failures
- Can monitor multiple clusters from one server
- Better for multi-cluster setups

**Cons:**
- More complex networking setup
- Requires external infrastructure

### Option 3: Hybrid

Multiple servers for different environments:
- Production server external
- Dev/staging servers in-cluster

---

## Installation

### Prerequisites

- Kubernetes 1.20+ cluster
- `kubectl` configured
- Helm 3+ (for Helm installation)
- Persistent storage (for server data)

### Method 1: Helm Chart (Recommended)

```bash
# Add LnMonja Helm repository
helm repo add lnmonja https://charts.lnmonja.io
helm repo update

# Install with default values
helm install lnmonja lnmonja/lnmonja \
  --namespace monitoring \
  --create-namespace

# Or with custom values
helm install lnmonja lnmonja/lnmonja \
  --namespace monitoring \
  --create-namespace \
  --values custom-values.yaml
```

#### Example `custom-values.yaml`

```yaml
server:
  replicas: 2
  resources:
    requests:
      memory: "2Gi"
      cpu: "1000m"
    limits:
      memory: "4Gi"
      cpu: "2000m"

  persistence:
    enabled: true
    size: 50Gi
    storageClass: "fast-ssd"

  ingress:
    enabled: true
    hostname: lnmonja.example.com
    tls:
      enabled: true
      secretName: lnmonja-tls

agent:
  resources:
    requests:
      memory: "50Mi"
      cpu: "100m"
    limits:
      memory: "200Mi"
      cpu: "500m"

  # Enable Kubernetes-specific collectors
  collectors:
    kubernetes:
      enabled: true
    container:
      enabled: true

kubernetesCollector:
  enabled: true
  replicas: 1

alerting:
  enabled: true
  slack:
    enabled: true
    webhookUrl: "https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
```

### Method 2: kubectl (Manual)

```bash
# Create namespace
kubectl create namespace monitoring

# Deploy server
kubectl apply -f deploy/kubernetes/server-deployment.yaml
kubectl apply -f deploy/kubernetes/server-service.yaml

# Deploy agents
kubectl apply -f deploy/kubernetes/agent-daemonset.yaml

# Deploy K8s collector
kubectl apply -f deploy/kubernetes/k8s-collector-deployment.yaml

# Deploy dashboard
kubectl apply -f deploy/kubernetes/dashboard-deployment.yaml
kubectl apply -f deploy/kubernetes/dashboard-ingress.yaml
```

### Method 3: Kustomize

```bash
kubectl apply -k deploy/kubernetes/overlays/production
```

---

## Configuration

### Server Configuration

```yaml
# ConfigMap: lnmonja-server-config
apiVersion: v1
kind: ConfigMap
metadata:
  name: lnmonja-server-config
  namespace: monitoring
data:
  config.yaml: |
    server:
      grpc:
        address: "0.0.0.0"
        port: 9090
      http:
        address: "0.0.0.0"
        port: 8080

    storage:
      path: "/data"
      retention_period: "720h"  # 30 days

    alerting:
      enabled: true
      evaluation_interval: "10s"

    kubernetes:
      enabled: true
      in_cluster: true  # Use in-cluster config
```

### Agent Configuration

```yaml
# ConfigMap: lnmonja-agent-config
apiVersion: v1
kind: ConfigMap
metadata:
  name: lnmonja-agent-config
  namespace: monitoring
data:
  config.yaml: |
    agent:
      server_address: "lnmonja-server:9090"

    collectors:
      system:
        enabled: true
        interval: "1s"

      container:
        enabled: true
        runtime: "auto"  # Detects Docker/containerd
        interval: "2s"

      kubernetes:
        enabled: true
        interval: "5s"
        node_name: "${NODE_NAME}"  # Injected via fieldRef
```

### RBAC Configuration

LnMonja requires specific permissions:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: lnmonja-agent
  namespace: monitoring
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: lnmonja-agent
rules:
- apiGroups: [""]
  resources: ["nodes", "pods", "services", "endpoints", "namespaces"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["apps"]
  resources: ["deployments", "daemonsets", "statefulsets", "replicasets"]
  verbs: ["get", "list", "watch"]
- apiGroups: [""]
  resources: ["events"]
  verbs: ["get", "list", "watch", "create"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: lnmonja-agent
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: lnmonja-agent
subjects:
- kind: ServiceAccount
  name: lnmonja-agent
  namespace: monitoring
```

---

## Monitoring Capabilities

### Automatic Pod Discovery

LnMonja automatically discovers and monitors:

1. **All pods** across all namespaces (configurable)
2. **Container metrics** for each pod
3. **Pod labels and annotations** for grouping and filtering
4. **Resource usage** vs requests/limits

**Filtering by Namespace:**

```yaml
kubernetes:
  namespaces:
    include: ["production", "staging"]  # Only these
    exclude: ["kube-system"]             # Exclude these
```

**Filtering by Labels:**

```yaml
kubernetes:
  pod_filters:
    - label: "app=web"
    - label: "environment=production"
```

### Custom Metrics Scraping

Annotate pods to expose custom metrics:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: my-app
  annotations:
    lnmonja.io/scrape: "true"
    lnmonja.io/port: "8080"
    lnmonja.io/path: "/metrics"
    lnmonja.io/interval: "10s"
spec:
  containers:
  - name: app
    image: my-app:v1
    ports:
    - containerPort: 8080
```

### Event Collection

Monitor Kubernetes events for:
- Pod failures and restarts
- Node issues
- Deployment rollouts
- Scheduling problems
- Resource constraints

Events are automatically correlated with metrics for root cause analysis.

---

## Auto-Discovery

### Network Auto-Discovery

Discover services via:

**1. Service Discovery**
```yaml
kubernetes:
  service_discovery:
    enabled: true
    namespaces: ["default", "production"]
```

Automatically finds:
- All Services
- Endpoints behind services
- External IPs and Load Balancers

**2. DNS-based Discovery**
```yaml
auto_discovery:
  dns:
    enabled: true
    domains:
      - "*.svc.cluster.local"
```

**3. Label-based Discovery**
```yaml
auto_discovery:
  labels:
    monitor: "true"
    environment: "production"
```

### Application Discovery

Discover applications via:

**Prometheus Annotations**
```yaml
metadata:
  annotations:
    prometheus.io/scrape: "true"
    prometheus.io/port: "9090"
    prometheus.io/path: "/metrics"
```

LnMonja is compatible with Prometheus annotations.

---

## Multi-Cluster Support

Monitor multiple Kubernetes clusters from a single LnMonja server.

### Option 1: External Server

```
┌──────────────────┐
│  LnMonja Server  │ ← Central monitoring
│  (External VM)   │
└────────┬─────────┘
         │
    ┌────┴────┬────────────┐
    ▼         ▼            ▼
  Cluster1  Cluster2  Cluster3
  (Agents)  (Agents)  (Agents)
```

Each cluster's agents connect to the central server:

```yaml
# Cluster 1 agents
agent:
  server_address: "lnmonja.example.com:9090"
  tags:
    cluster: "production-us-east"
    region: "us-east-1"

# Cluster 2 agents
agent:
  server_address: "lnmonja.example.com:9090"
  tags:
    cluster: "production-eu-west"
    region: "eu-west-1"
```

### Option 2: Federation

Deploy a server in each cluster, federate to central:

```
       ┌────────────────────┐
       │  Central Server    │
       │  (Aggregation)     │
       └──────┬─────────────┘
              │
    ┌─────────┼─────────┐
    ▼         ▼         ▼
 Server1   Server2   Server3
    │         │         │
 Cluster1  Cluster2  Cluster3
```

---

## Alert Examples for Kubernetes

### Pod Restart Alert

```yaml
name: "High Pod Restarts"
metric: "kubernetes_pod_restarts_total"
condition: ">"
threshold: 5
duration: "5m"
severity: "warning"
labels:
  component: "kubernetes"
notification:
  slack: true
  message: "Pod {{pod}} in namespace {{namespace}} has restarted {{value}} times"
```

### Node Pressure Alert

```yaml
name: "Node Memory Pressure"
metric: "kubernetes_node_memory_pressure"
condition: "=="
threshold: 1
severity: "critical"
notification:
  pagerduty: true
  slack: true
remediation:
  script: "/scripts/drain-and-cordon-node.sh"
```

### Deployment Unhealthy

```yaml
name: "Deployment Not Ready"
metric: "kubernetes_deployment_replicas_unavailable"
condition: ">"
threshold: 0
duration: "2m"
severity: "critical"
notification:
  slack: true
  jira: true
```

---

## Troubleshooting

### Agents Not Connecting

**Check agent logs:**
```bash
kubectl logs -n monitoring -l app=lnmonja-agent --tail=100
```

**Common issues:**
1. **Network policies** blocking gRPC (port 9090)
2. **Service name resolution** failing
3. **RBAC permissions** missing

**Fix:**
```bash
# Test connectivity
kubectl exec -n monitoring deploy/lnmonja-agent -- \
  nc -zv lnmonja-server 9090

# Check RBAC
kubectl auth can-i get pods --as=system:serviceaccount:monitoring:lnmonja-agent
```

### Missing Metrics

**Check collector status:**
```bash
kubectl exec -n monitoring -it lnmonja-agent-xxxxx -- /lnmonja-cli status
```

**Enable debug logging:**
```yaml
logging:
  level: "debug"
```

### High Resource Usage

**Agent consuming too much CPU/memory:**

1. **Reduce collection intervals:**
```yaml
collectors:
  system:
    interval: "5s"  # Instead of 1s
  container:
    interval: "10s"  # Instead of 2s
```

2. **Limit monitored namespaces:**
```yaml
kubernetes:
  namespaces:
    include: ["production"]  # Only monitor specific namespaces
```

3. **Adjust resource limits:**
```yaml
resources:
  limits:
    memory: "500Mi"  # Increase if needed
    cpu: "1000m"
```

---

## Best Practices

1. **Use namespace filtering** to reduce overhead in large clusters
2. **Enable persistent storage** for the server to survive restarts
3. **Set appropriate resource requests/limits** for agents
4. **Use taints/tolerations** to ensure agents run on all nodes
5. **Configure backup** for server data
6. **Use Ingress with TLS** for dashboard access
7. **Enable RBAC** and restrict permissions to minimum required
8. **Monitor the monitor** - set up alerts for LnMonja itself
9. **Use labels consistently** for better filtering and grouping
10. **Test in staging** before deploying to production

---

## Next Steps

- [Configure Alerts](docs/alerts.md)
- [Set Up Notifications](docs/notifications.md)
- [API Reference](docs/api.md)
- [Grafana Integration](docs/grafana.md)

---

**Questions?** Check our [FAQ](docs/faq.md) or join the [community Slack](https://slack.lnmonja.io).
