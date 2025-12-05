#!/bin/bash
set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log() { echo -e "${GREEN}[INFO]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1" >&2; }

# Defaults
SERVER_ADDR="localhost:9090"
NODE_ID=$(hostname)
INSTALL_DIR="/opt/lnmonja"
CONFIG_DIR="/etc/lnmonja"
SERVICE_USER="lnmonja"

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --server)
            SERVER_ADDR="$2"
            shift 2
            ;;
        --node-id)
            NODE_ID="$2"
            shift 2
            ;;
        --install-dir)
            INSTALL_DIR="$2"
            shift 2
            ;;
        *)
            warn "Unknown argument: $1"
            shift
            ;;
    esac
done

# Check root
if [[ $EUID -ne 0 ]]; then
    error "Must be run as root"
    exit 1
fi

# Detect OS/Arch
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case $ARCH in
    x86_64) ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    armv7l) ARCH="arm" ;;
    *) error "Unsupported arch: $ARCH"; exit 1 ;;
esac

log "Installing lnmonja Agent on $NODE_ID..."

# Create user
if ! id "$SERVICE_USER" &>/dev/null; then
    useradd -r -s /bin/false -d /var/lib/lnmonja "$SERVICE_USER"
fi

# Create dirs
mkdir -p "$INSTALL_DIR" "$CONFIG_DIR" "/var/lib/lnmonja" "/var/log/lnmonja"

# Download
VERSION="v0.1.0"
URL="https://github.com/yourusername/lnmonja/releases/download/$VERSION/lnmonja-agent-$OS-$ARCH"
log "Downloading agent..."
curl -L -o "$INSTALL_DIR/lnmonja-agent" "$URL"
chmod +x "$INSTALL_DIR/lnmonja-agent"

# Config
cat > "$CONFIG_DIR/config.yaml" << EOF
agent:
  node_id: "$NODE_ID"
  server_address: "$SERVER_ADDR"
  
collectors:
  system:
    enabled: true
    interval: 1s
  container:
    enabled: true
    docker_socket: "/var/run/docker.sock"
    
logging:
  level: info
  path: /var/log/lnmonja/agent.log
EOF

# Systemd service
cat > /etc/systemd/system/lnmonja-agent.service << EOF
[Unit]
Description=lnmonja Agent
After=network.target docker.service

[Service]
Type=simple
User=$SERVICE_USER
Group=$SERVICE_USER
ExecStart=$INSTALL_DIR/lnmonja-agent --config $CONFIG_DIR/config.yaml
Restart=always
RestartSec=5
LimitNOFILE=65536
LimitNPROC=65536

# Capabilities for monitoring
CapabilityBoundingSet=CAP_SYS_PTRACE CAP_DAC_READ_SEARCH
AmbientCapabilities=CAP_SYS_PTRACE CAP_DAC_READ_SEARCH

# Security
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ReadWritePaths=/var/lib/lnmonja /var/log/lnmonja

[Install]
WantedBy=multi-user.target
EOF

# Permissions
chown -R "$SERVICE_USER:$SERVICE_USER" "$INSTALL_DIR" "$CONFIG_DIR" /var/lib/lnmonja /var/log/lnmonja

# Enable and start
systemctl daemon-reload
systemctl enable lnmonja-agent
systemctl start lnmonja-agent

sleep 2

if systemctl is-active --quiet lnmonja-agent; then
    log "Agent installed successfully!"
    log "Status: systemctl status lnmonja-agent"
    log "Logs: journalctl -u lnmonja-agent -f"
else
    error "Agent failed to start"
    journalctl -u lnmonja-agent --no-pager -n 20
    exit 1
fi