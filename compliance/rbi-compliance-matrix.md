# RBI Compliance Matrix
**OpsOwl AI — Investigation Workspace**
Version 1.0 · Assessed 2026-05-26

RBI Master Directions on Information Technology Framework for NBFCs (2017) and
RBI Cyber Security Framework for Banks (2016).

---

## Information Security

| Control | RBI Requirement | OpsOwl Implementation | Status |
|---------|----------------|----------------------|--------|
| IS.1 | Information Security Policy | DPIA + DPA documented; roles enforced | ✅ |
| IS.2 | Access control — least privilege | 5-tier RBAC (viewer/analyst/manager/admin/super_admin) | ✅ |
| IS.3 | Multi-factor authentication | TOTP MFA with backup codes | ✅ |
| IS.4 | Password complexity | Min 12 chars, uppercase, number; 90-day expiry | ✅ |
| IS.5 | Session management | 15-min idle timeout; max sessions configurable | ✅ |
| IS.6 | Audit trails — tamper-evident | Hash-chain audit log (SHA-256 chain) + SIEM forwarding | ✅ |
| IS.7 | Encryption in transit | TLS 1.2+ (REQUIRE_TLS=true); HSTS | ✅ |
| IS.8 | Encryption at rest | AES-256 column encryption; MinIO SSE | ✅ |
| IS.9 | Patch management | Container images pinned; automated DB migration | ✅ |
| IS.10 | Vulnerability assessment | See vendor assessment checklist | 🔶 Customer |

---

## Cyber Security

| Control | RBI Requirement | OpsOwl Implementation | Status |
|---------|----------------|----------------------|--------|
| CS.1 | Real-time security event monitoring | SIEM webhook (queue depth 1000); CEF and JSON formats | ✅ |
| CS.2 | Incident response plan | See incident-response.md | ✅ |
| CS.3 | Network segmentation | Docker Compose internal network; no direct DB exposure | ✅ |
| CS.4 | API security | Rate limiting (100 req/min default); idempotency keys; API key auth | ✅ |
| CS.5 | Privileged access management | Admin/super_admin required for destructive ops; no shared accounts | ✅ |
| CS.6 | Data loss prevention | PMLA retention floor trigger; erasure blocked by hold | ✅ |
| CS.7 | Backup and recovery | PostgreSQL backup sidecar; point-in-time recovery | ✅ |

---

## AML / PMLA Compliance

| Control | PMLA / RBI Requirement | OpsOwl Implementation | Status |
|---------|----------------------|----------------------|--------|
| AML.1 | PMLA s.12 — 10-year record retention | Retention trigger `trg_pmla_retention_floor`; all PMLA resources = 3650 days | ✅ |
| AML.2 | STR filing — dual authorisation | Maker-checker workflow (submitted_by ≠ approved_by); DB CHECK + app layer | ✅ |
| AML.3 | FIU-IND FINnet 2.0 submission | XML export per FINnet 2.0 schema; namespace urn:fiu-ind:finnet2:report:v2.0 | ✅ |
| AML.4 | Tipping-off prevention (PMLA s.8A) | Analyst role blocked from STR routes and regulatory reports | ✅ |
| AML.5 | STR/SAR/CTR workflow tracking | Full lifecycle: draft → submitted → approved → filed | ✅ |
| AML.6 | Customer identification (PAN / Aadhaar) | PAN stored; Aadhaar masked (first4 XXXX last4) per DPDP Act 2023 s.4 | ✅ |

---

## Data Localisation (RBI 2018 Circular)

| Requirement | Status | Notes |
|-------------|--------|-------|
| Payment data stored in India | 🔶 Customer | Storage endpoint is configurable; PSU deployments must use India-region |
| Cross-border transfer prohibited | 🔶 Customer | AI service URL must point to on-premise or India-hosted inference |

Legend: ✅ Implemented by OpsOwl · 🔶 Customer / deployment responsibility
