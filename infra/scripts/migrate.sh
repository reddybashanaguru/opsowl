#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MIGRATIONS_DIR="$SCRIPT_DIR/../../apps/api/migrations"

if [[ -z "${DATABASE_URL:-}" ]]; then
  echo "❌ DATABASE_URL is not set"
  exit 1
fi

echo "▶ Running migrations from: $MIGRATIONS_DIR"
goose -dir "$MIGRATIONS_DIR" postgres "$DATABASE_URL" "$@"
echo "✅ Done"
