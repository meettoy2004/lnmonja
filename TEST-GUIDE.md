# Local Testing Guide for lnmonja

This guide will help you test lnmonja locally without Docker.

## Prerequisites

- Go 1.21+ installed
- Binaries built (run `make build`)

## Quick Start (3 Terminal Windows)

### Terminal 1: Start the Server

```bash
# Create data directory
mkdir -p data logs

# Start server with local config
./lnmonja-server -config configs/server-local.yaml
```

You should see output like:
```
Starting lnmonja Server version=dev
Server listening on :8080 (HTTP)
Server listening on :9090 (gRPC)
WebSocket server listening on :3000
```

### Terminal 2: Start the Agent

```bash
# Wait for server to be ready, then start agent
./lnmonja-agent -config configs/agent-local.yaml
```

You should see output like:
```
Starting lnmonja Agent version=dev
Agent registered node_id=your-hostname session_id=xxx
Starting collector name=system interval=1s
Agent started successfully
```

### Terminal 3: Test with CLI

```bash
# Check server health
curl http://localhost:8080/health

# List connected nodes (wait ~30 seconds for agent to register)
curl http://localhost:8080/api/v1/nodes

# Query recent metrics
curl http://localhost:8080/api/v1/metrics?node=your-hostname&metric=system_cpu_usage_total&limit=10

# Real-time metrics stream (if WebSocket endpoint exists)
# You can use websocat or a browser console to connect to ws://localhost:3000
```

## Testing Scenarios

### 1. Basic Connectivity Test
```bash
# Server should be running
curl http://localhost:8080/health
# Expected: {"status":"ok"}

# Check if agent is connected
curl http://localhost:8080/api/v1/nodes
# Expected: JSON with your node listed
```

### 2. Metrics Collection Test
```bash
# Wait 10-15 seconds for metrics to be collected, then:
curl "http://localhost:8080/api/v1/metrics?limit=100" | jq .
```

You should see metrics like:
- `system_cpu_usage_total`
- `system_memory_used_bytes`
- `system_load1`, `system_load5`, `system_load15`
- `system_disk_usage_percent`
- `system_network_receive_bytes_total`

### 3. Generate CPU Load (to see metrics change)
```bash
# Generate some CPU load in a new terminal
yes > /dev/null
```

Then query CPU metrics:
```bash
curl "http://localhost:8080/api/v1/metrics?metric=system_cpu_usage_total" | jq .
```

Press Ctrl+C to stop the CPU load test.

### 4. Stress Test Memory
```bash
# In another terminal, allocate memory
stress --vm 1 --vm-bytes 1G --timeout 30s
```

Monitor memory metrics:
```bash
watch -n 1 'curl -s "http://localhost:8080/api/v1/metrics?metric=system_memory_used_bytes" | jq ".[-1]"'
```

## Manual Build (Alternative)

If you prefer not to use Make:

```bash
# Build server
go build -o lnmonja-server ./cmd/lnmonja-server

# Build agent (disable CGO on macOS)
CGO_ENABLED=0 go build -o lnmonja-agent ./cmd/lnmonja-agent

# Build CLI
go build -o lnmonja-cli ./cmd/lnmonja-cli
```

## Troubleshooting

### Server won't start
- **Port already in use**: Check if ports 8080, 9090, or 3000 are already in use:
  ```bash
  # Linux/macOS
  lsof -i :8080
  lsof -i :9090
  lsof -i :3000
  ```
- **Permission denied on data directory**: Ensure `./data` is writable

### Agent can't connect
- **Server not ready**: Wait 5-10 seconds after starting the server
- **Firewall blocking**: Check firewall settings for port 9090
- **Wrong server address**: Verify `server_address: "localhost:9090"` in agent config

### No metrics showing up
- **Agent not collecting**: Check agent logs for collector errors
- **Time range issue**: Query without time filters first
- **Agent not registered**: Check server logs for registration messages

### CGO build errors on macOS
```bash
CGO_ENABLED=0 go build -o lnmonja-agent ./cmd/lnmonja-agent
```

## Configuration Options

### Server Config (`configs/server-local.yaml`)
- **HTTP Port**: Change `server.http.port` (default: 8080)
- **gRPC Port**: Change `server.grpc.port` (default: 9090)
- **WebSocket Port**: Change `server.websocket.port` (default: 3000)
- **Data Path**: Change `storage.path` (default: ./data)
- **Log Level**: Change `logging.level` (debug/info/warn/error)

### Agent Config (`configs/agent-local.yaml`)
- **Server Address**: Change `agent.server_address` (default: localhost:9090)
- **Collection Interval**: Change `collectors.system.interval` (default: 1s)
- **Disable Collectors**: Set `collectors.*.enabled: false`
- **Log Level**: Change `logging.level` (debug/info/warn/error)

## Viewing Logs

### Server Logs
```bash
# Follow server logs (if logging to file)
tail -f logs/server.log

# Or just watch stdout from Terminal 1
```

### Agent Logs
```bash
# Follow agent logs (if logging to file)
tail -f logs/agent.log

# Or just watch stdout from Terminal 2
```

## Cleanup

```bash
# Stop server and agent (Ctrl+C in their terminals)

# Clean up data
rm -rf data/ logs/

# Remove binaries
make clean
```

## Next Steps

1. **Enable Authentication**: Set `authentication.enabled: true` in server config
2. **Enable Alerting**: Set `alerting.enabled: true` and create alert rules
3. **Add More Agents**: Run multiple agents with different node IDs
4. **Enable Container Monitoring**: Set `collectors.container.enabled: true`
5. **Build Dashboard**: The web dashboard can be found in `web/dashboard/`

## API Endpoints Reference

- `GET /health` - Server health check
- `GET /api/v1/nodes` - List connected nodes
- `GET /api/v1/metrics` - Query metrics
  - `?node=<node_id>` - Filter by node
  - `?metric=<metric_name>` - Filter by metric name
  - `?start=<timestamp>` - Start time (Unix timestamp)
  - `?end=<timestamp>` - End time (Unix timestamp)
  - `?limit=<n>` - Limit results
- `WS ws://localhost:3000/ws` - WebSocket for real-time metrics

## Performance Notes

- **Memory Usage**: Server uses ~50-100MB, Agent uses ~10-30MB
- **CPU Usage**: Minimal when idle, 1-5% per collector
- **Disk Usage**: Depends on retention period and number of metrics
- **Collection Overhead**: ~0.1% CPU per second of collection interval

Enjoy testing lnmonja! 🚀
