#!/usr/bin/env bash
# OpsOwl — Restore Script
# Usage: ./restore.sh /path/to/opsowl_20260518_020000.tar.gz
set -euo pipefail

ARCHIVE="${1:?Usage: $0 <backup-archive.tar.gz>}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORK_DIR="/tmp/opsowl_restore_$$"

# Load env
ENV_FILE="$SCRIPT_DIR/../../.env"
if [[ -f "$ENV_FILE" ]]; then
  set -a; source "$ENV_FILE"; set +a
fi

POSTGRES_USER="${POSTGRES_USER:-opsowl}"
POSTGRES_DB="${POSTGRES_DB:-opsowl}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:?POSTGRES_PASSWORD is required}"
MINIO_ROOT_USER="${MINIO_ROOT_USER:-minioadmin}"
MINIO_ROOT_PASSWORD="${MINIO_ROOT_PASSWORD:?MINIO_ROOT_PASSWORD is required}"

echo "▶ Restoring OpsOwl from: $ARCHIVE"
echo ""
echo "⚠️  This will REPLACE all data in the running database and object storage."
read -r -p "Type 'yes' to confirm: " CONFIRM
[[ "$CONFIRM" == "yes" ]] || { echo "Aborted."; exit 1; }

mkdir -p "$WORK_DIR"
trap 'rm -rf "$WORK_DIR"' EXIT

# ── 1. Extract archive ────────────────────────────────────────────────────────
echo "  ▶ Extracting archive..."
tar -xzf "$ARCHIVE" -C "$WORK_DIR"
RESTORE_DIR=$(find "$WORK_DIR" -mindepth 1 -maxdepth 1 -type d | head -1)
echo "  ✅ Extracted to $RESTORE_DIR"

# ── 2. Stop API to prevent writes during restore ──────────────────────────────
echo "  ▶ Pausing API container..."
docker stop opsowl-api opsowl-ai 2>/dev/null || true

# ── 3. Restore PostgreSQL ─────────────────────────────────────────────────────
echo "  ▶ Restoring PostgreSQL..."
PGPASSWORD="$POSTGRES_PASSWORD" docker exec -i opsowl-postgres \
  psql -U "$POSTGRES_USER" -c "DROP DATABASE IF EXISTS ${POSTGRES_DB}_old;" 2>/dev/null || true
PGPASSWORD="$POSTGRES_PASSWORD" docker exec -i opsowl-postgres \
  psql -U "$POSTGRES_USER" postgres -c "
    SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname='$POSTGRES_DB' AND pid <> pg_backend_pid();
    DROP DATABASE IF EXISTS $POSTGRES_DB;
    CREATE DATABASE $POSTGRES_DB OWNER $POSTGRES_USER;
  "
PGPASSWORD="$POSTGRES_PASSWORD" docker exec -i opsowl-postgres \
  pg_restore -U "$POSTGRES_USER" -d "$POSTGRES_DB" --no-owner --role="$POSTGRES_USER" \
  < "$RESTORE_DIR/postgres.dump"
echo "  ✅ PostgreSQL restored"

# ── 4. Restore MinIO ──────────────────────────────────────────────────────────
if [[ -d "$RESTORE_DIR/minio" ]]; then
  echo "  ▶ Restoring MinIO storage..."
  docker run --rm --network opsowl_default \
    -v "$RESTORE_DIR/minio:/restore:ro" \
    --entrypoint sh \
    minio/mc:latest -c "
      mc alias set dst http://opsowl-minio:9000 '$MINIO_ROOT_USER' '$MINIO_ROOT_PASSWORD' --quiet &&
      mc mirror /restore dst/opsowl --overwrite --quiet
    " 2>/dev/null || echo "  ⚠️  MinIO restore via container failed — try manually with mc"
  echo "  ✅ MinIO restored"
fi

# ── 5. Restart services ───────────────────────────────────────────────────────
echo "  ▶ Restarting services..."
docker start opsowl-api opsowl-ai
echo ""
echo "✅ Restore complete. Allow ~30 seconds for the API to become healthy."
