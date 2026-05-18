#!/usr/bin/env bash
# Generates a self-signed TLS certificate for local/dev use.
# For production: replace the generated files with certs from your CA.
set -euo pipefail

CERT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../docker/nginx/ssl" && pwd)"
mkdir -p "$CERT_DIR"

echo "▶ Generating self-signed TLS certificate in: $CERT_DIR"

openssl req -x509 -newkey rsa:4096 -sha256 -days 3650 -nodes \
  -keyout "$CERT_DIR/server.key" \
  -out    "$CERT_DIR/server.crt" \
  -subj   "/CN=opsowl-local/O=OpsOwl/C=IN" \
  -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"

chmod 600 "$CERT_DIR/server.key"
chmod 644 "$CERT_DIR/server.crt"

echo "✅ Certificate written to $CERT_DIR"
echo ""
echo "For production: replace server.crt and server.key with your CA-signed certificate."
echo "The key file must be readable only by root (chmod 600)."
