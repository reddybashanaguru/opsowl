# OpsOwl AI — Deployment Guide

**Version:** 1.0 | **Last Updated:** June 2026

---

## Overview

OpsOwl AI deploys as a Docker Compose stack. Every service runs in your own infrastructure — nothing calls home, no data leaves your network. A single Linux server is all you need.

---

## Hardware Requirements

### Minimum (up to 25 investigators, ~5,000 active cases)

| Component | Minimum |
|-----------|---------|
| CPU | 8 cores (x86-64) |
| RAM | 16 GB |
| Disk | 100 GB SSD |
| Network | 100 Mbps LAN |
| OS | Ubuntu 22.04 LTS / RHEL 9 / Debian 12 |

### Recommended (up to 100 investigators, ~50,000 active cases)

| Component | Recommended |
|-----------|-------------|
| CPU | 16 cores (x86-64) |
| RAM | 32 GB |
| Disk | 500 GB NVMe SSD (OS + data) + 1 TB for document storage |
| Network | 1 Gbps LAN |
| OS | Ubuntu 22.04 LTS |

### With GPU (faster AI inference — optional)

| GPU | Notes |
|-----|-------|
| NVIDIA RTX 3060 / A4000 or better | Ollama auto-detects CUDA; reduces AI response time from ~30s to ~3s |
| Apple Silicon (M1/M2 Mac) | Supported; dev/eval only — not recommended for production |

> **Storage sizing rule of thumb:** 1 GB per 500 documents uploaded. Add 10–20 GB for Ollama AI model files (downloaded on first boot).

---

## Software Requirements

| Software | Version | Notes |
|----------|---------|-------|
| Docker Engine | 24.0+ | Install: https://docs.docker.com/engine/install/ |
| Docker Compose plugin | 2.20+ | Included in Docker Desktop; `apt install docker-compose-plugin` |
| OpenSSL | 1.1+ | Pre-installed on all major Linux distros |
| Python 3.8+ | Optional | Only needed if using the health check script |

---

## Network & Firewall Rules

### Inbound (ports to open on the server's firewall)

| Port | Protocol | Source | Purpose |
|------|----------|--------|---------|
| 443 | TCP | Investigator workstations (LAN/VPN) | OpsOwl web UI + API (HTTPS) |
| 80 | TCP | Investigator workstations | HTTP → HTTPS redirect |
| 22 | TCP | Admin IPs only | SSH management |

### Internal (container-to-container — no firewall rules needed)

| Service | Port | Notes |
|---------|------|-------|
| PostgreSQL | 5432 | Internal only, NOT exposed to host |
| MinIO | 9000 | Internal only |
| Go API | 8080 | Bound to 127.0.0.1 — not reachable externally |
| Python AI | 8000 | Internal only |
| Ollama | 11434 | Internal only |

### Localhost-only (accessible only from the server itself)

| Port | Service | Access Method |
|------|---------|---------------|
| 9090 | Prometheus | SSH tunnel: `ssh -L 9090:localhost:9090 your-server` |
| 3003 | Grafana | SSH tunnel: `ssh -L 3003:localhost:3003 your-server` |
| 9093 | Alertmanager | SSH tunnel |

### Outbound (what OpsOwl calls out to)

| Destination | Port | Purpose | Required? |
|-------------|------|---------|-----------|
| hub.docker.com / ghcr.io | 443 | Pull Docker images (install + upgrade only) | Yes (install time) |
| ollama.com registry | 443 | Download AI models (first boot only) | Yes (first boot) |
| Your SMTP server | 587 | Password reset, scheduled reports | Optional |
| Your LDAP/AD server | 389/636 | User authentication | Optional |
| Your SIEM webhook URL | 443 | Audit event forwarding | Optional |

> **Air-gapped deployments:** After initial install, no outbound internet is required. Subsequent upgrades require temporary internet access to pull new images.

---

## DNS Configuration

Create an A record pointing to your server's IP:

```
opsowl.yourcompany.com   A   192.168.1.100
```

Then set `PUBLIC_URL=https://opsowl.yourcompany.com` in `.env` before installing.

---

## TLS Certificate Options

### Option 1: Self-signed (internal deployments)
`install.sh` generates a 4096-bit RSA self-signed cert valid for 10 years. Users will see a browser warning on first visit — they can accept it and add an exception. Suitable for internal teams on a private network.

