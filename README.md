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

### Testing Locally

For detailed local testing instructions without Docker, see [TEST-GUIDE.md](TEST-GUIDE.md).

**Quick test:**
```bash
# Terminal 1: Start server
./lnmonja-server -config configs/server-local.yaml

# Terminal 2: Start agent
./lnmonja-agent -config configs/agent-local.yaml

# Terminal 3: Test API
curl http://localhost:8080/health
```

### Web Dashboard

LnMonja includes a modern web dashboard for managing nodes, viewing metrics, and configuring alerts.

**Start the dashboard:**
```bash
cd web/dashboard
npm install
cp .env.example .env
npm run dev
```

Access the dashboard at `http://localhost:5173`

Features:
- 📊 Real-time metrics visualization
- 🖥️ Node and agent management
- 🔔 Alert configuration and history
- ⚙️ Settings and configuration

For more details, see [web/dashboard/README.md](web/dashboard/README.md)

### Using Docker Compose
```bash
docker-compose up -d
# Access dashboard at http://localhost:80