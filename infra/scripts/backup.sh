#!/usr/bin/env bash
# OpsOwl — Backup Script
# Backs up PostgreSQL database + MinIO document storage to a timestamped archive.
# Schedule via cron: 0 2 * * * /opt/opsowl/infra/scripts/backup.sh >> /var/log/opsowl-backup.log 2>&1
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKUP_DIR="${BACKUP_DIR:-/opt/opsowl/backups}"
RETAIN_DAYS="${RETAIN_DAYS:-30}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_PATH="$BACKUP_DIR/$TIMESTAMP"

# Load env if available
ENV_FILE="$SCRIPT_DIR/../../.env"
if [[ -f "$ENV_FILE" ]]; then
  set -a; source "$ENV_FILE"; set +a
fi

POSTGRES_USER="${POSTGRES_USER:-opsowl}"
POSTGRES_DB="${POSTGRES_DB:-opsowl}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:?POSTGRES_PASSWORD is required}"
MINIO_ROOT_USER="${MINIO_ROOT_USER:-minioadmin}"
MINIO_ROOT_PASSWORD="${MINIO_ROOT_PASSWORD:?MINIO_ROOT_PASSWORD is required}"

mkdir -p "$BACKUP_PATH"
echo "[$TIMESTAMP] ▶ Starting OpsOwl backup to $BACKUP_PATH"

# ── 1. PostgreSQL dump ────────────────────────────────────────────────────────
echo "  ▶ Dumping PostgreSQL database..."
PGPASSWORD="$POSTGRES_PASSWORD" docker exec opsowl-postgres \
  pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" --format=custom \
  > "$BACKUP_PATH/postgres.dump"
echo "  ✅ PostgreSQL dump: $(du -sh "$BACKUP_PATH/postgres.dump" | cut -f1)"

# ── 2. MinIO document storage ─────────────────────────────────────────────────
echo "  ▶ Mirroring MinIO storage..."
docker run --rm --network opsowl_default \
  -v "$BACKUP_PATH:/backup" \
  --entrypoint sh \
  minio/mc:latest -c "
    mc alias set src http://opsowl-minio:9000 '$MINIO_ROOT_USER' '$MINIO_ROOT_PASSWORD' --quiet &&
    mc mirror src/opsowl /backup/minio/ --quiet
  " 2>/dev/null || {
    # Fallback: use host-mapped port
    docker run --rm \
      -v "$BACKUP_PATH:/backup" \
      --entrypoint sh \
      minio/mc:latest -c "
        mc alias set src http://host.docker.internal:9000 '$MINIO_ROOT_USER' '$MINIO_ROOT_PASSWORD' --quiet &&
        mc mirror src/opsowl /backup/minio/ --quiet
      "
  }
echo "  ✅ MinIO mirror: $(du -sh "$BACKUP_PATH/minio" 2>/dev/null | cut -f1 || echo '0B')"

# ── 3. Compress archive ───────────────────────────────────────────────────────
echo "  ▶ Compressing backup..."
tar -czf "$BACKUP_DIR/opsowl_$TIMESTAMP.tar.gz" -C "$BACKUP_DIR" "$TIMESTAMP"
rm -rf "$BACKUP_PATH"
ARCHIVE_SIZE=$(du -sh "$BACKUP_DIR/opsowl_$TIMESTAMP.tar.gz" | cut -f1)
echo "  ✅ Archive: opsowl_$TIMESTAMP.tar.gz ($ARCHIVE_SIZE)"

# ── 4. Rotate old backups ─────────────────────────────────────────────────────
echo "  ▶ Removing backups older than $RETAIN_DAYS days..."
find "$BACKUP_DIR" -name "opsowl_*.tar.gz" -mtime "+$RETAIN_DAYS" -delete
REMAINING=$(find "$BACKUP_DIR" -name "opsowl_*.tar.gz" | wc -l | tr -d ' ')
echo "  ✅ $REMAINING backup(s) retained"

echo "[$TIMESTAMP] ✅ Backup complete: $BACKUP_DIR/opsowl_$TIMESTAMP.tar.gz"
