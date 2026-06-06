#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# OpsOwl AI — Upgrade Script
# Usage: bash upgrade.sh [new-version]
# Example: bash upgrade.sh 1.1.0
# ─────────────────────────────────────────────────────────────────────────────
set -euo pipefail

INSTALL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="$INSTALL_DIR/.env"
LOG_FILE="$INSTALL_DIR/upgrade-$(date +%Y%m%d-%H%M%S).log"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'

info()    { echo -e "${CYAN}[INFO]${NC}  $*" | tee -a "$LOG_FILE"; }
success() { echo -e "${GREEN}[OK]${NC}    $*" | tee -a "$LOG_FILE"; }
warn()    { echo -e "${YELLOW}[WARN]${NC}  $*" | tee -a "$LOG_FILE"; }
error()   { echo -e "${RED}[ERROR]${NC} $*" | tee -a "$LOG_FILE"; exit 1; }

NEW_VERSION="${1:-}"

echo ""
echo -e "${CYAN}┌─────────────────────────────────────────────────────────┐${NC}"
echo -e "${CYAN}│         OpsOwl AI — Upgrade Script                      │${NC}"
echo -e "${CYAN}└─────────────────────────────────────────────────────────┘${NC}"
echo ""

if [[ ! -f "$ENV_FILE" ]]; then
  error "No .env found at $ENV_FILE. Is OpsOwl installed in this directory?"
fi

# Load current env
# shellcheck disable=SC2046
export $(grep -v '^#' "$ENV_FILE" | grep -v '^$' | xargs)

CURRENT_VERSION="${OPSOWL_VERSION:-unknown}"
info "Current version: $CURRENT_VERSION"

if [[ -z "$NEW_VERSION" ]]; then
  read -rp "  Target version (e.g. 1.1.0): " NEW_VERSION
fi

if [[ -z "$NEW_VERSION" ]]; then
  error "No version specified"
fi

info "Upgrading to: $NEW_VERSION"
echo ""

# ── Pre-upgrade Backup ────────────────────────────────────────────────────────
info "Taking pre-upgrade database snapshot..."
BACKUP_FILE="$INSTALL_DIR/pre-upgrade-${CURRENT_VERSION}-$(date +%Y%m%d-%H%M%S).sql.gz"
docker compose exec -T postgres \
  pg_dump -U "${POSTGRES_USER:-opsowl}" "${POSTGRES_DB:-opsowl}" \
  | gzip > "$BACKUP_FILE" && \
  success "Database snapshot: $BACKUP_FILE" || \
  warn "Could not take database snapshot — upgrade will continue"

# ── Update Version in .env ────────────────────────────────────────────────────
sed -i.bak "s|^OPSOWL_VERSION=.*|OPSOWL_VERSION=$NEW_VERSION|" "$ENV_FILE"
rm -f "$ENV_FILE.bak"
success ".env updated: OPSOWL_VERSION=$NEW_VERSION"

# ── Pull New Images ───────────────────────────────────────────────────────────
echo ""
info "Pulling new images..."
cd "$INSTALL_DIR"
docker compose pull api web ai-service 2>&1 | tee -a "$LOG_FILE"

# ── Rolling Restart ───────────────────────────────────────────────────────────
echo ""
info "Stopping application services (infrastructure stays running)..."
docker compose stop api web ai-service

info "Starting updated services (migrations run automatically on API startup)..."
docker compose up -d api

info "Waiting for API to be healthy (migrations may take up to 60s)..."
RETRIES=24
until docker compose exec -T api wget -qO- http://localhost:8080/health &>/dev/null; do
  RETRIES=$((RETRIES - 1))
  if [[ "$RETRIES" -le 0 ]]; then
    error "API did not come up after upgrade. Roll back with: bash upgrade.sh $CURRENT_VERSION"
  fi
  sleep 5
done
success "API healthy"

docker compose up -d web ai-service
sleep 3
success "All services restarted"

# ── Summary ───────────────────────────────────────────────────────────────────
echo ""
echo -e "${GREEN}═══════════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}   Upgrade to $NEW_VERSION complete!${NC}"
echo -e "${GREEN}═══════════════════════════════════════════════════════════${NC}"
echo ""
echo -e "  Previous version backup: ${CYAN}$BACKUP_FILE${NC}"
echo -e "  Upgrade log:             ${CYAN}$LOG_FILE${NC}"
echo ""
echo -e "  If something is broken:"
echo -e "    1. Restore DB:  gunzip -c $BACKUP_FILE | docker compose exec -T postgres psql -U ${POSTGRES_USER:-opsowl} ${POSTGRES_DB:-opsowl}"
echo -e "    2. Rollback:    bash upgrade.sh $CURRENT_VERSION"
echo ""
