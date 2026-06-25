#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# OpsOwl AI — Air-Gap Bundle Creator
#
# Run this on a machine WITH internet access, then copy the output bundle
# to the target air-gapped server and run install.sh --offline.
#
# Usage: bash bundle.sh [version]
#   version  optional OpsOwl version tag, defaults to OPSOWL_VERSION in .env
#            or "latest"
#
# Output:
#   deploy/opsowl-bundle-<version>.tar.gz  (~8–15 GB depending on AI model)
# ─────────────────────────────────────────────────────────────────────────────
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VERSION="${1:-}"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'
info()    { echo -e "${CYAN}[INFO]${NC}  $*"; }
success() { echo -e "${GREEN}[OK]${NC}    $*"; }
warn()    { echo -e "${YELLOW}[WARN]${NC}  $*"; }
error()   { echo -e "${RED}[ERROR]${NC} $*"; exit 1; }

echo ""
echo -e "${CYAN}┌─────────────────────────────────────────────────────────┐${NC}"
echo -e "${CYAN}│       OpsOwl AI — Air-Gap Bundle Creator                │${NC}"
echo -e "${CYAN}└─────────────────────────────────────────────────────────┘${NC}"
echo ""

# ── Resolve version ───────────────────────────────────────────────────────────
if [[ -z "$VERSION" ]]; then
  ENV_FILE="$SCRIPT_DIR/.env"
  if [[ -f "$ENV_FILE" ]]; then
    VERSION=$(grep -E "^OPSOWL_VERSION=" "$ENV_FILE" 2>/dev/null | cut -d= -f2 | tr -d '"' || true)
  fi
fi
VERSION="${VERSION:-latest}"
info "Bundling OpsOwl version: $VERSION"

# ── Preflight ─────────────────────────────────────────────────────────────────
command -v docker >/dev/null 2>&1 || error "Docker is required"
command -v gzip   >/dev/null 2>&1 || error "gzip is required"

FREE_GB=$(df -k "$SCRIPT_DIR" | awk 'NR==2{printf "%d", $4/1024/1024}')
if [[ "$FREE_GB" -lt 20 ]]; then
  warn "Only ${FREE_GB}GB free. Bundle may require 15–20 GB. Proceeding anyway."
fi

# ── Collect images ────────────────────────────────────────────────────────────
# Resolve OPSOWL_VERSION in the compose file before extracting image list
COMPOSE_FILE="$SCRIPT_DIR/docker-compose.yml"
[[ -f "$COMPOSE_FILE" ]] || error "docker-compose.yml not found in $SCRIPT_DIR"

info "Collecting image list from docker-compose.yml..."
IMAGES=$(OPSOWL_VERSION="$VERSION" docker compose -f "$COMPOSE_FILE" config --images 2>/dev/null | sort -u)
if [[ -z "$IMAGES" ]]; then
  error "No images found. Ensure docker-compose.yml is valid."
fi

echo ""
info "Images to bundle:"
while IFS= read -r img; do
  echo "    $img"
done <<< "$IMAGES"
echo ""

# ── Pull images ───────────────────────────────────────────────────────────────
info "Pulling all images (this may take 10–30 minutes on first run)..."
while IFS= read -r img; do
  echo -n "  Pulling $img ... "
  if docker pull --quiet "$img" >/dev/null 2>&1; then
    echo "done"
  else
    warn "Could not pull $img — it may already be local or unavailable"
  fi
done <<< "$IMAGES"

# ── Save images ───────────────────────────────────────────────────────────────
BUNDLE_NAME="opsowl-bundle-${VERSION}.tar.gz"
BUNDLE_PATH="$SCRIPT_DIR/$BUNDLE_NAME"

info "Saving images to bundle (this takes several minutes)..."
# docker save accepts multiple image arguments; pipe through gzip
# shellcheck disable=SC2086
docker save $IMAGES | gzip > "$BUNDLE_PATH"

BUNDLE_SIZE=$(du -sh "$BUNDLE_PATH" | cut -f1)
success "Bundle created: $BUNDLE_PATH  ($BUNDLE_SIZE)"

# ── Write manifest ────────────────────────────────────────────────────────────
MANIFEST_PATH="$SCRIPT_DIR/opsowl-bundle-${VERSION}.manifest"
{
  echo "opsowl_version=$VERSION"
  echo "created_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo "bundle_file=$BUNDLE_NAME"
  echo "sha256=$(shasum -a 256 "$BUNDLE_PATH" | cut -d' ' -f1)"
  echo "images_count=$(echo "$IMAGES" | wc -l | tr -d ' ')"
  echo "images:"
  while IFS= read -r img; do
    echo "  - $img"
  done <<< "$IMAGES"
} > "$MANIFEST_PATH"
success "Manifest written: $MANIFEST_PATH"

echo ""
echo -e "${GREEN}═══════════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}   Bundle ready for air-gapped deployment!${NC}"
echo -e "${GREEN}═══════════════════════════════════════════════════════════${NC}"
echo ""
echo "  Transfer these two files to the target server:"
echo "    $BUNDLE_PATH"
echo "    $MANIFEST_PATH"
echo "  ...along with the entire deploy/ directory."
echo ""
echo "  On the target server:"
echo "    bash install.sh --offline"
echo ""
echo "  The installer will load images from the bundle automatically."
echo ""
