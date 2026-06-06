#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# OpsOwl AI — On-Premise Installer
# Version: 1.0
#
# Usage: bash install.sh
#
# What this script does:
#   1. Preflight checks (Docker, Compose, disk, RAM)
#   2. Interactively configures .env if not already present
#   3. Generates TLS certificate (self-signed) or uses provided cert
#   4. Pulls OpsOwl images + standard images
#   5. Starts all services in dependency order
#   6. Waits for health checks to pass
#   7. Prints access URL and admin credentials
# ─────────────────────────────────────────────────────────────────────────────
set -euo pipefail

INSTALL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="$INSTALL_DIR/.env"
LOG_FILE="$INSTALL_DIR/install.log"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'

info()    { echo -e "${CYAN}[INFO]${NC}  $*" | tee -a "$LOG_FILE"; }
success() { echo -e "${GREEN}[OK]${NC}    $*" | tee -a "$LOG_FILE"; }
warn()    { echo -e "${YELLOW}[WARN]${NC}  $*" | tee -a "$LOG_FILE"; }
error()   { echo -e "${RED}[ERROR]${NC} $*" | tee -a "$LOG_FILE"; exit 1; }

# ── Banner ────────────────────────────────────────────────────────────────────
echo ""
echo -e "${CYAN}┌─────────────────────────────────────────────────────────┐${NC}"
echo -e "${CYAN}│         OpsOwl AI — On-Premise Installer v1.0           │${NC}"
echo -e "${CYAN}│         AI-Native Investigation Workspace                │${NC}"
echo -e "${CYAN}└─────────────────────────────────────────────────────────┘${NC}"
echo ""
info "Installation log: $LOG_FILE"
echo ""

# ── 1. Preflight Checks ───────────────────────────────────────────────────────
info "Running preflight checks..."

# Docker
if ! command -v docker &>/dev/null; then
  error "Docker is not installed. Install Docker Engine 24+ from https://docs.docker.com/engine/install/"
fi
DOCKER_VERSION=$(docker version --format '{{.Server.Version}}' 2>/dev/null || echo "0")
DOCKER_MAJOR=$(echo "$DOCKER_VERSION" | cut -d. -f1)
if [[ "$DOCKER_MAJOR" -lt 24 ]]; then
  error "Docker $DOCKER_VERSION detected. OpsOwl requires Docker 24+. Please upgrade."
fi
success "Docker $DOCKER_VERSION"

# Docker Compose (v2 plugin)
if ! docker compose version &>/dev/null; then
  error "Docker Compose plugin (v2) not found. Install via: sudo apt install docker-compose-plugin"
fi
COMPOSE_VERSION=$(docker compose version --short 2>/dev/null || echo "0.0.0")
success "Docker Compose $COMPOSE_VERSION"

# Available RAM
TOTAL_RAM_KB=$(grep MemTotal /proc/meminfo 2>/dev/null | awk '{print $2}' || sysctl -n hw.memsize 2>/dev/null | awk '{print $1/1024}' || echo 0)
TOTAL_RAM_GB=$(( TOTAL_RAM_KB / 1024 / 1024 ))
if [[ "$TOTAL_RAM_GB" -lt 8 ]]; then
  warn "Only ${TOTAL_RAM_GB}GB RAM detected. OpsOwl requires at least 8GB. Performance may be degraded."
else
  success "${TOTAL_RAM_GB}GB RAM available"
fi

# Available disk space (require 50GB free in install dir)
FREE_DISK_KB=$(df -k "$INSTALL_DIR" | awk 'NR==2 {print $4}')
FREE_DISK_GB=$(( FREE_DISK_KB / 1024 / 1024 ))
if [[ "$FREE_DISK_GB" -lt 50 ]]; then
  warn "Only ${FREE_DISK_GB}GB free disk space. Recommended: 50GB+ for data and AI models."
else
  success "${FREE_DISK_GB}GB disk space free"
fi

# CPU cores
CPU_CORES=$(nproc 2>/dev/null || sysctl -n hw.logicalcpu 2>/dev/null || echo "?")
success "$CPU_CORES CPU cores"
echo ""

