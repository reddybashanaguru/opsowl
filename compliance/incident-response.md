# Incident Response Plan
**OpsOwl AI — Investigation Workspace**
Version 1.0 · 2026-05-26

---

## 1. Scope

This plan covers security incidents, data breaches, and system outages affecting the OpsOwl AI deployment. It applies to all personnel with access to the platform and to the deploying organisation's IT/security teams.

---

## 2. Incident Classification

| Severity | Definition | Response SLA |
|----------|-----------|--------------|
| **P1 — Critical** | Active data breach; unauthorised STR access; system-wide outage | 1 hour |
| **P2 — High** | Potential data exposure; failed authentication spike; SIEM alert triggered | 4 hours |
| **P3 — Medium** | Single user account compromise; service degradation | 24 hours |
| **P4 — Low** | Policy violation; suspicious but unconfirmed activity | 72 hours |

---

## 3. Response Phases

### 3.1 Detect

- SIEM/SOAR receives audit events in real-time via `SIEM_WEBHOOK_URL` (queue depth 1000)
- Account lockout triggers after configurable failed login attempts
- Rate limiter (100 req/min default) blocks brute-force attempts
- Hash-chain audit log detects tampering (`entry_hash` chain breaks)

### 3.2 Contain

1. Admin disables affected user accounts via `PUT /api/v1/users/{id}` (`is_active: false`)
2. All active sessions revoked automatically on account disable
3. API keys for affected integrations revoked via `DELETE /api/v1/api-keys/{id}`
4. If DB compromise suspected: rotate `ENCRYPTION_KEY` and `DATABASE_URL` credentials; redeploy

### 3.3 Eradicate

1. Rotate all secrets: `ENCRYPTION_KEY`, JWT keys, `STORAGE_ACCESS_KEY`, `STORAGE_SECRET_KEY`
2. Audit all access logs for the affected period (`GET /api/v1/audit?from=&to=`)
3. Review hash-chain integrity for audit log tampering
4. Patch or redeploy affected containers

### 3.4 Recover

1. Restore from last known-good PostgreSQL backup
2. Verify data integrity via hash-chain validation
3. Re-enable affected users after credential rotation
4. Run full E2E test suite to confirm system integrity

### 3.5 Report

| Obligation | Deadline | Body |
|-----------|----------|------|
| DPDP Act 2023 — personal data breach | 72 hours | Data Protection Board of India |
| RBI Cyber Security Framework | 6 hours (for material incidents) | RBI |
| PMLA — STR-related breach | Immediate | FIU-IND |
| Internal DPO notification | 1 hour | Internal |

---

## 4. Communication Template (P1/P2)

```
SUBJECT: [SEVERITY] OpsOwl Security Incident — [DATE]

Incident detected: [timestamp]
Nature: [brief description — no PII in initial notification]
Affected data: [categories, estimated count]
Containment status: [contained / in progress]
Next update: [timestamp]
Incident commander: [name]
```

---

## 5. Post-Incident

- Root cause analysis completed within 5 business days
- Lessons learned shared with security team
- DPIA updated if processing activities changed
- Policy updated if gap identified

---

## 6. Contact List (to be completed by deploying organisation)

| Role | Name | Contact |
|------|------|---------|
| Incident Commander | | |
| DPO | | |
| CISO | | |
| RBI Nodal Officer | | |
| FIU-IND Contact | | |
| OpsOwl Support | security@opsowl.dev | |

---

*Review trigger: post-incident, annual, or material change to infrastructure.*
