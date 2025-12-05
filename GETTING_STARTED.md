# LnMonja - Getting Started

This document explains how to build and run LnMonja on your local system.

## What's Been Implemented

All missing components have been implemented:

### Core Components ✅
- **Time-Series Database** (`internal/storage/tsdb.go`)
- **Storage Layer** (`internal/storage/badger_store.go`) with BadgerDB
- **Retention Manager** (`internal/storage/retention.go`)
- **Compression Engine** (`internal/storage/compression.go`)

### Server Components ✅
- **Main Server** (`internal/server/server.go`)
- **gRPC Server** (`internal/server/grpc_server.go`)
- **Node Manager** (`internal/server/node_manager.go`)
- **Alert Engine** (`internal/server/alert_engine.go`)
- **WebSocket Server** (`internal/server/api/websocket.go`)

### Machine Learning ✅
- **Anomaly Detection** (`internal/ml/anomaly/`)
  - EWMA-based detector
  - Isolation Forest algorithm
  - Multi-detector ensemble
- **Forecasting** (`internal/ml/forecasting/prophet.go`)

### Protocol & Utilities ✅
- **Protocol Definitions** (`pkg/protocol/protocol.go`)
- **Configuration** (`pkg/utils/config.go`)
- **Crypto utilities** (`pkg/utils/crypto.go`)
- **Data Models** (`internal/models/metric.go`)

## Prerequisites

- Go 1.21 or higher
- Docker and Docker Compose (for containerized deployment)
- Node.js 18+ (for building the web dashboard)

## Quick Start

### 1. Download Dependencies

```bash
# Clone the repository if you haven't already
cd /path/to/lnmonja

# Download Go dependencies
go mod tidy

# Verify dependencies are downloaded
go mod verify
```

### 2. Build the Binaries

```bash
# Build all binaries
make build

# Or build individually:
go build -o lnmonja-server ./cmd/lnmonja-server
go build -o lnmonja-agent ./cmd/lnmonja-agent
go build -o lnmonja-cli ./cmd/lnmonja-cli
```

### 3. Build the Web Dashboard

```bash
cd web/dashboard
npm install
npm run build
cd ../..
```

### 4. Run with Docker Compose (Recommended)

```bash
# Build Docker images
docker-compose build

# Start all services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down
```

### 5. Run Locally (Development)

**Terminal 1 - Start the Server:**
```bash
./lnmonja-server --config configs/server-config.yaml
```

**Terminal 2 - Start an Agent:**
```bash
./lnmonja-agent --config configs/agent-config.yaml
```

**Terminal 3 - Use the CLI:**
```bash
./lnmonja-cli status
./lnmonja-cli nodes list
```

## Access the Dashboard

Once running, access the system at:

- **Dashboard**: http://localhost:80
- **HTTP API**: http://localhost:8080
- **WebSocket**: ws://localhost:3000
- **gRPC**: localhost:9090

## Configuration

### Server Configuration

Edit `configs/server-config.yaml`:

```yaml
server:
  grpc:
    port: 9090
  http:
    port: 8080
  websocket:
    port: 3000

storage:
  path: ./data
  retention_period: 720h  # 30 days
  compression: true

alerting:
  enabled: true
  rules_path: ./configs/alert-rules
```

### Agent Configuration

Edit `configs/agent-config.yaml`:

```yaml
agent:
  server_address: localhost:9090
  batch_size: 1000
  max_batch_wait: 1s

collectors:
  system:
    enabled: true
    interval: 1s
  container:
    enabled: true
    runtime: docker
```

## Testing the System

### 1. Check Server Health

```bash
curl http://localhost:8080/health
```

### 2. List Nodes

```bash
./lnmonja-cli nodes list
```

### 3. Query Metrics

```bash
./lnmonja-cli metrics --query "system_cpu_usage" --from "1h"
```

### 4. View Alerts

```bash
./lnmonja-cli alerts list
```

## Architecture Overview

```
┌─────────────────┐         ┌──────────────────┐
│   lnmonja-agent │────gRPC─▶│  lnmonja-server  │
│   (Collectors)  │         │   (Aggregator)   │
└─────────────────┘         └──────────────────┘
                                     │
                    ┌────────────────┼────────────────┐
                    │                │                │
              ┌─────▼─────┐   ┌─────▼─────┐   ┌─────▼─────┐
              │  BadgerDB  │   │   Alerts  │   │ WebSocket │
              │  (Storage) │   │  (Engine) │   │   (Live)  │
              └────────────┘   └───────────┘   └─────┬─────┘
                                                      │
                                               ┌──────▼──────┐
                                               │  Dashboard  │
                                               │   (Web UI)  │
                                               └─────────────┘
```

## Features

✅ **Real-time Monitoring** - 1-second granularity metrics
✅ **Auto-Discovery** - Zero-config service detection
✅ **Anomaly Detection** - ML-powered anomaly detection
✅ **Alerting** - Flexible alert rules with multiple notification channels
✅ **Time-Series Storage** - Efficient BadgerDB-based storage with compression
✅ **WebSocket Streaming** - Real-time dashboard updates
✅ **Container Monitoring** - Docker/containerd support
✅ **Kubernetes Support** - Native K8s monitoring

## Troubleshooting

### Issue: Cannot connect to server

**Solution**: Check if the server is running and firewall allows the ports:
```bash
netstat -an | grep -E "8080|9090|3000"
```

### Issue: No metrics appearing

**Solution**: Check agent logs and ensure it can connect to the server:
```bash
docker-compose logs lnmonja-agent
# Or if running locally:
./lnmonja-agent --debug
```

### Issue: Permission denied for eBPF collector

**Solution**: Run agent with required capabilities:
```bash
sudo setcap cap_sys_admin,cap_net_admin=eip ./lnmonja-agent
# Or run with sudo for development
```

## Development

### Run Tests

```bash
go test ./...
```

### Build for Production

```bash
CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o lnmonja-server ./cmd/lnmonja-server
```

### Enable Debug Logging

```bash
export LNMONJA_LOG_LEVEL=debug
./lnmonja-server
```

## What's Next?

1. ✅ All core components implemented
2. ✅ Storage layer complete
3. ✅ ML/AI features implemented
4. 🔄 Build and test locally (you can do this now!)
5. 📝 Add more alert rules as needed
6. 🎨 Customize the dashboard
7. 📊 Add custom collectors

## Support

- Documentation: `./docs/`
- Configuration examples: `./configs/`
- Alert rules: `./configs/alert-rules/`

## License

See LICENSE file for details.