# ── 2. Configure Environment ──────────────────────────────────────────────────
if [[ -f "$ENV_FILE" ]]; then
  warn ".env already exists at $ENV_FILE"
  read -rp "  Overwrite it? [y/N] " OVERWRITE
  if [[ "$OVERWRITE" != "y" && "$OVERWRITE" != "Y" ]]; then
    info "Using existing .env"
  else
    rm -f "$ENV_FILE"
    _configure_env=true
  fi
else
  _configure_env=true
fi

if [[ "${_configure_env:-false}" == "true" ]]; then
  info "Configuring OpsOwl environment..."
  echo ""
  cp "$INSTALL_DIR/.env.example" "$ENV_FILE"

  # Public URL
  read -rp "  Public URL (e.g. https://opsowl.yourcompany.com or https://192.168.1.100): " PUBLIC_URL
  PUBLIC_URL="${PUBLIC_URL:-https://localhost}"
  sed -i.bak "s|PUBLIC_URL=.*|PUBLIC_URL=$PUBLIC_URL|" "$ENV_FILE"
  sed -i.bak "s|CORS_ALLOWED_ORIGINS=.*|CORS_ALLOWED_ORIGINS=$PUBLIC_URL|" "$ENV_FILE"

  # Admin credentials
  read -rp "  Admin email address: " ADMIN_EMAIL
  ADMIN_EMAIL="${ADMIN_EMAIL:-admin@opsowl.local}"
  read -rsp "  Admin password (min 8 chars, uppercase + digit): " ADMIN_PASSWORD
  echo ""
  ADMIN_NAME_DEFAULT="System Administrator"
  read -rp "  Admin full name [$ADMIN_NAME_DEFAULT]: " ADMIN_NAME
  ADMIN_NAME="${ADMIN_NAME:-$ADMIN_NAME_DEFAULT}"

  sed -i.bak "s|SEED_ADMIN_EMAIL=.*|SEED_ADMIN_EMAIL=$ADMIN_EMAIL|" "$ENV_FILE"
  sed -i.bak "s|SEED_ADMIN_PASSWORD=.*|SEED_ADMIN_PASSWORD=$ADMIN_PASSWORD|" "$ENV_FILE"
  sed -i.bak "s|SEED_ADMIN_NAME=.*|SEED_ADMIN_NAME=$ADMIN_NAME|" "$ENV_FILE"

  # Generate random secrets
  info "Generating cryptographic secrets..."
  PG_PASS=$(openssl rand -base64 24)
  MINIO_PASS=$(openssl rand -base64 18 | tr -d '=+/')
  ENC_KEY=$(openssl rand -base64 32)
  AI_SECRET=$(openssl rand -hex 32)
  MINIO_KMS=$(openssl rand -base64 32)
  GRAFANA_PASS=$(openssl rand -base64 12 | tr -d '=+/')

  sed -i.bak "s|POSTGRES_PASSWORD=.*|POSTGRES_PASSWORD=$PG_PASS|" "$ENV_FILE"
  sed -i.bak "s|MINIO_ROOT_PASSWORD=.*|MINIO_ROOT_PASSWORD=$MINIO_PASS|" "$ENV_FILE"
  sed -i.bak "s|MINIO_KMS_SECRET_KEY=.*|MINIO_KMS_SECRET_KEY=opsowl-key:$MINIO_KMS|" "$ENV_FILE"
  sed -i.bak "s|ENCRYPTION_KEY=.*|ENCRYPTION_KEY=$ENC_KEY|" "$ENV_FILE"
  sed -i.bak "s|AI_SERVICE_SECRET=.*|AI_SERVICE_SECRET=$AI_SECRET|" "$ENV_FILE"
  sed -i.bak "s|GRAFANA_PASSWORD=.*|GRAFANA_PASSWORD=$GRAFANA_PASS|" "$ENV_FILE"
  rm -f "$ENV_FILE.bak"

  success "Environment configured"
fi

# Load env for use in this script
# shellcheck disable=SC2046
export $(grep -v '^#' "$ENV_FILE" | grep -v '^$' | xargs)
echo ""

# ── 3. TLS Certificate ────────────────────────────────────────────────────────
SSL_DIR="$INSTALL_DIR/ssl"
mkdir -p "$SSL_DIR"

if [[ -f "$SSL_DIR/server.crt" && -f "$SSL_DIR/server.key" ]]; then
  success "TLS certificate already present — skipping generation"
