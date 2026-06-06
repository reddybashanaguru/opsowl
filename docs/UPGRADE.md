# OpsOwl AI — Upgrade Guide

**Version:** 1.0 | **Last Updated:** June 2026

---

## Overview

OpsOwl upgrades are zero-downtime rolling restarts. Infrastructure services (PostgreSQL, MinIO, Ollama) stay running throughout. The application layer (API, Web, AI service) is replaced image-by-image. Database migrations run automatically when the new API container starts.

**Expected downtime:** ~30 seconds (API restart window)

---

## Before Every Upgrade

1. **Check the release notes** for the target version. Breaking changes and migration notes are listed there.
2. **Verify disk space:** migrations may grow the database. Ensure at least 2× the current database size is free.
3. **Schedule during off-hours.** While downtime is brief, avoid upgrading during peak investigation periods.
4. **Notify your team** — active browser sessions will need a page refresh after the upgrade.

---

## Upgrade Steps

### Automated (recommended)

```bash
cd /opt/opsowl
bash upgrade.sh 1.1.0
```

The script:
1. Takes a pre-upgrade database snapshot
2. Updates `OPSOWL_VERSION` in `.env`
3. Pulls new `api`, `web`, and `ai-service` images
4. Restarts the application layer
5. Waits for the API health check to pass

### Manual

```bash
cd /opt/opsowl

# 1. Backup
docker compose exec postgres pg_dump -U opsowl opsowl | gzip > /tmp/pre-upgrade.sql.gz

# 2. Set new version
sed -i 's/^OPSOWL_VERSION=.*/OPSOWL_VERSION=1.1.0/' .env

# 3. Pull new images
docker compose pull api web ai-service

# 4. Restart application services
docker compose up -d --no-deps api web ai-service

# 5. Verify health
docker compose ps
curl -sf https://localhost/api/v1/health
```

---

## Rolling Back

If the new version has an issue:

```bash
# 1. Revert to previous version
bash upgrade.sh 1.0.0

# OR — if the DB migration broke things:

# 2. Stop the new API
docker compose stop api

# 3. Restore database from snapshot
gunzip -c /tmp/pre-upgrade.sql.gz | docker compose exec -T postgres psql -U opsowl opsowl

# 4. Start old image
OPSOWL_VERSION=1.0.0 docker compose up -d api
```

---

## Version Release Cadence

| Channel | Frequency | Contents |
|---------|-----------|----------|
| Patch (1.0.x) | As needed | Bug fixes, security patches |
| Minor (1.x.0) | Monthly | New features, non-breaking changes |
| Major (x.0.0) | Annually | Significant changes; migration guide provided |

---

## Upgrade History Log

Maintain this table per deployment. The upgrade script writes a log file (`upgrade-YYYYMMDD-HHMMSS.log`) for each run.

| Date | From | To | Upgraded By | Notes |
|------|------|----|-------------|-------|
| (fill in) | | | | |

---

## Checking Your Current Version

```bash
# From the install directory
grep ^OPSOWL_VERSION .env

# From the API
curl -sf https://localhost/api/v1/health | python3 -m json.tool
```

---

## Infrastructure Image Updates

Standard images (PostgreSQL, MinIO, Nginx, etc.) are pinned to specific versions in `docker-compose.yml` and are **not** updated by `upgrade.sh`. To update them:

```bash
# Review pinned versions in docker-compose.yml first
# Update specific image, e.g. nginx
docker compose pull nginx
docker compose up -d --no-deps nginx
```

Apply PostgreSQL minor version upgrades only during a scheduled maintenance window — these require stopping the database container.