### Option 2: CA-signed certificate (recommended for production)
Obtain a cert from your internal CA or a public CA (Let's Encrypt, DigiCert, etc.) and place the files at:
```
deploy/ssl/server.crt   # full chain PEM
deploy/ssl/server.key   # private key PEM (chmod 600)
```
Then restart nginx: `docker compose restart nginx`

### Option 3: Let's Encrypt (public DNS required)
If the server has a public IP and you have a domain name:
```bash
sudo apt install certbot
sudo certbot certonly --standalone -d opsowl.yourcompany.com
sudo cp /etc/letsencrypt/live/opsowl.yourcompany.com/fullchain.pem deploy/ssl/server.crt
sudo cp /etc/letsencrypt/live/opsowl.yourcompany.com/privkey.pem deploy/ssl/server.key
docker compose restart nginx
```
Set up auto-renewal via cron: `0 2 * * 1 certbot renew --quiet && docker compose restart nginx`

---

## Installation Steps

```bash
# 1. Copy installer to your server
scp -r deploy/ admin@your-server:/opt/opsowl/

# 2. SSH in
ssh admin@your-server
cd /opt/opsowl

# 3. Run installer (interactive — takes ~5–15 minutes)
bash install.sh

# 4. Access the UI
# Open https://your-server-or-domain in a browser
```

---

## LDAP / Active Directory Integration

Configure these variables in `.env`:

```env
LDAP_URL=ldap://dc.yourcompany.com:389
LDAP_BASE_DN=DC=yourcompany,DC=com
LDAP_USER_ATTR=sAMAccountName        # Use 'uid' for OpenLDAP
LDAP_BIND_DN=CN=opsowl-svc,OU=Service Accounts,DC=yourcompany,DC=com
LDAP_BIND_PASSWORD=your-service-account-password
```

After editing `.env`, restart the API: `docker compose restart api`

OpsOwl users must still be created in the OpsOwl UI (to assign roles). On login, OpsOwl authenticates the password against LDAP. If LDAP is unreachable, local OpsOwl credentials are used as fallback.

---

## Data Directories

All persistent data is stored in Docker named volumes:

| Volume | Contents | Backup Priority |
|--------|----------|----------------|
| `opsowl_postgres_data` | All case data, users, audit log | **Critical** |
| `opsowl_minio_data` | Uploaded documents, evidence files | **Critical** |
| `opsowl_ollama_data` | Downloaded AI models (~10–20 GB) | Low (re-downloadable) |
| `opsowl_pg_backups` | Automated daily database dumps | Medium |

To find actual paths on disk: `docker volume inspect opsowl_postgres_data`

---

## Backup & Recovery

### Automated daily backups
The `pg-backup` container takes a `pg_dump` every 24 hours and stores compressed `.sql.gz` files in the `pg_backups` volume (7-day retention).

### Manual backup
```bash
docker compose exec postgres pg_dump -U opsowl opsowl | gzip > /tmp/backup-$(date +%Y%m%d).sql.gz
```

### Restore from backup
```bash
gunzip -c /tmp/backup-20260606.sql.gz | docker compose exec -T postgres psql -U opsowl opsowl
```

Full backup/restore runbook: [docs/BACKUP_RUNBOOK.md](BACKUP_RUNBOOK.md)

---

## Monitoring

- **Grafana:** `http://your-server:3003` (via SSH tunnel) — pre-configured dashboards for API latency, error rates, active cases
- **Prometheus:** `http://your-server:9090` (via SSH tunnel) — raw metrics
- **API health endpoint:** `https://your-server/api/v1/health` — returns 200 when all healthy

---

## Resource Tuning

### PostgreSQL memory (large deployments)
Edit `docker-compose.yml`, under `postgres → deploy → resources → limits`:
```yaml
memory: 4G   # increase from 2G for >100 users
cpus: "4.0"  # increase from 2.0
```

### Ollama AI model
For faster AI responses on GPU hardware, change `OLLAMA_MODEL` in `.env`:
```env
OLLAMA_MODEL=llama3.1:70b   # higher quality, needs 40GB RAM or GPU VRAM
```

---

## Common Issues

| Symptom | Cause | Fix |
|---------|-------|-----|
| Browser shows SSL error | Self-signed cert | Accept browser exception, or replace with CA cert |
| Login fails | Wrong credentials | Check `SEED_ADMIN_EMAIL` in `.env`; reset via DB if needed |
| "AI analysis failed" | Ollama model not downloaded | Wait 10–20 min; check `docker compose logs ollama-init` |
| Documents not uploading | MinIO not running | `docker compose restart minio` |
| Slow performance | Insufficient RAM | Check `docker stats`; increase server RAM or reduce `max_wal_size` |
| Case creation fails | Database full | Check `df -h`; free disk space |
