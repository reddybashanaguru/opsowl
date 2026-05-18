# OpsOwl — Backup & Disaster Recovery Runbook

## Overview

OpsOwl stores data in two places:
- **PostgreSQL** — all case data, users, audit logs, tasks, notes, SLA policies
- **MinIO** — uploaded documents (PDFs, spreadsheets, etc.)

Both must be backed up together. A database backup without the documents (or vice versa) is incomplete.

**RPO (Recovery Point Objective):** 24 hours with daily backups. Increase backup frequency to reduce this.
**RTO (Recovery Time Objective):** ~15–30 minutes for a full restore on the same hardware.

---

## Backup

### Automated daily backup (recommended)

Add this line to the root crontab (`sudo crontab -e`) on the server running OpsOwl:

```
0 2 * * * /opt/opsowl/infra/scripts/backup.sh >> /var/log/opsowl-backup.log 2>&1
```

This runs the backup at 2:00 AM every day. Backups older than 30 days are deleted automatically.

### Manual backup

```bash
cd /opt/opsowl
./infra/scripts/backup.sh
```

The archive is written to `/opt/opsowl/backups/` by default.
Override with: `BACKUP_DIR=/mnt/nas/opsowl ./infra/scripts/backup.sh`

### What gets backed up

| Component | Method | Output |
|-----------|--------|--------|
| PostgreSQL | `pg_dump --format=custom` | `postgres.dump` |
| MinIO documents | `mc mirror` | `minio/` directory |
| Both | compressed together | `opsowl_YYYYMMDD_HHMMSS.tar.gz` |

### Verify a backup

```bash
# Check the archive is not corrupt
tar -tzf /opt/opsowl/backups/opsowl_20260518_020000.tar.gz | head -20

# Check PostgreSQL dump is valid
tar -xzf /opt/opsowl/backups/opsowl_20260518_020000.tar.gz -C /tmp
pg_restore --list /tmp/20260518_020000/postgres.dump | head -20
```

---

## Disaster Recovery — Full Restore

### Prerequisites

- A working server with Docker and Docker Compose installed
- The `.env` file with the same credentials as the backup was taken from
- The OpsOwl source code or Docker images

### Steps

**1. Start infrastructure only (no API yet)**

```bash
cd /opt/opsowl
make infra          # starts postgres + minio only
```

Wait for postgres to be healthy:
```bash
docker ps | grep postgres
```

**2. Run the restore script**

```bash
./infra/scripts/restore.sh /path/to/opsowl_20260518_020000.tar.gz
```

The script will:
- Ask for confirmation before overwriting any data
- Stop the API container to prevent writes
- Drop and recreate the database from the dump
- Mirror documents back into MinIO
- Restart all services

**3. Verify the restore**

```bash
# API health
curl http://localhost:8080/health

# Check case count matches pre-failure
curl -s http://localhost:8080/health

# Check recent cases are present (after logging in)
curl -s http://localhost:8080/api/v1/cases \
  -H "Authorization: Bearer <your-token>" | python3 -m json.tool | head -20
```

---

## Backup to a remote location (recommended for production)

After the backup script runs, copy the archive off-server:

```bash
# Example: rsync to a NAS or backup server
rsync -avz /opt/opsowl/backups/ backup-server:/opsowl-backups/

# Example: upload to S3 (if client has AWS)
aws s3 sync /opt/opsowl/backups/ s3://client-opsowl-backups/ --storage-class GLACIER
```

Add this to the crontab after the backup line:
```
15 2 * * * rsync -avz /opt/opsowl/backups/ backup-server:/opsowl-backups/ >> /var/log/opsowl-backup.log 2>&1
```

---

## Environment variables for backup script

| Variable | Default | Description |
|----------|---------|-------------|
| `BACKUP_DIR` | `/opt/opsowl/backups` | Where archives are written |
| `RETAIN_DAYS` | `30` | Delete backups older than this |
| `POSTGRES_USER` | from `.env` | Database user |
| `POSTGRES_DB` | from `.env` | Database name |
| `POSTGRES_PASSWORD` | from `.env` | Database password |
| `MINIO_ROOT_USER` | from `.env` | MinIO access key |
| `MINIO_ROOT_PASSWORD` | from `.env` | MinIO secret key |

---

## Incident response checklist

When a data loss event occurs:

- [ ] Identify the last successful backup: `ls -lt /opt/opsowl/backups/`
- [ ] Notify the team of the outage and estimated RTO
- [ ] Stop the API: `docker stop opsowl-api opsowl-ai`
- [ ] Run restore: `./infra/scripts/restore.sh <archive>`
- [ ] Verify restore is complete (health check + spot-check cases)
- [ ] Inform users that the system is restored; note any data gap
- [ ] Document the incident and root cause
