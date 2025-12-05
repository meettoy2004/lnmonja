#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

# Default values
SERVER_VERSION="v0.1.0"
INSTALL_DIR="/opt/lnmonja"
CONFIG_DIR="/etc/lnmonja"
DATA_DIR="/var/lib/lnmonja"
LOG_DIR="/var/log/lnmonja"
SERVICE_USER="lnmonja"

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --version)
            SERVER_VERSION="$2"
            shift 2
            ;;
        --install-dir)
            INSTALL_DIR="$2"
            shift 2
            ;;
        --config-dir)
            CONFIG_DIR="$2"
            shift 2
            ;;
        *)
            warn "Unknown argument: $1"
            shift
            ;;
    esac
done

# Check if running as root
if [[ $EUID -ne 0 ]]; then
    error "This script must be run as root"
    exit 1
fi

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case $ARCH in
    x86_64) ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    armv7l) ARCH="arm" ;;
    *) error "Unsupported architecture: $ARCH"; exit 1 ;;
esac

log "Installing lnmonja Server $SERVER_VERSION on $OS/$ARCH..."

# Create user if doesn't exist
if ! id "$SERVICE_USER" &>/dev/null; then
    log "Creating user: $SERVICE_USER"
    useradd -r -s /bin/false -d "$DATA_DIR" "$SERVICE_USER"
fi

# Create directories
log "Creating directories..."
mkdir -p "$INSTALL_DIR" "$CONFIG_DIR" "$DATA_DIR" "$LOG_DIR"
mkdir -p "$CONFIG_DIR/certs" "$CONFIG_DIR/alert-rules"

# Download binary
BINARY_URL="https://github.com/yourusername/lnmonja/releases/download/$SERVER_VERSION/lnmonja-server-$OS-$ARCH"
log "Downloading binary from: $BINARY_URL"
curl -L -o "$INSTALL_DIR/lnmonja-server" "$BINARY_URL"
chmod +x "$INSTALL_DIR/lnmonja-server"

# Create config file
log "Creating configuration..."
cat > "$CONFIG_DIR/config.yaml" << EOF
server:
  grpc:
    address: "0.0.0.0"
    port: 9090
  http:
    address: "0.0.0.0"
    port: 8080
  websocket:
    address: "0.0.0.0"
    port: 3000

storage:
  path: "$DATA_DIR"
  retention_days: 30

logging:
  level: "info"
  path: "$LOG_DIR/server.log"
EOF

# Copy default alert rules
log "Copying default alert rules..."
cp -r "$(dirname "$0")/../configs/alert-rules/"* "$CONFIG_DIR/alert-rules/"

# Generate self-signed certificates if not exist
if [[ ! -f "$CONFIG_DIR/certs/server.crt" ]]; then
    log "Generating self-signed certificates..."
    openssl req -x509 -newkey rsa:4096 -keyout "$CONFIG_DIR/certs/server.key" \
        -out "$CONFIG_DIR/certs/server.crt" -days 365 -nodes \
        -subj "/CN=lnmonja-server" 2>/dev/null
fi

# Create systemd service
log "Creating systemd service..."
cat > /etc/systemd/system/lnmonja-server.service << EOF
[Unit]
Description=lnmonja Monitoring Server
After=network.target

[Service]
Type=simple
User=$SERVICE_USER
Group=$SERVICE_USER
ExecStart=$INSTALL_DIR/lnmonja-server --config $CONFIG_DIR/config.yaml
Restart=always
RestartSec=5
LimitNOFILE=65536
LimitNPROC=65536
Environment="GOMAXPROCS=\$(nproc)"

# Security
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ReadWritePaths=$DATA_DIR $LOG_DIR

[Install]
WantedBy=multi-user.target
EOF

# Set permissions
log "Setting permissions..."
chown -R "$SERVICE_USER:$SERVICE_USER" "$INSTALL_DIR" "$CONFIG_DIR" "$DATA_DIR" "$LOG_DIR"
chmod 600 "$CONFIG_DIR/certs/server.key"

# Reload systemd and enable service
log "Enabling service..."
systemctl daemon-reload
systemctl enable lnmonja-server

# Start service
log "Starting service..."
systemctl start lnmonja-server

# Wait for service to start
sleep 3

# Check if service is running
if systemctl is-active --quiet lnmonja-server; then
    log "lnmonja Server installed successfully!"
    log "Dashboard: http://localhost:8080"
    log "API: http://localhost:8080/api/v1"
    log "Logs: journalctl -u lnmonja-server -f"
else
    error "Service failed to start. Check logs: journalctl -u lnmonja-server"
    exit 1
fi