else
  info "TLS certificate setup..."
  echo ""
  echo "  Options:"
  echo "  [1] Generate self-signed certificate (for internal/private deployments)"
  echo "  [2] Provide your own certificate files"
  echo ""
  read -rp "  Choice [1]: " TLS_CHOICE
  TLS_CHOICE="${TLS_CHOICE:-1}"

  if [[ "$TLS_CHOICE" == "2" ]]; then
    read -rp "  Path to .crt file: " CERT_PATH
    read -rp "  Path to .key file: " KEY_PATH
    cp "$CERT_PATH" "$SSL_DIR/server.crt"
    cp "$KEY_PATH" "$SSL_DIR/server.key"
    chmod 600 "$SSL_DIR/server.key"
    success "TLS certificate installed"
  else
    # Derive domain from PUBLIC_URL
    DOMAIN=$(echo "${PUBLIC_URL:-localhost}" | sed 's|https://||' | sed 's|http://||' | cut -d'/' -f1 | cut -d':' -f1)
    info "Generating self-signed certificate for: $DOMAIN"
    openssl req -x509 -newkey rsa:4096 -sha256 -days 3650 -nodes \
      -keyout "$SSL_DIR/server.key" \
      -out "$SSL_DIR/server.crt" \
      -subj "/CN=$DOMAIN/O=OpsOwl/OU=OpsOwl AI" \
      -addext "subjectAltName=DNS:$DOMAIN,IP:127.0.0.1" \
      2>/dev/null
    chmod 600 "$SSL_DIR/server.key"
    success "Self-signed TLS certificate generated (valid 10 years)"
    warn "For production, replace ssl/server.crt and ssl/server.key with a CA-signed certificate"
  fi
fi
echo ""

# ── 4. Nginx Configuration ────────────────────────────────────────────────────
NGINX_CONF="$INSTALL_DIR/nginx.conf"
if [[ ! -f "$NGINX_CONF" ]]; then
  info "Writing nginx.conf..."
  cat > "$NGINX_CONF" << 'NGINXEOF'
worker_processes auto;
error_log /var/log/nginx/error.log warn;
pid /var/run/nginx.pid;

events { worker_connections 1024; }

http {
    include       /etc/nginx/mime.types;
    default_type  application/octet-stream;

    sendfile on;
    keepalive_timeout 65;
    client_max_body_size 50M;

    add_header X-Frame-Options SAMEORIGIN always;
    add_header X-Content-Type-Options nosniff always;
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;

    gzip on;
    gzip_types text/plain application/json application/javascript text/css;

    server {
        listen 80;
        server_name _;
        return 301 https://$host$request_uri;
    }

    server {
        listen 443 ssl;
        server_name _;

        ssl_certificate     /etc/nginx/ssl/server.crt;
        ssl_certificate_key /etc/nginx/ssl/server.key;
        ssl_protocols TLSv1.2 TLSv1.3;
        ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384;
        ssl_prefer_server_ciphers off;
        ssl_session_cache shared:SSL:10m;

        location /api/ {
            proxy_pass         http://api:8080;
            proxy_http_version 1.1;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
            proxy_read_timeout 300s;
        }

        location / {
            proxy_pass         http://web:3000;
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-Proto $scheme;
        }
    }
}
NGINXEOF
  success "nginx.conf written"
fi

# ── Prometheus Config ─────────────────────────────────────────────────────────
PROM_CONF="$INSTALL_DIR/prometheus.yml"
if [[ ! -f "$PROM_CONF" ]]; then
  cat > "$PROM_CONF" << 'PROMEOF'
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: opsowl-api
    static_configs:
      - targets: ['api:8080']
    metrics_path: /metrics
PROMEOF
fi

# ── 5. Pull Images ────────────────────────────────────────────────────────────
echo ""
info "Pulling Docker images (this may take 5–15 minutes on first install)..."
info "AI models (llama3.1:8b + nomic-embed-text) will be downloaded after services start."
echo ""

cd "$INSTALL_DIR"
docker compose pull --quiet postgres minio clamav nginx prometheus grafana 2>&1 | tee -a "$LOG_FILE" || true

info "Pulling OpsOwl application images (version: ${OPSOWL_VERSION:-latest})..."
docker compose pull --quiet api web ai-service 2>&1 | tee -a "$LOG_FILE" || \
  warn "Could not pull OpsOwl images — they may not be published yet for this version"

