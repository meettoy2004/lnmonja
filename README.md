LnMonja - Real-Time Monitoring System

LnMonja is a zero-config, high-resolution monitoring system with 1-second granularity, built-in anomaly detection, and real-time dashboards.

## Features
- **Real-time monitoring**: 1-second granularity metrics
- **Zero configuration**: Auto-discovers services and applications
- **Built-in anomaly detection**: Machine learning powered
- **Lightweight agents**: Less than 10MB memory footprint
- **Kubernetes native**: Full K8s monitoring out of the box
- **No dependencies**: Runs standalone, no external databases needed

## Quick Start

### Building from Source

**All platforms:**
```bash
make build
```

**macOS users:** If you encounter CGO errors, use:
```bash
make clean && make build
```

**Manual build (if you don't want to use Make):**
```bash
# Build server
go build -o lnmonja-server ./cmd/lnmonja-server

# Build agent (disable CGO on macOS)
CGO_ENABLED=0 go build -o lnmonja-agent ./cmd/lnmonja-agent

# Build CLI
go build -o lnmonja-cli ./cmd/lnmonja-cli
```

### Using Docker Compose
```bash
docker-compose up -d
# Access dashboard at http://localhost:8080