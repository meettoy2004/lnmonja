# LnMonja Deployment Guide

Complete guide for deploying LnMonja in various environments from development to production.

## 📋 Table of Contents

- [Quick Start](#quick-start)
- [Production Deployment](#production-deployment)
- [Kubernetes Deployment](#kubernetes-deployment)
- [High Availability Setup](#high-availability-setup)
- [Cloud Deployments](#cloud-deployments)
- [Security Hardening](#security-hardening)
- [Performance Tuning](#performance-tuning)
- [Backup and Recovery](#backup-and-recovery)

---

## Quick Start

### Docker Compose (Development/Testing)

```bash
# Clone the repository
git clone https://github.com/meettoy2004/lnmonja.git
cd lnmonja

# Start all services
docker-compose up -d

# Access points:
# - Server API: http://localhost:8080
# - Dashboard: http://localhost:80
# - gRPC: localhost:9090
```

**Services included:**
- LnMonja Server
- LnMonja Agent (monitoring the host)
- Web Dashboard (Nginx)
- All dependencies

### Binary Installation (Local Testing)

```bash
# Build binaries
make build

# Start server
./lnmonja-server -config configs/server-local.yaml

# Start agent
./lnmonja-agent -config configs/agent-local.yaml

# Start dashboard
cd web/dashboard && npm install && npm run dev
```

---

## Production Deployment

### System Requirements

#### Server Requirements

**Minimum (100-1,000 devices):**
- CPU: 2 cores
- RAM: 4 GB
- Disk: 50 GB SSD
- Network: 100 Mbps

**Recommended (1,000-10,000 devices):**
- CPU: 8 cores
- RAM: 16 GB
- Disk: 200 GB SSD
- Network: 1 Gbps

**Enterprise (10,000+ devices):**
- CPU: 16+ cores
- RAM: 32+ GB
- Disk: 500 GB+ NVMe SSD
- Network: 10 Gbps

#### Agent Requirements

- CPU: 0.1-0.5 cores (minimal impact)
- RAM: 10-30 MB
- Disk: <10 MB
- Network: Minimal (1-10 KB/s per agent)

### Installation Methods

#### Method 1: Systemd Service (Recommended)

**1. Install Server:**

```bash
# Download binary
wget https://github.com/meettoy2004/lnmonja/releases/latest/download/lnmonja-server-linux-amd64
chmod +x lnmonja-server-linux-amd64
sudo mv lnmonja-server-linux-amd64 /usr/local/bin/lnmonja-server

# Create user
sudo useradd --system --no-create-home --shell /bin/false lnmonja

# Create directories
sudo mkdir -p /etc/lnmonja /var/lib/lnmonja /var/log/lnmonja
sudo chown lnmonja:lnmonja /var/lib/lnmonja /var/log/lnmonja

# Copy configuration
sudo cp configs/server-config.yaml /etc/lnmonja/config.yaml
sudo chown lnmonja:lnmonja /etc/lnmonja/config.yaml

# Create systemd service
sudo cat > /etc/systemd/system/lnmonja-server.service << 'EOF'
[Unit]
Description=LnMonja Monitoring Server
After=network.target

[Service]
Type=simple
User=lnmonja
Group=lnmonja
ExecStart=/usr/local/bin/lnmonja-server -config /etc/lnmonja/config.yaml
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=lnmonja-server

# Security
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/lnmonja /var/log/lnmonja

[Install]
WantedBy=multi-user.target
EOF

# Start service
sudo systemctl daemon-reload
sudo systemctl enable lnmonja-server
sudo systemctl start lnmonja-server
sudo systemctl status lnmonja-server
```

**2. Install Agent:**

```bash
# Download binary
wget https://github.com/meettoy2004/lnmonja/releases/latest/download/lnmonja-agent-linux-amd64
chmod +x lnmonja-agent-linux-amd64
sudo mv lnmonja-agent-linux-amd64 /usr/local/bin/lnmonja-agent

# Create configuration
sudo cat > /etc/lnmonja/agent-config.yaml << 'EOF'
agent:
  server_address: "your-server:9090"
  node_id: ""  # Auto-detected

collectors:
  system:
    enabled: true
    interval: "1s"
  process:
    enabled: true
  container:
    enabled: true

logging:
  level: "info"
EOF

# Create systemd service
sudo cat > /etc/systemd/system/lnmonja-agent.service << 'EOF'
[Unit]
Description=LnMonja Monitoring Agent
After=network.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/lnmonja-agent -config /etc/lnmonja/agent-config.yaml
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

# Start service
sudo systemctl daemon-reload
sudo systemctl enable lnmonja-agent
sudo systemctl start lnmonja-agent
```

#### Method 2: Docker

**Server:**
```bash
docker run -d \
  --name lnmonja-server \
  -p 8080:8080 \
  -p 9090:9090 \
  -p 3000:3000 \
  -v /opt/lnmonja/data:/data \
  -v /opt/lnmonja/config:/etc/lnmonja \
  lnmonja/server:latest
```

**Agent:**
```bash
docker run -d \
  --name lnmonja-agent \
  --privileged \
  --network host \
  --pid host \
  -v /:/host:ro \
  -v /var/run/docker.sock:/var/run/docker.sock:ro \
  -e SERVER_ADDRESS=server:9090 \
  lnmonja/agent:latest
```

---

## Kubernetes Deployment

See [KUBERNETES.md](KUBERNETES.md) for detailed Kubernetes deployment instructions.

**Quick deployment:**

```bash
# Using Helm
helm repo add lnmonja https://charts.lnmonja.io
helm install lnmonja lnmonja/lnmonja --namespace monitoring --create-namespace

# Using kubectl
kubectl apply -f deploy/kubernetes/
```

---

## High Availability Setup

### Architecture

```
              ┌─────────────┐
              │Load Balancer│
              └──────┬──────┘
                     │
        ┌────────────┼────────────┐
        ▼            ▼            ▼
  ┌─────────┐  ┌─────────┐  ┌─────────┐
  │Server 1 │  │Server 2 │  │Server 3 │
  │(Active) │  │(Active) │  │(Active) │
  └────┬────┘  └────┬────┘  └────┬────┘
       │            │            │
       └────────────┴────────────┘
                    │
              ┌─────────────┐
              │Shared Storage│
              │   (NFS/S3)  │
              └─────────────┘
```

### Configuration

**1. Load Balancer (HAProxy):**

```bash
# /etc/haproxy/haproxy.cfg
frontend lnmonja_grpc
    bind *:9090
    mode tcp
    default_backend lnmonja_servers_grpc

backend lnmonja_servers_grpc
    mode tcp
    balance roundrobin
    server server1 192.168.1.10:9090 check
    server server2 192.168.1.11:9090 check
    server server3 192.168.1.12:9090 check

frontend lnmonja_http
    bind *:8080
    mode http
    default_backend lnmonja_servers_http

backend lnmonja_servers_http
    mode http
    balance roundrobin
    option httpchk GET /health
    server server1 192.168.1.10:8080 check
    server server2 192.168.1.11:8080 check
    server server3 192.168.1.12:8080 check
```

**2. Shared Storage:**

```yaml
# Server config
storage:
  path: "/mnt/shared/lnmonja"  # NFS mount
  sync_writes: true
```

**3. Database Clustering (Roadmap):**

Future versions will support:
- Raft consensus for leader election
- Data replication across nodes
- Automatic failover

---

## Cloud Deployments

### AWS

**Using EC2:**

```bash
# Launch EC2 instance (t3.xlarge recommended)
# Amazon Linux 2 or Ubuntu 20.04+

# Install
wget https://github.com/meettoy2004/lnmonja/releases/latest/download/lnmonja-server-linux-amd64
chmod +x lnmonja-server-linux-amd64
sudo mv lnmonja-server-linux-amd64 /usr/local/bin/lnmonja-server

# Configure
sudo mkdir -p /etc/lnmonja
sudo tee /etc/lnmonja/config.yaml > /dev/null <<EOF
server:
  grpc:
    address: "0.0.0.0"
    port: 9090
  http:
    address: "0.0.0.0"
    port: 8080

storage:
  path: "/mnt/ebs/lnmonja"
  retention_period: "720h"
EOF

# Start
sudo /usr/local/bin/lnmonja-server -config /etc/lnmonja/config.yaml
```

**Using ECS/Fargate:**

```json
{
  "family": "lnmonja-server",
  "containerDefinitions": [
    {
      "name": "server",
      "image": "lnmonja/server:latest",
      "portMappings": [
        {"containerPort": 8080, "protocol": "tcp"},
        {"containerPort": 9090, "protocol": "tcp"}
      ],
      "mountPoints": [
        {
          "sourceVolume": "data",
          "containerPath": "/data"
        }
      ]
    }
  ],
  "volumes": [
    {
      "name": "data",
      "efsVolumeConfiguration": {
        "fileSystemId": "fs-xxxxx"
      }
    }
  ]
}
```

### Azure

**Using Azure VMs:**

```bash
# Create VM
az vm create \
  --resource-group monitoring-rg \
  --name lnmonja-server \
  --image UbuntuLTS \
  --size Standard_D4s_v3 \
  --admin-username azureuser

# Install LnMonja (same as Linux installation above)
```

**Using AKS:**
See [KUBERNETES.md](KUBERNETES.md)

### GCP

**Using Compute Engine:**

```bash
gcloud compute instances create lnmonja-server \
  --zone=us-central1-a \
  --machine-type=n2-standard-4 \
  --image-family=ubuntu-2004-lts \
  --image-project=ubuntu-os-cloud \
  --boot-disk-size=100GB \
  --boot-disk-type=pd-ssd
```

**Using GKE:**
See [KUBERNETES.md](KUBERNETES.md)

---

## Security Hardening

### TLS/mTLS Configuration

**Generate certificates:**

```bash
# CA certificate
openssl genrsa -out ca-key.pem 4096
openssl req -new -x509 -days 3650 -key ca-key.pem -out ca.pem

# Server certificate
openssl genrsa -out server-key.pem 4096
openssl req -new -key server-key.pem -out server.csr
openssl x509 -req -days 365 -in server.csr -CA ca.pem -CAkey ca-key.pem -out server.pem

# Client certificate
openssl genrsa -out client-key.pem 4096
openssl req -new -key client-key.pem -out client.csr
openssl x509 -req -days 365 -in client.csr -CA ca.pem -CAkey ca-key.pem -out client.pem
```

**Server configuration:**

```yaml
server:
  grpc:
    tls:
      enabled: true
      cert_file: "/etc/lnmonja/certs/server.pem"
      key_file: "/etc/lnmonja/certs/server-key.pem"
      client_ca_file: "/etc/lnmonja/certs/ca.pem"
```

### Authentication

```yaml
authentication:
  enabled: true
  jwt_secret: "your-secret-key-change-this"
  token_expiry: "24h"

  users:
    - username: "admin"
      password_hash: "$2a$10$..."  # bcrypt hash
      role: "admin"
```

### Firewall Rules

```bash
# Allow only necessary ports
sudo ufw allow 22/tcp    # SSH
sudo ufw allow 8080/tcp  # HTTP API
sudo ufw allow 9090/tcp  # gRPC
sudo ufw allow 3000/tcp  # WebSocket
sudo ufw enable
```

---

## Performance Tuning

### Server Tuning

**1. Storage optimization:**

```yaml
storage:
  compression: true
  value_log_file_size: 2147483648  # 2GB
  mem_table_size: 134217728  # 128MB
  sync_writes: false  # Disable for better performance
```

**2. Connection limits:**

```yaml
limits:
  max_connections: 10000
  max_concurrent_queries: 100
  query_timeout: "30s"
```

**3. System limits:**

```bash
# /etc/security/limits.conf
lnmonja soft nofile 65536
lnmonja hard nofile 65536

# /etc/sysctl.conf
net.core.somaxconn = 4096
net.ipv4.tcp_max_syn_backlog = 4096
```

### Agent Tuning

```yaml
# Reduce collection frequency for less critical metrics
collectors:
  system:
    interval: "5s"  # Instead of 1s
  process:
    interval: "30s"
    max_processes: 100  # Limit top processes
```

---

## Backup and Recovery

### Backup Strategy

**1. Database backup:**

```bash
# Stop server
systemctl stop lnmonja-server

# Backup data directory
tar -czf lnmonja-backup-$(date +%Y%m%d).tar.gz /var/lib/lnmonja

# Upload to S3
aws s3 cp lnmonja-backup-*.tar.gz s3://your-backup-bucket/

# Restart server
systemctl start lnmonja-server
```

**2. Automated backups:**

```bash
#!/bin/bash
# /usr/local/bin/lnmonja-backup.sh

BACKUP_DIR="/backups/lnmonja"
DATA_DIR="/var/lib/lnmonja"
RETENTION_DAYS=30

# Create backup
mkdir -p $BACKUP_DIR
tar -czf $BACKUP_DIR/backup-$(date +%Y%m%d-%H%M%S).tar.gz $DATA_DIR

# Upload to cloud
aws s3 sync $BACKUP_DIR s3://your-backup-bucket/lnmonja/

# Clean old backups
find $BACKUP_DIR -name "backup-*.tar.gz" -mtime +$RETENTION_DAYS -delete
```

**3. Cron schedule:**

```bash
# Daily backups at 2 AM
0 2 * * * /usr/local/bin/lnmonja-backup.sh
```

### Recovery

```bash
# Stop server
systemctl stop lnmonja-server

# Restore from backup
cd /var/lib
rm -rf lnmonja
tar -xzf /backups/lnmonja/backup-YYYYMMDD-HHMMSS.tar.gz

# Set permissions
chown -R lnmonja:lnmonja /var/lib/lnmonja

# Start server
systemctl start lnmonja-server
```

---

## Monitoring the Monitor

Set up monitoring for LnMonja itself:

```yaml
# Alert if server is down
- name: "LnMonja Server Down"
  check_type: "http"
  url: "http://localhost:8080/health"
  interval: "30s"
  timeout: "10s"

# Alert if agents disconnected
- name: "Agents Disconnected"
  metric: "lnmonja_connected_agents"
  condition: "<"
  threshold: 1
```

---

## Next Steps

- [Configure Alerts](docs/alerts.md)
- [Set Up Notifications](docs/notifications.md)
- [Kubernetes Integration](KUBERNETES.md)
- [API Documentation](docs/api.md)

---

**Questions?** Join our [community Slack](https://slack.lnmonja.io) or check [docs.lnmonja.io](https://docs.lnmonja.io)