# ── 6. Start Services ─────────────────────────────────────────────────────────
echo ""
info "Starting infrastructure services..."
docker compose up -d postgres minio clamav

info "Waiting for PostgreSQL to be healthy..."
RETRIES=30
until docker compose exec -T postgres pg_isready -U "${POSTGRES_USER:-opsowl}" -d "${POSTGRES_DB:-opsowl}" &>/dev/null; do
  RETRIES=$((RETRIES - 1))
  if [[ "$RETRIES" -le 0 ]]; then
    error "PostgreSQL did not become healthy within 5 minutes. Check: docker compose logs postgres"
  fi
  sleep 10
done
success "PostgreSQL healthy"

info "Starting Ollama (local AI inference engine)..."
docker compose up -d ollama

info "Starting OpsOwl API server (runs database migrations + seeds admin user)..."
docker compose up -d api

info "Waiting for API to be healthy..."
RETRIES=36
until docker compose exec -T api wget -qO- http://localhost:8080/health &>/dev/null; do
  RETRIES=$((RETRIES - 1))
  if [[ "$RETRIES" -le 0 ]]; then
    error "API did not become healthy within 6 minutes. Check: docker compose logs api"
  fi
  sleep 10
done
success "API healthy"

info "Starting remaining services..."
docker compose up -d

# ── 7. Download Ollama AI Models (background) ─────────────────────────────────
info "Pulling AI models in background (llama3.1:8b + nomic-embed-text)..."
docker compose up ollama-init 2>&1 | tee -a "$LOG_FILE" &
MODEL_PID=$!

# ── 8. Final Health Check ─────────────────────────────────────────────────────
echo ""
info "Running final service health checks..."
sleep 5

ALL_HEALTHY=true
for svc in postgres minio api web nginx; do
  STATUS=$(docker compose ps "$svc" --format json 2>/dev/null | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('Health','unknown'))" 2>/dev/null || echo "unknown")
  if [[ "$STATUS" == "healthy" || "$STATUS" == "unknown" ]]; then
    success "$svc: running"
  else
    warn "$svc: $STATUS"
    ALL_HEALTHY=false
  fi
done

# ── 9. Summary ────────────────────────────────────────────────────────────────
echo ""
echo -e "${GREEN}═══════════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}   OpsOwl AI is installed and running!${NC}"
echo -e "${GREEN}═══════════════════════════════════════════════════════════${NC}"
echo ""
echo -e "  Web UI:       ${CYAN}${PUBLIC_URL:-https://localhost}${NC}"
echo -e "  Admin login:  ${CYAN}${SEED_ADMIN_EMAIL:-admin@opsowl.local}${NC}"
echo -e "  Password:     (the one you entered above)"
echo ""
echo -e "  Grafana:      ${CYAN}http://localhost:3003${NC}  (localhost only)"
echo -e "  Prometheus:   ${CYAN}http://localhost:9090${NC}  (localhost only)"
echo ""
echo -e "  Backups:      $(docker volume inspect opsowl_pg_backups --format '{{.Mountpoint}}' 2>/dev/null || echo 'docker volume opsowl_pg_backups')"
echo -e "  Logs:         docker compose logs -f [api|web|ai-service|postgres]"
echo ""

if [[ "$ALL_HEALTHY" == "false" ]]; then
  warn "One or more services are not yet healthy. Run 'docker compose ps' to check status."
  warn "AI models are still downloading in the background — this can take 10–20 min."
fi

echo -e "  AI models are downloading in the background (PID: $MODEL_PID)."
echo -e "  Run:  ${CYAN}docker compose logs -f ollama-init${NC}  to watch progress."
echo ""
echo -e "  Next steps:"
echo -e "  1. Open ${PUBLIC_URL:-https://localhost} in your browser"
echo -e "  2. Log in with the admin credentials above"
echo -e "  3. Go to Settings → Users to add your team"
echo -e "  4. Go to Settings → SLA Policies to configure deadlines"
echo -e "  5. If using LDAP, configure it in Settings → Authentication"
echo ""
echo -e "  Support: support@opsowl.dev | Docs: ${PUBLIC_URL:-https://localhost}/docs"
echo ""
